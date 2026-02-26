package handlers

import (
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/github"
)

// RepoBranches carries per-repo branch config without importing the config package.
type RepoBranches struct {
	Name     string
	Branches []string
}

type RepositoryStatus struct {
	Name     string
	Branches []BranchStatus
	Stale    bool
}

// HasDetails reports whether any branch has failures or running checks.
func (rs *RepositoryStatus) HasDetails() bool {
	for i := range rs.Branches {
		if rs.Branches[i].FailureCount > 0 || rs.Branches[i].RunningCount > 0 {
			return true
		}
	}
	return false
}

type BranchStatus struct {
	Branch        string
	IsMain        bool // true for main/master — used by TV template CSS class
	Checks        []github.Check
	HasChecks     bool
	CommitSHA     string
	CommitURL     string
	SuccessCount  int
	FailureCount  int
	RunningCount  int
	OverallStatus string
	StatusClass   string
}

// sortByConfigOrder sorts repositories to match config.yaml order.
// Unknown repos sort to end, alphabetical among themselves.
func sortByConfigOrder(repos []*RepositoryStatus, repoNames []string) {
	repoIndex := make(map[string]int, len(repoNames))
	for i, name := range repoNames {
		repoIndex[name] = i
	}
	sentinel := len(repoNames)
	sort.Slice(repos, func(i, j int) bool {
		idxI, okI := repoIndex[repos[i].Name]
		idxJ, okJ := repoIndex[repos[j].Name]
		if !okI {
			idxI = sentinel
		}
		if !okJ {
			idxJ = sentinel
		}
		if idxI != idxJ {
			return idxI < idxJ
		}
		return repos[i].Name < repos[j].Name
	})
}

// branchKey is a composite key for indexing branch checks.
type branchKey struct {
	repo, branch string
}

// groupByRepository groups branch checks into sorted repository statuses,
// using repoConfig to determine which branches appear and in what order.
func groupByRepository(branchChecks []github.BranchCheck, repoConfig []RepoBranches) []*RepositoryStatus {
	// Index branch checks by {repo, branch} for O(1) lookup.
	checkIndex := make(map[branchKey]*github.BranchCheck, len(branchChecks))
	for i := range branchChecks {
		bc := &branchChecks[i]
		checkIndex[branchKey{bc.Repository, bc.Branch}] = bc
	}

	// Track which repos we've seen from config.
	configRepos := make(map[string]bool, len(repoConfig))

	var repositories []*RepositoryStatus

	// Iterate config in order — this determines repo and branch ordering.
	for _, rc := range repoConfig {
		configRepos[rc.Name] = true
		rs := &RepositoryStatus{Name: rc.Name}
		hasData := false

		for _, branch := range rc.Branches {
			bs := BranchStatus{
				Branch: branch,
				IsMain: isMainBranch(branch),
				Checks: []github.Check{},
			}
			if bc, ok := checkIndex[branchKey{rc.Name, branch}]; ok {
				bs.Checks = bc.Checks
				bs.HasChecks = len(bc.Checks) > 0
				bs.CommitSHA = bc.CommitSHA
				bs.CommitURL = bc.CommitURL
				computeBranchCounts(&bs)
				hasData = true
			}
			rs.Branches = append(rs.Branches, bs)
		}

		if hasData {
			repositories = append(repositories, rs)
		}
	}

	// Append unknown repos (in branchChecks but not in config), sorted alphabetically.
	unknownRepos := make(map[string]*RepositoryStatus)
	for i := range branchChecks {
		bc := &branchChecks[i]
		if configRepos[bc.Repository] {
			continue
		}
		rs, exists := unknownRepos[bc.Repository]
		if !exists {
			rs = &RepositoryStatus{Name: bc.Repository}
			unknownRepos[bc.Repository] = rs
		}
		bs := BranchStatus{
			Branch:    bc.Branch,
			IsMain:    isMainBranch(bc.Branch),
			Checks:    bc.Checks,
			HasChecks: len(bc.Checks) > 0,
			CommitSHA: bc.CommitSHA,
			CommitURL: bc.CommitURL,
		}
		computeBranchCounts(&bs)
		rs.Branches = append(rs.Branches, bs)
	}
	// Collect and sort unknown repos alphabetically.
	var unknownNames []string
	for name := range unknownRepos {
		unknownNames = append(unknownNames, name)
	}
	sort.Strings(unknownNames)
	for _, name := range unknownNames {
		repositories = append(repositories, unknownRepos[name])
	}

	return repositories
}

func (h *Handler) BuildStatus(w http.ResponseWriter, r *http.Request) {
	branchChecks, lastUpdate := h.storage.GetBranchChecks()
	log.Printf("Serving /builds - Branch checks: %d", len(branchChecks))

	repositories := groupByRepository(branchChecks, h.repoConfig)

	repo := sanitizeRepo(r.URL.Query().Get("repo"), h.repoNames)
	if repo != "" {
		var filtered []*RepositoryStatus
		for _, r := range repositories {
			if r.Name == repo {
				filtered = append(filtered, r)
			}
		}
		repositories = filtered
	}

	// Compute staleness (skip on cold start)
	var staleMap map[string]bool
	var staleList []string
	if !lastUpdate.IsZero() {
		repoTimes := h.storage.RepoFetchTimes("checks")
		threshold := h.fetchIntervals.Actions * 3
		staleMap = staleRepos(repoTimes, threshold, h.repoNames)
		staleList = sortedKeys(staleMap)
	}
	if staleMap == nil {
		staleMap = make(map[string]bool)
	}
	for _, r := range repositories {
		r.Stale = staleMap[r.Name]
	}

	data := struct {
		PageID        string
		Organization  string
		Version       string
		Repositories  []*RepositoryStatus
		LastUpdate    time.Time
		Repo          string
		RepoNames     []string
		StaleRepos    map[string]bool
		StaleRepoList []string
	}{
		PageID:        "builds",
		Organization:  h.organization,
		Version:       h.version,
		Repositories:  repositories,
		LastUpdate:    lastUpdate,
		Repo:          repo,
		RepoNames:     h.repoNames,
		StaleRepos:    staleMap,
		StaleRepoList: staleList,
	}

	renderTemplate(w, h.buildTemplate, "base", data)
}

func (h *Handler) BuildsDashboard(w http.ResponseWriter, r *http.Request) {
	branchChecks, lastUpdate := h.storage.GetBranchChecks()
	log.Printf("Serving /builds-dashboard - Branch checks: %d", len(branchChecks))

	repositories := groupByRepository(branchChecks, h.repoConfig)

	// Ensure all configured repos appear, even without data
	repoSet := make(map[string]bool, len(repositories))
	for _, repo := range repositories {
		repoSet[repo.Name] = true
	}
	for _, rc := range h.repoConfig {
		if !repoSet[rc.Name] {
			rs := &RepositoryStatus{Name: rc.Name}
			for _, branch := range rc.Branches {
				rs.Branches = append(rs.Branches, BranchStatus{
					Branch: branch,
					IsMain: isMainBranch(branch),
					Checks: []github.Check{},
				})
			}
			repositories = append(repositories, rs)
		}
	}
	sortByConfigOrder(repositories, h.repoNames)

	// Compute staleness (skip on cold start)
	var staleMap map[string]bool
	var staleList []string
	if !lastUpdate.IsZero() {
		repoTimes := h.storage.RepoFetchTimes("checks")
		threshold := h.fetchIntervals.Actions * 3
		staleMap = staleRepos(repoTimes, threshold, h.repoNames)
		staleList = sortedKeys(staleMap)
	}
	if staleMap == nil {
		staleMap = make(map[string]bool)
	}
	for _, repo := range repositories {
		repo.Stale = staleMap[repo.Name]
	}

	data := struct {
		Organization  string
		Repositories  []*RepositoryStatus
		LastUpdate    time.Time
		StaleRepos    map[string]bool
		StaleRepoList []string
	}{
		Organization:  h.organization,
		Repositories:  repositories,
		LastUpdate:    lastUpdate,
		StaleRepos:    staleMap,
		StaleRepoList: staleList,
	}

	renderTemplate(w, h.dashboardTemplate, "builds_dashboard.html", data)
}

func computeBranchCounts(bs *BranchStatus) {
	for _, check := range bs.Checks {
		switch {
		case check.Status == "in_progress" || check.Status == "queued" || check.Status == "waiting" || check.Status == "pending":
			bs.RunningCount++
		case check.Conclusion == "failure" || check.Conclusion == "timed_out" || check.Conclusion == "action_required" || check.Conclusion == "cancelled":
			bs.FailureCount++
		case check.Conclusion == "success":
			bs.SuccessCount++
		}
	}
	switch {
	case bs.RunningCount > 0:
		bs.OverallStatus = "Running"
		bs.StatusClass = "status-running"
	case bs.FailureCount > 0:
		bs.OverallStatus = "Failed"
		bs.StatusClass = "status-failure"
	case bs.SuccessCount > 0:
		bs.OverallStatus = "Passed"
		bs.StatusClass = "status-success"
	default:
		bs.OverallStatus = "Unknown"
		bs.StatusClass = "status-neutral"
	}
}

func isMainBranch(branch string) bool {
	return branch == "main" || branch == "master"
}

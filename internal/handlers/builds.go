package handlers

import (
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/github"
)

type RepositoryStatus struct {
	Name          string
	MainBranch    BranchStatus
	DevelopBranch BranchStatus
}

type BranchStatus struct {
	Branch        string
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

// groupByRepository groups branch checks into sorted repository statuses.
func groupByRepository(branchChecks []github.BranchCheck, repoNames []string) []*RepositoryStatus {
	repoMap := make(map[string]*RepositoryStatus)

	for _, branchCheck := range branchChecks {
		if _, exists := repoMap[branchCheck.Repository]; !exists {
			repoMap[branchCheck.Repository] = &RepositoryStatus{
				Name: branchCheck.Repository,
				MainBranch: BranchStatus{
					Branch: getMainBranch(branchCheck.Repository, branchChecks),
					Checks: []github.Check{},
				},
				DevelopBranch: BranchStatus{
					Branch: "develop",
					Checks: []github.Check{},
				},
			}
		}

		repo := repoMap[branchCheck.Repository]

		if isMainBranch(branchCheck.Branch) {
			repo.MainBranch.Checks = branchCheck.Checks
			repo.MainBranch.HasChecks = len(branchCheck.Checks) > 0
			repo.MainBranch.CommitSHA = branchCheck.CommitSHA
			repo.MainBranch.CommitURL = branchCheck.CommitURL
			computeBranchCounts(&repo.MainBranch)
		} else if branchCheck.Branch == "develop" {
			repo.DevelopBranch.Checks = branchCheck.Checks
			repo.DevelopBranch.HasChecks = len(branchCheck.Checks) > 0
			repo.DevelopBranch.CommitSHA = branchCheck.CommitSHA
			repo.DevelopBranch.CommitURL = branchCheck.CommitURL
			computeBranchCounts(&repo.DevelopBranch)
		}
	}

	var repositories []*RepositoryStatus
	for _, repo := range repoMap {
		repositories = append(repositories, repo)
	}

	sortByConfigOrder(repositories, repoNames)

	return repositories
}

func (h *Handler) BuildStatus(w http.ResponseWriter, r *http.Request) {
	branchChecks, lastUpdate := h.storage.GetBranchChecks()
	log.Printf("Serving /builds - Branch checks: %d", len(branchChecks))

	repositories := groupByRepository(branchChecks, h.repoNames)

	data := struct {
		PageID       string
		Organization string
		Version      string
		Repositories []*RepositoryStatus
		LastUpdate   time.Time
	}{
		PageID:       "builds",
		Organization: h.organization,
		Version:      h.version,
		Repositories: repositories,
		LastUpdate:   lastUpdate,
	}

	renderTemplate(w, h.buildTemplate, "base", data)
}

func (h *Handler) BuildsDashboard(w http.ResponseWriter, r *http.Request) {
	branchChecks, lastUpdate := h.storage.GetBranchChecks()
	log.Printf("Serving /builds-dashboard - Branch checks: %d", len(branchChecks))

	repositories := groupByRepository(branchChecks, h.repoNames)

	// Ensure all configured repos appear, even without data
	repoSet := make(map[string]bool, len(repositories))
	for _, repo := range repositories {
		repoSet[repo.Name] = true
	}
	for _, name := range h.repoNames {
		if !repoSet[name] {
			repositories = append(repositories, &RepositoryStatus{
				Name:          name,
				MainBranch:    BranchStatus{Branch: "master", Checks: []github.Check{}},
				DevelopBranch: BranchStatus{Branch: "develop", Checks: []github.Check{}},
			})
		}
	}
	sortByConfigOrder(repositories, h.repoNames)

	data := struct {
		Organization string
		Repositories []*RepositoryStatus
		LastUpdate   time.Time
	}{
		Organization: h.organization,
		Repositories: repositories,
		LastUpdate:   lastUpdate,
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

func getMainBranch(repo string, checks []github.BranchCheck) string {
	for _, check := range checks {
		if check.Repository == repo && (check.Branch == "main" || check.Branch == "master") {
			return check.Branch
		}
	}
	return "main"
}

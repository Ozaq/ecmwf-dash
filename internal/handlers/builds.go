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

func (h *Handler) BuildStatus(w http.ResponseWriter, r *http.Request) {
	branchChecks, lastUpdate := h.storage.GetBranchChecks()
	log.Printf("Serving /builds - Branch checks: %d", len(branchChecks))

	cssFiles := h.cssFiles

	// Group checks by repository and branch
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

		// Determine if this is main/master or develop
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

	// Convert map to slice for template
	var repositories []*RepositoryStatus
	for _, repo := range repoMap {
		repositories = append(repositories, repo)
	}

	// Sort repositories by name
	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].Name < repositories[j].Name
	})

	data := struct {
		PageID       string
		Organization string
		Repositories []*RepositoryStatus
		LastUpdate   time.Time
		CSSFile      string
		CSSFiles     []CSSOption
	}{
		PageID:       "builds",
		Organization: h.organization,
		Repositories: repositories,
		LastUpdate:   lastUpdate,
		CSSFile:      h.cssFile,
		CSSFiles:     cssFiles,
	}

	renderTemplate(w, h.buildTemplate, "base", data)
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

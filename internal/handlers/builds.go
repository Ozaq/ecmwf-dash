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
	Branch    string
	Checks    []github.Check
	HasChecks bool
	CommitSHA string
	CommitURL string
}

func (h *Handler) BuildStatus(w http.ResponseWriter, r *http.Request) {
	branchChecks, lastUpdate := h.storage.GetBranchChecks()
	log.Printf("Serving /builds - Branch checks: %d", len(branchChecks))

	// Get available CSS files
	cssFiles := getAvailableCSS()

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
		} else if branchCheck.Branch == "develop" {
			repo.DevelopBranch.Checks = branchCheck.Checks
			repo.DevelopBranch.HasChecks = len(branchCheck.Checks) > 0
			repo.DevelopBranch.CommitSHA = branchCheck.CommitSHA
			repo.DevelopBranch.CommitURL = branchCheck.CommitURL
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
		Repositories []*RepositoryStatus
		LastUpdate   time.Time
		CSSFile      string
		CSSFiles     []string
	}{
		Repositories: repositories,
		LastUpdate:   lastUpdate,
		CSSFile:      h.cssFile,
		CSSFiles:     cssFiles,
	}

	if h.buildTemplate == nil {
		log.Printf("ERROR: buildTemplate is nil!")
		http.Error(w, "Template not initialized", http.StatusInternalServerError)
		return
	}

	if err := h.buildTemplate.Execute(w, data); err != nil {
		log.Printf("Error executing build template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func isMainBranch(branch string) bool {
	return branch == "main" || branch == "master"
}

func getMainBranch(repo string, checks []github.BranchCheck) string {
	// Check if repository has main or master branch
	for _, check := range checks {
		if check.Repository == repo && (check.Branch == "main" || check.Branch == "master") {
			return check.Branch
		}
	}
	return "main" // default
}

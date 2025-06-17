package handlers

import (
    "log"
    "net/http"
    "sort"
    "time"
    
    "github.com/ozaq/ecmwf-dash/internal/github"
)

type RepositoryStatus struct {
    Name        string
    MainBranch  BranchStatus
    DevelopBranch BranchStatus
}

type BranchStatus struct {
    Branch      string
    Runs        []github.WorkflowRun
    LatestRun   *github.WorkflowRun
    HasRuns     bool
}

func (h *Handler) BuildStatus(w http.ResponseWriter, r *http.Request) {
    runs, lastUpdate := h.storage.GetWorkflowRuns()
    log.Printf("Serving /builds - Workflow runs: %d", len(runs))

    // Get available CSS files
    cssFiles := getAvailableCSS()

    // Group runs by repository and branch
    repoMap := make(map[string]*RepositoryStatus)
    
    for _, run := range runs {
        if _, exists := repoMap[run.Repository]; !exists {
            repoMap[run.Repository] = &RepositoryStatus{
                Name: run.Repository,
                MainBranch: BranchStatus{
                    Branch: getMainBranch(run.Repository, runs),
                    Runs:   []github.WorkflowRun{},
                },
                DevelopBranch: BranchStatus{
                    Branch: "develop",
                    Runs:   []github.WorkflowRun{},
                },
            }
        }
        
        repo := repoMap[run.Repository]
        
        // Determine if this is main/master or develop
        if isMainBranch(run.Branch) {
            repo.MainBranch.Runs = append(repo.MainBranch.Runs, run)
            repo.MainBranch.HasRuns = true
            if repo.MainBranch.LatestRun == nil || run.CreatedAt.After(repo.MainBranch.LatestRun.CreatedAt) {
                repo.MainBranch.LatestRun = &run
            }
        } else if run.Branch == "develop" {
            repo.DevelopBranch.Runs = append(repo.DevelopBranch.Runs, run)
            repo.DevelopBranch.HasRuns = true
            if repo.DevelopBranch.LatestRun == nil || run.CreatedAt.After(repo.DevelopBranch.LatestRun.CreatedAt) {
                repo.DevelopBranch.LatestRun = &run
            }
        }
    }

    // Convert map to slice for template
    var repositories []*RepositoryStatus
    for _, repo := range repoMap {
        // Sort runs by creation time (newest first)
        sort.Slice(repo.MainBranch.Runs, func(i, j int) bool {
            return repo.MainBranch.Runs[i].CreatedAt.After(repo.MainBranch.Runs[j].CreatedAt)
        })
        sort.Slice(repo.DevelopBranch.Runs, func(i, j int) bool {
            return repo.DevelopBranch.Runs[i].CreatedAt.After(repo.DevelopBranch.Runs[j].CreatedAt)
        })
        
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

func getMainBranch(repo string, runs []github.WorkflowRun) string {
    // Check if repository has main or master branch
    for _, run := range runs {
        if run.Repository == repo && (run.Branch == "main" || run.Branch == "master") {
            return run.Branch
        }
    }
    return "main" // default
}

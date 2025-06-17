package fetcher

import (
    "context"
    "log"
    "time"

    "github.com/ozaq/ecmwf-dash/internal/config"
    "github.com/ozaq/ecmwf-dash/internal/github"
    "github.com/ozaq/ecmwf-dash/internal/storage"
)

type Fetcher struct {
    cfg     *config.Config
    gh      *github.Client
    storage *storage.Memory
}

func New(cfg *config.Config, gh *github.Client, storage *storage.Memory) *Fetcher {
    return &Fetcher{
        cfg:     cfg,
        gh:      gh,
        storage: storage,
    }
}

func (f *Fetcher) Start(ctx context.Context) {
    go f.runIssuesFetcher(ctx)
    go f.runPRsFetcher(ctx)
}

func (f *Fetcher) runIssuesFetcher(ctx context.Context) {
    f.fetchIssues(ctx)
    ticker := time.NewTicker(f.cfg.FetchIntervals.Issues)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            f.fetchIssues(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (f *Fetcher) fetchIssues(ctx context.Context) {
    log.Printf("Fetching issues for %s", f.cfg.GitHub.Organization)
    
    issues, err := f.gh.FetchIssues(ctx, f.cfg.GitHub.Organization, f.cfg.GitHub.Repositories)
    if err != nil {
        log.Printf("Error fetching issues: %v", err)
        return
    }

    f.storage.SetIssues(issues)
    log.Printf("Fetched %d issues", len(issues))
}

func (f *Fetcher) runPRsFetcher(ctx context.Context) {
    f.fetchPullRequests(ctx)
    ticker := time.NewTicker(f.cfg.FetchIntervals.PullRequests)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            f.fetchPullRequests(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (f *Fetcher) fetchPullRequests(ctx context.Context) {
    log.Printf("Fetching pull requests for %s", f.cfg.GitHub.Organization)
    
    prs, err := f.gh.FetchPullRequests(ctx, f.cfg.GitHub.Organization, f.cfg.GitHub.Repositories)
    if err != nil {
        log.Printf("Error fetching pull requests: %v", err)
        return
    }

    f.storage.SetPullRequests(prs)
    log.Printf("Fetched %d pull requests", len(prs))
}

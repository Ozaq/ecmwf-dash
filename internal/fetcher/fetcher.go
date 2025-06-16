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
    // Initial fetch
    f.fetchIssues(ctx)

    // Start periodic fetches
    go f.runIssuesFetcher(ctx)
}

func (f *Fetcher) runIssuesFetcher(ctx context.Context) {
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

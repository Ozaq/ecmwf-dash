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
	storage storage.Store
}

func New(cfg *config.Config, gh *github.Client, store storage.Store) *Fetcher {
	return &Fetcher{
		cfg:     cfg,
		gh:      gh,
		storage: store,
	}
}

func (f *Fetcher) Start(ctx context.Context) {
	go f.runIssuesFetcher(ctx)
	go f.runPRsFetcher(ctx)
	go f.runBranchChecksFetcher(ctx)
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

	issues, rate, err := f.gh.FetchIssues(ctx, f.cfg.GitHub.Organization, f.cfg.GitHub.Repositories)
	if err != nil {
		log.Printf("Error fetching issues: %v", err)
		return
	}

	f.storage.SetIssues(issues)
	log.Printf("Fetched %d issues", len(issues))
	f.gh.LogRate(rate)
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

	prs, rate, err := f.gh.FetchPullRequests(ctx, f.cfg.GitHub.Organization, f.cfg.GitHub.Repositories)
	if err != nil {
		log.Printf("Error fetching pull requests: %v", err)
		return
	}

	f.storage.SetPullRequests(prs)
	log.Printf("Fetched %d pull requests", len(prs))
	f.gh.LogRate(rate)
}

func (f *Fetcher) runBranchChecksFetcher(ctx context.Context) {
	f.fetchBranchChecks(ctx)
	ticker := time.NewTicker(f.cfg.FetchIntervals.Actions)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.fetchBranchChecks(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (f *Fetcher) fetchBranchChecks(ctx context.Context) {
	log.Printf("Fetching branch checks for %s", f.cfg.GitHub.Organization)

	checks, rate, err := f.gh.FetchBranchChecks(ctx, f.cfg.GitHub.Organization, f.cfg.GitHub.Repositories)
	if err != nil {
		log.Printf("Error fetching branch checks: %v", err)
		return
	}

	f.storage.SetBranchChecks(checks)
	log.Printf("Fetched %d branch checks", len(checks))
	f.gh.LogRate(rate)
}

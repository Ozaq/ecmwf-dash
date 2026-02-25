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

	result := f.gh.FetchIssues(ctx, f.cfg.GitHub.Organization, f.cfg.GitHub.Repositories)
	if result.Err != nil {
		log.Printf("Error fetching issues: %v", result.Err)
		return
	}

	if len(result.FailedRepos) > 0 {
		log.Printf("Issues: partial failure for repos: %v", result.FailedRepos)
	}
	f.storage.MergeIssues(result.Issues, result.FailedRepos, result.SucceededRepos)
	log.Printf("Fetched %d issues", len(result.Issues))
	f.gh.LogRate(result.Rate)
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

	result := f.gh.FetchPullRequests(ctx, f.cfg.GitHub.Organization, f.cfg.GitHub.Repositories)
	if result.Err != nil {
		log.Printf("Error fetching pull requests: %v", result.Err)
		return
	}

	if len(result.FailedRepos) > 0 {
		log.Printf("Pull requests: partial failure for repos: %v", result.FailedRepos)
	}
	f.storage.MergePullRequests(result.PullRequests, result.FailedRepos, result.SucceededRepos)
	log.Printf("Fetched %d pull requests", len(result.PullRequests))
	f.gh.LogRate(result.Rate)
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

	result := f.gh.FetchBranchChecks(ctx, f.cfg.GitHub.Organization, f.cfg.GitHub.Repositories)
	if result.Err != nil {
		log.Printf("Error fetching branch checks: %v", result.Err)
		return
	}

	if len(result.FailedRepos) > 0 {
		log.Printf("Branch checks: partial failure for repos: %v", result.FailedRepos)
	}
	f.storage.MergeBranchChecks(result.BranchChecks, result.FailedRepos, result.SucceededRepos)
	log.Printf("Fetched %d branch checks", len(result.BranchChecks))
	f.gh.LogRate(result.Rate)
}

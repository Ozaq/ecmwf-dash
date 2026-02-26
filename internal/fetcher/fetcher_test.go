package fetcher

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/config"
	"github.com/ozaq/ecmwf-dash/internal/github"
	"github.com/ozaq/ecmwf-dash/internal/storage"
)

// --- Mock GitHubFetcher ---

type mockGitHubFetcher struct {
	mu sync.Mutex

	issuesResult github.IssuesFetchResult
	prsResult    github.PRsFetchResult
	checksResult github.ChecksFetchResult

	fetchIssuesCalls int
	fetchPRsCalls    int
	fetchChecksCalls int
	logRateCalls     int
	lastRateLogged   github.RateInfo

	// Optional: capture args from most recent call
	lastOrg   string
	lastRepos []config.RepositoryConfig
}

func (m *mockGitHubFetcher) FetchIssues(_ context.Context, org string, repos []config.RepositoryConfig) github.IssuesFetchResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fetchIssuesCalls++
	m.lastOrg = org
	m.lastRepos = repos
	return m.issuesResult
}

func (m *mockGitHubFetcher) FetchPullRequests(_ context.Context, org string, repos []config.RepositoryConfig) github.PRsFetchResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fetchPRsCalls++
	m.lastOrg = org
	m.lastRepos = repos
	return m.prsResult
}

func (m *mockGitHubFetcher) FetchBranchChecks(_ context.Context, org string, repos []config.RepositoryConfig) github.ChecksFetchResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fetchChecksCalls++
	m.lastOrg = org
	m.lastRepos = repos
	return m.checksResult
}

func (m *mockGitHubFetcher) LogRate(r github.RateInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logRateCalls++
	m.lastRateLogged = r
}

// --- Mock Store (wraps real Memory for merge verification) ---

type mockStore struct {
	storage.Store // embed for unimplemented methods

	mu                 sync.Mutex
	mergeIssuesCalls   int
	mergePRsCalls      int
	mergeChecksCalls   int
	lastIssues         []github.Issue
	lastPRs            []github.PullRequest
	lastChecks         []github.BranchCheck
	lastFailedRepos    []string
	lastSucceededRepos []string
}

func (m *mockStore) MergeIssues(issues []github.Issue, failedRepos, succeededRepos []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mergeIssuesCalls++
	m.lastIssues = issues
	m.lastFailedRepos = failedRepos
	m.lastSucceededRepos = succeededRepos
}

func (m *mockStore) MergePullRequests(prs []github.PullRequest, failedRepos, succeededRepos []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mergePRsCalls++
	m.lastPRs = prs
	m.lastFailedRepos = failedRepos
	m.lastSucceededRepos = succeededRepos
}

func (m *mockStore) MergeBranchChecks(checks []github.BranchCheck, failedRepos, succeededRepos []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mergeChecksCalls++
	m.lastChecks = checks
	m.lastFailedRepos = failedRepos
	m.lastSucceededRepos = succeededRepos
}

// --- Helpers ---

func testConfig() *config.Config {
	return &config.Config{
		GitHub: config.GitHubConfig{
			Organization: "testorg",
			Repositories: []config.RepositoryConfig{
				{Name: "testrepo", Branches: []string{"main"}},
			},
		},
		FetchIntervals: config.FetchIntervalsConfig{
			Issues:       100 * time.Millisecond,
			PullRequests: 100 * time.Millisecond,
			Actions:      100 * time.Millisecond,
		},
	}
}

func testRate() github.RateInfo {
	return github.RateInfo{Remaining: 4500, Limit: 5000, Reset: time.Now().Add(time.Hour)}
}

// --- Tests ---

func TestNew_ReturnsConfiguredFetcher(t *testing.T) {
	cfg := testConfig()
	gh := &mockGitHubFetcher{}
	store := &mockStore{}

	f := New(cfg, gh, store)

	if f.cfg != cfg {
		t.Error("New() did not wire up config")
	}
	if f.gh != gh {
		t.Error("New() did not wire up GitHubFetcher")
	}
	if f.storage != store {
		t.Error("New() did not wire up Store")
	}
}

func TestFetchIssues_Success(t *testing.T) {
	rate := testRate()
	gh := &mockGitHubFetcher{
		issuesResult: github.IssuesFetchResult{
			Issues:         []github.Issue{{Repository: "testrepo", Number: 1, Title: "Bug"}},
			SucceededRepos: []string{"testrepo"},
			Rate:           rate,
		},
	}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	f.fetchIssues(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()
	gh.mu.Lock()
	defer gh.mu.Unlock()

	if store.mergeIssuesCalls != 1 {
		t.Fatalf("expected 1 MergeIssues call, got %d", store.mergeIssuesCalls)
	}
	if len(store.lastIssues) != 1 || store.lastIssues[0].Title != "Bug" {
		t.Errorf("unexpected issues: %v", store.lastIssues)
	}
	if len(store.lastSucceededRepos) != 1 || store.lastSucceededRepos[0] != "testrepo" {
		t.Errorf("unexpected succeeded repos: %v", store.lastSucceededRepos)
	}
	if gh.logRateCalls != 1 {
		t.Errorf("expected 1 LogRate call, got %d", gh.logRateCalls)
	}
}

func TestFetchIssues_TotalFailure(t *testing.T) {
	gh := &mockGitHubFetcher{
		issuesResult: github.IssuesFetchResult{
			Err: fmt.Errorf("all 1 repos failed"),
		},
	}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	f.fetchIssues(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()
	gh.mu.Lock()
	defer gh.mu.Unlock()

	if store.mergeIssuesCalls != 0 {
		t.Errorf("expected 0 MergeIssues calls on total failure, got %d", store.mergeIssuesCalls)
	}
	if gh.logRateCalls != 0 {
		t.Errorf("expected 0 LogRate calls on total failure, got %d", gh.logRateCalls)
	}
}

func TestFetchIssues_PartialFailure(t *testing.T) {
	gh := &mockGitHubFetcher{
		issuesResult: github.IssuesFetchResult{
			Issues:         []github.Issue{{Repository: "repo-a", Number: 1}},
			SucceededRepos: []string{"repo-a"},
			FailedRepos:    []string{"repo-b"},
			Rate:           testRate(),
		},
	}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	f.fetchIssues(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()
	gh.mu.Lock()
	defer gh.mu.Unlock()

	if store.mergeIssuesCalls != 1 {
		t.Fatalf("expected 1 MergeIssues call on partial failure, got %d", store.mergeIssuesCalls)
	}
	if len(store.lastFailedRepos) != 1 || store.lastFailedRepos[0] != "repo-b" {
		t.Errorf("expected failed repos [repo-b], got %v", store.lastFailedRepos)
	}
	if len(store.lastSucceededRepos) != 1 || store.lastSucceededRepos[0] != "repo-a" {
		t.Errorf("expected succeeded repos [repo-a], got %v", store.lastSucceededRepos)
	}
	if gh.logRateCalls != 1 {
		t.Errorf("expected 1 LogRate call on partial failure, got %d", gh.logRateCalls)
	}
}

func TestFetchPullRequests_Success(t *testing.T) {
	gh := &mockGitHubFetcher{
		prsResult: github.PRsFetchResult{
			PullRequests:   []github.PullRequest{{Repository: "testrepo", Number: 10, Title: "Add feature"}},
			SucceededRepos: []string{"testrepo"},
			Rate:           testRate(),
		},
	}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	f.fetchPullRequests(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()

	if store.mergePRsCalls != 1 {
		t.Fatalf("expected 1 MergePullRequests call, got %d", store.mergePRsCalls)
	}
	if len(store.lastPRs) != 1 || store.lastPRs[0].Title != "Add feature" {
		t.Errorf("unexpected PRs: %v", store.lastPRs)
	}
}

func TestFetchPullRequests_TotalFailure(t *testing.T) {
	gh := &mockGitHubFetcher{
		prsResult: github.PRsFetchResult{Err: fmt.Errorf("all repos failed")},
	}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	f.fetchPullRequests(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()

	if store.mergePRsCalls != 0 {
		t.Errorf("expected 0 MergePullRequests calls on total failure, got %d", store.mergePRsCalls)
	}
}

func TestFetchBranchChecks_Success(t *testing.T) {
	gh := &mockGitHubFetcher{
		checksResult: github.ChecksFetchResult{
			BranchChecks: []github.BranchCheck{
				{
					Repository: "testrepo",
					Branch:     "main",
					CommitSHA:  "abc123",
					Checks: []github.Check{
						{Name: "ci/test", Status: "completed", Conclusion: "success"},
						{Name: "ci/lint", Status: "completed", Conclusion: "failure"},
					},
				},
			},
			SucceededRepos: []string{"testrepo"},
			Rate:           testRate(),
		},
	}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	f.fetchBranchChecks(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()

	if store.mergeChecksCalls != 1 {
		t.Fatalf("expected 1 MergeBranchChecks call, got %d", store.mergeChecksCalls)
	}
	if len(store.lastChecks) != 1 {
		t.Fatalf("expected 1 branch check, got %d", len(store.lastChecks))
	}
	if len(store.lastChecks[0].Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(store.lastChecks[0].Checks))
	}
}

func TestFetchBranchChecks_TotalFailure(t *testing.T) {
	gh := &mockGitHubFetcher{
		checksResult: github.ChecksFetchResult{Err: fmt.Errorf("all repos failed")},
	}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	f.fetchBranchChecks(context.Background())

	store.mu.Lock()
	defer store.mu.Unlock()

	if store.mergeChecksCalls != 0 {
		t.Errorf("expected 0 MergeBranchChecks calls on total failure, got %d", store.mergeChecksCalls)
	}
}

func TestRunIssuesFetcher_ContextCancellation(t *testing.T) {
	gh := &mockGitHubFetcher{}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		f.runIssuesFetcher(ctx)
		close(done)
	}()

	// Let the initial fetch run
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// goroutine exited cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("runIssuesFetcher did not exit after context cancellation")
	}

	gh.mu.Lock()
	defer gh.mu.Unlock()

	if gh.fetchIssuesCalls < 1 {
		t.Errorf("expected at least 1 FetchIssues call, got %d", gh.fetchIssuesCalls)
	}
}

func TestRunPRsFetcher_ContextCancellation(t *testing.T) {
	gh := &mockGitHubFetcher{}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		f.runPRsFetcher(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runPRsFetcher did not exit after context cancellation")
	}
}

func TestRunBranchChecksFetcher_ContextCancellation(t *testing.T) {
	gh := &mockGitHubFetcher{}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		f.runBranchChecksFetcher(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runBranchChecksFetcher did not exit after context cancellation")
	}
}

func TestStart_LaunchesAllFetchers(t *testing.T) {
	gh := &mockGitHubFetcher{}
	store := &mockStore{}
	f := New(testConfig(), gh, store)

	ctx, cancel := context.WithCancel(context.Background())

	f.Start(ctx)

	// Wait for initial fetch + at least one tick (interval is 100ms)
	time.Sleep(250 * time.Millisecond)
	cancel()
	// Let goroutines exit
	time.Sleep(50 * time.Millisecond)

	gh.mu.Lock()
	defer gh.mu.Unlock()

	if gh.fetchIssuesCalls < 1 {
		t.Errorf("expected at least 1 FetchIssues call, got %d", gh.fetchIssuesCalls)
	}
	if gh.fetchPRsCalls < 1 {
		t.Errorf("expected at least 1 FetchPullRequests call, got %d", gh.fetchPRsCalls)
	}
	if gh.fetchChecksCalls < 1 {
		t.Errorf("expected at least 1 FetchBranchChecks call, got %d", gh.fetchChecksCalls)
	}
}

func TestFetchIssues_UsesConfigOrgAndRepos(t *testing.T) {
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			Organization: "myorg",
			Repositories: []config.RepositoryConfig{
				{Name: "repo-a", Branches: []string{"main", "develop"}},
				{Name: "repo-b", Branches: []string{"main"}},
			},
		},
		FetchIntervals: config.FetchIntervalsConfig{
			Issues:       100 * time.Millisecond,
			PullRequests: 100 * time.Millisecond,
			Actions:      100 * time.Millisecond,
		},
	}

	gh := &mockGitHubFetcher{}
	store := &mockStore{}
	f := New(cfg, gh, store)

	f.fetchIssues(context.Background())

	gh.mu.Lock()
	defer gh.mu.Unlock()

	if gh.lastOrg != "myorg" {
		t.Errorf("expected org 'myorg', got %q", gh.lastOrg)
	}
	if len(gh.lastRepos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(gh.lastRepos))
	}
	if gh.lastRepos[0].Name != "repo-a" {
		t.Errorf("expected first repo 'repo-a', got %q", gh.lastRepos[0].Name)
	}
}

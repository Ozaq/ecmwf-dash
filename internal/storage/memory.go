package storage

import (
	"sync"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/github"
)

type Memory struct {
	mu sync.RWMutex

	issues     []github.Issue
	issuesTime time.Time

	pullRequests []github.PullRequest
	prsTime      time.Time

	branchChecks     []github.BranchCheck
	branchChecksTime time.Time

	// Per-repo last-success timestamps
	issueRepoTimes map[string]time.Time
	prRepoTimes    map[string]time.Time
	checkRepoTimes map[string]time.Time
}

func New() *Memory {
	return &Memory{
		issueRepoTimes: make(map[string]time.Time),
		prRepoTimes:    make(map[string]time.Time),
		checkRepoTimes: make(map[string]time.Time),
	}
}

func (m *Memory) SetIssues(issues []github.Issue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.issues = deepCopyIssues(issues)
	now := time.Now()
	m.issuesTime = now
	m.updateRepoTimes(m.issueRepoTimes, repoNamesFromIssues(issues), now)
}

func (m *Memory) GetIssues() ([]github.Issue, time.Time) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return deepCopyIssues(m.issues), m.issuesTime
}

func (m *Memory) SetPullRequests(prs []github.PullRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pullRequests = deepCopyPullRequests(prs)
	now := time.Now()
	m.prsTime = now
	m.updateRepoTimes(m.prRepoTimes, repoNamesFromPRs(prs), now)
}

func (m *Memory) GetPullRequests() ([]github.PullRequest, time.Time) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return deepCopyPullRequests(m.pullRequests), m.prsTime
}

func (m *Memory) SetBranchChecks(checks []github.BranchCheck) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.branchChecks = deepCopyBranchChecks(checks)
	now := time.Now()
	m.branchChecksTime = now
	m.updateRepoTimes(m.checkRepoTimes, repoNamesFromChecks(checks), now)
}

func (m *Memory) GetBranchChecks() ([]github.BranchCheck, time.Time) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return deepCopyBranchChecks(m.branchChecks), m.branchChecksTime
}

func (m *Memory) LastFetchTimes() (issues, prs, checks time.Time) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.issuesTime, m.prsTime, m.branchChecksTime
}

// MergeIssues replaces data for successfully fetched repos while preserving
// old data for repos in failedRepos. succeededRepos is used for per-repo
// timestamp updates (handles repos with 0 items that wouldn't appear in data).
func (m *Memory) MergeIssues(issues []github.Issue, failedRepos, succeededRepos []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	failed := toSet(failedRepos)
	merged := keepByRepo(m.issues, failed)
	merged = append(merged, deepCopyIssues(issues)...)
	m.issues = merged

	if len(succeededRepos) > 0 {
		now := time.Now()
		m.issuesTime = now
		for _, name := range succeededRepos {
			m.issueRepoTimes[name] = now
		}
	}
}

// MergePullRequests replaces data for successfully fetched repos while preserving
// old data for repos in failedRepos.
func (m *Memory) MergePullRequests(prs []github.PullRequest, failedRepos, succeededRepos []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	failed := toSet(failedRepos)
	merged := keepPRsByRepo(m.pullRequests, failed)
	merged = append(merged, deepCopyPullRequests(prs)...)
	m.pullRequests = merged

	if len(succeededRepos) > 0 {
		now := time.Now()
		m.prsTime = now
		for _, name := range succeededRepos {
			m.prRepoTimes[name] = now
		}
	}
}

// MergeBranchChecks replaces data for successfully fetched repos while preserving
// old data for repos in failedRepos.
func (m *Memory) MergeBranchChecks(checks []github.BranchCheck, failedRepos, succeededRepos []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	failed := toSet(failedRepos)
	merged := keepChecksByRepo(m.branchChecks, failed)
	merged = append(merged, deepCopyBranchChecks(checks)...)
	m.branchChecks = merged

	if len(succeededRepos) > 0 {
		now := time.Now()
		m.branchChecksTime = now
		for _, name := range succeededRepos {
			m.checkRepoTimes[name] = now
		}
	}
}

// RepoFetchTimes returns a copy of per-repo last-success timestamps for the given category.
func (m *Memory) RepoFetchTimes(category string) map[string]time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var src map[string]time.Time
	switch category {
	case CategoryIssues:
		src = m.issueRepoTimes
	case CategoryPRs:
		src = m.prRepoTimes
	case CategoryChecks:
		src = m.checkRepoTimes
	default:
		return make(map[string]time.Time)
	}

	dst := make(map[string]time.Time, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// updateRepoTimes sets the timestamp for all repos present in the data.
// Called under lock by Set* methods.
func (m *Memory) updateRepoTimes(times map[string]time.Time, repos []string, now time.Time) {
	for _, name := range repos {
		times[name] = now
	}
}

// Helpers to extract unique repo names from data slices.

func repoNamesFromIssues(issues []github.Issue) []string {
	seen := make(map[string]bool)
	var names []string
	for _, issue := range issues {
		if !seen[issue.Repository] {
			seen[issue.Repository] = true
			names = append(names, issue.Repository)
		}
	}
	return names
}

func repoNamesFromPRs(prs []github.PullRequest) []string {
	seen := make(map[string]bool)
	var names []string
	for _, pr := range prs {
		if !seen[pr.Repository] {
			seen[pr.Repository] = true
			names = append(names, pr.Repository)
		}
	}
	return names
}

func repoNamesFromChecks(checks []github.BranchCheck) []string {
	seen := make(map[string]bool)
	var names []string
	for _, bc := range checks {
		if !seen[bc.Repository] {
			seen[bc.Repository] = true
			names = append(names, bc.Repository)
		}
	}
	return names
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

// keepByRepo returns deep copies of issues belonging to repos in the keepSet.
func keepByRepo(issues []github.Issue, keepSet map[string]bool) []github.Issue {
	var kept []github.Issue
	for _, issue := range issues {
		if keepSet[issue.Repository] {
			cp := issue
			cp.Labels = append([]github.Label(nil), issue.Labels...)
			kept = append(kept, cp)
		}
	}
	return kept
}

func keepPRsByRepo(prs []github.PullRequest, keepSet map[string]bool) []github.PullRequest {
	var kept []github.PullRequest
	for _, pr := range prs {
		if keepSet[pr.Repository] {
			cp := pr
			cp.Labels = append([]github.Label(nil), pr.Labels...)
			cp.Checks = append([]github.Check(nil), pr.Checks...)
			cp.Reviewers = append([]github.Reviewer(nil), pr.Reviewers...)
			kept = append(kept, cp)
		}
	}
	return kept
}

func keepChecksByRepo(checks []github.BranchCheck, keepSet map[string]bool) []github.BranchCheck {
	var kept []github.BranchCheck
	for _, bc := range checks {
		if keepSet[bc.Repository] {
			cp := bc
			cp.Checks = append([]github.Check(nil), bc.Checks...)
			kept = append(kept, cp)
		}
	}
	return kept
}

// Deep copy helpers — copy inner slices to prevent shared mutable state.

func deepCopyIssues(src []github.Issue) []github.Issue {
	dst := make([]github.Issue, len(src))
	for i, issue := range src {
		dst[i] = issue
		dst[i].Labels = append([]github.Label(nil), issue.Labels...)
	}
	return dst
}

func deepCopyPullRequests(src []github.PullRequest) []github.PullRequest {
	dst := make([]github.PullRequest, len(src))
	for i, pr := range src {
		dst[i] = pr
		dst[i].Labels = append([]github.Label(nil), pr.Labels...)
		dst[i].Checks = append([]github.Check(nil), pr.Checks...)
		dst[i].Reviewers = append([]github.Reviewer(nil), pr.Reviewers...)
	}
	return dst
}

func deepCopyBranchChecks(src []github.BranchCheck) []github.BranchCheck {
	dst := make([]github.BranchCheck, len(src))
	for i, bc := range src {
		dst[i] = bc
		dst[i].Checks = append([]github.Check(nil), bc.Checks...)
	}
	return dst
}

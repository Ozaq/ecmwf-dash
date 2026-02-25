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
}

func New() *Memory {
	return &Memory{}
}

func (m *Memory) SetIssues(issues []github.Issue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.issues = deepCopyIssues(issues)
	m.issuesTime = time.Now()
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
	m.prsTime = time.Now()
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
	m.branchChecksTime = time.Now()
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

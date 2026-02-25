package storage

import (
	"time"

	"github.com/ozaq/ecmwf-dash/internal/github"
)

// Store defines the interface for data access. All consumers should depend on
// this interface rather than the concrete *Memory type.
type Store interface {
	SetIssues([]github.Issue)
	GetIssues() ([]github.Issue, time.Time)
	SetPullRequests([]github.PullRequest)
	GetPullRequests() ([]github.PullRequest, time.Time)
	SetBranchChecks([]github.BranchCheck)
	GetBranchChecks() ([]github.BranchCheck, time.Time)
	LastFetchTimes() (issues, prs, checks time.Time)

	// Merge methods preserve old data for failed repos, replacing only successful ones.
	// succeededRepos lists repos that were fetched successfully (may have 0 items).
	MergeIssues(issues []github.Issue, failedRepos, succeededRepos []string)
	MergePullRequests(prs []github.PullRequest, failedRepos, succeededRepos []string)
	MergeBranchChecks(checks []github.BranchCheck, failedRepos, succeededRepos []string)

	// RepoFetchTimes returns per-repo last-success timestamps for a category ("issues"|"prs"|"checks").
	RepoFetchTimes(category string) map[string]time.Time
}

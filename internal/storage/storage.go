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
}

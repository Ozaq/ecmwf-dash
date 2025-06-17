package github

import "time"

type Issue struct {
    Repository        string
    Number            int
    Title             string
    URL               string
    Author            string
    AuthorAvatar      string
    AuthorAssociation string // OWNER, MEMBER, COLLABORATOR, CONTRIBUTOR, NONE
    IsExternal        bool   // true if not OWNER/MEMBER/COLLABORATOR
    CreatedAt         time.Time
    UpdatedAt         time.Time
    Labels            []Label
}

type Label struct {
    Name  string
    Color string
}

type PullRequest struct {
    Repository        string
    Number            int
    Title             string
    URL               string
    Author            string
    AuthorAvatar      string
    AuthorAssociation string
    IsExternal        bool
    CreatedAt         time.Time
    UpdatedAt         time.Time
    Labels            []Label
    
    // PR specific fields
    State           string // open, closed, merged
    Draft           bool
    BaseBranch      string
    HeadBranch      string
    ReviewStatus    string // approved, changes_requested, pending
    Reviewers       []Reviewer
    MergeableState  string // clean, blocked, unstable, dirty
    Comments        int
    ReviewComments  int
    Checks          []Check
    
    // Check counts
    ChecksSuccess   int
    ChecksFailure   int
    ChecksRunning   int
}

type Reviewer struct {
    Login    string
    Avatar   string
    State    string // APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING
}

type Check struct {
    Name       string
    Status     string // completed, in_progress, queued
    Conclusion string // success, failure, neutral, cancelled, skipped, timed_out
    URL        string
}
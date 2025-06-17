package storage

import (
    "sync"
    "time"

    "github.com/ozaq/ecmwf-dash/internal/github"
)

type Memory struct {
    mu sync.RWMutex

    issues      []github.Issue
    issuesTime  time.Time
    
    pullRequests []github.PullRequest
    prsTime      time.Time
    
    workflowRuns []github.WorkflowRun
    workflowTime time.Time
}

func New() *Memory {
    return &Memory{}
}

func (m *Memory) SetIssues(issues []github.Issue) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.issues = issues
    m.issuesTime = time.Now()
}

func (m *Memory) GetIssues() ([]github.Issue, time.Time) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.issues, m.issuesTime
}

func (m *Memory) SetPullRequests(prs []github.PullRequest) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.pullRequests = prs
    m.prsTime = time.Now()
}

func (m *Memory) GetPullRequests() ([]github.PullRequest, time.Time) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.pullRequests, m.prsTime
}

func (m *Memory) SetWorkflowRuns(runs []github.WorkflowRun) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.workflowRuns = runs
    m.workflowTime = time.Now()
}

func (m *Memory) GetWorkflowRuns() ([]github.WorkflowRun, time.Time) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.workflowRuns, m.workflowTime
}
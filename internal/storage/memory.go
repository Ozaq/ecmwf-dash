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

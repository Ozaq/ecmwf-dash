package handlers

import (
	"strconv"
	"testing"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/github"
)

func TestSortIssues(t *testing.T) {
	now := time.Now()

	makeIssues := func() []github.Issue {
		return []github.Issue{
			{Repository: "beta", Number: 2, Title: "Banana", Author: "charlie", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{Repository: "alpha", Number: 1, Title: "Apple", Author: "alice", CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now},
			{Repository: "gamma", Number: 3, Title: "Cherry", Author: "bob", CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		}
	}

	tests := []struct {
		name      string
		sortBy    string
		order     string
		wantField func([]github.Issue) []string
		wantOrder []string
	}{
		{"repo_asc", "repo", "asc", issueRepoExtractor, []string{"alpha", "beta", "gamma"}},
		{"repo_desc", "repo", "desc", issueRepoExtractor, []string{"gamma", "beta", "alpha"}},
		{"number_asc", "number", "asc", issueNumberExtractor, []string{"1", "2", "3"}},
		{"number_desc", "number", "desc", issueNumberExtractor, []string{"3", "2", "1"}},
		{"title_asc", "title", "asc", issueTitleExtractor, []string{"Apple", "Banana", "Cherry"}},
		{"title_desc", "title", "desc", issueTitleExtractor, []string{"Cherry", "Banana", "Apple"}},
		{"author_asc", "author", "asc", issueAuthorExtractor, []string{"alice", "bob", "charlie"}},
		{"author_desc", "author", "desc", issueAuthorExtractor, []string{"charlie", "bob", "alice"}},
		// For time fields: input order is [-2h, -3h, -1h] created; [-1h, 0, -2h] updated
		// asc = oldest first, desc = newest first
		{"created_asc", "created", "asc", issueRepoExtractor, []string{"alpha", "beta", "gamma"}},   // -3h, -2h, -1h
		{"created_desc", "created", "desc", issueRepoExtractor, []string{"gamma", "beta", "alpha"}}, // -1h, -2h, -3h
		{"updated_asc", "updated", "asc", issueRepoExtractor, []string{"gamma", "beta", "alpha"}},   // -2h, -1h, 0
		{"updated_desc", "updated", "desc", issueRepoExtractor, []string{"alpha", "beta", "gamma"}}, // 0, -1h, -2h
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := makeIssues()
			sortIssues(issues, tt.sortBy, tt.order)
			got := tt.wantField(issues)
			for i, want := range tt.wantOrder {
				if got[i] != want {
					t.Errorf("position %d: got %q, want %q (full: %v)", i, got[i], want, got)
				}
			}
		})
	}
}

func TestSortIssuesEmpty(t *testing.T) {
	sortIssues(nil, "updated", "desc")
	sortIssues([]github.Issue{}, "repo", "asc")
}

func TestSortPullRequests(t *testing.T) {
	now := time.Now()

	makePRs := func() []github.PullRequest {
		return []github.PullRequest{
			{Repository: "beta", Number: 2, Title: "Banana", Author: "charlie", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
			{Repository: "alpha", Number: 1, Title: "Apple", Author: "alice", CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now},
			{Repository: "gamma", Number: 3, Title: "Cherry", Author: "bob", CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		}
	}

	tests := []struct {
		name      string
		sortBy    string
		order     string
		wantField func([]github.PullRequest) []string
		wantOrder []string
	}{
		{"repo_asc", "repo", "asc", prRepoExtractor, []string{"alpha", "beta", "gamma"}},
		{"repo_desc", "repo", "desc", prRepoExtractor, []string{"gamma", "beta", "alpha"}},
		{"number_asc", "number", "asc", prNumberExtractor, []string{"1", "2", "3"}},
		{"number_desc", "number", "desc", prNumberExtractor, []string{"3", "2", "1"}},
		{"title_asc", "title", "asc", prTitleExtractor, []string{"Apple", "Banana", "Cherry"}},
		{"title_desc", "title", "desc", prTitleExtractor, []string{"Cherry", "Banana", "Apple"}},
		{"author_asc", "author", "asc", prAuthorExtractor, []string{"alice", "bob", "charlie"}},
		{"author_desc", "author", "desc", prAuthorExtractor, []string{"charlie", "bob", "alice"}},
		{"created_asc", "created", "asc", prRepoExtractor, []string{"alpha", "beta", "gamma"}},
		{"created_desc", "created", "desc", prRepoExtractor, []string{"gamma", "beta", "alpha"}},
		{"updated_asc", "updated", "asc", prRepoExtractor, []string{"gamma", "beta", "alpha"}},
		{"updated_desc", "updated", "desc", prRepoExtractor, []string{"alpha", "beta", "gamma"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prs := makePRs()
			sortPullRequests(prs, tt.sortBy, tt.order)
			got := tt.wantField(prs)
			for i, want := range tt.wantOrder {
				if got[i] != want {
					t.Errorf("position %d: got %q, want %q (full: %v)", i, got[i], want, got)
				}
			}
		})
	}
}

func TestSortPullRequestsEmpty(t *testing.T) {
	sortPullRequests(nil, "updated", "desc")
	sortPullRequests([]github.PullRequest{}, "repo", "asc")
}

// Issue field extractors.

func issueRepoExtractor(issues []github.Issue) []string {
	out := make([]string, len(issues))
	for i, v := range issues {
		out[i] = v.Repository
	}
	return out
}

func issueNumberExtractor(issues []github.Issue) []string {
	out := make([]string, len(issues))
	for i, v := range issues {
		out[i] = strconv.Itoa(v.Number)
	}
	return out
}

func issueTitleExtractor(issues []github.Issue) []string {
	out := make([]string, len(issues))
	for i, v := range issues {
		out[i] = v.Title
	}
	return out
}

func issueAuthorExtractor(issues []github.Issue) []string {
	out := make([]string, len(issues))
	for i, v := range issues {
		out[i] = v.Author
	}
	return out
}

// PR field extractors.

func prRepoExtractor(prs []github.PullRequest) []string {
	out := make([]string, len(prs))
	for i, v := range prs {
		out[i] = v.Repository
	}
	return out
}

func prNumberExtractor(prs []github.PullRequest) []string {
	out := make([]string, len(prs))
	for i, v := range prs {
		out[i] = strconv.Itoa(v.Number)
	}
	return out
}

func prTitleExtractor(prs []github.PullRequest) []string {
	out := make([]string, len(prs))
	for i, v := range prs {
		out[i] = v.Title
	}
	return out
}

func prAuthorExtractor(prs []github.PullRequest) []string {
	out := make([]string, len(prs))
	for i, v := range prs {
		out[i] = v.Author
	}
	return out
}

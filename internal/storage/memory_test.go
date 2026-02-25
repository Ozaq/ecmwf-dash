package storage

import (
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/ozaq/ecmwf-dash/internal/github"
)

func TestGetIssuesReturnsCopy(t *testing.T) {
	s := New()
	original := []github.Issue{
		{Repository: "b", Number: 2},
		{Repository: "a", Number: 1},
	}
	s.SetIssues(original)

	got, ts := s.GetIssues()
	if ts.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}

	// Mutate the returned slice
	sort.Slice(got, func(i, j int) bool {
		return got[i].Repository < got[j].Repository
	})

	// Original should be unchanged
	internal, _ := s.GetIssues()
	if internal[0].Repository != "b" {
		t.Errorf("internal slice was mutated: got %q, want %q", internal[0].Repository, "b")
	}
}

func TestGetPullRequestsReturnsCopy(t *testing.T) {
	s := New()
	original := []github.PullRequest{
		{Repository: "z", Number: 3},
		{Repository: "a", Number: 1},
	}
	s.SetPullRequests(original)

	got, _ := s.GetPullRequests()
	got[0].Repository = "mutated"

	internal, _ := s.GetPullRequests()
	if internal[0].Repository != "z" {
		t.Errorf("internal slice was mutated: got %q, want %q", internal[0].Repository, "z")
	}
}

func TestGetBranchChecksReturnsCopy(t *testing.T) {
	s := New()
	original := []github.BranchCheck{
		{Repository: "repo1", Branch: "main"},
	}
	s.SetBranchChecks(original)

	got, _ := s.GetBranchChecks()
	got[0].Branch = "mutated"

	internal, _ := s.GetBranchChecks()
	if internal[0].Branch != "main" {
		t.Errorf("internal slice was mutated: got %q, want %q", internal[0].Branch, "main")
	}
}

func TestSetGetRoundTrip(t *testing.T) {
	s := New()

	issues := []github.Issue{{Repository: "r", Number: 1, Title: "test"}}
	s.SetIssues(issues)
	got, ts := s.GetIssues()

	if len(got) != 1 || got[0].Title != "test" {
		t.Errorf("issue round-trip failed: got %+v", got)
	}
	if time.Since(ts) > time.Second {
		t.Errorf("timestamp too old: %v", ts)
	}
}

func TestEmptyStoreReturnsEmptySlice(t *testing.T) {
	s := New()

	issues, ts := s.GetIssues()
	if len(issues) != 0 {
		t.Errorf("expected empty slice, got %d items", len(issues))
	}
	if !ts.IsZero() {
		t.Errorf("expected zero timestamp, got %v", ts)
	}
}

// Deep copy tests for inner slices

func TestDeepCopyIssueLabels(t *testing.T) {
	s := New()
	s.SetIssues([]github.Issue{
		{Repository: "r", Labels: []github.Label{{Name: "bug", Color: "ff0000"}}},
	})

	got, _ := s.GetIssues()
	got[0].Labels[0].Name = "mutated"

	internal, _ := s.GetIssues()
	if internal[0].Labels[0].Name != "bug" {
		t.Errorf("inner label slice was mutated: got %q, want %q", internal[0].Labels[0].Name, "bug")
	}
}

func TestDeepCopyPRChecks(t *testing.T) {
	s := New()
	s.SetPullRequests([]github.PullRequest{
		{Repository: "r", Checks: []github.Check{{Name: "ci", Status: "completed"}}},
	})

	got, _ := s.GetPullRequests()
	got[0].Checks[0].Name = "mutated"

	internal, _ := s.GetPullRequests()
	if internal[0].Checks[0].Name != "ci" {
		t.Errorf("inner checks slice was mutated: got %q, want %q", internal[0].Checks[0].Name, "ci")
	}
}

func TestDeepCopyPRReviewers(t *testing.T) {
	s := New()
	s.SetPullRequests([]github.PullRequest{
		{Repository: "r", Reviewers: []github.Reviewer{{Login: "alice", State: "APPROVED"}}},
	})

	got, _ := s.GetPullRequests()
	got[0].Reviewers[0].Login = "mutated"

	internal, _ := s.GetPullRequests()
	if internal[0].Reviewers[0].Login != "alice" {
		t.Errorf("inner reviewers slice was mutated: got %q, want %q", internal[0].Reviewers[0].Login, "alice")
	}
}

func TestDeepCopyBranchCheckChecks(t *testing.T) {
	s := New()
	s.SetBranchChecks([]github.BranchCheck{
		{Repository: "r", Branch: "main", Checks: []github.Check{{Name: "lint"}}},
	})

	got, _ := s.GetBranchChecks()
	got[0].Checks[0].Name = "mutated"

	internal, _ := s.GetBranchChecks()
	if internal[0].Checks[0].Name != "lint" {
		t.Errorf("inner checks slice was mutated: got %q, want %q", internal[0].Checks[0].Name, "lint")
	}
}

func TestSetIssuesDefensiveCopy(t *testing.T) {
	s := New()
	input := []github.Issue{
		{Repository: "r", Labels: []github.Label{{Name: "bug"}}},
	}
	s.SetIssues(input)

	// Mutate the original input after Set
	input[0].Labels[0].Name = "mutated"

	got, _ := s.GetIssues()
	if got[0].Labels[0].Name != "bug" {
		t.Errorf("set did not defensively copy: got %q, want %q", got[0].Labels[0].Name, "bug")
	}
}

func TestLastFetchTimes(t *testing.T) {
	s := New()

	i, p, c := s.LastFetchTimes()
	if !i.IsZero() || !p.IsZero() || !c.IsZero() {
		t.Fatal("expected zero timestamps on empty store")
	}

	s.SetIssues(nil)
	i, _, _ = s.LastFetchTimes()
	if i.IsZero() {
		t.Error("expected non-zero issues timestamp after SetIssues")
	}
}

// Concurrency test — validates with -race

func TestConcurrentReadWrite(t *testing.T) {
	s := New()
	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Writers
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				s.SetIssues([]github.Issue{{Repository: "r", Labels: []github.Label{{Name: "l"}}}})
				s.SetPullRequests([]github.PullRequest{{Repository: "r", Checks: []github.Check{{Name: "c"}}}})
				s.SetBranchChecks([]github.BranchCheck{{Repository: "r", Checks: []github.Check{{Name: "c"}}}})
			}
		}()
	}

	// Readers
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				issues, _ := s.GetIssues()
				_ = issues
				prs, _ := s.GetPullRequests()
				_ = prs
				checks, _ := s.GetBranchChecks()
				_ = checks
				s.LastFetchTimes()
			}
		}()
	}

	wg.Wait()
}

// Verify Memory satisfies Store interface at compile time.
var _ Store = (*Memory)(nil)

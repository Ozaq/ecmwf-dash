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

// ---- Merge tests ----

func TestMergeIssuesRetainsFailedRepoData(t *testing.T) {
	s := New()
	s.SetIssues([]github.Issue{
		{Repository: "A", Number: 1},
		{Repository: "B", Number: 2},
		{Repository: "C", Number: 3},
	})

	// Merge: A+B succeeded, C failed
	s.MergeIssues(
		[]github.Issue{
			{Repository: "A", Number: 10},
			{Repository: "B", Number: 20},
		},
		[]string{"C"},
		[]string{"A", "B"},
	)

	got, _ := s.GetIssues()
	repoNumbers := make(map[string]int)
	for _, issue := range got {
		repoNumbers[issue.Repository] = issue.Number
	}

	if repoNumbers["A"] != 10 {
		t.Errorf("A should have updated number 10, got %d", repoNumbers["A"])
	}
	if repoNumbers["B"] != 20 {
		t.Errorf("B should have updated number 20, got %d", repoNumbers["B"])
	}
	if repoNumbers["C"] != 3 {
		t.Errorf("C should retain old number 3, got %d", repoNumbers["C"])
	}
}

func TestMergeIssuesUpdatesPerRepoTimestamps(t *testing.T) {
	s := New()
	s.SetIssues([]github.Issue{
		{Repository: "A", Number: 1},
		{Repository: "B", Number: 2},
	})

	seedTimes := s.RepoFetchTimes("issues")
	if seedTimes["A"].IsZero() || seedTimes["B"].IsZero() {
		t.Fatal("expected non-zero timestamps after SetIssues")
	}

	time.Sleep(10 * time.Millisecond)

	// Merge: A succeeds, B fails
	s.MergeIssues(
		[]github.Issue{{Repository: "A", Number: 10}},
		[]string{"B"},
		[]string{"A"},
	)

	times := s.RepoFetchTimes("issues")
	if !times["A"].After(seedTimes["A"]) {
		t.Error("A's timestamp should have been updated")
	}
	if !times["B"].Equal(seedTimes["B"]) {
		t.Error("B's timestamp should remain unchanged")
	}
}

func TestMergeIssuesDeepCopies(t *testing.T) {
	s := New()
	s.SetIssues([]github.Issue{{Repository: "A", Number: 1}})

	input := []github.Issue{{Repository: "A", Number: 10, Labels: []github.Label{{Name: "bug"}}}}
	s.MergeIssues(input, nil, []string{"A"})

	input[0].Labels[0].Name = "mutated"

	got, _ := s.GetIssues()
	for _, issue := range got {
		if issue.Repository == "A" && len(issue.Labels) > 0 && issue.Labels[0].Name == "mutated" {
			t.Error("MergeIssues did not deep copy input")
		}
	}
}

func TestMergeIssuesZeroItemRepoGetsTimestamp(t *testing.T) {
	s := New()
	// Merge with no data but succeededRepos includes "emptyrepo"
	s.MergeIssues(nil, nil, []string{"emptyrepo"})

	times := s.RepoFetchTimes("issues")
	if times["emptyrepo"].IsZero() {
		t.Error("repo with 0 items should still get a timestamp via succeededRepos")
	}
}

func TestMergeIssuesOnEmptyStore(t *testing.T) {
	s := New()
	s.MergeIssues(
		[]github.Issue{{Repository: "A", Number: 1}},
		[]string{"B"},
		[]string{"A"},
	)

	got, ts := s.GetIssues()
	if len(got) != 1 || got[0].Repository != "A" {
		t.Errorf("expected 1 issue for A, got %+v", got)
	}
	if ts.IsZero() {
		t.Error("expected non-zero global timestamp")
	}
}

func TestMergeIssuesAllFailed(t *testing.T) {
	s := New()
	s.SetIssues([]github.Issue{
		{Repository: "A", Number: 1},
		{Repository: "B", Number: 2},
	})

	// All repos failed — no new data, both in failedRepos
	s.MergeIssues(nil, []string{"A", "B"}, nil)

	got, _ := s.GetIssues()
	if len(got) != 2 {
		t.Errorf("expected all old data retained, got %d items", len(got))
	}
}

func TestMergeIssuesEmptyFailedRepos(t *testing.T) {
	s := New()
	s.SetIssues([]github.Issue{
		{Repository: "A", Number: 1},
		{Repository: "B", Number: 2},
	})

	// Full success — empty failedRepos discards old data for repos not in new data
	s.MergeIssues(
		[]github.Issue{{Repository: "A", Number: 10}},
		nil,
		[]string{"A", "B"},
	)

	got, _ := s.GetIssues()
	repoNumbers := make(map[string]int)
	for _, issue := range got {
		repoNumbers[issue.Repository] = issue.Number
	}

	if repoNumbers["A"] != 10 {
		t.Errorf("A should be 10, got %d", repoNumbers["A"])
	}
	// B had no items in new data and wasn't in failedRepos, so old B data is discarded
	if _, ok := repoNumbers["B"]; ok {
		t.Error("B should have been discarded (not in failedRepos)")
	}
}

func TestMergePullRequests(t *testing.T) {
	s := New()
	s.SetPullRequests([]github.PullRequest{
		{Repository: "A", Number: 1},
		{Repository: "B", Number: 2},
	})

	s.MergePullRequests(
		[]github.PullRequest{{Repository: "A", Number: 10}},
		[]string{"B"},
		[]string{"A"},
	)

	got, _ := s.GetPullRequests()
	repoNumbers := make(map[string]int)
	for _, pr := range got {
		repoNumbers[pr.Repository] = pr.Number
	}

	if repoNumbers["A"] != 10 {
		t.Errorf("A should have updated number 10, got %d", repoNumbers["A"])
	}
	if repoNumbers["B"] != 2 {
		t.Errorf("B should retain old number 2, got %d", repoNumbers["B"])
	}
}

func TestMergeBranchChecks(t *testing.T) {
	s := New()
	s.SetBranchChecks([]github.BranchCheck{
		{Repository: "A", Branch: "main"},
		{Repository: "B", Branch: "develop"},
	})

	s.MergeBranchChecks(
		[]github.BranchCheck{{Repository: "A", Branch: "main", CommitSHA: "new"}},
		[]string{"B"},
		[]string{"A"},
	)

	got, _ := s.GetBranchChecks()
	repoBranches := make(map[string]string)
	for _, bc := range got {
		repoBranches[bc.Repository] = bc.CommitSHA
	}

	if repoBranches["A"] != "new" {
		t.Errorf("A should have updated SHA 'new', got %q", repoBranches["A"])
	}
	if _, ok := repoBranches["B"]; !ok {
		t.Error("B should be retained from old data")
	}
}

func TestSetIssuesUpdatesPerRepoTimestamps(t *testing.T) {
	s := New()
	s.SetIssues([]github.Issue{
		{Repository: "eccodes", Number: 1},
		{Repository: "atlas", Number: 2},
	})

	times := s.RepoFetchTimes("issues")
	if times["eccodes"].IsZero() {
		t.Error("expected non-zero timestamp for eccodes after SetIssues")
	}
	if times["atlas"].IsZero() {
		t.Error("expected non-zero timestamp for atlas after SetIssues")
	}
}

func TestRepoFetchTimesReturnsCopy(t *testing.T) {
	s := New()
	s.SetIssues([]github.Issue{{Repository: "r", Number: 1}})

	times := s.RepoFetchTimes("issues")
	times["injected"] = time.Now()

	times2 := s.RepoFetchTimes("issues")
	if _, ok := times2["injected"]; ok {
		t.Error("RepoFetchTimes did not return a copy")
	}
}

func TestRepoFetchTimesUnknownCategory(t *testing.T) {
	s := New()
	got := s.RepoFetchTimes("unknown")
	if got == nil || len(got) != 0 {
		t.Errorf("expected empty map for unknown category, got %v", got)
	}
}

func TestConcurrentMergeAndGet(t *testing.T) {
	s := New()
	s.SetIssues([]github.Issue{{Repository: "r", Number: 1}})

	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				s.MergeIssues([]github.Issue{{Repository: "r", Number: i}}, []string{"other"}, []string{"r"})
				s.MergePullRequests([]github.PullRequest{{Repository: "r", Number: i}}, nil, []string{"r"})
				s.MergeBranchChecks([]github.BranchCheck{{Repository: "r", Branch: "main"}}, nil, []string{"r"})
			}
		}()
	}

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				s.GetIssues()
				s.GetPullRequests()
				s.GetBranchChecks()
				s.RepoFetchTimes("issues")
				s.RepoFetchTimes("prs")
				s.RepoFetchTimes("checks")
			}
		}()
	}

	wg.Wait()
}

// Verify Memory satisfies Store interface at compile time.
var _ Store = (*Memory)(nil)

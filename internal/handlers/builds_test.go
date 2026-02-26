package handlers

import (
	"testing"

	"github.com/ozaq/ecmwf-dash/internal/github"
)

func TestIsMainBranch(t *testing.T) {
	tests := []struct {
		branch string
		want   bool
	}{
		{"main", true},
		{"master", true},
		{"develop", false},
		{"", false},
		{"Main", false},
		{"release/1.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			if got := isMainBranch(tt.branch); got != tt.want {
				t.Errorf("isMainBranch(%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}
}

func TestComputeBranchCounts(t *testing.T) {
	tests := []struct {
		name        string
		checks      []github.Check
		wantSuccess int
		wantFailure int
		wantRunning int
		wantStatus  string
		wantClass   string
	}{
		{
			name: "all_success",
			checks: []github.Check{
				{Status: "completed", Conclusion: "success"},
				{Status: "completed", Conclusion: "success"},
			},
			wantSuccess: 2, wantStatus: "Passed", wantClass: "status-success",
		},
		{
			name: "mixed_failure",
			checks: []github.Check{
				{Status: "completed", Conclusion: "success"},
				{Status: "completed", Conclusion: "failure"},
			},
			wantSuccess: 1, wantFailure: 1, wantStatus: "Failed", wantClass: "status-failure",
		},
		{
			name: "running_takes_priority",
			checks: []github.Check{
				{Status: "completed", Conclusion: "failure"},
				{Status: "in_progress"},
				{Status: "completed", Conclusion: "success"},
			},
			wantSuccess: 1, wantFailure: 1, wantRunning: 1, wantStatus: "Running", wantClass: "status-running",
		},
		{
			name: "queued_counts_as_running",
			checks: []github.Check{
				{Status: "queued"},
			},
			wantRunning: 1, wantStatus: "Running", wantClass: "status-running",
		},
		{
			name: "waiting_counts_as_running",
			checks: []github.Check{
				{Status: "waiting"},
			},
			wantRunning: 1, wantStatus: "Running", wantClass: "status-running",
		},
		{
			name: "pending_counts_as_running",
			checks: []github.Check{
				{Status: "pending"},
			},
			wantRunning: 1, wantStatus: "Running", wantClass: "status-running",
		},
		{
			name: "timed_out_counts_as_failure",
			checks: []github.Check{
				{Status: "completed", Conclusion: "timed_out"},
			},
			wantFailure: 1, wantStatus: "Failed", wantClass: "status-failure",
		},
		{
			name: "cancelled_counts_as_failure",
			checks: []github.Check{
				{Status: "completed", Conclusion: "cancelled"},
			},
			wantFailure: 1, wantStatus: "Failed", wantClass: "status-failure",
		},
		{
			name: "action_required_counts_as_failure",
			checks: []github.Check{
				{Status: "completed", Conclusion: "action_required"},
			},
			wantFailure: 1, wantStatus: "Failed", wantClass: "status-failure",
		},
		{
			name:       "no_checks",
			checks:     []github.Check{},
			wantStatus: "Unknown", wantClass: "status-neutral",
		},
		{
			name: "neutral_counts_as_failure",
			checks: []github.Check{
				{Status: "completed", Conclusion: "neutral"},
			},
			wantFailure: 1, wantStatus: "Failed", wantClass: "status-failure",
		},
		{
			name:       "no_checks_unknown",
			checks:     nil,
			wantStatus: "Unknown", wantClass: "status-neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := &BranchStatus{Checks: tt.checks}
			computeBranchCounts(bs)

			if bs.SuccessCount != tt.wantSuccess {
				t.Errorf("SuccessCount = %d, want %d", bs.SuccessCount, tt.wantSuccess)
			}
			if bs.FailureCount != tt.wantFailure {
				t.Errorf("FailureCount = %d, want %d", bs.FailureCount, tt.wantFailure)
			}
			if bs.RunningCount != tt.wantRunning {
				t.Errorf("RunningCount = %d, want %d", bs.RunningCount, tt.wantRunning)
			}
			if bs.OverallStatus != tt.wantStatus {
				t.Errorf("OverallStatus = %q, want %q", bs.OverallStatus, tt.wantStatus)
			}
			if bs.StatusClass != tt.wantClass {
				t.Errorf("StatusClass = %q, want %q", bs.StatusClass, tt.wantClass)
			}
		})
	}
}

func TestSortByConfigOrder(t *testing.T) {
	configOrder := []string{"eccodes", "atlas", "odc"}
	repos := []*RepositoryStatus{
		{Name: "odc"},
		{Name: "unknown-z"},
		{Name: "atlas"},
		{Name: "eccodes"},
		{Name: "unknown-a"},
	}

	sortByConfigOrder(repos, configOrder)

	want := []string{"eccodes", "atlas", "odc", "unknown-a", "unknown-z"}
	for i, r := range repos {
		if r.Name != want[i] {
			t.Errorf("position %d: got %q, want %q", i, r.Name, want[i])
		}
	}
}

func TestSortByConfigOrderEmpty(t *testing.T) {
	// No panic on nil/empty inputs.
	sortByConfigOrder(nil, nil)
	sortByConfigOrder([]*RepositoryStatus{}, []string{})
}

func TestGroupByRepository(t *testing.T) {
	branchChecks := []github.BranchCheck{
		{
			Repository: "eccodes",
			Branch:     "master",
			CommitSHA:  "abc123",
			Checks:     []github.Check{{Status: "completed", Conclusion: "success"}},
		},
		{
			Repository: "eccodes",
			Branch:     "develop",
			CommitSHA:  "def456",
			Checks:     []github.Check{{Status: "completed", Conclusion: "failure"}},
		},
		{
			Repository: "atlas",
			Branch:     "main",
			CommitSHA:  "ghi789",
			Checks:     []github.Check{{Status: "in_progress"}},
		},
	}
	repoConfig := []RepoBranches{
		{Name: "atlas", Branches: []string{"main", "develop"}},
		{Name: "eccodes", Branches: []string{"master", "develop"}},
	}

	repos := groupByRepository(branchChecks, repoConfig)

	if len(repos) != 2 {
		t.Fatalf("got %d repos, want 2", len(repos))
	}

	// atlas should come first per config order.
	if repos[0].Name != "atlas" {
		t.Errorf("repos[0].Name = %q, want %q", repos[0].Name, "atlas")
	}
	if len(repos[0].Branches) != 2 {
		t.Fatalf("atlas has %d branches, want 2", len(repos[0].Branches))
	}
	if repos[0].Branches[0].Branch != "main" {
		t.Errorf("atlas Branches[0].Branch = %q, want %q", repos[0].Branches[0].Branch, "main")
	}
	if repos[0].Branches[0].CommitSHA != "ghi789" {
		t.Errorf("atlas Branches[0].CommitSHA = %q, want %q", repos[0].Branches[0].CommitSHA, "ghi789")
	}
	if repos[0].Branches[0].OverallStatus != "Running" {
		t.Errorf("atlas Branches[0].OverallStatus = %q, want %q", repos[0].Branches[0].OverallStatus, "Running")
	}
	if !repos[0].Branches[0].IsMain {
		t.Error("atlas Branches[0].IsMain should be true")
	}

	if repos[1].Name != "eccodes" {
		t.Errorf("repos[1].Name = %q, want %q", repos[1].Name, "eccodes")
	}
	if repos[1].Branches[0].Branch != "master" {
		t.Errorf("eccodes Branches[0].Branch = %q, want %q", repos[1].Branches[0].Branch, "master")
	}
	if repos[1].Branches[0].OverallStatus != "Passed" {
		t.Errorf("eccodes Branches[0].OverallStatus = %q, want %q", repos[1].Branches[0].OverallStatus, "Passed")
	}
	if repos[1].Branches[1].OverallStatus != "Failed" {
		t.Errorf("eccodes Branches[1].OverallStatus = %q, want %q", repos[1].Branches[1].OverallStatus, "Failed")
	}
}

func TestGroupByRepositoryEmpty(t *testing.T) {
	repos := groupByRepository(nil, nil)
	if len(repos) != 0 {
		t.Errorf("got %d repos from nil input, want 0", len(repos))
	}

	repos = groupByRepository(nil, []RepoBranches{})
	if len(repos) != 0 {
		t.Errorf("got %d repos from empty config, want 0", len(repos))
	}
}

func TestGroupByRepositoryAllBranches(t *testing.T) {
	// All configured branches are included, not just main/develop.
	branchChecks := []github.BranchCheck{
		{Repository: "eccodes", Branch: "master", Checks: []github.Check{{Status: "completed", Conclusion: "success"}}},
		{Repository: "eccodes", Branch: "develop", Checks: []github.Check{{Status: "completed", Conclusion: "success"}}},
		{Repository: "eccodes", Branch: "release/1.0", Checks: []github.Check{{Status: "completed", Conclusion: "failure"}}},
	}
	repoConfig := []RepoBranches{
		{Name: "eccodes", Branches: []string{"master", "develop", "release/1.0"}},
	}

	repos := groupByRepository(branchChecks, repoConfig)

	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if len(repos[0].Branches) != 3 {
		t.Fatalf("got %d branches, want 3", len(repos[0].Branches))
	}
	if repos[0].Branches[2].Branch != "release/1.0" {
		t.Errorf("Branches[2].Branch = %q, want %q", repos[0].Branches[2].Branch, "release/1.0")
	}
	if repos[0].Branches[2].OverallStatus != "Failed" {
		t.Errorf("Branches[2].OverallStatus = %q, want %q", repos[0].Branches[2].OverallStatus, "Failed")
	}
	if repos[0].Branches[2].IsMain {
		t.Error("release/1.0 should not be IsMain")
	}
}

func TestGroupByRepositoryConfigOrder(t *testing.T) {
	// Branch order within a repo matches config order.
	branchChecks := []github.BranchCheck{
		{Repository: "eccodes", Branch: "develop", Checks: []github.Check{{Status: "completed", Conclusion: "success"}}},
		{Repository: "eccodes", Branch: "master", Checks: []github.Check{{Status: "completed", Conclusion: "success"}}},
	}
	repoConfig := []RepoBranches{
		{Name: "eccodes", Branches: []string{"master", "develop"}},
	}

	repos := groupByRepository(branchChecks, repoConfig)

	if repos[0].Branches[0].Branch != "master" {
		t.Errorf("first branch = %q, want %q (config order)", repos[0].Branches[0].Branch, "master")
	}
	if repos[0].Branches[1].Branch != "develop" {
		t.Errorf("second branch = %q, want %q (config order)", repos[0].Branches[1].Branch, "develop")
	}
}

func TestGroupByRepositoryUnknownRepos(t *testing.T) {
	// Repos not in config are appended alphabetically.
	branchChecks := []github.BranchCheck{
		{Repository: "zebra", Branch: "main", Checks: []github.Check{{Status: "completed", Conclusion: "success"}}},
		{Repository: "alpha", Branch: "main", Checks: []github.Check{{Status: "completed", Conclusion: "success"}}},
		{Repository: "eccodes", Branch: "master", Checks: []github.Check{{Status: "completed", Conclusion: "success"}}},
	}
	repoConfig := []RepoBranches{
		{Name: "eccodes", Branches: []string{"master"}},
	}

	repos := groupByRepository(branchChecks, repoConfig)

	if len(repos) != 3 {
		t.Fatalf("got %d repos, want 3", len(repos))
	}
	if repos[0].Name != "eccodes" {
		t.Errorf("repos[0] = %q, want eccodes (config)", repos[0].Name)
	}
	if repos[1].Name != "alpha" {
		t.Errorf("repos[1] = %q, want alpha (unknown, alphabetical)", repos[1].Name)
	}
	if repos[2].Name != "zebra" {
		t.Errorf("repos[2] = %q, want zebra (unknown, alphabetical)", repos[2].Name)
	}
}

func TestHasDetails(t *testing.T) {
	tests := []struct {
		name     string
		branches []BranchStatus
		want     bool
	}{
		{
			name: "no_details",
			branches: []BranchStatus{
				{SuccessCount: 5},
				{SuccessCount: 3},
			},
			want: false,
		},
		{
			name: "failure_on_one_branch",
			branches: []BranchStatus{
				{SuccessCount: 5},
				{FailureCount: 1},
			},
			want: true,
		},
		{
			name: "running_on_one_branch",
			branches: []BranchStatus{
				{RunningCount: 2},
				{SuccessCount: 3},
			},
			want: true,
		},
		{
			name:     "empty_branches",
			branches: []BranchStatus{},
			want:     false,
		},
		{
			name:     "nil_branches",
			branches: nil,
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &RepositoryStatus{Branches: tt.branches}
			if got := rs.HasDetails(); got != tt.want {
				t.Errorf("HasDetails() = %v, want %v", got, tt.want)
			}
		})
	}
}

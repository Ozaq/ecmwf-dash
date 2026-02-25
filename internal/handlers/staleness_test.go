package handlers

import (
	"testing"
	"time"
)

func TestStaleReposNoneStale(t *testing.T) {
	now := time.Now()
	repoTimes := map[string]time.Time{
		"eccodes": now.Add(-1 * time.Minute),
		"atlas":   now.Add(-2 * time.Minute),
	}
	got := staleRepos(repoTimes, 10*time.Minute, []string{"eccodes", "atlas"})
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestStaleReposSomeStale(t *testing.T) {
	now := time.Now()
	repoTimes := map[string]time.Time{
		"eccodes": now.Add(-1 * time.Minute),
		"atlas":   now.Add(-20 * time.Minute),
	}
	got := staleRepos(repoTimes, 10*time.Minute, []string{"eccodes", "atlas"})
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if !got["atlas"] {
		t.Error("expected atlas to be stale")
	}
	if got["eccodes"] {
		t.Error("expected eccodes to NOT be stale")
	}
}

func TestStaleReposNeverFetched(t *testing.T) {
	repoTimes := map[string]time.Time{
		"eccodes": time.Now(),
	}
	got := staleRepos(repoTimes, 10*time.Minute, []string{"eccodes", "atlas"})
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if !got["atlas"] {
		t.Error("expected atlas (never fetched) to be stale")
	}
}

func TestStaleReposAllStale(t *testing.T) {
	repoTimes := map[string]time.Time{}
	got := staleRepos(repoTimes, 10*time.Minute, []string{"eccodes", "atlas"})
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 stale repos, got %d", len(got))
	}
}

func TestStaleReposEmptyAllRepos(t *testing.T) {
	got := staleRepos(map[string]time.Time{}, 10*time.Minute, nil)
	if got != nil {
		t.Errorf("expected nil for empty allRepos, got %v", got)
	}
}

func TestSortedKeys(t *testing.T) {
	got := sortedKeys(map[string]bool{"zebra": true, "apple": true, "mango": true})
	want := []string{"apple", "mango", "zebra"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSortedKeysNil(t *testing.T) {
	got := sortedKeys(nil)
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

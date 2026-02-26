package handlers

import (
	"sort"
	"time"
)

// FetchIntervals holds the configured fetch intervals for staleness computation.
type FetchIntervals struct {
	Issues       time.Duration
	PullRequests time.Duration
	Actions      time.Duration
}

// staleRepos returns repo names whose last-success timestamp is older than
// threshold, or that have never been fetched. Returns nil if allRepos is empty.
func staleRepos(repoTimes map[string]time.Time, threshold time.Duration, allRepos []string) map[string]bool {
	if len(allRepos) == 0 {
		return nil
	}

	now := time.Now()
	stale := make(map[string]bool)
	for _, name := range allRepos {
		ts, ok := repoTimes[name]
		if !ok || now.Sub(ts) > threshold {
			stale[name] = true
		}
	}

	if len(stale) == 0 {
		return nil
	}
	return stale
}

// computeStaleness returns a (staleMap, sortedStaleList) pair for the given
// fetch category. On cold start (zero lastUpdate) it returns an empty map
// and nil list.
func (h *Handler) computeStaleness(category string, interval time.Duration, lastUpdate time.Time) (map[string]bool, []string) {
	if lastUpdate.IsZero() {
		return make(map[string]bool), nil
	}
	repoTimes := h.storage.RepoFetchTimes(category)
	threshold := interval * 3
	staleMap := staleRepos(repoTimes, threshold, h.repoNames)
	if staleMap == nil {
		return make(map[string]bool), nil
	}
	return staleMap, sortedKeys(staleMap)
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

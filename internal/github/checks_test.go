package github

import "testing"

func TestClassifyCheck(t *testing.T) {
	tests := []struct {
		status     string
		conclusion string
		want       string
	}{
		// Running states
		{"in_progress", "", "running"},
		{"queued", "", "running"},
		{"waiting", "", "running"},
		{"pending", "", "running"},

		// Success
		{"completed", "success", "success"},

		// Explicit failures
		{"completed", "failure", "failure"},
		{"completed", "timed_out", "failure"},
		{"completed", "cancelled", "failure"},
		{"completed", "action_required", "failure"},

		// Previously inconsistent: neutral/stale now classified as failure
		{"completed", "neutral", "failure"},
		{"completed", "stale", "failure"},

		// Unknown conclusion
		{"completed", "something_unknown", "failure"},
		{"completed", "", "failure"},
	}

	for _, tt := range tests {
		t.Run(tt.status+"/"+tt.conclusion, func(t *testing.T) {
			got := ClassifyCheck(tt.status, tt.conclusion)
			if got != tt.want {
				t.Errorf("ClassifyCheck(%q, %q) = %q, want %q", tt.status, tt.conclusion, got, tt.want)
			}
		})
	}
}

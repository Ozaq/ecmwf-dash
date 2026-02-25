package github

import "testing"

func TestDeriveReviewStatus(t *testing.T) {
	tests := []struct {
		name      string
		reviewers map[string]*Reviewer
		want      string
	}{
		{
			name:      "no reviewers",
			reviewers: map[string]*Reviewer{},
			want:      "pending",
		},
		{
			name: "single approval",
			reviewers: map[string]*Reviewer{
				"alice": {Login: "alice", State: "APPROVED"},
			},
			want: "approved",
		},
		{
			name: "single changes requested",
			reviewers: map[string]*Reviewer{
				"alice": {Login: "alice", State: "CHANGES_REQUESTED"},
			},
			want: "changes_requested",
		},
		{
			name: "approve then changes requested by different reviewer",
			reviewers: map[string]*Reviewer{
				"alice": {Login: "alice", State: "APPROVED"},
				"bob":   {Login: "bob", State: "CHANGES_REQUESTED"},
			},
			want: "changes_requested",
		},
		{
			name: "changes requested then approve by different reviewer",
			reviewers: map[string]*Reviewer{
				"alice": {Login: "alice", State: "CHANGES_REQUESTED"},
				"bob":   {Login: "bob", State: "APPROVED"},
			},
			want: "changes_requested",
		},
		{
			name: "multiple approvals",
			reviewers: map[string]*Reviewer{
				"alice": {Login: "alice", State: "APPROVED"},
				"bob":   {Login: "bob", State: "APPROVED"},
			},
			want: "approved",
		},
		{
			name: "reviewer cleared objection - latest is approved",
			reviewers: map[string]*Reviewer{
				"alice": {Login: "alice", State: "APPROVED"},
			},
			want: "approved",
		},
		{
			name: "only dismissed reviews remain as pending",
			reviewers: map[string]*Reviewer{
				"alice": {Login: "alice", State: "DISMISSED"},
			},
			want: "pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveReviewStatus(tt.reviewers)
			if got != tt.want {
				t.Errorf("DeriveReviewStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

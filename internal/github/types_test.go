package github

import "testing"

func TestIsInternal(t *testing.T) {
	tests := []struct {
		association string
		want        bool
	}{
		{"OWNER", true},
		{"MEMBER", true},
		{"COLLABORATOR", true},
		{"CONTRIBUTOR", false},
		{"NONE", false},
		{"", false},
		{"owner", false}, // lowercase — case-sensitive
		{"Owner", false}, // mixed case — case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.association, func(t *testing.T) {
			got := isInternal(tt.association)
			if got != tt.want {
				t.Errorf("isInternal(%q) = %v, want %v", tt.association, got, tt.want)
			}
		})
	}
}

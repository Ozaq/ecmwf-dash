package handlers

import "testing"

func TestSanitizeSort(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"repo", "repo"},
		{"number", "number"},
		{"title", "title"},
		{"author", "author"},
		{"created", "created"},
		{"updated", "updated"},
		{"", "updated"},
		{"unknown", "updated"},
		{"REPO", "updated"},
		{"Updated", "updated"},
		{"repo; DROP TABLE", "updated"},
	}
	for _, tt := range tests {
		got := sanitizeSort(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeSort(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeOrder(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"asc", "asc"},
		{"desc", "desc"},
		{"", "desc"},
		{"ASC", "desc"},
		{"DESC", "desc"},
		{"Desc", "desc"},
		{"random", "desc"},
	}
	for _, tt := range tests {
		got := sanitizeOrder(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeOrder(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

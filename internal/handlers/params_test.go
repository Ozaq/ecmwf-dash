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

func TestGetNextOrder(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"asc", "desc"},
		{"desc", "asc"},
		{"", "asc"},
	}
	for _, tt := range tests {
		got := getNextOrder(tt.input)
		if got != tt.want {
			t.Errorf("getNextOrder(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPaginate(t *testing.T) {
	tests := []struct {
		name                               string
		total, page, pageSize              int
		wantStart, wantEnd, wantTotalPages int
	}{
		{"first page", 250, 1, 100, 0, 100, 3},
		{"middle page", 250, 2, 100, 100, 200, 3},
		{"last page partial", 250, 3, 100, 200, 250, 3},
		{"last page exact", 200, 2, 100, 100, 200, 2},
		{"single page", 50, 1, 100, 0, 50, 1},
		{"empty list", 0, 1, 100, 0, 0, 0},
		{"page beyond end", 50, 5, 100, 50, 50, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, totalPages := paginate(tt.total, tt.page, tt.pageSize)
			if start != tt.wantStart || end != tt.wantEnd || totalPages != tt.wantTotalPages {
				t.Errorf("paginate(%d, %d, %d) = (%d, %d, %d), want (%d, %d, %d)",
					tt.total, tt.page, tt.pageSize,
					start, end, totalPages,
					tt.wantStart, tt.wantEnd, tt.wantTotalPages)
			}
		})
	}
}

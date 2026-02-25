package github

import (
	"html/template"
	"strings"
	"testing"
)

func TestSanitizeLabelColor(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ff0000", "ff0000"},
		{"00FF00", "00FF00"},
		{"aaBBcc", "aaBBcc"},
		{"", "cccccc"},
		{"red", "cccccc"},
		{"ff000", "cccccc"},   // too short
		{"ff00000", "cccccc"}, // too long
		{"ff000g", "cccccc"},  // invalid hex char
		{"ff0000; background-image: url(evil)", "cccccc"}, // CSS injection
		{"<script>", "cccccc"},                            // XSS attempt
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeLabelColor(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeLabelColor(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestComputeTextColor(t *testing.T) {
	tests := []struct {
		hex  string
		want string
	}{
		{"000000", "#ffffff"}, // black bg -> white text
		{"ffffff", "#000000"}, // white bg -> black text
		{"ff0000", "#000000"}, // red bg -> black text (luminance 0.2126 > 0.179)
		{"00ff00", "#000000"}, // green bg -> black text
		{"0000ff", "#ffffff"}, // blue bg -> white text
		{"ffff00", "#000000"}, // yellow bg -> black text
		{"cccccc", "#000000"}, // light gray -> black text
		{"333333", "#ffffff"}, // dark gray -> white text
	}

	for _, tt := range tests {
		t.Run(tt.hex, func(t *testing.T) {
			got := computeTextColor(tt.hex)
			if got != tt.want {
				t.Errorf("computeTextColor(%q) = %q, want %q", tt.hex, got, tt.want)
			}
		})
	}
}

func TestComputeLabelStyle(t *testing.T) {
	style := computeLabelStyle("ff0000")

	s := string(style)
	if !strings.Contains(s, "background-color: #ff0000") {
		t.Errorf("expected background-color in style, got %q", s)
	}
	if !strings.Contains(s, "color: #") {
		t.Errorf("expected text color in style, got %q", s)
	}

	// Verify it's a valid template.CSS type
	var _ template.CSS = style
}

func TestComputeLabelStyleWithInvalidColor(t *testing.T) {
	style := computeLabelStyle("invalid")

	s := string(style)
	if !strings.Contains(s, "background-color: #cccccc") {
		t.Errorf("expected fallback color, got %q", s)
	}
}

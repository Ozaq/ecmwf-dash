package handlers

import (
	"bytes"
	"html/template"
	"testing"
)

func TestTemplateFuncsKeys(t *testing.T) {
	fm := TemplateFuncs()
	for _, key := range []string{"add", "mul", "affirm"} {
		if _, ok := fm[key]; !ok {
			t.Errorf("TemplateFuncs() missing key %q", key)
		}
	}
}

func TestTemplateFuncsAdd(t *testing.T) {
	fm := TemplateFuncs()
	add := fm["add"].(func(int, int) int)
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 3},
		{0, 0, 0},
		{-1, 1, 0},
		{100, -50, 50},
	}
	for _, tt := range tests {
		if got := add(tt.a, tt.b); got != tt.want {
			t.Errorf("add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestTemplateFuncsMul(t *testing.T) {
	fm := TemplateFuncs()
	mul := fm["mul"].(func(int, int) int)
	tests := []struct {
		a, b, want int
	}{
		{3, 4, 12},
		{0, 5, 0},
		{-2, 3, -6},
	}
	for _, tt := range tests {
		if got := mul(tt.a, tt.b); got != tt.want {
			t.Errorf("mul(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestTemplateFuncsAffirm(t *testing.T) {
	fm := TemplateFuncs()
	affirm := fm["affirm"].(func() string)
	got := affirm()
	if got == "" {
		t.Error("affirm() returned empty string")
	}
	// Verify it returns a value from the known list
	found := false
	for _, a := range affirmations {
		if a == got {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("affirm() = %q, not in affirmations list", got)
	}
}

func TestTemplateFuncsUsableInTemplate(t *testing.T) {
	tmpl, err := template.New("test").Funcs(TemplateFuncs()).Parse(`{{add 2 3}}-{{mul 4 5}}-{{affirm}}`)
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("template execute error: %v", err)
	}
	out := buf.String()
	if len(out) < 5 { // "5-20-" minimum
		t.Errorf("template output too short: %q", out)
	}
}

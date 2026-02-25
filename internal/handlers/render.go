package handlers

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
)

// renderTemplate executes a template into a buffer before writing to the
// ResponseWriter. This prevents partial HTML output on template errors.
func renderTemplate(w http.ResponseWriter, tmpl *template.Template, name string, data any) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("Error executing template %q: %v", name, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("Error writing response for template %q: %v", name, err)
	}
}

package handlers

import (
    "html/template"
    "net/http"
    "sort"
    "time"

    "github.com/ozaq/ecmwf-dash/internal/github"
    "github.com/ozaq/ecmwf-dash/internal/storage"
)

type Handler struct {
    storage  *storage.Memory
    template *template.Template
}

func New(storage *storage.Memory, tmpl *template.Template) *Handler {
    return &Handler{
        storage:  storage,
        template: tmpl,
    }
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
    issues, lastUpdate := h.storage.GetIssues()

    // Sort by most recent activity
    sort.Slice(issues, func(i, j int) bool {
        return issues[i].UpdatedAt.After(issues[j].UpdatedAt)
    })

    data := struct {
        Issues     []github.Issue
        LastUpdate time.Time
    }{
        Issues:     issues,
        LastUpdate: lastUpdate,
    }

    if err := h.template.Execute(w, data); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

package handlers

import (
    "net/http"
    "sort"
    "strconv"
    "time"
    
    "github.com/ozaq/ecmwf-dash/internal/github"
)

func (h *Handler) PullRequests(w http.ResponseWriter, r *http.Request) {
    prs, lastUpdate := h.storage.GetPullRequests()

    // Get query params
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 {
        page = 1
    }
    
    sortBy := r.URL.Query().Get("sort")
    if sortBy == "" {
        sortBy = "updated"
    }
    order := r.URL.Query().Get("order")
    if order == "" {
        order = "desc"
    }

    // Get available CSS files
    cssFiles := getAvailableCSS()

    // Sort PRs
    sortPullRequests(prs, sortBy, order)

    // Paginate
    totalPages := (len(prs) + issuesPerPage - 1) / issuesPerPage
    start := (page - 1) * issuesPerPage
    end := start + issuesPerPage
    if end > len(prs) {
        end = len(prs)
    }
    
    var pagePRs []github.PullRequest
    if start < len(prs) {
        pagePRs = prs[start:end]
    }

    data := struct {
        PullRequests []github.PullRequest
        LastUpdate   time.Time
        CurrentPage  int
        TotalPages   int
        TotalPRs     int
        Sort         string
        Order        string
        NextOrder    string
        CSSFile      string
        CSSFiles     []string
    }{
        PullRequests: pagePRs,
        LastUpdate:   lastUpdate,
        CurrentPage:  page,
        TotalPages:   totalPages,
        TotalPRs:     len(prs),
        Sort:         sortBy,
        Order:        order,
        NextOrder:    getNextOrder(order),
        CSSFile:      h.cssFile,
        CSSFiles:     cssFiles,
    }

    if err := h.prTemplate.Execute(w, data); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func sortPullRequests(prs []github.PullRequest, sortBy, order string) {
    switch sortBy {
    case "repo":
        sort.Slice(prs, func(i, j int) bool {
            if order == "asc" {
                return prs[i].Repository < prs[j].Repository
            }
            return prs[i].Repository > prs[j].Repository
        })
    case "number":
        sort.Slice(prs, func(i, j int) bool {
            if order == "asc" {
                return prs[i].Number < prs[j].Number
            }
            return prs[i].Number > prs[j].Number
        })
    case "title":
        sort.Slice(prs, func(i, j int) bool {
            if order == "asc" {
                return prs[i].Title < prs[j].Title
            }
            return prs[i].Title > prs[j].Title
        })
    case "author":
        sort.Slice(prs, func(i, j int) bool {
            if order == "asc" {
                return prs[i].Author < prs[j].Author
            }
            return prs[i].Author > prs[j].Author
        })
    case "created":
        sort.Slice(prs, func(i, j int) bool {
            if order == "asc" {
                return prs[i].CreatedAt.Before(prs[j].CreatedAt)
            }
            return prs[i].CreatedAt.After(prs[j].CreatedAt)
        })
    default: // "updated"
        sort.Slice(prs, func(i, j int) bool {
            if order == "asc" {
                return prs[i].UpdatedAt.Before(prs[j].UpdatedAt)
            }
            return prs[i].UpdatedAt.After(prs[j].UpdatedAt)
        })
    }
}
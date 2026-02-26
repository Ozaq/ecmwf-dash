package handlers

const itemsPerPage = 100

var validSortFields = map[string]bool{
	"repo":    true,
	"number":  true,
	"title":   true,
	"author":  true,
	"created": true,
	"updated": true,
}

func sanitizeSort(sortBy string) string {
	if validSortFields[sortBy] {
		return sortBy
	}
	return "updated"
}

func sanitizeOrder(order string) string {
	if order == "asc" || order == "desc" {
		return order
	}
	return "desc"
}

func getNextOrder(current string) string {
	if current == "asc" {
		return "desc"
	}
	return "asc"
}

func sanitizeRepo(repo string, validRepos []string) string {
	if repo == "" {
		return ""
	}
	for _, r := range validRepos {
		if r == repo {
			return repo
		}
	}
	return ""
}

// paginate returns clamped start/end indices and total page count for the
// given total item count, 1-based page number, and page size.
func paginate(total, page, pageSize int) (start, end, totalPages int) {
	if total <= 0 || pageSize <= 0 {
		return 0, 0, 0
	}
	totalPages = (total + pageSize - 1) / pageSize
	if page > totalPages {
		page = totalPages
	}
	start = (page - 1) * pageSize
	if start > total {
		start = total
	}
	end = start + pageSize
	if end > total {
		end = total
	}
	return start, end, totalPages
}

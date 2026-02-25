package handlers

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

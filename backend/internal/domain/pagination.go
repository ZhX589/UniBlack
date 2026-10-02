package domain

// NormalizePagination keeps the list API's existing pagination defaults:
// pages start at 1, and sizes outside 1..100 fall back to 20 (not 100).
// Both handlers and services use it so response metadata matches query limits.
func NormalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

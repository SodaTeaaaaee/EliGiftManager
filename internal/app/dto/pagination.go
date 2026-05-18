package dto

// PaginationInput carries pagination and sorting parameters from the frontend.
type PaginationInput struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	SortBy   string `json:"sortBy"`
	SortDesc bool   `json:"sortDesc"`
}

// PaginationResult wraps any paginated list with metadata.
type PaginationResult struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalCount int `json:"totalCount"`
	TotalPages int `json:"totalPages"`
}

// NormalizePagination fills in defaults and clamps page/pageSize to safe ranges.
func NormalizePagination(input PaginationInput) PaginationInput {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 500 {
		input.PageSize = 50
	}
	return input
}

// ComputePages calculates TotalPages from TotalCount and PageSize.
func (r *PaginationResult) ComputePages() {
	if r.PageSize <= 0 {
		r.PageSize = 50
	}
	r.TotalPages = (r.TotalCount + r.PageSize - 1) / r.PageSize
}

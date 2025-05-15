package dto

// PaginationInfo adalah struct untuk informasi pagination response API
// Dapat digunakan di seluruh handler yang membutuhkan response paginasi
type PaginationInfo struct {
	Page      int `json:"page"`
	Limit     int `json:"limit"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
}

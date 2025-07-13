package dto

import "time"

// TemplateListResponse represents the response for template list page
type TemplateListResponse struct {
	Templates  []TemplateListItem `json:"templates"`
	Pagination *WebPagination     `json:"pagination,omitempty"`
	Search     string             `json:"search,omitempty"`
	Status     string             `json:"status,omitempty"`
}

// TemplateListItem represents a single template in the list
type TemplateListItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WebPagination is pagination struct for web UI
type WebPagination struct {
	CurrentPage int   `json:"current_page"`
	TotalPages  int   `json:"total_pages"`
	Total       int64 `json:"total"`
	From        int   `json:"from"`
	To          int   `json:"to"`
	HasPrev     bool  `json:"has_prev"`
	HasNext     bool  `json:"has_next"`
	PrevPage    int   `json:"prev_page"`
	NextPage    int   `json:"next_page"`
	Pages       []int `json:"pages"`
}

// TemplateListRequest represents query parameters for template list
type TemplateListRequest struct {
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Search string `query:"search"`
	Status string `query:"status"`
}

// NewWebPagination creates pagination object for web UI
func NewWebPagination(page, limit int, total int64) *WebPagination {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages < 1 {
		totalPages = 1
	}

	from := (page-1)*limit + 1
	to := page * limit
	if int64(to) > total {
		to = int(total)
	}
	if total == 0 {
		from = 0
		to = 0
	}

	pagination := &WebPagination{
		CurrentPage: page,
		TotalPages:  totalPages,
		Total:       total,
		From:        from,
		To:          to,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
	}

	// Generate page numbers for pagination
	pages := []int{}
	start := page - 2
	end := page + 2

	if start < 1 {
		start = 1
	}
	if end > totalPages {
		end = totalPages
	}

	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}

	pagination.Pages = pages

	return pagination
}

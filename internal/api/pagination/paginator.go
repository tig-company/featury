package pagination

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tig-company/featury/internal/models"
)

// DefaultPageSize is the default number of items per page
const DefaultPageSize = 20

// MaxPageSize is the maximum allowed page size
const MaxPageSize = 100

// MinPageSize is the minimum allowed page size
const MinPageSize = 1

// Paginator handles pagination logic for API responses
type Paginator struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalCount int64 `json:"total_count"`
	TotalPages int   `json:"total_pages"`
	Offset     int   `json:"offset"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// NewPaginator creates a new paginator with the given parameters
func NewPaginator(page, pageSize int, totalCount int64) *Paginator {
	p := &Paginator{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
	}
	
	p.normalize()
	p.calculate()
	
	return p
}

// NewPaginatorFromRequest creates a paginator from Gin request context
func NewPaginatorFromRequest(c *gin.Context) *Paginator {
	page := getIntQueryParam(c, "page", 1)
	pageSize := getIntQueryParam(c, "limit", DefaultPageSize)
	
	// Alternative parameter names
	if pageSize == DefaultPageSize {
		pageSize = getIntQueryParam(c, "page_size", DefaultPageSize)
		pageSize = getIntQueryParam(c, "per_page", pageSize)
	}
	
	return &Paginator{
		Page:     page,
		PageSize: pageSize,
	}
}

// FromPaginationParams creates a paginator from models.PaginationParams
func FromPaginationParams(params *models.PaginationParams, totalCount int64) *Paginator {
	return NewPaginator(params.Page, params.PageSize, totalCount)
}

// ToPaginationParams converts the paginator to models.PaginationParams
func (p *Paginator) ToPaginationParams() *models.PaginationParams {
	return &models.PaginationParams{
		Page:     p.Page,
		PageSize: p.PageSize,
		Offset:   p.Offset,
	}
}

// normalize ensures pagination parameters have valid values
func (p *Paginator) normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	
	if p.PageSize < MinPageSize {
		p.PageSize = DefaultPageSize
	}
	
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
}

// calculate computes derived pagination values
func (p *Paginator) calculate() {
	p.Offset = (p.Page - 1) * p.PageSize
	
	if p.TotalCount > 0 {
		p.TotalPages = int((p.TotalCount + int64(p.PageSize) - 1) / int64(p.PageSize))
	} else {
		p.TotalPages = 1
	}
	
	p.HasNext = p.Page < p.TotalPages
	p.HasPrev = p.Page > 1
}

// SetTotalCount sets the total count and recalculates pagination metadata
func (p *Paginator) SetTotalCount(totalCount int64) {
	p.TotalCount = totalCount
	p.calculate()
}

// GetLinks generates pagination links for the current request
func (p *Paginator) GetLinks(c *gin.Context) PaginationLinks {
	baseURL := getBaseURL(c)
	query := copyQueryParams(c)
	
	links := PaginationLinks{}
	
	// Self link
	query.Set("page", strconv.Itoa(p.Page))
	query.Set("limit", strconv.Itoa(p.PageSize))
	links.Self = baseURL + "?" + query.Encode()
	
	// First link
	query.Set("page", "1")
	links.First = baseURL + "?" + query.Encode()
	
	// Last link
	if p.TotalPages > 0 {
		query.Set("page", strconv.Itoa(p.TotalPages))
		links.Last = baseURL + "?" + query.Encode()
	}
	
	// Previous link
	if p.HasPrev {
		query.Set("page", strconv.Itoa(p.Page-1))
		links.Prev = baseURL + "?" + query.Encode()
	}
	
	// Next link
	if p.HasNext {
		query.Set("page", strconv.Itoa(p.Page+1))
		links.Next = baseURL + "?" + query.Encode()
	}
	
	return links
}

// AddHeaders adds pagination headers to the response
func (p *Paginator) AddHeaders(c *gin.Context) {
	c.Header("X-Page", strconv.Itoa(p.Page))
	c.Header("X-Page-Size", strconv.Itoa(p.PageSize))
	c.Header("X-Total-Count", strconv.FormatInt(p.TotalCount, 10))
	c.Header("X-Total-Pages", strconv.Itoa(p.TotalPages))
	
	if p.HasNext {
		c.Header("X-Has-Next", "true")
	} else {
		c.Header("X-Has-Next", "false")
	}
	
	if p.HasPrev {
		c.Header("X-Has-Prev", "true")
	} else {
		c.Header("X-Has-Prev", "false")
	}
}

// IsValidPage checks if the current page is valid for the total count
func (p *Paginator) IsValidPage() bool {
	if p.TotalCount == 0 {
		return p.Page == 1
	}
	return p.Page >= 1 && p.Page <= p.TotalPages
}

// GetStartIndex returns the 1-based start index for the current page
func (p *Paginator) GetStartIndex() int {
	if p.TotalCount == 0 {
		return 0
	}
	return p.Offset + 1
}

// GetEndIndex returns the 1-based end index for the current page
func (p *Paginator) GetEndIndex() int {
	if p.TotalCount == 0 {
		return 0
	}
	
	end := p.Offset + p.PageSize
	if int64(end) > p.TotalCount {
		end = int(p.TotalCount)
	}
	
	return end
}

// PaginationLinks represents navigation links for pagination
type PaginationLinks struct {
	Self  string `json:"self,omitempty"`
	First string `json:"first,omitempty"`
	Last  string `json:"last,omitempty"`
	Next  string `json:"next,omitempty"`
	Prev  string `json:"prev,omitempty"`
}

// PaginationMeta represents pagination metadata for responses
type PaginationMeta struct {
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalCount int64           `json:"total_count"`
	TotalPages int             `json:"total_pages"`
	HasNext    bool            `json:"has_next"`
	HasPrev    bool            `json:"has_prev"`
	StartIndex int             `json:"start_index"`
	EndIndex   int             `json:"end_index"`
	Links      PaginationLinks `json:"links,omitempty"`
}

// GetMeta returns pagination metadata
func (p *Paginator) GetMeta(c *gin.Context, includeLinks bool) PaginationMeta {
	meta := PaginationMeta{
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalCount: p.TotalCount,
		TotalPages: p.TotalPages,
		HasNext:    p.HasNext,
		HasPrev:    p.HasPrev,
		StartIndex: p.GetStartIndex(),
		EndIndex:   p.GetEndIndex(),
	}
	
	if includeLinks && c != nil {
		meta.Links = p.GetLinks(c)
	}
	
	return meta
}

// PaginatedResponse represents a generic paginated response
type PaginatedResponse struct {
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}, paginator *Paginator, c *gin.Context) *PaginatedResponse {
	return &PaginatedResponse{
		Data:       data,
		Pagination: paginator.GetMeta(c, true),
	}
}

// Helper functions

// getIntQueryParam extracts an integer query parameter with a default value
func getIntQueryParam(c *gin.Context, key string, defaultValue int) int {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}
	
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	
	return intValue
}

// getBaseURL constructs the base URL for the current request
func getBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	
	// Check for forwarded protocol headers
	if forwarded := c.GetHeader("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	} else if forwarded := c.GetHeader("X-Forwarded-Protocol"); forwarded != "" {
		scheme = forwarded
	}
	
	host := c.Request.Host
	if forwarded := c.GetHeader("X-Forwarded-Host"); forwarded != "" {
		host = forwarded
	}
	
	return fmt.Sprintf("%s://%s%s", scheme, host, c.Request.URL.Path)
}

// copyQueryParams creates a copy of the current request's query parameters
func copyQueryParams(c *gin.Context) url.Values {
	original := c.Request.URL.Query()
	copy := url.Values{}
	
	for key, values := range original {
		// Skip page and limit parameters as they'll be set explicitly
		if key == "page" || key == "limit" || key == "page_size" || key == "per_page" {
			continue
		}
		copy[key] = values
	}
	
	return copy
}

// Validation functions

// ValidatePaginationParams validates pagination parameters
func ValidatePaginationParams(page, pageSize int) error {
	if page < 1 {
		return fmt.Errorf("page must be greater than 0")
	}
	
	if pageSize < MinPageSize {
		return fmt.Errorf("page_size must be at least %d", MinPageSize)
	}
	
	if pageSize > MaxPageSize {
		return fmt.Errorf("page_size must not exceed %d", MaxPageSize)
	}
	
	return nil
}

// Utility functions for common pagination patterns

// GetOffsetAndLimit calculates offset and limit from page and pageSize
func GetOffsetAndLimit(page, pageSize int) (offset, limit int) {
	if page < 1 {
		page = 1
	}
	if pageSize < MinPageSize {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	
	offset = (page - 1) * pageSize
	limit = pageSize
	
	return offset, limit
}

// CalculateTotalPages calculates total pages from total count and page size
func CalculateTotalPages(totalCount int64, pageSize int) int {
	if totalCount == 0 || pageSize <= 0 {
		return 1
	}
	
	return int((totalCount + int64(pageSize) - 1) / int64(pageSize))
}
package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Query struct {
	Page     int
	PageSize int
}

func Parse(c *gin.Context) Query {
	page := parsePositive(c.Query("page"), 1)
	pageSize := parsePositive(c.Query("pageSize"), 20)
	if pageSize > 200 {
		pageSize = 200
	}
	return Query{Page: page, PageSize: pageSize}
}

func Offset(q Query) int {
	return (q.Page - 1) * q.PageSize
}

func parsePositive(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

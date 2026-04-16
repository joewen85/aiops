package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appErr "devops-system/backend/internal/errors"
	"devops-system/backend/internal/pagination"
	"devops-system/backend/internal/response"
)

func parseID(c *gin.Context) (uint, bool) {
	idRaw := c.Param("id")
	if idRaw == "" {
		response.Error(c, http.StatusBadRequest, appErr.ErrBadRequest)
		return 0, false
	}
	id64, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, appErr.ErrBadRequest)
		return 0, false
	}
	return uint(id64), true
}

func bindJSON(c *gin.Context, out interface{}) bool {
	if err := c.ShouldBindJSON(out); err != nil {
		response.Error(c, http.StatusBadRequest, appErr.New(3001, err.Error()))
		return false
	}
	return true
}

func listByModel[T any](c *gin.Context, db *gorm.DB) {
	page := pagination.Parse(c)
	var (
		items []T
		total int64
	)
	if err := db.Model(new(T)).Count(&total).Error; err != nil {
		response.Internal(c, err)
		return
	}
	if err := db.Order("id desc").Limit(page.PageSize).Offset(pagination.Offset(page)).Find(&items).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.List(c, items, total, page.Page, page.PageSize)
}

func getByID[T any](c *gin.Context, db *gorm.DB) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var item T
	if err := db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, appErr.ErrNotFound)
			return
		}
		response.Internal(c, err)
		return
	}
	response.Success(c, item)
}

func createByModel[T any](c *gin.Context, db *gorm.DB) {
	var input T
	if !bindJSON(c, &input) {
		return
	}
	if err := db.Create(&input).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, input)
}

func updateByModel[T any](c *gin.Context, db *gorm.DB) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var updates map[string]interface{}
	if !bindJSON(c, &updates) {
		return
	}
	delete(updates, "id")
	delete(updates, "createdAt")
	if err := db.Model(new(T)).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Internal(c, err)
		return
	}
	getByID[T](c, db)
}

func deleteByModel[T any](c *gin.Context, db *gorm.DB) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := db.Delete(new(T), id).Error; err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, gin.H{"id": id})
}

package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appErr "devops-system/backend/internal/errors"
)

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func List(c *gin.Context, list interface{}, total int64, page int, pageSize int) {
	Success(c, PageData{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func Error(c *gin.Context, status int, err appErr.AppError) {
	c.JSON(status, APIResponse{
		Code:    err.Code,
		Message: err.Message,
	})
}

func Internal(c *gin.Context, err error) {
	Error(c, http.StatusInternalServerError, appErr.New(5000, err.Error()))
}

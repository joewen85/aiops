package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestListResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/list", func(c *gin.Context) {
		List(c, []string{"a"}, 1, 1, 20)
	})

	req := httptest.NewRequest(http.MethodGet, "/list", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data should be object")
	}
	for _, key := range []string{"list", "total", "page", "pageSize"} {
		if _, exists := data[key]; !exists {
			t.Fatalf("missing key: %s", key)
		}
	}
}

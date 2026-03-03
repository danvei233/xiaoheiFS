package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPaging_PreservesValidParamsWhenSomeInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/admin/api/v1/admins?limit=3000&offset=40&page=bad", nil)

	limit, offset := paging(c)
	if limit != 500 {
		t.Fatalf("expected limit 500, got %d", limit)
	}
	if offset != 40 {
		t.Fatalf("expected offset 40, got %d", offset)
	}
}

func TestPaging_PagesAndPageSizeAreClamped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/admin/api/v1/admins?pages=2000&page=2", nil)

	limit, offset := paging(c)
	if limit != 500 {
		t.Fatalf("expected limit 500, got %d", limit)
	}
	if offset != 500 {
		t.Fatalf("expected offset 500, got %d", offset)
	}
}

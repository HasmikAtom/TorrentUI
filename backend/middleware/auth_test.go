package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireUser())
	r.GET("/echo", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"id":    c.GetString("userId"),
			"email": c.GetString("userEmail"),
			"role":  c.GetString("userRole"),
		})
	})
	return r
}

func TestRequireUser_RejectsMissingHeaders(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireUser_PopulatesContextFromHeaders(t *testing.T) {
	r := setupRouter()
	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	req.Header.Set("X-User-Id", "abc")
	req.Header.Set("X-User-Email", "a@x.com")
	req.Header.Set("X-User-Role", "admin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	expected := `{"email":"a@x.com","id":"abc","role":"admin"}`
	if w.Body.String() != expected {
		t.Fatalf("expected %s, got %s", expected, w.Body.String())
	}
}

package corehttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthReturnsHealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{})
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}
	if body := response.Body.String(); body != "{\"status\":\"healthy\"}" {
		t.Fatalf("unexpected health response: %s", body)
	}
}

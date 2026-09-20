package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLiveness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterConfig{})
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Fatalf("expected status body %q, got %q", "ok", response.Status)
	}
}

func TestUnknownRouteUsesErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterConfig{})
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	request.Header.Set(requestIDHeader, "request-123")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var response errorEnvelope
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != "not_found" || response.Error.RequestID != "request-123" {
		t.Fatalf("unexpected error response: %#v", response)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterConfig{AllowedOrigins: []string{"https://app.example.com"}})
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(http.MethodOptions, "/health/live", nil)
	request.Header.Set("Origin", "https://app.example.com")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("unexpected CORS origin %q", got)
	}
}

func TestOpenAPISpecIsServed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterConfig{})
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/yaml; charset=utf-8" {
		t.Fatalf("unexpected content type %q", contentType)
	}
}

func TestReadinessReportsDependencyFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(RouterConfig{Ready: func(_ context.Context) error {
		return errors.New("database unavailable")
	}})
	recorder := httptest.NewRecorder()

	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
}

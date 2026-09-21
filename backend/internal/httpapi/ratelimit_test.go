package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestFixedWindowLimiterRejectsAndResets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, time.September, 21, 10, 0, 0, 0, time.UTC)
	limiter := newFixedWindowLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }

	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		t.Fatalf("disable trusted proxies: %v", err)
	}
	router.GET("/limited", limiter.middleware("test"), func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	for attempt, expected := range []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests} {
		response := performRateLimitedRequest(t, router)
		if response.Code != expected {
			t.Fatalf("attempt %d status = %d, want %d", attempt+1, response.Code, expected)
		}
		if expected == http.StatusTooManyRequests && response.Header().Get("Retry-After") != "60" {
			t.Fatalf("Retry-After = %q, want 60", response.Header().Get("Retry-After"))
		}
	}

	now = now.Add(time.Minute)
	if response := performRateLimitedRequest(t, router); response.Code != http.StatusNoContent {
		t.Fatalf("status after reset = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func performRateLimitedRequest(t *testing.T, router http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/limited", nil)
	request.RemoteAddr = "192.0.2.10:4321"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/placesearch"
)

type placeSearchServiceStub struct {
	results []placesearch.Suggestion
	err     error
	input   placesearch.SuggestInput
}

func (s *placeSearchServiceStub) Search(_ context.Context, input placesearch.SuggestInput) ([]placesearch.Suggestion, error) {
	s.input = input
	return s.results, s.err
}

func TestPlaceSearchHandlerReturnsSuggestions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &placeSearchServiceStub{results: []placesearch.Suggestion{{
		ResultID: "result", Name: "Bhappe Da Hotel", Address: "New Delhi", Latitude: 28.6, Longitude: 77.2,
	}}}
	router := gin.New()
	router.GET("/places/search", placeSearchHandler{service: service}.search)

	request := httptest.NewRequest(http.MethodGet, "/places/search?q=Bhappe+Da+Hot&limit=5", nil)
	request.Header.Set("Accept-Language", "en-IN,en;q=0.9")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.input.Query != "Bhappe Da Hot" || service.input.Limit != 5 || service.input.AcceptLanguage == "" {
		t.Fatalf("unexpected service input %#v", service.input)
	}
	if body := response.Body.String(); !containsAll(body, "Bhappe Da Hotel", "Powered by Geoapify", "OpenStreetMap") {
		t.Fatalf("unexpected response body %s", body)
	}
}

func TestPlaceSearchHandlerMapsQuotaError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/places/search", placeSearchHandler{service: &placeSearchServiceStub{err: placesearch.ErrQuotaExceeded}}.search)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/places/search?q=Delhi", nil))

	if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") == "" {
		t.Fatalf("expected quota response with Retry-After, got %d %#v", response.Code, response.Header())
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}

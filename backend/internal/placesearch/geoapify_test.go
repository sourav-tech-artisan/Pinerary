package placesearch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeoapifySuggest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/geocode/autocomplete" {
			t.Fatalf("unexpected path %q", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("text") != "Bhappe Da Hot" || query.Get("limit") != "5" || query.Get("lang") != "en" ||
			query.Get("format") != "json" || query.Get("apiKey") != "test-secret" {
			t.Fatalf("unexpected query %#v", query)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
            "results": [{
                "name": "Bhappe Da Hotel",
                "address_line1": "Bhappe Da Hotel",
                "formatted": "Bhappe Da Hotel, Karol Bagh, New Delhi, India",
                "place_id": "51example",
                "category": "catering.restaurant",
                "result_type": "amenity",
                "lat": 28.6512,
                "lon": 77.1905,
                "bbox": {"lon1": 77.1904, "lat1": 28.6511, "lon2": 77.1906, "lat2": 28.6513}
            }, {
                "name": "Invalid coordinate",
                "formatted": "Invalid coordinate",
                "lat": 95,
                "lon": 77
            }]
        }`))
	}))
	t.Cleanup(server.Close)

	client, err := NewGeoapifyClient(server.URL, "test-secret", server.Client())
	if err != nil {
		t.Fatalf("NewGeoapifyClient() error = %v", err)
	}
	client.minimumDelay = 0
	results, err := client.Suggest(context.Background(), SuggestInput{
		Query: "Bhappe Da Hot", Limit: 5, AcceptLanguage: "en",
	})
	if err != nil {
		t.Fatalf("Suggest() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one valid result, got %#v", results)
	}
	result := results[0]
	if result.ResultID != "51example" || result.Name != "Bhappe Da Hotel" || result.Type != "amenity" {
		t.Fatalf("unexpected result %#v", result)
	}
	if result.BoundingBox == nil || result.BoundingBox.South != 28.6511 || result.BoundingBox.East != 77.1906 {
		t.Fatalf("unexpected bounding box %#v", result.BoundingBox)
	}
}

func TestGeoapifyErrorDoesNotExposeAPIKey(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	const secret = "must-not-leak"
	client, err := NewGeoapifyClient(server.URL, secret, server.Client())
	if err != nil {
		t.Fatalf("NewGeoapifyClient() error = %v", err)
	}
	client.minimumDelay = 0
	_, err = client.Suggest(context.Background(), SuggestInput{Query: "Delhi", Limit: 5})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected provider unavailable, got %v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("provider error exposed the API key")
	}
}

func TestNewGeoapifyClientRejectsMissingConfiguration(t *testing.T) {
	t.Parallel()

	if _, err := NewGeoapifyClient("not-a-url", "secret", nil); err == nil {
		t.Fatal("expected invalid URL error")
	}
	if _, err := NewGeoapifyClient("https://api.geoapify.com", "", nil); err == nil {
		t.Fatal("expected missing API key error")
	}
}

package geocoding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNominatimReverse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("User-Agent") != "pinerary-test" {
			t.Errorf("unexpected user agent %q", request.Header.Get("User-Agent"))
		}
		if request.URL.Query().Get("lat") != "26.9373000" || request.URL.Query().Get("lon") != "75.8155000" {
			t.Errorf("unexpected coordinates: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"name":"Nahargarh Fort","display_name":"Nahargarh Fort, Jaipur"}`))
	}))
	defer server.Close()

	client, err := NewNominatimClient(server.URL, "pinerary-test", server.Client())
	if err != nil {
		t.Fatalf("NewNominatimClient() error = %v", err)
	}
	client.minimumDelay = 0

	result, err := client.Reverse(context.Background(), 26.9373, 75.8155)
	if err != nil {
		t.Fatalf("Reverse() error = %v", err)
	}
	if result.DisplayName != "Nahargarh Fort" {
		t.Fatalf("DisplayName = %q", result.DisplayName)
	}
}

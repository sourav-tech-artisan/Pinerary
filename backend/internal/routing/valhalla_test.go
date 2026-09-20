package routing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValhallaMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/trace_attributes" {
			t.Errorf("path = %q", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"shape":"_c` + "`" + `|@_gayB_ibE_ibE"}`))
	}))
	defer server.Close()

	client, err := NewValhallaClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewValhallaClient() error = %v", err)
	}
	matched, err := client.Match(context.Background(), []Coordinate{{}, {Latitude: 1, Longitude: 1}})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if len(matched) != 2 {
		t.Fatalf("len(matched) = %d", len(matched))
	}
}

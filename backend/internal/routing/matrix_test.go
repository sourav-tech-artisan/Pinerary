package routing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValhallaMatrixPreservesUnreachableCells(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/sources_to_targets" {
			t.Errorf("path = %q", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"sources_to_targets": [[
				{"time": null, "distance": null},
				{"time": 125, "distance": 2.75}
			]]
		}`))
	}))
	defer server.Close()

	client, err := NewValhallaClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewValhallaClient() error = %v", err)
	}
	costs, err := client.Matrix(context.Background(), Coordinate{}, []Coordinate{{}, {}}, ModeMotorcycle)
	if err != nil {
		t.Fatalf("Matrix() error = %v", err)
	}
	if len(costs) != 2 {
		t.Fatalf("len(costs) = %d", len(costs))
	}
	if costs[0].Reachable {
		t.Fatal("null matrix cell was marked reachable")
	}
	if !costs[1].Reachable || costs[1].DistanceM != 2750 || costs[1].Duration != 125 {
		t.Fatalf("reachable cost = %+v", costs[1])
	}
}

func TestValhallaMatrixRejectsPartialCost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"sources_to_targets":[[{"time":12,"distance":null}]]}`))
	}))
	defer server.Close()

	client, err := NewValhallaClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewValhallaClient() error = %v", err)
	}
	if _, err := client.Matrix(context.Background(), Coordinate{}, []Coordinate{{}}, ModeCar); err == nil {
		t.Fatal("Matrix() accepted a partial cost")
	}
}

func TestValhallaCosting(t *testing.T) {
	if got := valhallaCosting(ModeMotorcycle); got != "motorcycle" {
		t.Fatalf("motorcycle costing = %q", got)
	}
	if got := valhallaCosting(ModeCar); got != "auto" {
		t.Fatalf("car costing = %q", got)
	}
	if got := valhallaCosting(ModeWalking); got != "pedestrian" {
		t.Fatalf("walking costing = %q", got)
	}
}

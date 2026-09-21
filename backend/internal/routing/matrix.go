package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Mode string

const (
	ModeMotorcycle Mode = "motorcycle"
	ModeCar        Mode = "car"
	ModeWalking    Mode = "walking"
)

type Cost struct {
	DistanceM float64
	Duration  int32
}

type MatrixRouter interface {
	Matrix(context.Context, Coordinate, []Coordinate, Mode) ([]Cost, error)
}

func (c *ValhallaClient) Matrix(ctx context.Context, origin Coordinate, destinations []Coordinate, mode Mode) ([]Cost, error) {
	if len(destinations) == 0 {
		return []Cost{}, nil
	}
	targets := make([]map[string]float64, 0, len(destinations))
	for _, destination := range destinations {
		targets = append(targets, map[string]float64{"lat": destination.Latitude, "lon": destination.Longitude})
	}
	payload, err := json.Marshal(map[string]any{
		"sources": []map[string]float64{{"lat": origin.Latitude, "lon": origin.Longitude}},
		"targets": targets,
		"costing": valhallaCosting(mode),
		"units":   "kilometers",
	})
	if err != nil {
		return nil, fmt.Errorf("encode Valhalla matrix request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sources_to_targets", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create Valhalla matrix request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call Valhalla matrix: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("Valhalla matrix returned status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		SourcesToTargets [][]struct {
			Distance float64 `json:"distance"`
			Time     float64 `json:"time"`
		} `json:"sources_to_targets"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Valhalla matrix response: %w", err)
	}
	if len(result.SourcesToTargets) != 1 || len(result.SourcesToTargets[0]) != len(destinations) {
		return nil, fmt.Errorf("Valhalla matrix response shape is invalid")
	}

	costs := make([]Cost, 0, len(destinations))
	for _, cost := range result.SourcesToTargets[0] {
		costs = append(costs, Cost{DistanceM: cost.Distance * 1000, Duration: int32(cost.Time)})
	}
	return costs, nil
}

func valhallaCosting(mode Mode) string {
	switch mode {
	case ModeCar:
		return "auto"
	case ModeWalking:
		return "pedestrian"
	default:
		return "motorcycle"
	}
}

package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Coordinate struct {
	Latitude  float64
	Longitude float64
}

type MapMatcher interface {
	Match(context.Context, []Coordinate) ([]Coordinate, error)
}

type ValhallaClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewValhallaClient(baseURL string, httpClient *http.Client) (*ValhallaClient, error) {
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid Valhalla URL: %w", err)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &ValhallaClient{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}, nil
}

func (c *ValhallaClient) Match(ctx context.Context, coordinates []Coordinate) ([]Coordinate, error) {
	if len(coordinates) < 2 {
		return nil, fmt.Errorf("map matching needs at least two coordinates")
	}
	shape := make([]map[string]float64, 0, len(coordinates))
	for _, coordinate := range coordinates {
		shape = append(shape, map[string]float64{"lat": coordinate.Latitude, "lon": coordinate.Longitude})
	}
	payload, err := json.Marshal(map[string]any{
		"shape": shape, "costing": "motorcycle", "shape_match": "map_snap",
	})
	if err != nil {
		return nil, fmt.Errorf("encode Valhalla request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/trace_attributes", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create Valhalla request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call Valhalla: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("Valhalla returned status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Shape string `json:"shape"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Valhalla response: %w", err)
	}
	matched, err := decodePolyline6(result.Shape)
	if err != nil {
		return nil, fmt.Errorf("decode Valhalla shape: %w", err)
	}
	return matched, nil
}

func decodePolyline6(encoded string) ([]Coordinate, error) {
	coordinates := make([]Coordinate, 0)
	latitude, longitude := int64(0), int64(0)
	for index := 0; index < len(encoded); {
		latitudeDelta, next, err := decodePolylineValue(encoded, index)
		if err != nil {
			return nil, err
		}
		longitudeDelta, nextAfterLongitude, err := decodePolylineValue(encoded, next)
		if err != nil {
			return nil, err
		}
		index = nextAfterLongitude
		latitude += latitudeDelta
		longitude += longitudeDelta
		coordinates = append(coordinates, Coordinate{
			Latitude: float64(latitude) / 1e6, Longitude: float64(longitude) / 1e6,
		})
	}
	if len(coordinates) < 2 {
		return nil, fmt.Errorf("decoded shape has fewer than two coordinates")
	}
	return coordinates, nil
}

func decodePolylineValue(encoded string, start int) (int64, int, error) {
	var result int64
	shift := 0
	index := start
	for {
		if index >= len(encoded) || shift > 60 {
			return 0, index, fmt.Errorf("invalid encoded polyline")
		}
		value := int64(encoded[index]) - 63
		index++
		result |= (value & 0x1f) << shift
		shift += 5
		if value < 0x20 {
			break
		}
	}
	if result&1 != 0 {
		result = ^(result >> 1)
	} else {
		result >>= 1
	}
	return result, index, nil
}

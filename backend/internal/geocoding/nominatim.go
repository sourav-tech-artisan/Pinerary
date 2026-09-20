package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Result struct {
	DisplayName string          `json:"display_name"`
	Raw         json.RawMessage `json:"-"`
}

type ReverseGeocoder interface {
	Reverse(context.Context, float64, float64) (Result, error)
}

type NominatimClient struct {
	baseURL      string
	userAgent    string
	httpClient   *http.Client
	mu           sync.Mutex
	lastRequest  time.Time
	minimumDelay time.Duration
}

func NewNominatimClient(baseURL, userAgent string, httpClient *http.Client) (*NominatimClient, error) {
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid Nominatim URL: %w", err)
	}
	if strings.TrimSpace(userAgent) == "" {
		return nil, fmt.Errorf("Nominatim user agent is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}
	return &NominatimClient{
		baseURL: strings.TrimRight(baseURL, "/"), userAgent: userAgent,
		httpClient: httpClient, minimumDelay: time.Second,
	}, nil
}

func (c *NominatimClient) Reverse(ctx context.Context, latitude, longitude float64) (Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if wait := c.minimumDelay - time.Since(c.lastRequest); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		case <-timer.C:
		}
	}

	endpoint, err := url.Parse(c.baseURL + "/reverse")
	if err != nil {
		return Result{}, fmt.Errorf("build Nominatim endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("format", "jsonv2")
	query.Set("lat", fmt.Sprintf("%.7f", latitude))
	query.Set("lon", fmt.Sprintf("%.7f", longitude))
	query.Set("addressdetails", "1")
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Result{}, fmt.Errorf("create Nominatim request: %w", err)
	}
	request.Header.Set("User-Agent", c.userAgent)
	request.Header.Set("Accept", "application/json")

	c.lastRequest = time.Now()
	response, err := c.httpClient.Do(request)
	if err != nil {
		return Result{}, fmt.Errorf("call Nominatim: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("Nominatim returned status %d", response.StatusCode)
	}

	var payload struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&payload); err != nil {
		return Result{}, fmt.Errorf("decode Nominatim response: %w", err)
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = firstComponent(payload.DisplayName)
	}
	if name == "" {
		return Result{}, fmt.Errorf("Nominatim response did not contain a name")
	}
	raw, _ := json.Marshal(payload)
	return Result{DisplayName: name, Raw: raw}, nil
}

func firstComponent(displayName string) string {
	component, _, _ := strings.Cut(displayName, ",")
	return strings.TrimSpace(component)
}

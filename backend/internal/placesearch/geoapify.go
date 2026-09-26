package placesearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultProviderTimeout = 8 * time.Second
	maxProviderBodyBytes   = 1 << 20
	geoapifyMinimumDelay   = 250 * time.Millisecond
)

var ErrProviderUnavailable = errors.New("place search provider is temporarily unavailable")

type GeoapifyClient struct {
	baseURL      string
	apiKey       string
	httpClient   *http.Client
	mu           sync.Mutex
	lastRequest  time.Time
	minimumDelay time.Duration
}

func NewGeoapifyClient(baseURL, apiKey string, httpClient *http.Client) (*GeoapifyClient, error) {
	endpoint, err := url.ParseRequestURI(strings.TrimSpace(baseURL))
	if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" {
		return nil, fmt.Errorf("invalid Geoapify URL")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("Geoapify API key is required")
	}
	if httpClient == nil {
		// Do not wrap this transport with URL-recording telemetry: Geoapify authenticates
		// through a query parameter and the secret must never enter traces.
		httpClient = &http.Client{Timeout: defaultProviderTimeout}
	}
	return &GeoapifyClient{
		baseURL: strings.TrimRight(endpoint.String(), "/"), apiKey: apiKey,
		httpClient: httpClient, minimumDelay: geoapifyMinimumDelay,
	}, nil
}

func (c *GeoapifyClient) Name() string {
	return ProviderGeoapify
}

func (c *GeoapifyClient) Suggest(ctx context.Context, input SuggestInput) ([]Suggestion, error) {
	endpoint, err := url.Parse(c.baseURL + "/v1/geocode/autocomplete")
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	query := endpoint.Query()
	query.Set("format", "json")
	query.Set("text", input.Query)
	query.Set("limit", strconv.FormatInt(int64(input.Limit), 10))
	if input.AcceptLanguage != "" {
		query.Set("lang", input.AcceptLanguage)
	}
	query.Set("apiKey", c.apiKey)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	request.Header.Set("Accept", "application/json")

	if err := c.waitForSlot(ctx); err != nil {
		return nil, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		// A net/http error may contain the complete URL, including apiKey. Never wrap it.
		return nil, ErrProviderUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, ErrProviderUnavailable
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxProviderBodyBytes+1))
	if err != nil || len(body) > maxProviderBodyBytes {
		return nil, ErrProviderUnavailable
	}
	var payload geoapifyResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, ErrProviderUnavailable
	}

	results := make([]Suggestion, 0, len(payload.Results))
	for _, candidate := range payload.Results {
		if len(results) >= int(input.Limit) {
			break
		}
		if !validCoordinate(candidate.Latitude, candidate.Longitude) {
			continue
		}
		name := firstNonEmpty(candidate.Name, candidate.AddressLine1, firstComponent(candidate.Formatted))
		if name == "" {
			continue
		}
		resultID := strings.TrimSpace(candidate.PlaceID)
		if resultID == "" {
			hash := sha256.Sum256([]byte(fmt.Sprintf("%.7f\x00%.7f\x00%s", candidate.Latitude, candidate.Longitude, candidate.Formatted)))
			resultID = "geoapify:" + hex.EncodeToString(hash[:12])
		}
		results = append(results, Suggestion{
			ResultID: resultID, Name: name, Address: strings.TrimSpace(candidate.Formatted),
			Latitude: candidate.Latitude, Longitude: candidate.Longitude,
			Category: strings.TrimSpace(candidate.Category), Type: strings.TrimSpace(candidate.ResultType),
			BoundingBox: normalizedBounds(candidate.BoundingBox),
		})
	}
	return results, nil
}

func (c *GeoapifyClient) waitForSlot(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if wait := c.minimumDelay - time.Since(c.lastRequest); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	c.lastRequest = time.Now()
	return nil
}

type geoapifyResponse struct {
	Results []struct {
		Name         string  `json:"name"`
		AddressLine1 string  `json:"address_line1"`
		Formatted    string  `json:"formatted"`
		PlaceID      string  `json:"place_id"`
		Category     string  `json:"category"`
		ResultType   string  `json:"result_type"`
		Latitude     float64 `json:"lat"`
		Longitude    float64 `json:"lon"`
		BoundingBox  *struct {
			Longitude1 float64 `json:"lon1"`
			Latitude1  float64 `json:"lat1"`
			Longitude2 float64 `json:"lon2"`
			Latitude2  float64 `json:"lat2"`
		} `json:"bbox"`
	} `json:"results"`
}

func normalizedBounds(value *struct {
	Longitude1 float64 `json:"lon1"`
	Latitude1  float64 `json:"lat1"`
	Longitude2 float64 `json:"lon2"`
	Latitude2  float64 `json:"lat2"`
}) *BoundingBox {
	if value == nil || !validCoordinate(value.Latitude1, value.Longitude1) ||
		!validCoordinate(value.Latitude2, value.Longitude2) {
		return nil
	}
	return &BoundingBox{
		South: min(value.Latitude1, value.Latitude2), West: min(value.Longitude1, value.Longitude2),
		North: max(value.Latitude1, value.Latitude2), East: max(value.Longitude1, value.Longitude2),
	}
}

func validCoordinate(latitude, longitude float64) bool {
	return !math.IsNaN(latitude) && !math.IsNaN(longitude) && !math.IsInf(latitude, 0) && !math.IsInf(longitude, 0) &&
		latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func firstComponent(value string) string {
	component, _, _ := strings.Cut(value, ",")
	return strings.TrimSpace(component)
}

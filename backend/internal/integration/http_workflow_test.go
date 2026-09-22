//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/database/dbgen"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/httpapi"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/identity"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/journeys"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/objectstore"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/places"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/sharing"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/tracking"
)

func TestAuthenticatedJourneyHTTPWorkflow(t *testing.T) {
	pool := openTestPool(t)
	queries := dbgen.New(pool)
	router := httpapi.NewRouter(httpapi.RouterConfig{
		UserProvisioner: queries,
		ProfileStore:    queries,
		JourneyService:  journeys.NewService(pool),
		PlaceService:    places.NewService(pool),
		TrackingService: tracking.NewService(pool),
		SharingService:  sharing.NewService(queries, unusedObjectStore{}, "http://example.test"),
		Verifier:        identity.DevelopmentVerifier{},
	})

	ownerToken := "workflow-owner-" + uuid.NewString()
	otherToken := "workflow-other-" + uuid.NewString()
	var journeyID uuid.UUID
	t.Cleanup(func() {
		ctx := context.Background()
		if journeyID != uuid.Nil {
			_, _ = pool.Exec(ctx, "DELETE FROM background_jobs WHERE payload->>'journey_id' = $1", journeyID.String())
		}
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE oidc_subject IN ($1, $2)",
			"development:"+ownerToken, "development:"+otherToken)
	})

	clientRequestID := uuid.New()
	created := requestJSON[journeyHTTPResponse](t, router, http.MethodPost, "/api/v1/journeys", ownerToken, map[string]any{
		"client_request_id": clientRequestID,
		"kind":              "trip",
		"label":             "Integration Delhi ride",
	}, http.StatusCreated)
	journeyID = created.ID
	if journeyID == uuid.Nil || created.Status != "active" || created.Version != 1 {
		t.Fatalf("unexpected created journey: %#v", created)
	}

	repeated := requestJSON[journeyHTTPResponse](t, router, http.MethodPost, "/api/v1/journeys", ownerToken, map[string]any{
		"client_request_id": clientRequestID,
		"kind":              "trip",
		"label":             "Integration Delhi ride",
	}, http.StatusCreated)
	if repeated.ID != journeyID {
		t.Fatalf("idempotent journey create returned %s, want %s", repeated.ID, journeyID)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	stop := requestJSON[stopHTTPResponse](t, router, http.MethodPost,
		"/api/v1/journeys/"+journeyID.String()+"/stops", ownerToken, map[string]any{
			"client_request_id": uuid.New(),
			"name":              "India Gate",
			"note":              "Evening stop",
			"latitude":          28.6129,
			"longitude":         77.2295,
			"captured_at":       now,
		}, http.StatusCreated)
	if stop.JourneyID != journeyID || stop.ID == uuid.Nil || stop.SequenceNumber != 1 {
		t.Fatalf("unexpected pinned stop: %#v", stop)
	}

	ingested := requestJSON[tracking.IngestResult](t, router, http.MethodPost,
		"/api/v1/journeys/"+journeyID.String()+"/locations/batch", ownerToken, map[string]any{
			"points": []map[string]any{
				{
					"sample_id": uuid.New(), "captured_at": now.Add(-time.Minute),
					"latitude": 28.6119, "longitude": 77.2285, "accuracy_m": 8,
				},
				{
					"sample_id": uuid.New(), "captured_at": now,
					"latitude": 28.6129, "longitude": 77.2295, "accuracy_m": 8,
				},
			},
		}, http.StatusAccepted)
	if ingested.Received != 2 || ingested.Inserted != 2 || ingested.Duplicates != 0 {
		t.Fatalf("unexpected GPS ingest result: %#v", ingested)
	}

	var storedSamples int
	if err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM location_samples WHERE journey_id = $1", journeyID).Scan(&storedSamples); err != nil {
		t.Fatalf("count stored location samples: %v", err)
	}
	if storedSamples != 2 {
		t.Fatalf("stored location samples = %d, want 2", storedSamples)
	}

	status, body := performJSON(t, router, http.MethodGet,
		"/api/v1/journeys/"+journeyID.String(), otherToken, nil)
	if status != http.StatusNotFound {
		t.Fatalf("other user journey GET status = %d, want %d; body=%s", status, http.StatusNotFound, body)
	}

	ended := requestJSON[journeyHTTPResponse](t, router, http.MethodPost,
		"/api/v1/journeys/"+journeyID.String()+"/end", ownerToken,
		map[string]any{"version": created.Version}, http.StatusOK)
	if ended.Status != "ended" || ended.Version != created.Version+1 || ended.EndedAt == nil {
		t.Fatalf("unexpected ended journey: %#v", ended)
	}

	sharedName := "India Gate at sunset"
	share := requestJSON[shareHTTPResponse](t, router, http.MethodPost,
		"/api/v1/journeys/"+journeyID.String()+"/shares", ownerToken, map[string]any{
			"title": "Delhi evening ride",
			"items": []map[string]any{{
				"stop_id": stop.ID, "display_name": sharedName, "photo_ids": []uuid.UUID{},
			}},
		}, http.StatusCreated)
	shareURL, err := url.Parse(share.URL)
	if err != nil || share.ID == uuid.Nil || !strings.HasPrefix(share.Message, "Delhi evening ride") {
		t.Fatalf("unexpected created share: %#v (parse error: %v)", share, err)
	}
	token := strings.TrimPrefix(shareURL.Path, "/s/")
	if token == "" || token == shareURL.Path {
		t.Fatalf("share URL does not contain a public token: %q", share.URL)
	}

	public := requestJSON[sharing.PublicSnapshot](t, router, http.MethodGet,
		"/api/v1/shares/"+token, "", nil, http.StatusOK)
	if public.Title != "Delhi evening ride" || len(public.Items) != 1 ||
		public.Items[0].StopID != stop.ID || public.Items[0].DisplayName != sharedName {
		t.Fatalf("unexpected public snapshot: %#v", public)
	}

	pageStatus, page := performJSON(t, router, http.MethodGet, "/s/"+token, "", nil)
	if pageStatus != http.StatusOK || !bytes.Contains(page, []byte("Delhi evening ride")) ||
		!bytes.Contains(page, []byte(sharedName)) {
		t.Fatalf("public page status = %d or content missing", pageStatus)
	}
}

type journeyHTTPResponse struct {
	ID      uuid.UUID  `json:"id"`
	Status  string     `json:"status"`
	EndedAt *time.Time `json:"ended_at"`
	Version int32      `json:"version"`
}

type stopHTTPResponse struct {
	ID             uuid.UUID `json:"id"`
	JourneyID      uuid.UUID `json:"journey_id"`
	SequenceNumber int64     `json:"sequence_number"`
}

type shareHTTPResponse struct {
	ID      uuid.UUID `json:"id"`
	URL     string    `json:"url"`
	Message string    `json:"message"`
}

func requestJSON[T any](
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	token string,
	body any,
	wantStatus int,
) T {
	t.Helper()
	status, responseBody := performJSON(t, handler, method, path, token, body)
	if status != wantStatus {
		t.Fatalf("%s %s status = %d, want %d; body=%s", method, path, status, wantStatus, responseBody)
	}
	var result T
	if err := json.Unmarshal(responseBody, &result); err != nil {
		t.Fatalf("decode %s %s response: %v; body=%s", method, path, err, responseBody)
	}
	return result
}

func performJSON(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	token string,
	body any,
) (int, []byte) {
	t.Helper()
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode %s %s request: %v", method, path, err)
		}
		requestBody = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, requestBody)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder.Code, recorder.Body.Bytes()
}

type unusedObjectStore struct{}

func (unusedObjectStore) Get(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("object storage is not used by this workflow")
}

func (unusedObjectStore) Put(context.Context, string, io.Reader, int64, string) error {
	return errors.New("object storage is not used by this workflow")
}

func (unusedObjectStore) PresignPut(context.Context, string, time.Duration) (*url.URL, error) {
	return nil, errors.New("object storage is not used by this workflow")
}

func (unusedObjectStore) PresignGet(context.Context, string, time.Duration) (*url.URL, error) {
	return nil, errors.New("object storage is not used by this workflow")
}

func (unusedObjectStore) Stat(context.Context, string) (objectstore.ObjectInfo, error) {
	return objectstore.ObjectInfo{}, errors.New("object storage is not used by this workflow")
}

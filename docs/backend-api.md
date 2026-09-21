# Backend API Handbook

The authoritative machine-readable contract is [`backend/internal/httpapi/openapi.yaml`](../backend/internal/httpapi/openapi.yaml) and is served at `GET /openapi.yaml`. This handbook explains the intended client workflows and invariants.

## Conventions

- Authenticated base path: `/api/v1`
- Authentication: `Authorization: Bearer <access-token>`
- Content type: `application/json`
- Timestamps: RFC 3339 UTC on the wire
- IDs: UUIDs
- Errors: `{"error":{"code":"...","message":"...","request_id":"..."}}`
- Correlation: clients may send `X-Request-ID`; the API returns it

Public exceptions are `GET /api/v1/shares/{token}` for JSON and `GET /s/{token}` for the rendered page.

Offline-retryable creates carry a stable `client_request_id` in the JSON body. Retrying with the same authenticated owner and ID returns the same logical resource rather than duplicating it. Journey updates use the returned `version`; a stale version receives `409`.

A `429` response includes `Retry-After`. Current per-minute limits are 30 reverse-geocode requests, 30 upload reservations, and 60 nearby searches per authenticated user; public share reads allow 120 per client IP. These limits are per API process.

## Main workflow

### 1. Start a trip or outing

Creating a journey starts it immediately. Only one journey may be active per user.

```http
POST /api/v1/journeys
Authorization: Bearer alice
Content-Type: application/json

{
  "client_request_id": "4e004b9a-0e71-4aa0-aa0c-cbc95c7ced7e",
  "kind": "outing",
  "label": "Sunday ride"
}
```

The response includes an optimistic `version`. Send it to rename, end, or convert the journey. Active outings receive a durable warning job for hour 23 and expiry job for hour 24.

### 2. Capture route points

The client should persist points locally first and upload bounded batches of at most 500 samples. Each `sample_id` is stable across retries.

```http
POST /api/v1/journeys/{journeyId}/locations/batch

{
  "points": [{
    "sample_id": "dce4d2ca-e6e2-4cda-99eb-271a832d1a4e",
    "captured_at": "2026-09-21T07:15:30Z",
    "latitude": 28.6139,
    "longitude": 77.2090,
    "accuracy_m": 8.2,
    "speed_mps": 7.4,
    "heading_deg": 110
  }]
}
```

The API stores raw samples and queues asynchronous cleaning. `GET /journeys/{journeyId}/route` returns cleaned segments and uses map-matched geometry where available.

### 3. Pin a stop

```http
POST /api/v1/journeys/{journeyId}/stops

{
  "client_request_id": "d4394dc2-6243-48b2-b325-c47a2f913cdd",
  "name": "Lodhi Garden",
  "note": "Morning walk",
  "latitude": 28.5931,
  "longitude": 77.2197,
  "captured_at": "2026-09-21T08:00:00Z"
}
```

The server atomically creates a saved place and appends the stop sequence. Stop name and note remain editable after the journey ends, but its sequence, timestamp, and coordinates do not.

Use `GET /places/reverse-geocode?latitude=...&longitude=...` to propose a label. The user-entered name remains authoritative.

### 4. Attach a photo

Photo upload has three steps:

1. `POST /photos/upload-intents` with journey, optional stop, stable request ID, and MIME type.
2. `PUT` the bytes directly to the returned short-lived `upload_url`, using the declared `Content-Type`.
3. `POST /photos/{photoId}/complete`; processing happens asynchronously.

Supported types are JPEG, PNG, and WebP. The default maximum is 15 MiB. `GET /journeys/{journeyId}/stops/{stopId}/photos` returns processed photos with 15-minute private thumbnail URLs.

### 5. Finish or convert

- `POST /journeys/{journeyId}/convert-to-trip` changes an active outing into a trip.
- `POST /journeys/{journeyId}/end` preserves the timeline and stops further points/pins.

Both bodies are `{"version": <current-version>}`.

### 6. Publish an itinerary

```http
POST /api/v1/journeys/{journeyId}/shares

{
  "title": "Sunday ride",
  "expires_at": null,
  "items": [{
    "stop_id": "f22313a1-c541-4a14-a224-7f6eb333bd4e",
    "display_name": "Breakfast stop",
    "note": "Try the parathas",
    "photo_ids": []
  }]
}
```

`items` defines only the public snapshot order; it never rewrites the journey. Omitting `items` uses every stop in original timeline order. The response contains the share ID, public URL, and formatted text suitable for the platform share sheet or WhatsApp.

The owner can revoke with `DELETE /share-links/{shareId}`. Revoked, expired, or unknown links return `404`. Public tokens are stored only as SHA-256 hashes.

## Saved places and nearby search

`POST /places` saves a location without an active journey. Journey pins are also added to the same private place library.

```http
GET /api/v1/places/nearby?latitude=28.6139&longitude=77.2090&limit=10&mode=motorcycle
```

The default mode is `motorcycle`. PostGIS creates a bounded straight-line shortlist; Valhalla then calculates road distance/time for motorcycle, car, and walking. Successful results are ordered by road distance for the selected mode. If the selected road matrix is unavailable, the endpoint returns `503` instead of silently presenting straight-line distance as road distance.

## Endpoint groups

| Group | Endpoints |
| --- | --- |
| Profile/devices | `GET/PATCH /me`, `GET/POST /devices`, `DELETE /devices/{id}` |
| Journeys | `GET/POST /journeys`, `GET/PATCH /journeys/{id}`, convert, end |
| Stops/routes | Journey stops, location batches, derived route, stop photos |
| Places | Save, list, edit, soft-delete, reverse geocode, nearby |
| Media | Upload intent, upload completion, private stop gallery |
| Sharing | Create, resolve JSON/page, revoke |
| Platform | Liveness, readiness, metrics, OpenAPI |

## Client responsibilities

- Write pins, points, edits, and pending photo metadata to local storage before network calls.
- Reuse client request/sample IDs until acknowledged.
- Keep original photo bytes locally until completion is acknowledged.
- Pause retry queues on `401`; surface validation and conflict errors instead of retrying forever.
- Treat signed object URLs as ephemeral and never persist or share them as stable identifiers.
- Send the latest aggregate `version` for journey transitions.

# Pinerary

Pinerary is a travel and outing journal for recording a chronological place timeline, GPS route, and photos; finding saved places by road distance; and publishing revocable itinerary links.

The repository currently contains the completed backend foundation. The PWA is the next delivery phase, followed last by an optional Capacitor Android APK for reliable background tracking. No app-store distribution is required for the PWA or a sideloaded APK.

## Documentation

- [Architecture and detailed design](docs/architecture-design.md)
- [Readable architecture page](docs/architecture-design.html)
- [Backend setup](backend/README.md)
- [Backend API handbook](docs/backend-api.md)
- [Operations handbook](docs/operations.md)
- [Backend implementation history](docs/backend-implementation-plan.md)

The HTTP contract is served by a running API at `/openapi.yaml` and is stored at `backend/internal/httpapi/openapi.yaml`.

## Repository layout

```text
backend/   Go API, worker, migrations, tests, and OpenAPI contract
docs/      Architecture, API, and operations documentation
infra/     Local PostgreSQL/PostGIS and MinIO infrastructure
```

Start with [backend/README.md](backend/README.md) to run the system locally.

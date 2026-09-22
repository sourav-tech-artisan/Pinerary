# Pinerary

Pinerary is a travel and outing journal for recording a chronological place timeline, GPS route, and photos; finding saved places by road distance; and publishing revocable itinerary links.

The repository contains the Go backend foundation and an installable, offline-first Next.js PWA. Cross-browser/device hardening comes next, followed last by an optional Capacitor Android APK for reliable background tracking. No app-store distribution is required for the PWA or a sideloaded APK.

## Documentation

- [Architecture and detailed design](docs/architecture-design.md)
- [Readable architecture page](docs/architecture-design.html)
- [Backend detailed design and repository guide](docs/backend-detailed-design.md)
- [Readable backend design artifact](docs/backend-detailed-design.html)
- [PWA detailed design and repository guide](docs/pwa-detailed-design.md)
- [Readable PWA design artifact](docs/pwa-detailed-design.html)
- [PWA setup](web/README.md)
- [Backend setup](backend/README.md)
- [Backend API handbook](docs/backend-api.md)
- [Operations handbook](docs/operations.md)
- [Backend implementation history](docs/backend-implementation-plan.md)

The HTTP contract is served by a running API at `/openapi.yaml` and is stored at `backend/internal/httpapi/openapi.yaml`.

For manual testing, import the collection and local environment under `docs/postman/`.

## Repository layout

```text
backend/   Go API, worker, migrations, tests, and OpenAPI contract
web/       Static Next.js PWA, IndexedDB outbox, maps, and frontend tests
docs/      Architecture, API, and operations documentation
infra/     Local PostgreSQL/PostGIS and MinIO infrastructure
```

Start the backend with [backend/README.md](backend/README.md), then run the client using [web/README.md](web/README.md).

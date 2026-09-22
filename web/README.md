# Pinerary PWA

The browser client is a statically exported Next.js application. It talks directly to the Go API and keeps capture data in IndexedDB before attempting network synchronization.

## Run locally

Prerequisites: Node.js 24+, the backend dependencies, API, and worker described in `../backend/README.md`.

```bash
cp .env.example .env.local
npm install
npm run dev
```

Open `http://localhost:3000`. The default development bearer token is `alice`; both the token and API URL can be changed under Settings.

## Verification

```bash
npm run generate:api
npm run lint
npm run typecheck
npm test
npm run build
```

`npm run build` writes the static application to `out/`. Serve it over HTTPS outside localhost so installation, geolocation, camera capture, notifications, and service workers work correctly.

## Important behavior

- Dexie/IndexedDB is the immediate source of truth while offline.
- A durable client outbox retries idempotent Go API mutations.
- Foreground GPS capture follows the active journey across PWA screens.
- A browser may pause JavaScript when the PWA is backgrounded or the phone locks. Reliable background tracking belongs to the final Capacitor Android phase.
- OpenStreetMap tiles are used for the small development trial with attribution and a bounded runtime cache; bulk offline map downloads are not implemented.
- `src/lib/api/schema.d.ts` is generated from the backend OpenAPI contract and checked for drift in CI.

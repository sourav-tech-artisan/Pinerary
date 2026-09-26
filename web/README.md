# Pinerary PWA

The browser client is a statically exported Next.js application. It talks directly to the Go API and keeps capture data in IndexedDB before attempting network synchronization.

## Run locally

Prerequisites: Node.js 24+, the backend dependencies, API, and worker described in `../backend/README.md`.

```bash
cp .env.example .env.local
npm install
npm run dev
```

Open `http://localhost:3000` and use the Auth0 login or signup screen. The checked-in example contains Pinerary's public Auth0 domain, SPA client ID, and API audience; no client secret belongs in this application.

In the Auth0 SPA settings, allow `http://localhost:3000` as a callback URL, logout URL, and web origin. Enable refresh-token rotation for the SPA and **Allow Offline Access** for the API. The local Go API must run in `oidc` mode with the same issuer and audience.

For isolated backend/Postman development without Auth0, set `NEXT_PUBLIC_AUTH_MODE=development` and run the Go API with `PINERARY_AUTH_MODE=development`. Only that mode displays the editable development token.

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
- Advance-place autocomplete calls only the Go API. Configure the backend's Geoapify provider and key as described in `../backend/README.md`; no provider key belongs in the PWA.
- Discovery starts after three characters and a 450 ms pause. Selecting a suggestion opens an editable map preview, then the normal IndexedDB/outbox save path.
- Discovery needs connectivity, while saved-place filtering and already queued saves remain available offline. Typed discovery text is sent to Geoapify through the backend.
- Foreground GPS capture follows the active journey across PWA screens.
- A browser may pause JavaScript when the PWA is backgrounded or the phone locks. Reliable background tracking belongs to the final Capacitor Android phase.
- OpenStreetMap tiles are used for the small development trial with attribution and a bounded runtime cache; bulk offline map downloads are not implemented.
- `src/lib/api/schema.d.ts` is generated from the backend OpenAPI contract and checked for drift in CI.

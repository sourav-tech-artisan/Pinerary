# Pinerary Backend

The Pinerary backend is a Go HTTP API built with Gin.

## Requirements

- Go 1.24 or newer
- Docker with Compose for local infrastructure

## Local database

```bash
make db-up
make migrate
```

The default connection details match [`../infra/compose.yaml`](../infra/compose.yaml). Copy
`.env.example` when overriding local settings.

## Run

```bash
cd backend
go run ./cmd/api
```

The API listens on `:8080` by default. Override it with `PINERARY_HTTP_ADDR`:

```bash
PINERARY_HTTP_ADDR=127.0.0.1:9090 go run ./cmd/api
```

Check liveness:

```bash
curl http://localhost:8080/health/live
```

## Test

```bash
make test
```

`make fmt-check`, `make vet`, and `make lint` run the same quality gates used in CI.
SQL queries live in `internal/database/queries`; run `make generate` after changing them
or a migration. Generated query code is committed so normal builds do not require sqlc.

## Container

```bash
docker build -t pinerary-api .
docker run --rm -p 8080:8080 pinerary-api
```

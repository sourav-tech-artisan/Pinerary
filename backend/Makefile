.PHONY: build db-down db-up fmt fmt-check lint migrate run test vet

build:
	go build -o bin/pinerary-api ./cmd/api

db-down:
	docker compose -f ../infra/compose.yaml down

db-up:
	docker compose -f ../infra/compose.yaml up -d postgres

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))"

lint:
	golangci-lint run ./...

migrate:
	go run ./cmd/migrate up

run:
	go run ./cmd/api

test:
	go test -race -coverprofile=coverage.out ./...

vet:
	go vet ./...

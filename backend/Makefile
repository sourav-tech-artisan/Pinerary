.PHONY: build fmt fmt-check lint run test vet

build:
	go build -o bin/pinerary-api ./cmd/api

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))"

lint:
	golangci-lint run ./...

run:
	go run ./cmd/api

test:
	go test -race -coverprofile=coverage.out ./...

vet:
	go vet ./...

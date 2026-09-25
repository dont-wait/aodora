APP_NAME := aodora-api

.PHONY: run build test fmt lint migrate-up migrate-down migrate-up-one infra-up infra-down

run:
	go run ./cmd/api

build:
	go build -trimpath -ldflags="-s -w" -o bin/$(APP_NAME) ./cmd/api

test:
	go test -race ./...

fmt:
	gofumpt -w .

lint:
	golangci-lint run ./...

migrate-up:
	go run ./cmd/migrate -command up -path migrations

migrate-up-one:
	go run ./cmd/migrate -command up-by-one -path migrations

migrate-down:
	go run ./cmd/migrate -command down -path migrations

infra-up:
	docker compose up -d mongo

infra-down:
	docker compose down

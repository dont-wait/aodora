APP_NAME := aodora-api
NAVICAT_APPIMAGE ?= $(HOME)/Downloads/navicat18-mongodb-en-x86_64.AppImage
MIGRATIONS_PATH ?= migrations

.PHONY: run build test test-migrations fmt lint migrate-create migrate-up migrate-down migrate-up-one infra-up infra-down storage-up storage-down seed-fetch seed-demo navicat-build navicat-up navicat-down

run:
	go run ./cmd/api

build:
	go build -trimpath -ldflags="-s -w" -o bin/$(APP_NAME) ./cmd/api

test:
	go test -race ./...

test-migrations:
	@test -n "$${MONGO_TEST_URI:-}" || { echo "MONGO_TEST_URI is required"; exit 1; }
	go test -race -tags=integration ./internal/infrastructure/migration -count=1 -v

fmt:
	gofumpt -w .

lint:
	golangci-lint run ./...

migrate-up:
	go run ./cmd/migrate -command up -path "$(MIGRATIONS_PATH)"

migrate-up-one:
	go run ./cmd/migrate -command up-by-one -path "$(MIGRATIONS_PATH)"

migrate-down:
	go run ./cmd/migrate -command down -path "$(MIGRATIONS_PATH)"

migrate-create:
	go run ./cmd/migrate -command create -name "$(name)" -path "$(MIGRATIONS_PATH)"

infra-up:
	docker compose up -d mongo rustfs
	docker compose run --rm rustfs-init

storage-up:
	docker compose up -d rustfs
	docker compose run --rm rustfs-init

storage-down:
	docker compose stop rustfs

infra-down:
	docker compose down

seed-fetch:
	python3 seeds/fetch_images.py

seed-demo:
	$(MAKE) seed-fetch
	go run ./cmd/seed -check
	$(MAKE) infra-up
	$(MAKE) migrate-up
	docker compose --profile seed run --rm rustfs-seed
	go run ./cmd/seed

navicat-build:
	test -f "$(NAVICAT_APPIMAGE)"
	cp -- "$(NAVICAT_APPIMAGE)" docker/navicat/navicat.AppImage
	docker compose --profile tools build navicat

navicat-up: navicat-build
	docker compose --profile tools up -d --no-deps navicat

navicat-down:
	docker compose --profile tools stop navicat

APP_NAME := aodora-api
NAVICAT_APPIMAGE ?= $(HOME)/Downloads/navicat18-mongodb-en-x86_64.AppImage
MIGRATIONS_PATH ?= migrations
export AODORA_MIGRATION_DIRECTORY := $(value MIGRATIONS_PATH)
export AODORA_MIGRATION_NAME := $(value name)
export AODORA_LEGACY_FINGERPRINT := $(value legacy_fingerprint)
unexport name MIGRATIONS_PATH legacy_fingerprint

.PHONY: sumire-env sumire-fetch sumire-check sumire-images sumire-import sumire-purge run build test test-migrations test-orders standalone-up test-orders-standalone transactions-up transactions-down test-integration fmt lint migrate-create migrate-adopt migrate-up migrate-down migrate-up-one infra-up infra-down storage-up storage-down seed-fetch seed-demo navicat-build navicat-up navicat-down

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
	go run ./cmd/migrate -command up -path "$$AODORA_MIGRATION_DIRECTORY"

migrate-up-one:
	go run ./cmd/migrate -command up-by-one -path "$$AODORA_MIGRATION_DIRECTORY"

migrate-down:
	go run ./cmd/migrate -command down -path "$$AODORA_MIGRATION_DIRECTORY"

migrate-create:
	go run ./cmd/migrate -command create -name "$$AODORA_MIGRATION_NAME" -path "$$AODORA_MIGRATION_DIRECTORY"

migrate-adopt:
	go run ./cmd/migrate -command adopt-legacy -legacy-fingerprint "$$AODORA_LEGACY_FINGERPRINT" -path "$$AODORA_MIGRATION_DIRECTORY"

infra-up:
	docker compose up -d mongo rustfs
	docker compose run --rm mongo-init
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

# Sumire áo dài catalog (see ../specs/migrate-data.md). Data comes from the shared Drive folder.
sumire-fetch:
	@if command -v rclone >/dev/null 2>&1 && rclone listremotes | grep -q '^gdrive:$$'; then \
		rclone copy "gdrive:aodora-ao-dai-images" seeds/sumire --progress; \
	else \
		echo "Download the Drive folder 'aodora-ao-dai-images' (catalog.json + one folder per product) into server/seeds/sumire/,"; \
		echo "or configure an rclone remote named 'gdrive' and re-run. See seeds/README.md."; exit 1; \
	fi

sumire-check:
	go run ./cmd/importsumire -check

sumire-images:
	docker compose up -d rustfs
	docker compose run --rm rustfs-init
	docker compose --profile sumire run --rm rustfs-sumire

# MONGO_URI defaults to an unauthenticated URL, so a missing .env makes migrate/import fail with "requires authentication".
sumire-env:
	@test -f .env || { cp .env.example .env && echo "Created .env from .env.example (local defaults)"; }

sumire-import: sumire-env
	go run ./cmd/importsumire -verify-target
	$(MAKE) sumire-check
	$(MAKE) infra-up
	$(MAKE) migrate-up
	docker compose --profile sumire run --rm rustfs-sumire
	go run ./cmd/importsumire

sumire-purge:
	go run ./cmd/importsumire -purge -yes

navicat-build:
	test -f "$(NAVICAT_APPIMAGE)"
	cp -- "$(NAVICAT_APPIMAGE)" docker/navicat/navicat.AppImage
	docker compose --profile tools build navicat

navicat-up: navicat-build
	docker compose --profile tools up -d --no-deps navicat

navicat-down:
	docker compose --profile tools stop navicat

# Isolated transaction-capable database for DW integration tests.
transactions-up:
	docker compose -f docker-compose.transactions.yml up -d --wait mongo
	docker compose -f docker-compose.transactions.yml run --rm mongo-init

transactions-down:
	docker compose -f docker-compose.transactions.yml down

test-orders:
	@test -n "$${MONGO_TEST_URI:-}" || { echo "MONGO_TEST_URI is required"; exit 1; }
	go test -race -tags=integration ./internal/infrastructure/mongo ./internal/interfaces/http -count=1 -v

standalone-up:
	docker compose -f docker-compose.transactions.yml up -d --wait mongo-standalone

test-orders-standalone:
	@test -n "$${MONGO_STANDALONE_TEST_URI:-}" || { echo "MONGO_STANDALONE_TEST_URI is required"; exit 1; }
	go test -race -tags=integration_standalone ./internal/infrastructure/mongo -count=1 -v

test-integration: test-migrations test-orders

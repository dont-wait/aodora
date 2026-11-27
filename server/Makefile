APP_NAME := aodora-api
NAVICAT_APPIMAGE ?= $(HOME)/Downloads/navicat18-mongodb-en-x86_64.AppImage
MIGRATIONS_PATH ?= migrations
export AODORA_MIGRATION_DIRECTORY := $(value MIGRATIONS_PATH)
export AODORA_MIGRATION_NAME := $(value name)
export AODORA_LEGACY_FINGERPRINT := $(value legacy_fingerprint)
unexport name MIGRATIONS_PATH legacy_fingerprint

.DEFAULT_GOAL := help

.PHONY: help run build fmt lint test test-migrations test-orders test-orders-standalone test-integration \
	migrate-create migrate-adopt migrate-up migrate-up-one migrate-down \
	infra-up infra-down storage-up storage-down transactions-up transactions-down standalone-up \
	seed-fetch seed-demo seed-admin seed-orders seed-orders-purge sumire-env sumire-fetch sumire-check sumire-images sumire-import sumire-purge \
	navicat-build navicat-up navicat-down

##@ Chung

help: ## Liệt kê các lệnh
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make \033[36m<target>\033[0m\n"} \
		/^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5)} \
		/^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ Server

run: ## Chạy API server
	go run ./cmd/api

build: ## Build binary vào bin/
	go build -trimpath -ldflags="-s -w" -o bin/$(APP_NAME) ./cmd/api

fmt: ## Format code bằng gofumpt
	gofumpt -w .

lint: ## Chạy golangci-lint
	golangci-lint run ./...

##@ Test

test: ## Chạy unit test (race)
	go test -race ./...

test-migrations: ## Integration test migration (cần MONGO_TEST_URI)
	@test -n "$${MONGO_TEST_URI:-}" || { echo "MONGO_TEST_URI is required"; exit 1; }
	go test -race -tags=integration ./internal/infrastructure/migration -count=1 -v

test-orders: ## Integration test đơn hàng (cần MONGO_TEST_URI)
	@test -n "$${MONGO_TEST_URI:-}" || { echo "MONGO_TEST_URI is required"; exit 1; }
	go test -race -tags=integration ./internal/infrastructure/mongo ./internal/interfaces/http -count=1 -v

test-orders-standalone: ## Integration test standalone (cần MONGO_STANDALONE_TEST_URI)
	@test -n "$${MONGO_STANDALONE_TEST_URI:-}" || { echo "MONGO_STANDALONE_TEST_URI is required"; exit 1; }
	go test -race -tags=integration_standalone ./internal/infrastructure/mongo -count=1 -v

test-integration: test-migrations test-orders ## Chạy toàn bộ integration test

##@ Migration

migrate-up: ## Áp dụng toàn bộ migration
	go run ./cmd/migrate -command up -path "$$AODORA_MIGRATION_DIRECTORY"

migrate-up-one: ## Áp dụng 1 migration
	go run ./cmd/migrate -command up-by-one -path "$$AODORA_MIGRATION_DIRECTORY"

migrate-down: ## Rollback migration
	go run ./cmd/migrate -command down -path "$$AODORA_MIGRATION_DIRECTORY"

migrate-create: ## Tạo migration mới (make migrate-create name=...)
	go run ./cmd/migrate -command create -name "$$AODORA_MIGRATION_NAME" -path "$$AODORA_MIGRATION_DIRECTORY"

migrate-adopt: ## Adopt legacy (make migrate-adopt legacy_fingerprint=...)
	go run ./cmd/migrate -command adopt-legacy -legacy-fingerprint "$$AODORA_LEGACY_FINGERPRINT" -path "$$AODORA_MIGRATION_DIRECTORY"

##@ Docker / hạ tầng

infra-up: ## Bật mongo + rustfs
	docker compose up -d mongo rustfs
	docker compose run --rm mongo-init
	docker compose run --rm rustfs-init

infra-down: ## Tắt toàn bộ docker compose
	docker compose down

storage-up: ## Bật rustfs
	docker compose up -d rustfs
	docker compose run --rm rustfs-init

storage-down: ## Dừng rustfs
	docker compose stop rustfs

# Isolated transaction-capable database for DW integration tests.
transactions-up: ## Bật Mongo replica set cho integration test
	docker compose -f docker-compose.transactions.yml up -d --wait mongo
	docker compose -f docker-compose.transactions.yml run --rm mongo-init

transactions-down: ## Tắt Mongo test
	docker compose -f docker-compose.transactions.yml down

standalone-up: ## Bật Mongo standalone
	docker compose -f docker-compose.transactions.yml up -d --wait mongo-standalone

##@ Seed demo

seed-fetch: ## Tải ảnh seed demo
	python3 seeds/fetch_images.py

seed-admin: sumire-env ## Tạo tài khoản demo admin@aodora.local + user@aodora.local (mật khẩu password@123), không đụng catalog
	go run ./cmd/importsumire -verify-target
	go run ./cmd/seed -accounts-only

SEED_FROM ?= 2026-11-21
seed-orders: ## Sinh lịch sử mua hàng giả lập từ 21/11/2026 tới hiện tại cho dashboard (chạy lại sẽ thay batch cũ; SEED_FROM=YYYY-MM-DD để đổi ngày bắt đầu)
	go run ./cmd/seedorders -from "$(SEED_FROM)"

seed-orders-purge: ## Xoá lịch sử mua hàng giả lập
	go run ./cmd/seedorders -purge

seed-demo: ## Seed dữ liệu demo
	$(MAKE) seed-fetch
	go run ./cmd/seed -check
	$(MAKE) infra-up
	$(MAKE) migrate-up
	docker compose --profile seed run --rm rustfs-seed
	go run ./cmd/seed

##@ Sumire (catalog áo dài, xem ../specs/migrate-data.md)

# MONGO_URI defaults to an unauthenticated URL, so a missing .env makes migrate/import fail with "requires authentication".
sumire-env: ## Tạo .env từ .env.example nếu thiếu
	@test -f .env || { cp .env.example .env && echo "Created .env from .env.example (local defaults)"; }

# Data comes from the shared Drive folder.
sumire-fetch: ## Tải dữ liệu catalog từ Drive
	@if command -v rclone >/dev/null 2>&1 && rclone listremotes | grep -q '^gdrive:$$'; then \
		rclone copy "gdrive:aodora-ao-dai-images" seeds/sumire --progress; \
	else \
		echo "Download the Drive folder 'aodora-ao-dai-images' (catalog.json + one folder per product) into server/seeds/sumire/,"; \
		echo "or configure an rclone remote named 'gdrive' and re-run. See seeds/README.md."; exit 1; \
	fi

sumire-check: ## Kiểm tra dữ liệu catalog
	go run ./cmd/importsumire -check

sumire-images: ## Upload ảnh lên rustfs
	docker compose up -d rustfs
	docker compose run --rm rustfs-init
	docker compose --profile sumire run --rm rustfs-sumire

sumire-import: sumire-env ## Import catalog đầy đủ (env, check, infra, migrate, ảnh, data)
	go run ./cmd/importsumire -verify-target
	$(MAKE) sumire-check
	$(MAKE) infra-up
	$(MAKE) migrate-up
	docker compose --profile sumire run --rm rustfs-sumire
	go run ./cmd/importsumire

sumire-purge: ## Xoá dữ liệu Sumire đã import
	go run ./cmd/importsumire -purge -yes

##@ Công cụ

navicat-build: ## Build image Navicat
	test -f "$(NAVICAT_APPIMAGE)"
	cp -- "$(NAVICAT_APPIMAGE)" docker/navicat/navicat.AppImage
	docker compose --profile tools build navicat

navicat-up: navicat-build ## Chạy Navicat
	docker compose --profile tools up -d --no-deps navicat

navicat-down: ## Dừng Navicat
	docker compose --profile tools stop navicat

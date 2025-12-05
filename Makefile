.PHONY: help up up-build down reset logs sqlc-gen sqlc-gen-auth sqlc-gen-product sqlc-gen-order migrate-up migrate-down test

# ===================
# Help
# ===================
help:
	@echo "Available commands:"
	@echo "Code Generation:"
	@echo "  sqlc-gen          - Generate sqlc code for all services"
	@echo ""
	@echo "Docker Compose:"
	@echo "  up                - Start services (reuse images) + migrate"
	@echo "  up-build          - Start services (rebuild images) + migrate"
	@echo "  down              - Stop services"
	@echo "  reset             - Stop services and remove volumes"
	@echo "  logs              - Follow service logs"
	@echo ""
	@echo "Database:"
	@echo "  migrate-up        - Run all migrations"
	@echo "  migrate-down      - Rollback all migrations"
	@echo ""
	@echo ""
	@echo "Testing:"
	@echo "  test              - Run integration tests"

# ===================
# sqlc Code Generation
# ===================
sqlc-gen: sqlc-gen-auth sqlc-gen-product sqlc-gen-order

sqlc-gen-auth:
	cd services/auth && sqlc generate

sqlc-gen-product:
	cd services/product && sqlc generate

sqlc-gen-order:
	cd services/order && sqlc generate

# ===================
# Database Migration
# ===================
DB_URL_BASE ?= postgres://flash:flash1234@postgres:5432/flash_deals?sslmode=disable

migrate-up: migrate-up-auth migrate-up-product migrate-up-order

migrate-up-auth:
	docker compose run --rm migrate -path=/migrations/auth -database="$(DB_URL_BASE)&search_path=auth" up

migrate-up-product:
	docker compose run --rm migrate -path=/migrations/product -database="$(DB_URL_BASE)&search_path=product" up

migrate-up-order:
	docker compose run --rm migrate -path=/migrations/order -database="$(DB_URL_BASE)&search_path=orders" up

migrate-down: migrate-down-auth migrate-down-product migrate-down-order

migrate-down-auth:
	docker compose run --rm migrate -path=/migrations/auth -database="$(DB_URL_BASE)&search_path=auth" down -all

migrate-down-product:
	docker compose run --rm migrate -path=/migrations/product -database="$(DB_URL_BASE)&search_path=product" down -all

migrate-down-order:
	docker compose run --rm migrate -path=/migrations/order -database="$(DB_URL_BASE)&search_path=orders" down -all

# ===================
# Docker Compose
# ===================
up:
	docker compose up -d --wait
	$(MAKE) migrate-up

up-build:
	docker compose up -d --build --wait
	$(MAKE) migrate-up

down:
	docker compose down

reset:
	docker compose down -v

logs:
	docker compose logs -f

# ===================
# Integration Test
# ===================
test:
	@test -d tests/integration/.venv || (cd tests/integration && uv sync)
	cd tests/integration && uv run pytest -v

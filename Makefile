.PHONY: help sqlc-gen sqlc-gen-auth sqlc-gen-product sqlc-gen-order migrate-up migrate-down

# ===================
# Help
# ===================
help:
	@echo "Available commands:"
	@echo "  sqlc-gen          - Generate sqlc code for all services"
	@echo "  sqlc-gen-auth     - Generate sqlc code for auth service"
	@echo "  sqlc-gen-product  - Generate sqlc code for product service"
	@echo "  sqlc-gen-order    - Generate sqlc code for order service"
	@echo "  migrate-up        - Run all migrations"
	@echo "  migrate-down      - Rollback all migrations"

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
DB_URL ?= postgres://flash:flash1234@localhost:5432/flash_deals?sslmode=disable

migrate-up: migrate-up-auth migrate-up-product migrate-up-order

migrate-up-auth:
	docker compose run --rm migrate -path=/migrations/auth -database="$(DB_URL)" up

migrate-up-product:
	docker compose run --rm migrate -path=/migrations/product -database="$(DB_URL)" up

migrate-up-order:
	docker compose run --rm migrate -path=/migrations/order -database="$(DB_URL)" up

migrate-down: migrate-down-order migrate-down-product migrate-down-auth

migrate-down-auth:
	docker compose run --rm migrate -path=/migrations/auth -database="$(DB_URL)" down -all

migrate-down-product:
	docker compose run --rm migrate -path=/migrations/product -database="$(DB_URL)" down -all

migrate-down-order:
	docker compose run --rm migrate -path=/migrations/order -database="$(DB_URL)" down -all

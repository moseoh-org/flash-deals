.PHONY: help up up-build down reset logs up-monitoring down-monitoring logs-monitoring up up-build down reset logs sqlc-gen sqlc-gen-auth sqlc-gen-product sqlc-gen-order migrate-up migrate-down test

# ===================
# Help
# ===================
help:
	@echo "Available commands:"
	@echo "Code Generation:"
	@echo "  sqlc-gen          - Generate sqlc code for all services"
	@echo ""
	@echo "  Docker (Main Services):"
	@echo "    up              - Start all services"
	@echo "    up-build        - Build and start all services"
	@echo "    down            - Stop all services"
	@echo "    reset           - Stop services and remove volumes"
	@echo "    logs            - Follow logs from all services"
	@echo ""
	@echo "  Docker (Monitoring - OTel + Grafana):"
	@echo "    up-monitoring   - Start monitoring stack (Tempo, Prometheus, Grafana)"
	@echo "    down-monitoring - Stop monitoring stack"
	@echo "    reset-monitoring - Stop monitoring stack and remove volumes"
	@echo "    logs-monitoring - Follow logs from monitoring services"
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
# Docker (Main Services)
# ===================
up:
	docker compose up -d --wait

up-build:
	docker compose up -d --build --wait

down:
	docker compose down

reset:
	docker compose down -v

logs:
	docker compose logs -f

# ===================
# Docker (Monitoring)
# ===================
up-monitoring:
	docker compose -f docker-compose.monitoring.yml up -d --wait

down-monitoring:
	docker compose -f docker-compose.monitoring.yml down

reset-monitoring:
	docker compose -f docker-compose.monitoring.yml down -v

logs-monitoring:
	docker compose -f docker-compose.monitoring.yml logs -f

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
# Integration Test
# ===================
test:
	@test -d tests/integration/.venv || (cd tests/integration && uv sync)
	cd tests/integration && uv run pytest -v

# ===================
# Load Test (k6)
# ===================
K6_SCRIPT ?= product-insert
K6_PRODUCTS ?= 1000

load:
	k6 run --env BASE_URL=http://localhost:8000 \
		--env TOTAL_PRODUCTS=$(K6_PRODUCTS) \
		tests/load/scripts/$(K6_SCRIPT).js

load-1k:
	$(MAKE) load K6_PRODUCTS=1000

load-5k:
	$(MAKE) load K6_PRODUCTS=5000

load-10k:
	$(MAKE) load K6_PRODUCTS=10000

# 상품 목록 조회 부하 테스트
load-list:
	k6 run --env BASE_URL=http://localhost:8000 \
		--env DURATION=$(or $(DURATION),30s) \
		--env VUS=$(or $(VUS),10) \
		tests/load/scripts/product-list.js

# 주문 목록 조회 부하 테스트
load-order-list:
	k6 run --env BASE_URL=http://localhost:8000 \
		--env DURATION=$(or $(DURATION),30s) \
		--env VUS=$(or $(VUS),10) \
		--env ORDERS_PER_USER=$(or $(ORDERS),50) \
		tests/load/scripts/order-list.js

# 핫딜 트래픽 급증 테스트
load-deal-spike:
	k6 run --env BASE_URL=http://localhost:8000 \
		--env MAX_VUS=$(or $(MAX_VUS),100) \
		--env RAMP_DURATION=$(or $(RAMP_DURATION),30s) \
		--env HOLD_DURATION=$(or $(HOLD_DURATION),30s) \
		tests/load/scripts/deal-spike.js

# 동시 주문 재고 초과 테스트
load-concurrent-order:
	k6 run --env BASE_URL=http://localhost:8000 \
		--env CONCURRENT_USERS=$(or $(CONCURRENT_USERS),50) \
		--env INITIAL_STOCK=$(or $(INITIAL_STOCK),10) \
		tests/load/scripts/concurrent-order.js

# 인증 CPU 병목 테스트
load-auth-stress:
	k6 run --env BASE_URL=http://localhost:8000 \
		--env MAX_VUS=$(or $(MAX_VUS),100) \
		--env RAMP_DURATION=$(or $(RAMP_DURATION),30s) \
		--env HOLD_DURATION=$(or $(HOLD_DURATION),30s) \
		tests/load/scripts/auth-stress.js

# ===================
# Load Test Seeds
# ===================
seed-products:
	@echo "Seeding 1,000,000 products (this may take a few minutes)..."
	docker compose exec -T postgres psql -U flash -d flash_deals -f /dev/stdin < tests/load/seeds/product-seed.sql

seed-products-10m:
	@echo "Seeding 10,000,000 products (this may take 10-20 minutes)..."
	docker compose exec -T postgres psql -U flash -d flash_deals -f /dev/stdin < tests/load/seeds/product-seed-10m.sql

seed-products-clean:
	@echo "Truncating products table..."
	docker compose exec postgres psql -U flash -d flash_deals -c "TRUNCATE product.products CASCADE;"

.PHONY: dev dev-backend dev-frontend test test-backend test-backend-integration test-e2e build lint db-up db-down db-test-setup docker-up docker-down docker-build

# Infrastructure
db-up:
	docker compose up -d postgres redis

db-down:
	docker compose down

# Create the integration test database (run once after db-up)
db-test-setup: db-up
	@echo "Creating test database (if not exists)..."
	@docker compose exec -T postgres psql -U veilence -d veilence_mx -c "SELECT 1 FROM pg_database WHERE datname = 'veilence_mx_test'" | grep -q 1 || \
		docker compose exec -T postgres psql -U veilence -d veilence_mx -c "CREATE DATABASE veilence_mx_test;"
	@echo "Test database ready."

# Docker (full stack)
docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-build:
	docker compose build

docker-prod-up:
	docker compose -f docker-compose.prod.yml up -d --build

docker-prod-down:
	docker compose -f docker-compose.prod.yml down

# copilot-api (LLM proxy)
dev-copilot:
	npx copilot-api start --github-token "$$(gh auth token)" --port 4141 --rate-limit 5 --wait

# Backend
dev-backend:
	cd backend && go run ./cmd/server/

build-backend:
	cd backend && go build ./...

test-backend:
	cd backend && go test ./... -count=1

test-backend-integration: db-test-setup
	cd backend && go test -tags=integration ./... -v -count=1

lint-backend:
	cd backend && go vet ./...

# Frontend
dev-frontend:
	cd frontend && npm run dev

build-frontend:
	cd frontend && npm run build

lint-frontend:
	cd frontend && npm run lint

# E2E
test-e2e:
	cd e2e && npx playwright test

test-e2e-ui:
	cd e2e && npx playwright test --ui

# Combined
dev: db-up
	@echo "Starting backend and frontend..."
	@make dev-backend &
	@make dev-frontend

test: test-backend

build: build-backend build-frontend

lint: lint-backend lint-frontend

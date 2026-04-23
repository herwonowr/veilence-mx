.PHONY: dev dev-backend dev-frontend build lint db-up db-down db-destroy docker-up docker-down docker-destroy docker-build

# Infrastructure
db-up:
	docker compose up -d postgres redis

db-down:
	docker compose down

db-destroy:
	docker compose down -v

# Docker (full stack)
docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-destroy:
	docker compose down -v

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

lint-backend:
	cd backend && go vet ./...

# Frontend
dev-frontend:
	cd frontend && npm run dev

build-frontend:
	cd frontend && npm run build

lint-frontend:
	cd frontend && npm run lint

# Combined
dev: db-up
	@echo "Starting backend and frontend..."
	@make dev-backend &
	@make dev-frontend

build: build-backend build-frontend

lint: lint-backend lint-frontend

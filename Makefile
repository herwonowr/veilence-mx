.PHONY: dev dev-backend dev-frontend test test-backend test-e2e build lint db-up db-down

# Infrastructure
db-up:
	docker-compose up -d postgres

db-down:
	docker-compose down

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

test-backend-integration:
	cd backend && go test -tags=integration ./... -v

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

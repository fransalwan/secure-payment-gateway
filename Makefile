.PHONY: all build frontend-build frontend-dev run worker test clean docker-up docker-down prod-up prod-down prod-logs k6 tunnel tunnel-ngrok

all: frontend-build build

frontend-build:
	@echo "Building Vue 3 + TypeScript + Pinia Dashboard..."
	@cd frontend && npm run build

frontend-dev:
	@cd frontend && npm run dev

build:
	@echo "Building API server and Worker..."
	@go build -o bin/api.exe ./cmd/api
	@go build -o bin/worker.exe ./cmd/worker

run:
	@go run ./cmd/api

worker:
	@go run ./cmd/worker

test:
	@go test -v -race ./...

docker-up:
	@docker compose up -d

docker-down:
	@docker compose down

prod-up:
	@docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --build --remove-orphans

prod-down:
	@docker compose --env-file .env.prod -f docker-compose.prod.yml down

prod-logs:
	@docker compose --env-file .env.prod -f docker-compose.prod.yml logs -f oms-api oms-worker caddy

k6:
	@k6 run scripts/k6_load_test.js

tunnel:
	@powershell -ExecutionPolicy Bypass -File scripts/tunnel.ps1

tunnel-ngrok:
	@powershell -ExecutionPolicy Bypass -File scripts/tunnel.ps1 -Provider ngrok


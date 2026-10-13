.PHONY: all build run worker test clean docker-up docker-down

all: build

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

k6:
	@k6 run scripts/k6_load_test.js


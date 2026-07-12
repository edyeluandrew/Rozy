.PHONY: help run migrate migrate-down test test-api build redis

help:
	@echo Rozy backend commands (run from backend/):
	@echo   make run          - start Go API server
	@echo   make migrate      - apply database migrations
	@echo   make migrate-down - roll back one migration
	@echo   make test         - run Go unit tests
	@echo   make test-api     - integration test all HTTP APIs (API must be running)
	@echo   make build        - build API binary
	@echo   make redis        - start local Redis via Docker
	@echo Full stack: run "cd .. ; make dev" from repo root

run:
	go run ./cmd/api

migrate:
	go run ./cmd/migrate

migrate-down:
	go run ./cmd/migrate --down

test:
	go test ./...

test-api:
	go run ./cmd/testapi

build:
	go build -o bin/rozy-api.exe ./cmd/api

redis:
	docker run -d --name rozy-redis -p 6379:6379 redis:7-alpine || docker start rozy-redis

.PHONY: help run migrate migrate-down test test-api build admin dev redis

BACKEND_DIR := backend
ADMIN_DIR := admin
MOBILE_DIR := mobile

help:
	@echo Rozy dev commands (run from repo root):
	@echo   make run          - start Go API server
	@echo   make migrate      - apply database migrations
	@echo   make migrate-down - roll back one migration
	@echo   make test         - run Go unit tests
	@echo   make test-api     - integration test all HTTP APIs (API must be running)
	@echo   make build        - build API binary
	@echo   make admin        - start React admin dashboard
	@echo   make dev          - tips for running full stack
	@echo   make redis        - start local Redis via Docker
	@echo.
	@echo From backend/: cd backend && make run  (same commands)

run:
	cd $(BACKEND_DIR) && go run ./cmd/api

migrate:
	cd $(BACKEND_DIR) && go run ./cmd/migrate

migrate-down:
	cd $(BACKEND_DIR) && go run ./cmd/migrate --down

test:
	cd $(BACKEND_DIR) && go test ./...

test-api:
	cd $(BACKEND_DIR) && go run ./cmd/testapi

build:
	cd $(BACKEND_DIR) && go build -o bin/rozy-api.exe ./cmd/api

admin:
	cd $(ADMIN_DIR) && npm run dev

dev:
	@echo 1. make redis     (once, if no Redis yet)
	@echo 2. make migrate   (first time / after schema changes)
	@echo 3. make run       (API on :8080, /dev tester)
	@echo 4. make admin     (dashboard on :5173)
	@echo Flutter: cd mobile && flutter run -t lib/main_passenger.dart

redis:
	docker run -d --name rozy-redis -p 6379:6379 redis:7-alpine || docker start rozy-redis

# Claudeee Project Makefile
# Development and build automation for backend (Go) and frontend (Next.js)

.PHONY: all build run dev test clean help install deps backend-build backend-run backend-dev backend-test frontend-build frontend-run frontend-dev frontend-test frontend-install

# Default target
all: build

# Help target
help:
	@echo "Available targets:"
	@echo "  all                 - Build both backend and frontend"
	@echo "  build               - Build both backend and frontend"
	@echo "  run                 - Run both backend and frontend in production mode"
	@echo "  dev                 - Run both backend and frontend in development mode"
	@echo "  test                - Run tests for both backend and frontend"
	@echo "  clean               - Clean build artifacts"
	@echo "  install             - Install dependencies for both backend and frontend"
	@echo ""
	@echo "Backend targets:"
	@echo "  backend-build       - Build backend binary"
	@echo "  backend-run         - Run backend in production mode"
	@echo "  backend-dev         - Run backend in development mode"
	@echo "  backend-test        - Run backend tests"
	@echo ""
	@echo "Frontend targets:"
	@echo "  frontend-build      - Build frontend for production"
	@echo "  frontend-run        - Run frontend in production mode"
	@echo "  frontend-dev        - Run frontend in development mode"
	@echo "  frontend-test       - Run frontend linting"
	@echo "  frontend-install    - Install frontend dependencies"

# Combined targets
build: backend-build frontend-build

run: backend-run frontend-run

dev:
	@echo "Starting development servers..."
	@echo "Backend will run on http://localhost:8080"
	@echo "Frontend will run on http://localhost:3000"
	@echo "Press Ctrl+C to stop both servers"
	@(make backend-dev &) && (make frontend-dev &) && wait

test: backend-test frontend-test

clean:
	@echo "Cleaning build artifacts..."
	cd backend && rm -rf bin/
	cd frontend && rm -rf .next/ out/
	@echo "Clean completed"

install: frontend-install
	@echo "Installing backend dependencies..."
	cd backend && go mod download && go mod tidy
	@echo "All dependencies installed"

# Backend targets
backend-build:
	@echo "Building backend..."
	cd backend && go build -o bin/server cmd/server/main.go
	@echo "Backend build completed: backend/bin/server"

backend-run: backend-build
	@echo "Starting backend server on http://localhost:8080"
	cd backend && ./bin/server

backend-dev:
	@echo "Starting backend in development mode..."
	cd backend && go run cmd/server/main.go

backend-test:
	@echo "Running backend tests..."
	cd backend && go test ./...

# Frontend targets
frontend-install:
	@echo "Installing frontend dependencies..."
	cd frontend && npm install --legacy-peer-deps

frontend-build: frontend-install
	@echo "Building frontend for production..."
	cd frontend && npm run build

frontend-run: frontend-build
	@echo "Starting frontend in production mode on http://localhost:3000"
	cd frontend && npm run start

frontend-dev: frontend-install
	@echo "Starting frontend in development mode on http://localhost:3000"
	cd frontend && npm run dev

frontend-test: frontend-install
	@echo "Running frontend linting..."
	cd frontend && npm run lint

# Dependency management
deps: install

# Development helpers
.PHONY: backend-logs frontend-logs
backend-logs:
	@echo "Checking backend logs (if any)..."

frontend-logs:
	@echo "Checking frontend logs (if any)..."

# Quick start for development
.PHONY: start
start: dev

# Production deployment preparation
.PHONY: prod-build
prod-build:
	@echo "Building for production..."
	@make clean
	@make build
	@echo "Production build completed"

# Fix session times
.PHONY: fix-session-times
fix-session-times:
	@echo "Fixing session start times..."
	cd backend && go run cmd/fix-session-times/main.go


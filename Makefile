# Main Makefile to control backend and frontend builds

.PHONY: \
    backend-build \
    backend-run \
    backend-clean \
    backend-stop \
    frontend-install \
    frontend-start \
    frontend-build \
    frontend-test \
    build \
    run \
    frontend-stop \
    stop \
    kill \
    help \
    docker-build \
    docker-push \
    docker-publish

# Backend binary output directory (relative to backend/)
BACKEND_OUT_DIR ?= bin

# -- Backend Targets --
backend-build:
	@echo "--- Building Backend ---"
	@$(MAKE) -C backend build OUT_DIR=$(BACKEND_OUT_DIR)

backend-run:
	@echo "--- Running Backend ---"
	@$(MAKE) -C backend run OUT_DIR=$(BACKEND_OUT_DIR)

backend-clean:
	@echo "--- Cleaning Backend ---"
	@$(MAKE) -C backend clean OUT_DIR=$(BACKEND_OUT_DIR)

backend-test:
	@echo "--- Testing Backend ---"
	@go test ./backend/...

# Stop the running backend process (best-effort)
backend-stop:
	@echo "--- Stopping Backend ---"
	# Try killing by binary name
	-@pkill -f '(^|/)quimera-cards( |$)' 2>/dev/null || true
	# Try killing by port (Linux)
	-@fuser -k 8080/tcp 2>/dev/null || true
	# Fallback using lsof if available
	-@if command -v lsof >/dev/null 2>&1; then \
	  PIDS=$$(lsof -ti:8080); \
	  [ -z "$$PIDS" ] || kill $$PIDS 2>/dev/null || true; \
	fi

# -- Frontend Targets --
frontend-install:
	@echo "--- Installing Frontend Dependencies ---"
	@$(MAKE) -C frontend install

frontend-start:
	@echo "--- Starting Frontend ---"
	@$(MAKE) -C frontend start

frontend-build:
	@echo "--- Building Frontend ---"
	@$(MAKE) -C frontend build

frontend-test:
	@echo "--- Testing Frontend ---"
	@$(MAKE) -C frontend test

# Run all tests (backend + frontend)
test: backend-test frontend-test


# Stop the React dev server (best-effort)
frontend-stop:
	@echo "--- Stopping Frontend ---"
	# Try killing common dev-server processes
	-@pkill -f 'react-scripts start|node .*react-scripts|npm start' 2>/dev/null || true
	# Try killing by port (Linux)
	-@fuser -k 3000/tcp 2>/dev/null || true
	# Fallback using lsof if available
	-@if command -v lsof >/dev/null 2>&1; then \
	  PIDS=$$(lsof -ti:3000); \
	  [ -z "$$PIDS" ] || kill $$PIDS 2>/dev/null || true; \
	fi

# -- Aggregate Targets --
build:
	@echo "--- Building All ---"
	@$(MAKE) backend-build
	@$(MAKE) frontend-build


docker-build:
	@echo "--- Building Docker images (backend + frontend) ---"
	@$(MAKE) -C backend docker-build
	@$(MAKE) -C frontend docker-build

docker-push:
	@echo "--- Pushing Docker images (backend + frontend) ---"
	@$(MAKE) -C backend docker-push
	@$(MAKE) -C frontend docker-push

docker-publish: docker-build docker-push

run:
	@echo "--- Running All (in parallel) ---"
	@$(MAKE) backend-run &
	@$(MAKE) frontend-start

# Stop both backend and frontend (best-effort)
stop:
	@echo "--- Stopping Backend and Frontend ---"
	@$(MAKE) backend-stop
	@$(MAKE) frontend-stop

# Alias: `make kill` does the same as `make stop`
kill: stop


help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Backend targets:"
	@echo "  backend-build       Build the Go application"
	@echo "  backend-run         Run the Go application"
	@echo "  backend-clean       Remove build artifacts"
	@echo ""
	@echo "Frontend targets:"
	@echo "  frontend-install    Install npm dependencies"
	@echo "  frontend-start      Start the React development server"
	@echo "  frontend-build      Build the React application for production"
	@echo "  frontend-test       Run React tests"
	@echo ""
	@echo "Aggregate targets:"
	@echo "  build               Build both backend and frontend"
	@echo "  run                 Run backend and frontend concurrently"
	@echo "  stop | kill         Stop backend and frontend processes"
	@echo "  help                Show this help message"

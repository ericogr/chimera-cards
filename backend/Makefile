
# Makefile for the Go backend application

.PHONY: all build run clean docker-build docker-push docker-build-push

BINARY_NAME=chimera-cards
OUT_DIR?=.

# Version metadata: defaults computed from git when available but can be
# overridden by environment or CI (useful for deterministic builds).
VERSION ?= $(shell git describe --tags --dirty --always --long 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
DIRTY ?= $(shell [ -n "$$(git status --porcelain 2>/dev/null)" ] && echo true || echo false)

all: build

build:
	@echo "Building backend..."
	@mkdir -p $(OUT_DIR)
	@echo "Version: $(VERSION) Commit: $(COMMIT) Built: $(BUILD_DATE) Dirty: $(DIRTY)"
	@CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build -o $(OUT_DIR)/$(BINARY_NAME) \
		-ldflags "-s -w -X 'github.com/ericogr/chimera-cards/internal/version.Version=$(VERSION)' -X 'github.com/ericogr/chimera-cards/internal/version.Commit=$(COMMIT)' -X 'github.com/ericogr/chimera-cards/internal/version.Date=$(BUILD_DATE)' -X 'github.com/ericogr/chimera-cards/internal/version.Dirty=$(DIRTY)'" \
		./cmd/chimera-cards

run: build
	@echo "Running backend..."
	@$(OUT_DIR)/$(BINARY_NAME)

clean:
	@echo "Cleaning backend..."
	@go clean
	@rm -f $(OUT_DIR)/$(BINARY_NAME)
	@rm -f data/chimera.db

# Docker image variables (push to Docker Hub under ericogr namespace by default)
# Use a backend-specific image name to disambiguate frontend and backend images.
IMAGE ?= ericogr/chimera-cards-backend
TAG ?= latest
DOCKERFILE ?= Dockerfile

.PHONY: docker-build docker-push docker-build-push

docker-build:
	@echo "Building backend docker image $(IMAGE):$(TAG)"
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg DIRTY=$(DIRTY) \
		-t $(IMAGE):$(TAG) -f $(DOCKERFILE) .

docker-push:
	@echo "Pushing backend docker image $(IMAGE):$(TAG)"
	@docker push $(IMAGE):$(TAG)

docker-build-push: docker-build docker-push

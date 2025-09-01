# Makefile for the Go backend application

.PHONY: all build run clean docker-build docker-push docker-build-push

BINARY_NAME=quimera-cards
OUT_DIR?=.

all: build

build:
	@echo "Building backend..."
	@mkdir -p $(OUT_DIR)
	@go build -o $(OUT_DIR)/$(BINARY_NAME) ./cmd/quimera-cards

run: build
	@echo "Running backend..."
	@$(OUT_DIR)/$(BINARY_NAME)

clean:
	@echo "Cleaning backend..."
	@go clean
	@rm -f $(OUT_DIR)/$(BINARY_NAME)
	@rm -f quimera.db

# Docker image variables
IMAGE ?= quimera-backend
TAG ?= latest
DOCKERFILE ?= Dockerfile

.PHONY: docker-build docker-push docker-build-push

docker-build:
	@echo "Building backend docker image $(IMAGE):$(TAG)"
	@docker build -t $(IMAGE):$(TAG) -f $(DOCKERFILE) .

docker-push:
	@echo "Pushing backend docker image $(IMAGE):$(TAG)"
	@docker push $(IMAGE):$(TAG)

docker-build-push: docker-build docker-push

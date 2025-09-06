# Makefile for the React frontend application

.PHONY: all install start build test docker-build docker-push docker-build-push

all: install

install:
	@echo "Installing frontend dependencies..."
	@npm install

start:
	@echo "Starting frontend development server..."
	@bash ./scripts/generate-env-config.sh
	@npm start

build:
	@echo "Building frontend for production..."
	@npm run build

test:
	@echo "Running frontend tests (CI mode)..."
	@CI=true npm test -- --watchAll=false

# Docker image variables
IMAGE ?= chimera-frontend
TAG ?= latest
DOCKERFILE ?= Dockerfile

.PHONY: docker-build docker-push docker-build-push

docker-build:
	@echo "Building frontend docker image $(IMAGE):$(TAG)"
	@docker build \
		--build-arg REACT_APP_GOOGLE_CLIENT_ID=$(REACT_APP_GOOGLE_CLIENT_ID) \
		--build-arg REACT_APP_API_BASE_URL=$(REACT_APP_API_BASE_URL) \
		-t $(IMAGE):$(TAG) -f $(DOCKERFILE) .

docker-push:
	@echo "Pushing frontend docker image $(IMAGE):$(TAG)"
	@docker push $(IMAGE):$(TAG)

docker-build-push: docker-build docker-push

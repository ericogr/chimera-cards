# Makefile for the Go backend application

.PHONY: all build run clean

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

# Makefile for the React frontend application

.PHONY: all install start build test

all: install

install:
	@echo "Installing frontend dependencies..."
	@npm install

start:
	@echo "Starting frontend development server..."
	@npm start

build:
	@echo "Building frontend for production..."
	@npm run build

test:
	@echo "Running frontend tests (CI mode)..."
	@CI=true npm test -- --watchAll=false

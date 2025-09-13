# Multi-stage Dockerfile for the Go backend
# Builder: compile the Go binary with CGO enabled (required by sqlite driver)
FROM golang:1.25.0-bookworm AS builder

WORKDIR /src

# Copy module files first to leverage Docker layer caching
# When building inside the backend directory the module files are at ./
COPY go.mod go.sum ./
RUN go mod download

# Install minimal build dependencies required for cgo / sqlite build
RUN apt-get update && \
    apt-get install -y --no-install-recommends gcc build-essential libsqlite3-dev ca-certificates git && \
    rm -rf /var/lib/apt/lists/*

# Copy the backend source
COPY . ./

# Build-time metadata (can be overridden via --build-arg)
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
ARG DIRTY=false

# Build the server binary. Use CGO as github.com/mattn/go-sqlite3 requires it.
ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64
RUN go build -a -installsuffix cgo \
	-ldflags "-s -w -X 'github.com/ericogr/chimera-cards/internal/version.Version=${VERSION}' -X 'github.com/ericogr/chimera-cards/internal/version.Commit=${COMMIT}' -X 'github.com/ericogr/chimera-cards/internal/version.Date=${BUILD_DATE}' -X 'github.com/ericogr/chimera-cards/internal/version.Dirty=${DIRTY}'" \
	-o /app/chimera-cards ./cmd/chimera-cards

### Final image: distroless base (contains libc and TLS CA certs)
FROM gcr.io/distroless/base:nonroot

# Re-declare build args so they can be referenced in labels
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
ARG DIRTY=false

# Copy binary from builder stage
COPY --from=builder /app/chimera-cards /game/chimera-cards

# Default working dir (where the app will look for chimera_config.json by default)
WORKDIR /game

# Image labels with build metadata
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"

# Expose the port the app uses by default
EXPOSE 8080

# Use non-root user provided by distroless
USER 1000:1000

ENTRYPOINT ["/game/chimera-cards"]

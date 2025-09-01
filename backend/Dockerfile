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

# Build the server binary. Use CGO as github.com/mattn/go-sqlite3 requires it.
ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64
RUN go build -a -installsuffix cgo -ldflags "-s -w" -o /app/quimera-cards ./cmd/quimera-cards
# Build a tiny HTTP healthcheck binary (static) so we can run container
# healthchecks even when using a distroless final image.
ENV CGO_ENABLED=0
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /app/healthcheck ./cmd/healthcheck


### Final image: distroless base (contains libc and TLS CA certs)
FROM gcr.io/distroless/base:nonroot

# Copy binary from builder stage
COPY --from=builder /app/quimera-cards /quimera-cards
COPY --from=builder /app/healthcheck /healthcheck

# Default working dir (where the app will look for chimera_config.json by default)
WORKDIR /

# Expose the port the app uses by default
EXPOSE 8080

# Use non-root user provided by distroless
USER nonroot

ENTRYPOINT ["/quimera-cards"]

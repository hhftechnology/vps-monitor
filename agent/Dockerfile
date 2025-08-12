# agent/Dockerfile

# ---- Build Stage ----
FROM golang:1.22-alpine AS builder

# Install git for go mod download
RUN apk add --no-cache git

# Set the working directory inside the container.
WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code.
COPY main.go .

# Build the Go app with optimizations
# -ldflags="-s -w" strips debugging information and reduces binary size
# CGO_ENABLED=0 creates a static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.BuildTime=$(date -u +%Y%m%d.%H%M%S)" \
    -trimpath \
    -o /agent-binary .

# ---- Final Stage ----
FROM alpine:latest

# Install required tools for health checks and debugging
RUN apk add --no-cache \
    ca-certificates \
    wget \
    curl \
    procps \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy the static binary from the builder stage.
COPY --from=builder /agent-binary /usr/local/bin/agent

# Make sure the binary is executable
RUN chmod +x /usr/local/bin/agent

# Switch to non-root user
USER appuser

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep agent || exit 1

# Set metadata
LABEL maintainer="HHF Technology" \
      description="VPS Monitor Agent - Lightweight system monitoring" \
      version="1.1.0"

# Set default environment variables
ENV HOME_SERVER_URL="" \
    AGENT_NAME="" \
    AGENT_ID=""

# Command to run the application.
CMD ["/usr/local/bin/agent"]
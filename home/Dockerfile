# ---- Build Stage 1: Frontend ----
FROM node:22-alpine AS frontend-builder

# Set working directory
WORKDIR /app/web

# Install dependencies first for better caching
COPY web/package*.json ./
RUN npm ci --only=production --silent

# Copy frontend source code
COPY web/ ./

# Build production static files
RUN npm run build && \
    # Remove unnecessary files to reduce image size
    rm -rf node_modules src public package*.json

# ---- Build Stage 2: Backend ----
FROM golang:1.22-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source
COPY main.go ./

# Build Go binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.BuildTime=$(date -u +%Y%m%d.%H%M%S)" \
    -trimpath \
    -o /home-server-binary .

# ---- Final Stage ----
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    wget \
    curl \
    tzdata \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy Go binary
COPY --from=backend-builder /home-server-binary /app/home-server

# Copy frontend build output
COPY --from=frontend-builder /app/web/build /app/web/build

# Make binary executable
RUN chmod +x /app/home-server

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8085/api/health || exit 1

# Set metadata
LABEL maintainer="HHF Technology" \
      description="VPS Monitor Home Server - Web dashboard and API" \
      version="1.1.0"

# Expose server port
EXPOSE 8085

# Set default environment variables
ENV GIN_MODE=release \
    PORT=8085

# Run server
CMD ["/app/home-server"]
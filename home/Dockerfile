# Stage 1: Build the frontend
FROM oven/bun:latest AS frontend-builder

WORKDIR /frontend

# Copy frontend package files
COPY frontend/package.json frontend/bun.lock ./

# Install dependencies
RUN bun install --frozen-lockfile

# Copy frontend source
COPY frontend/ ./

# Build frontend (outputs to ./dist)
RUN bun run build

# Stage 2: Build the backend with embedded frontend
FROM --platform=$BUILDPLATFORM golang:1.23 AS backend-builder

WORKDIR /app

# Copy Go module files
COPY home/go.mod home/go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy backend source
COPY home/ .

# Copy built frontend into the static embed directory
COPY --from=frontend-builder /frontend/dist ./internal/static/dist

# Build Go binary with embedded frontend
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o vps-monitor ./cmd/server

# Stage 3: Final runtime image with SSH client for remote Docker hosts
FROM debian:bookworm-slim

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates openssh-client \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy only the binary (frontend is embedded)
COPY --from=backend-builder /app/vps-monitor ./vps-monitor

ENTRYPOINT ["/app/vps-monitor"]

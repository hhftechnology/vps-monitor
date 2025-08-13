# agent/Dockerfile - Ultra-Small Distroless Version

# ---- Build Stage ----
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code.
COPY main.go .

# Build the Go app with maximum optimizations for static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -extldflags '-static' -X main.BuildTime=$(date -u +%Y%m%d.%H%M%S)" \
    -trimpath \
    -a \
    -installsuffix cgo \
    -tags netgo,osusergo \
    -o /agent-binary .

RUN file /agent-binary && ldd /agent-binary 2>&1 | grep -q "not a dynamic executable" || exit 1

# ---- Final Stage ----
FROM gcr.io/distroless/static:nonroot

# Copy the static binary from the builder stage.
COPY --from=builder /agent-binary /usr/local/bin/agent

# Set metadata
LABEL maintainer="HHF Technology" \
      description="VPS Monitor Agent - Lightweight system monitoring" \
      version="1.1.0"

# Set default environment variables
ENV HOME_SERVER_URL="" \
    AGENT_NAME="" \
    AGENT_ID="" \
    HOST_PROC="/host/proc" \
    HOST_SYS="/host/sys" \
    HOST_ROOT="/host/root"

# No health check possible with distroless (no shell)
# Health checking should be done externally

# Command to run the application.
ENTRYPOINT ["/usr/local/bin/agent"]
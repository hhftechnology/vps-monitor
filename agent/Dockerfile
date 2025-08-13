# Dockerfile.scratch - Absolute Minimal Scratch Version

# ---- Build Stage ----
FROM golang:1.23-alpine AS builder

# Install ca-certificates that we'll copy to scratch
RUN apk add --no-cache ca-certificates git

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

# Create passwd file for non-root user
RUN echo 'appuser:x:65534:65534:appuser:/:/sbin/nologin' > /tmp/passwd

# ---- Final Stage ----
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy passwd file for user
COPY --from=builder /tmp/passwd /etc/passwd

# Copy the static binary from the builder stage.
COPY --from=builder /agent-binary /agent

# Run as non-root user
USER appuser

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


# Command to run the application.
ENTRYPOINT ["/agent"]
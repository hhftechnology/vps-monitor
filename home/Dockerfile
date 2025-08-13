# home/Dockerfile - Final minimal with embedded frontend

# ---- Build Stage 1: Frontend ----
FROM node:22-alpine AS frontend-builder

WORKDIR /app/web
COPY home/web/package*.json ./
RUN npm install
COPY home/web/ ./

# Build with maximum compression
ENV GENERATE_SOURCEMAP=false \
    INLINE_RUNTIME_CHUNK=false \
    IMAGE_INLINE_SIZE_LIMIT=0

RUN npm run build && \
    # Remove all source maps and unnecessary files
    find build -name "*.map" -delete && \
    find build -name "*.txt" -delete && \
    find build -name "favicon.ico" -delete && \
    # Compress all text assets
    find build -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" -o -name "*.json" \) \
        -exec gzip -9 {} \; -exec mv {}.gz {} \; && \
    # Report final size
    du -sh build

# ---- Build Stage 2: Backend with Embedded Assets ----
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git upx

WORKDIR /app

# Copy Go modules first for better caching
COPY home/go.mod home/go.sum ./
RUN go mod download

# Copy Go source
COPY home/main.go ./

# Copy compressed frontend build
COPY --from=frontend-builder /app/web/build ./web/build

# Build with maximum optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -a \
    -installsuffix cgo \
    -tags netgo \
    -o /app/vps-monitor . && \
    # Compress binary
    upx --best --lzma /app/vps-monitor && \
    # Report final binary size
    ls -lh /app/vps-monitor

# ---- Final Stage: Scratch ----
FROM scratch

# Copy CA certificates for HTTPS (if needed)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compressed binary with embedded frontend
COPY --from=builder /app/vps-monitor /vps-monitor

# Minimal labels
LABEL description="VPS Monitor - Minimal" \
      version="1.1.0-minimal"

EXPOSE 8085

ENTRYPOINT ["/vps-monitor"]
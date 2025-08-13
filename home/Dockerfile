# home/Dockerfile - Optimized for <10MB

# ---- Build Stage 1: Frontend ----
FROM node:22-alpine AS frontend-builder

WORKDIR /app/web
COPY home/web/package*.json ./
RUN npm ci --only=production --no-audit --no-fund
COPY home/web/ ./
RUN npm run build

# Clean and compress frontend assets
RUN find build -name "*.js" -exec gzip -9 {} \; -exec mv {}.gz {} \; && \
    find build -name "*.css" -exec gzip -9 {} \; -exec mv {}.gz {} \; && \
    find build -name "*.html" -exec gzip -9 {} \; -exec mv {}.gz {} \; && \
    rm -rf node_modules src public package*.json && \
    rm -rf build/static/**/*.map

# ---- Build Stage 2: Binary Compression ----
FROM alpine:latest AS upx-builder

RUN apk add --no-cache upx

# ---- Build Stage 3: Backend ----
FROM golang:1.22-alpine AS backend-builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY home/go.mod home/go.sum ./
RUN go mod download
COPY home/main.go ./

# Build with maximum optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.BuildTime=$(date -u +%Y%m%d.%H%M%S) -extldflags '-static'" \
    -trimpath \
    -a \
    -installsuffix cgo \
    -o /home-server-binary .

# Compress binary with UPX
COPY --from=upx-builder /usr/bin/upx /usr/bin/upx
RUN upx --best --lzma /home-server-binary

# ---- Final Stage: Scratch Base ----
FROM scratch

# Copy SSL certificates from Alpine
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy compressed binary
COPY --from=backend-builder /home-server-binary /home-server

# Copy compressed frontend assets
COPY --from=frontend-builder /app/web/build /web/build

# Health check will be handled by orchestrator
LABEL maintainer="HHF Technology" \
      description="VPS Monitor Home Server" \
      version="1.1.0"

EXPOSE 8085

# Set minimal environment
ENV GIN_MODE=release \
    PORT=8085

ENTRYPOINT ["/home-server"]
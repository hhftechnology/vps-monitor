# ---- Build Stage 1: Frontend ----
# Build the React application first.
FROM node:22-alpine AS frontend-builder

WORKDIR /app/web

# Install dependencies first (better cache)
COPY web/package.json web/package-lock.json ./
RUN npm install

# Copy the rest of the frontend code
COPY web/ ./

# Build production static files
RUN npm run build


# ---- Build Stage 2: Backend ----
# Build the Go backend
FROM golang:1.22-alpine AS backend-builder

WORKDIR /app

# Copy Go module files and download deps
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source
COPY main.go ./

# Build Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /home-server-binary .


# ---- Final Stage ----
FROM alpine:latest

WORKDIR /app

# Copy Go binary
COPY --from=backend-builder /home-server-binary /app/home-server

# Copy frontend build output
COPY --from=frontend-builder /app/web/build /app/web/build

# Security: run as non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Expose server port
EXPOSE 8085

# Run server
CMD ["/app/home-server"]

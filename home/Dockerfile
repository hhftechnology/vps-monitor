# home/Dockerfile

# ---- Build Stage 1: Frontend ----
# Build the React application first.
FROM node:18-alpine AS frontend-builder

WORKDIR /app/web

# Copy package.json and install dependencies.
COPY web/package.json web/package-lock.json ./
RUN npm install

# Add execute permissions to the react-scripts binary.

RUN chmod +x ./node_modules/.bin/react-scripts

# Copy the rest of the frontend code.
COPY web/ ./

# Build the production-ready static files.
RUN npm run build

# ---- Build Stage 2: Backend ----
# Build the Go backend.
FROM golang:1.21-alpine AS backend-builder

WORKDIR /app

# Copy Go module files and download dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy the backend source code.
COPY main.go .

# Build the Go application.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /home-server-binary .

# ---- Final Stage ----
# Create the final, lightweight image.
FROM alpine:latest

WORKDIR /app

# Copy the built Go binary from the backend-builder stage.
COPY --from=backend-builder /home-server-binary /app/home-server

# Copy the built React app from the frontend-builder stage.
COPY --from=frontend-builder /app/web/build /app/web/build

# Expose the port the server will run on.
EXPOSE 8080

# Set a non-root user for security.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Command to run the server.
CMD ["/app/home-server"]

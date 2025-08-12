# agent/Dockerfile

# ---- Build Stage ----
# Use the official Golang image to create a build artifact.
# This image is based on Alpine Linux, which is lightweight.
FROM golang:1.21-alpine AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies.
# We do this in a separate layer to leverage Docker's layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code.
COPY main.go .

# Build the Go app.
# -ldflags="-s -w" strips debugging information, which reduces the binary size.
# CGO_ENABLED=0 disables CGO, which is needed for a static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /agent-binary .

# ---- Final Stage ----
# Use a minimal base image. `scratch` is the smallest possible image,
# but `alpine` is also very small and gives you a shell for debugging if needed.
# We'll use alpine here.
FROM alpine:latest

# Copy the static binary from the builder stage.
COPY --from=builder /agent-binary /agent

# Set the user to a non-root user for better security.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Command to run the application.
CMD ["/agent"]

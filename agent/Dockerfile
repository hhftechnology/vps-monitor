# agent/Dockerfile

# ---- Build Stage ----
FROM golang:1.21-alpine AS builder

# Set the working directory inside the container.
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy the source code.
COPY main.go .

# Build the Go app.
# -ldflags="-s -w" strips debugging information, which reduces the binary size.
# CGO_ENABLED=0 disables CGO, which is needed for a static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /agent-binary .

# ---- Final Stage ----

FROM alpine:latest

# Copy the static binary from the builder stage.
COPY --from=builder /agent-binary /agent

# Set the user to a non-root user for better security.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Command to run the application.
CMD ["/agent"]

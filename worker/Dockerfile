FROM golang:1.19-alpine

WORKDIR /app

# Install Docker client and debugging tools
RUN apk add --no-cache docker curl bash procps

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o /app/worker-bin ./worker

# Add a healthcheck
HEALTHCHECK --interval=5s --timeout=3s --retries=3 CMD ps aux | grep worker-bin || exit 1

# Set environment variable to use host.docker.internal for Docker-in-Docker
ENV DOCKER_HOST=unix:///var/run/docker.sock

# Run the worker with more verbose logging
CMD ["sh", "-c", "echo 'Starting worker...' && ls -la /var/run/docker.sock && /app/worker-bin"] 
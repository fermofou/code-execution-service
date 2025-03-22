FROM golang:1.19-alpine

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o /app/api-bin ./api

# Expose port
EXPOSE 8080

# Run the API server
CMD ["/app/api-bin"] 
# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o api-gateway ./cmd

# Final stage
FROM alpine:latest

# Set working directory
WORKDIR /app

# Install CA certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/api-gateway .

# Copy the .sample.config file
COPY --from=builder /app/config/sample.config .

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./api-gateway"] 
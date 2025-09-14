# Build stage
FROM golang:1.25.1-alpine AS builder

# Set working directory
WORKDIR /app

COPY shared/modules /shared/modules

# Copy go mod files
COPY services/auth-service/go.mod services/auth-service/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/auth-service/ .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Create log directory
RUN mkdir -p /agrisa/log/auth_service

# Expose port
EXPOSE 8083

# Command to run
CMD ["./main"]

# Build stage
FROM golang:1.25.1-alpine AS builder
# Set working directory
WORKDIR /app
RUN apk add --no-cache git
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
# Create app directory
WORKDIR /app
RUN apk add --no-cache tzdata
ENV TZ=Asia/Ho_Chi_Minh
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
RUN apk --no-cache add ca-certificates && \
    addgroup -g 1001 appgroup && \
    adduser -D -u 1001 -G appgroup appuser
# Copy the binary from builder stage
COPY --from=builder /app/main /app/
RUN chown -R appuser:appgroup /app
# Create log directory
RUN mkdir -p /agrisa/log/auth_service
USER appuser
# Command to run
CMD ["./main"]

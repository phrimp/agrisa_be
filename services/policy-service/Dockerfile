# Build stage
FROM golang:1.25.1-alpine AS builder
# Set working directory
WORKDIR /app
RUN apk add --no-cache git
COPY shared/modules /shared/modules
# Copy go mod files
COPY services/policy-service/go.mod services/policy-service/go.sum ./
# Download dependencies
RUN go mod download
# Copy source code
COPY services/policy-service/ .
# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go
# Final stage
FROM debian:bookworm-slim
# Install ca-certificates, pdftk, and fonts for PDF form filling with Unicode/Vietnamese support
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    pdftk-java \
    fonts-dejavu-core \
    fonts-noto-core \
    && rm -rf /var/lib/apt/lists/* \
    && fc-cache -fv
ENV TZ=Asia/Ho_Chi_Minh
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/false appuser
# Copy the binary from builder stage
COPY --from=builder /app/main /app/
RUN chown -R appuser:appgroup /app
# Create log directory
RUN mkdir -p /agrisa/log/policy_service
RUN chown -R appuser:appgroup /agrisa/log/policy_service
# Create data directory for scheduler files
RUN mkdir -p /data
RUN chown -R appuser:appgroup /data
USER appuser
# Command to run
CMD ["./main"]

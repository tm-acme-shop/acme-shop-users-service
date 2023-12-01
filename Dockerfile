# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Copy shared-go dependency
COPY ../acme-shop-shared-go ../acme-shop-shared-go

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o users-service ./cmd/users

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -S acme && adduser -S acme -G acme
USER acme

# Copy binary from builder
COPY --from=builder /app/users-service .

# Copy configs
COPY --from=builder /app/configs ./configs

# Environment variables
ENV SERVICE_NAME=users-service
ENV ENVIRONMENT=production
ENV SERVER_PORT=8081

# TODO(TEAM-SEC): Ensure these are not hardcoded in production
# These should be provided via secrets management

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# Expose port
EXPOSE 8081

# Run the service
CMD ["./users-service"]

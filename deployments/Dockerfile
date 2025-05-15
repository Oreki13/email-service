# Stage 1: Build stage
FROM golang:1.24-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Download dependencies first (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o /email-service main.go

# Stage 2: Run stage
FROM alpine:3.21

# Add basic tools
RUN apk add --no-cache tzdata ca-certificates curl

# Create non-root user and necessary folders
RUN addgroup -S app && adduser -S -G app app && \
  mkdir -p /app/config /app/storage/attachments && \
  chown -R app:app /app

# Set timezone
ENV TZ=Asia/Jakarta

# Set working directory
WORKDIR /app

# Copy files from builder stage
COPY --from=builder /email-service /app/email-service
COPY --chown=app:app migrations/ /app/migrations/
COPY --chown=app:app .env.example /app/config/.env.example

# Use non-root user
USER app

# Set environment variables
ENV GO_ENV=production \
  APP_ENVIRONMENT=production \
  SERVER_HOST=0.0.0.0 \
  STORAGE_PROVIDER=local \
  LOCAL_STORAGE_PATH=/app/storage/attachments

# Expose HTTP port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health/live || exit 1

# Run command
ENTRYPOINT ["/app/email-service"]
CMD ["server"]

# Multi-stage build for PostgreSQL backup service
# Build with: docker build --build-arg PG_VERSION=16 -t pg-backup:16 .

ARG PG_VERSION=16

# Build stage
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

# Install git and ca-certificates (needed for go mod download and HTTPS)
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /pg-backup .

# Final stage
FROM alpine:3.21

ARG PG_VERSION

# Install ca-certificates for HTTPS, tzdata for timezone support, and PostgreSQL client
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    postgresql${PG_VERSION}-client

# Create necessary directories
RUN mkdir -p /backups

# Copy binary from builder
COPY --from=builder /pg-backup /usr/local/bin/pg-backup

# Default environment variables (non-sensitive only)
ENV PGHOST=localhost \
    PGPORT=5432 \
    PGUSER=postgres \
    PGDATABASE=postgres \
    BACKUP_CRON="0 0 * * *" \
    BACKUP_ON_START=false \
    BACKUP_COMPRESSION=true \
    PGDUMP_FORMAT=custom \
    STORAGE_TYPE=local \
    LOCAL_BACKUP_PATH=/backups \
    S3_REGION=us-east-1 \
    S3_PATH_STYLE=false \
    S3_BACKUP_PREFIX=pg-backups \
    RETENTION_COUNT=0

# Volume for local backups
VOLUME ["/backups"]

ENTRYPOINT ["/usr/local/bin/pg-backup"]

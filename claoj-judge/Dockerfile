# Multi-stage build for CLAOJ Go Judge
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

# Copy source code
COPY . .

# Build the judge
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o claoj-judge ./cmd/claoj-judge

# Runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    bash \
    gcc \
    g++ \
    python3 \
    python3-dev \
    openjdk11 \
    nodejs \
    rust \
    && rm -rf /var/cache/apk/*

# Create judge user
RUN addgroup -g 1000 judge && \
    adduser -u 1000 -G judge -h /home/judge -D judge

# Create necessary directories
RUN mkdir -p /problems /tmp/judge /etc/claoj && \
    chown -R judge:judge /problems /tmp/judge /etc/claoj

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/claoj-judge /app/claoj-judge

# Set permissions
RUN chown judge:judge /app/claoj-judge

# Switch to non-root user
USER judge

# Default command
ENTRYPOINT ["/app/claoj-judge"]
CMD []

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD nc -z localhost 9998 || exit 1

# Expose API port
EXPOSE 9998

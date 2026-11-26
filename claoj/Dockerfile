# compare50 (plagiarism fork) stage: venv with the fork installed at /opt/compare50
FROM alpine:latest AS compare50

RUN apk add --no-cache python3

COPY plagiarism /src/compare50
RUN python3 -m venv /opt/compare50 \
 && /opt/compare50/bin/pip install --no-cache-dir /src/compare50

# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Module and build caches are cache mounts (CI keeps them between runs), so
# a source change recompiles only the changed packages.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o main .

# Production stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget python3

# Copy binary from builder
COPY --from=builder /app/main .

# compare50 venv for plagiarism checks (jobs.Compare50Python)
COPY --from=compare50 /opt/compare50 /opt/compare50

# Copy static files if needed
# COPY --from=builder /app/static ./static

EXPOSE 8080

ENV GIN_MODE=release

CMD ["./main"]

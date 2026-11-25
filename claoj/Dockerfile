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

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

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

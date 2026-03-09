.PHONY: build test clean run docker-build docker-up docker-down help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

BINARY_NAME=claoj-judge
BINARY_DIR=./bin
CMD_PATH=./cmd/claoj-judge

# Docker parameters
DOCKER_IMAGE=claoj/judge-go
DOCKER_TAG=latest

all: build

## Build the judge binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_NAME) $(CMD_PATH)

## Build with optimizations
build-release:
	@echo "Building release version..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o $(BINARY_DIR)/$(BINARY_NAME)-release $(CMD_PATH)

## Run the judge (requires arguments)
run:
	@echo "Use 'make run-local' or 'make run-docker'"

## Run locally (provide your own args)
run-local: build
	$(BINARY_DIR)/$(BINARY_NAME) $(ARGS)

## Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./executors

## Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOGET) ./...
	$(GOMOD) tidy

## Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

## Start judges with Docker Compose
docker-up:
	@echo "Starting judges..."
	docker-compose up -d

## Stop judges
docker-down:
	@echo "Stopping judges..."
	docker-compose down

## Restart judges
docker-restart:
	@echo "Restarting judges..."
	docker-compose restart

## View judge logs
docker-logs:
	docker-compose logs -f

## View specific judge logs
docker-logs-judge1:
	docker-compose logs -f judge1

## Check judge status
docker-ps:
	docker-compose ps

## Build and start
docker-deploy: docker-build docker-up

## Setup (first time only)
setup:
	@echo "Running setup..."
	./setup.sh

## Show help
help:
	@echo "CLAOJ Go Judge - Makefile Commands"
	@echo ""
	@echo "Build:"
	@echo "  make build           - Build the judge binary"
	@echo "  make build-release   - Build optimized release binary"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make deps            - Download dependencies"
	@echo ""
	@echo "Test:"
	@echo "  make test            - Run all tests"
	@echo "  make test-coverage   - Run tests with coverage report"
	@echo "  make benchmark       - Run benchmarks"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build    - Build Docker image"
	@echo "  make docker-up       - Start judges (docker-compose)"
	@echo "  make docker-down     - Stop judges"
	@echo "  make docker-restart  - Restart judges"
	@echo "  make docker-logs     - View logs"
	@echo "  make docker-ps       - Check status"
	@echo "  make docker-deploy   - Build and start"
	@echo ""
	@echo "Other:"
	@echo "  make setup           - Run setup script"
	@echo "  make help            - Show this help"
	@echo ""

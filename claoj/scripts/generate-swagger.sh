#!/bin/bash
# Generate Swagger/OpenAPI documentation for CLAOJ API v2
# Usage: ./scripts/generate-swagger.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DOCS_DIR="$PROJECT_DIR/docs"

echo "=== CLAOJ API v2 Swagger Documentation Generator ==="
echo ""

# Check if swag is installed
if ! command -v swag &> /dev/null; then
    echo "Error: swag CLI not found. Install with:"
    echo "  go install github.com/swaggo/swag/cmd/swag@latest"
    exit 1
fi

echo "Swag version: $(swag -v)"
echo ""

# Change to project directory
cd "$PROJECT_DIR"

echo "Generating documentation..."
echo ""

# Run swag init
swag init \
    --parseDependency \
    --parseInternal \
    --generalInfo main.go \
    --output "$DOCS_DIR"

echo ""
echo "=== Documentation Generated Successfully ==="
echo ""
echo "Output directory: $DOCS_DIR"
echo ""
echo "Generated files:"
ls -la "$DOCS_DIR"/*.json "$DOCS_DIR"/*.yaml 2>/dev/null || echo "  (No JSON/YAML files generated yet)"
echo ""
echo "To view documentation:"
echo "  1. Start the server: go run main.go"
echo "  2. Open: http://localhost:8080/swagger/index.html"
echo ""

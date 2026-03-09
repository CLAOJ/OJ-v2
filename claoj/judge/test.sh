#!/bin/bash
# Judge Migration Test Runner
# This script runs tests for the migrated Go judge

set -e

echo "========================================"
echo "  CLAOJ Judge Migration Test Suite"
echo "========================================"
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test suite
run_test() {
    local package=$1
    local description=$2

    echo -e "${YELLOW}Testing: ${description}${NC}"
    echo "Package: $package"
    echo "----------------------------------------"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if go test -v -timeout 30s "$package" 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ FAILED${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi

    echo ""
}

# Function to run benchmarks
run_benchmark() {
    local package=$1
    local description=$2

    echo -e "${YELLOW}Benchmarking: ${description}${NC}"
    echo "Package: $package"
    echo "----------------------------------------"

    if go test -bench=. -benchmem -timeout 60s "$package" 2>&1; then
        echo -e "${GREEN}✓ Benchmark completed${NC}"
    else
        echo -e "${RED}✗ Benchmark failed${NC}"
    fi

    echo ""
}

# Check Go version
echo "Go version:"
go version
echo ""

# Check for required tools
echo "Checking dependencies..."
if command -v gcc &> /dev/null; then
    echo -e "${GREEN}✓ gcc found${NC}"
else
    echo -e "${YELLOW}! gcc not found (C/C++ tests will be skipped)${NC}"
fi

if command -v python3 &> /dev/null; then
    echo -e "${GREEN}✓ python3 found${NC}"
else
    echo -e "${YELLOW}! python3 not found (Python tests will be skipped)${NC}"
fi

if command -v node &> /dev/null; then
    echo -e "${GREEN}✓ node found${NC}"
else
    echo -e "${YELLOW}! node not found (Node.js tests will be skipped)${NC}"
fi

if command -v go &> /dev/null; then
    echo -e "${GREEN}✓ go found${NC}"
fi

if command -v rustc &> /dev/null; then
    echo -e "${GREEN}✓ rustc found${NC}"
else
    echo -e "${YELLOW}! rustc not found (Rust tests will be skipped)${NC}"
fi

if command -v javac &> /dev/null; then
    echo -e "${GREEN}✓ javac found${NC}"
else
    echo -e "${YELLOW}! javac not found (Java tests will be skipped)${NC}"
fi

echo ""
echo "========================================"
echo "  Running Unit Tests"
echo "========================================"
echo ""

# Run tests for each package
run_test "./executors" "Language Executors"
run_test "./sandbox" "Sandbox/Isolation"
run_test "./core" "Core Judge Logic"
run_test "./protocol" "Network Protocol"

echo "========================================"
echo "  Running Benchmarks"
echo "========================================"
echo ""

run_benchmark "./executors" "Executor Performance"

echo "========================================"
echo "  Test Summary"
echo "========================================"
echo ""
echo "Total test suites:  $TOTAL_TESTS"
echo -e "Passed:             ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed:             ${RED}$FAILED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi

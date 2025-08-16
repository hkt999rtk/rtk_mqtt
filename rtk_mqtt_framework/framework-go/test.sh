#!/bin/bash

# RTK MQTT Framework Go - Test Runner

set -e

echo "RTK MQTT Framework Go - Running Tests"
echo "====================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

print_status "Go version: $(go version)"

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    print_error "go.mod not found. Please run this script from the framework-go directory"
    exit 1
fi

# Download dependencies
print_status "Downloading dependencies..."
go mod download

# Verify dependencies
print_status "Verifying dependencies..."
go mod verify

# Run go mod tidy to clean up
print_status "Tidying up module..."
go mod tidy

# Run tests with coverage
print_status "Running tests with coverage..."

# Create coverage directory
mkdir -p coverage

# Run tests for each package
packages=(
    "./pkg/mqtt"
    "./pkg/topic"
    "./pkg/codec"
    "./pkg/device"
    "./pkg/config"
)

total_coverage=0
package_count=0

for pkg in "${packages[@]}"; do
    if [ -d "$pkg" ]; then
        print_status "Testing package: $pkg"
        
        # Run tests with coverage for this package
        coverage_file="coverage/$(basename $pkg).out"
        
        if go test -v -race -coverprofile="$coverage_file" -covermode=atomic "$pkg"; then
            print_status "✓ Tests passed for $pkg"
            
            # Calculate coverage for this package
            if [ -f "$coverage_file" ]; then
                coverage=$(go tool cover -func="$coverage_file" | grep total | awk '{print $3}' | sed 's/%//')
                if [ ! -z "$coverage" ]; then
                    print_status "Coverage for $pkg: ${coverage}%"
                    total_coverage=$(echo "$total_coverage + $coverage" | bc -l)
                    package_count=$((package_count + 1))
                fi
            fi
        else
            print_error "✗ Tests failed for $pkg"
            exit 1
        fi
        
        echo ""
    else
        print_warning "Package directory not found: $pkg"
    fi
done

# Calculate average coverage
if [ $package_count -gt 0 ]; then
    avg_coverage=$(echo "scale=2; $total_coverage / $package_count" | bc -l)
    print_status "Average test coverage: ${avg_coverage}%"
else
    print_warning "No coverage data collected"
fi

# Combine coverage reports
print_status "Combining coverage reports..."
echo "mode: atomic" > coverage/combined.out
tail -n +2 coverage/*.out >> coverage/combined.out

# Generate HTML coverage report
print_status "Generating HTML coverage report..."
go tool cover -html=coverage/combined.out -o coverage/coverage.html

print_status "Coverage report generated: coverage/coverage.html"

# Run go vet
print_status "Running go vet..."
if go vet ./...; then
    print_status "✓ go vet passed"
else
    print_error "✗ go vet failed"
    exit 1
fi

# Run gofmt check
print_status "Checking code formatting..."
unformatted=$(gofmt -l .)
if [ -z "$unformatted" ]; then
    print_status "✓ All files are properly formatted"
else
    print_error "✗ The following files are not properly formatted:"
    echo "$unformatted"
    exit 1
fi

# Check for ineffectual assignments (if ineffassign is available)
if command -v ineffassign &> /dev/null; then
    print_status "Checking for ineffectual assignments..."
    if ineffassign ./...; then
        print_status "✓ No ineffectual assignments found"
    else
        print_warning "! Ineffectual assignments detected (warnings only)"
    fi
else
    print_warning "ineffassign not installed, skipping check"
fi

# Check for unused variables (if unused is available)
if command -v unused &> /dev/null; then
    print_status "Checking for unused code..."
    if unused ./...; then
        print_status "✓ No unused code found"
    else
        print_warning "! Unused code detected (warnings only)"
    fi
else
    print_warning "unused not installed, skipping check"
fi

# Build all packages to ensure they compile
print_status "Building all packages..."
if go build ./...; then
    print_status "✓ All packages build successfully"
else
    print_error "✗ Build failed"
    exit 1
fi

# Build examples
print_status "Building examples..."
example_dirs=(
    "./examples/iot_sensor"
    "./examples/wifi_router"
    "./examples/framework_demo"
)

for example_dir in "${example_dirs[@]}"; do
    if [ -d "$example_dir" ]; then
        print_status "Building example: $example_dir"
        if (cd "$example_dir" && go build .); then
            print_status "✓ Example built successfully: $example_dir"
        else
            print_error "✗ Failed to build example: $example_dir"
            exit 1
        fi
    else
        print_warning "Example directory not found: $example_dir"
    fi
done

# Run benchmark tests if any exist
print_status "Looking for benchmark tests..."
if go test -bench=. -run=^$ ./... > /dev/null 2>&1; then
    print_status "Running benchmark tests..."
    go test -bench=. -run=^$ ./...
else
    print_status "No benchmark tests found"
fi

# Final summary
echo ""
print_status "========================================="
print_status "All tests completed successfully! ✓"
print_status "========================================="

if [ $package_count -gt 0 ]; then
    print_status "📊 Test Coverage Summary:"
    print_status "  - Average coverage: ${avg_coverage}%"
    print_status "  - Coverage report: coverage/coverage.html"
fi

print_status "🚀 Build Status: All packages build successfully"
print_status "📋 Code Quality: All checks passed"

echo ""
print_status "To view the coverage report:"
print_status "  open coverage/coverage.html"
echo ""
print_status "To run tests for a specific package:"
print_status "  go test -v ./pkg/mqtt"
echo ""
print_status "To run tests with race detection:"
print_status "  go test -race ./..."
echo ""

exit 0
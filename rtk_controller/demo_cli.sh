#!/bin/bash

# RTK Controller Demo CLI Script
# This script demonstrates the key features of the RTK Controller

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

CONTROLLER_BINARY="./rtk-controller"
CONFIG_FILE="configs/controller.yaml"

# Helper functions
success() {
    echo -e "${GREEN}✓ $1${NC}"
}

info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

error() {
    echo -e "${RED}✗ $1${NC}"
}

section() {
    echo -e "\n${PURPLE}=== $1 ===${NC}"
}

# Check if controller binary exists
check_binary() {
    if [ ! -f "$CONTROLLER_BINARY" ]; then
        error "Controller binary not found at $CONTROLLER_BINARY"
        info "Please build the controller first: make build"
        exit 1
    fi
    success "Controller binary found"
}

# Check version
show_version() {
    section "RTK Controller Version"
    $CONTROLLER_BINARY --version
}

# Show help information
show_help() {
    section "RTK Controller Help"
    $CONTROLLER_BINARY --help
}

# Validate configuration
validate_config() {
    section "Configuration Validation"
    if [ -f "$CONFIG_FILE" ]; then
        success "Configuration file found: $CONFIG_FILE"
        if $CONTROLLER_BINARY --config "$CONFIG_FILE" --validate; then
            success "Configuration is valid"
        else
            warning "Configuration validation failed"
        fi
    else
        warning "Configuration file not found: $CONFIG_FILE"
        info "Using default configuration"
    fi
}

# Run unit tests
run_tests() {
    section "Running Tests"
    info "Running unit tests..."
    make test-unit
    success "Unit tests completed"
}

# Demonstrate CLI features
demo_cli() {
    section "CLI Interactive Demo"
    info "The RTK Controller provides an interactive CLI interface"
    info "To start the interactive CLI, run: $CONTROLLER_BINARY --cli"
    echo ""
    info "Available CLI commands include:"
    echo "  • device list/info/state - Device management"
    echo "  • command send/list/status - Command operations"
    echo "  • system status/health/stats - System monitoring"
    echo "  • config show - Configuration viewing"
    echo "  • log list/stats - MQTT message logging"
    echo "  • diagnosis start/list - Network diagnosis"
    echo "  • download <seconds> - Download recent MQTT logs"
    echo ""
    info "For detailed help, use 'help' or 'help <command>' in the CLI"
}

# Show build information
show_build_info() {
    section "Build Information"
    info "Build targets available:"
    echo "  • make build - Build for current platform"
    echo "  • make build-linux - Build for Linux"
    echo "  • make build-windows - Build for Windows"
    echo "  • make build-darwin - Build for macOS"
    echo "  • make build-all - Build for all platforms"
    echo ""
    info "Test targets:"
    echo "  • make test - Run all tests"
    echo "  • make test-unit - Run unit tests only"
    echo "  • make test-integration - Run integration tests"
    echo "  • make test-functional - Run functional CLI tests"
    echo "  • make test-performance - Run performance tests"
    echo ""
    info "Development targets:"
    echo "  • make run - Run the controller"
    echo "  • make run-cli - Run in CLI mode"
    echo "  • make coverage - Generate test coverage report"
    echo "  • make clean - Clean build artifacts"
}

# Show project structure
show_project_structure() {
    section "Project Structure"
    info "RTK Controller Project Layout:"
    echo ""
    echo "rtk_controller/"
    echo "├── cmd/controller/          # Main application entry point"
    echo "├── internal/"
    echo "│   ├── cli/                 # Interactive CLI implementation"
    echo "│   ├── command/             # Command management"
    echo "│   ├── config/              # Configuration handling"
    echo "│   ├── device/              # Device management"
    echo "│   ├── diagnosis/           # Network diagnosis"
    echo "│   ├── mqtt/                # MQTT client and messaging"
    echo "│   ├── schema/              # JSON schema validation"
    echo "│   └── storage/             # BuntDB storage layer"
    echo "├── pkg/"
    echo "│   ├── types/               # Common type definitions"
    echo "│   └── utils/               # Utility functions"
    echo "├── test/"
    echo "│   ├── integration/         # Integration tests"
    echo "│   └── scripts/             # Test automation scripts"
    echo "├── configs/                 # Configuration files"
    echo "├── Makefile                 # Build automation"
    echo "└── PLAN.md                  # Development plan"
}

# Main demo function
main() {
    echo -e "${PURPLE}RTK Controller Demo${NC}"
    echo -e "${PURPLE}===================${NC}"
    echo ""
    
    check_binary
    show_version
    validate_config
    
    # Optionally run tests (comment out for faster demo)
    # run_tests
    
    demo_cli
    show_build_info
    show_project_structure
    
    section "Demo Completed"
    success "RTK Controller demo completed successfully!"
    echo ""
    info "Next steps:"
    echo "  1. Configure MQTT broker settings in $CONFIG_FILE"
    echo "  2. Start the controller: $CONTROLLER_BINARY --config $CONFIG_FILE"
    echo "  3. Or use interactive CLI: $CONTROLLER_BINARY --cli --config $CONFIG_FILE"
    echo "  4. Run tests: make test"
    echo "  5. Check documentation in PLAN.md"
    echo ""
    success "Happy controlling! 🎉"
}

# Handle script interruption
cleanup() {
    echo ""
    warning "Demo interrupted"
    exit 130
}

trap cleanup INT TERM

# Run main function
main "$@"
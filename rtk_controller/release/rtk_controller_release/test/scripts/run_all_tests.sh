#!/bin/bash

# RTK Controller Test Suite Runner
# Runs all types of tests: unit, integration, functional, and performance

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Test configuration
CONTROLLER_BINARY="./rtk-controller"
TEST_CONFIG="configs/controller.yaml"
RESULTS_DIR="test/results"
COVERAGE_DIR="$RESULTS_DIR/coverage"
LOG_FILE="$RESULTS_DIR/test_suite.log"

# Test options
RUN_UNIT_TESTS=true
RUN_INTEGRATION_TESTS=true
RUN_FUNCTIONAL_TESTS=true
RUN_PERFORMANCE_TESTS=true
GENERATE_COVERAGE=true
VERBOSE=false

# Test results tracking
UNIT_TEST_RESULT=0
INTEGRATION_TEST_RESULT=0
FUNCTIONAL_TEST_RESULT=0
PERFORMANCE_TEST_RESULT=0

# Helper functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✓ $1${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}✗ $1${NC}" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}⚠ $1${NC}" | tee -a "$LOG_FILE"
}

info() {
    echo -e "${BLUE}ℹ $1${NC}" | tee -a "$LOG_FILE"
}

section() {
    echo -e "\n${PURPLE}=== $1 ===${NC}" | tee -a "$LOG_FILE"
}

# Parse command line arguments
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-unit)
                RUN_UNIT_TESTS=false
                shift
                ;;
            --skip-integration)
                RUN_INTEGRATION_TESTS=false
                shift
                ;;
            --skip-functional)
                RUN_FUNCTIONAL_TESTS=false
                shift
                ;;
            --skip-performance)
                RUN_PERFORMANCE_TESTS=false
                shift
                ;;
            --no-coverage)
                GENERATE_COVERAGE=false
                shift
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Show help
show_help() {
    cat << EOF
RTK Controller Test Suite Runner

Usage: $0 [OPTIONS]

Options:
    --skip-unit         Skip unit tests
    --skip-integration  Skip integration tests
    --skip-functional   Skip functional tests
    --skip-performance  Skip performance tests
    --no-coverage       Skip code coverage generation
    --verbose, -v       Enable verbose output
    --help, -h          Show this help message

Examples:
    $0                          # Run all tests
    $0 --skip-performance       # Run all except performance tests
    $0 --skip-integration --skip-performance  # Run only unit and functional tests
    $0 --verbose               # Run with verbose output
EOF
}

# Setup test environment
setup_test_environment() {
    section "Setting up test environment"
    
    # Create results directories
    mkdir -p "$RESULTS_DIR"
    mkdir -p "$COVERAGE_DIR"
    
    # Initialize log file
    echo "RTK Controller Test Suite - $(date)" > "$LOG_FILE"
    echo "================================================" >> "$LOG_FILE"
    
    # Check if controller binary exists
    if [ ! -f "$CONTROLLER_BINARY" ]; then
        log "Building controller binary..."
        if go build -o "$CONTROLLER_BINARY" cmd/controller/main.go; then
            success "Controller binary built successfully"
        else
            error "Failed to build controller binary"
            exit 1
        fi
    else
        success "Controller binary found"
    fi
    
    # Check test configuration
    if [ ! -f "$TEST_CONFIG" ]; then
        error "Test configuration not found at $TEST_CONFIG"
        exit 1
    else
        success "Test configuration found"
    fi
    
    # Check Go tools
    if ! command -v go >/dev/null 2>&1; then
        error "Go is required but not installed"
        exit 1
    fi
    
    success "Test environment setup complete"
}

# Run unit tests
run_unit_tests() {
    section "Running Unit Tests"
    
    if [ "$RUN_UNIT_TESTS" = false ]; then
        warning "Unit tests skipped"
        return
    fi
    
    local test_args="-v"
    if [ "$GENERATE_COVERAGE" = true ]; then
        test_args="$test_args -coverprofile=$COVERAGE_DIR/unit.out -covermode=atomic"
    fi
    
    local unit_output="$RESULTS_DIR/unit_tests.log"
    
    log "Executing unit tests..."
    if [ "$VERBOSE" = true ]; then
        # Run with real-time output
        if go test $test_args ./internal/... ./pkg/... | tee "$unit_output"; then
            UNIT_TEST_RESULT=0
            success "Unit tests passed"
        else
            UNIT_TEST_RESULT=1
            error "Unit tests failed"
        fi
    else
        # Run with output to log file
        if go test $test_args ./internal/... ./pkg/... > "$unit_output" 2>&1; then
            UNIT_TEST_RESULT=0
            success "Unit tests passed"
        else
            UNIT_TEST_RESULT=1
            error "Unit tests failed"
            warning "Check $unit_output for details"
        fi
    fi
    
    # Show test summary
    if [ -f "$unit_output" ]; then
        local test_count=$(grep -c "=== RUN" "$unit_output" || echo "0")
        local pass_count=$(grep -c "--- PASS:" "$unit_output" || echo "0")
        local fail_count=$(grep -c "--- FAIL:" "$unit_output" || echo "0")
        
        info "Unit test summary: $test_count tests, $pass_count passed, $fail_count failed"
    fi
}

# Run integration tests
run_integration_tests() {
    section "Running Integration Tests"
    
    if [ "$RUN_INTEGRATION_TESTS" = false ]; then
        warning "Integration tests skipped"
        return
    fi
    
    local test_args="-v"
    if [ "$GENERATE_COVERAGE" = true ]; then
        test_args="$test_args -coverprofile=$COVERAGE_DIR/integration.out -covermode=atomic"
    fi
    
    local integration_output="$RESULTS_DIR/integration_tests.log"
    
    # Check if MQTT broker is available for integration tests
    if command -v mosquitto >/dev/null 2>&1; then
        info "Starting test MQTT broker for integration tests..."
        # Start mosquitto in background for tests
        mosquitto -p 1883 -v > "$RESULTS_DIR/mosquitto_integration.log" 2>&1 &
        MOSQUITTO_PID=$!
        sleep 3
        
        if ! kill -0 $MOSQUITTO_PID 2>/dev/null; then
            warning "Failed to start test MQTT broker, integration tests may fail"
        fi
    else
        warning "Mosquitto not found, integration tests may fail"
    fi
    
    log "Executing integration tests..."
    if [ "$VERBOSE" = true ]; then
        if go test $test_args -tags=integration ./test/integration/... | tee "$integration_output"; then
            INTEGRATION_TEST_RESULT=0
            success "Integration tests passed"
        else
            INTEGRATION_TEST_RESULT=1
            error "Integration tests failed"
        fi
    else
        if go test $test_args -tags=integration ./test/integration/... > "$integration_output" 2>&1; then
            INTEGRATION_TEST_RESULT=0
            success "Integration tests passed"
        else
            INTEGRATION_TEST_RESULT=1
            error "Integration tests failed"
            warning "Check $integration_output for details"
        fi
    fi
    
    # Cleanup test broker
    if [ ! -z "$MOSQUITTO_PID" ]; then
        kill $MOSQUITTO_PID 2>/dev/null || true
        wait $MOSQUITTO_PID 2>/dev/null || true
    fi
}

# Run functional tests
run_functional_tests() {
    section "Running Functional Tests"
    
    if [ "$RUN_FUNCTIONAL_TESTS" = false ]; then
        warning "Functional tests skipped"
        return
    fi
    
    local functional_script="test/scripts/test_cli_commands.sh"
    
    if [ ! -f "$functional_script" ]; then
        error "Functional test script not found at $functional_script"
        FUNCTIONAL_TEST_RESULT=1
        return
    fi
    
    if [ ! -x "$functional_script" ]; then
        chmod +x "$functional_script"
    fi
    
    log "Executing functional tests..."
    if [ "$VERBOSE" = true ]; then
        if "$functional_script"; then
            FUNCTIONAL_TEST_RESULT=0
            success "Functional tests passed"
        else
            FUNCTIONAL_TEST_RESULT=1
            error "Functional tests failed"
        fi
    else
        local functional_output="$RESULTS_DIR/functional_tests.log"
        if "$functional_script" > "$functional_output" 2>&1; then
            FUNCTIONAL_TEST_RESULT=0
            success "Functional tests passed"
        else
            FUNCTIONAL_TEST_RESULT=1
            error "Functional tests failed"
            warning "Check $functional_output for details"
        fi
    fi
}

# Run performance tests
run_performance_tests() {
    section "Running Performance Tests"
    
    if [ "$RUN_PERFORMANCE_TESTS" = false ]; then
        warning "Performance tests skipped"
        return
    fi
    
    local performance_script="test/scripts/performance_test.sh"
    
    if [ ! -f "$performance_script" ]; then
        error "Performance test script not found at $performance_script"
        PERFORMANCE_TEST_RESULT=1
        return
    fi
    
    if [ ! -x "$performance_script" ]; then
        chmod +x "$performance_script"
    fi
    
    log "Executing performance tests..."
    if [ "$VERBOSE" = true ]; then
        if "$performance_script"; then
            PERFORMANCE_TEST_RESULT=0
            success "Performance tests completed"
        else
            PERFORMANCE_TEST_RESULT=1
            error "Performance tests failed"
        fi
    else
        local performance_output="$RESULTS_DIR/performance_tests.log"
        if "$performance_script" > "$performance_output" 2>&1; then
            PERFORMANCE_TEST_RESULT=0
            success "Performance tests completed"
        else
            PERFORMANCE_TEST_RESULT=1
            error "Performance tests failed"
            warning "Check $performance_output for details"
        fi
    fi
}

# Generate code coverage report
generate_coverage_report() {
    section "Generating Code Coverage Report"
    
    if [ "$GENERATE_COVERAGE" = false ]; then
        warning "Code coverage generation skipped"
        return
    fi
    
    # Combine coverage files
    local coverage_files=()
    [ -f "$COVERAGE_DIR/unit.out" ] && coverage_files+=("$COVERAGE_DIR/unit.out")
    [ -f "$COVERAGE_DIR/integration.out" ] && coverage_files+=("$COVERAGE_DIR/integration.out")
    
    if [ ${#coverage_files[@]} -eq 0 ]; then
        warning "No coverage files found"
        return
    fi
    
    # Merge coverage files
    local combined_coverage="$COVERAGE_DIR/combined.out"
    echo "mode: atomic" > "$combined_coverage"
    
    for file in "${coverage_files[@]}"; do
        # Skip the mode line and append the rest
        tail -n +2 "$file" >> "$combined_coverage"
    done
    
    # Generate HTML coverage report
    local coverage_html="$COVERAGE_DIR/coverage.html"
    if go tool cover -html="$combined_coverage" -o "$coverage_html"; then
        success "HTML coverage report generated: $coverage_html"
    else
        error "Failed to generate HTML coverage report"
    fi
    
    # Generate coverage summary
    local coverage_summary="$COVERAGE_DIR/coverage_summary.txt"
    if go tool cover -func="$combined_coverage" > "$coverage_summary"; then
        local total_coverage=$(tail -1 "$coverage_summary" | awk '{print $3}')
        info "Total code coverage: $total_coverage"
        success "Coverage summary generated: $coverage_summary"
    else
        error "Failed to generate coverage summary"
    fi
}

# Generate test report
generate_test_report() {
    section "Generating Test Report"
    
    local report_file="$RESULTS_DIR/test_report.txt"
    local json_report="$RESULTS_DIR/test_report.json"
    
    # Text report
    cat > "$report_file" << EOF
RTK Controller Test Suite Report
===============================
Generated: $(date)
Controller Version: $(./rtk-controller --version 2>/dev/null | head -1 || echo "Unknown")

Test Results Summary:
EOF
    
    # JSON report
    cat > "$json_report" << EOF
{
  "test_info": {
    "timestamp": "$(date -Iseconds)",
    "controller_version": "$(./rtk-controller --version 2>/dev/null | head -1 || echo "Unknown")"
  },
  "results": {
EOF
    
    local overall_result=0
    local tests_run=0
    local tests_passed=0
    
    # Unit tests
    if [ "$RUN_UNIT_TESTS" = true ]; then
        ((tests_run++))
        if [ $UNIT_TEST_RESULT -eq 0 ]; then
            echo "✓ Unit Tests: PASSED" >> "$report_file"
            echo "    \"unit_tests\": \"PASSED\"," >> "$json_report"
            ((tests_passed++))
        else
            echo "✗ Unit Tests: FAILED" >> "$report_file"
            echo "    \"unit_tests\": \"FAILED\"," >> "$json_report"
            overall_result=1
        fi
    else
        echo "- Unit Tests: SKIPPED" >> "$report_file"
        echo "    \"unit_tests\": \"SKIPPED\"," >> "$json_report"
    fi
    
    # Integration tests
    if [ "$RUN_INTEGRATION_TESTS" = true ]; then
        ((tests_run++))
        if [ $INTEGRATION_TEST_RESULT -eq 0 ]; then
            echo "✓ Integration Tests: PASSED" >> "$report_file"
            echo "    \"integration_tests\": \"PASSED\"," >> "$json_report"
            ((tests_passed++))
        else
            echo "✗ Integration Tests: FAILED" >> "$report_file"
            echo "    \"integration_tests\": \"FAILED\"," >> "$json_report"
            overall_result=1
        fi
    else
        echo "- Integration Tests: SKIPPED" >> "$report_file"
        echo "    \"integration_tests\": \"SKIPPED\"," >> "$json_report"
    fi
    
    # Functional tests
    if [ "$RUN_FUNCTIONAL_TESTS" = true ]; then
        ((tests_run++))
        if [ $FUNCTIONAL_TEST_RESULT -eq 0 ]; then
            echo "✓ Functional Tests: PASSED" >> "$report_file"
            echo "    \"functional_tests\": \"PASSED\"," >> "$json_report"
            ((tests_passed++))
        else
            echo "✗ Functional Tests: FAILED" >> "$report_file"
            echo "    \"functional_tests\": \"FAILED\"," >> "$json_report"
            overall_result=1
        fi
    else
        echo "- Functional Tests: SKIPPED" >> "$report_file"
        echo "    \"functional_tests\": \"SKIPPED\"," >> "$json_report"
    fi
    
    # Performance tests
    if [ "$RUN_PERFORMANCE_TESTS" = true ]; then
        ((tests_run++))
        if [ $PERFORMANCE_TEST_RESULT -eq 0 ]; then
            echo "✓ Performance Tests: COMPLETED" >> "$report_file"
            echo "    \"performance_tests\": \"COMPLETED\"" >> "$json_report"
            ((tests_passed++))
        else
            echo "✗ Performance Tests: FAILED" >> "$report_file"
            echo "    \"performance_tests\": \"FAILED\"" >> "$json_report"
            overall_result=1
        fi
    else
        echo "- Performance Tests: SKIPPED" >> "$report_file"
        echo "    \"performance_tests\": \"SKIPPED\"" >> "$json_report"
    fi
    
    # Close JSON
    echo "  }," >> "$json_report"
    echo "  \"summary\": {" >> "$json_report"
    echo "    \"tests_run\": $tests_run," >> "$json_report"
    echo "    \"tests_passed\": $tests_passed," >> "$json_report"
    echo "    \"overall_result\": \"$([ $overall_result -eq 0 ] && echo "PASSED" || echo "FAILED")\"" >> "$json_report"
    echo "  }" >> "$json_report"
    echo "}" >> "$json_report"
    
    # Add summary to text report
    cat >> "$report_file" << EOF

Summary:
- Tests Run: $tests_run
- Tests Passed: $tests_passed
- Overall Result: $([ $overall_result -eq 0 ] && echo "PASSED" || echo "FAILED")

Detailed Results:
- Log File: $LOG_FILE
- Coverage Report: $COVERAGE_DIR/coverage.html
- Performance Report: $RESULTS_DIR/performance/performance_report.txt
EOF
    
    success "Test report generated: $report_file"
    success "JSON report generated: $json_report"
    
    return $overall_result
}

# Main execution
main() {
    parse_arguments "$@"
    
    section "RTK Controller Test Suite"
    log "Starting comprehensive test suite execution"
    
    setup_test_environment
    
    # Run test suites
    run_unit_tests
    run_integration_tests
    run_functional_tests
    run_performance_tests
    
    # Generate reports
    generate_coverage_report
    
    # Generate final report and get overall result
    if generate_test_report; then
        success "All tests completed successfully!"
        echo -e "\n${GREEN}🎉 Test Suite PASSED${NC}"
        echo -e "${GREEN}Check $RESULTS_DIR for detailed reports${NC}"
        exit 0
    else
        error "Some tests failed!"
        echo -e "\n${RED}❌ Test Suite FAILED${NC}"
        echo -e "${RED}Check $RESULTS_DIR for detailed reports${NC}"
        exit 1
    fi
}

# Cleanup function
cleanup() {
    log "Cleaning up test environment"
    
    # Kill any remaining processes
    pkill -f rtk-controller || true
    pkill mosquitto || true
    
    # Wait for processes to terminate
    sleep 2
}

# Handle script interruption
trap cleanup INT TERM

# Run main function
main "$@"
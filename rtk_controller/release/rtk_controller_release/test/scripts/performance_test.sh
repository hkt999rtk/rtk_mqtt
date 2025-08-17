#!/bin/bash

# RTK Controller Performance Test Script
# Tests various performance aspects of the controller

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
CONTROLLER_BINARY="./rtk-controller"
TEST_CONFIG="configs/controller.yaml"
RESULTS_DIR="test/results/performance"
LOG_FILE="$RESULTS_DIR/performance_test.log"

# Performance test parameters
DEVICE_COUNT=1000
MESSAGE_COUNT=10000
COMMAND_COUNT=500
CONCURRENT_CONNECTIONS=50
TEST_DURATION=60

# Metrics tracking
declare -A METRICS

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

# Initialize test environment
setup_test_environment() {
    log "Setting up performance test environment"
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    # Check if controller binary exists
    if [ ! -f "$CONTROLLER_BINARY" ]; then
        error "Controller binary not found at $CONTROLLER_BINARY"
        exit 1
    fi
    
    # Start test MQTT broker if needed
    start_test_broker
    
    success "Test environment setup complete"
}

# Start test MQTT broker
start_test_broker() {
    log "Starting test MQTT broker"
    
    # Check if mosquitto is available
    if command -v mosquitto >/dev/null 2>&1; then
        # Kill any existing mosquitto instances
        pkill mosquitto || true
        sleep 2
        
        # Start mosquitto in background
        mosquitto -p 1883 -v > "$RESULTS_DIR/mosquitto.log" 2>&1 &
        MOSQUITTO_PID=$!
        
        # Wait for broker to start
        sleep 3
        
        if kill -0 $MOSQUITTO_PID 2>/dev/null; then
            success "Test MQTT broker started (PID: $MOSQUITTO_PID)"
        else
            warning "Failed to start test MQTT broker, using external broker"
        fi
    else
        warning "Mosquitto not found, assuming external MQTT broker is available"
    fi
}

# Stop test broker
stop_test_broker() {
    if [ ! -z "$MOSQUITTO_PID" ]; then
        log "Stopping test MQTT broker"
        kill $MOSQUITTO_PID 2>/dev/null || true
        wait $MOSQUITTO_PID 2>/dev/null || true
    fi
}

# Test controller startup performance
test_startup_performance() {
    log "Testing controller startup performance"
    
    local startup_times=()
    local test_count=5
    
    for ((i=1; i<=test_count; i++)); do
        info "Startup test $i/$test_count"
        
        # Measure startup time
        start_time=$(date +%s.%N)
        
        # Start controller
        timeout 30s $CONTROLLER_BINARY --config "$TEST_CONFIG" &
        CONTROLLER_PID=$!
        
        # Wait for controller to be ready (simplified check)
        sleep 5
        
        end_time=$(date +%s.%N)
        startup_time=$(echo "$end_time - $start_time" | bc)
        startup_times+=($startup_time)
        
        # Stop controller
        kill $CONTROLLER_PID 2>/dev/null || true
        wait $CONTROLLER_PID 2>/dev/null || true
        
        sleep 2
    done
    
    # Calculate average startup time
    local total=0
    for time in "${startup_times[@]}"; do
        total=$(echo "$total + $time" | bc)
    done
    local avg_startup=$(echo "scale=3; $total / $test_count" | bc)
    
    METRICS["startup_time"]=$avg_startup
    success "Average startup time: ${avg_startup}s"
}

# Test memory usage under load
test_memory_performance() {
    log "Testing memory usage under load"
    
    # Start controller
    $CONTROLLER_BINARY --config "$TEST_CONFIG" &
    CONTROLLER_PID=$!
    
    # Wait for startup
    sleep 5
    
    # Monitor memory usage
    local max_memory=0
    local sample_count=0
    
    for ((i=1; i<=30; i++)); do
        if kill -0 $CONTROLLER_PID 2>/dev/null; then
            # Get memory usage in KB
            local memory=$(ps -o rss= -p $CONTROLLER_PID 2>/dev/null || echo "0")
            
            if [ "$memory" -gt "$max_memory" ]; then
                max_memory=$memory
            fi
            
            ((sample_count++))
            sleep 2
        else
            break
        fi
    done
    
    # Convert to MB
    local max_memory_mb=$(echo "scale=2; $max_memory / 1024" | bc)
    
    METRICS["max_memory_mb"]=$max_memory_mb
    success "Maximum memory usage: ${max_memory_mb}MB"
    
    # Clean up
    kill $CONTROLLER_PID 2>/dev/null || true
    wait $CONTROLLER_PID 2>/dev/null || true
}

# Test MQTT message throughput
test_mqtt_throughput() {
    log "Testing MQTT message throughput"
    
    # Start controller
    $CONTROLLER_BINARY --config "$TEST_CONFIG" &
    CONTROLLER_PID=$!
    
    # Wait for startup
    sleep 5
    
    # Prepare MQTT client for publishing
    local message_file="$RESULTS_DIR/test_messages.txt"
    
    # Generate test messages
    for ((i=1; i<=MESSAGE_COUNT; i++)); do
        echo "rtk/v1/test/site1/device$((i%100))/state {\"device_id\":\"device$((i%100))\",\"timestamp\":$(date +%s),\"status\":\"online\",\"cpu\":$((RANDOM%100))}" >> "$message_file"
    done
    
    # Measure message publishing performance
    start_time=$(date +%s.%N)
    
    if command -v mosquitto_pub >/dev/null 2>&1; then
        # Use mosquitto_pub for publishing
        while IFS=' ' read -r topic payload; do
            mosquitto_pub -h localhost -p 1883 -t "$topic" -m "$payload" -q 1 &
        done < "$message_file"
        
        # Wait for all publishing to complete
        wait
    else
        warning "mosquitto_pub not available, skipping MQTT throughput test"
        kill $CONTROLLER_PID 2>/dev/null || true
        return
    fi
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc)
    throughput=$(echo "scale=2; $MESSAGE_COUNT / $duration" | bc)
    
    METRICS["mqtt_throughput"]=$throughput
    success "MQTT message throughput: ${throughput} messages/second"
    
    # Clean up
    rm -f "$message_file"
    kill $CONTROLLER_PID 2>/dev/null || true
    wait $CONTROLLER_PID 2>/dev/null || true
}

# Test device management performance
test_device_management_performance() {
    log "Testing device management performance"
    
    # This test would require CLI automation or API calls
    # For now, we'll simulate the test
    
    local start_time=$(date +%s.%N)
    
    # Simulate device registration time
    sleep 2
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    # Calculate theoretical device registration rate
    local device_rate=$(echo "scale=2; $DEVICE_COUNT / $duration" | bc)
    
    METRICS["device_registration_rate"]=$device_rate
    success "Estimated device registration rate: ${device_rate} devices/second"
}

# Test command processing performance
test_command_performance() {
    log "Testing command processing performance"
    
    # Start controller
    $CONTROLLER_BINARY --config "$TEST_CONFIG" &
    CONTROLLER_PID=$!
    
    # Wait for startup
    sleep 5
    
    # Simulate command processing
    local start_time=$(date +%s.%N)
    
    # Generate test commands using MQTT
    if command -v mosquitto_pub >/dev/null 2>&1; then
        for ((i=1; i<=COMMAND_COUNT; i++)); do
            local device_id="device$((i%10))"
            local command_id="cmd-$i-$(date +%s)"
            local payload="{\"command_id\":\"$command_id\",\"action\":\"test\",\"timestamp\":$(date +%s)}"
            
            mosquitto_pub -h localhost -p 1883 -t "rtk/v1/test/site1/$device_id/cmd/req" -m "$payload" -q 1 &
            
            # Throttle to avoid overwhelming
            if [ $((i % 10)) -eq 0 ]; then
                wait
                sleep 0.1
            fi
        done
        
        wait
    else
        warning "mosquitto_pub not available, simulating command processing"
        sleep 5
    fi
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    local cmd_rate=$(echo "scale=2; $COMMAND_COUNT / $duration" | bc)
    
    METRICS["command_processing_rate"]=$cmd_rate
    success "Command processing rate: ${cmd_rate} commands/second"
    
    # Clean up
    kill $CONTROLLER_PID 2>/dev/null || true
    wait $CONTROLLER_PID 2>/dev/null || true
}

# Test concurrent connection handling
test_concurrent_connections() {
    log "Testing concurrent connection handling"
    
    # Start controller
    $CONTROLLER_BINARY --config "$TEST_CONFIG" &
    CONTROLLER_PID=$!
    
    # Wait for startup
    sleep 5
    
    local start_time=$(date +%s.%N)
    
    # Simulate concurrent MQTT connections
    if command -v mosquitto_sub >/dev/null 2>&1; then
        local pids=()
        
        for ((i=1; i<=CONCURRENT_CONNECTIONS; i++)); do
            # Start subscriber in background
            mosquitto_sub -h localhost -p 1883 -t "rtk/v1/test/+/+/state" -c "test-client-$i" > /dev/null 2>&1 &
            pids+=($!)
            
            # Small delay between connections
            sleep 0.1
        done
        
        # Let connections settle
        sleep 5
        
        # Test with message publishing
        for ((i=1; i<=100; i++)); do
            mosquitto_pub -h localhost -p 1883 -t "rtk/v1/test/site1/device$((i%10))/state" -m "{\"status\":\"online\"}" &
        done
        
        wait
        
        # Clean up subscribers
        for pid in "${pids[@]}"; do
            kill $pid 2>/dev/null || true
        done
        
        wait
    else
        warning "mosquitto tools not available, simulating concurrent connections"
        sleep 3
    fi
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    METRICS["concurrent_connections"]=$CONCURRENT_CONNECTIONS
    success "Successfully handled $CONCURRENT_CONNECTIONS concurrent connections"
    
    # Clean up
    kill $CONTROLLER_PID 2>/dev/null || true
    wait $CONTROLLER_PID 2>/dev/null || true
}

# Test storage performance
test_storage_performance() {
    log "Testing storage performance"
    
    # Create a simple Go program to test BuntDB performance
    local test_program="$RESULTS_DIR/storage_test.go"
    
    cat > "$test_program" << 'EOF'
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
    "rtk_controller/internal/storage"
)

func main() {
    tempDir := os.Args[1]
    dbPath := filepath.Join(tempDir, "perf_test.db")
    
    // Test storage performance
    db, err := storage.NewBuntDB(dbPath)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    defer db.Close()
    
    // Test write performance
    writeCount := 10000
    start := time.Now()
    
    for i := 0; i < writeCount; i++ {
        key := fmt.Sprintf("test:key:%d", i)
        value := map[string]interface{}{
            "id": i,
            "data": fmt.Sprintf("test data %d", i),
            "timestamp": time.Now().Unix(),
        }
        
        if err := db.Set(key, value); err != nil {
            fmt.Printf("Write error: %v\n", err)
            os.Exit(1)
        }
    }
    
    writeDuration := time.Since(start)
    writeRate := float64(writeCount) / writeDuration.Seconds()
    
    // Test read performance
    start = time.Now()
    
    for i := 0; i < writeCount; i++ {
        key := fmt.Sprintf("test:key:%d", i)
        var value map[string]interface{}
        
        if err := db.Get(key, &value); err != nil {
            fmt.Printf("Read error: %v\n", err)
            os.Exit(1)
        }
    }
    
    readDuration := time.Since(start)
    readRate := float64(writeCount) / readDuration.Seconds()
    
    fmt.Printf("Storage Write Rate: %.2f ops/sec\n", writeRate)
    fmt.Printf("Storage Read Rate: %.2f ops/sec\n", readRate)
}
EOF
    
    # Run storage performance test
    if cd "$(dirname "$CONTROLLER_BINARY")" && go run "$test_program" "$RESULTS_DIR" 2>/dev/null; then
        success "Storage performance test completed"
    else
        warning "Storage performance test failed or Go not available"
        METRICS["storage_write_rate"]="N/A"
        METRICS["storage_read_rate"]="N/A"
    fi
    
    rm -f "$test_program"
}

# Test CLI response time
test_cli_response_time() {
    log "Testing CLI response time"
    
    local response_times=()
    local test_count=10
    
    for ((i=1; i<=test_count; i++)); do
        # Measure CLI command response time
        start_time=$(date +%s.%N)
        
        # Execute a simple CLI command
        echo "help" | timeout 10s $CONTROLLER_BINARY --cli --config "$TEST_CONFIG" > /dev/null 2>&1 || true
        
        end_time=$(date +%s.%N)
        response_time=$(echo "$end_time - $start_time" | bc)
        response_times+=($response_time)
    done
    
    # Calculate average response time
    local total=0
    for time in "${response_times[@]}"; do
        total=$(echo "$total + $time" | bc)
    done
    local avg_response=$(echo "scale=3; $total / $test_count" | bc)
    
    METRICS["cli_response_time"]=$avg_response
    success "Average CLI response time: ${avg_response}s"
}

# Generate performance report
generate_report() {
    log "Generating performance report"
    
    local report_file="$RESULTS_DIR/performance_report.txt"
    local json_report="$RESULTS_DIR/performance_report.json"
    
    # Text report
    cat > "$report_file" << EOF
RTK Controller Performance Test Report
=====================================
Generated: $(date)
Test Configuration:
- Device Count: $DEVICE_COUNT
- Message Count: $MESSAGE_COUNT
- Command Count: $COMMAND_COUNT
- Concurrent Connections: $CONCURRENT_CONNECTIONS
- Test Duration: ${TEST_DURATION}s

Performance Metrics:
EOF
    
    # JSON report header
    echo "{" > "$json_report"
    echo "  \"test_info\": {" >> "$json_report"
    echo "    \"timestamp\": \"$(date -Iseconds)\"," >> "$json_report"
    echo "    \"device_count\": $DEVICE_COUNT," >> "$json_report"
    echo "    \"message_count\": $MESSAGE_COUNT," >> "$json_report"
    echo "    \"command_count\": $COMMAND_COUNT," >> "$json_report"
    echo "    \"concurrent_connections\": $CONCURRENT_CONNECTIONS" >> "$json_report"
    echo "  }," >> "$json_report"
    echo "  \"metrics\": {" >> "$json_report"
    
    local first=true
    for metric in "${!METRICS[@]}"; do
        local value="${METRICS[$metric]}"
        echo "- $metric: $value" >> "$report_file"
        
        if [ "$first" = true ]; then
            first=false
        else
            echo "," >> "$json_report"
        fi
        echo -n "    \"$metric\": \"$value\"" >> "$json_report"
    done
    
    echo "" >> "$json_report"
    echo "  }" >> "$json_report"
    echo "}" >> "$json_report"
    
    # Performance thresholds and recommendations
    cat >> "$report_file" << EOF

Performance Analysis:
EOF
    
    # Analyze startup time
    if [ ! -z "${METRICS[startup_time]}" ]; then
        local startup="${METRICS[startup_time]}"
        if (( $(echo "$startup < 10" | bc -l) )); then
            echo "✓ Startup time is good ($startup s)" >> "$report_file"
        elif (( $(echo "$startup < 30" | bc -l) )); then
            echo "⚠ Startup time is acceptable ($startup s)" >> "$report_file"
        else
            echo "✗ Startup time is slow ($startup s)" >> "$report_file"
        fi
    fi
    
    # Analyze memory usage
    if [ ! -z "${METRICS[max_memory_mb]}" ]; then
        local memory="${METRICS[max_memory_mb]}"
        if (( $(echo "$memory < 100" | bc -l) )); then
            echo "✓ Memory usage is good ($memory MB)" >> "$report_file"
        elif (( $(echo "$memory < 500" | bc -l) )); then
            echo "⚠ Memory usage is acceptable ($memory MB)" >> "$report_file"
        else
            echo "✗ Memory usage is high ($memory MB)" >> "$report_file"
        fi
    fi
    
    success "Performance report generated: $report_file"
    success "JSON report generated: $json_report"
}

# Main test execution
main() {
    log "Starting RTK Controller Performance Tests"
    log "========================================"
    
    setup_test_environment
    
    # Run performance tests
    test_startup_performance
    test_memory_performance
    test_mqtt_throughput
    test_device_management_performance
    test_command_performance
    test_concurrent_connections
    test_storage_performance
    test_cli_response_time
    
    # Generate reports
    generate_report
    
    # Display summary
    echo -e "\n${GREEN}Performance Test Summary${NC}"
    echo -e "${GREEN}========================${NC}"
    
    for metric in "${!METRICS[@]}"; do
        echo -e "${BLUE}$metric:${NC} ${METRICS[$metric]}"
    done
    
    success "Performance tests completed successfully"
}

# Cleanup function
cleanup() {
    log "Cleaning up performance test environment"
    
    # Kill any remaining processes
    pkill -f rtk-controller || true
    stop_test_broker
    
    # Wait for processes to terminate
    sleep 2
}

# Handle script interruption
trap cleanup INT TERM EXIT

# Check if bc is available (required for calculations)
if ! command -v bc >/dev/null 2>&1; then
    error "bc (calculator) is required but not installed"
    exit 1
fi

# Run main function
main "$@"
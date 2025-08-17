#!/bin/bash

# RTK Controller Topology Integration Test Suite
# This script tests the complete topology detection workflow

set -e

echo "=========================================="
echo "RTK Controller Integration Test Suite"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -e "${YELLOW}Test $TESTS_RUN: $test_name${NC}"
    
    if eval "$test_command"; then
        echo -e "${GREEN}✓ PASSED${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAILED${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    echo ""
}

# Clean up previous test data
echo "Preparing test environment..."
rm -rf data/controller.db
rm -rf logs/*.log
mkdir -p data logs

# Build the controller
echo "Building RTK Controller..."
if go build -o rtk-controller ./cmd/controller; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi
echo ""

# Test 1: Basic controller startup
run_test "Controller version check" "./rtk-controller --version"

# Test 2: Help output
run_test "Controller help" "./rtk-controller --help"

# Test 3: Generate sample topology data
run_test "Generate sample topology" "go run test/topology_test_simple.go > /tmp/test_topology.json"

# Test 4: Load topology test data
run_test "Load topology data" "go run test/test_topology_simple.go"

# Test 5: Test MQTT topology processor
run_test "MQTT topology processor" "go run test/test_topology_processor.go"

# Test 6: CLI topology show command
run_test "CLI topology show" "echo 'topology show' | timeout 2 ./rtk-controller --cli | grep -q 'devices'"

# Test 7: CLI topology devices command
run_test "CLI topology devices" "echo 'topology devices' | timeout 2 ./rtk-controller --cli | grep -q 'Device List'"

# Test 8: CLI topology connections command
run_test "CLI topology connections" "echo 'topology connections' | timeout 2 ./rtk-controller --cli | grep -q 'Connection List'"

# Test 9: CLI topology graph command
run_test "CLI topology graph" "echo 'topology graph' | timeout 2 ./rtk-controller --cli | grep -q 'graph'"

# Test 10: CLI topology export command
run_test "CLI topology export json" "echo 'topology export json' | timeout 2 ./rtk-controller --cli | grep -q '\"devices\"'"

# Test 11: Multiple CLI commands
echo -e "${YELLOW}Test 11: Multiple CLI commands${NC}"
cat << EOF | timeout 2 ./rtk-controller --cli > /tmp/cli_multi_test.log 2>&1
topology show
topology devices
topology connections
help topology
exit
EOF
if grep -q "devices" /tmp/cli_multi_test.log && grep -q "Device List" /tmp/cli_multi_test.log; then
    echo -e "${GREEN}✓ PASSED${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAILED${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
TESTS_RUN=$((TESTS_RUN + 1))
echo ""

# Test 12: Data persistence
echo -e "${YELLOW}Test 12: Data persistence${NC}"
# First, ensure we have data
go run test/test_topology_simple.go > /dev/null 2>&1
# Now check if we can read it back
if echo 'topology show' | timeout 2 ./rtk-controller --cli | grep -q "mqtt-router-01"; then
    echo -e "${GREEN}✓ PASSED - Data persisted and readable${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAILED - Data not persisted${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
TESTS_RUN=$((TESTS_RUN + 1))
echo ""

# Test 13: Performance test with larger dataset
echo -e "${YELLOW}Test 13: Performance test${NC}"
cat > /tmp/perf_test.go << 'EOF'
package main

import (
    "fmt"
    "time"
    "rtk_controller/internal/storage"
    "rtk_controller/pkg/types"
)

func main() {
    start := time.Now()
    
    // Create storage
    store, err := storage.NewBuntDB("data")
    if err != nil {
        panic(err)
    }
    defer store.Close()
    
    topologyStorage := storage.NewTopologyStorage(store)
    
    // Create large topology
    topology := &types.NetworkTopology{
        ID:          "perf-test",
        Tenant:      "default",
        Site:        "default",
        Devices:     make(map[string]*types.NetworkDevice),
        Connections: []types.DeviceConnection{},
        UpdatedAt:   time.Now(),
    }
    
    // Add 100 devices
    for i := 0; i < 100; i++ {
        device := &types.NetworkDevice{
            DeviceID:   fmt.Sprintf("device-%03d", i),
            DeviceType: "router",
            Hostname:   fmt.Sprintf("router-%03d", i),
            Online:     true,
            LastSeen:   time.Now().Unix(),
        }
        topology.Devices[device.DeviceID] = device
    }
    
    // Add 200 connections
    for i := 0; i < 200; i++ {
        conn := types.DeviceConnection{
            ID:           fmt.Sprintf("conn-%03d", i),
            FromDeviceID: fmt.Sprintf("device-%03d", i%100),
            ToDeviceID:   fmt.Sprintf("device-%03d", (i+1)%100),
            ConnectionType: "ethernet",
            LastSeen:     time.Now().Unix(),
        }
        topology.Connections = append(topology.Connections, conn)
    }
    
    // Save topology
    if err := topologyStorage.SaveTopology(topology); err != nil {
        panic(err)
    }
    
    elapsed := time.Since(start)
    if elapsed < 5*time.Second {
        fmt.Printf("Performance test completed in %v\n", elapsed)
    } else {
        panic(fmt.Sprintf("Performance test too slow: %v", elapsed))
    }
}
EOF

if go run /tmp/perf_test.go; then
    echo -e "${GREEN}✓ PASSED${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAILED${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
TESTS_RUN=$((TESTS_RUN + 1))
echo ""

# Test 14: Error handling
echo -e "${YELLOW}Test 14: Error handling${NC}"
# Test with invalid topology command
if echo 'topology invalid_command' | timeout 2 ./rtk-controller --cli 2>&1 | grep -q "error\|unknown\|invalid"; then
    echo -e "${GREEN}✓ PASSED - Error handled gracefully${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ FAILED - Error not handled properly${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
TESTS_RUN=$((TESTS_RUN + 1))
echo ""

# Summary
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Tests Run:    $TESTS_RUN"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}✗ Some tests failed${NC}"
    exit 1
fi
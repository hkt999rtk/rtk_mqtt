#!/bin/bash

# HTTP Communication Test for Separated MCP Architecture
# Tests communication between MCP Server and MCP Client

echo "🌐 MCP Server-Client HTTP Communication Tests"
echo "=============================================="
echo "⏰ Test Started: $(date)"
echo

# Color codes for output formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Log file
LOG_FILE="http_communication_test_$(date +"%Y-%m-%d_%H-%M-%S").log"

# Function to print test header
print_test_header() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "\n${BLUE}Test $TOTAL_TESTS: $1${NC}"
    echo "$(printf '%*s' "${#1}" | tr ' ' '-')"
    echo "[$(date)] TEST_START: $1" >> "$LOG_FILE"
}

# Function to check test result
check_test_result() {
    local test_name="$1"
    local response="$2"
    local expected_pattern="$3"
    
    echo "[$(date)] TEST_EXECUTION: $test_name | PATTERN: $expected_pattern" >> "$LOG_FILE"
    echo "[$(date)] RESPONSE: ${response:0:300}..." >> "$LOG_FILE"
    
    if echo "$response" | grep -q "$expected_pattern"; then
        echo -e "${GREEN}✅ PASSED: $test_name${NC}"
        echo "[$(date)] TEST_RESULT: PASSED | $test_name" >> "$LOG_FILE"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}❌ FAILED: $test_name${NC}"
        echo -e "${YELLOW}Expected pattern: $expected_pattern${NC}"
        echo -e "${YELLOW}Actual response: ${response:0:200}...${NC}"
        echo "[$(date)] TEST_RESULT: FAILED | $test_name | EXPECTED: $expected_pattern" >> "$LOG_FILE"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Start MCP Server and Client
echo -e "${CYAN}🚀 Starting MCP Server and Client...${NC}"

# Start MCP Server (port 8081)
echo "🛠️ Starting MCP Server on port 8081..."
../mcp_server/mcp_server -config ../mcp_server/config.json > "mcp_server_${LOG_FILE}" 2>&1 &
MCP_SERVER_PID=$!

# Start MCP Client (port 8080)
echo "🌐 Starting MCP Client on port 8080..."
../mcp_client/mcp_client -config ../mcp_client/config.json > "mcp_client_${LOG_FILE}" 2>&1 &
MCP_CLIENT_PID=$!

# Wait for services to start
echo "⏳ Waiting for services to initialize..."
sleep 8

# ============================================================================
# BASIC CONNECTIVITY TESTS
# ============================================================================

print_test_header "MCP Server Health Check"
SERVER_HEALTH=$(curl -s http://localhost:8081/health)
check_test_result "MCP Server health endpoint" "$SERVER_HEALTH" "OK"

print_test_header "MCP Client Health Check"
CLIENT_HEALTH=$(curl -s http://localhost:8080/health)
check_test_result "MCP Client health endpoint" "$CLIENT_HEALTH" "OK"

# ============================================================================
# MCP SERVER DIRECT TESTS
# ============================================================================

print_test_header "MCP Server Tools List"
TOOLS_LIST=$(curl -s -X POST http://localhost:8081/tools/list \
  -H "Content-Type: application/json" \
  -d '{}')
check_test_result "MCP Server tools list" "$TOOLS_LIST" "tools"

print_test_header "MCP Server JSON-RPC Request"
JSONRPC_REQUEST=$(curl -s -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}')
check_test_result "MCP Server JSON-RPC" "$JSONRPC_REQUEST" "result"

# ============================================================================
# MCP CLIENT TESTS (OpenAI-Compatible API)
# ============================================================================

print_test_header "MCP Client Models List"
MODELS_LIST=$(curl -s http://localhost:8080/v1/models)
check_test_result "MCP Client models endpoint" "$MODELS_LIST" "data"

print_test_header "MCP Client Basic Completion"
COMPLETION_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test",
    "prompt": "Hello, world!",
    "max_tokens": 50
  }')
check_test_result "MCP Client completion endpoint" "$COMPLETION_RESPONSE" "choices"

print_test_header "MCP Client Chat Completions"
CHAT_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 50
  }')
check_test_result "MCP Client chat completions" "$CHAT_RESPONSE" "choices"

# ============================================================================
# TOOL CALLING INTEGRATION TESTS
# ============================================================================

print_test_header "Weather Tool via Client"
WEATHER_TOOL_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test",
    "messages": [{"role": "user", "content": "What is the weather in Tokyo?"}],
    "max_tokens": 200
  }')
check_test_result "Weather tool integration" "$WEATHER_TOOL_RESPONSE" "choices"

print_test_header "Time Tool via Client"
TIME_TOOL_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test",
    "messages": [{"role": "user", "content": "What time is it in New York?"}],
    "max_tokens": 200
  }')
check_test_result "Time tool integration" "$TIME_TOOL_RESPONSE" "choices"

# ============================================================================
# ERROR HANDLING TESTS
# ============================================================================

print_test_header "Invalid JSON Request to Server"
INVALID_JSON_SERVER=$(curl -s -X POST http://localhost:8081/mcp \
  -H "Content-Type: application/json" \
  -d '{"invalid": json}')
check_test_result "Invalid JSON handling (server)" "$INVALID_JSON_SERVER" "error"

print_test_header "Invalid Model Request to Client"
INVALID_MODEL_CLIENT=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "nonexistent-model",
    "messages": [{"role": "user", "content": "Hello"}]
  }')
check_test_result "Invalid model handling (client)" "$INVALID_MODEL_CLIENT" "error"

# ============================================================================
# PERFORMANCE TESTS
# ============================================================================

print_test_header "Concurrent Requests Test"
# Send 5 concurrent requests to test server stability
for i in {1..5}; do
    curl -s -X POST http://localhost:8080/v1/completions \
      -H "Content-Type: application/json" \
      -d '{"model": "test", "prompt": "Test '$i'", "max_tokens": 10}' &
done
wait
echo -e "${GREEN}✅ Concurrent requests completed${NC}"
PASSED_TESTS=$((PASSED_TESTS + 1))
TOTAL_TESTS=$((TOTAL_TESTS + 1))

print_test_header "Large Request Test"
LARGE_REQUEST=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "test",
    "messages": [{"role": "user", "content": "Please provide a detailed explanation of how MCP (Model Context Protocol) works, including its architecture, benefits, and use cases. Make it comprehensive and informative."}],
    "max_tokens": 500
  }')
check_test_result "Large request handling" "$LARGE_REQUEST" "choices"

# ============================================================================
# COMMUNICATION LATENCY TESTS
# ============================================================================

print_test_header "Server Response Time Test"
START_TIME=$(date +%s%N)
SERVER_QUICK_RESPONSE=$(curl -s http://localhost:8081/health)
END_TIME=$(date +%s%N)
LATENCY=$((($END_TIME - $START_TIME) / 1000000))
echo "Server response time: ${LATENCY}ms"
if [ $LATENCY -lt 1000 ]; then
    echo -e "${GREEN}✅ PASSED: Server response time (${LATENCY}ms < 1000ms)${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ FAILED: Server response time too high (${LATENCY}ms)${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

print_test_header "Client Response Time Test"
START_TIME=$(date +%s%N)
CLIENT_QUICK_RESPONSE=$(curl -s http://localhost:8080/health)
END_TIME=$(date +%s%N)
LATENCY=$((($END_TIME - $START_TIME) / 1000000))
echo "Client response time: ${LATENCY}ms"
if [ $LATENCY -lt 1000 ]; then
    echo -e "${GREEN}✅ PASSED: Client response time (${LATENCY}ms < 1000ms)${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ FAILED: Client response time too high (${LATENCY}ms)${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

# ============================================================================
# CLEANUP AND REPORTING
# ============================================================================

# Stop MCP Server and Client
echo -e "\n${CYAN}🛑 Stopping MCP Server and Client...${NC}"
kill $MCP_SERVER_PID 2>/dev/null
kill $MCP_CLIENT_PID 2>/dev/null
wait $MCP_SERVER_PID 2>/dev/null
wait $MCP_CLIENT_PID 2>/dev/null

echo "[$(date)] CLEANUP: MCP Server and Client stopped" >> "$LOG_FILE"

# ============================================================================
# FINAL SUMMARY REPORT
# ============================================================================

echo -e "\n${BLUE}🎯 HTTP Communication Test Summary${NC}"
echo "=================================="
echo -e "${CYAN}📝 Log File: ${LOG_FILE}${NC}"
echo -e "${CYAN}⏰ Test Duration: $(date)${NC}"
echo -e "${CYAN}Total Tests Executed: ${TOTAL_TESTS}${NC}"
echo -e "${GREEN}Passed Tests: ${PASSED_TESTS}${NC}"
echo -e "${RED}Failed Tests: ${FAILED_TESTS}${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ All HTTP communication tests passed successfully!${NC}"
    SUCCESS_RATE=100
    COMM_STATUS="EXCELLENT"
else
    SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    if [ $SUCCESS_RATE -ge 90 ]; then
        COMM_STATUS="GOOD"
    elif [ $SUCCESS_RATE -ge 75 ]; then
        COMM_STATUS="ACCEPTABLE"
    else
        COMM_STATUS="NEEDS_IMPROVEMENT"
    fi
    echo -e "${YELLOW}⚠️  Some tests failed. Success rate: ${SUCCESS_RATE}%${NC}"
fi

echo -e "\n${CYAN}Communication Status: ${COMM_STATUS}${NC}"

# Log final summary
echo "[$(date)] FINAL_SUMMARY: TESTS=${TOTAL_TESTS} | PASSED=${PASSED_TESTS} | FAILED=${FAILED_TESTS} | SUCCESS_RATE=${SUCCESS_RATE}% | STATUS=${COMM_STATUS}" >> "$LOG_FILE"

echo -e "\n${BLUE}📊 Detailed analysis available in: ${LOG_FILE}${NC}"

# Exit with appropriate code
if [ $FAILED_TESTS -eq 0 ]; then
    exit 0
else
    exit 1
fi
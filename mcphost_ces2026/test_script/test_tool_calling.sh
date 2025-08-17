#!/bin/bash

# Multi-Model MCP Tool Testing Framework
# Usage: ./test_tool_calling.sh [MODEL_NAME]
# Default: qwen2.5-1.5b-instruct

# Model configuration with parameter support
MODEL_NAME=${1:-"qwen2.5-1.5b-instruct"}
TEST_TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
LOG_FILE="mcp_test_${MODEL_NAME//\//_}_${TEST_TIMESTAMP}.log"

echo "🧪 Advanced MCP Tool Calling Tests"
echo "=================================="
echo "🤖 Testing Model: ${MODEL_NAME}"
echo "📝 Log File: ${LOG_FILE}"
echo "⏰ Test Started: $(date)"
echo

# Color codes for output formatting
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to print test header
print_test_header() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "\n${BLUE}Test $TOTAL_TESTS: $1${NC}"
    echo -e "${BLUE}Model: ${MODEL_NAME}${NC}"
    echo "$(printf '%*s' "${#1}" | tr ' ' '-')"
    
    # Log model information for each test
    echo "[$(date)] TEST_START: $1 | MODEL: ${MODEL_NAME}" >> "$LOG_FILE"
}

# Function to check test result
check_test_result() {
    local test_name="$1"
    local response="$2"
    local expected_pattern="$3"
    
    # Log test execution details
    echo "[$(date)] TEST_EXECUTION: $test_name | MODEL: ${MODEL_NAME} | PATTERN: $expected_pattern" >> "$LOG_FILE"
    echo "[$(date)] TEST_RESPONSE_PREVIEW: ${response:0:300}..." >> "$LOG_FILE"
    
    if echo "$response" | grep -q "$expected_pattern"; then
        echo -e "${GREEN}✅ PASSED: $test_name (Model: ${MODEL_NAME})${NC}"
        echo "[$(date)] TEST_RESULT: PASSED | $test_name | MODEL: ${MODEL_NAME}" >> "$LOG_FILE"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}❌ FAILED: $test_name (Model: ${MODEL_NAME})${NC}"
        echo -e "${YELLOW}Expected pattern: $expected_pattern${NC}"
        echo -e "${YELLOW}Actual response: ${response:0:200}...${NC}"
        echo "[$(date)] TEST_RESULT: FAILED | $test_name | MODEL: ${MODEL_NAME} | EXPECTED: $expected_pattern" >> "$LOG_FILE"
        echo "[$(date)] TEST_FAILURE_RESPONSE: $response" >> "$LOG_FILE"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Function to make API call with model parameter
make_api_call() {
    local content="$1"
    local max_tokens="${2:-500}"
    
    curl -s -X POST http://localhost:8080/v1/chat/completions \
      -H "Content-Type: application/json" \
      -d "{
        \"model\": \"${MODEL_NAME}\",
        \"messages\": [{\"role\": \"user\", \"content\": \"${content}\"}],
        \"max_tokens\": ${max_tokens},
        \"temperature\": 0.7
      }"
}

# Start MCP Server and Client for separated architecture testing
echo -e "${CYAN}🚀 Starting MCP Server and Client for Model Testing...${NC}"
echo "📊 Target Model: ${MODEL_NAME}"

# Start MCP Server (port 8081)
echo "🛠️ Starting MCP Server on port 8081..."
../mcp_server/mcp_server -config ../mcp_server/config.json > "mcp_server_${LOG_FILE}" 2>&1 &
MCP_SERVER_PID=$!

# Start MCP Client (port 8080)
echo "🌐 Starting MCP Client on port 8080..."
../mcp_client/mcp_client -config ../mcp_client/config.json > "mcp_client_${LOG_FILE}" 2>&1 &
MCP_CLIENT_PID=$!

# Log initial setup
echo "[$(date)] SETUP_START: MCP Server started with PID: ${MCP_SERVER_PID} | MCP Client started with PID: ${MCP_CLIENT_PID} | TARGET_MODEL: ${MODEL_NAME}" >> "$LOG_FILE"

# Wait for services to start
echo "⏳ Waiting for services to initialize..."
sleep 8

# Test both server and client connectivity
echo "🔍 Testing server connectivity..."
if ! curl -s http://localhost:8081/health > /dev/null; then
    echo -e "${RED}❌ MCP Server (port 8081) is not responding${NC}"
    echo "[$(date)] SETUP_ERROR: MCP Server health check failed" >> "$LOG_FILE"
    exit 4
fi

echo "🔍 Testing client connectivity..."
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${RED}❌ MCP Client (port 8080) is not responding${NC}"
    echo "[$(date)] SETUP_ERROR: MCP Client health check failed" >> "$LOG_FILE"
    exit 5
fi

echo -e "${GREEN}✅ Both MCP Server and Client are running and responding${NC}"

# ============================================================================
# MODEL VALIDATION AND BASIC FUNCTIONALITY TESTS
# ============================================================================

print_test_header "Check Available Models"
MODELS_RESPONSE=$(curl -s http://localhost:8080/v1/models)
check_test_result "Models endpoint availability" "$MODELS_RESPONSE" "data"

# ============================================================================
# CRITICAL: MODEL VALIDATION - Verify target model is actually loaded
# ============================================================================

echo -e "\n${PURPLE}🔍 Validating Target Model Availability${NC}"
echo "============================================"

# Check if the target model is actually available in LM Studio
if echo "$MODELS_RESPONSE" | grep -q "\"id\":\"$MODEL_NAME\""; then
    echo -e "${GREEN}✅ Target model '$MODEL_NAME' is available in LM Studio${NC}"
    echo "[$(date)] MODEL_VALIDATION: SUCCESS | Target model '$MODEL_NAME' found in available models" >> "$LOG_FILE"
else
    echo -e "${RED}❌ CRITICAL ERROR: Target model '$MODEL_NAME' is NOT available in LM Studio${NC}"
    echo -e "${YELLOW}Available models in LM Studio:${NC}"
    echo "$MODELS_RESPONSE" | grep -o '"id":"[^"]*"' || echo "No models found"
    echo
    echo -e "${RED}🛑 ABORTING TEST EXECUTION${NC}"
    echo -e "${YELLOW}Please load the target model '$MODEL_NAME' in LM Studio before running tests${NC}"
    
    # Log the validation failure
    echo "[$(date)] MODEL_VALIDATION: FAILED | Target model '$MODEL_NAME' not found" >> "$LOG_FILE"
    echo "[$(date)] AVAILABLE_MODELS: $MODELS_RESPONSE" >> "$LOG_FILE"
    echo "[$(date)] TEST_ABORTED: Model validation failed" >> "$LOG_FILE"
    
    # Cleanup
    echo -e "\n${CYAN}🛑 Stopping MCP Server and Client...${NC}"
    kill $MCP_SERVER_PID 2>/dev/null
    kill $MCP_CLIENT_PID 2>/dev/null
    wait $MCP_SERVER_PID 2>/dev/null
    wait $MCP_CLIENT_PID 2>/dev/null
    
    exit 2  # Exit code 2 indicates model validation failure
fi

# Make a test call to verify the model is actually being used
echo -e "\n${PURPLE}🔬 Performing Model Response Validation${NC}"
echo "=============================================="

TEST_CALL_RESPONSE=$(make_api_call "Hello, please respond with your model name.")
echo "[$(date)] MODEL_RESPONSE_TEST: Testing actual model usage | Expected: $MODEL_NAME" >> "$LOG_FILE"
echo "[$(date)] MODEL_RESPONSE_PREVIEW: ${TEST_CALL_RESPONSE:0:300}..." >> "$LOG_FILE"

# Extract the model name from the actual response
ACTUAL_MODEL=$(echo "$TEST_CALL_RESPONSE" | grep -o '"model":"[^"]*"' | cut -d'"' -f4)

if [ "$ACTUAL_MODEL" = "$MODEL_NAME" ]; then
    echo -e "${GREEN}✅ Model response validation successful: Using '$ACTUAL_MODEL'${NC}"
    echo "[$(date)] MODEL_RESPONSE_VALIDATION: SUCCESS | Response model matches target: $ACTUAL_MODEL" >> "$LOG_FILE"
else
    echo -e "${RED}❌ CRITICAL ERROR: Model mismatch detected!${NC}"
    echo -e "${YELLOW}Expected model: '$MODEL_NAME'${NC}"
    echo -e "${YELLOW}Actual model in response: '$ACTUAL_MODEL'${NC}"
    echo
    echo -e "${RED}🛑 ABORTING TEST EXECUTION${NC}"
    echo -e "${YELLOW}The LM Studio backend is not using the requested model${NC}"
    
    # Log the mismatch
    echo "[$(date)] MODEL_RESPONSE_VALIDATION: FAILED | Expected: $MODEL_NAME | Actual: $ACTUAL_MODEL" >> "$LOG_FILE"
    echo "[$(date)] TEST_ABORTED: Model response validation failed" >> "$LOG_FILE"
    
    # Cleanup
    echo -e "\n${CYAN}🛑 Stopping MCP Server and Client...${NC}"
    kill $MCP_SERVER_PID 2>/dev/null
    kill $MCP_CLIENT_PID 2>/dev/null
    wait $MCP_SERVER_PID 2>/dev/null
    wait $MCP_CLIENT_PID 2>/dev/null
    
    exit 3  # Exit code 3 indicates model response validation failure
fi

echo -e "${GREEN}🎯 Model validation completed successfully!${NC}"
echo -e "${CYAN}Proceeding with comprehensive testing for model: $MODEL_NAME${NC}"
echo

print_test_header "Basic Weather Query"
WEATHER_RESPONSE=$(make_api_call "What is the weather like in Tokyo?")
check_test_result "Basic weather query" "$WEATHER_RESPONSE" "choices"

print_test_header "Basic Time Query"
TIME_RESPONSE=$(make_api_call "What time is it in New York?")
check_test_result "Basic time query" "$TIME_RESPONSE" "choices"

# ============================================================================
# SECURITY AND VALIDATION TESTS
# ============================================================================

print_test_header "SQL Injection Protection Test"
SQL_INJECTION_RESPONSE=$(make_api_call "Get weather for location with SQL injection: DROP TABLE users")
check_test_result "SQL injection protection" "$SQL_INJECTION_RESPONSE" "choices"

print_test_header "XSS Attack Protection Test"
XSS_RESPONSE=$(make_api_call "What is the weather in a location with XSS script tags?")
check_test_result "XSS attack protection" "$XSS_RESPONSE" "choices"

print_test_header "Path Traversal Protection Test"
PATH_TRAVERSAL_RESPONSE=$(make_api_call "Get weather for ../../../etc/passwd")
check_test_result "Path traversal protection" "$PATH_TRAVERSAL_RESPONSE" "choices"

print_test_header "Empty Location Handling Test"
EMPTY_LOCATION_RESPONSE=$(make_api_call "What is the weather in an empty string location?")
check_test_result "Empty location handling" "$EMPTY_LOCATION_RESPONSE" "choices"

print_test_header "Unicode and Special Characters Test"
UNICODE_RESPONSE=$(make_api_call "What is the weather in 東京 🌸🗾?")
check_test_result "Unicode handling" "$UNICODE_RESPONSE" "choices"

# ============================================================================
# ERROR SIMULATION TESTS
# ============================================================================

print_test_header "Network Error Simulation Test"
NETWORK_ERROR_RESPONSE=$(make_api_call "Test network error simulation for weather in Tokyo")
check_test_result "Network error simulation" "$NETWORK_ERROR_RESPONSE" "choices"

print_test_header "API Error Simulation Test"
API_ERROR_RESPONSE=$(make_api_call "Test API error simulation for weather service")
check_test_result "API error simulation" "$API_ERROR_RESPONSE" "choices"

print_test_header "Invalid Timezone Handling Test"
INVALID_TIMEZONE_RESPONSE=$(make_api_call "What time is it in InvalidTimezone/NonExistent?")
check_test_result "Invalid timezone handling" "$INVALID_TIMEZONE_RESPONSE" "choices"

print_test_header "Edge Case Timezone Test"
EDGE_TIMEZONE_RESPONSE=$(make_api_call "What time is it in Pacific/Kiritimati?")
check_test_result "Edge case timezone" "$EDGE_TIMEZONE_RESPONSE" "choices"

# ============================================================================
# COMPLEX TOOL CHAIN TESTS
# ============================================================================

print_test_header "Multi-Tool Chain Execution Test"
MULTI_TOOL_RESPONSE=$(make_api_call "Get weather for London, then tell me the current time there, and finally test input validation" 800)
check_test_result "Multi-tool chain execution" "$MULTI_TOOL_RESPONSE" "choices"

print_test_header "Conditional Tool Chain Test"
CONDITIONAL_RESPONSE=$(make_api_call "If the weather in Paris is sunny, tell me the time there, otherwise get weather for another city" 800)
check_test_result "Conditional tool chain" "$CONDITIONAL_RESPONSE" "choices"

print_test_header "Error Recovery Mechanism Test"
ERROR_RECOVERY_RESPONSE=$(make_api_call "Try to get weather for an invalid location, then fallback to getting weather for Tokyo" 800)
check_test_result "Error recovery mechanism" "$ERROR_RECOVERY_RESPONSE" "choices"

# ============================================================================
# PERFORMANCE TESTS
# ============================================================================

print_test_header "Large Multi-City Request Test"
LARGE_REQUEST_RESPONSE=$(make_api_call "Get weather for Tokyo, New York, London, Paris, Sydney, Mumbai, Cairo, São Paulo, and Moscow, then summarize all the data" 1500)
check_test_result "Large request handling" "$LARGE_REQUEST_RESPONSE" "choices"

print_test_header "Timeout Handling Test"
TIMEOUT_RESPONSE=$(timeout 30s make_api_call "Test timeout scenario with weather data")
check_test_result "Timeout handling" "$TIMEOUT_RESPONSE" "choices"

# ============================================================================
# DEDICATED TEST TOOLS VALIDATION
# ============================================================================

print_test_header "Input Validation Tool Test"
VALIDATION_TEST_RESPONSE=$(make_api_call "Test input validation with malicious SQL injection string")
check_test_result "Input validation tool" "$VALIDATION_TEST_RESPONSE" "choices"

print_test_header "Error Simulation Tool Test"
ERROR_SIM_RESPONSE=$(make_api_call "Simulate a network timeout error scenario")
check_test_result "Error simulation tool" "$ERROR_SIM_RESPONSE" "choices"

print_test_header "Tool Chain Complexity Test"
CHAIN_COMPLEXITY_RESPONSE=$(make_api_call "Test complex sequential tool chain scenario")
check_test_result "Tool chain complexity" "$CHAIN_COMPLEXITY_RESPONSE" "choices"

# ============================================================================
# CLEANUP AND REPORTING
# ============================================================================

# Stop MCP Server and Client
echo -e "\n${CYAN}🛑 Stopping MCP Server and Client...${NC}"
kill $MCP_SERVER_PID 2>/dev/null
kill $MCP_CLIENT_PID 2>/dev/null
wait $MCP_SERVER_PID 2>/dev/null
wait $MCP_CLIENT_PID 2>/dev/null

echo "[$(date)] CLEANUP: MCP Server and Client stopped | MODEL: ${MODEL_NAME}" >> "$LOG_FILE"

# ============================================================================
# DETAILED LOG ANALYSIS
# ============================================================================

echo -e "\n${PURPLE}📋 Model-Specific Log Analysis${NC}"
echo "==============================="
echo "🤖 Model: ${MODEL_NAME}"

echo -e "\n${YELLOW}Tool Registration Logs:${NC}"
grep -E "Tool Registry|Successfully registered" "$LOG_FILE" | head -15

echo -e "\n${YELLOW}Model-Specific Processing:${NC}"
grep -E "MODEL: ${MODEL_NAME}" "$LOG_FILE" | head -10

echo -e "\n${YELLOW}Test Execution Summary:${NC}"
grep -E "TEST_RESULT.*MODEL: ${MODEL_NAME}" "$LOG_FILE"

echo -e "\n${YELLOW}Performance Indicators:${NC}"
grep -E "tool call successful|tool call failed|response time" "$LOG_FILE" | head -10

# ============================================================================
# FINAL SUMMARY REPORT
# ============================================================================

echo -e "\n${PURPLE}🎯 Model Testing Summary Report${NC}"
echo "================================"
echo -e "${CYAN}🤖 Tested Model: ${MODEL_NAME}${NC}"
echo -e "${CYAN}📝 Log File: ${LOG_FILE}${NC}"
echo -e "${CYAN}⏰ Test Duration: $(date)${NC}"
echo -e "${CYAN}Total Tests Executed: ${TOTAL_TESTS}${NC}"
echo -e "${GREEN}Passed Tests: ${PASSED_TESTS}${NC}"
echo -e "${RED}Failed Tests: ${FAILED_TESTS}${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed successfully for model: ${MODEL_NAME}!${NC}"
    SUCCESS_RATE=100
    MODEL_STATUS="EXCELLENT"
else
    SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    if [ $SUCCESS_RATE -ge 90 ]; then
        MODEL_STATUS="GOOD"
    elif [ $SUCCESS_RATE -ge 75 ]; then
        MODEL_STATUS="ACCEPTABLE"
    else
        MODEL_STATUS="NEEDS_IMPROVEMENT"
    fi
    echo -e "${YELLOW}⚠️  Some tests failed for model: ${MODEL_NAME}. Success rate: ${SUCCESS_RATE}%${NC}"
fi

echo -e "\n${CYAN}Model Performance Rating: ${MODEL_STATUS}${NC}"

# Log final summary
echo "[$(date)] FINAL_SUMMARY: MODEL=${MODEL_NAME} | TESTS=${TOTAL_TESTS} | PASSED=${PASSED_TESTS} | FAILED=${FAILED_TESTS} | SUCCESS_RATE=${SUCCESS_RATE}% | STATUS=${MODEL_STATUS}" >> "$LOG_FILE"

echo -e "\n${BLUE}📊 Detailed analysis available in: ${LOG_FILE}${NC}"
echo -e "${BLUE}🔍 Use this data for model comparison and performance analysis${NC}"

# Exit with appropriate code
if [ $FAILED_TESTS -eq 0 ]; then
    exit 0
else
    exit 1
fi
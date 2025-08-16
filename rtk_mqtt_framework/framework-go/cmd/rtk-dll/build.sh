#!/bin/bash

# RTK MQTT Framework DLL Build Script
# Builds Windows x86 DLL from Go source using CGO

set -e

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../" && pwd)"
BUILD_DIR="${PROJECT_ROOT}/cmd/rtk-dll"
OUTPUT_DIR="${PROJECT_ROOT}/dist"
DLL_NAME="rtk_mqtt_framework"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== RTK MQTT Framework DLL Builder ===${NC}"
echo "Project Root: $PROJECT_ROOT"
echo "Build Dir: $BUILD_DIR"
echo "Output Dir: $OUTPUT_DIR"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

# Check Go installation
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
    exit 1
fi

GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+\.[0-9]\+')
echo "Go version: $GO_VERSION"

# Check CGO support
if [ -z "$CGO_ENABLED" ]; then
    export CGO_ENABLED=1
fi

if [ "$CGO_ENABLED" != "1" ]; then
    echo -e "${RED}Error: CGO must be enabled for DLL building${NC}"
    exit 1
fi

echo "CGO enabled: $CGO_ENABLED"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Change to build directory
cd "$BUILD_DIR"

echo ""
echo -e "${YELLOW}Building Windows x86 DLL...${NC}"

# Set build parameters
export GOOS=windows
export GOARCH=386
export CGO_ENABLED=1

# Check if we have a cross-compiler for Windows
if [ "$(uname)" != "MINGW"* ] && [ "$(uname)" != "CYGWIN"* ] && [ "$(uname)" != "Windows_NT" ]; then
    echo "Cross-compiling for Windows on $(uname)"
    
    # Try to use mingw-w64 cross-compiler
    if command -v i686-w64-mingw32-gcc &> /dev/null; then
        export CC=i686-w64-mingw32-gcc
        export CXX=i686-w64-mingw32-g++
        echo "Using MinGW-w64 cross-compiler: $CC"
    else
        echo -e "${YELLOW}Warning: MinGW-w64 cross-compiler not found${NC}"
        echo "Installing mingw-w64 cross-compiler..."
        
        if command -v brew &> /dev/null; then
            # macOS with Homebrew
            echo "Installing via Homebrew..."
            brew install mingw-w64
            export CC=i686-w64-mingw32-gcc
            export CXX=i686-w64-mingw32-g++
        elif command -v apt-get &> /dev/null; then
            # Ubuntu/Debian
            echo "Installing via apt-get..."
            sudo apt-get update
            sudo apt-get install -y gcc-mingw-w64-i686 g++-mingw-w64-i686
            export CC=i686-w64-mingw32-gcc
            export CXX=i686-w64-mingw32-g++
        elif command -v yum &> /dev/null; then
            # CentOS/RHEL
            echo "Installing via yum..."
            sudo yum install -y mingw32-gcc mingw32-gcc-c++
            export CC=i686-w64-mingw32-gcc
            export CXX=i686-w64-mingw32-g++
        else
            echo -e "${RED}Error: Cannot install MinGW-w64 cross-compiler automatically${NC}"
            echo "Please install mingw-w64 manually and re-run this script"
            exit 1
        fi
    fi
fi

# Build parameters
BUILD_MODE="c-shared"
OUTPUT_FILE="$OUTPUT_DIR/${DLL_NAME}.dll"
HEADER_FILE="$OUTPUT_DIR/${DLL_NAME}.h"

# Build flags
BUILD_FLAGS=(
    -buildmode="$BUILD_MODE"
    -ldflags="-s -w -X main.version=1.0.0"
    -tags="netgo,osusergo"
    -trimpath
    -o "$OUTPUT_FILE"
)

echo "Build flags: ${BUILD_FLAGS[*]}"
echo "Output file: $OUTPUT_FILE"
echo ""

# Ensure dependencies are available
echo -e "${YELLOW}Downloading dependencies...${NC}"
go mod download
go mod tidy

# Build the DLL
echo -e "${YELLOW}Compiling DLL...${NC}"
echo "Command: go build ${BUILD_FLAGS[*]} ."

if go build "${BUILD_FLAGS[@]}" .; then
    echo -e "${GREEN}✓ DLL compilation successful${NC}"
else
    echo -e "${RED}✗ DLL compilation failed${NC}"
    exit 1
fi

# Copy header file
echo -e "${YELLOW}Copying header file...${NC}"
cp "rtk_mqtt_framework.h" "$HEADER_FILE"

# Verify output files
echo ""
echo -e "${YELLOW}Verifying output files...${NC}"

if [ -f "$OUTPUT_FILE" ]; then
    DLL_SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
    echo -e "${GREEN}✓ DLL file created: $OUTPUT_FILE ($DLL_SIZE)${NC}"
else
    echo -e "${RED}✗ DLL file not found: $OUTPUT_FILE${NC}"
    exit 1
fi

if [ -f "$HEADER_FILE" ]; then
    echo -e "${GREEN}✓ Header file copied: $HEADER_FILE${NC}"
else
    echo -e "${RED}✗ Header file not found: $HEADER_FILE${NC}"
    exit 1
fi

# Check DLL dependencies (if on Windows or with appropriate tools)
if command -v objdump &> /dev/null; then
    echo ""
    echo -e "${YELLOW}DLL Dependencies:${NC}"
    objdump -p "$OUTPUT_FILE" | grep "DLL Name" | head -10 || true
fi

# Create test program
echo ""
echo -e "${YELLOW}Creating test program...${NC}"

TEST_PROGRAM="$OUTPUT_DIR/test_${DLL_NAME}.c"
cat > "$TEST_PROGRAM" << 'EOF'
#include "rtk_mqtt_framework.h"
#include <stdio.h>
#include <string.h>
#include <time.h>

int main() {
    printf("RTK MQTT Framework DLL Test\n");
    printf("Version: %s\n", rtk_get_version());
    
    // Create client
    rtk_client_handle_t client = rtk_create_client();
    if (client == 0) {
        printf("Failed to create client\n");
        return -1;
    }
    printf("Client created successfully: %p\n", (void*)client);
    
    // Configure MQTT (using public test broker)
    rtk_mqtt_config_t mqtt_config = {0};
    strcpy(mqtt_config.broker_host, "test.mosquitto.org");
    mqtt_config.broker_port = 1883;
    strcpy(mqtt_config.client_id, "rtk_test_dll_client");
    mqtt_config.keep_alive = 60;
    mqtt_config.clean_session = 1;
    mqtt_config.qos = 1;
    mqtt_config.retain = 0;
    
    int result = rtk_configure_mqtt(client, &mqtt_config);
    printf("MQTT configuration result: %d\n", result);
    
    // Configure device
    rtk_device_config_t device_config = {0};
    strcpy(device_config.device_id, "00:11:22:33:44:55");
    strcpy(device_config.device_type, "test_device");
    strcpy(device_config.tenant, "test_tenant");
    strcpy(device_config.site, "test_site");
    device_config.telemetry_interval = 60;
    device_config.state_interval = 30;
    device_config.heartbeat_interval = 10;
    
    result = rtk_configure_device(client, &device_config);
    printf("Device configuration result: %d\n", result);
    
    // Set device info
    rtk_device_info_t device_info = {0};
    strcpy(device_info.id, "00:11:22:33:44:55");
    strcpy(device_info.type, "test_device");
    strcpy(device_info.name, "Test DLL Device");
    strcpy(device_info.version, "1.0.0");
    strcpy(device_info.manufacturer, "RTK Corp");
    
    result = rtk_set_device_info(client, &device_info);
    printf("Device info result: %d\n", result);
    
    printf("Test completed - check connection manually if needed\n");
    
    // Cleanup
    rtk_destroy_client(client);
    printf("Client destroyed\n");
    
    return 0;
}
EOF

echo -e "${GREEN}✓ Test program created: $TEST_PROGRAM${NC}"

# Build summary
echo ""
echo -e "${GREEN}=== Build Summary ===${NC}"
echo "Target: Windows x86 DLL"
echo "DLL: $OUTPUT_FILE"
echo "Header: $HEADER_FILE"
echo "Test: $TEST_PROGRAM"
echo ""
echo -e "${BLUE}=== Usage Instructions ===${NC}"
echo "1. Copy the DLL and header files to your Windows project"
echo "2. Link against the DLL in your C/C++ project"
echo "3. Include the header file: #include \"rtk_mqtt_framework.h\""
echo "4. Use the API functions as documented in the header"
echo ""
echo -e "${BLUE}=== Compilation Example (on Windows) ===${NC}"
echo "gcc -o test_program test_rtk_mqtt_framework.c -L. -lrtk_mqtt_framework"
echo ""
echo -e "${GREEN}✓ DLL build completed successfully!${NC}"
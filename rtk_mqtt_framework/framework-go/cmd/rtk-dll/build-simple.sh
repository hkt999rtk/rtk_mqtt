#!/bin/bash

# Simplified DLL build script for testing (without MinGW dependency)
# This creates a basic shared library for demonstration

set -e

echo "=== RTK MQTT Framework Go DLL Builder (Simplified) ==="

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../" && pwd)"
BUILD_DIR="${PROJECT_ROOT}/cmd/rtk-dll"
OUTPUT_DIR="${PROJECT_ROOT}/dist"

echo "Project Root: $PROJECT_ROOT"
echo "Build Dir: $BUILD_DIR"
echo "Output Dir: $OUTPUT_DIR"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Change to build directory
cd "$BUILD_DIR"

# Build for host platform first (to test Go compilation)
echo ""
echo "Building Go shared library for host platform..."

export CGO_ENABLED=1

# Build as shared library for current platform
go build -buildmode=c-shared -o "$OUTPUT_DIR/rtk_mqtt_framework_$(uname -s)_$(uname -m).so" .

echo "✓ Host platform shared library built successfully"

# Copy header file
cp "rtk_mqtt_framework.h" "$OUTPUT_DIR/"
echo "✓ Header file copied"

# Create a simple test
echo ""
echo "Creating test program..."

cat > "$OUTPUT_DIR/test_dll.go" << 'EOF'
package main

/*
#cgo LDFLAGS: -L. -lrtk_mqtt_framework_Darwin_arm64
#include "rtk_mqtt_framework.h"
#include <stdio.h>
#include <string.h>
*/
import "C"
import "fmt"

func main() {
	fmt.Println("Testing RTK MQTT Framework DLL")
	
	// Test version function
	version := C.rtk_get_version()
	fmt.Printf("Version: %s\n", C.GoString(version))
	
	// Test client creation
	client := C.rtk_create_client()
	if client != 0 {
		fmt.Printf("Client created: %d\n", client)
		
		// Test client destruction
		result := C.rtk_destroy_client(client)
		fmt.Printf("Client destroyed: %d\n", result)
	} else {
		fmt.Println("Failed to create client")
	}
}
EOF

echo "✓ Test program created: $OUTPUT_DIR/test_dll.go"

# Show results
echo ""
echo "=== Build Results ==="
ls -la "$OUTPUT_DIR/"

echo ""
echo "=== Summary ==="
echo "✓ Go shared library built successfully"
echo "✓ Header file provided" 
echo "✓ Test program created"
echo ""
echo "Note: This is a host platform build for testing."
echo "For Windows DLL, install MinGW cross-compiler and run build.sh"
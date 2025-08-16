#!/bin/bash

# Windows DLL build script for RTK MQTT Framework
# This script cross-compiles the Go DLL for Windows x86/x64

set -e

echo "=== RTK MQTT Framework Windows DLL Builder ==="

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BUILD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${PROJECT_ROOT}/dist"

echo "Project Root: $PROJECT_ROOT"
echo "Build Dir: $BUILD_DIR"
echo "Output Dir: $OUTPUT_DIR"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Change to build directory
cd "$BUILD_DIR"

echo ""
echo "Checking Go version..."
go version

echo ""
echo "Checking for cross-compilation support..."

# Check if Windows cross-compilation is available
if ! GOOS=windows GOARCH=amd64 go env GOROOT >/dev/null 2>&1; then
    echo "Warning: Windows cross-compilation may not be available"
    echo "You may need to install Go with cross-compilation support"
fi

echo ""
echo "Building Windows DLL (x64)..."

# Set environment for Windows cross-compilation
export CGO_ENABLED=1
export GOOS=windows
export GOARCH=amd64

# For cross-compilation, we need a C cross-compiler
# This will work if mingw-w64 is installed
if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
    export CC=x86_64-w64-mingw32-gcc
    export CXX=x86_64-w64-mingw32-g++
    echo "Using MinGW cross-compiler: $CC"
    
    # Build Windows DLL
    echo "Building rtk_mqtt_framework.dll..."
    go build -buildmode=c-shared -o "$OUTPUT_DIR/rtk_mqtt_framework_windows_x64.dll" .
    
    echo "✓ Windows x64 DLL built successfully"
    
    # Also build 32-bit version if supported
    echo ""
    echo "Building Windows DLL (x86)..."
    export GOARCH=386
    if command -v i686-w64-mingw32-gcc >/dev/null 2>&1; then
        export CC=i686-w64-mingw32-gcc
        export CXX=i686-w64-mingw32-g++
        echo "Using MinGW 32-bit cross-compiler: $CC"
        
        go build -buildmode=c-shared -o "$OUTPUT_DIR/rtk_mqtt_framework_windows_x86.dll" .
        echo "✓ Windows x86 DLL built successfully"
    else
        echo "32-bit MinGW compiler not found, skipping x86 build"
        echo "Install with: brew install mingw-w64 (macOS) or apt install gcc-mingw-w64 (Ubuntu)"
    fi
    
else
    echo "MinGW cross-compiler not found!"
    echo ""
    echo "To install MinGW for cross-compilation:"
    echo ""
    echo "macOS:"
    echo "  brew install mingw-w64"
    echo ""
    echo "Ubuntu/Debian:"
    echo "  sudo apt update"
    echo "  sudo apt install gcc-mingw-w64"
    echo ""
    echo "Arch Linux:"
    echo "  sudo pacman -S mingw-w64-gcc"
    echo ""
    echo "Building with native Go compiler (no CGO)..."
    
    # Fallback: build without CGO (will have limitations)
    export CGO_ENABLED=0
    echo "Building Windows DLL without CGO..."
    go build -buildmode=c-shared -o "$OUTPUT_DIR/rtk_mqtt_framework_windows_x64_nocgo.dll" . || {
        echo "Error: c-shared buildmode requires CGO"
        echo "Please install MinGW for proper Windows DLL compilation"
        exit 1
    }
fi

# Copy header file
echo ""
echo "Copying header files..."
cp "rtk_mqtt_framework_simple.h" "$OUTPUT_DIR/rtk_mqtt_framework.h"
echo "✓ Header file copied"

# Create Windows test batch script
echo ""
echo "Creating Windows test scripts..."

cat > "$OUTPUT_DIR/test_dll_windows.bat" << 'EOF'
@echo off
echo === RTK MQTT Framework Windows DLL Test ===
echo.

if not exist rtk_mqtt_framework_windows_x64.dll (
    echo ERROR: DLL not found!
    echo Please build the DLL first using build-windows.sh
    pause
    exit /b 1
)

echo Testing DLL with C++ demo...
echo.

rem Note: You need to compile the C++ demo first
rem Use Visual Studio or MinGW to compile main.cpp with rtk_mqtt_framework.h

if exist cpp_dll_demo.exe (
    echo Running C++ demo...
    cpp_dll_demo.exe
) else (
    echo cpp_dll_demo.exe not found
    echo Please compile the C++ demo using:
    echo   g++ -std=c++11 -o cpp_dll_demo.exe main.cpp
    echo or use Visual Studio to build the project
)

echo.
echo Test completed.
pause
EOF

# Create PowerShell test script
cat > "$OUTPUT_DIR/test_dll.ps1" << 'EOF'
# RTK MQTT Framework Windows DLL Test Script
Write-Host "=== RTK MQTT Framework Windows DLL Test ===" -ForegroundColor Green

$dllPath = ".\rtk_mqtt_framework_windows_x64.dll"

if (-not (Test-Path $dllPath)) {
    Write-Host "ERROR: DLL not found at $dllPath" -ForegroundColor Red
    Write-Host "Please build the DLL first using build-windows.sh"
    exit 1
}

Write-Host "DLL found: $dllPath" -ForegroundColor Green

# Get DLL information
$dllInfo = Get-Item $dllPath
Write-Host "DLL Size: $($dllInfo.Length) bytes"
Write-Host "DLL Modified: $($dllInfo.LastWriteTime)"

# Test if we can load the DLL in PowerShell
try {
    Add-Type -TypeDefinition @"
    using System;
    using System.Runtime.InteropServices;
    
    public class RTKMQTTFramework
    {
        [DllImport("$dllPath", CharSet = CharSet.Ansi, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr rtk_get_version();
        
        [DllImport("$dllPath", CharSet = CharSet.Ansi, CallingConvention = CallingConvention.Cdecl)]
        public static extern UIntPtr rtk_create_client();
        
        [DllImport("$dllPath", CharSet = CharSet.Ansi, CallingConvention = CallingConvention.Cdecl)]
        public static extern int rtk_destroy_client(UIntPtr clientId);
        
        [DllImport("$dllPath", CharSet = CharSet.Ansi, CallingConvention = CallingConvention.Cdecl)]
        public static extern int rtk_get_client_count();
    }
"@

    Write-Host "DLL loaded successfully in PowerShell!" -ForegroundColor Green
    
    # Test basic functions
    $version = [System.Runtime.InteropServices.Marshal]::PtrToStringAnsi([RTKMQTTFramework]::rtk_get_version())
    Write-Host "RTK MQTT Framework Version: $version" -ForegroundColor Cyan
    
    $clientCount = [RTKMQTTFramework]::rtk_get_client_count()
    Write-Host "Current client count: $clientCount" -ForegroundColor Cyan
    
    # Test client creation/destruction
    Write-Host "Testing client creation..." -ForegroundColor Yellow
    $client = [RTKMQTTFramework]::rtk_create_client()
    if ($client -ne 0) {
        Write-Host "Client created successfully: $client" -ForegroundColor Green
        
        $newCount = [RTKMQTTFramework]::rtk_get_client_count()
        Write-Host "New client count: $newCount" -ForegroundColor Cyan
        
        $result = [RTKMQTTFramework]::rtk_destroy_client($client)
        if ($result -eq 0) {
            Write-Host "Client destroyed successfully" -ForegroundColor Green
        } else {
            Write-Host "Failed to destroy client: $result" -ForegroundColor Red
        }
    } else {
        Write-Host "Failed to create client" -ForegroundColor Red
    }
    
    Write-Host "DLL test completed successfully!" -ForegroundColor Green
    
} catch {
    Write-Host "Error testing DLL: $($_.Exception.Message)" -ForegroundColor Red
}
EOF

echo "✓ Windows test scripts created"

# Show results
echo ""
echo "=== Build Results ==="
ls -la "$OUTPUT_DIR/"

echo ""
echo "=== Summary ==="
echo "✓ Windows DLL compilation attempted"
echo "✓ Header file provided"
echo "✓ Windows test scripts created"
echo ""
echo "Output files:"
find "$OUTPUT_DIR" -name "rtk_mqtt_framework_windows*" -o -name "*.h" -o -name "*.bat" -o -name "*.ps1" | sort

echo ""
echo "Next steps:"
echo "1. Transfer files to Windows machine"
echo "2. Use Visual Studio or MinGW to compile C++ demo"
echo "3. Run test_dll_windows.bat or test_dll.ps1"
echo "4. Integrate with your Windows applications"
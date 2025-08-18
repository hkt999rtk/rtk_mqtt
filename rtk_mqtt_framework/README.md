# RTK MQTT Framework

A comprehensive cross-platform MQTT diagnostic communication framework for IoT devices, network equipment, and enterprise systems. Supporting POSIX (Linux/macOS), Windows, and ARM FreeRTOS environments with zero external dependencies.

**This README is for framework developers.** For end users who want to integrate the framework into their projects, please see [MANUAL.md](docs/MANUAL.md).

## Framework Architecture

```
rtk_mqtt_framework/
├── framework/              # C/C++ Implementation
│   ├── include/           # Header files  
│   ├── src/              # Core implementation
│   ├── platforms/        # Platform-specific code
│   └── examples/         # Framework demos
├── framework-go/          # Go Implementation  
│   ├── pkg/              # Go packages
│   ├── examples/         # Go examples
│   └── tests/            # Go test suites
├── tools/                # Development tools
│   ├── rtk_cli/          # Command-line interface
│   └── mock_broker/      # Test MQTT broker
├── external/             # Bundled dependencies
│   ├── paho-mqtt-c/      # Eclipse Paho MQTT C
│   ├── cjson/           # JSON processing
│   └── pubsubclient/    # Arduino-compatible MQTT
├── examples/             # User integration examples
├── docs/                 # Technical documentation
└── schemas/              # JSON schema definitions
```

## Key Design Decisions

### Zero External Dependencies
All MQTT and JSON libraries are bundled in the `external/` directory:
- **Eclipse Paho MQTT C** v1.3.13 - Production MQTT client
- **cJSON** v1.7.16 - JSON processing
- **PubSubClient** - Unified MQTT backend for all platforms

### Multi-Platform Strategy
- **POSIX** (Linux/macOS): Production-ready with full feature set
- **Windows**: Enterprise support with Windows Service integration  
- **FreeRTOS**: Embedded systems with lwIP/FreeRTOS+TCP support
- **Cross-compilation**: ARM Cortex-M support with GCC toolchain

### Dual Implementation
- **C/C++ Framework**: High-performance embedded and desktop systems
- **Go Framework**: Cloud-native and modern development environments
- **Unified Protocol**: Both implementations share the same RTK MQTT specification

## Development Setup

### Prerequisites

#### POSIX (Linux/macOS)
```bash
# Ubuntu/Debian - Only build tools needed
sudo apt-get install cmake build-essential pkg-config

# macOS (Homebrew) - Only build tools needed
brew install cmake

# CentOS/RHEL - Only build tools needed  
sudo yum install cmake gcc-c++ pkgconfig
```

#### Windows
```powershell
# Using Visual Studio (recommended)
# Install Visual Studio 2019+ with C++ workload

# Or using MinGW-w64
# Download from: https://www.mingw-w64.org/downloads/
```

#### ARM Cross-Compilation (FreeRTOS Embedded Systems)
```bash
# Install ARM GCC toolchain (macOS)
brew install arm-none-eabi-gcc arm-none-eabi-binutils

# Ubuntu/Debian Linux
sudo apt-get install gcc-arm-none-eabi binutils-arm-none-eabi

# Windows
# Download GNU Arm Embedded Toolchain from:
# https://developer.arm.com/tools-and-software/open-source-software/developer-tools/gnu-toolchain/gnu-rm

# Verify installation
arm-none-eabi-gcc --version
```

**Supported ARM Processors:**

| CPU        | FPU Support | Features      | Memory Requirements |
|------------|-------------|---------------|-------------------|
| cortex-m0  | No          | Basic version | Flash: 32KB+, RAM: 16KB+ |
| cortex-m3  | No          | Enhanced perf | Flash: 48KB+, RAM: 24KB+ |
| cortex-m4  | Hardware    | DSP + FPU     | Flash: 64KB+, RAM: 32KB+ |
| cortex-m7  | Hardware    | High perf     | Flash: 128KB+, RAM: 64KB+ |
| cortex-m33 | Hardware    | ARMv8-M       | Flash: 96KB+, RAM: 48KB+ |

**FreeRTOS Configuration Requirements:**
```c
// In FreeRTOSConfig.h
#define configUSE_MUTEXES                1
#define configUSE_RECURSIVE_MUTEXES      1  
#define configUSE_COUNTING_SEMAPHORES    1
#define configSUPPORT_DYNAMIC_ALLOCATION 1
#define configTOTAL_HEAP_SIZE           (64 * 1024)  // Adjust as needed
```

#### Go Environment (for Go implementation)
```bash
# Go 1.21+ required for Go-based features
go version  # Verify Go installation
```

## Build System

### Quick Development Build
```bash
git clone <repository-url>
cd rtk_mqtt_framework

# Using new Makefile (recommended)
make dev                    # Quick host build for development
make test                   # Run all tests (C++ + Go)

# Using CMake directly
mkdir build && cd build
cmake .. -DBUILD_EXAMPLES=ON -DBUILD_TOOLS=ON
make -j$(nproc)
```

### Platform-Specific Builds
```bash
# Platform targets (using Makefile)
make linux-amd64           # Linux x86_64 build
make linux-arm64           # Linux ARM64 build  
make darwin-arm64          # macOS ARM64 build
make windows-amd64         # Windows x86_64 build

# ARM embedded builds
make arm-cortex-m4         # ARM Cortex-M4 (FreeRTOS)
make arm-cortex-m3         # ARM Cortex-M3 (FreeRTOS)

# Go framework builds
make build-go              # Build Go framework examples
make build-go-cross        # Cross-compile Go for multiple platforms
```

### Advanced CMake Options
```bash
# POSIX Production Build
cmake .. \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_EXAMPLES=ON \
    -DBUILD_TOOLS=ON \
    -DBUILD_SHARED_LIBS=ON \
    -DCMAKE_INSTALL_PREFIX=/usr/local

# Windows Build
cmake .. \
    -DRTK_TARGET_WINDOWS=ON \
    -DRTK_ENABLE_WINDOWS_SERVICE=ON \
    -DBUILD_EXAMPLES=ON

# FreeRTOS ARM Build (Method 1: CMake with toolchain)
mkdir build-arm && cd build-arm
cmake .. \
    -DCMAKE_TOOLCHAIN_FILE=../cmake/arm-none-eabi-toolchain.cmake \
    -DCMAKE_BUILD_TYPE=Release \
    -DRTK_TARGET_FREERTOS=ON \
    -DARM_CPU=cortex-m4 \
    -DRTK_USE_LIGHTWEIGHT_JSON=ON \
    -DBUILD_EXAMPLES=OFF \
    -DBUILD_TOOLS=OFF \
    -DBUILD_SHARED_LIBS=OFF
make -j$(nproc)

# FreeRTOS ARM Build (Method 2: Using Makefile)
make build-arm              # Uses CMake with ARM toolchain internally
```

## Testing

### Running Tests
```bash
# Run all tests (recommended)
make test

# C/C++ tests only
make test-host

# Go tests only  
make test-go
cd framework-go && go test ./... -v -race -cover

# Comprehensive Go test suite
cd framework-go && ./test.sh
```

### Test Structure
- **C/C++ Tests**: Build verification and plugin loading tests
- **Go Tests**: Unit tests for MQTT client, topic builder, message codec, device management
- **Integration Tests**: End-to-end testing with mock broker
- **Examples**: Serve as integration tests for device plugins

## Framework Development

### Code Quality Standards
```bash
# C/C++ quality checks
make build-host             # Must build without warnings
# Static analysis via CMake compiler flags

# Go quality checks  
cd framework-go
make fmt                    # Format code
make vet                    # Run go vet
make lint                   # Run golangci-lint (if available)
make test                   # Tests must pass with race detection
```

### Adding New Platform Support
1. Create platform detection in main `CMakeLists.txt`
2. Add compatibility layer in `framework/src/{platform}_compat.c`
3. Implement network interface in `framework/src/network_{platform}.c`
4. Update `rtk_platform_compat.h` with platform macros
5. Add Makefile target following broker pattern
6. Test with platform-specific examples

### Creating Framework Examples
1. Copy example template from `examples/wifi_router/`
2. Implement `rtk_device_plugin_vtable_t` interface
3. Create device-specific JSON config file
4. Update `CMakeLists.txt` to build plugin
5. Test with `plugin_demo` tool

### Memory Management
Always use platform-abstracted memory functions (`RTK_MALLOC`, `RTK_FREE`) rather than direct malloc/free to ensure compatibility across platforms.

## Release Building

### Creating Release Packages
```bash
# Build complete release packages for all platforms
make release

# Individual platform releases
make linux-amd64 && make package-linux-amd64
make windows-amd64 && make package-windows-amd64
make arm-cortex-m4 && make package-arm-cortex-m4
```

### Release Package Contents
Each release package includes:
- Pre-compiled binaries and libraries
- Header files for integration
- User integration examples with standalone Makefiles
- Testing tools for functionality verification
- Complete user manual (MANUAL.md)

## Architecture Details

### MQTT Backend Abstraction
- Unified PubSubClient backend for all platforms
- Platform-specific network implementations via `rtk_network_interface.h`
- C++ wrapper for PubSubClient with C interface (`pubsub_adapter.cpp`)

### Plugin System
- Dynamic loading of device-specific implementations
- Standardized `rtk_device_plugin_vtable_t` interface
- Plugin lifecycle management (initialize, get_state, cleanup)

### Message Processing
- `topic_builder.c` - RTK topic format: `rtk/v1/{tenant}/{site}/{device_id}/{message_type}`
- `message_codec.c` - JSON encoding/decoding with schema validation
- `schema_validator.c` - JSON schema validation using bundled cJSON

### Cross-Platform Compatibility
```c
#ifdef RTK_PLATFORM_FREERTOS
    // FreeRTOS-specific implementation
#elif defined(RTK_PLATFORM_WINDOWS)  
    // Windows-specific implementation
#else
    // POSIX default implementation
#endif
```

## FreeRTOS Integration Guide

### Network Stack Support

The framework provides unified network abstraction supporting multiple FreeRTOS network stacks:

#### lwIP (Default - Recommended)
```c
// Default configuration - uses lwIP sockets API
#include "lwip/sockets.h"
#include "lwip/netdb.h"

// API mapping for portability
#define socket_create(f,t,p) lwip_socket(f,t,p)
#define socket_connect(s,a,l) lwip_connect(s,a,l)
#define socket_send(s,d,l,f) lwip_send(s,d,l,f)
#define socket_recv(s,b,l,f) lwip_recv(s,b,l,f)
#define socket_close(s) lwip_close(s)
```

#### FreeRTOS+TCP (Optional)
```c
// Enable in CMake: -DRTK_USE_FREERTOS_TCP=ON
#define RTK_USE_FREERTOS_TCP
#include "FreeRTOS_Sockets.h"

// API mapping for FreeRTOS+TCP
#define socket_create(f,t,p) FreeRTOS_socket(f,t,p)
#define socket_connect(s,a,l) FreeRTOS_connect(s,a,l) 
#define socket_send(s,d,l,f) FreeRTOS_send(s,d,l,f)
#define socket_recv(s,b,l,f) FreeRTOS_recv(s,b,l,f)
#define socket_close(s) FreeRTOS_closesocket(s)
```

### Memory Optimization for Embedded Systems

The framework provides several memory optimization options:

```cmake
# Enable in CMake configuration
-DRTK_USE_LIGHTWEIGHT_JSON=ON      # Use lightweight JSON parser (~50% less memory)
-DRTK_MINIMAL_MEMORY=ON            # Enable memory minimization mode
-DRTK_NO_DYNAMIC_ALLOCATION=ON     # Disable dynamic memory allocation
```

**Typical Memory Usage:**

| Component              | RAM (KB) | Flash (KB) | Notes |
|------------------------|----------|------------|-------|
| Core framework         | 2-4      | 8-12       | Basic MQTT functionality |
| MQTT client            | 4-8      | 15-20      | Including message buffers |
| JSON processing        | 1-2      | 5-8        | Lightweight mode |
| Device plugin (example)| 1-3      | 3-5        | Application-specific |
| **Total**             | **8-17** | **31-45**  | Complete implementation |

### FreeRTOS Integration Example

```c
#include "rtk_mqtt_client.h"
#include "rtk_device_plugin.h"
#include "FreeRTOS.h"
#include "task.h"

// MQTT task implementation
void mqtt_task(void *pvParameters) {
    rtk_mqtt_client_t *client;
    
    // Initialize MQTT client with platform-specific network interface
    rtk_mqtt_client_config_t config = {
        .broker_host = "iot.example.com",
        .broker_port = 1883,
        .client_id = "device_001",
        .username = "user", 
        .password = "pass"
    };
    
    client = rtk_mqtt_client_create(&config);
    if (!client) {
        printf("Failed to create MQTT client\n");
        vTaskDelete(NULL);
        return;
    }
    
    // Connect to MQTT broker
    if (rtk_mqtt_client_connect(client) != RTK_MQTT_SUCCESS) {
        printf("Failed to connect to MQTT broker\n");
        rtk_mqtt_client_destroy(client);
        vTaskDelete(NULL);
        return;
    }
    
    printf("Connected to MQTT broker successfully\n");
    
    // Main processing loop
    while (1) {
        // Process MQTT messages
        rtk_mqtt_client_loop(client);
        
        // Yield to other tasks
        vTaskDelay(pdMS_TO_TICKS(100));
    }
}

// Application entry point
int main(void) {
    // Initialize hardware (platform-specific)
    // ...
    
    // Create MQTT task
    xTaskCreate(mqtt_task, "MQTT", 4096, NULL, 2, NULL);
    
    // Start FreeRTOS scheduler
    vTaskStartScheduler();
    
    return 0;
}
```

### Network Interface Abstraction

The framework uses a unified network interface that abstracts platform differences:

```c
typedef struct rtk_network_interface {
    void* context;                                    // Platform-specific context
    int (*connect)(rtk_network_interface_t*, const char*, uint16_t);
    int (*disconnect)(rtk_network_interface_t*);
    int (*send)(rtk_network_interface_t*, const void*, size_t, size_t*);
    int (*receive)(rtk_network_interface_t*, void*, size_t, size_t*);
    int (*is_connected)(rtk_network_interface_t*);
    void (*cleanup)(rtk_network_interface_t*);
} rtk_network_interface_t;

// Platform-specific creation
#ifdef RTK_PLATFORM_FREERTOS
    rtk_network_create_freertos(&client->network_interface);
#elif defined(RTK_PLATFORM_WINDOWS)
    rtk_network_create_windows(&client->network_interface);
#else
    rtk_network_create_posix(&client->network_interface);
#endif
```

### Integrating into FreeRTOS Projects

1. **Copy Library Files:**
```
your_project/
├── lib/
│   └── librtk_mqtt_framework_arm.a
├── include/
│   └── rtk/
│       ├── rtk_mqtt_client.h
│       ├── rtk_device_plugin.h
│       └── rtk_network_interface.h
└── src/
    └── your_application.c
```

2. **Update Makefile:**
```makefile
# Include paths
CFLAGS += -I./include/rtk

# Link RTK library
LDFLAGS += -L./lib -lrtk_mqtt_framework_arm

# FreeRTOS platform definition
CFLAGS += -DRTK_PLATFORM_FREERTOS=1
```

3. **CMakeLists.txt Integration:**
```cmake
# Add include directory
target_include_directories(your_target PRIVATE include/rtk)

# Link RTK library
target_link_libraries(your_target 
    PRIVATE 
    ${CMAKE_CURRENT_SOURCE_DIR}/lib/librtk_mqtt_framework_arm.a
)

# Add platform definition
target_compile_definitions(your_target PRIVATE RTK_PLATFORM_FREERTOS=1)
```

## Contributing

### Development Workflow
1. Fork the repository
2. Create a feature branch
3. Make changes following code quality standards
4. Run all tests (`make test`)
5. Test on multiple platforms if possible
6. Submit pull request with clear description

### Code Style
- **C/C++**: Follow existing style, use platform compatibility macros
- **Go**: Standard Go formatting (`gofmt`, `go vet`)
- **Documentation**: Update relevant README sections and CLAUDE.md

### Release Notes
For version-specific changes and release information, see [RELEASE.md](RELEASE.md).

## Documentation

- **[docs/MANUAL.md](docs/MANUAL.md)** - User integration guide (Traditional Chinese)
- **[CLAUDE.md](CLAUDE.md)** - Claude Code assistant guidance
- **[RELEASE.md](RELEASE.md)** - Release notes and version information
- **[docs/](docs/)** - Technical documentation and specifications

## License

MIT License - see [LICENSE](LICENSE) file for details.
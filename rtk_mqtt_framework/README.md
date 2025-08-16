# RTK MQTT Framework

A unified multi-platform MQTT diagnostic framework supporting POSIX (Linux/macOS), Windows, and FreeRTOS environments. Features both C/C++ and Go implementations for maximum flexibility.

## 🏗️ Architecture Overview

```
rtk_mqtt_framework/
├── framework/              # C/C++ Implementation
│   ├── include/           # Header files  
│   ├── src/              # Core implementation
│   ├── platforms/        # Platform-specific code
│   └── examples/         # Platform examples
├── framework-go/          # Go Implementation  
│   ├── pkg/              # Go packages
│   ├── examples/         # Go examples
│   └── tests/            # Go test suites
├── tools/                # Development tools
│   ├── rtk_cli/          # Command-line interface
│   └── mock_broker/      # Test MQTT broker
├── docs/                 # Technical documentation
├── schemas/              # JSON schema definitions
└── cmake/                # Build system files
```

## 🎯 Key Features

### Multi-Platform Support
- **POSIX** (Linux/macOS): Production-ready with full feature set
- **Windows**: Enterprise support with Windows Service integration  
- **FreeRTOS**: Embedded systems with lwIP/FreeRTOS+TCP support
- **Cross-compilation**: ARM Cortex-M support with GCC toolchain

### Dual Implementation
- **C/C++ Framework**: High-performance embedded and desktop systems
- **Go Framework**: Cloud-native and modern development environments
- **Unified Protocol**: Both implementations share the same RTK MQTT specification

### Network Abstraction
- **Pluggable Backends**: Eclipse Paho MQTT C, PubSubClient, Paho Go
- **Network Layers**: POSIX sockets, Windows Winsock, FreeRTOS+TCP, lwIP
- **TLS/SSL Support**: Secure communication with certificate validation

## 🚀 Quick Start

### Prerequisites

> **🎉 Zero External Dependencies!** 
> 
> The RTK MQTT Framework now includes all necessary dependencies (Eclipse Paho MQTT C and cJSON) directly in the source tree. No external library installation required!

<details>
<summary><strong>📦 Required Tools by Platform</strong></summary>

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

#### ARM Cross-Compilation
```bash
# Install ARM GCC toolchain (macOS)
brew install arm-none-eabi-gcc arm-none-eabi-binutils

# Linux
sudo apt-get install gcc-arm-none-eabi binutils-arm-none-eabi
```

#### Go Environment (for Go implementation)
```bash
# Go 1.19+ required for Go-based features
go version  # Verify Go installation
```

</details>

### 🔋 Integrated Dependencies

The framework includes these dependencies locally (no installation needed):

- **Eclipse Paho MQTT C** v1.3.13 - Integrated in `framework/third_party/paho_mqtt_c/`
- **cJSON** v1.7.16 - Integrated in `framework/third_party/cjson/`  
- **Platform-specific networking** - Built-in support for POSIX sockets, Windows Winsock, FreeRTOS+TCP

These are automatically compiled as part of the build process.

### Basic Build

```bash
git clone <repository-url>
cd rtk_mqtt_framework

# Build C/C++ framework
mkdir build && cd build
cmake .. -DBUILD_EXAMPLES=ON -DBUILD_TOOLS=ON
make -j$(nproc)

# Build Go framework
cd ../framework-go
go build ./...
go test ./...
```

## 🔧 Platform-Specific Builds

### 1. POSIX (Linux/macOS) Build

<details>
<summary><strong>🐧 Linux Production Build</strong></summary>

```bash
mkdir build-linux && cd build-linux

# Production build with optimizations
cmake .. \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_EXAMPLES=ON \
    -DBUILD_TOOLS=ON \
    -DBUILD_SHARED_LIBS=ON \
    -DCMAKE_INSTALL_PREFIX=/usr/local

make -j$(nproc)
sudo make install

# Verify installation
pkg-config --cflags --libs rtk-mqtt-framework
```

#### System Service Integration
```bash
# Install as systemd service
sudo cp examples/systemd/rtk-mqtt-framework.service /etc/systemd/system/
sudo systemctl enable rtk-mqtt-framework
sudo systemctl start rtk-mqtt-framework
```

</details>

<details>
<summary><strong>🍎 macOS Development Build</strong></summary>

```bash
mkdir build-macos && cd build-macos

# Development build with debugging
cmake .. \
    -DCMAKE_BUILD_TYPE=Debug \
    -DBUILD_EXAMPLES=ON \
    -DBUILD_TOOLS=ON \
    -DENABLE_TESTING=ON

make -j$(sysctl -n hw.ncpu)

# Run tests
ctest --output-on-failure
```

</details>

### 2. Windows Build

<details>
<summary><strong>🪟 Windows Service Build</strong></summary>

```powershell
mkdir build-windows
cd build-windows

# Configure with Visual Studio
cmake .. `
    -G "Visual Studio 17 2022" `
    -A x64 `
    -DRTK_TARGET_WINDOWS=ON `
    -DRTK_ENABLE_WINDOWS_SERVICE=ON `
    -DBUILD_EXAMPLES=ON

# Build
cmake --build . --config Release --parallel

# Install Windows Service
.\examples\windows_service\install_service.bat
```

#### Service Management
```powershell
# Start/Stop RTK MQTT Service
sc start RTKMQTTFramework
sc stop RTKMQTTFramework

# View service logs
Get-EventLog -LogName Application -Source "RTK MQTT Framework" -Newest 10
```

</details>

### 3. FreeRTOS Embedded Build

<details>
<summary><strong>🔌 ARM Cortex-M Build (FreeRTOS)</strong></summary>

#### Prerequisites
```bash
# Install ARM toolchain (already installed in previous steps)
arm-none-eabi-gcc --version

# Download FreeRTOS
wget https://github.com/FreeRTOS/FreeRTOS/releases/download/202212.00/FreeRTOS-202212.00.zip
unzip FreeRTOS-202212.00.zip
```

#### Build for Cortex-M4
```bash
mkdir build-arm && cd build-arm

# Configure for ARM Cortex-M4 with FreeRTOS
cmake .. \
    -DCMAKE_TOOLCHAIN_FILE=../cmake/arm-none-eabi-toolchain.cmake \
    -DRTK_TARGET_FREERTOS=ON \
    -DARM_CPU=cortex-m4 \
    -DFREERTOS_PATH=/path/to/FreeRTOS \
    -DRTK_USE_LWIP=ON \
    -DRTK_USE_LIGHTWEIGHT_JSON=ON \
    -DCMAKE_BUILD_TYPE=Release

make -j$(nproc)

# Output files
ls -la *.bin *.hex *.list
```

#### Supported ARM Targets
| CPU | FPU | Flags |
|-----|-----|-------|
| Cortex-M0 | None | `-mcpu=cortex-m0 -mthumb -mfloat-abi=soft` |
| Cortex-M3 | None | `-mcpu=cortex-m3 -mthumb -mfloat-abi=soft` |
| Cortex-M4 | FPv4-SP | `-mcpu=cortex-m4 -mthumb -mfloat-abi=hard -mfpu=fpv4-sp-d16` |
| Cortex-M7 | FPv5 | `-mcpu=cortex-m7 -mthumb -mfloat-abi=hard -mfpu=fpv5-d16` |
| Cortex-M33 | FPv5-SP | `-mcpu=cortex-m33 -mthumb -mfloat-abi=hard -mfpu=fpv5-sp-d16` |

</details>

<details>
<summary><strong>📡 ESP32 Build (ESP-IDF)</strong></summary>

```bash
# Setup ESP-IDF environment
. $HOME/esp/esp-idf/export.sh

# Configure for ESP32
cd examples/freertos_device
idf.py set-target esp32
idf.py menuconfig

# Build and flash
idf.py build
idf.py flash monitor
```

#### ESP32 Configuration
```c
// In sdkconfig
CONFIG_LWIP_MAX_SOCKETS=16
CONFIG_MBEDTLS_TLS_ENABLED=y
CONFIG_MQTT_BUFFER_SIZE=4096
CONFIG_FREERTOS_HZ=1000
```

</details>

### 4. Go Framework Build

<details>
<summary><strong>🐹 Go Cross-Platform Build</strong></summary>

```bash
cd framework-go

# Build for current platform
go build ./...

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o rtk-mqtt-linux ./examples/framework_demo
GOOS=windows GOARCH=amd64 go build -o rtk-mqtt-windows.exe ./examples/framework_demo
GOOS=darwin GOARCH=arm64 go build -o rtk-mqtt-macos ./examples/framework_demo

# Build with optimizations
go build -ldflags="-s -w" -o rtk-mqtt-optimized ./examples/framework_demo
```

#### Docker Build
```bash
# Build Docker image
docker build -t rtk-mqtt-framework:latest .

# Run in container
docker run -d --name rtk-mqtt \
    -e MQTT_BROKER_HOST=mqtt.broker.com \
    -e DEVICE_ID=00:11:22:33:44:55 \
    rtk-mqtt-framework:latest
```

</details>

## 🪟 Windows DLL Integration

### Go-based Windows DLL

The RTK MQTT Framework provides a Windows DLL built from Go for easy integration with C/C++ applications.

<details>
<summary><strong>🔧 Building the Windows DLL</strong></summary>

#### Prerequisites
```bash
# Go 1.19+ required
go version

# For cross-compilation from non-Windows systems:
# macOS
brew install mingw-w64

# Ubuntu/Debian  
sudo apt-get install gcc-mingw-w64

# Arch Linux
sudo pacman -S mingw-w64-gcc
```

#### Build DLL
```bash
cd framework-go/cmd/rtk-dll-simple

# For host platform (Linux/macOS)
CGO_ENABLED=1 go build -buildmode=c-shared -o rtk_mqtt_framework.so .

# For Windows (with MinGW cross-compiler)
./build-windows.sh
```

#### Output Files
```
dist/
├── rtk_mqtt_framework_windows_x64.dll    # 64-bit Windows DLL
├── rtk_mqtt_framework_windows_x86.dll    # 32-bit Windows DLL  
├── rtk_mqtt_framework.h                  # C/C++ header file
├── test_dll_windows.bat                  # Windows test script
└── test_dll.ps1                         # PowerShell test script
```

</details>

<details>
<summary><strong>💻 C++ Integration Example</strong></summary>

#### Dynamic Loading in C++
```cpp
#include "rtk_mqtt_framework.h"
#include <iostream>
#include <windows.h>

class RTKMQTTClient {
private:
    HMODULE dll_handle;
    rtk_client_handle_t client_handle;
    
    // Function pointers
    rtk_create_client_func create_client;
    rtk_configure_mqtt_func configure_mqtt;
    rtk_connect_func connect;
    // ... other function pointers
    
public:
    bool Initialize(const std::string& dll_path) {
        dll_handle = LoadLibraryA(dll_path.c_str());
        if (!dll_handle) return false;
        
        // Load function pointers
        create_client = (rtk_create_client_func)GetProcAddress(dll_handle, "rtk_create_client");
        configure_mqtt = (rtk_configure_mqtt_func)GetProcAddress(dll_handle, "rtk_configure_mqtt");
        connect = (rtk_connect_func)GetProcAddress(dll_handle, "rtk_connect");
        
        if (!create_client || !configure_mqtt || !connect) {
            FreeLibrary(dll_handle);
            return false;
        }
        
        client_handle = create_client();
        return client_handle != 0;
    }
    
    bool ConfigureMQTT(const std::string& host, int port, const std::string& client_id) {
        rtk_mqtt_config_t config = {0};
        strncpy(config.broker_host, host.c_str(), sizeof(config.broker_host) - 1);
        config.broker_port = port;
        strncpy(config.client_id, client_id.c_str(), sizeof(config.client_id) - 1);
        
        return configure_mqtt(client_handle, &config) == RTK_SUCCESS;
    }
    
    bool Connect() {
        return connect(client_handle) == RTK_SUCCESS;
    }
    
    // ... other methods
};

// Usage
int main() {
    RTKMQTTClient client;
    
    if (!client.Initialize("rtk_mqtt_framework_windows_x64.dll")) {
        std::cerr << "Failed to initialize RTK MQTT client" << std::endl;
        return -1;
    }
    
    if (!client.ConfigureMQTT("test.mosquitto.org", 1883, "my_client")) {
        std::cerr << "Failed to configure MQTT" << std::endl;
        return -1;
    }
    
    if (!client.Connect()) {
        std::cerr << "Failed to connect to MQTT broker" << std::endl;
        return -1;
    }
    
    std::cout << "Connected successfully!" << std::endl;
    return 0;
}
```

#### Visual Studio Project Setup
```xml
<!-- In your .vcxproj file -->
<PropertyGroup>
  <IncludePath>path\to\rtk_mqtt_framework;$(IncludePath)</IncludePath>
</PropertyGroup>

<ItemDefinitionGroup>
  <ClCompile>
    <AdditionalIncludeDirectories>path\to\rtk_mqtt_framework</AdditionalIncludeDirectories>
  </ClCompile>
</ItemDefinitionGroup>
```

</details>

<details>
<summary><strong>🔌 C Plugin Development</strong></summary>

#### Plugin Structure
```c
// smart_device_plugin.c
#include "rtk_mqtt_framework.h"
#include <stdio.h>
#include <stdlib.h>
#include <time.h>

typedef struct {
    double temperature;
    double humidity;
    int status_led;
    char mode[32];
} smart_device_state_t;

static rtk_client_handle_t g_client = 0;
static smart_device_state_t g_device = {0};

int plugin_init(const char* device_id, const char* broker_host, int broker_port) {
    // Create and configure client
    g_client = rtk_create_client();
    if (g_client == 0) return -1;
    
    // Configure MQTT
    rtk_mqtt_config_t config = {0};
    strncpy(config.broker_host, broker_host, sizeof(config.broker_host) - 1);
    config.broker_port = broker_port;
    snprintf(config.client_id, sizeof(config.client_id), "smart_device_%s", device_id);
    
    if (rtk_configure_mqtt(g_client, &config) != RTK_SUCCESS) return -1;
    
    // Set device info
    rtk_device_info_t info = {0};
    strncpy(info.id, device_id, sizeof(info.id) - 1);
    strcpy(info.device_type, "smart_sensor");
    strcpy(info.name, "Smart Environmental Sensor");
    strcpy(info.version, "1.0.0");
    
    if (rtk_set_device_info(g_client, &info) != RTK_SUCCESS) return -1;
    
    // Connect
    return rtk_connect(g_client);
}

void plugin_update() {
    // Simulate sensor readings
    g_device.temperature = 20.0 + ((double)rand() / RAND_MAX) * 10.0;
    g_device.humidity = 40.0 + ((double)rand() / RAND_MAX) * 20.0;
    g_device.status_led = (g_device.temperature > 25.0) ? 1 : 0;
    strcpy(g_device.mode, g_device.status_led ? "cooling" : "normal");
    
    // Publish state
    rtk_device_state_t state = {0};
    strcpy(state.status, "online");
    strcpy(state.health, "healthy");
    state.uptime = time(NULL);
    state.last_seen = time(NULL);
    
    rtk_publish_state(g_client, &state);
    
    printf("Sensor update: %.1f°C, %.1f%% RH, Mode: %s\n",
           g_device.temperature, g_device.humidity, g_device.mode);
}

void plugin_cleanup() {
    if (g_client) {
        rtk_disconnect(g_client);
        rtk_destroy_client(g_client);
    }
}

// Main loop
int main(int argc, char* argv[]) {
    const char* device_id = (argc > 1) ? argv[1] : "smart_sensor_001";
    const char* broker_host = (argc > 2) ? argv[2] : "test.mosquitto.org";
    int broker_port = (argc > 3) ? atoi(argv[3]) : 1883;
    
    if (plugin_init(device_id, broker_host, broker_port) != RTK_SUCCESS) {
        printf("Failed to initialize plugin\n");
        return -1;
    }
    
    printf("Smart device plugin started (Press Ctrl+C to stop)\n");
    
    for (int i = 0; i < 100; i++) {  // Run for ~5 minutes
        plugin_update();
        Sleep(3000);  // 3 second interval
    }
    
    plugin_cleanup();
    return 0;
}
```

#### Compilation
```bash
# With Visual Studio
cl /I path\to\headers smart_device_plugin.c

# With MinGW
gcc -std=c99 -o smart_device_plugin.exe smart_device_plugin.c -L. -lrtk_mqtt_framework_windows_x64

# Make sure the DLL is in the same directory as the executable
```

</details>

## 🔌 Plugin Development Guide

### Plugin Architecture

RTK Framework uses a standardized plugin interface that works across all platforms:

```c
typedef struct rtk_device_plugin_vtable {
    int (*get_device_info)(rtk_device_info_t* info);
    int (*initialize)(const rtk_plugin_config_t* config);
    int (*start)(void);
    int (*stop)(void);
    int (*get_state)(rtk_device_state_t* state);
    int (*get_telemetry)(const char* metric, rtk_telemetry_data_t* data);
    int (*handle_command)(const rtk_command_t* command, rtk_command_response_t* response);
    int (*cleanup)(void);
} rtk_device_plugin_vtable_t;
```

### Creating a New Plugin

<details>
<summary><strong>📝 Step-by-Step Plugin Creation</strong></summary>

#### 1. Create Plugin Structure
```bash
mkdir examples/my_device
cd examples/my_device
```

#### 2. Implement Plugin Interface (`my_device_plugin.c`)
```c
#include "rtk_device_plugin.h"
#include "rtk_mqtt_client.h"
#include <stdio.h>
#include <string.h>

// Plugin state structure
typedef struct {
    char device_id[32];
    char location[64];
    int sensor_count;
    double last_temperature;
    bool is_initialized;
} my_device_context_t;

static my_device_context_t g_device_context = {0};

// Device information
static int get_device_info(rtk_device_info_t* info) {
    if (!info) return RTK_ERROR_INVALID_PARAM;
    
    strncpy(info->id, "aa:bb:cc:dd:ee:ff", sizeof(info->id));
    strncpy(info->type, "my_device", sizeof(info->type));
    strncpy(info->name, "My Custom Device", sizeof(info->name));
    strncpy(info->version, "1.0.0", sizeof(info->version));
    strncpy(info->manufacturer, "My Company", sizeof(info->manufacturer));
    
    return RTK_SUCCESS;
}

// Plugin initialization
static int initialize(const rtk_plugin_config_t* config) {
    if (!config) return RTK_ERROR_INVALID_PARAM;
    
    // Parse configuration
    cJSON* json = cJSON_Parse(config->json_config);
    if (!json) return RTK_ERROR_CONFIG_PARSE;
    
    cJSON* device_id = cJSON_GetObjectItem(json, "device_id");
    cJSON* location = cJSON_GetObjectItem(json, "location");
    cJSON* sensor_count = cJSON_GetObjectItem(json, "sensor_count");
    
    if (device_id && cJSON_IsString(device_id)) {
        strncpy(g_device_context.device_id, device_id->valuestring, 
                sizeof(g_device_context.device_id) - 1);
    }
    
    if (location && cJSON_IsString(location)) {
        strncpy(g_device_context.location, location->valuestring,
                sizeof(g_device_context.location) - 1);
    }
    
    if (sensor_count && cJSON_IsNumber(sensor_count)) {
        g_device_context.sensor_count = sensor_count->valueint;
    }
    
    cJSON_Delete(json);
    
    // Initialize hardware/sensors here
    printf("My Device initialized: ID=%s, Location=%s, Sensors=%d\n",
           g_device_context.device_id, g_device_context.location, 
           g_device_context.sensor_count);
    
    g_device_context.is_initialized = true;
    return RTK_SUCCESS;
}

// Start plugin operation
static int start(void) {
    if (!g_device_context.is_initialized) {
        return RTK_ERROR_NOT_INITIALIZED;
    }
    
    printf("My Device started\n");
    return RTK_SUCCESS;
}

// Stop plugin operation  
static int stop(void) {
    printf("My Device stopped\n");
    return RTK_SUCCESS;
}

// Get device state
static int get_state(rtk_device_state_t* state) {
    if (!state) return RTK_ERROR_INVALID_PARAM;
    
    strncpy(state->status, "online", sizeof(state->status));
    strncpy(state->health, "healthy", sizeof(state->health));
    state->uptime = 3600; // 1 hour
    state->last_seen = time(NULL);
    
    // Add custom properties
    state->properties = cJSON_CreateObject();
    cJSON_AddStringToObject(state->properties, "location", g_device_context.location);
    cJSON_AddNumberToObject(state->properties, "sensor_count", g_device_context.sensor_count);
    
    return RTK_SUCCESS;
}

// Get telemetry data
static int get_telemetry(const char* metric, rtk_telemetry_data_t* data) {
    if (!metric || !data) return RTK_ERROR_INVALID_PARAM;
    
    if (strcmp(metric, "temperature") == 0) {
        // Simulate temperature reading
        g_device_context.last_temperature = 20.0 + (rand() % 100) / 10.0;
        
        strncpy(data->metric, "temperature", sizeof(data->metric));
        data->value = g_device_context.last_temperature;
        strncpy(data->unit, "°C", sizeof(data->unit));
        data->timestamp = time(NULL);
        
        // Add labels
        data->labels = cJSON_CreateObject();
        cJSON_AddStringToObject(data->labels, "sensor", "ds18b20");
        cJSON_AddStringToObject(data->labels, "location", g_device_context.location);
        
        return RTK_SUCCESS;
    }
    
    return RTK_ERROR_NOT_FOUND;
}

// Handle commands
static int handle_command(const rtk_command_t* command, rtk_command_response_t* response) {
    if (!command || !response) return RTK_ERROR_INVALID_PARAM;
    
    strncpy(response->command_id, command->id, sizeof(response->command_id));
    response->timestamp = time(NULL);
    
    if (strcmp(command->action, "get_sensor_reading") == 0) {
        response->result = cJSON_CreateObject();
        cJSON_AddNumberToObject(response->result, "temperature", g_device_context.last_temperature);
        cJSON_AddNumberToObject(response->result, "sensor_count", g_device_context.sensor_count);
        
        strncpy(response->status, "success", sizeof(response->status));
        return RTK_SUCCESS;
    }
    else if (strcmp(command->action, "restart") == 0) {
        printf("Device restart requested\n");
        strncpy(response->status, "success", sizeof(response->status));
        return RTK_SUCCESS;
    }
    
    strncpy(response->status, "error", sizeof(response->status));
    strncpy(response->error_message, "Unknown command", sizeof(response->error_message));
    return RTK_ERROR_NOT_SUPPORTED;
}

// Cleanup resources
static int cleanup(void) {
    printf("My Device cleanup\n");
    memset(&g_device_context, 0, sizeof(g_device_context));
    return RTK_SUCCESS;
}

// Plugin entry point
RTK_PLUGIN_EXPORT const rtk_device_plugin_vtable_t* rtk_get_device_plugin_vtable(void) {
    static const rtk_device_plugin_vtable_t vtable = {
        .get_device_info = get_device_info,
        .initialize = initialize,
        .start = start,
        .stop = stop,
        .get_state = get_state,
        .get_telemetry = get_telemetry,
        .handle_command = handle_command,
        .cleanup = cleanup
    };
    
    return &vtable;
}
```

#### 3. Create Configuration (`my_device_config.json`)
```json
{
    "device_id": "aa:bb:cc:dd:ee:ff",
    "location": "Office Room 101",
    "sensor_count": 4,
    "telemetry_interval": 60,
    "state_interval": 30
}
```

#### 4. Create CMake Build (`CMakeLists.txt`)
```cmake
cmake_minimum_required(VERSION 3.10)

# Find RTK Framework
find_package(rtk-mqtt-framework REQUIRED)

# Create plugin library
add_library(my_device_plugin SHARED
    my_device_plugin.c
)

# Link against RTK Framework
target_link_libraries(my_device_plugin
    rtk-mqtt-framework
    ${CJSON_LIBRARIES}
)

target_include_directories(my_device_plugin PRIVATE
    ${RTK_INCLUDE_DIRS}
    ${CJSON_INCLUDE_DIRS}
)

# Set plugin properties
set_target_properties(my_device_plugin PROPERTIES
    PREFIX ""
    SUFFIX ".so"
)
```

#### 5. Build and Test
```bash
# Build plugin
mkdir build && cd build
cmake ..
make

# Test with plugin demo
cd ../../..
./build/examples/plugin_demo \
    -p ./examples/my_device/build/my_device_plugin.so \
    -c ./examples/my_device/my_device_config.json
```

</details>

### Advanced Plugin Features

<details>
<summary><strong>🚀 Advanced Plugin Capabilities</strong></summary>

#### Event Generation
```c
// In your plugin code
static void generate_custom_event(void) {
    rtk_event_t event = {0};
    strncpy(event.id, "evt_001", sizeof(event.id));
    strncpy(event.type, "sensor.anomaly", sizeof(event.type));
    strncpy(event.level, "warning", sizeof(event.level));
    strncpy(event.message, "Temperature spike detected", sizeof(event.message));
    event.timestamp = time(NULL);
    
    event.data = cJSON_CreateObject();
    cJSON_AddNumberToObject(event.data, "temperature", 45.2);
    cJSON_AddStringToObject(event.data, "sensor_id", "temp_01");
    
    // Send event through framework
    rtk_mqtt_publish_event(&event);
}
```

#### Plugin Configuration Validation
```c
static int validate_config(const cJSON* config) {
    // Check required fields
    if (!cJSON_GetObjectItem(config, "device_id")) {
        printf("Error: device_id is required\n");
        return RTK_ERROR_CONFIG_INVALID;
    }
    
    cJSON* sensor_count = cJSON_GetObjectItem(config, "sensor_count");
    if (sensor_count && (!cJSON_IsNumber(sensor_count) || sensor_count->valueint < 1)) {
        printf("Error: sensor_count must be a positive number\n");
        return RTK_ERROR_CONFIG_INVALID;
    }
    
    return RTK_SUCCESS;
}
```

#### Multi-threaded Plugin (FreeRTOS)
```c
#ifdef RTK_PLATFORM_FREERTOS
static TaskHandle_t sensor_task_handle = NULL;

static void sensor_task(void* pvParameters) {
    while (1) {
        // Read sensors
        rtk_telemetry_data_t data;
        get_telemetry("temperature", &data);
        
        // Publish telemetry
        rtk_mqtt_publish_telemetry(&data);
        
        vTaskDelay(pdMS_TO_TICKS(60000)); // 1 minute
    }
}

static int start(void) {
    BaseType_t result = xTaskCreate(
        sensor_task,
        "SensorTask", 
        2048,
        NULL,
        tskIDLE_PRIORITY + 2,
        &sensor_task_handle
    );
    
    return (result == pdPASS) ? RTK_SUCCESS : RTK_ERROR_MEMORY;
}
#endif
```

</details>

### Go Plugin Development

<details>
<summary><strong>🐹 Go Plugin Implementation</strong></summary>

#### Create Go Plugin (`my_device/plugin.go`)
```go
package main

import (
    "encoding/json"
    "fmt"
    "time"
    "math/rand"

    "github.com/rtk/mqtt-framework/pkg/device"
)

type MyDevicePlugin struct {
    config       *Config
    lastTemp     float64
    isRunning    bool
}

type Config struct {
    DeviceID     string `json:"device_id"`
    Location     string `json:"location"`
    SensorCount  int    `json:"sensor_count"`
}

func (p *MyDevicePlugin) GetDeviceInfo() (*device.Info, error) {
    return &device.Info{
        ID:           "aa:bb:cc:dd:ee:ff",
        Type:         "my_device",
        Name:         "My Custom Device",
        Version:      "1.0.0",
        Manufacturer: "My Company",
    }, nil
}

func (p *MyDevicePlugin) Initialize(configData []byte) error {
    p.config = &Config{}
    if err := json.Unmarshal(configData, p.config); err != nil {
        return fmt.Errorf("config parse error: %w", err)
    }
    
    fmt.Printf("My Device initialized: ID=%s, Location=%s, Sensors=%d\n",
               p.config.DeviceID, p.config.Location, p.config.SensorCount)
    
    return nil
}

func (p *MyDevicePlugin) Start() error {
    p.isRunning = true
    fmt.Println("My Device started")
    return nil
}

func (p *MyDevicePlugin) Stop() error {
    p.isRunning = false
    fmt.Println("My Device stopped")
    return nil
}

func (p *MyDevicePlugin) GetState() (*device.State, error) {
    return &device.State{
        Status:    "online",
        Health:    "healthy",
        Uptime:    time.Hour,
        LastSeen:  time.Now(),
        Timestamp: time.Now(),
        Properties: map[string]interface{}{
            "location":     p.config.Location,
            "sensor_count": p.config.SensorCount,
        },
    }, nil
}

func (p *MyDevicePlugin) GetTelemetryData(metric string) (*device.TelemetryData, error) {
    switch metric {
    case "temperature":
        p.lastTemp = 20.0 + rand.Float64()*10.0
        return &device.TelemetryData{
            Metric:    "temperature",
            Value:     p.lastTemp,
            Unit:      "°C",
            Timestamp: time.Now(),
            Labels: map[string]string{
                "sensor":   "ds18b20",
                "location": p.config.Location,
            },
        }, nil
    default:
        return nil, fmt.Errorf("metric not found: %s", metric)
    }
}

func (p *MyDevicePlugin) HandleCommand(cmd *device.Command) (*device.CommandResponse, error) {
    response := &device.CommandResponse{
        CommandID: cmd.ID,
        Timestamp: time.Now(),
    }
    
    switch cmd.Action {
    case "get_sensor_reading":
        response.Status = "success"
        response.Result = map[string]interface{}{
            "temperature":  p.lastTemp,
            "sensor_count": p.config.SensorCount,
        }
    case "restart":
        fmt.Println("Device restart requested")
        response.Status = "success"
    default:
        response.Status = "error"
        response.ErrorMessage = "Unknown command"
        return response, fmt.Errorf("unsupported command: %s", cmd.Action)
    }
    
    return response, nil
}

func (p *MyDevicePlugin) Cleanup() error {
    fmt.Println("My Device cleanup")
    return nil
}

// Plugin factory
func NewMyDevicePlugin() device.Plugin {
    return &MyDevicePlugin{}
}

func main() {
    // Plugin entry point for standalone execution
    plugin := NewMyDevicePlugin()
    
    // Initialize with test config
    config := Config{
        DeviceID:    "aa:bb:cc:dd:ee:ff",
        Location:    "Test Lab",
        SensorCount: 2,
    }
    
    configData, _ := json.Marshal(config)
    plugin.Initialize(configData)
    plugin.Start()
    
    // Simulate operation
    for i := 0; i < 5; i++ {
        state, _ := plugin.GetState()
        fmt.Printf("State: %+v\n", state)
        
        temp, _ := plugin.GetTelemetryData("temperature")
        fmt.Printf("Temperature: %.1f%s\n", temp.Value, temp.Unit)
        
        time.Sleep(2 * time.Second)
    }
    
    plugin.Stop()
    plugin.Cleanup()
}
```

#### Build and Run Go Plugin
```bash
cd examples/my_device
go mod init my-device-plugin
go mod tidy

# Build plugin
go build -o my_device_plugin .

# Run standalone
./my_device_plugin

# Use with Go framework
cd ../../framework-go
go run ./examples/framework_demo/ -plugin ../examples/my_device/my_device_plugin
```

</details>

## 🧪 Testing and Validation

### Unit Testing
```bash
# C/C++ tests
cd build
make test
ctest --output-on-failure

# Go tests
cd framework-go
go test ./... -v -race -cover
```

### Integration Testing
```bash
# Start test environment
./tools/mock_broker/mock_mqtt_broker &

# Run integration tests
./build/tools/rtk_cli/rtk_cli test-mqtt --broker localhost:1883

# Test plugin loading
./build/examples/plugin_demo --test-mode
```

### Performance Testing
```bash
# Load testing
./build/tools/rtk_cli/rtk_cli load-test \
    --connections 100 \
    --messages 1000 \
    --rate 10

# Memory profiling (Go)
cd framework-go
go test ./pkg/device -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## 📚 Examples and Use Cases

### Production Examples
- **[WiFi Router](examples/wifi_router/)**: Enterprise WiFi AP diagnostics
- **[IoT Sensor](examples/iot_sensor/)**: Environmental monitoring device  
- **[Smart Switch](examples/smart_switch/)**: Home automation device
- **[Industrial Gateway](examples/industrial_gateway/)**: Modbus/CAN gateway

### Deployment Scenarios
- **Edge Computing**: Raspberry Pi with multiple sensors
- **Industrial IoT**: ARM Cortex-M microcontrollers 
- **Cloud Integration**: Kubernetes deployments
- **Windows Services**: Enterprise network equipment

## 🔍 Troubleshooting

### Common Build Issues

<details>
<summary><strong>🔧 Build Configuration Issues</strong></summary>

#### CMake Configuration Problems
```bash
# Clear cache and reconfigure
rm -rf build/
mkdir build && cd build
cmake .. -DBUILD_EXAMPLES=ON -DBUILD_TOOLS=ON

# Check CMake version (3.10+ required)
cmake --version
```

#### Platform Detection Issues
```bash
# Force platform detection
cmake .. -DRTK_TARGET_POSIX=ON          # For Linux/macOS
cmake .. -DRTK_TARGET_WINDOWS=ON        # For Windows
cmake .. -DRTK_TARGET_FREERTOS=ON       # For FreeRTOS
```

#### ARM Toolchain Issues  
```bash
# Verify ARM GCC installation
arm-none-eabi-gcc --version
arm-none-eabi-gcc -print-multi-lib

# Missing newlib
brew install --cask gcc-arm-embedded
```

#### Go Module Problems
```bash
# Clean module cache
go clean -modcache
go mod download

# Verify Go version
go version  # Should be 1.19+
```

</details>

### Runtime Issues

<details>
<summary><strong>🔧 Common Runtime Problems</strong></summary>

#### MQTT Connection Failed
```bash
# Test MQTT broker connectivity
mosquitto_pub -h mqtt.broker.com -p 1883 -t test -m "hello"

# Check firewall/network
telnet mqtt.broker.com 1883

# Enable debug logging
export RTK_LOG_LEVEL=DEBUG
./your_application
```

#### Plugin Loading Failed
```bash
# Check plugin dependencies
ldd your_plugin.so

# Verify plugin interface
nm -D your_plugin.so | grep rtk_get_device_plugin_vtable

# Test plugin loading
./build/tools/rtk_cli/rtk_cli load-plugin your_plugin.so
```

#### Memory Issues (Embedded)
```c
// Monitor FreeRTOS heap
UBaseType_t uxHighWaterMark = uxTaskGetStackHighWaterMark(NULL);
size_t xFreeHeapSize = xPortGetFreeHeapSize();

printf("Stack high water mark: %lu\n", uxHighWaterMark);
printf("Free heap: %zu bytes\n", xFreeHeapSize);
```

</details>

## 📖 Documentation

- **[API Reference](docs/API_Reference.md)** - Complete API documentation
- **[FreeRTOS Network Architecture](docs/FreeRTOS_Network_Architecture.md)** - Embedded networking details
- **[Plugin Development](docs/Plugin_Development_Guide.md)** - Advanced plugin features
- **[Protocol Specification](docs/SPEC.md)** - RTK MQTT protocol details

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

Copyright (c) 2024 Realtek Semiconductor Corp.

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

---

## 🚀 Quick Links

- **[⚡ Quick Start Guide](QUICKSTART.md)** - Get running in 5 minutes
- **[📋 Universal Plan](UNIVERSAL_PLAN.md)** - Complete technical roadmap  
- **[🔧 Build Instructions](#platform-specific-builds)** - Platform-specific builds
- **[🔌 Plugin Development](#plugin-development-guide)** - Create custom plugins
- **[📚 Examples](#examples-and-use-cases)** - Real-world use cases
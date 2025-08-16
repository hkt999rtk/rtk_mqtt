# RTK MQTT Framework C++ DLL Demo

This directory contains examples demonstrating how to use the RTK MQTT Framework Go DLL from Windows C++ code and C plugin implementations.

## Overview

The RTK MQTT Framework provides a Go-based DLL that can be called from C/C++ applications on Windows and other platforms. This demo includes:

1. **C++ DLL Demo** (`main.cpp`) - Shows how to dynamically load and use the RTK MQTT Framework DLL from C++
2. **Smart Thermostat Plugin** (`smart_thermostat_plugin.c`) - A complete C plugin example that simulates a smart thermostat device

## Files

- `main.cpp` - C++ demonstration of DLL usage with wrapper class
- `smart_thermostat_plugin.c` - C plugin implementation for smart thermostat
- `CMakeLists.txt` - Cross-platform build configuration
- `README.md` - This documentation file

## Prerequisites

### Windows
- Visual Studio 2017+ or MinGW-w64
- CMake 3.10+
- RTK MQTT Framework Go DLL (`rtk_mqtt_framework.dll`)

### Linux/macOS
- GCC or Clang compiler
- CMake 3.10+
- RTK MQTT Framework Go shared library (`.so` or `.dylib`)

## Building

### Method 1: Using CMake (Recommended)

```bash
# Create build directory
mkdir build
cd build

# Configure and build
cmake ..
make

# On Windows with Visual Studio
cmake .. -G "Visual Studio 16 2019"
cmake --build . --config Release
```

### Method 2: Direct Compilation

#### Windows (MinGW)
```bash
# C++ Demo
g++ -std=c++11 -o cpp_dll_demo.exe main.cpp

# C Plugin
gcc -std=c99 -o smart_thermostat_plugin.exe smart_thermostat_plugin.c
```

#### Linux/macOS
```bash
# C++ Demo
g++ -std=c++11 -o cpp_dll_demo main.cpp -ldl -lpthread

# C Plugin  
gcc -std=c99 -o smart_thermostat_plugin smart_thermostat_plugin.c -ldl -lpthread -lm
```

## Running the Examples

### 1. C++ DLL Demo

This example demonstrates basic DLL usage including client management, configuration, and message publishing.

```bash
# Windows
./cpp_dll_demo.exe

# Linux/macOS
./cpp_dll_demo
```

**Expected Output:**
```
=== RTK MQTT Framework Windows C++ DLL Demo ===
RTK MQTT Framework Version: 1.0.0
Connected to MQTT broker successfully

=== Simulating Device Operation ===
Published device state successfully
Published telemetry (cpu_usage: 25.5 %) successfully
Published telemetry (memory_usage: 512 MB) successfully
Published telemetry (temperature: 35.2 °C) successfully
Published event (system.telemetry) successfully
...
=== Demo Completed ===
C++ DLL demo finished successfully!
```

### 2. Smart Thermostat Plugin

This example simulates a complete smart thermostat device with temperature control, energy monitoring, and smart scheduling.

```bash
# Basic usage (default settings)
./smart_thermostat_plugin

# Custom configuration
./smart_thermostat_plugin smart_thermostat_001 test.mosquitto.org 1883

# Windows
smart_thermostat_plugin.exe device_123 broker.example.com 1883
```

**Command Line Arguments:**
- `device_id` (optional): Device MAC address or unique ID (default: "smart_thermostat_001")
- `broker_host` (optional): MQTT broker hostname (default: "test.mosquitto.org")
- `broker_port` (optional): MQTT broker port (default: 1883)

**Expected Output:**
```
=== RTK MQTT Framework - Smart Thermostat Plugin ===
Configuration:
  Device ID: smart_thermostat_001
  MQTT Broker: test.mosquitto.org:1883

✓ Smart Thermostat Plugin initialized successfully
  RTK Framework Version: 1.0.0

Starting thermostat operation...
Press Ctrl+C to stop

Thermostat state published (Temp: 20.5°C, Target: 22.0°C, Status: heating [ACTIVE])
Telemetry published (Power: 1500W)
Thermostat state published (Temp: 21.2°C, Target: 22.0°C, Status: heating [ACTIVE])
Thermostat state published (Temp: 22.1°C, Target: 22.0°C, Status: idle)
```

## Code Structure

### C++ DLL Demo (`main.cpp`)

The C++ demo uses an object-oriented approach with a wrapper class:

```cpp
class RTKMQTTClient {
    // Dynamic DLL loading
    bool LoadDLL(const std::string& dll_path);
    
    // High-level interface methods
    bool ConfigureMQTT(const std::string& broker_host, int broker_port, const std::string& client_id);
    bool PublishState(const std::string& status, const std::string& health, int64_t uptime);
    bool PublishTelemetry(const std::string& metric, double value, const std::string& unit);
    // ...
};
```

**Key Features:**
- Automatic DLL loading and function binding
- RAII resource management
- Type-safe C++ interface
- Error handling and logging

### Smart Thermostat Plugin (`smart_thermostat_plugin.c`)

The C plugin demonstrates a complete IoT device implementation:

```c
typedef struct {
    double target_temperature;
    double current_temperature;
    int heating_enabled;
    int cooling_enabled;
    double power_consumption;
    char schedule_mode[32];
} smart_thermostat_state_t;
```

**Features:**
- Temperature control simulation
- Power consumption calculation
- Automatic heating/cooling logic
- Periodic telemetry and state reporting
- Event generation for system activities
- JSON data formatting

## MQTT Topic Structure

The examples publish to RTK-compliant MQTT topics:

```
rtk/v1/{tenant}/{site}/{device_id}/state          # Device state (retained)
rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}  # Telemetry data
rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}    # Events and alerts
```

### Example Topics:
```
rtk/v1/demo_tenant/demo_site/00:11:22:33:44:55/state
rtk/v1/demo_tenant/demo_site/00:11:22:33:44:55/telemetry/cpu_usage
rtk/v1/demo_tenant/demo_site/00:11:22:33:44:55/evt/system.startup

rtk/v1/home_automation/residence/smart_thermostat_001/state
rtk/v1/home_automation/residence/smart_thermostat_001/telemetry/temperature
rtk/v1/home_automation/residence/smart_thermostat_001/evt/thermostat.target_adjusted
```

## Message Examples

### Device State Message:
```json
{
  "status": "online",
  "health": "healthy", 
  "uptime": 3600,
  "last_seen": 1692123456,
  "properties": {
    "target_temp": 22.0,
    "current_temp": 21.8,
    "heating": false,
    "power_kw": 0.025,
    "mode": "auto"
  }
}
```

### Telemetry Message:
```json
{
  "metric": "temperature",
  "value": 21.8,
  "unit": "°C",
  "timestamp": 1692123456,
  "labels": {
    "sensor": "internal",
    "type": "ambient"
  }
}
```

### Event Message:
```json
{
  "id": "therm_1_1692123456",
  "type": "thermostat.target_adjusted",
  "level": "info",
  "message": "Target temperature adjusted",
  "timestamp": 1692123456,
  "data": {
    "old_target": 20.0,
    "new_target": 22.0
  }
}
```

## Troubleshooting

### Common Issues:

1. **DLL/SO not found**
   - Ensure the RTK MQTT Framework DLL/shared library is in the same directory as the executable
   - Check the library name matches the platform (`.dll` for Windows, `.so` for Linux, `.dylib` for macOS)

2. **Function not found errors**
   - Verify the DLL was built with the correct exports
   - Check that function names match the header file

3. **Connection failures**
   - Verify MQTT broker is accessible
   - Check firewall settings
   - Try using `test.mosquitto.org:1883` for testing

4. **Compilation errors**
   - Ensure the RTK MQTT Framework header file is available
   - Check compiler flags and standards (C++11, C99)

### Debug Mode:

Add debug output by defining `RTK_DEBUG`:

```bash
# Compile with debug output
gcc -DRTK_DEBUG -std=c99 -o smart_thermostat_plugin smart_thermostat_plugin.c -ldl -lpthread -lm
```

## Integration Guide

To integrate the RTK MQTT Framework DLL into your own projects:

1. **Copy the header file** (`rtk_mqtt_framework.h`) to your project
2. **Load the DLL dynamically** using platform-specific APIs
3. **Get function pointers** for the RTK functions you need
4. **Initialize a client** and configure MQTT/device settings
5. **Connect** to your MQTT broker
6. **Publish messages** using the RTK topic structure

See the `RTKMQTTClient` class in `main.cpp` for a complete implementation example.

## License

This demo code is provided as part of the RTK MQTT Framework project. See the main project license for details.
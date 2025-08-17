# RTK MQTT Framework

A comprehensive MQTT diagnostic communication system for IoT devices, providing standardized remote diagnostics and monitoring capabilities for WiFi devices, servers, and network equipment.

## Project Overview

This project provides a complete MQTT diagnostic ecosystem consisting of:
- **Protocol Specification** - Standardized MQTT diagnostic protocol for IoT devices
- **C Framework** - Production-ready C library for device integration  
- **Go Broker** - Lightweight MQTT broker implementation
- **Go Controller** - Command-line interface for device management and diagnostics

## Core Features

- **Hierarchical Topic Structure**: `rtk/v1/{tenant}/{site}/{device_id}/...`
- **Multiple Message Types**: State, telemetry, events, attributes, commands, and LWT
- **Diagnostic Scenarios**: WiFi roaming, connection failures, ARP packet loss, etc.
- **Schema Versioning**: Semantic versioning for schema identifiers
- **Cross-Platform Support**: FreeRTOS, Windows, Linux compatibility
- **Interactive CLI**: Command-line interface for device management and diagnostics

## Project Structure

```
rtk_mqtt/
├── docs/                           # Protocol specification and diagrams
│   ├── SPEC.md                     # Main protocol specification
│   ├── *.csv                       # Enums, events, and structure definitions
│   ├── *.puml                      # PlantUML sequence diagrams
│   ├── *_example.md                # Diagnostic scenario examples
│   └── gen_word/                   # Documentation generation tools
├── rtk_mqtt_broker/                # Go MQTT broker implementation
├── rtk_mqtt_framework/             # C/C++ client framework
├── rtk_controller/                 # Go CLI controller for device management
└── wifi_diagnosis_schemas.json     # WiFi diagnostic event JSON schemas
```

## Quick Start

### Requirements

- **Documentation**: Python 3.x, python-docx, markdown, beautifulsoup4
- **Broker & Controller**: Go 1.19+
- **Framework**: CMake 3.16+, C99/C++11 compiler

### RTK Controller

```bash
cd rtk_controller

# Build and run the controller
go build -o rtk-controller cmd/controller/main.go
./rtk-controller

# Interactive mode
./rtk-controller interactive

# Run demo commands
./demo_cli.sh
```

### MQTT Broker

```bash
cd rtk_mqtt_broker

# Build and run
go build -o rtk_mqtt_broker .
./rtk_mqtt_broker -config config/config.yaml

# Test client
go run test/mqtt_client.go
```

### C Framework

```bash
cd rtk_mqtt_framework

# Build framework and examples
mkdir build && cd build
cmake -DBUILD_EXAMPLES=ON -DBUILD_TOOLS=ON ..
make -j$(nproc)

# Run example plugin
./examples/plugin_demo -p ./examples/wifi_router/wifi_router_plugin.so -c ../examples/wifi_router/wifi_router_config.json
```

### Documentation Generation

```bash
cd docs/gen_word

# Generate Word specification document
python3 generate_mqtt_spec_doc.py

# Generate PlantUML sequence diagrams
python3 generate_plantuml_sequences.py

# Generate Graphviz diagrams
python3 generate_graphviz_diagrams.py
```

## Message Types

| Type | Topic Format | Description | Retained |
|------|--------------|-------------|----------|
| **state** | `rtk/v1/{tenant}/{site}/{device_id}/state` | Device health status summary | ✓ |
| **telemetry** | `rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}` | Performance metrics data | ✗ |
| **evt** | `rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}` | Events and alerts | ✗ |
| **attr** | `rtk/v1/{tenant}/{site}/{device_id}/attr` | Device attributes | ✓ |
| **cmd** | `rtk/v1/{tenant}/{site}/{device_id}/cmd/{req\|ack\|res}` | Command request/response | ✗ |
| **lwt** | `rtk/v1/{tenant}/{site}/{device_id}/lwt` | Online/offline status | ✓ |

## Diagnostic Scenarios

### WiFi Roaming Issues
- **Event Type**: `wifi.roam_miss`
- **Trigger**: Roaming failure or excessive delay
- **Data**: Original AP signal, target AP signal, roaming duration

### Connection Failures
- **Event Type**: `wifi.connect_fail`  
- **Trigger**: Authentication failure, timeout, etc.
- **Data**: Failure reason, retry count, error codes

### ARP Packet Loss
- **Event Type**: `wifi.arp_loss`
- **Trigger**: High ARP response loss rate
- **Data**: Packet loss rate, response time statistics

## Development Guide

### Adding New Diagnostic Scenarios

1. Update schema definitions in `wifi_diagnosis_schemas.json`
2. Add corresponding example documentation in `docs/`
3. Create PlantUML sequence diagrams
4. Update main specification document `docs/SPEC.md`

### Schema Versioning

```json
{
  "schema_id": "evt.wifi.roam_miss/1.0",
  "version": "1.0.0",
  "properties": {
    // Schema definition
  }
}
```

### Controller Usage

```bash
# Interactive CLI session
./rtk-controller interactive

# Device management commands
device list
device add --id router01 --type wifi_router
device status router01

# Command execution
command send router01 restart
command history

# System monitoring
system status
system logs
```

## Example Code

### C Framework - WiFi Roaming Event

```c
#include "rtk_mqtt_client.h"

// Create roaming failure event
rtk_wifi_roam_event_t roam_event = {
    .timestamp = get_current_timestamp(),
    .old_bssid = "aa:bb:cc:dd:ee:ff",
    .new_bssid = "11:22:33:44:55:66",
    .roam_duration_ms = 2500,
    .failure_reason = "signal_weak"
};

// Send event
rtk_mqtt_publish_event("wifi.roam_miss", &roam_event);
```

### C Framework - Command Subscription

```c
// Register command handler
rtk_mqtt_register_command_handler("config_update", handle_config_update);

void handle_config_update(const char* payload) {
    // Process configuration update command
    parse_and_apply_config(payload);
    
    // Send acknowledgment
    rtk_mqtt_send_command_ack("config_update", "success");
}
```

## Architecture

### System Components
1. **MQTT Broker** - Go-based broker with RTK-specific authentication and routing
2. **Device Framework** - C library providing standardized MQTT client functionality
3. **CLI Controller** - Interactive command-line interface for device management
4. **Plugin System** - Dynamic loading of device-specific implementations
5. **Schema Validation** - JSON schema enforcement for message consistency

### Framework Architecture (C)
- **MQTT Adapter** - Eclipse Paho C client wrapper
- **Message Codec** - JSON encoding/decoding with validation
- **Topic Builder** - RTK topic format construction
- **Plugin Manager** - Dynamic plugin loading and lifecycle
- **Platform Compatibility** - FreeRTOS, Windows, Linux support

### Broker Architecture (Go)  
- **Core Broker** - Mochi MQTT server wrapper
- **Authentication** - User credential validation
- **Configuration** - YAML-based settings management
- **Logging** - Structured logging with configurable levels

## License

[License Information]

## Contributing

Issues and Pull Requests are welcome to improve this project.

## Contact

For questions or suggestions, please contact the project maintenance team.
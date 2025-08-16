# RTK MQTT Framework - Go Implementation

A comprehensive MQTT diagnostic communication framework written in Go, implementing the RTK MQTT protocol specification for IoT devices, network equipment, and diagnostic systems.

## Overview

The RTK MQTT Framework Go implementation provides a modern, type-safe, and high-performance solution for building MQTT-based diagnostic systems. It supports multiple device types, real-time telemetry, event reporting, and command handling with comprehensive schema validation.

## Features

### Core Framework
- 🚀 **High-Performance MQTT Client** - Built on Eclipse Paho MQTT Go with connection pooling and automatic reconnection
- 🔧 **Plugin Architecture** - Extensible device plugin system with lifecycle management
- 📊 **Message Codec** - Comprehensive encoding/decoding for RTK protocol messages
- 🎯 **Topic Management** - Smart topic building and parsing with validation
- ⚙️ **Configuration Management** - Flexible configuration with Viper and validation
- 🔍 **Schema Validation** - JSON schema validation for message structures
- 📈 **Health Monitoring** - Built-in health checks and metrics collection

### Message Types
- **State Messages** - Device health and status reporting
- **Telemetry Messages** - Real-time sensor data and metrics
- **Event Messages** - Diagnostic events with schema validation
- **Command Messages** - Remote device control and configuration
- **Attribute Messages** - Device metadata and properties
- **LWT Messages** - Last Will Testament for connection monitoring

### Device Support
- **IoT Sensors** - Environmental sensors, industrial monitoring
- **WiFi Routers** - Network diagnostic events (roaming, connection failures, ARP loss)
- **Network Equipment** - Switches, access points, gateways
- **Custom Devices** - Extensible plugin system for any device type

## Architecture

```
framework-go/
├── pkg/                    # Public packages
│   ├── mqtt/              # MQTT client abstraction layer
│   ├── device/            # Device plugin system
│   ├── codec/             # Message encoding/decoding
│   ├── topic/             # Topic construction utilities
│   └── config/            # Configuration management
├── internal/              # Private packages
│   ├── platform/          # Platform-specific implementations
│   └── common/            # Shared utilities
├── examples/              # Example implementations
│   ├── wifi_router/       # WiFi router device plugin
│   ├── iot_sensor/        # IoT sensor device plugin
│   └── smart_switch/      # Smart switch device plugin
├── cmd/                   # Command-line tools
│   └── rtk-cli/           # RTK CLI utility
└── test/                  # Integration tests
```

## Quick Start

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd rtk_mqtt_framework/framework-go

# Download dependencies
go mod download

# Run tests
make test

# Build examples
make examples
```

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/rtk/mqtt-framework/pkg/config"
    "github.com/rtk/mqtt-framework/pkg/mqtt"
    "github.com/rtk/mqtt-framework/pkg/codec"
    "github.com/rtk/mqtt-framework/pkg/device"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }
    
    // Create MQTT client
    client, err := mqtt.NewClient(&cfg.MQTT)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create codec
    codec := codec.NewCodec()
    codec.SetDeviceInfo(cfg.Device.Tenant, cfg.Device.Site, cfg.Device.DeviceID)
    
    // Connect to broker
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect()
    
    // Create and publish a state message
    state := &device.State{
        Status:    "online",
        Health:    "healthy",
        Timestamp: time.Now(),
    }
    
    msg, err := codec.EncodeState(ctx, state)
    if err != nil {
        log.Fatal(err)
    }
    
    mqttMsg := &mqtt.Message{
        Topic:     msg.Topic,
        Payload:   msg.Payload,
        QoS:       mqtt.QoS(msg.QoS),
        Retained:  msg.Retained,
        Timestamp: time.Now(),
    }
    
    client.PublishMessage(ctx, mqttMsg)
    log.Println("State published successfully")
}
```

## Configuration

Configuration can be provided via YAML files, environment variables, or Go structs:

```yaml
mqtt:
  broker_host: "localhost"
  broker_port: 1883
  client_id: "rtk_device_001"
  keep_alive: 60
  clean_session: true

device:
  type: "wifi_router"
  tenant: "acme_corp"
  site: "office_main"
  device_id: "router_001"

logging:
  level: "info"
  format: "json"
```

## Development

### Building

```bash
# Build all packages
make build

# Build examples
make examples

# Run tests
make test

# Run tests with coverage
make coverage

# Run comprehensive test suite
./test.sh
```

### Testing

```bash
# Run all tests
make test

# Run tests with race detection
go test -race ./...

# Run specific package tests
go test -v ./pkg/mqtt

# Run benchmarks
make benchmark
```

### Code Quality

```bash
# Format code
make fmt

# Run go vet
make vet

# Run linters (if available)
make lint

# Install development tools
make install-deps
```

## Examples

### IoT Sensor

```bash
cd examples/iot_sensor
go run main.go
```

Simulates environmental sensors with:
- Temperature, humidity, pressure readings
- Configurable reading intervals
- Telemetry publishing
- Command handling (get readings, set intervals)

### WiFi Router

```bash
cd examples/wifi_router
go run main.go
```

Simulates WiFi router diagnostics with:
- Client connection monitoring
- Roaming event detection
- Connection failure reporting
- ARP loss diagnosis
- Network statistics

### Framework Demo

```bash
cd examples/framework_demo
go run main.go
```

Complete framework demonstration with:
- Multiple message types
- Event handling
- Command processing
- Subscription management
- Health monitoring

## Contributing

We welcome contributions! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run `make ci` to ensure all checks pass
6. Submit a pull request

## License

This project is licensed under the same terms as the parent RTK MQTT Framework.
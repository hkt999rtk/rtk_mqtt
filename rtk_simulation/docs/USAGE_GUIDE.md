# RTK Home Network Simulator - Usage Guide

## Table of Contents
1. [Quick Start](#quick-start)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Running Simulations](#running-simulations)
5. [Device Types](#device-types)
6. [Scenarios](#scenarios)
7. [Testing](#testing)
8. [Performance Optimization](#performance-optimization)
9. [Troubleshooting](#troubleshooting)

## Quick Start

### Basic Simulation
```bash
# Build the simulator
make build

# Run with default configuration
make run

# Run with custom configuration
./build/rtk-simulator run -c configs/home_basic.yaml
```

### Development Mode
```bash
# Run with verbose output
make dev

# Generate sample configurations
make config-basic
make config-advanced
make config-smart
```

## Installation

### Prerequisites
- Go 1.21 or higher
- MQTT Broker (optional, for full MQTT functionality)
- Make (for build automation)

### Building from Source
```bash
# Clone the repository
git clone <repository-url>
cd rtk_simulation

# Download dependencies
go mod download

# Build for current platform
make build

# Build for all platforms
make build-all
```

### Docker Installation
```bash
# Build Docker image
docker build -t rtk-simulator .

# Run container
docker run -v $(pwd)/configs:/configs rtk-simulator run -c /configs/home_basic.yaml
```

## Configuration

### Configuration File Structure
```yaml
simulation:
  name: "my_home"
  duration: 3600s
  network_type: "mesh_network"
  
broker:
  url: "tcp://localhost:1883"
  client_id: "rtk_simulator"
  username: "admin"
  password: "password"
  
devices:
  - id: "router_main"
    type: "router"
    tenant: "home"
    site: "main"
    location: "living_room"
    extra:
      ssid_2g: "HomeNetwork_2G"
      ssid_5g: "HomeNetwork_5G"
      max_clients: 50
      
  - id: "bulb_001"
    type: "smart_bulb"
    tenant: "home"
    site: "main"
    location: "bedroom"
    extra:
      max_brightness: 100
      color_support: true
      
scenarios:
  daily_routines:
    enabled: true
    patterns:
      - workday
      - weekend
      
  automation:
    enabled: true
    rules:
      - motion_lighting
      - temperature_control
      
  faults:
    enabled: false
    probability: 0.01
```

### Device Configuration Options

#### Network Devices
- **Router**: Main gateway device
  - `ssid_2g`, `ssid_5g`: WiFi network names
  - `channel_2g`, `channel_5g`: WiFi channels
  - `max_clients`: Maximum connected devices

- **Access Point**: WiFi extender
  - `ssid_2g`, `ssid_5g`: Network names
  - `wifi_password`: Network password
  - `security_type`: WPA2, WPA3

- **Switch**: Network switch
  - `port_count`: Number of ports (4-48)
  - `poe_enabled`: Power over Ethernet support

#### IoT Devices
- **Smart Bulb**: Controllable lighting
  - `max_brightness`: Maximum brightness level
  - `color_support`: RGB color support
  - `color_temp_range`: Color temperature range

- **Air Conditioner**: Climate control
  - `min_temp`, `max_temp`: Temperature range
  - `modes`: Available operation modes
  - `fan_speeds`: Fan speed levels

- **Security Camera**: Surveillance device
  - `resolution`: Video resolution
  - `night_vision`: Night vision capability
  - `motion_zones`: Motion detection zones

#### Client Devices
- **Smartphone**: Mobile device
  - `os`: Operating system (ios/android)
  - `battery_capacity`: Battery size
  - `apps`: Installed applications

- **Laptop**: Computer device
  - `os`: Operating system
  - `battery_capacity`: Battery size
  - `performance_mode`: Power profile

## Running Simulations

### Command Line Interface
```bash
# Basic run command
./build/rtk-simulator run -c config.yaml

# Validate configuration
./build/rtk-simulator validate configs/home_basic.yaml

# Run with specific duration
./build/rtk-simulator run -c config.yaml --duration 1h

# Run with verbose output
./build/rtk-simulator run -c config.yaml -v

# Run with specific log level
./build/rtk-simulator run -c config.yaml --log-level debug
```

### Programmatic Usage
```go
package main

import (
    "context"
    "rtk_simulation/pkg/config"
    "rtk_simulation/pkg/devices"
    "rtk_simulation/pkg/scenarios"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig("configs/home_basic.yaml")
    if err != nil {
        panic(err)
    }
    
    // Create device manager
    deviceManager := devices.NewDeviceManager()
    
    // Create devices
    for _, deviceCfg := range cfg.Devices {
        device, err := deviceManager.CreateDevice(deviceCfg, cfg.Broker)
        if err != nil {
            panic(err)
        }
        
        // Start device
        ctx := context.Background()
        device.Start(ctx)
    }
    
    // Run simulation
    // ...
}
```

## Scenarios

### Daily Routines
The simulator includes predefined daily routines that automatically control devices:

- **Morning Routine** (7:00-9:00)
  - Gradually increase lighting
  - Adjust temperature for comfort
  - Turn on morning appliances

- **Daytime Routine** (9:00-17:00)
  - Energy saving mode
  - Security monitoring
  - Minimal device activity

- **Evening Routine** (17:00-23:00)
  - Comfortable lighting
  - Entertainment systems
  - Dinner preparation

- **Night Routine** (23:00-7:00)
  - Security mode
  - Minimal lighting
  - Sleep environment

### Automation Rules
Pre-configured automation rules include:

- **Motion-Activated Lighting**: Lights turn on when motion detected
- **Temperature Control**: AC adjusts based on room temperature
- **Security Alerts**: Notifications on intrusion detection
- **Energy Saving**: Devices turn off when no presence detected

### Custom Scenarios
Create custom scenarios using the script engine:

```yaml
script:
  id: "custom_morning"
  name: "Custom Morning Routine"
  steps:
    - type: action
      action:
        type: device
        target: smart_bulb
        command: turn_on
        parameters:
          brightness: 30
          
    - type: wait
      delay: 5m
      
    - type: action
      action:
        type: device
        target: air_conditioner
        command: set_temperature
        parameters:
          temperature: 24
```

## Testing

### Running Tests
```bash
# Run all tests
make test

# Run unit tests only
go test ./tests/unit/...

# Run integration tests
go test ./tests/integration/...

# Run performance benchmarks
go test -bench=. ./tests/performance/...

# Generate coverage report
make coverage
```

### Test Categories

#### Unit Tests
- Device creation and configuration
- Command handling
- State generation
- Event processing

#### Integration Tests
- Multi-device interactions
- Scenario execution
- Network topology
- MQTT communication

#### Performance Tests
- Device creation benchmarks
- Event throughput
- Concurrent operations
- Memory usage

#### End-to-End Tests
- Complete home simulation
- 24-hour cycles
- Fault tolerance
- Load testing

## Performance Optimization

### Configuration Tips
1. **Limit concurrent devices**: Start with fewer devices and scale up
2. **Adjust update intervals**: Increase intervals for better performance
3. **Disable unused features**: Turn off scenarios you don't need
4. **Use appropriate log levels**: Debug logging impacts performance

### Resource Management
```yaml
performance:
  max_concurrent_devices: 100
  state_update_interval: 30s
  telemetry_interval: 10s
  event_batch_size: 100
  worker_pool_size: 10
```

### Monitoring Performance
```bash
# Run with performance profiling
./build/rtk-simulator run -c config.yaml --profile cpu.prof

# Analyze profile
go tool pprof cpu.prof

# Monitor resource usage
./build/rtk-simulator run -c config.yaml --metrics
```

## Troubleshooting

### Common Issues

#### MQTT Connection Failed
```
Error: failed to connect to MQTT broker
Solution: 
1. Verify broker is running
2. Check connection settings
3. Verify credentials
4. Check firewall settings
```

#### High Memory Usage
```
Issue: Memory consumption increases over time
Solution:
1. Reduce number of devices
2. Increase cleanup intervals
3. Disable event history
4. Check for memory leaks with profiling
```

#### Devices Not Responding
```
Issue: Devices don't respond to commands
Solution:
1. Check device health status
2. Verify device is started
3. Check command format
4. Review logs for errors
```

### Debug Mode
Enable debug mode for detailed logging:
```bash
# Set log level to debug
export RTK_SIM_LOG_LEVEL=debug

# Run with debug output
./build/rtk-simulator run -c config.yaml --debug

# Enable specific component debugging
./build/rtk-simulator run -c config.yaml --debug-components devices,scenarios
```

### Log Analysis
```bash
# Filter logs by component
grep "component=device_manager" simulation.log

# Find errors
grep -i error simulation.log

# Track specific device
grep "device_id=bulb_001" simulation.log
```

## Advanced Usage

### Custom Device Types
Create custom device types by implementing the Device interface:

```go
type CustomDevice struct {
    *base.BaseDevice
    // Custom fields
}

func NewCustomDevice(config base.DeviceConfig, mqttConfig base.MQTTConfig) (*CustomDevice, error) {
    baseDevice := base.NewBaseDeviceWithMQTT(config, mqttConfig)
    return &CustomDevice{
        BaseDevice: baseDevice,
    }, nil
}

func (d *CustomDevice) HandleCommand(cmd base.Command) error {
    // Custom command handling
    return nil
}
```

### Event Handlers
Register custom event handlers:

```go
handler := scenarios.EventHandler{
    ID:   "custom_handler",
    Name: "Custom Handler",
    Handler: func(event scenarios.Event) error {
        // Process event
        return nil
    },
}

eventBus.Subscribe("device.*", handler)
```

### Integration with External Systems
```go
// Export metrics to Prometheus
exporter := monitoring.NewPrometheusExporter()
exporter.Start(":9090")

// Send data to InfluxDB
influx := monitoring.NewInfluxDBClient("http://localhost:8086")
influx.WriteMetrics(metrics)
```

## Best Practices

1. **Start Small**: Begin with a few devices and gradually add more
2. **Use Realistic Configurations**: Match your actual home setup
3. **Monitor Performance**: Keep an eye on resource usage
4. **Validate Configurations**: Always validate before running
5. **Use Version Control**: Track configuration changes
6. **Document Custom Scenarios**: Keep notes on custom scripts
7. **Regular Testing**: Run tests after modifications
8. **Backup Configurations**: Keep backups of working configs

## Support

For issues, questions, or contributions:
- GitHub Issues: [Report bugs or request features]
- Documentation: [Extended documentation]
- Examples: Check the `configs/` directory for examples

## License

[License information]
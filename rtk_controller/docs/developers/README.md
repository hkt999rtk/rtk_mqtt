# RTK MQTT Developer Documentation

Welcome to the RTK MQTT system developer documentation. This comprehensive guide provides everything you need to understand, integrate, and extend the RTK MQTT protocol for IoT device communication.

## Documentation Structure

### 📚 Core Protocol Documentation
Essential protocol specifications and reference materials:

- **[MQTT Protocol Specification](core/MQTT_PROTOCOL_SPEC.md)** - Complete protocol definition and message formats
- **[Commands & Events Reference](core/COMMANDS_EVENTS_REFERENCE.md)** - Comprehensive command/response catalog 
- **[Topic Structure Guide](core/TOPIC_STRUCTURE.md)** - MQTT topic hierarchy and routing
- **[Schema Reference](core/SCHEMA_REFERENCE.md)** - JSON schema definitions and validation

### 🔌 Device Integration Guides
Device-specific implementation guides:

- **[AP/Router Integration](devices/AP_ROUTER_INTEGRATION.md)** - Access Point and Router integration
- **[NIC Integration](devices/NIC_INTEGRATION.md)** - Network Interface Card implementation
- **[IoT Device Integration](devices/IOT_DEVICE_INTEGRATION.md)** - General IoT sensor integration
- **[Mesh Node Integration](devices/MESH_NODE_INTEGRATION.md)** - Mesh networking support
- **[Switch Integration](devices/SWITCH_INTEGRATION.md)** - Network switch management

### 🚀 Implementation Guides
Step-by-step development resources:

- **[Quick Start Guide](guides/QUICK_START_GUIDE.md)** - Get started in 15 minutes
- **[Testing & Integration](guides/TESTING_INTEGRATION.md)** - Comprehensive testing strategies
- **[Deployment Guide](guides/DEPLOYMENT_GUIDE.md)** - Production deployment procedures
- **[Troubleshooting Guide](guides/TROUBLESHOOTING_GUIDE.md)** - Common issues and solutions

### 🔍 Diagnostics & Monitoring
Network analysis and monitoring tools:

- **[Network Diagnostics](diagnostics/NETWORK_DIAGNOSTICS.md)** - Network performance analysis
- **[WiFi Diagnostics](diagnostics/WIFI_DIAGNOSTICS.md)** - WiFi-specific diagnostic tools
- **[QoS Monitoring](diagnostics/QOS_MONITORING.md)** - Quality of Service analysis

### 🛠️ Development Tools
Essential tools for development and testing:

- **[CLI Tools](tools/CLI_TOOLS.md)** - Command-line interface reference
- **[MQTT Testing Tools](tools/MQTT_TESTING_TOOLS.md)** - Protocol testing framework

## Quick Navigation

### New to RTK MQTT?
1. Start with **[Quick Start Guide](guides/QUICK_START_GUIDE.md)**
2. Read **[MQTT Protocol Specification](core/MQTT_PROTOCOL_SPEC.md)**
3. Choose your **[Device Integration Guide](devices/)**

### Adding Device Support?
1. Review **[Topic Structure Guide](core/TOPIC_STRUCTURE.md)**
2. Check **[Commands & Events Reference](core/COMMANDS_EVENTS_REFERENCE.md)**
3. Follow device-specific integration guide in **[devices/](devices/)**
4. Use **[Testing Framework](guides/TESTING_INTEGRATION.md)**

### Debugging Issues?
1. Check **[Troubleshooting Guide](guides/TROUBLESHOOTING_GUIDE.md)**
2. Use **[Diagnostics Tools](diagnostics/)**
3. Analyze with **[CLI Tools](tools/CLI_TOOLS.md)**

### Setting up Production?
1. Follow **[Deployment Guide](guides/DEPLOYMENT_GUIDE.md)**
2. Set up **[QoS Monitoring](diagnostics/QOS_MONITORING.md)**
3. Configure **[Network Diagnostics](diagnostics/NETWORK_DIAGNOSTICS.md)**

## Protocol Overview

The RTK MQTT protocol provides standardized communication for IoT devices with:

**Topic Structure**: `rtk/v1/{tenant}/{site}/{device_id}/{message_type}[/{sub_type}]`

**Standard Message Format**: All messages follow a consistent JSON structure with `payload` wrapper:
```json
{
  "schema": "message_type/version",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    // Business logic fields wrapped here
  }
}
```

**Key Features**:
- Device state management and health monitoring
- Real-time telemetry and event reporting
- Command/response patterns with acknowledgments
- Network topology discovery and analysis
- Quality of Service monitoring and optimization
- Comprehensive diagnostics and troubleshooting
- **JSON Schema validation** for all message formats

**Supported Devices**:
- Access Points and Routers
- Network Interface Cards
- IoT Sensors and Actuators
- Mesh Network Nodes
- Network Switches

**Quality Assurance**: All JSON examples in this documentation comply with the RTK MQTT protocol specifications and can be directly used in implementations.

## Getting Help

- **Documentation Issues**: Create an issue in the repository
- **Protocol Questions**: Refer to [MQTT Protocol Specification](core/MQTT_PROTOCOL_SPEC.md)
- **Integration Help**: Check device-specific guides in [devices/](devices/)
- **Testing Issues**: Use tools in [MQTT Testing Framework](tools/MQTT_TESTING_TOOLS.md)

## Contributing

When contributing to RTK MQTT:

1. Follow the protocol specifications in [core/](core/)
2. Add appropriate tests using [Testing Framework](guides/TESTING_INTEGRATION.md)
3. Update relevant documentation
4. Use [CLI Tools](tools/CLI_TOOLS.md) for validation

---

*RTK MQTT Developer Documentation v1.0 - Complete developer guide for RTK MQTT protocol implementation*
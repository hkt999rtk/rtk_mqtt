# RTK Controller - Topology CLI Usage Guide

## Overview

The RTK Controller provides comprehensive network topology management through its interactive CLI. This guide covers all topology-related commands and their usage.

## Starting the CLI

```bash
./rtk-controller --cli
```

## Topology Commands

### Basic Topology Commands

#### Show Current Topology
Display the complete network topology in JSON format.

```
topology show
```

Output includes:
- Device inventory with detailed information
- Network connections between devices
- Gateway information
- Topology metadata (tenant, site, timestamps)

#### List All Devices
Display a summary list of all devices in the network.

```
topology devices
```

Output format:
```
Found 3 devices
- ap-01: 00:11:22:33:44:66 (online)
- client-01: AA:BB:CC:DD:EE:01 (online)
- gateway-01: 00:11:22:33:44:55 (online)
```

### Quality Monitoring Commands (Planned)

#### Quality Status
```
quality status
```
Shows overall connection quality metrics across the network.

#### Device Quality
```
quality device <device-id>
```
Shows quality metrics for a specific device.

#### Quality Report
```
quality report
```
Generates a comprehensive quality analysis report.

#### Quality Alerts
```
quality alerts
```
Lists active quality-related alerts and warnings.

### Roaming Analysis Commands (Planned)

#### Roaming Status
```
roaming status
```
Shows current roaming detection status and statistics.

#### Roaming Events
```
roaming events
```
Lists recent roaming events detected in the network.

#### Roaming Anomalies
```
roaming anomalies
```
Shows detected roaming anomalies and unusual patterns.

#### Device Roaming
```
roaming device <device-id>
```
Shows roaming information for a specific device.

### Monitoring Integration Commands (Planned)

#### Monitoring Status
```
monitor status
```
Shows monitoring system integration status.

#### Monitoring Dashboard
```
monitor dashboard
```
Displays a monitoring dashboard with key metrics.

#### System Health
```
monitor health
```
Shows overall system health status.

#### Monitoring Metrics
```
monitor metrics
```
Lists all available monitoring metrics.

## Sample Data Structure

### Device Information
Each device in the topology contains:
- **device_id**: Unique identifier
- **device_type**: router, ap, client, switch, iot
- **primary_mac**: Primary MAC address
- **hostname**: Device hostname
- **manufacturer**: Device manufacturer
- **model**: Device model
- **location**: Physical location
- **role**: Network role (gateway, access_point, client, etc.)
- **interfaces**: Network interfaces with detailed configuration
- **capabilities**: Device capabilities (routing, nat, dhcp, ap, etc.)
- **online**: Current online status
- **last_seen**: Last activity timestamp

### Connection Information
Network connections include:
- **id**: Connection identifier
- **from_device_id**: Source device
- **to_device_id**: Target device
- **from_interface**: Source interface
- **to_interface**: Target interface
- **connection_type**: ethernet, wireless, etc.
- **is_direct_link**: Direct connection flag
- **metrics**: Connection quality metrics
  - **rssi**: Signal strength (WiFi)
  - **link_speed**: Connection speed (Mbps)
  - **bandwidth**: Available bandwidth (Mbps)
  - **latency**: Connection latency (ms)

### Gateway Information
Gateway details include:
- **device_id**: Gateway device identifier
- **ip_address**: Internal IP address
- **external_ip**: External/WAN IP address
- **isp_info**: ISP information
- **dns_servers**: DNS server addresses

## Loading Test Data

To load sample topology data for testing:

1. Generate sample data:
```bash
go run test/topology_test_simple.go > test/sample_topology.json
```

2. Load data into controller:
```bash
go run test/load_topology.go
```

3. Verify loaded data:
```bash
./rtk-controller --cli
> topology show
> topology devices
```

## Implementation Status

### Completed Features
✅ Basic topology display (`topology show`)  
✅ Device listing (`topology devices`)  
✅ Topology data persistence  
✅ Sample data generation and loading  

### In Progress
🔄 Quality monitoring commands  
🔄 Roaming analysis commands  
🔄 Monitoring integration  
🔄 Alert management  

### Planned Features
📋 Topology visualization (ASCII, DOT, PlantUML)  
📋 Device discovery and auto-detection  
📋 Real-time topology updates via MQTT  
📋 Historical topology tracking  
📋 Network path analysis  
📋 Topology export/import  

## Architecture Notes

### Storage
- Uses BuntDB for persistent storage
- Data stored in `data/controller.db`
- Supports multi-tenant topology (tenant/site separation)

### Components
- **TopologyManager**: Core topology management
- **TopologyStorage**: Persistence layer
- **TopologyCommands**: CLI command handlers
- **TopologyProcessor**: MQTT message processing
- **DeviceDiscovery**: Network discovery engine

## Troubleshooting

### No devices shown
- Ensure topology data is loaded: `go run test/load_topology.go`
- Check tenant/site configuration matches (default: "default/default")
- Verify database file exists: `ls -la data/controller.db`

### Topology not updating
- Check MQTT connection status
- Verify topology processor is running
- Review logs for errors: `./rtk-controller --cli 2>&1 | grep topology`

### Command not implemented
- Many commands show "not yet implemented" - these are planned features
- Check implementation status above
- Refer to stub implementations in `topology_commands_stub.go`

## Development

### Adding New Commands
1. Add command handler in `internal/cli/topology_commands_stub.go`
2. Register command in interactive CLI handler
3. Implement actual functionality in topology modules
4. Update this documentation

### Testing Commands
1. Build controller: `go build -o rtk-controller ./cmd/controller`
2. Load test data: `go run test/load_topology.go`
3. Test commands: `./rtk-controller --cli`

## References
- [TOPOLOGY_DETECTION_PLAN.md](../TOPOLOGY_DETECTION_PLAN.md) - Implementation plan
- [test/topology_test_simple.go](../test/topology_test_simple.go) - Sample data generator
- [internal/topology/](../internal/topology/) - Topology implementation
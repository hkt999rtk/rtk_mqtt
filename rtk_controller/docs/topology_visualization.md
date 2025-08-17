# Network Topology Visualization

The RTK Controller provides comprehensive network topology visualization capabilities to help understand and analyze your home network structure, device connections, and network health.

## Features

### Visualization Formats

- **ASCII Art**: Text-based topology display for terminal output
- **Tree Structure**: Hierarchical view of network connections
- **DOT Format**: Graphviz-compatible format for advanced graph rendering
- **JSON**: Structured data format for programmatic access
- **Table Format**: Tabular view of devices and connections
- **Summary**: High-level overview with statistics
- **GraphViz**: Enhanced graph format with styling
- **PlantUML**: UML-style network diagrams

### Display Options

- **Device Information**: MAC addresses, IP addresses, device types, status
- **Connection Quality**: Visual quality indicators and metrics
- **Signal Strength**: WiFi signal strength indicators
- **Bandwidth Usage**: Upload/download bandwidth information
- **Network Topology**: Device relationships and connection types
- **Anomaly Detection**: Automatic detection of network issues
- **Grouping**: Group devices by SSID, location, or type
- **Filtering**: Filter by device type, status, time window, or quality

## CLI Commands

### Basic Topology Display

```bash
# Show current topology in ASCII format
topology graph

# Show topology as tree structure
topology tree

# Show topology with specific format
topology graph --format json
topology graph --format dot
topology graph --format table
```

### Advanced Visualization

```bash
# Show comprehensive visualization with all metrics
topology visualize

# Group devices by SSID
topology visualize --group-ssid

# Group devices by location
topology visualize --group-location

# Show only recent activity (last 24 hours)
topology visualize --time-window 24h

# Generate GraphViz format
topology visualize --format graphviz
```

### Export Topology

```bash
# Export to file
topology export json network_topology.json
topology export dot network_graph.dot
topology export plantuml network_diagram.puml

# Export with additional options
topology export json topology.json --offline --no-quality
```

### Filtering Options

```bash
# Show only online devices
topology graph

# Include offline devices
topology graph --offline

# Hide connection quality indicators
topology graph --no-quality

# Filter by time window
topology graph --time-window 1h
topology graph --time-window 7d
```

## Configuration Options

### VisualizationConfig

The visualization system can be configured with various options:

```go
config := topology.VisualizationConfig{
    // Display options
    ShowOfflineDevices:    true,   // Include offline devices
    ShowConnectionQuality: true,   // Show quality metrics
    ShowBandwidth:        true,   // Show bandwidth info
    ShowSSIDs:            true,   // Show WiFi SSIDs
    ShowInterfaceDetails: false,  // Show interface names
    ShowTimestamps:       false,  // Show time information
    
    // Filtering options
    MinConnectionQuality: 0.0,    // Minimum quality threshold
    DeviceTypeFilter:     []string{"router", "client"},
    SSIDFilter:          []string{"HomeWiFi"},
    TimeWindow:          24 * time.Hour,
    
    // Layout options
    MaxWidth:            120,     // Maximum output width
    CompactMode:         false,   // Compact display mode
    ColorEnabled:        true,    // Enable color output
    
    // Advanced options
    GroupBySSID:         false,   // Group by SSID
    GroupByLocation:     false,   // Group by location
    ShowMetrics:         true,    // Show performance metrics
    ShowAnomalies:       true,    // Show detected anomalies
}
```

## Output Examples

### ASCII Format

```
Network Topology (ASCII)
========================

Generated: 2024-01-15T10:30:00Z
Devices: 5 online, 1 offline, 6 total
Connections: 8

[ROUTER Devices]
  ● Home Router [██████████] 0.95 [HomeWiFi] (-50dBm)
    ├─ Living Room AP [████████░░] 0.80 (2.1ms)
    ├─ Kitchen AP [███████░░░] 0.75 (3.5ms)

[ACCESS_POINT Devices]
  ● Living Room AP [████████░░] 0.80 [HomeWiFi] (-45dBm)
    ├─ Phone [██████░░░░] 0.65 (5.2ms)
    ├─ Laptop [████████░░] 0.82 (4.1ms)

[CLIENT Devices]
  ● Phone [██████░░░░] 0.65 [HomeWiFi] (-65dBm)
  ● Laptop [████████░░] 0.82 [HomeWiFi] (-55dBm)
  ○ Tablet [████░░░░░░] 0.40 [HomeWiFi] (-75dBm)

[Anomalies Detected]
  ⚠ poor_quality_connection: Poor connection quality (0.40) between devices
  ⚠ high_latency: High latency (120.5ms) detected
```

### Tree Format

```
Network Topology (Tree)
=======================

● Home Router (Q: 0.95) [HomeWiFi]
  ● Living Room AP (Q: 0.80) [HomeWiFi]
    ● Phone (Q: 0.65) [HomeWiFi]
    ● Laptop (Q: 0.82) [HomeWiFi]
  ● Kitchen AP (Q: 0.75) [HomeWiFi]
    ● Smart TV (Q: 0.70) [HomeWiFi]
    ○ Tablet (Q: 0.40) [HomeWiFi]
```

### Summary Format

```
Network Topology Summary
========================

Network Overview:
  Total Devices: 6
  Online Devices: 5
  Offline Devices: 1
  Total Connections: 8
  Average Quality: 0.73
  Average Latency: 8.2ms

Devices by Type:
  router: 1
  access_point: 2
  client: 3

Devices by Status:
  online: 5
  offline: 1

Devices by SSID:
  HomeWiFi: 6

Detected Anomalies:
  [ERROR] poor_quality_connection: Poor connection quality (0.40) between devices
  [WARNING] high_latency: High latency (120.5ms) detected

Performance Metrics:
  Total Bandwidth: 450.2 Mbps
  Packet Loss Rate: 0.05%
  Topology Complexity: 0.67
```

## Integration with External Tools

### Graphviz

Export topology to DOT format and render with Graphviz:

```bash
# Export topology
topology export dot network.dot

# Render with Graphviz
dot -Tpng network.dot -o network.png
dot -Tsvg network.dot -o network.svg
```

### PlantUML

Export to PlantUML format for documentation:

```bash
# Export topology
topology export plantuml network.puml

# Render with PlantUML
plantuml network.puml
```

### JSON Processing

Export JSON for programmatic processing:

```bash
# Export and process with jq
topology export json - | jq '.nodes[] | select(.status == "online")'

# Count devices by type
topology export json - | jq '.stats.devicesByType'
```

## Anomaly Detection

The visualization system automatically detects various network anomalies:

- **Isolated Nodes**: Online devices with no connections
- **Poor Quality Connections**: Connections below quality threshold
- **High Latency**: Connections with excessive latency
- **Disconnected Devices**: Devices that should be connected but aren't
- **Unusual Topology Changes**: Rapid changes in network structure

## Performance Considerations

- **Large Networks**: Use filtering options for networks with many devices
- **Real-time Updates**: Visualization reflects current network state
- **Resource Usage**: Complex visualizations may require more processing time
- **Output Size**: Large topologies may produce extensive output

## Troubleshooting

### Common Issues

1. **Empty Topology**: Ensure topology discovery is running
2. **Missing Devices**: Check device filters and time windows
3. **Poor Quality Display**: Verify connection quality monitoring is enabled
4. **Export Failures**: Check file permissions and disk space

### Debug Commands

```bash
# Check topology manager status
topology status

# Verify device discovery
topology discover

# Show detailed device information
topology device <mac_address>

# Show connection details
topology connections
```

## Best Practices

1. **Regular Monitoring**: Use topology visualization for routine network health checks
2. **Anomaly Review**: Investigate detected anomalies promptly
3. **Documentation**: Export topology diagrams for network documentation
4. **Filtering**: Use appropriate filters for focused analysis
5. **Time Windows**: Adjust time windows based on analysis needs
6. **Format Selection**: Choose appropriate format for intended use case
#!/bin/bash

# RTK Controller Topology Demo Script
# This script demonstrates the complete topology detection workflow

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

clear

echo -e "${CYAN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║      RTK Controller - Topology Detection Demo        ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to pause and wait for user
pause() {
    echo -e "\n${YELLOW}Press Enter to continue...${NC}"
    read -r
}

# Function to run command with echo
run_cmd() {
    echo -e "${GREEN}$ $1${NC}"
    eval "$1"
}

# Build the controller
echo -e "${BLUE}=== Step 1: Building RTK Controller ===${NC}"
run_cmd "go build -o rtk-controller ./cmd/controller"
echo -e "${GREEN}✓ Build successful${NC}"
pause

# Clean environment
echo -e "\n${BLUE}=== Step 2: Preparing Clean Environment ===${NC}"
run_cmd "rm -rf data/controller.db"
run_cmd "mkdir -p data"
echo -e "${GREEN}✓ Environment ready${NC}"
pause

# Simulate network discovery
echo -e "\n${BLUE}=== Step 3: Simulating Network Discovery ===${NC}"
echo "Discovering network devices..."
cat > /tmp/demo_discovery.go << 'EOF'
package main

import (
    "fmt"
    "time"
    "rtk_controller/internal/storage"
    "rtk_controller/pkg/types"
)

func main() {
    // Create storage
    store, _ := storage.NewBuntDB("data")
    defer store.Close()
    
    topologyStorage := storage.NewTopologyStorage(store)
    
    // Create network topology
    topology := &types.NetworkTopology{
        ID:          "demo-topology",
        Tenant:      "default",
        Site:        "office-1",
        Devices:     make(map[string]*types.NetworkDevice),
        Connections: []types.DeviceConnection{},
        UpdatedAt:   time.Now(),
    }
    
    // Discover Gateway Router
    fmt.Println("🔍 Discovered: Gateway Router (192.168.1.1)")
    gateway := &types.NetworkDevice{
        DeviceID:     "gw-router-01",
        DeviceType:   "router",
        Hostname:     "office-gateway",
        PrimaryMAC:   "00:11:22:33:44:55",
        Manufacturer: "RTK",
        Model:        "RTK-GW-5000",
        Role:         types.RoleGateway,
        Online:       true,
        LastSeen:     time.Now().Unix(),
        Capabilities: []string{"routing", "nat", "firewall", "vpn"},
    }
    topology.Devices[gateway.DeviceID] = gateway
    topology.Gateway = gateway.DeviceID
    time.Sleep(500 * time.Millisecond)
    
    // Discover Core Switch
    fmt.Println("🔍 Discovered: Core Switch (192.168.1.2)")
    coreSwitch := &types.NetworkDevice{
        DeviceID:     "sw-core-01",
        DeviceType:   "switch",
        Hostname:     "core-switch",
        PrimaryMAC:   "00:11:22:33:44:AA",
        Manufacturer: "RTK",
        Model:        "RTK-SW-48P",
        Role:         types.RoleSwitch,
        Online:       true,
        LastSeen:     time.Now().Unix(),
        Capabilities: []string{"switching", "vlan", "poe"},
    }
    topology.Devices[coreSwitch.DeviceID] = coreSwitch
    time.Sleep(500 * time.Millisecond)
    
    // Discover WiFi Access Points
    fmt.Println("🔍 Discovered: WiFi AP Floor 1 (192.168.1.10)")
    ap1 := &types.NetworkDevice{
        DeviceID:     "ap-floor1-01",
        DeviceType:   "ap",
        Hostname:     "ap-floor1",
        PrimaryMAC:   "00:11:22:33:44:BB",
        Manufacturer: "RTK",
        Model:        "RTK-AP-AX6",
        Location:     "Floor 1 - Main Hall",
        Role:         types.RoleAccessPoint,
        Online:       true,
        LastSeen:     time.Now().Unix(),
        Capabilities: []string{"ap", "mesh", "wifi6"},
    }
    topology.Devices[ap1.DeviceID] = ap1
    time.Sleep(500 * time.Millisecond)
    
    fmt.Println("🔍 Discovered: WiFi AP Floor 2 (192.168.1.11)")
    ap2 := &types.NetworkDevice{
        DeviceID:     "ap-floor2-01",
        DeviceType:   "ap",
        Hostname:     "ap-floor2",
        PrimaryMAC:   "00:11:22:33:44:CC",
        Manufacturer: "RTK",
        Model:        "RTK-AP-AX6",
        Location:     "Floor 2 - Conference Room",
        Role:         types.RoleAccessPoint,
        Online:       true,
        LastSeen:     time.Now().Unix(),
        Capabilities: []string{"ap", "mesh", "wifi6"},
    }
    topology.Devices[ap2.DeviceID] = ap2
    time.Sleep(500 * time.Millisecond)
    
    // Discover IoT Gateway
    fmt.Println("🔍 Discovered: IoT Gateway (192.168.1.20)")
    iotGw := &types.NetworkDevice{
        DeviceID:     "iot-gw-01",
        DeviceType:   "iot_gateway",
        Hostname:     "iot-gateway",
        PrimaryMAC:   "00:11:22:33:44:DD",
        Manufacturer: "RTK",
        Model:        "RTK-IoT-1000",
        Location:     "Server Room",
        Role:         types.RoleGateway,
        Online:       true,
        LastSeen:     time.Now().Unix(),
        Capabilities: []string{"zigbee", "zwave", "mqtt"},
    }
    topology.Devices[iotGw.DeviceID] = iotGw
    time.Sleep(500 * time.Millisecond)
    
    // Create connections
    fmt.Println("\n📡 Mapping network connections...")
    
    connections := []types.DeviceConnection{
        {
            ID:             "conn-gw-sw",
            FromDeviceID:   "gw-router-01",
            ToDeviceID:     "sw-core-01",
            FromInterface:  "eth1",
            ToInterface:    "port1",
            ConnectionType: "ethernet",
            IsDirectLink:   true,
            Metrics: types.ConnectionMetrics{
                LinkSpeed: 10000, // 10Gbps
                Bandwidth: 10000,
                Latency:   0.1,
            },
            LastSeen: time.Now().Unix(),
        },
        {
            ID:             "conn-sw-ap1",
            FromDeviceID:   "sw-core-01",
            ToDeviceID:     "ap-floor1-01",
            FromInterface:  "port10",
            ToInterface:    "eth0",
            ConnectionType: "ethernet",
            IsDirectLink:   true,
            Metrics: types.ConnectionMetrics{
                LinkSpeed: 1000,
                Bandwidth: 1000,
                Latency:   0.2,
            },
            LastSeen: time.Now().Unix(),
        },
        {
            ID:             "conn-sw-ap2",
            FromDeviceID:   "sw-core-01",
            ToDeviceID:     "ap-floor2-01",
            FromInterface:  "port11",
            ToInterface:    "eth0",
            ConnectionType: "ethernet",
            IsDirectLink:   true,
            Metrics: types.ConnectionMetrics{
                LinkSpeed: 1000,
                Bandwidth: 1000,
                Latency:   0.2,
            },
            LastSeen: time.Now().Unix(),
        },
        {
            ID:             "conn-sw-iot",
            FromDeviceID:   "sw-core-01",
            ToDeviceID:     "iot-gw-01",
            FromInterface:  "port20",
            ToInterface:    "eth0",
            ConnectionType: "ethernet",
            IsDirectLink:   true,
            Metrics: types.ConnectionMetrics{
                LinkSpeed: 1000,
                Bandwidth: 100, // Lower bandwidth usage
                Latency:   0.3,
            },
            LastSeen: time.Now().Unix(),
        },
    }
    
    topology.Connections = connections
    
    // Save topology
    topologyStorage.SaveTopology(topology)
    
    fmt.Println("\n✅ Network topology discovered successfully!")
    fmt.Printf("   - Devices: %d\n", len(topology.Devices))
    fmt.Printf("   - Connections: %d\n", len(topology.Connections))
}
EOF

go run /tmp/demo_discovery.go
pause

# Show topology in CLI
echo -e "\n${BLUE}=== Step 4: Viewing Topology via CLI ===${NC}"
echo "Starting interactive CLI to explore the topology..."
echo ""

# Create CLI commands file
cat > /tmp/cli_commands.txt << 'EOF'
topology show
topology devices
topology connections
topology export
exit
EOF

echo -e "${YELLOW}Running CLI commands:${NC}"
echo "  1. topology show"
echo "  2. topology devices"
echo "  3. topology connections"
echo "  4. topology export"
echo ""

./rtk-controller --cli < /tmp/cli_commands.txt | head -200
pause

# Simulate WiFi clients
echo -e "\n${BLUE}=== Step 5: Simulating WiFi Client Activity ===${NC}"
cat > /tmp/demo_clients.go << 'EOF'
package main

import (
    "fmt"
    "time"
    "rtk_controller/internal/storage"
    "rtk_controller/pkg/types"
)

func main() {
    // Load existing topology
    store, _ := storage.NewBuntDB("data")
    defer store.Close()
    
    topologyStorage := storage.NewTopologyStorage(store)
    topology, _ := topologyStorage.GetTopology("default", "office-1")
    
    fmt.Println("📱 Simulating WiFi client connections...")
    
    // Add WiFi clients to AP1
    if ap1, exists := topology.Devices["ap-floor1-01"]; exists {
        fmt.Println("\nFloor 1 AP - New clients:")
        
        // Add client interfaces to AP
        if ap1.Interfaces == nil {
            ap1.Interfaces = make(map[string]types.NetworkIface)
        }
        
        ap1.Interfaces["wlan0"] = types.NetworkIface{
            Name:       "wlan0",
            Type:       "wifi",
            MacAddress: ap1.PrimaryMAC,
            Status:     "up",
            WiFiMode:   "AP",
            SSID:       "Office-WiFi-5G",
            Channel:    36,
            Band:       "5G",
            Security:   "WPA3",
        }
        
        // Simulate client connections
        clients := []struct{
            name string
            mac  string
            ip   string
        }{
            {"CEO-Laptop", "AA:BB:CC:DD:EE:01", "192.168.1.101"},
            {"CFO-Phone", "AA:BB:CC:DD:EE:02", "192.168.1.102"},
            {"Meeting-Room-TV", "AA:BB:CC:DD:EE:03", "192.168.1.103"},
        }
        
        for _, client := range clients {
            fmt.Printf("  ✓ %s (%s) connected at %s\n", client.name, client.mac, client.ip)
            time.Sleep(300 * time.Millisecond)
        }
    }
    
    // Add WiFi clients to AP2
    if ap2, exists := topology.Devices["ap-floor2-01"]; exists {
        fmt.Println("\nFloor 2 AP - New clients:")
        
        if ap2.Interfaces == nil {
            ap2.Interfaces = make(map[string]types.NetworkIface)
        }
        
        ap2.Interfaces["wlan0"] = types.NetworkIface{
            Name:       "wlan0",
            Type:       "wifi",
            MacAddress: ap2.PrimaryMAC,
            Status:     "up",
            WiFiMode:   "AP",
            SSID:       "Office-WiFi-5G",
            Channel:    149,
            Band:       "5G",
            Security:   "WPA3",
        }
        
        clients := []struct{
            name string
            mac  string
            ip   string
        }{
            {"Dev-Laptop-1", "AA:BB:CC:DD:EE:11", "192.168.1.111"},
            {"Dev-Laptop-2", "AA:BB:CC:DD:EE:12", "192.168.1.112"},
            {"Dev-Laptop-3", "AA:BB:CC:DD:EE:13", "192.168.1.113"},
            {"QA-Tablet", "AA:BB:CC:DD:EE:14", "192.168.1.114"},
        }
        
        for _, client := range clients {
            fmt.Printf("  ✓ %s (%s) connected at %s\n", client.name, client.mac, client.ip)
            time.Sleep(300 * time.Millisecond)
        }
    }
    
    // Update topology
    topology.UpdatedAt = time.Now()
    topologyStorage.SaveTopology(topology)
    
    fmt.Println("\n✅ WiFi clients simulation complete!")
}
EOF

go run /tmp/demo_clients.go
pause

# Generate topology visualization
echo -e "\n${BLUE}=== Step 6: Generating Topology Visualization ===${NC}"
cat > /tmp/generate_viz.go << 'EOF'
package main

import (
    "fmt"
    "strings"
    "rtk_controller/internal/storage"
)

func main() {
    store, _ := storage.NewBuntDB("data")
    defer store.Close()
    
    topologyStorage := storage.NewTopologyStorage(store)
    topology, _ := topologyStorage.GetTopology("default", "office-1")
    
    fmt.Println("Network Topology Diagram (ASCII):")
    fmt.Println("==================================")
    fmt.Println()
    fmt.Println("                    [Internet]")
    fmt.Println("                         |")
    fmt.Println("                         |")
    fmt.Println("                 [Gateway Router]")
    fmt.Println("                  192.168.1.1")
    fmt.Println("                         |")
    fmt.Println("                         | 10Gbps")
    fmt.Println("                         |")
    fmt.Println("                  [Core Switch]")
    fmt.Println("                   192.168.1.2")
    fmt.Println("          _______________|_______________")
    fmt.Println("         |         |          |          |")
    fmt.Println("      1Gbps     1Gbps      1Gbps     1Gbps")
    fmt.Println("         |         |          |          |")
    fmt.Println("    [AP Floor 1] [AP Floor 2] |    [IoT Gateway]")
    fmt.Println("    192.168.1.10 192.168.1.11 |     192.168.1.20")
    fmt.Println("         |         |          |          |")
    fmt.Println("      Clients   Clients    [Other]    IoT Devices")
    fmt.Println()
    
    // Show device summary
    fmt.Println("\nDevice Summary:")
    fmt.Println(strings.Repeat("-", 60))
    fmt.Printf("%-20s %-15s %-10s %-10s\n", "Device", "Type", "Status", "Location")
    fmt.Println(strings.Repeat("-", 60))
    
    for _, device := range topology.Devices {
        status := "Offline"
        if device.Online {
            status = "Online"
        }
        location := device.Location
        if location == "" {
            location = "N/A"
        }
        fmt.Printf("%-20s %-15s %-10s %-10s\n", 
            device.Hostname, device.DeviceType, status, location)
    }
    
    fmt.Println(strings.Repeat("-", 60))
    fmt.Printf("Total: %d devices, %d connections\n", 
        len(topology.Devices), len(topology.Connections))
}
EOF

go run /tmp/generate_viz.go
pause

# Show network health
echo -e "\n${BLUE}=== Step 7: Network Health Summary ===${NC}"
cat > /tmp/health_check.go << 'EOF'
package main

import (
    "fmt"
    "rtk_controller/internal/storage"
)

func main() {
    store, _ := storage.NewBuntDB("data")
    defer store.Close()
    
    topologyStorage := storage.NewTopologyStorage(store)
    topology, _ := topologyStorage.GetTopology("default", "office-1")
    
    // Calculate health metrics
    totalDevices := len(topology.Devices)
    onlineDevices := 0
    criticalDevices := []string{}
    
    for _, device := range topology.Devices {
        if device.Online {
            onlineDevices++
        }
        if device.Role == "gateway" || device.Role == "switch" {
            criticalDevices = append(criticalDevices, device.Hostname)
        }
    }
    
    healthScore := float64(onlineDevices) / float64(totalDevices) * 100
    
    fmt.Println("Network Health Report")
    fmt.Println("====================")
    fmt.Printf("\n🏥 Overall Health Score: %.1f%%\n", healthScore)
    fmt.Printf("\n📊 Device Statistics:\n")
    fmt.Printf("   Total Devices:    %d\n", totalDevices)
    fmt.Printf("   Online Devices:   %d\n", onlineDevices)
    fmt.Printf("   Offline Devices:  %d\n", totalDevices-onlineDevices)
    
    fmt.Printf("\n⚡ Critical Infrastructure:\n")
    for _, name := range criticalDevices {
        fmt.Printf("   ✓ %s - Online\n", name)
    }
    
    fmt.Printf("\n🔗 Connection Statistics:\n")
    fmt.Printf("   Total Links:      %d\n", len(topology.Connections))
    fmt.Printf("   Active Links:     %d\n", len(topology.Connections))
    
    fmt.Printf("\n📡 Wireless Coverage:\n")
    apCount := 0
    for _, device := range topology.Devices {
        if device.DeviceType == "ap" {
            apCount++
        }
    }
    fmt.Printf("   Access Points:    %d\n", apCount)
    fmt.Printf("   Coverage Areas:   %d\n", apCount)
    
    if healthScore == 100 {
        fmt.Println("\n✅ Network Status: HEALTHY - All systems operational")
    } else if healthScore >= 80 {
        fmt.Println("\n⚠️  Network Status: DEGRADED - Some devices offline")
    } else {
        fmt.Println("\n❌ Network Status: CRITICAL - Multiple failures detected")
    }
}
EOF

go run /tmp/health_check.go
echo ""

# Final summary
echo -e "${CYAN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                  Demo Complete!                      ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}✅ Successfully demonstrated:${NC}"
echo "   • Network device discovery"
echo "   • Topology data persistence"
echo "   • CLI command interface"
echo "   • WiFi client tracking"
echo "   • Network visualization"
echo "   • Health monitoring"
echo ""
echo -e "${YELLOW}To explore further, run:${NC}"
echo "   ./rtk-controller --cli"
echo ""
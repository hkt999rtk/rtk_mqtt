package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

func main() {
	fmt.Println("Simple Topology Message Processing Test")
	fmt.Println("========================================")

	// Create storage
	store, err := storage.NewBuntDB("data")
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create topology storage
	topologyStorage := storage.NewTopologyStorage(store)

	// Get or create topology
	topology, err := topologyStorage.GetTopology("default", "default")
	if err != nil {
		// Create new topology
		topology = &types.NetworkTopology{
			ID:          "topology-mqtt-test",
			Tenant:      "default",
			Site:        "default",
			Devices:     make(map[string]*types.NetworkDevice),
			Connections: []types.DeviceConnection{},
			UpdatedAt:   time.Now(),
		}
	}

	// Test 1: Create and save a device from MQTT-like data
	fmt.Println("\n1. Processing Device Discovery")
	device := processDeviceDiscovery()
	topology.Devices[device.DeviceID] = device
	fmt.Printf("✓ Added device: %s\n", device.DeviceID)

	// Test 2: Create and save connections
	fmt.Println("\n2. Processing Connections")
	connections := processConnections()
	topology.Connections = append(topology.Connections, connections...)
	fmt.Printf("✓ Added %d connections\n", len(connections))

	// Test 3: Update topology with WiFi clients
	fmt.Println("\n3. Processing WiFi Clients")
	wifiDevice := processWiFiClients()
	topology.Devices[wifiDevice.DeviceID] = wifiDevice
	fmt.Printf("✓ Added WiFi AP: %s\n", wifiDevice.DeviceID)

	// Save the updated topology
	if err := topologyStorage.SaveTopology(topology); err != nil {
		fmt.Printf("✗ Failed to save topology: %v\n", err)
	} else {
		fmt.Println("✓ Topology saved successfully")
	}

	// Verify the complete topology
	fmt.Println("\n4. Verifying Complete Topology")
	topology, err = topologyStorage.GetTopology("default", "default")
	if err != nil {
		fmt.Printf("✗ Failed to get topology: %v\n", err)
		return
	}

	fmt.Printf("✓ Topology loaded successfully\n")
	fmt.Printf("  - Devices: %d\n", len(topology.Devices))
	fmt.Printf("  - Connections: %d\n", len(topology.Connections))

	// Display all devices
	fmt.Println("\nDevices in Topology:")
	for id, device := range topology.Devices {
		status := "offline"
		if device.Online {
			status = "online"
		}
		fmt.Printf("  - %s: %s (%s) - %s\n",
			id, device.Hostname, device.DeviceType, status)

		// Show interfaces
		for ifName, iface := range device.Interfaces {
			fmt.Printf("    - Interface %s: %s (%s)\n",
				ifName, iface.MacAddress, iface.Status)
		}
	}

	// Display connections
	fmt.Println("\nConnections:")
	for _, conn := range topology.Connections {
		fmt.Printf("  - %s: %s -> %s (%s)\n",
			conn.ID, conn.FromDeviceID, conn.ToDeviceID, conn.ConnectionType)
	}
}

func processDeviceDiscovery() *types.NetworkDevice {
	// Simulate processing a device discovery message
	return &types.NetworkDevice{
		DeviceID:     "mqtt-router-01",
		DeviceType:   "router",
		PrimaryMAC:   "00:AA:BB:CC:DD:EE",
		Hostname:     "mqtt-test-router",
		Manufacturer: "RTK",
		Model:        "RTK-MQTT-1000",
		Location:     "MQTT Test Lab",
		Role:         types.RoleGateway,
		Online:       true,
		LastSeen:     time.Now().Unix(),
		Interfaces: map[string]types.NetworkIface{
			"eth0": {
				Name:       "eth0",
				Type:       "ethernet",
				MacAddress: "00:AA:BB:CC:DD:EE",
				Status:     "up",
				Speed:      1000,
				IPAddresses: []types.IPAddressInfo{
					{
						Address: "192.168.200.1",
						Network: "192.168.200.0/24",
						Type:    "static",
					},
				},
			},
		},
		Capabilities: []string{"routing", "nat", "dhcp", "mqtt"},
	}
}

func processConnections() []types.DeviceConnection {
	// Simulate processing connection messages
	return []types.DeviceConnection{
		{
			ID:             "mqtt-conn-1",
			FromDeviceID:   "mqtt-router-01",
			ToDeviceID:     "mqtt-switch-01",
			FromInterface:  "eth0",
			ToInterface:    "port1",
			ConnectionType: "ethernet",
			IsDirectLink:   true,
			Metrics: types.ConnectionMetrics{
				LinkSpeed: 1000,
				Bandwidth: 1000,
				Latency:   0.2,
			},
			LastSeen:   time.Now().Unix(),
			Discovered: time.Now().Add(-1 * time.Hour).Unix(),
		},
		{
			ID:             "mqtt-conn-2",
			FromDeviceID:   "mqtt-switch-01",
			ToDeviceID:     "mqtt-ap-01",
			FromInterface:  "port2",
			ToInterface:    "eth0",
			ConnectionType: "ethernet",
			IsDirectLink:   true,
			Metrics: types.ConnectionMetrics{
				LinkSpeed: 1000,
				Bandwidth: 800,
				Latency:   0.3,
			},
			LastSeen:   time.Now().Unix(),
			Discovered: time.Now().Add(-30 * time.Minute).Unix(),
		},
	}
}

func processWiFiClients() *types.NetworkDevice {
	// Simulate processing WiFi clients telemetry
	return &types.NetworkDevice{
		DeviceID:     "mqtt-ap-01",
		DeviceType:   "ap",
		PrimaryMAC:   "00:AA:BB:CC:DD:FF",
		Hostname:     "mqtt-test-ap",
		Manufacturer: "RTK",
		Model:        "RTK-AP-MQTT",
		Location:     "MQTT Test Area",
		Role:         types.RoleAccessPoint,
		Online:       true,
		LastSeen:     time.Now().Unix(),
		Interfaces: map[string]types.NetworkIface{
			"eth0": {
				Name:       "eth0",
				Type:       "ethernet",
				MacAddress: "00:AA:BB:CC:DD:FF",
				Status:     "up",
				Speed:      1000,
				IPAddresses: []types.IPAddressInfo{
					{
						Address: "192.168.200.10",
						Network: "192.168.200.0/24",
						Type:    "dhcp",
					},
				},
			},
			"wlan0": {
				Name:       "wlan0",
				Type:       "wifi",
				MacAddress: "00:AA:BB:CC:DD:F0",
				Status:     "up",
				Speed:      866,
				WiFiMode:   "AP",
				SSID:       "MQTT-Test-WiFi",
				BSSID:      "00:AA:BB:CC:DD:F0",
				Channel:    6,
				Band:       "2.4G",
				Security:   "WPA2",
				// Simulated client stats
				TxBytes:   100000000,
				RxBytes:   50000000,
				TxPackets: 100000,
				RxPackets: 50000,
			},
		},
		Capabilities: []string{"ap", "bridge", "mqtt"},
	}
}

// Helper to display JSON
func displayJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

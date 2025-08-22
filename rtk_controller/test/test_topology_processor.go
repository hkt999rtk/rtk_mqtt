package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"rtk_controller/internal/mqtt"
	"rtk_controller/internal/schema"
	"rtk_controller/internal/storage"
)

func main() {
	// Create storage
	store, err := storage.NewBuntDB("data")
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create storages
	topologyStorage := storage.NewTopologyStorage(store)
	identityStorage := storage.NewIdentityStorage(store)

	// Create schema manager
	schemaConfig := schema.Config{
		Enabled:             false, // Disable for testing without schemas
		SchemaFiles:         []string{},
		StrictValidation:    false,
		LogValidationErrors: true,
		CacheResults:        true,
		CacheSize:           100,
		StoreResults:        false,
	}
	schemaManager, err := schema.NewManager(schemaConfig, store)
	if err != nil {
		log.Printf("Warning: Failed to create schema manager: %v", err)
		// Create a nil schema manager for testing
		schemaManager = nil
	}

	// Create topology processor
	processor := mqtt.NewTopologyProcessor(schemaManager, topologyStorage, identityStorage)

	fmt.Println("Testing MQTT Topology Processor")
	fmt.Println("================================")

	// Test 1: Process topology discovery message
	testTopologyDiscovery(processor)

	// Test 2: Process WiFi clients message
	testWiFiClients(processor)

	// Test 3: Process connections message
	testConnections(processor)

	// Verify the processed data
	verifyTopologyData(topologyStorage)
}

func testTopologyDiscovery(processor *mqtt.TopologyProcessor) {
	fmt.Println("\n1. Testing Topology Discovery Message")

	message := map[string]interface{}{
		"schema":    "topology.discovery.v1",
		"timestamp": time.Now().Unix() * 1000,
		"device_id": "test-router-01",
		"device_info": map[string]interface{}{
			"hostname":     "test-router",
			"manufacturer": "RTK",
			"model":        "RTK-TEST",
			"device_type":  "router",
			"location":     "Test Lab",
			"capabilities": []string{"routing", "nat"},
		},
		"interfaces": []map[string]interface{}{
			{
				"name":       "eth0",
				"type":       "ethernet",
				"mac":        "00:11:22:33:44:AA",
				"ip_address": "192.168.100.1",
				"netmask":    "255.255.255.0",
				"status":     "up",
				"speed":      1000,
			},
		},
	}

	payload, _ := json.Marshal(message)
	topic := "rtk/v1/default/default/test-router-01/topology/discovery"

	// Check if it's a topology message
	if processor.IsTopologyMessage(topic) {
		fmt.Println("✓ Message identified as topology message")

		// Process the message
		if err := processor.ProcessMessage(topic, payload); err != nil {
			fmt.Printf("✗ Failed to process message: %v\n", err)
		} else {
			fmt.Println("✓ Message processed successfully")
		}
	} else {
		fmt.Println("✗ Message not identified as topology message")
	}
}

func testWiFiClients(processor *mqtt.TopologyProcessor) {
	fmt.Println("\n2. Testing WiFi Clients Message")

	message := map[string]interface{}{
		"schema":    "telemetry.wifi_clients.v1",
		"timestamp": time.Now().Unix() * 1000,
		"device_id": "test-ap-01",
		"ssid":      "TEST-WiFi",
		"bssid":     "00:11:22:33:44:BB",
		"clients": []map[string]interface{}{
			{
				"mac_address":    "AA:BB:CC:DD:EE:FF",
				"ip_address":     "192.168.100.50",
				"hostname":       "test-client",
				"rssi":           -45,
				"tx_rate":        433.3,
				"rx_rate":        433.3,
				"connected_time": 3600,
			},
		},
	}

	payload, _ := json.Marshal(message)
	topic := "rtk/v1/default/default/test-ap-01/telemetry/wifi_clients"

	if processor.IsTopologyMessage(topic) {
		fmt.Println("✓ Message identified as topology message")

		if err := processor.ProcessMessage(topic, payload); err != nil {
			fmt.Printf("✗ Failed to process message: %v\n", err)
		} else {
			fmt.Println("✓ Message processed successfully")
		}
	} else {
		fmt.Println("✗ Message not identified as topology message")
	}
}

func testConnections(processor *mqtt.TopologyProcessor) {
	fmt.Println("\n3. Testing Connections Message")

	message := map[string]interface{}{
		"schema":    "topology.connections.v1",
		"timestamp": time.Now().Unix() * 1000,
		"device_id": "test-switch-01",
		"connections": []map[string]interface{}{
			{
				"interface":     "port1",
				"remote_mac":    "00:11:22:33:44:AA",
				"remote_device": "test-router-01",
				"link_type":     "ethernet",
				"link_speed":    1000,
				"link_status":   "up",
			},
			{
				"interface":     "port2",
				"remote_mac":    "00:11:22:33:44:BB",
				"remote_device": "test-ap-01",
				"link_type":     "ethernet",
				"link_speed":    1000,
				"link_status":   "up",
			},
		},
	}

	payload, _ := json.Marshal(message)
	topic := "rtk/v1/default/default/test-switch-01/topology/connections"

	if processor.IsTopologyMessage(topic) {
		fmt.Println("✓ Message identified as topology message")

		if err := processor.ProcessMessage(topic, payload); err != nil {
			fmt.Printf("✗ Failed to process message: %v\n", err)
		} else {
			fmt.Println("✓ Message processed successfully")
		}
	} else {
		fmt.Println("✗ Message not identified as topology message")
	}
}

func verifyTopologyData(topologyStorage *storage.TopologyStorage) {
	fmt.Println("\n4. Verifying Processed Data")

	// Get topology
	topology, err := topologyStorage.GetTopology("default", "default")
	if err != nil {
		fmt.Printf("✗ Failed to get topology: %v\n", err)
		return
	}

	fmt.Printf("✓ Topology loaded: %d devices, %d connections\n",
		len(topology.Devices), len(topology.Connections))

	// Check for test devices
	testDevices := []string{"test-router-01", "test-ap-01", "test-switch-01"}
	for _, deviceID := range testDevices {
		if device, exists := topology.Devices[deviceID]; exists {
			fmt.Printf("✓ Found device: %s (%s)\n", deviceID, device.DeviceType)
		} else {
			fmt.Printf("✗ Device not found: %s\n", deviceID)
		}
	}

	// Display summary
	fmt.Println("\nTopology Summary:")
	fmt.Printf("- Total Devices: %d\n", len(topology.Devices))
	fmt.Printf("- Total Connections: %d\n", len(topology.Connections))

	for id, device := range topology.Devices {
		status := "offline"
		if device.Online {
			status = "online"
		}
		fmt.Printf("  - %s: %s (%s) - %s\n",
			id, device.Hostname, device.DeviceType, status)
	}
}

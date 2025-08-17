package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

func main() {
	// Read sample topology file
	data, err := ioutil.ReadFile("test/sample_topology.json")
	if err != nil {
		log.Fatalf("Failed to read topology file: %v", err)
	}

	// Parse topology
	var topology types.NetworkTopology
	if err := json.Unmarshal(data, &topology); err != nil {
		log.Fatalf("Failed to unmarshal topology: %v", err)
	}

	// Create storage
	store, err := storage.NewBuntDB("data")
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create topology storage
	topologyStorage := storage.NewTopologyStorage(store)

	// Save topology
	if err := topologyStorage.SaveTopology(&topology); err != nil {
		log.Fatalf("Failed to save topology: %v", err)
	}

	fmt.Println("Topology loaded successfully!")
	fmt.Printf("Loaded %d devices and %d connections\n", len(topology.Devices), len(topology.Connections))

	// Verify by reading back
	loaded, err := topologyStorage.GetTopology(topology.Tenant, topology.Site)
	if err != nil {
		log.Fatalf("Failed to load topology: %v", err)
	}

	fmt.Printf("\nVerification:\n")
	fmt.Printf("- Tenant: %s\n", loaded.Tenant)
	fmt.Printf("- Site: %s\n", loaded.Site)
	fmt.Printf("- Devices: %d\n", len(loaded.Devices))
	fmt.Printf("- Connections: %d\n", len(loaded.Connections))
	
	if loaded.Gateway != nil {
		fmt.Printf("- Gateway: %s (%s)\n", loaded.Gateway.DeviceID, loaded.Gateway.IPAddress)
	}

	// List devices
	fmt.Printf("\nDevices:\n")
	for id, device := range loaded.Devices {
		status := "offline"
		if device.Online {
			status = "online"
		}
		fmt.Printf("  - %s: %s (%s) - %s\n", id, device.Hostname, device.DeviceType, status)
	}

	// Test device queries
	fmt.Printf("\nDevice Queries:\n")
	
	// Check device count
	fmt.Printf("  - Total devices in topology: %d\n", len(loaded.Devices))
	
	// Check gateway
	if device, ok := loaded.Devices["gateway-01"]; ok {
		fmt.Printf("  - Found gateway device: %s (%s)\n", device.DeviceID, device.Hostname)
	}
	
	// Check connections
	fmt.Printf("  - Total connections: %d\n", len(loaded.Connections))

	fmt.Println("\nTopology data loaded and verified successfully!")
}
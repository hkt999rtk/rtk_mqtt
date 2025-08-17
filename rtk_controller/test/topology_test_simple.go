package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"rtk_controller/pkg/types"
)

func main() {
	// Create simple sample topology data
	topology := generateSimpleTopology()
	
	// Convert to JSON
	data, err := json.MarshalIndent(topology, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal topology: %v", err)
	}
	
	fmt.Println(string(data))
}

func generateSimpleTopology() *types.NetworkTopology {
	now := time.Now()
	
	// Create sample devices
	devices := map[string]*types.NetworkDevice{
		"gateway-01": {
			DeviceID:     "gateway-01",
			DeviceType:   "router",
			PrimaryMAC:   "00:11:22:33:44:55",
			Hostname:     "rtk-gateway-01",
			Manufacturer: "RTK",
			Model:        "RTK-GW1000",
			Location:     "Server Room",
			Role:         types.RoleGateway,
			Online:       true,
			LastSeen:     now.Unix(),
			Interfaces: map[string]types.NetworkIface{
				"eth0": {
					Name:       "eth0",
					Type:       "ethernet",
					MacAddress: "00:11:22:33:44:55",
					Status:     "up",
					Speed:      1000,
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.1.1",
							Network: "192.168.1.0/24",
							Type:    "static",
						},
					},
				},
			},
			Capabilities: []string{"routing", "nat", "dhcp"},
		},
		"ap-01": {
			DeviceID:     "ap-01",
			DeviceType:   "ap",
			PrimaryMAC:   "00:11:22:33:44:66",
			Hostname:     "rtk-ap-01",
			Manufacturer: "RTK",
			Model:        "RTK-AP500",
			Location:     "Office Area A",
			Role:         types.RoleAccessPoint,
			Online:       true,
			LastSeen:     now.Unix(),
			Interfaces: map[string]types.NetworkIface{
				"eth0": {
					Name:       "eth0",
					Type:       "ethernet",
					MacAddress: "00:11:22:33:44:66",
					Status:     "up",
					Speed:      1000,
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.1.10",
							Network: "192.168.1.0/24",
							Type:    "dhcp",
						},
					},
				},
				"wlan0": {
					Name:       "wlan0",
					Type:       "wifi",
					MacAddress: "00:11:22:33:44:67",
					Status:     "up",
					Speed:      866,
					WiFiMode:   "AP",
					SSID:       "RTK-Office",
					BSSID:      "00:11:22:33:44:67",
					Channel:    6,
					Band:       "2.4G",
					Security:   "WPA2",
				},
			},
			Capabilities: []string{"ap", "bridge"},
		},
		"client-01": {
			DeviceID:     "client-01",
			DeviceType:   "client",
			PrimaryMAC:   "AA:BB:CC:DD:EE:01",
			Hostname:     "laptop-01",
			Manufacturer: "Dell",
			Model:        "Latitude",
			Location:     "Office",
			Role:         types.RoleClient,
			Online:       true,
			LastSeen:     now.Unix(),
			Interfaces: map[string]types.NetworkIface{
				"wlan0": {
					Name:       "wlan0",
					Type:       "wifi",
					MacAddress: "AA:BB:CC:DD:EE:01",
					Status:     "up",
					Speed:      433,
					WiFiMode:   "STA",
					SSID:       "RTK-Office",
					BSSID:      "00:11:22:33:44:67",
					Channel:    6,
					Band:       "2.4G",
					RSSI:       -45,
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.1.101",
							Network: "192.168.1.0/24",
							Type:    "dhcp",
						},
					},
				},
			},
			Capabilities: []string{"client"},
		},
	}
	
	// Create connections
	connections := []types.DeviceConnection{
		{
			ID:             "conn-1",
			FromDeviceID:   "gateway-01",
			ToDeviceID:     "ap-01",
			FromInterface:  "eth0",
			ToInterface:    "eth0",
			ConnectionType: "ethernet",
			IsDirectLink:   true,
			Metrics: types.ConnectionMetrics{
				LinkSpeed: 1000,
				Bandwidth: 1000,
				Latency:   0.5,
			},
			LastSeen:   now.Unix(),
			Discovered: now.Add(-24 * time.Hour).Unix(),
		},
		{
			ID:             "conn-2",
			FromDeviceID:   "client-01",
			ToDeviceID:     "ap-01",
			FromInterface:  "wlan0",
			ToInterface:    "wlan0",
			ConnectionType: "wireless",
			IsDirectLink:   true,
			Metrics: types.ConnectionMetrics{
				RSSI:      -45,
				LinkSpeed: 433,
				Bandwidth: 200,
				Latency:   2.5,
			},
			LastSeen:   now.Unix(),
			Discovered: now.Add(-2 * time.Hour).Unix(),
		},
	}
	
	// Create gateway info
	gateway := &types.GatewayInfo{
		DeviceID:   "gateway-01",
		IPAddress:  "192.168.1.1",
		ExternalIP: "203.0.113.1",
		ISPInfo:    "Demo ISP",
		DNSServers: []string{"8.8.8.8", "8.8.4.4"},
	}
	
	return &types.NetworkTopology{
		ID:          "topology-default",
		Tenant:      "default",
		Site:        "default",
		Devices:     devices,
		Connections: connections,
		Gateway:     gateway,
		UpdatedAt:   now,
	}
}
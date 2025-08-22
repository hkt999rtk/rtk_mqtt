package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"rtk_controller/pkg/types"
)

func main() {
	// Create sample topology data
	topology := generateSampleTopology()

	// Convert to JSON
	data, err := json.MarshalIndent(topology, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal topology: %v", err)
	}

	fmt.Println(string(data))
}

func generateSampleTopology() *types.NetworkTopology {
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
					Duplex:     "full",
					MTU:        1500,
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.1.1",
							Network: "192.168.1.0/24",
							Type:    "ipv4",
						},
					},
					TxBytes:    1024000000,
					RxBytes:    2048000000,
					LastUpdate: now.Unix(),
				},
				"wlan0": {
					Name:       "wlan0",
					Type:       "wifi",
					MacAddress: "00:11:22:33:44:56",
					Status:     "up",
					Speed:      866,
					MTU:        1500,
					WiFiMode:   "AP",
					SSID:       "RTK-Gateway",
					Channel:    1,
					Band:       "5G",
					Security:   "WPA2",
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.10.1",
							Network: "192.168.1.0/24",
							Type:    "ipv4",
						},
					},
					LastUpdate: now.Unix(),
				},
			},
			Capabilities: []string{"routing", "nat", "dhcp", "firewall"},
			RoutingInfo: &types.RoutingInfo{
				RoutingTable: []types.RouteEntry{
					{
						Destination: "0.0.0.0/0",
						Gateway:     "10.0.0.1",
						Interface:   "eth1",
						Metric:      1,
					},
				},
				ForwardingEnabled: true,
			},
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
					Duplex:     "full",
					MTU:        1500,
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.1.10",
							Network: "192.168.1.0/24",
							Type:    "ipv4",
						},
					},
					LastUpdate: now.Unix(),
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
					LastUpdate: now.Unix(),
				},
				"wlan1": {
					Name:       "wlan1",
					Type:       "wifi",
					MacAddress: "00:11:22:33:44:68",
					Status:     "up",
					Speed:      1733,
					WiFiMode:   "AP",
					SSID:       "RTK-Office-5G",
					BSSID:      "00:11:22:33:44:68",
					Channel:    36,
					Band:       "5G",
					Security:   "WPA3",
					LastUpdate: now.Unix(),
				},
			},
			Capabilities: []string{"ap", "bridge"},
		},
		"ap-02": {
			DeviceID:     "ap-02",
			DeviceType:   "ap",
			PrimaryMAC:   "00:11:22:33:44:77",
			Hostname:     "rtk-ap-02",
			Manufacturer: "RTK",
			Model:        "RTK-AP500",
			Location:     "Office Area B",
			Role:         types.RoleAccessPoint,
			Online:       true,
			LastSeen:     now.Unix(),
			Interfaces: map[string]types.NetworkIface{
				"eth0": {
					Name:       "eth0",
					Type:       "ethernet",
					MacAddress: "00:11:22:33:44:77",
					Status:     "up",
					Speed:      1000,
					Duplex:     "full",
					MTU:        1500,
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.1.11",
							Network: "192.168.1.0/24",
							Type:    "ipv4",
						},
					},
					LastUpdate: now.Unix(),
				},
				"wlan0": {
					Name:       "wlan0",
					Type:       "wifi",
					MacAddress: "00:11:22:33:44:78",
					Status:     "up",
					Speed:      866,
					WiFiMode:   "AP",
					SSID:       "RTK-Office",
					BSSID:      "00:11:22:33:44:78",
					Channel:    11,
					Band:       "2.4G",
					Security:   "WPA2",
					LastUpdate: now.Unix(),
				},
			},
			Capabilities: []string{"ap", "bridge"},
		},
		"switch-01": {
			DeviceID:     "switch-01",
			DeviceType:   "switch",
			PrimaryMAC:   "00:11:22:33:44:88",
			Hostname:     "rtk-switch-01",
			Manufacturer: "RTK",
			Model:        "RTK-SW24",
			Location:     "Server Room",
			Role:         types.RoleSwitch,
			Online:       true,
			LastSeen:     now.Unix(),
			Interfaces: map[string]types.NetworkIface{
				"mgmt0": {
					Name:       "mgmt0",
					Type:       "ethernet",
					MacAddress: "00:11:22:33:44:88",
					Status:     "up",
					Speed:      1000,
					Duplex:     "full",
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.1.5",
							Network: "192.168.1.0/24",
							Type:    "ipv4",
						},
					},
					LastUpdate: now.Unix(),
				},
			},
			Capabilities: []string{"bridge", "vlan"},
		},
		"client-laptop-01": {
			DeviceID:     "client-laptop-01",
			DeviceType:   "client",
			PrimaryMAC:   "AA:BB:CC:DD:EE:01",
			Hostname:     "john-laptop",
			Manufacturer: "Dell",
			Model:        "Latitude 5520",
			Location:     "Office Area A",
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
					Security:   "WPA2",
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.1.101",
							Network: "192.168.1.0/24",
							Type:    "ipv4",
						},
					},
					TxBytes:    50000000,
					RxBytes:    100000000,
					LastUpdate: now.Unix(),
				},
			},
			Capabilities: []string{"client"},
		},
		"client-phone-01": {
			DeviceID:     "client-phone-01",
			DeviceType:   "client",
			PrimaryMAC:   "AA:BB:CC:DD:EE:02",
			Hostname:     "iPhone-14",
			Manufacturer: "Apple",
			Model:        "iPhone 14",
			Location:     "Office Area B",
			Role:         types.RoleClient,
			Online:       true,
			LastSeen:     now.Unix(),
			Interfaces: map[string]types.NetworkIface{
				"wlan0": {
					Name:       "wlan0",
					Type:       "wifi",
					MacAddress: "AA:BB:CC:DD:EE:02",
					Status:     "up",
					Speed:      173,
					WiFiMode:   "STA",
					SSID:       "RTK-Office",
					BSSID:      "00:11:22:33:44:78",
					Channel:    11,
					Band:       "2.4G",
					RSSI:       -52,
					Security:   "WPA2",
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.1.102",
							Network: "192.168.1.0/24",
							Type:    "ipv4",
						},
					},
					TxBytes:    10000000,
					RxBytes:    30000000,
					LastUpdate: now.Unix(),
				},
			},
			Capabilities: []string{"client"},
		},
		"client-iot-01": {
			DeviceID:     "client-iot-01",
			DeviceType:   "iot",
			PrimaryMAC:   "AA:BB:CC:DD:EE:03",
			Hostname:     "mi-smart-bulb",
			Manufacturer: "Xiaomi",
			Model:        "Mi Smart Bulb",
			Location:     "Office Area A",
			Role:         types.RoleClient,
			Online:       true,
			LastSeen:     now.Unix(),
			Interfaces: map[string]types.NetworkIface{
				"wlan0": {
					Name:       "wlan0",
					Type:       "wifi",
					MacAddress: "AA:BB:CC:DD:EE:03",
					Status:     "up",
					Speed:      72,
					WiFiMode:   "STA",
					SSID:       "RTK-Office",
					BSSID:      "00:11:22:33:44:67",
					Channel:    6,
					Band:       "2.4G",
					RSSI:       -68,
					Security:   "WPA2",
					IPAddresses: []types.IPAddressInfo{
						{
							Address: "192.168.1.103",
							Network: "192.168.1.0/24",
							Type:    "ipv4",
						},
					},
					TxBytes:    1000000,
					RxBytes:    2000000,
					LastUpdate: now.Unix(),
				},
			},
			Capabilities: []string{"client"},
		},
	}

	// Create connections
	connections := []types.DeviceConnection{
		// Gateway to Switch
		{
			FromDeviceID:   "gateway-01",
			ToDeviceID:     "switch-01",
			FromInterface:  "eth0",
			ToInterface:    "port1",
			ConnectionType: "ethernet",
			Metrics: &types.ConnectionMetrics{
				Latency:     0.5,
				PacketLoss:  0.0,
				Bandwidth:   1000.0,
				Jitter:      0.1,
				LastUpdated: now.Unix(),
			},
		},
		// Switch to AP-01
		{
			FromDeviceID:   "switch-01",
			ToDeviceID:     "ap-01",
			FromInterface:  "port2",
			ToInterface:    "eth0",
			ConnectionType: "ethernet",
			Metrics: &types.ConnectionMetrics{
				Latency:     0.3,
				PacketLoss:  0.0,
				Bandwidth:   1000.0,
				Jitter:      0.05,
				LastUpdated: now.Unix(),
			},
		},
		// Switch to AP-02
		{
			FromDeviceID:   "switch-01",
			ToDeviceID:     "ap-02",
			FromInterface:  "port3",
			ToInterface:    "eth0",
			ConnectionType: "ethernet",
			Metrics: &types.ConnectionMetrics{
				Latency:     0.3,
				PacketLoss:  0.0,
				Bandwidth:   1000.0,
				Jitter:      0.05,
				LastUpdated: now.Unix(),
			},
		},
		// Laptop to AP-01 (WiFi)
		{
			FromDeviceID:   "client-laptop-01",
			ToDeviceID:     "ap-01",
			FromInterface:  "wlan0",
			ToInterface:    "wlan0",
			ConnectionType: "wireless",
			Metrics: &types.ConnectionMetrics{
				Latency:     2.5,
				PacketLoss:  0.1,
				Bandwidth:   433.3,
				Jitter:      0.8,
				LastUpdated: now.Unix(),
			},
		},
		// Phone to AP-02 (WiFi)
		{
			FromDeviceID:   "client-phone-01",
			ToDeviceID:     "ap-02",
			FromInterface:  "wlan0",
			ToInterface:    "wlan0",
			ConnectionType: "wireless",
			Metrics: &types.ConnectionMetrics{
				Latency:     3.2,
				PacketLoss:  0.2,
				Bandwidth:   173.3,
				Jitter:      1.2,
				LastUpdated: now.Unix(),
			},
		},
		// IoT device to AP-01 (WiFi)
		{
			FromDeviceID:   "client-iot-01",
			ToDeviceID:     "ap-01",
			FromInterface:  "wlan0",
			ToInterface:    "wlan0",
			ConnectionType: "wireless",
			Metrics: &types.ConnectionMetrics{
				Latency:     5.5,
				PacketLoss:  0.5,
				Bandwidth:   72.2,
				Jitter:      2.0,
				LastUpdated: now.Unix(),
			},
		},
	}

	// Create gateway info
	gateway := &types.GatewayInfo{
		DeviceID:         "gateway-01",
		IPAddress:        "192.168.1.1",
		MACAddress:       "00:11:22:33:44:55",
		Interface:        "eth0",
		IsDefault:        true,
		ConnectedDevices: 6,
	}

	return &types.NetworkTopology{
		ID:          fmt.Sprintf("topology-%d", rand.Int63()),
		Tenant:      "demo",
		Site:        "office-1",
		Devices:     devices,
		Connections: connections,
		Gateway:     gateway,
		UpdatedAt:   now,
	}
}

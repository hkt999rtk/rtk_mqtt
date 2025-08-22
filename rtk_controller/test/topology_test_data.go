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
			PrimaryMAC:   "00:11:22:33:44:55",
			DeviceType:   "gateway",
			Manufacturer: "RTK",
			Model:        "RTK-GW1000",
			Firmware:     "v2.1.0",
			Online:       true,
			LastSeen:     now.Unix(),
			FirstSeen:    now.Add(-30 * 24 * time.Hour).Unix(),
			Interfaces: []types.NetworkInterface{
				{
					Name:       "eth0",
					Type:       "ethernet",
					MACAddress: "00:11:22:33:44:55",
					Status:     "up",
					Speed:      1000,
					IPAddresses: []types.IPAddress{
						{
							Address: "192.168.1.1",
							Type:    "ipv4",
							Scope:   "private",
						},
					},
				},
				{
					Name:       "wlan0",
					Type:       "wifi",
					MACAddress: "00:11:22:33:44:56",
					Status:     "up",
					Speed:      866,
				},
			},
		},
		"ap-01": {
			DeviceID:     "ap-01",
			PrimaryMAC:   "00:11:22:33:44:66",
			DeviceType:   "access_point",
			Manufacturer: "RTK",
			Model:        "RTK-AP500",
			Firmware:     "v1.5.2",
			Online:       true,
			LastSeen:     now.Unix(),
			FirstSeen:    now.Add(-15 * 24 * time.Hour).Unix(),
			WiFiInfo: &types.WiFiInfo{
				Mode:      "ap",
				SSID:      "RTK-Office",
				BSSID:     "00:11:22:33:44:66",
				Channel:   6,
				Frequency: 2437,
				Security:  "WPA2",
				Standard:  "802.11ac",
			},
			Interfaces: []types.NetworkInterface{
				{
					Name:       "eth0",
					Type:       "ethernet",
					MACAddress: "00:11:22:33:44:66",
					Status:     "up",
					Speed:      1000,
					IPAddresses: []types.IPAddress{
						{
							Address: "192.168.1.10",
							Type:    "ipv4",
							Scope:   "private",
						},
					},
				},
			},
		},
		"ap-02": {
			DeviceID:     "ap-02",
			PrimaryMAC:   "00:11:22:33:44:77",
			DeviceType:   "access_point",
			Manufacturer: "RTK",
			Model:        "RTK-AP500",
			Firmware:     "v1.5.2",
			Online:       true,
			LastSeen:     now.Unix(),
			FirstSeen:    now.Add(-10 * 24 * time.Hour).Unix(),
			WiFiInfo: &types.WiFiInfo{
				Mode:      "ap",
				SSID:      "RTK-Office",
				BSSID:     "00:11:22:33:44:77",
				Channel:   11,
				Frequency: 2462,
				Security:  "WPA2",
				Standard:  "802.11ac",
			},
			Interfaces: []types.NetworkInterface{
				{
					Name:       "eth0",
					Type:       "ethernet",
					MACAddress: "00:11:22:33:44:77",
					Status:     "up",
					Speed:      1000,
					IPAddresses: []types.IPAddress{
						{
							Address: "192.168.1.11",
							Type:    "ipv4",
							Scope:   "private",
						},
					},
				},
			},
		},
		"client-laptop-01": {
			DeviceID:     "client-laptop-01",
			PrimaryMAC:   "AA:BB:CC:DD:EE:01",
			DeviceType:   "laptop",
			Manufacturer: "Dell",
			Model:        "Latitude 5520",
			Online:       true,
			LastSeen:     now.Unix(),
			FirstSeen:    now.Add(-7 * 24 * time.Hour).Unix(),
			WiFiInfo: &types.WiFiInfo{
				Mode:           "station",
				SSID:           "RTK-Office",
				BSSID:          "00:11:22:33:44:66",
				Channel:        6,
				Frequency:      2437,
				SignalStrength: -45,
				LinkQuality:    85,
				TxRate:         433.3,
				RxRate:         433.3,
			},
			Interfaces: []types.NetworkInterface{
				{
					Name:       "wlan0",
					Type:       "wifi",
					MACAddress: "AA:BB:CC:DD:EE:01",
					Status:     "up",
					IPAddresses: []types.IPAddress{
						{
							Address: "192.168.1.101",
							Type:    "ipv4",
							Scope:   "private",
						},
					},
				},
			},
		},
		"client-phone-01": {
			DeviceID:     "client-phone-01",
			PrimaryMAC:   "AA:BB:CC:DD:EE:02",
			DeviceType:   "phone",
			Manufacturer: "Apple",
			Model:        "iPhone 14",
			Online:       true,
			LastSeen:     now.Unix(),
			FirstSeen:    now.Add(-3 * 24 * time.Hour).Unix(),
			WiFiInfo: &types.WiFiInfo{
				Mode:           "station",
				SSID:           "RTK-Office",
				BSSID:          "00:11:22:33:44:77",
				Channel:        11,
				Frequency:      2462,
				SignalStrength: -52,
				LinkQuality:    75,
				TxRate:         173.3,
				RxRate:         173.3,
			},
			Interfaces: []types.NetworkInterface{
				{
					Name:       "wlan0",
					Type:       "wifi",
					MACAddress: "AA:BB:CC:DD:EE:02",
					Status:     "up",
					IPAddresses: []types.IPAddress{
						{
							Address: "192.168.1.102",
							Type:    "ipv4",
							Scope:   "private",
						},
					},
				},
			},
		},
		"client-iot-01": {
			DeviceID:     "client-iot-01",
			PrimaryMAC:   "AA:BB:CC:DD:EE:03",
			DeviceType:   "iot",
			Manufacturer: "Xiaomi",
			Model:        "Mi Smart Bulb",
			Online:       true,
			LastSeen:     now.Unix(),
			FirstSeen:    now.Add(-30 * 24 * time.Hour).Unix(),
			WiFiInfo: &types.WiFiInfo{
				Mode:           "station",
				SSID:           "RTK-Office",
				BSSID:          "00:11:22:33:44:66",
				Channel:        6,
				Frequency:      2437,
				SignalStrength: -68,
				LinkQuality:    55,
				TxRate:         72.2,
				RxRate:         72.2,
			},
			Interfaces: []types.NetworkInterface{
				{
					Name:       "wlan0",
					Type:       "wifi",
					MACAddress: "AA:BB:CC:DD:EE:03",
					Status:     "up",
					IPAddresses: []types.IPAddress{
						{
							Address: "192.168.1.103",
							Type:    "ipv4",
							Scope:   "private",
						},
					},
				},
			},
		},
		"switch-01": {
			DeviceID:     "switch-01",
			PrimaryMAC:   "00:11:22:33:44:88",
			DeviceType:   "switch",
			Manufacturer: "RTK",
			Model:        "RTK-SW24",
			Firmware:     "v1.2.0",
			Online:       true,
			LastSeen:     now.Unix(),
			FirstSeen:    now.Add(-60 * 24 * time.Hour).Unix(),
			Interfaces: []types.NetworkInterface{
				{
					Name:       "mgmt0",
					Type:       "ethernet",
					MACAddress: "00:11:22:33:44:88",
					Status:     "up",
					Speed:      1000,
					IPAddresses: []types.IPAddress{
						{
							Address: "192.168.1.5",
							Type:    "ipv4",
							Scope:   "private",
						},
					},
				},
			},
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

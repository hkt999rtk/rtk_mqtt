package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
	
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// TopologyDiscoveryMessage represents a device discovery message
type TopologyDiscoveryMessage struct {
	Schema      string                 `json:"schema"`
	Timestamp   int64                  `json:"timestamp"`
	DeviceID    string                 `json:"device_id"`
	DeviceInfo  map[string]interface{} `json:"device_info"`
	Interfaces  []map[string]interface{} `json:"interfaces,omitempty"`
	RoutingInfo map[string]interface{} `json:"routing_info,omitempty"`
	BridgeInfo  map[string]interface{} `json:"bridge_info,omitempty"`
}

// WiFiClientsMessage represents WiFi client telemetry
type WiFiClientsMessage struct {
	Schema    string                 `json:"schema"`
	Timestamp int64                  `json:"timestamp"`
	DeviceID  string                 `json:"device_id"`
	SSID      string                 `json:"ssid"`
	BSSID     string                 `json:"bssid"`
	Clients   []map[string]interface{} `json:"clients"`
}

// ConnectionsMessage represents network connections
type ConnectionsMessage struct {
	Schema      string                 `json:"schema"`
	Timestamp   int64                  `json:"timestamp"`
	DeviceID    string                 `json:"device_id"`
	Connections []map[string]interface{} `json:"connections"`
}

func main() {
	// MQTT broker configuration
	broker := "tcp://localhost:1883"
	clientID := "topology-test-publisher"
	
	// Create MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetUsername("test")
	opts.SetPassword("test123")
	
	// Create and connect client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
	defer client.Disconnect(250)
	
	fmt.Println("Connected to MQTT broker")
	
	// Test 1: Send topology discovery message for a new router
	sendTopologyDiscovery(client, "router-02")
	
	// Test 2: Send WiFi clients telemetry
	sendWiFiClients(client, "ap-01")
	
	// Test 3: Send connection updates
	sendConnections(client, "switch-01")
	
	// Test 4: Send device identity update
	sendDeviceIdentity(client, "client-02")
	
	fmt.Println("All test messages sent successfully")
	
	// Wait a bit for messages to be processed
	time.Sleep(2 * time.Second)
}

func sendTopologyDiscovery(client mqtt.Client, deviceID string) {
	message := TopologyDiscoveryMessage{
		Schema:    "topology.discovery.v1",
		Timestamp: time.Now().Unix() * 1000,
		DeviceID:  deviceID,
		DeviceInfo: map[string]interface{}{
			"hostname":     "rtk-router-02",
			"manufacturer": "RTK",
			"model":        "RTK-RT2000",
			"device_type":  "router",
			"location":     "Building B",
			"capabilities": []string{"routing", "nat", "firewall", "vpn"},
		},
		Interfaces: []map[string]interface{}{
			{
				"name":       "eth0",
				"type":       "ethernet",
				"mac":        "00:11:22:33:44:99",
				"ip_address": "192.168.2.1",
				"netmask":    "255.255.255.0",
				"status":     "up",
				"speed":      1000,
			},
			{
				"name":       "eth1",
				"type":       "ethernet",
				"mac":        "00:11:22:33:44:9A",
				"ip_address": "10.0.0.2",
				"netmask":    "255.255.255.0",
				"status":     "up",
				"speed":      1000,
			},
		},
		RoutingInfo: map[string]interface{}{
			"default_gateway": "10.0.0.1",
			"routes": []map[string]interface{}{
				{
					"destination": "0.0.0.0/0",
					"gateway":     "10.0.0.1",
					"interface":   "eth1",
					"metric":      1,
				},
				{
					"destination": "192.168.1.0/24",
					"gateway":     "192.168.2.254",
					"interface":   "eth0",
					"metric":      10,
				},
			},
		},
	}
	
	topic := fmt.Sprintf("rtk/v1/default/default/%s/topology/discovery", deviceID)
	publishMessage(client, topic, message)
	fmt.Printf("Sent topology discovery for %s\n", deviceID)
}

func sendWiFiClients(client mqtt.Client, deviceID string) {
	message := WiFiClientsMessage{
		Schema:    "telemetry.wifi_clients.v1",
		Timestamp: time.Now().Unix() * 1000,
		DeviceID:  deviceID,
		SSID:      "RTK-Office",
		BSSID:     "00:11:22:33:44:67",
		Clients: []map[string]interface{}{
			{
				"mac_address":     "AA:BB:CC:DD:EE:02",
				"ip_address":      "192.168.1.102",
				"hostname":        "phone-02",
				"rssi":            -48,
				"tx_rate":         200.5,
				"rx_rate":         150.3,
				"connected_time":  3600,
				"idle_time":       10,
				"tx_bytes":        50000000,
				"rx_bytes":        20000000,
			},
			{
				"mac_address":     "AA:BB:CC:DD:EE:03",
				"ip_address":      "192.168.1.103",
				"hostname":        "tablet-01",
				"rssi":            -55,
				"tx_rate":         150.0,
				"rx_rate":         100.0,
				"connected_time":  7200,
				"idle_time":       120,
				"tx_bytes":        10000000,
				"rx_bytes":        5000000,
			},
		},
	}
	
	topic := fmt.Sprintf("rtk/v1/default/default/%s/telemetry/wifi_clients", deviceID)
	publishMessage(client, topic, message)
	fmt.Printf("Sent WiFi clients telemetry for %s\n", deviceID)
}

func sendConnections(client mqtt.Client, deviceID string) {
	message := ConnectionsMessage{
		Schema:    "topology.connections.v1",
		Timestamp: time.Now().Unix() * 1000,
		DeviceID:  deviceID,
		Connections: []map[string]interface{}{
			{
				"interface":     "port1",
				"remote_mac":    "00:11:22:33:44:55",
				"remote_device": "gateway-01",
				"link_type":     "ethernet",
				"link_speed":    1000,
				"link_status":   "up",
				"vlan_id":       1,
			},
			{
				"interface":     "port2",
				"remote_mac":    "00:11:22:33:44:66",
				"remote_device": "ap-01",
				"link_type":     "ethernet",
				"link_speed":    1000,
				"link_status":   "up",
				"vlan_id":       1,
			},
			{
				"interface":     "port3",
				"remote_mac":    "00:11:22:33:44:77",
				"remote_device": "ap-02",
				"link_type":     "ethernet",
				"link_speed":    1000,
				"link_status":   "up",
				"vlan_id":       1,
			},
			{
				"interface":     "port4",
				"remote_mac":    "00:11:22:33:44:99",
				"remote_device": "router-02",
				"link_type":     "ethernet",
				"link_speed":    1000,
				"link_status":   "up",
				"vlan_id":       10,
			},
		},
	}
	
	topic := fmt.Sprintf("rtk/v1/default/default/%s/topology/connections", deviceID)
	publishMessage(client, topic, message)
	fmt.Printf("Sent connections update for %s\n", deviceID)
}

func sendDeviceIdentity(client mqtt.Client, deviceID string) {
	message := map[string]interface{}{
		"schema":       "device.identity.v1",
		"timestamp":    time.Now().Unix() * 1000,
		"device_id":    deviceID,
		"mac_address":  "AA:BB:CC:DD:EE:04",
		"hostname":     "laptop-02",
		"manufacturer": "Lenovo",
		"model":        "ThinkPad X1",
		"device_type":  "laptop",
		"os_info": map[string]interface{}{
			"name":    "Windows",
			"version": "11",
			"build":   "22000",
		},
		"location":     "Office Area B",
		"owner":        "Jane Smith",
		"tags":         []string{"engineering", "development"},
	}
	
	topic := fmt.Sprintf("rtk/v1/default/default/%s/device/identity", deviceID)
	publishMessage(client, topic, message)
	fmt.Printf("Sent device identity for %s\n", deviceID)
}

func publishMessage(client mqtt.Client, topic string, message interface{}) {
	payload, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}
	
	token := client.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Failed to publish message: %v", token.Error())
	}
}
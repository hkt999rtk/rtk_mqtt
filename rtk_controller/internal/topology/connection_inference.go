package topology

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"rtk_controller/pkg/types"
)

// ConnectionInference handles advanced connection relationship inference
type ConnectionInference struct {
	// Device data
	devices map[string]*types.NetworkDevice
	
	// Inference algorithms
	algorithms []InferenceAlgorithm
	
	// Configuration
	config InferenceConfig
	
	// Results
	inferredConnections []types.DeviceConnection
	confidenceScores   map[string]float64
}

// InferenceConfig holds connection inference configuration
type InferenceConfig struct {
	// Algorithm weights
	Layer2Weight            float64 // Bridge table analysis
	Layer3Weight            float64 // Routing table analysis
	WiFiWeight              float64 // WiFi client associations
	DHCPWeight              float64 // DHCP lease analysis
	NetworkScanWeight       float64 // Network scanning results
	
	// Thresholds
	MinConfidenceThreshold  float64 // Minimum confidence to create connection
	DirectLinkThreshold     float64 // Threshold for direct link classification
	HopCountThreshold       int     // Maximum hop count for indirect connections
	
	// Timing
	ConnectionTimeout       time.Duration // How long to consider connections valid
	InferenceInterval       time.Duration // How often to run inference
	
	// Features
	EnableMultiPathDetection bool // Detect multiple paths between devices
	EnableLoadBalancing      bool // Detect load balancing configurations
	EnableRedundancy         bool // Detect redundant connections
}

// InferenceAlgorithm defines interface for connection inference algorithms
type InferenceAlgorithm interface {
	Name() string
	Weight() float64
	InferConnections(devices map[string]*types.NetworkDevice) ([]types.DeviceConnection, error)
}

// InferenceResult holds the results of connection inference
type InferenceResult struct {
	Connections      []types.DeviceConnection
	ConfidenceScores map[string]float64
	AlgorithmResults map[string][]types.DeviceConnection
	Errors          []error
}

// NewConnectionInference creates a new connection inference engine
func NewConnectionInference(config InferenceConfig) *ConnectionInference {
	ci := &ConnectionInference{
		devices:             make(map[string]*types.NetworkDevice),
		confidenceScores:   make(map[string]float64),
		config:             config,
	}
	
	// Initialize inference algorithms
	ci.algorithms = []InferenceAlgorithm{
		NewLayer2Inference(config.Layer2Weight),
		NewLayer3Inference(config.Layer3Weight),
		NewWiFiInference(config.WiFiWeight),
		NewDHCPInference(config.DHCPWeight),
		NewNetworkScanInference(config.NetworkScanWeight),
	}
	
	return ci
}

// InferConnections runs all inference algorithms and combines results
func (ci *ConnectionInference) InferConnections(devices map[string]*types.NetworkDevice) (*InferenceResult, error) {
	ci.devices = devices
	
	result := &InferenceResult{
		AlgorithmResults: make(map[string][]types.DeviceConnection),
		ConfidenceScores: make(map[string]float64),
	}
	
	// Run each inference algorithm
	var allConnections []types.DeviceConnection
	
	for _, algorithm := range ci.algorithms {
		connections, err := algorithm.InferConnections(devices)
		if err != nil {
			log.Printf("Error in %s algorithm: %v", algorithm.Name(), err)
			result.Errors = append(result.Errors, err)
			continue
		}
		
		result.AlgorithmResults[algorithm.Name()] = connections
		allConnections = append(allConnections, connections...)
		
		log.Printf("%s algorithm found %d connections", algorithm.Name(), len(connections))
	}
	
	// Combine and score connections
	finalConnections := ci.combineConnections(allConnections)
	
	// Apply confidence filtering
	var filteredConnections []types.DeviceConnection
	for _, conn := range finalConnections {
		confidence := ci.confidenceScores[conn.ID]
		if confidence >= ci.config.MinConfidenceThreshold {
			// TODO: Add Confidence field to ConnectionMetrics
			// conn.Metrics.Confidence = confidence
			filteredConnections = append(filteredConnections, conn)
		}
	}
	
	result.Connections = filteredConnections
	result.ConfidenceScores = ci.confidenceScores
	
	log.Printf("Connection inference completed: %d connections found", len(filteredConnections))
	return result, nil
}

// combineConnections merges connections from different algorithms
func (ci *ConnectionInference) combineConnections(connections []types.DeviceConnection) []types.DeviceConnection {
	// Group connections by device pair
	connectionGroups := make(map[string][]types.DeviceConnection)
	
	for _, conn := range connections {
		key := ci.getConnectionKey(conn.FromDeviceID, conn.ToDeviceID)
		connectionGroups[key] = append(connectionGroups[key], conn)
	}
	
	// Combine connections for each device pair
	var finalConnections []types.DeviceConnection
	
	for _, group := range connectionGroups {
		combined := ci.combineConnectionGroup(group)
		if combined != nil {
			finalConnections = append(finalConnections, *combined)
		}
	}
	
	return finalConnections
}

// combineConnectionGroup combines multiple connections between same device pair
func (ci *ConnectionInference) combineConnectionGroup(connections []types.DeviceConnection) *types.DeviceConnection {
	if len(connections) == 0 {
		return nil
	}
	
	// Start with the first connection
	combined := connections[0]
	combined.ID = fmt.Sprintf("%s-%s-%d", combined.FromDeviceID, combined.ToDeviceID, time.Now().UnixMilli())
	
	// Calculate combined confidence score
	var totalWeight float64
	var weightedConfidence float64
	
	// Aggregate evidence from all algorithms
	connectionTypes := make(map[string]bool)
	interfaces := make(map[string]string)
	
	for _, conn := range connections {
		// Weight based on algorithm type
		weight := ci.getAlgorithmWeight(conn.ConnectionType)
		totalWeight += weight
		weightedConfidence += weight * ci.getConnectionConfidence(conn)
		
		// Collect connection types
		connectionTypes[conn.ConnectionType] = true
		
		// Collect interface information
		if conn.FromInterface != "" {
			interfaces["from"] = conn.FromInterface
		}
		if conn.ToInterface != "" {
			interfaces["to"] = conn.ToInterface
		}
		
		// Use latest timestamps
		if conn.LastSeen > combined.LastSeen {
			combined.LastSeen = conn.LastSeen
		}
		if conn.Discovered > combined.Discovered {
			combined.Discovered = conn.Discovered
		}
	}
	
	// Calculate final confidence
	if totalWeight > 0 {
		confidence := weightedConfidence / totalWeight
		ci.confidenceScores[combined.ID] = confidence
		
		// Determine if this is a direct link
		combined.IsDirectLink = confidence >= ci.config.DirectLinkThreshold
		
		// Set primary connection type (most confident)
		combined.ConnectionType = ci.getPrimaryConnectionType(connectionTypes)
		
		// Set interfaces
		if fromIface, exists := interfaces["from"]; exists {
			combined.FromInterface = fromIface
		}
		if toIface, exists := interfaces["to"]; exists {
			combined.ToInterface = toIface
		}
	}
	
	return &combined
}

// getConnectionKey creates a normalized key for device pair
func (ci *ConnectionInference) getConnectionKey(device1, device2 string) string {
	// Always order devices consistently
	if device1 > device2 {
		device1, device2 = device2, device1
	}
	return fmt.Sprintf("%s-%s", device1, device2)
}

// getAlgorithmWeight returns weight for algorithm based on connection type
func (ci *ConnectionInference) getAlgorithmWeight(connectionType string) float64 {
	switch connectionType {
	case "bridge", "switch":
		return ci.config.Layer2Weight
	case "route", "gateway":
		return ci.config.Layer3Weight
	case "wifi":
		return ci.config.WiFiWeight
	case "dhcp":
		return ci.config.DHCPWeight
	case "scan":
		return ci.config.NetworkScanWeight
	default:
		return 1.0
	}
}

// getConnectionConfidence calculates confidence for a connection
func (ci *ConnectionInference) getConnectionConfidence(conn types.DeviceConnection) float64 {
	confidence := 0.5 // Base confidence
	
	// Adjust based on connection type
	switch conn.ConnectionType {
	case "bridge":
		confidence = 0.9 // Bridge table entries are very reliable
	case "route":
		confidence = 0.8 // Routing table entries are reliable
	case "wifi":
		confidence = 0.85 // WiFi associations are quite reliable
	case "dhcp":
		confidence = 0.7 // DHCP leases are moderately reliable
	case "scan":
		confidence = 0.6 // Network scanning is less reliable
	}
	
	// Adjust based on timing (newer is more confident)
	now := time.Now().UnixMilli()
	age := time.Duration(now - conn.LastSeen) * time.Millisecond
	
	if age < 1*time.Minute {
		confidence += 0.1
	} else if age > 10*time.Minute {
		confidence -= 0.2
	}
	
	// Adjust based on metrics
	if conn.Metrics.RSSI > -50 { // Strong WiFi signal
		confidence += 0.1
	}
	if conn.Metrics.LinkSpeed > 100 { // High-speed connection
		confidence += 0.05
	}
	
	// Ensure confidence is in valid range
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}
	
	return confidence
}

// getPrimaryConnectionType determines the primary connection type
func (ci *ConnectionInference) getPrimaryConnectionType(types map[string]bool) string {
	// Priority order for connection types
	priority := []string{"bridge", "wifi", "route", "dhcp", "scan"}
	
	for _, connType := range priority {
		if types[connType] {
			return connType
		}
	}
	
	return "unknown"
}

// Layer2Inference implements bridge table analysis
type Layer2Inference struct {
	weight float64
}

func NewLayer2Inference(weight float64) *Layer2Inference {
	return &Layer2Inference{weight: weight}
}

func (l2 *Layer2Inference) Name() string {
	return "Layer2"
}

func (l2 *Layer2Inference) Weight() float64 {
	return l2.weight
}

func (l2 *Layer2Inference) InferConnections(devices map[string]*types.NetworkDevice) ([]types.DeviceConnection, error) {
	var connections []types.DeviceConnection
	
	for _, device := range devices {
		if device.BridgeInfo == nil {
			continue
		}
		
		// Analyze bridge table entries
		for _, entry := range device.BridgeInfo.BridgeTable {
			if entry.IsLocal {
				continue // Skip local entries
			}
			
			// Find device with this MAC address
			targetDevice := l2.findDeviceByMAC(devices, entry.MacAddress)
			if targetDevice == nil {
				continue
			}
			
			// Create connection
			connection := types.DeviceConnection{
				ID:             fmt.Sprintf("%s-%s-bridge-%s", device.DeviceID, targetDevice.DeviceID, entry.Interface),
				FromDeviceID:   device.DeviceID,
				ToDeviceID:     targetDevice.DeviceID,
				FromInterface:  entry.Interface,
				ConnectionType: "bridge",
				IsDirectLink:   true,
				LastSeen:       time.Now().UnixMilli(),
				Discovered:     time.Now().UnixMilli(),
			}
			
			connections = append(connections, connection)
		}
	}
	
	return connections, nil
}

func (l2 *Layer2Inference) findDeviceByMAC(devices map[string]*types.NetworkDevice, macAddr string) *types.NetworkDevice {
	macAddr = strings.ToLower(macAddr)
	
	for _, device := range devices {
		// Check primary MAC
		if strings.ToLower(device.PrimaryMAC) == macAddr {
			return device
		}
		
		// Check interface MACs
		for _, iface := range device.Interfaces {
			if strings.ToLower(iface.MacAddress) == macAddr {
				return device
			}
		}
	}
	
	return nil
}

// Layer3Inference implements routing table analysis
type Layer3Inference struct {
	weight float64
}

func NewLayer3Inference(weight float64) *Layer3Inference {
	return &Layer3Inference{weight: weight}
}

func (l3 *Layer3Inference) Name() string {
	return "Layer3"
}

func (l3 *Layer3Inference) Weight() float64 {
	return l3.weight
}

func (l3 *Layer3Inference) InferConnections(devices map[string]*types.NetworkDevice) ([]types.DeviceConnection, error) {
	var connections []types.DeviceConnection
	
	for _, device := range devices {
		if device.RoutingInfo == nil {
			continue
		}
		
		// Analyze routing table entries
		for _, route := range device.RoutingInfo.RoutingTable {
			if route.Gateway == "" || route.Gateway == "0.0.0.0" {
				continue // Skip direct routes
			}
			
			// Find device with gateway IP
			gatewayDevice := l3.findDeviceByIP(devices, route.Gateway)
			if gatewayDevice == nil {
				continue
			}
			
			// Create connection
			connection := types.DeviceConnection{
				ID:             fmt.Sprintf("%s-%s-route-%s", device.DeviceID, gatewayDevice.DeviceID, route.Interface),
				FromDeviceID:   device.DeviceID,
				ToDeviceID:     gatewayDevice.DeviceID,
				FromInterface:  route.Interface,
				ConnectionType: "route",
				IsDirectLink:   false, // Routing connections are typically not direct
				LastSeen:       time.Now().UnixMilli(),
				Discovered:     time.Now().UnixMilli(),
			}
			
			connections = append(connections, connection)
		}
	}
	
	return connections, nil
}

func (l3 *Layer3Inference) findDeviceByIP(devices map[string]*types.NetworkDevice, ipAddr string) *types.NetworkDevice {
	for _, device := range devices {
		for _, iface := range device.Interfaces {
			for _, ipInfo := range iface.IPAddresses {
				if ipInfo.Address == ipAddr {
					return device
				}
			}
		}
	}
	
	return nil
}

// WiFiInference implements WiFi client association analysis
type WiFiInference struct {
	weight float64
}

func NewWiFiInference(weight float64) *WiFiInference {
	return &WiFiInference{weight: weight}
}

func (w *WiFiInference) Name() string {
	return "WiFi"
}

func (w *WiFiInference) Weight() float64 {
	return w.weight
}

func (w *WiFiInference) InferConnections(devices map[string]*types.NetworkDevice) ([]types.DeviceConnection, error) {
	var connections []types.DeviceConnection
	
	// This would typically be populated from WiFi client telemetry messages
	// For now, we'll look for WiFi interfaces and infer connections
	
	for _, device := range devices {
		for ifaceName, iface := range device.Interfaces {
			if iface.Type == "wifi" && iface.WiFiMode == "STA" {
				// This is a WiFi client, find its AP
				if iface.BSSID != "" {
					apDevice := w.findAPByBSSID(devices, iface.BSSID)
					if apDevice != nil {
						connection := types.DeviceConnection{
							ID:             fmt.Sprintf("%s-%s-wifi-%s", device.DeviceID, apDevice.DeviceID, ifaceName),
							FromDeviceID:   device.DeviceID,
							ToDeviceID:     apDevice.DeviceID,
							FromInterface:  ifaceName,
							ConnectionType: "wifi",
							IsDirectLink:   true,
							LastSeen:       time.Now().UnixMilli(),
							Discovered:     time.Now().UnixMilli(),
						}
						
						// Add WiFi-specific metrics
						connection.Metrics.RSSI = iface.RSSI
						// TODO: Add Channel field to ConnectionMetrics
						// connection.Metrics.Channel = iface.Channel
						
						connections = append(connections, connection)
					}
				}
			}
		}
	}
	
	return connections, nil
}

func (w *WiFiInference) findAPByBSSID(devices map[string]*types.NetworkDevice, bssid string) *types.NetworkDevice {
	bssid = strings.ToLower(bssid)
	
	for _, device := range devices {
		for _, iface := range device.Interfaces {
			if iface.Type == "wifi" && iface.WiFiMode == "AP" {
				if strings.ToLower(iface.BSSID) == bssid {
					return device
				}
			}
		}
	}
	
	return nil
}

// DHCPInference implements DHCP lease analysis
type DHCPInference struct {
	weight float64
}

func NewDHCPInference(weight float64) *DHCPInference {
	return &DHCPInference{weight: weight}
}

func (d *DHCPInference) Name() string {
	return "DHCP"
}

func (d *DHCPInference) Weight() float64 {
	return d.weight
}

func (d *DHCPInference) InferConnections(devices map[string]*types.NetworkDevice) ([]types.DeviceConnection, error) {
	var connections []types.DeviceConnection
	
	for _, device := range devices {
		if device.RoutingInfo == nil || device.RoutingInfo.DHCPServer == nil {
			continue
		}
		
		dhcpServer := device.RoutingInfo.DHCPServer
		if !dhcpServer.Enabled {
			continue
		}
		
		// Analyze DHCP leases
		for _, lease := range dhcpServer.ActiveLeases {
			// Find device with this MAC address
			clientDevice := d.findDeviceByMAC(devices, lease.MacAddress)
			if clientDevice == nil {
				continue
			}
			
			// Create connection
			connection := types.DeviceConnection{
				ID:             fmt.Sprintf("%s-%s-dhcp", device.DeviceID, clientDevice.DeviceID),
				FromDeviceID:   clientDevice.DeviceID, // Client -> Server
				ToDeviceID:     device.DeviceID,
				ConnectionType: "dhcp",
				IsDirectLink:   true, // DHCP implies same network segment
				LastSeen:       lease.LeaseEnd, // Use lease end time
				Discovered:     lease.LeaseStart,
			}
			
			connections = append(connections, connection)
		}
	}
	
	return connections, nil
}

func (d *DHCPInference) findDeviceByMAC(devices map[string]*types.NetworkDevice, macAddr string) *types.NetworkDevice {
	macAddr = strings.ToLower(macAddr)
	
	for _, device := range devices {
		if strings.ToLower(device.PrimaryMAC) == macAddr {
			return device
		}
		
		for _, iface := range device.Interfaces {
			if strings.ToLower(iface.MacAddress) == macAddr {
				return device
			}
		}
	}
	
	return nil
}

// NetworkScanInference implements network scanning-based inference
type NetworkScanInference struct {
	weight float64
}

func NewNetworkScanInference(weight float64) *NetworkScanInference {
	return &NetworkScanInference{weight: weight}
}

func (n *NetworkScanInference) Name() string {
	return "NetworkScan"
}

func (n *NetworkScanInference) Weight() float64 {
	return n.weight
}

func (n *NetworkScanInference) InferConnections(devices map[string]*types.NetworkDevice) ([]types.DeviceConnection, error) {
	var connections []types.DeviceConnection
	
	// Infer connections based on network topology
	// This involves analyzing IP subnets and inferring connectivity
	
	// Group devices by subnet
	subnetDevices := make(map[string][]*types.NetworkDevice)
	
	for _, device := range devices {
		for _, iface := range device.Interfaces {
			for _, ipInfo := range iface.IPAddresses {
				if ipInfo.Network != "" {
					_, network, err := net.ParseCIDR(ipInfo.Network)
					if err == nil {
						subnet := network.String()
						subnetDevices[subnet] = append(subnetDevices[subnet], device)
					}
				}
			}
		}
	}
	
	// For each subnet, infer that devices can communicate
	for subnet, subnetDeviceList := range subnetDevices {
		if len(subnetDeviceList) < 2 {
			continue // Need at least 2 devices
		}
		
		// Create connections between all devices in same subnet
		for i, device1 := range subnetDeviceList {
			for j, device2 := range subnetDeviceList {
				if i >= j { // Avoid duplicates and self-connections
					continue
				}
				
				connection := types.DeviceConnection{
					ID:             fmt.Sprintf("%s-%s-subnet-%s", device1.DeviceID, device2.DeviceID, subnet),
					FromDeviceID:   device1.DeviceID,
					ToDeviceID:     device2.DeviceID,
					ConnectionType: "scan",
					IsDirectLink:   true, // Same subnet implies direct connectivity
					LastSeen:       time.Now().UnixMilli(),
					Discovered:     time.Now().UnixMilli(),
				}
				
				connections = append(connections, connection)
			}
		}
	}
	
	return connections, nil
}
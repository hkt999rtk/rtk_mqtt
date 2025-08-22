package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"rtk_controller/internal/schema"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// TopologyProcessor handles topology-related MQTT messages
type TopologyProcessor struct {
	schemaManager   *schema.Manager
	topologyStorage *storage.TopologyStorage
	identityStorage *storage.IdentityStorage

	// Message handlers
	handlers map[string]func(string, []byte) error
}

// NewTopologyProcessor creates a new topology message processor
func NewTopologyProcessor(
	schemaManager *schema.Manager,
	topologyStorage *storage.TopologyStorage,
	identityStorage *storage.IdentityStorage,
) *TopologyProcessor {
	processor := &TopologyProcessor{
		schemaManager:   schemaManager,
		topologyStorage: topologyStorage,
		identityStorage: identityStorage,
		handlers:        make(map[string]func(string, []byte) error),
	}

	// Register message handlers
	processor.registerHandlers()

	return processor
}

// ProcessMessage processes a topology-related MQTT message
func (tp *TopologyProcessor) ProcessMessage(topic string, payload []byte) error {
	// Validate the message first
	result, err := tp.schemaManager.ValidateMessage(topic, payload)
	if err != nil {
		return fmt.Errorf("failed to validate message: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("invalid message format: %v", result.Errors)
	}

	// Determine message type and route to appropriate handler
	messageType := tp.getMessageType(topic, result.Schema)
	if handler, exists := tp.handlers[messageType]; exists {
		return handler(topic, payload)
	}

	return fmt.Errorf("no handler for message type: %s", messageType)
}

// IsTopologyMessage checks if a topic is related to topology
func (tp *TopologyProcessor) IsTopologyMessage(topic string) bool {
	topologyTopics := []string{
		"/topology/discovery",
		"/topology/connections",
		"/telemetry/wifi_clients",
		"/device/identity",
		"/diagnostics/network",
		"/telemetry/qos",
	}

	for _, topologyTopic := range topologyTopics {
		if strings.Contains(topic, topologyTopic) {
			return true
		}
	}

	return false
}

// Private methods

func (tp *TopologyProcessor) registerHandlers() {
	tp.handlers["topology.discovery"] = tp.handleTopologyDiscovery
	tp.handlers["topology.connections"] = tp.handleTopologyConnections
	tp.handlers["telemetry.wifi_clients"] = tp.handleWiFiClients
	tp.handlers["device.identity"] = tp.handleDeviceIdentity
	tp.handlers["diagnostics.network"] = tp.handleNetworkDiagnostics
	tp.handlers["telemetry.qos"] = tp.handleQoSInfo
}

func (tp *TopologyProcessor) getMessageType(topic string, schema string) string {
	// Extract message type from schema name or topic
	if schema != "unknown" && schema != "no_schema" {
		return schema
	}

	// Fallback: extract from topic
	if strings.Contains(topic, "/topology/discovery") {
		return "topology.discovery"
	} else if strings.Contains(topic, "/topology/connections") {
		return "topology.connections"
	} else if strings.Contains(topic, "/telemetry/wifi_clients") {
		return "telemetry.wifi_clients"
	} else if strings.Contains(topic, "/device/identity") {
		return "device.identity"
	} else if strings.Contains(topic, "/diagnostics/network") {
		return "diagnostics.network"
	} else if strings.Contains(topic, "/telemetry/qos") {
		return "telemetry.qos"
	}

	return "unknown"
}

func (tp *TopologyProcessor) handleTopologyDiscovery(topic string, payload []byte) error {
	var message struct {
		Schema      string                   `json:"schema"`
		Timestamp   int64                    `json:"timestamp"`
		DeviceID    string                   `json:"device_id"`
		DeviceInfo  map[string]interface{}   `json:"device_info"`
		Interfaces  []map[string]interface{} `json:"interfaces,omitempty"`
		RoutingInfo map[string]interface{}   `json:"routing_info,omitempty"`
		BridgeInfo  map[string]interface{}   `json:"bridge_info,omitempty"`
	}

	if err := json.Unmarshal(payload, &message); err != nil {
		return fmt.Errorf("failed to unmarshal topology discovery message: %w", err)
	}

	// Convert to NetworkDevice structure
	networkDevice, err := tp.convertToNetworkDevice(message.DeviceID, message.DeviceInfo, message.Interfaces, message.RoutingInfo, message.BridgeInfo)
	if err != nil {
		return fmt.Errorf("failed to convert to network device: %w", err)
	}

	// Update last seen timestamp
	networkDevice.LastSeen = message.Timestamp
	networkDevice.Online = true

	// Save the network device
	if err := tp.topologyStorage.SaveNetworkDevice(networkDevice); err != nil {
		return fmt.Errorf("failed to save network device: %w", err)
	}

	log.Printf("Processed topology discovery for device %s", message.DeviceID)
	return nil
}

func (tp *TopologyProcessor) handleTopologyConnections(topic string, payload []byte) error {
	var message struct {
		Schema      string                   `json:"schema"`
		Timestamp   int64                    `json:"timestamp"`
		DeviceID    string                   `json:"device_id"`
		Connections []map[string]interface{} `json:"connections"`
		GatewayInfo map[string]interface{}   `json:"gateway_info,omitempty"`
	}

	if err := json.Unmarshal(payload, &message); err != nil {
		return fmt.Errorf("failed to unmarshal topology connections message: %w", err)
	}

	// Process each connection
	for _, connData := range message.Connections {
		connection, err := tp.convertToDeviceConnection(message.DeviceID, connData, message.Timestamp)
		if err != nil {
			log.Printf("Failed to convert connection: %v", err)
			continue
		}

		if err := tp.topologyStorage.SaveDeviceConnection(connection); err != nil {
			log.Printf("Failed to save device connection: %v", err)
		}
	}

	// Process gateway info if present
	if message.GatewayInfo != nil {
		gatewayInfo, err := tp.convertToGatewayInfo(message.DeviceID, message.GatewayInfo, message.Timestamp)
		if err == nil {
			if err := tp.topologyStorage.SaveGatewayInfo(gatewayInfo); err != nil {
				log.Printf("Failed to save gateway info: %v", err)
			}
		}
	}

	log.Printf("Processed topology connections for device %s", message.DeviceID)
	return nil
}

func (tp *TopologyProcessor) handleWiFiClients(topic string, payload []byte) error {
	var message struct {
		Schema    string                   `json:"schema"`
		Timestamp int64                    `json:"timestamp"`
		DeviceID  string                   `json:"device_id"`
		Interface string                   `json:"interface"`
		APInfo    map[string]interface{}   `json:"ap_info,omitempty"`
		Clients   []map[string]interface{} `json:"clients"`
	}

	if err := json.Unmarshal(payload, &message); err != nil {
		return fmt.Errorf("failed to unmarshal wifi clients message: %w", err)
	}

	// Process WiFi clients to detect device connections
	for _, clientData := range message.Clients {
		if macAddr, ok := clientData["mac_address"].(string); ok {
			// Create a device connection for this WiFi client
			connection := &types.DeviceConnection{
				ID:             fmt.Sprintf("%s-%s-%d", message.DeviceID, macAddr, message.Timestamp),
				FromDeviceID:   macAddr,           // Client MAC as device ID
				ToDeviceID:     message.DeviceID,  // AP device ID
				FromInterface:  "wifi",            // Client interface (generic)
				ToInterface:    message.Interface, // AP interface
				ConnectionType: "wifi",
				IsDirectLink:   true,
				LastSeen:       message.Timestamp,
				Discovered:     message.Timestamp,
			}

			// Set connection metrics if available
			if rssi, ok := clientData["rssi"].(float64); ok {
				connection.Metrics.RSSI = int(rssi)
			}
			if txRate, ok := clientData["tx_rate"].(float64); ok {
				connection.Metrics.LinkSpeed = int(txRate)
			}
			if txBytes, ok := clientData["tx_bytes"].(float64); ok {
				connection.Metrics.TxBytes = int64(txBytes)
			}
			if rxBytes, ok := clientData["rx_bytes"].(float64); ok {
				connection.Metrics.RxBytes = int64(rxBytes)
			}

			connection.Metrics.LastUpdate = message.Timestamp

			// Save the connection
			if err := tp.topologyStorage.SaveDeviceConnection(connection); err != nil {
				log.Printf("Failed to save WiFi client connection: %v", err)
			}
		}
	}

	log.Printf("Processed WiFi clients for AP %s: %d clients", message.DeviceID, len(message.Clients))
	return nil
}

func (tp *TopologyProcessor) handleDeviceIdentity(topic string, payload []byte) error {
	var message struct {
		Schema         string                   `json:"schema"`
		Timestamp      int64                    `json:"timestamp"`
		MacAddress     string                   `json:"mac_address"`
		FriendlyName   string                   `json:"friendly_name,omitempty"`
		DeviceType     string                   `json:"device_type,omitempty"`
		Manufacturer   string                   `json:"manufacturer,omitempty"`
		Model          string                   `json:"model,omitempty"`
		Location       string                   `json:"location,omitempty"`
		Owner          string                   `json:"owner,omitempty"`
		Category       string                   `json:"category,omitempty"`
		Tags           []string                 `json:"tags,omitempty"`
		AutoDetected   bool                     `json:"auto_detected"`
		DetectionRules []map[string]interface{} `json:"detection_rules,omitempty"`
		Confidence     float64                  `json:"confidence"`
		FirstSeen      int64                    `json:"first_seen"`
		LastSeen       int64                    `json:"last_seen"`
		Notes          string                   `json:"notes,omitempty"`
	}

	if err := json.Unmarshal(payload, &message); err != nil {
		return fmt.Errorf("failed to unmarshal device identity message: %w", err)
	}

	// Convert detection rules
	var detectionRules []types.DetectionRuleMatch
	for _, ruleData := range message.DetectionRules {
		rule := types.DetectionRuleMatch{}
		if ruleID, ok := ruleData["rule_id"].(string); ok {
			rule.RuleID = ruleID
		}
		if ruleName, ok := ruleData["rule_name"].(string); ok {
			rule.RuleName = ruleName
		}
		if matchedField, ok := ruleData["matched_field"].(string); ok {
			rule.MatchedField = matchedField
		}
		if matchedValue, ok := ruleData["matched_value"].(string); ok {
			rule.MatchedValue = matchedValue
		}
		if confidence, ok := ruleData["confidence"].(float64); ok {
			rule.Confidence = confidence
		}
		if matchedAt, ok := ruleData["matched_at"].(float64); ok {
			rule.MatchedAt = time.UnixMilli(int64(matchedAt))
		}
		detectionRules = append(detectionRules, rule)
	}

	// Create device identity
	identity := &types.DeviceIdentity{
		MacAddress:     message.MacAddress,
		FriendlyName:   message.FriendlyName,
		DeviceType:     message.DeviceType,
		Manufacturer:   message.Manufacturer,
		Model:          message.Model,
		Location:       message.Location,
		Owner:          message.Owner,
		Category:       message.Category,
		Tags:           message.Tags,
		AutoDetected:   message.AutoDetected,
		DetectionRules: detectionRules,
		Confidence:     message.Confidence,
		FirstSeen:      time.UnixMilli(message.FirstSeen),
		LastSeen:       time.UnixMilli(message.LastSeen),
		LastUpdated:    time.UnixMilli(message.Timestamp),
		UpdatedBy:      "mqtt_message",
		Notes:          message.Notes,
	}

	// Save the device identity
	if err := tp.identityStorage.SaveDeviceIdentity(identity); err != nil {
		return fmt.Errorf("failed to save device identity: %w", err)
	}

	log.Printf("Processed device identity for MAC %s", message.MacAddress)
	return nil
}

func (tp *TopologyProcessor) handleNetworkDiagnostics(topic string, payload []byte) error {
	var message map[string]interface{}

	if err := json.Unmarshal(payload, &message); err != nil {
		return fmt.Errorf("failed to unmarshal network diagnostics message: %w", err)
	}

	// For now, just log the diagnostics message
	// In a full implementation, this would be stored and processed for network health analysis
	if deviceID, ok := message["device_id"].(string); ok {
		log.Printf("Received network diagnostics from device %s", deviceID)

		// Log specific diagnostic results
		if speedTest, ok := message["speed_test"].(map[string]interface{}); ok {
			if download, ok := speedTest["download_mbps"].(float64); ok {
				log.Printf("Speed test - Download: %.2f Mbps", download)
			}
			if upload, ok := speedTest["upload_mbps"].(float64); ok {
				log.Printf("Speed test - Upload: %.2f Mbps", upload)
			}
		}

		if wanTest, ok := message["wan_test"].(map[string]interface{}); ok {
			if connected, ok := wanTest["wan_connected"].(bool); ok {
				log.Printf("WAN connectivity: %t", connected)
			}
		}
	}

	return nil
}

func (tp *TopologyProcessor) handleQoSInfo(topic string, payload []byte) error {
	var message map[string]interface{}

	if err := json.Unmarshal(payload, &message); err != nil {
		return fmt.Errorf("failed to unmarshal QoS info message: %w", err)
	}

	// For now, just log the QoS information
	// In a full implementation, this would be stored and analyzed for traffic patterns
	if deviceID, ok := message["device_id"].(string); ok {
		log.Printf("Received QoS information from device %s", deviceID)

		if trafficStats, ok := message["traffic_stats"].(map[string]interface{}); ok {
			if totalBandwidth, ok := trafficStats["total_bandwidth"].(float64); ok {
				log.Printf("Total bandwidth: %.2f Mbps", totalBandwidth)
			}
			if usedBandwidth, ok := trafficStats["used_bandwidth"].(float64); ok {
				log.Printf("Used bandwidth: %.2f Mbps", usedBandwidth)
			}
		}
	}

	return nil
}

// Helper methods for data conversion

func (tp *TopologyProcessor) convertToNetworkDevice(deviceID string, deviceInfo map[string]interface{}, interfaces []map[string]interface{}, routingInfo map[string]interface{}, bridgeInfo map[string]interface{}) (*types.NetworkDevice, error) {
	device := &types.NetworkDevice{
		DeviceID:     deviceID,
		Interfaces:   make(map[string]types.NetworkIface),
		Capabilities: []string{},
	}

	// Extract device info
	if deviceType, ok := deviceInfo["device_type"].(string); ok {
		device.DeviceType = deviceType
	}
	if primaryMAC, ok := deviceInfo["primary_mac"].(string); ok {
		device.PrimaryMAC = primaryMAC
	}
	if hostname, ok := deviceInfo["hostname"].(string); ok {
		device.Hostname = hostname
	}
	if manufacturer, ok := deviceInfo["manufacturer"].(string); ok {
		device.Manufacturer = manufacturer
	}
	if model, ok := deviceInfo["model"].(string); ok {
		device.Model = model
	}
	if location, ok := deviceInfo["location"].(string); ok {
		device.Location = location
	}
	if role, ok := deviceInfo["role"].(string); ok {
		device.Role = types.DeviceRole(role)
	}
	if capabilities, ok := deviceInfo["capabilities"].([]interface{}); ok {
		for _, cap := range capabilities {
			if capStr, ok := cap.(string); ok {
				device.Capabilities = append(device.Capabilities, capStr)
			}
		}
	}

	// Convert interfaces
	for _, ifaceData := range interfaces {
		iface := tp.convertToNetworkIface(ifaceData)
		if name, ok := ifaceData["name"].(string); ok {
			device.Interfaces[name] = iface
		}
	}

	// Convert routing info
	if routingInfo != nil {
		device.RoutingInfo = tp.convertToRoutingInfo(routingInfo)
	}

	// Convert bridge info
	if bridgeInfo != nil {
		device.BridgeInfo = tp.convertToBridgeInfo(bridgeInfo)
	}

	return device, nil
}

func (tp *TopologyProcessor) convertToNetworkIface(ifaceData map[string]interface{}) types.NetworkIface {
	iface := types.NetworkIface{
		IPAddresses: []types.IPAddressInfo{},
	}

	if name, ok := ifaceData["name"].(string); ok {
		iface.Name = name
	}
	if ifaceType, ok := ifaceData["type"].(string); ok {
		iface.Type = ifaceType
	}
	if macAddr, ok := ifaceData["mac_address"].(string); ok {
		iface.MacAddress = macAddr
	}
	if status, ok := ifaceData["status"].(string); ok {
		iface.Status = status
	}
	if mtu, ok := ifaceData["mtu"].(float64); ok {
		iface.MTU = int(mtu)
	}
	if speed, ok := ifaceData["speed"].(float64); ok {
		iface.Speed = int(speed)
	}
	if duplex, ok := ifaceData["duplex"].(string); ok {
		iface.Duplex = duplex
	}
	if wifiMode, ok := ifaceData["wifi_mode"].(string); ok {
		iface.WiFiMode = wifiMode
	}
	if ssid, ok := ifaceData["ssid"].(string); ok {
		iface.SSID = ssid
	}
	if bssid, ok := ifaceData["bssid"].(string); ok {
		iface.BSSID = bssid
	}
	if channel, ok := ifaceData["channel"].(float64); ok {
		iface.Channel = int(channel)
	}
	if band, ok := ifaceData["band"].(string); ok {
		iface.Band = band
	}
	if rssi, ok := ifaceData["rssi"].(float64); ok {
		iface.RSSI = int(rssi)
	}
	if security, ok := ifaceData["security"].(string); ok {
		iface.Security = security
	}

	// Convert IP addresses
	if ipAddresses, ok := ifaceData["ip_addresses"].([]interface{}); ok {
		for _, ipData := range ipAddresses {
			if ipMap, ok := ipData.(map[string]interface{}); ok {
				ipInfo := types.IPAddressInfo{}
				if address, ok := ipMap["address"].(string); ok {
					ipInfo.Address = address
				}
				if network, ok := ipMap["network"].(string); ok {
					ipInfo.Network = network
				}
				if ipType, ok := ipMap["type"].(string); ok {
					ipInfo.Type = ipType
				}
				if gateway, ok := ipMap["gateway"].(string); ok {
					ipInfo.Gateway = gateway
				}
				if dnsServers, ok := ipMap["dns_servers"].([]interface{}); ok {
					for _, dns := range dnsServers {
						if dnsStr, ok := dns.(string); ok {
							ipInfo.DNSServers = append(ipInfo.DNSServers, dnsStr)
						}
					}
				}
				iface.IPAddresses = append(iface.IPAddresses, ipInfo)
			}
		}
	}

	// Convert statistics
	if stats, ok := ifaceData["statistics"].(map[string]interface{}); ok {
		if txBytes, ok := stats["tx_bytes"].(float64); ok {
			iface.TxBytes = int64(txBytes)
		}
		if rxBytes, ok := stats["rx_bytes"].(float64); ok {
			iface.RxBytes = int64(rxBytes)
		}
		if txPackets, ok := stats["tx_packets"].(float64); ok {
			iface.TxPackets = int64(txPackets)
		}
		if rxPackets, ok := stats["rx_packets"].(float64); ok {
			iface.RxPackets = int64(rxPackets)
		}
		if txErrors, ok := stats["tx_errors"].(float64); ok {
			iface.TxErrors = int64(txErrors)
		}
		if rxErrors, ok := stats["rx_errors"].(float64); ok {
			iface.RxErrors = int64(rxErrors)
		}
	}

	iface.LastUpdate = time.Now().UnixMilli()

	return iface
}

func (tp *TopologyProcessor) convertToRoutingInfo(routingData map[string]interface{}) *types.RoutingInfo {
	routingInfo := &types.RoutingInfo{
		RoutingTable: []types.RouteEntry{},
		NATRules:     []types.NATRule{},
	}

	if forwardingEnabled, ok := routingData["forwarding_enabled"].(bool); ok {
		routingInfo.ForwardingEnabled = forwardingEnabled
	}

	// Convert routing table
	if routingTable, ok := routingData["routing_table"].([]interface{}); ok {
		for _, routeData := range routingTable {
			if routeMap, ok := routeData.(map[string]interface{}); ok {
				route := types.RouteEntry{}
				if destination, ok := routeMap["destination"].(string); ok {
					route.Destination = destination
				}
				if gateway, ok := routeMap["gateway"].(string); ok {
					route.Gateway = gateway
				}
				if iface, ok := routeMap["interface"].(string); ok {
					route.Interface = iface
				}
				if metric, ok := routeMap["metric"].(float64); ok {
					route.Metric = int(metric)
				}
				if routeType, ok := routeMap["type"].(string); ok {
					route.Type = routeType
				}
				routingInfo.RoutingTable = append(routingInfo.RoutingTable, route)
			}
		}
	}

	// Convert NAT rules
	if natRules, ok := routingData["nat_rules"].([]interface{}); ok {
		for _, natData := range natRules {
			if natMap, ok := natData.(map[string]interface{}); ok {
				natRule := types.NATRule{}
				if natType, ok := natMap["type"].(string); ok {
					natRule.Type = natType
				}
				if sourceNet, ok := natMap["source_net"].(string); ok {
					natRule.SourceNet = sourceNet
				}
				if destNet, ok := natMap["dest_net"].(string); ok {
					natRule.DestNet = destNet
				}
				if iface, ok := natMap["interface"].(string); ok {
					natRule.Interface = iface
				}
				if protocol, ok := natMap["protocol"].(string); ok {
					natRule.Protocol = protocol
				}
				routingInfo.NATRules = append(routingInfo.NATRules, natRule)
			}
		}
	}

	// Convert DHCP server info
	if dhcpServer, ok := routingData["dhcp_server"].(map[string]interface{}); ok {
		dhcpInfo := &types.DHCPServerInfo{
			ActiveLeases: []types.DHCPLease{},
		}

		if enabled, ok := dhcpServer["enabled"].(bool); ok {
			dhcpInfo.Enabled = enabled
		}
		if ipRange, ok := dhcpServer["ip_range"].(string); ok {
			dhcpInfo.IPRange = ipRange
		}
		if subnetMask, ok := dhcpServer["subnet_mask"].(string); ok {
			dhcpInfo.SubnetMask = subnetMask
		}
		if leaseTime, ok := dhcpServer["lease_time"].(float64); ok {
			dhcpInfo.LeaseTime = int(leaseTime)
		}
		if gateway, ok := dhcpServer["gateway"].(string); ok {
			dhcpInfo.Gateway = gateway
		}
		if dnsServers, ok := dhcpServer["dns_servers"].([]interface{}); ok {
			for _, dns := range dnsServers {
				if dnsStr, ok := dns.(string); ok {
					dhcpInfo.DNSServers = append(dhcpInfo.DNSServers, dnsStr)
				}
			}
		}

		// Convert active leases
		if activeLeases, ok := dhcpServer["active_leases"].([]interface{}); ok {
			for _, leaseData := range activeLeases {
				if leaseMap, ok := leaseData.(map[string]interface{}); ok {
					lease := types.DHCPLease{}
					if macAddr, ok := leaseMap["mac_address"].(string); ok {
						lease.MacAddress = macAddr
					}
					if ipAddr, ok := leaseMap["ip_address"].(string); ok {
						lease.IPAddress = ipAddr
					}
					if hostname, ok := leaseMap["hostname"].(string); ok {
						lease.Hostname = hostname
					}
					if leaseStart, ok := leaseMap["lease_start"].(float64); ok {
						lease.LeaseStart = int64(leaseStart)
					}
					if leaseEnd, ok := leaseMap["lease_end"].(float64); ok {
						lease.LeaseEnd = int64(leaseEnd)
					}
					dhcpInfo.ActiveLeases = append(dhcpInfo.ActiveLeases, lease)
				}
			}
		}

		routingInfo.DHCPServer = dhcpInfo
	}

	return routingInfo
}

func (tp *TopologyProcessor) convertToBridgeInfo(bridgeData map[string]interface{}) *types.BridgeInfo {
	bridgeInfo := &types.BridgeInfo{
		BridgeTable: []types.BridgeTableEntry{},
	}

	if stpEnabled, ok := bridgeData["stp_enabled"].(bool); ok {
		bridgeInfo.STPEnabled = stpEnabled
	}
	if bridgeID, ok := bridgeData["bridge_id"].(string); ok {
		bridgeInfo.BridgeID = bridgeID
	}
	if rootBridge, ok := bridgeData["root_bridge"].(bool); ok {
		bridgeInfo.RootBridge = rootBridge
	}

	// Convert bridge table
	if bridgeTable, ok := bridgeData["bridge_table"].([]interface{}); ok {
		for _, entryData := range bridgeTable {
			if entryMap, ok := entryData.(map[string]interface{}); ok {
				entry := types.BridgeTableEntry{}
				if macAddr, ok := entryMap["mac_address"].(string); ok {
					entry.MacAddress = macAddr
				}
				if iface, ok := entryMap["interface"].(string); ok {
					entry.Interface = iface
				}
				if vlanID, ok := entryMap["vlan_id"].(float64); ok {
					entry.VlanID = int(vlanID)
				}
				if isLocal, ok := entryMap["is_local"].(bool); ok {
					entry.IsLocal = isLocal
				}
				if age, ok := entryMap["age"].(float64); ok {
					entry.Age = int(age)
				}
				bridgeInfo.BridgeTable = append(bridgeInfo.BridgeTable, entry)
			}
		}
	}

	return bridgeInfo
}

func (tp *TopologyProcessor) convertToDeviceConnection(fromDeviceID string, connData map[string]interface{}, timestamp int64) (*types.DeviceConnection, error) {
	connection := &types.DeviceConnection{
		FromDeviceID: fromDeviceID,
		LastSeen:     timestamp,
		Discovered:   timestamp,
	}

	// Extract connected device info
	if connectedDevice, ok := connData["connected_device"].(map[string]interface{}); ok {
		if deviceID, ok := connectedDevice["device_id"].(string); ok {
			connection.ToDeviceID = deviceID
		}
	}

	if connectionType, ok := connData["connection_type"].(string); ok {
		connection.ConnectionType = connectionType
	}
	if localInterface, ok := connData["local_interface"].(string); ok {
		connection.FromInterface = localInterface
	}
	if remoteInterface, ok := connData["remote_interface"].(string); ok {
		connection.ToInterface = remoteInterface
	}
	if isDirectLink, ok := connData["is_direct_link"].(bool); ok {
		connection.IsDirectLink = isDirectLink
	}

	// Generate connection ID
	connection.ID = fmt.Sprintf("%s-%s-%d", connection.FromDeviceID, connection.ToDeviceID, timestamp)

	// Convert metrics
	if metrics, ok := connData["metrics"].(map[string]interface{}); ok {
		if rssi, ok := metrics["rssi"].(float64); ok {
			connection.Metrics.RSSI = int(rssi)
		}
		if linkSpeed, ok := metrics["link_speed"].(float64); ok {
			connection.Metrics.LinkSpeed = int(linkSpeed)
		}
		if bandwidth, ok := metrics["bandwidth"].(float64); ok {
			connection.Metrics.Bandwidth = int(bandwidth)
		}
		if latency, ok := metrics["latency"].(float64); ok {
			connection.Metrics.Latency = latency
		}
		if txBytes, ok := metrics["tx_bytes"].(float64); ok {
			connection.Metrics.TxBytes = int64(txBytes)
		}
		if rxBytes, ok := metrics["rx_bytes"].(float64); ok {
			connection.Metrics.RxBytes = int64(rxBytes)
		}
		connection.Metrics.LastUpdate = timestamp
	}

	return connection, nil
}

func (tp *TopologyProcessor) convertToGatewayInfo(deviceID string, gatewayData map[string]interface{}, timestamp int64) (*types.GatewayInfo, error) {
	gatewayInfo := &types.GatewayInfo{
		DeviceID:   deviceID,
		LastCheck:  timestamp,
		DNSServers: []string{},
	}

	if ipAddress, ok := gatewayData["ip_address"].(string); ok {
		gatewayInfo.IPAddress = ipAddress
	}
	if externalIP, ok := gatewayData["external_ip"].(string); ok {
		gatewayInfo.ExternalIP = externalIP
	}
	if ispInfo, ok := gatewayData["isp_info"].(string); ok {
		gatewayInfo.ISPInfo = ispInfo
	}
	if connectionType, ok := gatewayData["connection_type"].(string); ok {
		gatewayInfo.ConnectionType = connectionType
	}
	if dnsServers, ok := gatewayData["dns_servers"].([]interface{}); ok {
		for _, dns := range dnsServers {
			if dnsStr, ok := dns.(string); ok {
				gatewayInfo.DNSServers = append(gatewayInfo.DNSServers, dnsStr)
			}
		}
	}

	return gatewayInfo, nil
}

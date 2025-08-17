package topology

import (
	"fmt"
	"io"
	"time"

	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// TopologyVisualizer provides various visualization formats for network topology
type TopologyVisualizer struct {
	topologyManager *Manager
	identityStorage *storage.IdentityStorage
	config          VisualizationConfig
}

// VisualizationConfig holds configuration for topology visualization
type VisualizationConfig struct {
	// Display options
	ShowOfflineDevices    bool
	ShowConnectionQuality bool
	ShowBandwidth        bool
	ShowSSIDs            bool
	ShowInterfaceDetails bool
	ShowTimestamps       bool
	
	// Filtering options
	MinConnectionQuality float64
	DeviceTypeFilter     []string
	SSIDFilter          []string
	TimeWindow          time.Duration
	
	// Layout options
	MaxWidth            int
	CompactMode         bool
	ColorEnabled        bool
	
	// Advanced options
	GroupBySSID         bool
	GroupByLocation     bool
	ShowMetrics         bool
	ShowAnomalies       bool
}

// VisualizationFormat defines output format for topology display
type VisualizationFormat string

const (
	FormatASCII     VisualizationFormat = "ascii"
	FormatTree      VisualizationFormat = "tree"
	FormatDOT       VisualizationFormat = "dot"
	// FormatJSON already defined in network_diagnostics_renderer.go
	FormatTable     VisualizationFormat = "table"
	FormatSummary   VisualizationFormat = "summary"
	FormatGraphViz  VisualizationFormat = "graphviz"
	FormatPlantUML  VisualizationFormat = "plantuml"
)

// TopologyGraph represents the complete topology for visualization
type TopologyGraph struct {
	Nodes       []TopologyNode       `json:"nodes"`
	Edges       []TopologyEdge       `json:"edges"`
	Groups      []TopologyGroup      `json:"groups"`
	Metadata    TopologyMetadata     `json:"metadata"`
	Stats       TopologyStats        `json:"stats"`
	Anomalies   []TopologyAnomaly    `json:"anomalies"`
}

// TopologyNode - Already defined in graph_builder.go

// TopologyEdge - Already defined in graph_builder.go

// TopologyGroup represents grouped nodes (e.g., by SSID or location)
type TopologyGroup struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	NodeIDs     []string `json:"node_ids"`
	Color       string   `json:"color,omitempty"`
	Collapsed   bool     `json:"collapsed"`
}

// TopologyMetadata provides context about the topology
type TopologyMetadata struct {
	GeneratedAt     time.Time             `json:"generated_at"`
	TimeWindow      time.Duration         `json:"time_window"`
	TotalDevices    int                   `json:"total_devices"`
	OnlineDevices   int                   `json:"online_devices"`
	OfflineDevices  int                   `json:"offline_devices"`
	TotalConnections int                  `json:"total_connections"`
	Filters         map[string]interface{} `json:"filters,omitempty"`
	Version         string                `json:"version"`
}

// TopologyStats provides statistical information
type TopologyStats struct {
	DevicesByType       map[string]int `json:"devices_by_type"`
	DevicesByStatus     map[string]int `json:"devices_by_status"`
	DevicesBySSID       map[string]int `json:"devices_by_ssid"`
	AverageQuality      float64        `json:"average_quality"`
	TotalBandwidth      float64        `json:"total_bandwidth"`
	AverageLatency      float64        `json:"average_latency"`
	PacketLossRate      float64        `json:"packet_loss_rate"`
	TopologyComplexity  float64        `json:"topology_complexity"`
}

// TopologyAnomaly represents detected anomalies in the topology
type TopologyAnomaly struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	NodeID      string                 `json:"node_id,omitempty"`
	EdgeID      string                 `json:"edge_id,omitempty"`
	DetectedAt  time.Time              `json:"detected_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BandwidthInfo represents bandwidth information for a node
type BandwidthInfo struct {
	Upload   float64 `json:"upload"`   // Mbps
	Download float64 `json:"download"` // Mbps
	Total    float64 `json:"total"`    // Mbps
}

// NodePosition - Already defined in graph_builder.go

// NewTopologyVisualizer creates a new topology visualizer
func NewTopologyVisualizer(
	manager *Manager,
	identityStorage *storage.IdentityStorage,
	config VisualizationConfig,
) *TopologyVisualizer {
	return &TopologyVisualizer{
		topologyManager: manager,
		identityStorage: identityStorage,
		config:          config,
	}
}

// GenerateTopologyGraph creates a topology graph from current network state
func (tv *TopologyVisualizer) GenerateTopologyGraph() (*TopologyGraph, error) {
	// Get current topology data
	currentTopology, err := tv.topologyManager.GetCurrentTopology()
	if err != nil {
		return nil, fmt.Errorf("failed to get current topology: %w", err)
	}

	graph := &TopologyGraph{
		Nodes:     []TopologyNode{},
		Edges:     []TopologyEdge{},
		Groups:    []TopologyGroup{},
		Metadata:  tv.buildMetadata(currentTopology),
		Stats:     TopologyStats{},
		Anomalies: []TopologyAnomaly{},
	}

	// Build nodes
	if err := tv.buildNodes(currentTopology, graph); err != nil {
		return nil, fmt.Errorf("failed to build nodes: %w", err)
	}

	// Build edges
	if err := tv.buildEdges(currentTopology, graph); err != nil {
		return nil, fmt.Errorf("failed to build edges: %w", err)
	}

	// Build groups
	if err := tv.buildGroups(graph); err != nil {
		return nil, fmt.Errorf("failed to build groups: %w", err)
	}

	// Calculate statistics
	tv.calculateStats(graph)

	// Detect anomalies
	if tv.config.ShowAnomalies {
		tv.detectAnomalies(graph)
	}

	// Apply layout
	tv.applyLayout(graph)

	return graph, nil
}

// RenderTopology renders topology in the specified format
func (tv *TopologyVisualizer) RenderTopology(
	format VisualizationFormat,
	writer io.Writer,
) error {
	graph, err := tv.GenerateTopologyGraph()
	if err != nil {
		return fmt.Errorf("failed to generate topology graph: %w", err)
	}

	switch format {
	case FormatASCII:
		return tv.renderASCII(graph, writer)
	case FormatTree:
		return tv.renderTree(graph, writer)
	case FormatDOT:
		return tv.renderDOT(graph, writer)
	// JSON format handled separately since FormatJSON is a ReportFormat
	// case FormatJSON:
	//	return tv.renderJSON(graph, writer)
	case FormatTable:
		return tv.renderTable(graph, writer)
	case FormatSummary:
		return tv.renderSummary(graph, writer)
	case FormatGraphViz:
		return tv.renderGraphViz(graph, writer)
	case FormatPlantUML:
		return tv.renderPlantUML(graph, writer)
	default:
		return fmt.Errorf("unsupported visualization format: %s", format)
	}
}

// buildNodes creates topology nodes from network topology
func (tv *TopologyVisualizer) buildNodes(topology *types.NetworkTopology, graph *TopologyGraph) error {
	for deviceID, device := range topology.Devices {
		// Apply filters
		if !tv.shouldIncludeDevice(device) {
			continue
		}

		// Get device identity
		identity, _ := tv.identityStorage.GetDeviceIdentity(device.PrimaryMAC)
		
		label := device.PrimaryMAC
		if identity != nil && identity.FriendlyName != "" {
			label = identity.FriendlyName
		}

		status := "offline"
		if device.Online {
			status = "online"
		}
		
		node := TopologyNode{
			ID:     deviceID,
			Device: device,
			Properties: NodeProperties{
				Label:      label,
				Status:     status,
				Importance: 0.5, // Default importance
				Color:      "#808080",
				Shape:      "circle",
			},
			Position:    NodePosition{},
			Connections: []*TopologyEdge{},
		}

		// Add WiFi specific information
		// TODO: Add WiFi info to node properties

		// Add bandwidth information
		// TODO: Add bandwidth info to node properties
		/*
		if tv.config.ShowBandwidth && device.NetworkMetrics != nil {
			node.Bandwidth = BandwidthInfo{
				Upload:   device.NetworkMetrics.UploadMbps,
				Download: device.NetworkMetrics.DownloadMbps,
				Total:    device.NetworkMetrics.UploadMbps + device.NetworkMetrics.DownloadMbps,
			}
		}

		// Add location if available
		if identity != nil {
			node.Location = identity.Location
		}
		*/

		// Add metadata
		/*
		if tv.config.ShowTimestamps {
			node.Metadata["first_seen"] = device.FirstSeen
			node.Metadata["last_updated"] = device.LastUpdated
		}

		if tv.config.ShowInterfaceDetails && len(device.Interfaces) > 0 {
			node.Metadata["interfaces"] = device.Interfaces
		}
		*/

		graph.Nodes = append(graph.Nodes, node)
	}

	return nil
}

// buildEdges creates topology edges from connections
func (tv *TopologyVisualizer) buildEdges(topology *types.NetworkTopology, graph *TopologyGraph) error {
	edgeID := 0
	
	for _, connection := range topology.Connections {
		// Apply quality filter
		// TODO: Add quality field to connection
		// if connection.Quality < tv.config.MinConnectionQuality {
		//	continue
		// }

		// Verify both nodes exist
		sourceExists := tv.nodeExists(graph, connection.FromDeviceID)
		targetExists := tv.nodeExists(graph, connection.ToDeviceID)
		
		if !sourceExists || !targetExists {
			continue
		}

		// Find the actual nodes
		var fromNode, toNode *TopologyNode
		for i := range graph.Nodes {
			if graph.Nodes[i].ID == connection.FromDeviceID {
				fromNode = &graph.Nodes[i]
			}
			if graph.Nodes[i].ID == connection.ToDeviceID {
				toNode = &graph.Nodes[i]
			}
		}
		
		if fromNode == nil || toNode == nil {
			continue
		}

		edge := TopologyEdge{
			ID:         fmt.Sprintf("edge_%d", edgeID),
			From:       fromNode,
			To:         toNode,
			Connection: connection,
			Properties: EdgeProperties{
				Quality: "fair", // Default quality
				Width:   2,
				Color:   "#808080",
				Style:   "solid",
			},
		}

		// Add performance metrics
		// TODO: Add these fields to TopologyEdge
		/*
		if connection.Metrics != nil {
			edge.Bandwidth = connection.Metrics.BandwidthMbps
			edge.Latency = connection.Metrics.LatencyMs
			edge.PacketLoss = connection.Metrics.PacketLoss
		}

		// Add protocol and interface information
		if connection.Protocol != "" {
			edge.Protocol = connection.Protocol
		}
		if connection.Interface != "" {
			edge.Interface = connection.Interface
		}
		*/

		graph.Edges = append(graph.Edges, edge)
		edgeID++
	}

	return nil
}

// buildGroups creates topology groups based on configuration
func (tv *TopologyVisualizer) buildGroups(graph *TopologyGraph) error {
	if tv.config.GroupBySSID {
		tv.buildSSIDGroups(graph)
	}

	if tv.config.GroupByLocation {
		tv.buildLocationGroups(graph)
	}

	return nil
}

// buildSSIDGroups creates groups based on SSID
func (tv *TopologyVisualizer) buildSSIDGroups(graph *TopologyGraph) {
	ssidGroups := make(map[string][]string)

	// TODO: Add SSID field to node
	/*
	for _, node := range graph.Nodes {
		if node.SSID != "" {
			ssidGroups[node.SSID] = append(ssidGroups[node.SSID], node.ID)
		}
	}
	*/

	groupID := 0
	for ssid, nodeIDs := range ssidGroups {
		if len(nodeIDs) > 1 {
			group := TopologyGroup{
				ID:      fmt.Sprintf("ssid_group_%d", groupID),
				Label:   fmt.Sprintf("SSID: %s", ssid),
				Type:    "ssid",
				NodeIDs: nodeIDs,
				Color:   tv.getGroupColor("ssid", groupID),
			}
			graph.Groups = append(graph.Groups, group)
			groupID++
		}
	}
}

// buildLocationGroups creates groups based on location
func (tv *TopologyVisualizer) buildLocationGroups(graph *TopologyGraph) {
	locationGroups := make(map[string][]string)

	// TODO: Add Location field to node
	/*
	for _, node := range graph.Nodes {
		if node.Location != "" {
			locationGroups[node.Location] = append(locationGroups[node.Location], node.ID)
		}
	}
	*/

	groupID := 0
	locationGroups = make(map[string][]string) // temp for compilation
	for location, nodeIDs := range locationGroups {
		if len(nodeIDs) > 1 {
			group := TopologyGroup{
				ID:      fmt.Sprintf("location_group_%d", groupID),
				Label:   fmt.Sprintf("Location: %s", location),
				Type:    "location",
				NodeIDs: nodeIDs,
				Color:   tv.getGroupColor("location", groupID),
			}
			graph.Groups = append(graph.Groups, group)
			groupID++
		}
	}
}

// calculateStats calculates topology statistics
func (tv *TopologyVisualizer) calculateStats(graph *TopologyGraph) {
	stats := &graph.Stats
	stats.DevicesByType = make(map[string]int)
	stats.DevicesByStatus = make(map[string]int)
	stats.DevicesBySSID = make(map[string]int)

	var totalQuality, totalBandwidth, totalLatency, totalPacketLoss float64
	var qualityCount, bandwidthCount, latencyCount, packetLossCount int

	// Calculate node statistics
	for _, node := range graph.Nodes {
		deviceType := "unknown"
		if node.Device != nil {
			deviceType = node.Device.DeviceType
		}
		stats.DevicesByType[deviceType]++
		stats.DevicesByStatus[node.Properties.Status]++
		
		// TODO: Add SSID field to node
		// if node.SSID != "" {
		//	stats.DevicesBySSID[node.SSID]++
		// }

		if node.Properties.Importance > 0 {
			totalQuality += node.Properties.Importance
			qualityCount++
		}

		// TODO: Add Bandwidth field to node
		/*
		if node.Bandwidth.Total > 0 {
			totalBandwidth += node.Bandwidth.Total
			bandwidthCount++
		}
		*/
	}

	// Calculate edge statistics
	// TODO: Add Latency and PacketLoss fields to TopologyEdge
	/*
	for _, edge := range graph.Edges {
		if edge.Latency > 0 {
			totalLatency += edge.Latency
			latencyCount++
		}

		if edge.PacketLoss > 0 {
			totalPacketLoss += edge.PacketLoss
			packetLossCount++
		}
	}
	*/

	// Calculate averages
	if qualityCount > 0 {
		stats.AverageQuality = totalQuality / float64(qualityCount)
	}
	if bandwidthCount > 0 {
		stats.TotalBandwidth = totalBandwidth
	}
	if latencyCount > 0 {
		stats.AverageLatency = totalLatency / float64(latencyCount)
	}
	if packetLossCount > 0 {
		stats.PacketLossRate = totalPacketLoss / float64(packetLossCount)
	}

	// Calculate topology complexity (nodes * connections / theoretical maximum)
	nodeCount := len(graph.Nodes)
	edgeCount := len(graph.Edges)
	maxPossibleEdges := nodeCount * (nodeCount - 1) / 2
	
	if maxPossibleEdges > 0 {
		stats.TopologyComplexity = float64(edgeCount) / float64(maxPossibleEdges)
	}
}

// detectAnomalies detects anomalies in the topology
func (tv *TopologyVisualizer) detectAnomalies(graph *TopologyGraph) {
	anomalyID := 0

	// Detect isolated nodes
	for _, node := range graph.Nodes {
		if tv.isNodeIsolated(node.ID, graph) && node.Properties.Status == "online" {
			anomaly := TopologyAnomaly{
				ID:          fmt.Sprintf("anomaly_%d", anomalyID),
				Type:        "isolated_node",
				Severity:    "warning",
				Description: fmt.Sprintf("Device %s appears to be isolated", node.Properties.Label),
				NodeID:      node.ID,
				DetectedAt:  time.Now(),
			}
			graph.Anomalies = append(graph.Anomalies, anomaly)
			anomalyID++
		}
	}

	// Detect poor quality connections
	// TODO: Add numerical quality field to TopologyEdge
	/*
	for _, edge := range graph.Edges {
		if edge.Quality < 0.3 {
			anomaly := TopologyAnomaly{
				ID:          fmt.Sprintf("anomaly_%d", anomalyID),
				Type:        "poor_quality_connection",
				Severity:    "error",
				Description: fmt.Sprintf("Poor connection quality (%.2f) between devices", edge.Quality),
				EdgeID:      edge.ID,
				DetectedAt:  time.Now(),
			}
			graph.Anomalies = append(graph.Anomalies, anomaly)
			anomalyID++
		}
	}
	*/

	// Detect high latency connections
	// TODO: Add Latency field to TopologyEdge
	/*
	for _, edge := range graph.Edges {
		if edge.Latency > 100 {
			anomaly := TopologyAnomaly{
				ID:          fmt.Sprintf("anomaly_%d", anomalyID),
				Type:        "high_latency",
				Severity:    "warning",
				Description: fmt.Sprintf("High latency (%.2fms) detected", edge.Latency),
				EdgeID:      edge.ID,
				DetectedAt:  time.Now(),
			}
			graph.Anomalies = append(graph.Anomalies, anomaly)
			anomalyID++
		}
	}
	*/
}

// applyLayout applies a layout algorithm to position nodes
func (tv *TopologyVisualizer) applyLayout(graph *TopologyGraph) {
	// Simple force-directed layout
	nodeCount := len(graph.Nodes)
	if nodeCount == 0 {
		return
	}

	// Initialize positions in a circle
	for i := range graph.Nodes {
		angle := 2 * 3.14159 * float64(i) / float64(nodeCount)
		radius := 100.0
		
		graph.Nodes[i].Position = NodePosition{
			X: radius * float64(angle),
			Y: radius * float64(angle),
		}
	}

	// TODO: Implement proper force-directed layout algorithm
	// For now, use simple circular layout
}

// Helper methods

func (tv *TopologyVisualizer) shouldIncludeDevice(device *types.NetworkDevice) bool {
	// Filter by device status
	if !tv.config.ShowOfflineDevices && !device.Online {
		return false
	}

	// Filter by device type
	if len(tv.config.DeviceTypeFilter) > 0 {
		found := false
		for _, filterType := range tv.config.DeviceTypeFilter {
			if device.DeviceType == filterType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by SSID
	// TODO: Add WiFiInfo field to NetworkDevice
	/*
	if len(tv.config.SSIDFilter) > 0 && device.WiFiInfo != nil {
		found := false
		for _, filterSSID := range tv.config.SSIDFilter {
			if device.WiFiInfo.SSID == filterSSID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	*/

	// Filter by time window
	if tv.config.TimeWindow > 0 {
		cutoff := time.Now().Add(-tv.config.TimeWindow)
		lastSeenTime := time.Unix(device.LastSeen, 0)
		if lastSeenTime.Before(cutoff) {
			return false
		}
	}

	return true
}

func (tv *TopologyVisualizer) nodeExists(graph *TopologyGraph, nodeID string) bool {
	for _, node := range graph.Nodes {
		if node.ID == nodeID {
			return true
		}
	}
	return false
}

func (tv *TopologyVisualizer) isNodeIsolated(nodeID string, graph *TopologyGraph) bool {
	for _, edge := range graph.Edges {
		if edge.From.ID == nodeID || edge.To.ID == nodeID {
			return false
		}
	}
	return true
}

func (tv *TopologyVisualizer) getGroupColor(groupType string, index int) string {
	colors := []string{
		"#FF6B6B", "#4ECDC4", "#45B7D1", "#FFA07A",
		"#98D8C8", "#F7DC6F", "#BB8FCE", "#85C1E9",
	}
	return colors[index%len(colors)]
}

func (tv *TopologyVisualizer) buildMetadata(topology *types.NetworkTopology) TopologyMetadata {
	onlineCount := 0
	offlineCount := 0
	
	for _, device := range topology.Devices {
		if device.Online {
			onlineCount++
		} else {
			offlineCount++
		}
	}

	return TopologyMetadata{
		GeneratedAt:      time.Now(),
		TimeWindow:       tv.config.TimeWindow,
		TotalDevices:     len(topology.Devices),
		OnlineDevices:    onlineCount,
		OfflineDevices:   offlineCount,
		TotalConnections: len(topology.Connections),
		Version:          "1.0",
	}
}


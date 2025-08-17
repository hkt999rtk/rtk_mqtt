package topology

import (
	"fmt"
	"log"
	"sort"
	"time"

	"rtk_controller/pkg/types"
)

// GraphBuilder constructs and maintains network topology graphs
type GraphBuilder struct {
	// Graph representation
	nodes map[string]*TopologyNode
	edges map[string]*TopologyEdge
	
	// Graph metadata
	metadata GraphMetadata
	
	// Configuration
	config GraphConfig
	
	// Identity management
	identityManager *DeviceIdentityManager
}

// TopologyNode represents a device in the topology graph
type TopologyNode struct {
	ID          string
	Device      *types.NetworkDevice
	Position    NodePosition
	Properties  NodeProperties
	Connections []*TopologyEdge
}

// TopologyEdge represents a connection between devices
type TopologyEdge struct {
	ID          string
	From        *TopologyNode
	To          *TopologyNode
	Connection  types.DeviceConnection
	Properties  EdgeProperties
}

// NodePosition holds graph layout position information
type NodePosition struct {
	X         float64
	Y         float64
	Z         float64 // For 3D layouts
	Layer     int     // Network layer (physical, data link, network, etc.)
	Group     string  // Grouping identifier (subnet, building, etc.)
}

// NodeProperties holds additional node visualization properties
type NodeProperties struct {
	Size         float64
	Color        string
	Shape        string
	Label        string
	Tooltip      string
	IconType     string
	Status       string // online, offline, warning, error
	Importance   float64 // For highlighting critical nodes
}

// EdgeProperties holds additional edge visualization properties
type EdgeProperties struct {
	Width       float64
	Color       string
	Style       string // solid, dashed, dotted
	Label       string
	Tooltip     string
	Animated    bool
	Bidirectional bool
	Quality     string // excellent, good, fair, poor
}

// GraphMetadata holds graph-wide information
type GraphMetadata struct {
	TotalNodes        int
	TotalEdges        int
	ConnectedComponents int
	MaxPathLength     int
	GraphDensity      float64
	LastUpdate        time.Time
	BuildDuration     time.Duration
}

// GraphConfig holds graph building configuration
type GraphConfig struct {
	// Layout settings
	EnableAutoLayout      bool
	LayoutAlgorithm      string  // force, hierarchical, circular, grid
	NodeSpacing          float64
	EdgeLength           float64
	LayerSeparation      float64
	
	// Visual settings
	ShowOfflineDevices   bool
	ShowWeakConnections  bool
	MinConnectionQuality float64
	NodeSizeBasedOnConnections bool
	EdgeWidthBasedOnBandwidth  bool
	
	// Grouping and clustering
	EnableClustering     bool
	ClusterBySubnet      bool
	ClusterByDeviceType  bool
	ClusterByLocation    bool
	MaxClusterSize       int
	
	// Performance settings
	MaxNodes             int
	MaxEdges             int
	UpdateThrottleMs     int
}

// NewGraphBuilder creates a new topology graph builder
func NewGraphBuilder(config GraphConfig, identityManager *DeviceIdentityManager) *GraphBuilder {
	return &GraphBuilder{
		nodes:           make(map[string]*TopologyNode),
		edges:           make(map[string]*TopologyEdge),
		config:          config,
		identityManager: identityManager,
		metadata: GraphMetadata{
			LastUpdate: time.Now(),
		},
	}
}

// BuildGraph constructs the topology graph from devices and connections
func (gb *GraphBuilder) BuildGraph(devices map[string]*types.NetworkDevice, connections []types.DeviceConnection) error {
	startTime := time.Now()
	
	log.Printf("Building topology graph with %d devices and %d connections", len(devices), len(connections))
	
	// Clear existing graph
	gb.clearGraph()
	
	// Add nodes (devices)
	if err := gb.addNodes(devices); err != nil {
		return fmt.Errorf("failed to add nodes: %w", err)
	}
	
	// Add edges (connections)
	if err := gb.addEdges(connections); err != nil {
		return fmt.Errorf("failed to add edges: %w", err)
	}
	
	// Apply layout algorithm
	if gb.config.EnableAutoLayout {
		if err := gb.applyLayout(); err != nil {
			log.Printf("Warning: layout algorithm failed: %v", err)
		}
	}
	
	// Calculate graph properties
	gb.calculateGraphProperties()
	
	// Update metadata
	gb.metadata.TotalNodes = len(gb.nodes)
	gb.metadata.TotalEdges = len(gb.edges)
	gb.metadata.LastUpdate = time.Now()
	gb.metadata.BuildDuration = time.Since(startTime)
	
	log.Printf("Topology graph built successfully in %v: %d nodes, %d edges", 
		gb.metadata.BuildDuration, gb.metadata.TotalNodes, gb.metadata.TotalEdges)
	
	return nil
}

// GetGraph returns the current graph structure
func (gb *GraphBuilder) GetGraph() (map[string]*TopologyNode, map[string]*TopologyEdge) {
	// Return copies to prevent external modification
	nodesCopy := make(map[string]*TopologyNode)
	edgesCopy := make(map[string]*TopologyEdge)
	
	for id, node := range gb.nodes {
		nodeCopy := *node
		nodesCopy[id] = &nodeCopy
	}
	
	for id, edge := range gb.edges {
		edgeCopy := *edge
		edgesCopy[id] = &edgeCopy
	}
	
	return nodesCopy, edgesCopy
}

// GetGraphMetadata returns graph metadata and statistics
func (gb *GraphBuilder) GetGraphMetadata() GraphMetadata {
	return gb.metadata
}

// GetNodesByType returns nodes filtered by device type
func (gb *GraphBuilder) GetNodesByType(deviceType string) []*TopologyNode {
	var nodes []*TopologyNode
	
	for _, node := range gb.nodes {
		if node.Device.DeviceType == deviceType {
			nodes = append(nodes, node)
		}
	}
	
	return nodes
}

// GetCriticalPath finds the path between two devices
func (gb *GraphBuilder) GetCriticalPath(fromDeviceID, toDeviceID string) ([]*TopologyNode, []*TopologyEdge, error) {
	fromNode, exists := gb.nodes[fromDeviceID]
	if !exists {
		return nil, nil, fmt.Errorf("source device not found: %s", fromDeviceID)
	}
	
	toNode, exists := gb.nodes[toDeviceID]
	if !exists {
		return nil, nil, fmt.Errorf("destination device not found: %s", toDeviceID)
	}
	
	// Use Dijkstra's algorithm to find shortest path
	path, edges := gb.findShortestPath(fromNode, toNode)
	return path, edges, nil
}

// Private methods

func (gb *GraphBuilder) clearGraph() {
	gb.nodes = make(map[string]*TopologyNode)
	gb.edges = make(map[string]*TopologyEdge)
}

func (gb *GraphBuilder) addNodes(devices map[string]*types.NetworkDevice) error {
	for deviceID, device := range devices {
		// Skip offline devices if configured
		if !gb.config.ShowOfflineDevices && !device.Online {
			continue
		}
		
		// Check node limit
		if len(gb.nodes) >= gb.config.MaxNodes {
			log.Printf("Warning: reached maximum node limit (%d)", gb.config.MaxNodes)
			break
		}
		
		node := &TopologyNode{
			ID:          deviceID,
			Device:      device,
			Position:    gb.calculateNodePosition(device),
			Properties:  gb.calculateNodeProperties(device),
			Connections: []*TopologyEdge{},
		}
		
		gb.nodes[deviceID] = node
	}
	
	return nil
}

func (gb *GraphBuilder) addEdges(connections []types.DeviceConnection) error {
	for _, conn := range connections {
		// Skip weak connections if configured
		// TODO: Check conn.Metrics.Confidence once field is added
		if 0.5 < gb.config.MinConnectionQuality {
			continue
		}
		
		// Check edge limit
		if len(gb.edges) >= gb.config.MaxEdges {
			log.Printf("Warning: reached maximum edge limit (%d)", gb.config.MaxEdges)
			break
		}
		
		// Get nodes
		fromNode, fromExists := gb.nodes[conn.FromDeviceID]
		toNode, toExists := gb.nodes[conn.ToDeviceID]
		
		if !fromExists || !toExists {
			continue // Skip if either node doesn't exist
		}
		
		edge := &TopologyEdge{
			ID:         conn.ID,
			From:       fromNode,
			To:         toNode,
			Connection: conn,
			Properties: gb.calculateEdgeProperties(conn),
		}
		
		gb.edges[conn.ID] = edge
		
		// Add edge to node connections
		fromNode.Connections = append(fromNode.Connections, edge)
		toNode.Connections = append(toNode.Connections, edge)
	}
	
	return nil
}

func (gb *GraphBuilder) calculateNodePosition(device *types.NetworkDevice) NodePosition {
	position := NodePosition{
		Layer: gb.getDeviceLayer(device),
		Group: gb.getDeviceGroup(device),
	}
	
	// Initial positioning based on device type and role
	switch device.Role {
	case "gateway": // TODO: Add DeviceRoleGateway constant
		position.X = 0
		position.Y = 0
		position.Layer = 0
	case "access_point": // TODO: Add DeviceRoleAccessPoint constant
		position.Layer = 1
	case "switch": // TODO: Add DeviceRoleSwitch constant
		position.Layer = 1
	case "client": // TODO: Add DeviceRoleClient
		position.Layer = 2
	default:
		position.Layer = 2
	}
	
	return position
}

func (gb *GraphBuilder) calculateNodeProperties(device *types.NetworkDevice) NodeProperties {
	properties := NodeProperties{
		Label:   gb.getDeviceLabel(device),
		Tooltip: gb.getDeviceTooltip(device),
	}
	
	// Set status based on device state
	if device.Online {
		properties.Status = "online"
		properties.Color = "#4CAF50" // Green
	} else {
		properties.Status = "offline"
		properties.Color = "#F44336" // Red
	}
	
	// Set size based on number of connections if configured
	if gb.config.NodeSizeBasedOnConnections {
		connectionCount := len(device.Interfaces)
		properties.Size = 10 + float64(connectionCount)*2
	} else {
		properties.Size = 15
	}
	
	// Set shape and icon based on device type
	switch device.DeviceType {
	case "router":
		properties.Shape = "diamond"
		properties.IconType = "router"
	case "ap":
		properties.Shape = "triangle"
		properties.IconType = "wifi"
	case "switch":
		properties.Shape = "square"
		properties.IconType = "switch"
	case "client":
		properties.Shape = "circle"
		properties.IconType = "device"
	default:
		properties.Shape = "circle"
		properties.IconType = "device"
	}
	
	// Calculate importance based on device role and connections
	properties.Importance = gb.calculateDeviceImportance(device)
	
	return properties
}

func (gb *GraphBuilder) calculateEdgeProperties(conn types.DeviceConnection) EdgeProperties {
	properties := EdgeProperties{
		Label:    gb.getConnectionLabel(conn),
		Tooltip:  gb.getConnectionTooltip(conn),
		Bidirectional: true,
	}
	
	// Set width based on bandwidth if configured
	if gb.config.EdgeWidthBasedOnBandwidth && conn.Metrics.Bandwidth > 0 {
		properties.Width = 1 + float64(conn.Metrics.Bandwidth)/100
		if properties.Width > 10 {
			properties.Width = 10
		}
	} else {
		properties.Width = 2
	}
	
	// Set color and style based on connection quality
	confidence := 0.5 // TODO: conn.Metrics.Confidence
	if confidence >= 0.8 {
		properties.Color = "#4CAF50" // Green
		properties.Quality = "excellent"
		properties.Style = "solid"
	} else if confidence >= 0.6 {
		properties.Color = "#FFC107" // Yellow
		properties.Quality = "good"
		properties.Style = "solid"
	} else if confidence >= 0.4 {
		properties.Color = "#FF9800" // Orange
		properties.Quality = "fair"
		properties.Style = "dashed"
	} else {
		properties.Color = "#F44336" // Red
		properties.Quality = "poor"
		properties.Style = "dotted"
	}
	
	// Set animation for active connections
	now := time.Now().UnixMilli()
	if (now - conn.LastSeen) < 60000 { // Active within last minute
		properties.Animated = true
	}
	
	return properties
}

func (gb *GraphBuilder) applyLayout() error {
	switch gb.config.LayoutAlgorithm {
	case "force":
		return gb.applyForceLayout()
	case "hierarchical":
		return gb.applyHierarchicalLayout()
	case "circular":
		return gb.applyCircularLayout()
	case "grid":
		return gb.applyGridLayout()
	default:
		return gb.applyForceLayout()
	}
}

func (gb *GraphBuilder) applyForceLayout() error {
	// Simplified force-directed layout algorithm
	iterations := 100
	
	for i := 0; i < iterations; i++ {
		// Apply repulsion forces between all nodes
		for _, node1 := range gb.nodes {
			for _, node2 := range gb.nodes {
				if node1.ID == node2.ID {
					continue
				}
				
				dx := node1.Position.X - node2.Position.X
				dy := node1.Position.Y - node2.Position.Y
				distance := dx*dx + dy*dy
				
				if distance < 0.01 {
					distance = 0.01
				}
				
				force := gb.config.NodeSpacing / distance
				node1.Position.X += dx * force * 0.01
				node1.Position.Y += dy * force * 0.01
			}
		}
		
		// Apply attraction forces along edges
		for _, edge := range gb.edges {
			dx := edge.To.Position.X - edge.From.Position.X
			dy := edge.To.Position.Y - edge.From.Position.Y
			distance := dx*dx + dy*dy
			
			force := (distance - gb.config.EdgeLength) * 0.01
			
			edge.From.Position.X += dx * force
			edge.From.Position.Y += dy * force
			edge.To.Position.X -= dx * force
			edge.To.Position.Y -= dy * force
		}
	}
	
	return nil
}

func (gb *GraphBuilder) applyHierarchicalLayout() error {
	// Arrange nodes in layers based on device role/type
	layers := make(map[int][]*TopologyNode)
	
	for _, node := range gb.nodes {
		layer := node.Position.Layer
		layers[layer] = append(layers[layer], node)
	}
	
	// Position nodes in each layer
	for layer, nodes := range layers {
		y := float64(layer) * gb.config.LayerSeparation
		
		for i, node := range nodes {
			node.Position.X = float64(i-len(nodes)/2) * gb.config.NodeSpacing
			node.Position.Y = y
		}
	}
	
	return nil
}

func (gb *GraphBuilder) applyCircularLayout() error {
	// Arrange nodes in concentric circles
	centerNodes := gb.GetNodesByType("router")
	if len(centerNodes) == 0 {
		centerNodes = gb.GetNodesByType("ap")
	}
	
	// Place central nodes at origin
	for i, node := range centerNodes {
		angle := float64(i) * 2.0 * 3.14159 / float64(len(centerNodes))
		node.Position.X = 10 * gb.math_cos(angle)
		node.Position.Y = 10 * gb.math_sin(angle)
	}
	
	// Place other nodes in outer circle
	otherNodes := []*TopologyNode{}
	for _, node := range gb.nodes {
		isCenter := false
		for _, center := range centerNodes {
			if node.ID == center.ID {
				isCenter = true
				break
			}
		}
		if !isCenter {
			otherNodes = append(otherNodes, node)
		}
	}
	
	for i, node := range otherNodes {
		angle := float64(i) * 2.0 * 3.14159 / float64(len(otherNodes))
		radius := 50.0
		node.Position.X = radius * gb.math_cos(angle)
		node.Position.Y = radius * gb.math_sin(angle)
	}
	
	return nil
}

func (gb *GraphBuilder) applyGridLayout() error {
	// Arrange nodes in a grid pattern
	nodeList := make([]*TopologyNode, 0, len(gb.nodes))
	for _, node := range gb.nodes {
		nodeList = append(nodeList, node)
	}
	
	// Sort by device type for consistent layout
	sort.Slice(nodeList, func(i, j int) bool {
		return nodeList[i].Device.DeviceType < nodeList[j].Device.DeviceType
	})
	
	gridSize := int(gb.math_sqrt(float64(len(nodeList)))) + 1
	
	for i, node := range nodeList {
		row := i / gridSize
		col := i % gridSize
		
		node.Position.X = float64(col) * gb.config.NodeSpacing
		node.Position.Y = float64(row) * gb.config.NodeSpacing
	}
	
	return nil
}

func (gb *GraphBuilder) calculateGraphProperties() {
	// Calculate connected components using DFS
	visited := make(map[string]bool)
	components := 0
	
	for nodeID := range gb.nodes {
		if !visited[nodeID] {
			gb.dfsMarkComponent(nodeID, visited)
			components++
		}
	}
	
	gb.metadata.ConnectedComponents = components
	
	// Calculate graph density
	maxEdges := len(gb.nodes) * (len(gb.nodes) - 1) / 2
	if maxEdges > 0 {
		gb.metadata.GraphDensity = float64(len(gb.edges)) / float64(maxEdges)
	}
}

func (gb *GraphBuilder) dfsMarkComponent(nodeID string, visited map[string]bool) {
	visited[nodeID] = true
	
	node := gb.nodes[nodeID]
	for _, edge := range node.Connections {
		var neighborID string
		if edge.From.ID == nodeID {
			neighborID = edge.To.ID
		} else {
			neighborID = edge.From.ID
		}
		
		if !visited[neighborID] {
			gb.dfsMarkComponent(neighborID, visited)
		}
	}
}

func (gb *GraphBuilder) findShortestPath(from, to *TopologyNode) ([]*TopologyNode, []*TopologyEdge) {
	// Dijkstra's algorithm implementation
	distances := make(map[string]float64)
	previous := make(map[string]*TopologyNode)
	edgeMap := make(map[string]*TopologyEdge)
	visited := make(map[string]bool)
	
	// Initialize distances
	for nodeID := range gb.nodes {
		distances[nodeID] = float64(^uint(0) >> 1) // Max float64
	}
	distances[from.ID] = 0
	
	for len(visited) < len(gb.nodes) {
		// Find unvisited node with minimum distance
		var current *TopologyNode
		minDist := float64(^uint(0) >> 1)
		
		for nodeID, node := range gb.nodes {
			if !visited[nodeID] && distances[nodeID] < minDist {
				current = node
				minDist = distances[nodeID]
			}
		}
		
		if current == nil || current.ID == to.ID {
			break
		}
		
		visited[current.ID] = true
		
		// Update distances to neighbors
		for _, edge := range current.Connections {
			var neighbor *TopologyNode
			if edge.From.ID == current.ID {
				neighbor = edge.To
			} else {
				neighbor = edge.From
			}
			
			if visited[neighbor.ID] {
				continue
			}
			
			// Use connection quality as weight (lower quality = higher cost)
			// TODO: Use edge.Connection.Metrics.Confidence once available
			weight := 1.0 / (0.5 + 0.1)
			newDist := distances[current.ID] + weight
			
			if newDist < distances[neighbor.ID] {
				distances[neighbor.ID] = newDist
				previous[neighbor.ID] = current
				edgeMap[neighbor.ID] = edge
			}
		}
	}
	
	// Reconstruct path
	var path []*TopologyNode
	var edges []*TopologyEdge
	
	current := to
	for current != nil {
		path = append([]*TopologyNode{current}, path...)
		if prev := previous[current.ID]; prev != nil {
			edges = append([]*TopologyEdge{edgeMap[current.ID]}, edges...)
		}
		current = previous[current.ID]
	}
	
	return path, edges
}

// Helper methods for math operations (since math package might not be available)
func (gb *GraphBuilder) math_cos(x float64) float64 {
	// Simple cosine approximation
	return 1.0 - x*x/2.0 + x*x*x*x/24.0
}

func (gb *GraphBuilder) math_sin(x float64) float64 {
	// Simple sine approximation
	return x - x*x*x/6.0 + x*x*x*x*x/120.0
}

func (gb *GraphBuilder) math_sqrt(x float64) float64 {
	// Newton's method for square root
	if x < 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2*z)
	}
	return z
}

// Utility methods
func (gb *GraphBuilder) getDeviceLayer(device *types.NetworkDevice) int {
	switch device.Role {
	case "gateway": // TODO: Add DeviceRoleGateway constant
		return 0
	case "router": // TODO: Add DeviceRoleRouter
		return 1
	case "switch", "access_point": // TODO: Add DeviceRoleSwitch/AccessPoint constants
		return 2
	case "bridge": // TODO: Add DeviceRoleBridge
		return 2
	default:
		return 3
	}
}

func (gb *GraphBuilder) getDeviceGroup(device *types.NetworkDevice) string {
	if device.Location != "" {
		return device.Location
	}
	
	// Group by subnet if location not available
	for _, iface := range device.Interfaces {
		for _, ipInfo := range iface.IPAddresses {
			if ipInfo.Network != "" {
				return ipInfo.Network
			}
		}
	}
	
	return "default"
}

func (gb *GraphBuilder) getDeviceLabel(device *types.NetworkDevice) string {
	// First try to get friendly name from identity manager
	if gb.identityManager != nil {
		identity, err := gb.identityManager.GetDeviceIdentity(device.DeviceID)
		if err == nil && identity.FriendlyName != "" {
			return identity.FriendlyName
		}
	}
	
	// Fall back to hostname if available
	if device.Hostname != "" {
		return device.Hostname
	}
	
	// Last resort: use device ID (MAC address)
	return device.DeviceID
}

func (gb *GraphBuilder) getDeviceTooltip(device *types.NetworkDevice) string {
	return fmt.Sprintf("%s\nType: %s\nRole: %s\nStatus: %s", 
		device.DeviceID, device.DeviceType, device.Role, 
		map[bool]string{true: "Online", false: "Offline"}[device.Online])
}

func (gb *GraphBuilder) getConnectionLabel(conn types.DeviceConnection) string {
	if conn.Metrics.LinkSpeed > 0 {
		return fmt.Sprintf("%d Mbps", conn.Metrics.LinkSpeed)
	}
	return conn.ConnectionType
}

func (gb *GraphBuilder) getConnectionTooltip(conn types.DeviceConnection) string {
	return fmt.Sprintf("Type: %s\nConfidence: %.2f\nBandwidth: %d Mbps", 
		conn.ConnectionType, 0.5, conn.Metrics.Bandwidth) // TODO: Use conn.Metrics.Confidence
}

func (gb *GraphBuilder) calculateDeviceImportance(device *types.NetworkDevice) float64 {
	importance := 0.5 // Base importance
	
	// Increase importance for infrastructure devices
	switch device.Role {
	case "gateway": // TODO: Add DeviceRoleGateway constant
		importance += 0.4
	case "router": // TODO: Add DeviceRoleRouter
		importance += 0.3
	case "switch", "access_point": // TODO: Add DeviceRoleSwitch/AccessPoint constants
		importance += 0.2
	}
	
	// Increase importance based on number of interfaces
	importance += float64(len(device.Interfaces)) * 0.05
	
	// Cap at 1.0
	if importance > 1.0 {
		importance = 1.0
	}
	
	return importance
}
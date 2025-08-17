package topology

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// renderASCII renders topology as ASCII art
func (tv *TopologyVisualizer) renderASCII(graph *TopologyGraph, writer io.Writer) error {
	fmt.Fprintf(writer, "Network Topology (ASCII)\n")
	fmt.Fprintf(writer, "========================\n\n")

	// Header information
	fmt.Fprintf(writer, "Generated: %s\n", graph.Metadata.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(writer, "Devices: %d online, %d offline, %d total\n", 
		graph.Metadata.OnlineDevices, graph.Metadata.OfflineDevices, graph.Metadata.TotalDevices)
	fmt.Fprintf(writer, "Connections: %d\n\n", graph.Metadata.TotalConnections)

	// Group devices by type for better visualization
	devicesByType := make(map[string][]TopologyNode)
	for _, node := range graph.Nodes {
		deviceType := "unknown"
		if node.Device != nil {
			deviceType = node.Device.DeviceType
		}
		devicesByType[deviceType] = append(devicesByType[deviceType], node)
	}

	// Render each device type
	for deviceType, nodes := range devicesByType {
		fmt.Fprintf(writer, "[%s Devices]\n", strings.ToUpper(deviceType))
		
		for _, node := range nodes {
			status := "●"
			if node.Properties.Status != "online" {
				status = "○"
			}
			
			// Use importance as a proxy for quality
			qualityBar := tv.renderQualityBar(node.Properties.Importance)
			
			fmt.Fprintf(writer, "  %s %s", status, node.Properties.Label)
			
			if tv.config.ShowConnectionQuality {
				fmt.Fprintf(writer, " %s", qualityBar)
			}
			
			// TODO: Add SSID field to node properties or device
			// if node.SSID != "" && tv.config.ShowSSIDs {
			//	fmt.Fprintf(writer, " [%s]", node.SSID)
			// }
			
			// TODO: Add SignalStrength field to node properties or device
			// if node.SignalStrength != 0 {
			//	fmt.Fprintf(writer, " (%ddBm)", node.SignalStrength)
			// }
			
			fmt.Fprintf(writer, "\n")
			
			// Show connections for this node
			connections := tv.getNodeConnections(node.ID, graph)
			for _, conn := range connections {
				targetNode := conn.To
				if targetNode == nil || targetNode.ID == node.ID {
					targetNode = conn.From
				}
				
				if targetNode != nil && targetNode.ID != node.ID {
					fmt.Fprintf(writer, "    ├─ %s", targetNode.Properties.Label)
					
					if tv.config.ShowConnectionQuality {
						// Convert quality string to float
						qualityVal := 0.5 // default
						switch conn.Properties.Quality {
						case "excellent":
							qualityVal = 1.0
						case "good":
							qualityVal = 0.75
						case "fair":
							qualityVal = 0.5
						case "poor":
							qualityVal = 0.25
						}
						connQualityBar := tv.renderQualityBar(qualityVal)
						fmt.Fprintf(writer, " %s", connQualityBar)
					}
					
					// TODO: Add latency field to edge properties
					// if conn.Latency > 0 {
					//	fmt.Fprintf(writer, " (%.1fms)", conn.Latency)
					// }
					
					fmt.Fprintf(writer, "\n")
				}
			}
		}
		fmt.Fprintf(writer, "\n")
	}

	// Show anomalies if any
	if len(graph.Anomalies) > 0 {
		fmt.Fprintf(writer, "[Anomalies Detected]\n")
		for _, anomaly := range graph.Anomalies {
			fmt.Fprintf(writer, "  ⚠ %s: %s\n", anomaly.Type, anomaly.Description)
		}
		fmt.Fprintf(writer, "\n")
	}

	return nil
}

// renderTree renders topology as a tree structure
func (tv *TopologyVisualizer) renderTree(graph *TopologyGraph, writer io.Writer) error {
	fmt.Fprintf(writer, "Network Topology (Tree)\n")
	fmt.Fprintf(writer, "=======================\n\n")

	// Find root nodes (nodes with no incoming connections)
	rootNodes := tv.findRootNodes(graph)
	
	for _, root := range rootNodes {
		tv.renderNodeTree(root, graph, writer, 0, make(map[string]bool))
	}

	return nil
}

// renderDOT renders topology in DOT format for Graphviz
func (tv *TopologyVisualizer) renderDOT(graph *TopologyGraph, writer io.Writer) error {
	fmt.Fprintf(writer, "digraph NetworkTopology {\n")
	fmt.Fprintf(writer, "  rankdir=TB;\n")
	fmt.Fprintf(writer, "  node [shape=box, style=rounded];\n")
	fmt.Fprintf(writer, "  edge [fontsize=10];\n\n")

	// Add nodes
	for _, node := range graph.Nodes {
		color := tv.getNodeColor(node)
		deviceType := "unknown"
		if node.Device != nil {
			deviceType = node.Device.DeviceType
		}
		shape := tv.getNodeShape(deviceType)
		
		label := fmt.Sprintf("%s\\n%s", node.Properties.Label, deviceType)
		// TODO: Add SSID field to node
		// if node.SSID != "" && tv.config.ShowSSIDs {
		//	label = fmt.Sprintf("%s\\n[%s]", label, node.SSID)
		// }
		
		fmt.Fprintf(writer, "  \"%s\" [label=\"%s\", color=\"%s\", shape=%s];\n",
			node.ID, label, color, shape)
	}

	fmt.Fprintf(writer, "\n")

	// Add edges
	for _, edge := range graph.Edges {
		label := ""
		if tv.config.ShowConnectionQuality {
			label = edge.Properties.Quality
		}
		// TODO: Add latency field to edge properties

		// Convert quality string to float for style calculation
		qualityVal := 0.5
		switch edge.Properties.Quality {
		case "excellent":
			qualityVal = 1.0
		case "good":
			qualityVal = 0.75
		case "fair":
			qualityVal = 0.5
		case "poor":
			qualityVal = 0.25
		}
		color := tv.getEdgeColor(qualityVal)
		style := tv.getEdgeStyle(edge.Connection.ConnectionType)

		fmt.Fprintf(writer, "  \"%s\" -> \"%s\" [label=\"%s\", color=\"%s\", style=%s];\n",
			edge.From.ID, edge.To.ID, label, color, style)
	}

	// Add groups as subgraphs
	for i, group := range graph.Groups {
		fmt.Fprintf(writer, "\n  subgraph cluster_%d {\n", i)
		fmt.Fprintf(writer, "    label=\"%s\";\n", group.Label)
		fmt.Fprintf(writer, "    color=\"%s\";\n", group.Color)
		
		for _, nodeID := range group.NodeIDs {
			fmt.Fprintf(writer, "    \"%s\";\n", nodeID)
		}
		
		fmt.Fprintf(writer, "  }\n")
	}

	fmt.Fprintf(writer, "}\n")
	return nil
}

// renderJSON renders topology as JSON
func (tv *TopologyVisualizer) renderJSON(graph *TopologyGraph, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(graph)
}

// renderTable renders topology as a table
func (tv *TopologyVisualizer) renderTable(graph *TopologyGraph, writer io.Writer) error {
	fmt.Fprintf(writer, "Network Topology (Table)\n")
	fmt.Fprintf(writer, "========================\n\n")

	// Devices table
	fmt.Fprintf(writer, "DEVICES:\n")
	fmt.Fprintf(writer, "%-20s %-15s %-10s %-15s %-10s %-20s\n",
		"Device", "Type", "Status", "IP Address", "Quality", "SSID")
	fmt.Fprintf(writer, "%s\n", strings.Repeat("-", 90))

	// Sort nodes by label for consistent output
	sortedNodes := make([]TopologyNode, len(graph.Nodes))
	copy(sortedNodes, graph.Nodes)
	sort.Slice(sortedNodes, func(i, j int) bool {
		return sortedNodes[i].Properties.Label < sortedNodes[j].Properties.Label
	})

	for _, node := range sortedNodes {
		qualityStr := fmt.Sprintf("%.2f", node.Properties.Importance)
		
		deviceType := "unknown"
		ipAddress := "-"
		if node.Device != nil {
			deviceType = node.Device.DeviceType
			// Get first IP if available
			for _, iface := range node.Device.Interfaces {
				if len(iface.IPAddresses) > 0 {
					ipAddress = iface.IPAddresses[0].Address
					break
				}
			}
		}
		
		fmt.Fprintf(writer, "%-20s %-15s %-10s %-15s %-10s %-20s\n",
			tv.truncateString(node.Properties.Label, 20),
			deviceType,
			node.Properties.Status,
			ipAddress,
			qualityStr,
			"-") // SSID not available
	}

	fmt.Fprintf(writer, "\n")

	// Connections table
	fmt.Fprintf(writer, "CONNECTIONS:\n")
	fmt.Fprintf(writer, "%-20s %-20s %-10s %-10s %-10s %-15s\n",
		"Source", "Target", "Type", "Quality", "Latency", "Interface")
	fmt.Fprintf(writer, "%s\n", strings.Repeat("-", 95))

	// Sort edges by quality (highest first)
	sortedEdges := make([]TopologyEdge, len(graph.Edges))
	copy(sortedEdges, graph.Edges)
	sort.Slice(sortedEdges, func(i, j int) bool {
		// Convert quality strings to values for comparison
		qI, qJ := 0.5, 0.5
		switch sortedEdges[i].Properties.Quality {
		case "excellent": qI = 1.0
		case "good": qI = 0.75
		case "fair": qI = 0.5
		case "poor": qI = 0.25
		}
		switch sortedEdges[j].Properties.Quality {
		case "excellent": qJ = 1.0
		case "good": qJ = 0.75
		case "fair": qJ = 0.5
		case "poor": qJ = 0.25
		}
		return qI > qJ
	})

	for _, edge := range sortedEdges {
		sourceNode := edge.From
		targetNode := edge.To
		
		sourceLabel := edge.From.ID
		targetLabel := edge.To.ID
		
		if sourceNode != nil {
			sourceLabel = sourceNode.Properties.Label
		}
		if targetNode != nil {
			targetLabel = targetNode.Properties.Label
		}

		qualityStr := edge.Properties.Quality
		latencyStr := "-"
		// TODO: Add latency to edge properties
		
		interfaceStr := edge.Connection.FromInterface
		if interfaceStr == "" {
			interfaceStr = "-"
		}

		fmt.Fprintf(writer, "%-20s %-20s %-10s %-10s %-10s %-15s\n",
			tv.truncateString(sourceLabel, 20),
			tv.truncateString(targetLabel, 20),
			edge.Connection.ConnectionType,
			qualityStr,
			latencyStr,
			tv.truncateString(interfaceStr, 15))
	}

	return nil
}

// renderSummary renders topology summary
func (tv *TopologyVisualizer) renderSummary(graph *TopologyGraph, writer io.Writer) error {
	fmt.Fprintf(writer, "Network Topology Summary\n")
	fmt.Fprintf(writer, "========================\n\n")

	// Basic statistics
	fmt.Fprintf(writer, "Network Overview:\n")
	fmt.Fprintf(writer, "  Total Devices: %d\n", graph.Metadata.TotalDevices)
	fmt.Fprintf(writer, "  Online Devices: %d\n", graph.Metadata.OnlineDevices)
	fmt.Fprintf(writer, "  Offline Devices: %d\n", graph.Metadata.OfflineDevices)
	fmt.Fprintf(writer, "  Total Connections: %d\n", graph.Metadata.TotalConnections)
	fmt.Fprintf(writer, "  Average Quality: %.2f\n", graph.Stats.AverageQuality)
	fmt.Fprintf(writer, "  Average Latency: %.1fms\n", graph.Stats.AverageLatency)
	fmt.Fprintf(writer, "\n")

	// Device breakdown by type
	fmt.Fprintf(writer, "Devices by Type:\n")
	for deviceType, count := range graph.Stats.DevicesByType {
		fmt.Fprintf(writer, "  %s: %d\n", deviceType, count)
	}
	fmt.Fprintf(writer, "\n")

	// Device breakdown by status
	fmt.Fprintf(writer, "Devices by Status:\n")
	for status, count := range graph.Stats.DevicesByStatus {
		fmt.Fprintf(writer, "  %s: %d\n", status, count)
	}
	fmt.Fprintf(writer, "\n")

	// SSID breakdown
	if len(graph.Stats.DevicesBySSID) > 0 {
		fmt.Fprintf(writer, "Devices by SSID:\n")
		for ssid, count := range graph.Stats.DevicesBySSID {
			fmt.Fprintf(writer, "  %s: %d\n", ssid, count)
		}
		fmt.Fprintf(writer, "\n")
	}

	// Anomalies
	if len(graph.Anomalies) > 0 {
		fmt.Fprintf(writer, "Detected Anomalies:\n")
		for _, anomaly := range graph.Anomalies {
			fmt.Fprintf(writer, "  [%s] %s: %s\n", 
				strings.ToUpper(anomaly.Severity), anomaly.Type, anomaly.Description)
		}
		fmt.Fprintf(writer, "\n")
	}

	// Performance metrics
	if graph.Stats.TotalBandwidth > 0 {
		fmt.Fprintf(writer, "Performance Metrics:\n")
		fmt.Fprintf(writer, "  Total Bandwidth: %.1f Mbps\n", graph.Stats.TotalBandwidth)
		fmt.Fprintf(writer, "  Packet Loss Rate: %.2f%%\n", graph.Stats.PacketLossRate*100)
		fmt.Fprintf(writer, "  Topology Complexity: %.2f\n", graph.Stats.TopologyComplexity)
	}

	return nil
}

// renderGraphViz renders topology in enhanced GraphViz format
func (tv *TopologyVisualizer) renderGraphViz(graph *TopologyGraph, writer io.Writer) error {
	fmt.Fprintf(writer, "graph NetworkTopology {\n")
	fmt.Fprintf(writer, "  layout=fdp;\n")
	fmt.Fprintf(writer, "  splines=true;\n")
	fmt.Fprintf(writer, "  overlap=false;\n")
	fmt.Fprintf(writer, "  sep=\"+25,25\";\n")
	fmt.Fprintf(writer, "  esep=\"+10,10\";\n\n")

	// Add nodes with rich styling
	for _, node := range graph.Nodes {
		color := tv.getNodeColor(node)
		
		deviceType := "unknown"
		if node.Device != nil {
			deviceType = node.Device.DeviceType
		}
		shape := tv.getNodeShape(deviceType)
		
		label := node.Properties.Label
		// TODO: Add SSID field to node
		
		if tv.config.ShowConnectionQuality {
			label = fmt.Sprintf("%s\\nQ: %.2f", label, node.Properties.Importance)
		}

		fontSize := "12"
		if deviceType == "router" || deviceType == "gateway" {
			fontSize = "14"
		}

		fmt.Fprintf(writer, "  \"%s\" [label=\"%s\", color=\"%s\", shape=%s, fontsize=%s",
			node.ID, label, color, shape, fontSize)
		
		if node.Properties.Status != "online" {
			fmt.Fprintf(writer, ", style=dashed")
		}
		
		fmt.Fprintf(writer, "];\n")
	}

	fmt.Fprintf(writer, "\n")

	// Add edges with weights and styling
	for _, edge := range graph.Edges {
		// Convert quality string to float
		qualityVal := 0.5
		switch edge.Properties.Quality {
		case "excellent": qualityVal = 1.0
		case "good": qualityVal = 0.75
		case "fair": qualityVal = 0.5
		case "poor": qualityVal = 0.25
		}
		
		weight := fmt.Sprintf("%.2f", qualityVal)
		color := tv.getEdgeColor(qualityVal)
		thickness := tv.getEdgeThickness(qualityVal)

		fmt.Fprintf(writer, "  \"%s\" -- \"%s\" [weight=%s, color=\"%s\", penwidth=%s",
			edge.From.ID, edge.To.ID, weight, color, thickness)
		
		if edge.Connection.ConnectionType == "wireless" {
			fmt.Fprintf(writer, ", style=dashed")
		}
		
		fmt.Fprintf(writer, "];\n")
	}

	fmt.Fprintf(writer, "}\n")
	return nil
}

// renderPlantUML renders topology in PlantUML format
func (tv *TopologyVisualizer) renderPlantUML(graph *TopologyGraph, writer io.Writer) error {
	fmt.Fprintf(writer, "@startuml\n")
	fmt.Fprintf(writer, "!define NETWORK_NODE(alias, label) node \"label\" as alias\n")
	fmt.Fprintf(writer, "!define WIFI_CONNECTION(node1, node2) node1 ..> node2\n")
	fmt.Fprintf(writer, "!define WIRED_CONNECTION(node1, node2) node1 --> node2\n\n")

	fmt.Fprintf(writer, "title Network Topology\\n%s\n\n", 
		graph.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"))

	// Define nodes
	for _, node := range graph.Nodes {
		alias := strings.ReplaceAll(node.ID, "-", "_")
		label := node.Properties.Label
		
		// TODO: Add SSID field to node

		deviceType := "unknown"
		if node.Device != nil {
			deviceType = node.Device.DeviceType
		}
		
		nodeType := "node"
		switch deviceType {
		case "router", "gateway":
			nodeType = "cloud"
		case "access_point", "ap":
			nodeType = "database"
		case "client":
			nodeType = "actor"
		}

		fmt.Fprintf(writer, "%s \"%s\" as %s\n", nodeType, label, alias)
	}

	fmt.Fprintf(writer, "\n")

	// Define connections
	for _, edge := range graph.Edges {
		sourceAlias := strings.ReplaceAll(edge.From.ID, "-", "_")
		targetAlias := strings.ReplaceAll(edge.To.ID, "-", "_")
		
		connector := "-->"
		if edge.Connection.ConnectionType == "wireless" {
			connector = "..>"
		}

		label := ""
		if tv.config.ShowConnectionQuality {
			label = fmt.Sprintf(" : %s", edge.Properties.Quality)
			// TODO: Add latency to edge properties
		}

		fmt.Fprintf(writer, "%s %s %s%s\n", sourceAlias, connector, targetAlias, label)
	}

	// Add notes for anomalies
	if len(graph.Anomalies) > 0 {
		fmt.Fprintf(writer, "\n")
		for _, anomaly := range graph.Anomalies {
			if anomaly.NodeID != "" {
				nodeAlias := strings.ReplaceAll(anomaly.NodeID, "-", "_")
				fmt.Fprintf(writer, "note right of %s : %s\\n%s\n", 
					nodeAlias, anomaly.Type, anomaly.Description)
			}
		}
	}

	fmt.Fprintf(writer, "\n@enduml\n")
	return nil
}

// Helper methods for rendering

func (tv *TopologyVisualizer) renderQualityBar(quality float64) string {
	if !tv.config.ShowConnectionQuality {
		return ""
	}

	length := 10
	filled := int(quality * float64(length))
	
	bar := "["
	for i := 0; i < length; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"
	
	return fmt.Sprintf("%s %.2f", bar, quality)
}

func (tv *TopologyVisualizer) getNodeConnections(nodeID string, graph *TopologyGraph) []TopologyEdge {
	var connections []TopologyEdge
	for _, edge := range graph.Edges {
		if edge.From.ID == nodeID || edge.To.ID == nodeID {
			connections = append(connections, edge)
		}
	}
	return connections
}

func (tv *TopologyVisualizer) findNode(nodeID string, graph *TopologyGraph) *TopologyNode {
	for i, node := range graph.Nodes {
		if node.ID == nodeID {
			return &graph.Nodes[i]
		}
	}
	return nil
}

func (tv *TopologyVisualizer) findRootNodes(graph *TopologyGraph) []TopologyNode {
	hasIncoming := make(map[string]bool)
	
	// Mark nodes with incoming connections
	for _, edge := range graph.Edges {
		hasIncoming[edge.To.ID] = true
	}

	// Find nodes without incoming connections
	var roots []TopologyNode
	for _, node := range graph.Nodes {
		if !hasIncoming[node.ID] {
			roots = append(roots, node)
		}
	}

	// If no roots found (circular topology), use highest quality nodes
	if len(roots) == 0 {
		sortedNodes := make([]TopologyNode, len(graph.Nodes))
		copy(sortedNodes, graph.Nodes)
		sort.Slice(sortedNodes, func(i, j int) bool {
			return sortedNodes[i].Properties.Importance > sortedNodes[j].Properties.Importance
		})
		
		if len(sortedNodes) > 0 {
			roots = append(roots, sortedNodes[0])
		}
	}

	return roots
}

func (tv *TopologyVisualizer) renderNodeTree(node TopologyNode, graph *TopologyGraph, writer io.Writer, depth int, visited map[string]bool) {
	if visited[node.ID] {
		return
	}
	visited[node.ID] = true

	indent := strings.Repeat("  ", depth)
	status := "●"
	if node.Properties.Status != "online" {
		status = "○"
	}

	fmt.Fprintf(writer, "%s%s %s", indent, status, node.Properties.Label)
	
	if tv.config.ShowConnectionQuality {
		fmt.Fprintf(writer, " (Q: %.2f)", node.Properties.Importance)
	}
	
	// TODO: Add SSID field to node
	// if node.SSID != "" && tv.config.ShowSSIDs {
	//	fmt.Fprintf(writer, " [%s]", node.SSID)
	// }
	
	fmt.Fprintf(writer, "\n")

	// Find and render children
	for _, edge := range graph.Edges {
		var childID string
		if edge.From.ID == node.ID {
			childID = edge.To.ID
		} else if edge.To.ID == node.ID {
			childID = edge.From.ID
		} else {
			continue
		}

		if !visited[childID] {
			childNode := tv.findNode(childID, graph)
			if childNode != nil {
				tv.renderNodeTree(*childNode, graph, writer, depth+1, visited)
			}
		}
	}
}

func (tv *TopologyVisualizer) getNodeColor(node TopologyNode) string {
	switch node.Properties.Status {
	case "online":
		if node.Properties.Importance > 0.8 {
			return "green"
		} else if node.Properties.Importance > 0.5 {
			return "orange"
		} else {
			return "red"
		}
	case "offline":
		return "gray"
	default:
		return "yellow"
	}
}

func (tv *TopologyVisualizer) getNodeShape(nodeType string) string {
	switch nodeType {
	case "router", "gateway":
		return "diamond"
	case "access_point":
		return "ellipse"
	case "switch", "hub":
		return "box"
	default:
		return "circle"
	}
}

func (tv *TopologyVisualizer) getEdgeColor(quality float64) string {
	if quality > 0.8 {
		return "green"
	} else if quality > 0.5 {
		return "orange"
	} else {
		return "red"
	}
}

func (tv *TopologyVisualizer) getEdgeStyle(edgeType string) string {
	switch edgeType {
	case "wireless":
		return "dashed"
	case "wired":
		return "solid"
	default:
		return "dotted"
	}
}

func (tv *TopologyVisualizer) getEdgeThickness(quality float64) string {
	if quality > 0.8 {
		return "3"
	} else if quality > 0.5 {
		return "2"
	} else {
		return "1"
	}
}

func (tv *TopologyVisualizer) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
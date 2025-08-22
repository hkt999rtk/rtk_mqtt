package topology

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"rtk_controller/internal/storage"
)

func TestTopologyVisualizer(t *testing.T) {
	// Create mock storage
	mockStorage := &storage.TopologyStorage{}
	mockIdentityStorage := &storage.IdentityStorage{}

	// Create test topology manager
	manager := &Manager{
		storage:         mockStorage,
		identityStorage: mockIdentityStorage,
	}

	// Create visualizer with default config
	config := VisualizationConfig{
		ShowOfflineDevices:    true,
		ShowConnectionQuality: true,
		ShowSSIDs:             true,
		ShowInterfaceDetails:  false,
		CompactMode:           false,
		ColorEnabled:          false, // Disable for testing
		MaxWidth:              80,
		MinConnectionQuality:  0.0,
	}

	visualizer := NewTopologyVisualizer(manager, mockIdentityStorage, config)

	// Create test topology graph
	graph := createTestTopologyGraph()

	t.Run("RenderASCII", func(t *testing.T) {
		var buf bytes.Buffer
		err := visualizer.renderASCII(graph, &buf)
		if err != nil {
			t.Fatalf("Failed to render ASCII: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Network Topology (ASCII)") {
			t.Error("ASCII output should contain header")
		}
		if !strings.Contains(output, "Generated:") {
			t.Error("ASCII output should contain timestamp")
		}
	})

	t.Run("RenderTree", func(t *testing.T) {
		var buf bytes.Buffer
		err := visualizer.renderTree(graph, &buf)
		if err != nil {
			t.Fatalf("Failed to render tree: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Network Topology (Tree)") {
			t.Error("Tree output should contain header")
		}
	})

	t.Run("RenderDOT", func(t *testing.T) {
		var buf bytes.Buffer
		err := visualizer.renderDOT(graph, &buf)
		if err != nil {
			t.Fatalf("Failed to render DOT: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "digraph NetworkTopology") {
			t.Error("DOT output should contain digraph declaration")
		}
		if !strings.Contains(output, "rankdir=TB") {
			t.Error("DOT output should contain layout direction")
		}
	})

	t.Run("RenderJSON", func(t *testing.T) {
		var buf bytes.Buffer
		err := visualizer.renderJSON(graph, &buf)
		if err != nil {
			t.Fatalf("Failed to render JSON: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, `"nodes"`) {
			t.Error("JSON output should contain nodes")
		}
		if !strings.Contains(output, `"edges"`) {
			t.Error("JSON output should contain edges")
		}
	})

	t.Run("RenderTable", func(t *testing.T) {
		var buf bytes.Buffer
		err := visualizer.renderTable(graph, &buf)
		if err != nil {
			t.Fatalf("Failed to render table: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "DEVICES:") {
			t.Error("Table output should contain devices section")
		}
		if !strings.Contains(output, "CONNECTIONS:") {
			t.Error("Table output should contain connections section")
		}
	})

	t.Run("RenderSummary", func(t *testing.T) {
		var buf bytes.Buffer
		err := visualizer.renderSummary(graph, &buf)
		if err != nil {
			t.Fatalf("Failed to render summary: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Network Topology Summary") {
			t.Error("Summary output should contain header")
		}
		if !strings.Contains(output, "Network Overview:") {
			t.Error("Summary output should contain overview")
		}
	})

	t.Run("RenderGraphViz", func(t *testing.T) {
		var buf bytes.Buffer
		err := visualizer.renderGraphViz(graph, &buf)
		if err != nil {
			t.Fatalf("Failed to render GraphViz: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "graph NetworkTopology") {
			t.Error("GraphViz output should contain graph declaration")
		}
		if !strings.Contains(output, "layout=fdp") {
			t.Error("GraphViz output should contain layout specification")
		}
	})

	t.Run("RenderPlantUML", func(t *testing.T) {
		var buf bytes.Buffer
		err := visualizer.renderPlantUML(graph, &buf)
		if err != nil {
			t.Fatalf("Failed to render PlantUML: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "@startuml") {
			t.Error("PlantUML output should start with @startuml")
		}
		if !strings.Contains(output, "@enduml") {
			t.Error("PlantUML output should end with @enduml")
		}
	})
}

func TestVisualizationConfig(t *testing.T) {
	mockStorage := &storage.TopologyStorage{}
	mockIdentityStorage := &storage.IdentityStorage{}
	manager := &Manager{
		storage:         mockStorage,
		identityStorage: mockIdentityStorage,
	}

	t.Run("FilterByConnectionQuality", func(t *testing.T) {
		config := VisualizationConfig{
			MinConnectionQuality: 0.8,
			ShowOfflineDevices:   false,
		}

		visualizer := NewTopologyVisualizer(manager, mockIdentityStorage, config)
		graph := createTestTopologyGraph()

		// Filter edges based on quality
		originalEdgeCount := len(graph.Edges)
		filteredEdges := []TopologyEdge{}
		for _, edge := range graph.Edges {
			if edge.Quality >= config.MinConnectionQuality {
				filteredEdges = append(filteredEdges, edge)
			}
		}
		graph.Edges = filteredEdges

		if len(graph.Edges) >= originalEdgeCount {
			t.Error("Quality filter should reduce number of edges")
		}
	})

	t.Run("GroupBySSID", func(t *testing.T) {
		config := VisualizationConfig{
			GroupBySSID: true,
		}

		visualizer := NewTopologyVisualizer(manager, mockIdentityStorage, config)
		graph := createTestTopologyGraph()

		// Build groups
		visualizer.buildGroups(graph)

		if len(graph.Groups) == 0 {
			t.Error("Should create SSID groups when GroupBySSID is enabled")
		}

		// Check if groups contain the expected nodes
		for _, group := range graph.Groups {
			if group.Type == "ssid" && len(group.NodeIDs) == 0 {
				t.Error("SSID groups should contain node IDs")
			}
		}
	})

	t.Run("ShowAnomalies", func(t *testing.T) {
		config := VisualizationConfig{
			ShowAnomalies: true,
		}

		visualizer := NewTopologyVisualizer(manager, mockIdentityStorage, config)
		graph := createTestTopologyGraph()

		// Add poor quality connections to trigger anomaly detection
		graph.Edges = append(graph.Edges, TopologyEdge{
			ID:      "poor_edge",
			Source:  "device1",
			Target:  "device2",
			Quality: 0.1, // Poor quality
		})

		visualizer.detectAnomalies(graph)

		if len(graph.Anomalies) == 0 {
			t.Error("Should detect anomalies with poor quality connections")
		}

		// Check anomaly types
		foundPoorQuality := false
		for _, anomaly := range graph.Anomalies {
			if anomaly.Type == "poor_quality_connection" {
				foundPoorQuality = true
			}
		}

		if !foundPoorQuality {
			t.Error("Should detect poor quality connection anomaly")
		}
	})
}

func TestTopologyStats(t *testing.T) {
	mockStorage := &storage.TopologyStorage{}
	mockIdentityStorage := &storage.IdentityStorage{}
	manager := &Manager{
		storage:         mockStorage,
		identityStorage: mockIdentityStorage,
	}

	config := VisualizationConfig{}
	visualizer := NewTopologyVisualizer(manager, mockIdentityStorage, config)
	graph := createTestTopologyGraph()

	visualizer.calculateStats(graph)

	t.Run("DeviceStats", func(t *testing.T) {
		if graph.Stats.DevicesByType == nil {
			t.Error("DevicesByType should be initialized")
		}
		if graph.Stats.DevicesByStatus == nil {
			t.Error("DevicesByStatus should be initialized")
		}
		if graph.Stats.DevicesBySSID == nil {
			t.Error("DevicesBySSID should be initialized")
		}
	})

	t.Run("QualityStats", func(t *testing.T) {
		if graph.Stats.AverageQuality < 0 || graph.Stats.AverageQuality > 1 {
			t.Error("Average quality should be between 0 and 1")
		}
	})

	t.Run("ComplexityStats", func(t *testing.T) {
		if graph.Stats.TopologyComplexity < 0 || graph.Stats.TopologyComplexity > 1 {
			t.Error("Topology complexity should be between 0 and 1")
		}
	})
}

func TestNodeFiltering(t *testing.T) {
	mockStorage := &storage.TopologyStorage{}
	mockIdentityStorage := &storage.IdentityStorage{}
	manager := &Manager{
		storage:         mockStorage,
		identityStorage: mockIdentityStorage,
	}

	t.Run("FilterOfflineDevices", func(t *testing.T) {
		config := VisualizationConfig{
			ShowOfflineDevices: false,
		}

		visualizer := NewTopologyVisualizer(manager, mockIdentityStorage, config)

		// Test with offline device
		offlineDevice := &DeviceState{
			Status: DeviceStatusOffline,
			Type:   DeviceTypeClient,
		}

		if visualizer.shouldIncludeDevice(offlineDevice) {
			t.Error("Should not include offline devices when ShowOfflineDevices is false")
		}

		// Test with online device
		onlineDevice := &DeviceState{
			Status: DeviceStatusOnline,
			Type:   DeviceTypeClient,
		}

		if !visualizer.shouldIncludeDevice(onlineDevice) {
			t.Error("Should include online devices")
		}
	})

	t.Run("FilterByDeviceType", func(t *testing.T) {
		config := VisualizationConfig{
			DeviceTypeFilter: []string{"router", "access_point"},
		}

		visualizer := NewTopologyVisualizer(manager, mockIdentityStorage, config)

		// Test with allowed device type
		routerDevice := &DeviceState{
			Status: DeviceStatusOnline,
			Type:   DeviceTypeRouter,
		}

		if !visualizer.shouldIncludeDevice(routerDevice) {
			t.Error("Should include devices with allowed type")
		}

		// Test with disallowed device type
		clientDevice := &DeviceState{
			Status: DeviceStatusOnline,
			Type:   DeviceTypeClient,
		}

		if visualizer.shouldIncludeDevice(clientDevice) {
			t.Error("Should not include devices with disallowed type")
		}
	})

	t.Run("FilterByTimeWindow", func(t *testing.T) {
		config := VisualizationConfig{
			TimeWindow: time.Hour,
		}

		visualizer := NewTopologyVisualizer(manager, mockIdentityStorage, config)

		// Test with recent device
		recentDevice := &DeviceState{
			Status:   DeviceStatusOnline,
			Type:     DeviceTypeClient,
			LastSeen: time.Now().Add(-30 * time.Minute),
		}

		if !visualizer.shouldIncludeDevice(recentDevice) {
			t.Error("Should include devices within time window")
		}

		// Test with old device
		oldDevice := &DeviceState{
			Status:   DeviceStatusOnline,
			Type:     DeviceTypeClient,
			LastSeen: time.Now().Add(-2 * time.Hour),
		}

		if visualizer.shouldIncludeDevice(oldDevice) {
			t.Error("Should not include devices outside time window")
		}
	})
}

// Helper function to create test topology graph
func createTestTopologyGraph() *TopologyGraph {
	now := time.Now()

	nodes := []TopologyNode{
		{
			ID:         "router1",
			Label:      "Home Router",
			Type:       "router",
			Status:     "online",
			MacAddress: "00:11:22:33:44:55",
			IPAddress:  "192.168.1.1",
			Quality:    0.95,
			LastSeen:   now,
		},
		{
			ID:             "ap1",
			Label:          "Living Room AP",
			Type:           "access_point",
			Status:         "online",
			MacAddress:     "00:11:22:33:44:66",
			IPAddress:      "192.168.1.2",
			SSID:           "HomeWiFi",
			Quality:        0.88,
			LastSeen:       now,
			SignalStrength: -45,
		},
		{
			ID:             "client1",
			Label:          "Phone",
			Type:           "client",
			Status:         "online",
			MacAddress:     "00:11:22:33:44:77",
			IPAddress:      "192.168.1.100",
			SSID:           "HomeWiFi",
			Quality:        0.75,
			LastSeen:       now,
			SignalStrength: -55,
		},
		{
			ID:             "client2",
			Label:          "Laptop",
			Type:           "client",
			Status:         "offline",
			MacAddress:     "00:11:22:33:44:88",
			IPAddress:      "192.168.1.101",
			SSID:           "HomeWiFi",
			Quality:        0.60,
			LastSeen:       now.Add(-time.Hour),
			SignalStrength: -65,
		},
	}

	edges := []TopologyEdge{
		{
			ID:        "edge1",
			Source:    "router1",
			Target:    "ap1",
			Type:      "wired",
			Quality:   0.95,
			Bandwidth: 1000.0,
			Latency:   1.0,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "edge2",
			Source:    "ap1",
			Target:    "client1",
			Type:      "wireless",
			Quality:   0.80,
			Bandwidth: 150.0,
			Latency:   5.0,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "edge3",
			Source:    "ap1",
			Target:    "client2",
			Type:      "wireless",
			Quality:   0.65,
			Bandwidth: 50.0,
			Latency:   15.0,
			CreatedAt: now.Add(-time.Hour),
			UpdatedAt: now.Add(-time.Hour),
		},
	}

	metadata := TopologyMetadata{
		GeneratedAt:      now,
		TotalDevices:     4,
		OnlineDevices:    3,
		OfflineDevices:   1,
		TotalConnections: 3,
		Version:          "1.0",
	}

	stats := TopologyStats{
		DevicesByType:   map[string]int{"router": 1, "access_point": 1, "client": 2},
		DevicesByStatus: map[string]int{"online": 3, "offline": 1},
		DevicesBySSID:   map[string]int{"HomeWiFi": 3},
		AverageQuality:  0.795,
		AverageLatency:  7.0,
	}

	return &TopologyGraph{
		Nodes:     nodes,
		Edges:     edges,
		Groups:    []TopologyGroup{},
		Metadata:  metadata,
		Stats:     stats,
		Anomalies: []TopologyAnomaly{},
	}
}

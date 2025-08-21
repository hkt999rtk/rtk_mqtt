package llm

import (
	"context"
	"fmt"

	"rtk_controller/internal/topology"
	"rtk_controller/pkg/types"
)

// TopologyGetFullTool implements the topology.get_full LLM tool
type TopologyGetFullTool struct {
	topologyManager *topology.Manager
}

// NewTopologyGetFullTool creates a new topology.get_full tool
func NewTopologyGetFullTool(topologyManager *topology.Manager) *TopologyGetFullTool {
	return &TopologyGetFullTool{
		topologyManager: topologyManager,
	}
}

// Name returns the tool name
func (t *TopologyGetFullTool) Name() string {
	return "topology.get_full"
}

// Category returns the tool category
func (t *TopologyGetFullTool) Category() types.ToolCategory {
	return types.ToolCategoryRead
}

// Description returns the tool description
func (t *TopologyGetFullTool) Description() string {
	return "Retrieves the complete network topology including all devices and connections"
}

// RequiredCapabilities returns the required device capabilities
func (t *TopologyGetFullTool) RequiredCapabilities() []string {
	return []string{"topology_query"}
}

// Validate validates the tool parameters
func (t *TopologyGetFullTool) Validate(params map[string]interface{}) error {
	// topology.get_full doesn't require any specific parameters
	// Optional parameters could be added later (e.g., include_offline_devices, detail_level)
	return nil
}

// Execute executes the tool
func (t *TopologyGetFullTool) Execute(ctx context.Context, params map[string]interface{}) (*types.ToolResult, error) {
	result := &types.ToolResult{
		ToolName:  t.Name(),
		Success:   false,
		Timestamp: getCurrentTime(),
	}
	
	// Get topology from the manager
	topology, err := t.topologyManager.GetCurrentTopology()
	if err != nil {
		result.Error = fmt.Sprintf("Failed to retrieve topology: %v", err)
		return result, nil
	}
	
	// Create structured response
	topologyData := map[string]interface{}{
		"devices":     topology.Devices,
		"connections": topology.Connections,
		"gateway":     topology.Gateway,
		"metadata": map[string]interface{}{
			"id":               topology.ID,
			"tenant":           topology.Tenant,
			"site":             topology.Site,
			"device_count":     len(topology.Devices),
			"connection_count": len(topology.Connections),
			"updated_at":       topology.UpdatedAt,
		},
	}
	
	result.Success = true
	result.Data = topologyData
	
	return result, nil
}

// ClientsListTool implements the clients.list LLM tool
type ClientsListTool struct {
	topologyManager *topology.Manager
}

// NewClientsListTool creates a new clients.list tool
func NewClientsListTool(topologyManager *topology.Manager) *ClientsListTool {
	return &ClientsListTool{
		topologyManager: topologyManager,
	}
}

// Name returns the tool name
func (c *ClientsListTool) Name() string {
	return "clients.list"
}

// Category returns the tool category
func (c *ClientsListTool) Category() types.ToolCategory {
	return types.ToolCategoryRead
}

// Description returns the tool description
func (c *ClientsListTool) Description() string {
	return "Lists all client devices in the network with their connection status and basic information"
}

// RequiredCapabilities returns the required device capabilities
func (c *ClientsListTool) RequiredCapabilities() []string {
	return []string{"device_discovery"}
}

// Validate validates the tool parameters
func (c *ClientsListTool) Validate(params map[string]interface{}) error {
	// Optional parameters validation
	if includeOffline, exists := params["include_offline"]; exists {
		if _, ok := includeOffline.(bool); !ok {
			return fmt.Errorf("include_offline must be a boolean")
		}
	}
	
	if deviceType, exists := params["device_type"]; exists {
		if _, ok := deviceType.(string); !ok {
			return fmt.Errorf("device_type must be a string")
		}
	}
	
	return nil
}

// Execute executes the tool
func (c *ClientsListTool) Execute(ctx context.Context, params map[string]interface{}) (*types.ToolResult, error) {
	result := &types.ToolResult{
		ToolName:  c.Name(),
		Success:   false,
		Timestamp: getCurrentTime(),
	}
	
	// Get topology to access device information
	topology, err := c.topologyManager.GetCurrentTopology()
	if err != nil {
		result.Error = fmt.Sprintf("Failed to retrieve topology: %v", err)
		return result, nil
	}
	
	// Parse optional parameters
	includeOffline := true // default to true
	if val, exists := params["include_offline"]; exists {
		if b, ok := val.(bool); ok {
			includeOffline = b
		}
	}
	
	var deviceTypeFilter string
	if val, exists := params["device_type"]; exists {
		if s, ok := val.(string); ok {
			deviceTypeFilter = s
		}
	}
	
	// Build client list
	var clients []map[string]interface{}
	var onlineCount, offlineCount int
	
	for deviceID, device := range topology.Devices {
		// Apply filters
		if !includeOffline && !device.Online {
			continue
		}
		
		if deviceTypeFilter != "" && device.DeviceType != deviceTypeFilter {
			continue
		}
		
		// Count online/offline status
		if device.Online {
			onlineCount++
		} else {
			offlineCount++
		}
		
		// Get primary IP address if available
		var primaryIP string
		if len(device.Interfaces) > 0 {
			for _, iface := range device.Interfaces {
				if len(iface.IPAddresses) > 0 {
					primaryIP = iface.IPAddresses[0].Address
					break
				}
			}
		}
		
		clientInfo := map[string]interface{}{
			"device_id":    deviceID,
			"hostname":     device.Hostname,
			"device_type":  device.DeviceType,
			"primary_mac":  device.PrimaryMAC,
			"online":       device.Online,
			"ip_address":   primaryIP,
			"manufacturer": device.Manufacturer,
			"model":        device.Model,
			"location":     device.Location,
			"role":         device.Role,
			"last_seen":    device.LastSeen,
		}
		
		clients = append(clients, clientInfo)
	}
	
	// Create structured response
	clientsData := map[string]interface{}{
		"clients": clients,
		"summary": map[string]interface{}{
			"total_clients":   len(clients),
			"online_clients":  onlineCount,
			"offline_clients": offlineCount,
			"filters_applied": map[string]interface{}{
				"include_offline": includeOffline,
				"device_type":     deviceTypeFilter,
			},
		},
		"metadata": map[string]interface{}{
			"retrieved_at": getCurrentTime(),
			"source":       "topology_manager",
		},
	}
	
	result.Success = true
	result.Data = clientsData
	
	return result, nil
}
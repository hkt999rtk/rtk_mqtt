package llm

import (
	"context"
	"fmt"
	"time"

	"rtk_controller/internal/qos"
	"rtk_controller/pkg/types"
)

// QoSGetStatusTool implements the qos.get_status LLM tool
type QoSGetStatusTool struct {
	qosManager *qos.QoSManager
}

// NewQoSGetStatusTool creates a new qos.get_status tool
func NewQoSGetStatusTool(qosManager *qos.QoSManager) *QoSGetStatusTool {
	return &QoSGetStatusTool{
		qosManager: qosManager,
	}
}

// Name returns the tool name
func (q *QoSGetStatusTool) Name() string {
	return "qos.get_status"
}

// Category returns the tool category
func (q *QoSGetStatusTool) Category() types.ToolCategory {
	return types.ToolCategoryRead
}

// Description returns the tool description
func (q *QoSGetStatusTool) Description() string {
	return "Retrieves the current QoS status including policies, bandwidth usage, and recommendations"
}

// RequiredCapabilities returns the required device capabilities
func (q *QoSGetStatusTool) RequiredCapabilities() []string {
	return []string{"qos_query"}
}

// Validate validates the tool parameters
func (q *QoSGetStatusTool) Validate(params map[string]interface{}) error {
	// Optional device_id parameter
	if deviceID, exists := params["device_id"]; exists {
		if _, ok := deviceID.(string); !ok {
			return fmt.Errorf("device_id must be a string")
		}
	}
	
	// Optional include_recommendations parameter
	if includeRecs, exists := params["include_recommendations"]; exists {
		if _, ok := includeRecs.(bool); !ok {
			return fmt.Errorf("include_recommendations must be a boolean")
		}
	}
	
	return nil
}

// Execute executes the tool
func (q *QoSGetStatusTool) Execute(ctx context.Context, params map[string]interface{}) (*types.ToolResult, error) {
	result := &types.ToolResult{
		ToolName:  q.Name(),
		Success:   false,
		Timestamp: getCurrentTime(),
	}
	
	// Parse optional parameters
	var deviceID string
	if val, exists := params["device_id"]; exists {
		if s, ok := val.(string); ok {
			deviceID = s
		}
	}
	
	includeRecommendations := true // default to true
	if val, exists := params["include_recommendations"]; exists {
		if b, ok := val.(bool); ok {
			includeRecommendations = b
		}
	}
	
	// Get QoS information from the manager
	qosInfo := q.qosManager.GetQoSInfo()
	
	// Build response data
	statusData := map[string]interface{}{
		"device_id": deviceID,
		"status":    qosInfo,
		"metadata": map[string]interface{}{
			"retrieved_at": getCurrentTime(),
			"source":       "qos_manager",
		},
	}
	
	// Add recommendations if requested
	if includeRecommendations {
		recommendations := q.qosManager.GetRecommendations()
		statusData["recommendations"] = recommendations
	}
	
	result.Success = true
	result.Data = statusData
	
	return result, nil
}

// TrafficGetStatsTool implements the traffic.get_stats LLM tool
type TrafficGetStatsTool struct {
	qosManager *qos.QoSManager
}

// NewTrafficGetStatsTool creates a new traffic.get_stats tool
func NewTrafficGetStatsTool(qosManager *qos.QoSManager) *TrafficGetStatsTool {
	return &TrafficGetStatsTool{
		qosManager: qosManager,
	}
}

// Name returns the tool name
func (t *TrafficGetStatsTool) Name() string {
	return "traffic.get_stats"
}

// Category returns the tool category
func (t *TrafficGetStatsTool) Category() types.ToolCategory {
	return types.ToolCategoryRead
}

// Description returns the tool description
func (t *TrafficGetStatsTool) Description() string {
	return "Retrieves detailed traffic statistics including bandwidth usage, top talkers, and usage patterns"
}

// RequiredCapabilities returns the required device capabilities
func (t *TrafficGetStatsTool) RequiredCapabilities() []string {
	return []string{"traffic_analysis"}
}

// Validate validates the tool parameters
func (t *TrafficGetStatsTool) Validate(params map[string]interface{}) error {
	// Optional device_id parameter
	if deviceID, exists := params["device_id"]; exists {
		if _, ok := deviceID.(string); !ok {
			return fmt.Errorf("device_id must be a string")
		}
	}
	
	// Optional time_range parameter (in hours)
	if timeRange, exists := params["time_range_hours"]; exists {
		if hours, ok := timeRange.(float64); ok {
			if hours <= 0 || hours > 168 { // max 7 days
				return fmt.Errorf("time_range_hours must be between 0 and 168")
			}
		} else {
			return fmt.Errorf("time_range_hours must be a number")
		}
	}
	
	// Optional include_top_talkers parameter
	if includeTopTalkers, exists := params["include_top_talkers"]; exists {
		if _, ok := includeTopTalkers.(bool); !ok {
			return fmt.Errorf("include_top_talkers must be a boolean")
		}
	}
	
	return nil
}

// Execute executes the tool
func (t *TrafficGetStatsTool) Execute(ctx context.Context, params map[string]interface{}) (*types.ToolResult, error) {
	result := &types.ToolResult{
		ToolName:  t.Name(),
		Success:   false,
		Timestamp: getCurrentTime(),
	}
	
	// Parse optional parameters
	var deviceID string
	if val, exists := params["device_id"]; exists {
		if s, ok := val.(string); ok {
			deviceID = s
		}
	}
	
	timeRangeHours := 24.0 // default to last 24 hours
	if val, exists := params["time_range_hours"]; exists {
		if hours, ok := val.(float64); ok {
			timeRangeHours = hours
		}
	}
	
	includeTopTalkers := true // default to true
	if val, exists := params["include_top_talkers"]; exists {
		if b, ok := val.(bool); ok {
			includeTopTalkers = b
		}
	}
	
	// Get traffic statistics from QoS manager via traffic analyzer
	stats := t.qosManager.GetQoSInfo().TrafficStats
	
	// Build response data
	statsData := map[string]interface{}{
		"device_id":   deviceID,
		"time_range":  fmt.Sprintf("%.1f hours", timeRangeHours),
		"statistics":  stats,
		"metadata": map[string]interface{}{
			"retrieved_at": getCurrentTime(),
			"source":       "qos_manager",
		},
	}
	
	// Add top talkers if requested
	if includeTopTalkers {
		// Get top talkers from traffic stats
		if stats != nil && len(stats.TopTalkers) > 0 {
			statsData["top_talkers"] = stats.TopTalkers
		} else {
			statsData["top_talkers"] = []interface{}{}
		}
	}
	
	result.Success = true
	result.Data = statsData
	
	return result, nil
}

// getCurrentTime returns the current time
func getCurrentTime() time.Time {
	return time.Now()
}
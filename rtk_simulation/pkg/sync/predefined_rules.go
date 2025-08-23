package sync

import (
	"time"
)

// GetPredefinedSyncRules 獲取預定義的同步規則
func GetPredefinedSyncRules() []SyncRule {
	return []SyncRule{
		// 路由器離線時的設備同步
		{
			ID:   "router_offline_sync",
			Name: "Router Offline Device Sync",
			Source: StatePattern{
				DeviceType: "router",
				Field:      "health",
				Operator:   "eq",
				Value:      "error",
			},
			Targets: []SyncAction{
				{
					TargetType: "smart_bulb",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":     "power_off",
						"reason":      "network_disconnected",
						"auto_resume": true,
					},
				},
				{
					TargetType: "air_conditioner",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command": "set_mode",
						"mode":    "offline",
					},
				},
			},
			Conditions: []SyncCondition{
				{
					DeviceID: "main_router",
					Field:    "uptime",
					Operator: "gt",
					Value:    300, // 運行超過5分鐘才觸發
					Required: true,
				},
			},
			Enabled:  true,
			Priority: 10,
			Delay:    5 * time.Second,
		},

		// 智能燈泡狀態同步
		{
			ID:   "bulb_brightness_sync",
			Name: "Smart Bulb Brightness Sync",
			Source: StatePattern{
				DeviceType: "smart_bulb",
				Field:      "brightness",
				Operator:   "gt",
				Value:      80,
			},
			Targets: []SyncAction{
				{
					TargetType: "smart_bulb",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":    "set_brightness",
						"brightness": 80,
						"duration":   "5s",
					},
				},
			},
			Enabled:  true,
			Priority: 5,
			Delay:    2 * time.Second,
		},

		// 空調溫度同步
		{
			ID:   "ac_temperature_sync",
			Name: "Air Conditioner Temperature Sync",
			Source: StatePattern{
				DeviceType: "air_conditioner",
				Field:      "current_temperature",
				Operator:   "gt",
				Value:      28.0,
			},
			Targets: []SyncAction{
				{
					TargetType: "air_conditioner",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":     "set_temperature",
						"temperature": 26.0,
						"mode":        "cool",
						"fan_speed":   "auto",
					},
				},
			},
			Conditions: []SyncCondition{
				{
					Field:    "power_state",
					Operator: "eq",
					Value:    "on",
					Required: true,
				},
			},
			Enabled:  true,
			Priority: 7,
			Delay:    10 * time.Second,
		},

		// 智能手機電量低時設備節能同步
		{
			ID:   "phone_battery_energy_sync",
			Name: "Phone Low Battery Energy Sync",
			Source: StatePattern{
				DeviceType: "smartphone",
				Field:      "battery_level",
				Operator:   "lt",
				Value:      20.0,
			},
			Targets: []SyncAction{
				{
					TargetType: "smart_bulb",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":    "set_brightness",
						"brightness": 30,
						"color":      "warm_white",
					},
				},
				{
					TargetType: "air_conditioner",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":     "set_mode",
						"mode":        "eco",
						"temperature": 25.0,
					},
				},
			},
			Enabled:  true,
			Priority: 6,
			Delay:    1 * time.Second,
		},

		// 設備健康狀態同步
		{
			ID:   "device_health_sync",
			Name: "Device Health Status Sync",
			Source: StatePattern{
				Field:    "health",
				Operator: "eq",
				Value:    "error",
			},
			Targets: []SyncAction{
				{
					TargetType: "router",
					Action:     "trigger_event",
					Parameters: map[string]interface{}{
						"event_type":    "device_health_alert",
						"severity":      "warning",
						"auto_diagnose": true,
					},
				},
			},
			Enabled:  true,
			Priority: 9,
			Delay:    0,
		},

		// 夜間模式設備同步
		{
			ID:   "night_mode_sync",
			Name: "Night Mode Device Sync",
			Source: StatePattern{
				DeviceID: "living_room_light",
				Field:    "power_state",
				Operator: "eq",
				Value:    "off",
			},
			Targets: []SyncAction{
				{
					TargetType: "air_conditioner",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":     "set_mode",
						"mode":        "sleep",
						"temperature": 26.5,
						"fan_speed":   "quiet",
					},
				},
				{
					TargetType: "smartphone",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":       "set_power_mode",
						"power_mode":    "sleep",
						"notifications": "silent",
					},
				},
			},
			Conditions: []SyncCondition{
				{
					Field:    "current_time_hour",
					Operator: "gt",
					Value:    22,
					Required: true,
				},
			},
			Enabled:  true,
			Priority: 4,
			Delay:    30 * time.Second,
		},

		// 網路品質同步
		{
			ID:   "network_quality_sync",
			Name: "Network Quality Sync",
			Source: StatePattern{
				DeviceType: "router",
				Field:      "network_quality",
				Operator:   "lt",
				Value:      0.7,
			},
			Targets: []SyncAction{
				{
					TargetType: "smartphone",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":         "adjust_network",
						"quality_mode":    "adaptive",
						"background_sync": false,
					},
				},
			},
			Enabled:  true,
			Priority: 3,
			Delay:    5 * time.Second,
		},

		// 安全模式設備同步
		{
			ID:   "security_mode_sync",
			Name: "Security Mode Device Sync",
			Source: StatePattern{
				Field:    "security_alert",
				Operator: "eq",
				Value:    true,
			},
			Targets: []SyncAction{
				{
					TargetType: "smart_bulb",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":    "set_brightness",
						"brightness": 100,
						"color":      "red",
						"blink":      true,
					},
				},
				{
					TargetType: "smartphone",
					Action:     "trigger_event",
					Parameters: map[string]interface{}{
						"event_type": "security_alert",
						"priority":   "critical",
						"sound":      "alarm",
					},
				},
			},
			Enabled:  true,
			Priority: 10,
			Delay:    0,
		},

		// 能源管理同步
		{
			ID:   "energy_management_sync",
			Name: "Energy Management Sync",
			Source: StatePattern{
				Field:    "power_consumption",
				Operator: "gt",
				Value:    80.0, // 總功耗超過80%
			},
			Targets: []SyncAction{
				{
					TargetType: "air_conditioner",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":   "set_mode",
						"mode":      "eco",
						"max_power": 60,
					},
				},
				{
					TargetType: "smart_bulb",
					Action:     "send_command",
					Parameters: map[string]interface{}{
						"command":       "set_brightness",
						"brightness":    50,
						"energy_saving": true,
					},
				},
			},
			Enabled:  true,
			Priority: 8,
			Delay:    10 * time.Second,
		},

		// 設備重啟同步
		{
			ID:   "device_restart_sync",
			Name: "Device Restart Sync",
			Source: StatePattern{
				Field:    "uptime",
				Operator: "lt",
				Value:    60, // 運行時間少於60秒，表示剛重啟
			},
			Targets: []SyncAction{
				{
					Action: "trigger_event",
					Parameters: map[string]interface{}{
						"event_type":             "device_restart",
						"restore_previous_state": true,
						"check_connections":      true,
					},
				},
			},
			Enabled:  true,
			Priority: 2,
			Delay:    15 * time.Second,
		},
	}
}

// GetSyncRulesByCategory 根據類別獲取同步規則
func GetSyncRulesByCategory(category string) []SyncRule {
	allRules := GetPredefinedSyncRules()
	var filteredRules []SyncRule

	for _, rule := range allRules {
		switch category {
		case "energy":
			if rule.ID == "phone_battery_energy_sync" || rule.ID == "energy_management_sync" {
				filteredRules = append(filteredRules, rule)
			}
		case "security":
			if rule.ID == "security_mode_sync" || rule.ID == "device_health_sync" {
				filteredRules = append(filteredRules, rule)
			}
		case "comfort":
			if rule.ID == "ac_temperature_sync" || rule.ID == "bulb_brightness_sync" || rule.ID == "night_mode_sync" {
				filteredRules = append(filteredRules, rule)
			}
		case "network":
			if rule.ID == "router_offline_sync" || rule.ID == "network_quality_sync" {
				filteredRules = append(filteredRules, rule)
			}
		case "maintenance":
			if rule.ID == "device_restart_sync" || rule.ID == "device_health_sync" {
				filteredRules = append(filteredRules, rule)
			}
		default:
			filteredRules = allRules
		}
	}

	return filteredRules
}

// CreateCustomSyncRule 創建自定義同步規則
func CreateCustomSyncRule(id, name string, source StatePattern, targets []SyncAction, conditions []SyncCondition, priority int, delay time.Duration) SyncRule {
	return SyncRule{
		ID:         id,
		Name:       name,
		Source:     source,
		Targets:    targets,
		Conditions: conditions,
		Enabled:    true,
		Priority:   priority,
		Delay:      delay,
	}
}

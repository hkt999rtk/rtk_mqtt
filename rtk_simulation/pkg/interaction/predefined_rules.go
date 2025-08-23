package interaction

import (
	"time"
)

// GetPredefinedRules 獲取預定義的互動規則
func GetPredefinedRules() []InteractionRule {
	return []InteractionRule{
		// 智能照明規則：當有人進入房間時開燈
		{
			ID:   "motion_light_on",
			Name: "Motion Detection Light On",
			Trigger: TriggerCondition{
				Type:      "event",
				DeviceID:  "front_door_camera",
				EventType: "motion_detected",
			},
			Actions: []InteractionAction{
				{
					Type:     "command",
					TargetID: "living_room_light",
					Command:  "power_on",
					Parameters: map[string]interface{}{
						"brightness": 80,
					},
				},
			},
			Probability: 0.9,
			Cooldown:    5 * time.Minute,
			Priority:    5,
		},

		// 節能規則：夜晚時降低燈泡亮度
		{
			ID:   "night_dim_lights",
			Name: "Night Time Dimming",
			Trigger: TriggerCondition{
				Type:        "time",
				TimePattern: "night",
			},
			Actions: []InteractionAction{
				{
					Type:     "command",
					TargetID: "living_room_light",
					Command:  "set_brightness",
					Parameters: map[string]interface{}{
						"brightness": 20,
					},
				},
			},
			Probability: 1.0,
			Cooldown:    8 * time.Hour,
			Priority:    3,
		},

		// 溫度控制：根據溫度感測器調整空調
		{
			ID:   "temperature_control",
			Name: "Auto Temperature Control",
			Trigger: TriggerCondition{
				Type:      "event",
				DeviceID:  "kitchen_sensor",
				EventType: "temperature_alert",
			},
			Actions: []InteractionAction{
				{
					Type:     "command",
					TargetID: "bedroom_ac",
					Command:  "set_temperature",
					Parameters: map[string]interface{}{
						"temperature": 24.0,
						"mode":        "auto",
					},
				},
			},
			Probability: 0.8,
			Cooldown:    15 * time.Minute,
			Priority:    7,
		},

		// 安全模式：檢測到異常時觸發警報
		{
			ID:   "security_alert",
			Name: "Security Alert System",
			Trigger: TriggerCondition{
				Type:      "event",
				DeviceID:  "front_door_camera",
				EventType: "suspicious_activity",
			},
			Actions: []InteractionAction{
				{
					Type:     "notification",
					TargetID: "smartphone_john",
					Command:  "send_alert",
					Parameters: map[string]interface{}{
						"message":  "Security Alert: Suspicious activity detected at front door",
						"priority": "high",
					},
				},
				{
					Type:     "command",
					TargetID: "living_room_light",
					Command:  "power_on",
					Parameters: map[string]interface{}{
						"brightness": 100,
						"color":      "red",
					},
					Delay: 2 * time.Second,
				},
			},
			Probability: 1.0,
			Cooldown:    30 * time.Second,
			Priority:    10,
		},

		// 早晨例行：根據時間激活早晨場景
		{
			ID:   "morning_routine",
			Name: "Morning Routine Activation",
			Trigger: TriggerCondition{
				Type:        "time",
				TimePattern: "07:00",
			},
			Actions: []InteractionAction{
				{
					Type:     "command",
					TargetID: "living_room_light",
					Command:  "power_on",
					Parameters: map[string]interface{}{
						"brightness": 60,
						"color":      "warm_white",
					},
				},
				{
					Type:     "command",
					TargetID: "bedroom_ac",
					Command:  "set_temperature",
					Parameters: map[string]interface{}{
						"temperature": 22.0,
						"mode":        "heat",
					},
					Delay: 1 * time.Minute,
				},
			},
			Probability: 0.7,
			Cooldown:    24 * time.Hour,
			Priority:    4,
		},

		// 離家模式：檢測到所有設備離線時啟動節能
		{
			ID:   "away_mode",
			Name: "Away Mode Energy Saving",
			Trigger: TriggerCondition{
				Type:      "event",
				DeviceID:  "smartphone_john",
				EventType: "device_disconnected",
			},
			Actions: []InteractionAction{
				{
					Type:     "command",
					TargetID: "living_room_light",
					Command:  "power_off",
				},
				{
					Type:     "command",
					TargetID: "bedroom_ac",
					Command:  "set_temperature",
					Parameters: map[string]interface{}{
						"temperature": 20.0,
						"mode":        "eco",
					},
					Delay: 5 * time.Minute,
				},
			},
			Probability: 0.9,
			Cooldown:    10 * time.Minute,
			Priority:    6,
		},

		// 空氣質量控制：根據空氣質量調整設備
		{
			ID:   "air_quality_control",
			Name: "Air Quality Management",
			Trigger: TriggerCondition{
				Type:      "event",
				DeviceID:  "kitchen_sensor",
				EventType: "air_quality_alert",
			},
			Actions: []InteractionAction{
				{
					Type:     "command",
					TargetID: "bedroom_ac",
					Command:  "set_mode",
					Parameters: map[string]interface{}{
						"mode":      "purify",
						"fan_speed": "high",
					},
				},
				{
					Type:     "notification",
					TargetID: "smartphone_john",
					Command:  "send_notification",
					Parameters: map[string]interface{}{
						"message": "Air quality alert - air purifier activated",
					},
				},
			},
			Probability: 1.0,
			Cooldown:    30 * time.Minute,
			Priority:    8,
		},

		// 電量管理：設備電量低時發送通知
		{
			ID:   "battery_management",
			Name: "Low Battery Alert",
			Trigger: TriggerCondition{
				Type:      "event",
				DeviceID:  "kitchen_sensor",
				EventType: "battery_low",
			},
			Actions: []InteractionAction{
				{
					Type:     "notification",
					TargetID: "smartphone_john",
					Command:  "send_notification",
					Parameters: map[string]interface{}{
						"message":  "Kitchen sensor battery is low - please replace",
						"priority": "medium",
					},
				},
			},
			Probability: 1.0,
			Cooldown:    6 * time.Hour,
			Priority:    5,
		},

		// 網路優化：檢測到網路擁塞時調整 QoS
		{
			ID:   "network_optimization",
			Name: "Network QoS Optimization",
			Trigger: TriggerCondition{
				Type:      "event",
				DeviceID:  "main_router",
				EventType: "network_congestion",
			},
			Actions: []InteractionAction{
				{
					Type:     "command",
					TargetID: "main_router",
					Command:  "adjust_qos",
					Parameters: map[string]interface{}{
						"priority_device": "laptop_mary",
						"bandwidth_limit": 80,
					},
				},
			},
			Probability: 0.8,
			Cooldown:    5 * time.Minute,
			Priority:    6,
		},

		// 睡眠模式：晚上時間啟動睡眠場景
		{
			ID:   "sleep_mode",
			Name: "Sleep Mode Activation",
			Trigger: TriggerCondition{
				Type:        "time",
				TimePattern: "22:30",
			},
			Actions: []InteractionAction{
				{
					Type:     "command",
					TargetID: "living_room_light",
					Command:  "power_off",
				},
				{
					Type:     "command",
					TargetID: "bedroom_ac",
					Command:  "set_temperature",
					Parameters: map[string]interface{}{
						"temperature": 26.0,
						"mode":        "sleep",
					},
					Delay: 30 * time.Second,
				},
				{
					Type:     "automation",
					TargetID: "system",
					Command:  "activate_scene",
					Parameters: map[string]interface{}{
						"type":       "scene_activation",
						"scene_name": "sleep_mode",
					},
				},
			},
			Probability: 0.8,
			Cooldown:    24 * time.Hour,
			Priority:    7,
		},
	}
}

// GetInteractionRulesByCategory 根據類別獲取互動規則
func GetInteractionRulesByCategory(category string) []InteractionRule {
	allRules := GetPredefinedRules()
	var filteredRules []InteractionRule

	for _, rule := range allRules {
		switch category {
		case "lighting":
			if rule.ID == "motion_light_on" || rule.ID == "night_dim_lights" {
				filteredRules = append(filteredRules, rule)
			}
		case "climate":
			if rule.ID == "temperature_control" || rule.ID == "air_quality_control" {
				filteredRules = append(filteredRules, rule)
			}
		case "security":
			if rule.ID == "security_alert" {
				filteredRules = append(filteredRules, rule)
			}
		case "energy":
			if rule.ID == "away_mode" || rule.ID == "battery_management" {
				filteredRules = append(filteredRules, rule)
			}
		case "automation":
			if rule.ID == "morning_routine" || rule.ID == "sleep_mode" {
				filteredRules = append(filteredRules, rule)
			}
		case "network":
			if rule.ID == "network_optimization" {
				filteredRules = append(filteredRules, rule)
			}
		default:
			filteredRules = allRules
		}
	}

	return filteredRules
}

// CreateCustomRule 創建自定義互動規則
func CreateCustomRule(id, name string, trigger TriggerCondition, actions []InteractionAction, probability float64, cooldown time.Duration, priority int) InteractionRule {
	return InteractionRule{
		ID:          id,
		Name:        name,
		Trigger:     trigger,
		Actions:     actions,
		Probability: probability,
		Cooldown:    cooldown,
		Priority:    priority,
		Enabled:     true,
	}
}

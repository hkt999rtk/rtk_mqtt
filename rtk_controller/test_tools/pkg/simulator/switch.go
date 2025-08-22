package simulator

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"rtk_test_tools/pkg/types"
)

// SwitchSimulator 交換機設備模擬器
type SwitchSimulator struct {
	*BaseSimulator
	
	// 交換機狀態
	ports       map[string]types.PortInfo
	vlans       []int
	stpStatus   string
	macTable    int // MAC 地址表項目數量
	portStats   map[string]PortStatistics
	fanSpeed    int // 風扇轉速
	powerStatus string
}

// PortStatistics 端口統計資訊
type PortStatistics struct {
	RxPackets int64 `json:"rx_packets"`
	TxPackets int64 `json:"tx_packets"`
	RxBytes   int64 `json:"rx_bytes"`
	TxBytes   int64 `json:"tx_bytes"`
	Errors    int64 `json:"errors"`
}

// NewSwitchSimulator 創建交換機模擬器
func NewSwitchSimulator(config types.DeviceConfig, mqttConfig types.MQTTConfig, verbose bool) (*SwitchSimulator, error) {
	base, err := NewBaseSimulator(config, mqttConfig, verbose)
	if err != nil {
		return nil, err
	}
	
	// 初始化端口（預設 24 個端口）
	ports := make(map[string]types.PortInfo)
	portStats := make(map[string]PortStatistics)
	
	portCount := 24
	if pc, ok := config.Properties["port_count"]; ok {
		switch v := pc.(type) {
		case int:
			portCount = v
		case float64:
			portCount = int(v)
		default:
			portCount = 24 // 預設值
		}
	}
	
	for i := 1; i <= portCount; i++ {
		portID := fmt.Sprintf("%d", i)
		
		// 隨機設置端口狀態
		status := "up"
		speed := 1000
		duplex := "full"
		
		if rand.Float64() < 0.2 { // 20% 端口可能是 down
			status = "down"
			speed = 0
			duplex = ""
		} else if rand.Float64() < 0.1 { // 10% 端口是 100Mbps
			speed = 100
		}
		
		ports[portID] = types.PortInfo{
			Status:    status,
			SpeedMbps: &speed,
			Duplex:    &duplex,
		}
		
		// 初始化端口統計
		portStats[portID] = PortStatistics{
			RxPackets: int64(rand.Intn(100000)),
			TxPackets: int64(rand.Intn(80000)),
			RxBytes:   int64(rand.Intn(50000000)),
			TxBytes:   int64(rand.Intn(40000000)),
			Errors:    0,
		}
	}
	
	return &SwitchSimulator{
		BaseSimulator: base,
		ports:         ports,
		vlans:         []int{1, 10, 20}, // 預設 VLAN
		stpStatus:     "enabled",
		macTable:      50 + rand.Intn(200), // 50-250 MAC entries
		portStats:     portStats,
		fanSpeed:      2000 + rand.Intn(1000), // 2000-3000 RPM
		powerStatus:   "normal",
	}, nil
}

// Start 啟動交換機模擬器
func (sw *SwitchSimulator) Start(ctx context.Context) error {
	if err := sw.BaseSimulator.Start(ctx); err != nil {
		return err
	}
	
	// 啟動交換機特定的循環
	go sw.switchStateLoop(ctx)
	go sw.switchTelemetryLoop(ctx)
	go sw.switchEventLoop(ctx)
	
	return nil
}

// GenerateStatePayload 生成交換機狀態 payload
func (sw *SwitchSimulator) GenerateStatePayload() types.StatePayload {
	payload := sw.GenerateBaseStatePayload()
	
	// 設置交換機 IP
	if ip, ok := sw.config.Properties["ip"]; ok {
		payload.Net.IP = ip.(string)
	} else {
		payload.Net.IP = "10.0.1.10"
	}
	
	if gateway, ok := sw.config.Properties["gateway"]; ok {
		payload.Net.Gateway = gateway.(string)
	} else {
		payload.Net.Gateway = "10.0.1.1"
	}
	
	// 添加交換機特定欄位
	payload.Extra = map[string]interface{}{
		"ports":        sw.ports,
		"vlans":        sw.vlans,
		"stp_status":   sw.stpStatus,
		"mac_table":    sw.macTable,
		"fan_speed":    sw.fanSpeed,
		"power_status": sw.powerStatus,
	}
	
	return payload
}

// GenerateTelemetryData 生成交換機遙測資料
func (sw *SwitchSimulator) GenerateTelemetryData() map[string]types.TelemetryPayload {
	telemetry := make(map[string]types.TelemetryPayload)
	
	// 端口統計
	telemetry["port_stats"] = types.TelemetryPayload{
		"port_statistics": sw.portStats,
	}
	
	// 交換統計
	totalRxPackets := int64(0)
	totalTxPackets := int64(0)
	totalRxBytes := int64(0)
	totalTxBytes := int64(0)
	
	for _, stats := range sw.portStats {
		totalRxPackets += stats.RxPackets
		totalTxPackets += stats.TxPackets
		totalRxBytes += stats.RxBytes
		totalTxBytes += stats.TxBytes
	}
	
	telemetry["switching_stats"] = types.TelemetryPayload{
		"mac_table_size":    sw.macTable,
		"mac_table_max":     1024,
		"total_rx_packets":  totalRxPackets,
		"total_tx_packets":  totalTxPackets,
		"total_rx_bytes":    totalRxBytes,
		"total_tx_bytes":    totalTxBytes,
		"forwarding_rate":   rand.Float64() * 100, // 0-100%
		"broadcast_packets": rand.Intn(1000),
		"multicast_packets": rand.Intn(5000),
	}
	
	// VLAN 統計
	vlanStats := make(map[string]interface{})
	for _, vlan := range sw.vlans {
		vlanStats[fmt.Sprintf("vlan_%d", vlan)] = map[string]interface{}{
			"active_ports": rand.Intn(10) + 1,
			"traffic_mb":   rand.Float64() * 100,
		}
	}
	
	telemetry["vlan_stats"] = types.TelemetryPayload(vlanStats)
	
	// 系統統計
	telemetry["system_stats"] = types.TelemetryPayload{
		"fan_speed":     sw.fanSpeed,
		"power_status":  sw.powerStatus,
		"power_usage":   40.0 + rand.Float64()*20.0, // 40-60W
		"uplinks_active": 2,
		"stp_status":    sw.stpStatus,
	}
	
	return telemetry
}

// GenerateEvents 生成交換機事件
func (sw *SwitchSimulator) GenerateEvents() []EventData {
	var events []EventData
	
	// 端口狀態變化事件
	if rand.Float64() < 0.05 { // 5% 機率
		for portID, port := range sw.ports {
			if rand.Float64() < 0.1 { // 每個端口 10% 機率
				if port.Status == "up" && rand.Float64() < 0.3 { // 30% 機率斷線
					port.Status = "down"
					port.SpeedMbps = nil
					port.Duplex = nil
					sw.ports[portID] = port
					
					events = append(events, EventData{
						EventType: "port.link_down",
						Payload: types.EventPayload{
							EventType: "port.link_down",
							Severity:  "warning",
							Extra: map[string]interface{}{
								"port_id": portID,
								"reason":  "link_failure",
							},
						},
					})
				} else if port.Status == "down" { // 恢復連線
					port.Status = "up"
					speed := 1000
					duplex := "full"
					port.SpeedMbps = &speed
					port.Duplex = &duplex
					sw.ports[portID] = port
					
					events = append(events, EventData{
						EventType: "port.link_up",
						Payload: types.EventPayload{
							EventType: "port.link_up",
							Severity:  "info",
							Extra: map[string]interface{}{
								"port_id":    portID,
								"speed_mbps": speed,
								"duplex":     duplex,
							},
						},
					})
				}
			}
		}
	}
	
	// 高溫警告
	if sw.tempC > 55.0 && rand.Float64() < 0.1 { // 10% 機率
		events = append(events, EventData{
			EventType: "system.temperature_high",
			Payload: types.EventPayload{
				EventType: "system.temperature_high",
				Severity:  "warning",
				Extra: map[string]interface{}{
					"temperature":  sw.tempC,
					"threshold":    55.0,
					"fan_speed":    sw.fanSpeed,
					"action_taken": "increase_fan_speed",
				},
			},
		})
	}
	
	// 風扇異常
	if sw.fanSpeed < 1500 && rand.Float64() < 0.02 { // 2% 機率
		events = append(events, EventData{
			EventType: "hardware.fan_failure",
			Payload: types.EventPayload{
				EventType: "hardware.fan_failure",
				Severity:  "error",
				Extra: map[string]interface{}{
					"fan_id":    1,
					"fan_speed": sw.fanSpeed,
					"min_speed": 1500,
				},
			},
		})
		sw.powerStatus = "degraded" // 風扇異常導致電源狀態降級
	}
	
	// 電源狀態變化
	if sw.powerStatus == "degraded" && rand.Float64() < 0.1 { // 10% 機率恢復
		sw.powerStatus = "normal"
		sw.fanSpeed = 2000 + rand.Intn(1000) // 恢復正常轉速
		
		events = append(events, EventData{
			EventType: "hardware.power_restored",
			Payload: types.EventPayload{
				EventType: "hardware.power_restored",
				Severity:  "info",
				Extra: map[string]interface{}{
					"power_status": sw.powerStatus,
					"fan_speed":    sw.fanSpeed,
				},
			},
		})
	}
	
	// MAC 地址表滿警告
	if sw.macTable > 900 { // 接近 1024 上限
		events = append(events, EventData{
			EventType: "system.mac_table_full",
			Payload: types.EventPayload{
				EventType: "system.mac_table_full",
				Severity:  "warning",
				Extra: map[string]interface{}{
					"mac_table_size": sw.macTable,
					"mac_table_max":  1024,
					"utilization":    float64(sw.macTable) / 1024.0 * 100,
				},
			},
		})
	}
	
	return events
}

// UpdateStatus 更新交換機特定狀態
func (sw *SwitchSimulator) UpdateStatus() {
	sw.BaseSimulator.UpdateStatus()
	
	// 更新端口統計
	for portID, port := range sw.ports {
		if port.Status == "up" {
			stats := sw.portStats[portID]
			
			// 增加流量統計
			rxInc := int64(rand.Intn(1000))
			txInc := int64(rand.Intn(1000))
			
			stats.RxPackets += rxInc
			stats.TxPackets += txInc
			stats.RxBytes += rxInc * 1024
			stats.TxBytes += txInc * 1024
			
			// 偶爾產生錯誤
			if rand.Float64() < 0.001 { // 0.1% 機率
				stats.Errors++
			}
			
			sw.portStats[portID] = stats
		}
	}
	
	// 更新 MAC 地址表大小
	sw.macTable += rand.Intn(10) - 5 // -5 to +5
	if sw.macTable < 20 {
		sw.macTable = 20
	} else if sw.macTable > 1024 {
		sw.macTable = 1024
	}
	
	// 根據溫度調整風扇轉速
	if sw.tempC > 50.0 {
		sw.fanSpeed = 2500 + rand.Intn(500) // 高轉速
	} else {
		sw.fanSpeed = 2000 + rand.Intn(500) // 正常轉速
	}
	
	// 確保風扇轉速不會太低
	if sw.fanSpeed < 1000 {
		sw.fanSpeed = 1000
	}
}

// 交換機特定的狀態循環
func (sw *SwitchSimulator) switchStateLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(sw.config.Intervals.StateS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !sw.IsRunning() {
				return
			}
			
			payload := sw.GenerateStatePayload()
			if err := sw.GetMQTTClient().PublishState(payload); err != nil && sw.verbose {
				log.Printf("[%s] Failed to publish state: %v", sw.GetDeviceID(), err)
			}
		}
	}
}

// 交換機特定的遙測循環
func (sw *SwitchSimulator) switchTelemetryLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(sw.config.Intervals.TelemetryS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !sw.IsRunning() {
				return
			}
			
			telemetryData := sw.GenerateTelemetryData()
			for metric, payload := range telemetryData {
				if err := sw.GetMQTTClient().PublishTelemetry(metric, payload); err != nil && sw.verbose {
					log.Printf("[%s] Failed to publish telemetry %s: %v", sw.GetDeviceID(), metric, err)
				}
			}
		}
	}
}

// 交換機特定的事件循環
func (sw *SwitchSimulator) switchEventLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(sw.config.Intervals.EventS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !sw.IsRunning() {
				return
			}
			
			events := sw.GenerateEvents()
			for _, event := range events {
				if err := sw.GetMQTTClient().PublishEvent(event.EventType, event.Payload); err != nil && sw.verbose {
					log.Printf("[%s] Failed to publish event %s: %v", sw.GetDeviceID(), event.EventType, err)
				}
			}
		}
	}
}
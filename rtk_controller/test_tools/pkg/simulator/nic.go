package simulator

import (
	"context"
	"log"
	"math/rand"
	"time"

	"rtk_test_tools/pkg/types"
)

// NICSimulator NIC 網卡設備模擬器
type NICSimulator struct {
	*BaseSimulator
	
	// NIC 狀態
	interfaceInfo types.InterfaceInfo
	rxPackets     int64
	txPackets     int64
	rxBytes       int64
	txBytes       int64
	rxErrors      int64
	txErrors      int64
	linkQuality   int
	driverVersion string
}

// NewNICSimulator 創建 NIC 模擬器
func NewNICSimulator(config types.DeviceConfig, mqttConfig types.MQTTConfig, verbose bool) (*NICSimulator, error) {
	base, err := NewBaseSimulator(config, mqttConfig, verbose)
	if err != nil {
		return nil, err
	}
	
	// 初始化介面資訊
	interfaceInfo := types.InterfaceInfo{
		Name:       "eth0",
		MACAddress: GenerateRandomMAC(),
		SpeedMbps:  1000,
		Duplex:     "full",
		LinkStatus: "up",
	}
	
	// 從配置中獲取介面名稱
	if ifname, ok := config.Properties["interface_name"]; ok {
		interfaceInfo.Name = ifname.(string)
	}
	
	return &NICSimulator{
		BaseSimulator: base,
		interfaceInfo: interfaceInfo,
		rxPackets:     int64(rand.Intn(50000)),
		txPackets:     int64(rand.Intn(40000)),
		rxBytes:       int64(rand.Intn(100000000)), // 0-100MB
		txBytes:       int64(rand.Intn(50000000)),  // 0-50MB
		rxErrors:      0,
		txErrors:      0,
		linkQuality:   95 + rand.Intn(5), // 95-100%
		driverVersion: "1.2.3",
	}, nil
}

// Start 啟動 NIC 模擬器
func (nic *NICSimulator) Start(ctx context.Context) error {
	if err := nic.BaseSimulator.Start(ctx); err != nil {
		return err
	}
	
	// 啟動 NIC 特定的循環
	go nic.nicStateLoop(ctx)
	go nic.nicTelemetryLoop(ctx)
	go nic.nicEventLoop(ctx)
	
	return nil
}

// GenerateStatePayload 生成 NIC 狀態 payload
func (nic *NICSimulator) GenerateStatePayload() types.StatePayload {
	payload := nic.GenerateBaseStatePayload()
	
	// 設置 NIC 特定欄位
	if ip, ok := nic.config.Properties["ip"]; ok {
		payload.Net.IP = ip.(string)
	} else {
		payload.Net.IP = "10.0.1.100"
	}
	
	if netmask, ok := nic.config.Properties["netmask"]; ok {
		payload.Net.Netmask = netmask.(string)
	} else {
		payload.Net.Netmask = "255.255.255.0"
	}
	
	if gateway, ok := nic.config.Properties["gateway"]; ok {
		payload.Net.Gateway = gateway.(string)
	} else {
		payload.Net.Gateway = "10.0.1.1"
	}
	
	// NIC 設備沒有磁碟和溫度感測器（通常）
	payload.DiskUsage = 0
	payload.TempC = 0
	
	// 添加介面資訊
	payload.Extra = map[string]interface{}{
		"interface": nic.interfaceInfo,
	}
	
	return payload
}

// GenerateTelemetryData 生成 NIC 遙測資料
func (nic *NICSimulator) GenerateTelemetryData() map[string]types.TelemetryPayload {
	telemetry := make(map[string]types.TelemetryPayload)
	
	// 介面統計
	telemetry["interface_stats"] = types.TelemetryPayload{
		"interface_name": nic.interfaceInfo.Name,
		"packets": map[string]interface{}{
			"rx_packets": nic.rxPackets,
			"tx_packets": nic.txPackets,
			"rx_errors":  nic.rxErrors,
			"tx_errors":  nic.txErrors,
		},
		"bytes": map[string]interface{}{
			"rx_bytes": nic.rxBytes,
			"tx_bytes": nic.txBytes,
		},
		"link_quality": nic.linkQuality,
	}
	
	// 網路性能
	telemetry["network_performance"] = types.TelemetryPayload{
		"bandwidth_utilization": rand.Float64() * 100, // 0-100%
		"latency_ms":           1.0 + rand.Float64()*5.0, // 1-6ms
		"jitter_ms":           rand.Float64() * 2.0,       // 0-2ms
		"packet_loss_rate":    rand.Float64() * 0.01,     // 0-1%
	}
	
	// 驅動資訊
	telemetry["driver_info"] = types.TelemetryPayload{
		"driver_name":    "rtl8139",
		"driver_version": nic.driverVersion,
		"firmware_version": "1.0.1",
		"bus_info":      "0000:02:00.0",
	}
	
	return telemetry
}

// GenerateEvents 生成 NIC 事件
func (nic *NICSimulator) GenerateEvents() []EventData {
	var events []EventData
	
	// 連結狀態變化事件
	if rand.Float64() < 0.01 { // 1% 機率
		if nic.interfaceInfo.LinkStatus == "up" && rand.Float64() < 0.3 { // 30% 機率斷線
			nic.interfaceInfo.LinkStatus = "down"
			events = append(events, EventData{
				EventType: "interface.link_down",
				Payload: types.EventPayload{
					EventType: "interface.link_down",
					Severity:  "error",
					Extra: map[string]interface{}{
						"interface_name": nic.interfaceInfo.Name,
						"mac_address":   nic.interfaceInfo.MACAddress,
						"reason":        "cable_disconnected",
					},
				},
			})
		} else if nic.interfaceInfo.LinkStatus == "down" { // 恢復連線
			nic.interfaceInfo.LinkStatus = "up"
			events = append(events, EventData{
				EventType: "interface.link_up",
				Payload: types.EventPayload{
					EventType: "interface.link_up",
					Severity:  "info",
					Extra: map[string]interface{}{
						"interface_name": nic.interfaceInfo.Name,
						"mac_address":   nic.interfaceInfo.MACAddress,
						"speed_mbps":    nic.interfaceInfo.SpeedMbps,
						"duplex":        nic.interfaceInfo.Duplex,
					},
				},
			})
		}
	}
	
	// 錯誤率過高警告
	totalPackets := nic.rxPackets + nic.txPackets
	totalErrors := nic.rxErrors + nic.txErrors
	if totalPackets > 1000 && float64(totalErrors)/float64(totalPackets) > 0.01 { // 錯誤率 > 1%
		events = append(events, EventData{
			EventType: "interface.high_error_rate",
			Payload: types.EventPayload{
				EventType: "interface.high_error_rate",
				Severity:  "warning",
				Extra: map[string]interface{}{
					"interface_name": nic.interfaceInfo.Name,
					"error_rate":    float64(totalErrors) / float64(totalPackets),
					"rx_errors":     nic.rxErrors,
					"tx_errors":     nic.txErrors,
					"total_packets": totalPackets,
				},
			},
		})
	}
	
	// 速度變化事件（例如從 1000Mbps 降到 100Mbps）
	if rand.Float64() < 0.005 { // 0.5% 機率
		oldSpeed := nic.interfaceInfo.SpeedMbps
		if oldSpeed == 1000 && rand.Float64() < 0.5 {
			nic.interfaceInfo.SpeedMbps = 100
			events = append(events, EventData{
				EventType: "interface.speed_change",
				Payload: types.EventPayload{
					EventType: "interface.speed_change",
					Severity:  "warning",
					Extra: map[string]interface{}{
						"interface_name": nic.interfaceInfo.Name,
						"old_speed":     oldSpeed,
						"new_speed":     nic.interfaceInfo.SpeedMbps,
						"reason":        "auto_negotiation",
					},
				},
			})
		} else if oldSpeed == 100 {
			nic.interfaceInfo.SpeedMbps = 1000
			events = append(events, EventData{
				EventType: "interface.speed_change",
				Payload: types.EventPayload{
					EventType: "interface.speed_change",
					Severity:  "info",
					Extra: map[string]interface{}{
						"interface_name": nic.interfaceInfo.Name,
						"old_speed":     oldSpeed,
						"new_speed":     nic.interfaceInfo.SpeedMbps,
						"reason":        "link_improvement",
					},
				},
			})
		}
	}
	
	return events
}

// UpdateStatus 更新 NIC 特定狀態
func (nic *NICSimulator) UpdateStatus() {
	nic.BaseSimulator.UpdateStatus()
	
	// 只有在連線時才更新統計
	if nic.interfaceInfo.LinkStatus == "up" {
		// 更新封包統計
		rxInc := int64(rand.Intn(1000))
		txInc := int64(rand.Intn(800))
		
		nic.rxPackets += rxInc
		nic.txPackets += txInc
		
		// 更新位元組統計（假設平均封包大小 1KB）
		nic.rxBytes += rxInc * 1024
		nic.txBytes += txInc * 1024
		
		// 偶爾產生錯誤
		if rand.Float64() < 0.001 { // 0.1% 機率
			if rand.Float64() < 0.5 {
				nic.rxErrors++
			} else {
				nic.txErrors++
			}
		}
		
		// 更新連線品質
		nic.linkQuality += rand.Intn(10) - 5 // -5 to +5 變化
		if nic.linkQuality < 50 {
			nic.linkQuality = 50
		} else if nic.linkQuality > 100 {
			nic.linkQuality = 100
		}
	}
}

// NIC 特定的狀態循環
func (nic *NICSimulator) nicStateLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(nic.config.Intervals.StateS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !nic.IsRunning() {
				return
			}
			
			payload := nic.GenerateStatePayload()
			if err := nic.GetMQTTClient().PublishState(payload); err != nil && nic.verbose {
				log.Printf("[%s] Failed to publish state: %v", nic.GetDeviceID(), err)
			}
		}
	}
}

// NIC 特定的遙測循環
func (nic *NICSimulator) nicTelemetryLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(nic.config.Intervals.TelemetryS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !nic.IsRunning() {
				return
			}
			
			telemetryData := nic.GenerateTelemetryData()
			for metric, payload := range telemetryData {
				if err := nic.GetMQTTClient().PublishTelemetry(metric, payload); err != nil && nic.verbose {
					log.Printf("[%s] Failed to publish telemetry %s: %v", nic.GetDeviceID(), metric, err)
				}
			}
		}
	}
}

// NIC 特定的事件循環
func (nic *NICSimulator) nicEventLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(nic.config.Intervals.EventS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !nic.IsRunning() {
				return
			}
			
			events := nic.GenerateEvents()
			for _, event := range events {
				if err := nic.GetMQTTClient().PublishEvent(event.EventType, event.Payload); err != nil && nic.verbose {
					log.Printf("[%s] Failed to publish event %s: %v", nic.GetDeviceID(), event.EventType, err)
				}
			}
		}
	}
}


package simulator

import (
	"context"
	"log"
	"math/rand"
	"time"

	"rtk_test_tools/pkg/types"
)

// GatewaySimulator Gateway/Router 設備模擬器
type GatewaySimulator struct {
	*BaseSimulator
	
	// WiFi 狀態
	wifiStatus    types.WiFiStatus
	clientCount   int
	bytesRx       int64
	bytesTx       int64
	packetsRx     int64
	packetsTx     int64
	connectionCount int
}

// NewGatewaySimulator 創建 Gateway 模擬器
func NewGatewaySimulator(config types.DeviceConfig, mqttConfig types.MQTTConfig, verbose bool) (*GatewaySimulator, error) {
	base, err := NewBaseSimulator(config, mqttConfig, verbose)
	if err != nil {
		return nil, err
	}
	
	// 初始化 WiFi 狀態
	wifiStatus := types.WiFiStatus{
		APMode:      true,
		ClientCount: 5 + rand.Intn(15), // 5-20 個客戶端
		Channel2G:   6,
		Channel5G:   36,
		TxPower2G:   20,
		TxPower5G:   23,
	}
	
	return &GatewaySimulator{
		BaseSimulator: base,
		wifiStatus:    wifiStatus,
		clientCount:   wifiStatus.ClientCount,
		bytesRx:       int64(rand.Intn(10000000)), // 初始流量
		bytesTx:       int64(rand.Intn(5000000)),
		packetsRx:     int64(rand.Intn(100000)),
		packetsTx:     int64(rand.Intn(80000)),
		connectionCount: 0,
	}, nil
}

// Start 啟動 Gateway 模擬器
func (g *GatewaySimulator) Start(ctx context.Context) error {
	if err := g.BaseSimulator.Start(ctx); err != nil {
		return err
	}
	
	// 啟動 Gateway 特定的循環
	go g.gatewayStateLoop(ctx)
	go g.gatewayTelemetryLoop(ctx)
	go g.gatewayEventLoop(ctx)
	
	return nil
}

// GenerateStatePayload 生成 Gateway 狀態 payload
func (g *GatewaySimulator) GenerateStatePayload() types.StatePayload {
	payload := g.GenerateBaseStatePayload()
	
	// 獲取 Gateway IP 配置
	if ip, ok := g.config.Properties["ip"]; ok {
		payload.Net.IP = ip.(string)
	} else {
		payload.Net.IP = "192.168.1.1"
	}
	
	if gateway, ok := g.config.Properties["gateway"]; ok {
		payload.Net.Gateway = gateway.(string)
	} else {
		payload.Net.Gateway = "10.0.1.1"
	}
	
	payload.Net.BytesRx = g.bytesRx
	payload.Net.BytesTx = g.bytesTx
	
	// 添加 WiFi 狀態
	payload.Extra = map[string]interface{}{
		"wifi_status": g.wifiStatus,
	}
	
	return payload
}

// GenerateTelemetryData 生成 Gateway 遙測資料
func (g *GatewaySimulator) GenerateTelemetryData() map[string]types.TelemetryPayload {
	telemetry := make(map[string]types.TelemetryPayload)
	
	// 網路統計
	telemetry["network_stats"] = types.TelemetryPayload{
		"throughput": map[string]interface{}{
			"rx_bytes":   g.bytesRx,
			"tx_bytes":   g.bytesTx,
			"rx_packets": g.packetsRx,
			"tx_packets": g.packetsTx,
		},
		"errors": map[string]interface{}{
			"rx_errors":   0,
			"tx_errors":   0,
			"collisions":  0,
		},
	}
	
	// WiFi 統計
	telemetry["wifi_stats"] = types.TelemetryPayload{
		"client_count":     g.clientCount,
		"channel_2g":       g.wifiStatus.Channel2G,
		"channel_5g":       g.wifiStatus.Channel5G,
		"signal_quality":   85 + rand.Intn(15), // 85-100%
		"interference":     rand.Intn(20),      // 0-20%
		"connection_count": g.connectionCount,
	}
	
	return telemetry
}

// GenerateEvents 生成 Gateway 事件
func (g *GatewaySimulator) GenerateEvents() []EventData {
	var events []EventData
	
	// 隨機生成客戶端連接/斷開事件
	if rand.Float64() < 0.1 { // 10% 機率
		if rand.Float64() < 0.7 { // 70% 連接，30% 斷開
			// 客戶端連接事件
			events = append(events, EventData{
				EventType: "wifi.client_connected",
				Payload: types.EventPayload{
					EventType: "wifi.client_connected",
					Severity:  "info",
					Extra: map[string]interface{}{
						"client_mac":      GenerateRandomMAC(),
						"client_ip":       GenerateRandomIP(),
						"signal_strength": -40 - rand.Intn(30), // -40 to -70 dBm
						"connection_time": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
					},
				},
			})
			g.connectionCount++
			
			// 可能增加客戶端計數
			if g.clientCount < 20 {
				g.clientCount++
				g.wifiStatus.ClientCount = g.clientCount
			}
		} else {
			// 客戶端斷開事件
			events = append(events, EventData{
				EventType: "wifi.client_disconnected",
				Payload: types.EventPayload{
					EventType: "wifi.client_disconnected",
					Severity:  "info",
					Extra: map[string]interface{}{
						"client_mac":        GenerateRandomMAC(),
						"reason":           "normal_disconnect",
						"session_duration": rand.Intn(3600), // 0-3600 秒
					},
				},
			})
			
			// 減少客戶端計數
			if g.clientCount > 1 {
				g.clientCount--
				g.wifiStatus.ClientCount = g.clientCount
			}
		}
	}
	
	// 系統告警事件
	if g.health == "warn" && rand.Float64() < 0.05 { // 5% 機率
		events = append(events, EventData{
			EventType: "system.warning",
			Payload: types.EventPayload{
				EventType: "system.warning",
				Severity:  "warning",
				Extra: map[string]interface{}{
					"warning_type": "high_load",
					"cpu_usage":   g.cpuUsage,
					"memory_usage": g.memoryUsage,
					"message":     "System load is high",
				},
			},
		})
	}
	
	return events
}

// UpdateStatus 更新 Gateway 特定狀態
func (g *GatewaySimulator) UpdateStatus() {
	g.BaseSimulator.UpdateStatus()
	
	// 更新網路流量（模擬網路活動）
	rxInc := int64(rand.Intn(100000)) // 0-100KB
	txInc := int64(rand.Intn(50000))  // 0-50KB
	
	g.bytesRx += rxInc
	g.bytesTx += txInc
	
	// 更新封包計數
	g.packetsRx += rxInc / 1024 // 假設平均封包大小 1KB
	g.packetsTx += txInc / 1024
	
	// 隨機調整客戶端數量
	if rand.Float64() < 0.1 { // 10% 機率變化
		if rand.Float64() < 0.5 && g.clientCount > 1 {
			g.clientCount--
		} else if g.clientCount < 20 {
			g.clientCount++
		}
		g.wifiStatus.ClientCount = g.clientCount
	}
}

// Gateway 特定的狀態循環
func (g *GatewaySimulator) gatewayStateLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(g.config.Intervals.StateS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !g.IsRunning() {
				return
			}
			
			payload := g.GenerateStatePayload()
			if err := g.GetMQTTClient().PublishState(payload); err != nil && g.verbose {
				log.Printf("[%s] Failed to publish state: %v", g.GetDeviceID(), err)
			}
		}
	}
}

// Gateway 特定的遙測循環
func (g *GatewaySimulator) gatewayTelemetryLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(g.config.Intervals.TelemetryS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !g.IsRunning() {
				return
			}
			
			telemetryData := g.GenerateTelemetryData()
			for metric, payload := range telemetryData {
				if err := g.GetMQTTClient().PublishTelemetry(metric, payload); err != nil && g.verbose {
					log.Printf("[%s] Failed to publish telemetry %s: %v", g.GetDeviceID(), metric, err)
				}
			}
		}
	}
}

// Gateway 特定的事件循環
func (g *GatewaySimulator) gatewayEventLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(g.config.Intervals.EventS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !g.IsRunning() {
				return
			}
			
			events := g.GenerateEvents()
			for _, event := range events {
				if err := g.GetMQTTClient().PublishEvent(event.EventType, event.Payload); err != nil && g.verbose {
					log.Printf("[%s] Failed to publish event %s: %v", g.GetDeviceID(), event.EventType, err)
				}
			}
		}
	}
}


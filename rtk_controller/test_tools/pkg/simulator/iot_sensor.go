package simulator

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"rtk_test_tools/pkg/types"
)

// IoTSensorSimulator IoT 感測器設備模擬器
type IoTSensorSimulator struct {
	*BaseSimulator
	
	// 感測器狀態
	batteryV        float64
	parentDevice    string
	rssi            int
	temperature     float64
	humidity        float64
	lightLevel      int
	pressure        float64
	lastSensorRead  time.Time
	sensorErrors    int
}

// NewIoTSensorSimulator 創建 IoT 感測器模擬器
func NewIoTSensorSimulator(config types.DeviceConfig, mqttConfig types.MQTTConfig, verbose bool) (*IoTSensorSimulator, error) {
	base, err := NewBaseSimulator(config, mqttConfig, verbose)
	if err != nil {
		return nil, err
	}
	
	// 初始化感測器狀態
	return &IoTSensorSimulator{
		BaseSimulator:  base,
		batteryV:       3.5 + rand.Float64()*0.5, // 3.5-4.0V
		parentDevice:   "ap01", // 預設連接到 AP
		rssi:          -50 - rand.Intn(30),      // -50 to -80 dBm
		temperature:   20.0 + rand.Float64()*15.0, // 20-35°C
		humidity:      40.0 + rand.Float64()*40.0,  // 40-80%
		lightLevel:    200 + rand.Intn(600),        // 200-800 lux
		pressure:      1000.0 + rand.Float64()*50.0, // 1000-1050 hPa
		lastSensorRead: time.Now(),
		sensorErrors:   0,
	}, nil
}

// Start 啟動 IoT 感測器模擬器
func (iot *IoTSensorSimulator) Start(ctx context.Context) error {
	if err := iot.BaseSimulator.Start(ctx); err != nil {
		return err
	}
	
	// 啟動 IoT 特定的循環
	go iot.iotStateLoop(ctx)
	go iot.iotTelemetryLoop(ctx)
	go iot.iotEventLoop(ctx)
	
	return nil
}

// GenerateStatePayload 生成 IoT 感測器狀態 payload
func (iot *IoTSensorSimulator) GenerateStatePayload() types.StatePayload {
	payload := iot.GenerateBaseStatePayload()
	
	// 設置 IoT 特定欄位
	payload.BatteryV = iot.batteryV
	
	// 獲取 IoT IP 配置
	if ip, ok := iot.config.Properties["ip"]; ok {
		payload.Net.IP = ip.(string)
	} else {
		payload.Net.IP = "10.0.1.105"
	}
	
	payload.Net.RSSI = iot.rssi
	payload.Net.ParentDevice = iot.parentDevice
	
	// IoT 設備通常沒有磁碟
	payload.DiskUsage = 0
	
	return payload
}

// GenerateTelemetryData 生成 IoT 感測器遙測資料
func (iot *IoTSensorSimulator) GenerateTelemetryData() map[string]types.TelemetryPayload {
	telemetry := make(map[string]types.TelemetryPayload)
	
	// 環境感測資料
	telemetry["environment"] = types.TelemetryPayload{
		"temperature": iot.temperature,
		"humidity":    iot.humidity,
		"light_level": iot.lightLevel,
		"pressure":    iot.pressure,
	}
	
	// 電池狀態
	batteryPercentage := (iot.batteryV - 3.2) / (4.2 - 3.2) * 100
	if batteryPercentage < 0 {
		batteryPercentage = 0
	} else if batteryPercentage > 100 {
		batteryPercentage = 100
	}
	
	telemetry["battery"] = types.TelemetryPayload{
		"voltage":    iot.batteryV,
		"percentage": batteryPercentage,
		"status":     iot.getBatteryStatus(),
	}
	
	// 連線品質
	telemetry["connectivity"] = types.TelemetryPayload{
		"rssi":           iot.rssi,
		"parent_device":  iot.parentDevice,
		"signal_quality": iot.getSignalQuality(),
		"packet_loss":    rand.Float64() * 0.05, // 0-5% 封包遺失
	}
	
	return telemetry
}

// GenerateEvents 生成 IoT 感測器事件
func (iot *IoTSensorSimulator) GenerateEvents() []EventData {
	var events []EventData
	
	// 低電量警告
	if iot.batteryV < 3.3 && rand.Float64() < 0.2 { // 20% 機率
		events = append(events, EventData{
			EventType: "system.battery_low",
			Payload: types.EventPayload{
				EventType: "system.battery_low",
				Severity:  "warning",
				Extra: map[string]interface{}{
					"battery_voltage":    iot.batteryV,
					"battery_percentage": (iot.batteryV - 3.2) / (4.2 - 3.2) * 100,
					"threshold":         3.3,
				},
			},
		})
	}
	
	// 訊號弱警告
	if iot.rssi < -75 && rand.Float64() < 0.1 { // 10% 機率
		events = append(events, EventData{
			EventType: "network.signal_weak",
			Payload: types.EventPayload{
				EventType: "network.signal_weak",
				Severity:  "warning",
				Extra: map[string]interface{}{
					"rssi":           iot.rssi,
					"parent_device":  iot.parentDevice,
					"signal_quality": iot.getSignalQuality(),
				},
			},
		})
	}
	
	// 感測器異常
	if iot.temperature > 40.0 || iot.humidity > 90.0 {
		events = append(events, EventData{
			EventType: "sensor.abnormal_reading",
			Payload: types.EventPayload{
				EventType: "sensor.abnormal_reading",
				Severity:  "warning",
				Extra: map[string]interface{}{
					"sensor_type":  "environmental",
					"temperature":  iot.temperature,
					"humidity":     iot.humidity,
					"light_level":  iot.lightLevel,
				},
			},
		})
	}
	
	// 斷線重連事件
	if rand.Float64() < 0.02 { // 2% 機率
		if rand.Float64() < 0.3 { // 30% 斷線
			events = append(events, EventData{
				EventType: "network.disconnected",
				Payload: types.EventPayload{
					EventType: "network.disconnected",
					Severity:  "warning",
					Extra: map[string]interface{}{
						"reason":             "signal_weak",
						"last_rssi":          iot.rssi,
						"parent_device":      iot.parentDevice,
						"reconnect_attempts": rand.Intn(5) + 1,
					},
				},
			})
		} else { // 70% 重連
			events = append(events, EventData{
				EventType: "network.reconnected",
				Payload: types.EventPayload{
					EventType: "network.reconnected",
					Severity:  "info",
					Extra: map[string]interface{}{
						"parent_device": iot.parentDevice,
						"rssi":         iot.rssi,
						"downtime":     fmt.Sprintf("%ds", rand.Intn(30)+5),
					},
				},
			})
		}
	}
	
	return events
}

// UpdateStatus 更新 IoT 感測器特定狀態
func (iot *IoTSensorSimulator) UpdateStatus() {
	iot.BaseSimulator.UpdateStatus()
	
	// 更新電池電壓（緩慢下降）
	iot.batteryV -= rand.Float64() * 0.001 // 每次更新下降 0-0.001V
	if iot.batteryV < 3.0 {
		iot.batteryV = 3.0
	}
	
	// 更新信號強度
	iot.rssi += rand.Intn(10) - 5 // -5 to +5 dBm 變化
	if iot.rssi > -30 {
		iot.rssi = -30
	} else if iot.rssi < -90 {
		iot.rssi = -90
	}
	
	// 更新感測器數值
	iot.temperature += (rand.Float64() - 0.5) * 2.0 // ±1°C
	if iot.temperature < 10.0 {
		iot.temperature = 10.0
	} else if iot.temperature > 50.0 {
		iot.temperature = 50.0
	}
	
	iot.humidity += (rand.Float64() - 0.5) * 5.0 // ±2.5%
	if iot.humidity < 20.0 {
		iot.humidity = 20.0
	} else if iot.humidity > 100.0 {
		iot.humidity = 100.0
	}
	
	iot.lightLevel += rand.Intn(100) - 50 // -50 to +50 lux
	if iot.lightLevel < 0 {
		iot.lightLevel = 0
	} else if iot.lightLevel > 1000 {
		iot.lightLevel = 1000
	}
	
	iot.pressure += (rand.Float64() - 0.5) * 2.0 // ±1 hPa
	if iot.pressure < 980.0 {
		iot.pressure = 980.0
	} else if iot.pressure > 1050.0 {
		iot.pressure = 1050.0
	}
	
	iot.lastSensorRead = time.Now()
}

// getBatteryStatus 獲取電池狀態
func (iot *IoTSensorSimulator) getBatteryStatus() string {
	if iot.batteryV > 3.8 {
		return "good"
	} else if iot.batteryV > 3.4 {
		return "medium"
	} else {
		return "low"
	}
}

// getSignalQuality 獲取信號品質
func (iot *IoTSensorSimulator) getSignalQuality() int {
	if iot.rssi > -50 {
		return 100
	} else if iot.rssi > -60 {
		return 80
	} else if iot.rssi > -70 {
		return 60
	} else if iot.rssi > -80 {
		return 40
	} else {
		return 20
	}
}

// IoT 特定的狀態循環
func (iot *IoTSensorSimulator) iotStateLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(iot.config.Intervals.StateS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !iot.IsRunning() {
				return
			}
			
			payload := iot.GenerateStatePayload()
			if err := iot.GetMQTTClient().PublishState(payload); err != nil && iot.verbose {
				log.Printf("[%s] Failed to publish state: %v", iot.GetDeviceID(), err)
			}
		}
	}
}

// IoT 特定的遙測循環
func (iot *IoTSensorSimulator) iotTelemetryLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(iot.config.Intervals.TelemetryS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !iot.IsRunning() {
				return
			}
			
			telemetryData := iot.GenerateTelemetryData()
			for metric, payload := range telemetryData {
				if err := iot.GetMQTTClient().PublishTelemetry(metric, payload); err != nil && iot.verbose {
					log.Printf("[%s] Failed to publish telemetry %s: %v", iot.GetDeviceID(), metric, err)
				}
			}
		}
	}
}

// IoT 特定的事件循環
func (iot *IoTSensorSimulator) iotEventLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(iot.config.Intervals.EventS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !iot.IsRunning() {
				return
			}
			
			events := iot.GenerateEvents()
			for _, event := range events {
				if err := iot.GetMQTTClient().PublishEvent(event.EventType, event.Payload); err != nil && iot.verbose {
					log.Printf("[%s] Failed to publish event %s: %v", iot.GetDeviceID(), event.EventType, err)
				}
			}
		}
	}
}
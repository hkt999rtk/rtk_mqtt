package simulator

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"rtk_test_tools/pkg/mqtt"
	"rtk_test_tools/pkg/types"
)

// DeviceSimulator 設備模擬器介面
type DeviceSimulator interface {
	Start(ctx context.Context) error
	Stop() error
	GetDeviceID() string
	GetDeviceType() string
	GenerateStatePayload() types.StatePayload
	GenerateTelemetryData() map[string]types.TelemetryPayload
	GenerateEvents() []EventData
	UpdateStatus() // 更新設備內部狀態
}

// EventData 事件資料
type EventData struct {
	EventType string
	Payload   types.EventPayload
}

// BaseSimulator 基礎模擬器
type BaseSimulator struct {
	config     types.DeviceConfig
	mqttClient *mqtt.Client
	running    bool
	startTime  time.Time
	mu         sync.RWMutex
	verbose    bool
	
	// 設備狀態
	health      string
	cpuUsage    float64
	memoryUsage float64
	diskUsage   float64
	tempC       float64
	errorCount  int
	restartCount int
}

// NewBaseSimulator 創建基礎模擬器
func NewBaseSimulator(config types.DeviceConfig, mqttConfig types.MQTTConfig, verbose bool) (*BaseSimulator, error) {
	client, err := mqtt.NewClient(mqttConfig, config.ID, config.Tenant, config.Site, verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT client: %v", err)
	}
	
	return &BaseSimulator{
		config:      config,
		mqttClient:  client,
		startTime:   time.Now(),
		health:      "ok",
		cpuUsage:    20.0 + rand.Float64()*30.0,  // 20-50%
		memoryUsage: 30.0 + rand.Float64()*40.0,  // 30-70%
		diskUsage:   20.0 + rand.Float64()*60.0,  // 20-80%
		tempC:       30.0 + rand.Float64()*20.0,  // 30-50°C
		verbose:     verbose,
	}, nil
}

// Start 啟動模擬器
func (b *BaseSimulator) Start(ctx context.Context) error {
	if err := b.mqttClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect MQTT: %v", err)
	}
	
	b.mu.Lock()
	b.running = true
	b.startTime = time.Now()
	b.mu.Unlock()
	
	if b.verbose {
		log.Printf("[%s] Starting device simulator", b.config.ID)
	}
	
	// 啟動消息發送 goroutines
	go b.stateLoop(ctx)
	go b.telemetryLoop(ctx)
	go b.eventLoop(ctx)
	go b.statusUpdateLoop(ctx)
	
	return nil
}

// Stop 停止模擬器
func (b *BaseSimulator) Stop() error {
	b.mu.Lock()
	b.running = false
	b.mu.Unlock()
	
	b.mqttClient.Disconnect()
	
	if b.verbose {
		log.Printf("[%s] Stopped device simulator", b.config.ID)
	}
	
	return nil
}

// GetDeviceID 獲取設備 ID
func (b *BaseSimulator) GetDeviceID() string {
	return b.config.ID
}

// GetDeviceType 獲取設備類型
func (b *BaseSimulator) GetDeviceType() string {
	return b.config.Type
}

// UpdateStatus 更新設備狀態（模擬設備行為）
func (b *BaseSimulator) UpdateStatus() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// 隨機變化 CPU 使用率
	b.cpuUsage += (rand.Float64() - 0.5) * 10.0
	if b.cpuUsage < 5.0 {
		b.cpuUsage = 5.0
	} else if b.cpuUsage > 95.0 {
		b.cpuUsage = 95.0
	}
	
	// 隨機變化記憶體使用率
	b.memoryUsage += (rand.Float64() - 0.5) * 5.0
	if b.memoryUsage < 10.0 {
		b.memoryUsage = 10.0
	} else if b.memoryUsage > 90.0 {
		b.memoryUsage = 90.0
	}
	
	// 溫度變化
	b.tempC += (rand.Float64() - 0.5) * 2.0
	if b.tempC < 25.0 {
		b.tempC = 25.0
	} else if b.tempC > 70.0 {
		b.tempC = 70.0
	}
	
	// 根據負載決定健康狀態
	if b.cpuUsage > 85.0 || b.memoryUsage > 85.0 || b.tempC > 60.0 {
		b.health = "warn"
	} else if b.cpuUsage > 95.0 || b.memoryUsage > 95.0 || b.tempC > 70.0 {
		b.health = "error"
	} else {
		b.health = "ok"
	}
}

// GenerateStatePayload 生成基礎狀態 payload
func (b *BaseSimulator) GenerateBaseStatePayload() types.StatePayload {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	uptime := int64(time.Since(b.startTime).Seconds())
	
	// 從配置中獲取設備屬性
	firmware := "1.0.0"
	if fw, ok := b.config.Properties["firmware"]; ok {
		firmware = fw.(string)
	}
	
	return types.StatePayload{
		Health:      b.health,
		Firmware:    firmware,
		UptimeS:     uptime,
		CPUUsage:    b.cpuUsage,
		MemoryUsage: b.memoryUsage,
		DiskUsage:   b.diskUsage,
		TempC:       b.tempC,
		DeviceType:  b.config.Type,
		Net: types.NetworkInfo{
			IP: "10.0.1.100", // 默認值，子類可覆蓋
		},
		Diagnosis: types.DiagnosisInfo{
			LastError:    nil,
			ErrorCount:   b.errorCount,
			RestartCount: b.restartCount,
		},
	}
}

// 狀態消息循環
func (b *BaseSimulator) stateLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(b.config.Intervals.StateS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.mu.RLock()
			running := b.running
			b.mu.RUnlock()
			
			if !running {
				return
			}
			
			// 這裡需要由具體的模擬器實現
			// 基類只能提供框架
		}
	}
}

// 遙測消息循環
func (b *BaseSimulator) telemetryLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(b.config.Intervals.TelemetryS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.mu.RLock()
			running := b.running
			b.mu.RUnlock()
			
			if !running {
				return
			}
			
			// 這裡需要由具體的模擬器實現
		}
	}
}

// 事件消息循環
func (b *BaseSimulator) eventLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(b.config.Intervals.EventS) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.mu.RLock()
			running := b.running
			b.mu.RUnlock()
			
			if !running {
				return
			}
			
			// 這裡需要由具體的模擬器實現
		}
	}
}

// 狀態更新循環
func (b *BaseSimulator) statusUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // 每5秒更新一次內部狀態
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.mu.RLock()
			running := b.running
			b.mu.RUnlock()
			
			if !running {
				return
			}
			
			b.UpdateStatus()
		}
	}
}

// IsRunning 檢查是否正在運行
func (b *BaseSimulator) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// GetMQTTClient 獲取 MQTT 客戶端（供子類使用）
func (b *BaseSimulator) GetMQTTClient() *mqtt.Client {
	return b.mqttClient
}

// GenerateRandomMAC 生成隨機 MAC 地址（共用函數）
func GenerateRandomMAC() string {
	mac := make([]byte, 6)
	rand.Read(mac)
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

// GenerateRandomIP 生成隨機 IP 地址（共用函數）
func GenerateRandomIP() string {
	return fmt.Sprintf("10.0.1.%d", 100+rand.Intn(155))
}
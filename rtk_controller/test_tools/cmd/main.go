package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"rtk_test_tools/pkg/config"
	"rtk_test_tools/pkg/simulator"
	"rtk_test_tools/pkg/types"
)

var (
	configFile    = flag.String("config", "configs/test_config.yaml", "Path to test configuration file")
	generateConfig = flag.Bool("generate-config", false, "Generate example configuration file")
	verbose       = flag.Bool("verbose", false, "Enable verbose logging")
	duration      = flag.Int("duration", 0, "Test duration in seconds (overrides config)")
)

func main() {
	flag.Parse()
	
	if *generateConfig {
		if err := config.SaveExampleConfig(*configFile); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		fmt.Printf("Generated example configuration: %s\n", *configFile)
		return
	}
	
	// 載入配置
	testConfig, err := config.LoadTestConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// 命令列參數覆蓋配置
	if *verbose {
		testConfig.Test.Verbose = true
	}
	
	if *duration > 0 {
		testConfig.Test.DurationS = *duration
	}
	
	// 設置日誌級別
	if testConfig.Test.Verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
	
	fmt.Printf("🚀 Starting RTK MQTT Test Suite\n")
	fmt.Printf("📝 Configuration: %s\n", *configFile)
	fmt.Printf("🏃 Test Duration: %d seconds\n", testConfig.Test.DurationS)
	fmt.Printf("📡 MQTT Broker: %s:%d\n", testConfig.MQTT.Broker, testConfig.MQTT.Port)
	fmt.Printf("🔧 Devices: %d configured\n", len(testConfig.Devices))
	
	// 統計設備類型
	deviceTypes := make(map[string]int)
	enabledDevices := 0
	for _, device := range testConfig.Devices {
		deviceTypes[device.Type]++
		if device.Enabled {
			enabledDevices++
		}
	}
	
	fmt.Printf("   └─ Enabled: %d\n", enabledDevices)
	for deviceType, count := range deviceTypes {
		fmt.Printf("   └─ %s: %d\n", deviceType, count)
	}
	
	if enabledDevices == 0 {
		log.Fatal("No devices enabled in configuration")
	}
	
	// 創建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// 設置信號處理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	
	// 創建並啟動設備模擬器
	simulators, err := createSimulators(*testConfig)
	if err != nil {
		log.Fatalf("Failed to create simulators: %v", err)
	}
	
	fmt.Printf("\n⚡ Starting %d device simulators...\n", len(simulators))
	
	// 啟動所有模擬器
	var wg sync.WaitGroup
	for i, sim := range simulators {
		wg.Add(1)
		go func(index int, simulator simulator.DeviceSimulator) {
			defer wg.Done()
			
			if err := simulator.Start(ctx); err != nil {
				log.Printf("❌ Failed to start simulator %s: %v", simulator.GetDeviceID(), err)
				return
			}
			
			if testConfig.Test.Verbose {
				log.Printf("✅ [%d] Started %s simulator: %s", 
					index+1, simulator.GetDeviceType(), simulator.GetDeviceID())
			}
			
			// 等待停止信號
			<-ctx.Done()
			
			if err := simulator.Stop(); err != nil {
				log.Printf("❌ Error stopping simulator %s: %v", simulator.GetDeviceID(), err)
			} else if testConfig.Test.Verbose {
				log.Printf("🛑 Stopped simulator: %s", simulator.GetDeviceID())
			}
		}(i, sim)
	}
	
	fmt.Printf("✅ All simulators started successfully\n\n")
	
	// 開始測試
	testStartTime := time.Now()
	fmt.Printf("🔍 Test started at: %s\n", testStartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("⏱️  Test will run for %d seconds\n", testConfig.Test.DurationS)
	fmt.Println("Press Ctrl+C to stop early...")
	fmt.Println()
	
	// 統計協程
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				elapsed := time.Since(testStartTime)
				remaining := time.Duration(testConfig.Test.DurationS)*time.Second - elapsed
				
				if remaining <= 0 {
					fmt.Printf("⏰ Test completed after %v\n", elapsed)
					cancel()
					return
				}
				
				fmt.Printf("📊 Test running for %v, %v remaining\n", 
					elapsed.Truncate(time.Second), remaining.Truncate(time.Second))
			}
		}
	}()
	
	// 等待測試完成或中斷信號
	select {
	case <-time.After(time.Duration(testConfig.Test.DurationS) * time.Second):
		fmt.Printf("\n⏰ Test completed after %d seconds\n", testConfig.Test.DurationS)
	case <-sigCh:
		fmt.Printf("\n🛑 Test interrupted by user\n")
	}
	
	// 停止所有模擬器
	fmt.Printf("🔄 Stopping simulators...\n")
	cancel()
	
	// 等待所有模擬器停止
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		fmt.Printf("✅ All simulators stopped cleanly\n")
	case <-time.After(10 * time.Second):
		fmt.Printf("⚠️  Some simulators didn't stop within 10 seconds\n")
	}
	
	// 測試總結
	totalDuration := time.Since(testStartTime)
	fmt.Printf("\n📈 Test Summary:\n")
	fmt.Printf("   Total Duration: %v\n", totalDuration.Truncate(time.Second))
	fmt.Printf("   Simulators: %d\n", len(simulators))
	fmt.Printf("   Average uptime: %v per simulator\n", totalDuration.Truncate(time.Second))
	
	// 計算預估消息數量
	totalMessages := estimateMessageCount(*testConfig, totalDuration)
	fmt.Printf("   Estimated messages sent: ~%d\n", totalMessages)
	
	fmt.Printf("\n🎉 Test completed successfully!\n")
}

// createSimulators 創建所有設備模擬器
func createSimulators(config types.TestConfig) ([]simulator.DeviceSimulator, error) {
	var simulators []simulator.DeviceSimulator
	
	for _, deviceConfig := range config.Devices {
		if !deviceConfig.Enabled {
			continue
		}
		
		sim, err := simulator.CreateSimulator(deviceConfig, config.MQTT, config.Test.Verbose)
		if err != nil {
			return nil, fmt.Errorf("failed to create simulator for device %s: %v", 
				deviceConfig.ID, err)
		}
		
		simulators = append(simulators, sim)
	}
	
	return simulators, nil
}

// estimateMessageCount 估算測試期間發送的消息總數
func estimateMessageCount(config types.TestConfig, duration time.Duration) int {
	total := 0
	durationSeconds := int(duration.Seconds())
	
	for _, device := range config.Devices {
		if !device.Enabled {
			continue
		}
		
		// 狀態消息
		stateMessages := durationSeconds / device.Intervals.StateS
		
		// 遙測消息（每種設備類型有不同數量的遙測指標）
		telemetryCount := getTelemetryMetricCount(device.Type)
		telemetryMessages := (durationSeconds / device.Intervals.TelemetryS) * telemetryCount
		
		// 事件消息（估算，基於事件發生機率）
		eventMessages := (durationSeconds / device.Intervals.EventS) / 4 // 假設 25% 機率有事件
		
		total += stateMessages + telemetryMessages + eventMessages
	}
	
	return total
}

// getTelemetryMetricCount 獲取每種設備類型的遙測指標數量
func getTelemetryMetricCount(deviceType string) int {
	switch deviceType {
	case "gateway":
		return 2 // network_stats, wifi_stats
	case "iot_sensor":
		return 3 // environment, battery, connectivity
	case "nic":
		return 3 // interface_stats, network_performance, driver_info
	case "switch":
		return 4 // port_stats, switching_stats, vlan_stats, system_stats
	default:
		return 1
	}
}
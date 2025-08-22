package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"rtk_wrapper/internal/config"
	"rtk_wrapper/internal/monitoring"
	"rtk_wrapper/internal/mqtt"
	"rtk_wrapper/internal/registry"
	"rtk_wrapper/pkg/types"
	"rtk_wrapper/wrappers/example"
	"rtk_wrapper/wrappers/homeassistant"
)

var (
	// 版本資訊（由建置時注入）
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"

	// 命令列參數
	configFile string
	logLevel   string
	daemon     bool
)

// DefaultMessageHandler 預設訊息處理器
type DefaultMessageHandler struct{}

func (h *DefaultMessageHandler) HandleMessage(msg *types.WrapperMessage) error {
	log.Printf("Received message: topic=%s, direction=%d, source=%d",
		msg.MQTTInfo.Topic, msg.Direction, msg.Source)
	return nil
}

// MonitoredMessageHandler 帶監控的訊息處理器
type MonitoredMessageHandler struct {
	monitor *monitoring.Monitor
}

func (h *MonitoredMessageHandler) HandleMessage(msg *types.WrapperMessage) error {
	start := time.Now()

	// 處理訊息
	err := h.handleMessage(msg)

	processingTime := time.Since(start)

	// 記錄到監控系統
	if h.monitor != nil {
		wrapperName := "unknown"
		if meta, ok := msg.Meta["wrapper_name"]; ok {
			if name, ok := meta.(string); ok {
				wrapperName = name
			}
		}

		h.monitor.RecordMessage(
			wrapperName,
			fmt.Sprintf("%d", int(msg.Direction)),
			msg.MQTTInfo.Topic,
			len(msg.MQTTInfo.Payload),
			processingTime,
			err == nil,
			err,
		)
	}

	return err
}

func (h *MonitoredMessageHandler) handleMessage(msg *types.WrapperMessage) error {
	// 獲取結構化日誌記錄器
	logger := h.monitor.GetLogger()

	wrapperName := "unknown"
	if meta, ok := msg.Meta["wrapper_name"]; ok {
		if name, ok := meta.(string); ok {
			wrapperName = name
		}
	}

	if logger != nil {
		ctx := logger.NewLogContext()
		ctx.WrapperName = wrapperName
		ctx.Direction = msg.Direction
		ctx.Topic = msg.MQTTInfo.Topic
		ctx.PayloadSize = int64(len(msg.MQTTInfo.Payload))

		messageLogger := logger.NewMessageLogger(ctx)
		messageLogger.LogMessageReceived(msg.MQTTInfo.Topic, len(msg.MQTTInfo.Payload))

		// 釋放上下文
		logger.ReleaseLogContext(ctx)
	} else {
		log.Printf("Received message: topic=%s, direction=%d, wrapper=%s, payload_size=%d",
			msg.MQTTInfo.Topic, msg.Direction, wrapperName, len(msg.MQTTInfo.Payload))
	}

	return nil
}

var rootCmd = &cobra.Command{
	Use:   "rtk_wrapper",
	Short: "RTK MQTT Wrapper - Device message format converter",
	Long: `RTK MQTT Wrapper is a middleware layer that converts different MQTT device 
message formats to RTK standard format, enabling seamless integration between
various IoT devices and RTK Controller.`,
	Version: fmt.Sprintf("%s (built: %s, commit: %s)", Version, BuildTime, GitCommit),
	Run:     runWrapper,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("RTK MQTT Wrapper\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
	},
}

func init() {
	// 添加 flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "configs/wrapper.yaml", "Configuration file path")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Log level (trace,debug,info,warn,error)")
	rootCmd.PersistentFlags().BoolVar(&daemon, "daemon", false, "Run as daemon")

	// 添加子命令
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Command execution failed: %v", err)
	}
}

func runWrapper(cmd *cobra.Command, args []string) {
	log.Printf("Starting RTK MQTT Wrapper %s", Version)

	// 載入配置
	cfg, err := config.Load(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 覆寫日志級別（如果指定）
	if logLevel != "" {
		cfg.Wrapper.Logging.Level = logLevel
	}

	log.Printf("Loaded config from: %s", configFile)

	// 創建註冊表
	registry := registry.NewRegistry(
		cfg.Wrapper.Registry.AutoDiscovery,
		int(cfg.Wrapper.Registry.DiscoveryTimeout.Seconds()),
	)

	// 初始化監控系統
	monitorConfig := monitoring.GetDefaultMonitorConfig()
	// 使用配置檔案中的監控設定
	monitorConfig.Logging = monitoring.LogConfig{
		Level:      monitoring.LogLevel(cfg.Wrapper.Logging.Level),
		Format:     monitoring.LogFormat(cfg.Wrapper.Logging.Format),
		Output:     monitoring.LogOutput(cfg.Wrapper.Logging.Output),
		FilePath:   "logs/wrapper.log",
		MaxSize:    100,
		MaxAge:     7,
		MaxBackups: 3,
		Compress:   true,
	}

	if cfg.Wrapper.Monitoring.Enabled {
		monitorConfig.Health.Port = cfg.Wrapper.Monitoring.HealthCheckPort
		monitorConfig.Metrics.Port = cfg.Wrapper.Monitoring.MetricsPort
	}

	monitor, err := monitoring.NewMonitor(monitorConfig, registry)
	if err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}

	// 設置信號處理
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	// 啟動監控系統
	if err := monitor.Start(ctx); err != nil {
		log.Fatalf("Failed to start monitor: %v", err)
	}

	// 從監控系統獲取結構化日誌記錄器
	var logger *monitoring.StructuredLogger
	if monitor.GetLogger() != nil {
		logger = monitor.GetLogger()
		logger.Info("RTK MQTT Wrapper starting", map[string]interface{}{
			"version":     Version,
			"build_time":  BuildTime,
			"git_commit":  GitCommit,
			"config_file": configFile,
		})
	}

	// 註冊範例 wrapper
	exampleConfigFile := "configs/wrappers/example.yaml"
	if err := example.RegisterExampleWrapper(registry, exampleConfigFile); err != nil {
		if logger != nil {
			logger.Error("Failed to register example wrapper", err, map[string]interface{}{
				"config_file": exampleConfigFile,
			})
		} else {
			log.Printf("Failed to register example wrapper: %v (continuing without it)", err)
		}
	} else {
		if logger != nil {
			logger.Info("Successfully registered example wrapper", map[string]interface{}{
				"config_file": exampleConfigFile,
			})
		} else {
			log.Printf("Successfully registered example wrapper")
		}
		monitor.RecordWrapperRegistration("example", true)
	}

	// 註冊 Home Assistant wrapper
	haConfigFile := "configs/wrappers/homeassistant.yaml"
	if err := homeassistant.RegisterHomeAssistantWrapper(registry, haConfigFile); err != nil {
		if logger != nil {
			logger.Error("Failed to register Home Assistant wrapper", err, map[string]interface{}{
				"config_file": haConfigFile,
			})
		} else {
			log.Printf("Failed to register Home Assistant wrapper: %v (continuing without it)", err)
		}
		monitor.RecordWrapperRegistration("homeassistant", false)
	} else {
		if logger != nil {
			logger.Info("Successfully registered Home Assistant wrapper", map[string]interface{}{
				"config_file": haConfigFile,
			})
		} else {
			log.Printf("Successfully registered Home Assistant wrapper")
		}
		monitor.RecordWrapperRegistration("homeassistant", true)
	}

	// TODO: 註冊其他 wrapper (Tasmota, Xiaomi, 等等)

	stats := registry.Stats()
	if logger != nil {
		logger.Info("Registry initialized", map[string]interface{}{
			"total_wrappers":  stats.TotalWrappers,
			"uplink_routes":   stats.UplinkRoutes,
			"downlink_routes": stats.DownlinkRoutes,
		})
	} else {
		log.Printf("Registry initialized: %d wrappers, %d uplink routes, %d downlink routes",
			stats.TotalWrappers, stats.UplinkRoutes, stats.DownlinkRoutes)
	}

	// 創建訊息處理器（集成監控）
	msgHandler := &MonitoredMessageHandler{
		monitor: monitor,
	}

	// 創建 MQTT 客戶端
	mqttClient, err := mqtt.NewClient(
		&cfg.Wrapper.MQTT,
		&cfg.Wrapper.RTK,
		registry,
		msgHandler,
	)
	if err != nil {
		if logger != nil {
			logger.Fatal("Failed to create MQTT client", err)
		} else {
			log.Fatalf("Failed to create MQTT client: %v", err)
		}
	}

	// 啟動 MQTT 客戶端
	if err := mqttClient.Start(); err != nil {
		if logger != nil {
			logger.Fatal("Failed to start MQTT client", err)
		} else {
			log.Fatalf("Failed to start MQTT client: %v", err)
		}
	}

	if logger != nil {
		logger.Info("RTK MQTT Wrapper started successfully", map[string]interface{}{
			"mqtt_broker":     cfg.Wrapper.MQTT.Broker,
			"client_id":       cfg.Wrapper.MQTT.ClientID,
			"default_tenant":  cfg.Wrapper.RTK.DefaultTenant,
			"default_site":    cfg.Wrapper.RTK.DefaultSite,
			"monitoring_port": cfg.Wrapper.Monitoring.MetricsPort,
			"health_port":     cfg.Wrapper.Monitoring.HealthCheckPort,
		})
	} else {
		log.Printf("RTK MQTT Wrapper started successfully")
		log.Printf("MQTT Broker: %s", cfg.Wrapper.MQTT.Broker)
		log.Printf("Client ID: %s", cfg.Wrapper.MQTT.ClientID)
		log.Printf("Default Tenant/Site: %s/%s", cfg.Wrapper.RTK.DefaultTenant, cfg.Wrapper.RTK.DefaultSite)
	}

	// 等待信號
	<-ctx.Done()

	// 關閉
	if logger != nil {
		logger.Info("Shutting down RTK MQTT Wrapper")
	} else {
		log.Printf("Shutting down...")
	}

	// 停止 MQTT 客戶端
	if err := mqttClient.Stop(); err != nil {
		if logger != nil {
			logger.Error("Error stopping MQTT client", err)
		} else {
			log.Printf("Error stopping MQTT client: %v", err)
		}
	}

	// 停止監控系統
	if err := monitor.Stop(); err != nil {
		log.Printf("Error stopping monitor: %v", err)
	}

	if logger != nil {
		logger.Info("RTK MQTT Wrapper stopped successfully")
	} else {
		log.Printf("RTK MQTT Wrapper stopped")
	}
}

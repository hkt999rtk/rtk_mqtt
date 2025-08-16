package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"rtk_controller/internal/config"
	"rtk_controller/internal/mqtt"
	"rtk_controller/internal/storage"
	"rtk_controller/internal/device"
	"rtk_controller/internal/command"
	"rtk_controller/internal/diagnosis"
	"rtk_controller/internal/cli"
	"rtk_controller/internal/schema"
	"rtk_controller/internal/logging"

	log "github.com/sirupsen/logrus"
)

var (
	Version = "dev"
	BuildTime = "unknown"
)

func main() {
	var (
		configPath = flag.String("config", "configs/controller.yaml", "Path to configuration file")
		cliMode    = flag.Bool("cli", false, "Run in interactive CLI mode")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		fmt.Printf("RTK Controller %s (built at %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	// Initialize configuration manager
	configManager, err := config.NewManager(*configPath)
	if err != nil {
		log.Fatalf("Failed to create config manager: %v", err)
	}
	defer configManager.Stop()

	// Get initial configuration
	cfg := configManager.GetConfig()

	// Setup enhanced logging
	appLogger, auditLogger, perfLogger, err := setupEnhancedLogging(cfg.Logging)
	if err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}
	defer func() {
		appLogger.Close()
		auditLogger.Close()
		perfLogger.Close()
	}()

	// If CLI mode is specified, run interactive CLI
	if *cliMode {
		// Initialize storage for CLI
		storage, err := storage.NewBuntDB(cfg.Storage.Path)
		if err != nil {
			log.Fatalf("Failed to initialize storage: %v", err)
		}
		defer storage.Close()

		// Initialize MQTT client for CLI
		mqttClient, err := mqtt.NewClient(cfg.MQTT, storage)
		if err != nil {
			log.Fatalf("Failed to create MQTT client: %v", err)
		}

		// Initialize core services for CLI
		deviceManager := device.NewManager(storage)
		commandManager := command.NewManager(mqttClient, storage)
		diagnosisManager := diagnosis.NewManager(cfg.Diagnosis, storage)

		// Create and start interactive CLI
		interactiveCLI := cli.NewInteractiveCLI(cfg, mqttClient, storage, deviceManager, commandManager, diagnosisManager)
		interactiveCLI.Start()
		return
	}

	log.Info("Starting RTK Controller...")
	printBanner()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize storage
	storage, err := storage.NewBuntDB(cfg.Storage.Path)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Initialize schema manager
	schemaConfig := schema.Config{
		Enabled:             cfg.Schema.Enabled,
		SchemaFiles:         cfg.Schema.SchemaFiles,
		StrictValidation:    cfg.Schema.StrictValidation,
		LogValidationErrors: cfg.Schema.LogValidationErrors,
		CacheResults:        cfg.Schema.CacheResults,
		CacheSize:           cfg.Schema.CacheSize,
		StoreResults:        cfg.Schema.StoreResults,
	}
	schemaManager, err := schema.NewManager(schemaConfig, storage)
	if err != nil {
		log.Fatalf("Failed to create schema manager: %v", err)
	}
	
	if err := schemaManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize schema manager: %v", err)
	}

	// Initialize MQTT client
	mqttClient, err := mqtt.NewClient(cfg.MQTT, storage)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}

	// Initialize core services
	deviceManager := device.NewManager(storage)
	eventProcessor := device.NewEventProcessor(storage)
	commandManager := command.NewManager(mqttClient, storage)
	diagnosisManager := diagnosis.NewManager(cfg.Diagnosis, storage)
	
	// Connect MQTT client to device management and schema validation
	mqttClient.SetDeviceManager(deviceManager)
	mqttClient.SetEventProcessor(eventProcessor)
	
	// Create schema validator adapter for MQTT client
	schemaAdapter := mqtt.NewSchemaValidatorAdapter(schemaManager)
	mqttClient.SetSchemaValidator(schemaAdapter)

	// Web Console and API server removed - using CLI only

	// Start services
	log.Info("Connecting to MQTT broker...")
	if err := mqttClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to MQTT: %v", err)
	}

	log.Info("Starting device manager...")
	if err := deviceManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start device manager: %v", err)
	}
	
	log.Info("Starting event processor...")
	if err := eventProcessor.Start(ctx); err != nil {
		log.Fatalf("Failed to start event processor: %v", err)
	}

	log.Info("Starting command manager...")
	if err := commandManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start command manager: %v", err)
	}

	log.Info("Starting diagnosis manager...")
	if err := diagnosisManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start diagnosis manager: %v", err)
	}

	log.WithFields(log.Fields{
		"mqtt_broker": cfg.MQTT.Broker,
		"mode":        "daemon",
	}).Info("RTK Controller started successfully")

	// Wait for shutdown signal
	waitForShutdown()

	log.Info("Shutting down RTK Controller...")
	cancel()

	// Stop services gracefully
	diagnosisManager.Stop()
	commandManager.Stop()
	eventProcessor.Stop()
	deviceManager.Stop()
	mqttClient.Disconnect()

	log.Info("RTK Controller stopped gracefully")
}

func setupEnhancedLogging(cfg config.LoggingConfig) (*logging.Logger, *logging.AuditLogger, *logging.PerformanceLogger, error) {
	// Setup main application logger
	appLogger, err := logging.NewLogger(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create application logger: %w", err)
	}

	// Replace global logrus logger with our structured logger
	log.SetLevel(appLogger.GetLevel())
	log.SetFormatter(appLogger.Formatter)
	log.SetOutput(appLogger.Out)

	var auditLogger *logging.AuditLogger
	var perfLogger *logging.PerformanceLogger

	// Setup audit logger if enabled
	if cfg.Audit {
		auditLogger, err = logging.NewAuditLogger(cfg)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create audit logger: %w", err)
		}
	} else {
		// Create a no-op audit logger
		auditLogger, _ = logging.NewAuditLogger(config.LoggingConfig{
			Level:  "fatal", // Effectively disable
			Format: cfg.Format,
		})
	}

	// Setup performance logger if enabled
	if cfg.Performance {
		perfLogger, err = logging.NewPerformanceLogger(cfg)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create performance logger: %w", err)
		}
	} else {
		// Create a no-op performance logger
		perfLogger, _ = logging.NewPerformanceLogger(config.LoggingConfig{
			Level:  "fatal", // Effectively disable
			Format: cfg.Format,
		})
	}

	return appLogger, auditLogger, perfLogger, nil
}

func setupLogging(level string) {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})

	logLevel, err := log.ParseLevel(level)
	if err != nil {
		log.Warnf("Invalid log level '%s', using INFO", level)
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)
}

func printBanner() {
	banner := `
╔════════════════════════════════════════════════════════════════╗
║                        RTK Controller                          ║
║                                                                ║
║    MQTT Diagnostic Communication Controller for IoT Devices   ║
║    Copyright (c) 2024 Realtek Semiconductor Corp.             ║
╚════════════════════════════════════════════════════════════════╝`

	fmt.Println(banner)
}

func waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rtk/mqtt-framework/pkg/codec"
	"github.com/rtk/mqtt-framework/pkg/config"
	"github.com/rtk/mqtt-framework/pkg/device"
	"github.com/rtk/mqtt-framework/pkg/mqtt"
	"github.com/rtk/mqtt-framework/pkg/topic"
)

// FrameworkDemo demonstrates the complete RTK MQTT framework
func main() {
	log.Println("RTK MQTT Framework Demo")
	log.Println("========================")

	// Load configuration
	cfg, err := config.LoadFromFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded for device: %s", cfg.Device.DeviceID)

	// Create MQTT client
	mqttClient, err := mqtt.NewClient(&cfg.MQTT)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}

	// Create codec and topic builder
	codec := codec.NewCodec()
	codec.SetDeviceInfo(cfg.Device.Tenant, cfg.Device.Site, cfg.Device.DeviceID)

	topicBuilder := topic.NewBuilder(cfg.Device.Tenant, cfg.Device.Site, cfg.Device.DeviceID)
	topicParser := topic.NewParser()

	log.Printf("MQTT client created with codec for device: %s/%s/%s", 
		cfg.Device.Tenant, cfg.Device.Site, cfg.Device.DeviceID)

	// Create device manager
	managerConfig := &device.ManagerConfig{
		EventWorkers:        4,
		EventQueueSize:      1000,
		HealthCheckInterval: 30 * time.Second,
		MetricsInterval:     60 * time.Second,
		PluginTimeout:       30 * time.Second,
	}

	manager := device.NewManager(managerConfig)

	// Register event handler
	manager.AddEventHandler(&DemoEventHandler{
		mqttClient: mqttClient,
		codec:      codec,
	})

	// Start manager
	if err := manager.Start(); err != nil {
		log.Fatalf("Failed to start device manager: %v", err)
	}

	log.Println("Device manager started")

	// Connect to MQTT broker
	ctx := context.Background()
	if err := mqttClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer mqttClient.Disconnect()

	log.Printf("Connected to MQTT broker: %s:%d", cfg.MQTT.BrokerHost, cfg.MQTT.BrokerPort)

	// Subscribe to various topics for demonstration
	subscribeToTopics(mqttClient, topicBuilder, topicParser, codec)

	// Create and start demo plugins
	startDemoPlugins(manager, mqttClient, codec)

	// Start demo message publisher
	go publishDemoMessages(mqttClient, codec, topicBuilder)

	// Start command sender
	go sendDemoCommands(mqttClient, codec, topicBuilder)

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Framework demo is running. Press Ctrl+C to stop.")
	log.Println("Monitoring MQTT topics and processing messages...")

	<-sigChan

	log.Println("Shutting down framework demo...")

	// Stop manager
	if err := manager.Stop(); err != nil {
		log.Printf("Error stopping manager: %v", err)
	}

	log.Println("Framework demo stopped")
}

// DemoEventHandler handles events for demonstration
type DemoEventHandler struct {
	mqttClient mqtt.Client
	codec      *codec.Codec
}

// HandleEvent processes events
func (h *DemoEventHandler) HandleEvent(ctx context.Context, event *device.Event) error {
	log.Printf("Event received: %s - %s", event.Type, event.Message)

	// Encode and publish event
	msg, err := h.codec.EncodeEvent(ctx, event)
	if err != nil {
		log.Printf("Failed to encode event: %v", err)
		return err
	}

	mqttMsg := &mqtt.Message{
		Topic:     msg.Topic,
		Payload:   msg.Payload,
		QoS:       mqtt.QoS(msg.QoS),
		Retained:  msg.Retained,
		Timestamp: time.Now(),
	}

	return h.mqttClient.PublishMessage(ctx, mqttMsg)
}

// subscribeToTopics subscribes to various topics for demonstration
func subscribeToTopics(mqttClient mqtt.Client, topicBuilder *topic.Builder, topicParser *topic.Parser, codec *codec.Codec) {
	ctx := context.Background()

	// Subscribe to all events
	eventTopic := topicBuilder.Event("+")
	err := mqttClient.Subscribe(ctx, eventTopic, func(ctx context.Context, msg *mqtt.Message) error {
		log.Printf("Event received on topic: %s", msg.Topic)

		rtkMsg, err := codec.Decode(ctx, msg.Topic, msg.Payload)
		if err != nil {
			log.Printf("Failed to decode event message: %v", err)
			return err
		}

		event, err := codec.DecodeEvent(ctx, rtkMsg)
		if err != nil {
			log.Printf("Failed to decode event: %v", err)
			return err
		}

		log.Printf("Event: %s [%s] - %s", event.Type, event.Level, event.Message)
		return nil
	}, nil)

	if err != nil {
		log.Printf("Failed to subscribe to events: %v", err)
	} else {
		log.Printf("Subscribed to events: %s", eventTopic)
	}

	// Subscribe to telemetry
	telemetryTopic := topicBuilder.Telemetry("+")
	err = mqttClient.Subscribe(ctx, telemetryTopic, func(ctx context.Context, msg *mqtt.Message) error {
		log.Printf("Telemetry received on topic: %s", msg.Topic)

		rtkMsg, err := codec.Decode(ctx, msg.Topic, msg.Payload)
		if err != nil {
			log.Printf("Failed to decode telemetry message: %v", err)
			return err
		}

		telemetry, err := codec.DecodeTelemetry(ctx, rtkMsg)
		if err != nil {
			log.Printf("Failed to decode telemetry: %v", err)
			return err
		}

		log.Printf("Telemetry: %s = %v %s", telemetry.Metric, telemetry.Value, telemetry.Unit)
		return nil
	}, nil)

	if err != nil {
		log.Printf("Failed to subscribe to telemetry: %v", err)
	} else {
		log.Printf("Subscribed to telemetry: %s", telemetryTopic)
	}

	// Subscribe to state changes
	stateTopic := topicBuilder.State()
	err = mqttClient.Subscribe(ctx, stateTopic, func(ctx context.Context, msg *mqtt.Message) error {
		log.Printf("State received on topic: %s", msg.Topic)

		rtkMsg, err := codec.Decode(ctx, msg.Topic, msg.Payload)
		if err != nil {
			log.Printf("Failed to decode state message: %v", err)
			return err
		}

		state, err := codec.DecodeState(ctx, rtkMsg)
		if err != nil {
			log.Printf("Failed to decode state: %v", err)
			return err
		}

		log.Printf("State: %s [%s] - Uptime: %s", state.Status, state.Health, state.Uptime)
		return nil
	}, nil)

	if err != nil {
		log.Printf("Failed to subscribe to state: %v", err)
	} else {
		log.Printf("Subscribed to state: %s", stateTopic)
	}

	// Subscribe to command responses
	responsesTopic := topicBuilder.CommandResponse()
	err = mqttClient.Subscribe(ctx, responsesTopic, func(ctx context.Context, msg *mqtt.Message) error {
		log.Printf("Command response received on topic: %s", msg.Topic)

		rtkMsg, err := codec.Decode(ctx, msg.Topic, msg.Payload)
		if err != nil {
			log.Printf("Failed to decode response message: %v", err)
			return err
		}

		// Decode as generic JSON for display
		var response map[string]interface{}
		if err := json.Unmarshal(rtkMsg.Payload, &response); err != nil {
			log.Printf("Failed to decode command response: %v", err)
			return err
		}

		if cmdID, ok := response["command_id"].(string); ok {
			if status, ok := response["status"].(string); ok {
				log.Printf("Command Response: %s [%s]", cmdID, status)
				if status == "error" {
					if errMsg, ok := response["error"].(string); ok {
						log.Printf("  Error: %s", errMsg)
					}
				} else if result, ok := response["result"]; ok {
					log.Printf("  Result: %v", result)
				}
			}
		}

		return nil
	}, nil)

	if err != nil {
		log.Printf("Failed to subscribe to command responses: %v", err)
	} else {
		log.Printf("Subscribed to command responses: %s", responsesTopic)
	}
}

// startDemoPlugins creates and starts demo plugins
func startDemoPlugins(manager *device.Manager, mqttClient mqtt.Client, codec *codec.Codec) {
	// Create a simple demo plugin
	demoPlugin := NewDemoPlugin()
	demoPlugin.SetMQTTClient(mqttClient)
	demoPlugin.SetCodec(codec)

	// Plugin configuration
	pluginConfig := &device.PluginConfig{
		Type:    "demo",
		Name:    "demo_device",
		Enabled: true,
		Config: json.RawMessage(`{
			"update_interval": "10s",
			"demo_mode": true
		}`),
	}

	// Create and start plugin
	if err := manager.CreatePlugin("demo_device", pluginConfig); err != nil {
		log.Printf("Failed to create demo plugin: %v", err)
		return
	}

	if err := manager.StartPlugin("demo_device"); err != nil {
		log.Printf("Failed to start demo plugin: %v", err)
		return
	}

	log.Println("Demo plugin started")
}

// publishDemoMessages publishes various demo messages
func publishDemoMessages(mqttClient mqtt.Client, codec *codec.Codec, topicBuilder *topic.Builder) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			// Publish demo state
			state := &device.State{
				Status:    "online",
				Health:    "healthy",
				LastSeen:  time.Now(),
				Uptime:    time.Since(time.Now().Add(-time.Hour)),
				Timestamp: time.Now(),
				Properties: map[string]interface{}{
					"demo_mode": true,
					"version":   "1.0.0",
				},
			}

			if stateMsg, err := codec.EncodeState(ctx, state); err == nil {
				mqttMsg := &mqtt.Message{
					Topic:     stateMsg.Topic,
					Payload:   stateMsg.Payload,
					QoS:       mqtt.QoS(stateMsg.QoS),
					Retained:  stateMsg.Retained,
					Timestamp: time.Now(),
				}
				mqttClient.PublishMessage(ctx, mqttMsg)
				log.Println("Published demo state")
			}

			// Publish demo telemetry
			telemetry := &device.TelemetryData{
				Metric:    "cpu_usage",
				Value:     45.2,
				Unit:      "percent",
				Timestamp: time.Now(),
			}

			if telemetryMsg, err := codec.EncodeTelemetry(ctx, telemetry); err == nil {
				mqttMsg := &mqtt.Message{
					Topic:     telemetryMsg.Topic,
					Payload:   telemetryMsg.Payload,
					QoS:       mqtt.QoS(telemetryMsg.QoS),
					Retained:  telemetryMsg.Retained,
					Timestamp: time.Now(),
				}
				mqttClient.PublishMessage(ctx, mqttMsg)
				log.Println("Published demo telemetry")
			}

			// Publish demo event
			event := &device.Event{
				ID:        "demo_event_001",
				Type:      "system.startup",
				Level:     "info",
				Message:   "Demo system started successfully",
				Source:    "demo_framework",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"startup_time": time.Now().Format(time.RFC3339),
					"demo_mode":    true,
				},
			}

			if eventMsg, err := codec.EncodeEvent(ctx, event); err == nil {
				mqttMsg := &mqtt.Message{
					Topic:     eventMsg.Topic,
					Payload:   eventMsg.Payload,
					QoS:       mqtt.QoS(eventMsg.QoS),
					Retained:  eventMsg.Retained,
					Timestamp: time.Now(),
				}
				mqttClient.PublishMessage(ctx, mqttMsg)
				log.Println("Published demo event")
			}

			cancel()
		}
	}
}

// sendDemoCommands sends demo commands
func sendDemoCommands(mqttClient mqtt.Client, codec *codec.Codec, topicBuilder *topic.Builder) {
	time.Sleep(60 * time.Second) // Wait a bit before sending commands

	ticker := time.NewTicker(120 * time.Second)
	defer ticker.Stop()

	commands := []map[string]interface{}{
		{"action": "get_status", "params": map[string]interface{}{}},
		{"action": "restart", "params": map[string]interface{}{"graceful": true}},
		{"action": "get_info", "params": map[string]interface{}{}},
	}

	cmdIndex := 0

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			cmd := commands[cmdIndex%len(commands)]
			cmdIndex++

			command := &device.Command{
				ID:        fmt.Sprintf("demo_cmd_%d", time.Now().UnixNano()),
				Type:      "system",
				Action:    cmd["action"].(string),
				Params:    cmd["params"].(map[string]interface{}),
				Timeout:   30 * time.Second,
				Timestamp: time.Now(),
			}

			commandMsg := map[string]interface{}{
				"command_id":       command.ID,
				"type":             command.Type,
				"action":           command.Action,
				"params":           command.Params,
				"timeout_seconds":  int64(command.Timeout.Seconds()),
				"timestamp":        command.Timestamp,
			}

			payload, err := json.Marshal(commandMsg)
			if err != nil {
				cancel()
				continue
			}

			mqttMsg := &mqtt.Message{
				Topic:     topicBuilder.CommandRequest(),
				Payload:   payload,
				QoS:       mqtt.QoSAtLeastOnce,
				Retained:  false,
				Timestamp: time.Now(),
			}

			if err := mqttClient.PublishMessage(ctx, mqttMsg); err == nil {
				log.Printf("Sent demo command: %s", command.Action)
			}

			cancel()
		}
	}
}

// DemoPlugin is a simple demonstration plugin
type DemoPlugin struct {
	*device.BasePlugin
	mqttClient mqtt.Client
	codec      *codec.Codec
}

// NewDemoPlugin creates a new demo plugin
func NewDemoPlugin() *DemoPlugin {
	info := &device.Info{
		Name:        "Demo Device",
		Type:        "demo",
		Version:     "1.0.0",
		Description: "Demonstration device for RTK framework",
		Vendor:      "RTK",
	}

	plugin := &DemoPlugin{
		BasePlugin: device.NewBasePlugin(info),
	}

	// Set callbacks
	plugin.SetCommandCallback(plugin.handleDemoCommand)

	return plugin
}

// handleDemoCommand handles demo commands
func (p *DemoPlugin) handleDemoCommand(ctx context.Context, cmd *device.Command) (*device.CommandResponse, error) {
	switch cmd.Action {
	case "get_status":
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "success",
			Result: map[string]interface{}{
				"status": "running",
				"uptime": time.Since(time.Now().Add(-time.Hour)).String(),
			},
			Timestamp: time.Now(),
		}, nil
	case "get_info":
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "success",
			Result: map[string]interface{}{
				"name":    p.GetInfo().Name,
				"type":    p.GetInfo().Type,
				"version": p.GetInfo().Version,
			},
			Timestamp: time.Now(),
		}, nil
	case "restart":
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "success",
			Result: map[string]interface{}{
				"message": "Restart initiated",
			},
			Timestamp: time.Now(),
		}, nil
	default:
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     "Unknown command: " + cmd.Action,
			Timestamp: time.Now(),
		}, nil
	}
}

// SetMQTTClient sets the MQTT client
func (p *DemoPlugin) SetMQTTClient(client mqtt.Client) {
	p.mqttClient = client
}

// SetCodec sets the codec
func (p *DemoPlugin) SetCodec(codec *codec.Codec) {
	p.codec = codec
}
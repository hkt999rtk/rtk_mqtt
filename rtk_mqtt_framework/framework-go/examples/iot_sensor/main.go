package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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

// IoTSensorPlugin implements a simple IoT sensor device plugin
type IoTSensorPlugin struct {
	*device.BasePlugin
	
	mqttClient   mqtt.Client
	codec        *codec.Codec
	topicBuilder *topic.Builder
	
	// Sensor configuration
	config       *SensorConfig
	readings     map[string]float64
	lastReadings time.Time
	
	// Channels for coordination
	stopChan     chan struct{}
	telemetryChan chan *device.TelemetryData
}

// SensorConfig represents the sensor configuration
type SensorConfig struct {
	SensorType        string        `json:"sensor_type"`
	ReadingInterval   time.Duration `json:"reading_interval"`
	TelemetryInterval time.Duration `json:"telemetry_interval"`
	Sensors           []SensorDef   `json:"sensors"`
}

// SensorDef defines a sensor
type SensorDef struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Unit     string  `json:"unit"`
	MinValue float64 `json:"min_value"`
	MaxValue float64 `json:"max_value"`
}

// NewIoTSensorPlugin creates a new IoT sensor plugin
func NewIoTSensorPlugin() *IoTSensorPlugin {
	info := &device.Info{
		Name:        "IoT Sensor",
		Type:        "iot_sensor",
		Version:     "1.0.0",
		Description: "Simulated IoT sensor device",
		Vendor:      "RTK",
	}

	plugin := &IoTSensorPlugin{
		BasePlugin:    device.NewBasePlugin(info),
		readings:      make(map[string]float64),
		stopChan:      make(chan struct{}),
		telemetryChan: make(chan *device.TelemetryData, 100),
	}

	// Set callbacks
	plugin.SetStartCallback(plugin.startSensor)
	plugin.SetStopCallback(plugin.stopSensor)
	plugin.SetCommandCallback(plugin.handleSensorCommand)
	plugin.SetTelemetryCallback(plugin.getSensorTelemetry)
	plugin.SetHealthCallback(plugin.getSensorHealth)

	return plugin
}

// Initialize initializes the sensor plugin
func (p *IoTSensorPlugin) Initialize(ctx context.Context, configData json.RawMessage) error {
	if err := p.BasePlugin.Initialize(ctx, configData); err != nil {
		return err
	}

	// Parse sensor configuration
	var sensorConfig SensorConfig
	if err := json.Unmarshal(configData, &sensorConfig); err != nil {
		return fmt.Errorf("failed to parse sensor config: %w", err)
	}

	p.config = &sensorConfig

	// Set default values
	if p.config.ReadingInterval == 0 {
		p.config.ReadingInterval = 5 * time.Second
	}
	if p.config.TelemetryInterval == 0 {
		p.config.TelemetryInterval = 30 * time.Second
	}

	p.GetLogger().WithField("config", p.config).Info("Sensor plugin initialized")
	return nil
}

// startSensor starts the sensor operations
func (p *IoTSensorPlugin) startSensor(ctx context.Context) error {
	p.GetLogger().Info("Starting IoT sensor")

	// Start reading sensors
	go p.sensorReadingLoop()
	
	// Start telemetry publishing
	go p.telemetryPublishLoop()

	return nil
}

// stopSensor stops the sensor operations
func (p *IoTSensorPlugin) stopSensor() error {
	p.GetLogger().Info("Stopping IoT sensor")

	close(p.stopChan)
	return nil
}

// sensorReadingLoop continuously reads sensor values
func (p *IoTSensorPlugin) sensorReadingLoop() {
	ticker := time.NewTicker(p.config.ReadingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.readSensors()
		case <-p.stopChan:
			return
		}
	}
}

// readSensors simulates reading sensor values
func (p *IoTSensorPlugin) readSensors() {
	p.lastReadings = time.Now()

	for _, sensor := range p.config.Sensors {
		// Simulate sensor reading with some variation
		value := sensor.MinValue + rand.Float64()*(sensor.MaxValue-sensor.MinValue)
		p.readings[sensor.Name] = value

		// Create telemetry data
		telemetry := &device.TelemetryData{
			Metric:    sensor.Name,
			Value:     value,
			Unit:      sensor.Unit,
			Labels:    map[string]string{"sensor_type": sensor.Type},
			Timestamp: time.Now(),
		}

		// Send to telemetry channel
		select {
		case p.telemetryChan <- telemetry:
		default:
			p.GetLogger().Warn("Telemetry channel full, dropping reading")
		}
	}

	// Update plugin state
	p.UpdateState(map[string]interface{}{
		"last_reading_time": p.lastReadings,
		"active_sensors":    len(p.config.Sensors),
		"readings":          p.readings,
	})
}

// telemetryPublishLoop publishes telemetry data
func (p *IoTSensorPlugin) telemetryPublishLoop() {
	ticker := time.NewTicker(p.config.TelemetryInterval)
	defer ticker.Stop()

	var batch []*device.TelemetryData

	for {
		select {
		case telemetry := <-p.telemetryChan:
			batch = append(batch, telemetry)

		case <-ticker.C:
			if len(batch) > 0 {
				p.publishTelemetryBatch(batch)
				batch = batch[:0] // Clear batch
			}

		case <-p.stopChan:
			// Publish remaining batch before stopping
			if len(batch) > 0 {
				p.publishTelemetryBatch(batch)
			}
			return
		}
	}
}

// publishTelemetryBatch publishes a batch of telemetry data
func (p *IoTSensorPlugin) publishTelemetryBatch(batch []*device.TelemetryData) {
	if p.mqttClient == nil || p.codec == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, telemetry := range batch {
		msg, err := p.codec.EncodeTelemetry(ctx, telemetry)
		if err != nil {
			p.GetLogger().WithError(err).Error("Failed to encode telemetry")
			continue
		}

		mqttMsg := &mqtt.Message{
			Topic:     msg.Topic,
			Payload:   msg.Payload,
			QoS:       mqtt.QoS(msg.QoS),
			Retained:  msg.Retained,
			Timestamp: time.Now(),
		}

		if err := p.mqttClient.PublishMessage(ctx, mqttMsg); err != nil {
			p.GetLogger().WithError(err).Error("Failed to publish telemetry")
		}
	}

	p.GetLogger().WithField("count", len(batch)).Debug("Published telemetry batch")
}

// handleSensorCommand handles incoming commands
func (p *IoTSensorPlugin) handleSensorCommand(ctx context.Context, cmd *device.Command) (*device.CommandResponse, error) {
	p.GetLogger().WithField("command", cmd.Action).Info("Handling command")

	switch cmd.Action {
	case "get_reading":
		return p.handleGetReading(cmd)
	case "set_interval":
		return p.handleSetInterval(cmd)
	case "get_status":
		return p.handleGetStatus(cmd)
	default:
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     "Unknown command: " + cmd.Action,
			Timestamp: time.Now(),
		}, nil
	}
}

// handleGetReading handles get reading command
func (p *IoTSensorPlugin) handleGetReading(cmd *device.Command) (*device.CommandResponse, error) {
	sensorName, ok := cmd.Params["sensor"].(string)
	if !ok {
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     "Missing sensor parameter",
			Timestamp: time.Now(),
		}, nil
	}

	value, exists := p.readings[sensorName]
	if !exists {
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     "Sensor not found: " + sensorName,
			Timestamp: time.Now(),
		}, nil
	}

	return &device.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Result: map[string]interface{}{
			"sensor": sensorName,
			"value":  value,
			"time":   p.lastReadings,
		},
		Timestamp: time.Now(),
	}, nil
}

// handleSetInterval handles set interval command
func (p *IoTSensorPlugin) handleSetInterval(cmd *device.Command) (*device.CommandResponse, error) {
	intervalStr, ok := cmd.Params["interval"].(string)
	if !ok {
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     "Missing interval parameter",
			Timestamp: time.Now(),
		}, nil
	}

	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return &device.CommandResponse{
			CommandID: cmd.ID,
			Status:    "error",
			Error:     "Invalid interval format: " + err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	p.config.ReadingInterval = interval
	
	return &device.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Result: map[string]interface{}{
			"new_interval": interval.String(),
		},
		Timestamp: time.Now(),
	}, nil
}

// handleGetStatus handles get status command
func (p *IoTSensorPlugin) handleGetStatus(cmd *device.Command) (*device.CommandResponse, error) {
	return &device.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Result: map[string]interface{}{
			"sensor_count":       len(p.config.Sensors),
			"reading_interval":   p.config.ReadingInterval.String(),
			"telemetry_interval": p.config.TelemetryInterval.String(),
			"last_reading":       p.lastReadings,
			"readings":           p.readings,
		},
		Timestamp: time.Now(),
	}, nil
}

// getSensorTelemetry returns telemetry for a specific metric
func (p *IoTSensorPlugin) getSensorTelemetry(ctx context.Context, metric string) (*device.TelemetryData, error) {
	value, exists := p.readings[metric]
	if !exists {
		return nil, fmt.Errorf("metric not found: %s", metric)
	}

	// Find sensor definition
	var unit string
	var sensorType string
	for _, sensor := range p.config.Sensors {
		if sensor.Name == metric {
			unit = sensor.Unit
			sensorType = sensor.Type
			break
		}
	}

	return &device.TelemetryData{
		Metric:    metric,
		Value:     value,
		Unit:      unit,
		Labels:    map[string]string{"sensor_type": sensorType},
		Timestamp: p.lastReadings,
	}, nil
}

// getSensorHealth returns sensor health status
func (p *IoTSensorPlugin) getSensorHealth(ctx context.Context) (*device.Health, error) {
	health := &device.Health{
		Status:      "healthy",
		Score:       1.0,
		Checks:      make(map[string]device.HealthCheck),
		LastCheck:   time.Now(),
		Diagnostics: make(map[string]interface{}),
	}

	// Check if readings are recent
	timeSinceLastReading := time.Since(p.lastReadings)
	if timeSinceLastReading > 2*p.config.ReadingInterval {
		health.Status = "warning"
		health.Score = 0.5
		health.Checks["readings"] = device.HealthCheck{
			Name:        "Recent Readings",
			Status:      "warning",
			Message:     "Readings are stale",
			LastChecked: time.Now(),
		}
	} else {
		health.Checks["readings"] = device.HealthCheck{
			Name:        "Recent Readings",
			Status:      "healthy",
			Message:     "Readings are current",
			LastChecked: time.Now(),
		}
	}

	// Check sensor count
	health.Checks["sensors"] = device.HealthCheck{
		Name:        "Sensor Count",
		Status:      "healthy",
		Value:       len(p.config.Sensors),
		Message:     fmt.Sprintf("%d sensors active", len(p.config.Sensors)),
		LastChecked: time.Now(),
	}

	health.Diagnostics["reading_interval"] = p.config.ReadingInterval.String()
	health.Diagnostics["last_reading_time"] = p.lastReadings
	health.Diagnostics["active_sensors"] = len(p.config.Sensors)

	return health, nil
}

// SetMQTTClient sets the MQTT client for the plugin
func (p *IoTSensorPlugin) SetMQTTClient(client mqtt.Client) {
	p.mqttClient = client
}

// SetCodec sets the message codec for the plugin
func (p *IoTSensorPlugin) SetCodec(codec *codec.Codec) {
	p.codec = codec
}

func main() {
	// Load configuration
	cfg, err := config.LoadFromFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create MQTT client
	mqttClient, err := mqtt.NewClient(&cfg.MQTT)
	if err != nil {
		log.Fatalf("Failed to create MQTT client: %v", err)
	}

	// Create codec
	codec := codec.NewCodec()
	codec.SetDeviceInfo(cfg.Device.Tenant, cfg.Device.Site, cfg.Device.DeviceID)

	// Create topic builder
	topicBuilder := topic.NewBuilder(cfg.Device.Tenant, cfg.Device.Site, cfg.Device.DeviceID)

	// Create sensor plugin
	plugin := NewIoTSensorPlugin()
	plugin.SetMQTTClient(mqttClient)
	plugin.SetCodec(codec)

	// Load plugin configuration
	pluginConfigData := json.RawMessage(`{
		"sensor_type": "environmental",
		"reading_interval": "5s",
		"telemetry_interval": "30s",
		"sensors": [
			{
				"name": "temperature",
				"type": "temperature",
				"unit": "°C",
				"min_value": 15.0,
				"max_value": 35.0
			},
			{
				"name": "humidity",
				"type": "humidity",
				"unit": "%",
				"min_value": 30.0,
				"max_value": 80.0
			},
			{
				"name": "pressure",
				"type": "pressure",
				"unit": "hPa",
				"min_value": 980.0,
				"max_value": 1030.0
			}
		]
	}`)

	// Initialize plugin
	ctx := context.Background()
	if err := plugin.Initialize(ctx, pluginConfigData); err != nil {
		log.Fatalf("Failed to initialize plugin: %v", err)
	}

	// Connect to MQTT broker
	if err := mqttClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer mqttClient.Disconnect()

	log.Println("Connected to MQTT broker")

	// Subscribe to command topics
	commandTopic := topicBuilder.CommandRequest()
	err = mqttClient.Subscribe(ctx, commandTopic, func(ctx context.Context, msg *mqtt.Message) error {
		// Decode command
		rtkMsg, err := codec.Decode(ctx, msg.Topic, msg.Payload)
		if err != nil {
			log.Printf("Failed to decode message: %v", err)
			return err
		}

		cmd, err := codec.DecodeCommand(ctx, rtkMsg)
		if err != nil {
			log.Printf("Failed to decode command: %v", err)
			return err
		}

		// Handle command
		response, err := plugin.HandleCommand(ctx, cmd)
		if err != nil {
			log.Printf("Command handling failed: %v", err)
			return err
		}

		// Publish response
		responseMsg, err := codec.EncodeCommandResponse(ctx, response)
		if err != nil {
			log.Printf("Failed to encode response: %v", err)
			return err
		}

		mqttResponse := &mqtt.Message{
			Topic:     responseMsg.Topic,
			Payload:   responseMsg.Payload,
			QoS:       mqtt.QoS(responseMsg.QoS),
			Retained:  responseMsg.Retained,
			Timestamp: time.Now(),
		}

		return mqttClient.PublishMessage(ctx, mqttResponse)
	}, nil)

	if err != nil {
		log.Fatalf("Failed to subscribe to commands: %v", err)
	}

	log.Printf("Subscribed to commands on: %s", commandTopic)

	// Start plugin
	if err := plugin.Start(ctx); err != nil {
		log.Fatalf("Failed to start plugin: %v", err)
	}

	log.Println("IoT sensor plugin started")

	// Publish initial state
	state, err := plugin.GetState(ctx)
	if err == nil {
		stateMsg, err := codec.EncodeState(ctx, state)
		if err == nil {
			mqttState := &mqtt.Message{
				Topic:     stateMsg.Topic,
				Payload:   stateMsg.Payload,
				QoS:       mqtt.QoS(stateMsg.QoS),
				Retained:  stateMsg.Retained,
				Timestamp: time.Now(),
			}
			mqttClient.PublishMessage(ctx, mqttState)
		}
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("IoT sensor is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutting down...")

	// Stop plugin
	if err := plugin.Stop(); err != nil {
		log.Printf("Error stopping plugin: %v", err)
	}

	log.Println("IoT sensor stopped")
}
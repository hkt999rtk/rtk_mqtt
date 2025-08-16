package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"

	"rtk_controller/internal/config"
	"rtk_controller/internal/storage"
	"rtk_controller/pkg/utils"
)

// DeviceManager interface for device state management
type DeviceManager interface {
	UpdateDeviceState(topic string, payload []byte) error
}

// EventProcessor interface for event processing
type EventProcessor interface {
	ProcessEvent(topic string, payload []byte) error
}

// SchemaValidator interface for schema validation  
type SchemaValidator interface {
	ValidateMessage(topic string, payload []byte) (*ValidationResult, error)
}

// ValidationResult represents schema validation result
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
	Schema string   `json:"schema"`
}

// Client represents an MQTT client wrapper
type Client struct {
	client     mqtt.Client
	config     config.MQTTConfig
	storage    storage.Storage
	logger     *MessageLogger
	subscriber *Subscriber
	publisher  *Publisher
	mu         sync.RWMutex
	connected  bool
	handlers   map[string]MessageHandler
	
	// Device management (injected after creation)
	deviceManager   DeviceManager
	eventProcessor  EventProcessor
	schemaValidator SchemaValidator
}

// MessageHandler defines the interface for handling MQTT messages
type MessageHandler interface {
	HandleMessage(topic string, payload []byte) error
}

// NewClient creates a new MQTT client
func NewClient(cfg config.MQTTConfig, storage storage.Storage) (*Client, error) {
	client := &Client{
		config:   cfg,
		storage:  storage,
		handlers: make(map[string]MessageHandler),
	}

	// Initialize message logger if enabled
	if cfg.Logging.Enabled {
		logger, err := NewMessageLogger(cfg.Logging, storage)
		if err != nil {
			return nil, fmt.Errorf("failed to create message logger: %w", err)
		}
		client.logger = logger
	}

	// Initialize subscriber and publisher
	client.subscriber = NewSubscriber(client)
	client.publisher = NewPublisher(client)

	// Create MQTT client options
	opts := mqtt.NewClientOptions()
	
	// Broker URL
	brokerURL := fmt.Sprintf("tcp://%s:%d", cfg.Broker, cfg.Port)
	if cfg.TLS.Enabled {
		brokerURL = fmt.Sprintf("ssl://%s:%d", cfg.Broker, cfg.Port)
		
		// Configure TLS
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.TLS.SkipVerify,
		}
		
		if cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		
		opts.SetTLSConfig(tlsConfig)
	}
	
	opts.AddBroker(brokerURL)
	opts.SetClientID(cfg.ClientID)
	
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}
	
	// Set handlers
	opts.SetDefaultPublishHandler(client.defaultMessageHandler)
	opts.SetOnConnectHandler(client.onConnect)
	opts.SetConnectionLostHandler(client.onConnectionLost)
	
	// Connection settings
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(1 * time.Minute)
	
	// Set LWT (Last Will Testament)
	lwtTopic := fmt.Sprintf("rtk/controller/%s/lwt", cfg.ClientID)
	lwtPayload := fmt.Sprintf(`{"status":"offline","ts":%d}`, time.Now().UnixMilli())
	opts.SetWill(lwtTopic, lwtPayload, 1, true)
	
	client.client = mqtt.NewClient(opts)
	
	return client, nil
}

// Connect establishes connection to MQTT broker
func (c *Client) Connect(ctx context.Context) error {
	log.WithField("broker", fmt.Sprintf("%s:%d", c.config.Broker, c.config.Port)).Info("Connecting to MQTT broker")
	
	token := c.client.Connect()
	
	// Wait for connection with context timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		if !token.WaitTimeout(1 * time.Second) {
			return fmt.Errorf("connection timeout")
		}
	default:
		token.Wait()
	}
	
	if token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}
	
	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()
	
	// Start message logger if enabled
	if c.logger != nil {
		if err := c.logger.Start(ctx); err != nil {
			log.WithError(err).Warn("Failed to start message logger")
		}
	}
	
	log.Info("Connected to MQTT broker successfully")
	return nil
}

// Disconnect closes the connection to MQTT broker
func (c *Client) Disconnect() {
	log.Info("Disconnecting from MQTT broker")
	
	// Stop message logger
	if c.logger != nil {
		c.logger.Stop()
	}
	
	// Publish online status before disconnecting
	lwtTopic := fmt.Sprintf("rtk/controller/%s/lwt", c.config.ClientID)
	lwtPayload := fmt.Sprintf(`{"status":"offline","ts":%d,"reason":"normal_shutdown"}`, time.Now().UnixMilli())
	c.client.Publish(lwtTopic, 1, true, lwtPayload)
	
	c.client.Disconnect(250)
	
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()
	
	log.Info("Disconnected from MQTT broker")
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.client.IsConnected()
}

// Subscribe subscribes to topics
func (c *Client) Subscribe(topics []string) error {
	return c.subscriber.Subscribe(topics)
}

// Publish publishes a message to a topic
func (c *Client) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	return c.publisher.Publish(topic, qos, retained, payload)
}

// RegisterHandler registers a message handler for specific topic patterns
func (c *Client) RegisterHandler(pattern string, handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[pattern] = handler
}

// UnregisterHandler removes a message handler
func (c *Client) UnregisterHandler(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.handlers, pattern)
}

// GetMessageLogger returns the message logger instance
func (c *Client) GetMessageLogger() *MessageLogger {
	return c.logger
}

// SetDeviceManager sets the device manager for handling device state updates
func (c *Client) SetDeviceManager(dm DeviceManager) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deviceManager = dm
}

// SetEventProcessor sets the event processor for handling device events
func (c *Client) SetEventProcessor(ep EventProcessor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventProcessor = ep
}

// SetSchemaValidator sets the schema validator for message validation
func (c *Client) SetSchemaValidator(sv SchemaValidator) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.schemaValidator = sv
}

// defaultMessageHandler handles incoming MQTT messages
func (c *Client) defaultMessageHandler(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()
	
	log.WithFields(log.Fields{
		"topic": topic,
		"qos":   msg.Qos(),
		"size":  len(payload),
	}).Debug("Received MQTT message")
	
	// Validate message using schema validator if enabled
	c.mu.RLock()
	schemaValidator := c.schemaValidator
	c.mu.RUnlock()
	
	if schemaValidator != nil {
		validationResult, err := schemaValidator.ValidateMessage(topic, payload)
		if err != nil {
			log.WithFields(log.Fields{
				"topic": topic,
				"error": err,
			}).Error("Schema validation error")
		} else if !validationResult.Valid {
			log.WithFields(log.Fields{
				"topic":  topic,
				"schema": validationResult.Schema,
				"errors": validationResult.Errors,
			}).Warn("Message failed schema validation")
			
			// For strict validation, we could skip processing invalid messages
			// For now, we log the validation failure and continue processing
		} else {
			log.WithFields(log.Fields{
				"topic":  topic,
				"schema": validationResult.Schema,
			}).Debug("Message passed schema validation")
		}
	}
	
	// Log message if logger is enabled
	if c.logger != nil {
		if err := c.logger.LogMessage(topic, payload, msg.Qos(), msg.Retained()); err != nil {
			log.WithError(err).Warn("Failed to log MQTT message")
		}
	}
	
	// Process device state updates and events
	c.mu.RLock()
	deviceManager := c.deviceManager
	eventProcessor := c.eventProcessor
	c.mu.RUnlock()
	
	// Check if this is a device-related message (rtk/v1/...)
	if len(topic) > 6 && topic[:6] == "rtk/v1" {
		// Extract message type from topic
		parts := strings.Split(topic, "/")
		if len(parts) >= 6 {
			messageType := parts[5]
			
			switch {
			case messageType == "state" || messageType == "attr" || messageType == "lwt":
				// Handle device state update
				if deviceManager != nil {
					if err := deviceManager.UpdateDeviceState(topic, payload); err != nil {
						log.WithFields(log.Fields{
							"topic": topic,
							"error": err,
						}).Error("Failed to update device state")
					}
				}
				
			case messageType == "evt" || strings.HasPrefix(messageType, "evt/"):
				// Handle device event
				if eventProcessor != nil {
					if err := eventProcessor.ProcessEvent(topic, payload); err != nil {
						log.WithFields(log.Fields{
							"topic": topic,
							"error": err,
						}).Error("Failed to process device event")
					}
				}
				
			case strings.HasPrefix(messageType, "telemetry"):
				// Handle telemetry data (also updates device state)
				if deviceManager != nil {
					if err := deviceManager.UpdateDeviceState(topic, payload); err != nil {
						log.WithFields(log.Fields{
							"topic": topic,
							"error": err,
						}).Error("Failed to update device telemetry")
					}
				}
			}
		}
	}
	
	// Find and execute matching handlers
	c.mu.RLock()
	handlers := make(map[string]MessageHandler)
	for pattern, handler := range c.handlers {
		handlers[pattern] = handler
	}
	c.mu.RUnlock()
	
	for pattern, handler := range handlers {
		if c.topicMatches(pattern, topic) {
			if err := handler.HandleMessage(topic, payload); err != nil {
				log.WithFields(log.Fields{
					"topic":   topic,
					"pattern": pattern,
					"error":   err,
				}).Error("Handler failed to process message")
			}
		}
	}
}

// onConnect is called when connection is established
func (c *Client) onConnect(client mqtt.Client) {
	log.Info("MQTT connection established")
	
	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()
	
	// Publish online status
	lwtTopic := fmt.Sprintf("rtk/controller/%s/lwt", c.config.ClientID)
	lwtPayload := fmt.Sprintf(`{"status":"online","ts":%d}`, time.Now().UnixMilli())
	client.Publish(lwtTopic, 1, true, lwtPayload)
	
	// Subscribe to configured topics
	if err := c.Subscribe(c.config.Topics.Subscribe); err != nil {
		log.WithError(err).Error("Failed to subscribe to topics after reconnection")
	}
}

// onConnectionLost is called when connection is lost
func (c *Client) onConnectionLost(client mqtt.Client, err error) {
	log.WithError(err).Warn("MQTT connection lost")
	
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()
}

// topicMatches checks if a topic matches a pattern (supports MQTT wildcards)
func (c *Client) topicMatches(pattern, topic string) bool {
	// Use the utility function for proper MQTT wildcard matching
	return utils.TopicMatches(pattern, topic)
}
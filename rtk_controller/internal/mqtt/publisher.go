package mqtt

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// Publisher handles MQTT message publishing
type Publisher struct {
	client *Client
}

// NewPublisher creates a new publisher
func NewPublisher(client *Client) *Publisher {
	return &Publisher{
		client: client,
	}
}

// Publish publishes a message to a topic
func (p *Publisher) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	if !p.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	var data []byte
	var err error

	// Convert payload to bytes
	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	log.WithFields(log.Fields{
		"topic":    topic,
		"qos":      qos,
		"retained": retained,
		"size":     len(data),
	}).Debug("Publishing MQTT message")

	token := p.client.client.Publish(topic, qos, retained, data)
	
	// Wait for publish with timeout
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish timeout for topic %s", topic)
	}

	if token.Error() != nil {
		return fmt.Errorf("failed to publish to topic %s: %w", topic, token.Error())
	}

	log.WithField("topic", topic).Debug("Successfully published message")
	return nil
}

// PublishCommand publishes a command to a device
func (p *Publisher) PublishCommand(deviceID, operation string, args map[string]interface{}, timeout time.Duration) (string, error) {
	// Generate command ID
	cmdID := fmt.Sprintf("cmd-%d", time.Now().UnixMilli())
	
	// Build command payload
	command := map[string]interface{}{
		"id":         cmdID,
		"op":         operation,
		"schema":     fmt.Sprintf("cmd.%s/1.0", operation),
		"args":       args,
		"timeout_ms": int64(timeout.Milliseconds()),
		"expect":     "result",
		"reply_to":   nil,
		"ts":         time.Now().UnixMilli(),
	}

	// Extract tenant, site from device ID (assuming format: tenant-site-device)
	// For now, use default values
	topic := fmt.Sprintf("rtk/v1/default/default/%s/cmd/req", deviceID)
	
	err := p.Publish(topic, 1, false, command)
	if err != nil {
		return "", fmt.Errorf("failed to publish command: %w", err)
	}

	log.WithFields(log.Fields{
		"device_id": deviceID,
		"cmd_id":    cmdID,
		"operation": operation,
	}).Info("Published command to device")

	return cmdID, nil
}

// PublishState publishes controller state
func (p *Publisher) PublishState() error {
	state := map[string]interface{}{
		"schema":     "state/1.0",
		"ts":         time.Now().UnixMilli(),
		"health":     "ok",
		"uptime_s":   int64(time.Since(time.Now()).Seconds()), // TODO: Track actual uptime
		"version":    "1.0.0", // TODO: Get from build info
		"components": map[string]interface{}{
			"mqtt":      "connected",
			"storage":   "ready",
			"api":       "running",
			"console":   "running",
			"diagnosis": "ready",
		},
	}

	topic := fmt.Sprintf("rtk/controller/%s/state", p.client.config.ClientID)
	return p.Publish(topic, 0, true, state)
}

// PublishEvent publishes an event
func (p *Publisher) PublishEvent(eventType, severity string, data map[string]interface{}) error {
	event := map[string]interface{}{
		"schema":   fmt.Sprintf("evt.%s/1.0", eventType),
		"ts":       time.Now().UnixMilli(),
		"severity": severity,
	}

	// Merge event data
	for k, v := range data {
		event[k] = v
	}

	topic := fmt.Sprintf("rtk/controller/%s/evt/%s", p.client.config.ClientID, eventType)
	return p.Publish(topic, 1, false, event)
}
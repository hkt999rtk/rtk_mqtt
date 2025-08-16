package codec

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/rtk/mqtt-framework/pkg/device"
	"github.com/rtk/mqtt-framework/pkg/topic"
)

// MessageType represents different types of RTK messages
type MessageType string

const (
	MessageTypeState      MessageType = "state"
	MessageTypeTelemetry  MessageType = "telemetry"
	MessageTypeEvent      MessageType = "event"
	MessageTypeAttribute  MessageType = "attribute"
	MessageTypeCommand    MessageType = "command"
	MessageTypeLWT        MessageType = "lwt"
)

// RTKMessage represents a standardized RTK MQTT message
type RTKMessage struct {
	// Message metadata
	MessageID   string            `json:"message_id"`
	MessageType MessageType       `json:"message_type"`
	SchemaID    string            `json:"schema_id,omitempty"`
	Version     string            `json:"version"`
	Timestamp   time.Time         `json:"timestamp"`
	Headers     map[string]string `json:"headers,omitempty"`

	// Device information
	DeviceID string `json:"device_id"`
	Tenant   string `json:"tenant"`
	Site     string `json:"site"`

	// Message payload
	Payload json.RawMessage `json:"payload"`

	// MQTT properties
	Topic    string `json:"topic"`
	QoS      byte   `json:"qos"`
	Retained bool   `json:"retained"`
}

// StateMessage represents a device state message
type StateMessage struct {
	Status      string                 `json:"status"`
	Health      string                 `json:"health"`
	LastSeen    time.Time              `json:"last_seen"`
	Uptime      int64                  `json:"uptime_seconds"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Diagnostics map[string]interface{} `json:"diagnostics,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// TelemetryMessage represents a telemetry message
type TelemetryMessage struct {
	Metric    string                 `json:"metric"`
	Value     interface{}            `json:"value"`
	Unit      string                 `json:"unit,omitempty"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// EventMessage represents an event message
type EventMessage struct {
	EventID   string                 `json:"event_id"`
	Type      string                 `json:"type"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Category  string                 `json:"category,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	SchemaID  string                 `json:"schema_id,omitempty"`
}

// AttributeMessage represents device attributes
type AttributeMessage struct {
	Attributes map[string]interface{} `json:"attributes"`
	Timestamp  time.Time              `json:"timestamp"`
}

// CommandMessage represents a command message
type CommandMessage struct {
	CommandID string                 `json:"command_id"`
	Type      string                 `json:"type"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Timeout   int64                  `json:"timeout_seconds,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// CommandResponseMessage represents a command response
type CommandResponseMessage struct {
	CommandID   string                 `json:"command_id"`
	RequestID   string                 `json:"request_id,omitempty"`
	Status      string                 `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	ProcessTime int64                  `json:"process_time_ms,omitempty"`
}

// LWTMessage represents a Last Will Testament message
type LWTMessage struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

// Codec handles encoding and decoding of RTK messages
type Codec struct {
	topicBuilder *topic.Builder
	topicParser  *topic.Parser
}

// NewCodec creates a new message codec
func NewCodec() *Codec {
	return &Codec{
		topicParser: topic.NewParser(),
	}
}

// SetDeviceInfo sets the device information for topic building
func (c *Codec) SetDeviceInfo(tenant, site, deviceID string) {
	c.topicBuilder = topic.NewBuilder(tenant, site, deviceID)
}

// EncodeState encodes a device state message
func (c *Codec) EncodeState(ctx context.Context, state *device.State) (*RTKMessage, error) {
	if c.topicBuilder == nil {
		return nil, fmt.Errorf("device info not set")
	}

	stateMsg := &StateMessage{
		Status:      state.Status,
		Health:      state.Health,
		LastSeen:    state.LastSeen,
		Uptime:      int64(state.Uptime.Seconds()),
		Properties:  state.Properties,
		Diagnostics: state.Diagnostics,
		Timestamp:   state.Timestamp,
	}

	payload, err := json.Marshal(stateMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state message: %w", err)
	}

	return &RTKMessage{
		MessageID:   generateMessageID(),
		MessageType: MessageTypeState,
		Version:     "1.0",
		Timestamp:   time.Now(),
		DeviceID:    c.topicBuilder.Base(),
		Payload:     payload,
		Topic:       c.topicBuilder.State(),
		QoS:         1,
		Retained:    true,
	}, nil
}

// EncodeTelemetry encodes a telemetry message
func (c *Codec) EncodeTelemetry(ctx context.Context, telemetry *device.TelemetryData) (*RTKMessage, error) {
	if c.topicBuilder == nil {
		return nil, fmt.Errorf("device info not set")
	}

	telemetryMsg := &TelemetryMessage{
		Metric:    telemetry.Metric,
		Value:     telemetry.Value,
		Unit:      telemetry.Unit,
		Labels:    telemetry.Labels,
		Metadata:  telemetry.Metadata,
		Timestamp: telemetry.Timestamp,
	}

	payload, err := json.Marshal(telemetryMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal telemetry message: %w", err)
	}

	return &RTKMessage{
		MessageID:   generateMessageID(),
		MessageType: MessageTypeTelemetry,
		Version:     "1.0",
		Timestamp:   time.Now(),
		DeviceID:    c.topicBuilder.Base(),
		Payload:     payload,
		Topic:       c.topicBuilder.Telemetry(telemetry.Metric),
		QoS:         0,
		Retained:    false,
	}, nil
}

// EncodeEvent encodes an event message
func (c *Codec) EncodeEvent(ctx context.Context, event *device.Event) (*RTKMessage, error) {
	if c.topicBuilder == nil {
		return nil, fmt.Errorf("device info not set")
	}

	eventMsg := &EventMessage{
		EventID:   event.ID,
		Type:      event.Type,
		Level:     event.Level,
		Message:   event.Message,
		Source:    event.Source,
		Category:  event.Category,
		Data:      event.Data,
		Timestamp: event.Timestamp,
		SchemaID:  event.SchemaID,
	}

	payload, err := json.Marshal(eventMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event message: %w", err)
	}

	return &RTKMessage{
		MessageID:   generateMessageID(),
		MessageType: MessageTypeEvent,
		SchemaID:    event.SchemaID,
		Version:     "1.0",
		Timestamp:   time.Now(),
		DeviceID:    c.topicBuilder.Base(),
		Payload:     payload,
		Topic:       c.topicBuilder.Event(event.Type),
		QoS:         1,
		Retained:    false,
	}, nil
}

// EncodeAttributes encodes device attributes
func (c *Codec) EncodeAttributes(ctx context.Context, attributes map[string]interface{}) (*RTKMessage, error) {
	if c.topicBuilder == nil {
		return nil, fmt.Errorf("device info not set")
	}

	attrMsg := &AttributeMessage{
		Attributes: attributes,
		Timestamp:  time.Now(),
	}

	payload, err := json.Marshal(attrMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes message: %w", err)
	}

	return &RTKMessage{
		MessageID:   generateMessageID(),
		MessageType: MessageTypeAttribute,
		Version:     "1.0",
		Timestamp:   time.Now(),
		DeviceID:    c.topicBuilder.Base(),
		Payload:     payload,
		Topic:       c.topicBuilder.Attribute(),
		QoS:         1,
		Retained:    true,
	}, nil
}

// EncodeCommandResponse encodes a command response
func (c *Codec) EncodeCommandResponse(ctx context.Context, response *device.CommandResponse) (*RTKMessage, error) {
	if c.topicBuilder == nil {
		return nil, fmt.Errorf("device info not set")
	}

	responseMsg := &CommandResponseMessage{
		CommandID: response.CommandID,
		Status:    response.Status,
		Result:    response.Result,
		Error:     response.Error,
		Timestamp: response.Timestamp,
	}

	payload, err := json.Marshal(responseMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command response: %w", err)
	}

	return &RTKMessage{
		MessageID:   generateMessageID(),
		MessageType: MessageTypeCommand,
		Version:     "1.0",
		Timestamp:   time.Now(),
		DeviceID:    c.topicBuilder.Base(),
		Payload:     payload,
		Topic:       c.topicBuilder.CommandResponse(),
		QoS:         1,
		Retained:    false,
	}, nil
}

// EncodeLWT encodes a Last Will Testament message
func (c *Codec) EncodeLWT(ctx context.Context, status string, message string) (*RTKMessage, error) {
	if c.topicBuilder == nil {
		return nil, fmt.Errorf("device info not set")
	}

	lwtMsg := &LWTMessage{
		Status:    status,
		Timestamp: time.Now(),
		Message:   message,
	}

	payload, err := json.Marshal(lwtMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LWT message: %w", err)
	}

	return &RTKMessage{
		MessageID:   generateMessageID(),
		MessageType: MessageTypeLWT,
		Version:     "1.0",
		Timestamp:   time.Now(),
		DeviceID:    c.topicBuilder.Base(),
		Payload:     payload,
		Topic:       c.topicBuilder.LWT(),
		QoS:         1,
		Retained:    true,
	}, nil
}

// Decode decodes an RTK message from topic and payload
func (c *Codec) Decode(ctx context.Context, topicStr string, payload []byte) (*RTKMessage, error) {
	// Parse topic
	topicInfo := c.topicParser.Parse(topicStr)
	if !topicInfo.IsValid {
		return nil, fmt.Errorf("invalid RTK topic: %s", topicStr)
	}

	// Create base message
	msg := &RTKMessage{
		Topic:    topicStr,
		DeviceID: topicInfo.DeviceID,
		Tenant:   topicInfo.Tenant,
		Site:     topicInfo.Site,
		Payload:  payload,
	}

	// Determine message type and decode accordingly
	switch topicInfo.MessageType {
	case "state":
		msg.MessageType = MessageTypeState
	case "telemetry":
		msg.MessageType = MessageTypeTelemetry
	case "evt":
		msg.MessageType = MessageTypeEvent
	case "attr":
		msg.MessageType = MessageTypeAttribute
	case "cmd":
		msg.MessageType = MessageTypeCommand
	case "lwt":
		msg.MessageType = MessageTypeLWT
	default:
		return nil, fmt.Errorf("unknown message type: %s", topicInfo.MessageType)
	}

	// Try to extract message metadata if payload is JSON
	var envelope map[string]interface{}
	if err := json.Unmarshal(payload, &envelope); err == nil {
		if messageID, ok := envelope["message_id"].(string); ok {
			msg.MessageID = messageID
		}
		if version, ok := envelope["version"].(string); ok {
			msg.Version = version
		}
		if timestampStr, ok := envelope["timestamp"].(string); ok {
			if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
				msg.Timestamp = timestamp
			}
		}
		if schemaID, ok := envelope["schema_id"].(string); ok {
			msg.SchemaID = schemaID
		}
	}

	return msg, nil
}

// DecodeCommand decodes a command message
func (c *Codec) DecodeCommand(ctx context.Context, msg *RTKMessage) (*device.Command, error) {
	if msg.MessageType != MessageTypeCommand {
		return nil, fmt.Errorf("message is not a command")
	}

	var cmdMsg CommandMessage
	if err := json.Unmarshal(msg.Payload, &cmdMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal command message: %w", err)
	}

	cmd := &device.Command{
		ID:        cmdMsg.CommandID,
		Type:      cmdMsg.Type,
		Action:    cmdMsg.Action,
		Params:    cmdMsg.Params,
		Timestamp: cmdMsg.Timestamp,
	}

	if cmdMsg.Timeout > 0 {
		cmd.Timeout = time.Duration(cmdMsg.Timeout) * time.Second
	}

	return cmd, nil
}

// DecodeState decodes a state message
func (c *Codec) DecodeState(ctx context.Context, msg *RTKMessage) (*device.State, error) {
	if msg.MessageType != MessageTypeState {
		return nil, fmt.Errorf("message is not a state message")
	}

	var stateMsg StateMessage
	if err := json.Unmarshal(msg.Payload, &stateMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state message: %w", err)
	}

	state := &device.State{
		Status:      stateMsg.Status,
		Health:      stateMsg.Health,
		LastSeen:    stateMsg.LastSeen,
		Uptime:      time.Duration(stateMsg.Uptime) * time.Second,
		Properties:  stateMsg.Properties,
		Diagnostics: stateMsg.Diagnostics,
		Timestamp:   stateMsg.Timestamp,
	}

	return state, nil
}

// DecodeTelemetry decodes a telemetry message
func (c *Codec) DecodeTelemetry(ctx context.Context, msg *RTKMessage) (*device.TelemetryData, error) {
	if msg.MessageType != MessageTypeTelemetry {
		return nil, fmt.Errorf("message is not a telemetry message")
	}

	var telemetryMsg TelemetryMessage
	if err := json.Unmarshal(msg.Payload, &telemetryMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal telemetry message: %w", err)
	}

	telemetry := &device.TelemetryData{
		Metric:    telemetryMsg.Metric,
		Value:     telemetryMsg.Value,
		Unit:      telemetryMsg.Unit,
		Labels:    telemetryMsg.Labels,
		Metadata:  telemetryMsg.Metadata,
		Timestamp: telemetryMsg.Timestamp,
	}

	return telemetry, nil
}

// DecodeEvent decodes an event message
func (c *Codec) DecodeEvent(ctx context.Context, msg *RTKMessage) (*device.Event, error) {
	if msg.MessageType != MessageTypeEvent {
		return nil, fmt.Errorf("message is not an event message")
	}

	var eventMsg EventMessage
	if err := json.Unmarshal(msg.Payload, &eventMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event message: %w", err)
	}

	event := &device.Event{
		ID:        eventMsg.EventID,
		Type:      eventMsg.Type,
		Level:     eventMsg.Level,
		Message:   eventMsg.Message,
		Source:    eventMsg.Source,
		Category:  eventMsg.Category,
		Data:      eventMsg.Data,
		Timestamp: eventMsg.Timestamp,
		SchemaID:  eventMsg.SchemaID,
	}

	return event, nil
}

var messageIDCounter uint64

// generateMessageID generates a unique message ID
func generateMessageID() string {
	counter := atomic.AddUint64(&messageIDCounter, 1)
	return fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), counter)
}
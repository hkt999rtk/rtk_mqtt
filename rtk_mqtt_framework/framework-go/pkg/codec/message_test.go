package codec

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rtk/mqtt-framework/pkg/device"
)

func TestNewCodec(t *testing.T) {
	codec := NewCodec()
	
	if codec == nil {
		t.Fatal("Expected codec to be created")
	}
	
	if codec.topicParser == nil {
		t.Error("Expected topic parser to be initialized")
	}
}

func TestCodecSetDeviceInfo(t *testing.T) {
	codec := NewCodec()
	
	tenant := "demo"
	site := "office"
	deviceID := "sensor001"
	
	codec.SetDeviceInfo(tenant, site, deviceID)
	
	if codec.topicBuilder == nil {
		t.Error("Expected topic builder to be set")
	}
	
	// Test that builder works correctly
	expectedBase := "rtk/v1/demo/office/sensor001"
	actualBase := codec.topicBuilder.Base()
	
	if actualBase != expectedBase {
		t.Errorf("Expected base topic %s, got %s", expectedBase, actualBase)
	}
}

func TestEncodeState(t *testing.T) {
	codec := NewCodec()
	codec.SetDeviceInfo("demo", "office", "00:11:22:33:44:55")
	
	state := &device.State{
		Status:    "online",
		Health:    "healthy",
		LastSeen:  time.Now(),
		Uptime:    time.Hour,
		Timestamp: time.Now(),
		Properties: map[string]interface{}{
			"version": "1.0.0",
		},
		Diagnostics: map[string]interface{}{
			"cpu_usage": 45.2,
		},
	}
	
	ctx := context.Background()
	msg, err := codec.EncodeState(ctx, state)
	
	if err != nil {
		t.Fatalf("Failed to encode state: %v", err)
	}
	
	if msg == nil {
		t.Fatal("Expected message to be created")
	}
	
	if msg.MessageType != MessageTypeState {
		t.Errorf("Expected message type %s, got %s", MessageTypeState, msg.MessageType)
	}
	
	if msg.Topic != "rtk/v1/demo/office/00:11:22:33:44:55/state" {
		t.Errorf("Expected topic rtk/v1/demo/office/sensor001/state, got %s", msg.Topic)
	}
	
	if msg.QoS != 1 {
		t.Errorf("Expected QoS 1, got %d", msg.QoS)
	}
	
	if !msg.Retained {
		t.Error("Expected state message to be retained")
	}
	
	// Decode payload to verify content
	var stateMsg StateMessage
	if err := json.Unmarshal(msg.Payload, &stateMsg); err != nil {
		t.Fatalf("Failed to unmarshal state payload: %v", err)
	}
	
	if stateMsg.Status != state.Status {
		t.Errorf("Expected status %s, got %s", state.Status, stateMsg.Status)
	}
	
	if stateMsg.Health != state.Health {
		t.Errorf("Expected health %s, got %s", state.Health, stateMsg.Health)
	}
}

func TestEncodeTelemetry(t *testing.T) {
	codec := NewCodec()
	codec.SetDeviceInfo("demo", "office", "00:11:22:33:44:55")
	
	telemetry := &device.TelemetryData{
		Metric:    "temperature",
		Value:     23.5,
		Unit:      "°C",
		Labels:    map[string]string{"sensor": "ds18b20"},
		Timestamp: time.Now(),
	}
	
	ctx := context.Background()
	msg, err := codec.EncodeTelemetry(ctx, telemetry)
	
	if err != nil {
		t.Fatalf("Failed to encode telemetry: %v", err)
	}
	
	if msg.MessageType != MessageTypeTelemetry {
		t.Errorf("Expected message type %s, got %s", MessageTypeTelemetry, msg.MessageType)
	}
	
	if msg.Topic != "rtk/v1/demo/office/00:11:22:33:44:55/telemetry/temperature" {
		t.Errorf("Expected topic rtk/v1/demo/office/sensor001/telemetry/temperature, got %s", msg.Topic)
	}
	
	if msg.QoS != 0 {
		t.Errorf("Expected QoS 0, got %d", msg.QoS)
	}
	
	if msg.Retained {
		t.Error("Expected telemetry message not to be retained")
	}
	
	// Decode payload
	var telemetryMsg TelemetryMessage
	if err := json.Unmarshal(msg.Payload, &telemetryMsg); err != nil {
		t.Fatalf("Failed to unmarshal telemetry payload: %v", err)
	}
	
	if telemetryMsg.Metric != telemetry.Metric {
		t.Errorf("Expected metric %s, got %s", telemetry.Metric, telemetryMsg.Metric)
	}
	
	if telemetryMsg.Value != telemetry.Value {
		t.Errorf("Expected value %v, got %v", telemetry.Value, telemetryMsg.Value)
	}
	
	if telemetryMsg.Unit != telemetry.Unit {
		t.Errorf("Expected unit %s, got %s", telemetry.Unit, telemetryMsg.Unit)
	}
}

func TestEncodeEvent(t *testing.T) {
	codec := NewCodec()
	codec.SetDeviceInfo("demo", "office", "aa:bb:cc:dd:ee:ff")
	
	event := &device.Event{
		ID:       "event_001",
		Type:     "wifi.roam_miss",
		Level:    "warning",
		Message:  "WiFi roaming failed",
		Source:   "wifi_monitor",
		Category: "roaming",
		Data: map[string]interface{}{
			"current_ap": map[string]interface{}{
				"bssid": "aa:bb:cc:dd:ee:ff",
				"ssid":  "TestNetwork",
			},
		},
		Timestamp: time.Now(),
		SchemaID:  "evt.wifi.roam_miss/1.0",
	}
	
	ctx := context.Background()
	msg, err := codec.EncodeEvent(ctx, event)
	
	if err != nil {
		t.Fatalf("Failed to encode event: %v", err)
	}
	
	if msg.MessageType != MessageTypeEvent {
		t.Errorf("Expected message type %s, got %s", MessageTypeEvent, msg.MessageType)
	}
	
	if msg.SchemaID != event.SchemaID {
		t.Errorf("Expected schema ID %s, got %s", event.SchemaID, msg.SchemaID)
	}
	
	if msg.Topic != "rtk/v1/demo/office/aa:bb:cc:dd:ee:ff/evt/wifi.roam_miss" {
		t.Errorf("Expected topic rtk/v1/demo/office/router001/evt/wifi.roam_miss, got %s", msg.Topic)
	}
	
	// Decode payload
	var eventMsg EventMessage
	if err := json.Unmarshal(msg.Payload, &eventMsg); err != nil {
		t.Fatalf("Failed to unmarshal event payload: %v", err)
	}
	
	if eventMsg.EventID != event.ID {
		t.Errorf("Expected event ID %s, got %s", event.ID, eventMsg.EventID)
	}
	
	if eventMsg.Type != event.Type {
		t.Errorf("Expected type %s, got %s", event.Type, eventMsg.Type)
	}
	
	if eventMsg.Level != event.Level {
		t.Errorf("Expected level %s, got %s", event.Level, eventMsg.Level)
	}
}

func TestEncodeCommandResponse(t *testing.T) {
	codec := NewCodec()
	codec.SetDeviceInfo("demo", "office", "00:11:22:33:44:55")
	
	response := &device.CommandResponse{
		CommandID: "cmd_123",
		Status:    "success",
		Result: map[string]interface{}{
			"temperature": 23.5,
			"humidity":    65.0,
		},
		Timestamp: time.Now(),
	}
	
	ctx := context.Background()
	msg, err := codec.EncodeCommandResponse(ctx, response)
	
	if err != nil {
		t.Fatalf("Failed to encode command response: %v", err)
	}
	
	if msg.MessageType != MessageTypeCommand {
		t.Errorf("Expected message type %s, got %s", MessageTypeCommand, msg.MessageType)
	}
	
	if msg.Topic != "rtk/v1/demo/office/00:11:22:33:44:55/cmd/res" {
		t.Errorf("Expected topic rtk/v1/demo/office/sensor001/cmd/res, got %s", msg.Topic)
	}
	
	// Decode payload
	var responseMsg CommandResponseMessage
	if err := json.Unmarshal(msg.Payload, &responseMsg); err != nil {
		t.Fatalf("Failed to unmarshal response payload: %v", err)
	}
	
	if responseMsg.CommandID != response.CommandID {
		t.Errorf("Expected command ID %s, got %s", response.CommandID, responseMsg.CommandID)
	}
	
	if responseMsg.Status != response.Status {
		t.Errorf("Expected status %s, got %s", response.Status, responseMsg.Status)
	}
}

func TestEncodeLWT(t *testing.T) {
	codec := NewCodec()
	codec.SetDeviceInfo("demo", "office", "00:11:22:33:44:55")
	
	ctx := context.Background()
	msg, err := codec.EncodeLWT(ctx, "offline", "Device disconnected unexpectedly")
	
	if err != nil {
		t.Fatalf("Failed to encode LWT: %v", err)
	}
	
	if msg.MessageType != MessageTypeLWT {
		t.Errorf("Expected message type %s, got %s", MessageTypeLWT, msg.MessageType)
	}
	
	if msg.Topic != "rtk/v1/demo/office/00:11:22:33:44:55/lwt" {
		t.Errorf("Expected topic rtk/v1/demo/office/sensor001/lwt, got %s", msg.Topic)
	}
	
	if !msg.Retained {
		t.Error("Expected LWT message to be retained")
	}
	
	// Decode payload
	var lwtMsg LWTMessage
	if err := json.Unmarshal(msg.Payload, &lwtMsg); err != nil {
		t.Fatalf("Failed to unmarshal LWT payload: %v", err)
	}
	
	if lwtMsg.Status != "offline" {
		t.Errorf("Expected status offline, got %s", lwtMsg.Status)
	}
	
	if lwtMsg.Message != "Device disconnected unexpectedly" {
		t.Errorf("Expected message 'Device disconnected unexpectedly', got %s", lwtMsg.Message)
	}
}

func TestDecode(t *testing.T) {
	codec := NewCodec()
	
	tests := []struct {
		name           string
		topic          string
		payload        string
		expectedType   MessageType
		expectError    bool
	}{
		{
			name:         "State topic",
			topic:        "rtk/v1/demo/office/00:11:22:33:44:55/state",
			payload:      `{"status": "online", "health": "healthy"}`,
			expectedType: MessageTypeState,
			expectError:  false,
		},
		{
			name:         "Telemetry topic",
			topic:        "rtk/v1/demo/office/00:11:22:33:44:55/telemetry/temperature",
			payload:      `{"metric": "temperature", "value": 23.5}`,
			expectedType: MessageTypeTelemetry,
			expectError:  false,
		},
		{
			name:         "Event topic",
			topic:        "rtk/v1/demo/office/sensor001/evt/error",
			payload:      `{"event_id": "001", "type": "error", "level": "error"}`,
			expectedType: MessageTypeEvent,
			expectError:  false,
		},
		{
			name:         "Invalid topic",
			topic:        "other/v1/demo/office/sensor001/state",
			payload:      `{}`,
			expectedType: "",
			expectError:  true,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			msg, err := codec.Decode(ctx, test.topic, []byte(test.payload))
			
			if test.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if msg.MessageType != test.expectedType {
				t.Errorf("Expected message type %s, got %s", test.expectedType, msg.MessageType)
			}
			
			if msg.Topic != test.topic {
				t.Errorf("Expected topic %s, got %s", test.topic, msg.Topic)
			}
		})
	}
}

func TestDecodeCommand(t *testing.T) {
	codec := NewCodec()
	
	// Create a command message
	cmdMsg := CommandMessage{
		CommandID: "cmd_123",
		Type:      "system",
		Action:    "restart",
		Params: map[string]interface{}{
			"graceful": true,
		},
		Timeout:   30,
		Timestamp: time.Now(),
	}
	
	payload, _ := json.Marshal(cmdMsg)
	
	rtkMsg := &RTKMessage{
		MessageType: MessageTypeCommand,
		Payload:     payload,
	}
	
	ctx := context.Background()
	cmd, err := codec.DecodeCommand(ctx, rtkMsg)
	
	if err != nil {
		t.Fatalf("Failed to decode command: %v", err)
	}
	
	if cmd.ID != cmdMsg.CommandID {
		t.Errorf("Expected command ID %s, got %s", cmdMsg.CommandID, cmd.ID)
	}
	
	if cmd.Type != cmdMsg.Type {
		t.Errorf("Expected type %s, got %s", cmdMsg.Type, cmd.Type)
	}
	
	if cmd.Action != cmdMsg.Action {
		t.Errorf("Expected action %s, got %s", cmdMsg.Action, cmd.Action)
	}
	
	if cmd.Timeout != time.Duration(cmdMsg.Timeout)*time.Second {
		t.Errorf("Expected timeout %v, got %v", time.Duration(cmdMsg.Timeout)*time.Second, cmd.Timeout)
	}
}

func TestDecodeState(t *testing.T) {
	codec := NewCodec()
	
	stateMsg := StateMessage{
		Status:    "online",
		Health:    "healthy",
		Uptime:    3600, // 1 hour in seconds
		Timestamp: time.Now(),
		Properties: map[string]interface{}{
			"version": "1.0.0",
		},
	}
	
	payload, _ := json.Marshal(stateMsg)
	
	rtkMsg := &RTKMessage{
		MessageType: MessageTypeState,
		Payload:     payload,
	}
	
	ctx := context.Background()
	state, err := codec.DecodeState(ctx, rtkMsg)
	
	if err != nil {
		t.Fatalf("Failed to decode state: %v", err)
	}
	
	if state.Status != stateMsg.Status {
		t.Errorf("Expected status %s, got %s", stateMsg.Status, state.Status)
	}
	
	if state.Health != stateMsg.Health {
		t.Errorf("Expected health %s, got %s", stateMsg.Health, state.Health)
	}
	
	if state.Uptime != time.Duration(stateMsg.Uptime)*time.Second {
		t.Errorf("Expected uptime %v, got %v", time.Duration(stateMsg.Uptime)*time.Second, state.Uptime)
	}
}

func TestGenerateMessageID(t *testing.T) {
	id1 := generateMessageID()
	id2 := generateMessageID()
	
	if id1 == id2 {
		t.Error("Expected different message IDs")
	}
	
	if len(id1) == 0 {
		t.Error("Expected non-empty message ID")
	}
	
	// Check format
	if len(id1) < 4 || id1[:4] != "msg_" {
		t.Errorf("Expected message ID to start with 'msg_', got %s", id1)
	}
}

func TestEncodeWithoutDeviceInfo(t *testing.T) {
	codec := NewCodec()
	// Don't set device info
	
	state := &device.State{
		Status:    "online",
		Health:    "healthy",
		Timestamp: time.Now(),
	}
	
	ctx := context.Background()
	_, err := codec.EncodeState(ctx, state)
	
	if err == nil {
		t.Error("Expected error when device info not set")
	}
	
	expectedError := "device info not set"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}
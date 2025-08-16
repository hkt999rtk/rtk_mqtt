package main

/*
#include <stdlib.h>
#include <stdint.h>

// RTK MQTT Framework DLL Error Codes
#define RTK_SUCCESS             0
#define RTK_ERROR_INVALID_PARAM -1
#define RTK_ERROR_MEMORY        -2
#define RTK_ERROR_NOT_FOUND     -3
#define RTK_ERROR_CONNECTION    -4
#define RTK_ERROR_TIMEOUT       -5
#define RTK_ERROR_AUTH          -6

// Device States
#define RTK_DEVICE_STATE_OFFLINE    0
#define RTK_DEVICE_STATE_ONLINE     1
#define RTK_DEVICE_STATE_ERROR      2
#define RTK_DEVICE_STATE_CONNECTING 3

// Message Types
#define RTK_MSG_TYPE_STATE      1
#define RTK_MSG_TYPE_TELEMETRY  2
#define RTK_MSG_TYPE_EVENT      3
#define RTK_MSG_TYPE_COMMAND    4
#define RTK_MSG_TYPE_ATTRIBUTE  5

// Device Info Structure
typedef struct {
    char id[64];
    char device_type[32];  // renamed from 'type' to avoid Go keyword conflict
    char name[128];
    char version[16];
    char manufacturer[64];
} rtk_device_info_t;

// Device State Structure
typedef struct {
    char status[32];
    char health[32];
    long uptime;
    long last_seen;
    char* properties_json;
} rtk_device_state_t;

// Telemetry Data Structure
typedef struct {
    char metric[64];
    double value;
    char unit[16];
    long timestamp;
    char* labels_json;
} rtk_telemetry_data_t;

// Event Structure
typedef struct {
    char id[64];
    char event_type[64];  // renamed from 'type' to avoid Go keyword conflict
    char level[16];
    char message[256];
    long timestamp;
    char* data_json;
} rtk_event_t;

// Command Structure
typedef struct {
    char id[64];
    char action[64];
    char* params_json;
    long timestamp;
} rtk_command_t;

// Command Response Structure
typedef struct {
    char command_id[64];
    char status[32];
    char* result_json;
    char error_message[256];
    long timestamp;
} rtk_command_response_t;

// MQTT Configuration Structure
typedef struct {
    char broker_host[256];
    int broker_port;
    char client_id[64];
    char username[64];
    char password[64];
    int keep_alive;
    int clean_session;
    int qos;
    int retain;
    char* ca_cert_path;
    char* client_cert_path;
    char* client_key_path;
} rtk_mqtt_config_t;

// Device Configuration Structure
typedef struct {
    char device_id[64];
    char device_type[32];
    char tenant[32];
    char site[32];
    int telemetry_interval;
    int state_interval;
    int heartbeat_interval;
} rtk_device_config_t;
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	"unsafe"

	"github.com/rtk/mqtt-framework/pkg/codec"
	"github.com/rtk/mqtt-framework/pkg/device"
	"github.com/rtk/mqtt-framework/pkg/mqtt"
	"github.com/rtk/mqtt-framework/pkg/topic"
)

// Global variables for DLL state management
var (
	clients    = make(map[uintptr]*RTKClient)
	clientsMux sync.RWMutex
	nextID     uintptr = 1
)

// RTKClient wraps the Go MQTT framework for C interface
type RTKClient struct {
	ctx        context.Context
	cancel     context.CancelFunc
	client     mqtt.Client
	manager    *device.Manager
	config     *RTKConfig
	deviceInfo *device.Info
}

// RTKConfig holds the complete configuration
type RTKConfig struct {
	MQTT   *MQTTConfig   `json:"mqtt"`
	Device *DeviceConfig `json:"device"`
}

type MQTTConfig struct {
	BrokerHost     string `json:"broker_host"`
	BrokerPort     int    `json:"broker_port"`
	ClientID       string `json:"client_id"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	KeepAlive      int    `json:"keep_alive"`
	CleanSession   bool   `json:"clean_session"`
	QoS            int    `json:"qos"`
	Retain         bool   `json:"retain"`
	CACertPath     string `json:"ca_cert_path,omitempty"`
	ClientCertPath string `json:"client_cert_path,omitempty"`
	ClientKeyPath  string `json:"client_key_path,omitempty"`
}

type DeviceConfig struct {
	DeviceID          string `json:"device_id"`
	DeviceType        string `json:"device_type"`
	Tenant            string `json:"tenant"`
	Site              string `json:"site"`
	TelemetryInterval int    `json:"telemetry_interval"`
	StateInterval     int    `json:"state_interval"`
	HeartbeatInterval int    `json:"heartbeat_interval"`
}

// Helper functions for C string conversion
func cString(s string) *C.char {
	return C.CString(s)
}

func goString(cs *C.char) string {
	if cs == nil {
		return ""
	}
	return C.GoString(cs)
}

func freeCString(cs *C.char) {
	if cs != nil {
		C.free(unsafe.Pointer(cs))
	}
}

//export rtk_create_client
func rtk_create_client() uintptr {
	clientsMux.Lock()
	defer clientsMux.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	client := &RTKClient{
		ctx:    ctx,
		cancel: cancel,
	}

	id := nextID
	nextID++
	clients[id] = client

	return id
}

//export rtk_destroy_client
func rtk_destroy_client(clientID uintptr) C.int {
	clientsMux.Lock()
	defer clientsMux.Unlock()

	client, exists := clients[clientID]
	if !exists {
		return C.RTK_ERROR_NOT_FOUND
	}

	// Cleanup client resources
	if client.cancel != nil {
		client.cancel()
	}
	if client.client != nil {
		client.client.Disconnect()
	}

	delete(clients, clientID)
	return C.RTK_SUCCESS
}

//export rtk_configure_mqtt
func rtk_configure_mqtt(clientID uintptr, config *C.rtk_mqtt_config_t) C.int {
	clientsMux.RLock()
	client, exists := clients[clientID]
	clientsMux.RUnlock()

	if !exists {
		return C.RTK_ERROR_NOT_FOUND
	}
	if config == nil {
		return C.RTK_ERROR_INVALID_PARAM
	}

	// Convert C config to Go config
	mqttConfig := &MQTTConfig{
		BrokerHost:     goString(config.broker_host),
		BrokerPort:     int(config.broker_port),
		ClientID:       goString(config.client_id),
		Username:       goString(config.username),
		Password:       goString(config.password),
		KeepAlive:      int(config.keep_alive),
		CleanSession:   config.clean_session != 0,
		QoS:            int(config.qos),
		Retain:         config.retain != 0,
		CACertPath:     goString(config.ca_cert_path),
		ClientCertPath: goString(config.client_cert_path),
		ClientKeyPath:  goString(config.client_key_path),
	}

	if client.config == nil {
		client.config = &RTKConfig{}
	}
	client.config.MQTT = mqttConfig

	return C.RTK_SUCCESS
}

//export rtk_configure_device
func rtk_configure_device(clientID uintptr, config *C.rtk_device_config_t) C.int {
	clientsMux.RLock()
	client, exists := clients[clientID]
	clientsMux.RUnlock()

	if !exists {
		return C.RTK_ERROR_NOT_FOUND
	}
	if config == nil {
		return C.RTK_ERROR_INVALID_PARAM
	}

	// Convert C config to Go config
	deviceConfig := &DeviceConfig{
		DeviceID:          goString(config.device_id),
		DeviceType:        goString(config.device_type),
		Tenant:            goString(config.tenant),
		Site:              goString(config.site),
		TelemetryInterval: int(config.telemetry_interval),
		StateInterval:     int(config.state_interval),
		HeartbeatInterval: int(config.heartbeat_interval),
	}

	if client.config == nil {
		client.config = &RTKConfig{}
	}
	client.config.Device = deviceConfig

	return C.RTK_SUCCESS
}

//export rtk_connect
func rtk_connect(clientID uintptr) C.int {
	clientsMux.RLock()
	client, exists := clients[clientID]
	clientsMux.RUnlock()

	if !exists {
		return C.RTK_ERROR_NOT_FOUND
	}
	if client.config == nil || client.config.MQTT == nil {
		return C.RTK_ERROR_INVALID_PARAM
	}

	// Create MQTT client
	mqttClient, err := mqtt.NewClient("paho", &mqtt.Config{
		BrokerURL: fmt.Sprintf("tcp://%s:%d", 
			client.config.MQTT.BrokerHost, 
			client.config.MQTT.BrokerPort),
		ClientID:     client.config.MQTT.ClientID,
		Username:     client.config.MQTT.Username,
		Password:     client.config.MQTT.Password,
		KeepAlive:    time.Duration(client.config.MQTT.KeepAlive) * time.Second,
		CleanSession: client.config.MQTT.CleanSession,
		QoS:          byte(client.config.MQTT.QoS),
		Retain:       client.config.MQTT.Retain,
	})
	if err != nil {
		log.Printf("Failed to create MQTT client: %v", err)
		return C.RTK_ERROR_CONNECTION
	}

	// Connect to broker
	err = mqttClient.Connect(client.ctx)
	if err != nil {
		log.Printf("Failed to connect to MQTT broker: %v", err)
		return C.RTK_ERROR_CONNECTION
	}

	client.client = mqttClient

	// Create device manager
	client.manager = device.NewManager(mqttClient)

	return C.RTK_SUCCESS
}

//export rtk_disconnect
func rtk_disconnect(clientID uintptr) C.int {
	clientsMux.RLock()
	client, exists := clients[clientID]
	clientsMux.RUnlock()

	if !exists {
		return C.RTK_ERROR_NOT_FOUND
	}

	if client.client != nil {
		client.client.Disconnect()
		client.client = nil
	}

	return C.RTK_SUCCESS
}

//export rtk_set_device_info
func rtk_set_device_info(clientID uintptr, info *C.rtk_device_info_t) C.int {
	clientsMux.RLock()
	client, exists := clients[clientID]
	clientsMux.RUnlock()

	if !exists {
		return C.RTK_ERROR_NOT_FOUND
	}
	if info == nil {
		return C.RTK_ERROR_INVALID_PARAM
	}

	// Convert C info to Go info
	client.deviceInfo = &device.Info{
		ID:           goString((*C.char)(&info.id[0])),
		Type:         goString((*C.char)(&info.device_type[0])),
		Name:         goString((*C.char)(&info.name[0])),
		Version:      goString((*C.char)(&info.version[0])),
		Manufacturer: goString((*C.char)(&info.manufacturer[0])),
	}

	return C.RTK_SUCCESS
}

//export rtk_publish_state
func rtk_publish_state(clientID uintptr, state *C.rtk_device_state_t) C.int {
	clientsMux.RLock()
	client, exists := clients[clientID]
	clientsMux.RUnlock()

	if !exists {
		return C.RTK_ERROR_NOT_FOUND
	}
	if state == nil || client.client == nil || client.config == nil {
		return C.RTK_ERROR_INVALID_PARAM
	}

	// Convert C state to Go state
	goState := &device.State{
		Status:    goString(state.status),
		Health:    goString(state.health),
		Uptime:    time.Duration(state.uptime) * time.Second,
		LastSeen:  time.Unix(state.last_seen, 0),
		Timestamp: time.Now(),
	}

	// Parse properties JSON if provided
	if state.properties_json != nil {
		propertiesStr := goString(state.properties_json)
		if propertiesStr != "" {
			var properties map[string]interface{}
			if err := json.Unmarshal([]byte(propertiesStr), &properties); err == nil {
				goState.Properties = properties
			}
		}
	}

	// Build topic
	topicBuilder := topic.NewBuilder()
	stateTopic := topicBuilder.State(
		client.config.Device.Tenant,
		client.config.Device.Site,
		client.config.Device.DeviceID,
	)

	// Encode message
	encoder := codec.NewMessageEncoder()
	payload, err := encoder.EncodeState(goState)
	if err != nil {
		log.Printf("Failed to encode state: %v", err)
		return C.RTK_ERROR_INVALID_PARAM
	}

	// Publish message
	err = client.client.Publish(client.ctx, stateTopic, payload, byte(client.config.MQTT.QoS), client.config.MQTT.Retain)
	if err != nil {
		log.Printf("Failed to publish state: %v", err)
		return C.RTK_ERROR_CONNECTION
	}

	return C.RTK_SUCCESS
}

//export rtk_publish_telemetry
func rtk_publish_telemetry(clientID uintptr, telemetry *C.rtk_telemetry_data_t) C.int {
	clientsMux.RLock()
	client, exists := clients[clientID]
	clientsMux.RUnlock()

	if !exists {
		return C.RTK_ERROR_NOT_FOUND
	}
	if telemetry == nil || client.client == nil || client.config == nil {
		return C.RTK_ERROR_INVALID_PARAM
	}

	// Convert C telemetry to Go telemetry
	goTelemetry := &device.TelemetryData{
		Metric:    goString(telemetry.metric),
		Value:     float64(telemetry.value),
		Unit:      goString(telemetry.unit),
		Timestamp: time.Unix(telemetry.timestamp, 0),
	}

	// Parse labels JSON if provided
	if telemetry.labels_json != nil {
		labelsStr := goString(telemetry.labels_json)
		if labelsStr != "" {
			var labels map[string]string
			if err := json.Unmarshal([]byte(labelsStr), &labels); err == nil {
				goTelemetry.Labels = labels
			}
		}
	}

	// Build topic
	topicBuilder := topic.NewBuilder()
	telemetryTopic := topicBuilder.Telemetry(
		client.config.Device.Tenant,
		client.config.Device.Site,
		client.config.Device.DeviceID,
		goTelemetry.Metric,
	)

	// Encode message
	encoder := codec.NewMessageEncoder()
	payload, err := encoder.EncodeTelemetry(goTelemetry)
	if err != nil {
		log.Printf("Failed to encode telemetry: %v", err)
		return C.RTK_ERROR_INVALID_PARAM
	}

	// Publish message
	err = client.client.Publish(client.ctx, telemetryTopic, payload, byte(client.config.MQTT.QoS), false)
	if err != nil {
		log.Printf("Failed to publish telemetry: %v", err)
		return C.RTK_ERROR_CONNECTION
	}

	return C.RTK_SUCCESS
}

//export rtk_publish_event
func rtk_publish_event(clientID uintptr, event *C.rtk_event_t) C.int {
	clientsMux.RLock()
	client, exists := clients[clientID]
	clientsMux.RUnlock()

	if !exists {
		return C.RTK_ERROR_NOT_FOUND
	}
	if event == nil || client.client == nil || client.config == nil {
		return C.RTK_ERROR_INVALID_PARAM
	}

	// Convert C event to Go event
	goEvent := &device.Event{
		ID:        goString((*C.char)(&event.id[0])),
		Type:      goString((*C.char)(&event.event_type[0])),
		Level:     goString((*C.char)(&event.level[0])),
		Message:   goString((*C.char)(&event.message[0])),
		Timestamp: time.Unix(event.timestamp, 0),
	}

	// Parse data JSON if provided
	if event.data_json != nil {
		dataStr := goString(event.data_json)
		if dataStr != "" {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(dataStr), &data); err == nil {
				goEvent.Data = data
			}
		}
	}

	// Build topic
	topicBuilder := topic.NewBuilder()
	eventTopic := topicBuilder.Event(
		client.config.Device.Tenant,
		client.config.Device.Site,
		client.config.Device.DeviceID,
		goEvent.Type,
	)

	// Encode message
	encoder := codec.NewMessageEncoder()
	payload, err := encoder.EncodeEvent(goEvent)
	if err != nil {
		log.Printf("Failed to encode event: %v", err)
		return C.RTK_ERROR_INVALID_PARAM
	}

	// Publish message
	err = client.client.Publish(client.ctx, eventTopic, payload, byte(client.config.MQTT.QoS), false)
	if err != nil {
		log.Printf("Failed to publish event: %v", err)
		return C.RTK_ERROR_CONNECTION
	}

	return C.RTK_SUCCESS
}

//export rtk_get_version
func rtk_get_version() *C.char {
	return cString("1.0.0")
}

//export rtk_get_last_error
func rtk_get_last_error() *C.char {
	// TODO: Implement error tracking
	return cString("No error")
}

// Main function required for building as shared library
func main() {
	// This function is required but not used when building as DLL
}
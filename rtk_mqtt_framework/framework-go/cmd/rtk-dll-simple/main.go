package main

/*
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

// RTK MQTT Framework DLL Error Codes
#define RTK_SUCCESS             0
#define RTK_ERROR_INVALID_PARAM -1
#define RTK_ERROR_MEMORY        -2
#define RTK_ERROR_NOT_FOUND     -3
#define RTK_ERROR_CONNECTION    -4

// Simple Device Info Structure
typedef struct {
    char id[64];
    char device_type[32];
    char name[128];
    char version[16];
} rtk_device_info_t;

// Simple Device State Structure
typedef struct {
    char status[32];
    char health[32];
    int64_t uptime;
    int64_t last_seen;
} rtk_device_state_t;

// Simple MQTT Configuration
typedef struct {
    char broker_host[256];
    int broker_port;
    char client_id[64];
} rtk_mqtt_config_t;
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

// Global variables for DLL state management
var (
	clients    = make(map[uintptr]*RTKClient)
	clientsMux sync.RWMutex
	nextID     uintptr = 1
)

// RTKClient simple wrapper
type RTKClient struct {
	ID         uintptr
	BrokerHost string
	BrokerPort int
	ClientID   string
	DeviceInfo *DeviceInfo
	IsConnected bool
}

type DeviceInfo struct {
	ID         string
	DeviceType string
	Name       string
	Version    string
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

func goStringFromCharArray(arr []C.char) string {
	// Convert C char array to Go string
	bytes := make([]byte, 0, len(arr))
	for _, c := range arr {
		if c == 0 {
			break
		}
		bytes = append(bytes, byte(c))
	}
	return string(bytes)
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

	client := &RTKClient{
		ID: nextID,
	}

	clients[nextID] = client
	id := nextID
	nextID++

	fmt.Printf("Created client: %d\n", id)
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

	delete(clients, clientID)
	fmt.Printf("Destroyed client: %d\n", client.ID)
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

	// Convert C config to Go config using char arrays
	client.BrokerHost = goStringFromCharArray(config.broker_host[:])
	client.BrokerPort = int(config.broker_port)
	client.ClientID = goStringFromCharArray(config.client_id[:])

	fmt.Printf("Configured MQTT for client %d: %s:%d (ID: %s)\n", 
		clientID, client.BrokerHost, client.BrokerPort, client.ClientID)

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

	// Convert C info to Go info using char arrays
	client.DeviceInfo = &DeviceInfo{
		ID:         goStringFromCharArray(info.id[:]),
		DeviceType: goStringFromCharArray(info.device_type[:]),
		Name:       goStringFromCharArray(info.name[:]),
		Version:    goStringFromCharArray(info.version[:]),
	}

	fmt.Printf("Set device info for client %d: %s (%s)\n", 
		clientID, client.DeviceInfo.Name, client.DeviceInfo.ID)

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

	// Simulate connection
	client.IsConnected = true
	fmt.Printf("Connected client %d to %s:%d\n", 
		clientID, client.BrokerHost, client.BrokerPort)

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

	client.IsConnected = false
	fmt.Printf("Disconnected client %d\n", clientID)

	return C.RTK_SUCCESS
}

//export rtk_publish_state
func rtk_publish_state(clientID uintptr, state *C.rtk_device_state_t) C.int {
	clientsMux.RLock()
	_, exists := clients[clientID]
	clientsMux.RUnlock()

	if !exists {
		return C.RTK_ERROR_NOT_FOUND
	}
	if state == nil {
		return C.RTK_ERROR_INVALID_PARAM
	}

	status := goStringFromCharArray(state.status[:])
	health := goStringFromCharArray(state.health[:])

	fmt.Printf("Published state for client %d: status=%s, health=%s, uptime=%d\n", 
		clientID, status, health, state.uptime)

	return C.RTK_SUCCESS
}

//export rtk_get_version
func rtk_get_version() *C.char {
	return cString("1.0.0-dll")
}

//export rtk_get_last_error
func rtk_get_last_error() *C.char {
	return cString("No error")
}

//export rtk_is_connected
func rtk_is_connected(clientID uintptr) C.int {
	clientsMux.RLock()
	client, exists := clients[clientID]
	clientsMux.RUnlock()

	if !exists {
		return 0
	}

	if client.IsConnected {
		return 1
	}
	return 0
}

//export rtk_get_client_count
func rtk_get_client_count() C.int {
	clientsMux.RLock()
	defer clientsMux.RUnlock()
	return C.int(len(clients))
}

// Main function required for building as shared library
func main() {
	// This function is required but not used when building as DLL
	fmt.Println("RTK MQTT Framework DLL - should not be called directly")
}
#ifndef RTK_MQTT_FRAMEWORK_H
#define RTK_MQTT_FRAMEWORK_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>

// === RTK MQTT Framework DLL Interface ===
// Windows DLL exports for RTK MQTT Framework Go implementation

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

// === Data Structures ===

// Device Info Structure
typedef struct {
    char id[64];          // Device MAC address (e.g., "00:11:22:33:44:55")
    char device_type[32]; // Device type (e.g., "wifi_router", "iot_sensor")
    char name[128];       // Human-readable device name
    char version[16];     // Device firmware/software version
    char manufacturer[64]; // Device manufacturer
} rtk_device_info_t;

// Device State Structure
typedef struct {
    char status[32];      // Device status: "online", "offline", "error"
    char health[32];      // Device health: "healthy", "warning", "critical"
    int64_t uptime;       // Device uptime in seconds
    int64_t last_seen;    // Last seen timestamp (Unix timestamp)
    char* properties_json; // Additional properties as JSON string (optional, can be NULL)
} rtk_device_state_t;

// Telemetry Data Structure
typedef struct {
    char metric[64];      // Metric name (e.g., "temperature", "cpu_usage")
    double value;         // Metric value
    char unit[16];        // Unit of measurement (e.g., "°C", "%", "MB")
    int64_t timestamp;    // Measurement timestamp (Unix timestamp)
    char* labels_json;    // Additional labels as JSON string (optional, can be NULL)
} rtk_telemetry_data_t;

// Event Structure
typedef struct {
    char id[64];          // Unique event ID
    char event_type[64];  // Event type (e.g., "wifi.roam_miss", "system.restart")
    char level[16];       // Event level: "info", "warning", "error", "critical"
    char message[256];    // Human-readable event message
    int64_t timestamp;    // Event timestamp (Unix timestamp)
    char* data_json;      // Additional event data as JSON string (optional, can be NULL)
} rtk_event_t;

// Command Structure
typedef struct {
    char id[64];          // Unique command ID
    char action[64];      // Command action (e.g., "restart", "get_config")
    char* params_json;    // Command parameters as JSON string (optional, can be NULL)
    int64_t timestamp;    // Command timestamp (Unix timestamp)
} rtk_command_t;

// Command Response Structure
typedef struct {
    char command_id[64];  // Original command ID
    char status[32];      // Response status: "success", "error", "timeout"
    char* result_json;    // Command result as JSON string (optional, can be NULL)
    char error_message[256]; // Error message if status is "error"
    int64_t timestamp;    // Response timestamp (Unix timestamp)
} rtk_command_response_t;

// MQTT Configuration Structure
typedef struct {
    char broker_host[256]; // MQTT broker hostname or IP
    int broker_port;       // MQTT broker port (default: 1883 for TCP, 8883 for TLS)
    char client_id[64];    // MQTT client ID
    char username[64];     // MQTT username (optional, can be empty)
    char password[64];     // MQTT password (optional, can be empty)
    int keep_alive;        // Keep alive interval in seconds (default: 60)
    int clean_session;     // Clean session flag (0 = false, 1 = true)
    int qos;              // Quality of Service level (0, 1, or 2)
    int retain;           // Retain flag for published messages (0 = false, 1 = true)
    char* ca_cert_path;   // CA certificate file path for TLS (optional, can be NULL)
    char* client_cert_path; // Client certificate file path for TLS (optional, can be NULL)
    char* client_key_path;  // Client private key file path for TLS (optional, can be NULL)
} rtk_mqtt_config_t;

// Device Configuration Structure
typedef struct {
    char device_id[64];   // Device ID (MAC address format)
    char device_type[32]; // Device type
    char tenant[32];      // Tenant identifier
    char site[32];        // Site identifier
    int telemetry_interval; // Telemetry publishing interval in seconds
    int state_interval;     // State publishing interval in seconds
    int heartbeat_interval; // Heartbeat interval in seconds
} rtk_device_config_t;

// === Client Handle Type ===
typedef uintptr_t rtk_client_handle_t;

// === DLL Export Functions ===

#ifdef _WIN32
#define RTK_EXPORT __declspec(dllexport)
#else
#define RTK_EXPORT __attribute__((visibility("default")))
#endif

// === Client Management ===

/**
 * @brief Create a new RTK MQTT client instance
 * @return Client handle on success, 0 on failure
 */
RTK_EXPORT rtk_client_handle_t rtk_create_client(void);

/**
 * @brief Destroy an RTK MQTT client instance
 * @param client_id Client handle returned by rtk_create_client()
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_destroy_client(rtk_client_handle_t client_id);

// === Configuration ===

/**
 * @brief Configure MQTT connection parameters
 * @param client_id Client handle
 * @param config MQTT configuration structure
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_configure_mqtt(rtk_client_handle_t client_id, const rtk_mqtt_config_t* config);

/**
 * @brief Configure device parameters
 * @param client_id Client handle
 * @param config Device configuration structure
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_configure_device(rtk_client_handle_t client_id, const rtk_device_config_t* config);

// === Connection Management ===

/**
 * @brief Connect to MQTT broker
 * @param client_id Client handle
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_connect(rtk_client_handle_t client_id);

/**
 * @brief Disconnect from MQTT broker
 * @param client_id Client handle
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_disconnect(rtk_client_handle_t client_id);

// === Device Information ===

/**
 * @brief Set device information
 * @param client_id Client handle
 * @param info Device information structure
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_set_device_info(rtk_client_handle_t client_id, const rtk_device_info_t* info);

// === Message Publishing ===

/**
 * @brief Publish device state
 * @param client_id Client handle
 * @param state Device state structure
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_publish_state(rtk_client_handle_t client_id, const rtk_device_state_t* state);

/**
 * @brief Publish telemetry data
 * @param client_id Client handle
 * @param telemetry Telemetry data structure
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_publish_telemetry(rtk_client_handle_t client_id, const rtk_telemetry_data_t* telemetry);

/**
 * @brief Publish event
 * @param client_id Client handle
 * @param event Event structure
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_publish_event(rtk_client_handle_t client_id, const rtk_event_t* event);

// === Utility Functions ===

/**
 * @brief Get RTK MQTT Framework version
 * @return Version string (caller should not free this string)
 */
RTK_EXPORT const char* rtk_get_version(void);

/**
 * @brief Get last error message
 * @return Error message string (caller should not free this string)
 */
RTK_EXPORT const char* rtk_get_last_error(void);

// === Usage Example ===
/*

// Example usage in C/C++:

#include "rtk_mqtt_framework.h"
#include <stdio.h>
#include <string.h>
#include <time.h>

int main() {
    // Create client
    rtk_client_handle_t client = rtk_create_client();
    if (client == 0) {
        printf("Failed to create client\n");
        return -1;
    }

    // Configure MQTT
    rtk_mqtt_config_t mqtt_config = {0};
    strcpy(mqtt_config.broker_host, "mqtt.eclipse.org");
    mqtt_config.broker_port = 1883;
    strcpy(mqtt_config.client_id, "rtk_test_client");
    mqtt_config.keep_alive = 60;
    mqtt_config.clean_session = 1;
    mqtt_config.qos = 1;
    mqtt_config.retain = 0;

    int result = rtk_configure_mqtt(client, &mqtt_config);
    if (result != RTK_SUCCESS) {
        printf("Failed to configure MQTT: %d\n", result);
        rtk_destroy_client(client);
        return -1;
    }

    // Configure device
    rtk_device_config_t device_config = {0};
    strcpy(device_config.device_id, "00:11:22:33:44:55");
    strcpy(device_config.device_type, "test_device");
    strcpy(device_config.tenant, "test_tenant");
    strcpy(device_config.site, "test_site");
    device_config.telemetry_interval = 60;
    device_config.state_interval = 30;
    device_config.heartbeat_interval = 10;

    result = rtk_configure_device(client, &device_config);
    if (result != RTK_SUCCESS) {
        printf("Failed to configure device: %d\n", result);
        rtk_destroy_client(client);
        return -1;
    }

    // Set device info
    rtk_device_info_t device_info = {0};
    strcpy(device_info.id, "00:11:22:33:44:55");
    strcpy(device_info.device_type, "test_device");
    strcpy(device_info.name, "Test Device");
    strcpy(device_info.version, "1.0.0");
    strcpy(device_info.manufacturer, "Test Corp");

    result = rtk_set_device_info(client, &device_info);
    if (result != RTK_SUCCESS) {
        printf("Failed to set device info: %d\n", result);
        rtk_destroy_client(client);
        return -1;
    }

    // Connect
    result = rtk_connect(client);
    if (result != RTK_SUCCESS) {
        printf("Failed to connect: %d\n", result);
        rtk_destroy_client(client);
        return -1;
    }

    printf("Connected successfully!\n");

    // Publish state
    rtk_device_state_t state = {0};
    strcpy(state.status, "online");
    strcpy(state.health, "healthy");
    state.uptime = 3600; // 1 hour
    state.last_seen = time(NULL);
    state.properties_json = "{\"version\":\"1.0.0\",\"location\":\"test_lab\"}";

    result = rtk_publish_state(client, &state);
    if (result == RTK_SUCCESS) {
        printf("State published successfully!\n");
    }

    // Publish telemetry
    rtk_telemetry_data_t telemetry = {0};
    strcpy(telemetry.metric, "temperature");
    telemetry.value = 25.5;
    strcpy(telemetry.unit, "°C");
    telemetry.timestamp = time(NULL);
    telemetry.labels_json = "{\"sensor\":\"ds18b20\",\"location\":\"cpu\"}";

    result = rtk_publish_telemetry(client, &telemetry);
    if (result == RTK_SUCCESS) {
        printf("Telemetry published successfully!\n");
    }

    // Publish event
    rtk_event_t event = {0};
    strcpy(event.id, "evt_001");
    strcpy(event.event_type, "system.startup");
    strcpy(event.level, "info");
    strcpy(event.message, "Device started successfully");
    event.timestamp = time(NULL);
    event.data_json = "{\"boot_time\":3.2,\"memory_usage\":45}";

    result = rtk_publish_event(client, &event);
    if (result == RTK_SUCCESS) {
        printf("Event published successfully!\n");
    }

    // Disconnect and cleanup
    rtk_disconnect(client);
    rtk_destroy_client(client);

    printf("Cleanup completed\n");
    return 0;
}

*/

#ifdef __cplusplus
}
#endif

#endif // RTK_MQTT_FRAMEWORK_H
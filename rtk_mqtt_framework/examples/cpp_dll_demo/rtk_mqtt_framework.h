#ifndef RTK_MQTT_FRAMEWORK_SIMPLE_H
#define RTK_MQTT_FRAMEWORK_SIMPLE_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>

// === RTK MQTT Framework Simple DLL Interface ===
// Simplified version for demonstration and testing

// RTK MQTT Framework DLL Error Codes
#define RTK_SUCCESS             0
#define RTK_ERROR_INVALID_PARAM -1
#define RTK_ERROR_MEMORY        -2
#define RTK_ERROR_NOT_FOUND     -3
#define RTK_ERROR_CONNECTION    -4

// === Simplified Data Structures ===

// Simple Device Info Structure
typedef struct {
    char id[64];          // Device MAC address (e.g., "00:11:22:33:44:55")
    char device_type[32]; // Device type (e.g., "wifi_router", "iot_sensor")
    char name[128];       // Human-readable device name
    char version[16];     // Device firmware/software version
} rtk_device_info_t;

// Simple Device State Structure
typedef struct {
    char status[32];      // Device status: "online", "offline", "error"
    char health[32];      // Device health: "healthy", "warning", "critical"
    int64_t uptime;       // Device uptime in seconds
    int64_t last_seen;    // Last seen timestamp (Unix timestamp)
} rtk_device_state_t;

// Simple MQTT Configuration
typedef struct {
    char broker_host[256]; // MQTT broker hostname or IP
    int broker_port;       // MQTT broker port (default: 1883)
    char client_id[64];    // MQTT client ID
} rtk_mqtt_config_t;

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
 * @brief Configure MQTT connection parameters (simplified)
 * @param client_id Client handle
 * @param config MQTT configuration structure
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_configure_mqtt(rtk_client_handle_t client_id, const rtk_mqtt_config_t* config);

/**
 * @brief Set device information
 * @param client_id Client handle
 * @param info Device information structure
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_set_device_info(rtk_client_handle_t client_id, const rtk_device_info_t* info);

// === Connection Management ===

/**
 * @brief Connect to MQTT broker (simulated)
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

// === Message Publishing ===

/**
 * @brief Publish device state (simulated)
 * @param client_id Client handle
 * @param state Device state structure
 * @return RTK_SUCCESS on success, error code on failure
 */
RTK_EXPORT int rtk_publish_state(rtk_client_handle_t client_id, const rtk_device_state_t* state);

// === Status and Utility Functions ===

/**
 * @brief Check if client is connected
 * @param client_id Client handle
 * @return 1 if connected, 0 if not connected or invalid client
 */
RTK_EXPORT int rtk_is_connected(rtk_client_handle_t client_id);

/**
 * @brief Get number of active clients
 * @return Number of active clients
 */
RTK_EXPORT int rtk_get_client_count(void);

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

#ifdef __cplusplus
}
#endif

#endif // RTK_MQTT_FRAMEWORK_SIMPLE_H
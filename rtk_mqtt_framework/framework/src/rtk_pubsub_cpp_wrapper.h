/**
 * @file rtk_pubsub_cpp_wrapper.h
 * @brief C++ wrapper header for PubSubClient C interface
 * 
 * This header defines the C interface for the C++-based PubSubClient
 * wrapper, allowing C code to use PubSubClient functionality.
 */

#ifndef RTK_PUBSUB_CPP_WRAPPER_H
#define RTK_PUBSUB_CPP_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>

// Use same error codes as the main header to avoid conflicts
// These will be mapped from C++ wrapper errors to RTK errors

// Message structure for callbacks (use different name to avoid conflicts)
typedef struct {
    char topic[256];
    uint8_t* payload;
    size_t payload_len;
    int qos;
    int retained;
} rtk_cpp_mqtt_message_t;

// Message callback function type
typedef void (*rtk_mqtt_message_callback_t)(const rtk_cpp_mqtt_message_t* message, void* user_data);

/**
 * @brief Initialize PubSubClient wrapper
 * 
 * @param broker_host MQTT broker hostname or IP address
 * @param broker_port MQTT broker port (usually 1883 or 8883)
 * @param client_id Unique client identifier
 * @return RTK_PUBSUB_SUCCESS on success, error code on failure
 */
int rtk_pubsub_cpp_init(const char* broker_host, uint16_t broker_port, const char* client_id);

/**
 * @brief Cleanup PubSubClient wrapper and free resources
 */
void rtk_pubsub_cpp_cleanup(void);

/**
 * @brief Connect to MQTT broker
 * 
 * @return RTK_PUBSUB_SUCCESS on success, error code on failure
 */
int rtk_pubsub_cpp_connect(void);

/**
 * @brief Disconnect from MQTT broker
 * 
 * @return RTK_PUBSUB_SUCCESS on success, error code on failure
 */
int rtk_pubsub_cpp_disconnect(void);

/**
 * @brief Check if connected to MQTT broker
 * 
 * @return 1 if connected, 0 if not connected
 */
int rtk_pubsub_cpp_is_connected(void);

/**
 * @brief Publish message to MQTT topic
 * 
 * @param topic MQTT topic to publish to
 * @param payload Message payload data
 * @param payload_len Length of payload data
 * @param qos Quality of Service level (0, 1, or 2)
 * @param retained Whether message should be retained by broker
 * @return RTK_PUBSUB_SUCCESS on success, error code on failure
 */
int rtk_pubsub_cpp_publish(const char* topic, const void* payload, size_t payload_len, 
                          int qos, int retained);

/**
 * @brief Subscribe to MQTT topic
 * 
 * @param topic MQTT topic to subscribe to
 * @param qos Quality of Service level (0, 1, or 2)
 * @return RTK_PUBSUB_SUCCESS on success, error code on failure
 */
int rtk_pubsub_cpp_subscribe(const char* topic, int qos);

/**
 * @brief Unsubscribe from MQTT topic
 * 
 * @param topic MQTT topic to unsubscribe from
 * @return RTK_PUBSUB_SUCCESS on success, error code on failure
 */
int rtk_pubsub_cpp_unsubscribe(const char* topic);

/**
 * @brief Process incoming MQTT messages (call regularly)
 * 
 * @return RTK_PUBSUB_SUCCESS on success, error code on failure
 */
int rtk_pubsub_cpp_loop(void);

/**
 * @brief Set MQTT authentication credentials
 * 
 * @param username MQTT username (can be NULL)
 * @param password MQTT password (can be NULL)
 * @return RTK_PUBSUB_SUCCESS on success, error code on failure
 */
int rtk_pubsub_cpp_set_credentials(const char* username, const char* password);

/**
 * @brief Set message callback function
 * 
 * @param callback Callback function to handle incoming messages
 * @param user_data User data to pass to callback function
 * @return RTK_PUBSUB_SUCCESS on success, error code on failure
 */
int rtk_pubsub_cpp_set_callback(rtk_mqtt_message_callback_t callback, void* user_data);

/**
 * @brief Get last error message
 * 
 * @return String describing the last error that occurred
 */
const char* rtk_pubsub_cpp_get_last_error(void);

/**
 * @brief Get wrapper version string
 * 
 * @return Version string of the wrapper
 */
const char* rtk_pubsub_cpp_get_version(void);

#ifdef __cplusplus
}
#endif

#endif // RTK_PUBSUB_CPP_WRAPPER_H
#include "rtk_mqtt_client.h"
#include "rtk_pubsub_adapter.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

/**
 * @file mqtt_client.c
 * @brief MQTT 客戶端後端管理器實作
 * 
 * 提供統一的 MQTT 操作介面，支援運行時後端切換
 */

// === 內部狀態管理 ===

#define MAX_BACKENDS 8
#define MAX_ERROR_MSG_LEN 256

static struct {
    const rtk_mqtt_backend_ops_t* backends[MAX_BACKENDS];
    int backend_count;
    
    const rtk_mqtt_backend_ops_t* current_backend;
    rtk_mqtt_config_t current_config;
    
    int is_initialized;
    int is_configured;
    
    char last_error[MAX_ERROR_MSG_LEN];
    rtk_mqtt_error_t last_error_code;
    
    rtk_mqtt_callback_t global_message_callback;
    rtk_mqtt_connection_callback_t global_connection_callback;
    void* global_user_data;
} mqtt_manager = { 0 };

// === 內部輔助函式 ===

static void set_last_error(rtk_mqtt_error_t code, const char* message) {
    mqtt_manager.last_error_code = code;
    if (message) {
        strncpy(mqtt_manager.last_error, message, MAX_ERROR_MSG_LEN - 1);
        mqtt_manager.last_error[MAX_ERROR_MSG_LEN - 1] = '\0';
    } else {
        strcpy(mqtt_manager.last_error, rtk_mqtt_get_error_string(code));
    }
}

static int find_backend_by_name(const char* name) {
    if (!name) return -1;
    
    for (int i = 0; i < mqtt_manager.backend_count; i++) {
        if (mqtt_manager.backends[i] && mqtt_manager.backends[i]->name &&
            strcmp(mqtt_manager.backends[i]->name, name) == 0) {
            return i;
        }
    }
    return -1;
}

static const rtk_mqtt_backend_ops_t* get_default_backend(rtk_mqtt_backend_type_t type) {
    // RTK Framework now uses PubSubClient exclusively
    (void)type; // Suppress unused parameter warning
    
    // Always return PubSubClient backend
    const rtk_mqtt_backend_ops_t* pubsub_backend = rtk_mqtt_find_backend("pubsub");
    if (pubsub_backend) {
        return pubsub_backend;
    }
    
    // Fallback to first available backend if PubSubClient not found
    return (mqtt_manager.backend_count > 0) ? mqtt_manager.backends[0] : NULL;
}

// === 公開 API 實作 ===

int rtk_mqtt_init(rtk_mqtt_backend_type_t backend_type) {
    if (mqtt_manager.is_initialized) {
        set_last_error(RTK_MQTT_SUCCESS, "Already initialized");
        return RTK_MQTT_SUCCESS;
    }
    
    // 初始化管理器狀態
    memset(&mqtt_manager, 0, sizeof(mqtt_manager));
    mqtt_manager.is_initialized = 1;  // 先設定初始化狀態
    
    // Register PubSubClient backend automatically
    extern int rtk_pubsub_register_mqtt_backend(void);
    rtk_pubsub_register_mqtt_backend();
    
    // 載入 PubSubClient 後端
    const rtk_mqtt_backend_ops_t* pubsub_backend = get_default_backend(backend_type);
    if (pubsub_backend) {
        mqtt_manager.current_backend = pubsub_backend;
    } else {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "PubSubClient backend not available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    set_last_error(RTK_MQTT_SUCCESS, "MQTT client manager initialized with PubSubClient");
    
    printf("[RTK-MQTT] Client manager initialized with PubSubClient backend\n");
    
    return RTK_MQTT_SUCCESS;
}

void rtk_mqtt_cleanup(void) {
    if (!mqtt_manager.is_initialized) {
        return;
    }
    
    // 如果有連線，先斷開
    if (mqtt_manager.current_backend && rtk_mqtt_is_connected()) {
        rtk_mqtt_disconnect();
    }
    
    // 清理當前後端
    if (mqtt_manager.current_backend && mqtt_manager.current_backend->cleanup) {
        mqtt_manager.current_backend->cleanup();
    }
    
    // 重置管理器狀態
    memset(&mqtt_manager, 0, sizeof(mqtt_manager));
    
    printf("[RTK-MQTT] Client manager cleaned up\n");
}

int rtk_mqtt_set_backend(const rtk_mqtt_backend_ops_t* ops) {
    if (!mqtt_manager.is_initialized) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Manager not initialized");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (!ops) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Backend operations is NULL");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    // 驗證後端操作介面的完整性
    if (!ops->connect || !ops->disconnect || !ops->publish || 
        !ops->subscribe || !ops->unsubscribe || !ops->is_connected) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Incomplete backend operations");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    // 如果當前有連線，需要先斷開
    if (mqtt_manager.current_backend && rtk_mqtt_is_connected()) {
        printf("[RTK-MQTT] Disconnecting current backend before switching\n");
        rtk_mqtt_disconnect();
    }
    
    // 清理當前後端
    if (mqtt_manager.current_backend && mqtt_manager.current_backend->cleanup) {
        mqtt_manager.current_backend->cleanup();
    }
    
    mqtt_manager.current_backend = ops;
    mqtt_manager.is_configured = 0;  // 需要重新配置
    
    printf("[RTK-MQTT] Switched to backend: %s v%s\n", 
           ops->name ? ops->name : "unknown", 
           ops->version ? ops->version : "unknown");
    
    set_last_error(RTK_MQTT_SUCCESS, "Backend switched successfully");
    return RTK_MQTT_SUCCESS;
}

const rtk_mqtt_backend_ops_t* rtk_mqtt_get_backend(void) {
    return mqtt_manager.current_backend;
}

int rtk_mqtt_register_backend(const char* name, const rtk_mqtt_backend_ops_t* ops) {
    if (!mqtt_manager.is_initialized) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Manager not initialized");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (!name || !ops) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Invalid parameters");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (mqtt_manager.backend_count >= MAX_BACKENDS) {
        set_last_error(RTK_MQTT_ERROR_MEMORY, "Too many backends registered");
        return RTK_MQTT_ERROR_MEMORY;
    }
    
    // 檢查是否已存在同名後端
    if (find_backend_by_name(name) >= 0) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Backend already registered");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    mqtt_manager.backends[mqtt_manager.backend_count] = ops;
    mqtt_manager.backend_count++;
    
    printf("[RTK-MQTT] Registered backend: %s\n", name);
    
    set_last_error(RTK_MQTT_SUCCESS, "Backend registered successfully");
    return RTK_MQTT_SUCCESS;
}

const rtk_mqtt_backend_ops_t* rtk_mqtt_find_backend(const char* name) {
    int index = find_backend_by_name(name);
    return (index >= 0) ? mqtt_manager.backends[index] : NULL;
}

// === 統一 MQTT 操作 API 實作 ===

int rtk_mqtt_configure(const rtk_mqtt_config_t* config) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "No backend available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    
    if (!config) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Config is NULL");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    // 驗證配置
    int validation_result = rtk_mqtt_validate_config(config);
    if (validation_result != RTK_MQTT_SUCCESS) {
        return validation_result;
    }
    
    // 保存配置
    memcpy(&mqtt_manager.current_config, config, sizeof(rtk_mqtt_config_t));
    
    // 初始化後端
    if (mqtt_manager.current_backend->init) {
        int ret = mqtt_manager.current_backend->init(config);
        if (ret != RTK_MQTT_SUCCESS) {
            set_last_error(ret, "Backend initialization failed");
            return ret;
        }
    }
    
    mqtt_manager.is_configured = 1;
    set_last_error(RTK_MQTT_SUCCESS, "MQTT client configured successfully");
    
    printf("[RTK-MQTT] Client configured for broker: %s:%d\n", 
           config->broker_host, config->broker_port);
    
    return RTK_MQTT_SUCCESS;
}

int rtk_mqtt_connect(void) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "No backend available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    
    if (!mqtt_manager.is_configured) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Client not configured");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (rtk_mqtt_is_connected()) {
        set_last_error(RTK_MQTT_ERROR_ALREADY_CONNECTED, "Already connected");
        return RTK_MQTT_ERROR_ALREADY_CONNECTED;
    }
    
    int ret = mqtt_manager.current_backend->connect();
    if (ret != RTK_MQTT_SUCCESS) {
        set_last_error(ret, "Connection failed");
        return ret;
    }
    
    set_last_error(RTK_MQTT_SUCCESS, "Connected successfully");
    printf("[RTK-MQTT] Connected to broker\n");
    
    return RTK_MQTT_SUCCESS;
}

int rtk_mqtt_disconnect(void) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "No backend available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    
    if (!rtk_mqtt_is_connected()) {
        set_last_error(RTK_MQTT_SUCCESS, "Already disconnected");
        return RTK_MQTT_SUCCESS;
    }
    
    int ret = mqtt_manager.current_backend->disconnect();
    if (ret != RTK_MQTT_SUCCESS) {
        set_last_error(ret, "Disconnect failed");
        return ret;
    }
    
    set_last_error(RTK_MQTT_SUCCESS, "Disconnected successfully");
    printf("[RTK-MQTT] Disconnected from broker\n");
    
    return RTK_MQTT_SUCCESS;
}

int rtk_mqtt_is_connected(void) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        return 0;
    }
    
    return mqtt_manager.current_backend->is_connected();
}

int rtk_mqtt_reconnect(void) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "No backend available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    
    if (mqtt_manager.current_backend->reconnect) {
        return mqtt_manager.current_backend->reconnect();
    } else {
        // 如果後端沒有提供 reconnect，使用 disconnect + connect
        rtk_mqtt_disconnect();
        return rtk_mqtt_connect();
    }
}

int rtk_mqtt_publish(const char* topic, const void* payload, size_t payload_len,
                     rtk_mqtt_qos_t qos, int retained) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "No backend available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    
    if (!topic || !payload) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Invalid parameters");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (!rtk_mqtt_is_connected()) {
        set_last_error(RTK_MQTT_ERROR_NOT_CONNECTED, "Not connected to broker");
        return RTK_MQTT_ERROR_NOT_CONNECTED;
    }
    
    rtk_mqtt_message_t message = {0};
    strncpy(message.topic, topic, sizeof(message.topic) - 1);
    message.payload = (void*)payload;
    message.payload_len = payload_len;
    message.qos = qos;
    message.retained = retained;
    
    return rtk_mqtt_publish_message(&message);
}

int rtk_mqtt_publish_message(const rtk_mqtt_message_t* message) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "No backend available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    
    if (!message) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Message is NULL");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (!rtk_mqtt_is_connected()) {
        set_last_error(RTK_MQTT_ERROR_NOT_CONNECTED, "Not connected to broker");
        return RTK_MQTT_ERROR_NOT_CONNECTED;
    }
    
    int ret = mqtt_manager.current_backend->publish(message);
    if (ret != RTK_MQTT_SUCCESS) {
        set_last_error(ret, "Publish failed");
        return ret;
    }
    
    return RTK_MQTT_SUCCESS;
}

int rtk_mqtt_subscribe(const char* topic, rtk_mqtt_qos_t qos) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "No backend available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    
    if (!topic) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Topic is NULL");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (!rtk_mqtt_is_connected()) {
        set_last_error(RTK_MQTT_ERROR_NOT_CONNECTED, "Not connected to broker");
        return RTK_MQTT_ERROR_NOT_CONNECTED;
    }
    
    int ret = mqtt_manager.current_backend->subscribe(topic, qos);
    if (ret != RTK_MQTT_SUCCESS) {
        set_last_error(ret, "Subscribe failed");
        return ret;
    }
    
    printf("[RTK-MQTT] Subscribed to topic: %s\n", topic);
    return RTK_MQTT_SUCCESS;
}

int rtk_mqtt_unsubscribe(const char* topic) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "No backend available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    
    if (!topic) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Topic is NULL");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (!rtk_mqtt_is_connected()) {
        set_last_error(RTK_MQTT_ERROR_NOT_CONNECTED, "Not connected to broker");
        return RTK_MQTT_ERROR_NOT_CONNECTED;
    }
    
    int ret = mqtt_manager.current_backend->unsubscribe(topic);
    if (ret != RTK_MQTT_SUCCESS) {
        set_last_error(ret, "Unsubscribe failed");
        return ret;
    }
    
    printf("[RTK-MQTT] Unsubscribed from topic: %s\n", topic);
    return RTK_MQTT_SUCCESS;
}

int rtk_mqtt_loop(int timeout_ms) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "No backend available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    
    if (mqtt_manager.current_backend->loop) {
        return mqtt_manager.current_backend->loop(timeout_ms);
    }
    
    return RTK_MQTT_SUCCESS;
}

int rtk_mqtt_yield(int timeout_ms) {
    if (!mqtt_manager.is_initialized || !mqtt_manager.current_backend) {
        set_last_error(RTK_MQTT_ERROR_BACKEND_NOT_FOUND, "No backend available");
        return RTK_MQTT_ERROR_BACKEND_NOT_FOUND;
    }
    
    if (mqtt_manager.current_backend->yield) {
        return mqtt_manager.current_backend->yield(timeout_ms);
    } else if (mqtt_manager.current_backend->loop) {
        // 如果沒有 yield，使用 loop 作為備用
        return mqtt_manager.current_backend->loop(timeout_ms);
    }
    
    return RTK_MQTT_SUCCESS;
}

int rtk_mqtt_set_message_callback(rtk_mqtt_callback_t callback, void* user_data) {
    mqtt_manager.global_message_callback = callback;
    mqtt_manager.global_user_data = user_data;
    
    if (mqtt_manager.current_backend && mqtt_manager.current_backend->set_callback) {
        return mqtt_manager.current_backend->set_callback(callback, user_data);
    }
    
    return RTK_MQTT_SUCCESS;
}

int rtk_mqtt_set_connection_callback(rtk_mqtt_connection_callback_t callback, void* user_data) {
    mqtt_manager.global_connection_callback = callback;
    return RTK_MQTT_SUCCESS;
}

// === 輔助函式實作 ===

const char* rtk_mqtt_get_error_string(rtk_mqtt_error_t error_code) {
    switch (error_code) {
        case RTK_MQTT_SUCCESS: return "Success";
        case RTK_MQTT_ERROR_INVALID_PARAM: return "Invalid parameter";
        case RTK_MQTT_ERROR_NOT_CONNECTED: return "Not connected";
        case RTK_MQTT_ERROR_CONNECTION_FAILED: return "Connection failed";
        case RTK_MQTT_ERROR_TIMEOUT: return "Timeout";
        case RTK_MQTT_ERROR_MEMORY: return "Memory allocation error";
        case RTK_MQTT_ERROR_BACKEND_NOT_FOUND: return "Backend not found";
        case RTK_MQTT_ERROR_ALREADY_CONNECTED: return "Already connected";
        case RTK_MQTT_ERROR_PUBLISH_FAILED: return "Publish failed";
        case RTK_MQTT_ERROR_SUBSCRIBE_FAILED: return "Subscribe failed";
        default: return "Unknown error";
    }
}

const char* rtk_mqtt_get_last_error(void) {
    return mqtt_manager.last_error;
}

const char* rtk_mqtt_get_connection_status_string(void) {
    if (!mqtt_manager.is_initialized) {
        return "Not initialized";
    }
    
    if (!mqtt_manager.current_backend) {
        return "No backend";
    }
    
    if (!mqtt_manager.is_configured) {
        return "Not configured";
    }
    
    return rtk_mqtt_is_connected() ? "Connected" : "Disconnected";
}

void rtk_mqtt_create_default_config(rtk_mqtt_config_t* config, 
                                   const char* broker_host, int broker_port, 
                                   const char* client_id) {
    if (!config) return;
    
    memset(config, 0, sizeof(rtk_mqtt_config_t));
    
    if (broker_host) {
        strncpy(config->broker_host, broker_host, sizeof(config->broker_host) - 1);
    }
    config->broker_port = (broker_port > 0) ? broker_port : 1883;
    
    if (client_id) {
        strncpy(config->client_id, client_id, sizeof(config->client_id) - 1);
    } else {
        snprintf(config->client_id, sizeof(config->client_id), "rtk_client_%ld", 
                (long)time(NULL));
    }
    
    // 設定預設值
    config->keep_alive_interval = 60;
    config->clean_session = 1;
    config->connect_timeout = 30000;
    config->retry_interval = 5000;
    config->max_retry_count = 3;
    config->lwt_qos = RTK_MQTT_QOS_1;
    config->lwt_retained = 0;
}

int rtk_mqtt_validate_config(const rtk_mqtt_config_t* config) {
    if (!config) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Config is NULL");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (strlen(config->broker_host) == 0) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Broker host is empty");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (config->broker_port <= 0 || config->broker_port > 65535) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Invalid broker port");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    if (strlen(config->client_id) == 0) {
        set_last_error(RTK_MQTT_ERROR_INVALID_PARAM, "Client ID is empty");
        return RTK_MQTT_ERROR_INVALID_PARAM;
    }
    
    return RTK_MQTT_SUCCESS;
}
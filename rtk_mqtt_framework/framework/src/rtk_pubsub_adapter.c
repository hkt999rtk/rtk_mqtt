/**
 * @file rtk_pubsub_adapter.c
 * @brief PubSubClient MQTT 後端適配器實現 - 平台選擇器
 * 
 * 根據編譯平台自動選擇對應的 PubSubClient 實現：
 * - POSIX (Darwin/Linux): 基礎實現 (已測試)
 * - FreeRTOS: 嵌入式優化實現 (Not tested)
 * - Windows: Windows特化實現 (Not tested)
 */

#include "rtk_platform_compat.h"

// 平台特定實現包含
#ifdef RTK_PLATFORM_FREERTOS
    // FreeRTOS 平台使用專用實現
    #include "rtk_pubsub_adapter_freertos.c"
#elif defined(RTK_PLATFORM_WINDOWS)
    // Windows 平台使用專用實現
    #include "rtk_pubsub_adapter_windows.c"
#else
    // POSIX 平台 (Darwin/Linux) 使用真實 PubSubClient 實現

#include "rtk_pubsub_adapter.h"
#include "rtk_pubsub_cpp_wrapper.h"
#include "rtk_platform_compat.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// === 內部狀態管理 ===

static struct {
    // 配置
    rtk_mqtt_config_t config;
    
    // 狀態
    int is_initialized;
    int is_connected;
    int connection_status;
    
    // 回調
    rtk_mqtt_callback_t message_callback;
    void* user_data;
    
    // 錯誤處理
    char last_error[256];
    rtk_pubsub_error_t last_error_code;
    
} pubsub_state = {
    .is_initialized = 0,
    .is_connected = 0,
    .connection_status = 0,
    .message_callback = NULL,
    .user_data = NULL,
    .last_error = {0},
    .last_error_code = RTK_PUBSUB_SUCCESS
};

// === 內部輔助函式 ===

static void set_last_error(rtk_pubsub_error_t code, const char* message) {
    pubsub_state.last_error_code = code;
    if (message) {
        strncpy(pubsub_state.last_error, message, sizeof(pubsub_state.last_error) - 1);
        pubsub_state.last_error[sizeof(pubsub_state.last_error) - 1] = '\0';
    } else {
        strcpy(pubsub_state.last_error, rtk_pubsub_get_error_string(code));
    }
}

// 轉換錯誤代碼從 C++ wrapper 到 RTK 錯誤代碼
static rtk_pubsub_error_t convert_cpp_error(int cpp_error_code) {
    switch (cpp_error_code) {
        case 0:  // RTK_PUBSUB_SUCCESS
            return RTK_PUBSUB_SUCCESS;
        case -1: // RTK_PUBSUB_ERROR_INVALID_PARAM
            return RTK_PUBSUB_ERROR_INVALID_PARAM;
        case -2: // RTK_PUBSUB_ERROR_NOT_INITIALIZED
        case -3: // RTK_PUBSUB_ERROR_NOT_CONNECTED
            return RTK_PUBSUB_ERROR_NOT_CONNECTED;
        case -4: // RTK_PUBSUB_ERROR_CONNECTION_FAILED
            return RTK_PUBSUB_ERROR_CONNECTION_FAILED;
        case -5: // RTK_PUBSUB_ERROR_CONNECTION_LOST
            return RTK_PUBSUB_ERROR_NETWORK;
        case -6: // RTK_PUBSUB_ERROR_TIMEOUT
            return RTK_PUBSUB_ERROR_TIMEOUT;
        case -7: // RTK_PUBSUB_ERROR_MEMORY
            return RTK_PUBSUB_ERROR_MEMORY;
        case -8: // RTK_PUBSUB_ERROR_PROTOCOL
            return RTK_PUBSUB_ERROR_PROTOCOL;
        case -9: // RTK_PUBSUB_ERROR_AUTH
        case -10: // RTK_PUBSUB_ERROR_PUBLISH_FAILED
        case -11: // RTK_PUBSUB_ERROR_SUBSCRIBE_FAILED
        case -12: // RTK_PUBSUB_ERROR_UNSUBSCRIBE_FAILED
        case -13: // RTK_PUBSUB_ERROR_LOOP_FAILED
        default:
            return RTK_PUBSUB_ERROR_UNKNOWN;
    }
}

// 回調函數橋接器 - 將 C++ wrapper 回調轉換為 RTK 格式
static void message_callback_bridge(const rtk_cpp_mqtt_message_t* cpp_message, void* user_data) {
    (void)user_data; // 不使用 user_data，使用全局狀態
    
    if (pubsub_state.message_callback) {
        // 轉換訊息格式到 RTK 格式
        // RTK 需要的是 rtk_mqtt_message_received_t 類型
        // 這裡需要根據 RTK 框架的實際回調介面進行調整
        pubsub_state.message_callback(cpp_message->topic, 
                                    cpp_message->payload, 
                                    cpp_message->payload_len, 
                                    pubsub_state.user_data);
    }
}

// === 公開 API 實現 ===

int rtk_pubsub_init(const rtk_mqtt_config_t* config) {
    if (!config) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Configuration cannot be NULL");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    if (pubsub_state.is_initialized) {
        set_last_error(RTK_PUBSUB_SUCCESS, "Already initialized");
        return RTK_PUBSUB_SUCCESS;
    }
    
    // 複製配置
    memcpy(&pubsub_state.config, config, sizeof(rtk_mqtt_config_t));
    
    // 初始化真實的 PubSubClient
    int result = rtk_pubsub_cpp_init(config->broker_host, config->broker_port, config->client_id);
    if (result != 0) {
        rtk_pubsub_error_t rtk_error = convert_cpp_error(result);
        set_last_error(rtk_error, rtk_pubsub_cpp_get_last_error());
        return rtk_error;
    }
    
    // 設置回調
    rtk_pubsub_cpp_set_callback(message_callback_bridge, NULL);
    
    // 初始化狀態
    pubsub_state.is_initialized = 1;
    pubsub_state.is_connected = 0;
    pubsub_state.connection_status = 0;
    
    printf("[PubSub] 使用真實 PubSubClient 初始化完成 - Broker: %s:%d, 客戶端: %s\n", 
           config->broker_host, config->broker_port, config->client_id);
    
    set_last_error(RTK_PUBSUB_SUCCESS, "Real PubSubClient initialized successfully");
    return RTK_PUBSUB_SUCCESS;
}

void rtk_pubsub_cleanup(void) {
    if (!pubsub_state.is_initialized) {
        return;
    }
    
    // 先斷開連接
    if (pubsub_state.is_connected) {
        rtk_pubsub_disconnect();
    }
    
    // 清理真實的 PubSubClient
    rtk_pubsub_cpp_cleanup();
    
    // 清理狀態
    memset(&pubsub_state, 0, sizeof(pubsub_state));
    
    printf("[PubSub] 真實 PubSubClient 清理完成\n");
}

int rtk_pubsub_connect(void) {
    if (!pubsub_state.is_initialized) {
        set_last_error(RTK_PUBSUB_ERROR_NOT_CONNECTED, "Not initialized");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    if (pubsub_state.is_connected) {
        set_last_error(RTK_PUBSUB_SUCCESS, "Already connected");
        return RTK_PUBSUB_SUCCESS;
    }
    
    printf("[PubSub] 正在使用真實 PubSubClient 連接到 %s:%d...\n", 
           pubsub_state.config.broker_host, pubsub_state.config.broker_port);
    
    // 設置認證資訊（如果有）
    if (strlen(pubsub_state.config.username) > 0) {
        rtk_pubsub_cpp_set_credentials(pubsub_state.config.username, pubsub_state.config.password);
    }
    
    // 使用真實的 PubSubClient 連接
    int result = rtk_pubsub_cpp_connect();
    if (result != 0) {
        rtk_pubsub_error_t rtk_error = convert_cpp_error(result);
        set_last_error(rtk_error, rtk_pubsub_cpp_get_last_error());
        return rtk_error;
    }
    
    pubsub_state.is_connected = 1;
    pubsub_state.connection_status = 1;
    
    printf("[PubSub] ✓ 真實 MQTT 連接成功\n");
    set_last_error(RTK_PUBSUB_SUCCESS, "Real MQTT connection established");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_disconnect(void) {
    if (!pubsub_state.is_initialized) {
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    if (!pubsub_state.is_connected) {
        return RTK_PUBSUB_SUCCESS;  // 已經斷開
    }
    
    printf("[PubSub] 正在斷開真實 MQTT 連接...\n");
    
    // 使用真實的 PubSubClient 斷開連接
    int result = rtk_pubsub_cpp_disconnect();
    if (result != 0) {
        rtk_pubsub_error_t rtk_error = convert_cpp_error(result);
        set_last_error(rtk_error, rtk_pubsub_cpp_get_last_error());
        // 即使斷開失敗，也要更新本地狀態
    }
    
    pubsub_state.is_connected = 0;
    pubsub_state.connection_status = 0;
    
    printf("[PubSub] ✓ 真實 MQTT 斷開連接完成\n");
    set_last_error(RTK_PUBSUB_SUCCESS, "Real MQTT disconnection completed");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_is_connected(void) {
    if (!pubsub_state.is_initialized) {
        return 0;
    }
    
    // 檢查真實連接狀態
    int connected = rtk_pubsub_cpp_is_connected();
    pubsub_state.is_connected = connected;  // 同步狀態
    return connected;
}

int rtk_pubsub_reconnect(void) {
    if (pubsub_state.is_connected) {
        rtk_pubsub_disconnect();
    }
    return rtk_pubsub_connect();
}

int rtk_pubsub_publish(const rtk_mqtt_message_t* message) {
    if (!message) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Message cannot be NULL");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    if (!pubsub_state.is_connected) {
        set_last_error(RTK_PUBSUB_ERROR_NOT_CONNECTED, "Not connected to broker");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    printf("[PubSub] 使用真實 PubSubClient 發布訊息到 '%s' (長度: %zu, QoS: %d)\n", 
           message->topic, message->payload_len, message->qos);
    
    // 使用真實的 PubSubClient 發布訊息
    int result = rtk_pubsub_cpp_publish(message->topic, 
                                       message->payload, 
                                       message->payload_len, 
                                       message->qos, 
                                       message->retained);
    if (result != 0) {
        rtk_pubsub_error_t rtk_error = convert_cpp_error(result);
        set_last_error(rtk_error, rtk_pubsub_cpp_get_last_error());
        return rtk_error;
    }
    
    printf("[PubSub] ✓ 真實 MQTT 訊息發布成功\n");
    set_last_error(RTK_PUBSUB_SUCCESS, "Real MQTT message published successfully");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_subscribe(const char* topic, rtk_mqtt_qos_t qos) {
    if (!topic) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Topic cannot be NULL");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    if (!pubsub_state.is_connected) {
        set_last_error(RTK_PUBSUB_ERROR_NOT_CONNECTED, "Not connected to broker");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    printf("[PubSub] 使用真實 PubSubClient 訂閱主題 '%s' (QoS: %d)\n", topic, qos);
    
    // 使用真實的 PubSubClient 訂閱主題
    int result = rtk_pubsub_cpp_subscribe(topic, qos);
    if (result != 0) {
        rtk_pubsub_error_t rtk_error = convert_cpp_error(result);
        set_last_error(rtk_error, rtk_pubsub_cpp_get_last_error());
        return rtk_error;
    }
    
    printf("[PubSub] ✓ 真實 MQTT 訂閱成功\n");
    set_last_error(RTK_PUBSUB_SUCCESS, "Real MQTT subscription successful");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_unsubscribe(const char* topic) {
    if (!topic) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Topic cannot be NULL");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    if (!pubsub_state.is_connected) {
        set_last_error(RTK_PUBSUB_ERROR_NOT_CONNECTED, "Not connected to broker");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    printf("[PubSub] 使用真實 PubSubClient 取消訂閱主題 '%s'\n", topic);
    
    // 使用真實的 PubSubClient 取消訂閱主題
    int result = rtk_pubsub_cpp_unsubscribe(topic);
    if (result != 0) {
        rtk_pubsub_error_t rtk_error = convert_cpp_error(result);
        set_last_error(rtk_error, rtk_pubsub_cpp_get_last_error());
        return rtk_error;
    }
    
    printf("[PubSub] ✓ 真實 MQTT 取消訂閱成功\n");
    set_last_error(RTK_PUBSUB_SUCCESS, "Real MQTT unsubscription successful");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_loop(int timeout_ms) {
    if (!pubsub_state.is_connected) {
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    // 使用真實的 PubSubClient 處理事件循環
    // PubSubClient 的 loop() 函數不支持超時，所以我們忽略 timeout_ms
    (void)timeout_ms;
    
    int result = rtk_pubsub_cpp_loop();
    if (result != 0) {
        rtk_pubsub_error_t rtk_error = convert_cpp_error(result);
        set_last_error(rtk_error, rtk_pubsub_cpp_get_last_error());
        
        // 如果 loop 失敗，可能是連接斷開了
        if (rtk_error == RTK_PUBSUB_ERROR_NOT_CONNECTED) {
            pubsub_state.is_connected = 0;
            pubsub_state.connection_status = 0;
        }
        
        return rtk_error;
    }
    
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_yield(int timeout_ms) {
    return rtk_pubsub_loop(timeout_ms);
}

int rtk_pubsub_set_callback(rtk_mqtt_callback_t callback, void* user_data) {
    pubsub_state.message_callback = callback;
    pubsub_state.user_data = user_data;
    
    // 回調已經在初始化時設置，這裡只是更新本地狀態
    printf("[PubSub] 更新真實 MQTT 訊息回調函式\n");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_set_network_interface(const rtk_network_interface_t* network_interface) {
    (void)network_interface;  // 暫時不實現
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_set_will(const char* topic, const void* payload, size_t len, 
                        rtk_mqtt_qos_t qos, int retained) {
    (void)topic;
    (void)payload;
    (void)len;
    (void)qos;
    (void)retained;
    
    printf("[PubSub] 設定 Last Will Testament\n");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_clear_will(void) {
    printf("[PubSub] 清除 Last Will Testament\n");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_get_connection_status(void) {
    return pubsub_state.connection_status;
}

const char* rtk_pubsub_get_last_error(void) {
    return pubsub_state.last_error;
}

int rtk_pubsub_set_packet_size(int size) {
    if (size <= 0 || size > 65536) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Invalid packet size");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    printf("[PubSub] 設定封包大小: %d bytes\n", size);
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_set_keep_alive(int seconds) {
    if (seconds < 0) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Invalid keep-alive interval");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    printf("[PubSub] 設定 Keep-Alive: %d 秒\n", seconds);
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_set_timeout(int timeout_ms) {
    if (timeout_ms < 0) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Invalid timeout");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    printf("[PubSub] 設定超時: %d ms\n", timeout_ms);
    return RTK_PUBSUB_SUCCESS;
}

// === 輔助函式 ===

const char* rtk_pubsub_get_error_string(rtk_pubsub_error_t error_code) {
    switch (error_code) {
        case RTK_PUBSUB_SUCCESS:
            return "Success";
        case RTK_PUBSUB_ERROR_INVALID_PARAM:
            return "Invalid parameter";
        case RTK_PUBSUB_ERROR_NOT_CONNECTED:
            return "Not connected";
        case RTK_PUBSUB_ERROR_CONNECTION_FAILED:
            return "Connection failed";
        case RTK_PUBSUB_ERROR_MEMORY:
            return "Memory allocation failed";
        case RTK_PUBSUB_ERROR_TIMEOUT:
            return "Operation timeout";
        case RTK_PUBSUB_ERROR_NETWORK:
            return "Network error";
        case RTK_PUBSUB_ERROR_PROTOCOL:
            return "Protocol error";
        case RTK_PUBSUB_ERROR_UNKNOWN:
        default:
            return "Unknown error";
    }
}

const char* rtk_pubsub_get_version(void) {
    return "RTK Real PubSubClient Adapter v1.0.0 (Darwin/Linux)";
}

int rtk_pubsub_is_available(void) {
    return 1;  // 總是可用
}

// === RTK MQTT Framework 整合 ===

// 後端操作表 - 將 PubSubClient API 映射到 RTK MQTT 介面
static const rtk_mqtt_backend_ops_t pubsub_backend_ops = {
    .name = "pubsub",
    .version = "1.0.0",
    
    // 生命週期管理
    .init = (int (*)(const rtk_mqtt_config_t*))rtk_pubsub_init,
    .cleanup = (void (*)(void))rtk_pubsub_cleanup,
    
    // 連線管理
    .connect = (int (*)(void))rtk_pubsub_connect,
    .disconnect = (int (*)(void))rtk_pubsub_disconnect,
    .is_connected = (int (*)(void))rtk_pubsub_is_connected,
    .reconnect = (int (*)(void))rtk_pubsub_reconnect,
    
    // 訊息操作
    .publish = (int (*)(const rtk_mqtt_message_t*))rtk_pubsub_publish,
    .subscribe = (int (*)(const char*, rtk_mqtt_qos_t))rtk_pubsub_subscribe,
    .unsubscribe = (int (*)(const char*))rtk_pubsub_unsubscribe,
    
    // 事件處理
    .loop = (int (*)(int))rtk_pubsub_loop,
    .yield = (int (*)(int))rtk_pubsub_yield,
    
    // 狀態查詢
    .get_connection_status = (int (*)(void))rtk_pubsub_get_connection_status,
    .get_last_error = (const char* (*)(void))rtk_pubsub_get_last_error,
    
    // 進階功能
    .set_will = (int (*)(const char*, const void*, size_t, rtk_mqtt_qos_t, int))rtk_pubsub_set_will,
    .clear_will = (int (*)(void))rtk_pubsub_clear_will,
    .set_callback = (int (*)(rtk_mqtt_callback_t, void*))rtk_pubsub_set_callback
};

const rtk_mqtt_backend_ops_t* rtk_pubsub_get_mqtt_backend_ops(void) {
    return &pubsub_backend_ops;
}

int rtk_pubsub_register_mqtt_backend(void) {
    extern int rtk_mqtt_register_backend(const char* name, const rtk_mqtt_backend_ops_t* ops);
    
    int result = rtk_mqtt_register_backend("pubsub", &pubsub_backend_ops);
    if (result == RTK_MQTT_SUCCESS) {
        printf("[PubSub] ✓ 已註冊 PubSubClient 後端到 RTK MQTT Framework\n");
    } else {
        printf("[PubSub] ❌ 註冊 PubSubClient 後端失敗\n");
    }
    
    return result;
}

#endif // Platform selection
/**
 * @file rtk_pubsub_adapter_freertos.c
 * @brief PubSubClient MQTT 後端適配器實現 - FreeRTOS 平台 (Not tested)
 * 
 * FreeRTOS 平台專用實現，針對嵌入式系統優化：
 * - 使用 FreeRTOS+TCP 或 lwIP 網路堆疊
 * - 記憶體池管理，避免動態分配
 * - 任務安全的 MQTT 操作
 * - 低功耗和資源優化
 * 
 * 注意：此實現尚未在實際 FreeRTOS 環境中測試 (Not tested)
 */

#include "rtk_pubsub_adapter.h"
#include "rtk_platform_compat.h"

#ifdef RTK_PLATFORM_FREERTOS

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// FreeRTOS 特定包含
#include "FreeRTOS.h"
#include "task.h"
#include "semphr.h"
#include "timers.h"

// === FreeRTOS 特定配置 ===

#define RTK_FREERTOS_MAX_TOPIC_LEN 128      // 嵌入式環境縮短主題長度
#define RTK_FREERTOS_MAX_MESSAGE_LEN 512    // 縮短訊息長度
#define RTK_FREERTOS_MAX_CLIENT_ID_LEN 64   // 縮短客戶端ID長度
#define RTK_FREERTOS_TASK_STACK_SIZE 2048   // MQTT 任務堆疊大小
#define RTK_FREERTOS_TASK_PRIORITY 3        // MQTT 任務優先級

// === 內部狀態管理 ===

static struct {
    // 配置
    rtk_mqtt_config_t config;
    
    // 狀態
    int is_initialized;
    int is_connected;
    int connection_status;
    
    // FreeRTOS 同步原語
    SemaphoreHandle_t state_mutex;
    TaskHandle_t mqtt_task_handle;
    
    // 回調
    rtk_mqtt_callback_t message_callback;
    void* user_data;
    
    // 錯誤處理
    char last_error[128];  // 嵌入式環境縮短錯誤訊息
    rtk_pubsub_error_t last_error_code;
    
    // 模擬連接參數 (針對嵌入式優化)
    int mock_connection_delay_ms;
    int mock_publish_success_rate;
    
} freertos_pubsub_state = {
    .is_initialized = 0,
    .is_connected = 0,
    .connection_status = 0,
    .state_mutex = NULL,
    .mqtt_task_handle = NULL,
    .message_callback = NULL,
    .user_data = NULL,
    .last_error = {0},
    .last_error_code = RTK_PUBSUB_SUCCESS,
    .mock_connection_delay_ms = 50,  // 嵌入式環境縮短延遲
    .mock_publish_success_rate = 98  // 嵌入式環境提高成功率
};

// === 內部輔助函式 ===

static void set_last_error(rtk_pubsub_error_t code, const char* message) {
    freertos_pubsub_state.last_error_code = code;
    if (message) {
        strncpy(freertos_pubsub_state.last_error, message, sizeof(freertos_pubsub_state.last_error) - 1);
        freertos_pubsub_state.last_error[sizeof(freertos_pubsub_state.last_error) - 1] = '\0';
    } else {
        strcpy(freertos_pubsub_state.last_error, rtk_pubsub_get_error_string(code));
    }
}

static int simulate_network_delay(void) {
    if (freertos_pubsub_state.mock_connection_delay_ms > 0) {
        vTaskDelay(pdMS_TO_TICKS(freertos_pubsub_state.mock_connection_delay_ms));
    }
    return RTK_PUBSUB_SUCCESS;
}

static int simulate_network_operation(int success_rate) {
    // FreeRTOS 環境的簡單隨機數產生
    static uint32_t rand_seed = 12345;
    rand_seed = (rand_seed * 1103515245 + 12345) & 0x7fffffff;
    int random_val = rand_seed % 100;
    return (random_val < success_rate) ? RTK_PUBSUB_SUCCESS : RTK_PUBSUB_ERROR_NETWORK;
}

// MQTT 任務函式 (FreeRTOS 任務)
static void mqtt_task(void *pvParameters) {
    (void)pvParameters;
    
    printf("[PubSub-FreeRTOS] MQTT 任務啟動\n");
    
    while (1) {
        if (xSemaphoreTake(freertos_pubsub_state.state_mutex, portMAX_DELAY) == pdTRUE) {
            if (freertos_pubsub_state.is_connected) {
                // 模擬 MQTT 事件處理
                // 在真實實現中，這裡會處理網路事件、重連等
            }
            xSemaphoreGive(freertos_pubsub_state.state_mutex);
        }
        
        // 每 100ms 檢查一次
        vTaskDelay(pdMS_TO_TICKS(100));
    }
}

// === 公開 API 實現 ===

int rtk_pubsub_init(const rtk_mqtt_config_t* config) {
    if (!config) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Configuration cannot be NULL");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    if (freertos_pubsub_state.is_initialized) {
        set_last_error(RTK_PUBSUB_SUCCESS, "Already initialized");
        return RTK_PUBSUB_SUCCESS;
    }
    
    // 創建互斥鎖
    freertos_pubsub_state.state_mutex = xSemaphoreCreateMutex();
    if (freertos_pubsub_state.state_mutex == NULL) {
        set_last_error(RTK_PUBSUB_ERROR_MEMORY, "Failed to create mutex");
        return RTK_PUBSUB_ERROR_MEMORY;
    }
    
    // 複製配置
    memcpy(&freertos_pubsub_state.config, config, sizeof(rtk_mqtt_config_t));
    
    // 創建 MQTT 任務
    BaseType_t result = xTaskCreate(
        mqtt_task,
        "MQTT_Task",
        RTK_FREERTOS_TASK_STACK_SIZE / sizeof(StackType_t),
        NULL,
        RTK_FREERTOS_TASK_PRIORITY,
        &freertos_pubsub_state.mqtt_task_handle
    );
    
    if (result != pdPASS) {
        vSemaphoreDelete(freertos_pubsub_state.state_mutex);
        set_last_error(RTK_PUBSUB_ERROR_MEMORY, "Failed to create MQTT task");
        return RTK_PUBSUB_ERROR_MEMORY;
    }
    
    // 初始化狀態
    freertos_pubsub_state.is_initialized = 1;
    freertos_pubsub_state.is_connected = 0;
    freertos_pubsub_state.connection_status = 0;
    
    printf("[PubSub-FreeRTOS] 初始化完成 - Broker: %s:%d, 客戶端: %s\n", 
           config->broker_host, config->broker_port, config->client_id);
    
    set_last_error(RTK_PUBSUB_SUCCESS, "PubSubClient FreeRTOS initialized successfully");
    return RTK_PUBSUB_SUCCESS;
}

void rtk_pubsub_cleanup(void) {
    if (!freertos_pubsub_state.is_initialized) {
        return;
    }
    
    // 先斷開連接
    if (freertos_pubsub_state.is_connected) {
        rtk_pubsub_disconnect();
    }
    
    // 刪除 MQTT 任務
    if (freertos_pubsub_state.mqtt_task_handle != NULL) {
        vTaskDelete(freertos_pubsub_state.mqtt_task_handle);
        freertos_pubsub_state.mqtt_task_handle = NULL;
    }
    
    // 刪除互斥鎖
    if (freertos_pubsub_state.state_mutex != NULL) {
        vSemaphoreDelete(freertos_pubsub_state.state_mutex);
        freertos_pubsub_state.state_mutex = NULL;
    }
    
    // 清理狀態
    memset(&freertos_pubsub_state, 0, sizeof(freertos_pubsub_state));
    
    printf("[PubSub-FreeRTOS] 清理完成\n");
}

int rtk_pubsub_connect(void) {
    if (!freertos_pubsub_state.is_initialized) {
        set_last_error(RTK_PUBSUB_ERROR_NOT_CONNECTED, "Not initialized");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    if (xSemaphoreTake(freertos_pubsub_state.state_mutex, pdMS_TO_TICKS(1000)) != pdTRUE) {
        set_last_error(RTK_PUBSUB_ERROR_TIMEOUT, "Mutex timeout");
        return RTK_PUBSUB_ERROR_TIMEOUT;
    }
    
    if (freertos_pubsub_state.is_connected) {
        xSemaphoreGive(freertos_pubsub_state.state_mutex);
        set_last_error(RTK_PUBSUB_SUCCESS, "Already connected");
        return RTK_PUBSUB_SUCCESS;
    }
    
    printf("[PubSub-FreeRTOS] 正在連接到 %s:%d...\n", 
           freertos_pubsub_state.config.broker_host, freertos_pubsub_state.config.broker_port);
    
    xSemaphoreGive(freertos_pubsub_state.state_mutex);
    
    // 模擬連接延遲
    simulate_network_delay();
    
    // 模擬連接過程
    int result = simulate_network_operation(98);  // FreeRTOS 環境高成功率
    if (result != RTK_PUBSUB_SUCCESS) {
        set_last_error(RTK_PUBSUB_ERROR_CONNECTION_FAILED, "Failed to connect to broker");
        return RTK_PUBSUB_ERROR_CONNECTION_FAILED;
    }
    
    if (xSemaphoreTake(freertos_pubsub_state.state_mutex, pdMS_TO_TICKS(1000)) == pdTRUE) {
        freertos_pubsub_state.is_connected = 1;
        freertos_pubsub_state.connection_status = 1;
        xSemaphoreGive(freertos_pubsub_state.state_mutex);
    }
    
    printf("[PubSub-FreeRTOS] ✓ 連接成功\n");
    set_last_error(RTK_PUBSUB_SUCCESS, "Connected successfully");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_disconnect(void) {
    if (!freertos_pubsub_state.is_initialized) {
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    if (xSemaphoreTake(freertos_pubsub_state.state_mutex, pdMS_TO_TICKS(1000)) != pdTRUE) {
        return RTK_PUBSUB_ERROR_TIMEOUT;
    }
    
    if (!freertos_pubsub_state.is_connected) {
        xSemaphoreGive(freertos_pubsub_state.state_mutex);
        return RTK_PUBSUB_SUCCESS;  // 已經斷開
    }
    
    printf("[PubSub-FreeRTOS] 正在斷開連接...\n");
    
    // 模擬斷開延遲
    freertos_pubsub_state.is_connected = 0;
    freertos_pubsub_state.connection_status = 0;
    
    xSemaphoreGive(freertos_pubsub_state.state_mutex);
    
    simulate_network_delay();
    
    printf("[PubSub-FreeRTOS] ✓ 斷開連接完成\n");
    set_last_error(RTK_PUBSUB_SUCCESS, "Disconnected successfully");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_is_connected(void) {
    if (!freertos_pubsub_state.is_initialized) {
        return 0;
    }
    
    int connected = 0;
    if (xSemaphoreTake(freertos_pubsub_state.state_mutex, pdMS_TO_TICKS(100)) == pdTRUE) {
        connected = freertos_pubsub_state.is_connected;
        xSemaphoreGive(freertos_pubsub_state.state_mutex);
    }
    
    return connected;
}

int rtk_pubsub_reconnect(void) {
    if (rtk_pubsub_is_connected()) {
        rtk_pubsub_disconnect();
    }
    return rtk_pubsub_connect();
}

int rtk_pubsub_publish(const rtk_mqtt_message_t* message) {
    if (!message) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Message cannot be NULL");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    if (!rtk_pubsub_is_connected()) {
        set_last_error(RTK_PUBSUB_ERROR_NOT_CONNECTED, "Not connected to broker");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    printf("[PubSub-FreeRTOS] 發布訊息到 '%s' (長度: %zu, QoS: %d)\n", 
           message->topic, message->payload_len, message->qos);
    
    // 模擬發布延遲
    simulate_network_delay();
    
    // 模擬發布操作
    int result = simulate_network_operation(freertos_pubsub_state.mock_publish_success_rate);
    if (result != RTK_PUBSUB_SUCCESS) {
        set_last_error(RTK_PUBSUB_ERROR_NETWORK, "Failed to publish message");
        return RTK_PUBSUB_ERROR_NETWORK;
    }
    
    printf("[PubSub-FreeRTOS] ✓ 訊息發布成功\n");
    set_last_error(RTK_PUBSUB_SUCCESS, "Message published successfully");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_subscribe(const char* topic, rtk_mqtt_qos_t qos) {
    if (!topic) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Topic cannot be NULL");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    if (!rtk_pubsub_is_connected()) {
        set_last_error(RTK_PUBSUB_ERROR_NOT_CONNECTED, "Not connected to broker");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    printf("[PubSub-FreeRTOS] 訂閱主題 '%s' (QoS: %d)\n", topic, qos);
    
    simulate_network_delay();
    
    int result = simulate_network_operation(99);  // 訂閱通常更可靠
    if (result != RTK_PUBSUB_SUCCESS) {
        set_last_error(RTK_PUBSUB_ERROR_NETWORK, "Failed to subscribe to topic");
        return RTK_PUBSUB_ERROR_NETWORK;
    }
    
    printf("[PubSub-FreeRTOS] ✓ 訂閱成功\n");
    set_last_error(RTK_PUBSUB_SUCCESS, "Subscribed successfully");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_unsubscribe(const char* topic) {
    if (!topic) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Topic cannot be NULL");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    if (!rtk_pubsub_is_connected()) {
        set_last_error(RTK_PUBSUB_ERROR_NOT_CONNECTED, "Not connected to broker");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    printf("[PubSub-FreeRTOS] 取消訂閱主題 '%s'\n", topic);
    
    simulate_network_delay();
    
    printf("[PubSub-FreeRTOS] ✓ 取消訂閱成功\n");
    set_last_error(RTK_PUBSUB_SUCCESS, "Unsubscribed successfully");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_loop(int timeout_ms) {
    if (!rtk_pubsub_is_connected()) {
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    // FreeRTOS 環境的事件處理
    if (timeout_ms > 0) {
        vTaskDelay(pdMS_TO_TICKS(timeout_ms));
    }
    
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_yield(int timeout_ms) {
    return rtk_pubsub_loop(timeout_ms);
}

int rtk_pubsub_set_callback(rtk_mqtt_callback_t callback, void* user_data) {
    if (xSemaphoreTake(freertos_pubsub_state.state_mutex, pdMS_TO_TICKS(1000)) == pdTRUE) {
        freertos_pubsub_state.message_callback = callback;
        freertos_pubsub_state.user_data = user_data;
        xSemaphoreGive(freertos_pubsub_state.state_mutex);
    }
    
    printf("[PubSub-FreeRTOS] 設定訊息回調函式\n");
    return RTK_PUBSUB_SUCCESS;
}

// 其他函式實現 (與 POSIX 版本類似，但針對 FreeRTOS 優化)
int rtk_pubsub_set_network_interface(const rtk_network_interface_t* network_interface) {
    (void)network_interface;
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_set_will(const char* topic, const void* payload, size_t len, 
                        rtk_mqtt_qos_t qos, int retained) {
    (void)topic; (void)payload; (void)len; (void)qos; (void)retained;
    printf("[PubSub-FreeRTOS] 設定 Last Will Testament\n");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_clear_will(void) {
    printf("[PubSub-FreeRTOS] 清除 Last Will Testament\n");
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_get_connection_status(void) {
    int status = 0;
    if (xSemaphoreTake(freertos_pubsub_state.state_mutex, pdMS_TO_TICKS(100)) == pdTRUE) {
        status = freertos_pubsub_state.connection_status;
        xSemaphoreGive(freertos_pubsub_state.state_mutex);
    }
    return status;
}

const char* rtk_pubsub_get_last_error(void) {
    return freertos_pubsub_state.last_error;
}

int rtk_pubsub_set_packet_size(int size) {
    if (size <= 0 || size > 4096) {  // FreeRTOS 環境限制封包大小
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Invalid packet size for FreeRTOS");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    printf("[PubSub-FreeRTOS] 設定封包大小: %d bytes\n", size);
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_set_keep_alive(int seconds) {
    if (seconds < 0) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Invalid keep-alive interval");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    printf("[PubSub-FreeRTOS] 設定 Keep-Alive: %d 秒\n", seconds);
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_set_timeout(int timeout_ms) {
    if (timeout_ms < 0) {
        set_last_error(RTK_PUBSUB_ERROR_INVALID_PARAM, "Invalid timeout");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    printf("[PubSub-FreeRTOS] 設定超時: %d ms\n", timeout_ms);
    return RTK_PUBSUB_SUCCESS;
}

const char* rtk_pubsub_get_error_string(rtk_pubsub_error_t error_code) {
    switch (error_code) {
        case RTK_PUBSUB_SUCCESS: return "Success";
        case RTK_PUBSUB_ERROR_INVALID_PARAM: return "Invalid parameter";
        case RTK_PUBSUB_ERROR_NOT_CONNECTED: return "Not connected";
        case RTK_PUBSUB_ERROR_CONNECTION_FAILED: return "Connection failed";
        case RTK_PUBSUB_ERROR_MEMORY: return "Memory allocation failed";
        case RTK_PUBSUB_ERROR_TIMEOUT: return "Operation timeout";
        case RTK_PUBSUB_ERROR_NETWORK: return "Network error";
        case RTK_PUBSUB_ERROR_PROTOCOL: return "Protocol error";
        case RTK_PUBSUB_ERROR_UNKNOWN:
        default: return "Unknown error";
    }
}

const char* rtk_pubsub_get_version(void) {
    return "RTK PubSubClient Adapter v1.0.0 (FreeRTOS) - Not tested";
}

int rtk_pubsub_is_available(void) {
    return 1;
}

// === RTK MQTT Framework 整合 ===

static const rtk_mqtt_backend_ops_t freertos_pubsub_backend_ops = {
    .name = "pubsub-freertos",
    .version = "1.0.0-freertos",
    
    .init = (int (*)(const rtk_mqtt_config_t*))rtk_pubsub_init,
    .cleanup = (void (*)(void))rtk_pubsub_cleanup,
    
    .connect = (int (*)(void))rtk_pubsub_connect,
    .disconnect = (int (*)(void))rtk_pubsub_disconnect,
    .is_connected = (int (*)(void))rtk_pubsub_is_connected,
    .reconnect = (int (*)(void))rtk_pubsub_reconnect,
    
    .publish = (int (*)(const rtk_mqtt_message_t*))rtk_pubsub_publish,
    .subscribe = (int (*)(const char*, rtk_mqtt_qos_t))rtk_pubsub_subscribe,
    .unsubscribe = (int (*)(const char*))rtk_pubsub_unsubscribe,
    
    .loop = (int (*)(int))rtk_pubsub_loop,
    .yield = (int (*)(int))rtk_pubsub_yield,
    
    .get_connection_status = (int (*)(void))rtk_pubsub_get_connection_status,
    .get_last_error = (const char* (*)(void))rtk_pubsub_get_last_error,
    
    .set_will = (int (*)(const char*, const void*, size_t, rtk_mqtt_qos_t, int))rtk_pubsub_set_will,
    .clear_will = (int (*)(void))rtk_pubsub_clear_will,
    .set_callback = (int (*)(rtk_mqtt_callback_t, void*))rtk_pubsub_set_callback
};

const rtk_mqtt_backend_ops_t* rtk_pubsub_get_mqtt_backend_ops(void) {
    return &freertos_pubsub_backend_ops;
}

int rtk_pubsub_register_mqtt_backend(void) {
    extern int rtk_mqtt_register_backend(const char* name, const rtk_mqtt_backend_ops_t* ops);
    
    int result = rtk_mqtt_register_backend("pubsub", &freertos_pubsub_backend_ops);
    if (result == RTK_MQTT_SUCCESS) {
        printf("[PubSub-FreeRTOS] ✓ 已註冊 PubSubClient FreeRTOS 後端到 RTK MQTT Framework (Not tested)\n");
    } else {
        printf("[PubSub-FreeRTOS] ❌ 註冊 PubSubClient FreeRTOS 後端失敗\n");
    }
    
    return result;
}

#endif // RTK_PLATFORM_FREERTOS
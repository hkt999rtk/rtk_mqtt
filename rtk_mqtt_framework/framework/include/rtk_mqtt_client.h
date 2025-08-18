#ifndef RTK_MQTT_CLIENT_H
#define RTK_MQTT_CLIENT_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @file rtk_mqtt_client.h
 * @brief 統一 MQTT 客戶端介面 - 支援多種後端實作
 * 
 * 此介面提供了一個統一的 MQTT 客戶端抽象層，使用：
 * - PubSubClient (跨平台統一後端)
 * - 平台特定的優化實作
 */

// === 前向宣告 ===

typedef struct rtk_mqtt_backend_ops rtk_mqtt_backend_ops_t;
typedef struct rtk_mqtt_config rtk_mqtt_config_t;
typedef struct rtk_mqtt_message rtk_mqtt_message_t;

// === 回調函式類型 ===

/**
 * @brief MQTT 訊息接收回調函式
 * @param topic 主題名稱
 * @param payload 訊息內容
 * @param payload_len 訊息長度
 * @param user_data 使用者資料
 */
typedef void (*rtk_mqtt_callback_t)(const char* topic, const void* payload, 
                                    size_t payload_len, void* user_data);

/**
 * @brief MQTT 連線狀態回調函式
 * @param connected 是否已連線 (1=已連線, 0=斷線)
 * @param reason_code 狀態變更原因碼
 * @param user_data 使用者資料
 */
typedef void (*rtk_mqtt_connection_callback_t)(int connected, int reason_code, void* user_data);

// === 列舉定義 ===

/**
 * @brief MQTT 後端類型
 */
typedef enum {
    RTK_MQTT_BACKEND_PUBSUB = 0,    /**< PubSubClient (default) */
    RTK_MQTT_BACKEND_CUSTOM = 99    /**< 自定義後端 */
} rtk_mqtt_backend_type_t;

/**
 * @brief MQTT QoS 等級
 */
typedef enum {
    RTK_MQTT_QOS_0 = 0,    /**< 最多一次傳送 */
    RTK_MQTT_QOS_1 = 1,    /**< 至少一次傳送 */
    RTK_MQTT_QOS_2 = 2     /**< 確實一次傳送 */
} rtk_mqtt_qos_t;

/**
 * @brief MQTT 錯誤碼
 */
typedef enum {
    RTK_MQTT_SUCCESS = 0,
    RTK_MQTT_ERROR_INVALID_PARAM = -1,
    RTK_MQTT_ERROR_NOT_CONNECTED = -2,
    RTK_MQTT_ERROR_CONNECTION_FAILED = -3,
    RTK_MQTT_ERROR_TIMEOUT = -4,
    RTK_MQTT_ERROR_MEMORY = -5,
    RTK_MQTT_ERROR_BACKEND_NOT_FOUND = -6,
    RTK_MQTT_ERROR_ALREADY_CONNECTED = -7,
    RTK_MQTT_ERROR_PUBLISH_FAILED = -8,
    RTK_MQTT_ERROR_SUBSCRIBE_FAILED = -9,
    RTK_MQTT_ERROR_UNKNOWN = -99
} rtk_mqtt_error_t;

// === 結構定義 ===

/**
 * @brief MQTT 連線配置
 */
struct rtk_mqtt_config {
    char broker_host[256];           /**< Broker 主機位址 */
    int broker_port;                 /**< Broker 埠號 */
    char client_id[128];             /**< 客戶端 ID */
    char username[128];              /**< 使用者名稱 */
    char password[128];              /**< 密碼 */
    
    int keep_alive_interval;         /**< Keep-alive 間隔 (秒) */
    int clean_session;               /**< 是否使用 clean session */
    int connect_timeout;             /**< 連線超時 (毫秒) */
    int retry_interval;              /**< 重連間隔 (毫秒) */
    int max_retry_count;             /**< 最大重試次數 */
    
    // Last Will Testament
    char lwt_topic[256];             /**< LWT 主題 */
    char lwt_message[512];           /**< LWT 訊息 */
    rtk_mqtt_qos_t lwt_qos;          /**< LWT QoS */
    int lwt_retained;                /**< LWT retained 標誌 */
    
    // 回調設定
    rtk_mqtt_callback_t message_callback;           /**< 訊息回調 */
    rtk_mqtt_connection_callback_t connection_callback; /**< 連線狀態回調 */
    void* user_data;                 /**< 使用者資料 */
};

/**
 * @brief MQTT 訊息結構
 */
struct rtk_mqtt_message {
    char topic[256];                 /**< 主題 */
    void* payload;                   /**< 訊息內容 */
    size_t payload_len;              /**< 訊息長度 */
    rtk_mqtt_qos_t qos;              /**< QoS 等級 */
    int retained;                    /**< Retained 標誌 */
    uint16_t message_id;             /**< 訊息 ID */
};

/**
 * @brief MQTT 後端操作介面
 * 
 * 每個 MQTT 後端實作都必須提供這些函式的實作
 */
struct rtk_mqtt_backend_ops {
    const char* name;                /**< 後端名稱 */
    const char* version;             /**< 後端版本 */
    
    // 生命週期管理
    int (*init)(const rtk_mqtt_config_t* config);
    void (*cleanup)(void);
    
    // 連線管理
    int (*connect)(void);
    int (*disconnect)(void);
    int (*is_connected)(void);
    int (*reconnect)(void);
    
    // 訊息操作
    int (*publish)(const rtk_mqtt_message_t* message);
    int (*subscribe)(const char* topic, rtk_mqtt_qos_t qos);
    int (*unsubscribe)(const char* topic);
    
    // 事件處理
    int (*loop)(int timeout_ms);     /**< 處理網路事件和回調 */
    int (*yield)(int timeout_ms);    /**< 非阻塞的事件處理 */
    
    // 狀態查詢
    int (*get_connection_status)(void);
    const char* (*get_last_error)(void);
    
    // 進階功能 (可選)
    int (*set_will)(const char* topic, const void* payload, size_t len, 
                    rtk_mqtt_qos_t qos, int retained);
    int (*clear_will)(void);
    int (*set_callback)(rtk_mqtt_callback_t callback, void* user_data);
};

// === 公開 API ===

/**
 * @brief 初始化 MQTT 客戶端系統
 * @param backend_type 後端類型
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_init(rtk_mqtt_backend_type_t backend_type);

/**
 * @brief 清理 MQTT 客戶端系統
 */
void rtk_mqtt_cleanup(void);

/**
 * @brief 設定 MQTT 後端實作
 * @param ops 後端操作介面
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_set_backend(const rtk_mqtt_backend_ops_t* ops);

/**
 * @brief 獲取當前 MQTT 後端實作
 * @return 後端操作介面指標，如果未設定則返回 NULL
 */
const rtk_mqtt_backend_ops_t* rtk_mqtt_get_backend(void);

/**
 * @brief 註冊自定義 MQTT 後端
 * @param name 後端名稱
 * @param ops 後端操作介面
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_register_backend(const char* name, const rtk_mqtt_backend_ops_t* ops);

/**
 * @brief 按名稱查找後端
 * @param name 後端名稱
 * @return 後端操作介面指標，如果未找到則返回 NULL
 */
const rtk_mqtt_backend_ops_t* rtk_mqtt_find_backend(const char* name);

// === 統一 MQTT 操作 API ===

/**
 * @brief 配置 MQTT 客戶端
 * @param config 配置參數
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_configure(const rtk_mqtt_config_t* config);

/**
 * @brief 連接到 MQTT Broker
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_connect(void);

/**
 * @brief 斷開 MQTT 連線
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_disconnect(void);

/**
 * @brief 檢查是否已連線
 * @return 1 已連線，0 未連線
 */
int rtk_mqtt_is_connected(void);

/**
 * @brief 重新連線
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_reconnect(void);

/**
 * @brief 發布訊息
 * @param topic 主題
 * @param payload 訊息內容
 * @param payload_len 訊息長度
 * @param qos QoS 等級
 * @param retained 是否設定為 retained
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_publish(const char* topic, const void* payload, size_t payload_len,
                     rtk_mqtt_qos_t qos, int retained);

/**
 * @brief 發布訊息 (進階版本)
 * @param message 訊息結構
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_publish_message(const rtk_mqtt_message_t* message);

/**
 * @brief 訂閱主題
 * @param topic 主題名稱
 * @param qos QoS 等級
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_subscribe(const char* topic, rtk_mqtt_qos_t qos);

/**
 * @brief 取消訂閱主題
 * @param topic 主題名稱
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_unsubscribe(const char* topic);

/**
 * @brief 處理 MQTT 事件 (阻塞版本)
 * @param timeout_ms 超時時間 (毫秒)，-1 表示無限等待
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_loop(int timeout_ms);

/**
 * @brief 處理 MQTT 事件 (非阻塞版本)
 * @param timeout_ms 超時時間 (毫秒)
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_yield(int timeout_ms);

/**
 * @brief 設定訊息接收回調
 * @param callback 回調函式
 * @param user_data 使用者資料
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_set_message_callback(rtk_mqtt_callback_t callback, void* user_data);

/**
 * @brief 設定連線狀態回調
 * @param callback 回調函式
 * @param user_data 使用者資料
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_mqtt_set_connection_callback(rtk_mqtt_connection_callback_t callback, void* user_data);

// === 輔助函式 ===

/**
 * @brief 獲取錯誤碼的描述字串
 * @param error_code 錯誤碼
 * @return 錯誤描述字串
 */
const char* rtk_mqtt_get_error_string(rtk_mqtt_error_t error_code);

/**
 * @brief 獲取最後的錯誤描述
 * @return 錯誤描述字串
 */
const char* rtk_mqtt_get_last_error(void);

/**
 * @brief 獲取連線狀態描述
 * @return 狀態描述字串
 */
const char* rtk_mqtt_get_connection_status_string(void);

/**
 * @brief 建立預設配置
 * @param config 配置結構指標
 * @param broker_host Broker 主機
 * @param broker_port Broker 埠號
 * @param client_id 客戶端 ID
 */
void rtk_mqtt_create_default_config(rtk_mqtt_config_t* config, 
                                   const char* broker_host, int broker_port, 
                                   const char* client_id);

/**
 * @brief 驗證配置參數
 * @param config 配置參數
 * @return RTK_MQTT_SUCCESS 有效，其他值表示無效
 */
int rtk_mqtt_validate_config(const rtk_mqtt_config_t* config);

#ifdef __cplusplus
}
#endif

#endif // RTK_MQTT_CLIENT_H
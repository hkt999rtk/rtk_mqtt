#ifndef RTK_PUBSUB_ADAPTER_H
#define RTK_PUBSUB_ADAPTER_H

#include "rtk_mqtt_client.h"
#include "rtk_network_interface.h"

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @file rtk_pubsub_adapter.h
 * @brief PubSubClient MQTT 後端適配器介面
 * 
 * 為 PubSubClient 的 C++ API 提供 C 語言包裝，
 * 使其能夠與 RTK MQTT Framework 整合
 */

// === 配置常數 ===

#define RTK_PUBSUB_MAX_PACKET_SIZE      512
#define RTK_PUBSUB_DEFAULT_KEEP_ALIVE   60
#define RTK_PUBSUB_DEFAULT_TIMEOUT      5000

// === 錯誤碼 ===

typedef enum {
    RTK_PUBSUB_SUCCESS = 0,
    RTK_PUBSUB_ERROR_INVALID_PARAM = -1,
    RTK_PUBSUB_ERROR_NOT_CONNECTED = -2,
    RTK_PUBSUB_ERROR_CONNECTION_FAILED = -3,
    RTK_PUBSUB_ERROR_MEMORY = -4,
    RTK_PUBSUB_ERROR_TIMEOUT = -5,
    RTK_PUBSUB_ERROR_NETWORK = -6,
    RTK_PUBSUB_ERROR_PROTOCOL = -7,
    RTK_PUBSUB_ERROR_UNKNOWN = -99
} rtk_pubsub_error_t;

// === 回調函式類型 ===

/**
 * @brief PubSubClient 訊息回調函式類型
 * @param topic 主題名稱
 * @param payload 訊息內容
 * @param length 訊息長度
 */
typedef void (*rtk_pubsub_callback_t)(char* topic, uint8_t* payload, unsigned int length);

// === 公開 API ===

/**
 * @brief 初始化 PubSubClient 適配器
 * @param config MQTT 配置
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_init(const rtk_mqtt_config_t* config);

/**
 * @brief 清理 PubSubClient 適配器
 */
void rtk_pubsub_cleanup(void);

/**
 * @brief 連接到 MQTT Broker
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_connect(void);

/**
 * @brief 斷開 MQTT 連線
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_disconnect(void);

/**
 * @brief 檢查是否已連線
 * @return 1 已連線，0 未連線
 */
int rtk_pubsub_is_connected(void);

/**
 * @brief 重新連線
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_reconnect(void);

/**
 * @brief 發布訊息
 * @param message MQTT 訊息結構
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_publish(const rtk_mqtt_message_t* message);

/**
 * @brief 訂閱主題
 * @param topic 主題名稱
 * @param qos QoS 等級
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_subscribe(const char* topic, rtk_mqtt_qos_t qos);

/**
 * @brief 取消訂閱主題
 * @param topic 主題名稱
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_unsubscribe(const char* topic);

/**
 * @brief 處理 MQTT 事件 (阻塞版本)
 * @param timeout_ms 超時時間 (毫秒)
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_loop(int timeout_ms);

/**
 * @brief 處理 MQTT 事件 (非阻塞版本)
 * @param timeout_ms 超時時間 (毫秒)
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_yield(int timeout_ms);

/**
 * @brief 設定訊息回調函式
 * @param callback 回調函式
 * @param user_data 使用者資料
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_set_callback(rtk_mqtt_callback_t callback, void* user_data);

/**
 * @brief 設定網路介面
 * @param network_interface 網路介面
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_set_network_interface(const rtk_network_interface_t* network_interface);

/**
 * @brief 設定 Last Will Testament
 * @param topic LWT 主題
 * @param payload LWT 訊息
 * @param len 訊息長度
 * @param qos QoS 等級
 * @param retained 是否為 retained 訊息
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_set_will(const char* topic, const void* payload, size_t len, 
                        rtk_mqtt_qos_t qos, int retained);

/**
 * @brief 清除 Last Will Testament
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_clear_will(void);

/**
 * @brief 獲取連線狀態
 * @return 連線狀態碼
 */
int rtk_pubsub_get_connection_status(void);

/**
 * @brief 獲取最後的錯誤描述
 * @return 錯誤描述字串
 */
const char* rtk_pubsub_get_last_error(void);

/**
 * @brief 設定封包大小限制
 * @param size 封包大小 (位元組)
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_set_packet_size(int size);

/**
 * @brief 設定 Keep-Alive 間隔
 * @param seconds Keep-Alive 間隔 (秒)
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_set_keep_alive(int seconds);

/**
 * @brief 設定連線超時
 * @param timeout_ms 超時時間 (毫秒)
 * @return RTK_PUBSUB_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_set_timeout(int timeout_ms);

// === 輔助函式 ===

/**
 * @brief 獲取錯誤碼的描述字串
 * @param error_code 錯誤碼
 * @return 錯誤描述字串
 */
const char* rtk_pubsub_get_error_string(rtk_pubsub_error_t error_code);

/**
 * @brief 獲取 PubSubClient 版本資訊
 * @return 版本字串
 */
const char* rtk_pubsub_get_version(void);

/**
 * @brief 檢查 PubSubClient 是否可用
 * @return 1 可用，0 不可用
 */
int rtk_pubsub_is_available(void);

// === RTK MQTT Framework 整合 ===

/**
 * @brief 獲取 PubSubClient 的 RTK MQTT 後端操作介面
 * @return 後端操作介面指標
 */
const rtk_mqtt_backend_ops_t* rtk_pubsub_get_mqtt_backend_ops(void);

/**
 * @brief 註冊 PubSubClient 後端到 RTK MQTT Framework
 * @return RTK_MQTT_SUCCESS 成功，其他值表示失敗
 */
int rtk_pubsub_register_mqtt_backend(void);

#ifdef __cplusplus
}
#endif

#endif // RTK_PUBSUB_ADAPTER_H
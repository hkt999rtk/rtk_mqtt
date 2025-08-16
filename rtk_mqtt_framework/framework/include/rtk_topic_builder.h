#ifndef RTK_TOPIC_BUILDER_H
#define RTK_TOPIC_BUILDER_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @file rtk_topic_builder.h
 * @brief RTK MQTT Topic 建構器
 * 
 * 依據 RTK MQTT 診斷規格建構標準 topic 路徑
 * 格式: rtk/v1/{tenant}/{site}/{device_id}/{message_type}
 */

// Topic 類型定義
typedef enum {
    RTK_TOPIC_STATE,        // state - 設備狀態摘要 (retained)
    RTK_TOPIC_TELEMETRY,    // telemetry/{metric} - 遙測資料
    RTK_TOPIC_EVENT,        // evt/{event_type} - 事件/告警
    RTK_TOPIC_ATTRIBUTE,    // attr - 設備屬性 (retained)
    RTK_TOPIC_CMD_REQ,      // cmd/req - 命令請求 (下行)
    RTK_TOPIC_CMD_ACK,      // cmd/ack - 命令確認 (上行)
    RTK_TOPIC_CMD_RES,      // cmd/res - 命令結果 (上行)
    RTK_TOPIC_LWT,          // lwt - Last Will Testament (retained)
    RTK_TOPIC_GROUP_CMD     // group/{group_id}/cmd/req - 群組命令
} rtk_topic_type_t;

// Topic 建構配置
typedef struct rtk_topic_config {
    char tenant[64];        // 租戶/場域
    char site[64];          // 站點
    char device_id[64];     // 設備 ID
    char group_id[64];      // 群組 ID (群組命令使用)
} rtk_topic_config_t;

/**
 * 設定 Topic 建構配置
 * @param config Topic 配置
 * @return 0 成功，< 0 失敗
 */
int rtk_topic_set_config(const rtk_topic_config_t* config);

/**
 * 建構標準 topic 路徑
 * @param type Topic 類型
 * @param metric_or_event 遙測指標名稱或事件類型 (可選)
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return 建構的 topic 長度，< 0 失敗
 */
int rtk_topic_build(rtk_topic_type_t type, const char* metric_or_event, 
                    char* buffer, size_t buffer_size);

/**
 * 建構狀態 topic: rtk/v1/{tenant}/{site}/{device_id}/state
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return topic 長度，< 0 失敗
 */
int rtk_topic_build_state(char* buffer, size_t buffer_size);

/**
 * 建構遙測 topic: rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}
 * @param metric 遙測指標名稱
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return topic 長度，< 0 失敗
 */
int rtk_topic_build_telemetry(const char* metric, char* buffer, size_t buffer_size);

/**
 * 建構事件 topic: rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}
 * @param event_type 事件類型
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return topic 長度，< 0 失敗
 */
int rtk_topic_build_event(const char* event_type, char* buffer, size_t buffer_size);

/**
 * 建構屬性 topic: rtk/v1/{tenant}/{site}/{device_id}/attr
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return topic 長度，< 0 失敗
 */
int rtk_topic_build_attribute(char* buffer, size_t buffer_size);

/**
 * 建構命令請求 topic: rtk/v1/{tenant}/{site}/{device_id}/cmd/req
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return topic 長度，< 0 失敗
 */
int rtk_topic_build_cmd_req(char* buffer, size_t buffer_size);

/**
 * 建構命令確認 topic: rtk/v1/{tenant}/{site}/{device_id}/cmd/ack
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return topic 長度，< 0 失敗
 */
int rtk_topic_build_cmd_ack(char* buffer, size_t buffer_size);

/**
 * 建構命令結果 topic: rtk/v1/{tenant}/{site}/{device_id}/cmd/res
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return topic 長度，< 0 失敗
 */
int rtk_topic_build_cmd_res(char* buffer, size_t buffer_size);

/**
 * 建構 LWT topic: rtk/v1/{tenant}/{site}/{device_id}/lwt
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return topic 長度，< 0 失敗
 */
int rtk_topic_build_lwt(char* buffer, size_t buffer_size);

/**
 * 建構群組命令 topic: rtk/v1/{tenant}/{site}/group/{group_id}/cmd/req
 * @param group_id 群組 ID
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return topic 長度，< 0 失敗
 */
int rtk_topic_build_group_cmd(const char* group_id, char* buffer, size_t buffer_size);

/**
 * 解析 topic 路徑，提取組成部分
 * @param topic 完整 topic 路徑
 * @param config 解析出的配置 (輸出)
 * @param type 解析出的 topic 類型 (輸出)
 * @param metric_or_event 解析出的指標或事件名稱 (輸出，可選)
 * @param metric_buffer_size 指標緩衝區大小
 * @return 0 成功，< 0 失敗
 */
int rtk_topic_parse(const char* topic, rtk_topic_config_t* config, 
                    rtk_topic_type_t* type, char* metric_or_event, 
                    size_t metric_buffer_size);

/**
 * 驗證 topic 路徑是否符合 RTK 規格
 * @param topic topic 路徑
 * @return 1 有效，0 無效
 */
int rtk_topic_is_valid(const char* topic);

/**
 * 建構訂閱用的通配符 topic
 * @param pattern 訂閱模式類型
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return topic 長度，< 0 失敗
 */
typedef enum {
    RTK_SUB_ALL_DEVICES,       // rtk/v1/{tenant}/{site}/+/state
    RTK_SUB_ALL_EVENTS,        // rtk/v1/{tenant}/{site}/+/evt/#
    RTK_SUB_ALL_TELEMETRY,     // rtk/v1/{tenant}/{site}/+/telemetry/#
    RTK_SUB_ALL_COMMANDS,      // rtk/v1/{tenant}/{site}/+/cmd/#
    RTK_SUB_DEVICE_ALL,        // rtk/v1/{tenant}/{site}/{device_id}/#
    RTK_SUB_GLOBAL_MONITOR     // rtk/v1/+/+/+/evt/#
} rtk_subscribe_pattern_t;

int rtk_topic_build_subscribe_pattern(rtk_subscribe_pattern_t pattern, 
                                       char* buffer, size_t buffer_size);

#ifdef __cplusplus
}
#endif

#endif // RTK_TOPIC_BUILDER_H
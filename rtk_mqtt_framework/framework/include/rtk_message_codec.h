#ifndef RTK_MESSAGE_CODEC_H
#define RTK_MESSAGE_CODEC_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @file rtk_message_codec.h
 * @brief RTK MQTT 訊息編解碼器
 * 
 * 處理 JSON 訊息的編碼與解碼，自動添加共通欄位 (schema, ts, trace)
 * 符合 RTK MQTT 診斷規格 v1.0
 */

// 訊息類型定義
typedef enum {
    RTK_MSG_STATE,          // 狀態訊息
    RTK_MSG_TELEMETRY,      // 遙測訊息
    RTK_MSG_EVENT,          // 事件訊息
    RTK_MSG_ATTRIBUTE,      // 屬性訊息
    RTK_MSG_COMMAND_REQ,    // 命令請求
    RTK_MSG_COMMAND_ACK,    // 命令確認
    RTK_MSG_COMMAND_RES,    // 命令結果
    RTK_MSG_LWT             // Last Will Testament
} rtk_message_type_t;

// 嚴重程度等級
typedef enum {
    RTK_SEVERITY_INFO,      // 資訊
    RTK_SEVERITY_WARNING,   // 警告
    RTK_SEVERITY_ERROR,     // 錯誤
    RTK_SEVERITY_CRITICAL   // 嚴重
} rtk_severity_level_t;

// 追蹤資訊
typedef struct rtk_trace_info {
    char req_id[64];            // 請求唯一識別碼
    char correlation_id[64];    // 關聯識別碼
    char span_id[64];           // 分散式追蹤區段識別碼
} rtk_trace_info_t;

// 共通訊息標頭
typedef struct rtk_message_header {
    char schema[64];            // Schema 版本 (如 "state/1.0")
    int64_t timestamp;          // Unix timestamp (毫秒)
    rtk_trace_info_t trace;     // 追蹤資訊 (可選)
    int has_trace;              // 是否包含追蹤資訊
} rtk_message_header_t;

// 狀態訊息
typedef struct rtk_state_message {
    rtk_message_header_t header;
    char health[16];            // ok, warn, error
    char fw_version[32];        // 韌體版本
    int uptime_seconds;         // 運行時間 (秒)
    float cpu_usage;            // CPU 使用率 (%)
    float memory_usage;         // 記憶體使用率 (%)
    float temperature;          // 溫度 (°C)
    char custom_data[1024];     // 自訂 JSON 資料
} rtk_state_message_t;

// 事件訊息
typedef struct rtk_event_message {
    rtk_message_header_t header;
    rtk_severity_level_t severity;
    char event_type[64];        // 事件類型
    int sequence;               // 序號
    char message[256];          // 事件描述
    char source[64];            // 事件來源
    char custom_data[1024];     // 自訂 JSON 資料
} rtk_event_message_t;

// 命令訊息
typedef struct rtk_command_message {
    rtk_message_header_t header;
    char id[64];                // 命令 ID
    char operation[64];         // 操作名稱
    char args[1024];            // 參數 JSON
    int timeout_ms;             // 超時時間 (毫秒)
    char expect[16];            // ack, result, none
    char reply_to[256];         // 回覆目標 (可選)
} rtk_command_message_t;

// 命令回應訊息
typedef struct rtk_command_response {
    rtk_message_header_t header;
    char id[64];                // 命令 ID
    int ok;                     // 是否成功
    char result[1024];          // 結果 JSON
    char progress[64];          // 進度資訊
    char error_code[32];        // 錯誤碼
    char error_message[256];    // 錯誤訊息
} rtk_command_response_t;

// === 編碼 API ===

/**
 * 編碼狀態訊息為 JSON
 * @param message 狀態訊息
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return JSON 長度，< 0 失敗
 */
int rtk_encode_state_message(const rtk_state_message_t* message, 
                             char* buffer, size_t buffer_size);

/**
 * 編碼事件訊息為 JSON
 * @param message 事件訊息
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return JSON 長度，< 0 失敗
 */
int rtk_encode_event_message(const rtk_event_message_t* message,
                            char* buffer, size_t buffer_size);

/**
 * 編碼命令訊息為 JSON
 * @param message 命令訊息
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return JSON 長度，< 0 失敗
 */
int rtk_encode_command_message(const rtk_command_message_t* message,
                              char* buffer, size_t buffer_size);

/**
 * 編碼命令回應為 JSON
 * @param response 命令回應
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return JSON 長度，< 0 失敗
 */
int rtk_encode_command_response(const rtk_command_response_t* response,
                               char* buffer, size_t buffer_size);

/**
 * 編碼 LWT 訊息為 JSON
 * @param status 狀態 ("online" 或 "offline")
 * @param reason 原因 (可選)
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return JSON 長度，< 0 失敗
 */
int rtk_encode_lwt_message(const char* status, const char* reason,
                          char* buffer, size_t buffer_size);

/**
 * 編碼通用 JSON 訊息 (自動添加共通欄位)
 * @param schema Schema 版本字串
 * @param custom_json 自訂 JSON 資料 (不含共通欄位)
 * @param trace 追蹤資訊 (可選)
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return JSON 長度，< 0 失敗
 */
int rtk_encode_generic_message(const char* schema, const char* custom_json,
                              const rtk_trace_info_t* trace,
                              char* buffer, size_t buffer_size);

// === 解碼 API ===

/**
 * 解碼 JSON 訊息標頭
 * @param json JSON 字串
 * @param header 解碼出的標頭 (輸出)
 * @return 0 成功，< 0 失敗
 */
int rtk_decode_message_header(const char* json, rtk_message_header_t* header);

/**
 * 解碼狀態訊息
 * @param json JSON 字串
 * @param message 解碼出的狀態訊息 (輸出)
 * @return 0 成功，< 0 失敗
 */
int rtk_decode_state_message(const char* json, rtk_state_message_t* message);

/**
 * 解碼事件訊息
 * @param json JSON 字串
 * @param message 解碼出的事件訊息 (輸出)
 * @return 0 成功，< 0 失敗
 */
int rtk_decode_event_message(const char* json, rtk_event_message_t* message);

/**
 * 解碼命令訊息
 * @param json JSON 字串
 * @param message 解碼出的命令訊息 (輸出)
 * @return 0 成功，< 0 失敗
 */
int rtk_decode_command_message(const char* json, rtk_command_message_t* message);

/**
 * 解碼命令回應
 * @param json JSON 字串
 * @param response 解碼出的命令回應 (輸出)
 * @return 0 成功，< 0 失敗
 */
int rtk_decode_command_response(const char* json, rtk_command_response_t* response);

// === 輔助函式 ===

/**
 * 取得當前時間戳 (毫秒)
 * @return Unix timestamp (毫秒)
 */
int64_t rtk_get_current_timestamp(void);

/**
 * 產生唯一請求 ID
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return 產生的 ID 長度，< 0 失敗
 */
int rtk_generate_request_id(char* buffer, size_t buffer_size);

/**
 * 驗證 Schema 版本格式
 * @param schema Schema 字串
 * @return 1 有效，0 無效
 */
int rtk_validate_schema(const char* schema);

/**
 * 從 JSON 字串提取特定欄位值
 * @param json JSON 字串
 * @param field_name 欄位名稱
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return 欄位值長度，< 0 失敗或不存在
 */
int rtk_extract_json_field(const char* json, const char* field_name,
                          char* buffer, size_t buffer_size);

/**
 * 合併兩個 JSON 物件
 * @param json1 第一個 JSON 物件
 * @param json2 第二個 JSON 物件
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return 合併後的 JSON 長度，< 0 失敗
 */
int rtk_merge_json_objects(const char* json1, const char* json2,
                          char* buffer, size_t buffer_size);

/**
 * 轉換嚴重程度等級為字串
 * @param severity 嚴重程度等級
 * @return 嚴重程度字串
 */
const char* rtk_severity_to_string(rtk_severity_level_t severity);

/**
 * 從字串解析嚴重程度等級
 * @param severity_str 嚴重程度字串
 * @return 嚴重程度等級，失敗時返回 RTK_SEVERITY_INFO
 */
rtk_severity_level_t rtk_severity_from_string(const char* severity_str);

#ifdef __cplusplus
}
#endif

#endif // RTK_MESSAGE_CODEC_H
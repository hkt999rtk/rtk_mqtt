#ifndef RTK_SCHEMA_VALIDATOR_H
#define RTK_SCHEMA_VALIDATOR_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @file rtk_schema_validator.h
 * @brief RTK MQTT Schema 驗證器
 * 
 * 提供 JSON Schema 驗證功能，確保訊息格式符合 RTK MQTT 規格
 * 支援版本管理與內建 Schema 定義
 */

// Schema 類型定義
typedef enum {
    RTK_SCHEMA_STATE_V1_0,              // state/1.0
    RTK_SCHEMA_TELEMETRY_WIFI_SCAN_V1_0, // telemetry.wifi.scan_result/1.0
    RTK_SCHEMA_EVT_WIFI_ROAM_MISS_V1_0,  // evt.wifi.roam_miss/1.0
    RTK_SCHEMA_EVT_WIFI_CONNECT_FAIL_V1_0, // evt.wifi.connect_fail/1.0
    RTK_SCHEMA_EVT_WIFI_ARP_LOSS_V1_0,   // evt.wifi.arp_loss/1.0
    RTK_SCHEMA_CMD_DIAGNOSIS_GET_V1_0,   // cmd.diagnosis.get/1.0
    RTK_SCHEMA_CMD_DEVICE_REBOOT_V1_0,   // cmd.device.reboot/1.0
    RTK_SCHEMA_ATTR_V1_0,                // attr/1.0
    RTK_SCHEMA_LWT_V1_0,                 // lwt/1.0
    RTK_SCHEMA_CUSTOM                    // 自訂 Schema
} rtk_schema_type_t;

// Schema 定義結構
typedef struct rtk_schema_definition {
    char name[64];                  // Schema 名稱 (如 "state/1.0")
    char version[16];               // 版本號
    char description[256];          // 描述
    const char* json_schema;        // JSON Schema 內容
    size_t schema_length;           // Schema 長度
} rtk_schema_definition_t;

// 驗證結果
typedef struct rtk_validation_result {
    int is_valid;                   // 是否有效
    char error_message[512];        // 錯誤訊息
    char error_path[256];           // 錯誤位置
    int error_line;                 // 錯誤行數 (如果適用)
} rtk_validation_result_t;

// === Schema 管理 API ===

/**
 * 初始化 Schema 驗證器
 * @return 0 成功，< 0 失敗
 */
int rtk_schema_validator_init(void);

/**
 * 清理 Schema 驗證器資源
 */
void rtk_schema_validator_cleanup(void);

/**
 * 註冊內建 Schema 定義
 * @return 0 成功，< 0 失敗
 */
int rtk_schema_register_builtin_schemas(void);

/**
 * 註冊自訂 Schema
 * @param schema_def Schema 定義
 * @return 0 成功，< 0 失敗
 */
int rtk_schema_register_custom(const rtk_schema_definition_t* schema_def);

/**
 * 根據名稱查找 Schema
 * @param schema_name Schema 名稱 (如 "state/1.0")
 * @return Schema 定義指標，NULL 表示未找到
 */
const rtk_schema_definition_t* rtk_schema_find_by_name(const char* schema_name);

/**
 * 根據類型查找 Schema
 * @param schema_type Schema 類型
 * @return Schema 定義指標，NULL 表示未找到
 */
const rtk_schema_definition_t* rtk_schema_find_by_type(rtk_schema_type_t schema_type);

/**
 * 列出所有已註冊的 Schema
 * @param schemas 輸出 Schema 陣列
 * @param max_count 最大 Schema 數量
 * @return 實際 Schema 數量
 */
int rtk_schema_list_all(const rtk_schema_definition_t** schemas, int max_count);

// === 驗證 API ===

/**
 * 驗證 JSON 訊息
 * @param json JSON 字串
 * @param schema_name Schema 名稱
 * @param result 驗證結果 (輸出)
 * @return 0 成功，< 0 失敗
 */
int rtk_schema_validate_json(const char* json, const char* schema_name, 
                            rtk_validation_result_t* result);

/**
 * 驗證 JSON 訊息 (使用 Schema 類型)
 * @param json JSON 字串
 * @param schema_type Schema 類型
 * @param result 驗證結果 (輸出)
 * @return 0 成功，< 0 失敗
 */
int rtk_schema_validate_json_by_type(const char* json, rtk_schema_type_t schema_type,
                                    rtk_validation_result_t* result);

/**
 * 自動檢測並驗證 JSON 訊息
 * 從 JSON 中的 "schema" 欄位自動判斷 Schema 類型
 * @param json JSON 字串
 * @param result 驗證結果 (輸出)
 * @return 0 成功，< 0 失敗
 */
int rtk_schema_auto_validate_json(const char* json, rtk_validation_result_t* result);

/**
 * 快速驗證 - 只檢查必要欄位
 * @param json JSON 字串
 * @param schema_name Schema 名稱
 * @return 1 有效，0 無效
 */
int rtk_schema_quick_validate(const char* json, const char* schema_name);

// === 輔助功能 ===

/**
 * 檢查 Schema 版本相容性
 * @param schema_v1 第一個 Schema 版本
 * @param schema_v2 第二個 Schema 版本
 * @return 1 相容，0 不相容
 */
int rtk_schema_check_compatibility(const char* schema_v1, const char* schema_v2);

/**
 * 從 JSON 中提取 Schema 名稱
 * @param json JSON 字串
 * @param buffer 輸出緩衝區
 * @param buffer_size 緩衝區大小
 * @return Schema 名稱長度，< 0 失敗
 */
int rtk_schema_extract_name_from_json(const char* json, char* buffer, size_t buffer_size);

/**
 * 驗證 Schema 名稱格式
 * @param schema_name Schema 名稱
 * @return 1 有效，0 無效
 */
int rtk_schema_validate_name_format(const char* schema_name);

/**
 * 解析 Schema 版本
 * @param schema_name Schema 名稱
 * @param major 主版本號 (輸出)
 * @param minor 次版本號 (輸出)
 * @return 0 成功，< 0 失敗
 */
int rtk_schema_parse_version(const char* schema_name, int* major, int* minor);

/**
 * 比較 Schema 版本
 * @param version1 第一個版本
 * @param version2 第二個版本
 * @return 1 version1 > version2, 0 相等, -1 version1 < version2
 */
int rtk_schema_compare_versions(const char* version1, const char* version2);

// === 內建 Schema 常數 ===

// Schema 名稱常數
#define RTK_SCHEMA_NAME_STATE_V1_0              "state/1.0"
#define RTK_SCHEMA_NAME_TELEMETRY_WIFI_SCAN_V1_0 "telemetry.wifi.scan_result/1.0"
#define RTK_SCHEMA_NAME_EVT_WIFI_ROAM_MISS_V1_0  "evt.wifi.roam_miss/1.0"
#define RTK_SCHEMA_NAME_EVT_WIFI_CONNECT_FAIL_V1_0 "evt.wifi.connect_fail/1.0"
#define RTK_SCHEMA_NAME_EVT_WIFI_ARP_LOSS_V1_0   "evt.wifi.arp_loss/1.0"
#define RTK_SCHEMA_NAME_CMD_DIAGNOSIS_GET_V1_0   "cmd.diagnosis.get/1.0"
#define RTK_SCHEMA_NAME_CMD_DEVICE_REBOOT_V1_0   "cmd.device.reboot/1.0"
#define RTK_SCHEMA_NAME_ATTR_V1_0                "attr/1.0"
#define RTK_SCHEMA_NAME_LWT_V1_0                 "lwt/1.0"

// === 錯誤碼 ===
#define RTK_SCHEMA_SUCCESS              0
#define RTK_SCHEMA_ERROR_INVALID_PARAM -1
#define RTK_SCHEMA_ERROR_NOT_FOUND     -2
#define RTK_SCHEMA_ERROR_INVALID_JSON  -3
#define RTK_SCHEMA_ERROR_VALIDATION    -4
#define RTK_SCHEMA_ERROR_MEMORY        -5
#define RTK_SCHEMA_ERROR_VERSION       -6

/**
 * 取得錯誤描述
 * @param error_code 錯誤碼
 * @return 錯誤描述字串
 */
const char* rtk_schema_get_error_string(int error_code);

#ifdef __cplusplus
}
#endif

#endif // RTK_SCHEMA_VALIDATOR_H
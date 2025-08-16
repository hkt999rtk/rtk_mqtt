#ifndef RTK_DEVICE_PLUGIN_H
#define RTK_DEVICE_PLUGIN_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @file rtk_device_plugin.h
 * @brief RTK 設備插件標準介面
 * 
 * 定義統一的設備插件 API，支援狀態回報、遙測、事件處理與命令執行
 * 符合 RTK MQTT 診斷規格 v1.0
 */

// === 基本資料結構 ===

// 設備基本資訊
typedef struct rtk_device_info {
    char id[64];                    // 設備 ID (全域唯一)
    char type[32];                  // 設備類型
    char model[64];                 // 設備型號
    char serial_number[32];         // 序號
    char hw_version[16];            // 硬體版本
    char fw_version[16];            // 韌體版本
    int capability_count;           // 能力數量
    char capabilities[16][32];      // 支援的能力列表
} rtk_device_info_t;

// 插件配置
typedef struct rtk_plugin_config {
    // MQTT 配置
    char mqtt_broker[256];          // MQTT Broker 地址
    int mqtt_port;                  // MQTT 埠號
    char device_id[64];             // 設備 ID
    char tenant[64];                // 租戶
    char site[64];                  // 站點
    char mqtt_username[64];         // MQTT 使用者名稱 (可選)
    char mqtt_password[128];        // MQTT 密碼 (可選)
    
    // 插件特定配置 (JSON 格式)
    char plugin_config[1024];       // 插件特定配置
    
    // 遙測配置
    int telemetry_interval;         // 遙測上報間隔 (秒)
    
    // 事件配置
    int event_cooldown;             // 事件冷卻時間 (秒)
} rtk_plugin_config_t;

// === 插件介面定義 ===

// 設備插件虛函式表
typedef struct rtk_device_plugin_vtable {
    // === 基本資訊 ===
    int (*get_device_info)(rtk_device_info_t* info);
    int (*get_capabilities)(char** capabilities, int* count);
    
    // === 狀態回報 ===
    int (*get_state)(char** json_state, size_t* len);
    int (*get_attributes)(char** json_attrs, size_t* len);
    
    // === 遙測資料 ===
    int (*get_telemetry)(const char* metric, char** json_data, size_t* len);
    int (*list_telemetry_metrics)(char** metrics, int* count);
    
    // === 事件處理 ===
    int (*on_event_trigger)(const char* event_type, const char* data);
    int (*get_supported_events)(char** events, int* count);
    
    // === 命令處理 ===
    int (*handle_command)(const char* cmd_json, char** response_json, size_t* len);
    int (*get_supported_commands)(char** commands, int* count);
    
    // === 生命週期管理 ===
    int (*initialize)(const rtk_plugin_config_t* config);
    int (*start)(void);
    int (*stop)(void);
    int (*health_check)(void);
    
    // === 記憶體管理 ===
    void (*free_json_string)(char* json_str);
} rtk_device_plugin_vtable_t;

// === 插件註冊介面 ===

// 每個插件必須實作的註冊函式
typedef const rtk_device_plugin_vtable_t* (*rtk_plugin_get_vtable_func)(void);
typedef const char* (*rtk_plugin_get_version_func)(void);
typedef const char* (*rtk_plugin_get_name_func)(void);

// 插件資訊
typedef struct rtk_plugin_info {
    char name[64];                  // 插件名稱
    char version[16];               // 插件版本
    char description[256];          // 插件描述
    const rtk_device_plugin_vtable_t* vtable; // 函式表
    void* handle;                   // 動態庫句柄 (內部使用)
} rtk_plugin_info_t;

// === 插件管理器 API ===

/**
 * 初始化插件管理器
 * @return 0 成功，< 0 失敗
 */
int rtk_plugin_manager_init(void);

/**
 * 清理插件管理器
 */
void rtk_plugin_manager_cleanup(void);

/**
 * 載入插件 (從動態庫)
 * @param plugin_path 插件路徑
 * @return 0 成功，< 0 失敗
 */
int rtk_plugin_load(const char* plugin_path);

/**
 * 卸載插件
 * @param plugin_name 插件名稱
 * @return 0 成功，< 0 失敗
 */
int rtk_plugin_unload(const char* plugin_name);

/**
 * 查找插件
 * @param plugin_name 插件名稱
 * @return 插件資訊指標，NULL 表示未找到
 */
const rtk_plugin_info_t* rtk_plugin_find(const char* plugin_name);

/**
 * 列出所有已載入的插件
 * @param plugins 輸出插件陣列
 * @param max_count 最大插件數量
 * @return 實際插件數量
 */
int rtk_plugin_list_all(const rtk_plugin_info_t** plugins, int max_count);

// === 插件實例管理 API ===

// 插件實例
typedef struct rtk_plugin_instance {
    char name[64];                  // 實例名稱
    const rtk_plugin_info_t* plugin_info; // 插件資訊
    rtk_plugin_config_t config;     // 配置
    int is_running;                 // 是否正在運行
    void* user_data;                // 使用者資料
} rtk_plugin_instance_t;

/**
 * 建立插件實例
 * @param plugin_name 插件名稱
 * @param instance_name 實例名稱
 * @param config 配置
 * @return 實例指標，NULL 表示失敗
 */
rtk_plugin_instance_t* rtk_plugin_create_instance(const char* plugin_name,
                                                  const char* instance_name,
                                                  const rtk_plugin_config_t* config);

/**
 * 銷毀插件實例
 * @param instance 實例指標
 */
void rtk_plugin_destroy_instance(rtk_plugin_instance_t* instance);

/**
 * 啟動插件實例
 * @param instance 實例指標
 * @return 0 成功，< 0 失敗
 */
int rtk_plugin_start_instance(rtk_plugin_instance_t* instance);

/**
 * 停止插件實例
 * @param instance 實例指標
 * @return 0 成功，< 0 失敗
 */
int rtk_plugin_stop_instance(rtk_plugin_instance_t* instance);

/**
 * 檢查插件實例健康狀態
 * @param instance 實例指標
 * @return 1 健康，0 不健康，< 0 錯誤
 */
int rtk_plugin_health_check(rtk_plugin_instance_t* instance);

// === 插件輔助 API ===

/**
 * 驗證插件介面
 * @param vtable 虛函式表
 * @return 1 有效，0 無效
 */
int rtk_plugin_validate_vtable(const rtk_device_plugin_vtable_t* vtable);

/**
 * 取得插件預設配置
 * @param config 配置結構 (輸出)
 */
void rtk_plugin_get_default_config(rtk_plugin_config_t* config);

/**
 * 從 JSON 檔案載入插件配置
 * @param json_file JSON 檔案路徑
 * @param config 配置結構 (輸出)
 * @return 0 成功，< 0 失敗
 */
int rtk_plugin_load_config_from_file(const char* json_file, rtk_plugin_config_t* config);

/**
 * 儲存插件配置到 JSON 檔案
 * @param config 配置結構
 * @param json_file JSON 檔案路徑
 * @return 0 成功，< 0 失敗
 */
int rtk_plugin_save_config_to_file(const rtk_plugin_config_t* config, const char* json_file);

// === 錯誤碼定義 ===
#define RTK_PLUGIN_SUCCESS              0
#define RTK_PLUGIN_ERROR_INVALID_PARAM -1
#define RTK_PLUGIN_ERROR_NOT_FOUND     -2
#define RTK_PLUGIN_ERROR_LOAD_FAILED   -3
#define RTK_PLUGIN_ERROR_ALREADY_LOADED -4
#define RTK_PLUGIN_ERROR_NOT_RUNNING   -5
#define RTK_PLUGIN_ERROR_INVALID_VTABLE -6
#define RTK_PLUGIN_ERROR_MEMORY        -7
#define RTK_PLUGIN_ERROR_CONFIG        -8

/**
 * 取得錯誤描述
 * @param error_code 錯誤碼
 * @return 錯誤描述字串
 */
const char* rtk_plugin_get_error_string(int error_code);

// === 插件開發巨集 ===

/**
 * 插件註冊巨集 - 插件開發者使用
 * 
 * 使用範例:
 * RTK_PLUGIN_REGISTER(my_plugin, "1.0", "My Device Plugin", &my_plugin_vtable);
 */
#define RTK_PLUGIN_REGISTER(name, version, description, vtable_ptr) \
    extern "C" { \
        const rtk_device_plugin_vtable_t* rtk_plugin_get_vtable(void) { \
            return vtable_ptr; \
        } \
        const char* rtk_plugin_get_version(void) { \
            return version; \
        } \
        const char* rtk_plugin_get_name(void) { \
            return #name; \
        } \
    }

// === 便利函式 ===

/**
 * 呼叫插件函式的安全包裝
 * @param instance 插件實例
 * @param func_name 函式名稱 (用於錯誤訊息)
 * @param func_call 函式呼叫
 * @return 函式返回值，或錯誤碼
 */
#define RTK_PLUGIN_CALL(instance, func_name, func_call) \
    ((instance) && (instance)->plugin_info && (instance)->plugin_info->vtable && \
     (instance)->plugin_info->vtable->func_name) ? \
    (func_call) : RTK_PLUGIN_ERROR_INVALID_VTABLE

/**
 * 安全釋放 JSON 字串
 * @param instance 插件實例
 * @param json_str JSON 字串指標
 */
void rtk_plugin_safe_free_json(rtk_plugin_instance_t* instance, char* json_str);

#ifdef __cplusplus
}
#endif

#endif // RTK_DEVICE_PLUGIN_H
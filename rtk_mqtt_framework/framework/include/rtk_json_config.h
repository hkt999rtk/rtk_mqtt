#ifndef RTK_JSON_CONFIG_H
#define RTK_JSON_CONFIG_H

#include "rtk_platform_compat.h"
#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @file rtk_json_config.h
 * @brief 平台特定 JSON 處理配置
 * 
 * 提供針對不同平台優化的 JSON 處理配置：
 * - FreeRTOS: 輕量化設定，減少記憶體使用
 * - Windows: 高效能設定，利用 Windows 堆疊 API
 * - POSIX: 標準設定，平衡效能與相容性
 */

// === 平台特定 JSON 配置 ===

#ifdef RTK_PLATFORM_FREERTOS
    // FreeRTOS 環境使用輕量化設定
    #define RTK_JSON_MAX_DEPTH          8      /**< 最大解析深度 */
    #define RTK_JSON_BUFFER_SIZE        1024   /**< JSON 緩衝區大小 */
    #define RTK_JSON_POOL_SIZE          4      /**< 記憶體池大小 */
    #define RTK_JSON_STRING_MAX_LEN     256    /**< 最大字串長度 */
    #define RTK_JSON_OBJECT_MAX_ITEMS   16     /**< 物件最大項目數 */
    #define RTK_JSON_ARRAY_MAX_ITEMS    16     /**< 陣列最大項目數 */
    #define RTK_JSON_USE_STATIC_MEMORY  1      /**< 使用靜態記憶體 */
    #define RTK_JSON_ENABLE_FLOAT       0      /**< 禁用浮點數支援 */
    
#elif defined(RTK_PLATFORM_WINDOWS)
    // Windows 環境使用標準設定 + 效能優化
    #define RTK_JSON_MAX_DEPTH          32     /**< 最大解析深度 */
    #define RTK_JSON_BUFFER_SIZE        8192   /**< JSON 緩衝區大小 */
    #define RTK_JSON_POOL_SIZE          16     /**< 記憶體池大小 */
    #define RTK_JSON_STRING_MAX_LEN     2048   /**< 最大字串長度 */
    #define RTK_JSON_OBJECT_MAX_ITEMS   128    /**< 物件最大項目數 */
    #define RTK_JSON_ARRAY_MAX_ITEMS    128    /**< 陣列最大項目數 */
    #define RTK_JSON_USE_STATIC_MEMORY  0      /**< 使用動態記憶體 */
    #define RTK_JSON_ENABLE_FLOAT       1      /**< 啟用浮點數支援 */
    #define RTK_JSON_USE_HEAP_API       1      /**< 使用 Windows 堆疊 API */
    
#else  // RTK_PLATFORM_POSIX
    // Linux/POSIX 環境使用標準設定
    #define RTK_JSON_MAX_DEPTH          32     /**< 最大解析深度 */
    #define RTK_JSON_BUFFER_SIZE        4096   /**< JSON 緩衝區大小 */
    #define RTK_JSON_POOL_SIZE          8      /**< 記憶體池大小 */
    #define RTK_JSON_STRING_MAX_LEN     1024   /**< 最大字串長度 */
    #define RTK_JSON_OBJECT_MAX_ITEMS   64     /**< 物件最大項目數 */
    #define RTK_JSON_ARRAY_MAX_ITEMS    64     /**< 陣列最大項目數 */
    #define RTK_JSON_USE_STATIC_MEMORY  0      /**< 使用動態記憶體 */
    #define RTK_JSON_ENABLE_FLOAT       1      /**< 啟用浮點數支援 */
#endif

// === JSON 記憶體管理配置 ===

#ifdef RTK_JSON_USE_HEAP_API
    // Windows 專用堆疊 API
    #define RTK_JSON_MALLOC(size)       HeapAlloc(GetProcessHeap(), 0, size)
    #define RTK_JSON_FREE(ptr)          HeapFree(GetProcessHeap(), 0, ptr)
    #define RTK_JSON_REALLOC(ptr, size) HeapReAlloc(GetProcessHeap(), 0, ptr, size)
#else
    // 使用平台統一的記憶體管理
    #define RTK_JSON_MALLOC(size)       RTK_MALLOC(size)
    #define RTK_JSON_FREE(ptr)          RTK_FREE(ptr)
    #define RTK_JSON_REALLOC(ptr, size) RTK_REALLOC(ptr, size)
#endif

// === cJSON 庫配置 ===

#ifdef RTK_USE_LIGHTWEIGHT_JSON
    // 使用輕量化 JSON 處理
    #include "rtk_json_minimal.h"
    #define RTK_JSON_LIBRARY_NAME       "RTK JSON Minimal"
#else
    // 使用標準 cJSON 庫
    #include <cJSON.h>
    
    // 配置 cJSON 記憶體管理
    #define CJSON_MALLOC(size)          RTK_JSON_MALLOC(size)
    #define CJSON_FREE(ptr)             RTK_JSON_FREE(ptr)
    #define CJSON_REALLOC(ptr, size)    RTK_JSON_REALLOC(ptr, size)
    
    #define RTK_JSON_LIBRARY_NAME       "cJSON"
#endif

// === JSON 性能優化配置 ===

// 字串預分配大小 (減少重新分配)
#define RTK_JSON_STRING_PREALLOC_SIZE   128

// 是否啟用 JSON 壓縮 (移除空白字符)
#if defined(RTK_PLATFORM_FREERTOS)
    #define RTK_JSON_ENABLE_MINIFY      1
#else
    #define RTK_JSON_ENABLE_MINIFY      0
#endif

// 是否啟用 JSON 驗證 (開發階段建議啟用)
#ifdef RTK_DEBUG
    #define RTK_JSON_ENABLE_VALIDATION  1
#else
    #define RTK_JSON_ENABLE_VALIDATION  0
#endif

// === JSON 記憶體池結構 ===

typedef struct {
    char buffer[RTK_JSON_BUFFER_SIZE];  /**< JSON 緩衝區 */
    int in_use;                         /**< 是否使用中 */
    rtk_mutex_t mutex;                  /**< 互斥鎖 */
    uint32_t allocation_count;          /**< 分配次數 */
    uint32_t last_used_time;            /**< 最後使用時間 */
} rtk_json_buffer_t;

typedef struct {
    rtk_json_buffer_t buffers[RTK_JSON_POOL_SIZE];  /**< 緩衝區池 */
    rtk_mutex_t pool_mutex;                         /**< 池互斥鎖 */
    uint32_t total_allocations;                     /**< 總分配次數 */
    uint32_t peak_usage;                            /**< 峰值使用量 */
    uint32_t current_usage;                         /**< 當前使用量 */
} rtk_json_pool_t;

// === JSON 統計結構 ===

typedef struct {
    uint32_t parse_count;           /**< 解析次數 */
    uint32_t parse_success_count;   /**< 解析成功次數 */
    uint32_t parse_error_count;     /**< 解析錯誤次數 */
    uint32_t create_count;          /**< 建立次數 */
    uint32_t delete_count;          /**< 刪除次數 */
    uint64_t total_parse_time_us;   /**< 總解析時間 (微秒) */
    uint32_t max_parse_time_us;     /**< 最大解析時間 (微秒) */
    uint32_t avg_parse_time_us;     /**< 平均解析時間 (微秒) */
    size_t max_memory_used;         /**< 最大記憶體使用量 */
    size_t current_memory_used;     /**< 當前記憶體使用量 */
} rtk_json_stats_t;

// === 公開 API ===

/**
 * @brief 初始化 JSON 記憶體池
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_json_pool_init(void);

/**
 * @brief 清理 JSON 記憶體池
 */
void rtk_json_pool_cleanup(void);

/**
 * @brief 分配 JSON 緩衝區
 * @return 緩衝區指標，NULL 表示失敗
 */
char* rtk_json_alloc_buffer(void);

/**
 * @brief 釋放 JSON 緩衝區
 * @param buffer 緩衝區指標
 */
void rtk_json_free_buffer(char* buffer);

/**
 * @brief 獲取記憶體池使用量
 * @return 使用量百分比 (0-100)
 */
int rtk_json_get_pool_usage(void);

/**
 * @brief 獲取 JSON 統計資訊
 * @param stats 統計結構指標
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_json_get_stats(rtk_json_stats_t* stats);

/**
 * @brief 重設 JSON 統計資訊
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_json_reset_stats(void);

/**
 * @brief 解析 JSON 字串 (帶統計)
 * @param json JSON 字串
 * @return cJSON 物件指標，NULL 表示失敗
 */
cJSON* rtk_json_parse_with_stats(const char* json);

/**
 * @brief 建立 JSON 字串 (帶統計)
 * @param object cJSON 物件
 * @param minify 是否壓縮
 * @return JSON 字串，NULL 表示失敗
 */
char* rtk_json_print_with_stats(const cJSON* object, int minify);

/**
 * @brief 安全刪除 JSON 物件
 * @param object cJSON 物件指標
 */
void rtk_json_delete_safe(cJSON* object);

/**
 * @brief 安全釋放 JSON 字串
 * @param string JSON 字串指標
 */
void rtk_json_free_string_safe(char* string);

/**
 * @brief 驗證 JSON 格式
 * @param json JSON 字串
 * @param error_msg 錯誤訊息緩衝區
 * @param error_msg_size 錯誤訊息緩衝區大小
 * @return 1 有效，0 無效
 */
int rtk_json_validate_format(const char* json, char* error_msg, size_t error_msg_size);

/**
 * @brief 獲取 JSON 庫資訊
 * @return JSON 庫名稱和版本字串
 */
const char* rtk_json_get_library_info(void);

/**
 * @brief 測試 JSON 效能
 * @param iterations 測試迭代次數
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_json_benchmark(int iterations);

// === 輔助巨集 ===

/**
 * @brief 安全取得 JSON 字串值
 */
#define RTK_JSON_GET_STRING_SAFE(obj, key, default_val) \
    (cJSON_GetObjectItemCaseSensitive(obj, key) && \
     cJSON_IsString(cJSON_GetObjectItemCaseSensitive(obj, key))) ? \
    cJSON_GetStringValue(cJSON_GetObjectItemCaseSensitive(obj, key)) : (default_val)

/**
 * @brief 安全取得 JSON 數值
 */
#define RTK_JSON_GET_NUMBER_SAFE(obj, key, default_val) \
    (cJSON_GetObjectItemCaseSensitive(obj, key) && \
     cJSON_IsNumber(cJSON_GetObjectItemCaseSensitive(obj, key))) ? \
    cJSON_GetNumberValue(cJSON_GetObjectItemCaseSensitive(obj, key)) : (default_val)

/**
 * @brief 安全取得 JSON 布林值
 */
#define RTK_JSON_GET_BOOL_SAFE(obj, key, default_val) \
    (cJSON_GetObjectItemCaseSensitive(obj, key) && \
     cJSON_IsBool(cJSON_GetObjectItemCaseSensitive(obj, key))) ? \
    cJSON_IsTrue(cJSON_GetObjectItemCaseSensitive(obj, key)) : (default_val)

/**
 * @brief 檢查 JSON 物件是否包含指定鍵
 */
#define RTK_JSON_HAS_KEY(obj, key) \
    (cJSON_GetObjectItemCaseSensitive(obj, key) != NULL)

/**
 * @brief 安全新增字串到 JSON 物件
 */
#define RTK_JSON_ADD_STRING_SAFE(obj, key, value) \
    do { \
        if ((value) != NULL) { \
            cJSON_AddStringToObject(obj, key, value); \
        } \
    } while(0)

/**
 * @brief 安全新增數值到 JSON 物件
 */
#define RTK_JSON_ADD_NUMBER_SAFE(obj, key, value) \
    cJSON_AddNumberToObject(obj, key, value)

/**
 * @brief 安全新增布林值到 JSON 物件
 */
#define RTK_JSON_ADD_BOOL_SAFE(obj, key, value) \
    cJSON_AddBoolToObject(obj, key, value)

#ifdef __cplusplus
}
#endif

#endif // RTK_JSON_CONFIG_H
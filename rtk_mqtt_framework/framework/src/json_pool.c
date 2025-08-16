/**
 * @file json_pool.c
 * @brief JSON 記憶體池管理實作
 * 
 * 提供跨平台的 JSON 記憶體池管理功能，包括：
 * - 平台特定的記憶體池初始化和清理
 * - JSON 緩衝區的分配和釋放
 * - 統計資訊收集和性能監控
 * - 安全的 JSON 解析和建立操作
 */

#include "rtk_json_config.h"
#include "rtk_platform_compat.h"
#include <string.h>
#include <stdio.h>

#ifdef RTK_PLATFORM_WINDOWS
    #include <windows.h>
    #include <winbase.h>
#elif defined(RTK_PLATFORM_FREERTOS)
    #include "FreeRTOS.h"
    #include "task.h"
#else
    #include <sys/time.h>
    #include <unistd.h>
#endif

// === 全域變數 ===

static rtk_json_pool_t g_json_pool;
static rtk_json_stats_t g_json_stats;
static int g_pool_initialized = 0;

// === 內部輔助函數 ===

/**
 * @brief 獲取當前時間戳 (微秒)
 */
static uint64_t rtk_json_get_timestamp_us(void)
{
#ifdef RTK_PLATFORM_WINDOWS
    LARGE_INTEGER frequency, counter;
    QueryPerformanceFrequency(&frequency);
    QueryPerformanceCounter(&counter);
    return (uint64_t)((counter.QuadPart * 1000000LL) / frequency.QuadPart);
#elif defined(RTK_PLATFORM_FREERTOS)
    // FreeRTOS 使用 tick 轉換為微秒
    TickType_t ticks = xTaskGetTickCount();
    return (uint64_t)(ticks * (1000000 / configTICK_RATE_HZ));
#else
    // POSIX
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return (uint64_t)(tv.tv_sec * 1000000LL + tv.tv_usec);
#endif
}

/**
 * @brief 獲取當前時間戳 (32位，適用於 FreeRTOS)
 */
static uint32_t rtk_json_get_timestamp_32(void)
{
#ifdef RTK_PLATFORM_FREERTOS
    return (uint32_t)xTaskGetTickCount();
#else
    return (uint32_t)(rtk_json_get_timestamp_us() / 1000);  // 轉為毫秒
#endif
}

/**
 * @brief 初始化單一 JSON 緩衝區
 */
static int rtk_json_init_buffer(rtk_json_buffer_t* buffer)
{
    if (!buffer) return RTK_PLATFORM_ERROR_INVALID_PARAM;
    
    memset(buffer->buffer, 0, RTK_JSON_BUFFER_SIZE);
    buffer->in_use = 0;
    buffer->allocation_count = 0;
    buffer->last_used_time = 0;
    
    return rtk_mutex_init(&buffer->mutex);
}

/**
 * @brief 清理單一 JSON 緩衝區
 */
static void rtk_json_cleanup_buffer(rtk_json_buffer_t* buffer)
{
    if (!buffer) return;
    
    rtk_mutex_destroy(&buffer->mutex);
    memset(buffer, 0, sizeof(rtk_json_buffer_t));
}

/**
 * @brief 查找可用的 JSON 緩衝區
 */
static rtk_json_buffer_t* rtk_json_find_available_buffer(void)
{
    for (int i = 0; i < RTK_JSON_POOL_SIZE; i++) {
        rtk_json_buffer_t* buffer = &g_json_pool.buffers[i];
        
        if (rtk_mutex_trylock(&buffer->mutex) == RTK_PLATFORM_SUCCESS) {
            if (!buffer->in_use) {
                buffer->in_use = 1;
                buffer->allocation_count++;
                buffer->last_used_time = rtk_json_get_timestamp_32();
                return buffer;
            }
            rtk_mutex_unlock(&buffer->mutex);
        }
    }
    
    return NULL;  // 沒有可用緩衝區
}

/**
 * @brief 根據緩衝區指標查找緩衝區結構
 */
static rtk_json_buffer_t* rtk_json_find_buffer_by_ptr(const char* ptr)
{
    if (!ptr) return NULL;
    
    for (int i = 0; i < RTK_JSON_POOL_SIZE; i++) {
        rtk_json_buffer_t* buffer = &g_json_pool.buffers[i];
        if (ptr >= buffer->buffer && ptr < buffer->buffer + RTK_JSON_BUFFER_SIZE) {
            return buffer;
        }
    }
    
    return NULL;
}

// === 公開 API 實作 ===

int rtk_json_pool_init(void)
{
    if (g_pool_initialized) {
        return RTK_PLATFORM_SUCCESS;  // 已初始化
    }
    
    // 初始化池互斥鎖
    int result = rtk_mutex_init(&g_json_pool.pool_mutex);
    if (result != RTK_PLATFORM_SUCCESS) {
        return result;
    }
    
    // 初始化所有緩衝區
    for (int i = 0; i < RTK_JSON_POOL_SIZE; i++) {
        result = rtk_json_init_buffer(&g_json_pool.buffers[i]);
        if (result != RTK_PLATFORM_SUCCESS) {
            // 清理已初始化的緩衝區
            for (int j = 0; j < i; j++) {
                rtk_json_cleanup_buffer(&g_json_pool.buffers[j]);
            }
            rtk_mutex_destroy(&g_json_pool.pool_mutex);
            return result;
        }
    }
    
    // 初始化池統計
    g_json_pool.total_allocations = 0;
    g_json_pool.peak_usage = 0;
    g_json_pool.current_usage = 0;
    
    // 初始化全域統計
    memset(&g_json_stats, 0, sizeof(rtk_json_stats_t));
    
#ifdef RTK_USE_HEAP_API
    // Windows 特定初始化
    // 在 Windows 上可以預先配置堆疊
#endif
    
    g_pool_initialized = 1;
    
    RTK_PLATFORM_LOG_INFO("JSON pool initialized with %d buffers (%d bytes each)", 
                          RTK_JSON_POOL_SIZE, RTK_JSON_BUFFER_SIZE);
    
    return RTK_PLATFORM_SUCCESS;
}

void rtk_json_pool_cleanup(void)
{
    if (!g_pool_initialized) {
        return;
    }
    
    rtk_mutex_lock(&g_json_pool.pool_mutex);
    
    // 清理所有緩衝區
    for (int i = 0; i < RTK_JSON_POOL_SIZE; i++) {
        rtk_json_cleanup_buffer(&g_json_pool.buffers[i]);
    }
    
    rtk_mutex_unlock(&g_json_pool.pool_mutex);
    rtk_mutex_destroy(&g_json_pool.pool_mutex);
    
    // 重設統計
    memset(&g_json_pool, 0, sizeof(rtk_json_pool_t));
    memset(&g_json_stats, 0, sizeof(rtk_json_stats_t));
    
    g_pool_initialized = 0;
    
    RTK_PLATFORM_LOG_INFO("JSON pool cleaned up");
}

char* rtk_json_alloc_buffer(void)
{
    if (!g_pool_initialized) {
        if (rtk_json_pool_init() != RTK_PLATFORM_SUCCESS) {
            return NULL;
        }
    }
    
    rtk_mutex_lock(&g_json_pool.pool_mutex);
    
    rtk_json_buffer_t* buffer = rtk_json_find_available_buffer();
    if (!buffer) {
        rtk_mutex_unlock(&g_json_pool.pool_mutex);
        RTK_PLATFORM_LOG_WARNING("JSON pool exhausted, no available buffers");
        return NULL;
    }
    
    // 更新池統計
    g_json_pool.total_allocations++;
    g_json_pool.current_usage++;
    if (g_json_pool.current_usage > g_json_pool.peak_usage) {
        g_json_pool.peak_usage = g_json_pool.current_usage;
    }
    
    rtk_mutex_unlock(&g_json_pool.pool_mutex);
    
    RTK_PLATFORM_LOG_DEBUG("JSON buffer allocated, current usage: %d/%d", 
                           g_json_pool.current_usage, RTK_JSON_POOL_SIZE);
    
    return buffer->buffer;
}

void rtk_json_free_buffer(char* buffer)
{
    if (!buffer || !g_pool_initialized) {
        return;
    }
    
    rtk_mutex_lock(&g_json_pool.pool_mutex);
    
    rtk_json_buffer_t* buf_struct = rtk_json_find_buffer_by_ptr(buffer);
    if (!buf_struct) {
        rtk_mutex_unlock(&g_json_pool.pool_mutex);
        RTK_PLATFORM_LOG_WARNING("Attempting to free invalid JSON buffer");
        return;
    }
    
    rtk_mutex_lock(&buf_struct->mutex);
    
    if (buf_struct->in_use) {
        buf_struct->in_use = 0;
        memset(buf_struct->buffer, 0, RTK_JSON_BUFFER_SIZE);
        g_json_pool.current_usage--;
        
        RTK_PLATFORM_LOG_DEBUG("JSON buffer freed, current usage: %d/%d", 
                               g_json_pool.current_usage, RTK_JSON_POOL_SIZE);
    } else {
        RTK_PLATFORM_LOG_WARNING("Attempting to free already freed JSON buffer");
    }
    
    rtk_mutex_unlock(&buf_struct->mutex);
    rtk_mutex_unlock(&g_json_pool.pool_mutex);
}

int rtk_json_get_pool_usage(void)
{
    if (!g_pool_initialized) {
        return 0;
    }
    
    rtk_mutex_lock(&g_json_pool.pool_mutex);
    int usage_percent = (g_json_pool.current_usage * 100) / RTK_JSON_POOL_SIZE;
    rtk_mutex_unlock(&g_json_pool.pool_mutex);
    
    return usage_percent;
}

int rtk_json_get_stats(rtk_json_stats_t* stats)
{
    if (!stats || !g_pool_initialized) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_mutex_lock(&g_json_pool.pool_mutex);
    
    // 複製統計資料
    memcpy(stats, &g_json_stats, sizeof(rtk_json_stats_t));
    
    // 計算平均解析時間
    if (g_json_stats.parse_success_count > 0) {
        stats->avg_parse_time_us = (uint32_t)(g_json_stats.total_parse_time_us / g_json_stats.parse_success_count);
    }
    
    rtk_mutex_unlock(&g_json_pool.pool_mutex);
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_json_reset_stats(void)
{
    if (!g_pool_initialized) {
        return RTK_PLATFORM_ERROR_INVALID_STATE;
    }
    
    rtk_mutex_lock(&g_json_pool.pool_mutex);
    
    // 保留記憶體相關統計，重設其他統計
    uint32_t total_allocations = g_json_pool.total_allocations;
    uint32_t peak_usage = g_json_pool.peak_usage;
    
    memset(&g_json_stats, 0, sizeof(rtk_json_stats_t));
    
    // 恢復記憶體統計
    g_json_pool.total_allocations = total_allocations;
    g_json_pool.peak_usage = peak_usage;
    
    rtk_mutex_unlock(&g_json_pool.pool_mutex);
    
    RTK_PLATFORM_LOG_INFO("JSON statistics reset");
    
    return RTK_PLATFORM_SUCCESS;
}

cJSON* rtk_json_parse_with_stats(const char* json)
{
    if (!json) {
        return NULL;
    }
    
    uint64_t start_time = rtk_json_get_timestamp_us();
    
    // 嘗試解析 JSON
    cJSON* result = NULL;
    
#if RTK_JSON_ENABLE_VALIDATION
    // 先驗證 JSON 格式
    char error_msg[256];
    if (!rtk_json_validate_format(json, error_msg, sizeof(error_msg))) {
        RTK_PLATFORM_LOG_WARNING("JSON validation failed: %s", error_msg);
        if (g_pool_initialized) {
            rtk_mutex_lock(&g_json_pool.pool_mutex);
            g_json_stats.parse_error_count++;
            rtk_mutex_unlock(&g_json_pool.pool_mutex);
        }
        return NULL;
    }
#endif
    
    result = cJSON_Parse(json);
    
    uint64_t end_time = rtk_json_get_timestamp_us();
    uint32_t parse_time = (uint32_t)(end_time - start_time);
    
    if (g_pool_initialized) {
        rtk_mutex_lock(&g_json_pool.pool_mutex);
        
        g_json_stats.parse_count++;
        if (result) {
            g_json_stats.parse_success_count++;
            g_json_stats.total_parse_time_us += parse_time;
            if (parse_time > g_json_stats.max_parse_time_us) {
                g_json_stats.max_parse_time_us = parse_time;
            }
        } else {
            g_json_stats.parse_error_count++;
        }
        
        rtk_mutex_unlock(&g_json_pool.pool_mutex);
    }
    
    if (!result) {
        RTK_PLATFORM_LOG_WARNING("JSON parse failed, time: %u us", parse_time);
    } else {
        RTK_PLATFORM_LOG_DEBUG("JSON parsed successfully, time: %u us", parse_time);
    }
    
    return result;
}

char* rtk_json_print_with_stats(const cJSON* object, int minify)
{
    if (!object) {
        return NULL;
    }
    
    uint64_t start_time = rtk_json_get_timestamp_us();
    
    char* result = NULL;
    
#if RTK_JSON_ENABLE_MINIFY
    if (minify || RTK_JSON_ENABLE_MINIFY) {
        result = cJSON_PrintUnformatted(object);
    } else {
        result = cJSON_Print(object);
    }
#else
    (void)minify;  // 避免未使用警告
    result = cJSON_Print(object);
#endif
    
    uint64_t end_time = rtk_json_get_timestamp_us();
    uint32_t print_time = (uint32_t)(end_time - start_time);
    
    if (g_pool_initialized) {
        rtk_mutex_lock(&g_json_pool.pool_mutex);
        g_json_stats.create_count++;
        rtk_mutex_unlock(&g_json_pool.pool_mutex);
    }
    
    RTK_PLATFORM_LOG_DEBUG("JSON printed, time: %u us, minify: %d", print_time, minify);
    
    return result;
}

void rtk_json_delete_safe(cJSON* object)
{
    if (!object) {
        return;
    }
    
    if (g_pool_initialized) {
        rtk_mutex_lock(&g_json_pool.pool_mutex);
        g_json_stats.delete_count++;
        rtk_mutex_unlock(&g_json_pool.pool_mutex);
    }
    
    cJSON_Delete(object);
}

void rtk_json_free_string_safe(char* string)
{
    if (!string) {
        return;
    }
    
    // 檢查是否為池中的緩衝區
    if (g_pool_initialized && rtk_json_find_buffer_by_ptr(string)) {
        rtk_json_free_buffer(string);
    } else {
        // 使用 cJSON 的釋放函數
        cJSON_free(string);
    }
}

int rtk_json_validate_format(const char* json, char* error_msg, size_t error_msg_size)
{
    if (!json) {
        if (error_msg && error_msg_size > 0) {
            strncpy(error_msg, "JSON string is NULL", error_msg_size - 1);
            error_msg[error_msg_size - 1] = '\0';
        }
        return 0;
    }
    
    // 基本格式檢查
    size_t len = strlen(json);
    if (len == 0) {
        if (error_msg && error_msg_size > 0) {
            strncpy(error_msg, "JSON string is empty", error_msg_size - 1);
            error_msg[error_msg_size - 1] = '\0';
        }
        return 0;
    }
    
    // 檢查長度限制
    if (len > RTK_JSON_STRING_MAX_LEN) {
        if (error_msg && error_msg_size > 0) {
            snprintf(error_msg, error_msg_size, "JSON string too long: %zu > %d", 
                    len, RTK_JSON_STRING_MAX_LEN);
        }
        return 0;
    }
    
    // 檢查基本結構
    const char* start = json;
    while (*start && (*start == ' ' || *start == '\t' || *start == '\n' || *start == '\r')) {
        start++;
    }
    
    if (*start != '{' && *start != '[') {
        if (error_msg && error_msg_size > 0) {
            strncpy(error_msg, "JSON must start with { or [", error_msg_size - 1);
            error_msg[error_msg_size - 1] = '\0';
        }
        return 0;
    }
    
    // 嘗試使用 cJSON 解析 (更準確的驗證)
    cJSON* test = cJSON_Parse(json);
    if (!test) {
        if (error_msg && error_msg_size > 0) {
            const char* cjson_error = cJSON_GetErrorPtr();
            if (cjson_error) {
                snprintf(error_msg, error_msg_size, "Parse error near: %.32s", cjson_error);
            } else {
                strncpy(error_msg, "Invalid JSON format", error_msg_size - 1);
                error_msg[error_msg_size - 1] = '\0';
            }
        }
        return 0;
    }
    
    cJSON_Delete(test);
    return 1;  // 驗證成功
}

const char* rtk_json_get_library_info(void)
{
    static char info_buffer[256];
    
#ifdef RTK_USE_LIGHTWEIGHT_JSON
    snprintf(info_buffer, sizeof(info_buffer), 
            "%s (Pool: %d buffers x %d bytes)", 
            RTK_JSON_LIBRARY_NAME, RTK_JSON_POOL_SIZE, RTK_JSON_BUFFER_SIZE);
#else
    snprintf(info_buffer, sizeof(info_buffer), 
            "%s %s (Pool: %d buffers x %d bytes)", 
            RTK_JSON_LIBRARY_NAME, cJSON_Version(), 
            RTK_JSON_POOL_SIZE, RTK_JSON_BUFFER_SIZE);
#endif
    
    return info_buffer;
}

int rtk_json_benchmark(int iterations)
{
    if (iterations <= 0 || !g_pool_initialized) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    RTK_PLATFORM_LOG_INFO("Starting JSON benchmark with %d iterations", iterations);
    
    // 測試用的 JSON 字串
    const char* test_json = 
        "{"
        "\"device_id\":\"RTK_TEST_001\","
        "\"timestamp\":1234567890,"
        "\"temperature\":25.5,"
        "\"humidity\":60.2,"
        "\"status\":\"active\","
        "\"sensors\":["
            "{\"id\":1,\"type\":\"temp\",\"value\":25.5},"
            "{\"id\":2,\"type\":\"hum\",\"value\":60.2}"
        "],"
        "\"metadata\":{\"version\":\"1.0\",\"location\":\"lab\"}"
        "}";
    
    uint64_t total_parse_time = 0;
    uint64_t total_print_time = 0;
    int successful_operations = 0;
    
    for (int i = 0; i < iterations; i++) {
        uint64_t start_time = rtk_json_get_timestamp_us();
        
        // 解析測試
        cJSON* json_obj = rtk_json_parse_with_stats(test_json);
        if (!json_obj) {
            continue;
        }
        
        uint64_t parse_end = rtk_json_get_timestamp_us();
        total_parse_time += (parse_end - start_time);
        
        // 列印測試
        char* json_str = rtk_json_print_with_stats(json_obj, 1);
        if (!json_str) {
            rtk_json_delete_safe(json_obj);
            continue;
        }
        
        uint64_t print_end = rtk_json_get_timestamp_us();
        total_print_time += (print_end - parse_end);
        
        // 清理
        rtk_json_free_string_safe(json_str);
        rtk_json_delete_safe(json_obj);
        
        successful_operations++;
    }
    
    if (successful_operations > 0) {
        uint32_t avg_parse_time = (uint32_t)(total_parse_time / successful_operations);
        uint32_t avg_print_time = (uint32_t)(total_print_time / successful_operations);
        
        RTK_PLATFORM_LOG_INFO("JSON benchmark completed:");
        RTK_PLATFORM_LOG_INFO("  Successful operations: %d/%d", successful_operations, iterations);
        RTK_PLATFORM_LOG_INFO("  Average parse time: %u us", avg_parse_time);
        RTK_PLATFORM_LOG_INFO("  Average print time: %u us", avg_print_time);
        RTK_PLATFORM_LOG_INFO("  Total time: %u us", avg_parse_time + avg_print_time);
        RTK_PLATFORM_LOG_INFO("  Pool usage: %d%%", rtk_json_get_pool_usage());
    } else {
        RTK_PLATFORM_LOG_ERROR("JSON benchmark failed - no successful operations");
        return RTK_PLATFORM_ERROR_OPERATION_FAILED;
    }
    
    return RTK_PLATFORM_SUCCESS;
}
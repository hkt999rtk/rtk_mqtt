#ifndef RTK_PLATFORM_COMPAT_H
#define RTK_PLATFORM_COMPAT_H

#include <stdint.h>
#include <stddef.h>
#include <time.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @file rtk_platform_compat.h
 * @brief 多平台相容層 - 統一不同平台的 API
 * 
 * 提供跨平台的記憶體管理、同步機制、任務管理等功能，支援：
 * - FreeRTOS 嵌入式系統
 * - Windows 桌面環境  
 * - POSIX (Linux/macOS/Unix)
 */

// === 平台檢測 ===

#if defined(FREERTOS) || defined(__FREERTOS__)
    #define RTK_PLATFORM_FREERTOS 1
#elif defined(_WIN32) || defined(_WIN64) || defined(__CYGWIN__)
    #define RTK_PLATFORM_WINDOWS 1
#elif defined(__unix__) || defined(__unix) || defined(__linux__) || defined(__APPLE__)
    #define RTK_PLATFORM_POSIX 1
#else
    #define RTK_PLATFORM_UNKNOWN 1
#endif

// === 平台特定標頭檔 ===

#ifdef RTK_PLATFORM_FREERTOS
    #include "FreeRTOS.h"
    #include "task.h"
    #include "queue.h"
    #include "semphr.h"
    #include "timers.h"
    #include "event_groups.h"
#elif defined(RTK_PLATFORM_WINDOWS)
    #ifndef WIN32_LEAN_AND_MEAN
        #define WIN32_LEAN_AND_MEAN
    #endif
    #include <windows.h>
    #include <process.h>
    #include <synchapi.h>
    #include <handleapi.h>
    #include <processthreadsapi.h>
#elif defined(RTK_PLATFORM_POSIX)
    #include <pthread.h>
    #include <unistd.h>
    #include <sys/time.h>
    #include <semaphore.h>
    #include <errno.h>
    #ifdef __APPLE__
        // macOS doesn't have POSIX timers, use pthread-based alternative
        typedef struct rtk_posix_timer* rtk_posix_timer_t;
    #else
        #include <signal.h>
        typedef timer_t rtk_posix_timer_t;
    #endif
#endif

#include <stdlib.h>
#include <string.h>

// === 記憶體管理抽象 ===

#ifdef RTK_PLATFORM_FREERTOS
    #define RTK_MALLOC(size)            pvPortMalloc(size)
    #define RTK_FREE(ptr)               vPortFree(ptr)
    #define RTK_REALLOC(ptr, size)      pvPortRealloc(ptr, size)
    
#elif defined(RTK_PLATFORM_WINDOWS)
    #define RTK_MALLOC(size)            HeapAlloc(GetProcessHeap(), 0, size)
    #define RTK_FREE(ptr)               HeapFree(GetProcessHeap(), 0, ptr)
    #define RTK_REALLOC(ptr, size)      HeapReAlloc(GetProcessHeap(), 0, ptr, size)
    
#else // POSIX
    #define RTK_MALLOC(size)            malloc(size)
    #define RTK_FREE(ptr)               free(ptr)
    #define RTK_REALLOC(ptr, size)      realloc(ptr, size)
#endif

// === 時間與延遲抽象 ===

#ifdef RTK_PLATFORM_FREERTOS
    #define RTK_DELAY_MS(ms)            vTaskDelay(pdMS_TO_TICKS(ms))
    #define RTK_GET_TICK_COUNT()        xTaskGetTickCount()
    #define RTK_TICKS_TO_MS(ticks)      ((ticks) * portTICK_PERIOD_MS)
    #define RTK_MS_TO_TICKS(ms)         pdMS_TO_TICKS(ms)
    
#elif defined(RTK_PLATFORM_WINDOWS)
    #define RTK_DELAY_MS(ms)            Sleep(ms)
    #define RTK_GET_TICK_COUNT()        GetTickCount64()
    #define RTK_TICKS_TO_MS(ticks)      (ticks)
    #define RTK_MS_TO_TICKS(ms)         (ms)
    
#else // POSIX
    #define RTK_DELAY_MS(ms)            usleep((ms) * 1000)
    #define RTK_GET_TICK_COUNT()        rtk_posix_get_tick_count()
    #define RTK_TICKS_TO_MS(ticks)      (ticks)
    #define RTK_MS_TO_TICKS(ms)         (ms)
#endif

// === 同步原語類型定義 ===

#ifdef RTK_PLATFORM_FREERTOS
    typedef SemaphoreHandle_t           rtk_mutex_t;
    typedef SemaphoreHandle_t           rtk_semaphore_t;
    typedef QueueHandle_t               rtk_queue_t;
    typedef EventGroupHandle_t          rtk_event_group_t;
    typedef TaskHandle_t                rtk_task_handle_t;
    typedef TimerHandle_t               rtk_timer_handle_t;
    
#elif defined(RTK_PLATFORM_WINDOWS)
    typedef HANDLE                      rtk_mutex_t;
    typedef HANDLE                      rtk_semaphore_t;
    typedef HANDLE                      rtk_queue_t;
    typedef HANDLE                      rtk_event_group_t;
    typedef HANDLE                      rtk_task_handle_t;
    typedef HANDLE                      rtk_timer_handle_t;
    
#else // POSIX
    typedef pthread_mutex_t*            rtk_mutex_t;
    typedef sem_t*                      rtk_semaphore_t;
    typedef struct rtk_posix_queue*     rtk_queue_t;
    typedef struct rtk_posix_event*     rtk_event_group_t;
    typedef pthread_t*                  rtk_task_handle_t;
    typedef rtk_posix_timer_t           rtk_timer_handle_t;
#endif

// === 前向聲明 ===

typedef void* (*rtk_thread_func_t)(void* arg);
typedef struct rtk_sem rtk_sem_t;
typedef struct rtk_thread rtk_thread_t;

// === 常數定義 ===

#define RTK_WAIT_FOREVER                (-1)
#define RTK_NO_WAIT                     (0)

#ifdef RTK_PLATFORM_FREERTOS
    #define RTK_MAX_DELAY               portMAX_DELAY
    #define RTK_TASK_PRIORITY_IDLE      0
    #define RTK_TASK_PRIORITY_LOW       1
    #define RTK_TASK_PRIORITY_NORMAL    2
    #define RTK_TASK_PRIORITY_HIGH      3
    #define RTK_TASK_PRIORITY_REALTIME  4
    
#elif defined(RTK_PLATFORM_WINDOWS)
    #define RTK_MAX_DELAY               INFINITE
    #define RTK_TASK_PRIORITY_IDLE      THREAD_PRIORITY_IDLE
    #define RTK_TASK_PRIORITY_LOW       THREAD_PRIORITY_BELOW_NORMAL
    #define RTK_TASK_PRIORITY_NORMAL    THREAD_PRIORITY_NORMAL
    #define RTK_TASK_PRIORITY_HIGH      THREAD_PRIORITY_ABOVE_NORMAL
    #define RTK_TASK_PRIORITY_REALTIME  THREAD_PRIORITY_TIME_CRITICAL
    
#else // POSIX
    #define RTK_MAX_DELAY               (-1)
    #define RTK_TASK_PRIORITY_IDLE      1
    #define RTK_TASK_PRIORITY_LOW       20
    #define RTK_TASK_PRIORITY_NORMAL    0
    #define RTK_TASK_PRIORITY_HIGH      -10
    #define RTK_TASK_PRIORITY_REALTIME  -20
#endif

// === 錯誤碼 ===

typedef enum {
    RTK_PLATFORM_SUCCESS = 0,
    RTK_PLATFORM_ERROR_INVALID_PARAM = -1,
    RTK_PLATFORM_ERROR_TIMEOUT = -2,
    RTK_PLATFORM_ERROR_MEMORY = -3,
    RTK_PLATFORM_ERROR_RESOURCE = -4,
    RTK_PLATFORM_ERROR_NOT_SUPPORTED = -5,
    RTK_PLATFORM_ERROR_UNKNOWN = -99
} rtk_platform_error_t;

// === 任務管理結構 ===

typedef struct {
    const char* name;               /**< 任務名稱 */
    uint32_t stack_size;           /**< 堆疊大小 (位元組) */
    int priority;                  /**< 任務優先級 */
    void (*task_function)(void*);  /**< 任務函式 */
    void* parameters;              /**< 任務參數 */
} rtk_task_config_t;

typedef struct {
    const char* name;               /**< 計時器名稱 */
    uint32_t period_ms;            /**< 計時器週期 (毫秒) */
    int auto_reload;               /**< 是否自動重載 */
    void (*callback)(void*);       /**< 回調函式 */
    void* callback_data;           /**< 回調資料 */
} rtk_timer_config_t;

// === Mutex API ===

/**
 * @brief 建立互斥鎖
 * @param mutex 互斥鎖指標
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_mutex_create(rtk_mutex_t* mutex);

/**
 * @brief 銷毀互斥鎖
 * @param mutex 互斥鎖指標
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_mutex_destroy(rtk_mutex_t* mutex);

/**
 * @brief 取得互斥鎖
 * @param mutex 互斥鎖指標
 * @param timeout_ms 超時時間 (毫秒)，RTK_WAIT_FOREVER 表示無限等待
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_mutex_take(rtk_mutex_t* mutex, int timeout_ms);

/**
 * @brief 釋放互斥鎖
 * @param mutex 互斥鎖指標
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_mutex_give(rtk_mutex_t* mutex);

// === Semaphore API ===

/**
 * @brief 建立信號量
 * @param semaphore 信號量指標
 * @param initial_count 初始計數
 * @param max_count 最大計數
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_semaphore_create(rtk_semaphore_t* semaphore, int initial_count, int max_count);

/**
 * @brief 銷毀信號量
 * @param semaphore 信號量指標
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_semaphore_destroy(rtk_semaphore_t* semaphore);

/**
 * @brief 取得信號量
 * @param semaphore 信號量指標
 * @param timeout_ms 超時時間 (毫秒)
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_semaphore_take(rtk_semaphore_t* semaphore, int timeout_ms);

/**
 * @brief 釋放信號量
 * @param semaphore 信號量指標
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_semaphore_give(rtk_semaphore_t* semaphore);

// === Queue API ===

/**
 * @brief 建立佇列
 * @param queue 佇列指標
 * @param length 佇列長度
 * @param item_size 項目大小 (位元組)
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_queue_create(rtk_queue_t* queue, int length, size_t item_size);

/**
 * @brief 銷毀佇列
 * @param queue 佇列指標
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_queue_destroy(rtk_queue_t* queue);

/**
 * @brief 發送項目到佇列
 * @param queue 佇列指標
 * @param item 項目指標
 * @param timeout_ms 超時時間 (毫秒)
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_queue_send(rtk_queue_t* queue, const void* item, int timeout_ms);

/**
 * @brief 從佇列接收項目
 * @param queue 佇列指標
 * @param item 項目緩衝區
 * @param timeout_ms 超時時間 (毫秒)
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_queue_receive(rtk_queue_t* queue, void* item, int timeout_ms);

/**
 * @brief 獲取佇列中的項目數量
 * @param queue 佇列指標
 * @return 項目數量，負值表示錯誤
 */
int rtk_queue_count(rtk_queue_t* queue);

// === Task API ===

/**
 * @brief 建立任務
 * @param config 任務配置
 * @param handle 任務句柄指標
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_task_create(const rtk_task_config_t* config, rtk_task_handle_t* handle);

/**
 * @brief 刪除任務
 * @param handle 任務句柄
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_task_delete(rtk_task_handle_t handle);

/**
 * @brief 暫停任務
 * @param handle 任務句柄
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_task_suspend(rtk_task_handle_t handle);

/**
 * @brief 恢復任務
 * @param handle 任務句柄
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_task_resume(rtk_task_handle_t handle);

/**
 * @brief 設定任務優先級
 * @param handle 任務句柄
 * @param priority 優先級
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_task_set_priority(rtk_task_handle_t handle, int priority);

/**
 * @brief 獲取任務優先級
 * @param handle 任務句柄
 * @return 優先級，負值表示錯誤
 */
int rtk_task_get_priority(rtk_task_handle_t handle);

/**
 * @brief 獲取當前任務句柄
 * @return 當前任務句柄
 */
rtk_task_handle_t rtk_task_get_current(void);

/**
 * @brief 讓出 CPU 給其他任務
 */
void rtk_task_yield(void);

// === Timer API ===

/**
 * @brief 建立計時器
 * @param config 計時器配置
 * @param handle 計時器句柄指標
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_timer_create(const rtk_timer_config_t* config, rtk_timer_handle_t* handle);

/**
 * @brief 刪除計時器
 * @param handle 計時器句柄
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_timer_delete(rtk_timer_handle_t handle);

/**
 * @brief 啟動計時器
 * @param handle 計時器句柄
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_timer_start(rtk_timer_handle_t handle);

/**
 * @brief 停止計時器
 * @param handle 計時器句柄
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_timer_stop(rtk_timer_handle_t handle);

/**
 * @brief 重設計時器
 * @param handle 計時器句柄
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_timer_reset(rtk_timer_handle_t handle);

// === 輔助函式 ===

/**
 * @brief 獲取錯誤碼描述
 * @param error_code 錯誤碼
 * @return 錯誤描述字串
 */
const char* rtk_platform_get_error_string(rtk_platform_error_t error_code);

/**
 * @brief 獲取平台名稱
 * @return 平台名稱字串
 */
const char* rtk_platform_get_name(void);

/**
 * @brief 獲取平台版本
 * @return 平台版本字串
 */
const char* rtk_platform_get_version(void);

/**
 * @brief 初始化平台相容層
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_platform_init(void);

/**
 * @brief 清理平台相容層
 */
void rtk_platform_cleanup(void);

// === 平台特定輔助函式 ===

#ifdef RTK_PLATFORM_POSIX
/**
 * @brief POSIX 平台獲取 tick count
 * @return tick count (毫秒)
 */
uint64_t rtk_posix_get_tick_count(void);
#endif

// === 記憶體統計 (可選) ===

typedef struct {
    size_t total_allocated;         /**< 總分配記憶體 */
    size_t current_allocated;       /**< 當前分配記憶體 */
    size_t peak_allocated;          /**< 峰值分配記憶體 */
    size_t allocation_count;        /**< 分配次數 */
    size_t free_count;              /**< 釋放次數 */
} rtk_memory_stats_t;

/**
 * @brief 獲取記憶體統計 (如果支援)
 * @param stats 統計結構指標
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_platform_get_memory_stats(rtk_memory_stats_t* stats);

/**
 * @brief 重設記憶體統計
 * @return RTK_PLATFORM_SUCCESS 成功，其他值表示失敗
 */
int rtk_platform_reset_memory_stats(void);

#ifdef __cplusplus
}
#endif

#endif // RTK_PLATFORM_COMPAT_H
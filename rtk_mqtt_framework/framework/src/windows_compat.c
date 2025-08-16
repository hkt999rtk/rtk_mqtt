#include "rtk_platform_compat.h"

#ifdef RTK_PLATFORM_WINDOWS

#include <stdio.h>
#include <string.h>

/**
 * @file windows_compat.c
 * @brief Windows 平台相容層實作
 * 
 * 提供 Windows 平台的統一 API 實作
 */

// === 全域變數 ===

static int platform_initialized = 0;
static rtk_memory_stats_t memory_stats = {0};
static CRITICAL_SECTION memory_stats_lock;

// === 記憶體管理包裝 (用於統計) ===

void* rtk_windows_malloc(size_t size) {
    void* ptr = HeapAlloc(GetProcessHeap(), 0, size);
    if (ptr) {
        EnterCriticalSection(&memory_stats_lock);
        memory_stats.total_allocated += size;
        memory_stats.current_allocated += size;
        memory_stats.allocation_count++;
        if (memory_stats.current_allocated > memory_stats.peak_allocated) {
            memory_stats.peak_allocated = memory_stats.current_allocated;
        }
        LeaveCriticalSection(&memory_stats_lock);
    }
    return ptr;
}

void rtk_windows_free(void* ptr) {
    if (ptr) {
        SIZE_T size = HeapSize(GetProcessHeap(), 0, ptr);
        if (size != (SIZE_T)-1) {
            EnterCriticalSection(&memory_stats_lock);
            memory_stats.current_allocated -= size;
            memory_stats.free_count++;
            LeaveCriticalSection(&memory_stats_lock);
        }
        HeapFree(GetProcessHeap(), 0, ptr);
    }
}

// === Mutex API 實作 ===

int rtk_mutex_create(rtk_mutex_t* mutex) {
    if (!mutex) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    *mutex = CreateMutex(NULL, FALSE, NULL);
    if (*mutex == NULL) {
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_mutex_destroy(rtk_mutex_t* mutex) {
    if (!mutex || *mutex == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    CloseHandle(*mutex);
    *mutex = NULL;
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_mutex_take(rtk_mutex_t* mutex, int timeout_ms) {
    if (!mutex || *mutex == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    DWORD timeout;
    if (timeout_ms == RTK_WAIT_FOREVER) {
        timeout = INFINITE;
    } else if (timeout_ms == RTK_NO_WAIT) {
        timeout = 0;
    } else {
        timeout = (DWORD)timeout_ms;
    }
    
    DWORD result = WaitForSingleObject(*mutex, timeout);
    switch (result) {
        case WAIT_OBJECT_0:
            return RTK_PLATFORM_SUCCESS;
        case WAIT_TIMEOUT:
            return RTK_PLATFORM_ERROR_TIMEOUT;
        case WAIT_ABANDONED:
        case WAIT_FAILED:
        default:
            return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

int rtk_mutex_give(rtk_mutex_t* mutex) {
    if (!mutex || *mutex == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    if (ReleaseMutex(*mutex)) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

// === Semaphore API 實作 ===

int rtk_semaphore_create(rtk_semaphore_t* semaphore, int initial_count, int max_count) {
    if (!semaphore || max_count <= 0 || initial_count < 0 || initial_count > max_count) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    *semaphore = CreateSemaphore(NULL, initial_count, max_count, NULL);
    if (*semaphore == NULL) {
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_semaphore_destroy(rtk_semaphore_t* semaphore) {
    if (!semaphore || *semaphore == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    CloseHandle(*semaphore);
    *semaphore = NULL;
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_semaphore_take(rtk_semaphore_t* semaphore, int timeout_ms) {
    if (!semaphore || *semaphore == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    DWORD timeout;
    if (timeout_ms == RTK_WAIT_FOREVER) {
        timeout = INFINITE;
    } else if (timeout_ms == RTK_NO_WAIT) {
        timeout = 0;
    } else {
        timeout = (DWORD)timeout_ms;
    }
    
    DWORD result = WaitForSingleObject(*semaphore, timeout);
    switch (result) {
        case WAIT_OBJECT_0:
            return RTK_PLATFORM_SUCCESS;
        case WAIT_TIMEOUT:
            return RTK_PLATFORM_ERROR_TIMEOUT;
        case WAIT_FAILED:
        default:
            return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

int rtk_semaphore_give(rtk_semaphore_t* semaphore) {
    if (!semaphore || *semaphore == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    if (ReleaseSemaphore(*semaphore, 1, NULL)) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

// === Queue API 實作 (使用 Windows I/O Completion Port 模擬) ===

typedef struct {
    HANDLE completion_port;
    int item_size;
    int max_items;
    int current_items;
    CRITICAL_SECTION lock;
} rtk_windows_queue_t;

int rtk_queue_create(rtk_queue_t* queue, int length, size_t item_size) {
    if (!queue || length <= 0 || item_size == 0) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_queue_t* win_queue = (rtk_windows_queue_t*)HeapAlloc(
        GetProcessHeap(), HEAP_ZERO_MEMORY, sizeof(rtk_windows_queue_t));
    if (!win_queue) {
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    win_queue->completion_port = CreateIoCompletionPort(INVALID_HANDLE_VALUE, NULL, 0, 0);
    if (win_queue->completion_port == NULL) {
        HeapFree(GetProcessHeap(), 0, win_queue);
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    win_queue->item_size = (int)item_size;
    win_queue->max_items = length;
    win_queue->current_items = 0;
    InitializeCriticalSection(&win_queue->lock);
    
    *queue = (rtk_queue_t)win_queue;
    return RTK_PLATFORM_SUCCESS;
}

int rtk_queue_destroy(rtk_queue_t* queue) {
    if (!queue || *queue == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_queue_t* win_queue = (rtk_windows_queue_t*)*queue;
    
    CloseHandle(win_queue->completion_port);
    DeleteCriticalSection(&win_queue->lock);
    HeapFree(GetProcessHeap(), 0, win_queue);
    
    *queue = NULL;
    return RTK_PLATFORM_SUCCESS;
}

int rtk_queue_send(rtk_queue_t* queue, const void* item, int timeout_ms) {
    if (!queue || *queue == NULL || !item) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_queue_t* win_queue = (rtk_windows_queue_t*)*queue;
    
    EnterCriticalSection(&win_queue->lock);
    if (win_queue->current_items >= win_queue->max_items) {
        LeaveCriticalSection(&win_queue->lock);
        return RTK_PLATFORM_ERROR_TIMEOUT; // 佇列已滿
    }
    
    // 分配記憶體複製項目
    void* item_copy = HeapAlloc(GetProcessHeap(), 0, win_queue->item_size);
    if (!item_copy) {
        LeaveCriticalSection(&win_queue->lock);
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    memcpy(item_copy, item, win_queue->item_size);
    win_queue->current_items++;
    LeaveCriticalSection(&win_queue->lock);
    
    // 發送到完成埠
    if (!PostQueuedCompletionStatus(win_queue->completion_port, win_queue->item_size, 
                                   (ULONG_PTR)item_copy, NULL)) {
        EnterCriticalSection(&win_queue->lock);
        win_queue->current_items--;
        LeaveCriticalSection(&win_queue->lock);
        HeapFree(GetProcessHeap(), 0, item_copy);
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_queue_receive(rtk_queue_t* queue, void* item, int timeout_ms) {
    if (!queue || *queue == NULL || !item) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_queue_t* win_queue = (rtk_windows_queue_t*)*queue;
    
    DWORD timeout;
    if (timeout_ms == RTK_WAIT_FOREVER) {
        timeout = INFINITE;
    } else if (timeout_ms == RTK_NO_WAIT) {
        timeout = 0;
    } else {
        timeout = (DWORD)timeout_ms;
    }
    
    DWORD bytes_transferred;
    ULONG_PTR completion_key;
    LPOVERLAPPED overlapped;
    
    if (GetQueuedCompletionStatus(win_queue->completion_port, &bytes_transferred,
                                 &completion_key, &overlapped, timeout)) {
        void* item_data = (void*)completion_key;
        memcpy(item, item_data, win_queue->item_size);
        HeapFree(GetProcessHeap(), 0, item_data);
        
        EnterCriticalSection(&win_queue->lock);
        win_queue->current_items--;
        LeaveCriticalSection(&win_queue->lock);
        
        return RTK_PLATFORM_SUCCESS;
    } else {
        DWORD error = GetLastError();
        if (error == WAIT_TIMEOUT) {
            return RTK_PLATFORM_ERROR_TIMEOUT;
        } else {
            return RTK_PLATFORM_ERROR_RESOURCE;
        }
    }
}

int rtk_queue_count(rtk_queue_t* queue) {
    if (!queue || *queue == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_queue_t* win_queue = (rtk_windows_queue_t*)*queue;
    
    EnterCriticalSection(&win_queue->lock);
    int count = win_queue->current_items;
    LeaveCriticalSection(&win_queue->lock);
    
    return count;
}

// === Task API 實作 ===

typedef struct {
    rtk_task_config_t config;
    HANDLE thread_handle;
    DWORD thread_id;
} rtk_windows_task_t;

static DWORD WINAPI task_wrapper(LPVOID param) {
    rtk_windows_task_t* task = (rtk_windows_task_t*)param;
    if (task && task->config.task_function) {
        task->config.task_function(task->config.parameters);
    }
    return 0;
}

int rtk_task_create(const rtk_task_config_t* config, rtk_task_handle_t* handle) {
    if (!config || !config->task_function || !handle) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_task_t* task = (rtk_windows_task_t*)HeapAlloc(
        GetProcessHeap(), HEAP_ZERO_MEMORY, sizeof(rtk_windows_task_t));
    if (!task) {
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    // 複製配置
    memcpy(&task->config, config, sizeof(rtk_task_config_t));
    
    // 建立線程
    task->thread_handle = CreateThread(
        NULL,                           // 安全屬性
        config->stack_size,            // 堆疊大小
        task_wrapper,                  // 線程函式
        task,                          // 參數
        0,                             // 建立標誌
        &task->thread_id               // 線程 ID
    );
    
    if (task->thread_handle == NULL) {
        HeapFree(GetProcessHeap(), 0, task);
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    // 設定優先級
    SetThreadPriority(task->thread_handle, config->priority);
    
    *handle = (rtk_task_handle_t)task;
    return RTK_PLATFORM_SUCCESS;
}

int rtk_task_delete(rtk_task_handle_t handle) {
    if (handle == NULL) {
        // 刪除當前線程
        ExitThread(0);
        return RTK_PLATFORM_SUCCESS;
    }
    
    rtk_windows_task_t* task = (rtk_windows_task_t*)handle;
    
    // 終止線程並等待
    TerminateThread(task->thread_handle, 0);
    WaitForSingleObject(task->thread_handle, INFINITE);
    CloseHandle(task->thread_handle);
    HeapFree(GetProcessHeap(), 0, task);
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_task_suspend(rtk_task_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_task_t* task = (rtk_windows_task_t*)handle;
    
    if (SuspendThread(task->thread_handle) != (DWORD)-1) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

int rtk_task_resume(rtk_task_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_task_t* task = (rtk_windows_task_t*)handle;
    
    if (ResumeThread(task->thread_handle) != (DWORD)-1) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

int rtk_task_set_priority(rtk_task_handle_t handle, int priority) {
    if (handle == NULL) {
        // 設定當前線程優先級
        if (SetThreadPriority(GetCurrentThread(), priority)) {
            return RTK_PLATFORM_SUCCESS;
        } else {
            return RTK_PLATFORM_ERROR_RESOURCE;
        }
    }
    
    rtk_windows_task_t* task = (rtk_windows_task_t*)handle;
    
    if (SetThreadPriority(task->thread_handle, priority)) {
        task->config.priority = priority;
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

int rtk_task_get_priority(rtk_task_handle_t handle) {
    if (handle == NULL) {
        return GetThreadPriority(GetCurrentThread());
    }
    
    rtk_windows_task_t* task = (rtk_windows_task_t*)handle;
    return GetThreadPriority(task->thread_handle);
}

rtk_task_handle_t rtk_task_get_current(void) {
    // Windows 沒有直接的方式獲取當前任務的句柄結構
    // 這裡返回 GetCurrentThread() 的假值，實際使用時需要維護任務列表
    return (rtk_task_handle_t)GetCurrentThread();
}

void rtk_task_yield(void) {
    SwitchToThread();
}

// === Timer API 實作 ===

typedef struct {
    rtk_timer_config_t config;
    HANDLE timer_handle;
    HANDLE timer_queue;
} rtk_windows_timer_t;

static VOID CALLBACK timer_callback_wrapper(PVOID param, BOOLEAN timer_or_wait_fired) {
    rtk_windows_timer_t* timer = (rtk_windows_timer_t*)param;
    if (timer && timer->config.callback) {
        timer->config.callback(timer->config.callback_data);
    }
}

int rtk_timer_create(const rtk_timer_config_t* config, rtk_timer_handle_t* handle) {
    if (!config || !config->callback || !handle) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_timer_t* timer = (rtk_windows_timer_t*)HeapAlloc(
        GetProcessHeap(), HEAP_ZERO_MEMORY, sizeof(rtk_windows_timer_t));
    if (!timer) {
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    memcpy(&timer->config, config, sizeof(rtk_timer_config_t));
    
    timer->timer_queue = CreateTimerQueue();
    if (timer->timer_queue == NULL) {
        HeapFree(GetProcessHeap(), 0, timer);
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    *handle = (rtk_timer_handle_t)timer;
    return RTK_PLATFORM_SUCCESS;
}

int rtk_timer_delete(rtk_timer_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_timer_t* timer = (rtk_windows_timer_t*)handle;
    
    if (timer->timer_handle) {
        DeleteTimerQueueTimer(timer->timer_queue, timer->timer_handle, INVALID_HANDLE_VALUE);
    }
    
    DeleteTimerQueue(timer->timer_queue);
    HeapFree(GetProcessHeap(), 0, timer);
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_timer_start(rtk_timer_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_timer_t* timer = (rtk_windows_timer_t*)handle;
    
    ULONG flags = timer->config.auto_reload ? 0 : WT_EXECUTEONLYONCE;
    
    if (CreateTimerQueueTimer(
        &timer->timer_handle,
        timer->timer_queue,
        timer_callback_wrapper,
        timer,
        timer->config.period_ms,
        timer->config.auto_reload ? timer->config.period_ms : 0,
        flags)) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

int rtk_timer_stop(rtk_timer_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    rtk_windows_timer_t* timer = (rtk_windows_timer_t*)handle;
    
    if (timer->timer_handle) {
        if (DeleteTimerQueueTimer(timer->timer_queue, timer->timer_handle, NULL)) {
            timer->timer_handle = NULL;
            return RTK_PLATFORM_SUCCESS;
        } else {
            return RTK_PLATFORM_ERROR_RESOURCE;
        }
    }
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_timer_reset(rtk_timer_handle_t handle) {
    // Windows 計時器沒有直接的重設 API，先停止再啟動
    int result = rtk_timer_stop(handle);
    if (result == RTK_PLATFORM_SUCCESS) {
        result = rtk_timer_start(handle);
    }
    return result;
}

// === 輔助函式實作 ===

const char* rtk_platform_get_error_string(rtk_platform_error_t error_code) {
    switch (error_code) {
        case RTK_PLATFORM_SUCCESS: return "Success";
        case RTK_PLATFORM_ERROR_INVALID_PARAM: return "Invalid parameter";
        case RTK_PLATFORM_ERROR_TIMEOUT: return "Timeout";
        case RTK_PLATFORM_ERROR_MEMORY: return "Memory allocation error";
        case RTK_PLATFORM_ERROR_RESOURCE: return "Resource error";
        case RTK_PLATFORM_ERROR_NOT_SUPPORTED: return "Not supported";
        default: return "Unknown error";
    }
}

const char* rtk_platform_get_name(void) {
    return "Windows";
}

const char* rtk_platform_get_version(void) {
    static char version_buffer[64];
    OSVERSIONINFO osvi;
    ZeroMemory(&osvi, sizeof(OSVERSIONINFO));
    osvi.dwOSVersionInfoSize = sizeof(OSVERSIONINFO);
    
    if (GetVersionEx(&osvi)) {
        snprintf(version_buffer, sizeof(version_buffer), "%lu.%lu.%lu", 
                osvi.dwMajorVersion, osvi.dwMinorVersion, osvi.dwBuildNumber);
        return version_buffer;
    } else {
        return "Unknown";
    }
}

int rtk_platform_init(void) {
    if (platform_initialized) {
        return RTK_PLATFORM_SUCCESS;
    }
    
    InitializeCriticalSection(&memory_stats_lock);
    memset(&memory_stats, 0, sizeof(memory_stats));
    
    platform_initialized = 1;
    
    printf("[RTK-PLATFORM] Windows platform initialized\n");
    return RTK_PLATFORM_SUCCESS;
}

void rtk_platform_cleanup(void) {
    if (platform_initialized) {
        DeleteCriticalSection(&memory_stats_lock);
        platform_initialized = 0;
        printf("[RTK-PLATFORM] Windows platform cleaned up\n");
    }
}

// === 記憶體統計實作 ===

int rtk_platform_get_memory_stats(rtk_memory_stats_t* stats) {
    if (!stats) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    EnterCriticalSection(&memory_stats_lock);
    memcpy(stats, &memory_stats, sizeof(rtk_memory_stats_t));
    LeaveCriticalSection(&memory_stats_lock);
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_platform_reset_memory_stats(void) {
    EnterCriticalSection(&memory_stats_lock);
    memset(&memory_stats, 0, sizeof(memory_stats));
    LeaveCriticalSection(&memory_stats_lock);
    
    return RTK_PLATFORM_SUCCESS;
}

#endif // RTK_PLATFORM_WINDOWS
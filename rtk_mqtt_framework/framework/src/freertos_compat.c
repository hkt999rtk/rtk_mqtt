#include "rtk_platform_compat.h"

#ifdef RTK_PLATFORM_FREERTOS

#include <stdio.h>
#include <string.h>

/**
 * @file freertos_compat.c
 * @brief FreeRTOS 平台相容層實作
 * 
 * 提供 FreeRTOS 平台的統一 API 實作
 */

// === 全域變數 ===

static int platform_initialized = 0;

// === Mutex API 實作 ===

int rtk_mutex_create(rtk_mutex_t* mutex) {
    if (!mutex) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    *mutex = xSemaphoreCreateMutex();
    if (*mutex == NULL) {
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_mutex_destroy(rtk_mutex_t* mutex) {
    if (!mutex || *mutex == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    vSemaphoreDelete(*mutex);
    *mutex = NULL;
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_mutex_take(rtk_mutex_t* mutex, int timeout_ms) {
    if (!mutex || *mutex == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    TickType_t timeout_ticks;
    if (timeout_ms == RTK_WAIT_FOREVER) {
        timeout_ticks = portMAX_DELAY;
    } else if (timeout_ms == RTK_NO_WAIT) {
        timeout_ticks = 0;
    } else {
        timeout_ticks = pdMS_TO_TICKS(timeout_ms);
    }
    
    BaseType_t result = xSemaphoreTake(*mutex, timeout_ticks);
    if (result == pdTRUE) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_TIMEOUT;
    }
}

int rtk_mutex_give(rtk_mutex_t* mutex) {
    if (!mutex || *mutex == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    BaseType_t result = xSemaphoreGive(*mutex);
    if (result == pdTRUE) {
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
    
    if (max_count == 1) {
        // 建立二進制信號量
        *semaphore = xSemaphoreCreateBinary();
        if (initial_count == 1) {
            xSemaphoreGive(*semaphore);
        }
    } else {
        // 建立計數信號量
        *semaphore = xSemaphoreCreateCounting(max_count, initial_count);
    }
    
    if (*semaphore == NULL) {
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_semaphore_destroy(rtk_semaphore_t* semaphore) {
    if (!semaphore || *semaphore == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    vSemaphoreDelete(*semaphore);
    *semaphore = NULL;
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_semaphore_take(rtk_semaphore_t* semaphore, int timeout_ms) {
    if (!semaphore || *semaphore == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    TickType_t timeout_ticks;
    if (timeout_ms == RTK_WAIT_FOREVER) {
        timeout_ticks = portMAX_DELAY;
    } else if (timeout_ms == RTK_NO_WAIT) {
        timeout_ticks = 0;
    } else {
        timeout_ticks = pdMS_TO_TICKS(timeout_ms);
    }
    
    BaseType_t result = xSemaphoreTake(*semaphore, timeout_ticks);
    if (result == pdTRUE) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_TIMEOUT;
    }
}

int rtk_semaphore_give(rtk_semaphore_t* semaphore) {
    if (!semaphore || *semaphore == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    BaseType_t result;
    if (xPortInIsrContext()) {
        // 在中斷服務程序中
        BaseType_t higher_priority_task_woken = pdFALSE;
        result = xSemaphoreGiveFromISR(*semaphore, &higher_priority_task_woken);
        if (higher_priority_task_woken == pdTRUE) {
            portYIELD_FROM_ISR(higher_priority_task_woken);
        }
    } else {
        // 在正常任務中
        result = xSemaphoreGive(*semaphore);
    }
    
    if (result == pdTRUE) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

// === Queue API 實作 ===

int rtk_queue_create(rtk_queue_t* queue, int length, size_t item_size) {
    if (!queue || length <= 0 || item_size == 0) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    *queue = xQueueCreate(length, item_size);
    if (*queue == NULL) {
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_queue_destroy(rtk_queue_t* queue) {
    if (!queue || *queue == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    vQueueDelete(*queue);
    *queue = NULL;
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_queue_send(rtk_queue_t* queue, const void* item, int timeout_ms) {
    if (!queue || *queue == NULL || !item) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    TickType_t timeout_ticks;
    if (timeout_ms == RTK_WAIT_FOREVER) {
        timeout_ticks = portMAX_DELAY;
    } else if (timeout_ms == RTK_NO_WAIT) {
        timeout_ticks = 0;
    } else {
        timeout_ticks = pdMS_TO_TICKS(timeout_ms);
    }
    
    BaseType_t result;
    if (xPortInIsrContext()) {
        // 在中斷服務程序中
        BaseType_t higher_priority_task_woken = pdFALSE;
        result = xQueueSendFromISR(*queue, item, &higher_priority_task_woken);
        if (higher_priority_task_woken == pdTRUE) {
            portYIELD_FROM_ISR(higher_priority_task_woken);
        }
    } else {
        // 在正常任務中
        result = xQueueSend(*queue, item, timeout_ticks);
    }
    
    if (result == pdTRUE) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_TIMEOUT;
    }
}

int rtk_queue_receive(rtk_queue_t* queue, void* item, int timeout_ms) {
    if (!queue || *queue == NULL || !item) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    TickType_t timeout_ticks;
    if (timeout_ms == RTK_WAIT_FOREVER) {
        timeout_ticks = portMAX_DELAY;
    } else if (timeout_ms == RTK_NO_WAIT) {
        timeout_ticks = 0;
    } else {
        timeout_ticks = pdMS_TO_TICKS(timeout_ms);
    }
    
    BaseType_t result;
    if (xPortInIsrContext()) {
        // 在中斷服務程序中
        BaseType_t higher_priority_task_woken = pdFALSE;
        result = xQueueReceiveFromISR(*queue, item, &higher_priority_task_woken);
        if (higher_priority_task_woken == pdTRUE) {
            portYIELD_FROM_ISR(higher_priority_task_woken);
        }
    } else {
        // 在正常任務中
        result = xQueueReceive(*queue, item, timeout_ticks);
    }
    
    if (result == pdTRUE) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_TIMEOUT;
    }
}

int rtk_queue_count(rtk_queue_t* queue) {
    if (!queue || *queue == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    return (int)uxQueueMessagesWaiting(*queue);
}

// === Task API 實作 ===

int rtk_task_create(const rtk_task_config_t* config, rtk_task_handle_t* handle) {
    if (!config || !config->task_function || !handle) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    if (config->stack_size == 0 || config->priority < 0) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    BaseType_t result = xTaskCreate(
        config->task_function,
        config->name ? config->name : "RTK_Task",
        config->stack_size / sizeof(StackType_t), // FreeRTOS 需要 stack words，不是 bytes
        config->parameters,
        config->priority,
        handle
    );
    
    if (result == pdPASS) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_MEMORY;
    }
}

int rtk_task_delete(rtk_task_handle_t handle) {
    // 如果 handle 為 NULL，刪除當前任務
    vTaskDelete(handle);
    return RTK_PLATFORM_SUCCESS;
}

int rtk_task_suspend(rtk_task_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    vTaskSuspend(handle);
    return RTK_PLATFORM_SUCCESS;
}

int rtk_task_resume(rtk_task_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    if (xPortInIsrContext()) {
        BaseType_t higher_priority_task_woken = pdFALSE;
        xTaskResumeFromISR(handle);
        if (higher_priority_task_woken == pdTRUE) {
            portYIELD_FROM_ISR(higher_priority_task_woken);
        }
    } else {
        vTaskResume(handle);
    }
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_task_set_priority(rtk_task_handle_t handle, int priority) {
    if (priority < 0) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    vTaskPrioritySet(handle, priority);
    return RTK_PLATFORM_SUCCESS;
}

int rtk_task_get_priority(rtk_task_handle_t handle) {
    return (int)uxTaskPriorityGet(handle);
}

rtk_task_handle_t rtk_task_get_current(void) {
    return xTaskGetCurrentTaskHandle();
}

void rtk_task_yield(void) {
    taskYIELD();
}

// === Timer API 實作 ===

static void timer_callback_wrapper(TimerHandle_t timer) {
    rtk_timer_config_t* config = (rtk_timer_config_t*)pvTimerGetTimerID(timer);
    if (config && config->callback) {
        config->callback(config->callback_data);
    }
}

int rtk_timer_create(const rtk_timer_config_t* config, rtk_timer_handle_t* handle) {
    if (!config || !config->callback || !handle) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    if (config->period_ms == 0) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    TickType_t period_ticks = pdMS_TO_TICKS(config->period_ms);
    BaseType_t auto_reload = config->auto_reload ? pdTRUE : pdFALSE;
    
    *handle = xTimerCreate(
        config->name ? config->name : "RTK_Timer",
        period_ticks,
        auto_reload,
        (void*)config, // 將配置作為 timer ID 傳遞
        timer_callback_wrapper
    );
    
    if (*handle == NULL) {
        return RTK_PLATFORM_ERROR_MEMORY;
    }
    
    return RTK_PLATFORM_SUCCESS;
}

int rtk_timer_delete(rtk_timer_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    BaseType_t result = xTimerDelete(handle, portMAX_DELAY);
    if (result == pdPASS) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

int rtk_timer_start(rtk_timer_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    BaseType_t result;
    if (xPortInIsrContext()) {
        BaseType_t higher_priority_task_woken = pdFALSE;
        result = xTimerStartFromISR(handle, &higher_priority_task_woken);
        if (higher_priority_task_woken == pdTRUE) {
            portYIELD_FROM_ISR(higher_priority_task_woken);
        }
    } else {
        result = xTimerStart(handle, portMAX_DELAY);
    }
    
    if (result == pdPASS) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

int rtk_timer_stop(rtk_timer_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    BaseType_t result;
    if (xPortInIsrContext()) {
        BaseType_t higher_priority_task_woken = pdFALSE;
        result = xTimerStopFromISR(handle, &higher_priority_task_woken);
        if (higher_priority_task_woken == pdTRUE) {
            portYIELD_FROM_ISR(higher_priority_task_woken);
        }
    } else {
        result = xTimerStop(handle, portMAX_DELAY);
    }
    
    if (result == pdPASS) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
}

int rtk_timer_reset(rtk_timer_handle_t handle) {
    if (handle == NULL) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    BaseType_t result;
    if (xPortInIsrContext()) {
        BaseType_t higher_priority_task_woken = pdFALSE;
        result = xTimerResetFromISR(handle, &higher_priority_task_woken);
        if (higher_priority_task_woken == pdTRUE) {
            portYIELD_FROM_ISR(higher_priority_task_woken);
        }
    } else {
        result = xTimerReset(handle, portMAX_DELAY);
    }
    
    if (result == pdPASS) {
        return RTK_PLATFORM_SUCCESS;
    } else {
        return RTK_PLATFORM_ERROR_RESOURCE;
    }
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
    return "FreeRTOS";
}

const char* rtk_platform_get_version(void) {
    return tskKERNEL_VERSION_NUMBER;
}

int rtk_platform_init(void) {
    if (platform_initialized) {
        return RTK_PLATFORM_SUCCESS;
    }
    
    // FreeRTOS 通常在 main() 中已經初始化，這裡只做標記
    platform_initialized = 1;
    
    printf("[RTK-PLATFORM] FreeRTOS platform initialized\n");
    return RTK_PLATFORM_SUCCESS;
}

void rtk_platform_cleanup(void) {
    platform_initialized = 0;
    printf("[RTK-PLATFORM] FreeRTOS platform cleaned up\n");
}

// === 記憶體統計實作 (FreeRTOS 特定) ===

int rtk_platform_get_memory_stats(rtk_memory_stats_t* stats) {
    if (!stats) {
        return RTK_PLATFORM_ERROR_INVALID_PARAM;
    }
    
    memset(stats, 0, sizeof(rtk_memory_stats_t));
    
#if (configUSE_TRACE_FACILITY == 1) && (configUSE_STATS_FORMATTING_FUNCTIONS == 1)
    // 如果 FreeRTOS 啟用了統計功能
    size_t free_heap = xPortGetFreeHeapSize();
    size_t min_free_heap = xPortGetMinimumEverFreeHeapSize();
    
    stats->current_allocated = configTOTAL_HEAP_SIZE - free_heap;
    stats->peak_allocated = configTOTAL_HEAP_SIZE - min_free_heap;
    stats->total_allocated = configTOTAL_HEAP_SIZE;
    
    return RTK_PLATFORM_SUCCESS;
#else
    return RTK_PLATFORM_ERROR_NOT_SUPPORTED;
#endif
}

int rtk_platform_reset_memory_stats(void) {
    // FreeRTOS 沒有重設記憶體統計的標準 API
    return RTK_PLATFORM_ERROR_NOT_SUPPORTED;
}

#endif // RTK_PLATFORM_FREERTOS
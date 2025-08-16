/**
 * @file FreeRTOSConfig.h
 * @brief FreeRTOS 配置檔案 - RTK MQTT Framework 範例
 * 
 * 針對 MQTT 設備應用優化的 FreeRTOS 配置，適用於：
 * - ARM Cortex-M 微控制器
 * - 網路連接功能
 * - JSON 處理需求
 * - 中等記憶體容量 (128KB+ RAM)
 */

#ifndef FREERTOS_CONFIG_H
#define FREERTOS_CONFIG_H

// === 處理器和編譯器設定 ===

#ifdef __ICCARM__
    #include <stdint.h>
    extern uint32_t SystemCoreClock;
#endif

// 假設使用 ARM Cortex-M4 @ 168MHz
#define configCPU_CLOCK_HZ              168000000UL
#define configTICK_RATE_HZ              1000
#define configSYSTICK_CLOCK_HZ          configCPU_CLOCK_HZ

// === 記憶體配置 ===

#define configTOTAL_HEAP_SIZE           ((size_t)(64 * 1024))  // 64KB 堆疊
#define configMINIMAL_STACK_SIZE        ((unsigned short)128)   // 128 words = 512 bytes
#define configMAX_TASK_NAME_LEN         16

// === 任務管理 ===

#define configMAX_PRIORITIES            8
#define configUSE_PREEMPTION            1
#define configUSE_IDLE_HOOK             1
#define configUSE_TICK_HOOK             1
#define configUSE_MALLOC_FAILED_HOOK    1
#define configCHECK_FOR_STACK_OVERFLOW  2

// === 時間管理 ===

#define configUSE_16_BIT_TICKS          0  // 使用 32 位 tick
#define configIDLE_SHOULD_YIELD         1

// === 同步和通訊 ===

#define configUSE_MUTEXES               1
#define configUSE_RECURSIVE_MUTEXES     1
#define configUSE_COUNTING_SEMAPHORES   1
#define configUSE_BINARY_SEMAPHORES     1
#define configQUEUE_REGISTRY_SIZE       8

// === 軟體定時器 ===

#define configUSE_TIMERS                1
#define configTIMER_TASK_PRIORITY       (configMAX_PRIORITIES - 1)
#define configTIMER_QUEUE_LENGTH        5
#define configTIMER_TASK_STACK_DEPTH    (configMINIMAL_STACK_SIZE * 2)

// === 任務通知 ===

#define configUSE_TASK_NOTIFICATIONS    1

// === 共同例程 (Co-routines) ===

#define configUSE_CO_ROUTINES           0
#define configMAX_CO_ROUTINE_PRIORITIES 2

// === 除錯和追蹤 ===

#define configGENERATE_RUN_TIME_STATS           0
#define configUSE_TRACE_FACILITY                1
#define configUSE_STATS_FORMATTING_FUNCTIONS    1

// === 記憶體分配 ===

#define configSUPPORT_STATIC_ALLOCATION         0
#define configSUPPORT_DYNAMIC_ALLOCATION        1

// === 中斷優先權 ===

// Cortex-M 使用 4 位元優先權 (0-15)
// 較低數值 = 較高優先權
#define configKERNEL_INTERRUPT_PRIORITY         255  // 最低優先權
#define configMAX_SYSCALL_INTERRUPT_PRIORITY    191  // 5 << (8-4)

// === API 函數包含控制 ===

#define INCLUDE_vTaskPrioritySet            1
#define INCLUDE_uxTaskPriorityGet           1
#define INCLUDE_vTaskDelete                 1
#define INCLUDE_vTaskCleanUpResources       0
#define INCLUDE_vTaskSuspend                1
#define INCLUDE_vTaskDelayUntil             1
#define INCLUDE_vTaskDelay                  1
#define INCLUDE_xTaskGetSchedulerState      1
#define INCLUDE_xTaskGetCurrentTaskHandle   1
#define INCLUDE_uxTaskGetStackHighWaterMark 1
#define INCLUDE_xTaskGetIdleTaskHandle      1
#define INCLUDE_eTaskGetState               1
#define INCLUDE_xEventGroupSetBitFromISR    1
#define INCLUDE_xTimerPendFunctionCall      1
#define INCLUDE_xTaskAbortDelay             1
#define INCLUDE_xTaskGetHandle              1

// === MQTT 應用程式特定設定 ===

// 針對網路應用程式的最佳化設定
#define configUSE_QUEUE_SETS                1  // 用於 select() 類似功能
#define configUSE_TIME_SLICING              1  // 同優先權任務輪轉

// === Cortex-M 特定設定 ===

#define configPRIO_BITS                     4  // Cortex-M4 使用 4 位元優先權

// SVC 和 PendSV 中斷優先權設定
#define configKERNEL_INTERRUPT_PRIORITY_BITS    4
#define configMAX_SYSCALL_INTERRUPT_PRIORITY_BITS 4

// === 低功耗模式支援 ===

#define configUSE_TICKLESS_IDLE             1  // 啟用 tickless 空閒模式
#define configEXPECTED_IDLE_TIME_BEFORE_SLEEP 2  // 進入睡眠前的最小空閒時間

// === 網路棧整合 ===

// 如果使用 FreeRTOS+TCP，啟用以下設定
#define configNUM_THREAD_LOCAL_STORAGE_POINTERS 3  // 用於網路棧

// === 除錯輔助 ===

#ifdef DEBUG
    #define configASSERT(x) \
        if ((x) == 0) { \
            taskDISABLE_INTERRUPTS(); \
            printf("ASSERT: %s:%d\n", __FILE__, __LINE__); \
            for (;;); \
        }
#else
    #define configASSERT(x)
#endif

// === 中斷服務例程名稱定義 ===

// Cortex-M NVIC 中斷名稱
#define vPortSVCHandler     SVC_Handler
#define xPortPendSVHandler  PendSV_Handler
#define xPortSysTickHandler SysTick_Handler

// === 應用程式特定鉤子函數 ===

// 在 main.c 中實作的鉤子函數
extern void vApplicationStackOverflowHook(TaskHandle_t xTask, char* pcTaskName);
extern void vApplicationMallocFailedHook(void);
extern void vApplicationIdleHook(void);
extern void vApplicationTickHook(void);

// === 記憶體管理設定 ===

// 使用 heap_4.c 記憶體分配器 (推薦用於網路應用)
// 特點：合併相鄰空閒區塊，適合頻繁分配/釋放
#define configHEAP_ALLOCATION_SCHEME        4

// === 編譯器相容性 ===

#if defined(__GNUC__)
    // GCC 編譯器設定
    #define configUSE_PORT_OPTIMISED_TASK_SELECTION 1
    
#elif defined(__ICCARM__)
    // IAR 編譯器設定
    #define configUSE_PORT_OPTIMISED_TASK_SELECTION 0
    
#elif defined(__CC_ARM) || defined(__ARMCC_VERSION)
    // Keil/ARM 編譯器設定
    #define configUSE_PORT_OPTIMISED_TASK_SELECTION 0
    
#else
    #define configUSE_PORT_OPTIMISED_TASK_SELECTION 0
#endif

// === 效能調校設定 ===

// 針對 MQTT 應用程式的調校
#define configIDLE_TASK_STACK_SIZE          configMINIMAL_STACK_SIZE
#define configTIMER_SERVICE_TASK_NAME       "TmrSvc"

// === 記憶體保護單元 (MPU) 支援 ===

#define configENABLE_MPU                    0  // 大部分應用不需要 MPU
#define configENABLE_FPU                    1  // 啟用浮點運算單元 (用於感測器計算)
#define configENABLE_TRUSTZONE              0  // TrustZone 支援 (Cortex-M33+ 才需要)

// === 新版 FreeRTOS 特性 ===

#if (configUSE_TRACE_FACILITY == 1)
    #define configUSE_POSIX_ERRNO           1  // POSIX 錯誤碼支援
#endif

// === 任務標籤支援 ===

#define configUSE_APPLICATION_TASK_TAG      0

// === 條件變數支援 (FreeRTOS v10.2.0+) ===

#define configUSE_SB_COMPLETED_CALLBACK     0

#endif /* FREERTOS_CONFIG_H */
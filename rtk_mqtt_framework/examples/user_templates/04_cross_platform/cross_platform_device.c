/*
 * RTK MQTT Framework 跨平台設備範例
 * 這個範例展示如何建立可在多個平台運行的設備程式：
 * 1. POSIX 系統 (Linux/macOS) 支援
 * 2. Windows 系統支援  
 * 3. ARM FreeRTOS 嵌入式系統支援
 * 4. 平台抽象層和條件編譯
 * 5. 統一的設備介面
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

// RTK Framework 核心標頭檔
#include <rtk_mqtt_client.h>
#include <rtk_topic_builder.h>
#include <rtk_message_codec.h>
#include <rtk_platform_compat.h>

// 平台特定標頭檔
#ifdef RTK_PLATFORM_FREERTOS
    #include <FreeRTOS.h>
    #include <task.h>
    #include <semphr.h>
    #include <timers.h>
#elif defined(RTK_PLATFORM_WINDOWS)
    #include <windows.h>
    #include <process.h>
#else  // POSIX (Linux/macOS)
    #include <pthread.h>
    #include <unistd.h>
    #include <signal.h>
    #include <sys/time.h>
#endif

// === 平台抽象層定義 ===

// 執行緒類型定義
#ifdef RTK_PLATFORM_FREERTOS
    typedef TaskHandle_t rtk_thread_t;
    typedef SemaphoreHandle_t rtk_mutex_t;
    typedef TimerHandle_t rtk_timer_t;
#elif defined(RTK_PLATFORM_WINDOWS)
    typedef HANDLE rtk_thread_t;
    typedef CRITICAL_SECTION rtk_mutex_t;
    typedef HANDLE rtk_timer_t;
#else  // POSIX
    typedef pthread_t rtk_thread_t;
    typedef pthread_mutex_t rtk_mutex_t;
    typedef timer_t rtk_timer_t;
#endif

// 時間類型定義
#ifdef RTK_PLATFORM_FREERTOS
    typedef TickType_t rtk_time_t;
#elif defined(RTK_PLATFORM_WINDOWS)
    typedef DWORD rtk_time_t;
#else  // POSIX
    typedef time_t rtk_time_t;
#endif

// === 設備狀態結構 ===

typedef struct {
    char device_id[64];
    char platform_name[32];
    char device_type[32];
    float cpu_usage;
    float memory_usage;
    float temperature;
    rtk_time_t uptime;
    int connected;
} cross_platform_device_t;

// 全域設備實例
static cross_platform_device_t g_device = {0};
static rtk_mqtt_client_t* g_mqtt_client = NULL;
static volatile int g_running = 1;

// 平台抽象層的互斥鎖
static rtk_mutex_t g_device_mutex;

// === 平台抽象層實作 ===

// 互斥鎖操作
int rtk_mutex_init(rtk_mutex_t* mutex) {
#ifdef RTK_PLATFORM_FREERTOS
    *mutex = xSemaphoreCreateMutex();
    return (*mutex != NULL) ? 0 : -1;
#elif defined(RTK_PLATFORM_WINDOWS)
    InitializeCriticalSection(mutex);
    return 0;
#else  // POSIX
    return pthread_mutex_init(mutex, NULL);
#endif
}

void rtk_mutex_lock(rtk_mutex_t* mutex) {
#ifdef RTK_PLATFORM_FREERTOS
    xSemaphoreTake(*mutex, portMAX_DELAY);
#elif defined(RTK_PLATFORM_WINDOWS)
    EnterCriticalSection(mutex);
#else  // POSIX
    pthread_mutex_lock(mutex);
#endif
}

void rtk_mutex_unlock(rtk_mutex_t* mutex) {
#ifdef RTK_PLATFORM_FREERTOS
    xSemaphoreGive(*mutex);
#elif defined(RTK_PLATFORM_WINDOWS)
    LeaveCriticalSection(mutex);
#else  // POSIX
    pthread_mutex_unlock(mutex);
#endif
}

void rtk_mutex_destroy(rtk_mutex_t* mutex) {
#ifdef RTK_PLATFORM_FREERTOS
    vSemaphoreDelete(*mutex);
#elif defined(RTK_PLATFORM_WINDOWS)
    DeleteCriticalSection(mutex);
#else  // POSIX
    pthread_mutex_destroy(mutex);
#endif
}

// 時間相關函式
rtk_time_t rtk_get_time(void) {
#ifdef RTK_PLATFORM_FREERTOS
    return xTaskGetTickCount();
#elif defined(RTK_PLATFORM_WINDOWS)
    return GetTickCount();
#else  // POSIX
    return time(NULL);
#endif
}

void rtk_sleep(int seconds) {
#ifdef RTK_PLATFORM_FREERTOS
    vTaskDelay(pdMS_TO_TICKS(seconds * 1000));
#elif defined(RTK_PLATFORM_WINDOWS)
    Sleep(seconds * 1000);
#else  // POSIX
    sleep(seconds);
#endif
}

// 執行緒創建
typedef void (*rtk_thread_func_t)(void* arg);

int rtk_thread_create(rtk_thread_t* thread, rtk_thread_func_t func, void* arg) {
#ifdef RTK_PLATFORM_FREERTOS
    BaseType_t result = xTaskCreate(
        (TaskFunction_t)func,
        "RTK_Worker",
        configMINIMAL_STACK_SIZE * 4,
        arg,
        tskIDLE_PRIORITY + 1,
        thread
    );
    return (result == pdPASS) ? 0 : -1;
#elif defined(RTK_PLATFORM_WINDOWS)
    *thread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)func, arg, 0, NULL);
    return (*thread != NULL) ? 0 : -1;
#else  // POSIX
    return pthread_create(thread, NULL, (void*(*)(void*))func, arg);
#endif
}

void rtk_thread_join(rtk_thread_t* thread) {
#ifdef RTK_PLATFORM_FREERTOS
    // FreeRTOS 任務通常不需要 join，任務會自動清理
    // 這裡我們可以等待任務完成的信號
#elif defined(RTK_PLATFORM_WINDOWS)
    WaitForSingleObject(*thread, INFINITE);
    CloseHandle(*thread);
#else  // POSIX
    pthread_join(*thread, NULL);
#endif
}

// === 平台特定的系統資訊獲取 ===

float get_cpu_usage(void) {
#ifdef RTK_PLATFORM_FREERTOS
    // FreeRTOS: 模擬 CPU 使用率（實際可用 FreeRTOS 統計功能）
    static float base_cpu = 30.0f;
    float variation = ((float)(rand() % 200) / 100.0f - 1.0f) * 15.0f;
    return base_cpu + variation;
#elif defined(RTK_PLATFORM_WINDOWS)
    // Windows: 可使用 GetSystemTimes 或 PDH API
    // 這裡簡化為模擬實作
    static float base_cpu = 25.0f;
    float variation = ((float)(rand() % 200) / 100.0f - 1.0f) * 20.0f;
    return base_cpu + variation;
#else  // POSIX
    // Linux/macOS: 可讀取 /proc/stat 或使用 sysinfo
    // 這裡簡化為模擬實作
    static float base_cpu = 35.0f;
    float variation = ((float)(rand() % 200) / 100.0f - 1.0f) * 25.0f;
    return base_cpu + variation;
#endif
}

float get_memory_usage(void) {
#ifdef RTK_PLATFORM_FREERTOS
    // FreeRTOS: 使用 xPortGetFreeHeapSize()
    size_t free_heap = xPortGetFreeHeapSize();
    size_t total_heap = configTOTAL_HEAP_SIZE;
    return ((float)(total_heap - free_heap) / total_heap) * 100.0f;
#elif defined(RTK_PLATFORM_WINDOWS)
    // Windows: 使用 GlobalMemoryStatusEx
    MEMORYSTATUSEX mem_info;
    mem_info.dwLength = sizeof(MEMORYSTATUSEX);
    if (GlobalMemoryStatusEx(&mem_info)) {
        return (float)mem_info.dwMemoryLoad;
    }
    return 50.0f;  // 預設值
#else  // POSIX
    // Linux/macOS: 可讀取 /proc/meminfo 或使用 sysinfo
    // 這裡簡化為模擬實作
    static float base_mem = 45.0f;
    float variation = ((float)(rand() % 200) / 100.0f - 1.0f) * 10.0f;
    return base_mem + variation;
#endif
}

const char* get_platform_name(void) {
#ifdef RTK_PLATFORM_FREERTOS
    return "FreeRTOS";
#elif defined(RTK_PLATFORM_WINDOWS)
    return "Windows";
#elif defined(__APPLE__)
    return "macOS";
#elif defined(__linux__)
    return "Linux";
#else
    return "Unknown";
#endif
}

// === 設備功能實作 ===

void update_device_metrics(void) {
    rtk_mutex_lock(&g_device_mutex);
    
    g_device.cpu_usage = get_cpu_usage();
    g_device.memory_usage = get_memory_usage();
    
    // 模擬溫度感測器（平台特定的調整）
#ifdef RTK_PLATFORM_FREERTOS
    // 嵌入式系統通常溫度較高
    static float base_temp = 45.0f;
#else
    // 桌面系統溫度較低
    static float base_temp = 35.0f;
#endif
    float temp_variation = ((float)(rand() % 200) / 100.0f - 1.0f) * 8.0f;
    g_device.temperature = base_temp + temp_variation;
    
    g_device.uptime = rtk_get_time();
    
    rtk_mutex_unlock(&g_device_mutex);
}

int publish_device_telemetry(void) {
    if (!g_mqtt_client || !g_device.connected) {
        return 0;
    }
    
    int success_count = 0;
    
    rtk_mutex_lock(&g_device_mutex);
    
    // 發布 CPU 使用率
    if (rtk_mqtt_client_publish_telemetry(g_mqtt_client, "cpu_usage", 
                                         g_device.cpu_usage, "%") == RTK_SUCCESS) {
        success_count++;
    }
    
    // 發布記憶體使用率
    if (rtk_mqtt_client_publish_telemetry(g_mqtt_client, "memory_usage", 
                                         g_device.memory_usage, "%") == RTK_SUCCESS) {
        success_count++;
    }
    
    // 發布溫度
    if (rtk_mqtt_client_publish_telemetry(g_mqtt_client, "temperature", 
                                         g_device.temperature, "°C") == RTK_SUCCESS) {
        success_count++;
    }
    
    // 發布平台資訊
    char platform_info[128];
    snprintf(platform_info, sizeof(platform_info), 
             "platform=%s,arch=%s", 
             g_device.platform_name,
#ifdef RTK_PLATFORM_FREERTOS
             "ARM"
#elif defined(_WIN64)
             "x64"
#elif defined(_WIN32)
             "x86"
#elif defined(__x86_64__)
             "x64"
#elif defined(__arm__)
             "ARM"
#else
             "unknown"
#endif
    );
    
    if (rtk_mqtt_client_publish_event(g_mqtt_client, "platform.info", platform_info) == RTK_SUCCESS) {
        success_count++;
    }
    
    rtk_mutex_unlock(&g_device_mutex);
    
    printf("[%s] 遙測資料發布: %d/4 成功 (CPU: %.1f%%, 記憶體: %.1f%%, 溫度: %.1f°C)\n",
           get_platform_name(), success_count, 
           g_device.cpu_usage, g_device.memory_usage, g_device.temperature);
    
    return success_count;
}

// === 工作執行緒 ===

void sensor_worker_thread(void* arg) {
    printf("[%s] 感測器工作執行緒啟動\n", get_platform_name());
    
    while (g_running) {
        if (g_device.connected) {
            // 更新設備指標
            update_device_metrics();
            
            // 發布遙測資料
            publish_device_telemetry();
            
            // 發布設備狀態
            rtk_mqtt_client_publish_state(g_mqtt_client, "online", "healthy");
        }
        
        // 等待下次週期（30秒）
        rtk_sleep(30);
    }
    
    printf("[%s] 感測器工作執行緒結束\n", get_platform_name());
    
#ifdef RTK_PLATFORM_FREERTOS
    vTaskDelete(NULL);  // FreeRTOS 任務自行刪除
#endif
}

// === 信號處理（僅適用於支援的平台）===

#if !defined(RTK_PLATFORM_FREERTOS) && !defined(RTK_PLATFORM_WINDOWS)
void signal_handler(int signal) {
    printf("\n[%s] 收到信號 %d，正在停止設備...\n", get_platform_name(), signal);
    g_running = 0;
}
#endif

// === 設備初始化和清理 ===

int initialize_device(void) {
    printf("[%s] 正在初始化跨平台設備...\n", get_platform_name());
    
    // 初始化隨機數種子
#ifdef RTK_PLATFORM_FREERTOS
    // FreeRTOS 可能需要不同的隨機數初始化方式
    srand(xTaskGetTickCount());
#else
    srand((unsigned int)time(NULL));
#endif
    
    // 初始化設備資訊
    snprintf(g_device.device_id, sizeof(g_device.device_id), 
             "cross_platform_%s_001", get_platform_name());
    strncpy(g_device.platform_name, get_platform_name(), sizeof(g_device.platform_name) - 1);
    strncpy(g_device.device_type, "cross_platform_demo", sizeof(g_device.device_type) - 1);
    g_device.connected = 0;
    
    // 初始化互斥鎖
    if (rtk_mutex_init(&g_device_mutex) != 0) {
        printf("[%s] 錯誤: 互斥鎖初始化失敗\n", get_platform_name());
        return -1;
    }
    
    // 創建 MQTT 客戶端
    g_mqtt_client = rtk_mqtt_client_create(
        "test.mosquitto.org",
        1883,
        g_device.device_id
    );
    
    if (!g_mqtt_client) {
        printf("[%s] 錯誤: MQTT 客戶端創建失敗\n", get_platform_name());
        return -1;
    }
    
    printf("[%s] 設備初始化完成 (設備 ID: %s)\n", get_platform_name(), g_device.device_id);
    return 0;
}

void cleanup_device(void) {
    printf("[%s] 正在清理設備資源...\n", get_platform_name());
    
    g_running = 0;
    
    // 發布離線狀態
    if (g_mqtt_client && g_device.connected) {
        rtk_mqtt_client_publish_state(g_mqtt_client, "offline", "shutdown");
        rtk_mqtt_client_publish_event(g_mqtt_client, "device.lifecycle.shutdown", 
                                     "跨平台設備正常關閉");
    }
    
    // 清理 MQTT 客戶端
    if (g_mqtt_client) {
        rtk_mqtt_client_disconnect(g_mqtt_client);
        rtk_mqtt_client_destroy(g_mqtt_client);
        g_mqtt_client = NULL;
    }
    
    // 清理互斥鎖
    rtk_mutex_destroy(&g_device_mutex);
    
    printf("[%s] 設備資源清理完成\n", get_platform_name());
}

// === 主程式 ===

#ifdef RTK_PLATFORM_FREERTOS
// FreeRTOS 應用程式入口點
void app_main(void) {
#else
int main(int argc, char* argv[]) {
#endif
    printf("RTK MQTT Framework 跨平台設備範例\n");
    printf("================================\n");
    printf("平台: %s\n", get_platform_name());
    printf("編譯時間: %s %s\n", __DATE__, __TIME__);
    
#ifdef PROJECT_VERSION
    printf("版本: %s\n", PROJECT_VERSION);
#endif
    
    printf("\n");
    
    // 初始化設備
    if (initialize_device() != 0) {
        printf("[%s] 錯誤: 設備初始化失敗\n", get_platform_name());
#ifdef RTK_PLATFORM_FREERTOS
        return;
#else
        return -1;
#endif
    }
    
    // 設置信號處理（非 FreeRTOS 和 Windows）
#if !defined(RTK_PLATFORM_FREERTOS) && !defined(RTK_PLATFORM_WINDOWS)
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
#endif
    
    // 連接到 MQTT broker
    printf("[%s] 正在連接到 MQTT broker...\n", get_platform_name());
    
    if (rtk_mqtt_client_connect(g_mqtt_client) != RTK_SUCCESS) {
        printf("[%s] 錯誤: MQTT 連接失敗\n", get_platform_name());
        cleanup_device();
#ifdef RTK_PLATFORM_FREERTOS
        return;
#else
        return -1;
#endif
    }
    
    g_device.connected = 1;
    printf("[%s] ✓ MQTT 連接成功\n", get_platform_name());
    
    // 發布啟動事件
    rtk_mqtt_client_publish_event(g_mqtt_client, "device.lifecycle.startup", 
                                 "跨平台設備已啟動");
    
    // 啟動工作執行緒
    rtk_thread_t sensor_thread;
    if (rtk_thread_create(&sensor_thread, sensor_worker_thread, NULL) != 0) {
        printf("[%s] 錯誤: 工作執行緒創建失敗\n", get_platform_name());
        cleanup_device();
#ifdef RTK_PLATFORM_FREERTOS
        return;
#else
        return -1;
#endif
    }
    
    printf("[%s] 工作執行緒已啟動，設備正常運行\n", get_platform_name());
    printf("[%s] 按 Ctrl+C 停止設備 (或在 FreeRTOS 中等待外部停止信號)\n", get_platform_name());
    
#ifdef RTK_PLATFORM_FREERTOS
    // FreeRTOS: 主任務進入等待狀態
    while (g_running) {
        vTaskDelay(pdMS_TO_TICKS(1000));
    }
#else
    // POSIX/Windows: 等待信號或使用者中斷
    while (g_running) {
        rtk_sleep(1);
    }
    
    // 等待工作執行緒結束
    rtk_thread_join(&sensor_thread);
#endif
    
    // 清理資源
    cleanup_device();
    
    printf("\n[%s] 📊 設備運行總結:\n", get_platform_name());
    printf("   - 平台: %s\n", get_platform_name());
    printf("   - 設備 ID: %s\n", g_device.device_id);
    printf("   - 最終 CPU 使用率: %.1f%%\n", g_device.cpu_usage);
    printf("   - 最終記憶體使用率: %.1f%%\n", g_device.memory_usage);
    printf("   - 最終溫度: %.1f°C\n", g_device.temperature);
    
    printf("\n[%s] 🎉 跨平台設備範例執行完成！\n", get_platform_name());
    
#ifdef RTK_PLATFORM_FREERTOS
    // FreeRTOS 應用程式通常不會到達這裡
    vTaskDelete(NULL);
#else
    return 0;
#endif
}
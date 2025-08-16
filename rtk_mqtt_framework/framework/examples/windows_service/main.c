/**
 * @file main.c
 * @brief Windows 服務範例：MQTT 資料收集器
 * 
 * 本範例展示如何在 Windows 環境中使用 RTK MQTT Framework：
 * - Windows 服務的完整實作
 * - 高效能 MQTT 通訊
 * - JSON 資料處理和轉發
 * - 事件日誌記錄
 * - 服務控制和配置
 */

#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <winsvc.h>
#include <tchar.h>
#include <strsafe.h>
#include <stdio.h>
#include <stdlib.h>
#include <signal.h>

#include "rtk_mqtt_client.h"
#include "rtk_platform_compat.h"
#include "rtk_json_config.h"
#include "rtk_topic_builder.h"
#include "rtk_message_codec.h"

// === 服務配置 ===

#define SERVICE_NAME            TEXT("RTKMqttCollector")
#define SERVICE_DISPLAY_NAME    TEXT("RTK MQTT Data Collector")
#define SERVICE_DESCRIPTION     TEXT("Collects and forwards MQTT device data using RTK Framework")

#define APP_MQTT_BROKER_HOST    "mqtt.example.com"
#define APP_MQTT_BROKER_PORT    1883
#define APP_MQTT_CLIENT_ID      "windows_collector_001"
#define APP_MQTT_USERNAME       "collector"
#define APP_MQTT_PASSWORD       "password"

#define APP_WORKER_THREAD_COUNT 4
#define APP_MAX_PENDING_MESSAGES 1000
#define APP_STATS_REPORT_INTERVAL_MS 60000  // 1 分鐘

// === 全域變數 ===

static SERVICE_STATUS g_service_status = {0};
static SERVICE_STATUS_HANDLE g_service_status_handle = NULL;
static HANDLE g_service_stop_event = NULL;
static HANDLE g_worker_threads[APP_WORKER_THREAD_COUNT] = {0};
static rtk_mqtt_client_t* g_mqtt_client = NULL;

// 統計資料
typedef struct {
    DWORD messages_received;
    DWORD messages_processed;
    DWORD messages_forwarded;
    DWORD messages_error;
    DWORD bytes_received;
    DWORD bytes_processed;
    SYSTEMTIME last_reset_time;
} service_stats_t;

static service_stats_t g_stats = {0};
static CRITICAL_SECTION g_stats_cs;

// 訊息佇列
typedef struct message_item {
    char* topic;
    char* payload;
    size_t payload_len;
    SYSTEMTIME timestamp;
    struct message_item* next;
} message_item_t;

static message_item_t* g_message_queue_head = NULL;
static message_item_t* g_message_queue_tail = NULL;
static CRITICAL_SECTION g_message_queue_cs;
static HANDLE g_message_available_event = NULL;
static DWORD g_queue_size = 0;

// === 事件日誌輔助函數 ===

/**
 * @brief 寫入事件日誌
 */
static void write_event_log(WORD type, DWORD event_id, LPCTSTR message)
{
    HANDLE event_source = RegisterEventSource(NULL, SERVICE_NAME);
    if (event_source) {
        LPCTSTR strings[1] = {message};
        ReportEvent(event_source, type, 0, event_id, NULL, 1, 0, strings, NULL);
        DeregisterEventSource(event_source);
    }
    
    // 同時輸出到控制台 (除錯模式)
#ifdef _DEBUG
    _tprintf(TEXT("[%s] %s\n"), 
             (type == EVENTLOG_ERROR_TYPE) ? TEXT("ERROR") :
             (type == EVENTLOG_WARNING_TYPE) ? TEXT("WARN") : TEXT("INFO"),
             message);
#endif
}

/**
 * @brief 格式化事件日誌訊息
 */
static void write_event_log_formatted(WORD type, DWORD event_id, LPCTSTR format, ...)
{
    TCHAR buffer[1024];
    va_list args;
    
    va_start(args, format);
    StringCchVPrintf(buffer, ARRAYSIZE(buffer), format, args);
    va_end(args);
    
    write_event_log(type, event_id, buffer);
}

// === 訊息佇列管理 ===

/**
 * @brief 新增訊息到佇列
 */
static BOOL enqueue_message(const char* topic, const char* payload, size_t payload_len)
{
    if (g_queue_size >= APP_MAX_PENDING_MESSAGES) {
        InterlockedIncrement(&g_stats.messages_error);
        return FALSE;  // 佇列已滿
    }
    
    message_item_t* item = (message_item_t*)HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, sizeof(message_item_t));
    if (!item) return FALSE;
    
    // 分配並複製主題
    size_t topic_len = strlen(topic) + 1;
    item->topic = (char*)HeapAlloc(GetProcessHeap(), 0, topic_len);
    if (!item->topic) {
        HeapFree(GetProcessHeap(), 0, item);
        return FALSE;
    }
    strcpy_s(item->topic, topic_len, topic);
    
    // 分配並複製內容
    item->payload = (char*)HeapAlloc(GetProcessHeap(), 0, payload_len + 1);
    if (!item->payload) {
        HeapFree(GetProcessHeap(), 0, item->topic);
        HeapFree(GetProcessHeap(), 0, item);
        return FALSE;
    }
    memcpy(item->payload, payload, payload_len);
    item->payload[payload_len] = '\0';
    item->payload_len = payload_len;
    
    GetSystemTime(&item->timestamp);
    item->next = NULL;
    
    EnterCriticalSection(&g_message_queue_cs);
    
    if (g_message_queue_tail) {
        g_message_queue_tail->next = item;
    } else {
        g_message_queue_head = item;
    }
    g_message_queue_tail = item;
    g_queue_size++;
    
    LeaveCriticalSection(&g_message_queue_cs);
    
    SetEvent(g_message_available_event);
    InterlockedIncrement(&g_stats.messages_received);
    InterlockedExchangeAdd(&g_stats.bytes_received, (LONG)payload_len);
    
    return TRUE;
}

/**
 * @brief 從佇列取出訊息
 */
static message_item_t* dequeue_message(void)
{
    message_item_t* item = NULL;
    
    EnterCriticalSection(&g_message_queue_cs);
    
    if (g_message_queue_head) {
        item = g_message_queue_head;
        g_message_queue_head = item->next;
        if (!g_message_queue_head) {
            g_message_queue_tail = NULL;
        }
        g_queue_size--;
        
        if (g_queue_size == 0) {
            ResetEvent(g_message_available_event);
        }
    }
    
    LeaveCriticalSection(&g_message_queue_cs);
    
    return item;
}

/**
 * @brief 釋放訊息項目
 */
static void free_message_item(message_item_t* item)
{
    if (item) {
        if (item->topic) HeapFree(GetProcessHeap(), 0, item->topic);
        if (item->payload) HeapFree(GetProcessHeap(), 0, item->payload);
        HeapFree(GetProcessHeap(), 0, item);
    }
}

// === MQTT 回調函數 ===

/**
 * @brief MQTT 連接狀態回調
 */
static void mqtt_connection_callback(int connected, void* user_data)
{
    (void)user_data;
    
    if (connected) {
        write_event_log(EVENTLOG_INFORMATION_TYPE, 0, TEXT("Connected to MQTT broker"));
        
        // 訂閱所有設備資料主題
        rtk_mqtt_subscribe(g_mqtt_client, "devices/+/data", 1);
        rtk_mqtt_subscribe(g_mqtt_client, "sensors/+/readings", 1);
        rtk_mqtt_subscribe(g_mqtt_client, "gateways/+/status", 1);
        
    } else {
        write_event_log(EVENTLOG_WARNING_TYPE, 0, TEXT("Disconnected from MQTT broker"));
    }
}

/**
 * @brief MQTT 訊息接收回調
 */
static void mqtt_message_callback(const char* topic, const char* payload, size_t payload_len, void* user_data)
{
    (void)user_data;
    
    // 將訊息加入處理佇列
    if (!enqueue_message(topic, payload, payload_len)) {
        write_event_log_formatted(EVENTLOG_WARNING_TYPE, 0, 
                                TEXT("Failed to enqueue message from topic: %hs"), topic);
    }
}

// === 訊息處理函數 ===

/**
 * @brief 處理設備資料訊息
 */
static BOOL process_device_data(const char* topic, const char* payload, size_t payload_len)
{
    // 解析 JSON 資料
    char* json_buffer = rtk_json_alloc_buffer();
    if (!json_buffer) return FALSE;
    
    if (payload_len >= RTK_JSON_BUFFER_SIZE) {
        rtk_json_free_buffer(json_buffer);
        return FALSE;
    }
    
    memcpy(json_buffer, payload, payload_len);
    json_buffer[payload_len] = '\0';
    
    cJSON* json = rtk_json_parse_with_stats(json_buffer);
    if (!json) {
        rtk_json_free_buffer(json_buffer);
        return FALSE;
    }
    
    // 提取設備資訊
    const char* device_id = RTK_JSON_GET_STRING_SAFE(json, "device_id", "unknown");
    double temperature = RTK_JSON_GET_NUMBER_SAFE(json, "temperature", 0.0);
    double humidity = RTK_JSON_GET_NUMBER_SAFE(json, "humidity", 0.0);
    double timestamp = RTK_JSON_GET_NUMBER_SAFE(json, "timestamp", 0.0);
    
    // 建立轉發資料
    cJSON* forward_data = cJSON_CreateObject();
    if (forward_data) {
        // 新增處理時間戳
        SYSTEMTIME sys_time;
        GetSystemTime(&sys_time);
        FILETIME file_time;
        SystemTimeToFileTime(&sys_time, &file_time);
        ULARGE_INTEGER time_value;
        time_value.LowPart = file_time.dwLowDateTime;
        time_value.HighPart = file_time.dwHighDateTime;
        double processed_time = (double)(time_value.QuadPart / 10000);  // 轉為毫秒
        
        RTK_JSON_ADD_STRING_SAFE(forward_data, "device_id", device_id);
        RTK_JSON_ADD_NUMBER_SAFE(forward_data, "temperature", temperature);
        RTK_JSON_ADD_NUMBER_SAFE(forward_data, "humidity", humidity);
        RTK_JSON_ADD_NUMBER_SAFE(forward_data, "original_timestamp", timestamp);
        RTK_JSON_ADD_NUMBER_SAFE(forward_data, "processed_timestamp", processed_time);
        RTK_JSON_ADD_STRING_SAFE(forward_data, "processor", "windows_collector");
        RTK_JSON_ADD_STRING_SAFE(forward_data, "original_topic", topic);
        
        char* forward_json = rtk_json_print_with_stats(forward_data, 1);
        if (forward_json) {
            // 轉發到處理過的資料主題
            char forward_topic[256];
            sprintf_s(forward_topic, sizeof(forward_topic), "processed/devices/%s/data", device_id);
            
            int result = rtk_mqtt_publish_simple(g_mqtt_client, forward_topic, forward_json, 0, 0);
            if (result == RTK_PLATFORM_SUCCESS) {
                InterlockedIncrement(&g_stats.messages_forwarded);
                InterlockedExchangeAdd(&g_stats.bytes_processed, (LONG)strlen(forward_json));
            }
            
            rtk_json_free_string_safe(forward_json);
        }
        
        rtk_json_delete_safe(forward_data);
    }
    
    rtk_json_delete_safe(json);
    rtk_json_free_buffer(json_buffer);
    
    InterlockedIncrement(&g_stats.messages_processed);
    return TRUE;
}

// === 工作執行緒 ===

/**
 * @brief 訊息處理工作執行緒
 */
static DWORD WINAPI worker_thread_proc(LPVOID param)
{
    DWORD thread_id = GetCurrentThreadId();
    
    write_event_log_formatted(EVENTLOG_INFORMATION_TYPE, 0, 
                             TEXT("Worker thread %d started"), thread_id);
    
    while (WaitForSingleObject(g_service_stop_event, 0) != WAIT_OBJECT_0) {
        // 等待訊息可用
        HANDLE events[] = {g_service_stop_event, g_message_available_event};
        DWORD wait_result = WaitForMultipleObjects(2, events, FALSE, 1000);
        
        if (wait_result == WAIT_OBJECT_0) {
            break;  // 服務停止
        }
        
        if (wait_result == WAIT_OBJECT_0 + 1 || wait_result == WAIT_TIMEOUT) {
            // 處理佇列中的訊息
            message_item_t* message;
            while ((message = dequeue_message()) != NULL) {
                // 根據主題類型處理訊息
                if (strstr(message->topic, "/data") != NULL) {
                    process_device_data(message->topic, message->payload, message->payload_len);
                } else {
                    // 其他類型的訊息直接轉發
                    InterlockedIncrement(&g_stats.messages_processed);
                }
                
                free_message_item(message);
                
                // 檢查是否需要停止
                if (WaitForSingleObject(g_service_stop_event, 0) == WAIT_OBJECT_0) {
                    break;
                }
            }
        }
    }
    
    write_event_log_formatted(EVENTLOG_INFORMATION_TYPE, 0, 
                             TEXT("Worker thread %d stopped"), thread_id);
    
    return 0;
}

/**
 * @brief 統計報告執行緒
 */
static DWORD WINAPI stats_thread_proc(LPVOID param)
{
    (void)param;
    
    while (WaitForSingleObject(g_service_stop_event, APP_STATS_REPORT_INTERVAL_MS) != WAIT_OBJECT_0) {
        // 報告統計資訊
        EnterCriticalSection(&g_stats_cs);
        
        write_event_log_formatted(EVENTLOG_INFORMATION_TYPE, 0,
                                TEXT("Stats: Received=%d, Processed=%d, Forwarded=%d, Errors=%d, Queue=%d"),
                                g_stats.messages_received,
                                g_stats.messages_processed,
                                g_stats.messages_forwarded,
                                g_stats.messages_error,
                                g_queue_size);
        
        // JSON 池統計
        int pool_usage = rtk_json_get_pool_usage();
        rtk_json_stats_t json_stats;
        if (rtk_json_get_stats(&json_stats) == RTK_PLATFORM_SUCCESS) {
            write_event_log_formatted(EVENTLOG_INFORMATION_TYPE, 0,
                                    TEXT("JSON Stats: Pool=%d%%, Parse=%d/%d, Avg=%dμs"),
                                    pool_usage,
                                    json_stats.parse_success_count,
                                    json_stats.parse_count,
                                    json_stats.avg_parse_time_us);
        }
        
        LeaveCriticalSection(&g_stats_cs);
    }
    
    return 0;
}

// === 服務控制處理 ===

/**
 * @brief 服務控制處理程序
 */
static VOID WINAPI service_ctrl_handler(DWORD ctrl_code)
{
    switch (ctrl_code) {
        case SERVICE_CONTROL_STOP:
            write_event_log(EVENTLOG_INFORMATION_TYPE, 0, TEXT("Service stop requested"));
            
            g_service_status.dwControlsAccepted = 0;
            g_service_status.dwCurrentState = SERVICE_STOP_PENDING;
            g_service_status.dwWin32ExitCode = 0;
            g_service_status.dwCheckPoint = 4;
            
            if (SetServiceStatus(g_service_status_handle, &g_service_status) == FALSE) {
                write_event_log(EVENTLOG_ERROR_TYPE, 0, TEXT("SetServiceStatus failed"));
            }
            
            SetEvent(g_service_stop_event);
            break;
            
        case SERVICE_CONTROL_INTERROGATE:
            break;
            
        default:
            break;
    }
}

/**
 * @brief 服務主函數
 */
static VOID WINAPI service_main(DWORD argc, LPTSTR* argv)
{
    (void)argc;
    (void)argv;
    
    // 註冊服務控制處理程序
    g_service_status_handle = RegisterServiceCtrlHandler(SERVICE_NAME, service_ctrl_handler);
    if (g_service_status_handle == NULL) {
        return;
    }
    
    // 初始化服務狀態
    ZeroMemory(&g_service_status, sizeof(g_service_status));
    g_service_status.dwServiceType = SERVICE_WIN32_OWN_PROCESS;
    g_service_status.dwControlsAccepted = 0;
    g_service_status.dwCurrentState = SERVICE_START_PENDING;
    g_service_status.dwWin32ExitCode = 0;
    g_service_status.dwServiceSpecificExitCode = 0;
    g_service_status.dwCheckPoint = 0;
    
    if (SetServiceStatus(g_service_status_handle, &g_service_status) == FALSE) {
        write_event_log(EVENTLOG_ERROR_TYPE, 0, TEXT("SetServiceStatus failed"));
        return;
    }
    
    write_event_log(EVENTLOG_INFORMATION_TYPE, 0, TEXT("Service starting..."));
    
    // 初始化同步物件
    InitializeCriticalSection(&g_stats_cs);
    InitializeCriticalSection(&g_message_queue_cs);
    
    g_service_stop_event = CreateEvent(NULL, TRUE, FALSE, NULL);
    g_message_available_event = CreateEvent(NULL, FALSE, FALSE, NULL);
    
    if (!g_service_stop_event || !g_message_available_event) {
        write_event_log(EVENTLOG_ERROR_TYPE, 0, TEXT("Failed to create events"));
        g_service_status.dwCurrentState = SERVICE_STOPPED;
        g_service_status.dwWin32ExitCode = GetLastError();
        SetServiceStatus(g_service_status_handle, &g_service_status);
        return;
    }
    
    // 初始化 JSON 記憶體池
    if (rtk_json_pool_init() != RTK_PLATFORM_SUCCESS) {
        write_event_log(EVENTLOG_ERROR_TYPE, 0, TEXT("Failed to initialize JSON pool"));
        g_service_status.dwCurrentState = SERVICE_STOPPED;
        g_service_status.dwWin32ExitCode = ERROR_INTERNAL_ERROR;
        SetServiceStatus(g_service_status_handle, &g_service_status);
        return;
    }
    
    // 初始化 MQTT 客戶端
    rtk_mqtt_config_t mqtt_config = {
        .client_id = APP_MQTT_CLIENT_ID,
        .broker_host = APP_MQTT_BROKER_HOST,
        .broker_port = APP_MQTT_BROKER_PORT,
        .username = APP_MQTT_USERNAME,
        .password = APP_MQTT_PASSWORD,
        .keep_alive = 60,
        .clean_session = 1,
        .connection_callback = mqtt_connection_callback,
        .message_callback = mqtt_message_callback,
        .user_data = NULL
    };
    
    g_mqtt_client = rtk_mqtt_create(&mqtt_config);
    if (!g_mqtt_client) {
        write_event_log(EVENTLOG_ERROR_TYPE, 0, TEXT("Failed to create MQTT client"));
        g_service_status.dwCurrentState = SERVICE_STOPPED;
        g_service_status.dwWin32ExitCode = ERROR_INTERNAL_ERROR;
        SetServiceStatus(g_service_status_handle, &g_service_status);
        return;
    }
    
    // 連接到 MQTT Broker
    if (rtk_mqtt_connect(g_mqtt_client) != RTK_PLATFORM_SUCCESS) {
        write_event_log(EVENTLOG_ERROR_TYPE, 0, TEXT("Failed to connect to MQTT broker"));
        g_service_status.dwCurrentState = SERVICE_STOPPED;
        g_service_status.dwWin32ExitCode = ERROR_NETWORK_UNREACHABLE;
        SetServiceStatus(g_service_status_handle, &g_service_status);
        return;
    }
    
    // 建立工作執行緒
    for (int i = 0; i < APP_WORKER_THREAD_COUNT; i++) {
        g_worker_threads[i] = CreateThread(NULL, 0, worker_thread_proc, NULL, 0, NULL);
        if (!g_worker_threads[i]) {
            write_event_log_formatted(EVENTLOG_ERROR_TYPE, 0, 
                                    TEXT("Failed to create worker thread %d"), i);
        }
    }
    
    // 建立統計報告執行緒
    HANDLE stats_thread = CreateThread(NULL, 0, stats_thread_proc, NULL, 0, NULL);
    
    // 初始化統計
    GetSystemTime(&g_stats.last_reset_time);
    
    // 服務已啟動
    g_service_status.dwControlsAccepted = SERVICE_ACCEPT_STOP;
    g_service_status.dwCurrentState = SERVICE_RUNNING;
    g_service_status.dwWin32ExitCode = 0;
    g_service_status.dwCheckPoint = 0;
    
    if (SetServiceStatus(g_service_status_handle, &g_service_status) == FALSE) {
        write_event_log(EVENTLOG_ERROR_TYPE, 0, TEXT("SetServiceStatus failed"));
    }
    
    write_event_log(EVENTLOG_INFORMATION_TYPE, 0, TEXT("Service started successfully"));
    
    // 主服務迴圈
    while (WaitForSingleObject(g_service_stop_event, 1000) != WAIT_OBJECT_0) {
        // 處理 MQTT 事件
        if (g_mqtt_client) {
            rtk_mqtt_loop(g_mqtt_client, 10);
        }
    }
    
    // 服務正在停止
    write_event_log(EVENTLOG_INFORMATION_TYPE, 0, TEXT("Service stopping..."));
    
    // 等待所有工作執行緒結束
    WaitForMultipleObjects(APP_WORKER_THREAD_COUNT, g_worker_threads, TRUE, 5000);
    
    // 等待統計執行緒結束
    if (stats_thread) {
        WaitForSingleObject(stats_thread, 1000);
        CloseHandle(stats_thread);
    }
    
    // 清理資源
    for (int i = 0; i < APP_WORKER_THREAD_COUNT; i++) {
        if (g_worker_threads[i]) {
            CloseHandle(g_worker_threads[i]);
        }
    }
    
    if (g_mqtt_client) {
        rtk_mqtt_disconnect(g_mqtt_client);
        rtk_mqtt_destroy(g_mqtt_client);
    }
    
    rtk_json_pool_cleanup();
    
    // 清理佇列中剩餘的訊息
    message_item_t* item;
    while ((item = dequeue_message()) != NULL) {
        free_message_item(item);
    }
    
    DeleteCriticalSection(&g_stats_cs);
    DeleteCriticalSection(&g_message_queue_cs);
    
    if (g_service_stop_event) CloseHandle(g_service_stop_event);
    if (g_message_available_event) CloseHandle(g_message_available_event);
    
    write_event_log(EVENTLOG_INFORMATION_TYPE, 0, TEXT("Service stopped"));
    
    g_service_status.dwControlsAccepted = 0;
    g_service_status.dwCurrentState = SERVICE_STOPPED;
    g_service_status.dwWin32ExitCode = 0;
    g_service_status.dwCheckPoint = 3;
    
    SetServiceStatus(g_service_status_handle, &g_service_status);
}

// === 主程式 ===

/**
 * @brief 控制台模式執行 (除錯用)
 */
static void run_console_mode(void)
{
    printf("=== RTK MQTT Framework - Windows Service Example ===\n");
    printf("Running in console mode for debugging...\n");
    printf("Press Ctrl+C to stop\n\n");
    
    // 模擬服務主函數
    LPTSTR argv[] = {TEXT("console")};
    service_main(1, argv);
}

/**
 * @brief 主程式入口點
 */
int _tmain(int argc, TCHAR* argv[])
{
    if (argc > 1) {
        if (_tcscmp(argv[1], TEXT("console")) == 0) {
            // 控制台模式
            run_console_mode();
            return 0;
        }
        
        if (_tcscmp(argv[1], TEXT("install")) == 0) {
            // 安裝服務 (這裡僅作示範，實際應用需要更完整的安裝程式)
            printf("Service installation not implemented in this example.\n");
            printf("Please use 'sc create' command or service installer.\n");
            return 0;
        }
        
        if (_tcscmp(argv[1], TEXT("uninstall")) == 0) {
            // 移除服務
            printf("Service uninstallation not implemented in this example.\n");
            printf("Please use 'sc delete' command.\n");
            return 0;
        }
    }
    
    // 正常服務模式
    SERVICE_TABLE_ENTRY service_table[] = {
        {SERVICE_NAME, (LPSERVICE_MAIN_FUNCTION)service_main},
        {NULL, NULL}
    };
    
    if (StartServiceCtrlDispatcher(service_table) == FALSE) {
        DWORD error = GetLastError();
        if (error == ERROR_FAILED_SERVICE_CONTROLLER_CONNECT) {
            printf("This program must be run as a Windows Service.\n");
            printf("Use '%s console' for console mode debugging.\n", argv[0]);
        } else {
            printf("StartServiceCtrlDispatcher failed with error: %d\n", error);
        }
        return error;
    }
    
    return 0;
}
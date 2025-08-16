/**
 * @file network_example.c
 * @brief FreeRTOS 網路抽象層使用範例
 * 
 * 展示如何在 FreeRTOS 環境中使用 RTK 網路介面
 * 支援 FreeRTOS+TCP 和 lwIP 兩種網路堆疊
 */

#include "rtk_network_interface.h"
#include "rtk_platform_compat.h"
#include "FreeRTOS.h"
#include "task.h"

// === 應用配置 ===
#define MQTT_BROKER_HOST "mqtt.eclipse.org"
#define MQTT_BROKER_PORT 1883
#define CONNECT_TIMEOUT_MS 30000
#define SEND_TIMEOUT_MS 10000
#define RECV_TIMEOUT_MS 10000

// === 全域變數 ===
static rtk_network_interface_t g_network_interface;
static TaskHandle_t g_network_task_handle = NULL;

// === 網路任務 ===
static void network_demo_task(void *pvParameters) {
    (void)pvParameters;
    
    int result;
    size_t sent, received;
    char buffer[1024];
    
    printf("=== RTK Network Interface Demo (FreeRTOS) ===\n");
    
    // 1. 初始化網路介面
    printf("1. Initializing network interface...\n");
    result = rtk_network_create_freertos(&g_network_interface);
    if (result != RTK_NETWORK_SUCCESS) {
        printf("Failed to create network interface: %d\n", result);
        goto cleanup;
    }
    printf("Network interface created successfully\n");
    
    // 2. 設定網路參數
    printf("2. Configuring network timeouts...\n");
    result = rtk_network_freertos_set_timeouts(&g_network_interface, 
                                              CONNECT_TIMEOUT_MS,
                                              SEND_TIMEOUT_MS, 
                                              RECV_TIMEOUT_MS);
    if (result != RTK_NETWORK_SUCCESS) {
        printf("Failed to set timeouts: %d\n", result);
        goto cleanup;
    }
    printf("Timeouts configured: connect=%dms, send=%dms, recv=%dms\n",
           CONNECT_TIMEOUT_MS, SEND_TIMEOUT_MS, RECV_TIMEOUT_MS);
    
    // 3. 連線到 MQTT 伺服器
    printf("3. Connecting to %s:%d...\n", MQTT_BROKER_HOST, MQTT_BROKER_PORT);
    result = g_network_interface.connect(&g_network_interface, 
                                        MQTT_BROKER_HOST, 
                                        MQTT_BROKER_PORT);
    if (result != RTK_NETWORK_SUCCESS) {
        printf("Failed to connect: %d\n", result);
        goto cleanup;
    }
    printf("Connected successfully!\n");
    
    // 4. 檢查連線狀態
    printf("4. Checking connection status...\n");
    rtk_network_status_t status;
    result = g_network_interface.get_status(&g_network_interface, &status);
    if (result == RTK_NETWORK_SUCCESS) {
        printf("Status: connected=%d, host=%s, port=%d, socket_fd=%d\n",
               status.connected, status.remote_host, 
               status.remote_port, status.socket_fd);
    }
    
    // 5. 發送簡單的 HTTP 請求 (用於測試)
    printf("5. Sending test HTTP request...\n");
    const char* http_request = 
        "GET / HTTP/1.1\r\n"
        "Host: " MQTT_BROKER_HOST "\r\n"
        "Connection: close\r\n\r\n";
    
    result = g_network_interface.send(&g_network_interface, 
                                     http_request, 
                                     strlen(http_request), 
                                     &sent);
    if (result != RTK_NETWORK_SUCCESS) {
        printf("Failed to send data: %d\n", result);
        goto cleanup;
    }
    printf("Sent %zu bytes\n", sent);
    
    // 6. 接收回應
    printf("6. Receiving response...\n");
    vTaskDelay(pdMS_TO_TICKS(1000)); // 等待回應
    
    result = g_network_interface.receive(&g_network_interface, 
                                        buffer, 
                                        sizeof(buffer) - 1, 
                                        &received);
    if (result == RTK_NETWORK_SUCCESS && received > 0) {
        buffer[received] = '\0';
        printf("Received %zu bytes:\n%s\n", received, buffer);
    } else {
        printf("No data received or error: %d\n", result);
    }
    
    // 7. 網路狀態監控循環
    printf("7. Starting network monitoring loop...\n");
    int loop_count = 0;
    while (loop_count < 10) {
        // 檢查連線狀態
        int connected = g_network_interface.is_connected(&g_network_interface);
        printf("Loop %d: Connection status = %s\n", 
               loop_count + 1, connected ? "Connected" : "Disconnected");
        
        if (!connected) {
            printf("Connection lost, breaking loop\n");
            break;
        }
        
        // 嘗試發送 keepalive 封包
        const char* keepalive = "PING\r\n";
        result = g_network_interface.send(&g_network_interface,
                                         keepalive,
                                         strlen(keepalive),
                                         &sent);
        if (result == RTK_NETWORK_SUCCESS) {
            printf("Keepalive sent: %zu bytes\n", sent);
        } else {
            printf("Keepalive failed: %d\n", result);
        }
        
        loop_count++;
        vTaskDelay(pdMS_TO_TICKS(5000)); // 5 秒間隔
    }
    
cleanup:
    // 8. 清理和斷線
    printf("8. Cleaning up...\n");
    if (g_network_interface.is_connected(&g_network_interface)) {
        g_network_interface.disconnect(&g_network_interface);
        printf("Disconnected from server\n");
    }
    
    if (g_network_interface.cleanup) {
        g_network_interface.cleanup(&g_network_interface);
        printf("Network interface cleaned up\n");
    }
    
    printf("=== Network Demo Completed ===\n");
    
    // 刪除任務
    g_network_task_handle = NULL;
    vTaskDelete(NULL);
}

// === 錯誤處理輔助函式 ===
static void print_network_error(int error_code) {
    const char* error_msg = rtk_network_get_error_string(error_code);
    printf("Network Error [%d]: %s\n", error_code, error_msg);
    
    // 可選：獲取更詳細的錯誤資訊
    const char* last_error = rtk_network_get_last_error();
    if (last_error) {
        printf("Last Error: %s\n", last_error);
    }
}

// === 網路事件處理 ===
static void network_event_handler(rtk_network_event_t event, void* event_data) {
    switch (event) {
        case RTK_NETWORK_EVENT_CONNECTED:
            printf("Network Event: Connected\n");
            break;
            
        case RTK_NETWORK_EVENT_DISCONNECTED:
            printf("Network Event: Disconnected\n");
            break;
            
        case RTK_NETWORK_EVENT_DATA_RECEIVED:
            printf("Network Event: Data received\n");
            break;
            
        case RTK_NETWORK_EVENT_ERROR:
            printf("Network Event: Error occurred\n");
            if (event_data) {
                int* error_code = (int*)event_data;
                print_network_error(*error_code);
            }
            break;
            
        default:
            printf("Network Event: Unknown event %d\n", event);
            break;
    }
}

// === 進階使用範例 ===
static void advanced_network_example(void) {
    rtk_network_config_t config;
    
    // 創建進階網路配置
    rtk_network_create_default_config(&config, RTK_NETWORK_TYPE_TCP);
    
    // 自定義配置
    config.socket_timeout_ms = 15000;
    config.connect_timeout_ms = 30000;
    config.recv_timeout_ms = 10000;
    config.send_timeout_ms = 10000;
    config.keep_alive = 1;
    config.tcp_nodelay = 1;
    config.reuse_addr = 1;
    
    // 驗證配置
    int result = rtk_network_validate_config(&config);
    if (result != RTK_NETWORK_SUCCESS) {
        printf("Invalid network configuration: %d\n", result);
        return;
    }
    
    // 配置網路介面
    result = rtk_network_configure(&config);
    if (result != RTK_NETWORK_SUCCESS) {
        printf("Failed to configure network: %d\n", result);
        return;
    }
    
    printf("Advanced network configuration applied\n");
}

// === 公開 API ===
int start_network_demo(void) {
    if (g_network_task_handle != NULL) {
        printf("Network demo task already running\n");
        return -1;
    }
    
    BaseType_t result = xTaskCreate(
        network_demo_task,           // 任務函式
        "NetworkDemo",               // 任務名稱
        4096,                        // 堆疊大小 (字)
        NULL,                        // 任務參數
        tskIDLE_PRIORITY + 3,        // 任務優先級
        &g_network_task_handle       // 任務控制代碼
    );
    
    if (result != pdPASS) {
        printf("Failed to create network demo task\n");
        return -1;
    }
    
    printf("Network demo task started\n");
    return 0;
}

int stop_network_demo(void) {
    if (g_network_task_handle == NULL) {
        printf("Network demo task not running\n");
        return -1;
    }
    
    vTaskDelete(g_network_task_handle);
    g_network_task_handle = NULL;
    
    printf("Network demo task stopped\n");
    return 0;
}

// === WiFi 連線整合範例 (ESP32) ===
#ifdef CONFIG_IDF_TARGET_ESP32
#include "esp_wifi.h"
#include "esp_event.h"

static void wifi_event_handler(void* arg, esp_event_base_t event_base,
                               int32_t event_id, void* event_data) {
    if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_START) {
        esp_wifi_connect();
    } else if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_DISCONNECTED) {
        printf("WiFi disconnected, retrying...\n");
        esp_wifi_connect();
    } else if (event_base == IP_EVENT && event_id == IP_EVENT_STA_GOT_IP) {
        ip_event_got_ip_t* event = (ip_event_got_ip_t*) event_data;
        printf("Got IP: " IPSTR "\n", IP2STR(&event->ip_info.ip));
        
        // WiFi 連線成功後啟動網路示範
        start_network_demo();
    }
}

int init_wifi_and_network(const char* ssid, const char* password) {
    // 初始化 TCP/IP 和事件循環
    esp_netif_init();
    esp_event_loop_create_default();
    esp_netif_create_default_wifi_sta();
    
    // 初始化 WiFi
    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
    esp_wifi_init(&cfg);
    
    // 註冊事件處理器
    esp_event_handler_register(WIFI_EVENT, ESP_EVENT_ANY_ID, &wifi_event_handler, NULL);
    esp_event_handler_register(IP_EVENT, IP_EVENT_STA_GOT_IP, &wifi_event_handler, NULL);
    
    // 配置 WiFi
    wifi_config_t wifi_config = {
        .sta = {
            .threshold.authmode = WIFI_AUTH_WPA2_PSK,
            .pmf_cfg = {
                .capable = true,
                .required = false
            },
        },
    };
    
    strcpy((char*)wifi_config.sta.ssid, ssid);
    strcpy((char*)wifi_config.sta.password, password);
    
    esp_wifi_set_mode(WIFI_MODE_STA);
    esp_wifi_set_config(ESP_IF_WIFI_STA, &wifi_config);
    esp_wifi_start();
    
    printf("WiFi initialization completed. Connecting to %s...\n", ssid);
    return 0;
}
#endif // CONFIG_IDF_TARGET_ESP32
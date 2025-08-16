/**
 * @file main.c
 * @brief FreeRTOS 設備範例：溫濕度感測器 MQTT 客戶端
 * 
 * 本範例展示如何在 FreeRTOS 環境中使用 RTK MQTT Framework：
 * - 初始化 FreeRTOS 任務和系統
 * - 連接到 MQTT Broker
 * - 定期發送感測器資料
 * - 處理遠程命令
 * - 管理設備狀態
 */

#include "FreeRTOS.h"
#include "task.h"
#include "queue.h"
#include "timers.h"
#include "semphr.h"

#include "rtk_mqtt_client.h"
#include "rtk_platform_compat.h"
#include "rtk_json_config.h"
#include "rtk_topic_builder.h"
#include "rtk_message_codec.h"

#include <stdio.h>
#include <string.h>
#include <stdlib.h>

// === 應用程式配置 ===

#define APP_DEVICE_ID           "FREERTOS_SENSOR_001"
#define APP_MQTT_BROKER_HOST    "mqtt.example.com"
#define APP_MQTT_BROKER_PORT    1883
#define APP_MQTT_CLIENT_ID      "freertos_client_001"
#define APP_MQTT_USERNAME       "device"
#define APP_MQTT_PASSWORD       "password"

#define APP_SENSOR_TASK_PRIORITY        (tskIDLE_PRIORITY + 2)
#define APP_MQTT_TASK_PRIORITY          (tskIDLE_PRIORITY + 3)
#define APP_COMMAND_TASK_PRIORITY       (tskIDLE_PRIORITY + 1)

#define APP_SENSOR_READ_INTERVAL_MS     5000    // 5 秒讀取一次感測器
#define APP_MQTT_PUBLISH_INTERVAL_MS    10000   // 10 秒發送一次資料
#define APP_HEARTBEAT_INTERVAL_MS       30000   // 30 秒發送心跳

#define APP_TASK_STACK_SIZE             2048

// === 感測器資料結構 ===

typedef struct {
    float temperature;      // 溫度 (攝氏度)
    float humidity;         // 濕度 (%)
    uint32_t timestamp;     // 時間戳
    uint8_t status;         // 感測器狀態
} sensor_data_t;

typedef struct {
    char command[32];       // 命令名稱
    char param[64];         // 命令參數
    uint32_t timestamp;     // 命令時間戳
} device_command_t;

// === 全域變數 ===

static rtk_mqtt_client_t* g_mqtt_client = NULL;
static QueueHandle_t g_sensor_data_queue = NULL;
static QueueHandle_t g_command_queue = NULL;
static SemaphoreHandle_t g_mqtt_connected_sem = NULL;
static TimerHandle_t g_heartbeat_timer = NULL;

static sensor_data_t g_latest_sensor_data;
static uint8_t g_device_online = 0;
static uint32_t g_message_counter = 0;

// === 感測器模擬函數 ===

/**
 * @brief 模擬讀取溫度感測器
 */
static float read_temperature_sensor(void)
{
    // 模擬溫度範圍：20-30度，加上隨機變化
    static float base_temp = 25.0f;
    float variation = ((float)(rand() % 100) - 50.0f) / 100.0f;  // -0.5 到 +0.5
    
    base_temp += variation * 0.1f;  // 緩慢變化
    if (base_temp < 20.0f) base_temp = 20.0f;
    if (base_temp > 30.0f) base_temp = 30.0f;
    
    return base_temp + ((float)(rand() % 20) - 10.0f) / 10.0f;  // 加入雜訊
}

/**
 * @brief 模擬讀取濕度感測器
 */
static float read_humidity_sensor(void)
{
    // 模擬濕度範圍：40-80%，加上隨機變化
    static float base_humidity = 60.0f;
    float variation = ((float)(rand() % 100) - 50.0f) / 100.0f;
    
    base_humidity += variation * 0.2f;
    if (base_humidity < 40.0f) base_humidity = 40.0f;
    if (base_humidity > 80.0f) base_humidity = 80.0f;
    
    return base_humidity + ((float)(rand() % 10) - 5.0f);
}

/**
 * @brief 讀取感測器資料
 */
static void read_sensor_data(sensor_data_t* data)
{
    if (!data) return;
    
    data->temperature = read_temperature_sensor();
    data->humidity = read_humidity_sensor();
    data->timestamp = xTaskGetTickCount() * portTICK_PERIOD_MS;
    data->status = 1;  // 正常狀態
    
    printf("[SENSOR] T: %.1f°C, H: %.1f%%, TS: %lu\n", 
           data->temperature, data->humidity, data->timestamp);
}

// === MQTT 回調函數 ===

/**
 * @brief MQTT 連接狀態回調
 */
static void mqtt_connection_callback(int connected, void* user_data)
{
    (void)user_data;
    
    if (connected) {
        printf("[MQTT] Connected to broker\n");
        g_device_online = 1;
        xSemaphoreGive(g_mqtt_connected_sem);
        
        // 訂閱命令主題
        char command_topic[128];
        snprintf(command_topic, sizeof(command_topic), "devices/%s/commands", APP_DEVICE_ID);
        rtk_mqtt_subscribe(g_mqtt_client, command_topic, 1);
        
        // 發送上線訊息
        char status_topic[128];
        snprintf(status_topic, sizeof(status_topic), "devices/%s/status", APP_DEVICE_ID);
        rtk_mqtt_publish_simple(g_mqtt_client, status_topic, "online", 1, 1);
        
    } else {
        printf("[MQTT] Disconnected from broker\n");
        g_device_online = 0;
    }
}

/**
 * @brief MQTT 訊息接收回調
 */
static void mqtt_message_callback(const char* topic, const char* payload, size_t payload_len, void* user_data)
{
    (void)user_data;
    
    printf("[MQTT] Received message on %s: %.*s\n", topic, (int)payload_len, payload);
    
    // 解析命令訊息
    if (strstr(topic, "/commands") != NULL) {
        // 解析 JSON 命令
        char* json_buffer = rtk_json_alloc_buffer();
        if (json_buffer && payload_len < RTK_JSON_BUFFER_SIZE - 1) {
            memcpy(json_buffer, payload, payload_len);
            json_buffer[payload_len] = '\0';
            
            cJSON* json = rtk_json_parse_with_stats(json_buffer);
            if (json) {
                device_command_t command;
                memset(&command, 0, sizeof(command));
                
                // 提取命令資訊
                const char* cmd = RTK_JSON_GET_STRING_SAFE(json, "command", "");
                const char* param = RTK_JSON_GET_STRING_SAFE(json, "parameter", "");
                
                strncpy(command.command, cmd, sizeof(command.command) - 1);
                strncpy(command.param, param, sizeof(command.param) - 1);
                command.timestamp = xTaskGetTickCount() * portTICK_PERIOD_MS;
                
                // 發送到命令處理任務
                if (xQueueSend(g_command_queue, &command, 100 / portTICK_PERIOD_MS) != pdTRUE) {
                    printf("[APP] Command queue full, dropping command\n");
                }
                
                rtk_json_delete_safe(json);
            }
            
            rtk_json_free_buffer(json_buffer);
        }
    }
}

// === FreeRTOS 任務 ===

/**
 * @brief 感測器讀取任務
 */
static void sensor_task(void* parameters)
{
    (void)parameters;
    
    sensor_data_t sensor_data;
    TickType_t last_read_time = xTaskGetTickCount();
    
    printf("[TASK] Sensor task started\n");
    
    while (1) {
        // 讀取感測器資料
        read_sensor_data(&sensor_data);
        g_latest_sensor_data = sensor_data;  // 保存最新資料
        
        // 發送到 MQTT 任務
        if (xQueueSend(g_sensor_data_queue, &sensor_data, 100 / portTICK_PERIOD_MS) != pdTRUE) {
            printf("[SENSOR] Data queue full, dropping data\n");
        }
        
        // 等待下次讀取
        vTaskDelayUntil(&last_read_time, APP_SENSOR_READ_INTERVAL_MS / portTICK_PERIOD_MS);
    }
}

/**
 * @brief MQTT 通訊任務
 */
static void mqtt_task(void* parameters)
{
    (void)parameters;
    
    sensor_data_t sensor_data;
    TickType_t last_publish_time = xTaskGetTickCount();
    
    printf("[TASK] MQTT task started\n");
    
    // 等待 MQTT 連接
    xSemaphoreTake(g_mqtt_connected_sem, portMAX_DELAY);
    
    while (1) {
        // 處理接收到的感測器資料
        if (xQueueReceive(g_sensor_data_queue, &sensor_data, 100 / portTICK_PERIOD_MS) == pdTRUE) {
            
            // 檢查是否該發送資料
            TickType_t current_time = xTaskGetTickCount();
            if ((current_time - last_publish_time) >= (APP_MQTT_PUBLISH_INTERVAL_MS / portTICK_PERIOD_MS)) {
                
                if (g_device_online && g_mqtt_client) {
                    // 建立 JSON 資料
                    char* json_buffer = rtk_json_alloc_buffer();
                    if (json_buffer) {
                        cJSON* json = cJSON_CreateObject();
                        if (json) {
                            RTK_JSON_ADD_STRING_SAFE(json, "device_id", APP_DEVICE_ID);
                            RTK_JSON_ADD_NUMBER_SAFE(json, "temperature", sensor_data.temperature);
                            RTK_JSON_ADD_NUMBER_SAFE(json, "humidity", sensor_data.humidity);
                            RTK_JSON_ADD_NUMBER_SAFE(json, "timestamp", sensor_data.timestamp);
                            RTK_JSON_ADD_NUMBER_SAFE(json, "status", sensor_data.status);
                            RTK_JSON_ADD_NUMBER_SAFE(json, "message_id", ++g_message_counter);
                            
                            char* json_string = rtk_json_print_with_stats(json, 1);
                            if (json_string) {
                                // 發送到 MQTT
                                char data_topic[128];
                                snprintf(data_topic, sizeof(data_topic), "devices/%s/data", APP_DEVICE_ID);
                                
                                int result = rtk_mqtt_publish_simple(g_mqtt_client, data_topic, json_string, 0, 0);
                                if (result == RTK_PLATFORM_SUCCESS) {
                                    printf("[MQTT] Published sensor data #%lu\n", g_message_counter);
                                } else {
                                    printf("[MQTT] Failed to publish data, error: %d\n", result);
                                }
                                
                                rtk_json_free_string_safe(json_string);
                            }
                            
                            rtk_json_delete_safe(json);
                        }
                        rtk_json_free_buffer(json_buffer);
                    }
                }
                
                last_publish_time = current_time;
            }
        }
        
        // 處理 MQTT 事件
        if (g_mqtt_client) {
            rtk_mqtt_loop(g_mqtt_client, 10);  // 10ms 超時
        }
    }
}

/**
 * @brief 命令處理任務
 */
static void command_task(void* parameters)
{
    (void)parameters;
    
    device_command_t command;
    
    printf("[TASK] Command task started\n");
    
    while (1) {
        if (xQueueReceive(g_command_queue, &command, portMAX_DELAY) == pdTRUE) {
            printf("[CMD] Processing command: %s, param: %s\n", command.command, command.param);
            
            // 處理不同命令
            if (strcmp(command.command, "status") == 0) {
                // 回報設備狀態
                if (g_device_online && g_mqtt_client) {
                    char* json_buffer = rtk_json_alloc_buffer();
                    if (json_buffer) {
                        cJSON* response = cJSON_CreateObject();
                        if (response) {
                            RTK_JSON_ADD_STRING_SAFE(response, "device_id", APP_DEVICE_ID);
                            RTK_JSON_ADD_STRING_SAFE(response, "command", "status");
                            RTK_JSON_ADD_STRING_SAFE(response, "status", "online");
                            RTK_JSON_ADD_NUMBER_SAFE(response, "uptime", xTaskGetTickCount() * portTICK_PERIOD_MS);
                            RTK_JSON_ADD_NUMBER_SAFE(response, "free_heap", xPortGetFreeHeapSize());
                            RTK_JSON_ADD_NUMBER_SAFE(response, "temperature", g_latest_sensor_data.temperature);
                            RTK_JSON_ADD_NUMBER_SAFE(response, "humidity", g_latest_sensor_data.humidity);
                            
                            char* response_string = rtk_json_print_with_stats(response, 1);
                            if (response_string) {
                                char response_topic[128];
                                snprintf(response_topic, sizeof(response_topic), "devices/%s/response", APP_DEVICE_ID);
                                rtk_mqtt_publish_simple(g_mqtt_client, response_topic, response_string, 0, 0);
                                rtk_json_free_string_safe(response_string);
                            }
                            
                            rtk_json_delete_safe(response);
                        }
                        rtk_json_free_buffer(json_buffer);
                    }
                }
                
            } else if (strcmp(command.command, "reboot") == 0) {
                printf("[CMD] Reboot command received, restarting system...\n");
                // 實際系統中應該進行安全關機程序
                vTaskDelay(1000 / portTICK_PERIOD_MS);
                // NVIC_SystemReset();  // 重啟系統
                
            } else if (strcmp(command.command, "set_interval") == 0) {
                // 設定資料發送間隔
                int new_interval = atoi(command.param);
                if (new_interval >= 1000 && new_interval <= 60000) {
                    printf("[CMD] Setting publish interval to %d ms\n", new_interval);
                    // 在實際應用中，應該更新發送間隔設定
                }
                
            } else {
                printf("[CMD] Unknown command: %s\n", command.command);
            }
        }
    }
}

/**
 * @brief 心跳定時器回調
 */
static void heartbeat_timer_callback(TimerHandle_t timer)
{
    (void)timer;
    
    if (g_device_online && g_mqtt_client) {
        char heartbeat_topic[128];
        snprintf(heartbeat_topic, sizeof(heartbeat_topic), "devices/%s/heartbeat", APP_DEVICE_ID);
        
        char heartbeat_msg[64];
        snprintf(heartbeat_msg, sizeof(heartbeat_msg), "%lu", xTaskGetTickCount() * portTICK_PERIOD_MS);
        
        rtk_mqtt_publish_simple(g_mqtt_client, heartbeat_topic, heartbeat_msg, 0, 0);
        printf("[HEARTBEAT] Sent at %s ms\n", heartbeat_msg);
    }
}

// === 主程式 ===

/**
 * @brief 應用程式初始化
 */
static int app_init(void)
{
    printf("[APP] Initializing FreeRTOS MQTT device...\n");
    
    // 初始化 JSON 記憶體池
    if (rtk_json_pool_init() != RTK_PLATFORM_SUCCESS) {
        printf("[ERROR] Failed to initialize JSON pool\n");
        return -1;
    }
    
    // 建立佇列
    g_sensor_data_queue = xQueueCreate(5, sizeof(sensor_data_t));
    g_command_queue = xQueueCreate(3, sizeof(device_command_t));
    
    if (!g_sensor_data_queue || !g_command_queue) {
        printf("[ERROR] Failed to create queues\n");
        return -1;
    }
    
    // 建立信號量
    g_mqtt_connected_sem = xSemaphoreCreateBinary();
    if (!g_mqtt_connected_sem) {
        printf("[ERROR] Failed to create semaphore\n");
        return -1;
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
        printf("[ERROR] Failed to create MQTT client\n");
        return -1;
    }
    
    // 連接到 MQTT Broker
    if (rtk_mqtt_connect(g_mqtt_client) != RTK_PLATFORM_SUCCESS) {
        printf("[ERROR] Failed to connect to MQTT broker\n");
        return -1;
    }
    
    // 建立心跳定時器
    g_heartbeat_timer = xTimerCreate(
        "Heartbeat",
        APP_HEARTBEAT_INTERVAL_MS / portTICK_PERIOD_MS,
        pdTRUE,  // 自動重載
        NULL,
        heartbeat_timer_callback
    );
    
    if (!g_heartbeat_timer) {
        printf("[ERROR] Failed to create heartbeat timer\n");
        return -1;
    }
    
    printf("[APP] Initialization completed\n");
    return 0;
}

/**
 * @brief 應用程式啟動
 */
static void app_start(void)
{
    // 建立任務
    xTaskCreate(sensor_task, "SensorTask", APP_TASK_STACK_SIZE, NULL, APP_SENSOR_TASK_PRIORITY, NULL);
    xTaskCreate(mqtt_task, "MqttTask", APP_TASK_STACK_SIZE, NULL, APP_MQTT_TASK_PRIORITY, NULL);
    xTaskCreate(command_task, "CommandTask", APP_TASK_STACK_SIZE, NULL, APP_COMMAND_TASK_PRIORITY, NULL);
    
    // 啟動心跳定時器
    xTimerStart(g_heartbeat_timer, 0);
    
    printf("[APP] All tasks started\n");
}

/**
 * @brief 主程式入口點
 */
int main(void)
{
    printf("=== RTK MQTT Framework - FreeRTOS Device Example ===\n");
    printf("Device ID: %s\n", APP_DEVICE_ID);
    printf("MQTT Broker: %s:%d\n", APP_MQTT_BROKER_HOST, APP_MQTT_BROKER_PORT);
    printf("==============================================\n\n");
    
    // 初始化應用程式
    if (app_init() != 0) {
        printf("[FATAL] Application initialization failed\n");
        return -1;
    }
    
    // 啟動應用程式
    app_start();
    
    // 啟動 FreeRTOS 排程器
    printf("[APP] Starting FreeRTOS scheduler...\n");
    vTaskStartScheduler();
    
    // 不應該執行到這裡
    printf("[FATAL] FreeRTOS scheduler returned\n");
    return -1;
}

// === FreeRTOS 鉤子函數 ===

void vApplicationStackOverflowHook(TaskHandle_t xTask, char* pcTaskName)
{
    printf("[FATAL] Stack overflow in task: %s\n", pcTaskName);
    (void)xTask;
    for (;;) {
        vTaskDelay(1000 / portTICK_PERIOD_MS);
    }
}

void vApplicationMallocFailedHook(void)
{
    printf("[FATAL] Memory allocation failed\n");
    for (;;) {
        vTaskDelay(1000 / portTICK_PERIOD_MS);
    }
}

void vApplicationIdleHook(void)
{
    // 空閒任務處理，可以在這裡實作低功耗模式
}

void vApplicationTickHook(void)
{
    // 系統 tick 處理
}
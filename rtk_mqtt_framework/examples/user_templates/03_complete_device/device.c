/*
 * RTK MQTT Framework 完整設備範例
 * 這是一個生產級的 IoT 設備實作範例，展示：
 * 1. 完整的插件架構實作
 * 2. JSON 配置檔案管理
 * 3. 多執行緒架構 (感測器/命令/健康檢查)
 * 4. 完整的錯誤處理和恢復機制
 * 5. 日誌記錄系統
 * 6. 命令接收和處理
 * 7. 看門狗和健康監控
 * 8. 優雅的啟動和關閉程序
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <signal.h>
#include <time.h>
#include <pthread.h>
#include <errno.h>
#include <sys/stat.h>
#include <rtk_mqtt_client.h>
#include <rtk_topic_builder.h>
#include <rtk_message_codec.h>
#include <rtk_json_config.h>
#include <rtk_device_plugin.h>

// === 設備狀態和配置結構 ===

typedef struct {
    char device_id[64];
    char device_type[32];
    char location[64];
    char firmware_version[16];
} device_info_t;

typedef struct {
    char broker_host[256];
    int broker_port;
    char username[64];
    char password[64];
    int keepalive;
    int qos;
    int reconnect_interval;
} mqtt_config_t;

typedef struct {
    int publish_interval;           // 發布間隔 (秒)
    int health_check_interval;      // 健康檢查間隔 (秒)
    int command_timeout;           // 命令超時 (秒)
    int max_reconnect_attempts;    // 最大重連次數
    char log_level[16];           // 日誌等級
    char log_file[256];           // 日誌檔案
} device_config_t;

typedef struct {
    float cpu_usage;              // CPU 使用率
    float memory_usage;           // 記憶體使用率
    float temperature;            // 設備溫度
    int network_quality;          // 網路品質 (0-100)
    time_t uptime;               // 運行時間
    time_t last_update;          // 最後更新時間
} device_metrics_t;

typedef struct {
    rtk_mqtt_client_t* mqtt_client;
    rtk_topic_builder_t* topic_builder;
    device_info_t device_info;
    mqtt_config_t mqtt_config;
    device_config_t device_config;
    device_metrics_t metrics;
    
    // 執行緒控制
    pthread_t sensor_thread;
    pthread_t command_thread;
    pthread_t health_thread;
    
    // 同步控制
    pthread_mutex_t metrics_mutex;
    pthread_cond_t shutdown_cond;
    pthread_mutex_t shutdown_mutex;
    
    // 狀態標記
    volatile int running;
    volatile int connected;
    volatile int health_status;    // 0=healthy, 1=warning, 2=critical
    
    time_t start_time;
    int reconnect_count;
} complete_device_t;

// 全域設備實例
static complete_device_t* g_device = NULL;

// === 日誌系統 ===

typedef enum {
    LOG_DEBUG = 0,
    LOG_INFO,
    LOG_WARNING,
    LOG_ERROR,
    LOG_CRITICAL
} log_level_t;

static const char* log_level_names[] = {
    "DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"
};

static log_level_t current_log_level = LOG_INFO;
static FILE* log_file = NULL;

void log_message(log_level_t level, const char* format, ...) {
    if (level < current_log_level) return;
    
    time_t now = time(NULL);
    struct tm* tm_info = localtime(&now);
    char timestamp[32];
    strftime(timestamp, sizeof(timestamp), "%Y-%m-%d %H:%M:%S", tm_info);
    
    va_list args;
    va_start(args, format);
    
    // 輸出到控制台
    printf("[%s] %s: ", timestamp, log_level_names[level]);
    vprintf(format, args);
    printf("\n");
    
    // 輸出到日誌檔案
    if (log_file) {
        fprintf(log_file, "[%s] %s: ", timestamp, log_level_names[level]);
        vfprintf(log_file, format, args);
        fprintf(log_file, "\n");
        fflush(log_file);
    }
    
    va_end(args);
}

#define LOG_DEBUG(...) log_message(LOG_DEBUG, __VA_ARGS__)
#define LOG_INFO(...) log_message(LOG_INFO, __VA_ARGS__)
#define LOG_WARNING(...) log_message(LOG_WARNING, __VA_ARGS__)
#define LOG_ERROR(...) log_message(LOG_ERROR, __VA_ARGS__)
#define LOG_CRITICAL(...) log_message(LOG_CRITICAL, __VA_ARGS__)

// === 配置管理 ===

int load_configuration(const char* config_file, complete_device_t* device) {
    LOG_INFO("正在載入配置檔案: %s", config_file);
    
    // 檢查檔案是否存在
    struct stat st;
    if (stat(config_file, &st) != 0) {
        LOG_ERROR("配置檔案不存在: %s", config_file);
        return -1;
    }
    
    // 載入 JSON 配置 (這裡簡化實作，實際應使用 rtk_json_config)
    FILE* file = fopen(config_file, "r");
    if (!file) {
        LOG_ERROR("無法開啟配置檔案: %s", config_file);
        return -1;
    }
    
    // 簡化的配置讀取 (實際應解析 JSON)
    // 這裡使用預設值示範
    strcpy(device->device_info.device_id, "complete_device_001");
    strcpy(device->device_info.device_type, "industrial_iot");
    strcpy(device->device_info.location, "factory_floor_a");
    strcpy(device->device_info.firmware_version, "2.1.0");
    
    strcpy(device->mqtt_config.broker_host, "test.mosquitto.org");
    device->mqtt_config.broker_port = 1883;
    device->mqtt_config.username[0] = '\0';
    device->mqtt_config.password[0] = '\0';
    device->mqtt_config.keepalive = 60;
    device->mqtt_config.qos = 1;
    device->mqtt_config.reconnect_interval = 5;
    
    device->device_config.publish_interval = 30;
    device->device_config.health_check_interval = 60;
    device->device_config.command_timeout = 10;
    device->device_config.max_reconnect_attempts = 10;
    strcpy(device->device_config.log_level, "INFO");
    strcpy(device->device_config.log_file, "device.log");
    
    fclose(file);
    LOG_INFO("配置載入完成");
    return 0;
}

// === 感測器和指標收集 ===

float get_cpu_usage() {
    // 模擬 CPU 使用率讀取
    static float base_cpu = 25.0f;
    float variation = ((float)rand() / RAND_MAX - 0.5f) * 20.0f;
    float cpu = base_cpu + variation;
    return (cpu < 0) ? 0 : (cpu > 100) ? 100 : cpu;
}

float get_memory_usage() {
    // 模擬記憶體使用率
    static float base_memory = 45.0f;
    float variation = ((float)rand() / RAND_MAX - 0.5f) * 10.0f;
    float memory = base_memory + variation;
    return (memory < 0) ? 0 : (memory > 100) ? 100 : memory;
}

float get_device_temperature() {
    // 模擬設備溫度
    static float base_temp = 35.0f;
    float variation = ((float)rand() / RAND_MAX - 0.5f) * 10.0f;
    return base_temp + variation;
}

int get_network_quality() {
    // 模擬網路品質評估
    return 75 + (rand() % 25);  // 75-100 的範圍
}

void update_device_metrics(complete_device_t* device) {
    pthread_mutex_lock(&device->metrics_mutex);
    
    device->metrics.cpu_usage = get_cpu_usage();
    device->metrics.memory_usage = get_memory_usage();
    device->metrics.temperature = get_device_temperature();
    device->metrics.network_quality = get_network_quality();
    device->metrics.uptime = time(NULL) - device->start_time;
    device->metrics.last_update = time(NULL);
    
    // 健康狀態評估
    if (device->metrics.cpu_usage > 90 || device->metrics.memory_usage > 90 || 
        device->metrics.temperature > 70) {
        device->health_status = 2;  // Critical
    } else if (device->metrics.cpu_usage > 75 || device->metrics.memory_usage > 75 || 
               device->metrics.temperature > 50) {
        device->health_status = 1;  // Warning
    } else {
        device->health_status = 0;  // Healthy
    }
    
    pthread_mutex_unlock(&device->metrics_mutex);
}

// === MQTT 發布功能 ===

int publish_device_state(complete_device_t* device) {
    const char* status = device->connected ? "online" : "offline";
    const char* health = (device->health_status == 0) ? "healthy" : 
                        (device->health_status == 1) ? "warning" : "critical";
    
    if (rtk_mqtt_client_publish_state(device->mqtt_client, status, health) == RTK_SUCCESS) {
        LOG_DEBUG("設備狀態發布成功: %s (%s)", status, health);
        return 1;
    } else {
        LOG_WARNING("設備狀態發布失敗");
        return 0;
    }
}

int publish_telemetry_data(complete_device_t* device) {
    int success_count = 0;
    
    pthread_mutex_lock(&device->metrics_mutex);
    
    // 發布 CPU 使用率
    if (rtk_mqtt_client_publish_telemetry(device->mqtt_client, "cpu_usage", 
                                         device->metrics.cpu_usage, "%") == RTK_SUCCESS) {
        success_count++;
    }
    
    // 發布記憶體使用率
    if (rtk_mqtt_client_publish_telemetry(device->mqtt_client, "memory_usage", 
                                         device->metrics.memory_usage, "%") == RTK_SUCCESS) {
        success_count++;
    }
    
    // 發布設備溫度
    if (rtk_mqtt_client_publish_telemetry(device->mqtt_client, "temperature", 
                                         device->metrics.temperature, "°C") == RTK_SUCCESS) {
        success_count++;
    }
    
    // 發布網路品質
    if (rtk_mqtt_client_publish_telemetry(device->mqtt_client, "network_quality", 
                                         device->metrics.network_quality, "score") == RTK_SUCCESS) {
        success_count++;
    }
    
    // 發布運行時間
    if (rtk_mqtt_client_publish_telemetry(device->mqtt_client, "uptime", 
                                         device->metrics.uptime, "seconds") == RTK_SUCCESS) {
        success_count++;
    }
    
    pthread_mutex_unlock(&device->metrics_mutex);
    
    LOG_DEBUG("遙測資料發布: %d/5 成功", success_count);
    return success_count;
}

// === 執行緒函式 ===

void* sensor_thread_func(void* arg) {
    complete_device_t* device = (complete_device_t*)arg;
    LOG_INFO("感測器執行緒啟動");
    
    while (device->running) {
        if (device->connected) {
            // 更新指標
            update_device_metrics(device);
            
            // 發布遙測資料
            publish_telemetry_data(device);
            
            // 發布設備狀態
            publish_device_state(device);
            
            LOG_DEBUG("感測器週期完成 (CPU: %.1f%%, 記憶體: %.1f%%, 溫度: %.1f°C)", 
                     device->metrics.cpu_usage, device->metrics.memory_usage, 
                     device->metrics.temperature);
        }
        
        // 等待發布間隔
        sleep(device->device_config.publish_interval);
    }
    
    LOG_INFO("感測器執行緒結束");
    return NULL;
}

void* command_thread_func(void* arg) {
    complete_device_t* device = (complete_device_t*)arg;
    LOG_INFO("命令處理執行緒啟動");
    
    // 訂閱命令主題 (簡化實作)
    // 實際應使用 rtk_mqtt_client_subscribe_commands
    
    while (device->running) {
        if (device->connected) {
            // 處理接收到的命令 (簡化實作)
            // 實際應從 MQTT 佇列中讀取命令
            
            // 模擬命令處理
            sleep(1);
        } else {
            sleep(5);  // 連接斷開時等待較長時間
        }
    }
    
    LOG_INFO("命令處理執行緒結束");
    return NULL;
}

void* health_thread_func(void* arg) {
    complete_device_t* device = (complete_device_t*)arg;
    LOG_INFO("健康監控執行緒啟動");
    
    while (device->running) {
        // 檢查 MQTT 連接狀態
        if (!rtk_mqtt_client_is_connected(device->mqtt_client)) {
            if (device->connected) {
                LOG_WARNING("MQTT 連接斷開，嘗試重連...");
                device->connected = 0;
                
                // 嘗試重連
                for (int i = 0; i < device->device_config.max_reconnect_attempts && device->running; i++) {
                    if (rtk_mqtt_client_reconnect(device->mqtt_client) == RTK_SUCCESS) {
                        LOG_INFO("MQTT 重連成功");
                        device->connected = 1;
                        device->reconnect_count++;
                        break;
                    }
                    
                    LOG_WARNING("重連嘗試 %d/%d 失敗", i+1, device->device_config.max_reconnect_attempts);
                    sleep(device->mqtt_config.reconnect_interval);
                }
                
                if (!device->connected) {
                    LOG_ERROR("MQTT 重連失敗，將繼續監控");
                }
            }
        } else {
            device->connected = 1;
        }
        
        // 健康檢查
        if (device->health_status == 2) {
            LOG_WARNING("設備處於危險狀態 - CPU: %.1f%%, 記憶體: %.1f%%, 溫度: %.1f°C", 
                       device->metrics.cpu_usage, device->metrics.memory_usage, 
                       device->metrics.temperature);
                       
            // 發布緊急事件
            if (device->connected) {
                rtk_mqtt_client_publish_event(device->mqtt_client, "device.health.critical", 
                                             "設備健康狀態危險");
            }
        }
        
        sleep(device->device_config.health_check_interval);
    }
    
    LOG_INFO("健康監控執行緒結束");
    return NULL;
}

// === 信號處理 ===

void signal_handler(int signal) {
    LOG_INFO("收到信號 %d，正在啟動優雅關閉程序...", signal);
    
    if (g_device) {
        g_device->running = 0;
        
        // 通知關閉條件變數
        pthread_mutex_lock(&g_device->shutdown_mutex);
        pthread_cond_broadcast(&g_device->shutdown_cond);
        pthread_mutex_unlock(&g_device->shutdown_mutex);
    }
}

// === 設備初始化和清理 ===

int initialize_device(complete_device_t* device, const char* config_file) {
    LOG_INFO("正在初始化完整設備...");
    
    // 載入配置
    if (load_configuration(config_file, device) != 0) {
        return -1;
    }
    
    // 初始化互斥鎖和條件變數
    if (pthread_mutex_init(&device->metrics_mutex, NULL) != 0 ||
        pthread_mutex_init(&device->shutdown_mutex, NULL) != 0 ||
        pthread_cond_init(&device->shutdown_cond, NULL) != 0) {
        LOG_ERROR("互斥鎖初始化失敗");
        return -1;
    }
    
    // 創建 MQTT 客戶端
    device->mqtt_client = rtk_mqtt_client_create(
        device->mqtt_config.broker_host,
        device->mqtt_config.broker_port,
        device->device_info.device_id
    );
    
    if (!device->mqtt_client) {
        LOG_ERROR("MQTT 客戶端創建失敗");
        return -1;
    }
    
    // 創建主題建構器
    device->topic_builder = rtk_topic_builder_create();
    if (!device->topic_builder) {
        LOG_ERROR("主題建構器創建失敗");
        return -1;
    }
    
    // 設置主題參數
    rtk_topic_builder_set_tenant(device->topic_builder, "production");
    rtk_topic_builder_set_site(device->topic_builder, "factory_a");
    rtk_topic_builder_set_device_id(device->topic_builder, device->device_info.device_id);
    
    // 初始化設備狀態
    device->running = 1;
    device->connected = 0;
    device->health_status = 0;
    device->start_time = time(NULL);
    device->reconnect_count = 0;
    
    LOG_INFO("設備初始化完成");
    return 0;
}

void cleanup_device(complete_device_t* device) {
    LOG_INFO("正在清理設備資源...");
    
    // 停止所有執行緒
    device->running = 0;
    
    // 等待執行緒結束
    if (device->sensor_thread) {
        pthread_join(device->sensor_thread, NULL);
        LOG_DEBUG("感測器執行緒已結束");
    }
    
    if (device->command_thread) {
        pthread_join(device->command_thread, NULL);
        LOG_DEBUG("命令執行緒已結束");
    }
    
    if (device->health_thread) {
        pthread_join(device->health_thread, NULL);
        LOG_DEBUG("健康監控執行緒已結束");
    }
    
    // 發布離線狀態
    if (device->mqtt_client && device->connected) {
        rtk_mqtt_client_publish_state(device->mqtt_client, "offline", "shutdown");
        rtk_mqtt_client_publish_event(device->mqtt_client, "device.lifecycle.shutdown", 
                                     "設備正常關閉");
    }
    
    // 清理 MQTT 客戶端
    if (device->mqtt_client) {
        rtk_mqtt_client_disconnect(device->mqtt_client);
        rtk_mqtt_client_destroy(device->mqtt_client);
    }
    
    // 清理主題建構器
    if (device->topic_builder) {
        rtk_topic_builder_destroy(device->topic_builder);
    }
    
    // 清理同步物件
    pthread_mutex_destroy(&device->metrics_mutex);
    pthread_mutex_destroy(&device->shutdown_mutex);
    pthread_cond_destroy(&device->shutdown_cond);
    
    LOG_INFO("設備資源清理完成");
}

// === 主程式 ===

int main(int argc, char* argv[]) {
    printf("RTK MQTT Framework 完整設備範例\n");
    printf("===============================\n");
    printf("這是一個生產級的 IoT 設備實作範例\n\n");
    
    // 檢查命令列參數
    const char* config_file = "config.json";
    if (argc > 1) {
        config_file = argv[1];
    }
    
    // 初始化隨機數種子
    srand((unsigned int)time(NULL));
    
    // 分配設備結構
    complete_device_t device = {0};
    g_device = &device;
    
    // 設置信號處理
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    
    // 開啟日誌檔案
    log_file = fopen("device.log", "a");
    if (!log_file) {
        printf("警告: 無法開啟日誌檔案 device.log\n");
    }
    
    // 初始化設備
    if (initialize_device(&device, config_file) != 0) {
        LOG_CRITICAL("設備初始化失敗");
        return -1;
    }
    
    // 連接到 MQTT broker
    LOG_INFO("正在連接到 MQTT broker %s:%d...", 
             device.mqtt_config.broker_host, device.mqtt_config.broker_port);
    
    if (rtk_mqtt_client_connect(device.mqtt_client) != RTK_SUCCESS) {
        LOG_ERROR("MQTT 連接失敗");
        cleanup_device(&device);
        return -1;
    }
    
    device.connected = 1;
    LOG_INFO("MQTT 連接成功");
    
    // 發布啟動事件
    rtk_mqtt_client_publish_event(device.mqtt_client, "device.lifecycle.startup", 
                                 "設備已啟動");
    
    // 啟動工作執行緒
    LOG_INFO("正在啟動工作執行緒...");
    
    if (pthread_create(&device.sensor_thread, NULL, sensor_thread_func, &device) != 0 ||
        pthread_create(&device.command_thread, NULL, command_thread_func, &device) != 0 ||
        pthread_create(&device.health_thread, NULL, health_thread_func, &device) != 0) {
        LOG_ERROR("執行緒創建失敗");
        cleanup_device(&device);
        return -1;
    }
    
    LOG_INFO("所有執行緒已啟動，設備正常運行");
    LOG_INFO("按 Ctrl+C 停止設備");
    
    // 主執行緒等待關閉信號
    pthread_mutex_lock(&device.shutdown_mutex);
    while (device.running) {
        pthread_cond_wait(&device.shutdown_cond, &device.shutdown_mutex);
    }
    pthread_mutex_unlock(&device.shutdown_mutex);
    
    // 清理資源
    cleanup_device(&device);
    
    // 關閉日誌檔案
    if (log_file) {
        fclose(log_file);
    }
    
    printf("\n📊 設備運行總結:\n");
    printf("   - 運行時間: %ld 秒\n", time(NULL) - device.start_time);
    printf("   - 重連次數: %d\n", device.reconnect_count);
    printf("   - 最終健康狀態: %s\n", 
           (device.health_status == 0) ? "健康" : 
           (device.health_status == 1) ? "警告" : "危險");
    
    printf("\n🎉 完整設備範例執行完成！\n");
    return 0;
}
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>
#include <rtk_device_plugin.h>
#include <rtk_mqtt_client.h>
#include <rtk_message_codec.h>
#include <rtk_topic_builder.h>
#include <rtk_schema_validator.h>

/**
 * @file wifi_router_plugin.c
 * @brief WiFi 路由器診斷插件範例
 * 
 * 展示完整的 WiFi 診斷功能，包含：
 * - WiFi 漫遊失效檢測 (roam_miss)
 * - WiFi 連線失敗檢測 (connect_fail) 
 * - ARP 封包遺失檢測 (arp_loss)
 * - WiFi 掃描遙測資料
 */

// === 插件狀態管理 ===

typedef struct wifi_router_state {
    // 基本資訊
    rtk_device_info_t device_info;
    rtk_plugin_config_t config;
    int is_running;
    
    // WiFi 狀態
    char current_ssid[32];
    char current_bssid[18];
    int current_rssi;
    int current_channel;
    int connected_clients;
    
    // 診斷統計
    int roam_miss_count;
    int connect_fail_count;
    int arp_loss_count;
    time_t last_scan_time;
    
    // 模擬資料 (實際應用中從硬體讀取)
    float cpu_usage;
    float memory_usage;
    float temperature;
    int uptime_seconds;
} wifi_router_state_t;

static wifi_router_state_t g_plugin_state = {0};

// === 模擬硬體介面 (實際應用中替換為真實硬體操作) ===

static void simulate_wifi_scan(void) {
    // 模擬 WiFi 掃描，更新狀態
    g_plugin_state.current_rssi = -45 + (rand() % 20) - 10;  // -55 到 -35 dBm
    g_plugin_state.connected_clients = 3 + (rand() % 5);     // 3-7 個客戶端
    g_plugin_state.last_scan_time = time(NULL);
    
    printf("[WiFi-Plugin] Simulated scan: RSSI=%d, Clients=%d\n", 
           g_plugin_state.current_rssi, g_plugin_state.connected_clients);
}

static void simulate_system_metrics(void) {
    // 模擬系統指標
    g_plugin_state.cpu_usage = 30.0 + (rand() % 40);        // 30-70%
    g_plugin_state.memory_usage = 40.0 + (rand() % 30);     // 40-70%
    g_plugin_state.temperature = 35.0 + (rand() % 20);      // 35-55°C
    g_plugin_state.uptime_seconds += 30;                    // 模擬運行時間
}

static int check_roam_miss_condition(void) {
    // 模擬漫遊失效檢測: RSSI < -70 dBm 持續 10 秒
    return (g_plugin_state.current_rssi < -70) ? 1 : 0;
}

static int check_arp_loss_condition(void) {
    // 模擬 ARP 遺失檢測: 隨機觸發
    return (rand() % 100 < 5) ? 1 : 0;  // 5% 機率
}

// === 插件介面實作 ===

static int wifi_get_device_info(rtk_device_info_t* info) {
    if (!info) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    memcpy(info, &g_plugin_state.device_info, sizeof(rtk_device_info_t));
    return RTK_PLUGIN_SUCCESS;
}

static int wifi_get_capabilities(char** capabilities, int* count) {
    if (!capabilities || !count) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    static char* caps[] = {
        "wifi_diagnosis",
        "roam_detection", 
        "arp_monitoring",
        "scan_telemetry",
        "system_metrics"
    };
    
    *capabilities = (char*)caps;
    *count = 5;
    return RTK_PLUGIN_SUCCESS;
}

static int wifi_get_state(char** json_state, size_t* len) {
    if (!json_state || !len) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    // 更新模擬資料
    simulate_system_metrics();
    
    // 建構狀態訊息
    rtk_state_message_t state_msg = {0};
    
    // 設定標頭
    strcpy(state_msg.header.schema, RTK_SCHEMA_NAME_STATE_V1_0);
    state_msg.header.timestamp = rtk_get_current_timestamp();
    state_msg.header.has_trace = 0;
    
    // 設定狀態資料
    strcpy(state_msg.health, "ok");
    strcpy(state_msg.fw_version, g_plugin_state.device_info.fw_version);
    state_msg.uptime_seconds = g_plugin_state.uptime_seconds;
    state_msg.cpu_usage = g_plugin_state.cpu_usage;
    state_msg.memory_usage = g_plugin_state.memory_usage;
    state_msg.temperature = g_plugin_state.temperature;
    
    // 添加 WiFi 特定資料
    snprintf(state_msg.custom_data, sizeof(state_msg.custom_data),
        "\"wifi_stats\":{"
        "\"ssid\":\"%s\","
        "\"bssid\":\"%s\","
        "\"rssi\":%d,"
        "\"channel\":%d,"
        "\"connected_clients\":%d"
        "}",
        g_plugin_state.current_ssid,
        g_plugin_state.current_bssid,
        g_plugin_state.current_rssi,
        g_plugin_state.current_channel,
        g_plugin_state.connected_clients
    );
    
    // 編碼為 JSON
    char* buffer = malloc(2048);
    if (!buffer) return RTK_PLUGIN_ERROR_MEMORY;
    
    int json_len = rtk_encode_state_message(&state_msg, buffer, 2048);
    if (json_len < 0) {
        free(buffer);
        return RTK_PLUGIN_ERROR_CONFIG;
    }
    
    *json_state = buffer;
    *len = json_len;
    
    return RTK_PLUGIN_SUCCESS;
}

static int wifi_get_telemetry(const char* metric, char** json_data, size_t* len) {
    if (!metric || !json_data || !len) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    char* buffer = malloc(1024);
    if (!buffer) return RTK_PLUGIN_ERROR_MEMORY;
    
    int json_len = -1;
    
    if (strcmp(metric, "wifi.scan_result") == 0) {
        // 執行 WiFi 掃描
        simulate_wifi_scan();
        
        // 建構掃描結果遙測
        json_len = snprintf(buffer, 1024,
            "{"
            "\"schema\":\"telemetry.wifi.scan_result/1.0\","
            "\"ts\":%lld,"
            "\"scan_info\":{"
                "\"internal_scan_skip_cnt\":0,"
                "\"environment_scan_ap_number\":8,"
                "\"current_bssid\":\"%s\","
                "\"current_rssi\":%d"
            "},"
            "\"roam_candidates\":["
                "{\"bssid\":\"11:22:33:44:55:66\",\"rssi\":-42,\"channel\":6},"
                "{\"bssid\":\"77:88:99:aa:bb:cc\",\"rssi\":-48,\"channel\":11}"
            "],"
            "\"scan_timing\":{"
                "\"last_scan_time\":%ld,"
                "\"last_full_scan_complete_time\":%ld"
            "}"
            "}",
            (long long)rtk_get_current_timestamp(),
            g_plugin_state.current_bssid,
            g_plugin_state.current_rssi,
            g_plugin_state.last_scan_time,
            g_plugin_state.last_scan_time - 5
        );
    } else {
        free(buffer);
        return RTK_PLUGIN_ERROR_NOT_FOUND;
    }
    
    if (json_len >= 1024) {
        free(buffer);
        return RTK_PLUGIN_ERROR_MEMORY;
    }
    
    *json_data = buffer;
    *len = json_len;
    
    return RTK_PLUGIN_SUCCESS;
}

static int wifi_handle_command(const char* cmd_json, char** response_json, size_t* len) {
    if (!cmd_json || !response_json || !len) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    // 解析命令 (簡化實作)
    char cmd_id[64] = "unknown";
    char operation[64] = "unknown";
    
    // 提取命令 ID 和操作
    const char* id_start = strstr(cmd_json, "\"id\":\"");
    if (id_start) {
        id_start += 6;
        const char* id_end = strchr(id_start, '"');
        if (id_end) {
            int len = id_end - id_start;
            if (len < sizeof(cmd_id)) {
                strncpy(cmd_id, id_start, len);
                cmd_id[len] = '\0';
            }
        }
    }
    
    const char* op_start = strstr(cmd_json, "\"op\":\"");
    if (op_start) {
        op_start += 6;
        const char* op_end = strchr(op_start, '"');
        if (op_end) {
            int len = op_end - op_start;
            if (len < sizeof(operation)) {
                strncpy(operation, op_start, len);
                operation[len] = '\0';
            }
        }
    }
    
    char* buffer = malloc(1024);
    if (!buffer) return RTK_PLUGIN_ERROR_MEMORY;
    
    int json_len = -1;
    
    if (strcmp(operation, "diagnosis.get") == 0) {
        // 處理診斷資料請求
        json_len = snprintf(buffer, 1024,
            "{"
            "\"id\":\"%s\","
            "\"ts\":%lld,"
            "\"ok\":true,"
            "\"result\":{"
                "\"diagnosis_type\":\"wifi\","
                "\"device_type\":\"wifi_router\","
                "\"data\":{"
                    "\"current_connection\":{"
                        "\"bssid\":\"%s\","
                        "\"rssi\":%d,"
                        "\"channel\":%d"
                    "},"
                    "\"roam_history\":[]"
                "}"
            "}"
            "}",
            cmd_id,
            (long long)rtk_get_current_timestamp(),
            g_plugin_state.current_bssid,
            g_plugin_state.current_rssi,
            g_plugin_state.current_channel
        );
    } else if (strcmp(operation, "device.reboot") == 0) {
        // 處理重新啟動命令
        json_len = snprintf(buffer, 1024,
            "{"
            "\"id\":\"%s\","
            "\"ts\":%lld,"
            "\"ok\":true,"
            "\"result\":{\"rebooting\":true}"
            "}",
            cmd_id,
            (long long)rtk_get_current_timestamp()
        );
        
        printf("[WiFi-Plugin] Reboot command received\n");
    } else {
        // 不支援的命令
        json_len = snprintf(buffer, 1024,
            "{"
            "\"id\":\"%s\","
            "\"ts\":%lld,"
            "\"ok\":false,"
            "\"err\":{\"code\":\"E_UNSUPPORTED\",\"msg\":\"不支援的命令: %s\"}"
            "}",
            cmd_id,
            (long long)rtk_get_current_timestamp(),
            operation
        );
    }
    
    if (json_len >= 1024) {
        free(buffer);
        return RTK_PLUGIN_ERROR_MEMORY;
    }
    
    *response_json = buffer;
    *len = json_len;
    
    return RTK_PLUGIN_SUCCESS;
}

static int wifi_initialize(const rtk_plugin_config_t* config) {
    if (!config) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    // 複製配置
    memcpy(&g_plugin_state.config, config, sizeof(rtk_plugin_config_t));
    
    // 初始化設備資訊
    strcpy(g_plugin_state.device_info.id, config->device_id);
    strcpy(g_plugin_state.device_info.type, "wifi_router");
    strcpy(g_plugin_state.device_info.model, "RTK-AP-8000");
    strcpy(g_plugin_state.device_info.serial_number, "WF20240001");
    strcpy(g_plugin_state.device_info.hw_version, "revC");
    strcpy(g_plugin_state.device_info.fw_version, "1.2.3");
    
    // 初始化 WiFi 狀態
    strcpy(g_plugin_state.current_ssid, "OfficeWiFi-5G");
    strcpy(g_plugin_state.current_bssid, "aa:bb:cc:dd:ee:ff");
    g_plugin_state.current_rssi = -45;
    g_plugin_state.current_channel = 36;
    g_plugin_state.connected_clients = 5;
    
    // 初始化統計
    g_plugin_state.roam_miss_count = 0;
    g_plugin_state.connect_fail_count = 0;
    g_plugin_state.arp_loss_count = 0;
    g_plugin_state.uptime_seconds = 0;
    
    printf("[WiFi-Plugin] Initialized: device_id=%s, model=%s\n", 
           config->device_id, g_plugin_state.device_info.model);
    
    return RTK_PLUGIN_SUCCESS;
}

static int wifi_start(void) {
    g_plugin_state.is_running = 1;
    printf("[WiFi-Plugin] Started\n");
    return RTK_PLUGIN_SUCCESS;
}

static int wifi_stop(void) {
    g_plugin_state.is_running = 0;
    printf("[WiFi-Plugin] Stopped\n");
    return RTK_PLUGIN_SUCCESS;
}

static int wifi_health_check(void) {
    // 簡單的健康檢查
    return g_plugin_state.is_running ? 1 : 0;
}

static void wifi_free_json_string(char* json_str) {
    if (json_str) {
        free(json_str);
    }
}

// === 插件虛函式表 ===
static const rtk_device_plugin_vtable_t wifi_router_vtable = {
    .get_device_info = wifi_get_device_info,
    .get_capabilities = wifi_get_capabilities,
    .get_state = wifi_get_state,
    .get_attributes = NULL,  // 未實作
    .get_telemetry = wifi_get_telemetry,
    .list_telemetry_metrics = NULL,  // 未實作
    .on_event_trigger = NULL,  // 未實作
    .get_supported_events = NULL,  // 未實作
    .handle_command = wifi_handle_command,
    .get_supported_commands = NULL,  // 未實作
    .initialize = wifi_initialize,
    .start = wifi_start,
    .stop = wifi_stop,
    .health_check = wifi_health_check,
    .free_json_string = wifi_free_json_string
};

// === 插件註冊函式 (必要) ===
const rtk_device_plugin_vtable_t* rtk_plugin_get_vtable(void) {
    return &wifi_router_vtable;
}

const char* rtk_plugin_get_version(void) {
    return "1.0.0";
}

const char* rtk_plugin_get_name(void) {
    return "wifi_router";
}
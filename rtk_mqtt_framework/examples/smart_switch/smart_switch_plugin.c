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
 * @file smart_switch_plugin.c
 * @brief 智能開關插件範例
 * 
 * 展示智能電源控制功能，包含：
 * - 多通道電源控制 (4路繼電器)
 * - 電流/功率監測
 * - 定時開關功能
 * - 過載保護事件 (overload_protection)
 * - 遠端控制命令 (switch.set, schedule.set)
 */

// === 智能開關狀態管理 ===

typedef struct switch_channel {
    int channel_id;                // 通道編號 (1-4)
    int is_on;                    // 開關狀態
    float current_amps;           // 當前電流 (A)
    float power_watts;            // 當前功率 (W)
    float voltage;                // 電壓 (V)
    int overload_count;           // 過載次數
    time_t last_switch_time;      // 最後切換時間
    
    // 定時設定
    struct {
        int enabled;
        int hour_on;
        int minute_on;
        int hour_off;
        int minute_off;
        int repeat_days;          // 位元遮罩 (bit 0-6 = 日-六)
    } schedule;
} switch_channel_t;

typedef struct smart_switch_state {
    // 基本資訊
    rtk_device_info_t device_info;
    rtk_plugin_config_t config;
    int is_running;
    
    // 開關通道狀態
    switch_channel_t channels[4];
    int channel_count;
    
    // 整體統計
    float total_power;            // 總功率
    float total_current;          // 總電流
    int total_switch_operations;  // 總開關次數
    int overload_events;          // 過載事件次數
    
    // 保護設定
    struct {
        float max_current_per_channel;  // 單通道最大電流
        float max_total_current;        // 總最大電流
        int overload_protection_enabled;
    } protection;
    
    // 系統狀態
    float temperature;            // 內部溫度
    float cpu_usage;
    int uptime_seconds;
} smart_switch_state_t;

static smart_switch_state_t g_switch_state = {0};

// === 模擬電氣控制 (實際應用中控制真實硬體) ===

static void simulate_electrical_measurements(void) {
    g_switch_state.total_current = 0.0;
    g_switch_state.total_power = 0.0;
    
    for (int i = 0; i < g_switch_state.channel_count; i++) {
        switch_channel_t* ch = &g_switch_state.channels[i];
        
        if (ch->is_on) {
            // 模擬負載電流 (0.5-8A)
            ch->current_amps = 0.5 + (rand() % 750) / 100.0;
            ch->voltage = 220.0 + (rand() % 20 - 10);  // 210-230V
            ch->power_watts = ch->current_amps * ch->voltage;
            
            // 檢查過載
            if (ch->current_amps > g_switch_state.protection.max_current_per_channel) {
                ch->overload_count++;
                g_switch_state.overload_events++;
                printf("[Smart-Switch] Channel %d overload detected: %.2fA\n", 
                       ch->channel_id, ch->current_amps);
                
                if (g_switch_state.protection.overload_protection_enabled) {
                    ch->is_on = 0;  // 自動斷開
                    ch->last_switch_time = time(NULL);
                    printf("[Smart-Switch] Channel %d automatically turned off due to overload\n", 
                           ch->channel_id);
                }
            }
        } else {
            ch->current_amps = 0.0;
            ch->power_watts = 0.0;
            ch->voltage = 0.0;
        }
        
        g_switch_state.total_current += ch->current_amps;
        g_switch_state.total_power += ch->power_watts;
    }
    
    // 總功率過載檢查
    if (g_switch_state.total_current > g_switch_state.protection.max_total_current) {
        printf("[Smart-Switch] Total current overload: %.2fA\n", g_switch_state.total_current);
        g_switch_state.overload_events++;
    }
    
    // 模擬內部溫度 (與總功率相關)
    g_switch_state.temperature = 25.0 + (g_switch_state.total_power / 200.0) + 
                                 (rand() % 50 - 25) / 10.0;
}

static void check_scheduled_operations(void) {
    time_t now = time(NULL);
    struct tm* tm_info = localtime(&now);
    int current_hour = tm_info->tm_hour;
    int current_minute = tm_info->tm_min;
    int current_day = tm_info->tm_wday;  // 0 = 日, 1 = 一, ...
    
    for (int i = 0; i < g_switch_state.channel_count; i++) {
        switch_channel_t* ch = &g_switch_state.channels[i];
        
        if (!ch->schedule.enabled) continue;
        
        // 檢查是否在重複日程內
        if (!(ch->schedule.repeat_days & (1 << current_day))) continue;
        
        // 檢查開啟時間
        if (current_hour == ch->schedule.hour_on && 
            current_minute == ch->schedule.minute_on && 
            !ch->is_on) {
            ch->is_on = 1;
            ch->last_switch_time = now;
            g_switch_state.total_switch_operations++;
            printf("[Smart-Switch] Channel %d turned ON by schedule\n", ch->channel_id);
        }
        
        // 檢查關閉時間
        if (current_hour == ch->schedule.hour_off && 
            current_minute == ch->schedule.minute_off && 
            ch->is_on) {
            ch->is_on = 0;
            ch->last_switch_time = now;
            g_switch_state.total_switch_operations++;
            printf("[Smart-Switch] Channel %d turned OFF by schedule\n", ch->channel_id);
        }
    }
}

static void simulate_system_metrics(void) {
    g_switch_state.cpu_usage = 8.0 + (rand() % 15);  // 8-23%
    g_switch_state.uptime_seconds += 30;
}

// === 插件介面實作 ===

static int switch_get_device_info(rtk_device_info_t* info) {
    if (!info) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    memcpy(info, &g_switch_state.device_info, sizeof(rtk_device_info_t));
    return RTK_PLUGIN_SUCCESS;
}

static int switch_get_capabilities(char** capabilities, int* count) {
    if (!capabilities || !count) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    static char* caps[] = {
        "multi_channel_control",
        "power_monitoring",
        "overload_protection",
        "scheduled_operations",
        "remote_control"
    };
    
    *capabilities = (char*)caps;
    *count = 5;
    return RTK_PLUGIN_SUCCESS;
}

static int switch_get_state(char** json_state, size_t* len) {
    if (!json_state || !len) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    // 更新模擬資料
    simulate_electrical_measurements();
    check_scheduled_operations();
    simulate_system_metrics();
    
    // 建構狀態訊息
    rtk_state_message_t state_msg = {0};
    
    // 設定標頭
    strcpy(state_msg.header.schema, RTK_SCHEMA_NAME_STATE_V1_0);
    state_msg.header.timestamp = rtk_get_current_timestamp();
    state_msg.header.has_trace = 0;
    
    // 設定狀態資料
    strcpy(state_msg.health, "ok");
    strcpy(state_msg.fw_version, g_switch_state.device_info.fw_version);
    state_msg.uptime_seconds = g_switch_state.uptime_seconds;
    state_msg.cpu_usage = g_switch_state.cpu_usage;
    state_msg.memory_usage = 30.0 + (rand() % 20);  // 模擬記憶體使用
    state_msg.temperature = g_switch_state.temperature;
    
    // 添加開關特定資料
    char channels_json[512] = "";
    for (int i = 0; i < g_switch_state.channel_count; i++) {
        switch_channel_t* ch = &g_switch_state.channels[i];
        char ch_json[128];
        snprintf(ch_json, sizeof(ch_json),
            "%s{\"id\":%d,\"on\":%s,\"current\":%.2f,\"power\":%.1f,\"overloads\":%d}",
            (i > 0) ? "," : "",
            ch->channel_id,
            ch->is_on ? "true" : "false",
            ch->current_amps,
            ch->power_watts,
            ch->overload_count
        );
        strncat(channels_json, ch_json, sizeof(channels_json) - strlen(channels_json) - 1);
    }
    
    snprintf(state_msg.custom_data, sizeof(state_msg.custom_data),
        "\"channels\":[%s],"
        "\"power_stats\":{"
        "\"total_power\":%.1f,"
        "\"total_current\":%.2f,"
        "\"switch_operations\":%d"
        "},"
        "\"protection\":{"
        "\"enabled\":%s,"
        "\"overload_events\":%d,"
        "\"max_current_per_channel\":%.1f,"
        "\"max_total_current\":%.1f"
        "}",
        channels_json,
        g_switch_state.total_power,
        g_switch_state.total_current,
        g_switch_state.total_switch_operations,
        g_switch_state.protection.overload_protection_enabled ? "true" : "false",
        g_switch_state.overload_events,
        g_switch_state.protection.max_current_per_channel,
        g_switch_state.protection.max_total_current
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

static int switch_get_telemetry(const char* metric, char** json_data, size_t* len) {
    if (!metric || !json_data || !len) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    char* buffer = malloc(1024);
    if (!buffer) return RTK_PLUGIN_ERROR_MEMORY;
    
    int json_len = -1;
    
    if (strcmp(metric, "power.current") == 0) {
        // 當前功率資料
        simulate_electrical_measurements();
        
        json_len = snprintf(buffer, 1024,
            "{"
            "\"schema\":\"telemetry.power.current/1.0\","
            "\"ts\":%lld,"
            "\"measurements\":{"
                "\"total_power\":%.2f,"
                "\"total_current\":%.3f,"
                "\"efficiency\":%.1f,"
                "\"power_factor\":%.2f"
            "},"
            "\"channels\":[",
            (long long)rtk_get_current_timestamp(),
            g_switch_state.total_power, g_switch_state.total_current, 95.5, 0.98
        );
        
        // 添加各通道資料
        for (int i = 0; i < g_switch_state.channel_count; i++) {
            switch_channel_t* ch = &g_switch_state.channels[i];
            char ch_data[128];
            snprintf(ch_data, sizeof(ch_data),
                "%s{\"id\":%d,\"power\":%.2f,\"current\":%.3f,\"voltage\":%.1f}",
                (i > 0) ? "," : "",
                ch->channel_id,
                ch->power_watts,
                ch->current_amps,
                ch->voltage
            );
            
            if (strlen(buffer) + strlen(ch_data) + 10 < 1024) {
                strcat(buffer, ch_data);
            }
        }
        
        strcat(buffer, "]}");
        json_len = strlen(buffer);
        
    } else if (strcmp(metric, "switch.operations") == 0) {
        // 開關操作歷史
        json_len = snprintf(buffer, 1024,
            "{"
            "\"schema\":\"telemetry.switch.operations/1.0\","
            "\"ts\":%lld,"
            "\"statistics\":{"
                "\"total_operations\":%d,"
                "\"overload_events\":%d,"
                "\"protection_activations\":%d"
            "},"
            "\"channel_status\":[",
            (long long)rtk_get_current_timestamp(),
            g_switch_state.total_switch_operations,
            g_switch_state.overload_events,
            0  // protection_activations placeholder
        );
        
        for (int i = 0; i < g_switch_state.channel_count; i++) {
            switch_channel_t* ch = &g_switch_state.channels[i];
            char ch_status[128];
            snprintf(ch_status, sizeof(ch_status),
                "%s{\"id\":%d,\"state\":\"%s\",\"last_switch\":%ld,\"overloads\":%d}",
                (i > 0) ? "," : "",
                ch->channel_id,
                ch->is_on ? "on" : "off",
                ch->last_switch_time,
                ch->overload_count
            );
            
            if (strlen(buffer) + strlen(ch_status) + 10 < 1024) {
                strcat(buffer, ch_status);
            }
        }
        
        strcat(buffer, "]}");
        json_len = strlen(buffer);
        
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

static int switch_handle_command(const char* cmd_json, char** response_json, size_t* len) {
    if (!cmd_json || !response_json || !len) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    // 解析命令 (簡化實作)
    char cmd_id[64] = "unknown";
    char operation[64] = "unknown";
    
    // 提取命令 ID
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
    
    // 提取操作
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
    
    if (strcmp(operation, "switch.set") == 0) {
        // 處理開關控制命令
        
        // 提取通道和狀態參數
        int channel_id = 1;
        int new_state = 0;
        
        const char* channel_str = strstr(cmd_json, "\"channel\":");
        if (channel_str) {
            channel_id = atoi(channel_str + 10);
        }
        
        const char* state_str = strstr(cmd_json, "\"state\":");
        if (state_str) {
            state_str += 8;
            if (strncmp(state_str, "true", 4) == 0 || strncmp(state_str, "\"on\"", 4) == 0) {
                new_state = 1;
            }
        }
        
        // 驗證通道編號
        if (channel_id < 1 || channel_id > g_switch_state.channel_count) {
            json_len = snprintf(buffer, 1024,
                "{"
                "\"id\":\"%s\","
                "\"ts\":%lld,"
                "\"ok\":false,"
                "\"err\":{\"code\":\"E_INVALID_CHANNEL\",\"msg\":\"無效的通道編號: %d\"}"
                "}",
                cmd_id,
                (long long)rtk_get_current_timestamp(),
                channel_id
            );
        } else {
            // 執行開關操作
            switch_channel_t* ch = &g_switch_state.channels[channel_id - 1];
            int prev_state = ch->is_on;
            ch->is_on = new_state;
            ch->last_switch_time = time(NULL);
            
            if (prev_state != new_state) {
                g_switch_state.total_switch_operations++;
            }
            
            json_len = snprintf(buffer, 1024,
                "{"
                "\"id\":\"%s\","
                "\"ts\":%lld,"
                "\"ok\":true,"
                "\"result\":{"
                    "\"channel\":%d,"
                    "\"previous_state\":\"%s\","
                    "\"new_state\":\"%s\","
                    "\"switch_time\":%ld"
                "}"
                "}",
                cmd_id,
                (long long)rtk_get_current_timestamp(),
                channel_id,
                prev_state ? "on" : "off",
                new_state ? "on" : "off",
                ch->last_switch_time
            );
            
            printf("[Smart-Switch] Channel %d switched %s by remote command\n", 
                   channel_id, new_state ? "ON" : "OFF");
        }
        
    } else if (strcmp(operation, "schedule.set") == 0) {
        // 處理定時設定命令
        
        int channel_id = 1;
        const char* channel_str = strstr(cmd_json, "\"channel\":");
        if (channel_str) {
            channel_id = atoi(channel_str + 10);
        }
        
        if (channel_id < 1 || channel_id > g_switch_state.channel_count) {
            json_len = snprintf(buffer, 1024,
                "{"
                "\"id\":\"%s\","
                "\"ts\":%lld,"
                "\"ok\":false,"
                "\"err\":{\"code\":\"E_INVALID_CHANNEL\",\"msg\":\"無效的通道編號: %d\"}"
                "}",
                cmd_id,
                (long long)rtk_get_current_timestamp(),
                channel_id
            );
        } else {
            switch_channel_t* ch = &g_switch_state.channels[channel_id - 1];
            
            // 解析定時參數 (簡化實作)
            const char* enabled_str = strstr(cmd_json, "\"enabled\":");
            if (enabled_str) {
                ch->schedule.enabled = (strstr(enabled_str + 10, "true") != NULL) ? 1 : 0;
            }
            
            const char* hour_on_str = strstr(cmd_json, "\"hour_on\":");
            if (hour_on_str) {
                ch->schedule.hour_on = atoi(hour_on_str + 10);
            }
            
            const char* minute_on_str = strstr(cmd_json, "\"minute_on\":");
            if (minute_on_str) {
                ch->schedule.minute_on = atoi(minute_on_str + 12);
            }
            
            json_len = snprintf(buffer, 1024,
                "{"
                "\"id\":\"%s\","
                "\"ts\":%lld,"
                "\"ok\":true,"
                "\"result\":{"
                    "\"channel\":%d,"
                    "\"schedule_updated\":true,"
                    "\"enabled\":%s"
                "}"
                "}",
                cmd_id,
                (long long)rtk_get_current_timestamp(),
                channel_id,
                ch->schedule.enabled ? "true" : "false"
            );
            
            printf("[Smart-Switch] Schedule updated for channel %d: %s\n", 
                   channel_id, ch->schedule.enabled ? "enabled" : "disabled");
        }
        
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

static int switch_initialize(const rtk_plugin_config_t* config) {
    if (!config) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    // 複製配置
    memcpy(&g_switch_state.config, config, sizeof(rtk_plugin_config_t));
    
    // 初始化設備資訊
    strcpy(g_switch_state.device_info.id, config->device_id);
    strcpy(g_switch_state.device_info.type, "smart_switch");
    strcpy(g_switch_state.device_info.model, "RTK-SWITCH-4CH");
    strcpy(g_switch_state.device_info.serial_number, "SW20240001");
    strcpy(g_switch_state.device_info.hw_version, "revA");
    strcpy(g_switch_state.device_info.fw_version, "1.3.2");
    
    // 初始化通道
    g_switch_state.channel_count = 4;
    for (int i = 0; i < g_switch_state.channel_count; i++) {
        switch_channel_t* ch = &g_switch_state.channels[i];
        ch->channel_id = i + 1;
        ch->is_on = 0;
        ch->current_amps = 0.0;
        ch->power_watts = 0.0;
        ch->voltage = 0.0;
        ch->overload_count = 0;
        ch->last_switch_time = 0;
        
        // 預設定時設定 (關閉)
        ch->schedule.enabled = 0;
        ch->schedule.hour_on = 8;
        ch->schedule.minute_on = 0;
        ch->schedule.hour_off = 18;
        ch->schedule.minute_off = 0;
        ch->schedule.repeat_days = 0x7F;  // 每天
    }
    
    // 初始化保護設定
    g_switch_state.protection.max_current_per_channel = 10.0;  // 10A
    g_switch_state.protection.max_total_current = 30.0;        // 30A
    g_switch_state.protection.overload_protection_enabled = 1;
    
    // 初始化統計
    g_switch_state.total_switch_operations = 0;
    g_switch_state.overload_events = 0;
    g_switch_state.uptime_seconds = 0;
    
    printf("[Smart-Switch] Initialized: device_id=%s, channels=%d\n", 
           config->device_id, g_switch_state.channel_count);
    
    return RTK_PLUGIN_SUCCESS;
}

static int switch_start(void) {
    g_switch_state.is_running = 1;
    printf("[Smart-Switch] Started - %d channel control active\n", g_switch_state.channel_count);
    return RTK_PLUGIN_SUCCESS;
}

static int switch_stop(void) {
    g_switch_state.is_running = 0;
    
    // 安全關閉所有通道
    for (int i = 0; i < g_switch_state.channel_count; i++) {
        g_switch_state.channels[i].is_on = 0;
    }
    
    printf("[Smart-Switch] Stopped - All channels turned off\n");
    return RTK_PLUGIN_SUCCESS;
}

static int switch_health_check(void) {
    if (!g_switch_state.is_running) {
        return 0;
    }
    
    // 檢查溫度是否過高
    if (g_switch_state.temperature > 80.0) {
        return 0;  // 過熱
    }
    
    // 檢查是否有太多過載事件
    if (g_switch_state.overload_events > 10) {
        return 0;  // 頻繁過載
    }
    
    return 1;  // 健康
}

static void switch_free_json_string(char* json_str) {
    if (json_str) {
        free(json_str);
    }
}

// === 插件虛函式表 ===
static const rtk_device_plugin_vtable_t smart_switch_vtable = {
    .get_device_info = switch_get_device_info,
    .get_capabilities = switch_get_capabilities,
    .get_state = switch_get_state,
    .get_attributes = NULL,  // 未實作
    .get_telemetry = switch_get_telemetry,
    .list_telemetry_metrics = NULL,  // 未實作
    .on_event_trigger = NULL,  // 未實作
    .get_supported_events = NULL,  // 未實作
    .handle_command = switch_handle_command,
    .get_supported_commands = NULL,  // 未實作
    .initialize = switch_initialize,
    .start = switch_start,
    .stop = switch_stop,
    .health_check = switch_health_check,
    .free_json_string = switch_free_json_string
};

// === 插件註冊函式 (必要) ===
const rtk_device_plugin_vtable_t* rtk_plugin_get_vtable(void) {
    return &smart_switch_vtable;
}

const char* rtk_plugin_get_version(void) {
    return "1.0.0";
}

const char* rtk_plugin_get_name(void) {
    return "smart_switch";
}
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>
#include <math.h>
#include <rtk_device_plugin.h>
#include <rtk_mqtt_client.h>
#include <rtk_message_codec.h>
#include <rtk_topic_builder.h>
#include <rtk_schema_validator.h>

/**
 * @file iot_sensor_plugin.c
 * @brief IoT 多功能感測器插件範例
 * 
 * 展示環境感測器功能，包含：
 * - 溫濕度感測 (temperature, humidity)
 * - 空氣品質監測 (PM2.5, CO2)
 * - 光照感測 (luminance)
 * - 動作偵測事件 (motion_detected)
 * - 感測器校準命令 (sensor.calibrate)
 */

// === 感測器狀態管理 ===

typedef struct sensor_readings {
    float temperature;          // 溫度 (°C)
    float humidity;            // 濕度 (%)
    int pm25;                  // PM2.5 (μg/m³)
    int co2;                   // CO2 (ppm)
    int luminance;             // 光照強度 (lux)
    int motion_detected;       // 動作偵測
    time_t last_motion_time;   // 最後動作時間
} sensor_readings_t;

typedef struct iot_sensor_state {
    // 基本資訊
    rtk_device_info_t device_info;
    rtk_plugin_config_t config;
    int is_running;
    
    // 感測器讀數
    sensor_readings_t current_readings;
    sensor_readings_t previous_readings;
    
    // 校準資料
    struct {
        float temp_offset;
        float humidity_offset;
        int is_calibrated;
        time_t last_calibration_time;
    } calibration;
    
    // 事件觸發統計
    int motion_event_count;
    int air_quality_alert_count;
    
    // 系統狀態
    float cpu_usage;
    float memory_usage;
    int uptime_seconds;
} iot_sensor_state_t;

static iot_sensor_state_t g_sensor_state = {0};

// === 模擬感測器讀取 (實際應用中從硬體讀取) ===

static void simulate_environmental_sensors(void) {
    // 模擬溫度感測器 (20-35°C，帶些許變化)
    static float base_temp = 25.0;
    base_temp += (rand() % 100 - 50) / 100.0;  // ±0.5°C 變化
    if (base_temp < 18.0) base_temp = 18.0;
    if (base_temp > 40.0) base_temp = 40.0;
    g_sensor_state.current_readings.temperature = base_temp + g_sensor_state.calibration.temp_offset;
    
    // 模擬濕度感測器 (30-80%)
    static float base_humidity = 55.0;
    base_humidity += (rand() % 100 - 50) / 50.0;  // ±1% 變化
    if (base_humidity < 20.0) base_humidity = 20.0;
    if (base_humidity > 90.0) base_humidity = 90.0;
    g_sensor_state.current_readings.humidity = base_humidity + g_sensor_state.calibration.humidity_offset;
    
    // 模擬 PM2.5 感測器 (5-150 μg/m³)
    static int base_pm25 = 25;
    base_pm25 += (rand() % 20 - 10);
    if (base_pm25 < 0) base_pm25 = 0;
    if (base_pm25 > 200) base_pm25 = 200;
    g_sensor_state.current_readings.pm25 = base_pm25;
    
    // 模擬 CO2 感測器 (400-2000 ppm)
    static int base_co2 = 800;
    base_co2 += (rand() % 100 - 50);
    if (base_co2 < 350) base_co2 = 350;
    if (base_co2 > 3000) base_co2 = 3000;
    g_sensor_state.current_readings.co2 = base_co2;
    
    // 模擬光照感測器 (0-1000 lux)
    time_t now = time(NULL);
    struct tm* tm_info = localtime(&now);
    int hour = tm_info->tm_hour;
    
    // 根據時間模擬光照變化
    if (hour >= 6 && hour <= 18) {
        // 白天光照
        g_sensor_state.current_readings.luminance = 200 + (rand() % 300);
    } else {
        // 夜間光照
        g_sensor_state.current_readings.luminance = rand() % 50;
    }
    
    // 模擬動作偵測 (5% 機率)
    int prev_motion = g_sensor_state.current_readings.motion_detected;
    g_sensor_state.current_readings.motion_detected = (rand() % 100 < 5) ? 1 : 0;
    
    if (!prev_motion && g_sensor_state.current_readings.motion_detected) {
        g_sensor_state.current_readings.last_motion_time = now;
        g_sensor_state.motion_event_count++;
    }
}

static void simulate_system_metrics(void) {
    // 模擬系統指標
    g_sensor_state.cpu_usage = 15.0 + (rand() % 25);        // 15-40%
    g_sensor_state.memory_usage = 25.0 + (rand() % 35);     // 25-60%
    g_sensor_state.uptime_seconds += 30;                    // 模擬運行時間
}

static int check_air_quality_alert(void) {
    // 空氣品質警報條件: PM2.5 > 75 或 CO2 > 1500
    return (g_sensor_state.current_readings.pm25 > 75 || 
            g_sensor_state.current_readings.co2 > 1500) ? 1 : 0;
}

// === 插件介面實作 ===

static int sensor_get_device_info(rtk_device_info_t* info) {
    if (!info) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    memcpy(info, &g_sensor_state.device_info, sizeof(rtk_device_info_t));
    return RTK_PLUGIN_SUCCESS;
}

static int sensor_get_capabilities(char** capabilities, int* count) {
    if (!capabilities || !count) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    static char* caps[] = {
        "environmental_monitoring",
        "air_quality_detection",
        "motion_detection",
        "sensor_calibration",
        "multi_sensor_fusion"
    };
    
    *capabilities = (char*)caps;
    *count = 5;
    return RTK_PLUGIN_SUCCESS;
}

static int sensor_get_state(char** json_state, size_t* len) {
    if (!json_state || !len) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    // 更新感測器讀數
    simulate_environmental_sensors();
    simulate_system_metrics();
    
    // 建構狀態訊息
    rtk_state_message_t state_msg = {0};
    
    // 設定標頭
    strcpy(state_msg.header.schema, RTK_SCHEMA_NAME_STATE_V1_0);
    state_msg.header.timestamp = rtk_get_current_timestamp();
    state_msg.header.has_trace = 0;
    
    // 設定狀態資料
    strcpy(state_msg.health, "ok");
    strcpy(state_msg.fw_version, g_sensor_state.device_info.fw_version);
    state_msg.uptime_seconds = g_sensor_state.uptime_seconds;
    state_msg.cpu_usage = g_sensor_state.cpu_usage;
    state_msg.memory_usage = g_sensor_state.memory_usage;
    state_msg.temperature = g_sensor_state.current_readings.temperature;
    
    // 添加感測器特定資料
    snprintf(state_msg.custom_data, sizeof(state_msg.custom_data),
        "\"sensor_readings\":{"
        "\"temperature\":%.1f,"
        "\"humidity\":%.1f,"
        "\"pm25\":%d,"
        "\"co2\":%d,"
        "\"luminance\":%d,"
        "\"motion_detected\":%s"
        "},"
        "\"calibration\":{"
        "\"is_calibrated\":%s,"
        "\"last_calibration\":%ld"
        "},"
        "\"statistics\":{"
        "\"motion_events\":%d,"
        "\"air_quality_alerts\":%d"
        "}",
        g_sensor_state.current_readings.temperature,
        g_sensor_state.current_readings.humidity,
        g_sensor_state.current_readings.pm25,
        g_sensor_state.current_readings.co2,
        g_sensor_state.current_readings.luminance,
        g_sensor_state.current_readings.motion_detected ? "true" : "false",
        g_sensor_state.calibration.is_calibrated ? "true" : "false",
        g_sensor_state.calibration.last_calibration_time,
        g_sensor_state.motion_event_count,
        g_sensor_state.air_quality_alert_count
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

static int sensor_get_telemetry(const char* metric, char** json_data, size_t* len) {
    if (!metric || !json_data || !len) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    char* buffer = malloc(1024);
    if (!buffer) return RTK_PLUGIN_ERROR_MEMORY;
    
    int json_len = -1;
    
    if (strcmp(metric, "environmental.current") == 0) {
        // 當前環境資料
        simulate_environmental_sensors();
        
        json_len = snprintf(buffer, 1024,
            "{"
            "\"schema\":\"telemetry.environmental.current/1.0\","
            "\"ts\":%lld,"
            "\"readings\":{"
                "\"temperature\":%.2f,"
                "\"humidity\":%.2f,"
                "\"pm25\":%d,"
                "\"co2\":%d,"
                "\"luminance\":%d,"
                "\"motion_active\":%s"
            "},"
            "\"quality\":{"
                "\"air_quality_index\":%.1f,"
                "\"comfort_level\":\"%s\""
            "}"
            "}",
            (long long)rtk_get_current_timestamp(),
            g_sensor_state.current_readings.temperature,
            g_sensor_state.current_readings.humidity,
            g_sensor_state.current_readings.pm25,
            g_sensor_state.current_readings.co2,
            g_sensor_state.current_readings.luminance,
            g_sensor_state.current_readings.motion_detected ? "true" : "false",
            // 簡化的空氣品質指數計算
            (g_sensor_state.current_readings.pm25 * 2.0 + g_sensor_state.current_readings.co2 / 20.0),
            (g_sensor_state.current_readings.temperature >= 20.0 && g_sensor_state.current_readings.temperature <= 26.0 &&
             g_sensor_state.current_readings.humidity >= 40.0 && g_sensor_state.current_readings.humidity <= 70.0) ? "comfortable" : "suboptimal"
        );
        
    } else if (strcmp(metric, "motion.history") == 0) {
        // 動作偵測歷史
        json_len = snprintf(buffer, 1024,
            "{"
            "\"schema\":\"telemetry.motion.history/1.0\","
            "\"ts\":%lld,"
            "\"motion_stats\":{"
                "\"total_events\":%d,"
                "\"last_motion_time\":%ld,"
                "\"current_status\":\"%s\""
            "}"
            "}",
            (long long)rtk_get_current_timestamp(),
            g_sensor_state.motion_event_count,
            g_sensor_state.current_readings.last_motion_time,
            g_sensor_state.current_readings.motion_detected ? "motion_detected" : "no_motion"
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

static int sensor_handle_command(const char* cmd_json, char** response_json, size_t* len) {
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
    
    if (strcmp(operation, "sensor.calibrate") == 0) {
        // 處理感測器校準命令
        
        // 提取校準參數 (簡化實作)
        float temp_offset = 0.0;
        float humidity_offset = 0.0;
        
        const char* temp_offset_str = strstr(cmd_json, "\"temp_offset\":");
        if (temp_offset_str) {
            temp_offset = atof(temp_offset_str + 14);
        }
        
        const char* humidity_offset_str = strstr(cmd_json, "\"humidity_offset\":");
        if (humidity_offset_str) {
            humidity_offset = atof(humidity_offset_str + 18);
        }
        
        // 應用校準
        g_sensor_state.calibration.temp_offset = temp_offset;
        g_sensor_state.calibration.humidity_offset = humidity_offset;
        g_sensor_state.calibration.is_calibrated = 1;
        g_sensor_state.calibration.last_calibration_time = time(NULL);
        
        json_len = snprintf(buffer, 1024,
            "{"
            "\"id\":\"%s\","
            "\"ts\":%lld,"
            "\"ok\":true,"
            "\"result\":{"
                "\"calibration_applied\":true,"
                "\"temp_offset\":%.2f,"
                "\"humidity_offset\":%.2f,"
                "\"calibration_time\":%ld"
            "}"
            "}",
            cmd_id,
            (long long)rtk_get_current_timestamp(),
            temp_offset,
            humidity_offset,
            g_sensor_state.calibration.last_calibration_time
        );
        
        printf("[IoT-Sensor] Calibration applied: temp_offset=%.2f, humidity_offset=%.2f\n",
               temp_offset, humidity_offset);
               
    } else if (strcmp(operation, "readings.get") == 0) {
        // 處理即時讀數請求
        simulate_environmental_sensors();
        
        json_len = snprintf(buffer, 1024,
            "{"
            "\"id\":\"%s\","
            "\"ts\":%lld,"
            "\"ok\":true,"
            "\"result\":{"
                "\"current_readings\":{"
                    "\"temperature\":%.2f,"
                    "\"humidity\":%.2f,"
                    "\"pm25\":%d,"
                    "\"co2\":%d,"
                    "\"luminance\":%d,"
                    "\"motion_detected\":%s"
                "}"
            "}"
            "}",
            cmd_id,
            (long long)rtk_get_current_timestamp(),
            g_sensor_state.current_readings.temperature,
            g_sensor_state.current_readings.humidity,
            g_sensor_state.current_readings.pm25,
            g_sensor_state.current_readings.co2,
            g_sensor_state.current_readings.luminance,
            g_sensor_state.current_readings.motion_detected ? "true" : "false"
        );
        
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

static int sensor_initialize(const rtk_plugin_config_t* config) {
    if (!config) return RTK_PLUGIN_ERROR_INVALID_PARAM;
    
    // 複製配置
    memcpy(&g_sensor_state.config, config, sizeof(rtk_plugin_config_t));
    
    // 初始化設備資訊
    strcpy(g_sensor_state.device_info.id, config->device_id);
    strcpy(g_sensor_state.device_info.type, "iot_sensor");
    strcpy(g_sensor_state.device_info.model, "RTK-SENSOR-5000");
    strcpy(g_sensor_state.device_info.serial_number, "SN20240001");
    strcpy(g_sensor_state.device_info.hw_version, "revB");
    strcpy(g_sensor_state.device_info.fw_version, "2.1.0");
    
    // 初始化感測器狀態
    memset(&g_sensor_state.current_readings, 0, sizeof(sensor_readings_t));
    g_sensor_state.current_readings.temperature = 22.5;
    g_sensor_state.current_readings.humidity = 55.0;
    g_sensor_state.current_readings.pm25 = 15;
    g_sensor_state.current_readings.co2 = 450;
    g_sensor_state.current_readings.luminance = 300;
    
    // 初始化校準資料
    g_sensor_state.calibration.temp_offset = 0.0;
    g_sensor_state.calibration.humidity_offset = 0.0;
    g_sensor_state.calibration.is_calibrated = 0;
    
    // 初始化統計
    g_sensor_state.motion_event_count = 0;
    g_sensor_state.air_quality_alert_count = 0;
    g_sensor_state.uptime_seconds = 0;
    
    printf("[IoT-Sensor] Initialized: device_id=%s, model=%s\n", 
           config->device_id, g_sensor_state.device_info.model);
    
    return RTK_PLUGIN_SUCCESS;
}

static int sensor_start(void) {
    g_sensor_state.is_running = 1;
    printf("[IoT-Sensor] Started - Multi-sensor monitoring active\n");
    return RTK_PLUGIN_SUCCESS;
}

static int sensor_stop(void) {
    g_sensor_state.is_running = 0;
    printf("[IoT-Sensor] Stopped\n");
    return RTK_PLUGIN_SUCCESS;
}

static int sensor_health_check(void) {
    // 檢查感測器健康狀態
    if (!g_sensor_state.is_running) {
        return 0;
    }
    
    // 簡單健康檢查 - 確保讀數在合理範圍內
    if (g_sensor_state.current_readings.temperature < -50 || 
        g_sensor_state.current_readings.temperature > 100 ||
        g_sensor_state.current_readings.humidity < 0 || 
        g_sensor_state.current_readings.humidity > 100) {
        return 0;  // 感測器讀數異常
    }
    
    return 1;  // 健康
}

static void sensor_free_json_string(char* json_str) {
    if (json_str) {
        free(json_str);
    }
}

// === 插件虛函式表 ===
static const rtk_device_plugin_vtable_t iot_sensor_vtable = {
    .get_device_info = sensor_get_device_info,
    .get_capabilities = sensor_get_capabilities,
    .get_state = sensor_get_state,
    .get_attributes = NULL,  // 未實作
    .get_telemetry = sensor_get_telemetry,
    .list_telemetry_metrics = NULL,  // 未實作
    .on_event_trigger = NULL,  // 未實作
    .get_supported_events = NULL,  // 未實作
    .handle_command = sensor_handle_command,
    .get_supported_commands = NULL,  // 未實作
    .initialize = sensor_initialize,
    .start = sensor_start,
    .stop = sensor_stop,
    .health_check = sensor_health_check,
    .free_json_string = sensor_free_json_string
};

// === 插件註冊函式 (必要) ===
const rtk_device_plugin_vtable_t* rtk_plugin_get_vtable(void) {
    return &iot_sensor_vtable;
}

const char* rtk_plugin_get_version(void) {
    return "1.0.0";
}

const char* rtk_plugin_get_name(void) {
    return "iot_sensor";
}
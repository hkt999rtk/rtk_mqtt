#include "rtk_message_codec.h"
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <time.h>
#include <sys/time.h>
#include <cJSON.h>

/**
 * @file message_codec.c
 * @brief RTK MQTT 訊息編解碼器實作
 */

// === 輔助函式實作 ===

int64_t rtk_get_current_timestamp(void) {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return (int64_t)tv.tv_sec * 1000 + tv.tv_usec / 1000;
}

int rtk_generate_request_id(char* buffer, size_t buffer_size) {
    if (!buffer || buffer_size < 16) {
        return -1;
    }
    
    // 產生基於時間的唯一 ID
    int64_t timestamp = rtk_get_current_timestamp();
    static int counter = 0;
    
    int len = snprintf(buffer, buffer_size, "req-%lld-%d", 
                       (long long)timestamp, ++counter);
    
    return (len < buffer_size) ? len : -1;
}

int rtk_validate_schema(const char* schema) {
    if (!schema || strlen(schema) == 0) {
        return 0;
    }
    
    // 基本格式驗證: 應包含 '/' 且不為空
    const char* slash = strchr(schema, '/');
    return (slash != NULL && slash != schema && *(slash + 1) != '\0');
}

const char* rtk_severity_to_string(rtk_severity_level_t severity) {
    switch (severity) {
        case RTK_SEVERITY_INFO: return "info";
        case RTK_SEVERITY_WARNING: return "warning";
        case RTK_SEVERITY_ERROR: return "error";
        case RTK_SEVERITY_CRITICAL: return "critical";
        default: return "info";
    }
}

rtk_severity_level_t rtk_severity_from_string(const char* severity_str) {
    if (!severity_str) return RTK_SEVERITY_INFO;
    
    if (strcmp(severity_str, "warning") == 0) return RTK_SEVERITY_WARNING;
    if (strcmp(severity_str, "error") == 0) return RTK_SEVERITY_ERROR;
    if (strcmp(severity_str, "critical") == 0) return RTK_SEVERITY_CRITICAL;
    
    return RTK_SEVERITY_INFO;  // 預設
}

// === 編碼函式實作 ===

static int encode_trace_info(const rtk_trace_info_t* trace, char* buffer, size_t buffer_size) {
    if (!trace) {
        return 0;  // 無追蹤資訊
    }
    
    int len = 0;
    if (strlen(trace->req_id) > 0 || strlen(trace->correlation_id) > 0 || strlen(trace->span_id) > 0) {
        len = snprintf(buffer, buffer_size, ",\"trace\":{");
        
        int first = 1;
        if (strlen(trace->req_id) > 0) {
            len += snprintf(buffer + len, buffer_size - len, "%s\"req_id\":\"%s\"", 
                           first ? "" : ",", trace->req_id);
            first = 0;
        }
        if (strlen(trace->correlation_id) > 0) {
            len += snprintf(buffer + len, buffer_size - len, "%s\"correlation_id\":\"%s\"", 
                           first ? "" : ",", trace->correlation_id);
            first = 0;
        }
        if (strlen(trace->span_id) > 0) {
            len += snprintf(buffer + len, buffer_size - len, "%s\"span_id\":\"%s\"", 
                           first ? "" : ",", trace->span_id);
        }
        
        len += snprintf(buffer + len, buffer_size - len, "}");
    }
    
    return len;
}

int rtk_encode_state_message(const rtk_state_message_t* message, 
                             char* buffer, size_t buffer_size) {
    if (!message || !buffer || buffer_size == 0) {
        return -1;
    }
    
    // 建立 JSON 物件
    cJSON *json = cJSON_CreateObject();
    if (json == NULL) {
        return -1;
    }
    
    // 添加標頭資訊
    cJSON_AddStringToObject(json, "schema", message->header.schema);
    cJSON_AddNumberToObject(json, "ts", (double)message->header.timestamp);
    
    // Trace 資訊 (如果有)
    if (message->header.has_trace) {
        cJSON *trace = cJSON_CreateObject();
        if (trace != NULL) {
            if (strlen(message->header.trace.req_id) > 0) {
                cJSON_AddStringToObject(trace, "req_id", message->header.trace.req_id);
            }
            if (strlen(message->header.trace.correlation_id) > 0) {
                cJSON_AddStringToObject(trace, "correlation_id", message->header.trace.correlation_id);
            }
            if (strlen(message->header.trace.span_id) > 0) {
                cJSON_AddStringToObject(trace, "span_id", message->header.trace.span_id);
            }
            cJSON_AddItemToObject(json, "trace", trace);
        }
    }
    
    // 狀態資料
    cJSON_AddStringToObject(json, "health", message->health);
    
    if (strlen(message->fw_version) > 0) {
        cJSON_AddStringToObject(json, "fw", message->fw_version);
    }
    
    if (message->uptime_seconds > 0) {
        cJSON_AddNumberToObject(json, "uptime_s", message->uptime_seconds);
    }
    
    if (message->cpu_usage >= 0) {
        cJSON_AddNumberToObject(json, "cpu_usage", message->cpu_usage);
    }
    
    if (message->memory_usage >= 0) {
        cJSON_AddNumberToObject(json, "memory_usage", message->memory_usage);
    }
    
    if (message->temperature > -273.15) {  // 絕對零度以上
        cJSON_AddNumberToObject(json, "temperature_c", message->temperature);
    }
    
    // 自訂資料 (解析並合併到主物件)
    if (strlen(message->custom_data) > 0) {
        cJSON *custom = cJSON_Parse(message->custom_data);
        if (custom != NULL && cJSON_IsObject(custom)) {
            // 合併自訂資料到主 JSON 物件
            cJSON *item = custom->child;
            while (item != NULL) {
                cJSON *next = item->next;  // 儲存下一個項目
                // 創建副本並添加到主物件 (避免記憶體問題)
                cJSON *copy = cJSON_Duplicate(item, 1);
                if (copy != NULL) {
                    cJSON_AddItemToObject(json, item->string, copy);
                }
                item = next;
            }
            cJSON_Delete(custom);
        } else {
            // 如果解析失敗，記錄警告但不中斷處理
            printf("[RTK-Codec] Warning: Failed to parse custom_data as JSON\n");
        }
    }
    
    // 序列化為字串
    char *json_string = cJSON_PrintUnformatted(json);
    cJSON_Delete(json);
    
    if (json_string == NULL) {
        return -1;
    }
    
    // 檢查緩衝區大小
    size_t json_len = strlen(json_string);
    if (json_len >= buffer_size) {
        free(json_string);
        return -1;
    }
    
    // 複製到緩衝區
    strcpy(buffer, json_string);
    free(json_string);
    
    return (int)json_len;
}

int rtk_encode_event_message(const rtk_event_message_t* message,
                            char* buffer, size_t buffer_size) {
    if (!message || !buffer || buffer_size == 0) {
        return -1;
    }
    
    // 建立 JSON 物件
    cJSON *json = cJSON_CreateObject();
    if (json == NULL) {
        return -1;
    }
    
    // 添加標頭資訊
    cJSON_AddStringToObject(json, "schema", message->header.schema);
    cJSON_AddNumberToObject(json, "ts", (double)message->header.timestamp);
    
    // Trace 資訊 (如果有)
    if (message->header.has_trace) {
        cJSON *trace = cJSON_CreateObject();
        if (trace != NULL) {
            if (strlen(message->header.trace.req_id) > 0) {
                cJSON_AddStringToObject(trace, "req_id", message->header.trace.req_id);
            }
            if (strlen(message->header.trace.correlation_id) > 0) {
                cJSON_AddStringToObject(trace, "correlation_id", message->header.trace.correlation_id);
            }
            if (strlen(message->header.trace.span_id) > 0) {
                cJSON_AddStringToObject(trace, "span_id", message->header.trace.span_id);
            }
            cJSON_AddItemToObject(json, "trace", trace);
        }
    }
    
    // 事件資料
    cJSON_AddNumberToObject(json, "seq", message->sequence);
    cJSON_AddStringToObject(json, "severity", rtk_severity_to_string(message->severity));
    cJSON_AddStringToObject(json, "message", message->message);
    cJSON_AddStringToObject(json, "source", message->source);
    
    // 自訂資料 (解析並合併到主物件)
    if (strlen(message->custom_data) > 0) {
        cJSON *custom = cJSON_Parse(message->custom_data);
        if (custom != NULL && cJSON_IsObject(custom)) {
            // 合併自訂資料到主 JSON 物件
            cJSON *item = custom->child;
            while (item != NULL) {
                cJSON *next = item->next;
                // 創建副本並添加到主物件
                cJSON *copy = cJSON_Duplicate(item, 1);
                if (copy != NULL) {
                    cJSON_AddItemToObject(json, item->string, copy);
                }
                item = next;
            }
            cJSON_Delete(custom);
        } else {
            printf("[RTK-Codec] Warning: Failed to parse event custom_data as JSON\n");
        }
    }
    
    // 序列化為字串
    char *json_string = cJSON_PrintUnformatted(json);
    cJSON_Delete(json);
    
    if (json_string == NULL) {
        return -1;
    }
    
    // 檢查緩衝區大小並複製
    size_t json_len = strlen(json_string);
    if (json_len >= buffer_size) {
        free(json_string);
        return -1;
    }
    
    strcpy(buffer, json_string);
    free(json_string);
    
    return (int)json_len;
}

int rtk_encode_command_message(const rtk_command_message_t* message,
                              char* buffer, size_t buffer_size) {
    if (!message || !buffer || buffer_size == 0) {
        return -1;
    }
    
    char trace_json[256] = {0};
    if (message->header.has_trace) {
        encode_trace_info(&message->header.trace, trace_json, sizeof(trace_json));
    }
    
    int len = snprintf(buffer, buffer_size,
        "{"
        "\"id\":\"%s\","
        "\"op\":\"%s\","
        "\"schema\":\"%s\","
        "\"args\":%s,"
        "\"timeout_ms\":%d,"
        "\"expect\":\"%s\","
        "\"ts\":%lld"
        "%s"
        "%s%s%s"
        "}",
        message->id,
        message->operation,
        message->header.schema,
        strlen(message->args) > 0 ? message->args : "{}",
        message->timeout_ms,
        message->expect,
        (long long)message->header.timestamp,
        trace_json,
        strlen(message->reply_to) > 0 ? ",\"reply_to\":\"" : "",
        strlen(message->reply_to) > 0 ? message->reply_to : "",
        strlen(message->reply_to) > 0 ? "\"" : ""
    );
    
    return (len < buffer_size) ? len : -1;
}

int rtk_encode_command_response(const rtk_command_response_t* response,
                               char* buffer, size_t buffer_size) {
    if (!response || !buffer || buffer_size == 0) {
        return -1;
    }
    
    int len = snprintf(buffer, buffer_size,
        "{"
        "\"id\":\"%s\","
        "\"ts\":%lld,"
        "\"ok\":%s,"
        "\"result\":%s"
        "%s%s%s"
        "%s%s%s"
        "%s%s%s"
        "}",
        response->id,
        (long long)response->header.timestamp,
        response->ok ? "true" : "false",
        strlen(response->result) > 0 ? response->result : "null",
        strlen(response->progress) > 0 ? ",\"progress\":\"" : "",
        strlen(response->progress) > 0 ? response->progress : "",
        strlen(response->progress) > 0 ? "\"" : "",
        strlen(response->error_code) > 0 ? ",\"err\":{\"code\":\"" : "",
        strlen(response->error_code) > 0 ? response->error_code : "",
        strlen(response->error_code) > 0 ? "\"" : "",
        strlen(response->error_message) > 0 ? ",\"msg\":\"" : "",
        strlen(response->error_message) > 0 ? response->error_message : "",
        strlen(response->error_message) > 0 ? "\"}" : ""
    );
    
    return (len < buffer_size) ? len : -1;
}

int rtk_encode_lwt_message(const char* status, const char* reason,
                          char* buffer, size_t buffer_size) {
    if (!status || !buffer || buffer_size == 0) {
        return -1;
    }
    
    int64_t timestamp = rtk_get_current_timestamp();
    
    int len = snprintf(buffer, buffer_size,
        "{"
        "\"status\":\"%s\","
        "\"ts\":%lld"
        "%s%s%s"
        "}",
        status,
        (long long)timestamp,
        reason ? ",\"reason\":\"" : "",
        reason ? reason : "",
        reason ? "\"" : ""
    );
    
    return (len < buffer_size) ? len : -1;
}

int rtk_encode_generic_message(const char* schema, const char* custom_json,
                              const rtk_trace_info_t* trace,
                              char* buffer, size_t buffer_size) {
    if (!schema || !buffer || buffer_size == 0) {
        return -1;
    }
    
    int64_t timestamp = rtk_get_current_timestamp();
    
    char trace_json[256] = {0};
    if (trace) {
        encode_trace_info(trace, trace_json, sizeof(trace_json));
    }
    
    int len = snprintf(buffer, buffer_size,
        "{"
        "\"schema\":\"%s\","
        "\"ts\":%lld"
        "%s"
        "%s%s%s"
        "}",
        schema,
        (long long)timestamp,
        trace_json,
        custom_json ? "," : "",
        custom_json ? custom_json : "",
        custom_json ? "" : ""
    );
    
    return (len < buffer_size) ? len : -1;
}

// === 基本解碼函式實作 (簡化版) ===

int rtk_decode_message_header(const char* json, rtk_message_header_t* header) {
    if (!json || !header) {
        return -1;
    }
    
    // 簡化的 JSON 解析 - 實際應用中建議使用專業的 JSON 庫
    // 這裡僅提供基本實作示範
    
    // 提取 schema
    const char* schema_start = strstr(json, "\"schema\":\"");
    if (schema_start) {
        schema_start += 10;  // 跳過 "schema":"
        const char* schema_end = strchr(schema_start, '"');
        if (schema_end) {
            int len = schema_end - schema_start;
            if (len < sizeof(header->schema)) {
                strncpy(header->schema, schema_start, len);
                header->schema[len] = '\0';
            }
        }
    }
    
    // 提取 timestamp
    const char* ts_start = strstr(json, "\"ts\":");
    if (ts_start) {
        ts_start += 5;  // 跳過 "ts":
        header->timestamp = strtoll(ts_start, NULL, 10);
    }
    
    // 檢查是否有 trace 資訊
    header->has_trace = (strstr(json, "\"trace\":") != NULL);
    
    return 0;
}

// 其他解碼函式的簡化實作...
// 實際專案中建議使用 cJSON 或其他 JSON 庫進行完整實作

int rtk_extract_json_field(const char* json, const char* field_name,
                          char* buffer, size_t buffer_size) {
    if (!json || !field_name || !buffer || buffer_size == 0) {
        return -1;
    }
    
    char search_pattern[128];
    snprintf(search_pattern, sizeof(search_pattern), "\"%s\":\"", field_name);
    
    const char* field_start = strstr(json, search_pattern);
    if (!field_start) {
        return -1;  // 欄位不存在
    }
    
    field_start += strlen(search_pattern);
    const char* field_end = strchr(field_start, '"');
    if (!field_end) {
        return -1;
    }
    
    int len = field_end - field_start;
    if (len >= buffer_size) {
        return -1;  // 緩衝區太小
    }
    
    strncpy(buffer, field_start, len);
    buffer[len] = '\0';
    
    return len;
}
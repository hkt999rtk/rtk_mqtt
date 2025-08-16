#include "rtk_schema_validator.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <cJSON.h>

/**
 * @file schema_validator.c
 * @brief RTK MQTT Schema 驗證器實作
 */

// === 內建 Schema 定義 ===

// 狀態訊息 Schema (state/1.0)
static const char* STATE_V1_0_SCHEMA = 
"{"
  "\"$schema\": \"https://json-schema.org/draft/2020-12/schema\","
  "\"title\": \"RTK Device State Message v1.0\","
  "\"type\": \"object\","
  "\"required\": [\"schema\", \"ts\", \"health\"],"
  "\"properties\": {"
    "\"schema\": {\"const\": \"state/1.0\"},"
    "\"ts\": {\"type\": \"integer\", \"minimum\": 0},"
    "\"health\": {\"enum\": [\"ok\", \"warn\", \"error\"]},"
    "\"fw\": {\"type\": \"string\"},"
    "\"uptime_s\": {\"type\": \"integer\", \"minimum\": 0},"
    "\"cpu_usage\": {\"type\": \"number\", \"minimum\": 0, \"maximum\": 100},"
    "\"memory_usage\": {\"type\": \"number\", \"minimum\": 0, \"maximum\": 100},"
    "\"temperature_c\": {\"type\": \"number\"},"
    "\"trace\": {"
      "\"type\": \"object\","
      "\"properties\": {"
        "\"req_id\": {\"type\": \"string\"},"
        "\"correlation_id\": {\"type\": \"string\"},"
        "\"span_id\": {\"type\": \"string\"}"
      "}"
    "}"
  "},"
  "\"additionalProperties\": true"
"}";

// WiFi 漫遊失效事件 Schema (evt.wifi.roam_miss/1.0)
static const char* WIFI_ROAM_MISS_V1_0_SCHEMA =
"{"
  "\"$schema\": \"https://json-schema.org/draft/2020-12/schema\","
  "\"title\": \"RTK WiFi Roaming Miss Event v1.0\","
  "\"type\": \"object\","
  "\"required\": [\"schema\", \"ts\", \"severity\", \"trigger_info\", \"diagnosis\"],"
  "\"properties\": {"
    "\"schema\": {\"const\": \"evt.wifi.roam_miss/1.0\"},"
    "\"ts\": {\"type\": \"integer\", \"minimum\": 0},"
    "\"severity\": {\"enum\": [\"info\", \"warning\", \"error\", \"critical\"]},"
    "\"trigger_info\": {"
      "\"type\": \"object\","
      "\"required\": [\"rssi_threshold\", \"duration_ms\", \"cooldown_ms\"],"
      "\"properties\": {"
        "\"rssi_threshold\": {\"type\": \"integer\", \"maximum\": 0},"
        "\"duration_ms\": {\"type\": \"integer\", \"const\": 10000},"
        "\"cooldown_ms\": {\"type\": \"integer\", \"const\": 300000}"
      "}"
    "},"
    "\"diagnosis\": {"
      "\"type\": \"object\","
      "\"required\": [\"internal_scan_skip_count\", \"environment_ap_count\", \"current_bssid\", \"current_rssi\"],"
      "\"properties\": {"
        "\"internal_scan_skip_count\": {\"type\": \"integer\", \"minimum\": 0},"
        "\"environment_ap_count\": {\"type\": \"integer\", \"minimum\": 0},"
        "\"current_bssid\": {\"type\": \"string\", \"pattern\": \"^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$\"},"
        "\"current_rssi\": {\"type\": \"integer\", \"minimum\": -100, \"maximum\": 0}"
      "}"
    "}"
  "},"
  "\"additionalProperties\": true"
"}";

// LWT 訊息 Schema (lwt/1.0)
static const char* LWT_V1_0_SCHEMA =
"{"
  "\"$schema\": \"https://json-schema.org/draft/2020-12/schema\","
  "\"title\": \"RTK Last Will Testament Message v1.0\","
  "\"type\": \"object\","
  "\"required\": [\"status\", \"ts\"],"
  "\"properties\": {"
    "\"status\": {\"enum\": [\"online\", \"offline\"]},"
    "\"ts\": {\"type\": \"integer\", \"minimum\": 0},"
    "\"reason\": {\"type\": \"string\"}"
  "},"
  "\"additionalProperties\": false"
"}";

// 診斷命令 Schema (cmd.diagnosis.get/1.0)
static const char* CMD_DIAGNOSIS_GET_V1_0_SCHEMA =
"{"
  "\"$schema\": \"https://json-schema.org/draft/2020-12/schema\","
  "\"title\": \"RTK Diagnosis Get Command v1.0\","
  "\"type\": \"object\","
  "\"required\": [\"id\", \"op\", \"schema\", \"args\"],"
  "\"properties\": {"
    "\"id\": {\"type\": \"string\", \"minLength\": 1},"
    "\"op\": {\"const\": \"diagnosis.get\"},"
    "\"schema\": {\"const\": \"cmd.diagnosis.get/1.0\"},"
    "\"args\": {"
      "\"type\": \"object\","
      "\"required\": [\"type\"],"
      "\"properties\": {"
        "\"type\": {\"enum\": [\"wifi\", \"system\", \"network\", \"hardware\"]},"
        "\"detail_level\": {\"enum\": [\"basic\", \"full\"]},"
        "\"include_history\": {\"type\": \"boolean\"}"
      "}"
    "},"
    "\"timeout_ms\": {\"type\": \"integer\", \"minimum\": 1000, \"maximum\": 60000},"
    "\"expect\": {\"enum\": [\"ack\", \"result\", \"none\"]},"
    "\"ts\": {\"type\": \"integer\", \"minimum\": 0}"
  "},"
  "\"additionalProperties\": true"
"}";

// === 內部狀態管理 ===

#define MAX_SCHEMAS 32

static rtk_schema_definition_t registered_schemas[MAX_SCHEMAS];
static int schema_count = 0;
static int is_initialized = 0;

// === 內部輔助函式 ===

static int add_schema_definition(const char* name, const char* version, 
                                const char* description, const char* json_schema) {
    if (schema_count >= MAX_SCHEMAS) {
        printf("[RTK-SCHEMA] Schema registry full\n");
        return RTK_SCHEMA_ERROR_MEMORY;
    }
    
    rtk_schema_definition_t* schema = &registered_schemas[schema_count];
    
    strncpy(schema->name, name, sizeof(schema->name) - 1);
    schema->name[sizeof(schema->name) - 1] = '\0';
    
    strncpy(schema->version, version, sizeof(schema->version) - 1);
    schema->version[sizeof(schema->version) - 1] = '\0';
    
    strncpy(schema->description, description, sizeof(schema->description) - 1);
    schema->description[sizeof(schema->description) - 1] = '\0';
    
    schema->json_schema = json_schema;
    schema->schema_length = strlen(json_schema);
    
    schema_count++;
    
    printf("[RTK-SCHEMA] Registered schema: %s\n", name);
    return RTK_SCHEMA_SUCCESS;
}

// 使用 cJSON 的 JSON 驗證器 (更可靠的 JSON 處理)
static int validate_json_against_schema(const char* json, const char* schema_json,
                                       rtk_validation_result_t* result) {
    if (!json || !schema_json || !result) {
        return RTK_SCHEMA_ERROR_INVALID_PARAM;
    }
    
    // 清空結果
    memset(result, 0, sizeof(rtk_validation_result_t));
    result->is_valid = 1;  // 預設為有效
    
    // 使用 cJSON 驗證 JSON 格式
    cJSON *json_obj = cJSON_Parse(json);
    if (json_obj == NULL) {
        result->is_valid = 0;
        const char *error_ptr = cJSON_GetErrorPtr();
        if (error_ptr != NULL) {
            snprintf(result->error_message, sizeof(result->error_message),
                    "Invalid JSON syntax near: %.50s", error_ptr);
        } else {
            strcpy(result->error_message, "Invalid JSON syntax");
        }
        return RTK_SCHEMA_ERROR_INVALID_JSON;
    }
    
    // 解析 Schema 來檢查必要欄位 (簡化實作)
    cJSON *schema_obj = cJSON_Parse(schema_json);
    if (schema_obj != NULL) {
        cJSON *required = cJSON_GetObjectItemCaseSensitive(schema_obj, "required");
        if (cJSON_IsArray(required)) {
            cJSON *required_field = NULL;
            cJSON_ArrayForEach(required_field, required) {
                if (cJSON_IsString(required_field)) {
                    const char *field_name = cJSON_GetStringValue(required_field);
                    cJSON *field = cJSON_GetObjectItemCaseSensitive(json_obj, field_name);
                    
                    if (field == NULL) {
                        result->is_valid = 0;
                        snprintf(result->error_message, sizeof(result->error_message),
                                "Missing required field: %s", field_name);
                        snprintf(result->error_path, sizeof(result->error_path), "/%s", field_name);
                        cJSON_Delete(json_obj);
                        cJSON_Delete(schema_obj);
                        return RTK_SCHEMA_ERROR_VALIDATION;
                    }
                }
            }
        }
        
        // 檢查特定欄位的值約束 (簡化實作)
        cJSON *properties = cJSON_GetObjectItemCaseSensitive(schema_obj, "properties");
        if (cJSON_IsObject(properties)) {
            cJSON *schema_prop = cJSON_GetObjectItemCaseSensitive(properties, "schema");
            if (schema_prop != NULL) {
                cJSON *schema_const = cJSON_GetObjectItemCaseSensitive(schema_prop, "const");
                if (cJSON_IsString(schema_const)) {
                    cJSON *json_schema = cJSON_GetObjectItemCaseSensitive(json_obj, "schema");
                    if (json_schema != NULL && cJSON_IsString(json_schema)) {
                        const char *expected = cJSON_GetStringValue(schema_const);
                        const char *actual = cJSON_GetStringValue(json_schema);
                        if (strcmp(expected, actual) != 0) {
                            result->is_valid = 0;
                            snprintf(result->error_message, sizeof(result->error_message),
                                    "Schema field mismatch: expected '%s', got '%s'", expected, actual);
                            strcpy(result->error_path, "/schema");
                            cJSON_Delete(json_obj);
                            cJSON_Delete(schema_obj);
                            return RTK_SCHEMA_ERROR_VALIDATION;
                        }
                    }
                }
            }
        }
        
        cJSON_Delete(schema_obj);
    }
    
    cJSON_Delete(json_obj);
    return RTK_SCHEMA_SUCCESS;
}

// === 公開 API 實作 ===

int rtk_schema_validator_init(void) {
    if (is_initialized) {
        return RTK_SCHEMA_SUCCESS;
    }
    
    schema_count = 0;
    memset(registered_schemas, 0, sizeof(registered_schemas));
    
    // 註冊內建 Schema
    int ret = rtk_schema_register_builtin_schemas();
    if (ret != RTK_SCHEMA_SUCCESS) {
        return ret;
    }
    
    is_initialized = 1;
    printf("[RTK-SCHEMA] Schema validator initialized with %d schemas\n", schema_count);
    
    return RTK_SCHEMA_SUCCESS;
}

void rtk_schema_validator_cleanup(void) {
    schema_count = 0;
    is_initialized = 0;
    printf("[RTK-SCHEMA] Schema validator cleaned up\n");
}

int rtk_schema_register_builtin_schemas(void) {
    int ret;
    
    // 註冊狀態訊息 Schema
    ret = add_schema_definition(
        RTK_SCHEMA_NAME_STATE_V1_0, "1.0",
        "Device state message with health status and metrics",
        STATE_V1_0_SCHEMA
    );
    if (ret != RTK_SCHEMA_SUCCESS) return ret;
    
    // 註冊 WiFi 漫遊事件 Schema
    ret = add_schema_definition(
        RTK_SCHEMA_NAME_EVT_WIFI_ROAM_MISS_V1_0, "1.0",
        "WiFi roaming miss event with diagnosis information",
        WIFI_ROAM_MISS_V1_0_SCHEMA
    );
    if (ret != RTK_SCHEMA_SUCCESS) return ret;
    
    // 註冊 LWT 訊息 Schema
    ret = add_schema_definition(
        RTK_SCHEMA_NAME_LWT_V1_0, "1.0",
        "Last Will Testament message for device online/offline status",
        LWT_V1_0_SCHEMA
    );
    if (ret != RTK_SCHEMA_SUCCESS) return ret;
    
    // 註冊診斷命令 Schema
    ret = add_schema_definition(
        RTK_SCHEMA_NAME_CMD_DIAGNOSIS_GET_V1_0, "1.0",
        "Diagnosis get command for requesting device diagnostic data",
        CMD_DIAGNOSIS_GET_V1_0_SCHEMA
    );
    if (ret != RTK_SCHEMA_SUCCESS) return ret;
    
    return RTK_SCHEMA_SUCCESS;
}

int rtk_schema_register_custom(const rtk_schema_definition_t* schema_def) {
    if (!schema_def || !is_initialized) {
        return RTK_SCHEMA_ERROR_INVALID_PARAM;
    }
    
    return add_schema_definition(
        schema_def->name,
        schema_def->version,
        schema_def->description,
        schema_def->json_schema
    );
}

const rtk_schema_definition_t* rtk_schema_find_by_name(const char* schema_name) {
    if (!schema_name || !is_initialized) {
        return NULL;
    }
    
    for (int i = 0; i < schema_count; i++) {
        if (strcmp(registered_schemas[i].name, schema_name) == 0) {
            return &registered_schemas[i];
        }
    }
    
    return NULL;
}

const rtk_schema_definition_t* rtk_schema_find_by_type(rtk_schema_type_t schema_type) {
    const char* schema_name = NULL;
    
    switch (schema_type) {
        case RTK_SCHEMA_STATE_V1_0:
            schema_name = RTK_SCHEMA_NAME_STATE_V1_0;
            break;
        case RTK_SCHEMA_EVT_WIFI_ROAM_MISS_V1_0:
            schema_name = RTK_SCHEMA_NAME_EVT_WIFI_ROAM_MISS_V1_0;
            break;
        case RTK_SCHEMA_CMD_DIAGNOSIS_GET_V1_0:
            schema_name = RTK_SCHEMA_NAME_CMD_DIAGNOSIS_GET_V1_0;
            break;
        case RTK_SCHEMA_LWT_V1_0:
            schema_name = RTK_SCHEMA_NAME_LWT_V1_0;
            break;
        default:
            return NULL;
    }
    
    return rtk_schema_find_by_name(schema_name);
}

int rtk_schema_validate_json(const char* json, const char* schema_name, 
                            rtk_validation_result_t* result) {
    if (!json || !schema_name || !result || !is_initialized) {
        return RTK_SCHEMA_ERROR_INVALID_PARAM;
    }
    
    const rtk_schema_definition_t* schema = rtk_schema_find_by_name(schema_name);
    if (!schema) {
        result->is_valid = 0;
        snprintf(result->error_message, sizeof(result->error_message),
                "Schema not found: %s", schema_name);
        return RTK_SCHEMA_ERROR_NOT_FOUND;
    }
    
    return validate_json_against_schema(json, schema->json_schema, result);
}

int rtk_schema_validate_json_by_type(const char* json, rtk_schema_type_t schema_type,
                                    rtk_validation_result_t* result) {
    const rtk_schema_definition_t* schema = rtk_schema_find_by_type(schema_type);
    if (!schema) {
        if (result) {
            result->is_valid = 0;
            strcpy(result->error_message, "Schema type not found");
        }
        return RTK_SCHEMA_ERROR_NOT_FOUND;
    }
    
    return rtk_schema_validate_json(json, schema->name, result);
}

int rtk_schema_auto_validate_json(const char* json, rtk_validation_result_t* result) {
    if (!json || !result || !is_initialized) {
        return RTK_SCHEMA_ERROR_INVALID_PARAM;
    }
    
    // 提取 schema 欄位
    char schema_name[64];
    int ret = rtk_schema_extract_name_from_json(json, schema_name, sizeof(schema_name));
    if (ret < 0) {
        result->is_valid = 0;
        strcpy(result->error_message, "Cannot extract schema name from JSON");
        return RTK_SCHEMA_ERROR_INVALID_JSON;
    }
    
    return rtk_schema_validate_json(json, schema_name, result);
}

int rtk_schema_quick_validate(const char* json, const char* schema_name) {
    rtk_validation_result_t result;
    int ret = rtk_schema_validate_json(json, schema_name, &result);
    return (ret == RTK_SCHEMA_SUCCESS && result.is_valid) ? 1 : 0;
}

int rtk_schema_extract_name_from_json(const char* json, char* buffer, size_t buffer_size) {
    if (!json || !buffer || buffer_size == 0) {
        return RTK_SCHEMA_ERROR_INVALID_PARAM;
    }
    
    // 使用 cJSON 解析 JSON 並提取 schema 欄位
    cJSON *json_obj = cJSON_Parse(json);
    if (json_obj == NULL) {
        return RTK_SCHEMA_ERROR_INVALID_JSON;
    }
    
    cJSON *schema = cJSON_GetObjectItemCaseSensitive(json_obj, "schema");
    if (schema == NULL || !cJSON_IsString(schema)) {
        cJSON_Delete(json_obj);
        return RTK_SCHEMA_ERROR_NOT_FOUND;
    }
    
    const char *schema_value = cJSON_GetStringValue(schema);
    if (schema_value == NULL) {
        cJSON_Delete(json_obj);
        return RTK_SCHEMA_ERROR_NOT_FOUND;
    }
    
    size_t len = strlen(schema_value);
    if (len >= buffer_size) {
        cJSON_Delete(json_obj);
        return RTK_SCHEMA_ERROR_MEMORY;
    }
    
    strcpy(buffer, schema_value);
    cJSON_Delete(json_obj);
    
    return (int)len;
}

int rtk_schema_validate_name_format(const char* schema_name) {
    if (!schema_name || strlen(schema_name) == 0) {
        return 0;
    }
    
    // 檢查格式: {type}/{version}
    const char* slash = strchr(schema_name, '/');
    if (!slash || slash == schema_name || *(slash + 1) == '\0') {
        return 0;
    }
    
    // 檢查版本格式 (簡化檢查)
    const char* version = slash + 1;
    char* dot = strchr(version, '.');
    if (!dot) {
        return 0;  // 版本號應包含 '.'
    }
    
    return 1;
}

int rtk_schema_parse_version(const char* schema_name, int* major, int* minor) {
    if (!schema_name || !major || !minor) {
        return RTK_SCHEMA_ERROR_INVALID_PARAM;
    }
    
    const char* slash = strchr(schema_name, '/');
    if (!slash) {
        return RTK_SCHEMA_ERROR_INVALID_PARAM;
    }
    
    const char* version = slash + 1;
    char* dot = strchr(version, '.');
    if (!dot) {
        return RTK_SCHEMA_ERROR_INVALID_PARAM;
    }
    
    *major = atoi(version);
    *minor = atoi(dot + 1);
    
    return RTK_SCHEMA_SUCCESS;
}

int rtk_schema_list_all(const rtk_schema_definition_t** schemas, int max_count) {
    if (!schemas || !is_initialized) {
        return 0;
    }
    
    int count = (schema_count < max_count) ? schema_count : max_count;
    for (int i = 0; i < count; i++) {
        schemas[i] = &registered_schemas[i];
    }
    
    return count;
}

const char* rtk_schema_get_error_string(int error_code) {
    switch (error_code) {
        case RTK_SCHEMA_SUCCESS: return "Success";
        case RTK_SCHEMA_ERROR_INVALID_PARAM: return "Invalid parameter";
        case RTK_SCHEMA_ERROR_NOT_FOUND: return "Schema not found";
        case RTK_SCHEMA_ERROR_INVALID_JSON: return "Invalid JSON format";
        case RTK_SCHEMA_ERROR_VALIDATION: return "Schema validation failed";
        case RTK_SCHEMA_ERROR_MEMORY: return "Memory allocation error";
        case RTK_SCHEMA_ERROR_VERSION: return "Version format error";
        default: return "Unknown error";
    }
}
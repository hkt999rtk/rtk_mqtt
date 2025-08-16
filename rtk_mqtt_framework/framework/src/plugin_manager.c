#include "rtk_device_plugin.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dlfcn.h>
#include <unistd.h>
#include <cJSON.h>

/**
 * @file plugin_manager.c
 * @brief RTK 插件管理器實作
 * 
 * 提供動態載入、實例管理、生命週期控制等功能
 */

// === 內部狀態管理 ===

#define MAX_PLUGINS 16
#define MAX_INSTANCES 32

static rtk_plugin_info_t loaded_plugins[MAX_PLUGINS];
static rtk_plugin_instance_t plugin_instances[MAX_INSTANCES];
static int plugin_count = 0;
static int instance_count = 0;
static int is_initialized = 0;

// === 內部輔助函式 ===

static int find_plugin_by_name(const char* plugin_name) {
    for (int i = 0; i < plugin_count; i++) {
        if (strcmp(loaded_plugins[i].name, plugin_name) == 0) {
            return i;
        }
    }
    return -1;
}

static int find_instance_by_name(const char* instance_name) {
    for (int i = 0; i < instance_count; i++) {
        if (strcmp(plugin_instances[i].name, instance_name) == 0) {
            return i;
        }
    }
    return -1;
}

static int find_free_instance_slot(void) {
    for (int i = 0; i < MAX_INSTANCES; i++) {
        if (plugin_instances[i].plugin_info == NULL) {
            return i;
        }
    }
    return -1;
}

// === 公開 API 實作 ===

int rtk_plugin_manager_init(void) {
    if (is_initialized) {
        return RTK_PLUGIN_SUCCESS;
    }
    
    memset(loaded_plugins, 0, sizeof(loaded_plugins));
    memset(plugin_instances, 0, sizeof(plugin_instances));
    plugin_count = 0;
    instance_count = 0;
    
    printf("[RTK-PLUGIN] Plugin manager initialized\n");
    is_initialized = 1;
    
    return RTK_PLUGIN_SUCCESS;
}

void rtk_plugin_manager_cleanup(void) {
    if (!is_initialized) {
        return;
    }
    
    // 停止並銷毀所有實例
    for (int i = 0; i < MAX_INSTANCES; i++) {
        if (plugin_instances[i].plugin_info != NULL) {
            rtk_plugin_destroy_instance(&plugin_instances[i]);
        }
    }
    
    // 卸載所有插件
    for (int i = 0; i < plugin_count; i++) {
        if (loaded_plugins[i].handle) {
            dlclose(loaded_plugins[i].handle);
            printf("[RTK-PLUGIN] Unloaded plugin: %s\n", loaded_plugins[i].name);
        }
    }
    
    plugin_count = 0;
    instance_count = 0;
    is_initialized = 0;
    
    printf("[RTK-PLUGIN] Plugin manager cleaned up\n");
}

int rtk_plugin_load(const char* plugin_path) {
    if (!plugin_path || !is_initialized) {
        return RTK_PLUGIN_ERROR_INVALID_PARAM;
    }
    
    if (plugin_count >= MAX_PLUGINS) {
        printf("[RTK-PLUGIN] Plugin registry full\n");
        return RTK_PLUGIN_ERROR_MEMORY;
    }
    
    // 檢查檔案是否存在
    if (access(plugin_path, F_OK) != 0) {
        printf("[RTK-PLUGIN] Plugin file not found: %s\n", plugin_path);
        return RTK_PLUGIN_ERROR_LOAD_FAILED;
    }
    
    // 載入動態庫
    void* handle = dlopen(plugin_path, RTLD_LAZY);
    if (!handle) {
        printf("[RTK-PLUGIN] Failed to load plugin: %s\n", dlerror());
        return RTK_PLUGIN_ERROR_LOAD_FAILED;
    }
    
    // 清除之前的錯誤
    dlerror();
    
    // 獲取註冊函式
    rtk_plugin_get_vtable_func get_vtable = 
        (rtk_plugin_get_vtable_func)dlsym(handle, "rtk_plugin_get_vtable");
    rtk_plugin_get_version_func get_version = 
        (rtk_plugin_get_version_func)dlsym(handle, "rtk_plugin_get_version");
    rtk_plugin_get_name_func get_name = 
        (rtk_plugin_get_name_func)dlsym(handle, "rtk_plugin_get_name");
    
    char* error = dlerror();
    if (error != NULL || !get_vtable || !get_version || !get_name) {
        printf("[RTK-PLUGIN] Missing plugin registration functions: %s\n", 
               error ? error : "unknown error");
        dlclose(handle);
        return RTK_PLUGIN_ERROR_LOAD_FAILED;
    }
    
    // 獲取插件資訊
    const char* name = get_name();
    const char* version = get_version();
    const rtk_device_plugin_vtable_t* vtable = get_vtable();
    
    if (!name || !version || !vtable) {
        printf("[RTK-PLUGIN] Invalid plugin registration data\n");
        dlclose(handle);
        return RTK_PLUGIN_ERROR_LOAD_FAILED;
    }
    
    // 檢查插件是否已載入
    if (find_plugin_by_name(name) >= 0) {
        printf("[RTK-PLUGIN] Plugin already loaded: %s\n", name);
        dlclose(handle);
        return RTK_PLUGIN_ERROR_ALREADY_LOADED;
    }
    
    // 驗證虛函式表
    if (!rtk_plugin_validate_vtable(vtable)) {
        printf("[RTK-PLUGIN] Invalid plugin vtable: %s\n", name);
        dlclose(handle);
        return RTK_PLUGIN_ERROR_INVALID_VTABLE;
    }
    
    // 註冊插件
    rtk_plugin_info_t* plugin = &loaded_plugins[plugin_count];
    
    strncpy(plugin->name, name, sizeof(plugin->name) - 1);
    plugin->name[sizeof(plugin->name) - 1] = '\0';
    
    strncpy(plugin->version, version, sizeof(plugin->version) - 1);
    plugin->version[sizeof(plugin->version) - 1] = '\0';
    
    snprintf(plugin->description, sizeof(plugin->description),
             "Plugin: %s v%s (loaded from %s)", name, version, plugin_path);
    
    plugin->vtable = vtable;
    plugin->handle = handle;
    
    plugin_count++;
    
    printf("[RTK-PLUGIN] Loaded plugin: %s v%s\n", name, version);
    
    return RTK_PLUGIN_SUCCESS;
}

int rtk_plugin_unload(const char* plugin_name) {
    if (!plugin_name || !is_initialized) {
        return RTK_PLUGIN_ERROR_INVALID_PARAM;
    }
    
    int index = find_plugin_by_name(plugin_name);
    if (index < 0) {
        return RTK_PLUGIN_ERROR_NOT_FOUND;
    }
    
    rtk_plugin_info_t* plugin = &loaded_plugins[index];
    
    // 檢查是否有正在運行的實例
    for (int i = 0; i < MAX_INSTANCES; i++) {
        if (plugin_instances[i].plugin_info == plugin && plugin_instances[i].is_running) {
            printf("[RTK-PLUGIN] Cannot unload plugin %s: instances still running\n", plugin_name);
            return RTK_PLUGIN_ERROR_NOT_FOUND;  // 改用適當的錯誤碼
        }
    }
    
    // 卸載動態庫
    if (plugin->handle) {
        dlclose(plugin->handle);
    }
    
    // 移除插件 (向前移動其他插件)
    for (int i = index; i < plugin_count - 1; i++) {
        loaded_plugins[i] = loaded_plugins[i + 1];
    }
    memset(&loaded_plugins[plugin_count - 1], 0, sizeof(rtk_plugin_info_t));
    plugin_count--;
    
    printf("[RTK-PLUGIN] Unloaded plugin: %s\n", plugin_name);
    
    return RTK_PLUGIN_SUCCESS;
}

const rtk_plugin_info_t* rtk_plugin_find(const char* plugin_name) {
    if (!plugin_name || !is_initialized) {
        return NULL;
    }
    
    int index = find_plugin_by_name(plugin_name);
    return (index >= 0) ? &loaded_plugins[index] : NULL;
}

int rtk_plugin_list_all(const rtk_plugin_info_t** plugins, int max_count) {
    if (!plugins || !is_initialized) {
        return 0;
    }
    
    int count = (plugin_count < max_count) ? plugin_count : max_count;
    for (int i = 0; i < count; i++) {
        plugins[i] = &loaded_plugins[i];
    }
    
    return count;
}

// === 插件實例管理 ===

rtk_plugin_instance_t* rtk_plugin_create_instance(const char* plugin_name,
                                                  const char* instance_name,
                                                  const rtk_plugin_config_t* config) {
    if (!plugin_name || !instance_name || !config || !is_initialized) {
        return NULL;
    }
    
    // 檢查插件是否存在
    const rtk_plugin_info_t* plugin_info = rtk_plugin_find(plugin_name);
    if (!plugin_info) {
        printf("[RTK-PLUGIN] Plugin not found: %s\n", plugin_name);
        return NULL;
    }
    
    // 檢查實例名稱是否已存在
    if (find_instance_by_name(instance_name) >= 0) {
        printf("[RTK-PLUGIN] Instance already exists: %s\n", instance_name);
        return NULL;
    }
    
    // 找空閒實例槽位
    int slot = find_free_instance_slot();
    if (slot < 0) {
        printf("[RTK-PLUGIN] No free instance slots\n");
        return NULL;
    }
    
    rtk_plugin_instance_t* instance = &plugin_instances[slot];
    
    // 初始化實例
    strncpy(instance->name, instance_name, sizeof(instance->name) - 1);
    instance->name[sizeof(instance->name) - 1] = '\0';
    
    instance->plugin_info = plugin_info;
    memcpy(&instance->config, config, sizeof(rtk_plugin_config_t));
    instance->is_running = 0;
    instance->user_data = NULL;
    
    // 呼叫插件初始化
    if (plugin_info->vtable->initialize) {
        int ret = plugin_info->vtable->initialize(config);
        if (ret != RTK_PLUGIN_SUCCESS) {
            printf("[RTK-PLUGIN] Plugin initialization failed: %s\n", plugin_name);
            memset(instance, 0, sizeof(rtk_plugin_instance_t));
            return NULL;
        }
    }
    
    if (slot >= instance_count) {
        instance_count = slot + 1;
    }
    
    printf("[RTK-PLUGIN] Created instance: %s (plugin: %s)\n", 
           instance_name, plugin_name);
    
    return instance;
}

void rtk_plugin_destroy_instance(rtk_plugin_instance_t* instance) {
    if (!instance || !instance->plugin_info) {
        return;
    }
    
    // 停止實例
    if (instance->is_running) {
        rtk_plugin_stop_instance(instance);
    }
    
    printf("[RTK-PLUGIN] Destroyed instance: %s\n", instance->name);
    
    // 清空實例
    memset(instance, 0, sizeof(rtk_plugin_instance_t));
    
    // 重新計算實例數量
    instance_count = 0;
    for (int i = MAX_INSTANCES - 1; i >= 0; i--) {
        if (plugin_instances[i].plugin_info != NULL) {
            instance_count = i + 1;
            break;
        }
    }
}

int rtk_plugin_start_instance(rtk_plugin_instance_t* instance) {
    if (!instance || !instance->plugin_info) {
        return RTK_PLUGIN_ERROR_INVALID_PARAM;
    }
    
    if (instance->is_running) {
        return RTK_PLUGIN_SUCCESS;  // 已在運行
    }
    
    const rtk_device_plugin_vtable_t* vtable = instance->plugin_info->vtable;
    
    if (vtable->start) {
        int ret = vtable->start();
        if (ret != RTK_PLUGIN_SUCCESS) {
            printf("[RTK-PLUGIN] Failed to start instance: %s\n", instance->name);
            return ret;
        }
    }
    
    instance->is_running = 1;
    
    printf("[RTK-PLUGIN] Started instance: %s\n", instance->name);
    
    return RTK_PLUGIN_SUCCESS;
}

int rtk_plugin_stop_instance(rtk_plugin_instance_t* instance) {
    if (!instance || !instance->plugin_info) {
        return RTK_PLUGIN_ERROR_INVALID_PARAM;
    }
    
    if (!instance->is_running) {
        return RTK_PLUGIN_SUCCESS;  // 已停止
    }
    
    const rtk_device_plugin_vtable_t* vtable = instance->plugin_info->vtable;
    
    if (vtable->stop) {
        int ret = vtable->stop();
        if (ret != RTK_PLUGIN_SUCCESS) {
            printf("[RTK-PLUGIN] Failed to stop instance: %s\n", instance->name);
            return ret;
        }
    }
    
    instance->is_running = 0;
    
    printf("[RTK-PLUGIN] Stopped instance: %s\n", instance->name);
    
    return RTK_PLUGIN_SUCCESS;
}

int rtk_plugin_health_check(rtk_plugin_instance_t* instance) {
    if (!instance || !instance->plugin_info) {
        return RTK_PLUGIN_ERROR_INVALID_PARAM;
    }
    
    if (!instance->is_running) {
        return 0;  // 未運行
    }
    
    const rtk_device_plugin_vtable_t* vtable = instance->plugin_info->vtable;
    
    if (vtable->health_check) {
        return vtable->health_check();
    }
    
    return 1;  // 預設健康
}

// === 插件輔助 API ===

int rtk_plugin_validate_vtable(const rtk_device_plugin_vtable_t* vtable) {
    if (!vtable) {
        return 0;
    }
    
    // 檢查必要函式
    if (!vtable->get_device_info || !vtable->initialize) {
        printf("[RTK-PLUGIN] Missing required vtable functions\n");
        return 0;
    }
    
    return 1;
}

void rtk_plugin_get_default_config(rtk_plugin_config_t* config) {
    if (!config) {
        return;
    }
    
    memset(config, 0, sizeof(rtk_plugin_config_t));
    
    // 設定預設值
    strcpy(config->mqtt_broker, "localhost");
    config->mqtt_port = 1883;
    strcpy(config->tenant, "default");
    strcpy(config->site, "site1");
    strcpy(config->device_id, "device001");
    config->telemetry_interval = 60;  // 60 秒
    config->event_cooldown = 300;     // 5 分鐘
}

int rtk_plugin_load_config_from_file(const char* json_file, rtk_plugin_config_t* config) {
    if (!json_file || !config) {
        return RTK_PLUGIN_ERROR_INVALID_PARAM;
    }
    
    FILE* file = fopen(json_file, "r");
    if (!file) {
        printf("[RTK-PLUGIN] Cannot open config file: %s\n", json_file);
        return RTK_PLUGIN_ERROR_NOT_FOUND;
    }
    
    // 讀取檔案內容 (簡化實作)
    char buffer[4096];
    size_t bytes_read = fread(buffer, 1, sizeof(buffer) - 1, file);
    fclose(file);
    
    if (bytes_read == 0) {
        return RTK_PLUGIN_ERROR_CONFIG;
    }
    
    buffer[bytes_read] = '\0';
    
    // 使用 cJSON 解析配置檔案
    rtk_plugin_get_default_config(config);  // 先設定預設值
    
    cJSON *json = cJSON_Parse(buffer);
    if (json == NULL) {
        printf("[RTK-PLUGIN] Invalid JSON in config file: %s\n", json_file);
        return RTK_PLUGIN_ERROR_CONFIG;
    }
    
    // 解析 MQTT 設定
    cJSON *mqtt_broker = cJSON_GetObjectItemCaseSensitive(json, "mqtt_broker");
    if (cJSON_IsString(mqtt_broker) && mqtt_broker->valuestring != NULL) {
        strncpy(config->mqtt_broker, mqtt_broker->valuestring, sizeof(config->mqtt_broker) - 1);
        config->mqtt_broker[sizeof(config->mqtt_broker) - 1] = '\0';
    }
    
    cJSON *mqtt_port = cJSON_GetObjectItemCaseSensitive(json, "mqtt_port");
    if (cJSON_IsNumber(mqtt_port)) {
        config->mqtt_port = mqtt_port->valueint;
    }
    
    cJSON *device_id = cJSON_GetObjectItemCaseSensitive(json, "device_id");
    if (cJSON_IsString(device_id) && device_id->valuestring != NULL) {
        strncpy(config->device_id, device_id->valuestring, sizeof(config->device_id) - 1);
        config->device_id[sizeof(config->device_id) - 1] = '\0';
    }
    
    cJSON *tenant = cJSON_GetObjectItemCaseSensitive(json, "tenant");
    if (cJSON_IsString(tenant) && tenant->valuestring != NULL) {
        strncpy(config->tenant, tenant->valuestring, sizeof(config->tenant) - 1);
        config->tenant[sizeof(config->tenant) - 1] = '\0';
    }
    
    cJSON *site = cJSON_GetObjectItemCaseSensitive(json, "site");
    if (cJSON_IsString(site) && site->valuestring != NULL) {
        strncpy(config->site, site->valuestring, sizeof(config->site) - 1);
        config->site[sizeof(config->site) - 1] = '\0';
    }
    
    cJSON *mqtt_username = cJSON_GetObjectItemCaseSensitive(json, "mqtt_username");
    if (cJSON_IsString(mqtt_username) && mqtt_username->valuestring != NULL) {
        strncpy(config->mqtt_username, mqtt_username->valuestring, sizeof(config->mqtt_username) - 1);
        config->mqtt_username[sizeof(config->mqtt_username) - 1] = '\0';
    }
    
    cJSON *telemetry_interval = cJSON_GetObjectItemCaseSensitive(json, "telemetry_interval");
    if (cJSON_IsNumber(telemetry_interval)) {
        config->telemetry_interval = telemetry_interval->valueint;
    }
    
    cJSON *event_cooldown = cJSON_GetObjectItemCaseSensitive(json, "event_cooldown");
    if (cJSON_IsNumber(event_cooldown)) {
        config->event_cooldown = event_cooldown->valueint;
    }
    
    // 處理插件特定配置
    cJSON *plugin_config = cJSON_GetObjectItemCaseSensitive(json, "plugin_config");
    if (plugin_config != NULL) {
        char *plugin_config_str = cJSON_PrintUnformatted(plugin_config);
        if (plugin_config_str != NULL) {
            strncpy(config->plugin_config, plugin_config_str, sizeof(config->plugin_config) - 1);
            config->plugin_config[sizeof(config->plugin_config) - 1] = '\0';
            free(plugin_config_str);
        }
    }
    
    cJSON_Delete(json);
    
    printf("[RTK-PLUGIN] Loaded config from: %s\n", json_file);
    
    return RTK_PLUGIN_SUCCESS;
}

int rtk_plugin_save_config_to_file(const rtk_plugin_config_t* config, const char* json_file) {
    if (!config || !json_file) {
        return RTK_PLUGIN_ERROR_INVALID_PARAM;
    }
    
    FILE* file = fopen(json_file, "w");
    if (!file) {
        printf("[RTK-PLUGIN] Cannot create config file: %s\n", json_file);
        return RTK_PLUGIN_ERROR_CONFIG;
    }
    
    // 寫入 JSON 格式配置
    fprintf(file, "{\n");
    fprintf(file, "  \"mqtt_broker\": \"%s\",\n", config->mqtt_broker);
    fprintf(file, "  \"mqtt_port\": %d,\n", config->mqtt_port);
    fprintf(file, "  \"device_id\": \"%s\",\n", config->device_id);
    fprintf(file, "  \"tenant\": \"%s\",\n", config->tenant);
    fprintf(file, "  \"site\": \"%s\",\n", config->site);
    fprintf(file, "  \"mqtt_username\": \"%s\",\n", config->mqtt_username);
    fprintf(file, "  \"telemetry_interval\": %d,\n", config->telemetry_interval);
    fprintf(file, "  \"event_cooldown\": %d,\n", config->event_cooldown);
    fprintf(file, "  \"plugin_config\": %s\n", 
            strlen(config->plugin_config) > 0 ? config->plugin_config : "{}");
    fprintf(file, "}\n");
    
    fclose(file);
    
    printf("[RTK-PLUGIN] Saved config to: %s\n", json_file);
    
    return RTK_PLUGIN_SUCCESS;
}

const char* rtk_plugin_get_error_string(int error_code) {
    switch (error_code) {
        case RTK_PLUGIN_SUCCESS: return "Success";
        case RTK_PLUGIN_ERROR_INVALID_PARAM: return "Invalid parameter";
        case RTK_PLUGIN_ERROR_NOT_FOUND: return "Plugin or instance not found";
        case RTK_PLUGIN_ERROR_LOAD_FAILED: return "Plugin load failed";
        case RTK_PLUGIN_ERROR_ALREADY_LOADED: return "Plugin already loaded";
        case RTK_PLUGIN_ERROR_NOT_RUNNING: return "Plugin instance not running";
        case RTK_PLUGIN_ERROR_INVALID_VTABLE: return "Invalid plugin vtable";
        case RTK_PLUGIN_ERROR_MEMORY: return "Memory allocation error";
        case RTK_PLUGIN_ERROR_CONFIG: return "Configuration error";
        default: return "Unknown error";
    }
}

void rtk_plugin_safe_free_json(rtk_plugin_instance_t* instance, char* json_str) {
    if (!instance || !json_str || !instance->plugin_info) {
        return;
    }
    
    const rtk_device_plugin_vtable_t* vtable = instance->plugin_info->vtable;
    
    if (vtable->free_json_string) {
        vtable->free_json_string(json_str);
    } else {
        free(json_str);  // 預設使用 free
    }
}
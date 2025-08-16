#include "rtk_topic_builder.h"
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

/**
 * @file topic_builder.c
 * @brief RTK MQTT Topic 建構器實作
 */

// 內部配置
static rtk_topic_config_t current_config = {0};
static int is_configured = 0;

// RTK MQTT 規格常數
#define RTK_TOPIC_VERSION "v1"
#define RTK_TOPIC_PREFIX "rtk"

// === 內部輔助函式 ===

static int validate_component(const char* component, const char* name) {
    if (!component || strlen(component) == 0) {
        printf("[RTK-TOPIC] Invalid %s: empty or null\n", name);
        return 0;
    }
    
    // 檢查字元是否符合 MQTT topic 規範 (避免 +, #, /, null)
    for (const char* p = component; *p; p++) {
        if (*p == '+' || *p == '#' || *p == '/' || *p == '\0') {
            printf("[RTK-TOPIC] Invalid %s: contains forbidden character '%c'\n", name, *p);
            return 0;
        }
    }
    
    return 1;
}

static int build_base_topic(char* buffer, size_t buffer_size) {
    if (!is_configured) {
        printf("[RTK-TOPIC] Topic builder not configured\n");
        return -1;
    }
    
    int len = snprintf(buffer, buffer_size, "%s/%s/%s/%s/%s", 
                       RTK_TOPIC_PREFIX, RTK_TOPIC_VERSION,
                       current_config.tenant, current_config.site, 
                       current_config.device_id);
    
    if (len >= buffer_size) {
        printf("[RTK-TOPIC] Buffer too small for base topic\n");
        return -1;
    }
    
    return len;
}

// === 公開 API 實作 ===

int rtk_topic_set_config(const rtk_topic_config_t* config) {
    if (!config) {
        return -1;
    }
    
    // 驗證必要組件
    if (!validate_component(config->tenant, "tenant") ||
        !validate_component(config->site, "site") ||
        !validate_component(config->device_id, "device_id")) {
        return -1;
    }
    
    memcpy(&current_config, config, sizeof(rtk_topic_config_t));
    is_configured = 1;
    
    printf("[RTK-TOPIC] Configuration set: tenant=%s, site=%s, device_id=%s\n",
           current_config.tenant, current_config.site, current_config.device_id);
    
    return 0;
}

int rtk_topic_build(rtk_topic_type_t type, const char* metric_or_event, 
                    char* buffer, size_t buffer_size) {
    if (!buffer || buffer_size == 0) {
        return -1;
    }
    
    int base_len = build_base_topic(buffer, buffer_size);
    if (base_len < 0) {
        return -1;
    }
    
    const char* suffix = NULL;
    char extended_suffix[256] = {0};
    
    switch (type) {
        case RTK_TOPIC_STATE:
            suffix = "/state";
            break;
            
        case RTK_TOPIC_TELEMETRY:
            if (!metric_or_event) {
                printf("[RTK-TOPIC] Telemetry topic requires metric name\n");
                return -1;
            }
            snprintf(extended_suffix, sizeof(extended_suffix), "/telemetry/%s", metric_or_event);
            suffix = extended_suffix;
            break;
            
        case RTK_TOPIC_EVENT:
            if (!metric_or_event) {
                printf("[RTK-TOPIC] Event topic requires event type\n");
                return -1;
            }
            snprintf(extended_suffix, sizeof(extended_suffix), "/evt/%s", metric_or_event);
            suffix = extended_suffix;
            break;
            
        case RTK_TOPIC_ATTRIBUTE:
            suffix = "/attr";
            break;
            
        case RTK_TOPIC_CMD_REQ:
            suffix = "/cmd/req";
            break;
            
        case RTK_TOPIC_CMD_ACK:
            suffix = "/cmd/ack";
            break;
            
        case RTK_TOPIC_CMD_RES:
            suffix = "/cmd/res";
            break;
            
        case RTK_TOPIC_LWT:
            suffix = "/lwt";
            break;
            
        case RTK_TOPIC_GROUP_CMD:
            printf("[RTK-TOPIC] Use rtk_topic_build_group_cmd for group commands\n");
            return -1;
            
        default:
            printf("[RTK-TOPIC] Unknown topic type: %d\n", type);
            return -1;
    }
    
    int total_len = base_len + strlen(suffix);
    if (total_len >= buffer_size) {
        printf("[RTK-TOPIC] Buffer too small for complete topic\n");
        return -1;
    }
    
    strcat(buffer, suffix);
    return total_len;
}

int rtk_topic_build_state(char* buffer, size_t buffer_size) {
    return rtk_topic_build(RTK_TOPIC_STATE, NULL, buffer, buffer_size);
}

int rtk_topic_build_telemetry(const char* metric, char* buffer, size_t buffer_size) {
    return rtk_topic_build(RTK_TOPIC_TELEMETRY, metric, buffer, buffer_size);
}

int rtk_topic_build_event(const char* event_type, char* buffer, size_t buffer_size) {
    return rtk_topic_build(RTK_TOPIC_EVENT, event_type, buffer, buffer_size);
}

int rtk_topic_build_attribute(char* buffer, size_t buffer_size) {
    return rtk_topic_build(RTK_TOPIC_ATTRIBUTE, NULL, buffer, buffer_size);
}

int rtk_topic_build_cmd_req(char* buffer, size_t buffer_size) {
    return rtk_topic_build(RTK_TOPIC_CMD_REQ, NULL, buffer, buffer_size);
}

int rtk_topic_build_cmd_ack(char* buffer, size_t buffer_size) {
    return rtk_topic_build(RTK_TOPIC_CMD_ACK, NULL, buffer, buffer_size);
}

int rtk_topic_build_cmd_res(char* buffer, size_t buffer_size) {
    return rtk_topic_build(RTK_TOPIC_CMD_RES, NULL, buffer, buffer_size);
}

int rtk_topic_build_lwt(char* buffer, size_t buffer_size) {
    return rtk_topic_build(RTK_TOPIC_LWT, NULL, buffer, buffer_size);
}

int rtk_topic_build_group_cmd(const char* group_id, char* buffer, size_t buffer_size) {
    if (!group_id || !validate_component(group_id, "group_id")) {
        return -1;
    }
    
    if (!is_configured) {
        printf("[RTK-TOPIC] Topic builder not configured\n");
        return -1;
    }
    
    int len = snprintf(buffer, buffer_size, "%s/%s/%s/%s/group/%s/cmd/req",
                       RTK_TOPIC_PREFIX, RTK_TOPIC_VERSION,
                       current_config.tenant, current_config.site, group_id);
    
    if (len >= buffer_size) {
        printf("[RTK-TOPIC] Buffer too small for group command topic\n");
        return -1;
    }
    
    return len;
}

int rtk_topic_parse(const char* topic, rtk_topic_config_t* config, 
                    rtk_topic_type_t* type, char* metric_or_event, 
                    size_t metric_buffer_size) {
    if (!topic || !config || !type) {
        return -1;
    }
    
    // 複製 topic 用於解析 (strtok 會修改原字串)
    char topic_copy[512];
    strncpy(topic_copy, topic, sizeof(topic_copy) - 1);
    topic_copy[sizeof(topic_copy) - 1] = '\0';
    
    // 解析組件
    char* token = strtok(topic_copy, "/");
    char* components[10] = {0};
    int component_count = 0;
    
    while (token && component_count < 10) {
        components[component_count++] = token;
        token = strtok(NULL, "/");
    }
    
    // 驗證基本格式: rtk/v1/{tenant}/{site}/{device_id}/...
    if (component_count < 6 || 
        strcmp(components[0], RTK_TOPIC_PREFIX) != 0 ||
        strcmp(components[1], RTK_TOPIC_VERSION) != 0) {
        printf("[RTK-TOPIC] Invalid topic format: %s\n", topic);
        return -1;
    }
    
    // 提取基本配置
    strncpy(config->tenant, components[2], sizeof(config->tenant) - 1);
    strncpy(config->site, components[3], sizeof(config->site) - 1);
    strncpy(config->device_id, components[4], sizeof(config->device_id) - 1);
    
    // 解析訊息類型
    if (component_count == 6) {
        if (strcmp(components[5], "state") == 0) {
            *type = RTK_TOPIC_STATE;
        } else if (strcmp(components[5], "attr") == 0) {
            *type = RTK_TOPIC_ATTRIBUTE;
        } else if (strcmp(components[5], "lwt") == 0) {
            *type = RTK_TOPIC_LWT;
        } else {
            return -1;
        }
    } else if (component_count == 7) {
        if (strcmp(components[5], "telemetry") == 0) {
            *type = RTK_TOPIC_TELEMETRY;
            if (metric_or_event && metric_buffer_size > 0) {
                strncpy(metric_or_event, components[6], metric_buffer_size - 1);
                metric_or_event[metric_buffer_size - 1] = '\0';
            }
        } else if (strcmp(components[5], "evt") == 0) {
            *type = RTK_TOPIC_EVENT;
            if (metric_or_event && metric_buffer_size > 0) {
                strncpy(metric_or_event, components[6], metric_buffer_size - 1);
                metric_or_event[metric_buffer_size - 1] = '\0';
            }
        } else {
            return -1;
        }
    } else if (component_count == 8 && strcmp(components[5], "cmd") == 0) {
        if (strcmp(components[6], "req") == 0) {
            *type = RTK_TOPIC_CMD_REQ;
        } else if (strcmp(components[6], "ack") == 0) {
            *type = RTK_TOPIC_CMD_ACK;
        } else if (strcmp(components[6], "res") == 0) {
            *type = RTK_TOPIC_CMD_RES;
        } else {
            return -1;
        }
    } else {
        return -1;
    }
    
    return 0;
}

int rtk_topic_is_valid(const char* topic) {
    rtk_topic_config_t config;
    rtk_topic_type_t type;
    char metric[128];
    
    return rtk_topic_parse(topic, &config, &type, metric, sizeof(metric)) == 0;
}

int rtk_topic_build_subscribe_pattern(rtk_subscribe_pattern_t pattern, 
                                       char* buffer, size_t buffer_size) {
    if (!buffer || buffer_size == 0) {
        return -1;
    }
    
    if (!is_configured) {
        printf("[RTK-TOPIC] Topic builder not configured\n");
        return -1;
    }
    
    int len = -1;
    
    switch (pattern) {
        case RTK_SUB_ALL_DEVICES:
            len = snprintf(buffer, buffer_size, "%s/%s/%s/%s/+/state",
                          RTK_TOPIC_PREFIX, RTK_TOPIC_VERSION,
                          current_config.tenant, current_config.site);
            break;
            
        case RTK_SUB_ALL_EVENTS:
            len = snprintf(buffer, buffer_size, "%s/%s/%s/%s/+/evt/#",
                          RTK_TOPIC_PREFIX, RTK_TOPIC_VERSION,
                          current_config.tenant, current_config.site);
            break;
            
        case RTK_SUB_ALL_TELEMETRY:
            len = snprintf(buffer, buffer_size, "%s/%s/%s/%s/+/telemetry/#",
                          RTK_TOPIC_PREFIX, RTK_TOPIC_VERSION,
                          current_config.tenant, current_config.site);
            break;
            
        case RTK_SUB_ALL_COMMANDS:
            len = snprintf(buffer, buffer_size, "%s/%s/%s/%s/+/cmd/#",
                          RTK_TOPIC_PREFIX, RTK_TOPIC_VERSION,
                          current_config.tenant, current_config.site);
            break;
            
        case RTK_SUB_DEVICE_ALL:
            len = snprintf(buffer, buffer_size, "%s/%s/%s/%s/%s/#",
                          RTK_TOPIC_PREFIX, RTK_TOPIC_VERSION,
                          current_config.tenant, current_config.site,
                          current_config.device_id);
            break;
            
        case RTK_SUB_GLOBAL_MONITOR:
            len = snprintf(buffer, buffer_size, "%s/%s/+/+/+/evt/#",
                          RTK_TOPIC_PREFIX, RTK_TOPIC_VERSION);
            break;
            
        default:
            printf("[RTK-TOPIC] Unknown subscribe pattern: %d\n", pattern);
            return -1;
    }
    
    if (len >= buffer_size) {
        printf("[RTK-TOPIC] Buffer too small for subscribe pattern\n");
        return -1;
    }
    
    return len;
}
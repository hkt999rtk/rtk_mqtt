/**
 * @file test_minimal_build.c  
 * @brief 最小構建測試 - 驗證核心組件可以編譯
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// 測試核心頭文件
#include "framework/include/rtk_platform_compat.h"
#include "framework/include/rtk_json_config.h" 
#include "framework/include/rtk_topic_builder.h"
#include "framework/include/rtk_message_codec.h"

// cJSON
#include "external/cjson/cJSON.h"

int main() {
    printf("=== RTK MQTT Framework 最小構建測試 ===\n");
    
    // 測試 JSON 配置
    printf("測試 JSON 配置...\n");
    cJSON *config = cJSON_CreateObject();
    cJSON *device_id = cJSON_CreateString("test_device_001");
    cJSON_AddItemToObject(config, "device_id", device_id);
    
    char *config_str = cJSON_Print(config);
    printf("配置 JSON: %s\n", config_str);
    
    // 測試主題構建
    printf("測試主題構建...\n");
    
    // 設置配置
    rtk_topic_config_t topic_config = {0};
    strcpy(topic_config.tenant, "test_tenant");
    strcpy(topic_config.site, "test_site");
    strcpy(topic_config.device_id, "device_001");
    
    int result = rtk_topic_set_config(&topic_config);
    if (result != 0) {
        printf("❌ 設置主題配置失敗\n");
        return -1;
    }
    
    // 構建狀態主題
    char topic[256];
    result = rtk_topic_build_state(topic, sizeof(topic));
    if (result > 0) {
        printf("✅ 狀態主題: %s\n", topic);
    } else {
        printf("❌ 主題構建失敗: %d\n", result);
    }
    
    // 清理
    free(config_str);
    cJSON_Delete(config);
    
    printf("✅ 最小構建測試完成\n");
    return 0;
}
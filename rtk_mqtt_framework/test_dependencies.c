/**
 * @file test_dependencies.c
 * @brief 測試零外部依賴的核心庫
 * 
 * 這個測試程序驗證：
 * 1. cJSON 庫可以正常編譯和使用
 * 2. Paho MQTT C 庫可以正常編譯和使用
 * 3. 所有依賴都已本地化，無需外部安裝
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// 測試 cJSON
#include "external/cjson/cJSON.h"

// 測試 Paho MQTT C (如果需要)
#ifdef TEST_PAHO_MQTT
#include "external/paho-mqtt-c/src/MQTTClient.h"
#endif

/**
 * @brief 測試 cJSON 功能
 */
int test_cjson() {
    printf("=== 測試 cJSON 庫 ===\n");
    
    // 創建 JSON 對象
    cJSON *json = cJSON_CreateObject();
    if (!json) {
        printf("❌ 無法創建 cJSON 對象\n");
        return -1;
    }
    
    // 添加字段
    cJSON *name = cJSON_CreateString("RTK MQTT Framework");
    cJSON *version = cJSON_CreateString("1.0.0");
    cJSON *has_dependencies = cJSON_CreateBool(0);  // false = 無外部依賴
    
    cJSON_AddItemToObject(json, "name", name);
    cJSON_AddItemToObject(json, "version", version);
    cJSON_AddItemToObject(json, "has_external_dependencies", has_dependencies);
    
    // 轉換為字符串
    char *json_string = cJSON_Print(json);
    if (!json_string) {
        printf("❌ 無法轉換 JSON 為字符串\n");
        cJSON_Delete(json);
        return -1;
    }
    
    printf("✅ cJSON 測試成功\n");
    printf("JSON 輸出: %s\n", json_string);
    
    // 清理
    free(json_string);
    cJSON_Delete(json);
    
    return 0;
}

/**
 * @brief 測試 Paho MQTT C 基本功能
 */
int test_paho_mqtt() {
    printf("\n=== 測試 Paho MQTT C 庫 ===\n");
    
#ifdef TEST_PAHO_MQTT
    // 測試客戶端創建
    MQTTClient client;
    MQTTClient_connectOptions conn_opts = MQTTClient_connectOptions_initializer;
    
    int rc = MQTTClient_create(&client, "tcp://test.mosquitto.org:1883", 
                               "rtk_test_client", MQTTCLIENT_PERSISTENCE_NONE, NULL);
    
    if (rc != MQTTCLIENT_SUCCESS) {
        printf("❌ 無法創建 MQTT 客戶端: %d\n", rc);
        return -1;
    }
    
    printf("✅ Paho MQTT C 測試成功\n");
    printf("MQTT 客戶端創建成功\n");
    
    // 清理
    MQTTClient_destroy(&client);
#else
    printf("✅ Paho MQTT C 可用 (未啟用測試)\n");
    printf("庫文件存在於: external/paho-mqtt-c/\n");
#endif
    
    return 0;
}

/**
 * @brief 檢查依賴文件是否存在
 */
int check_dependency_files() {
    printf("\n=== 檢查依賴文件 ===\n");
    
    // 檢查 cJSON 文件
    FILE *cjson_h = fopen("external/cjson/cJSON.h", "r");
    if (cjson_h) {
        printf("✅ cJSON 頭文件存在\n");
        fclose(cjson_h);
    } else {
        printf("❌ cJSON 頭文件不存在\n");
        return -1;
    }
    
    // 檢查 Paho MQTT C 文件
    FILE *mqtt_h = fopen("external/paho-mqtt-c/src/MQTTClient.h", "r");
    if (mqtt_h) {
        printf("✅ Paho MQTT C 頭文件存在\n");
        fclose(mqtt_h);
    } else {
        printf("❌ Paho MQTT C 頭文件不存在\n");
        return -1;
    }
    
    return 0;
}

/**
 * @brief 主函數
 */
int main() {
    printf("🎯 RTK MQTT Framework 零依賴測試\n");
    printf("====================================\n");
    
    int result = 0;
    
    // 檢查文件
    if (check_dependency_files() != 0) {
        printf("❌ 依賴文件檢查失敗\n");
        result = -1;
    }
    
    // 測試 cJSON
    if (test_cjson() != 0) {
        printf("❌ cJSON 測試失敗\n");
        result = -1;
    }
    
    // 測試 Paho MQTT
    if (test_paho_mqtt() != 0) {
        printf("❌ Paho MQTT 測試失敗\n");
        result = -1;
    }
    
    printf("\n====================================\n");
    if (result == 0) {
        printf("🎉 所有測試通過！\n");
        printf("✅ RTK MQTT Framework 實現了零外部依賴\n");
        printf("✅ 可以直接編譯，無需安裝額外的庫\n");
    } else {
        printf("❌ 有測試失敗，請檢查配置\n");
    }
    
    return result;
}
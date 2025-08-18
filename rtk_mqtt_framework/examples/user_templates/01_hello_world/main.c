/*
 * RTK MQTT Framework Hello World 範例
 * 這是一個最小的範例，展示如何：
 * 1. 初始化 RTK MQTT 框架
 * 2. 連接到 MQTT broker
 * 3. 發布一條簡單訊息
 * 4. 清理資源並退出
 */

#include <stdio.h>
#include <string.h>
#include <rtk_mqtt_client.h>

int main() {
    printf("RTK MQTT Framework Hello World 範例\n");
    printf("===================================\n");
    
    // 步驟1：初始化 MQTT 客戶端系統
    if (rtk_mqtt_init(RTK_MQTT_BACKEND_PUBSUB) != RTK_MQTT_SUCCESS) {
        printf("❌ 無法初始化 MQTT 客戶端\n");
        return -1;
    }
    
    // 步驟2：配置 MQTT 客戶端
    rtk_mqtt_config_t config;
    rtk_mqtt_create_default_config(&config, "test.mosquitto.org", 1883, "hello_world_device");
    
    if (rtk_mqtt_configure(&config) != RTK_MQTT_SUCCESS) {
        printf("❌ 無法配置 MQTT 客戶端\n");
        rtk_mqtt_cleanup();
        return -1;
    }
    
    // 步驟3：連接到 MQTT broker
    printf("正在連接到 MQTT broker...\n");
    if (rtk_mqtt_connect() != RTK_MQTT_SUCCESS) {
        printf("❌ 無法連接到 MQTT broker\n");
        rtk_mqtt_cleanup();
        return -1;
    }
    printf("✓ 成功連接到 test.mosquitto.org:1883\n");
    
    // 步驟4：發布 Hello World 訊息
    printf("正在發布 Hello World 訊息...\n");
    const char* message = "Hello World from RTK MQTT Framework!";
    if (rtk_mqtt_publish("rtk/v1/demo/site1/hello_world_device/state", 
                         message, strlen(message), RTK_MQTT_QOS_0, 0) == RTK_MQTT_SUCCESS) {
        printf("✓ Hello World 訊息發布成功！\n");
    } else {
        printf("❌ 訊息發布失敗\n");
    }
    
    // 步驟5：清理資源
    printf("正在斷開連接...\n");
    rtk_mqtt_disconnect();
    rtk_mqtt_cleanup();
    printf("✓ 資源清理完成\n");
    
    printf("\n🎉 Hello World 範例執行完成！\n");
    return 0;
}
/*
 * RTK MQTT Framework Hello World 範例
 * 這是一個最小的 20 行範例，展示如何：
 * 1. 初始化 RTK MQTT 框架
 * 2. 連接到 MQTT broker
 * 3. 發布一條簡單訊息
 * 4. 清理資源並退出
 */

#include <stdio.h>
#include <rtk_mqtt_client.h>

int main() {
    printf("RTK MQTT Framework Hello World 範例\n");
    printf("===================================\n");
    
    // 步驟1：創建 MQTT 客戶端 (連接到公共測試 broker)
    rtk_mqtt_client_t* client = rtk_mqtt_client_create("test.mosquitto.org", 1883, "hello_world_device");
    if (client == NULL) {
        printf("❌ 無法創建 MQTT 客戶端\n");
        return -1;
    }
    
    // 步驟2：連接到 MQTT broker
    printf("正在連接到 MQTT broker...\n");
    if (rtk_mqtt_client_connect(client) != RTK_SUCCESS) {
        printf("❌ 無法連接到 MQTT broker\n");
        rtk_mqtt_client_destroy(client);
        return -1;
    }
    printf("✓ 成功連接到 test.mosquitto.org:1883\n");
    
    // 步驟3：發布 Hello World 訊息
    printf("正在發布 Hello World 訊息...\n");
    rtk_mqtt_client_publish_state(client, "online", "healthy");
    printf("✓ Hello World 訊息發布成功！\n");
    
    // 步驟4：清理資源
    rtk_mqtt_client_disconnect(client);
    rtk_mqtt_client_destroy(client);
    printf("✓ 資源清理完成\n");
    
    printf("\n🎉 Hello World 範例執行完成！\n");
    return 0;
}
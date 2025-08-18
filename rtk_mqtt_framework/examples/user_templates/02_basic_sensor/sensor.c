/*
 * RTK MQTT Framework 基本感測器範例
 * 這個範例展示如何：
 * 1. 模擬溫度和濕度感測器
 * 2. 週期性發布遙測資料
 * 3. 使用 RTK 標準主題格式
 * 4. 處理基本錯誤情況
 * 5. 優雅地處理中斷信號
 */

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <signal.h>
#include <time.h>
#include <math.h>
#include <rtk_mqtt_client.h>
#include <rtk_topic_builder.h>

// 全域變數控制程式執行
static volatile int running = 1;

// 感測器狀態結構
typedef struct {
    float temperature;    // 溫度 (攝氏度)
    float humidity;      // 濕度 (百分比)
    int battery_level;   // 電池電量 (百分比)
    time_t last_update;  // 最後更新時間
} sensor_state_t;

// 信號處理函式 - 優雅退出
void signal_handler(int signal) {
    printf("\n收到信號 %d，正在停止感測器...\n", signal);
    running = 0;
}

// 模擬溫度讀取 (帶有隨機變化)
float read_temperature() {
    static float base_temp = 25.0f;  // 基礎溫度
    static int first_call = 1;
    
    if (first_call) {
        srand((unsigned int)time(NULL));  // 初始化隨機數種子
        first_call = 0;
    }
    
    // 模擬溫度變化 ±2度
    float variation = ((float)rand() / RAND_MAX - 0.5f) * 4.0f;
    return base_temp + variation;
}

// 模擬濕度讀取
float read_humidity() {
    static float base_humidity = 60.0f;  // 基礎濕度
    
    // 模擬濕度變化 ±10%
    float variation = ((float)rand() / RAND_MAX - 0.5f) * 20.0f;
    float humidity = base_humidity + variation;
    
    // 確保濕度在合理範圍內 (0-100%)
    if (humidity < 0) humidity = 0;
    if (humidity > 100) humidity = 100;
    
    return humidity;
}

// 模擬電池電量讀取 (逐漸下降)
int read_battery_level() {
    static int battery = 100;
    static int call_count = 0;
    
    call_count++;
    // 每10次讀取下降1%
    if (call_count % 10 == 0 && battery > 0) {
        battery--;
    }
    
    return battery;
}

// 更新感測器狀態
void update_sensor_state(sensor_state_t* state) {
    state->temperature = read_temperature();
    state->humidity = read_humidity();
    state->battery_level = read_battery_level();
    state->last_update = time(NULL);
}

// 發布感測器遙測資料
int publish_telemetry(rtk_mqtt_client_t* client, const sensor_state_t* state) {
    int success_count = 0;
    
    // 發布溫度遙測
    if (rtk_mqtt_client_publish_telemetry(client, "temperature", 
                                         state->temperature, "°C") == RTK_SUCCESS) {
        printf("  ✓ 溫度: %.1f°C\n", state->temperature);
        success_count++;
    } else {
        printf("  ❌ 溫度發布失敗\n");
    }
    
    // 發布濕度遙測
    if (rtk_mqtt_client_publish_telemetry(client, "humidity", 
                                         state->humidity, "%") == RTK_SUCCESS) {
        printf("  ✓ 濕度: %.1f%%\n", state->humidity);
        success_count++;
    } else {
        printf("  ❌ 濕度發布失敗\n");
    }
    
    // 發布電池電量
    if (rtk_mqtt_client_publish_telemetry(client, "battery", 
                                         state->battery_level, "%") == RTK_SUCCESS) {
        printf("  ✓ 電池: %d%%\n", state->battery_level);
        success_count++;
    } else {
        printf("  ❌ 電池電量發布失敗\n");
    }
    
    return success_count;
}

// 發布設備狀態
int publish_device_state(rtk_mqtt_client_t* client, const sensor_state_t* state) {
    const char* status = "online";
    const char* health = "healthy";
    
    // 檢查電池電量，如果過低則報告不健康
    if (state->battery_level < 20) {
        health = "warning";  // 電池電量低
    }
    if (state->battery_level < 5) {
        health = "critical"; // 電池電量極低
    }
    
    if (rtk_mqtt_client_publish_state(client, status, health) == RTK_SUCCESS) {
        printf("  ✓ 設備狀態: %s (%s)\n", status, health);
        return 1;
    } else {
        printf("  ❌ 設備狀態發布失敗\n");
        return 0;
    }
}

int main() {
    printf("RTK MQTT Framework 基本感測器範例\n");
    printf("================================\n");
    printf("按 Ctrl+C 停止程式\n\n");
    
    // 設置信號處理
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    
    // 初始化感測器狀態
    sensor_state_t sensor_state = {0};
    
    // 創建 MQTT 客戶端
    rtk_mqtt_client_t* client = rtk_mqtt_client_create(
        "test.mosquitto.org",   // broker 主機
        1883,                   // broker 埠
        "basic_sensor_001"      // 客戶端 ID
    );
    
    if (client == NULL) {
        printf("❌ 無法創建 MQTT 客戶端\n");
        return -1;
    }
    
    // 連接到 MQTT broker
    printf("正在連接到 MQTT broker...\n");
    if (rtk_mqtt_client_connect(client) != RTK_SUCCESS) {
        printf("❌ 無法連接到 MQTT broker\n");
        rtk_mqtt_client_destroy(client);
        return -1;
    }
    printf("✓ 成功連接到 test.mosquitto.org:1883\n\n");
    
    // 主感測器迴圈
    int cycle_count = 0;
    while (running) {
        cycle_count++;
        printf("=== 感測器週期 #%d ===\n", cycle_count);
        
        // 更新感測器讀數
        update_sensor_state(&sensor_state);
        
        // 發布遙測資料
        printf("發布遙測資料:\n");
        int telemetry_success = publish_telemetry(client, &sensor_state);
        
        // 發布設備狀態
        printf("發布設備狀態:\n");
        int state_success = publish_device_state(client, &sensor_state);
        
        // 顯示本週期總結
        printf("本週期發布成功: %d/4 條訊息\n", telemetry_success + state_success);
        
        // 檢查電池電量警告
        if (sensor_state.battery_level < 20) {
            printf("⚠️  警告: 電池電量低 (%d%%)\n", sensor_state.battery_level);
        }
        if (sensor_state.battery_level <= 0) {
            printf("🔋 電池耗盡，感測器將停止運作\n");
            break;
        }
        
        printf("\n下次更新將在 10 秒後...\n\n");
        
        // 等待 10 秒，但每秒檢查一次是否需要退出
        for (int i = 0; i < 10 && running; i++) {
            sleep(1);
        }
    }
    
    // 清理資源
    printf("正在關閉感測器...\n");
    rtk_mqtt_client_disconnect(client);
    rtk_mqtt_client_destroy(client);
    printf("✓ 感測器已安全關閉\n");
    
    printf("\n📊 感測器運行總結:\n");
    printf("   - 運行週期: %d\n", cycle_count);
    printf("   - 最終電池電量: %d%%\n", sensor_state.battery_level);
    printf("   - 最後溫度: %.1f°C\n", sensor_state.temperature);
    printf("   - 最後濕度: %.1f%%\n", sensor_state.humidity);
    
    return 0;
}
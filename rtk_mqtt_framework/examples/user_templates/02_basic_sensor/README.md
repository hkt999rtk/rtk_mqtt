# 基本感測器範例

這個範例展示了如何建立一個持續運行的感測器，模擬溫度、濕度和電池電量監控，並週期性地將資料發布到 MQTT broker。

## 🎯 學習目標

- 了解週期性遙測資料發布
- 學習 RTK 主題結構的使用
- 掌握感測器資料的 JSON 格式化
- 熟悉信號處理和優雅退出
- 理解基本的錯誤處理機制

## 📋 功能特色

### 感測器模擬
- **溫度感測器**: 基準 25°C，隨機變化 ±2°C
- **濕度感測器**: 基準 60%，隨機變化 ±10%
- **電池監控**: 從 100% 開始逐漸下降

### MQTT 功能
- 週期性遙測資料發布 (每10秒)
- 設備狀態監控和回報
- 電池電量警告機制
- 優雅的連接和斷線處理

### 進階特性
- 信號處理 (Ctrl+C 優雅退出)
- 建構系統支援除錯和發布模式
- 背景執行模式
- 自動化測試功能

## 🚀 快速開始

### 1. 編譯範例

```bash
# 基本編譯
make

# 除錯模式編譯
make debug

# 發布模式編譯 (優化)
make release
```

### 2. 執行範例

```bash
# 正常執行 (會持續運行，按 Ctrl+C 停止)
make run

# 短時間測試 (10秒自動停止)
make run-short

# 背景執行
make run-continuous
make status    # 檢查狀態
make stop      # 停止背景程式
```

### 3. 預期輸出

```
RTK MQTT Framework 基本感測器範例
================================
按 Ctrl+C 停止程式

正在連接到 MQTT broker...
✓ 成功連接到 test.mosquitto.org:1883

=== 感測器週期 #1 ===
發布遙測資料:
  ✓ 溫度: 24.3°C
  ✓ 濕度: 65.2%
  ✓ 電池: 100%
發布設備狀態:
  ✓ 設備狀態: online (healthy)
本週期發布成功: 4/4 條訊息

下次更新將在 10 秒後...

=== 感測器週期 #2 ===
...
```

## 📖 代碼深入解析

### 核心結構

```c
// 感測器狀態結構
typedef struct {
    float temperature;    // 溫度 (攝氏度)
    float humidity;      // 濕度 (百分比)
    int battery_level;   // 電池電量 (百分比)
    time_t last_update;  // 最後更新時間
} sensor_state_t;
```

### 主要功能函式

#### 1. 感測器讀取模擬
```c
float read_temperature() {
    static float base_temp = 25.0f;
    float variation = ((float)rand() / RAND_MAX - 0.5f) * 4.0f;
    return base_temp + variation;
}
```

#### 2. 遙測資料發布
```c
int publish_telemetry(rtk_mqtt_client_t* client, const sensor_state_t* state) {
    // 發布溫度、濕度、電池電量
    rtk_mqtt_client_publish_telemetry(client, "temperature", state->temperature, "°C");
    // ...
}
```

#### 3. 信號處理
```c
void signal_handler(int signal) {
    printf("\n收到信號 %d，正在停止感測器...\n", signal);
    running = 0;  // 設置停止標記
}
```

### RTK 主題結構

此範例使用的 MQTT 主題格式：

```
# 遙測資料
rtk/v1/demo_tenant/demo_site/basic_sensor_001/telemetry/temperature
rtk/v1/demo_tenant/demo_site/basic_sensor_001/telemetry/humidity
rtk/v1/demo_tenant/demo_site/basic_sensor_001/telemetry/battery

# 設備狀態
rtk/v1/demo_tenant/demo_site/basic_sensor_001/state
```

### 訊息格式範例

#### 溫度遙測訊息
```json
{
  "metric": "temperature",
  "value": 24.3,
  "unit": "°C",
  "timestamp": 1692123456,
  "device_id": "basic_sensor_001"
}
```

#### 設備狀態訊息
```json
{
  "status": "online",
  "health": "healthy",
  "uptime": 3600,
  "last_seen": 1692123456,
  "battery_level": 85
}
```

## 🔧 進階設定

### 自訂感測器參數

修改 `sensor.c` 中的感測器模擬參數：

```c
// 溫度範圍調整
static float base_temp = 30.0f;  // 改變基礎溫度
float variation = ((float)rand() / RAND_MAX - 0.5f) * 6.0f;  // 增加變化範圍

// 發布頻率調整
sleep(30);  // 改為30秒發布一次
```

### 修改 MQTT 設定

```c
// 連接到您自己的 broker
rtk_mqtt_client_t* client = rtk_mqtt_client_create(
    "your.mqtt.broker.com",  // 您的 broker 位址
    1883,                    // 埠號
    "your_sensor_001"        // 唯一的設備 ID
);
```

### 添加新的遙測指標

```c
// 在 publish_telemetry 函式中添加
if (rtk_mqtt_client_publish_telemetry(client, "pressure", 
                                     read_pressure(), "hPa") == RTK_SUCCESS) {
    printf("  ✓ 氣壓: %.1f hPa\n", read_pressure());
    success_count++;
}
```

## 🧪 測試和驗證

### 自動化測試

```bash
# 執行內建測試
make test

# 檢查編譯環境
make check-env

# 檢查 RTK 框架可用性
make check-rtk
```

### 手動驗證

#### 1. 使用 MQTT 客戶端監聽

```bash
# 監聽所有遙測資料
mosquitto_sub -h test.mosquitto.org -t "rtk/v1/+/+/basic_sensor_001/telemetry/+"

# 監聽設備狀態
mosquitto_sub -h test.mosquitto.org -t "rtk/v1/+/+/basic_sensor_001/state"
```

#### 2. 使用網頁 MQTT 客戶端

訪問 [HiveMQ Websocket Client](http://www.hivemq.com/demos/websocket-client/) 並：

1. 連接到 `test.mosquitto.org:8000`
2. 訂閱主題：`rtk/v1/+/+/basic_sensor_001/telemetry/+`
3. 執行感測器範例
4. 觀察即時資料

#### 3. 監控系統資源

```bash
# 檢查記憶體使用
ps aux | grep basic_sensor

# 檢查 CPU 使用率
top -p $(pgrep basic_sensor)

# 檢查網路連接
netstat -an | grep 1883
```

## 🐛 除錯和故障排除

### 啟用除錯模式

```bash
# 編譯除錯版本
make DEBUG=1

# 或直接執行除錯版本
make debug run
```

### 常見問題解決

#### Q: 感測器無法連接到 broker

**A**: 
1. 檢查網路連接：`ping test.mosquitto.org`
2. 檢查防火牆設定：`telnet test.mosquitto.org 1883`
3. 嘗試其他 broker：修改 `sensor.c` 中的 broker 位址

#### Q: 遙測資料發布失敗

**A**:
1. 檢查 broker 連接狀態
2. 確認主題格式正確
3. 檢查 QoS 設定和 broker 限制

#### Q: 程式意外退出

**A**:
1. 檢查除錯日誌：`make DEBUG=1 run`
2. 檢查記憶體問題：使用 `valgrind ./basic_sensor`
3. 檢查信號處理：確認是否收到意外信號

#### Q: 背景程式無法停止

**A**:
```bash
# 查找程序
ps aux | grep basic_sensor

# 強制終止
pkill -f basic_sensor

# 或使用 kill
kill $(pgrep basic_sensor)
```

### 除錯技巧

#### 1. 添加詳細日誌

```c
#ifdef DEBUG
    printf("[DEBUG] 溫度讀取: %.2f°C\n", temperature);
    printf("[DEBUG] MQTT 發布狀態: %d\n", publish_result);
#endif
```

#### 2. 記憶體洩漏檢查

```bash
# 使用 Valgrind 檢查記憶體洩漏
valgrind --leak-check=full --show-leak-kinds=all ./basic_sensor
```

#### 3. 網路連接診斷

```bash
# 檢查 DNS 解析
nslookup test.mosquitto.org

# 檢查連接埠可達性
nc -zv test.mosquitto.org 1883
```

## 📈 效能和最佳化

### 記憶體使用最佳化

```c
// 減少不必要的字串操作
static char topic_buffer[256];  // 重複使用緩衝區

// 避免頻繁的記憶體分配
static sensor_state_t static_state;  // 使用靜態變數
```

### 網路效能調校

```c
// 調整 MQTT 參數
rtk_mqtt_config_t config = {
    .keepalive = 30,        // 減少心跳頻率
    .qos = 0,              // 使用 QoS 0 提高效能
    .clean_session = 1,     // 清除會話減少伺服器負載
};
```

### CPU 使用最佳化

```c
// 減少浮點運算
int temperature_int = (int)(temperature * 10);  // 改為整數運算

// 批次發布減少網路調用
publish_batch_telemetry(client, &sensor_state);
```

## 🔄 下一步學習

完成基本感測器範例後，建議您：

1. **查看 03_complete_device** - 學習生產級設備實作
2. **實驗感測器類型** - 添加更多感測器模擬
3. **研究 QoS 設定** - 了解 MQTT 可靠性選項
4. **學習配置管理** - 使用 JSON 配置檔案
5. **探索插件架構** - 建立可重複使用的感測器組件

## 📚 相關資源

- **[MANUAL.md](../../../docs/MANUAL.md)** - 完整使用手冊  
- **[01_hello_world](../01_hello_world/)** - 前一個學習範例
- **[03_complete_device](../03_complete_device/)** - 下一個學習範例
- **[RTK MQTT 協議規範](../../../docs/SPEC.md)** - 協議詳細說明
# Hello World 範例

這是 RTK MQTT Framework 最簡單的入門範例，只需要 20 行代碼就能完成一個基本的 MQTT 客戶端。

## 🎯 學習目標

- 了解 RTK MQTT 框架的基本使用流程
- 學習如何連接到 MQTT broker
- 掌握發布簡單訊息的方法
- 熟悉資源管理和清理

## 📋 前置需求

- C 編譯器 (GCC, Clang, 或 Visual Studio)
- RTK MQTT Framework 發行包
- 網路連接 (用於連接測試 broker)

## 🚀 快速開始

### 1. 編譯範例

```bash
# 方法1：使用我們提供的 Makefile
make

# 方法2：手動編譯
gcc -std=c99 -Wall -Wextra \
    -I../../include/rtk_mqtt_framework \
    -o hello_rtk_mqtt main.c \
    -L../../lib -lrtk_mqtt_framework -lpthread -lm
```

### 2. 執行範例

```bash
./hello_rtk_mqtt
```

### 3. 預期輸出

```
RTK MQTT Framework Hello World 範例
===================================
正在連接到 MQTT broker...
✓ 成功連接到 test.mosquitto.org:1883
正在發布 Hello World 訊息...
✓ Hello World 訊息發布成功！
✓ 資源清理完成

🎉 Hello World 範例執行完成！
```

## 📖 代碼解析

### 主要步驟

```c
// 1. 創建 MQTT 客戶端
rtk_mqtt_client_t* client = rtk_mqtt_client_create("test.mosquitto.org", 1883, "hello_world_device");

// 2. 連接到 broker
rtk_mqtt_client_connect(client);

// 3. 發布訊息
rtk_mqtt_client_publish_state(client, "online", "healthy");

// 4. 清理資源
rtk_mqtt_client_disconnect(client);
rtk_mqtt_client_destroy(client);
```

### 關鍵概念

- **MQTT 客戶端**: 負責與 MQTT broker 通訊的核心組件
- **Broker 連接**: 使用公共測試 broker `test.mosquitto.org:1883`
- **狀態發布**: 使用 RTK 標準格式發布設備狀態
- **資源管理**: 確保正確清理記憶體和網路連接

## 🔧 自訂設定

### 修改 Broker 設定

如果您想連接到自己的 MQTT broker，請修改 `main.c` 中的連接參數：

```c
// 將以下行
rtk_mqtt_client_t* client = rtk_mqtt_client_create("test.mosquitto.org", 1883, "hello_world_device");

// 修改為您的 broker 設定
rtk_mqtt_client_t* client = rtk_mqtt_client_create("your.mqtt.broker.com", 1883, "your_device_id");
```

### 修改設備 ID

設備 ID 是您設備的唯一識別符，建議使用有意義的名稱：

```c
// 範例：
rtk_mqtt_client_create("broker.com", 1883, "sensor_001");          // 感測器
rtk_mqtt_client_create("broker.com", 1883, "gateway_office_1");    // 閘道器
rtk_mqtt_client_create("broker.com", 1883, "thermostat_room_a");   // 恆溫器
```

## 🧪 測試驗證

### 使用 MQTT 客戶端監聽

您可以使用任何 MQTT 客戶端工具來驗證訊息是否成功發布：

```bash
# 使用 mosquitto_sub 監聽 (如果已安裝)
mosquitto_sub -h test.mosquitto.org -t "rtk/v1/+/+/hello_world_device/state"

# 或使用 RTK 框架提供的測試工具
../../bin/rtk_cli subscribe -h test.mosquitto.org -t "rtk/v1/+/+/+/state"
```

### 使用網頁 MQTT 客戶端

訪問 [HiveMQ Websocket Client](http://www.hivemq.com/demos/websocket-client/) 並：

1. 連接到 `test.mosquitto.org:8000`
2. 訂閱主題：`rtk/v1/+/+/hello_world_device/state`
3. 執行您的範例程式
4. 查看收到的訊息

## 🔄 下一步

完成 Hello World 範例後，建議您：

1. **查看 02_basic_sensor 範例** - 學習週期性資料發布
2. **修改代碼** - 嘗試發布不同類型的訊息
3. **實驗設定** - 連接到不同的 MQTT broker
4. **閱讀 docs/MANUAL.md** - 了解更進階的功能

## ❓ 常見問題

### Q: 編譯時找不到標頭檔

**A**: 檢查 Makefile 中的 `RTK_INCLUDE_DIR` 路徑設定是否正確。

### Q: 連接 broker 失敗

**A**: 
1. 檢查網路連接
2. 確認 broker 位址和埠號正確
3. 嘗試使用其他公共 broker（如 `broker.hivemq.com:1883`）

### Q: 程式執行後沒有輸出

**A**: 
1. 確認編譯成功
2. 檢查執行檔權限：`chmod +x hello_rtk_mqtt`
3. 在除錯模式下執行：添加 `-DRTK_DEBUG` 編譯選項

### Q: 如何在 Windows 上編譯？

**A**: 
```bash
# 使用 MinGW
gcc -std=c99 -I../../include/rtk_mqtt_framework ^
    -o hello_rtk_mqtt.exe main.c ^
    -L../../lib -lrtk_mqtt_framework -lws2_32

# 或使用 Visual Studio
cl /I"..\..\include\rtk_mqtt_framework" main.c ^
   /link ..\..\lib\rtk_mqtt_framework.lib ws2_32.lib
```

## 🐛 故障排除

### 常見編譯問題

**Q: 找不到 RTK 標頭檔**
```
error: rtk_mqtt_client.h: No such file or directory
```
**A**: 檢查 RTK 框架路徑設定
```bash
# 檢查檔案是否存在
ls -la ../../include/rtk_mqtt_framework/rtk_mqtt_client.h

# 如果路徑錯誤，修改 Makefile 中的 RTK_INCLUDE_DIR
nano Makefile
```

**Q: 連結時找不到函式庫**
```
undefined reference to `rtk_mqtt_client_create'
```
**A**: 檢查函式庫路徑
```bash
# 檢查函式庫檔案
ls -la ../../lib/librtk_mqtt_framework.a

# 確認 Makefile 中的 RTK_LIB_DIR 設定正確
```

### 執行時問題

**Q: 無法連接到 MQTT broker**
```
錯誤: 連接失敗
```
**A**: 逐步排除網路問題
```bash
# 1. 測試網路連接
ping test.mosquitto.org

# 2. 測試 MQTT 埠號
telnet test.mosquitto.org 1883

# 3. 嘗試其他 broker
# 修改代碼中的 broker 位址為 broker.hivemq.com
```

**Q: 程式執行後沒有輸出**

**A**: 啟用除錯模式
```bash
# 重新編譯並加入除錯資訊
make clean
make DEBUG=1

# 執行除錯版本
./hello_rtk_mqtt
```

## 📚 相關資源

- **[MANUAL.md](../../../docs/MANUAL.md)** - 完整使用手冊
- **[02_basic_sensor](../02_basic_sensor/)** - 下一個學習範例
- **[RTK MQTT 協議規範](../../../docs/SPEC.md)** - 協議詳細說明
# 完整設備範例

這是一個生產級的 IoT 設備實作範例，展示企業級的建構系統、多執行緒架構、完整的日誌系統、配置管理、健康監控、和命令處理功能。

## 🎯 學習目標

- 掌握生產級設備實作模式
- 學習多執行緒架構設計  
- 理解企業級建構系統配置
- 熟悉配置檔案管理
- 掌握健康監控和診斷機制
- 學習命令接收和處理流程
- 理解優雅啟動和關閉程序

## 🏗️ 架構特色

### 多執行緒設計
- **感測器執行緒**: 週期性收集和發布遙測資料
- **命令處理執行緒**: 處理遠端命令和配置更新
- **健康監控執行緒**: 監控設備健康狀態和 MQTT 連線

### 企業級功能
- 完整的日誌記錄系統（控制台 + 檔案）
- JSON 配置檔案管理
- 版本資訊和建構追溯
- 優雅的信號處理和關閉程序
- 健康狀態評估和警報

### 生產級建構系統
- 多模式建構（debug/release/profile/static）
- 安全性編譯選項
- 靜態分析和記憶體檢查
- 自動化測試和品質保證
- 分發包生成和安裝支援

## 🚀 快速開始

### 1. 建立配置檔案

```bash
# 複製範例配置檔案
cp config.json.example config.json

# 編輯配置檔案以符合您的環境
nano config.json
```

### 2. 編譯和執行

```bash
# 基本建構
make

# 除錯模式建構
make debug

# 發布模式建構（高度最佳化）
make release

# 直接執行
make run

# 使用配置檔案執行
make run-config
```

### 3. 背景執行模式

```bash
# 啟動背景服務
make run-daemon

# 檢查服務狀態
make status

# 查看即時日誌
tail -f device.log

# 停止背景服務
make stop
```

## 📋 配置檔案說明

### 基本設備配置
```json
{
  "device": {
    "id": "complete_device_001",
    "type": "industrial_iot",
    "location": "factory_floor_a",
    "firmware_version": "2.1.0"
  }
}
```

### MQTT 連線配置
```json
{
  "mqtt": {
    "broker_host": "test.mosquitto.org",
    "broker_port": 1883,
    "username": "",
    "password": "",
    "keepalive": 60,
    "qos": 1,
    "reconnect_interval": 5,
    "max_reconnect_attempts": 10
  }
}
```

### 運行參數配置
```json
{
  "settings": {
    "publish_interval": 30,
    "health_check_interval": 60,
    "command_timeout": 10,
    "log_level": "INFO",
    "log_file": "device.log"
  }
}
```

## 🔧 進階建構選項

### 建構變體

```bash
# 除錯模式（包含 AddressSanitizer）
make debug run

# 效能分析模式（包含 gprof 支援）
make profile run

# 靜態連結模式
make static

# 自訂建構類型
make BUILD_TYPE=custom DEBUG=1 PROFILE=1
```

### 品質保證工具

```bash
# 靜態分析
make analyze

# 記憶體洩漏檢查
make memory-check

# 代碼風格檢查
make lint

# 自動格式化
make format

# 綜合測試
make test
```

### 分發和部署

```bash
# 建立分發包
make package

# 系統安裝
sudo make install

# 系統移除
sudo make uninstall
```

## 🧵 多執行緒架構

### 執行緒同步機制

```c
// 指標保護
pthread_mutex_t metrics_mutex;

// 關閉同步
pthread_cond_t shutdown_cond;
pthread_mutex_t shutdown_mutex;

// 原子狀態標記
volatile int running;
volatile int connected;
volatile int health_status;
```

### 感測器執行緒
- 週期性收集 CPU、記憶體、溫度、網路品質指標
- 自動健康狀態評估
- 遙測資料格式化和發布
- 設備狀態維護

### 命令處理執行緒
- 訂閱和處理遠端命令
- 配置更新處理
- 命令執行結果回報
- 超時和錯誤處理

### 健康監控執行緒
- MQTT 連線狀態監控
- 自動重連機制
- 危險狀態檢測和警報
- 緊急事件發布

## 📊 遙測資料類型

### 系統指標
```json
{
  "metric": "cpu_usage",
  "value": 45.2,
  "unit": "%",
  "timestamp": 1692123456,
  "device_id": "complete_device_001"
}
```

### 設備狀態
```json
{
  "status": "online",
  "health": "healthy",
  "uptime": 7200,
  "last_seen": 1692123456,
  "cpu_usage": 45.2,
  "memory_usage": 38.7,
  "temperature": 42.3,
  "network_quality": 85
}
```

### 事件通知
```json
{
  "event_type": "device.health.critical",
  "severity": "critical",
  "message": "設備健康狀態危險",
  "timestamp": 1692123456,
  "device_id": "complete_device_001",
  "additional_data": {
    "cpu_usage": 95.2,
    "memory_usage": 89.1,
    "temperature": 71.5
  }
}
```

## 🛡️ 健康監控系統

### 健康狀態分級
- **健康 (Healthy)**: 所有指標正常
- **警告 (Warning)**: CPU > 75% 或記憶體 > 75% 或溫度 > 50°C
- **危險 (Critical)**: CPU > 90% 或記憶體 > 90% 或溫度 > 70°C

### 自動恢復機制
```c
// MQTT 重連邏輯
for (int i = 0; i < max_reconnect_attempts && running; i++) {
    if (rtk_mqtt_client_reconnect(client) == RTK_SUCCESS) {
        device->connected = 1;
        device->reconnect_count++;
        break;
    }
    sleep(reconnect_interval);
}
```

### 警報機制
- 危險狀態自動發布緊急事件
- 連線斷開時觸發重連流程
- 健康狀態變化時更新設備狀態

## 📝 日誌系統

### 日誌等級
- **DEBUG**: 詳細除錯資訊
- **INFO**: 一般操作資訊
- **WARNING**: 警告訊息
- **ERROR**: 錯誤訊息
- **CRITICAL**: 致命錯誤

### 日誌輸出範例
```
[2024-08-18 14:30:45] INFO: 正在初始化完整設備...
[2024-08-18 14:30:45] INFO: 配置載入完成
[2024-08-18 14:30:46] INFO: 正在連接到 MQTT broker test.mosquitto.org:1883...
[2024-08-18 14:30:46] INFO: MQTT 連接成功
[2024-08-18 14:30:46] INFO: 感測器執行緒啟動
[2024-08-18 14:30:46] INFO: 命令處理執行緒啟動
[2024-08-18 14:30:46] INFO: 健康監控執行緒啟動
[2024-08-18 14:30:46] INFO: 所有執行緒已啟動，設備正常運行
[2024-08-18 14:31:16] DEBUG: 感測器週期完成 (CPU: 42.1%, 記憶體: 38.5%, 溫度: 36.2°C)
[2024-08-18 14:31:16] DEBUG: 遙測資料發布: 5/5 成功
```

## 🧪 測試和驗證

### 自動化測試套件

```bash
# 完整測試流程
make test

# 測試項目包含：
# 1. 可執行檔檢查
# 2. 依賴庫檢查  
# 3. 配置檔案檢查
# 4. 短時間執行測試
```

### 手動測試流程

#### 1. 基本功能測試
```bash
# 啟動設備
make run-daemon

# 檢查程序狀態
make status

# 監控日誌
tail -f device.log

# 停止設備
make stop
```

#### 2. MQTT 通信驗證
```bash
# 監聽遙測資料
mosquitto_sub -h test.mosquitto.org -t "rtk/v1/+/+/complete_device_001/telemetry/+"

# 監聽設備狀態
mosquitto_sub -h test.mosquitto.org -t "rtk/v1/+/+/complete_device_001/state"

# 監聽事件通知
mosquitto_sub -h test.mosquitto.org -t "rtk/v1/+/+/complete_device_001/evt/+"
```

#### 3. 健康監控測試
```bash
# 模擬高負載狀態（在其他終端）
stress --cpu 4 --timeout 60

# 觀察設備狀態變化
tail -f device.log | grep "健康狀態"
```

## 🐛 故障排除

### 常見問題與解決方案

#### Q: 設備無法連接到 MQTT broker
**A**: 
1. 檢查網路連接：`ping test.mosquitto.org`
2. 檢查配置檔案中的 broker 設定
3. 檢查防火牆和代理設定
4. 嘗試除錯模式：`make debug run`

#### Q: 多執行緒同步問題
**A**:
1. 檢查死鎖情況：`make memory-check`
2. 使用除錯模式排查：`make DEBUG=1`
3. 檢查執行緒生命週期日誌

#### Q: 記憶體使用持續增長
**A**:
```bash
# 記憶體洩漏檢查
make memory-check

# 長時間監控
watch -n 5 'ps aux | grep complete_device'
```

#### Q: 健康監控誤報
**A**:
1. 調整健康狀態閾值（修改源碼）
2. 檢查系統資源實際使用情況
3. 確認感測器讀取邏輯正確性

### 除錯技巧

#### 1. 啟用詳細日誌
```bash
# 編譯除錯版本
make DEBUG=1

# 或直接在配置檔案中設置
{
  "logging": {
    "level": "DEBUG"
  }
}
```

#### 2. 使用系統工具監控
```bash
# 監控程序資源使用
top -p $(pgrep complete_device)

# 檢查執行緒狀態
ps -T -p $(pgrep complete_device)

# 監控網路連接
netstat -an | grep 1883
```

#### 3. 分析執行緒狀態
```bash
# 檢查執行緒數量
cat /proc/$(pgrep complete_device)/status | grep Threads

# 檢查執行緒堆疊
cat /proc/$(pgrep complete_device)/task/*/stack
```

## ⚡ 效能最佳化

### 系統資源最佳化

```c
// 減少記憶體佔用
#define MAX_TOPIC_LENGTH 256
static char topic_buffer[MAX_TOPIC_LENGTH];  // 重複使用

// 最佳化發布頻率
int adaptive_publish_interval = calculate_optimal_interval();
```

### 網路效能調校

```json
{
  "mqtt": {
    "keepalive": 30,
    "qos": 0,
    "clean_session": true,
    "reconnect_interval": 3
  }
}
```

### CPU 使用最佳化

```c
// 減少系統調用
static struct timespec sleep_time = {30, 0};  // 30 秒
nanosleep(&sleep_time, NULL);

// 批次處理遙測資料
publish_batch_telemetry(client, metrics_array, count);
```

## 📈 監控和運維

### 系統監控指標

```bash
# CPU 和記憶體監控
watch -n 10 'ps aux | grep complete_device | head -1'

# 網路流量監控
iftop -i eth0 -B -P

# 日誌檔案大小監控
watch -n 60 'ls -lh device.log'
```

### 運維腳本範例

```bash
#!/bin/bash
# 設備健康檢查腳本
if ! pgrep -f complete_device > /dev/null; then
    echo "設備程序已停止，重新啟動..."
    cd /path/to/device && make run-daemon
fi

# 檢查日誌檔案大小
if [ $(stat -c%s device.log) -gt 10485760 ]; then  # 10MB
    echo "日誌檔案過大，進行輪換..."
    mv device.log device.log.old
fi
```

## 🔄 下一步學習

完成完整設備範例後，建議您：

1. **查看 04_cross_platform** - 學習跨平台建構
2. **研究插件系統** - 建立可重複使用的組件
3. **探索進階 MQTT 功能** - QoS、保留消息、遺囑消息
4. **學習容器化部署** - Docker 和 Kubernetes 整合
5. **研究設備管理平台** - 與雲端設備管理系統整合

## 📚 相關資源

- **[MANUAL.md](../../../docs/MANUAL.md)** - 完整使用手冊
- **[02_basic_sensor](../02_basic_sensor/)** - 前一個學習範例
- **[04_cross_platform](../04_cross_platform/)** - 下一個學習範例
- **[RTK MQTT 協議規範](../../../docs/SPEC.md)** - 協議詳細說明
- **[建構系統指南](../../../README.md)** - 開發環境設置

## 🏆 企業級特色總結

這個範例展示了生產環境所需的關鍵特性：

✅ **多執行緒架構** - 可靠的並行處理  
✅ **完整的錯誤處理** - 豐富的恢復機制  
✅ **企業級建構系統** - 專業的開發工具鏈  
✅ **配置檔案管理** - 靈活的部署配置  
✅ **健康監控系統** - 主動的狀態管理  
✅ **日誌記錄系統** - 完整的操作追蹤  
✅ **優雅關閉機制** - 安全的程序終止  
✅ **自動化測試** - 品質保證流程  
✅ **分發包支援** - 標準化的部署包  
✅ **運維友善設計** - 便於監控和維護  

通過這個範例，您將掌握構建工業級 IoT 設備所需的完整技能集。
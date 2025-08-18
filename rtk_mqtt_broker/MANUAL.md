# RTK MQTT Broker - 工程師快速使用手冊

> **目標讀者**: 拿到 binary 檔案需要快速部署和測試的工程師  
> **本手冊**: 包含完整的部署、測試和故障排除指南  
> **測試工具**: 提供預編譯的測試工具，無需 Go 環境即可執行測試

## 🚀 30秒快速啟動

### 1. 選擇可執行檔案
```bash
# Linux ARM64
./bin/rtk_mqtt_broker-linux-arm64

# Linux x86_64 
./bin/rtk_mqtt_broker-linux-amd64

# macOS ARM64
./bin/rtk_mqtt_broker-darwin-arm64

# Windows x86_64
bin\rtk_mqtt_broker-windows-amd64.exe
```

### 2. 啟動檢查
看到以下輸出代表啟動成功：
```
[RTK-MQTT] [INFO] Realtek Embedded MQTT Broker started successfully
[RTK-MQTT] [INFO] Listening on interface: 0.0.0.0
[RTK-MQTT] [INFO] Listening on port: 1883
```

## ✅ 功能驗證測試

### 測試 1: 基本連接測試
```bash
# 使用預編譯的測試工具 (無需 Go 環境)
./test-tools/mqtt_client

# 或從 test 目錄執行原始碼 (需要 Go 環境)
cd test/ && go run mqtt_client.go
```
**期望結果**: 顯示連接成功、發布/訂閱測試通過

### 測試 2: Topic 監聽測試
```bash
# 開啟監聽器 (終端視窗 1) - 使用預編譯工具
./test-tools/topic_listener -topic "test/+" -verbose

# 發布測試訊息 (終端視窗 2) - 使用預編譯工具
./test-tools/simple_publisher

# 或使用原始碼 (需要 Go 環境)
# 終端 1: cd test/ && go run topic_listener.go -topic "test/+" -verbose
# 終端 2: cd test/ && go run simple_publisher.go
```
**期望結果**: 監聽器收到發布的訊息

### 測試 3: 外部工具測試
```bash
# 安裝 mosquitto 客戶端 (如果沒有)
# Ubuntu/Debian: sudo apt-get install mosquitto-clients
# macOS: brew install mosquitto
# Windows: 下載 mosquitto 客戶端

# 訂閱測試 (終端視窗 1)
mosquitto_sub -h localhost -p 1883 -t "test/topic"

# 發布測試 (終端視窗 2)
mosquitto_pub -h localhost -p 1883 -t "test/topic" -m "Hello from engineer!"
```
**期望結果**: 訂閱端收到 "Hello from engineer!" 訊息

## 🔧 常用配置修改

### 修改連接埠
編輯 `config/config.yaml`:
```yaml
server:
  port: 8883  # 改為你需要的埠號
```

### 啟用認證 (生產環境建議)
```yaml
security:
  enable_auth: true
  users:
    - username: "device001"
      password: "secure_pass_123"
```

### 高並發配置
```yaml
server:
  max_clients: 5000
  enable_stats: false  # 關閉統計提升效能
logging:
  level: "warn"       # 減少日誌輸出
```


## 📊 監控和維護

### 查看統計資訊
啟用統計後 (enable_stats: true)，每 30 秒顯示：
```
[RTK-MQTT] [INFO] Stats - Clients: 15, Messages Received: 2500, Messages Sent: 2500
```


### 效能監控腳本
```bash
#!/bin/bash
# monitor_mqtt.sh - 簡單的效能監控

while true; do
    echo "=== $(date) ==="
    echo "連接數: $(netstat -an | grep :1883 | grep ESTABLISHED | wc -l)"
    echo "記憶體使用: $(ps aux | grep rtk_mqtt_broker | grep -v grep | awk '{print $6}')KB"
    echo "CPU 使用: $(ps aux | grep rtk_mqtt_broker | grep -v grep | awk '{print $3}')%"
    echo ""
    sleep 30
done
```

## 🚨 常見問題快速排除

### 問題 1: 無法啟動 - 埠號被占用
```bash
# 檢查埠號占用
netstat -ln | grep 1883
lsof -i :1883

# 解決方案
# 1. 停止占用程序或
# 2. 修改 config.yaml 中的 port 設定
```

### 問題 2: 客戶端連接被拒絕
```bash
# 檢查清單:
# 1. Broker 是否正在運行? ps aux | grep rtk_mqtt_broker
# 2. 防火牆是否開放? sudo ufw allow 1883
# 3. 網路是否可達? ping BROKER_IP
# 4. 認證設定是否正確? 檢查 config.yaml
```

### 問題 3: 訊息遺失
```bash
# 檢查清單:
# 1. QoS 等級是否設定正確? (建議使用 QoS 1)
# 2. 網路是否穩定? ping BROKER_IP
# 3. Broker 資源是否充足? top | grep rtk_mqtt_broker
```

### 問題 4: 效能不佳
```bash
# 調整建議:
# 1. 增加 max_clients 限制
# 2. 關閉 enable_stats
# 3. 調整日誌等級為 warn 或 error
# 4. 檢查系統資源: htop
```

## 🧪 壓力測試

### 使用內建測試工具進行壓力測試
```bash
# 方法 1: 使用 topic_listener 監控 + simple_publisher 批次發送
# 終端 1: 啟動監聽器
./test-tools/topic_listener -topic "#" -verbose > stress_test_results.log &

# 終端 2: 批次執行 simple_publisher
for i in {1..10}; do
    echo "執行第 $i 輪測試..."
    ./test-tools/simple_publisher &
done
wait

# 檢查結果
cat stress_test_results.log | grep "📥" | wc -l
```

### 使用 mosquitto 客戶端的壓力測試腳本
```bash
#!/bin/bash
# stress_test.sh - 簡單的 MQTT 壓力測試

BROKER_IP="localhost"
BROKER_PORT="1883"
NUM_CLIENTS=100

echo "開始壓力測試: $NUM_CLIENTS 個客戶端"

for i in $(seq 1 $NUM_CLIENTS); do
    mosquitto_pub -h $BROKER_IP -p $BROKER_PORT \
        -t "stress/test/$i" \
        -m "Test message from client $i" \
        -i "client_$i" &
done

wait
echo "壓力測試完成"

# 檢查 Broker 統計
echo "檢查 Broker 日誌中的統計資訊"
```


## 🔗 有用的工具和資源

### MQTT 客戶端工具
- **命令列**: mosquitto-clients
- **圖形化**: MQTT.fx, MQTT Explorer
- **網頁版**: HiveMQ WebSocket Client
- **行動裝置**: IoT MQTT Panel (Android)

### 監控工具
```bash
# 即時監控連接數
watch "netstat -an | grep :1883 | grep ESTABLISHED | wc -l"

# 監控記憶體使用
watch "ps aux | grep rtk_mqtt_broker | grep -v grep"

# 網路流量監控
sudo iftop -i eth0 -P
```

---

**需要幫助?**
- 技術支援: support@realtek.com
- 問題回報: 提供詳細的錯誤日誌和環境資訊
- 記住: 90% 的問題都是配置或網路問題 😊
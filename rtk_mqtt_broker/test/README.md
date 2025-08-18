# RTK MQTT Broker 測試工具

這個目錄包含了用於測試 RTK MQTT Broker 的測試程序。這些工具可以編譯成二進制文件，無需在目標機器上安裝 Go 環境。

## 測試程序說明

### 1. mqtt_client
綜合測試程序，執行完整的 MQTT 功能測試包括：
- 基本連接測試
- 多主題訂閱和發布
- QoS 等級測試 (0, 1, 2)
- 訊息統計和結果驗證

**用法：**
```bash
./mqtt_client
```

### 2. simple_publisher
簡單的訊息發布工具，用於發布測試訊息到 sensor/ 主題。

**用法：**
```bash
./simple_publisher
```

### 3. topic_listener
靈活的主題監聽工具，支援多種配置選項。

**用法：**
```bash
./topic_listener [選項]

選項：
  -broker string     MQTT broker 地址 (預設: localhost)
  -port int         MQTT broker 埠號 (預設: 1883)
  -topic string     要監聽的主題，支援通配符 (預設: "test/+")
  -client string    客戶端 ID (預設: "topic_listener")
  -qos int         QoS 等級 0, 1, 2 (預設: 0)
  -duration int    監聽時間（秒，0=無限） (預設: 0)
  -verbose         顯示詳細訊息
```

**範例：**
```bash
# 監聽所有 sensor 主題，顯示詳細資訊
./topic_listener -topic "sensor/+" -verbose

# 監聽特定 broker 的所有主題，持續 60 秒
./topic_listener -broker "192.168.1.100" -topic "#" -duration 60

# 使用 QoS 1 監聽溫度感測器資料
./topic_listener -topic "sensor/temperature" -qos 1
```

## 編譯二進制文件

### 開發環境編譯（需要 Go）

編譯本機版本的測試工具：
```bash
make build-test-tools-dev
```
編譯後的文件位於 `build/test-tools/` 目錄。

### 跨平台編譯

編譯所有支援平台的測試工具：
```bash
make build-test-tools
```

打包測試工具到 tar.gz 文件：
```bash
make package-test-tools
```

支援的平台：
- `linux-arm64` - ARM64 Linux (樹莓派 4, ARM64 伺服器)
- `linux-amd64` - x86_64 Linux (大多數 Linux 伺服器)
- `darwin-arm64` - ARM64 macOS (Apple M1/M2 Mac)
- `windows-amd64` - x86_64 Windows

### 手動編譯單一程序

如果需要編譯特定平台的單一程序：

```bash
# Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o mqtt_client_linux ./mqtt_client.go

# Windows AMD64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o mqtt_client.exe ./mqtt_client.go

# ARM64 Linux (樹莓派)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o mqtt_client_arm64 ./mqtt_client.go
```

## 使用場景

### 1. 基本連接測試
使用 `mqtt_client` 驗證 broker 是否正常運行：
```bash
./mqtt_client
```

### 2. 持續監控
使用 `topic_listener` 監控生產環境的訊息流：
```bash
./topic_listener -topic "#" -verbose > mqtt_log.txt &
```

### 3. 效能測試
結合使用 `simple_publisher` 和 `topic_listener` 進行吞吐量測試：
```bash
# 終端 1：啟動監聽器
./topic_listener -topic "sensor/+"

# 終端 2：執行發布器
./simple_publisher
```

### 4. 網路診斷
使用 `topic_listener` 檢查特定設備的連線狀態：
```bash
./topic_listener -topic "device/+/status" -duration 300
```

## 故障排除

### 連接失敗
- 檢查 broker 是否在運行：`netstat -an | grep 1883`
- 確認防火牆設定允許 1883 埠
- 驗證 broker 配置檔案中的認證設定

### 訊息遺失
- 嘗試使用更高的 QoS 等級
- 檢查網路延遲和穩定性
- 確認 broker 的記憶體和儲存空間足夠

### 權限問題
- 確保二進制文件有執行權限：`chmod +x ./mqtt_client`
- 在 Linux 上，某些系統可能需要安裝 `ca-certificates` 套件

## 整合到 CI/CD

這些測試工具可以整合到持續整合流程中：

```bash
#!/bin/bash
# 在 CI 環境中執行基本測試
./mqtt_client > test_results.log 2>&1
if [ $? -eq 0 ]; then
    echo "MQTT 測試通過"
    exit 0
else
    echo "MQTT 測試失敗，查看 test_results.log"
    exit 1
fi
```
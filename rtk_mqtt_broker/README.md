# Realtek Embedded MQTT Broker

## 概述

Realtek Embedded MQTT Broker 是一個專為 RTK IoT 設備設計的高效能 MQTT 伺服器，使用 Go 語言和 Mochi MQTT 實現的輕量級 MQTT broker。本發行版本包含針對不同平台預編譯的可執行檔案，可直接部署使用。

## 🏷️ 版本資訊

- **產品名稱**: Realtek Embedded MQTT Broker
- **版本號**: v1.2.0
- **發布日期**: 2024-08-16
- **支援協定**: MQTT 3.1.1
- **預設連接埠**: 1883
- **開發商**: Realtek Semiconductor Corp.

## ✨ 主要特性

- 🚀 輕量化設計，最小化資源占用
- 📡 完整支援 MQTT 3.1.1 協定
- 🔧 靈活的 YAML 配置文件支援
- 🔐 可選的客戶端認證機制
- 📊 即時連接統計和監控
- 🛡️ 優雅關閉和錯誤處理機制
- 📝 結構化日誌記錄
- 🌐 跨平台支援 (Linux ARM64/x86_64, macOS ARM64, Windows x86_64)

## 📦 套件內容

發行套件包含以下檔案：

```
rtk_mqtt_broker_release/
├── bin/                        # 可執行檔案目錄
│   ├── rtk_mqtt_broker-linux-arm64      # ARM64 Linux 版本
│   ├── rtk_mqtt_broker-linux-amd64      # x86_64 Linux 版本
│   ├── rtk_mqtt_broker-darwin-arm64     # ARM64 macOS 版本
│   └── rtk_mqtt_broker-windows-amd64.exe # x86_64 Windows 版本
├── config/                     # 配置文件目錄
│   └── config.yaml            # 預設配置文件
├── test/                      # 測試工具目錄
│   ├── mqtt_client.go         # MQTT 基本功能測試
│   ├── topic_listener.go      # Topic 監聽測試工具
│   └── simple_publisher.go    # 簡單發布測試工具
├── README.md                  # 本使用指南
├── MANUAL.md                  # 工程師快速使用手冊
└── LICENSE                    # 授權條款
```

## 🚀 快速啟動

### 1. 選擇適合的可執行檔案

根據您的目標平台選擇對應的可執行檔案：

- **ARM64 Linux**: `bin/rtk_mqtt_broker-linux-arm64`
- **x86_64 Linux**: `bin/rtk_mqtt_broker-linux-amd64`
- **ARM64 macOS**: `bin/rtk_mqtt_broker-darwin-arm64`
- **x86_64 Windows**: `bin/rtk_mqtt_broker-windows-amd64.exe`

### 2. 啟動 MQTT Broker

#### Linux/macOS 系統

```bash
# 給予執行權限
chmod +x bin/rtk_mqtt_broker-linux-amd64

# 使用預設配置啟動
./bin/rtk_mqtt_broker-linux-amd64

# 指定自訂配置檔案
./bin/rtk_mqtt_broker-linux-amd64 -config /path/to/your/config.yaml
```

#### Windows 系統

```cmd
# 使用預設配置啟動
bin\rtk_mqtt_broker-windows-amd64.exe

# 指定自訂配置檔案
bin\rtk_mqtt_broker-windows-amd64.exe -config C:\path\to\your\config.yaml
```

### 3. 驗證服務啟動

成功啟動後，您會看到類似以下的輸出：

```
╔═══════════════════════════════════════════════════════════════╗
║                  Realtek Embedded MQTT Broker                ║
║                                                               ║
║  High-performance MQTT 3.1.1 broker for RTK IoT devices     ║
║  Copyright (c) 2024 Realtek Semiconductor Corp.              ║
╚═══════════════════════════════════════════════════════════════╝

[RTK-MQTT] [INFO] Realtek Embedded MQTT Broker started successfully
[RTK-MQTT] [INFO] Listening on interface: 0.0.0.0
[RTK-MQTT] [INFO] Listening on port: 1883
[RTK-MQTT] [INFO] Full address: 0.0.0.0:1883
[RTK-MQTT] [INFO] Max clients: 1000
```

## 🧪 功能驗證和測試

### 內建測試工具

發行包中包含完整的測試工具，可用於驗證 broker 功能：

#### 1. MQTT 基本功能測試

```bash
# 進入測試目錄
cd test/

# 執行基本 MQTT 功能測試
go run mqtt_client.go
```

測試內容包括：
- 基本連接測試
- 發布/訂閱功能驗證
- 多 Topic 測試
- QoS 等級測試 (0, 1, 2)
- 訊息統計和延遲測試

#### 2. Topic 監聽測試

```bash
# 基本 topic 監聽
go run topic_listener.go -topic "test/+"

# 完整參數監聽
go run topic_listener.go \
  -broker localhost \
  -port 1883 \
  -topic "sensor/+/data" \
  -qos 1 \
  -duration 60 \
  -verbose
```

參數說明：
- `-broker`: MQTT broker 地址 (預設: localhost)
- `-port`: MQTT broker 埠號 (預設: 1883)
- `-topic`: 監聽的 Topic，支援通配符 `+` 和 `#`
- `-qos`: QoS 等級 0-2 (預設: 0)
- `-duration`: 監聽時間秒數，0=無限 (預設: 0)
- `-verbose`: 顯示詳細訊息
- `-client`: 客戶端 ID (預設: topic_listener)

#### 3. 簡單發布測試

```bash
# 發布測試訊息
go run simple_publisher.go
```

### 外部工具測試

您也可以使用第三方 MQTT 客戶端進行測試：

#### 使用 mosquitto 客戶端

```bash
# 訂閱主題 (開啟新終端視窗)
mosquitto_sub -h YOUR_BROKER_IP -p 1883 -t "test/topic"

# 發布訊息
mosquitto_pub -h YOUR_BROKER_IP -p 1883 -t "test/topic" -m "Hello World"
```

#### 使用圖形化 MQTT 工具

推薦工具：MQTT.fx, MQTT Explorer, HiveMQ WebSocket Client

連接參數：
- **Broker Address**: 您的伺服器 IP
- **Broker Port**: 1883
- **Protocol**: MQTT 3.1.1
- **Client ID**: 任意唯一識別碼

## ⚙️ 配置說明

### 配置文件結構

預設配置文件 `config/config.yaml` 包含以下設定：

```yaml
server:
  port: 1883              # MQTT 服務埠號
  host: "0.0.0.0"         # 監聽地址 (0.0.0.0 表示監聽所有網路介面)
  max_clients: 1000       # 最大客戶端連接數
  enable_stats: true      # 啟用統計輸出

security:
  enable_auth: false      # 啟用客戶端認證
  users: []              # 用戶清單 (當 enable_auth 為 true 時使用)

logging:
  level: "info"          # 日誌等級 (debug, info, warn, error)
```

### 常用配置調整

#### 修改監聽埠號

```yaml
server:
  port: 8883              # 改為其他埠號
```

#### 啟用客戶端認證

```yaml
security:
  enable_auth: true
  users:
    - username: "device1"
      password: "password1"
    - username: "device2"
      password: "password2"
```

#### 調整日誌等級

```yaml
logging:
  level: "debug"          # 顯示更詳細的除錯資訊
```

### 進階配置範例

#### 高效能生產環境配置

```yaml
server:
  port: 1883
  host: "0.0.0.0"
  max_clients: 5000       # 高並發支援
  enable_stats: false     # 關閉統計以提升效能

security:
  enable_auth: true
  users:
    - username: "production_device"
      password: "secure_password_123"
    - username: "monitoring_system"
      password: "monitor_pass_456"

logging:
  level: "warn"          # 減少日誌輸出提升效能
```

#### 開發測試環境配置

```yaml
server:
  port: 1883
  host: "0.0.0.0"
  max_clients: 100
  enable_stats: true     # 啟用統計便於監控

security:
  enable_auth: false     # 開發環境可關閉認證

logging:
  level: "debug"         # 詳細日誌便於除錯
```

## 🔌 客戶端連接和使用範例

### 連接參數

- **伺服器地址**: 您部署 broker 的主機 IP 位址
- **埠號**: 1883 (預設) 或您在配置中設定的埠號
- **協定**: MQTT 3.1.1
- **服務地址**: `0.0.0.0:1883` (預設監聽所有網路介面)

### 程式開發範例

#### 基本發布/訂閱 (Go)

```go
import "github.com/eclipse/paho.mqtt.golang"

// 建立客戶端
opts := mqtt.NewClientOptions()
opts.AddBroker("tcp://localhost:1883")
opts.SetClientID("my_client")

client := mqtt.NewClient(opts)
if token := client.Connect(); token.Wait() && token.Error() != nil {
    panic(token.Error())
}

// 訂閱溫度感測器資料
client.Subscribe("sensor/temperature", 0, func(client mqtt.Client, msg mqtt.Message) {
    fmt.Printf("收到溫度資料: %s\n", msg.Payload())
})

// 發布溫度資料
client.Publish("sensor/temperature", 0, false, "23.5°C")
```

#### 通配符 Topic 使用

```go
// 訂閱所有感測器資料
client.Subscribe("sensor/+/data", 0, messageHandler)

// 訂閱設備下的所有子主題
client.Subscribe("device/ESP32_001/#", 0, messageHandler)
```

#### 帶認證的連接

```go
opts := mqtt.NewClientOptions()
opts.AddBroker("tcp://192.168.1.100:1883")
opts.SetClientID("authenticated_client")
opts.SetUsername("device1")
opts.SetPassword("password1")
```

## 🛠️ 系統服務部署

### Linux systemd 服務

建立服務文件 `/etc/systemd/system/rtk-mqtt-broker.service`：

```ini
[Unit]
Description=Realtek Embedded MQTT Broker
After=network.target

[Service]
Type=simple
User=mqtt
Group=mqtt
WorkingDirectory=/opt/rtk_mqtt_broker
ExecStart=/opt/rtk_mqtt_broker/bin/rtk_mqtt_broker-linux-amd64 -config /opt/rtk_mqtt_broker/config/config.yaml
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

啟用和啟動服務：

```bash
sudo systemctl enable rtk-mqtt-broker
sudo systemctl start rtk-mqtt-broker
sudo systemctl status rtk-mqtt-broker
```

### Windows 服務

建議使用 NSSM (Non-Sucking Service Manager) 將程式註冊為 Windows 服務：

```cmd
# 下載並安裝 NSSM
nssm install "RTK MQTT Broker" "C:\rtk_mqtt_broker\bin\rtk_mqtt_broker-windows-amd64.exe"
nssm set "RTK MQTT Broker" AppParameters "-config C:\rtk_mqtt_broker\config\config.yaml"
nssm start "RTK MQTT Broker"
```

## 🔍 監控和維護

### 查看統計資訊

當 `enable_stats: true` 時，broker 每 30 秒會輸出統計資訊：

```
[RTK-MQTT] [INFO] Stats - Clients: 5, Messages Received: 1250, Messages Sent: 1250
```

### 日誌監控

#### Linux

```bash
# 即時查看日誌 (如果使用 systemd)
journalctl -u rtk-mqtt-broker -f

# 搜尋錯誤
journalctl -u rtk-mqtt-broker | grep ERROR
```

#### Windows

檢查 Windows 事件檢視器或設定日誌輸出到檔案。

### 效能調校

#### 高負載環境

```yaml
server:
  max_clients: 5000       # 增加最大客戶端數
  enable_stats: false     # 關閉統計以提升效能

logging:
  level: "warn"          # 減少日誌輸出
```

## 🚨 故障排除

### 常見問題

#### 1. 無法啟動 - 埠號被占用

```bash
# 檢查埠號使用情況
netstat -ln | grep 1883
lsof -i :1883

# 解決方案：修改配置中的埠號或停止占用程序
```

#### 2. 客戶端連接被拒絕

- 檢查防火牆設定
- 確認 broker 正在監聽正確的網路介面
- 檢查是否啟用了認證但客戶端未提供憑證

#### 3. 訊息遺失

- 檢查客戶端 QoS 等級設定
- 確認網路連接穩定性
- 檢查 broker 資源使用情況

#### 4. 記憶體使用過高

- 調整 `max_clients` 參數
- 檢查是否有殭屍連接
- 監控訊息積壓情況

### 取得技術支援

如需技術支援，請聯繫：
- **電子郵件**: support@realtek.com
- **技術文檔**: 請參考完整的開發者文檔
- **問題回報**: 提供詳細的錯誤日誌和環境資訊

## 📋 系統需求

### 執行環境最低需求

- **CPU**: ARM64 或 x86_64 架構
- **記憶體**: 512MB RAM
- **硬碟空間**: 50MB
- **網路**: TCP/IP 支援

### 開發和測試環境需求

如需使用內建的測試工具，需要：
- **Go 語言**: Go 1.19+ (用於執行測試工具)
- **MQTT 客戶端**: mosquitto-clients (可選，用於外部測試)

### 建議生產配置

- **CPU**: 多核心處理器
- **記憶體**: 2GB+ RAM (高並發環境)
- **網路**: 千兆位元網路
- **作業系統**: 
  - Linux: Ubuntu 18.04+, CentOS 7+, RHEL 7+
  - Windows: Windows Server 2016+, Windows 10+
  - macOS: macOS 10.14+

### 效能指標

在建議配置下的效能表現：
- **最大並發連接**: 5000+ 客戶端
- **訊息吞吐量**: 10,000+ 訊息/秒
- **記憶體占用**: < 100MB (1000 個連接)
- **CPU 使用率**: < 10% (正常負載)

## 📄 授權資訊

本軟體遵循 MIT 授權條款。詳細資訊請參考 LICENSE 檔案。

Copyright (c) 2024 Realtek Semiconductor Corp. All rights reserved.

---

**Realtek Embedded MQTT Broker** - 為您的 IoT 生態系統提供可靠的 MQTT 通訊服務。
# RTK Home Network Environment Simulator

一個基於 Go 語言開發的全面性家用網路環境模擬器，支援各種 IoT 設備、網路設備和客戶端設備，使用 RTK MQTT 協議進行通信。

## 🎯 專案目標

本模擬器旨在為 RTK Controller 和相關 IoT 系統提供一個真實的家庭網路環境測試平台，支援：

- **多樣化設備模擬**: 路由器、智慧燈泡、空調、感測器、攝影機等 20+ 種設備類型
- **真實網路行為**: 包含信號干擾、延遲、封包遺失等真實網路特性
- **情境化模擬**: 日常作息、週末模式、故障情境等多種生活場景
- **RTK MQTT 協議**: 完整支援 RTK MQTT v1.0 協議規格
- **可擴展架構**: 支援自定義設備類型和網路拓撲

## 🏗️ 架構概覽

```
┌─────────────────┐    RTK MQTT     ┌─────────────────┐
│   模擬設備群     │ ←──────────────→ │   MQTT Broker   │
│                 │                 │                 │
│ • 網路設備      │    上行訊息      │ • RTK Controller│
│ • IoT 設備      │    - state      │ • 診斷分析      │
│ • 客戶端設備    │    - telemetry  │ • 拓撲管理      │
│                 │    - events     │ • LLM 整合      │
│                 │                 │                 │
│                 │    下行命令      │                 │
│                 │    - commands   │                 │
└─────────────────┘                 └─────────────────┘
```

## 🚀 快速開始

### 前置需求

- Go 1.21 或更高版本
- MQTT Broker (推薦 Eclipse Mosquitto)
- Make (可選，用於建構自動化)

### 安裝和建構

```bash
# 複製專案
git clone <repository-url>
cd rtk_simulation

# 下載依賴
make deps

# 建構程式
make build

# 或直接執行
make run
```

### 基本使用

```bash
# 使用預設配置執行模擬器
./build/rtk-simulator run

# 使用指定配置檔案
./build/rtk-simulator run -c configs/home_basic.yaml

# 生成配置範本
./build/rtk-simulator generate home_basic -o my_config.yaml

# 驗證配置檔案
./build/rtk-simulator validate configs/home_basic.yaml

# 顯示幫助
./build/rtk-simulator --help
```

## 📋 設備類型支援

### 網路設備
- **路由器 (Router)**: 主要網路閘道，支援 WiFi 和有線連接
- **交換機 (Switch)**: 有線網路交換，支援 PoE 供電
- **存取點 (Access Point)**: WiFi 訊號擴展
- **Mesh 節點 (Mesh Node)**: Mesh 網路節點

### IoT 設備
- **智慧燈泡 (Smart Bulb)**: 可調光調色的智慧照明
- **空調 (Air Conditioner)**: 溫控設備，支援多種模式
- **環境感測器 (Environmental Sensor)**: 溫濕度、空氣品質監測
- **安全攝影機 (Security Camera)**: 監控設備，支援動作偵測
- **智慧插座 (Smart Plug)**: 電源控制和用電監測
- **門鈴 (Smart Doorbell)**: 智慧門鈴系統

### 客戶端設備
- **智慧手機 (Smartphone)**: 行動設備，支援漫遊
- **筆記型電腦 (Laptop)**: 工作設備，高頻寬需求
- **平板電腦 (Tablet)**: 娛樂設備
- **智慧電視 (Smart TV)**: 串流媒體設備

## 🔧 配置系統

### 基本配置結構

```yaml
simulation:
  name: "my_simulation"
  duration: 1h
  real_time_factor: 1.0
  max_devices: 50

mqtt:
  broker: "localhost"
  port: 1883
  client_prefix: "rtk_home"

network:
  topology: "single_router"
  subnet: "192.168.1.0/24"
  internet_bandwidth: 100

devices:
  network_devices: [...]
  iot_devices: [...]
  client_devices: [...]
```

### 預設配置範本

- **home_basic.yaml**: 基本家庭網路 (1 路由器 + 4 IoT 設備 + 2 客戶端)
- **home_advanced.yaml**: 進階家庭網路 (Mesh 網路 + 更多設備)
- **apartment.yaml**: 公寓環境配置
- **smart_home.yaml**: 完整智慧家庭配置

### 設備配置範例

```yaml
iot_devices:
  - id: "living_room_light"
    type: "smart_bulb"
    tenant: "home"
    site: "living_room"
    category: "lighting"
    power_source: "ac"
    usage_patterns:
      - name: "evening_mode"
        time_range:
          start_time: "18:00"
          end_time: "23:00"
        probability: 0.8
        behavior:
          power_state: "on"
          brightness: 70
```

## 🌐 網路拓撲支援

### 單一路由器拓撲
```
Internet ── Router ──┬── IoT Device 1
                     ├── IoT Device 2
                     ├── Smartphone
                     └── Laptop
```

### Mesh 網路拓撲
```
Internet ── Main Router ──┬── Mesh Node 1 ── IoT Devices
                          ├── Mesh Node 2 ── IoT Devices
                          └── Direct Devices
```

### 混合拓撲
支援有線 + 無線 + Mesh 的混合網路架構

## 📊 監控和統計

### 即時監控
- 設備狀態監控
- 網路連接狀態
- 頻寬使用統計
- 延遲和封包遺失率

### 統計資訊
```bash
# 檢視拓撲統計
curl http://localhost:8080/stats

# 檢視設備列表
curl http://localhost:8080/devices

# 檢視連接狀態
curl http://localhost:8080/connections
```

## 🎭 情境模擬

### 日常作息模擬
- **早晨模式**: 設備啟動、連接增加
- **工作時段**: 辦公設備高使用率
- **晚間模式**: 娛樂設備活躍、照明調節
- **夜間模式**: 設備休眠、安全監控

### 特殊情境
- **週末模式**: 不同的使用時間模式
- **假期模式**: 外出安全模式
- **派對模式**: 多媒體設備高負載
- **故障模式**: 網路中斷、設備故障

### 故障情境模擬
```yaml
scenarios:
  - name: "network_outage"
    events:
      - time: 30m
        type: "network_failure"
        target: "main_router"
        parameters:
          duration: 5m
        probability: 0.1
```

## 🔌 RTK MQTT 協議支援

### 訊息類型
- **狀態訊息 (state)**: 設備健康狀態
- **遙測訊息 (telemetry)**: 感測器數據
- **事件訊息 (events)**: 設備事件通知
- **命令訊息 (commands)**: 設備控制指令

### Topic 結構
```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}
```

### 訊息格式範例
```json
{
  "schema": "state/1.0",
  "ts": "1699123456789",
  "device_id": "aabbccddeeff",
  "payload": {
    "health": "ok",
    "uptime_s": 86400,
    "cpu_usage": 45.2,
    "temperature_c": 42.1
  }
}
```

## 🛠️ 開發指南

### 專案結構
```
rtk_simulation/
├── cmd/simulator/          # 主程式入口
├── pkg/
│   ├── devices/            # 設備模擬器
│   │   ├── base/           # 基礎設備類別
│   │   ├── network/        # 網路設備
│   │   ├── iot/            # IoT 設備
│   │   └── client/         # 客戶端設備
│   ├── network/            # 網路拓撲管理
│   ├── config/             # 配置系統
│   └── scenarios/          # 情境系統
├── configs/                # 配置檔案範本
└── docs/                   # 文檔
```

### 新增設備類型

1. 在 `pkg/devices/iot/` 建立新設備檔案
2. 實作 `base.Device` 介面
3. 在 factory 中註冊設備類型
4. 更新配置結構定義

### 建構指令

```bash
# 格式化程式碼
make fmt

# 執行測試
make test

# 執行 linter
make lint

# 建構所有平台版本
make build-all

# 生成配置範本
make config-basic
make config-advanced
make config-smart
```

## 📈 效能指標

### 設計目標
- **並行設備**: 支援 100+ 設備同時模擬
- **CPU 使用率**: < 50% (100 設備)
- **記憶體使用**: < 2GB (100 設備)
- **MQTT 延遲**: < 100ms
- **模擬精確度**: > 95%

### 最佳化建議
- 調整設備更新間隔
- 使用合適的 QoS 設定
- 限制並行連接數
- 啟用壓縮和批次處理

## 🐳 Docker 支援

### 建構 Docker 映像
```bash
make docker
```

### 執行容器
```bash
docker run -d \
  --name rtk-simulator \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs \
  rtk-simulator:latest run -c configs/home_basic.yaml
```

### Docker Compose
```yaml
version: '3.8'
services:
  mqtt-broker:
    image: eclipse-mosquitto
    ports:
      - "1883:1883"
  
  rtk-simulator:
    image: rtk-simulator:latest
    depends_on:
      - mqtt-broker
    environment:
      - RTK_SIM_MQTT_BROKER=mqtt-broker
    volumes:
      - ./configs:/app/configs
    command: run -c configs/home_basic.yaml
```

## 🔧 故障排除

### 常見問題

1. **MQTT 連接失敗**
   ```bash
   # 檢查 MQTT Broker 狀態
   mosquitto_pub -h localhost -t test -m "hello"
   ```

2. **設備無法啟動**
   ```bash
   # 使用詳細日誌模式
   ./rtk-simulator run --verbose --log-level debug
   ```

3. **配置驗證失敗**
   ```bash
   # 驗證 YAML 語法
   ./rtk-simulator validate configs/my_config.yaml
   ```

### 日誌分析
```bash
# 檢視特定設備日誌
grep "device_id=smart_bulb_01" simulation.log

# 檢視錯誤訊息
grep "ERROR" simulation.log
```

## 🤝 貢獻指南

1. Fork 專案
2. 建立功能分支 (`git checkout -b feature/new-device`)
3. 提交變更 (`git commit -am 'Add new device type'`)
4. 推送分支 (`git push origin feature/new-device`)
5. 建立 Pull Request

### 程式碼規範
- 遵循 Go 官方程式碼風格
- 添加適當的單元測試
- 更新相關文檔
- 執行 `make pre-commit` 檢查

## 📝 授權條款

本專案採用 MIT 授權條款 - 詳見 [LICENSE](LICENSE) 檔案

## 🙏 致謝

- RTK MQTT 協議規格團隊
- Eclipse Paho MQTT 客戶端程式庫
- Go 社群的各種開源程式庫

## 📞 支援

- **問題回報**: [GitHub Issues](issues)
- **功能請求**: [GitHub Discussions](discussions)
- **文檔**: [Wiki](wiki)

---

**版本**: v1.0.0  
**最後更新**: 2024-08-22  
**相容性**: RTK MQTT Protocol v1.0
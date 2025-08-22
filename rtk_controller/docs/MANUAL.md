# RTK Controller 使用手冊

> **工程師快速部署指南** - 本手冊專為拿到 release binary package 的工程師而設計

## 目錄

1. [快速開始](#快速開始)
2. [系統架構](#系統架構)
3. [系統需求](#系統需求)
4. [安裝部署](#安裝部署)
5. [基本使用](#基本使用)
6. [配置說明](#配置說明)
7. [功能驗證](#功能驗證)
8. [故障排除](#故障排除)
9. [維護運行](#維護運行)
10. [架構詳解](#架構詳解)

## 快速開始

### 1. 解壓縮發行包

```bash
# 解壓縮發行包
tar -xzf rtk_controller-[版本]_[平台].tar.gz
cd rtk_controller-[平台]/

# 檢查包內容
ls -la
```

發行包包含：
```
rtk_controller-[平台]/
├── bin/                    # 可執行檔案
│   └── rtk_controller-*    # 對應平台的執行檔
├── configs/                # 配置檔案
│   └── controller.yaml     # 主配置檔案
├── docs/                   # 技術文檔
├── test-tools/             # 測試工具 (可選)
├── test/scripts/           # 測試腳本
├── demo_cli.sh             # 功能演示腳本
├── MANUAL.md               # 本手冊
├── LICENSE                 # 許可證
└── VERSION                 # 版本資訊
```

### 2. 選擇對應平台執行檔

根據您的系統選擇正確的執行檔：

**Linux ARM64 (樹莓派、ARM 伺服器)**
```bash
cp bin/rtk_controller-linux-arm64 ./rtk_controller
chmod +x rtk_controller
```

**Linux x86_64 (標準 Linux 伺服器)**
```bash
cp bin/rtk_controller-linux-amd64 ./rtk_controller
chmod +x rtk_controller
```

**macOS ARM64 (Apple Silicon Mac)**
```bash
cp bin/rtk_controller-darwin-arm64 ./rtk_controller
chmod +x rtk_controller
```

**Windows x86_64**
```cmd
copy bin\rtk_controller-windows-amd64.exe rtk_controller.exe
```

### 3. 立即測試

```bash
# 檢查版本
./rtk_controller --version

# 啟動交互式 CLI
./rtk_controller --cli
```

成功啟動會顯示：
```
RTK Controller Interactive CLI
==============================
Version: [版本號]
Type 'help' for available commands, 'exit' to quit

rtk> 
```

## 系統需求

### 硬體需求
- **CPU**: 雙核心 1GHz 以上
- **記憶體**: 最少 512MB，建議 1GB 以上
- **磁碟空間**: 最少 100MB 可用空間
- **網路**: 支援 TCP/IP 網路連接

### 作業系統支援
- **Linux**: Ubuntu 18.04+, CentOS 7+, Debian 9+
- **macOS**: macOS 10.15+ (Catalina)
- **Windows**: Windows 10, Windows Server 2016+

### 網路需求
- **MQTT Broker**: 需要可連接的 MQTT broker (如 Mosquitto)
- **連接埠**: 
  - MQTT: 預設 1883 (可設定)
  - MQTT over TLS: 預設 8883 (可設定)

## 安裝部署

### 生產環境部署

#### 1. 建立專用用戶 (Linux/macOS)

```bash
# 建立 rtk 用戶
sudo useradd -m -s /bin/bash rtk
sudo passwd rtk

# 切換到 rtk 用戶
sudo su - rtk

# 建立工作目錄
mkdir -p ~/rtk_controller
cd ~/rtk_controller
```

#### 2. 部署執行檔

```bash
# 將發行包複製到部署目錄
tar -xzf rtk_controller-*.tar.gz --strip-components=1

# 設定執行權限
chmod +x bin/rtk_controller-*

# 建立符號連結
ln -sf bin/rtk_controller-[您的平台] rtk_controller
```

#### 3. 建立系統服務 (Linux)

建立 systemd 服務檔案：

```bash
sudo tee /etc/systemd/system/rtk-controller.service > /dev/null <<EOF
[Unit]
Description=RTK Controller Network Management System
After=network.target

[Service]
Type=simple
User=rtk
WorkingDirectory=/home/rtk/rtk_controller
ExecStart=/home/rtk/rtk_controller/rtk_controller --config configs/controller.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 啟用並啟動服務
sudo systemctl daemon-reload
sudo systemctl enable rtk-controller
sudo systemctl start rtk-controller

# 檢查服務狀態
sudo systemctl status rtk-controller
```

## 基本使用

### 交互式 CLI 模式

最適合初次使用和測試：

```bash
./rtk_controller --cli
```

#### 常用命令

```bash
# 顯示所有可用命令
rtk> help

# 檢查系統狀態
rtk> system status

# 查看網絡拓撲
rtk> topology show

# 列出設備
rtk> device list

# 執行網絡診斷
rtk> diagnostics run speed-test

# 查看 QoS 統計
rtk> qos show stats

# 重新載入配置
rtk> config reload

# 退出
rtk> exit
```

## 系統架構

### 核心架構概覽

RTK Controller 採用模組化設計，主要由以下核心系統組成：

```
┌─────────────────────────────────────────────────────────────┐
│                    RTK Controller                           │
├─────────────────────────────────────────────────────────────┤
│  CLI Mode    │  Service Mode    │  MCP Server Mode         │
├─────────────────────────────────────────────────────────────┤
│                  核心管理層                                 │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐  │
│  │ Topology    │ Device      │ Command     │ Diagnosis   │  │
│  │ Manager     │ Manager     │ Manager     │ Manager     │  │
│  └─────────────┴─────────────┴─────────────┴─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                  智能處理層                                 │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐  │
│  │ LLM Tool    │ Workflow    │ QoS         │ Identity    │  │
│  │ Engine      │ Engine      │ Manager     │ Manager     │  │
│  └─────────────┴─────────────┴─────────────┴─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                  通訊存儲層                                 │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐  │
│  │ MQTT        │ BuntDB      │ Schema      │ Logging     │  │
│  │ Client      │ Storage     │ Validator   │ System      │  │
│  └─────────────┴─────────────┴─────────────┴─────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 三種運行模式

#### 1. **交互式 CLI 模式** (`--cli`)
- 適用於開發、測試、手動診斷
- 提供即時命令執行和結果顯示
- 支援拓撲探索、設備管理、診斷測試

#### 2. **服務模式** (預設)
- 適用於生產環境長期運行
- 背景處理 MQTT 訊息和拓撲更新
- 自動診斷和告警

#### 3. **MCP 伺服器模式** (`--mcp`)
- 提供 Model Context Protocol 整合
- 支援外部 AI 系統呼叫
- 適用於 LLM 驅動的網路管理

### 數據流向

```
IoT 設備 → MQTT → Schema Validation → Device Manager → Topology Manager
    ↓                                                        ↓
事件處理 ← Identity Manager ← Connection Inference ← 拓撲更新
    ↓                                                        ↓
Diagnosis Manager → LLM Tool Engine → Workflow Engine → 智能分析
    ↓                                                        ↓
告警/報告 ← QoS Manager ← Command Manager ← 自動修復建議
```

### 服務模式

適合生產環境長期運行：

```bash
# 前台運行 (測試用)
./rtk_controller --config configs/controller.yaml

# 後台運行
nohup ./rtk_controller --config configs/controller.yaml > logs/controller.log 2>&1 &
```

### 常用操作

#### 快速健康檢查
```bash
# 檢查連線狀態
./rtk_controller --cli --execute "system status"

# 檢查 MQTT 連線
./rtk_controller --cli --execute "mqtt status"
```

#### 匯出拓撲資料
```bash
# 匯出為 JSON
./rtk_controller --cli --execute "topology export --format json --output topology.json"
```

## 配置說明

主配置檔案：`configs/controller.yaml`

### 基本配置

```yaml
# MQTT 連線設定
mqtt:
  broker: "localhost"        # MQTT broker 地址
  port: 1883                # MQTT 連接埠
  client_id: "rtk-controller"
  username: ""              # MQTT 用戶名 (可選)
  password: ""              # MQTT 密碼 (可選)

# 資料存儲
storage:
  path: "data"              # 資料目錄

# 日誌設定
logging:
  level: "info"             # debug, info, warn, error
  file: "logs/controller.log"
```

### 進階配置

```yaml
# TLS 加密 (生產環境建議)
mqtt:
  tls:
    enabled: true
    cert_file: "certs/client.crt"
    key_file: "certs/client.key"
    ca_file: "certs/ca.crt"

# 診斷設定
diagnosis:
  enabled: true
  default_analyzers:
    - "builtin_wifi_analyzer"
```

### 環境變數覆蓋

```bash
# 覆蓋 MQTT broker 地址
export RTK_MQTT_BROKER=192.168.1.100
export RTK_MQTT_PORT=1883

# 啟動
./rtk_controller --config configs/controller.yaml
```

## 功能驗證

### 1. 基本功能測試

執行演示腳本：
```bash
./demo_cli.sh
```

### 2. 連線測試

```bash
./rtk_controller --cli --execute "mqtt connect"
```

### 3. 使用測試工具

如果發行包包含測試工具：

```bash
# 基本 MQTT 功能測試
./test-tools/mqtt_client

# 拓撲測試
./test-tools/test_topology_simple

# 診斷測試
./test-tools/test_diagnostics
```

### 4. 手動驗證步驟

1. **檢查服務啟動**
   ```bash
   ps aux | grep rtk_controller
   ```

2. **檢查日誌**
   ```bash
   tail -f logs/controller.log
   ```

3. **檢查資料目錄**
   ```bash
   ls -la data/
   ```

4. **檢查網路連線**
   ```bash
   netstat -tulpn | grep rtk_controller
   ```

## 故障排除

### 常見問題

#### 1. 執行檔無法啟動

**問題**: `Permission denied` 或 `Command not found`

**解決方案**:
```bash
# 檢查檔案權限
ls -la rtk_controller

# 設定執行權限
chmod +x rtk_controller

# 檢查是否為正確平台
file rtk_controller
```

#### 2. MQTT 連線失敗

**問題**: `Failed to connect to MQTT broker`

**解決方案**:
```bash
# 檢查 broker 是否運行
telnet [broker_ip] 1883

# 檢查配置檔案
cat configs/controller.yaml | grep -A 10 mqtt

# 測試基本連線
mosquitto_pub -h [broker_ip] -p 1883 -t test -m "hello"
```

#### 3. 權限錯誤

**問題**: `Permission denied` 存取 data/ 或 logs/ 目錄

**解決方案**:
```bash
# 建立目錄並設定權限
mkdir -p data logs
chmod 755 data logs

# 或以 root 身份執行 (不建議生產環境)
sudo ./rtk_controller --config configs/controller.yaml
```

#### 4. 配置檔案錯誤

**問題**: `Failed to parse config file`

**解決方案**:
```bash
# 驗證 YAML 語法
python3 -c "import yaml; yaml.safe_load(open('configs/controller.yaml'))"

# 重置為預設配置
cp configs/controller.yaml.example configs/controller.yaml
```

### 診斷命令

```bash
# 檢查系統資源
./rtk_controller --cli --execute "system info"

# 檢查配置
./rtk_controller --cli --execute "config show"

# 檢查 MQTT 狀態
./rtk_controller --cli --execute "mqtt status"

# 檢查儲存狀態
./rtk_controller --cli --execute "storage status"
```

### 日誌分析

```bash
# 檢查錯誤日誌
grep -i error logs/controller.log

# 檢查警告
grep -i warning logs/controller.log

# 即時監控
tail -f logs/controller.log | grep -E "(ERROR|WARN|FATAL)"
```

## 維護運行

### 日常維護

#### 1. 日誌輪轉

```bash
# 手動輪轉日誌
mv logs/controller.log logs/controller.log.$(date +%Y%m%d)
kill -USR1 $(pgrep rtk_controller)  # 重新開啟日誌檔案
```

#### 2. 資料備份

```bash
# 備份資料目錄
tar -czf backup/rtk_data_$(date +%Y%m%d).tar.gz data/

# 定期清理舊備份 (保留 7 天)
find backup/ -name "rtk_data_*.tar.gz" -mtime +7 -delete
```

#### 3. 效能監控

```bash
# 檢查記憶體使用
ps aux | grep rtk_controller | awk '{print $4"%", $6/1024"MB"}'

# 檢查 CPU 使用
top -p $(pgrep rtk_controller)
```

### 版本升級

```bash
# 停止服務
sudo systemctl stop rtk-controller

# 備份當前版本
cp rtk_controller rtk_controller.backup

# 替換執行檔
cp bin/rtk_controller-[新平台] rtk_controller

# 重新啟動
sudo systemctl start rtk-controller

# 檢查版本
./rtk_controller --version
```

### 安全建議

1. **不使用 root 權限運行**
2. **啟用 TLS 加密**
3. **定期更新密碼**
4. **監控異常連線**
5. **定期備份資料**

---

## 技術支援

**版本資訊**: 請執行 `./rtk_controller --version` 取得詳細版本資訊

**配置檔案**: 詳細配置說明請參考 `configs/controller.yaml` 中的註釋

**技術文檔**: 更多技術細節請參考 `docs/` 目錄

**問題回報**: 請提供以下資訊：
- 執行環境 (OS, 版本)
- 錯誤訊息
- 相關日誌
- 配置檔案 (去除敏感資訊)

---

**發行包版本**: 請查看 `VERSION` 檔案  
**授權許可**: 請查看 `LICENSE` 檔案  
**最後更新**: 2025-08-18

## 架構詳解

### 核心管理層架構

#### 拓撲管理器 (Topology Manager)
**檔案位置**: `internal/topology/manager.go`

負責網路拓撲的建構、更新與維護：
- **8階段拓撲偵測系統**：設備發現 → 身份識別 → 連接推論 → 品質監控 → 異常偵測 → 漫遊追蹤 → 診斷整合 → 告警處理
- **即時更新機制**：監聽 MQTT 訊息並即時更新網路拓撲
- **連接推論引擎**：基於 WiFi 信號強度、路由表、ARP 表自動推論設備連接關係
- **設備分類系統**：自動識別 Gateway、Access Point、Switch、Client 等設備角色

```go
type ManagerConfig struct {
    TopologyUpdateInterval     time.Duration // 拓撲更新間隔
    MetricsUpdateInterval      time.Duration // 指標更新間隔
    ConnectionHistoryRetention time.Duration // 連接歷史保留期
    EnableRealTimeUpdates      bool         // 啟用即時更新
    EnableConnectionInference  bool         // 啟用連接推論
    EnableDeviceClassification bool         // 啟用設備分類
}
```

#### 設備管理器 (Device Manager)
**檔案位置**: `internal/device/manager.go`

管理網路中的所有設備：
- **設備生命週期管理**：註冊、更新、離線檢測、清理
- **狀態同步**：與 MQTT 訊息同步設備狀態
- **事件處理**：處理設備上線/下線事件

#### 命令管理器 (Command Manager)
**檔案位置**: `internal/command/manager.go`

處理設備命令執行：
- **命令佇列管理**：支援批次命令和優先級排程
- **執行狀態追蹤**：監控命令執行進度和結果
- **重試機制**：失敗命令的自動重試

#### 診斷管理器 (Diagnosis Manager)
**檔案位置**: `internal/diagnosis/manager.go`

執行網路診斷測試：
- **內建診斷器**：WiFi 分析、速度測試、延遲檢測
- **LLM 整合**：支援 AI 驅動的智能診斷
- **工作流支援**：複雜診斷場景的自動化執行

### 智能處理層架構

#### LLM 工具引擎 (LLM Tool Engine)
**檔案位置**: `internal/llm/tool_engine.go`

提供 AI 驅動的網路分析：
- **工具註冊系統**：動態載入和管理 LLM 工具
- **會話管理**：支援多並發的診斷會話
- **指標收集**：記錄工具執行效能和結果

**可用工具類別**：
- **網格網路工具** (`tools_mesh_network.go`)：mesh 拓撲分析和優化
- **WiFi 進階工具** (`tools_wifi_advanced.go`)：信號分析、頻道優化
- **拓撲工具** (`tools_topology.go`)：拓撲視覺化和分析
- **網路工具** (`tools_network.go`)：基礎網路診斷
- **配置管理工具** (`tools_config_management.go`)：設備配置操作

#### 工作流引擎 (Workflow Engine)
**檔案位置**: `internal/workflow/workflow_engine.go`

自動化複雜診斷流程：
- **意圖分類器**：識別用戶診斷意圖
- **工作流執行器**：編排多步驟診斷流程
- **配置驗證器**：確保工作流配置正確性

```yaml
# 工作流配置範例
workflows:
  weak_signal_diagnosis:
    name: "WiFi 弱信號診斷"
    steps:
      - action: "scan_wifi_environment"
      - action: "analyze_signal_strength"
      - action: "suggest_optimization"
```

#### QoS 管理器 (QoS Manager)
**檔案位置**: `internal/qos/qos_manager.go`

網路服務品質管理：
- **異常偵測器**：識別網路性能問題
- **熱點識別器**：找出網路瓶頸位置
- **策略引擎**：生成 QoS 優化建議
- **流量分析器**：分析網路流量模式

#### 身份管理器 (Identity Manager)
**檔案位置**: `internal/identity/manager.go`

設備身份識別與管理：
- **自動發現機制**：掃描並識別新設備
- **指紋識別技術**：基於 MAC 地址、廠商資訊進行設備分類
- **製造商規則引擎**：根據 OUI 資料庫識別設備廠商

### 通訊存儲層架構

#### MQTT 客戶端 (MQTT Client)
**檔案位置**: `internal/mqtt/client.go`

 MQTT 通訊管理：
- **連接管理**：自動重連、TLS 支援
- **訊息路由**：根據主題分發訊息到對應處理器
- **Schema 驗證**：整合 Schema 驗證器確保訊息格式正確
- **日誌記錄**：記錄所有 MQTT 訊息用於除錯

**支援的主題結構**：
```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}
```

#### BuntDB 存儲 (BuntDB Storage)
**檔案位置**: `internal/storage/buntdb.go`

高效能嵌入式資料庫：
- **事務支援**：確保數據一致性
- **索引優化**：快速查詢設備和拓撲資料
- **型別轉換層**：分離業務邏輯型別和存儲型別

**存儲架構**：
- **拓撲存儲** (`topology.go`)：網路拓撲和連接關係
- **身份存儲** (`identity.go`)：設備身份和製造商資訊
- **通用存儲** (`interface.go`)：基礎存儲操作介面

#### Schema 驗證器 (Schema Validator)
**檔案位置**: `internal/schema/validator.go`

MQTT 訊息格式驗證：
- **JSON Schema 支援**：驗證訊息結構正確性
- **快取機制**：提升驗證效能
- **錯誤記錄**：詳細記錄驗證失敗原因

#### 日誌系統 (Logging System)
**檔案位置**: `internal/logging/logger.go`

結構化日誌管理：
- **應用程式日誌**：記錄系統運行狀態
- **審計日誌**：記錄重要操作和變更
- **效能日誌**：記錄系統效能指標
- **MQTT 訊息日誌**：記錄所有 MQTT 通訊

### MCP (Model Context Protocol) 整合

**檔案位置**: `internal/mcp/`

提供標準化的 AI 模型整合介面：
- **MCP 伺服器** (`server.go`)：HTTP 和 WebSocket 支援
- **資源管理** (`resources.go`)：拓撲、設備、診斷資源暴露
- **工具適配器** (`tools/adapters.go`)：LLM 工具的 MCP 格式轉換
- **工作流適配器** (`workflow_adapter.go`)：工作流的 MCP 整合

### 測試工具架構

**檔案位置**: `test_tools/`

完整的設備模擬和測試框架：
- **設備模擬器** (`pkg/simulator/`)：模擬各種 IoT 設備
- **MQTT 測試客戶端** (`pkg/mqtt/`)：測試 MQTT 通訊
- **配置管理** (`pkg/config/`)：測試場景配置

**支援的設備類型**：
- **閘道器** (`gateway.go`)：網路閘道模擬
- **交換器** (`switch.go`)：網路交換器模擬
- **IoT 感測器** (`iot_sensor.go`)：各種感測器模擬
- **網路介面** (`nic.go`)：網路介面模擬

### 型別系統設計

**檔案位置**: `pkg/types/`

統一的資料結構定義：
- **拓撲型別** (`topology.go`)：網路拓撲相關結構
- **設備型別** (`device.go`)：設備資訊結構
- **命令型別** (`command.go`)：命令執行結構
- **診斷型別** (`diagnostics.go`)：診斷結果結構
- **LLM 型別** (`llm.go`)：LLM 工具和會話結構

### 配置系統

**檔案位置**: `internal/config/`

統一的配置管理：
- **配置管理器** (`manager.go`)：支援熱重載
- **配置結構** (`config.go`)：所有配置項目定義
- **YAML 支援**：使用 YAML 格式配置檔案

**主要配置檔案**：
- `configs/controller.yaml`：主控制器配置
- `configs/workflows.yaml`：工作流定義
- `configs/mcp-server.yaml`：MCP 伺服器配置

---

### 開發建議

1. **添加新功能時**：先在 `pkg/types/` 定義資料結構，再在對應的 `internal/` 套件實作邏輯
2. **修改存儲結構時**：注意更新 `internal/storage/` 中的型別轉換函數
3. **添加 CLI 命令時**：在 `internal/cli/` 中添加命令處理器
4. **添加 LLM 工具時**：在 `internal/llm/tools_*.go` 中實作工具邏輯
5. **添加測試時**：使用 `test_tools/` 中的設備模擬器進行整合測試
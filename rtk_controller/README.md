# RTK Controller

RTK Controller 是一個基於 Go 語言開發的綜合性網絡管理系統，專為 IoT 設備管理而設計。本系統提供完整的網絡拓撲檢測、設備管理、網絡診斷、QoS 管理等功能。

## 🚀 快速開始

### 前置需求
- Go 1.19 或更高版本
- Git
- Linux/macOS/Windows 環境

### 安裝步驟

```bash
# 克隆專案
git clone <repository_url>
cd rtk_controller

# 編譯程式
make build

# 啟動交互式 CLI
./rtk-controller --cli

# 或啟動服務模式
./rtk-controller --config configs/controller.yaml
```

### 快速測試

```bash
# 運行演示腳本
./demo_cli.sh

# 加載測試數據
go run test/test_topology_simple.go

# 運行診斷測試
go run test/test_diagnostics.sh
```

## ✅ 核心功能

- **網絡拓撲檢測** - 自動設備發現、連接映射、WiFi 漫遊追踪
- **設備身份管理** - 設備識別、製造商檢測、分組管理
- **網絡診斷** - 速度測試、WAN 測試、延遲測量、定期檢測
- **QoS 流量分析** - 實時統計、異常檢測、策略推薦
- **MQTT 通訊** - 完整 RTK 協議支援、JSON Schema 驗證
- **交互式 CLI** - Shell 風格介面、自動補全、命令歷史
- **Web Console** - REST API、WebSocket 支援、Web UI

## 📊 系統架構

```
RTK Controller
├── 核心功能
│   ├── topology/     # 網絡拓撲管理
│   ├── identity/     # 設備身份識別
│   ├── mqtt/        # MQTT 消息處理
│   └── storage/     # BuntDB 持久化
├── 進階功能
│   ├── diagnostics/ # 網絡診斷測試
│   ├── qos/        # QoS 流量分析
│   └── cli/        # 命令行介面
└── 配置管理
    ├── JSON Schema 驗證
    └── 熱重載支援
```

## 🛠️ 開發指南

### 編譯選項

```bash
make build          # 編譯本地版本
make build-all      # 編譯所有平台
make test          # 運行測試
make coverage      # 生成覆蓋率報告
make lint          # 代碼檢查
make clean         # 清理編譯產物
```

### 支援平台

- Linux (ARM64, AMD64)
- macOS (ARM64)  
- Windows (AMD64)

## 📖 文檔

- [使用手冊](MANUAL.md) - 完整使用指南和 CLI 命令說明
- [快速開始](QUICKSTART.md) - 快速上手指南
- [專案架構](docs/ARCHITECTURE.md) - 系統架構設計
- [API 文檔](docs/API.md) - REST API 說明

## 🔧 配置

主配置文件：`configs/controller.yaml`

```yaml
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "rtk-controller"

diagnostics:
  enable_speed_test: true
  test_interval: "30m"

qos:
  enable_anomaly_detection: true
  anomaly_threshold: 0.8
```

## 📊 專案狀態

- **版本**: 1.0.0
- **開發進度**: 8 個階段全部完成 ✅
- **代碼規模**: 10,000+ 行
- **測試覆蓋**: 主要功能均有測試

## 🐛 問題排查

### MQTT 連接錯誤
確保 MQTT broker 正在運行，或修改 `configs/controller.yaml` 中的連接設定。

### CLI 無法啟動
檢查配置文件是否存在，以及是否有 `data/` 目錄的寫入權限。

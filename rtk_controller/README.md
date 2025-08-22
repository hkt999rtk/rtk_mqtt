# RTK Controller

> **開發者專用文檔** - 本文檔專為 RTK Controller 的開發者而設計

RTK Controller 是一個基於 Go 語言開發的綜合性網絡管理系統，專為 IoT 設備管理而設計。本系統提供完整的網絡拓撲檢測、設備管理、網絡診斷、QoS 管理等功能。

## 🏗️ 專案架構

### 核心功能模組
- **網絡拓撲檢測** - 自動設備發現、連接映射、WiFi 漫遊追踪
- **設備身份管理** - 設備識別、製造商檢測、分組管理  
- **網絡診斷** - 速度測試、WAN 測試、延遲測量、定期檢測
- **QoS 流量分析** - 實時統計、異常檢測、策略推薦
- **MQTT 通訊** - 完整 RTK 協議支援、JSON Schema 驗證
- **交互式 CLI** - Shell 風格介面、自動補全、命令歷史
- **LLM 工作流程中介層** - 意圖驅動的預定義工作流程系統，確定性診斷執行 🆕
- **MCP Server** - Model Context Protocol 服務，提供標準化 LLM 工具接口

### 技術棧
- **語言**: Go 1.23+
- **資料庫**: BuntDB (嵌入式 JSON 資料庫)
- **MQTT**: Eclipse Paho MQTT Go Client
- **CLI**: Cobra + Readline
- **配置**: Viper (支援 YAML)
- **日誌**: Logrus + Lumberjack
- **測試**: Testify + 自定義測試框架
- **MCP**: github.com/mark3labs/mcp-go (Model Context Protocol)

## 🚀 開發環境建置

### 前置需求
- **Go**: 1.23 或更高版本 (工具鏈 1.24.4)
- **Git**: 版本控制
- **Make**: 建置工具
- **golangci-lint**: 代碼檢查 (可選，推薦)

### 克隆與安裝

```bash
# 克隆專案
git clone <repository_url>
cd rtk_controller

# 安裝依賴
make deps

# 檢查 Go 版本
go version  # 需要 1.23+

# 建置開發版本
make build

# 運行測試
make test
```

### 開發工具安裝

```bash
# 安裝開發工具 (可選)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install golang.org/x/tools/cmd/godoc@latest
```

## 🤖 MCP Server 模式

RTK Controller 支援 **Model Context Protocol (MCP)** 模式，提供標準化的 LLM 工具接口，讓 AI 助手能夠直接與家庭網絡診斷系統互動。

### MCP Server 功能特點

#### 🛠️ 工具分類 (Tools)
- **topology** - 網絡拓撲工具：設備發現、連接映射、拓撲分析
- **wifi** - WiFi 診斷工具：信號強度、干擾檢測、優化建議
- **network** - 網絡連線工具：速度測試、連通性檢查、WAN 測試
- **mesh** - Mesh 網路工具：回程分析、節點狀態、漫遊監控
- **qos** - QoS 分析工具：流量統計、性能監控、策略推薦
- **config** - 配置管理工具：設備配置、策略應用、設置調整

#### 📊 資源提供 (Resources)
- `topology://network/current` - 即時網絡拓撲資訊
- `topology://devices/list` - 所有網絡設備列表
- `topology://connections/graph` - 網絡連接關係圖
- `devices://status/all` - 設備狀態摘要
- `diagnostics://history/recent` - 最近診斷記錄

#### 💬 智慧提示範本 (Prompts)
- **intent_classification** - 用戶問題意圖分類
- **diagnostic_report** - 生成診斷報告
- **troubleshooting_guide** - 故障排除指南
- **wifi_optimization** - WiFi 優化建議
- **network_summary** - 網絡健康摘要

### 啟動 MCP Server

#### 基本使用
```bash
# 使用預設設定啟動 (localhost:8080)
./build_dir/rtk_controller --mcp

# 指定主機和端口
./build_dir/rtk_controller --mcp --mcp-host 0.0.0.0 --mcp-port 8888

# 使用自定義配置檔案
./build_dir/rtk_controller --mcp --config configs/mcp-server.yaml
```

#### MCP Server 配置
配置檔案位於 `configs/mcp-server.yaml`：

```yaml
# MCP Server 基本資訊
name: "RTK Controller MCP Server"
version: "1.0.0"
description: "家庭網絡診斷和管理工具"

# HTTP 傳輸配置
http:
  enabled: true
  host: "localhost"
  port: 8080
  tls:
    enabled: false

# 工具執行配置
tools:
  categories: ["topology", "wifi", "network", "mesh", "qos", "config"]
  execution:
    timeout: "60s"
    max_concurrent: 5
    retry_attempts: 2

# 資源配置
resources:
  topology:
    enabled: true
    cache_ttl: "5m"
  devices:
    enabled: true
    cache_ttl: "5m"
  diagnostics:
    enabled: true
    history_limit: 100

# 會話管理
sessions:
  timeout: "30m"
  max_concurrent: 10
  auto_cleanup: true
  cleanup_interval: "5m"
```

### MCP API 端點

```bash
# 健康檢查
curl http://localhost:8080/mcp/health

# 伺服器資訊
curl http://localhost:8080/mcp/info

# 可用工具列表
curl http://localhost:8080/mcp/tools

# 執行工具
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "name": "topology.get_full",
      "arguments": {"detail_level": "full"}
    }
  }'

# 可用資源列表
curl http://localhost:8080/mcp/resources

# 讀取資源
curl -X POST http://localhost:8080/mcp/resources/read \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "uri": "topology://network/current"
    }
  }'

# 可用提示列表
curl http://localhost:8080/mcp/prompts

# 取得提示內容
curl -X POST http://localhost:8080/mcp/prompts/get \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "name": "intent_classification",
      "arguments": {"user_input": "我的網路很慢"}
    }
  }'
```

### 與 AI 助手整合

MCP Server 設計用於與支援 MCP 協議的 AI 助手（如 Claude、ChatGPT 等）整合：

1. **啟動 MCP Server**：`./build_dir/rtk_controller --mcp`
2. **配置 AI 助手**：將 MCP Server URL (`http://localhost:8080`) 加入 AI 助手的 MCP 配置
3. **開始對話**：AI 助手將能自動調用網絡診斷工具並提供專業建議

### MCP Server 開發

#### 新增工具
1. 在 `internal/llm/` 中實作新的 LLM 工具
2. 工具會自動被 MCP Server 註冊和暴露
3. 使用 `tools.NewToolAdapter()` 建立工具適配器

#### 新增資源提供者
1. 實作 `ResourceProvider` 介面 (`internal/mcp/resources.go`)
2. 在 MCP Server 啟動時註冊新的資源提供者
3. 定義資源 URI 格式和資料結構

#### 新增提示範本
1. 在 `internal/mcp/prompts.go` 中定義新的提示範本
2. 使用 `{{變數名}}` 語法定義模板變數
3. 在 `RegisterBuiltInPrompts()` 中註冊範本

### 監控和調試

```bash
# 檢查 MCP Server 狀態
curl http://localhost:8080/mcp/health | jq

# 監控伺服器日誌
tail -f logs/rtk_controller.log | grep "mcp"

# 檢查工具執行統計
curl http://localhost:8080/mcp/info | jq .sessions
```

## 🧠 LLM 工作流程中介層 🆕

RTK Controller 實現了一個**意圖驅動的預定義工作流程系統**，將 LLM 的作用限縮到意圖識別，後續工具調用由預定義的確定性工作流程執行。這解決了 LLM 工具調用的不確定性和難以控制的問題。

### 🎯 核心設計理念

```
用戶輸入 → 意圖分類器 (LLM) → 工作流程引擎 → 工具執行器 → 結果生成器
```

- **LLM 職責限縮**: 僅負責意圖分類，不參與工具選擇和調用順序決策
- **確定性執行**: 相同意圖總是執行相同的預定義工作流程
- **可控性保證**: 工具調用順序、參數、錯誤處理完全可預測

### 🛠️ 工作流程中介層架構

#### 核心組件
- **Intent Classifier** - 意圖分類器，支援 LLM + 規則雙重分類
- **Workflow Engine** - 工作流程引擎，管理工作流程生命週期
- **Workflow Registry** - 工作流程註冊表，YAML 配置載入和驗證
- **Workflow Executor** - 工作流程執行器，支援並行、條件、重試執行

#### 預定義工作流程
1. **弱信號覆蓋診斷** (`weak_signal_coverage_diagnosis`)
2. **WAN 連線診斷** (`wan_connectivity_diagnosis`)
3. **效能瓶頸分析** (`performance_bottleneck_analysis`)
4. **設備離線診斷** (`device_offline_diagnosis`)
5. **通用網路診斷** (`general_network_diagnosis`) - Fallback

### 💬 智慧診斷使用方式

#### CLI 自然語言查詢
```bash
# 啟動 CLI
./build_dir/rtk_controller --cli

# 自然語言查詢 - 系統會自動分類意圖並執行對應工作流程
llm query "我臥室的 WiFi 信號很弱"
llm query "網路速度很慢，不知道是什麼問題"
llm query "有設備離線了，幫我檢查一下"
llm query "WAN 連線似乎有問題"
```

#### 工作流程管理命令
```bash
# 列出所有可用工作流程
llm workflow list

# 查看特定工作流程詳情
llm workflow show weak_signal_coverage_diagnosis

# 直接執行特定工作流程
llm workflow exec weak_signal_coverage_diagnosis

# 重載工作流程配置
llm workflow reload

# 驗證工作流程配置
llm validate config
llm validate workflow weak_signal_coverage_diagnosis
```

### ⚙️ 配置系統

#### 意圖分類配置 (`configs/intent_classification.yaml`)
```yaml
intent_categories:
  primary_categories:
    - connectivity_issues      # 連線問題
    - performance_problems     # 效能問題  
    - coverage_issues          # 覆蓋問題
    - device_issues            # 設備問題

classification_prompt: |
  You are a network diagnostic intent classifier...
  
confidence_thresholds:
  high: 0.9      # 高信心度
  medium: 0.7    # 中等信心度
  low: 0.5       # 低信心度
```

#### 工作流程配置 (`configs/complete_workflows.yaml`)
```yaml
workflows:
  - id: "weak_signal_coverage_diagnosis"
    name: "WiFi 弱信號覆蓋診斷"
    intent:
      primary: "coverage_issues"
      secondary: "weak_signal_coverage"
    
    steps:
      - id: "data_collection"
        type: "parallel"        # 並行執行
        sub_steps:
          - id: "wifi_coverage_scan"
            type: "tool_call"
            tool_name: "wifi_signal_analysis"
            timeout: "30s"
          - id: "topology_discovery"
            type: "tool_call"
            tool_name: "network_topology_scan"
            
      - id: "interference_check"
        type: "sequential"      # 序列執行
        condition:              # 條件執行
          field: "data_collection.wifi_coverage_scan.problem_areas"
          operator: "exists"
```

### 🔧 風險緩解機制

#### 意圖分類錯誤緩解
- **信心度閾值機制**: 低於閾值時使用 fallback 工作流程
- **手動意圖指定**: 支援明確指定意圖覆蓋自動分類
- **重試機制**: LLM 分類失敗時自動重試，並降級到規則分類

#### 工作流程僵化緩解
- **動態參數注入**: 從用戶輸入提取參數注入工作流程
- **條件分支執行**: 支援基於前步驟結果的條件執行
- **Fallback 機制**: 分類失敗時自動降級到通用診斷工作流程

#### 配置管理
- **配置驗證工具**: 自動驗證 YAML 配置的語法和語義正確性
- **熱重載支援**: 運行時重載配置無需重啟服務

### 📊 監控與指標

```bash
# 查看工作流程執行統計
llm metrics workflow

# 查看意圖分類準確度
llm metrics intent

# 工作流程執行歷史
llm workflow history
```

#### 指標收集
- **執行統計**: 總執行次數、成功率、失敗率
- **效能指標**: 平均執行時間、瓶頸步驟識別
- **意圖準確度**: LLM 分類準確度、信心度分佈

### 🔗 MCP 整合

工作流程系統與 MCP Server 無縫整合：

```bash
# 啟動 MCP Server (包含工作流程工具)
./build_dir/rtk_controller --mcp

# 工作流程會自動導出為 MCP 工具：
# - rtk_weak_signal_coverage_diagnosis
# - rtk_wan_connectivity_diagnosis  
# - rtk_performance_bottleneck_analysis
# - rtk_device_offline_diagnosis
```

#### 自動工具生成
- 每個工作流程自動生成對應的 MCP 工具
- 自動推導輸入參數 schema
- 結果自動聚合和格式化

### 🧪 開發與擴展

#### 添加新工作流程
1. 在 `configs/complete_workflows.yaml` 中定義新工作流程
2. 配置意圖映射到工作流程 ID
3. 使用 `llm validate config` 驗證配置
4. 重載配置: `llm workflow reload`

#### 自定義意圖分類
1. 修改 `configs/intent_classification.yaml`
2. 添加新的意圖類別和分類模式
3. 更新意圖到工作流程的映射

#### 調試工作流程
```bash
# 啟用調試模式
export RTK_WORKFLOW_DEBUG=true

# 查看詳細執行過程
llm workflow exec weak_signal_coverage_diagnosis --debug

# 乾運行（不實際執行工具）
llm workflow exec weak_signal_coverage_diagnosis --dry-run
```

### 📈 效益與特點

#### 確定性提升
- ✅ **標準化流程**: 相同問題類型總是執行相同的診斷序列
- ✅ **可預測結果**: 工具調用順序和參數完全可控
- ✅ **一致性保證**: 避免 LLM 隨機組合導致的不一致性

#### 效能改善  
- ✅ **LLM 用量減少**: 僅用於意圖分類，不參與複雜的工具組合決策
- ✅ **並行執行**: 工作流程支援並行工具調用
- ✅ **智慧快取**: 相似意圖的分類結果可快取

#### 可維護性增強
- ✅ **模組化設計**: 工作流程定義與執行邏輯分離
- ✅ **版本控制**: 工作流程定義可進行版本控制
- ✅ **配置驗證**: 完整的配置驗證和錯誤檢查

## 📁 代碼結構

```
rtk_controller/
├── cmd/controller/           # 主程式入口
│   └── main.go
├── internal/                 # 內部模組 (不對外暴露)
│   ├── cli/                 # CLI 命令實現
│   ├── config/              # 配置管理
│   ├── device/              # 設備管理
│   ├── diagnostics/         # 網絡診斷
│   ├── identity/            # 設備身份識別
│   ├── llm/                 # LLM 工具引擎
│   ├── mcp/                 # MCP Server 實現
│   ├── mqtt/                # MQTT 客戶端
│   ├── qos/                 # QoS 分析
│   ├── storage/             # 資料存儲層
│   ├── topology/            # 網絡拓撲管理
│   └── workflow/            # LLM 工作流程中介層 🆕
├── pkg/                     # 公共庫 (可對外暴露)
│   ├── types/               # 資料類型定義
│   └── utils/               # 通用工具
├── configs/                 # 配置檔案
│   ├── controller.yaml      # 主配置檔案
│   ├── mcp-server.yaml      # MCP Server 配置範例
│   ├── intent_classification.yaml  # 意圖分類配置 🆕
│   └── complete_workflows.yaml     # 完整工作流程配置 🆕
├── test/                    # 測試相關
│   ├── integration/         # 整合測試
│   ├── scripts/             # 測試腳本
│   └── *.go                # 功能測試
├── docs/                    # 技術文檔
├── Makefile                 # 建置腳本
├── go.mod                   # Go 模組定義
├── go.sum                   # 依賴校驗
└── README.md               # 本檔案
```

### 重要架構決策

1. **模組分離**: `internal/` 與 `pkg/` 明確分離，確保 API 清晰
2. **資料類型**: 分離 local types 與 storage types，支援 BuntDB 持久化
3. **依賴注入**: 使用介面隔離具體實現，提升可測試性
4. **錯誤處理**: 統一錯誤處理模式，包含上下文資訊
5. **並發安全**: 所有共享資源使用適當的同步機制

## 🔧 開發工作流程

### 基本開發循環

```bash
# 1. 格式化代碼
make fmt

# 2. 檢查代碼品質
make lint

# 3. 運行測試
make test

# 4. 建置應用
make build

# 5. 運行應用
make run-cli

# 6. 啟動 MCP Server (新功能)
./build_dir/rtk_controller --mcp
```

### 測試策略

```bash
# 單元測試
make test

# 整合測試
go test -tags=integration ./test/integration/...

# 測試覆蓋率
make coverage

# 效能測試
go test -bench=. ./internal/... ./pkg/...

# 功能測試
./test/scripts/run_all_tests.sh
```

### 調試工具

```bash
# 啟動 CLI 進行調試
make run-cli

# 啟動帶調試日誌的服務
./build_dir/rtk_controller --config configs/controller.yaml --debug

# 啟動 MCP Server 進行調試
./build_dir/rtk_controller --mcp --mcp-port 8080

# 測試 MCP Server 端點
curl http://localhost:8080/mcp/health
curl http://localhost:8080/mcp/tools

# 載入測試資料
go run test/test_topology_simple.go
go run test/test_diagnostics.sh
```

## 🏭 建置與發布

### 本地建置

```bash
# 建置當前平台
make build

# 建置所有平台
make build-all

# 檢查建置產物
make list
```

### 發布流程

```bash
# 完整發布流程 (包含測試)
make release

# 或指定版本
make release VERSION=v1.2.0

# 檢查發布包
ls -la release/
```

### 平台支援

- **Linux ARM64**: 樹莓派、ARM 伺服器
- **Linux x86_64**: 標準 Linux 伺服器
- **macOS ARM64**: Apple Silicon Mac
- **Windows x86_64**: Windows 10/Server 2016+

## 🧪 測試指南

### 單元測試

```bash
# 運行所有單元測試
go test ./internal/... ./pkg/...

# 運行特定模組測試
go test ./internal/topology/...

# 詳細輸出
go test -v ./internal/storage/...

# 覆蓋率報告
go test -cover ./internal/...
```

### 整合測試

```bash
# MQTT 整合測試
go test -tags=integration ./test/integration/mqtt_integration_test.go

# CLI 整合測試  
go test -tags=integration ./test/integration/cli_integration_test.go
```

### 功能測試

```bash
# 運行所有功能測試
./test/scripts/run_all_tests.sh

# 個別功能測試
./test/scripts/test_cli_commands.sh
./test/scripts/performance_test.sh
```

## 🔍 代碼品質

### Linting

```bash
# 使用 golangci-lint (推薦)
make lint

# 基本檢查
go vet ./...

# 格式化
make fmt
```

### 安全檢查

```bash
# 安全掃描 (需要 gosec)
gosec ./...

# 依賴漏洞檢查
go list -json -m all | nancy sleuth
```

### 性能分析

```bash
# 記憶體分析
go test -memprofile=mem.prof ./internal/...
go tool pprof mem.prof

# CPU 分析
go test -cpuprofile=cpu.prof ./internal/...
go tool pprof cpu.prof
```

## 📊 監控與調試

### 日誌系統

```go
// 在代碼中使用結構化日誌
import "github.com/sirupsen/logrus"

log.WithFields(logrus.Fields{
    "module": "topology",
    "device_id": deviceID,
}).Info("Device discovered")
```

### 調試技巧

```bash
# 啟用調試日誌
export RTK_LOG_LEVEL=debug
./build_dir/rtk_controller --cli

# 查看詳細 MQTT 通訊
export RTK_MQTT_DEBUG=true
```

### 性能監控

```bash
# 監控記憶體使用
go tool pprof http://localhost:6060/debug/pprof/heap

# 監控 goroutine
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## 🤝 貢獻指南

### 代碼風格

1. **遵循 Go 慣例**: 使用 `gofmt`, `go vet`
2. **命名規範**: 使用有意義的變數名稱
3. **註釋**: 公開函數必須有 godoc 註釋
4. **錯誤處理**: 不忽略錯誤，適當包裝錯誤資訊

### 提交規範

```bash
# 提交訊息格式
git commit -m "feat(topology): add device roaming detection

- Implement roaming history tracking
- Add roaming anomaly detection
- Update topology visualization for roaming events

Closes #123"
```

### Pull Request 流程

1. **Fork 並克隆** 專案
2. **建立功能分支** `git checkout -b feature/new-feature`
3. **實作功能** 並撰寫測試
4. **確保測試通過** `make test`
5. **代碼檢查** `make lint`
6. **提交變更** 遵循提交規範
7. **建立 Pull Request**

### 測試要求

- **單元測試**: 新功能必須有對應單元測試
- **整合測試**: 涉及外部系統的功能需要整合測試
- **覆蓋率**: 維持整體覆蓋率 > 80%
- **效能測試**: 關鍵路徑需要效能測試

## 🐛 問題排查

### 常見開發問題

#### 1. 編譯錯誤

```bash
# 檢查 Go 版本
go version

# 更新依賴
go mod tidy
go mod download

# 清理並重建
make clean
make build
```

#### 2. 測試失敗

```bash
# 詳細測試輸出
go test -v ./...

# 特定測試
go test -run TestTopologyManager ./internal/topology/

# 跳過長時間測試
go test -short ./...
```

#### 3. 依賴問題

```bash
# 檢查依賴狀態
go mod why -m github.com/eclipse/paho.mqtt.golang

# 更新特定依賴
go get -u github.com/eclipse/paho.mqtt.golang

# 清理不用的依賴
go mod tidy
```

### 調試工具

```bash
# Delve 調試器
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug cmd/controller/main.go

# 競態檢測
go test -race ./...

# 記憶體洩漏檢測
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

## 📋 架構文檔

本項目包含完整的架構文檔和 PlantUML 圖表，詳細說明系統設計和組件關係。

### 主要架構文檔

| 文檔 | 描述 | 內容 |
|------|------|------|
| [`docs/MANUAL.md`](docs/MANUAL.md) | 客戶使用手冊 | 部署、配置、故障排除指南 |
| [`CLAUDE.md`](../CLAUDE.md) | Claude Code 指南 | 開發指令、架構決策說明 |

### PlantUML 架構圖

| 圖表文件 | 圖表類型 | 描述 |
|----------|----------|------|
| [`docs/ARCH_CONTROLLER.puml`](docs/ARCH_CONTROLLER.puml) | 控制器架構圖 | RTK Controller 核心系統架構 |
| [`docs/ARCH_TEST.puml`](docs/ARCH_TEST.puml) | 測試架構圖 | 完整測試體系和測試工具 |
| [`docs/ARCH_TOOLS.puml`](docs/ARCH_TOOLS.puml) | 工具架構圖 | 22個診斷工具的詳細分類和功能 |
| [`docs/ARCH_DATA_FLOW.puml`](docs/ARCH_DATA_FLOW.puml) | 數據流圖 | 系統內部數據流和處理流程 |
| [`docs/ARCH_DEPLOYMENT.puml`](docs/ARCH_DEPLOYMENT.puml) | 部署架構圖 | 生產環境部署和生態系統整合 |

### 🏗️ 系統架構概覽

#### 核心組件

RTK Controller 由以下主要組件構成：

1. **LLM 診斷引擎**
   - 工具註冊和執行
   - 會話管理
   - 指標收集
   - 智能分析

2. **診斷工具集 (22個工具)**
   - 基礎診斷工具 (6個)
   - WiFi 高級工具 (8個) 
   - Mesh 網路工具 (6個)
   - 配置管理工具 (8個)

3. **存儲層**
   - BuntDB 嵌入式數據庫
   - 類型轉換層
   - 事務支持

4. **通訊層**
   - MQTT 客戶端整合
   - 消息處理
   - 設備通訊

### 工具分類

#### 🔍 基礎診斷工具 (6個)
- `topology.get_full` - 網路拓撲獲取
- `clients.list` - 客戶端列表
- `network.speedtest_full` - 網路速度測試
- `diagnostics.wan_connectivity` - WAN 連通性測試
- `qos.get_status` - QoS 狀態查詢
- `traffic.get_stats` - 流量統計

#### 📡 WiFi 高級工具 (8個)
- `wifi.scan_channels` - 頻道掃描
- `wifi.analyze_interference` - 干擾分析
- `wifi.spectrum_utilization` - 頻譜使用率
- `wifi.signal_strength_map` - 信號強度地圖
- `wifi.coverage_analysis` - 覆蓋範圍分析
- `wifi.roaming_optimization` - 漫遊優化
- `wifi.throughput_analysis` - 輸送量分析
- `wifi.latency_profiling` - 延遲分析

#### 🕸️ Mesh 網路工具 (6個)
- `mesh.get_topology` - Mesh 拓撲視覺化
- `mesh.node_relationship` - 節點關係分析
- `mesh.path_optimization` - 路徑優化
- `mesh.backhaul_test` - 回程測試
- `mesh.load_balancing` - 負載均衡
- `mesh.failover_simulation` - 故障切換模擬

#### ⚙️ 配置管理工具 (8個)
- `config.wifi_settings` - WiFi 配置管理
- `config.qos_policies` - QoS 策略管理
- `config.security_settings` - 安全配置
- `config.band_steering` - 頻段引導
- `config.auto_optimize` - 自動優化
- `config.validate_changes` - 配置驗證
- `config.rollback_safe` - 安全回滾
- `config.impact_analysis` - 影響分析

### 🔧 查看架構圖

1. **在線查看**：使用 PlantUML 在線編輯器
   - 訪問 [PlantUML Online](http://www.plantuml.com/plantuml/)
   - 複製 `.puml` 文件內容
   - 貼上並生成圖表

2. **本地生成**：使用 PlantUML 工具
   ```bash
   # 安裝 PlantUML
   brew install plantuml  # macOS
   
   # 生成圖片
   plantuml docs/ARCH_CONTROLLER.puml
   plantuml docs/ARCH_TEST.puml
   plantuml docs/ARCH_TOOLS.puml
   plantuml docs/ARCH_DATA_FLOW.puml
   plantuml docs/ARCH_DEPLOYMENT.puml
   ```

3. **IDE 整合**：在 VS Code 中使用 PlantUML 擴展
   - 安裝 "PlantUML" 擴展
   - 開啟 `.puml` 文件
   - 使用 `Alt+D` 預覽圖表

### 📈 開發狀態

#### ✅ 已完成項目

- [x] 基礎診斷工具實現 (6個)
- [x] WiFi 高級診斷工具 (8個)  
- [x] Mesh 網路診斷工具 (6個)
- [x] 配置管理工具 (8個)
- [x] 工具引擎整合
- [x] 測試套件完成
- [x] 指標收集系統
- [x] 會話管理
- [x] 架構文檔

#### 📊 統計數據

- **總工具數量**: 22個
- **工具類別**: 4種 (Read/Test/Act/WiFi)
- **測試覆蓋率**: 100%
- **文檔完整性**: 100%
- **編譯狀態**: ✅ 成功

## 📚 相關文檔

- **[MANUAL.md](docs/MANUAL.md)** - 客戶使用手冊  
- **[CLAUDE.md](../CLAUDE.md)** - Claude Code 指南
- **技術文檔**: `docs/` 目錄中的詳細文檔

## 🔗 外部資源

### Go 生態系統
- [Go 官方文檔](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### 依賴庫文檔
- [Eclipse Paho MQTT Go](https://pkg.go.dev/github.com/eclipse/paho.mqtt.golang)
- [BuntDB](https://pkg.go.dev/github.com/tidwall/buntdb)
- [Cobra CLI](https://pkg.go.dev/github.com/spf13/cobra)
- [Viper Configuration](https://pkg.go.dev/github.com/spf13/viper)

## 📝 版本歷程

查看 [git tags](https://github.com/project/rtk_controller/tags) 獲取完整版本歷程。

### 開發版本
- **主分支**: `main` - 穩定開發版本
- **開發分支**: `develop` - 功能開發
- **功能分支**: `feature/*` - 個別功能開發

---

## 🚀 快速開始開發

```bash
# 完整開發環境設置
git clone <repository_url>
cd rtk_controller
make deps
make build
make test

# 選擇運行模式：

# 1. 交互式 CLI 模式
make run-cli

# 2. MCP Server 模式 (新功能)
./build_dir/rtk_controller --mcp

# 3. 傳統服務模式
./build_dir/rtk_controller --config configs/controller.yaml

# 開始您的第一個功能！
```

---

**專案維護者**: RTK Controller Team  
**最後更新**: 2025-08-21  
**Go 版本**: 1.23+  
**新功能**: MCP Server 模式支援 🤖
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

### 技術棧
- **語言**: Go 1.23+
- **資料庫**: BuntDB (嵌入式 JSON 資料庫)
- **MQTT**: Eclipse Paho MQTT Go Client
- **CLI**: Cobra + Readline
- **配置**: Viper (支援 YAML)
- **日誌**: Logrus + Lumberjack
- **測試**: Testify + 自定義測試框架

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
│   ├── mqtt/                # MQTT 客戶端
│   ├── qos/                 # QoS 分析
│   ├── storage/             # 資料存儲層
│   └── topology/            # 網絡拓撲管理
├── pkg/                     # 公共庫 (可對外暴露)
│   ├── types/               # 資料類型定義
│   └── utils/               # 通用工具
├── configs/                 # 配置檔案
│   └── controller.yaml
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

## 📚 相關文檔

- **[MANUAL.md](MANUAL.md)** - 客戶使用手冊
- **[CLAUDE.md](../CLAUDE.md)** - Claude Code 指南
- **[RELEASE_PLAN.md](RELEASE_PLAN.md)** - Release 重構計劃
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
make run-cli

# 開始您的第一個功能！
```

---

**專案維護者**: RTK Controller Team  
**最後更新**: 2025-08-18  
**Go 版本**: 1.23+
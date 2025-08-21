# RTK Controller LLM 診斷系統架構文檔

## 📋 概述

RTK Controller 的 LLM 診斷系統是一個企業級的智能網路診斷平台，提供基於 LLM (Large Language Model) 的自動化網路診斷和問題解決能力。該系統通過工具化的方式將複雜的網路診斷操作封裝為 LLM 可調用的工具，實現人工智能輔助的網路運維。

### 🎯 系統目標

- **智能診斷**: 提供 LLM 可調用的網路診斷工具集
- **會話管理**: 支援複雜的診斷會話和狀態追蹤
- **變更控制**: 實現安全的配置變更和回滾機制
- **可擴展性**: 支援插件化的工具擴展
- **企業就緒**: 提供生產級的監控、安全和可靠性

---

## 🏗️ 系統架構

### 核心組件關係圖

```
┌─────────────────────────────────────────────────────────────┐
│                    RTK Controller                           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   CLI Interface │  │   API Gateway   │  │  Web Portal  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│                 LLM Diagnosis Manager                       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │  Tool Engine    │  │Session Manager  │  │ Changeset Mgr│ │
│  │   Core System   │  │   & Tracing     │  │  & Rollback  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│              Diagnostic Tool Categories                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   READ Tools    │  │   TEST Tools    │  │   ACT Tools  │ │
│  │ • Topology      │  │ • Speed Test    │  │ • WiFi Config│ │
│  │ • QoS Status    │  │ • Connectivity  │  │ • QoS Policy │ │
│  │ • Device Info   │  │ • Performance   │  │ • Mesh Setup │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│               Existing RTK Systems                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │ Topology Manager│  │   QoS Manager   │  │Command Manager│ │
│  │ • Device Disc.  │  │ • Traffic Anal. │  │ • MQTT Client│ │
│  │ • Connection    │  │ • Anomaly Det.  │  │ • ACK Track  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │  Storage Layer  │  │  Identity Mgr   │  │ Config Mgr   │ │
│  │ • BuntDB        │  │ • Device ID     │  │ • Hot Reload │ │
│  │ • Persistence   │  │ • Capabilities  │  │ • Validation │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 數據流向圖

```
LLM/API Client
      │
      ▼
┌──────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ CLI/API      │───▶│ Diagnosis       │───▶│ Tool Engine     │
│ Interface    │    │ Manager         │    │ Core            │
└──────────────┘    └─────────────────┘    └─────────────────┘
                            │                       │
                            ▼                       ▼
                    ┌─────────────────┐    ┌─────────────────┐
                    │ Session         │    │ Tool Registry   │
                    │ Management      │    │ & Validation    │
                    └─────────────────┘    └─────────────────┘
                            │                       │
                            ▼                       ▼
                    ┌─────────────────┐    ┌─────────────────┐
                    │ Changeset       │    │ Tool Execution  │
                    │ Management      │    │ & Results       │
                    └─────────────────┘    └─────────────────┘
                            │                       │
                            ▼                       ▼
                    ┌─────────────────────────────────────────┐
                    │        Existing RTK Systems             │
                    │ • Topology  • QoS  • Command  • MQTT   │
                    └─────────────────────────────────────────┘
```

---

## 🔧 核心組件詳解

### 1. LLM Tool Engine (`internal/llm/tool_engine.go`)

**職責**: LLM 工具執行的核心引擎

**核心功能**:
- **工具註冊與發現**: 管理所有可用的診斷工具
- **會話生命週期**: 創建、管理和清理診斷會話
- **工具執行**: 協調工具的參數驗證和執行
- **結果管理**: 收集、格式化和持久化工具執行結果
- **並發控制**: 管理多個並發診斷會話

**關鍵類型**:
```go
type ToolEngine struct {
    tools           map[string]types.LLMTool     // 工具註冊表
    sessions        map[string]*types.LLMSession // 活躍會話
    storage         storage.Storage              // 持久化存儲
    commandManager  *command.Manager             // 命令管理器
    topologyManager *topology.Manager            // 拓撲管理器
    qosManager      *qos.QoSManager             // QoS 管理器
    config          *EngineConfig               // 引擎配置
}
```

**配置參數**:
```go
type EngineConfig struct {
    SessionTimeout        time.Duration // 會話超時時間 (預設 30 分鐘)
    ToolTimeout          time.Duration // 工具超時時間 (預設 60 秒)
    MaxConcurrentSessions int          // 最大並發會話 (預設 10)
    EnableTracing        bool         // 啟用分散式追蹤
}
```

### 2. 診斷工具分類 (`internal/llm/tools_*.go`)

#### 2.1 READ 工具 (查詢類)
**文件**: `tools_topology.go`, `tools_network.go`

**已實作工具**:
- `topology.get_full`: 獲取完整網路拓撲
- `clients.list`: 列出所有網路客戶端
- `qos.get_status`: 獲取 QoS 狀態
- `traffic.get_stats`: 獲取流量統計

**特性**:
- 只讀操作，無副作用
- 快速響應 (< 2 秒)
- 支援參數化查詢
- 結果快取機制

#### 2.2 TEST 工具 (測試類)
**文件**: `tools_testing.go`

**已實作工具**:
- `network.speedtest_full`: 綜合網路速度測試
- `diagnostics.wan_connectivity`: WAN 連通性測試

**特性**:
- 主動網路測試
- 較長執行時間 (10-60 秒)
- 詳細測試報告
- 支援並發測試

#### 2.3 ACT 工具 (操作類)
**狀態**: 計劃中，第二階段實作

**計劃工具**:
- `config.wifi_settings`: WiFi 配置變更
- `config.qos_policies`: QoS 策略更新
- `config.security_settings`: 安全配置管理

### 3. 會話管理系統

#### 3.1 會話生命週期
```go
type LLMSession struct {
    SessionID string              // 唯一會話標識
    TraceID   string              // 分散式追蹤標識
    Status    LLMSessionStatus    // 會話狀態
    ToolCalls []ToolCall          // 工具調用序列
    CreatedAt time.Time           // 創建時間
    UpdatedAt time.Time           // 更新時間
    DeviceID  string              // 目標設備 (可選)
    UserID    string              // 用戶標識
    Metadata  map[string]interface{} // 會話元數據
}
```

#### 3.2 會話狀態管理
- **Active**: 會話活躍，可執行工具
- **Completed**: 會話正常完成
- **Failed**: 會話執行失敗
- **Cancelled**: 會話被取消

#### 3.3 自動清理機制
- 超時會話自動清理 (預設 30 分鐘)
- 定期清理過期數據
- 記憶體使用優化

### 4. 變更集管理 (`internal/changeset/simple_manager.go`)

**職責**: 管理配置變更的事務性操作

**核心功能**:
- **變更追蹤**: 記錄所有配置變更操作
- **事務支援**: 支援原子性的批量變更
- **回滾機制**: 安全的配置回滾
- **審計日誌**: 完整的變更歷史記錄

**變更集生命週期**:
```go
type Changeset struct {
    ID                string            // 變更集標識
    Status            ChangesetStatus   // 變更集狀態
    Commands          []*Command        // 變更命令列表
    RollbackCommands  []*Command        // 回滾命令列表
    Results           []*CommandResult  // 執行結果
    CreatedAt         time.Time         // 創建時間
    ExecutedAt        *time.Time        // 執行時間
    RolledBackAt      *time.Time        // 回滾時間
}
```

---

## 🔌 工具開發規範

### 工具介面定義
```go
type LLMTool interface {
    Name() string                                              // 工具名稱
    Category() ToolCategory                                    // 工具分類
    Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) // 執行工具
    Validate(params map[string]interface{}) error              // 參數驗證
    RequiredCapabilities() []string                            // 所需設備能力
    Description() string                                       // 工具描述
}
```

### 工具命名規範
- **格式**: `<category>.<action>[_<modifier>]`
- **範例**:
  - `topology.get_full`: 獲取完整拓撲
  - `network.speedtest_full`: 全面速度測試
  - `config.wifi_settings`: WiFi 配置

### 工具開發最佳實踐
1. **參數驗證**: 嚴格驗證輸入參數
2. **錯誤處理**: 使用結構化錯誤資訊
3. **超時控制**: 遵守工具超時限制
4. **結果格式**: 統一的結果數據結構
5. **日誌記錄**: 詳細的執行日誌

---

## 🔐 安全和權限

### 當前安全機制
1. **參數驗證**: 嚴格的輸入驗證
2. **會話隔離**: 會話間數據隔離
3. **超時保護**: 防止資源耗盡
4. **錯誤限制**: 防止信息洩露

### 計劃中的安全增強 (階段 3)
1. **RBAC 權限控制**: 基於角色的訪問控制
2. **操作審計**: 完整的操作審計日誌
3. **數據加密**: 敏感數據加密存儲
4. **API 認證**: 多因子身份認證

---

## 📊 監控和可觀測性

### 當前監控指標
- 工具執行次數和成功率
- 會話創建和關閉統計
- 執行時間分佈
- 錯誤率統計

### 計劃中的監控增強 (階段 1.3)
1. **Metrics 收集**: Prometheus 格式指標
2. **分散式追蹤**: OpenTelemetry 整合
3. **健康檢查**: HTTP 健康檢查端點
4. **告警機制**: 基於指標的告警

---

## 🚀 性能特性

### 當前性能指標
- **工具執行時間**: < 2s (READ 工具), < 60s (TEST 工具)
- **並發會話支援**: 10+ 並發會話
- **記憶體使用**: < 100MB 增長 (1000 次執行)
- **響應時間**: < 500ms (API 調用)

### 性能優化策略
1. **結果快取**: 智能結果快取機制
2. **並發執行**: 工具並行執行
3. **資源池**: 連接和資源池管理
4. **數據壓縮**: 大型結果數據壓縮

---

## 🔄 數據持久化

### 存儲架構
```
BuntDB (嵌入式數據庫)
├── llm_session:{session_id}     # LLM 會話數據
├── changeset:{changeset_id}     # 變更集數據
├── tool_result:{result_id}      # 工具執行結果
├── device_capabilities:{dev_id} # 設備能力快取
└── metrics:{timestamp}          # 性能指標數據
```

### 數據生命週期
- **會話數據**: 30 分鐘自動清理
- **變更集數據**: 30 天保留期
- **工具結果**: 7 天保留期
- **指標數據**: 90 天滾動保留

---

## 🔮 未來發展路線

### 階段 2: 功能擴展 (6 週)
1. **WiFi 高級診斷**: 頻譜分析、干擾檢測
2. **Mesh 網路支援**: 多層 Mesh 拓撲診斷
3. **主動配置管理**: 智能配置優化

### 階段 3: 企業增強 (4 週)
1. **安全框架**: RBAC、審計、加密
2. **插件系統**: 第三方工具整合
3. **API 生態**: REST、GraphQL、gRPC

### 長期願景
1. **AI 輔助決策**: 基於歷史數據的智能建議
2. **自動化運維**: 無人值守的網路維護
3. **多租戶支援**: 企業級部署架構
4. **雲原生整合**: Kubernetes、微服務架構

---

## 📚 相關文檔

- **[PLAN_A.md](../discussion/PLAN_A.md)**: 完整發展計劃
- **[LLM_TOOLS_API_REFERENCE.md](LLM_TOOLS_API_REFERENCE.md)**: 工具 API 參考 (計劃中)
- **[LLM_USER_GUIDE.md](LLM_USER_GUIDE.md)**: 用戶使用指南 (計劃中)
- **[CLAUDE.md](../CLAUDE.md)**: Claude Code 開發指南

---

**文檔版本**: v1.0  
**最後更新**: 2025-08-20  
**更新者**: Claude Code Assistant

---

*這份架構文檔描述了 RTK Controller LLM 診斷系統的完整架構設計，為開發者和運維人員提供全面的技術參考。*
# 🏠 家用網路診斷系統 - RTK Controller MCP 改造計劃

## 📋 專案概述

將現有 RTK Controller 改造成支援 LLM 的智能網路診斷系統。系統採用**常駐服務架構**，透過 **HTTP/gRPC API** 為 Go LLM Client 提供網路診斷工具，結合**雙向 MQTT 協議**調用遠端設備功能，實現 Read/Test/Act 三層工具的自動化網路診斷與修復。

**改造範圍**: 現有 RTK Controller 專案  
**開發語言**: Go (重用現有架構)  
**通訊協議**: HTTP/gRPC API + MQTT (重用 `github.com/eclipse/paho.mqtt.golang`)  
**核心特色**: 常駐服務 + API 調用 + 雙向 MQTT + LLM 工具整合

---

## 🎯 核心目標

- **設計完整的 MQTT 協議規範 (PROTOCOL.md)，供各部門協同開發**
- 改造 RTK Controller 為常駐服務，提供 HTTP/gRPC API 標準化工具介面  
- 實現雙向 MQTT 調用：Controller ↔ Network devices
- 整合本地診斷工具 (ping, speedtest) 與遠端設備功能
- 支援 9 種常見網路問題的自動化診斷流程
- 提供安全的網路參數調整功能 (dry-run/rollback)

---

## 🏗️ 系統架構

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│                 │    │                  │    │                 │
│  Go LLM Client  │◄──►│ RTK Controller   │◄──►│  RTK MQTT       │
│  (你們自製)      │    │ (常駐 Daemon)     │    │  Broker         │
│  • HTTP Client  │    │ • HTTP Server    │    │                 │
│  • gRPC Client  │    │ • gRPC Server    │    └─────────────────┘
│  • Tool calls   │    │ • 狀態維護        │             │
│  • Result proc  │    │ • MQTT 連接池     │             ▼
└─────────────────┘    │ • Session mgmt   │    ┌──────────────────┐
        │               └──────────────────┘    │  MQTT Messages   │
        │                        │               │  (雙向調用協議)    │
        ▼                        ▼               └──────────────────┘
┌─────────────────┐    ┌──────────────────┐              │
│   API Gateway   │    │  Local Tools     │              ▼
│  • /api/v1/     │    │  • ping/iperf    │    ┌─────────────────┐
│  • Auth/RBAC    │    │  • curl/speedtest│    │  Home Network   │
│  • Rate limit   │    │  • system tools  │    │  Devices        │
│  • Audit log    │    └──────────────────┘    │  • Router/AP    │
└─────────────────┘                            │  • Switch/Mesh  │
                                               │  • IoT devices  │
                                               └─────────────────┘
```

---

## 📦 開發組件清單

**註**: 基於現有 RTK Controller 架構進行改造，重用現有 MQTT 客戶端、storage、CLI 等組件。

### Phase 1: 協議設計與 API 規範 (Week 1-2) 

#### 1.1 MQTT Payload 協議設計 (`docs/MQTT_PROTOCOL_v2.md`)

**目標**: 設計完整的 Controller ↔ Device MQTT 調用協議，供各部門協同開發

**Controller 端需求 (我們負責)**:
```go
// Controller 發送的遠端調用
type RemoteCommand struct {
    ID       string                 `json:"id"`         // 唯一請求ID
    Op       string                 `json:"op"`         // 操作類型 
    Schema   string                 `json:"schema"`     // 協議版本
    Args     map[string]interface{} `json:"args"`       // 操作參數
    TraceID  string                 `json:"trace_id"`   // 分散式追蹤
    Timeout  int                    `json:"timeout_ms"` // 超時設定
    DryRun   bool                   `json:"dry_run,omitempty"` // 預覽模式
}
```

**Device 端需求 (其他部門負責)**:
- WiFi AP / Router 設備
- IoT 終端設備  
- Network Interface Card (NIC)
- Mesh 節點設備

#### 1.2 API 規範設計 (`docs/API_SPEC.md`)

**目標**: 設計完整的 HTTP/gRPC API 規範，定義 Go LLM Client 與 RTK Controller 的介面

**API 設計規範**:
1. **RESTful HTTP API**:
   ```
   POST /api/v1/tools/{tool_name}/execute
   GET  /api/v1/tools/{tool_name}/schema  
   GET  /api/v1/sessions/{session_id}/status
   POST /api/v1/sessions/{session_id}/rollback
   GET  /api/v1/health
   ```

2. **gRPC Service Definition**:
   ```protobuf
   service RTKController {
     rpc ExecuteTool(ToolRequest) returns (ToolResponse);
     rpc GetToolSchema(ToolSchemaRequest) returns (ToolSchemaResponse);
     rpc CreateSession(SessionRequest) returns (SessionResponse);
     rpc GetHealth(HealthRequest) returns (HealthResponse);
   }
   ```

3. **Go LLM Client Interface**:
   ```go
   type RTKNetworkClient interface {
     ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (*ToolResult, error)
     CreateDiagnosticSession(ctx context.Context, intent string) (*DiagnosticSession, error)
     GetAvailableTools(ctx context.Context) ([]ToolDefinition, error)
   }
   ```

#### 1.3 MQTT 協議文檔輸出 (`docs/PROTOCOL.md`)

**目標**: 產出完整的 MQTT 協議規範文檔，供各部門協同開發使用

**文檔內容規劃**:
1. **協議概述與版本**
   - 基於現有 SPEC.md 擴展
   - 新增 Controller ↔ Device 雙向調用規範
   
2. **設備分類與各部門分工**:
   - WiFi AP/Router (網路設備部)  
   - IoT 終端設備 (IoT 部門)
   - NIC/Driver (系統軟體部)
   - Mesh 節點 (網路設備部)

3. **操作分類與 Payload 定義**:
   - **Read 操作**: `topology.*`, `wifi.get_*`, `clients.*` 
   - **Test 操作**: `network.ping`, `network.speedtest`, `wifi.roam_probe`
   - **Act 操作**: `wifi.set_*`, `mesh.set_*` (需 dry_run 支援)

4. **錯誤處理與回應格式**
5. **設備能力發現機制**  
6. **安全與權限控制**
7. **測試與驗證指南**

**交付物**: 
- `docs/API_SPEC.md` - HTTP/gRPC API 完整規範
- `docs/PROTOCOL.md` - MQTT 協議規範  
- `api/proto/` - Protocol Buffer 定義檔案
- `examples/go-client/` - Go LLM Client 使用範例
- `schemas/` - JSON Schema 驗證檔案

#### 1.4 Session 管理架構設計 (`internal/session/`)

**目標**: 設計多會話併發管理機制，支援診斷過程的狀態追蹤

```go
type SessionManager interface {
    CreateSession(ctx context.Context, req *SessionRequest) (*Session, error)
    GetSession(sessionID string) (*Session, error)
    UpdateSession(sessionID string, update *SessionUpdate) error
    CloseSession(sessionID string) error
    ListActiveSessions() ([]*Session, error)
}

type Session struct {
    ID            string
    UserID        string  
    Intent        string
    State         SessionState  // active, suspended, completed, failed
    Tools         []ToolExecution
    CreatedAt     time.Time
    LastActiveAt  time.Time
    Metadata      map[string]interface{}
    
    // 診斷上下文
    DiagnosticCtx *DiagnosticContext
    ChangeSet     *ChangeSet  // for Act operations
}
```

#### 1.2 雙向 MQTT 指令擴展 (`internal/mqtt/remote_command.go`)
```go
// 擴展現有 MQTT 客戶端支援遠端指令調用
type RemoteCommandClient struct {
    client      *mqtt.Client  // 重用現有實作
    pendingCmds map[string]chan *CommandResponse
    timeout     time.Duration
    mu          sync.RWMutex
}

// 遠端指令格式 (基於現有 SPEC.md)
type RemoteCommand struct {
    ID       string                 `json:"id"`
    Op       string                 `json:"op"`      // 操作名稱
    Schema   string                 `json:"schema"`  
    Args     map[string]interface{} `json:"args"`
    TraceID  string                 `json:"trace_id"`
    Timeout  int                    `json:"timeout_ms"`
    DryRun   bool                   `json:"dry_run,omitempty"`
}
```

#### 1.3 工具分類架構 (`internal/mcp/tools/`)
```go
// 工具執行分類
type ToolCategory int

const (
    LocalTool   ToolCategory = iota  // Controller 本機執行
    RemoteTool                       // 透過 MQTT 調用設備
    HybridTool                       // 本機+遠端結合
)

// 本機工具：重用 internal/diagnostics
type LocalTool struct {
    name        string
    diagnostics *diagnostics.NetworkDiagnostics
}

// 遠端工具：透過 MQTT 調用
type RemoteTool struct {
    name      string
    operation string
    client    *RemoteCommandClient
}
```

### Phase 2: Read 工具實作 (Week 3-4)

#### 2.1 拓撲與資產工具 (`internal/mcp/tools/topology/`)
- **`net.get_topology()`** (Hybrid 工具)
  - **本機部分**: 從現有 storage 讀取拓撲快取
  - **遠端部分**: 發送 `topology.discover` 命令刷新資料
  - **MQTT 調用**: 
    ```json
    {
      "op": "topology.discover", 
      "args": {"discovery_type": "full", "include_inactive": false}
    }
    ```
  - **資料整合**: 合併本機快取與遠端最新資料

- **`wifi.get_radios()`** (Remote 工具)
  - **MQTT 調用**:
    ```json
    {
      "op": "diagnosis.get",
      "args": {"type": "wifi.radio_status", "detail_level": "full"}
    }
    ```
  - **設備實作需求**: RF 狀態、功率、信道、DFS 狀態、鄰居干擾

- **`clients.list()`** (Hybrid 工具)
  - **本機部分**: 從現有設備管理器讀取客戶端快取
  - **遠端部分**: 查詢最新客戶端狀態
  - **MQTT 調用**: 
    ```json
    {
      "op": "clients.get_all",
      "args": {"include_history": true, "active_only": false}
    }
    ```
  - **整合資料**: RSSI、SNR、PHY 速率、漫遊歷史、驅動資訊

#### 2.2 無線現況工具 (`pkg/tools/wifi/`)
- **`wifi.survey()`**: WiFi 環境掃描
  - 發送命令: `diagnosis.get` with `type: "wifi.survey"`
  - 分析鄰居 AP、信道重疊、干擾源

- **`wifi.utilization()`**: 空口利用率分析
  - 收集各 BSS 的利用率、重傳率統計

#### 2.3 網路連通性工具 (`pkg/tools/network/`)
- **`dhcpdns.get_config()`**: DHCP/DNS 配置檢查
  - 收集 DHCP 設定、租約資訊
  - 檢測 Rogue DHCP、IP 衝突

### Phase 3: Test 工具實作 (Week 5-6)

#### 3.1 主動測試工具 (`pkg/tools/testing/`)
- **`net.ping()`**: 網路延遲測試
  - 發送 ping 測試命令到指定設備
  - 測量延遲、抖動、丟包率

- **`net.speedtest()`**: 頻寬速度測試
  - 支援 router-side 和 client-side 測試
  - 比對 WAN 端與內網端性能差異

- **`mesh.backhaul_test()`**: Mesh 回程測試
  - 測試 Mesh 節點間的回程品質
  - 評估有線/無線回程性能

#### 3.2 WiFi 特定測試 (`pkg/tools/wifi/`)
- **`wifi.roam_probe()`**: 漫遊測試
  - 主動觸發客戶端漫遊
  - 測量漫遊延遲與成功率

### Phase 4: Act 工具實作 (Week 7-8)

#### 4.1 安全控制框架 (`pkg/tools/actions/`)
```go
type ActionTool struct {
    BaseTool
    DryRunSupport bool
    RiskLevel     RiskLevel  // Low, Medium, High
    ApprovalRequired bool
}

type ChangeSet struct {
    ID          string
    Actions     []Action
    DryRunResult *DryRunResult
    AppliedAt   time.Time
    RollbackData map[string]interface{}
}
```

#### 4.2 WiFi 調整工具
- **`wifi.set_power()`**: RF 功率調整
  - 支援 dry-run 模式預覽影響範圍
  - 自動建議最佳功率值

- **`wifi.set_channel()`**: 信道調整
  - DFS 信道檢查
  - 干擾迴避建議

- **`wifi.set_roaming()`**: 漫遊參數調整
  - 啟用 802.11r/k/v
  - 調整 RSSI 閾值

#### 4.3 回滾機制
```go
type RollbackManager interface {
    CreateChangeSet(actions []Action) (*ChangeSet, error)
    ApplyChangeSet(changeSetID string, approvalToken string) error
    RollbackChangeSet(changeSetID string) error
    GetChangeHistory() ([]*ChangeSet, error)
}
```

### Phase 5: Go LLM Client 設計與 Intent 處理 (Week 9-10)

#### 5.1 Go LLM Client 架構 (`go-llm-client/`)
```go
// RTK Network Diagnostics Client
type RTKClient struct {
    httpClient   *http.Client
    grpcConn     *grpc.ClientConn
    config       *ClientConfig
    toolRegistry *ToolRegistry
    sessionMgr   *SessionManager
}

// 工具調用介面
type Tool interface {
    Name() string
    Description() string
    Parameters() []Parameter
    Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}

// 診斷流程編排
type DiagnosticFlow struct {
    Intent    string              `json:"intent"`
    Steps     []DiagnosticStep    `json:"steps"`
    Results   []StepResult        `json:"results"`
    Summary   *DiagnosticSummary  `json:"summary"`
}
```

#### 5.2 Intent 分類器 (`internal/intent/`)
```go
type IntentClassifier interface {
    ClassifyIntent(userInput string) (*Intent, float64, error)
    GetSupportedIntents() []IntentDefinition
}

type Intent struct {
    Type        string    // no_internet, slow_speed, etc.
    Confidence  float64
    Parameters  map[string]interface{}
    Utterance   string
}
```

#### 5.3 診斷編排器 (`internal/orchestrator/`)
```go
type DiagnosticOrchestrator interface {
    ExecuteDiagnostic(intent *Intent) (*DiagnosticReport, error)
    GetToolChain(intentType string) ([]Tool, error)
}

type DiagnosticReport struct {
    Intent      *Intent           `json:"intent"`
    Findings    []Finding         `json:"findings"`
    RootCause   *RootCause       `json:"root_cause"`
    Recommendations []Recommendation `json:"recommendations"`
    FollowUp    []FollowUpAction `json:"follow_up"`
    Confidence  float64          `json:"confidence"`
}
```

#### 5.3 API Server 介面 (`internal/api/`)
```go
// HTTP/gRPC API Server，提供工具給 Go LLM Client 調用
type APIServer struct {
    httpServer   *http.Server
    grpcServer   *grpc.Server
    tools        map[string]Tool
    orchestrator DiagnosticOrchestrator
    mqttClient   MQTTClient
    sessions     SessionManager
}

// HTTP API Handlers
type HTTPHandler struct {
    orchestrator DiagnosticOrchestrator
    authService  AuthService
    logger       Logger
}

// gRPC Service Implementation
type GRPCService struct {
    orchestrator DiagnosticOrchestrator
    authService  AuthService
    logger       Logger
    pb.UnimplementedRTKControllerServer
}
```

### Phase 6: 可觀測性與監控 (Week 11-12)

#### 6.1 分散式系統監控 (`internal/observability/`)

**目標**: 實現完整的系統可觀測性，支援生產環境運維

```go
// Metrics 收集
type MetricsCollector interface {
    RecordAPILatency(endpoint string, duration time.Duration)
    RecordMQTTMessageCount(topic string, messageType string)
    RecordToolExecutionResult(toolName string, success bool, duration time.Duration)
    RecordSessionLifecycle(event SessionEvent)
}

// Distributed Tracing
type TracingService interface {
    StartSpan(ctx context.Context, operationName string) (context.Context, trace.Span)
    AddSpanTags(span trace.Span, tags map[string]interface{})
    LogSpanError(span trace.Span, err error)
}

// 健康檢查
type HealthChecker interface {
    CheckMQTTConnection() HealthStatus
    CheckDatabaseConnection() HealthStatus  
    CheckDeviceConnectivity() map[string]HealthStatus
    GetSystemHealth() *SystemHealthReport
}
```

**監控指標設計**:
- API 延遲和吞吐量 (P50, P95, P99)
- MQTT 連接狀態和訊息處理速度
- 工具執行成功率和耗時分佈
- 設備在線狀態和響應時間
- 系統資源使用率 (CPU, Memory, Disk)

#### 6.2 錯誤處理與恢復機制

**目標**: 設計完善的錯誤處理策略，提升系統可靠性

```go
// 錯誤分類和處理
type ErrorHandler interface {
    HandleAPIError(ctx context.Context, err error) *APIErrorResponse
    HandleMQTTError(ctx context.Context, err error) error
    HandleToolExecutionError(ctx context.Context, toolName string, err error) *ToolErrorResult
}

// 重試機制
type RetryPolicy struct {
    MaxAttempts   int
    InitialDelay  time.Duration
    MaxDelay      time.Duration
    BackoffFactor float64
    RetryableErrors []error
}

// 斷路器模式
type CircuitBreaker interface {
    Execute(ctx context.Context, operation func() error) error
    GetState() CircuitState  // closed, open, half-open
    GetMetrics() CircuitMetrics
}
```

### Phase 7: 測試與優化 (Week 11-12)

#### 7.1 測試策略

**單元測試**:
- API Handler 單元測試 (HTTP/gRPC)  
- 工具執行邏輯測試 (Local/Remote/Hybrid)
- Session 管理測試
- MQTT 命令建構和解析測試

**整合測試**:
- Go LLM Client ↔ RTK Controller API 測試
- RTK Controller ↔ 設備 MQTT 通訊測試
- 完整診斷流程端對端測試
- 併發會話處理測試

**性能測試**:
- API 吞吐量和延遲測試 (單會話/多會話)
- MQTT 高併發訊息處理測試
- 長期運行穩定性測試 (24小時+)
- 記憶體洩漏和資源使用測試

#### 7.2 部署和運維設計

**容器化部署**:
```dockerfile
FROM golang:1.21-alpine AS builder
# ... build steps ...

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/rtk-controller .
COPY configs/ ./configs/
CMD ["./rtk-controller", "--daemon", "--config", "./configs/controller.yaml"]
```

**部署配置**:
- 單一實例部署 (簡化架構)
- 本地會話存儲或 Redis
- 持久化 MQTT 連接

---

## 📊 Intent → 工具鏈映射

### A. no_internet
1. `net.get_wan_status()` (Read)
2. `net.ping(gateway, 8.8.8.8)` (Test)
3. `dhcpdns.get_config()` (Read)
4. `dhcpdns.scan_rogue()` (Test)
5. **Act**: `dhcpdns.set()`, `net.restart_wan()`

### B. slow_speed
1. `net.speedtest(scope=router)` (Test)
2. `net.speedtest(scope=client)` (Test)
3. `wifi.survey()` + `wifi.utilization()` (Read)
4. `mesh.get_backhaul()` (Read)
5. **Act**: `wifi.set_channel()`, `mesh.set_backhaul()`

### C. roaming_issue
1. `clients.list()` (Read)
2. `wifi.roam_probe()` (Test)
3. `clients.roam_history()` (Read)
4. **Act**: `wifi.set_roaming()`, `wifi.set_power()`

### ... (其他 Intent 類推)

---

## 🛠️ 技術選型

### Go 套件依賴
```go
require (
    // MQTT 相關
    github.com/eclipse/paho.mqtt.golang v1.4.3
    
    // HTTP API Server
    github.com/gin-gonic/gin v1.9.1
    github.com/gorilla/mux v1.8.0
    github.com/rs/cors v1.10.1
    
    // gRPC 相關
    google.golang.org/grpc v1.58.3
    google.golang.org/protobuf v1.31.0
    
    // 基礎工具
    github.com/google/uuid v1.6.0
    github.com/sirupsen/logrus v1.9.3
    github.com/spf13/cobra v1.7.0
    github.com/spf13/viper v1.16.0
    
    // 認證與安全
    github.com/golang-jwt/jwt/v5 v5.0.0
    github.com/gin-contrib/sessions v0.0.5
    
    // 測試
    github.com/stretchr/testify v1.8.4
    
    // 並發控制
    golang.org/x/sync v0.3.0
    golang.org/x/time v0.3.0
)
```

### 專案結構 (基於現有 RTK Controller)
```
rtk_controller/  (現有專案)
├── cmd/
│   └── controller/
│       └── main.go         # 新增 --daemon 模式 (HTTP/gRPC servers)
├── internal/               # 重用現有架構
│   ├── mqtt/              # 現有 MQTT 客戶端 (擴展 remote command 支持)
│   ├── diagnostics/       # 現有診斷功能 (整合為 Local Tools)
│   ├── topology/          # 現有拓撲管理 (作為 Read Tools 資料源)
│   ├── cli/               # 現有 CLI (新增 MCP 命令)
│   ├── api/               # 新增 API 服務層
│   │   ├── http/         # HTTP API Server
│   │   │   ├── server.go # HTTP Server 實作
│   │   │   ├── handlers/ # API Handlers
│   │   │   └── middleware/ # 中間件 (auth, logging, cors)
│   │   ├── grpc/         # gRPC API Server  
│   │   │   ├── server.go # gRPC Server 實作
│   │   │   ├── service.go # gRPC Service 實作
│   │   │   └── proto/    # Protocol Buffer 定義
│   │   ├── auth/         # 認證服務
│   │   └── session/      # Session 管理
│   ├── tools/            # 工具整合層
│   │   ├── local/        # 本機工具 (ping, speedtest, etc.)
│   │   ├── remote/       # 遠端工具 (MQTT 調用)
│   │   └── hybrid/       # 混合工具 (本機+遠端)
│   └── orchestrator/     # Intent 編排器
├── pkg/
│   ├── types/             # 現有類型定義 (擴展工具結果格式)
│   └── utils/             # 現有工具函數
├── configs/
│   └── controller.yaml    # 現有配置 (新增 MCP 設定區塊)
└── test_scripts/          # 現有測試腳本 (新增 MCP 工具測試)
```

---

## 🚀 開發里程碑

| 週次 | 目標 | 可交付成果 |
|------|------|-----------|
| W1-2 | **協議與 API 規範設計** | **API_SPEC.md、PROTOCOL.md、Proto 定義、Session 架構設計** |
| W3-4 | **常駐服務與 API 實作** | HTTP/gRPC Server、Session 管理、認證機制、健康檢查 |
| W5-6 | **工具層整合實作** | Local/Remote/Hybrid 工具、設備能力發現、錯誤處理 |
| W7-8 | **診斷流程與安全機制** | Intent 分類器、診斷編排、Dry-run 機制、權限控制 |
| W9-10 | **可觀測性與監控** | Metrics 收集、Distributed Tracing、斷路器、重試機制 |
| W11-12 | **Go Client SDK & 整合測試** | Go LLM Client SDK、端到端測試、性能調優、部署方案 |

---

## ⚠️ 技術風險與對策

### 風險 1: 現有系統穩定性影響
**對策**: 
- MCP 功能作為獨立模組，不影響現有 CLI 功能
- 重用現有架構，最小化程式碼變更
- 完整的向下相容性測試
- 常駐服務穩定性設計 (graceful shutdown, health checks)

### 風險 2: MQTT 指令與設備實作不同步
**對策**:
- 設備能力發現與版本檢查機制
- 優雅降級 (設備不支援時使用現有資料)
- 分階段部署 (Read → Test → Act)

### 風險 5: Go LLM Client 連接性能
**對策**:
- HTTP/2 或 gRPC 使用連接池 (connection pooling)
- 合理的 timeout 與 retry 機制
- 健康檢查與服務監控
- 背壓 (backpressure) 控制機制

### 風險 6: 併發會話管理複雜度
**對策**:
- 無狀態 API 設計，會話數據存儲至 Redis
- 會話隔離機制，避免交叉影響
- 會話超時和自動清理機制
- 併發限制和資源配額控制

### 風險 7: 分散式系統故障處理
**對策**:
- 完整的錯誤分類和處理策略
- 斷路器模式防止級聯故障
- Distributed tracing 快速定位問題
- 設備離線時的優雅降級處理

### 風險 3: 本機與遠端工具結果不一致
**對策**:
- Hybrid 工具提供結果比較分析
- 時間戳與資料時效性檢查
- 結果可信度評分機制

### 風險 4: Act 工具的安全性
**對策**:
- 強制 dry-run 模式預覽
- 設備端變更權限控制
- 完整的操作審計日誌
- 自動回滾機制

---

## 📈 成功指標

1. **架構整合度**: MCP 功能完全整合現有 RTK Controller，無破壞性變更
2. **工具覆蓋率**: Read(15個) + Test(8個) + Act(6個) 工具完整實作
3. **調用效率**: Local 工具 < 1s, Remote 工具 < 3s, Hybrid 工具 < 5s
4. **Intent 支援度**: 9 種 Intent 的完整工具鏈序列
5. **設備相容性**: 支援不同能力等級的設備 (優雅降級)
6. **安全性**: 所有 Act 工具支援 dry-run + rollback，零誤操作

---

## 🔄 後續擴展計劃

### Phase 7: 高級功能 (未來)
- **機器學習整合**: 基於歷史資料的異常檢測
- **預測性維護**: 提前預警網路問題
- **多租戶支援**: 支援多個家庭環境管理
- **視覺化介面**: 網路拓撲圖形化展示
- **行動應用**: 手機 App 整合

### Phase 8: 企業級功能
- **大規模部署**: 支援數千台設備管理
- **API Gateway**: RESTful API 介面
- **監控告警**: Prometheus/Grafana 整合
- **日誌分析**: ELK Stack 整合

---

## 📝 交付清單

### 程式碼交付
- [ ] Go 原始程式碼 (完整專案)
- [ ] 單元測試 (覆蓋率 > 80%)
- [ ] 整合測試套件

### 文檔交付  
- [ ] API 文檔 (GoDoc)
- [ ] 部署指南
- [ ] 使用者手冊
- [ ] 故障排除指南

### 配置檔案
- [ ] 預設配置模板
- [ ] 環境特定配置範例
- [ ] 安全配置建議

---

## 📱 設備端實作需求 (依據 PROTOCOL.md)

**重要**: 各部門設備端開發將依據 Week 1-2 產出的 `PROTOCOL.md` 文檔進行

**設備端需要實作的新命令 (詳細規範見 PROTOCOL.md)**:

### Router/AP 設備需支援:
```json
// 拓撲相關
{"op": "topology.discover"}           // 網路發現
{"op": "clients.get_all"}              // 客戶端清單
{"op": "network.get_dhcp_config"}      // DHCP 配置

// WiFi 相關  
{"op": "wifi.get_environment"}         // 環境掃描 + 利用率
{"op": "wifi.trigger_roam"}           // 漫遊觸發
{"op": "wifi.set_power"}              // 功率調整 (with dry_run)
{"op": "wifi.set_channel"}            // 信道調整
{"op": "wifi.configure_roaming"}      // 漫遊參數

// 測試相關
{"op": "network.ping"}                // 網路測試
{"op": "network.speedtest"}           // 速度測試
{"op": "mesh.test_backhaul"}          // 回程測試

// 流量分析
{"op": "traffic.get_statistics"}      // 流量統計
{"op": "qos.get_status"}              // QoS 狀態
```

### 實作指導原則:
1. **所有 Act 命令必須支援 dry_run 模式**
2. **命令回應格式統一遵循 SPEC.md**
3. **使用現有 cmd/ack/res 流程**
4. **支援變更追蹤 (change_set_id)**

---

## 🔧 MCP 配置範例

### Go LLM Client 配置範例
```go
// Go LLM Client 配置
type RTKClientConfig struct {
    ServerURL    string `yaml:"server_url"` // http://localhost:8080
    GRPCAddress  string `yaml:"grpc_addr"`  // localhost:9090
    APIKey       string `yaml:"api_key"`
    Timeout      time.Duration `yaml:"timeout"`
    RetryAttempts int   `yaml:"retry_attempts"`
}

// HTTP Client 使用範例
client := &http.Client{Timeout: 30 * time.Second}
req, _ := http.NewRequest("POST", "http://localhost:8080/api/v1/tools/net.ping", 
    bytes.NewBuffer(payload))
req.Header.Set("Authorization", "Bearer "+apiKey)
resp, err := client.Do(req)

// gRPC Client 使用範例
conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
client := pb.NewRTKControllerClient(conn)
response, err := client.ExecuteTool(ctx, &pb.ToolRequest{...})
```

### RTK Controller 配置擴展 (`configs/controller.yaml`)
```yaml
# 原有配置保持不變...

# 新增 API 服務配置區塊
api:
  enabled: true
  http:
    enabled: true
    port: 8080
    host: "0.0.0.0"
    tls:
      enabled: false
      cert_file: ""
      key_file: ""
  grpc:
    enabled: true
    port: 9090
    host: "0.0.0.0"
    tls:
      enabled: false
      cert_file: ""
      key_file: ""
  auth:
    enabled: true
    method: "jwt"  # jwt, api_key, oauth2
    jwt_secret: "your-secret-key"
    token_expiry: "24h"
  tools:
    read_tools:
      - net.get_topology
      - wifi.get_radios
      - clients.list
    test_tools:
      - net.ping
      - net.speedtest
      - wifi.roam_probe
    act_tools:
      - wifi.set_power
      - wifi.set_channel
  
  # Intent 配置
  intents:
    thresholds:
      rssi_warn: -70.0
      jitter_warn: 30.0
      loss_warn: 1.0
      uplink_min: 10.0
      backhaul_min: 50.0
  
  # 安全設定
  security:
    require_dry_run: true
    require_approval: ["wifi.set_power", "wifi.set_channel"]
    audit_log: true
```

---

*此計劃基於現有 RTK Controller 架構改造，預計 12 週完成 MCP 工具層開發。*
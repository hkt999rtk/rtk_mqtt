# API 參考文檔

## 設備介面 (base.Device)

所有設備必須實作以下介面：

```go
type Device interface {
    // 基本資訊
    GetDeviceID() string
    GetDeviceType() string
    GetHealth() string
    GetIPAddress() string
    GetMACAddress() string
    GetSite() string
    GetTenant() string
    
    // 生命週期
    Start(ctx context.Context) error
    Stop() error
    
    // 資料生成
    GenerateStatePayload() StatePayload
    GenerateTelemetryData() map[string]TelemetryPayload
    GenerateEvents() []Event
    
    // 命令處理
    HandleCommand(cmd Command) error
    
    // 網路資訊
    GetNetworkInfo() NetworkInfo
    
    // 事件發布
    PublishEvent(event Event) error
}
```

## 設備管理器 API

### 建立設備管理器
```go
func NewDeviceManager(logger *logrus.Logger) *DeviceManager
```

### 註冊設備類型
```go
func (dm *DeviceManager) RegisterDeviceType(deviceType string, factory DeviceFactory)
```

### 建立設備
```go
func (dm *DeviceManager) CreateDevice(deviceConfig config.DeviceConfig) (base.Device, error)
```

### 取得設備
```go
func (dm *DeviceManager) GetDevice(deviceID string) (base.Device, error)
```

### 列出設備
```go
func (dm *DeviceManager) ListDevices() []base.Device
func (dm *DeviceManager) ListDevicesByType(deviceType string) []base.Device
```

### 移除設備
```go
func (dm *DeviceManager) RemoveDevice(deviceID string) error
```

### 批量操作
```go
func (dm *DeviceManager) StartAllDevices(ctx context.Context) error
func (dm *DeviceManager) StopAllDevices() error
```

## 場景管理 API

### 自動化管理器

#### 建立管理器
```go
func NewAutomationManager(config *AutomationConfig) *AutomationManager
```

#### 生命週期
```go
func (am *AutomationManager) Start(ctx context.Context) error
func (am *AutomationManager) Stop() error
```

#### 設備註冊
```go
func (am *AutomationManager) RegisterDevice(deviceID string, device base.Device)
```

#### 規則管理
```go
func (am *AutomationManager) LoadRule(rule *AutomationRule) error
func (am *AutomationManager) RemoveRule(ruleID string) error
func (am *AutomationManager) EnableRule(ruleID string) error
func (am *AutomationManager) DisableRule(ruleID string) error
```

#### 事件處理
```go
func (am *AutomationManager) PublishEvent(event Event)
```

#### 統計資訊
```go
func (am *AutomationManager) GetStatistics() map[string]interface{}
func (am *AutomationManager) GetActiveRules() []*ActiveAutomation
func (am *AutomationManager) GetActiveScenes() []*ActiveScene
```

### 日常作息管理器

#### 建立管理器
```go
func NewDailyRoutineManager(config *RoutineConfig) *DailyRoutineManager
```

#### 作息管理
```go
func (drm *DailyRoutineManager) LoadRoutine(routine *DailyRoutine) error
func (drm *DailyRoutineManager) RemoveRoutine(routineID string) error
func (drm *DailyRoutineManager) EnableRoutine(routineID string) error
func (drm *DailyRoutineManager) DisableRoutine(routineID string) error
```

#### 手動觸發
```go
func (drm *DailyRoutineManager) ManualTriggerRoutine(routineID string) error
```

#### 模式管理
```go
func (drm *DailyRoutineManager) SetHomeMode(mode string) error
func (drm *DailyRoutineManager) GetCurrentMode() string
```

### 腳本引擎

#### 建立引擎
```go
func NewScriptEngine(config *ScriptConfig) *ScriptEngine
```

#### 腳本管理
```go
func (se *ScriptEngine) LoadScript(script *Script) error
func (se *ScriptEngine) UnloadScript(scriptID string) error
func (se *ScriptEngine) EnableScript(scriptID string) error
func (se *ScriptEngine) DisableScript(scriptID string) error
```

#### 腳本執行
```go
func (se *ScriptEngine) ExecuteScript(scriptID string, params map[string]interface{}) (*ScriptExecution, error)
func (se *ScriptEngine) StopExecution(executionID string) error
```

#### 函數註冊
```go
func (se *ScriptEngine) RegisterFunction(name string, fn ScriptFunction)
```

## 網路拓撲 API

### 拓撲管理器

#### 建立管理器
```go
func NewTopologyManager() *TopologyManager
```

#### 設備管理
```go
func (tm *TopologyManager) AddDevice(device base.Device)
func (tm *TopologyManager) RemoveDevice(deviceID string)
```

#### 連接管理
```go
func (tm *TopologyManager) AddConnection(device1ID, device2ID string, connType ConnectionType)
func (tm *TopologyManager) RemoveConnection(device1ID, device2ID string)
```

#### 拓撲查詢
```go
func (tm *TopologyManager) GetDevice(deviceID string) base.Device
func (tm *TopologyManager) GetConnections(deviceID string) []Connection
```

#### 拓撲建構
```go
func (tm *TopologyManager) BuildSingleRouterTopology(routerID string, deviceIDs []string)
func (tm *TopologyManager) BuildMeshTopology(nodeIDs []string)
```

## 事件系統 API

### 事件匯流排

#### 建立匯流排
```go
func NewEventBus(bufferSize int) *EventBus
```

#### 生命週期
```go
func (eb *EventBus) Start(ctx context.Context) error
func (eb *EventBus) Stop()
```

#### 訂閱/發布
```go
func (eb *EventBus) Subscribe(pattern string, handler EventHandler) string
func (eb *EventBus) Unsubscribe(subscriptionID string)
func (eb *EventBus) Publish(event Event)
```

#### 統計資訊
```go
func (eb *EventBus) GetStatistics() map[string]interface{}
```

## 配置 API

### 載入配置
```go
func LoadConfig(path string) (*SimulationConfig, error)
```

### 驗證配置
```go
func ValidateConfig(config *SimulationConfig) error
```

### 配置結構

#### SimulationConfig
```go
type SimulationConfig struct {
    Simulation SimulationSettings
    MQTT       MQTTSettings
    Network    NetworkSettings
    Devices    DeviceConfigs
    Scenarios  []ScenarioConfig
    Logging    LoggingConfig
}
```

#### DeviceConfig
```go
type DeviceConfig struct {
    ID             string
    Type           string
    Tenant         string
    Site           string
    IPAddress      string
    ConnectionType string
    Firmware       string
    Protocols      []string
    Location       *Location
}
```

## 命令結構

### Command
```go
type Command struct {
    Type       string
    Parameters map[string]interface{}
    RequestID  string
    Timestamp  time.Time
}
```

### 支援的命令類型

#### 智慧燈泡
- `turn_on`: 開啟（參數: brightness, color）
- `turn_off`: 關閉
- `set_brightness`: 設定亮度（參數: brightness）
- `set_color`: 設定顏色（參數: r, g, b）
- `set_color_temp`: 設定色溫（參數: temperature）

#### 空調
- `turn_on`: 開啟
- `turn_off`: 關閉
- `set_temperature`: 設定溫度（參數: temperature）
- `set_mode`: 設定模式（參數: mode）
- `set_fan_speed`: 設定風速（參數: speed）

#### 安全攝像頭
- `start_recording`: 開始錄影
- `stop_recording`: 停止錄影
- `enable_motion_detection`: 啟用移動偵測
- `disable_motion_detection`: 停用移動偵測
- `capture_snapshot`: 擷取快照

## 狀態負載結構

### StatePayload
```go
type StatePayload struct {
    DeviceID     string
    DeviceType   string
    Timestamp    time.Time
    Health       string
    CPUUsage     float64
    MemoryUsage  float64
    Temperature  float64
    Uptime       int64
    NetworkInfo  NetworkInfo
    Extra        map[string]interface{}
}
```

## 遙測資料結構

### TelemetryPayload
```go
type TelemetryPayload struct {
    Metric    string
    Value     float64
    Unit      string
    Timestamp time.Time
    Tags      map[string]string
}
```

## 事件結構

### Event
```go
type Event struct {
    EventType   string
    Severity    string  // info, warning, error, critical
    Message     string
    DeviceID    string
    Timestamp   time.Time
    Extra       map[string]interface{}
}
```

## 錯誤處理

所有 API 函數遵循 Go 的錯誤處理慣例，返回 error 作為最後一個返回值。

### 常見錯誤
- `ErrDeviceNotFound`: 設備不存在
- `ErrInvalidConfig`: 配置無效
- `ErrConnectionFailed`: 連接失敗
- `ErrTimeout`: 操作超時
- `ErrNotImplemented`: 功能未實作

## 最佳實踐

### 資源管理
1. 始終使用 context 進行生命週期管理
2. 確保正確關閉設備和管理器
3. 使用 defer 確保資源釋放

### 錯誤處理
1. 檢查所有錯誤返回值
2. 提供有意義的錯誤訊息
3. 使用結構化日誌記錄錯誤

### 並發安全
1. 使用 mutex 保護共享狀態
2. 透過 channel 傳遞資料
3. 避免 goroutine 洩漏

### 效能優化
1. 批量處理操作
2. 使用物件池減少 GC 壓力
3. 避免不必要的記憶體分配
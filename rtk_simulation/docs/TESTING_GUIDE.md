# 測試指南

## 測試架構概覽

RTK Home Network Simulator 採用分層測試架構，包含單元測試、整合測試、效能測試和端到端測試。

## 執行測試

### 執行所有測試
```bash
make test
```

### 執行特定類型測試
```bash
# 單元測試
go test ./tests/unit/...

# 整合測試
go test ./tests/integration/...

# 效能測試
go test ./tests/performance/... -bench=.

# 端到端測試
go test ./tests/e2e/...
```

### 生成測試覆蓋率報告
```bash
make coverage
# 報告將生成在 build/coverage.html
```

## 單元測試

### 設備測試
位置: `tests/unit/device_test.go`

測試內容：
- 設備建立和初始化
- 命令處理
- 狀態生成
- 遙測資料生成
- 事件生成

範例：
```go
func TestSmartBulb(t *testing.T) {
    config := base.DeviceConfig{
        ID:       "bulb_001",
        Type:     "smart_bulb",
        Location: &base.Location{
            Room: "living_room",
        },
    }
    
    bulb, err := iot.NewSmartBulb(config, base.MQTTConfig{})
    require.NoError(t, err)
    
    // 測試開燈命令
    cmd := base.Command{
        Type: "turn_on",
        Parameters: map[string]interface{}{
            "brightness": 80,
        },
    }
    err = bulb.HandleCommand(cmd)
    assert.NoError(t, err)
}
```

### 場景測試
位置: `tests/unit/scenario_test.go`

測試內容：
- 自動化規則執行
- 日常作息觸發
- 腳本引擎執行
- 事件處理

## 整合測試

### 完整模擬測試
位置: `tests/integration/simulation_test.go`

測試內容：
- 多設備協同工作
- MQTT 通信
- 網路拓撲建構
- 場景協調

範例：
```go
func TestFullSimulation(t *testing.T) {
    // 建立模擬配置
    cfg := &config.SimulationConfig{
        Simulation: config.SimulationSettings{
            Name:     "test_home",
            Duration: 10 * time.Second,
        },
        // ... 設備配置
    }
    
    // 建立並啟動設備
    deviceManager := devices.NewDeviceManager(logger)
    // ... 建立設備並測試互動
}
```

### MQTT 通信測試
測試 MQTT 訊息發布和訂閱功能：
- 狀態訊息發布
- 遙測資料發布
- 命令處理
- 連線管理

## 效能測試

### 基準測試
位置: `tests/performance/benchmark_test.go`

測試項目：
- 設備建立效能
- 狀態生成效能
- 事件處理吞吐量
- 記憶體使用情況

執行基準測試：
```bash
# 執行所有基準測試
go test -bench=. ./tests/performance/

# 執行特定基準測試
go test -bench=BenchmarkDeviceCreation ./tests/performance/

# 包含記憶體分析
go test -bench=. -benchmem ./tests/performance/

# 設定執行時間
go test -bench=. -benchtime=10s ./tests/performance/
```

基準測試結果解讀：
```
BenchmarkDeviceCreation-8    1000    1052358 ns/op    245632 B/op    1234 allocs/op
```
- `-8`: 使用 8 個 CPU 核心
- `1000`: 執行次數
- `1052358 ns/op`: 每次操作耗時（奈秒）
- `245632 B/op`: 每次操作分配記憶體（位元組）
- `1234 allocs/op`: 每次操作記憶體分配次數

## 端到端測試

### 24小時模擬測試
位置: `tests/e2e/full_simulation_test.go`

測試內容：
- 完整日常週期模擬
- 故障恢復測試
- 高負載測試（100+ 設備）
- 資料一致性驗證

範例：
```go
func TestCompleteHomeSimulation(t *testing.T) {
    env := utils.NewTestEnvironment(t)
    defer env.Cleanup()
    
    // 建立家庭設備
    devices := createHomeDevices(t, env)
    
    // 模擬 24 小時週期
    simulateDailyCycle(t, env)
    
    // 驗證結果
    validateSimulation(t, env)
}
```

## 測試工具

### TestEnvironment
位置: `tests/utils/test_helpers.go`

提供完整的測試環境管理：
```go
env := utils.NewTestEnvironment(t)
defer env.Cleanup()

// 建立設備
device := env.CreateDevice(t, "bulb_001", "smart_bulb")

// 使用各種管理器
env.AutomationManager.LoadRule(rule)
env.ScriptEngine.ExecuteScript(scriptID, params)
```

### Mock 設備
用於測試不需要完整設備實作的場景：
```go
type MockDevice struct {
    id         string
    deviceType string
    health     string
}

// 實作 base.Device 介面
func (m MockDevice) GetDeviceID() string { return m.id }
// ... 其他方法
```

### 測試配置生成器
```go
// 生成隨機設備配置
configs := utils.GenerateRandomDeviceConfig(100)

// 建立測試配置檔案
configPath := utils.CreateTestConfig(t, "test_config")
```

## 測試資料

### Fixtures
位置: `tests/fixtures/`

包含預定義的測試資料：
- 設備配置範本
- 場景定義
- 預期輸出範例

載入 fixture：
```go
data := utils.LoadFixture(t, "device_config.json")
```

## 測試最佳實踐

### 1. 使用表格驅動測試
```go
func TestDeviceCommands(t *testing.T) {
    tests := []struct {
        name    string
        command base.Command
        wantErr bool
    }{
        {
            name: "valid turn on",
            command: base.Command{Type: "turn_on"},
            wantErr: false,
        },
        // ... 更多測試案例
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := device.HandleCommand(tt.command)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 2. 適當的測試隔離
- 每個測試應該獨立執行
- 使用 `t.Parallel()` 加速測試
- 清理測試資源

### 3. 有意義的斷言
```go
// 好的斷言
assert.Equal(t, "healthy", device.GetHealth(), 
    "Device should be healthy after initialization")

// 避免
assert.True(t, device.GetHealth() == "healthy")
```

### 4. 測試錯誤情況
```go
func TestErrorHandling(t *testing.T) {
    // 測試無效命令
    err := device.HandleCommand(base.Command{Type: "invalid"})
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "unsupported command")
}
```

### 5. 使用 context 控制超時
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := manager.Start(ctx)
require.NoError(t, err)
```

## 持續整合

### GitHub Actions 配置
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - run: make test
      - run: make coverage
```

## 調試測試

### 執行單個測試
```bash
go test -run TestSmartBulb ./tests/unit/
```

### 顯示詳細輸出
```bash
go test -v ./tests/unit/
```

### 使用 delve 調試器
```bash
dlv test ./tests/unit/ -- -test.run TestSmartBulb
```

## 測試覆蓋率目標

- 單元測試: > 80%
- 整合測試: > 60%
- 關鍵路徑: 100%

## 常見問題

### Q: 測試執行太慢
A: 使用 `-short` 標誌跳過長時間測試：
```bash
go test -short ./...
```

### Q: 測試不穩定（flaky）
A: 檢查：
- 時間相關的邏輯
- 並發競爭條件
- 外部依賴

### Q: 記憶體洩漏
A: 使用 pprof 分析：
```bash
go test -memprofile mem.prof ./tests/performance/
go tool pprof mem.prof
```

## 測試檢查清單

- [ ] 所有新功能都有對應的單元測試
- [ ] 整合測試覆蓋主要使用場景
- [ ] 效能測試確認沒有效能退化
- [ ] 端到端測試驗證完整功能
- [ ] 測試覆蓋率達到目標
- [ ] 所有測試在 CI 環境通過
# RTK Home Network Simulator 架構文檔

## 系統架構總覽

RTK Home Network Simulator 是一個基於 Go 語言開發的家庭網路環境模擬系統，用於測試和驗證 RTK Controller 的功能。

## 核心組件

### 1. 設備層 (pkg/devices/)

#### 基礎設備框架 (base/)
- **BaseDevice**: 所有設備的基礎類，提供通用功能
  - MQTT 客戶端管理
  - 狀態/遙測/事件生成
  - 網路資訊追蹤
  - 健康監控

#### 設備類型
- **網路設備** (network/)
  - Router: 路由器模擬
  - Switch: 交換機模擬
  - AccessPoint: 無線接入點
  - MeshNode: Mesh 節點

- **IoT 設備** (iot/)
  - SmartBulb: 智慧燈泡
  - AirConditioner: 空調設備
  - SecurityCamera: 安全攝像頭
  - EnvironmentalSensor: 環境感測器
  - SmartPlug: 智慧插座
  - SmartThermostat: 智慧溫控器

- **客戶端設備** (client/)
  - Smartphone: 智慧手機
  - Laptop: 筆記型電腦
  - Tablet: 平板電腦
  - SmartTV: 智慧電視

### 2. 場景管理 (pkg/scenarios/)

#### 自動化管理器
- 規則引擎
- 場景控制
- 事件處理
- 觸發器管理

#### 日常作息管理
- 早晨/白天/晚間/夜間模式
- 工作日/週末模式
- 假期模式
- 自定義時間表

#### 腳本引擎
- 動作序列執行
- 條件判斷
- 變數管理
- 函數調用

#### 行為模式
- 使用者行為模擬
- 設備使用模式
- 預測性行為

### 3. 網路拓撲 (pkg/network/)

#### 拓撲管理器
- 設備連接管理
- 路徑計算
- 鄰居發現
- 連接類型（WiFi/Ethernet/Mesh）

#### 網路模擬
- 流量生成
- 延遲模擬
- 封包丟失
- 頻寬限制

### 4. 同步機制 (pkg/sync/)

#### 狀態同步
- 設備狀態管理
- 狀態變更通知
- 歷史記錄
- 同步規則

#### 群組同步
- 設備群組管理
- 批量命令執行
- 同步策略

### 5. 互動管理 (pkg/interaction/)

#### 設備互動
- 設備間通信
- 觸發器鏈
- 聯動效果
- 事件傳播

### 6. 監控系統 (pkg/monitoring/)

#### 指標收集
- 設備指標
- 網路指標
- 效能指標
- 事件統計

## 資料流程

```
使用者配置 (YAML)
    ↓
配置解析器 (config/)
    ↓
設備管理器 (DeviceManager)
    ↓
設備實例化 (Factory Pattern)
    ↓
MQTT 連接建立
    ↓
狀態/遙測/事件發布循環
    ↓
場景/自動化執行
    ↓
監控資料收集
```

## MQTT 通信架構

### 主題結構
```
rtk/v1/{tenant}/{site}/{device_id}/state       # 設備狀態（保留）
rtk/v1/{tenant}/{site}/{device_id}/telemetry/* # 遙測資料
rtk/v1/{tenant}/{site}/{device_id}/evt/*       # 事件
rtk/v1/{tenant}/{site}/{device_id}/cmd/req     # 命令請求
rtk/v1/{tenant}/{site}/{device_id}/cmd/ack     # 命令確認
rtk/v1/{tenant}/{site}/{device_id}/cmd/res     # 命令回應
rtk/v1/{tenant}/{site}/{device_id}/lwt         # 最後遺囑
rtk/v1/{tenant}/{site}/{device_id}/attr        # 設備屬性（保留）
```

### 訊息格式
- 狀態訊息：JSON 格式，包含設備健康、CPU、記憶體等
- 遙測訊息：時序資料，包含度量值和時間戳
- 事件訊息：包含事件類型、嚴重性、描述等

## 配置系統

### 配置結構
```yaml
simulation:
  name: 模擬名稱
  duration: 持續時間
  real_time_factor: 時間加速因子

network:
  topology: 拓撲類型
  subnet: 子網路
  interference: 干擾配置

devices:
  network_devices: 網路設備列表
  iot_devices: IoT 設備列表
  client_devices: 客戶端設備列表

scenarios:
  - 場景配置列表
```

## 並發模型

### Goroutine 管理
- 每個設備運行獨立的 goroutine
- MQTT 發布循環使用獨立 goroutine
- 場景管理器使用 worker pool
- Context 用於生命週期管理

### 同步機制
- Mutex 保護共享狀態
- Channel 用於事件傳遞
- WaitGroup 協調並發操作

## 錯誤處理

### 錯誤策略
- 優雅降級：設備故障不影響整體系統
- 重試機制：網路錯誤自動重試
- 日誌記錄：結構化日誌（logrus）
- 錯誤傳播：通過 error 返回值傳遞

## 效能優化

### 記憶體管理
- 物件池用於頻繁創建的物件
- 定期清理過期資料
- 避免記憶體洩漏

### CPU 優化
- 批量處理訊息
- 避免忙等待
- 使用高效的資料結構

## 擴展性設計

### 新增設備類型
1. 在相應目錄創建新設備檔案
2. 嵌入 BaseDevice
3. 實作必要的介面方法
4. 在 factory.go 註冊

### 新增場景
1. 定義場景結構
2. 實作場景邏輯
3. 註冊到場景管理器

### 新增網路拓撲
1. 實作拓撲建構邏輯
2. 定義連接規則
3. 整合到拓撲管理器

## 測試架構

### 單元測試
- 設備功能測試
- 場景邏輯測試
- 工具函數測試

### 整合測試
- 完整模擬測試
- MQTT 通信測試
- 場景協調測試

### 效能測試
- 設備建立基準測試
- 事件處理吞吐量
- 記憶體使用分析

### 端到端測試
- 24 小時模擬
- 故障恢復測試
- 負載測試

## 部署考慮

### 環境需求
- Go 1.21+
- MQTT Broker（可選）
- 足夠的記憶體（100 設備約需 2GB）

### 配置管理
- YAML 配置檔案
- 環境變數覆蓋
- 命令列參數

### 監控整合
- Prometheus 指標匯出
- 日誌聚合
- 健康檢查端點

## 安全考慮

### MQTT 安全
- TLS/SSL 支援
- 使用者名稱/密碼認證
- 客戶端證書（未來）

### 資料保護
- 敏感資訊不記錄
- 配置檔案權限管理
- 安全的預設值

## 已知限制

1. 整合測試中部分 MockDevice 實作不完整
2. 測試覆蓋率需要提升
3. 部分進階功能（FaultManager、BehaviorManager）尚未實作
4. 效能監控整合待完善

## 未來改進方向

1. 增加測試覆蓋率到 80%+
2. 實作完整的故障注入系統
3. 加入 AI 驅動的行為預測
4. 支援更多 IoT 協議（CoAP、Zigbee）
5. 實作分散式模擬支援
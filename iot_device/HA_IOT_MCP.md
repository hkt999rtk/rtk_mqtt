# Home Assistant / MQTT / IoT Device / MCP / Claude Desktop 整合架構

## 系統架構概覽

本文檔說明 Home Assistant (HA)、MQTT Broker、IoT 設備、MCP Server 和 Claude Desktop 之間的整合關係和數據流向。

![系統架構圖](HA_IoT_MCP_Architecture.png)

## 核心組件關係

### 1. Home Assistant (智能家居中心)
- **角色**: 智能家居控制平台
- **功能**: 
  - 提供 Web UI 用於設備控制
  - 管理設備狀態和自動化規則
  - 通過 MQTT 與 IoT 設備通信

### 2. MQTT Broker (消息中間件)
- **角色**: 消息路由中心
- **功能**:
  - 接收來自 HA 的控制命令
  - 轉發消息給訂閱的 IoT 設備
  - 收集設備狀態回報給 HA

### 3. IoT Device (物聯網設備)
- **角色**: 實際執行設備
- **功能**:
  - 訂閱 MQTT 命令主題
  - 執行收到的控制命令
  - 回報設備狀態

### 4. MCP Server (模型上下文協議服務器)
- **角色**: AI 助手與 HA 系統的橋接層
- **部署位置**: 安裝於 Home Assistant 環境中
- **功能**:
  - 提供 AI 可調用的工具接口
  - 接收 Claude Desktop 的 API 調用
  - 通知 Home Assistant 執行設備控制
  - 封裝 HA 服務調用為結構化 API

### 5. Claude Desktop (AI 助手)
- **角色**: 智能控制界面
- **功能**:
  - 理解用戶自然語言指令
  - 通過 MCP 調用設備控制 API
  - 提供智能化的設備管理

## 數據流向示例

### 場景: 控制客廳燈光

#### HA 配置示例
```yaml
light:
  - platform: mqtt
    name: "Living Room Light"
    state_topic: "home/livingroom/light1/state"
    command_topic: "home/livingroom/light1/set"
    payload_on: "ON"
    payload_off: "OFF"
    qos: 1
    retain: true
```

#### 控制流程

##### 1. 通過 HA UI 控制

![HA UI 控制流程](HA_UI_Control_Flow.png)

```
用戶點擊 HA UI 開關
    ↓
HA 發送 MQTT 消息: 
   Topic: "home/livingroom/light1/set"
   Payload: "ON"
    ↓
MQTT Broker 轉發消息
    ↓
IoT 燈泡設備接收命令
    ↓
燈泡執行開燈操作
    ↓
燈泡回報狀態:
   Topic: "home/livingroom/light1/state"
   Payload: "ON"
    ↓
HA UI 更新顯示狀態
```

##### 2. 通過 Claude Desktop 控制

![Claude Desktop 控制流程](Claude_Control_Flow.png)

```
用戶對 Claude 說: "請打開客廳的燈"
    ↓
Claude Desktop 理解指令
    ↓
Claude 通過 MCP Server 調用控制 API
    ↓
MCP Server (安裝於 HA 環境) 通知 Home Assistant
    ↓
Home Assistant 調用 MQTT light 服務
    ↓
HA 發送 MQTT 消息:
   Topic: "home/livingroom/light1/set"
   Payload: "ON"
    ↓
MQTT Broker 轉發消息
    ↓
IoT 燈泡設備接收並執行
    ↓
狀態回報給 HA，再傳回 MCP Server
    ↓
Claude 回應: "客廳燈已開啟"
```

## MCP Server 實現要點

### 工具接口設計
```json
{
  "tools": [
    {
      "name": "control_light",
      "description": "控制指定燈光設備",
      "parameters": {
        "device_id": "設備 ID",
        "action": "on/off/toggle",
        "brightness": "亮度值 (可選)"
      }
    },
    {
      "name": "get_device_status",
      "description": "獲取設備當前狀態",
      "parameters": {
        "device_id": "設備 ID"
      }
    }
  ]
}
```

### MQTT 主題規範
```
狀態主題: home/{room}/{device_type}{device_number}/state
命令主題: home/{room}/{device_type}{device_number}/set
```

## 技術重點

### 1. 分離關注點
- **HA**: 專注於設備管理和自動化
- **MQTT**: 專注於可靠的消息傳遞
- **MCP**: 專注於 AI 工具接口
- **Claude**: 專注於自然語言理解

### 2. 擴展性
- 新增設備只需配置 HA 和 MCP 工具
- 支持多種設備類型 (燈光、溫控、安防等)
- 可集成其他 AI 助手

### 3. 可維護性
- 標準化的 MQTT 協議
- 結構化的 MCP 工具定義
- 清晰的數據流向

### 4. 實現建議 - 設備標準化
- 統一 MQTT 主題命名規範
- 標準化狀態和命令格式
- 實現設備自動發現機制

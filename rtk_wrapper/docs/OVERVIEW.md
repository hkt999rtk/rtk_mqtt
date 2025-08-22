# MQTT Wrapper 中介層概述

## 目標概述

建立一個 MQTT 中介層 (Wrapper) 系統，用於轉換不同 MQTT 來源的訊息格式到 RTK 標準格式。此中介層將：

1. **統一訊息格式**: 將各種設備的原生 MQTT 格式轉換為 RTK 標準格式 (`rtk/v1/{tenant}/{site}/{device_id}/{message_type}`)
2. **靈活註冊機制**: 支援 static 註冊不同類型的設備包裝器
3. **透明轉發**: 連接到同一個 MQTT Broker，對 RTK Controller 完全透明
4. **可擴展架構**: 支援未來添加新的設備類型和協議

## 現有 RTK 架構要點

基於代碼分析，RTK Controller 具有以下關鍵特性：

1. **MQTT 客戶端架構** (`internal/mqtt/`)：
   - 基於 Eclipse Paho MQTT 客戶端
   - 支援 TLS、認證、QoS 控制
   - 訊息處理採用 Handler 模式
   - 內建訊息日志和追踪

2. **標準 Topic 格式**：
   ```
   rtk/v1/{tenant}/{site}/{device_id}/{message_type}
   ```

3. **訊息類型**：
   - `state` - 設備狀態 (retained)
   - `telemetry/{metric}` - 遙測數據
   - `evt/{event_type}` - 事件通知
   - `attr` - 設備屬性 (retained)
   - `cmd/req|ack|res` - 命令流程
   - `lwt` - Last Will Testament

4. **標準訊息格式**：
   ```json
   {
     "schema": "<message_type>/<version>",
     "ts": 1699123456789,
     "device_id": "aabbccddeeff",
     "payload": {
       // 業務數據
     },
     "trace": {
       "req_id": "...",
       "session_id": "..."
     }
   }
   ```

## 設計原則

1. **非侵入性**: 不修改現有 RTK Controller 代碼
2. **可配置性**: 支援動態配置 Wrapper 規則
3. **高性能**: 低延遲訊息轉換
4. **可觀測性**: 完整的轉換日志和統計
5. **錯誤處理**: 優雅的錯誤處理和重試機制

## 完整雙向訊息流程

```
                              MQTT Broker
                           ┌─────────────────┐
            ┌─────────────┬│                 │┬─────────────┐
            │             ││                 ││             │
            ↓             ││                 ││             ↓
    ┌─────────────┐       ││   Wrapper       ││      ┌──────────────┐
    │   設備      │       ││   中介層        ││      │ RTK Controller│
    │ (HA/Tasmota)│       ││                 ││      │              │
    └─────────────┘       ││                 ││      └──────────────┘
            ↑             ││                 ││             ↑
            └─────────────┴│                 │┴─────────────┘
                           └─────────────────┘

上行流程（設備 → RTK）:
1. 設備發布: homeassistant/light/living_room/state
2. Wrapper 接收並識別為上行訊息  
3. 轉換為 RTK 格式
4. 發布: rtk/v1/home/main/living_room_light/state
5. RTK Controller 接收處理

下行流程（RTK → 設備）:
1. RTK Controller 發布: rtk/v1/home/main/living_room_light/cmd/req
2. Wrapper 接收並識別為下行訊息
3. 轉換為設備格式
4. 發布: homeassistant/light/living_room/set
5. 設備接收並執行命令
```

**雙向監聽**:
- **Wrapper 訂閱上行**: `homeassistant/+/+/state`, `tasmota/+/STATE` 等
- **Wrapper 訂閱下行**: `rtk/v1/+/+/+/cmd/req` 等
- **自動方向判斷**: 根據 topic 前綴 `rtk/v1/` 自動識別方向

## 實際轉換範例

### 上行轉換範例

**步驟 1: Home Assistant 設備發布原始訊息**
```
Topic: homeassistant/light/living_room/state  
Payload: {
  "entity_id": "light.living_room",
  "state": "on",
  "attributes": {
    "brightness": 255,
    "color_temp": 300
  },
  "last_changed": "2023-11-04T12:34:56Z"
}
QoS: 0, Retained: false
```

**步驟 2: Wrapper 轉換為 RTK 格式**
```
Topic: rtk/v1/home/main/living_room_light/state
Payload: {
  "schema": "state/1.0",
  "ts": 1699123456789,
  "device_id": "living_room_light", 
  "payload": {
    "health": "ok",
    "brightness": 255,
    "color_temp": 300,
    "power_state": "on"
  }
}
QoS: 1, Retained: true  // 根據 schema 自動決定
```

### 下行轉換範例

**步驟 1: RTK Controller 發布命令**
```
Topic: rtk/v1/home/main/living_room_light/cmd/req
Payload: {
  "schema": "cmd.turn_on/1.0",
  "ts": 1699123456789,
  "device_id": "living_room_light",
  "payload": {
    "command": "turn_on",
    "brightness": 200
  }
}
```

**步驟 2: Wrapper 轉換為設備格式**
```
Topic: homeassistant/light/living_room/set
Payload: {
  "state": "ON",
  "brightness": 200
}
```

## 主要優勢

1. **無縫整合**: RTK Controller 無需任何修改
2. **廠牌支援**: 支援多種主流 IoT 設備廠牌
3. **雙向通信**: 完整支援監控和控制
4. **高可用性**: 獨立部署，不影響現有系統
5. **易於擴展**: 模塊化設計，輕鬆添加新廠牌支援

## 下一步

請閱讀 [ARCHITECTURE.md](ARCHITECTURE.md) 了解詳細的技術架構設計。
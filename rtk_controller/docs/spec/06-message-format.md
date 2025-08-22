# 共通 Payload 格式與規則

## 概述

RTK MQTT 協議使用標準化的 JSON 訊息格式，確保所有透過 MQTT 傳送的訊息都具有一致的結構。無論是狀態回報、遙測資料、事件通知還是命令，都遵循相同的基本格式。

### 標準化的重要性

共通 Payload 格式提供以下優勢：
- **版本控制**: 確保不同版本的設備和系統能正確解析訊息
- **時間追蹤**: 記錄事件發生的精確時間
- **除錯追蹤**: 在複雜系統中追蹤訊息流向
- **一致性**: 所有訊息類型使用相同的基本結構

## 必要欄位

### `schema` (字串)
訊息類型和版本標識符，讓接收端知道如何解析訊息。

**格式**: `<訊息類型>/<版本號>`

**範例**:
```json
{
  "schema": "state/1.0",              // 設備狀態訊息 v1.0
  "schema": "evt.wifi.roam_miss/1.0", // WiFi 漫遊失效事件 v1.0  
  "schema": "cmd.diagnosis.get/1.0",  // 診斷資料請求命令 v1.0
  "schema": "telemetry.network/1.0"   // 網路遙測資料 v1.0
}
```

**Schema 命名規範**:
- 使用小寫字母和點號分隔
- 版本號採用 `major.minor` 格式
- 破壞性變更時增加 major 版本號

### `ts` (整數)
記錄訊息產生或事件發生的精確時間。

**格式**: Unix 時間戳（毫秒）

**範例**:
```json
{
  "ts": 1699123456789  // 2023-11-04T23:24:16.789Z
}
```

**時間格式要求**:
- 使用 Unix 毫秒時間戳
- 必須使用 UTC 時區
- 包含毫秒精度
- 所有設備必須使用同步時間

## 選用欄位

### `device_id` (字串)
設備唯一標識符，通常為 MAC 地址格式。

```json
{
  "device_id": "aabbccddeeff"
}
```

### `trace` (物件)
分散式系統追蹤資訊，特別支援 LLM 自動化診斷系統。

**常用欄位**:
```json
{
  "trace": {
    "req_id": "diag-001",                    // 請求唯一識別碼
    "correlation_id": "session-12345",       // 關聯多個相關訊息
    "span_id": "span-001",                   // 分散式追蹤區段
    "session_id": "llm-diag-session-001",   // LLM 診斷會話 ID
    "trace_id": "diag-session-001-step-03"  // LLM 診斷步驟追蹤
  }
}
```

**使用場景**:
- **req_id**: 單一命令執行追蹤
- **correlation_id**: 關聯多個相關操作
- **session_id**: LLM 長期診斷會話
- **trace_id**: 跨設備操作追蹤

## 標準訊息結構

### 基本結構模板
```json
{
  // === 必要共通欄位 ===
  "schema": "<message_type>/<version>",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  
  // === 業務資料包裝 ===
  "payload": {
    // 所有訊息特定的業務邏輯欄位都包裝在 payload 中
    "health": "ok",
    // 其他業務欄位...
  },
  
  // === 選用欄位 ===
  "trace": {
    "req_id": "unique-request-id",
    "session_id": "llm-session-id"
  },
  "meta": {
    // 可選的元資料
  }
}
```

### 欄位分類指南

| 欄位類型 | 欄位名稱 | 必要性 | 資料型態 | 說明 |
|---------|---------|--------|---------|------|
| **共通欄位** | `schema` | 必要 | string | 訊息類型與版本標識 |
| **共通欄位** | `ts` | 必要 | integer | Unix 時間戳（毫秒） |
| **共通欄位** | `device_id` | 必要 | string | 設備唯一標識符 |
| **業務欄位** | `payload` | 必要 | object | 包裝所有業務邏輯資料 |
| **追蹤欄位** | `trace` | 選用 | object | 分散式追蹤資訊 |
| **追蹤欄位** | `trace.req_id` | 選用 | string | 請求識別碼 |
| **追蹤欄位** | `trace.correlation_id` | 選用 | string | 相關性追蹤 ID |
| **追蹤欄位** | `trace.session_id` | 選用 | string | LLM 診斷會話 ID |
| **追蹤欄位** | `trace.trace_id` | 選用 | string | LLM 診斷步驟追蹤 |
| **元資料欄位** | `meta` | 選用 | object | 可選的元資料資訊 |

## 訊息類型範例

### 狀態訊息 (State)
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "health": "ok",
    "uptime_s": 86400,
    "connection_status": "connected",
    "cpu_usage": 45.2,
    "memory_usage": 62.8
  }
}
```

### 遙測訊息 (Telemetry)
```json
{
  "schema": "telemetry.temperature/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "sensor_id": "temp_001",
    "value": 23.5,
    "unit": "celsius",
    "location": "cpu"
  }
}
```

### 事件訊息 (Event)
```json
{
  "schema": "evt.wifi.connection_lost/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "event_type": "connection_lost",
    "interface": "wlan0",
    "previous_bssid": "aabbccddeeff",
    "reason": "signal_lost",
    "duration_ms": 1500
  }
}
```

### 命令訊息 (Command)
```json
{
  "schema": "cmd.diagnosis.get/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "command": "get_network_status",
    "parameters": {
      "include_topology": true,
      "detailed": false
    }
  },
  "trace": {
    "req_id": "diag-20241104-001",
    "session_id": "llm-session-abc123"
  }
}
```

## 版本相容性規則

### 向後相容性原則
1. **新增欄位**: 必須為選用欄位，提供合理的預設值
2. **刪除欄位**: 必須增加 major 版本號
3. **修改欄位**: 必須增加 major 版本號
4. **重新命名欄位**: 必須增加 major 版本號

### 版本演進範例
```json
// v1.0 - 原始版本
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "health": "ok"
}

// v1.1 - 新增選用欄位 (向後相容)
{
  "schema": "state/1.1", 
  "ts": 1699123456789,
  "health": "ok",
  "uptime_s": 86400        // 新增欄位
}

// v2.0 - 破壞性變更
{
  "schema": "state/2.0",
  "ts": 1699123456789,
  "status": "healthy",     // 重新命名 health -> status
  "uptime_s": 86400
}
```

## 資料驗證規則

### JSON Schema 驗證
每種訊息類型都有對應的 JSON Schema 用於驗證，所有 schema 文件都位於 `schemas/` 目錄中。

### Schema 文件組織
- **基礎 Schema**: `schemas/base.json` - 所有消息的基礎結構
- **狀態訊息**: `schemas/state.json` - 設備狀態訊息
- **命令 Schema**: `schemas/cmd-*.json` - 各種命令定義
- **遙測 Schema**: `schemas/telemetry-*.json` - 各類遙測資料
- **事件 Schema**: `schemas/evt-*.json` - 各種事件定義
- **屬性 Schema**: `schemas/attr.json` - 設備屬性定義

### 使用範例
```bash
# 使用 ajv-cli 驗證狀態訊息
ajv validate -s schemas/state.json -d state_message.json

# 使用 ajv-cli 驗證命令訊息
ajv validate -s schemas/cmd-wifi-scan.json -d wifi_scan_command.json
```

詳細的 schema 定義請參考：[Schema 目錄說明](schemas/README.md)

### 資料型態要求

| 欄位類型 | JSON 型態 | 格式要求 | 範例 |
|---------|-----------|----------|------|
| 時間戳 | integer | Unix 毫秒時間戳 | `1699123456789` |
| 設備 ID | string | 12 字符小寫十六進制 | `"aabbccddeeff"` |
| Schema | string | `type/version` 格式 | `"state/1.0"` |
| 布林值 | boolean | true/false | `true` |
| 數值 | number | 數字或浮點數 | `45.2` |
| 字串 | string | UTF-8 編碼 | `"ok"` |

## 錯誤處理

### 格式錯誤處理
當接收到格式不正確的訊息時：

1. **記錄錯誤**: 寫入錯誤日誌
2. **丟棄訊息**: 不處理無效訊息
3. **發送告警**: 通知監控系統
4. **統計錯誤**: 累計錯誤統計

### 常見錯誤類型
```json
{
  "error_type": "schema_validation_failed",
  "error_details": {
    "received_schema": "invalid/format",
    "expected_pattern": "^[a-z_]+/[0-9]+\\.[0-9]+$",
    "timestamp": 1699123456789
  }
}
```

## 最佳實踐

### 1. 欄位命名規範
- 使用 snake_case 命名格式
- 避免縮寫，使用完整單詞
- 使用語義化的欄位名稱

```json
// ✅ 好的命名
{
  "cpu_usage": 45.2,
  "memory_available_mb": 1024,
  "network_interface": "eth0"
}

// ❌ 避免的命名  
{
  "cpu": 45.2,
  "mem": 1024,
  "if": "eth0"
}
```

### 2. 資料結構設計
- 保持嵌套層級在 3 層以內
- 使用一致的資料型態
- 提供合理的預設值

### 3. 效能考量
- 避免過大的 JSON 訊息（建議 < 64KB）
- 頻繁的遙測資料使用簡化格式
- 對於大型資料使用分段傳送

### 4. 安全性考量
- 不在訊息中包含敏感資訊（密碼、金鑰）
- 對包含個人資訊的欄位進行加密或遮罩
- 驗證所有輸入資料

---

**下一步**: 閱讀 [MQTT 使用指南](07-mqtt-usage.md) 了解 QoS 設定和發布頻率最佳實踐。
# MQTT Diag 通訊協定規格 v1.0 (rtkMQTT)

定義設備與控制器間的 MQTT 診斷通訊協定，包含狀態回報、遙測、事件與命令的訊息格式。適用於 IoT 裝置、伺服器、網路設備等各類設備的診斷狀態回報。

---

## 1. 術語定義

* **Device（設備）**: 連接 MQTT Broker 的設備，包含 IoT 裝置、伺服器、網路設備等
* **Controller（控制器）**: 發送命令並收集設備診斷資料的雲端或本地服務
* **Broker**: MQTT 訊息代理伺服器
* **Tenant/Site**: 租戶/場域，用於資料隔離與路由
* **Topic**: MQTT 主題路徑
* **LWT**: Last Will and Testament 遺囑訊息
* **Diagnosis（診斷）**: 設備健康狀態、效能指標、錯誤資訊等診斷資料

---

## 2. 設計原則

* **統一主題結構**: 以設備為中心的上下行訊息路徑設計
* **可擴充性**: 命令與 Schema 獨立演進，使用語義化版本控制
* **可觀測性**: 支援全域與範圍訂閱模式，便於監控與除錯
* **安全性**: TLS 加密、ClientID 認證與 ACL 存取控制 (這一個版本不需要)
* **診斷導向**: 專注於設備健康狀態、效能指標與故障診斷資訊的結構化傳輸，特別支援 WiFi 連線品質、漫遊事件、掃描結果等無線網路診斷

---

## 3. Topic 命名空間

![MQTT Topic Structure](topic_structure.png)

根路徑採版本化：

```
rtk/v1/{tenant}/{site}/{device_id}/...
```

### 3.1 上行（Device → Controller）

* `state`：設備狀態摘要與診斷資訊（retained）
* `telemetry/{metric}`：遙測資料與效能指標，支援高頻分流
* `evt/{event_type}`：事件、告警、錯誤與診斷事件
* `attr`：設備靜態屬性與規格資訊（retained）
* `cmd/ack`：命令接收確認
* `cmd/res`：命令結果回覆
* `lwt`：LWT 上下線通知（retained）

### 3.2 下行（Controller → Device）

* `cmd/req`：命令請求（統一路徑）。

### 3.3 群組/廣播（可選）

裝置可額外訂閱其所屬群組命令：

```
rtk/v1/{tenant}/{site}/group/{group_id}/cmd/req
```

裝置對群組命令的 `ack/res` 仍回自己的裝置路徑，便於逐台追蹤。

---

## 4. Device ID 規範

* `device_id`：全域唯一標識符，不可變更，格式：`^[a-z0-9-]{20,40}$`（建議使用 ULID/KSUID）
* MQTT `clientId` = `device_id`
* 避免使用 MAC 或序號作為主鍵，此類資訊應存放於 `/attr` topic 或後端資料庫

---

## 5. 安全與存取控制 (Future Work)

此章節定義完整的安全機制與存取控制規範，包含：

### 5.1 傳輸安全 (待實作)
* TLS 加密設定
* Client certificate 認證
* Token-based 認證機制

### 5.2 存取控制清單 (待實作)
* 裝置端 Topic 發布/訂閱權限
* Controller 端權限管理
* 租戶間資料隔離

### 5.3 審計與監控 (待實作)
* 連線記錄與審計日誌
* 異常行為偵測
* 安全事件通報機制

> **注意**: 目前開發版本暫不實作安全控制機制，所有連線預設為可信任環境。

---

## 6. 設備上下線偵測

### 6.1 Last Will Testament (LWT) 機制

LWT 是 MQTT 的內建機制，用於偵測設備異常斷線。當設備與 MQTT Broker 的連線意外中斷時（如網路故障、設備當機），Broker 會自動發布預先設定的「遺囑訊息」通知其他訂閱者。

### 6.2 LWT 設定要求

設備在建立 MQTT 連線時必須設定以下 LWT 參數：

* **LWT Topic**: `rtk/v1/{tenant}/{site}/{device_id}/lwt`
* **LWT Payload**: `{"status":"offline","ts":<timestamp>}`
* **LWT Retained**: true

### 6.3 上下線狀態管理

**設備上線時**:
主動發布上線狀態到相同 topic：
```json
{
  "status": "online",
  "ts": 1723526400000
}
```

**設備正常下線時**:
發布離線狀態後再斷開連線：
```json
{
  "status": "offline", 
  "ts": 1723526500000,
  "reason": "normal_shutdown"
}
```

**設備異常斷線時**:
MQTT Broker 自動發布預設的 LWT 訊息，讓 Controller 立即得知設備離線狀態。

---

## 7. 共通 Payload 格式

### 7.1 什麼是共通 Payload 格式

所有透過 MQTT 傳送的 JSON 訊息都應該包含一些共通欄位，這些欄位用於：
- **版本控制**: 確保不同版本的設備和系統能正確解析訊息
- **時間追蹤**: 記錄事件發生的精確時間
- **除錯追蹤**: 在複雜系統中追蹤訊息流向

無論是狀態回報、遙測資料、事件通知還是命令，都使用相同的基本結構。

### 7.2 必要欄位說明

#### `schema` (字串)
- **用途**: 標識訊息的類型和版本，讓接收端知道如何解析這個訊息
- **格式**: `<訊息類型>/<版本號>` 
- **範例**: 
  - `state/1.0` - 設備狀態訊息 v1.0
  - `evt.wifi.roam_miss/1.0` - WiFi 漫遊失效事件 v1.0
  - `cmd.diagnosis.get/1.0` - 診斷資料請求命令 v1.0

#### `ts` (整數)
- **用途**: 記錄訊息產生或事件發生的時間，用於排序、關聯和分析
- **格式**: Unix timestamp 毫秒 (int64)
- **範例**: `1723526400000` (對應 2024-08-13 08:00:00 UTC)
- **注意**: 所有設備必須使用相同的時間基準

### 7.3 選用欄位說明

#### `trace` (物件，選用)
- **用途**: 在分散式系統中追蹤單一操作的完整流程
- **常用欄位**:
  - `req_id`: 請求唯一識別碼
  - `correlation_id`: 關聯多個相關訊息
  - `span_id`: 分散式追蹤的區段識別碼

**範例**:
```json
{
  "schema": "evt.wifi.connect_fail/1.0",
  "ts": 1723526400000,
  "trace": {
    "req_id": "conn-retry-001", 
    "correlation_id": "user-session-12345"
  },
  "severity": "error",
  "connection_info": {
    "ssid": "OfficeWiFi"
  }
}
```

### 7.4 完整訊息範例

**設備狀態訊息**:
```json
{
  "schema": "state/1.0",
  "ts": 1723526400000,
  "health": "ok",
  "cpu_usage": 45.2,
  "wifi_stats": {
    "rssi": -52,
    "connected": true
  }
}
```

**診斷事件訊息**:
```json
{
  "schema": "evt.wifi.arp_loss/1.0", 
  "ts": 1723526401000,
  "trace": {
    "correlation_id": "network-issue-001"
  },
  "severity": "warning",
  "arp_statistics": {
    "success_rate": 0.6
  }
}
```

### 7.5 版本相容性規則

* **前向相容**: 實作必須忽略未知的 JSON 欄位
* **版本升級**: 
  - 小版本升級 (1.0 → 1.1): 新增欄位，不移除現有欄位
  - 大版本升級 (1.x → 2.0): 可能移除或改變現有欄位
* **解析原則**: 如果 `schema` 版本不相容，應記錄警告但不中斷處理

---

## 8. MQTT 使用時機

### 8.1 發布頻率
* **狀態資料** (`state`): 每 30-60 秒或狀態改變時
* **遙測資料** (`telemetry/*`): 每 10-60 秒，依重要性調整
* **事件資料** (`evt/*`): 立即發布
* **設備屬性** (`attr`): 啟動時或屬性變更時

### 8.2 訂閱模式
* Controller 使用通配符訂閱: `rtk/v1/+/+/+/state`、`rtk/v1/+/+/+/evt/#`
* Device 只訂閱自己的命令: `rtk/v1/{tenant}/{site}/{device_id}/cmd/req`

### 8.3 命令處理
* Device 收到 `cmd/req` 後立即回 `cmd/ack` (< 1 秒)
* 命令執行完成後發布 `cmd/res`，包含結果或錯誤

---

## 9. 上行結構定義（Device → Controller）

### 9.1 `state`（retained）

**Topic**

```
rtk/v1/{tenant}/{site}/{device_id}/state
```

**Payload（範例）**

```json
{
  "schema": "state/1.0",
  "ts": 1723526400,
  "health": "ok",
  "fw": "1.2.3",
  "uptime_s": 4567,
  "cpu_usage": 45.2,
  "memory_usage": 62.8,
  "disk_usage": 78.5,
  "temperature_c": 42.1,
  "net": { "rssi": -62, "ip": "10.0.1.23", "bytes_rx": 1048576, "bytes_tx": 524288 },
  "diagnosis": {
    "last_error": null,
    "error_count": 0,
    "restart_count": 3
  }
}
```

**欄位說明**

* `health`: `ok|warn|error`，設備整體健康狀態
* `cpu_usage`: CPU 使用率百分比（0-100）
* `memory_usage`: 記憶體使用率百分比（0-100）
* `disk_usage`: 磁碟使用率百分比（0-100）
* `temperature_c`: 設備溫度（攝氏度）
* `net`: 網路介面資訊與統計
* `diagnosis`: 診斷相關資訊，包含錯誤狀態與統計

### 9.2 `telemetry/{metric}`

**Topic**

```
rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}
```

**Payload（範例：wifi.scan_result - 以 WiFi 設備為例）**

```json
{
  "schema": "telemetry.wifi.scan_result/1.0",
  "ts": 1723526400,
  "scan_info": {
    "internal_scan_skip_cnt": 10,
    "environment_scan_ap_number": 8,
    "current_bssid": "aa:bb:cc:dd:ee:ff",
    "current_rssi": -45
  },
  "roam_candidates": [
    {
      "bssid": "11:22:33:44:55:66",
      "rssi": -42,
      "channel": 6,
      "ap_load": 30
    },
    {
      "bssid": "77:88:99:aa:bb:cc", 
      "rssi": -48,
      "channel": 11,
      "ap_load": 25
    }
  ],
  "scan_timing": {
    "last_scan_time": 1723526395,
    "last_full_scan_complete_time": 1723526380
  }
}
```

**常見 metric 類型:**
* **硬體診斷**: `temperature`, `cpu_usage`, `memory_usage`, `disk_usage`, `fan_speed`
* **網路診斷**: `interface.eth0.rx_bytes`, `interface.eth0.tx_bytes`, `ping_latency`, `bandwidth_usage`
* **WiFi 診斷**: `wifi.scan_result`, `wifi.roam_candidate`, `wifi.connection_quality`, `wifi.rssi`, `wifi.channel_utilization`
* **應用診斷**: `response_time`, `error_rate`, `queue_depth`, `connection_count`
* **IoT 特定**: `battery_voltage`, `humidity`, `power_consumption`, `signal_strength`

高頻診斷資料應分 metric 發布以降低傳輸成本。

### 9.3 `evt/{event_type}`（事件/告警）

**Topic**

```
rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}
```

**Payload（範例：system.error）**

```json
{
  "schema": "evt.system.error/1.0",
  "ts": 1723526401,
  "seq": 1023,
  "severity": "warning",
  "error_code": "HIGH_TEMPERATURE",
  "message": "CPU temperature exceeded 80°C",
  "source": "thermal_monitor",
  "details": {
    "current_temp": 82.5,
    "threshold": 80.0,
    "location": "cpu_core_0"
  }
}
```

**常見 event_type 類型:**
* **系統診斷**: `system.error`, `system.warning`, `system.recovery`
* **硬體事件**: `hardware.fault`, `hardware.overheat`, `power.failure`
* **網路事件**: `network.disconnected`, `network.latency_high`, `interface.down`
* **WiFi 事件**: `wifi.roam_triggered`, `wifi.scan_failed`, `wifi.signal_weak`, `wifi.ap_changed`
* **應用事件**: `service.crashed`, `memory.low`, `disk.full`

### 9.4 `attr`（retained，裝置屬性）

**Topic**

```
rtk/v1/{tenant}/{site}/{device_id}/attr
```

**Payload（範例）**

```json
{
  "schema": "attr/1.0",
  "model": "TB-Hub-8K",
  "sn": "A1B2C3",
  "hw": "revC",
  "schema_state": "1.0",
  "cap": { "light": true, "ports": 8 }
}
```

**欄位說明**

* `schema`: 訊息格式版本標識（必要的共通欄位）
* `model`: 設備型號，用於識別硬體規格和相容性
* `sn`: 設備序號，用於保固查詢和硬體追蹤
* `hw`: 硬體版本，如 PCB 版本，用於韌體相容性判斷
* `schema_state`: 此設備支援的 `state` 訊息格式版本
* `cap`: 設備能力描述，讓 Controller 知道設備支援哪些功能
  - `light`: 是否支援燈控功能
  - `ports`: 設備的埠數量

---

## 10. 下行命令（Controller → Device）

所有命令統一走 `cmd/req`，裝置收到即回 `cmd/ack`，完成後回 `cmd/res`。

### 10.1 `cmd/req`（Controller → Device）

**Topic**

```
rtk/v1/{tenant}/{site}/{device_id}/cmd/req
```

**Payload**

```json
{
  "id": "c-1001",
  "op": "light.set",
  "schema": "cmd.light.set/1.0",
  "args": { "on": true, "brightness": 80 },
  "timeout_ms": 5000,
  "expect": "result",  
  "reply_to": null,
  "ts": 1723526400
}
```

* `id`：命令唯一識別，去重與關聯 `ack/res`。
* `op`：命令名稱（資源導向，如 `device.reboot`、`net.wifi.config`）。
* `expect`：`ack|result|none`。
* `reply_to`：如需回到不同 topic 可覆寫（一般為空）。

### 10.2 `cmd/ack`（Device → Controller）

**Topic**

```
rtk/v1/{tenant}/{site}/{device_id}/cmd/ack
```

**Payload**

```json
{
  "id": "c-1001",
  "ts": 1723526401,
  "accepted": true,
  "err": null
}
```

### 10.3 `cmd/res`（Device → Controller）

**Topic**

```
rtk/v1/{tenant}/{site}/{device_id}/cmd/res
```

**Payload（成功）**

```json
{
  "id": "c-1001",
  "ts": 1723526403,
  "ok": true,
  "result": { "on": true, "brightness": 80 },
  "progress": null,
  "err": null
}
```

**Payload（失敗）**

```json
{
  "id": "c-1001",
  "ts": 1723526403,
  "ok": false,
  "result": null,
  "err": { "code": "E_UNSUPPORTED", "msg": "capability not available" }
}
```

* 長任務可在處理期間定期發布含 `progress` 的 `res`（0\~100 或階段字串），最後再送最終 `res`。

---

## 11. 典型命令清單（建議命名）

| `op`              | 說明       | `args` 範例                                                       | 結果 `result` 範例                                     |
| ----------------- | -------- | --------------------------------------------------------------- | -------------------------------------------------- |
| `light.set`       | 設定燈狀態    | `{ "on": true, "brightness": 80, "color": "#ffaa00" }`          | `{ "on": true, "brightness": 80 }`                 |
| `device.reboot`   | 重新啟動     | `{}`                                                            | `{ "uptime_s": 0 }`                                |
| `report.push`     | 立即回報特定資料 | `{ "what": ["state", "telemetry.temperature"] }`                | `{ "pushed": ["state", "telemetry.temperature"] }` |
| `diagnosis.get`   | 取得診斷資料   | `{ "type": "wifi", "detail_level": "full" }`                    | 設備相依的診斷資料結構                                        |
| `fw.update`       | 韌體更新     | `{ "version": "1.2.4", "url": "https://...", "sha256": "..." }` | `{ "phase": "done", "version": "1.2.4" }`          |
| `net.wifi.config` | 設定 Wi‑Fi | `{ "ssid": "x", "psk": "y" }`                                   | `{ "connected": true, "ip": "..." }`               |

> 命令的實際清單由各產品線維護，並以 `schema` 版本化。

---

## 12. 診斷資料傳輸機制

### 12.1 主動診斷事件傳輸
設備在檢測到異常狀況時，會主動透過 `evt/{event_type}` 發送診斷事件，包含：
- 觸發條件與嚴重程度
- 初步診斷資訊與環境參數  
- 建議後續動作

### 12.2 被動詳細診斷請求  
Controller 可透過 `diagnosis.get` 命令主動請求設備回報特定類型的詳細診斷資料。設備收到診斷命令後，應立即透過 `cmd/res` 回傳當前的完整診斷狀態。

### 12.3 診斷命令格式

**Command Request:**
```json
{
  "id": "diag-001",
  "op": "diagnosis.get",
  "schema": "cmd.diagnosis.get/1.0",
  "args": {
    "type": "wifi",
    "detail_level": "full",
    "include_history": false
  },
  "timeout_ms": 10000,
  "expect": "result",
  "ts": 1723526400
}
```

**Command Response (範例 - WiFi 設備):**
```json
{
  "id": "diag-001",
  "ts": 1723526401,
  "ok": true,
  "result": {
    "diagnosis_type": "wifi",
    "device_type": "wifi_router",
    "data": {
      "current_connection": {
        "bssid": "aa:bb:cc:dd:ee:ff",
        "rssi": -45,
        "channel": 6,
        "link_speed": 150
      },
      "scan_results": [
        {
          "bssid": "11:22:33:44:55:66",
          "ssid": "AP_Name_1",
          "rssi": -42,
          "channel": 6
        }
      ],
      "roam_history": [
        {
          "timestamp": 1723526300,
          "from_bssid": "ff:ee:dd:cc:bb:aa",
          "to_bssid": "aa:bb:cc:dd:ee:ff",
          "reason": "signal_weak"
        }
      ]
    }
  }
}
```

### 12.3 設備相依性
* **診斷資料內容**: 每種設備類型的診斷資料結構完全不同
* **支援的診斷類型**: 各設備根據硬體能力支援不同的 `type` 參數
* **回應時間**: 複雜診斷可能需要較長處理時間，建議設定適當的 `timeout_ms`

### 12.4 常見診斷類型
* `wifi`: WiFi 連線狀態、掃描結果、漫遊記錄
* `network`: 網路介面統計、路由表、連線狀態
* `system`: CPU、記憶體、磁碟使用狀況
* `hardware`: 硬體感測器資料、溫度、風扇轉速
* `application`: 應用程式狀態、服務運行情況

---

## 13. 錯誤碼建議

| 代碼               | 說明              |
| ---------------- | --------------- |
| `E_TIMEOUT`      | 命令處理逾時          |
| `E_UNSUPPORTED`  | 裝置不支援該 `op` 或參數 |
| `E_BUSY`         | 裝置忙碌，無法處理       |
| `E_INVALID_ARGS` | 參數格式錯誤          |
| `E_FORBIDDEN`    | 權限不足或 ACL 拒絕    |
| `E_INTERNAL`     | 內部錯誤            |

---

## 14. 版本控管

* **Topic 版本**：`rtk/v1/...`；破壞式變更升 `v2`。
* **Schema 版本**：每種 payload `schema` 採語意化版本（SemVer）。
* **相容原則**：

  * 裝置/後端對未知欄位需忽略。
  * 新增欄位為不破壞性；移除/改義需升大版並逐步淘汰。

---

## 15. 順序、重送與冪等

* **命令去重**：裝置應以 `id` 做去重，對重複 `id` 只執行一次，重傳先回覆既有結果。
* **超時與重試**：Controller 端可在 `timeout_ms` 到期後重試；重試必須沿用同一個 `id`。
* **送達順序**：MQTT 僅保證同一連線與同一 topic 的消息順序；跨 topic 需以 `ts/seq` 校正。

---

## 16. 監控與審計建議

* 監控訂閱：`rtk/v1/+/+/+/cmd/#`、`rtk/v1/+/+/+/evt/#`、`rtk/v1/+/+/+/lwt`。
* 寫入審計：所有 `cmd` 的 `req/ack/res` 需落庫（含 `tenant/site/device_id/id/op/ts`）。
* 異常告警：

  * 同一 `device_id` 多重連線（疑似複製）。
  * 過久未更新 `state` 或頻繁 `offline`。

---

## 17. 測試案例（最低集合）

1. **裝置上線／下線**：

   * 上線發布 `lwt: online`（retained），Broker 斷線自動發布 `offline`。
2. **狀態 retained**：

   * 新訂閱者立即收到最後一筆 `state`/`attr`。
3. **命令 RPC**：

   * Controller 發 `cmd/req` → 裝置回 `ack` → 完成回 `res`（成功/失敗皆測）。
4. **重試與冪等**：

   * Controller 重送相同 `id`，裝置不得重複執行，需回覆既有結果。
5. **群組命令**（如使用）：

   * 群組下發 → 裝置各自回到自身 `ack/res`。
6. **ACL 驗證**：

   * 裝置嘗試超權限 publish/subscribe 應被拒絕。

---

## 18. JSON Schema（簡化示例）

> 下列為示例片段，實務可拆成多檔版本化維護。

### 17.1 `state/1.0`

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "state/1.0",
  "type": "object",
  "required": ["schema", "ts", "health"],
  "properties": {
    "schema": {"const": "state/1.0"},
    "ts": {"type": "number"},
    "health": {"enum": ["ok", "warn", "error"]},
    "fw": {"type": "string"},
    "uptime_s": {"type": "number"},
    "battery_v": {"type": "number"},
    "net": {
      "type": "object",
      "properties": {
        "rssi": {"type": "number"},
        "ip": {"type": "string"}
      },
      "additionalProperties": true
    }
  },
  "additionalProperties": true
}
```

### 17.2 `cmd.light.set/1.0`

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "cmd.light.set/1.0",
  "type": "object",
  "required": ["id", "op", "schema", "args"],
  "properties": {
    "id": {"type": "string"},
    "op": {"const": "light.set"},
    "schema": {"const": "cmd.light.set/1.0"},
    "args": {
      "type": "object",
      "properties": {
        "on": {"type": "boolean"},
        "brightness": {"type": "number", "minimum": 0, "maximum": 100},
        "color": {"type": "string"}
      },
      "additionalProperties": false
    },
    "timeout_ms": {"type": "number"},
    "expect": {"enum": ["ack", "result", "none"]},
    "reply_to": {"type": ["string", "null"]},
    "ts": {"type": "number"}
  },
  "additionalProperties": true
}
```

---

## 19. 實作指南（裝置端）

1. 以 `clientId=device_id` 連線 Broker，設置 LWT `offline`（retained）。
2. 上線後發布 `lwt=online` 與最新 `attr/state`。
3. 訂閱 `cmd/req`（以及群組 req，如有）。
4. 收到命令：

   * 解析 payload → 立刻回 `ack`（含 `id`）。
   * 執行命令；長任務定期回 `res`（含 `progress`）。
   * 完成後回最終 `res`（`ok=true/false`）。
5. 週期性發布 `state` 與必要 `telemetry`，異常以 `evt/*` 通報。

---

## 20. 實作指南（Controller/後端）

1. 封裝 `sendCommand(op, args, device_id, timeout)`：

   * 產生 `id` → 發布 `cmd/req`。
   * 等待 `ack`，超時重試（同 `id`）。
   * 視 `expect` 等待 `res`，處理超時與最終態。
2. 日誌/審計：完整記錄 `req/ack/res` 與錯誤碼。
3. 指標/告警：命令成功率、延遲分佈、裝置上線率、事件頻度等。

---

## 21. 範例訂閱樣式

* 全租戶所有命令流：`rtk/v1/+/+/+/cmd/#`
* 某站台所有裝置狀態：`rtk/v1/{tenant}/{site}/+/state`
* 某裝置所有遙測：`rtk/v1/{tenant}/{site}/{device_id}/telemetry/#`

---

## 22. WiFi 診斷實際應用範例

本章節基於真實的 WiFi 診斷資料，展示完整的 MQTT 診斷通訊流程，涵蓋漫遊問題、連線失敗、網路異常等典型場景。

### 23.1 漫遊問題診斷範例

#### 情境描述
辦公室 AP `office-ap-001` 檢測到 RSSI 降至 -75dBm 持續 10 秒，但未觸發漫遊機制。

#### 完整 MQTT 流程

**步驟 1: 事件觸發**  
Topic: `rtk/v1/office/floor1/office-ap-001/evt/wifi.roam_miss`

```json
{
  "schema": "evt.wifi.roam_miss/1.0",
  "ts": 1723526401000,
  "severity": "warning",
  "trigger_info": {
    "rssi_threshold": -70,
    "duration_ms": 10000,
    "cooldown_ms": 300000
  },
  "diagnosis": {
    "internal_scan_skip_count": 3,
    "environment_ap_count": 8,
    "candidate_ap_count": 2,
    "current_bssid": "aa:bb:cc:dd:ee:ff",
    "current_rssi": -75,
    "candidates": {
      "5g": {
        "bssid": "11:22:33:44:55:66",
        "rssi": -42,
        "channel": 36
      },
      "6g": {
        "bssid": "77:88:99:aa:bb:cc",
        "rssi": -48,
        "channel": 37
      }
    },
    "scan_timing": {
      "last_scan_time": 1723526395000,
      "last_full_scan_complete_time": 1723526380000
    }
  }
}
```

**步驟 2: Controller 請求詳細診斷**  
Topic: `rtk/v1/office/floor1/office-ap-001/cmd/req`

```json
{
  "id": "roam-diag-001",
  "op": "diagnosis.get",
  "schema": "cmd.diagnosis.get/1.0",
  "args": {
    "type": "wifi.roaming",
    "detail_level": "full",
    "include_history": true,
    "include_rf_stats": true
  },
  "timeout_ms": 15000,
  "expect": "result",
  "ts": 1723526402000
}
```

**步驟 3: Device 回傳詳細診斷結果**  
Topic: `rtk/v1/office/floor1/office-ap-001/cmd/res`

```json
{
  "id": "roam-diag-001",
  "ts": 1723526403500,
  "ok": true,
  "result": {
    "diagnosis_type": "wifi.roaming",
    "data": {
      "roaming_analysis": {
        "trigger_reasons": ["poor_signal_quality", "scan_skip_detected"],
        "skip_analysis": {
          "total_skips_10sec": 3,
          "skip_reasons": ["scan_in_progress", "channel_switch_delay"],
          "last_successful_scan": 1723526395000
        }
      },
      "current_connection": {
        "bssid": "aa:bb:cc:dd:ee:ff",
        "ssid": "OfficeWiFi-5G",
        "rssi": -75,
        "channel": 149,
        "bandwidth": "80MHz",
        "connection_duration_ms": 1847500
      },
      "roam_candidates": [
        {
          "band": "5G",
          "bssid": "11:22:33:44:55:66",
          "rssi": -42,
          "channel": 36,
          "load_percentage": 25,
          "roam_score": 85
        },
        {
          "band": "6G", 
          "bssid": "77:88:99:aa:bb:cc",
          "rssi": -48,
          "channel": 37,
          "load_percentage": 15,
          "roam_score": 92
        }
      ],
      "rf_statistics": {
        "interference_level": "moderate",
        "channel_utilization_percent": 45,
        "retry_rate_percent": 12.5
      }
    }
  }
}
```

### 23.2 連線失敗診斷範例

#### 情境描述
筆記型電腦 `laptop-005` 嘗試連線企業 WiFi 時在 WPA3 SAE 認證階段失敗。

**事件觸發**  
Topic: `rtk/v1/corporate/building-a/laptop-005/evt/wifi.connect_fail`

```json
{
  "schema": "evt.wifi.connect_fail/1.0",
  "ts": 1723526501000,
  "severity": "error",
  "connection_info": {
    "role_type": "STA",
    "target_bssid": "cc:dd:ee:ff:00:11",
    "ssid": "CorporateNet-5G",
    "security_type": "WPA3_SAE"
  },
  "failure_details": {
    "join_status": "AUTH_TIMEOUT",
    "tx_fail_category": "AUTH",
    "auth_mode": "WPA3SAE",
    "auth_algo": "SAE",
    "response_status_code": 0,
    "failure_stage": "authentication"
  }
}
```

**診斷結果** (簡化)  
Topic: `rtk/v1/corporate/building-a/laptop-005/cmd/res`

```json
{
  "id": "connect-diag-002",
  "ok": true,
  "result": {
    "data": {
      "connection_stages": {
        "authentication": {
          "status": "timeout",
          "duration_ms": 2500,
          "sae_exchange": {
            "commit_sent": true,
            "commit_response_received": false
          }
        }
      },
      "failure_analysis": {
        "primary_cause": "sae_timeout",
        "contributing_factors": ["ap_sae_processing_delay", "possible_ap_overload"],
        "recommendations": ["retry_with_different_ap", "check_ap_load_balance"]
      }
    }
  }
}
```

### 23.3 ARP 遺失診斷範例

#### 情境描述
智能攝影機 `smart-camera-003` 偵測到連續 2 次 ARP response 未收到。

**事件觸發**  
Topic: `rtk/v1/factory/workshop/smart-camera-003/evt/wifi.arp_loss`

```json
{
  "schema": "evt.wifi.arp_loss/1.0",
  "ts": 1723526601000,
  "severity": "warning",
  "trigger_info": {
    "consecutive_loss_count": 2,
    "cooldown_ms": 300000
  },
  "network_info": {
    "current_bssid": "bb:cc:dd:ee:ff:00",
    "current_rssi": -67,
    "channel_hw_match": true,
    "scan_in_progress": false
  },
  "arp_statistics": {
    "source_ip": "192.168.10.203",
    "destination_ip": "192.168.10.1",
    "req_tx_fail_count": 2,
    "req_count": 5,
    "rsp_count": 3,
    "success_rate": 0.6
  }
}
```

**診斷分析結果** (簡化)

```json
{
  "result": {
    "data": {
      "root_cause_analysis": {
        "primary_causes": ["channel_interference", "ap_load_fluctuation"],
        "probability_scores": {
          "rf_interference": 0.8,
          "ap_overload": 0.6,
          "network_congestion": 0.7
        }
      },
      "rf_diagnostics": {
        "channel_analysis": {
          "channel_utilization": 65,
          "interference_sources": [
            {
              "type": "industrial_equipment",
              "strength": "moderate",
              "estimated_impact": "medium"
            }
          ]
        }
      }
    }
  }
}
```

### 23.4 時序圖與通訊流程

#### 典型診斷流程
![Generic MQTT Diagnostic Sequence](generic_sequence_simple.png)

#### 訂閱模式建議
```
# Controller 全域監控
rtk/v1/+/+/+/evt/wifi.#     # 所有 WiFi 事件
rtk/v1/+/+/+/lwt             # 設備上下線狀態
rtk/v1/+/+/+/state           # 設備健康狀態

# 特定場域監控  
rtk/v1/office/+/+/evt/#      # 辦公室所有事件
rtk/v1/factory/+/+/evt/wifi.# # 工廠 WiFi 事件

# 設備類型監控
rtk/v1/+/+/smart-camera-+/evt/# # 智能攝影機事件
```

### 23.5 診斷資料結構定義

#### WiFi 漫遊候選 AP 結構
```json
{
  "wifi_roam_candidate": {
    "bssid": "string (MAC address format)",
    "rssi": "integer (-100 to 0)",
    "channel": "integer (1-165)",
    "load_percentage": "integer (0-100)",
    "roam_score": "integer (0-100)"
  }
}
```

#### 連線失敗分析結構
```json
{
  "connection_failure_analysis": {
    "primary_cause": "enum [auth_timeout, assoc_timeout, key_install_fail]",
    "failure_stage": "enum [beacon_detection, authentication, association, four_way_handshake]",
    "contributing_factors": ["array of strings"],
    "recommendations": ["array of strings"]
  }
}
```

#### ARP 統計結構
```json
{
  "arp_statistics": {
    "source_ip": "string (IPv4)",
    "destination_ip": "string (IPv4)",
    "req_count": "integer",
    "rsp_count": "integer", 
    "success_rate": "number (0-1)",
    "avg_response_time_ms": "number"
  }
}
```

### 23.6 實作指南

#### Device 端實作要點
1. **事件觸發條件**: 依據個別單位資料中的觸發條件實作
2. **診斷資料收集**: 結合 RF 統計、流量分析、環境掃描
3. **冷卻機制**: 5 分鐘內相同事件不重複發送
4. **資料結構**: 遵循 JSON Schema 定義

#### Controller 端實作要點
1. **智能診斷請求**: 根據事件類型請求相應的診斷資料
2. **根因分析**: 利用診斷結果進行自動化分析
3. **修復建議**: 提供可執行的修復動作命令
4. **趨勢監控**: 追蹤診斷事件的模式和頻率

---

## 23. 變更紀錄（Changelog）

* **1.0**：首版，定義統一路徑、RPC 命令模型、Retained/LWT、安全與 ACL、群組命令與 Shadow 選項、JSON Schema 範例、WiFi 診斷實際應用範例。

---


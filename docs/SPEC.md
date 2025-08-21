# MQTT Diagnosis 通訊協定規格 v1.0 (rtkMQTT)

定義設備與控制器間的 MQTT 診斷通訊協定，包含狀態回報、遙測、事件與命令的訊息格式。適用於 IoT 裝置、伺服器、網路設備等各類設備的診斷狀態回報。

📋 **相關文檔**: 詳細的 MQTT payload 格式與範例請參考 [EXAMPLES.md](EXAMPLES.md)

## 前置要求：必要的第三方程式庫

在開始實作本協議前，請先確保您的開發環境已正確安裝以下經過測試驗證的第三方程式庫：

### cJSON - JSON 解析程式庫

**用途**: 處理 MQTT payload 中的 JSON 訊息格式，包括編碼、解碼和驗證
**官方網站**: https://github.com/DaveGamble/cJSON
**測試版本**: v1.7.15 或更新版本

**安裝方式**:
```bash
# Ubuntu/Debian
sudo apt-get install libcjson-dev

# CentOS/RHEL
sudo yum install cjson-devel

# 或從原始碼編譯 (FreeRTOS)
git clone https://github.com/DaveGamble/cJSON.git
cd cJSON && mkdir build && cd build
cmake .. && make && sudo make install
```

### libmosquitto - MQTT 客戶端程式庫

**用途**: 提供 MQTT 協議的完整實作，包括連接、發布、訂閱和 LWT 機制
**官方網站**: https://mosquitto.org/
**測試版本**: v2.0.15 或更新版本

**安裝方式**:
```bash
# Ubuntu/Debian
sudo apt-get install libmosquitto-dev

# CentOS/RHEL
sudo yum install mosquitto-devel

# 或從原始碼編譯 (FreeRTOS 可以直接使用 AmebaSDK)
wget https://mosquitto.org/files/source/mosquitto-2.0.15.tar.gz
tar -xzf mosquitto-2.0.15.tar.gz
cd mosquitto-2.0.15 && make && sudo make install
```

**重要提醒**:
- 這兩個程式庫已在我們的測試環境中驗證相容性和穩定性
- 建議優先使用指定版本以確保最佳相容性
- 安裝完成後請執行基本測試確認程式庫正常運作

---

## 1. 術語定義

* **Device（設備）**: 連接 MQTT Broker 的設備，包含 IoT 裝置、伺服器、網路設備等
* **Controller（控制器）**: 發送命令並收集設備診斷資料的雲端或本地服務
* **Broker**: MQTT 訊息代理伺服器
* **Tenant/Site**: 租戶/場域，用於資料隔離與路由
* **Topic**: MQTT 主題路徑
* **LWT**: Last Will and Testament 遺囑訊息。MQTT 的內建機制，當設備意外斷線時（如網路故障、設備當機），MQTT Broker 會自動發布事先設定的「遺囑訊息」通知其他訂閱者，實現即時的離線檢測。這是確保 IoT 系統可靠性的關鍵機制。(Optional)
* **Diagnosis（診斷）**: 設備健康狀態、效能指標、錯誤資訊等診斷資料

---

## 2. 設計原則

* **統一主題結構**: 以設備為中心的上下行訊息路徑設計
* **可擴充性**: 命令與 Schema 獨立演進，使用語義化版本控制
* **可觀測性**: 支援全域與範圍訂閱模式，便於監控與除錯
* **診斷導向**: 專注於設備健康狀態、效能指標與故障診斷資訊的結構化傳輸，特別支援 WiFi 連線品質、漫遊事件、掃描結果等無線網路診斷

---

## 3. Topic 命名空間

### 3.0 路徑結構詳解

根路徑採版本化設計，提供清晰的層級式組織：

```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}
```

**路徑組成說明**：
- `rtk/v1`：協議命名空間與版本，破壞性變更時升級至 v2
- `{tenant}`：租戶標識，用於多租戶環境的資料隔離（如：`office`、`factory`、`home`）
- `{site}`：場域標識，同租戶下的物理位置區分（如：`floor1`、`workshop-a`、`living-room`）
- `{device_id}`：設備 MAC 地址，格式：`^[a-f0-9]{12}$`（如：`aabbccddeeff`）
- `{message_type}`：訊息類型，定義訊息的用途與格式，支援多層級結構
  - **狀態類**：`state`（設備整體狀態）、`attr`（設備屬性）、`lwt`（上下線狀態）
  - **遙測類**：`telemetry/{metric}` 如 `telemetry/temperature`、`telemetry/cpu_usage`、`telemetry/wifi_clients`
  - **事件類**：`evt/{event_type}` 如 `evt/wifi.connection_lost`、`evt/system.error`、`evt/hardware.overheat`
  - **命令類**：`cmd/req`（命令請求）、`cmd/ack`（命令確認）、`cmd/res`（命令結果）
  - **拓撲類**：`topology/discovery`（拓撲發現）、`topology/connections`（連接狀態）
  - **診斷類**：`diagnostics/speed_test`（速度測試）、`diagnostics/wan_test`（WAN測試）

**實際範例**：
```
rtk/v1/office/floor1/aabbccddeeff/state
rtk/v1/factory/workshop-a/112233445566/telemetry/temperature
rtk/v1/home/living-room/778899aabbcc/evt/wifi.connection_lost
rtk/v1/corporate/building-a/ddeeff001122/cmd/req
```

**命名規範**：
- tenant/site 使用小寫字母、數字和連字號
- tenant/site 建議使用有意義的業務標識
- device_id 必須使用設備 MAC 地址的小寫無冒號格式

### 3.1 上行（Device → Controller）

設備主動向 Controller 發送的訊息類型，涵蓋狀態、遙測、事件和拓撲資訊。

#### 狀態與屬性類
* **`state`**：設備狀態摘要與診斷資訊（retained）
  - **使用時機**：每 30-60 秒或狀態改變時
  - **範例**：`rtk/v1/office/floor1/aabbccddeeff/state`
  - **內容**：健康狀態、資源使用率、網路連線狀態
  
* **`attr`**：設備靜態屬性與規格資訊（retained）
  - **使用時機**：設備啟動時或屬性變更時
  - **範例**：`rtk/v1/factory/line-a/445566778899/attr`
  - **內容**：型號、序號、硬體版本、設備能力

#### 遙測資料類
* **`telemetry/{metric}`**：遙測資料與效能指標，支援高頻分流
  - **使用時機**：每 10-60 秒，依重要性調整頻率
  - **範例**：
    ```
    rtk/v1/home/kitchen/112233445566/telemetry/temperature
    rtk/v1/office/floor2/778899aabbcc/telemetry/wifi_clients
    rtk/v1/factory/workshop/ddeeff001122/telemetry/vibration
    ```
  - **常見 metric**：`temperature`、`cpu_usage`、`wifi_clients`、`bandwidth_usage`

#### 事件與告警類
* **`evt/{event_type}`**：事件、告警、錯誤與診斷事件
  - **使用時機**：事件發生時立即發布
  - **範例**：
    ```
    rtk/v1/home/living-room/445566778899/evt/wifi.connection_lost
    rtk/v1/office/floor1/aabbccddeeff/evt/system.low_toner
    rtk/v1/factory/line-b/112233445566/evt/hardware.overheat
    ```
  - **常見事件**：`wifi.roam_triggered`、`system.error`、`hardware.fault`

#### 命令回應類
* **`cmd/ack`**：命令接收確認
  - **使用時機**：收到 `cmd/req` 後立即回應（< 1 秒）
  - **範例**：`rtk/v1/home/bedroom/778899aabbcc/cmd/ack`
  
* **`cmd/res`**：命令結果回覆
  - **使用時機**：命令執行完成後發布結果
  - **範例**：`rtk/v1/office/conference/ddeeff001122/cmd/res`

#### 網路拓撲類
* **`topology/discovery`**：網路拓撲發現資訊（設備介面、能力等）
  - **使用時機**：設備啟動或網路配置變更時
  - **範例**：`rtk/v1/office/floor1/445566778899/topology/discovery`
  - **內容**：網路介面、路由表、設備能力

* **`topology/connections`**：設備連接狀態報告（ARP 表、DHCP 租約等）
  - **使用時機**：每 2-5 分鐘或連接狀態變更時
  - **範例**：`rtk/v1/home/main/aabbccddeeff/topology/connections`
  - **內容**：已連接設備、ARP 表、DHCP 租約

#### 上下線狀態類
* **`lwt`**：LWT 上下線通知（retained）
  - **使用時機**：連線建立、正常斷線、異常斷線
  - **範例**：`rtk/v1/factory/warehouse/112233445566/lwt`
  - **內容**：`{"status":"online/offline","ts":"2024-08-13T08:00:00.000Z"}`

### 3.2 下行（Controller → Device）

Controller 向特定設備發送的命令與控制訊息，採用統一的命令通道設計。

* **`cmd/req`**：命令請求（統一路徑）
  - **使用時機**：需要設備執行特定操作時
  - **範例**：
    ```
    rtk/v1/home/living-room/air-conditioner/cmd/req    # 冷氣控制
    rtk/v1/office/floor1/router-001/cmd/req            # 路由器配置
    rtk/v1/factory/line-a/plc-controller/cmd/req       # PLC 控制
    rtk/v1/hospital/ward-b/monitor-003/cmd/req         # 監控設備操作
    ```
  - **常見命令**：
    - 設備控制：`light.set`、`device.reboot`、`net.wifi.config`
    - 診斷請求：`diagnosis.get`、`topology.discover`
    - 配置管理：`fw.update`、`identity.set`
    - 測試執行：`diagnostics.speed_test`、`diagnostics.wan_test`
  
  **訂閱模式**：設備只需訂閱自己的命令通道
  ```
  rtk/v1/{tenant}/{site}/{device_id}/cmd/req
  ```

### 3.3 群組/廣播（Optional）

支援群組命令發送，適用於需要同時控制多個設備的場景。

**群組命令 Topic**：
```
rtk/v1/{tenant}/{site}/group/{group_id}/cmd/req
```

**使用場景與範例**：
- **照明控制**：`rtk/v1/office/floor1/group/meeting-room-lights/cmd/req`
- **安全系統**：`rtk/v1/factory/production/group/emergency-stops/cmd/req`  
- **環境控制**：`rtk/v1/hospital/icu/group/air-purifiers/cmd/req`
- **網路設備**：`rtk/v1/campus/building-a/group/access-points/cmd/req`

**回應機制**：
- 設備收到群組命令後，`ack/res` 仍回到各自的設備路徑
- Controller 可逐台追蹤群組命令的執行狀態
- 範例回應路徑：
  ```
  rtk/v1/office/floor1/light-001/cmd/ack  # 燈具1回應
  rtk/v1/office/floor1/light-002/cmd/ack  # 燈具2回應
  rtk/v1/office/floor1/light-003/cmd/ack  # 燈具3回應
  ```

**訂閱配置**：設備需要同時訂閱個別命令和群組命令
```
rtk/v1/{tenant}/{site}/{device_id}/cmd/req        # 個別命令
rtk/v1/{tenant}/{site}/group/{group_id}/cmd/req   # 群組命令
```

### 3.4 訂閱模式指南

不同角色的訂閱模式設計，提供 Controller 靈活的監控與管理能力。

#### Controller 全域監控訂閱
```bash
# 全域事件監控
rtk/v1/+/+/+/evt/#              # 所有租戶的所有事件
rtk/v1/+/+/+/lwt                # 所有設備上下線狀態
rtk/v1/+/+/+/state              # 所有設備健康狀態

# 分類監控
rtk/v1/+/+/+/evt/wifi.#         # 所有 WiFi 相關事件
rtk/v1/+/+/+/evt/system.#       # 所有系統事件
rtk/v1/+/+/+/topology/#         # 所有拓撲相關訊息
```

#### 租戶/場域層級監控訂閱
```bash
# 特定租戶監控
rtk/v1/office/+/+/evt/#         # 辦公室所有事件
rtk/v1/factory/+/+/state        # 工廠所有設備狀態
rtk/v1/hospital/+/+/telemetry/# # 醫院所有遙測資料

# 特定場域監控
rtk/v1/office/floor1/+/evt/#    # 辦公室1樓所有事件
rtk/v1/factory/workshop-a/+/telemetry/temperature  # A車間溫度
```

#### 設備類型監控訂閱
```bash
# 按設備類型分組
rtk/v1/+/+/router-+/topology/#  # 所有路由器拓撲資訊
rtk/v1/+/+/sensor-+/telemetry/# # 所有感測器遙測資料
rtk/v1/+/+/ap-+/evt/wifi.#      # 所有 AP 的 WiFi 事件
rtk/v1/+/+/camera-+/evt/#       # 所有攝影機事件
```

#### 設備端訂閱（設備自身）
```bash
# 基本訂閱（必須）
rtk/v1/{tenant}/{site}/{device_id}/cmd/req

# 群組訂閱（可選）
rtk/v1/{tenant}/{site}/group/{group_id}/cmd/req

# 廣播訂閱（可選）
rtk/v1/{tenant}/broadcast/cmd/req
rtk/v1/broadcast/cmd/req
```

#### 萬用字元使用規則
- **`+`**：匹配單一層級的任意內容
  - `rtk/v1/+/floor1/+/state` - 任意租戶的1樓任意設備狀態
- **`#`**：匹配多層級的任意內容（只能在 topic 末尾使用）
  - `rtk/v1/office/+/+/evt/#` - 辦公室所有設備的所有事件

#### 效能優化建議
- **避免過寬的訂閱**：如 `rtk/v1/+/+/+/telemetry/#` 可能產生大量流量
- **使用具體的 event_type**：`rtk/v1/+/+/+/evt/wifi.roam_triggered` 比 `rtk/v1/+/+/+/evt/#` 更精確
- **分層訂閱**：先訂閱關鍵事件，再根據需要擴展

### 3.5 實際應用場景範例

#### 智慧辦公室場景
```
租戶：office
場域：floor1, floor2, conference-room-a
設備類型：router, ap, printer, projector, air-conditioner

典型 Topics：
rtk/v1/office/floor1/aabbccddeeff/state
rtk/v1/office/floor1/112233445566/telemetry/wifi_clients
rtk/v1/office/floor2/778899aabbcc/evt/system.low_toner
rtk/v1/office/conference-room-a/ddeeff001122/cmd/req
rtk/v1/office/floor1/group/hvac-system/cmd/req
```

#### 工廠生產線場景
```
租戶：factory
場域：workshop-a, workshop-b, quality-control
設備類型：plc, sensor, conveyor, robot-arm

典型 Topics：
rtk/v1/factory/workshop-a/aabbccddeeff/state
rtk/v1/factory/workshop-a/112233445566/telemetry/temperature
rtk/v1/factory/workshop-b/778899aabbcc/evt/hardware.overheat
rtk/v1/factory/quality-control/ddeeff001122/cmd/req
rtk/v1/factory/workshop-a/group/emergency-stops/cmd/req
```

#### 智慧家庭場景
```
租戶：home
場域：living-room, kitchen, bedroom, garage
設備類型：router, smart-tv, refrigerator, door-sensor

典型 Topics：
rtk/v1/home/main/aabbccddeeff/topology/connections
rtk/v1/home/living-room/778899aabbcc/evt/wifi.connection_lost
rtk/v1/home/kitchen/112233445566/telemetry/temperature
rtk/v1/home/bedroom/ddeeff001122/cmd/req
rtk/v1/home/group/security-sensors/cmd/req
```

#### 網路診斷與運維場景
```
租戶：network-ops
場域：datacenter-1, edge-sites, remote-branches
設備類型：router, switch, firewall, load-balancer, network-probe

典型 Topics：
rtk/v1/network-ops/datacenter-1/aabbccddeeff/diagnostics/speed_test
rtk/v1/network-ops/edge-sites/112233445566/telemetry/latency
rtk/v1/network-ops/remote-branches/778899aabbcc/evt/wan.connection_loss
rtk/v1/network-ops/datacenter-1/ddeeff001122/topology/discovery
rtk/v1/network-ops/edge-sites/group/border-routers/cmd/req
```

**場景特色**：
- **大規模監控**：支援數百台網路設備的即時監控
- **故障快速定位**：透過拓撲資訊快速識別故障節點和影響範圍
- **主動式診斷**：定期執行速度測試、連通性檢查、延遲測量
- **效能基線管理**：建立網路效能基線，偵測異常變化
- **分散式部署**：支援多地點、多層級的網路架構監控

### 3.6 最佳實踐指南

#### 命名規範最佳實踐
- **一致性**：在同一系統中保持命名風格一致
- **可讀性**：使用有意義的名稱，避免過度縮寫
- **擴展性**：預留未來擴展的命名空間

```bash
# 良好的命名範例
rtk/v1/factory/production-line-1/plc-siemens-001/state
rtk/v1/office/floor-2-east/ap-cisco-meeting-room/telemetry/wifi_clients

# 避免的命名範例  
rtk/v1/f/pl1/plc1/state                    # 過度縮寫
rtk/v1/factory/123/abc-def-xyz/state       # 無意義標識
```

#### 效能與可維護性考量
- **Topic 深度適中**：避免過深的層級結構影響效能
- **Retained 訊息管理**：定期清理不再需要的 retained 訊息
- **萬用字元謹慎使用**：避免訂閱過於寬泛的 topic 模式

#### 安全與權限控制
- **基於 Topic 的 ACL**：設計細緻的存取控制清單
- **租戶隔離**：確保不同租戶間的資料完全隔離
- **最小權限原則**：設備只訂閱必要的 topic

#### 版本演進策略
- **向下相容**：新版本協議需相容舊版設備
- **漸進升級**：提供 v1 到 v2 的平滑升級路徑
- **Schema 版本管理**：payload 層面的版本控制獨立於 topic 版本

---

## 4. Device ID 規範

* **`device_id`**：必須使用設備的 MAC 地址作為全域唯一標識符，不可變更
* **格式要求**：`^[a-f0-9]{12}$`（12位連續小寫十六進制字符，無冒號分隔）
* **MQTT `clientId`**：直接使用 `device_id`（即 MAC 地址的無冒號格式）
* **範例**：
  - MAC 地址：`aa:bb:cc:dd:ee:ff` → device_id：`aabbccddeeff`
  - MAC 地址：`00:11:22:33:44:55` → device_id：`001122334455`
* **優勢**：
  - 硬體層級的唯一性保證
  - 無需額外的 ID 管理機制
  - 便於設備識別和網路管理
  - 簡化設備註冊和認證流程

---

## 5. 設備上下線偵測

### 5.1 Last Will Testament (LWT) 機制

LWT 是 MQTT 的內建機制，用於偵測設備異常斷線。當設備與 MQTT Broker 的連線意外中斷時（如網路故障、設備當機），Broker 會自動發布預先設定的「遺囑訊息」通知其他訂閱者。

### 5.2 LWT 設定要求

設備在建立 MQTT 連線時必須設定以下 LWT 參數：

* **LWT Topic**: `rtk/v1/{tenant}/{site}/{device_id}/lwt`
* **範例**: `rtk/v1/office/floor1/aabbccddeeff/lwt`
* **LWT Payload**: `{"status":"offline","ts":<timestamp>}`
* **LWT Retained**: true

### 5.3 上下線狀態管理

**設備上線時**:
主動發布上線狀態到相同 topic：
```json
{
  "status": "online",
  "ts": "2024-08-13T08:00:00.000Z"
}
```

**設備正常下線時**:
發布離線狀態後再斷開連線：
```json
{
  "status": "offline", 
  "ts": "2024-08-13T08:01:40.000Z",
  "reason": "normal_shutdown"
}
```

**設備異常斷線時**:
MQTT Broker 自動發布預設的 LWT 訊息，讓 Controller 立即得知設備離線狀態。

---

## 6. 共通 Payload 格式

### 6.1 什麼是共通 Payload 格式

所有透過 MQTT 傳送的 JSON 訊息都應該包含一些共通欄位，這些欄位用於：
- **版本控制**: 確保不同版本的設備和系統能正確解析訊息
- **時間追蹤**: 記錄事件發生的精確時間
- **除錯追蹤**: 在複雜系統中追蹤訊息流向

無論是狀態回報、遙測資料、事件通知還是命令，都使用相同的基本結構。

### 6.2 必要欄位說明

#### `schema` (字串)
- **用途**: 標識訊息的類型和版本，讓接收端知道如何解析這個訊息
- **格式**: `<訊息類型>/<版本號>` 
- **範例**: 
  - `state/1.0` - 設備狀態訊息 v1.0
  - `evt.wifi.roam_miss/1.0` - WiFi 漫遊失效事件 v1.0
  - `cmd.diagnosis.get/1.0` - 診斷資料請求命令 v1.0

#### `ts` (字串)
- **用途**: 記錄訊息產生或事件發生的時間，用於排序、關聯和分析
- **格式**: ISO 8601 標準 GMT 時間格式 `YYYY-MM-DDTHH:mm:ss.sssZ`
- **範例**: `"2024-08-13T08:00:00.000Z"` (UTC 時間)
- **注意**: 
  - 必須使用 UTC 時區 (以 Z 結尾)
  - 包含毫秒精度 (.sss)
  - 所有設備必須使用相同的時間基準

### 6.3 選用欄位說明

#### `trace` (物件，Optional)
- **用途**: 在分散式系統中追蹤單一操作的完整流程
- **常用欄位**:
  - `req_id`: 請求唯一識別碼
  - `correlation_id`: 關聯多個相關訊息
  - `span_id`: 分散式追蹤的區段識別碼

### 6.4 欄位分類指南

為了快速理解 JSON payload 結構，以下提供欄位分類說明：

#### 欄位分類表格

| 欄位類型 | 欄位名稱 | 是否必要 | 資料型態 | 說明 |
|---------|---------|---------|---------|------|
| **共通欄位** | `schema` | 必要 | 字串 | 訊息類型與版本標識 |
| **共通欄位** | `ts` | 必要 | 字串 | ISO 8601 GMT 時間戳 |
| **共通欄位** | `trace` | 選用 | 物件 | 分散式追蹤資訊 |
| **業務欄位** | `payload` | 建議 | 物件 | 業務資料容器 |

#### 推薦的訊息結構

```json
{
  // === 必要共通欄位 ===
  "schema": "state/1.0",                    // 所有訊息必須包含
  "ts": "2024-08-13T08:00:00.000Z",        // 所有訊息必須包含
  
  // === 選用共通欄位 ===
  "trace": {                               // 需要追蹤時使用
    "req_id": "diag-001",
    "correlation_id": "session-12345"
  },
  
  // === 業務資料容器 ===
  "payload": {                             // 固定容器名稱
    "health": "ok",                        // 業務邏輯相關欄位
    "cpu_usage": 45.2,
    "wifi_stats": {
      "rssi": -52,
      "connected": true
    }
  }
}
```

#### trace 欄位詳細說明

**用途與應用場景**：
- **分散式追蹤**：在複雜系統中追蹤一個操作的完整流程
- **問題診斷**：當出現問題時，可以根據 trace 資訊找到相關的所有訊息
- **效能分析**：追蹤請求在不同組件間的傳遞時間

**實際應用場景**：
- 追蹤一個診斷命令從 Controller 發送到設備，再到設備回應的完整流程
- 關聯設備重啟前後的所有相關事件和狀態變化
- 分析網路漫遊事件中涉及的多個 AP 設備間的訊息流

**使用時機**：需要跨多個設備或長時間追蹤操作流程時使用

#### JSON Schema 用途說明

**在協議中的重要性**：
- **訊息驗證**：確保收到的 JSON 訊息格式正確，包含必要欄位
- **開發指導**：為開發者提供明確的資料結構定義和約束條件
- **自動化測試**：可用於自動驗證測試資料的正確性
- **API 文檔**：作為 API 介面的正式規格文檔

**實際應用場景**：
- MQTT Broker 接收訊息時驗證格式是否符合協議規範
- 設備端發送訊息前進行自我檢查
- Controller 端解析訊息時確保資料完整性
- 開發工具自動生成程式碼結構和驗證邏輯

**版本管理**：不同 schema 版本確保向下相容性和演進策略

#### 開發者檢查清單

使用此清單確保 JSON 訊息符合協議要求：

- [ ] 包含 `schema` 欄位，格式為 `{類型}/{版本}`
- [ ] 包含 `ts` 欄位，格式為 ISO 8601 GMT 時間
- [ ] 如需追蹤，正確設定 `trace` 欄位
- [ ] 業務資料放在 `payload` 容器內（建議）
- [ ] 遵循相應的 JSON Schema 定義
- [ ] 時間戳使用正確的 UTC 時區格式

### 6.5 完整訊息範例

**設備狀態訊息**（採用推薦結構）:
```json
{
  "schema": "state/1.0",
  "ts": "2024-08-13T08:00:00.000Z",
  "payload": {
    "health": "ok",
    "cpu_usage": 45.2,
    "wifi_stats": {
      "rssi": -52,
      "connected": true
    }
  }
}
```

**診斷事件訊息**（包含分散式追蹤）:
```json
{
  "schema": "evt.wifi.arp_loss/1.0", 
  "ts": "2024-08-13T08:00:01.000Z",
  "trace": {
    "req_id": "network-diag-001",
    "correlation_id": "network-issue-001"
  },
  "payload": {
    "severity": "warning",
    "arp_statistics": {
      "success_rate": 0.6
    }
  }
}
```

**命令訊息範例**:
```json
{
  "schema": "cmd.diagnosis.get/1.0",
  "ts": "2024-08-13T08:00:00.000Z",
  "trace": {
    "req_id": "cmd-12345"
  },
  "payload": {
    "id": "c-1001",
    "op": "diagnosis.get",
    "args": {
      "type": "wifi",
      "detail_level": "full"
    },
    "timeout_ms": 10000,
    "expect": "result"
  }
}
```

### 6.6 版本相容性規則

* **前向相容**: 實作必須忽略未知的 JSON 欄位
* **版本升級**: 
  - 小版本升級 (1.0 → 1.1): 新增欄位，不移除現有欄位
  - 大版本升級 (1.x → 2.0): 可能移除或改變現有欄位
* **解析原則**: 如果 `schema` 版本不相容，應記錄警告但不中斷處理

---

## 7. MQTT 使用時機

### 7.1 發布頻率
* **狀態資料** (`state`): 每 30-60 秒或狀態改變時
* **遙測資料** (`telemetry/*`): 每 10-60 秒，依重要性調整
* **事件資料** (`evt/*`): 立即發布
* **設備屬性** (`attr`): 啟動時或屬性變更時

### 7.2 訂閱模式
* Controller 使用通配符訂閱: `rtk/v1/+/+/+/state`、`rtk/v1/+/+/+/evt/#`
* Device 只訂閱自己的命令: `rtk/v1/{tenant}/{site}/{device_id}/cmd/req`

### 7.3 命令處理
* Device 收到 `cmd/req` 後立即回 `cmd/ack` (< 1 秒)
* 命令執行完成後發布 `cmd/res`，包含結果或錯誤

---

## 8. 上行結構定義（Device → Controller）

📋 **詳細範例**: 完整的 payload 格式和範例請參考 [EXAMPLES.md](EXAMPLES.md)

### 8.1 `state`（retained）

**Topic**
```
rtk/v1/{tenant}/{site}/{device_id}/state
```

**主要欄位**
* `health`: `ok|warn|error`，設備整體健康狀態
* `cpu_usage`: CPU 使用率百分比（0-100）
* `memory_usage`: 記憶體使用率百分比（0-100）
* `disk_usage`: 磁碟使用率百分比（0-100）
* `temperature_c`: 設備溫度（攝氏度）
* `net`: 網路介面資訊與統計
* `diagnosis`: 診斷相關資訊，包含錯誤狀態與統計

### 8.2 `telemetry/{metric}`

**Topic**
```
rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}
```

**常見 metric 類型:**
* **硬體診斷**: `temperature`, `cpu_usage`, `memory_usage`, `disk_usage`, `fan_speed`
* **網路診斷**: `interface.eth0.rx_bytes`, `interface.eth0.tx_bytes`, `ping_latency`, `bandwidth_usage`
* **WiFi 診斷**: `wifi.scan_result`, `wifi_clients`, `wifi.connection_quality`, `wifi.rssi`, `wifi.channel_utilization`
* **拓撲相關**: `connected_devices`, `arp_table`, `dhcp_leases`, `bridge_table`, `routing_table`
* **應用診斷**: `response_time`, `error_rate`, `queue_depth`, `connection_count`
* **IoT 特定**: `battery_voltage`, `humidity`, `power_consumption`, `signal_strength`

高頻診斷資料應分 metric 發布以降低傳輸成本。

### 8.3 `evt/{event_type}`（事件/告警）

**Topic**
```
rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}
```

**常見 event_type 類型:**
* **系統診斷**: `system.error`, `system.warning`, `system.recovery`
* **硬體事件**: `hardware.fault`, `hardware.overheat`, `power.failure`
* **網路事件**: `network.disconnected`, `network.latency_high`, `interface.down`
* **WiFi 事件**: `wifi.roam_triggered`, `wifi.scan_failed`, `wifi.signal_weak`, `wifi.ap_changed`
* **應用事件**: `service.crashed`, `memory.low`, `disk.full`

### 8.4 `attr`（retained，裝置屬性）

**Topic**
```
rtk/v1/{tenant}/{site}/{device_id}/attr
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

## 9. 下行命令（Controller → Device）

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
  "ts": "2024-08-13T08:00:00.000Z"
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
  "ts": "2024-08-13T08:00:01.000Z",
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
  "ts": "2024-08-13T08:00:03.000Z",
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
  "ts": "2024-08-13T08:00:03.000Z",
  "ok": false,
  "result": null,
  "err": { "code": "E_UNSUPPORTED", "msg": "capability not available" }
}
```

* 長任務可在處理期間定期發布含 `progress` 的 `res`（0\~100 或階段字串），最後再送最終 `res`。

---

## 10. 典型命令清單（建議命名）

| `op`                          | 說明            | `args` 範例                                                       | 結果 `result` 範例                                     |
| ----------------------------- | ------------- | --------------------------------------------------------------- | -------------------------------------------------- |
| `light.set`                   | 設定燈狀態         | `{ "on": true, "brightness": 80, "color": "#ffaa00" }`          | `{ "on": true, "brightness": 80 }`                 |
| `device.reboot`               | 重新啟動          | `{}`                                                            | `{ "uptime_s": 0 }`                                |
| `report.push`                 | 立即回報特定資料      | `{ "what": ["state", "telemetry.temperature"] }`                | `{ "pushed": ["state", "telemetry.temperature"] }` |
| `diagnosis.get`               | 取得診斷資料        | `{ "type": "wifi", "detail_level": "full" }`                    | 設備相依的診斷資料結構                                        |
| `fw.update`                   | 韌體更新          | `{ "version": "1.2.4", "url": "https://...", "sha256": "..." }` | `{ "phase": "done", "version": "1.2.4" }`          |
| `net.wifi.config`             | 設定 Wi‑Fi      | `{ "ssid": "x", "psk": "y" }`                                   | `{ "connected": true, "ip": "..." }`               |
| `topology.discover`           | 拓撲發現          | `{ "discovery_type": "full", "include_inactive": false }`       | 網路拓撲資訊結構                                           |
| `topology.query_interfaces`   | 查詢網路介面        | `{ "interface_filter": ["eth*", "wlan*"], "include_routing": true }` | 介面詳細資訊                                             |
| `topology.trace_path`         | 路徑追蹤          | `{ "target_ip": "192.168.1.100", "measure_latency": true }`     | 網路路徑資訊                                             |
| `topology.query_dhcp_leases`  | 查詢 DHCP 租約   | `{ "interface": "br0", "include_expired": false }`              | DHCP 租約清單                                          |
| `topology.query_bridge_table` | 查詢橋接表         | `{ "bridge_name": "br0", "include_aging_info": true }`          | 橋接表資訊                                              |
| `identity.set`                | 設置設備標識        | `{ "mac_address": "aa:bb:cc:dd:ee:ff", "friendly_name": "客廳冷氣" }` | `{ "updated": true }`                              |
| `identity.get`                | 查詢設備標識        | `{ "mac_address": "aa:bb:cc:dd:ee:ff" }`                        | 設備標識資訊                                             |

> 命令的實際清單由各產品線維護，並以 `schema` 版本化。

---

## 11. 診斷資料傳輸機制

### 11.1 主動診斷事件傳輸
設備在檢測到異常狀況時，會主動透過 `evt/{event_type}` 發送診斷事件，包含：
- 觸發條件與嚴重程度
- 初步診斷資訊與環境參數  
- 建議後續動作

### 11.2 被動詳細診斷請求  
Controller 可透過 `diagnosis.get` 命令主動請求設備回報特定類型的詳細診斷資料。設備收到診斷命令後，應立即透過 `cmd/res` 回傳當前的完整診斷狀態。

### 11.3 診斷命令格式

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
  "ts": "2024-08-13T08:00:00.000Z"
}
```

**Command Response (範例 - WiFi 設備):**
```json
{
  "id": "diag-001",
  "ts": "2024-08-13T08:00:01.000Z",
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

### 11.4 設備相依性
* **診斷資料內容**: 每種設備類型的診斷資料結構完全不同
* **支援的診斷類型**: 各設備根據硬體能力支援不同的 `type` 參數
* **回應時間**: 複雜診斷可能需要較長處理時間，建議設定適當的 `timeout_ms`

### 11.5 常見診斷類型
* `wifi`: WiFi 連線狀態、掃描結果、漫遊記錄
* `network`: 網路介面統計、路由表、連線狀態
* `system`: CPU、記憶體、磁碟使用狀況
* `hardware`: 硬體感測器資料、溫度、風扇轉速
* `application`: 應用程式狀態、服務運行情況

---

## 12. 錯誤碼建議

| 代碼               | 說明              |
| ---------------- | --------------- |
| `E_TIMEOUT`      | 命令處理逾時          |
| `E_UNSUPPORTED`  | 裝置不支援該 `op` 或參數 |
| `E_BUSY`         | 裝置忙碌，無法處理       |
| `E_INVALID_ARGS` | 參數格式錯誤          |
| `E_FORBIDDEN`    | 權限不足或 ACL 拒絕    |
| `E_INTERNAL`     | 內部錯誤            |

---

## 13. 版本控管

* **Topic 版本**：`rtk/v1/...`；破壞式變更升 `v2`。
* **Schema 版本**：每種 payload `schema` 採語意化版本（SemVer）。
* **相容原則**：

  * 裝置/後端對未知欄位需忽略。
  * 新增欄位為不破壞性；移除/改義需升大版並逐步淘汰。

---

## 14. 順序、重送與冪等

* **命令去重**：裝置應以 `id` 做去重，對重複 `id` 只執行一次，重傳先回覆既有結果。
* **超時與重試**：Controller 端可在 `timeout_ms` 到期後重試；重試必須沿用同一個 `id`。
* **送達順序**：MQTT 僅保證同一連線與同一 topic 的消息順序；跨 topic 需以 `ts/seq` 校正。

---

## 15. 監控與審計建議

* 監控訂閱：`rtk/v1/+/+/+/cmd/#`、`rtk/v1/+/+/+/evt/#`、`rtk/v1/+/+/+/lwt`。
* 寫入審計：所有 `cmd` 的 `req/ack/res` 需落庫（含 `tenant/site/device_id/id/op/ts`）。
* 異常告警：

  * 同一 `device_id` 多重連線（疑似複製）。
  * 過久未更新 `state` 或頻繁 `offline`。

---

## 16. 測試案例（最低集合）

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

---

## 17. JSON Schema（簡化示例）

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

## 18. 實作指南（裝置端）

1. 以 `clientId=device_id` 連線 Broker，設置 LWT `offline`（retained）。
2. 上線後發布 `lwt=online` 與最新 `attr/state`。
3. 訂閱 `cmd/req`（以及群組 req，如有）。
4. 收到命令：

   * 解析 payload → 立刻回 `ack`（含 `id`）。
   * 執行命令；長任務定期回 `res`（含 `progress`）。
   * 完成後回最終 `res`（`ok=true/false`）。
5. 週期性發布 `state` 與必要 `telemetry`，異常以 `evt/*` 通報。

---

## 19. 實作指南（Controller/後端）

1. 封裝 `sendCommand(op, args, device_id, timeout)`：

   * 產生 `id` → 發布 `cmd/req`。
   * 等待 `ack`，超時重試（同 `id`）。
   * 視 `expect` 等待 `res`，處理超時與最終態。
2. 日誌/審計：完整記錄 `req/ack/res` 與錯誤碼。
3. 指標/告警：命令成功率、延遲分佈、裝置上線率、事件頻度等。

---

## 20. 範例訂閱樣式

* 全租戶所有命令流：`rtk/v1/+/+/+/cmd/#`
* 某站台所有裝置狀態：`rtk/v1/{tenant}/{site}/+/state`
* 某裝置所有遙測：`rtk/v1/{tenant}/{site}/{device_id}/telemetry/#`

---

## 21. WiFi 診斷實際應用範例

📋 **詳細範例**: 完整的診斷場景與 payload 範例請參考 [EXAMPLES.md](EXAMPLES.md)

### 23.1 漫遊問題診斷範例

#### 情境描述
辦公室 AP `office-ap-001` 檢測到 RSSI 降至 -75dBm 持續 10 秒，但未觸發漫遊機制。

#### 完整 MQTT 流程
典型的診斷流程包含：
1. 事件觸發 - 設備發送診斷事件
2. Controller 請求詳細診斷
3. Device 回傳詳細診斷結果

詳細的 payload 格式請參考 [EXAMPLES.md](EXAMPLES.md)。


### 23.2 連線失敗診斷範例

#### 情境描述
筆記型電腦 `laptop-005` 嘗試連線企業 WiFi 時在 WPA3 SAE 認證階段失敗。

典型的連線失敗診斷包含認證超時、關聯失敗、密鑰安裝錯誤等不同階段的問題。詳細 payload 範例請參考 [EXAMPLES.md](EXAMPLES.md)。

### 23.3 ARP 遺失診斷範例

#### 情境描述
智能攝影機 `smart-camera-003` 偵測到連續 2 次 ARP response 未收到。

ARP 遺失診斷包含網路層連通性問題分析，涵蓋射頻干擾、AP 負載、網路壅塞等根因分析。

### 23.4 時序圖與通訊流程

#### 典型診斷流程
標準的診斷流程遵循：事件觸發 → 詳細診斷請求 → 診斷結果回傳 → 根因分析

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

主要的診斷資料結構包含：
- WiFi 漫遊候選 AP 結構  
- 連線失敗分析結構
- ARP 統計結構
- RF 診斷結構

詳細的結構定義請參考 [EXAMPLES.md](EXAMPLES.md)。

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

## 22. 網路拓撲檢測與設備標識管理

📋 **詳細範例**: 拓撲檢測相關的 payload 範例請參考 [EXAMPLES.md](EXAMPLES.md)

### 23.1 概述

RTK Controller 提供完整的網路拓撲檢測系統，透過 MQTT 訊息收集網路設備拓撲資訊、建立完整的網路連接關係圖、提供設備友好名稱管理功能，以及實現主動式網路診斷與 QoS 分析。

#### 主要功能特色
- **智能拓撲發現**: 自動檢測路由器、AP、交換機、橋接器等複雜網路設備
- **設備身份管理**: MAC 地址到友好名稱轉換（如 "aa:bb:cc:dd:ee:ff" → "客廳冷氣"）
- **Roaming 檢測**: Controller 側智能推斷 WiFi 漫遊事件
- **時間軸分析**: 追蹤網路拓撲變化歷史和設備移動軌跡
- **主動式診斷**: 速度測試、WAN 連線檢測、延遲測量
- **QoS 監控**: 即時流量分析、頻寬使用監控、高流量設備識別

### 23.2 網路拓撲發現機制

#### 拓撲發現訊息

**Topic**: `rtk/v1/{tenant}/{site}/{device_id}/topology/discovery`

拓撲發現訊息包含設備基本資訊、網路介面、路由資訊等完整的網路配置資料。

#### 設備連接報告

**Topic**: `rtk/v1/{tenant}/{site}/{device_id}/topology/connections`

設備連接報告包含已發現的設備清單、ARP 表、DHCP 租約等連接狀態資訊。

#### WiFi 客戶端狀態報告

**Topic**: `rtk/v1/{tenant}/{site}/{device_id}/telemetry/wifi_clients`

WiFi 客戶端狀態報告包含 AP 資訊和已連接客戶端的詳細狀態。

### 23.3 WiFi Roaming 檢測

**重要架構說明**: 由於 Controller 與每個 AP 設備獨立通訊，roaming 檢測需要由 Controller 側分析跨 AP 的客戶端移動來推斷。

#### Roaming 檢測演算法
1. **收集階段**: Controller 收集各 AP 的 `telemetry/wifi_clients` 訊息
2. **相關性分析**: 識別相同 MAC 地址在不同 BSSID 的出現時間
3. **事件推斷**: 當客戶端在 AP-A 斷開後短時間內在 AP-B 出現，推斷為 roaming 事件
4. **品質評估**: 根據訊號強度變化、時間間隔判斷 roaming 品質

#### 典型 Roaming 場景
- **成功 Roaming**: 客戶端平滑從弱訊號 AP 切換到強訊號 AP
- **延遲 Roaming**: 客戶端延遲切換，造成連線中斷
- **頻繁 Roaming**: 客戶端在多個 AP 間頻繁切換，影響連線穩定性

### 23.4 設備標識管理

RTK Controller 提供設備友好名稱管理功能，支援：

1. **手動設置**: 透過 `identity.set` 命令設置設備名稱
2. **自動檢測**: 基於 MAC OUI、hostname 模式、DHCP vendor 等規則自動識別設備類型
3. **批量管理**: 支援 CSV 檔案批量導入設備標識

#### 設備標識命令範例

```json
{
  "id": "identity-set-001",
  "op": "identity.set",
  "schema": "cmd.identity.set/1.0",
  "args": {
    "mac_address": "aa:bb:cc:dd:ee:ff",
    "friendly_name": "客廳冷氣",
    "device_type": "air_conditioner",
    "location": "living_room",
    "owner": "Kevin"
  },
  "timeout_ms": 5000,
  "expect": "result"
}
```

### 23.5 多介面設備支援

協議完整支援路由器、交換機、橋接器等多介面設備，包含：

- **多介面管理**: 每個介面獨立的 MAC、IP、狀態資訊
- **橋接支援**: 橋接表、VLAN、STP 等 Layer 2 功能
- **路由支援**: 路由表、NAT 規則、轉發邏輯
- **DHCP 服務**: DHCP 伺服器設定與租約管理

### 23.6 主動式網路診斷

RTK Controller 支援主動式網路測試，透過設備執行各種診斷測試，評估網路健康狀態。

#### 網路診斷訊息

**Topic**: `rtk/v1/{tenant}/{site}/{device_id}/diagnostics/network`

```json
{
  "schema": "diagnostics.network/1.0",
  "timestamp": 1723526400000,
  "device_id": "router-main-001",
  "speed_test": {
    "download_mbps": 425.6,
    "upload_mbps": 87.3,
    "jitter": 2.1,
    "packet_loss": 0.0,
    "test_server": "speedtest.hinet.net",
    "status": "completed"
  },
  "wan_test": {
    "isp_gateway_reachable": true,
    "isp_gateway_latency": 8.5,
    "external_dns_latency": 12.3,
    "wan_connected": true,
    "public_ip": "203.66.xxx.xxx"
  },
  "connectivity_test": {
    "internal_reachability": [
      {
        "device_id": "ap-living-room-001",
        "ip_address": "192.168.1.10",
        "reachable": true,
        "latency": 1.2,
        "method": "ping"
      }
    ]
  }
}
```

### 23.7 QoS 與流量分析

提供即時流量監控與 QoS 配置檢測，幫助識別網路瓶頸和優化建議。

#### QoS 資訊訊息

**Topic**: `rtk/v1/{tenant}/{site}/{device_id}/telemetry/qos`

```json
{
  "schema": "telemetry.qos/1.0",
  "timestamp": 1723526400000,
  "device_id": "router-main-001",
  "enabled": true,
  "traffic_stats": {
    "total_bandwidth": 500.0,
    "used_bandwidth": 156.8,
    "device_traffic": [
      {
        "device_mac": "11:22:33:44:55:66",
        "friendly_name": "Kevin的iPhone",
        "upload_mbps": 12.5,
        "download_mbps": 45.2,
        "bandwidth_percent": 11.5
      }
    ],
    "top_talkers": [
      {
        "device_id": "kevin-laptop",
        "friendly_name": "Kevin的MacBook",
        "total_mbps": 78.9,
        "traffic_type": "download",
        "rank": 1
      }
    ]
  }
}
```

### 23.8 時間軸拓撲分析

Controller 提供歷史拓撲變化追蹤，支援時間區間查詢和異常檢測。

#### 支援功能
- **拓撲變化歷史**: 追蹤設備接入/離開時間
- **Roaming 軌跡**: 分析設備在不同 AP 間的移動路徑  
- **連線品質趨勢**: 監控訊號強度、速度變化
- **異常行為檢測**: 識別頻繁漫遊、連線不穩定等問題

### 23.9 拓撲查詢與診斷命令

Controller 提供豐富的拓撲查詢和診斷命令：

#### 基礎拓撲命令
- `topology.discover`: 完整拓撲發現
- `topology.query_interfaces`: 網路介面詳細查詢
- `topology.trace_path`: 網路路徑追蹤
- `topology.query_dhcp_leases`: DHCP 租約查詢
- `topology.query_bridge_table`: 橋接表查詢

#### 設備身份管理命令
- `identity.set`: 設置設備友好名稱
- `identity.get`: 查詢設備身份資訊
- `identity.auto_detect`: 執行自動檢測
- `identity.import`: 批量導入設備身份

#### 診斷測試命令
- `diagnostics.speed_test`: 執行速度測試
- `diagnostics.wan_test`: WAN 連線測試
- `diagnostics.latency_test`: 延遲測試
- `diagnostics.connectivity_test`: 連通性測試

#### 歷史查詢命令
- `topology.history`: 查詢拓撲變化歷史
- `topology.roaming_history`: 查詢 roaming 事件歷史
- `topology.anomalies`: 檢測網路異常行為

這些功能讓 Controller 能夠提供完整的網路拓撲可視化、診斷和歷史分析能力。

---


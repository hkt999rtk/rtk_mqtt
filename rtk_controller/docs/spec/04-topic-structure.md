# Topic 命名空間完整說明

## 基本主題結構

### 標準格式
```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}
```

### 路徑組成詳解

| 組件 | 說明 | 格式規範 | 範例 |
|------|------|----------|------|
| `rtk/v1` | 協議命名空間與版本 | 固定值 | `rtk/v1` |
| `{tenant}` | 租戶標識，用於多租戶環境資料隔離 | `^[a-z0-9][a-z0-9-]{1,30}[a-z0-9]$` | `office`, `factory`, `home` |
| `{site}` | 場域標識，同租戶下的物理位置區分 | `^[a-z0-9][a-z0-9-]{1,30}[a-z0-9]$` | `floor1`, `workshop-a` |
| `{device_id}` | 設備唯一標識符 | `^[a-f0-9]{12}$` (MAC地址) | `aabbccddeeff` |
| `{message_type}` | 訊息類型，支援多層級結構 | 詳見下方分類 | `state`, `telemetry/cpu` |

### 實際範例
```
rtk/v1/office/floor1/aabbccddeeff/state
rtk/v1/factory/workshop-a/112233445566/telemetry/temperature  
rtk/v1/home/living-room/778899aabbcc/evt/wifi.connection_lost
rtk/v1/corporate/building-a/ddeeff001122/cmd/req
```

## 訊息類型分類

### 狀態與屬性類
| 類型 | 用途 | Retained | 範例 Topic |
|------|------|----------|------------|
| `state` | 設備狀態摘要與診斷資訊 | ✅ | `rtk/v1/office/floor1/aabbccddeeff/state` |
| `attr` | 設備靜態屬性與規格資訊 | ✅ | `rtk/v1/factory/line-a/445566778899/attr` |
| `lwt` | Last Will Testament 上下線狀態 | ✅ | `rtk/v1/home/main/112233445566/lwt` |

### 遙測資料類
```
telemetry/{metric}
```

**常見 Metric 類型**:
- `temperature` - 溫度
- `cpu_usage` - CPU 使用率
- `wifi_clients` - WiFi 客戶端數量
- `bandwidth_usage` - 頻寬使用量
- `signal_strength` - 信號強度

**範例**:
```
rtk/v1/home/kitchen/112233445566/telemetry/temperature
rtk/v1/office/floor2/778899aabbcc/telemetry/wifi_clients
rtk/v1/factory/workshop/ddeeff001122/telemetry/vibration
```

### 事件與告警類
```
evt/{event_type}
```

**事件分類**:
- `wifi.*` - WiFi 相關事件
- `system.*` - 系統事件
- `hardware.*` - 硬體事件
- `network.*` - 網路事件

**範例**:
```
rtk/v1/home/living-room/445566778899/evt/wifi.connection_lost
rtk/v1/office/floor1/aabbccddeeff/evt/system.low_memory
rtk/v1/factory/line-b/112233445566/evt/hardware.overheat
```

### 命令執行類
| 類型 | 用途 | QoS | 範例 Topic |
|------|------|-----|------------|
| `cmd/req` | 命令請求 | 1 | `rtk/v1/office/floor1/aabbccddeeff/cmd/req` |
| `cmd/ack` | 命令確認 | 1 | `rtk/v1/office/floor1/aabbccddeeff/cmd/ack` |
| `cmd/res` | 命令結果 | 1 | `rtk/v1/office/floor1/aabbccddeeff/cmd/res` |

### 拓撲與診斷類
```
topology/discovery     # 網路拓撲發現
topology/connections   # 設備連接狀態
diagnostics/{test_type}  # 診斷測試
```

## 上行通信 (Device → Controller)

### 狀態報告
設備定期或狀態變更時發送的健康狀態摘要。

**使用頻率**: 每 30-60 秒或狀態改變時  
**Topic 格式**: `rtk/v1/{tenant}/{site}/{device_id}/state`  
**QoS**: 1, Retained: true

### 遙測數據
設備定期發送的監控數據和效能指標。

**使用頻率**: 每 10-60 秒，依重要性調整  
**Topic 格式**: `rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}`  
**QoS**: 0-1, Retained: false

### 事件通知  
設備發生重要事件時的即時通知。

**使用時機**: 事件發生時立即發布  
**Topic 格式**: `rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}`  
**QoS**: 1, Retained: false

### 命令回應
設備對控制器命令的確認和結果回覆。

**ACK 回應**: 收到命令後 < 1 秒內確認  
**RES 回應**: 命令執行完成後發布結果  

## 下行通信 (Controller → Device)

### 命令請求
控制器向設備發送的操作指令和配置變更。

**Topic 格式**: `rtk/v1/{tenant}/{site}/{device_id}/cmd/req`  
**QoS**: 1, Retained: false

**常見命令類型**:
- **設備控制**: `device.reboot`, `light.set`, `net.wifi.config`
- **診斷請求**: `diagnosis.get`, `topology.discover`  
- **配置管理**: `fw.update`, `identity.set`
- **測試執行**: `diagnostics.speed_test`, `diagnostics.wan_test`

## 群組與廣播 (可選)

### 群組命令
同時控制多個相關設備的命令。

**Topic 格式**: `rtk/v1/{tenant}/{site}/group/{group_id}/cmd/req`

**使用場景**:
```
rtk/v1/office/floor1/group/meeting-room-lights/cmd/req    # 會議室照明
rtk/v1/factory/production/group/emergency-stops/cmd/req  # 緊急停止系統
rtk/v1/hospital/icu/group/air-purifiers/cmd/req          # 空氣清淨設備
```

### 廣播命令
租戶或全域廣播命令。

**Topic 格式**:
```
rtk/v1/{tenant}/broadcast/cmd/req  # 租戶廣播
rtk/v1/broadcast/cmd/req           # 全域廣播
```

## 訂閱模式指南

### Controller 監控訂閱

#### 全域監控
```bash
rtk/v1/+/+/+/evt/#              # 所有事件
rtk/v1/+/+/+/lwt                # 所有設備上下線
rtk/v1/+/+/+/state              # 所有設備狀態
```

#### 分類監控  
```bash
rtk/v1/+/+/+/evt/wifi.#         # WiFi 相關事件
rtk/v1/+/+/+/evt/system.#       # 系統事件
rtk/v1/+/+/+/topology/#         # 拓撲訊息
```

#### 租戶/場域監控
```bash
rtk/v1/office/+/+/evt/#         # 辦公室所有事件
rtk/v1/factory/+/+/state        # 工廠所有設備狀態
rtk/v1/office/floor1/+/evt/#    # 1樓所有事件
```

### 設備端訂閱

#### 基本訂閱 (必須)
```bash
rtk/v1/{tenant}/{site}/{device_id}/cmd/req
```

#### 群組訂閱 (可選)
```bash
rtk/v1/{tenant}/{site}/group/{group_id}/cmd/req
```

#### 廣播訂閱 (可選)
```bash
rtk/v1/{tenant}/broadcast/cmd/req
rtk/v1/broadcast/cmd/req
```

## 萬用字元使用規則

### 單層級萬用字元 (`+`)
匹配單一層級的任意內容，不能跨越 `/` 分隔符。

**範例**:
```bash
rtk/v1/+/floor1/+/state         # 任意租戶的1樓任意設備狀態
rtk/v1/office/+/aabbccddeeff/+  # 辦公室特定設備的所有訊息類型
```

### 多層級萬用字元 (`#`)
匹配多層級的任意內容，只能在 topic 末尾使用。

**範例**:
```bash
rtk/v1/office/+/+/evt/#         # 辦公室所有設備的所有事件
rtk/v1/+/+/+/telemetry/#        # 所有遙測數據
```

## 最佳實踐指南

### 良好的命名範例
```bash
# 清楚的租戶和場域標識
rtk/v1/smart-office/floor-3/aabbccddeeff/state
rtk/v1/manufacturing/assembly-line-1/112233445566/telemetry/vibration

# 語義化的事件名稱  
rtk/v1/home/living-room/778899aabbcc/evt/wifi.roam_completed
rtk/v1/datacenter/rack-a1/ddeeff001122/evt/hardware.temperature_warning
```

### 應避免的命名範例
```bash
# 過於簡化或不明確
rtk/v1/t1/s1/dev1/state
rtk/v1/a/b/c/evt/e1

# 不符合格式規範
rtk/v1/Office/Floor_1/AA:BB:CC:DD:EE:FF/state  # 大小寫和冒號
rtk/v1/very-very-very-long-tenant-name-exceeds-limit/site/device/state  # 超長名稱
```

### 效能優化建議

1. **避免過寬的訂閱**
   ```bash
   # ❌ 避免
   rtk/v1/+/+/+/telemetry/#  # 可能產生大量流量
   
   # ✅ 建議  
   rtk/v1/office/+/+/telemetry/temperature  # 具體的metric
   ```

2. **使用精確的事件類型**
   ```bash
   # ❌ 過於寬泛
   rtk/v1/+/+/+/evt/#
   
   # ✅ 精確匹配
   rtk/v1/+/+/+/evt/wifi.roam_triggered
   ```

3. **分層訂閱策略**
   - 先訂閱關鍵事件和狀態
   - 根據實際需求逐步擴展監控範圍
   - 定期檢查和優化訂閱模式

## 命名規範摘要

### 格式要求
- **tenant/site**: 小寫字母、數字、連字號，長度 3-32 字符
- **device_id**: 小寫十六進制，12 字符 MAC 地址格式
- **message_type**: 小寫字母、數字、點號、斜線

### 字符限制
- 允許: `a-z`, `0-9`, `-`, `.`, `/`
- 禁止: 大寫字母、底線、冒號、特殊字符
- 總長度: 建議不超過 200 字符

### 語義化要求
- 使用有意義的業務標識
- 避免過於抽象的縮寫
- 保持命名的一致性和可讀性

---

**下一步**: 閱讀 [設備管理](05-device-management.md) 了解 Device ID 規範和生命週期管理。
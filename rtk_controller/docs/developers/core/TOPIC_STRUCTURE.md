# RTK MQTT 主題結構詳解

## 概述

本文檔詳細說明RTK MQTT協議的主題(Topic)結構設計原則、命名規範和使用指南。

## 基本結構

### 標準格式
```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}[/{sub_type}]
```

### 組件說明

#### rtk
- **用途**: 協議命名空間標識
- **固定值**: "rtk"
- **說明**: 用於區分RTK協議消息與其他MQTT消息

#### v1
- **用途**: 協議版本標識
- **當前值**: "v1"
- **說明**: 支持協議版本演進，未來可能有v2, v3等

#### tenant
- **用途**: 租戶/組織標識
- **格式**: 字母數字和下劃線，長度3-32字符
- **示例**: 
  - `company_a` - 企業租戶
  - `user_123` - 個人用戶
  - `demo` - 演示環境

#### site
- **用途**: 站點/位置標識
- **格式**: 字母數字和下劃線，長度3-32字符
- **示例**:
  - `office_main` - 主辦公室
  - `home_network` - 家庭網絡
  - `branch_01` - 分支機構

#### device_id
- **用途**: 設備唯一標識符
- **格式**: 通常為MAC地址，也可以是其他唯一標識
- **示例**:
  - `aabbccddeeff` - MAC地址格式
  - `device_001` - 自定義標識
  - `router-main` - 描述性標識

#### message_type
- **用途**: 消息類型標識
- **支持類型**:
  - `state` - 設備狀態
  - `telemetry` - 遙測數據
  - `evt` - 事件通知
  - `attr` - 設備屬性
  - `cmd` - 命令相關
  - `lwt` - 遺言消息
  - `topology` - 拓撲信息

#### sub_type (可選)
- **用途**: 消息子類型
- **示例**:
  - `telemetry/cpu` - CPU遙測
  - `evt/wifi` - WiFi事件
  - `cmd/req` - 命令請求

## 詳細主題分類

### 狀態消息 (state)

#### 基本狀態
```
rtk/v1/{tenant}/{site}/{device_id}/state
```

**用途**: 設備健康狀態和基本信息
**QoS**: 1 (至少一次傳遞)
**Retained**: true (保留消息)
**發送頻率**: 每5分鐘或狀態變化時

**消息示例**:
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "health": "ok",
  "uptime_s": 86400,
  "cpu_usage": 25.4,
  "memory_usage": 45.2,
  "connection_status": "connected"
}
```

### 遙測數據 (telemetry)

#### CPU遙測
```
rtk/v1/{tenant}/{site}/{device_id}/telemetry/cpu
```

#### 內存遙測  
```
rtk/v1/{tenant}/{site}/{device_id}/telemetry/memory
```

#### 網絡遙測
```
rtk/v1/{tenant}/{site}/{device_id}/telemetry/network
```

#### WiFi遙測
```
rtk/v1/{tenant}/{site}/{device_id}/telemetry/wifi
```

**特性**:
- **QoS**: 0-1 (根據重要性)
- **Retained**: false
- **發送頻率**: 30秒-5分鐘間隔

### 事件通知 (evt)

#### WiFi事件
```
rtk/v1/{tenant}/{site}/{device_id}/evt/wifi
rtk/v1/{tenant}/{site}/{device_id}/evt/wifi/connection_lost
rtk/v1/{tenant}/{site}/{device_id}/evt/wifi/roam_triggered
```

#### 網絡事件
```
rtk/v1/{tenant}/{site}/{device_id}/evt/network
rtk/v1/{tenant}/{site}/{device_id}/evt/network/link_up
rtk/v1/{tenant}/{site}/{device_id}/evt/network/link_down
```

#### 系統事件
```
rtk/v1/{tenant}/{site}/{device_id}/evt/system
rtk/v1/{tenant}/{site}/{device_id}/evt/system/error
rtk/v1/{tenant}/{site}/{device_id}/evt/system/warning
```

**特性**:
- **QoS**: 1 (確保傳遞)
- **Retained**: false
- **觸發**: 事件發生時立即發送

### 設備屬性 (attr)

#### 基本屬性
```
rtk/v1/{tenant}/{site}/{device_id}/attr
```

**用途**: 設備靜態屬性和配置信息
**內容示例**:
```json
{
  "schema": "attr/1.0",
  "ts": 1699123456789,
  "device_type": "router",
  "manufacturer": "Example Corp",
  "model": "EX-R1000",
  "firmware_version": "1.2.3",
  "hardware_version": "A1",
  "serial_number": "ABC123456789",
  "capabilities": ["wifi", "ethernet", "mesh"]
}
```

### 命令相關 (cmd)

#### 命令請求
```
rtk/v1/{tenant}/{site}/{device_id}/cmd/req
```

#### 命令確認
```
rtk/v1/{tenant}/{site}/{device_id}/cmd/ack
```

#### 命令響應
```
rtk/v1/{tenant}/{site}/{device_id}/cmd/res
```

**流程**:
1. Controller → Device: 發送命令到 `/cmd/req`
2. Device → Controller: 發送確認到 `/cmd/ack` 
3. Device → Controller: 發送結果到 `/cmd/res`

### 遺言消息 (lwt)

#### 設備遺言
```
rtk/v1/{tenant}/{site}/{device_id}/lwt
```

**用途**: 設備斷線檢測
**內容**:
```json
{
  "schema": "lwt/1.0",
  "ts": 1699123456789,
  "status": "offline",
  "last_seen": 1699123456789
}
```

### 拓撲信息 (topology)

#### 拓撲更新
```
rtk/v1/{tenant}/{site}/{device_id}/topology/update
```

#### 鄰居發現
```
rtk/v1/{tenant}/{site}/{device_id}/topology/neighbors
```

## 主題權限設計

### 讀權限 (Subscribe)

#### 控制器權限
```
rtk/v1/+/+/+/state
rtk/v1/+/+/+/telemetry/+
rtk/v1/+/+/+/evt/+
rtk/v1/+/+/+/cmd/ack
rtk/v1/+/+/+/cmd/res
rtk/v1/+/+/+/lwt
rtk/v1/+/+/+/topology/+
```

#### 設備權限 (自己的消息)
```
rtk/v1/{tenant}/{site}/{device_id}/cmd/req
```

### 寫權限 (Publish)

#### 控制器權限
```
rtk/v1/+/+/+/cmd/req
```

#### 設備權限
```
rtk/v1/{tenant}/{site}/{device_id}/state
rtk/v1/{tenant}/{site}/{device_id}/telemetry/+
rtk/v1/{tenant}/{site}/{device_id}/evt/+
rtk/v1/{tenant}/{site}/{device_id}/attr
rtk/v1/{tenant}/{site}/{device_id}/cmd/ack
rtk/v1/{tenant}/{site}/{device_id}/cmd/res
rtk/v1/{tenant}/{site}/{device_id}/topology/+
```

## 通配符使用

### 單層通配符 (+)
```
rtk/v1/company_a/+/router-01/state  # 匹配company_a租戶下所有站點的router-01狀態
rtk/v1/+/office/+/telemetry/cpu     # 匹配所有租戶office站點的CPU遙測
```

### 多層通配符 (#)
```
rtk/v1/company_a/office/router-01/# # 匹配router-01的所有消息
rtk/v1/+/+/+/evt/#                  # 匹配所有設備的所有事件
```

## 主題命名最佳實踐

### 命名規範
- 使用小寫字母
- 用下劃線分隔單詞
- 避免特殊字符 (除了: - _ .)
- 保持簡潔但有意義

### 好的示例
```
rtk/v1/acme_corp/hq_office/ap_lobby/state
rtk/v1/home_user/living_room/smart_tv/evt/network
rtk/v1/demo/test_lab/sensor_01/telemetry/temperature
```

### 避免的示例
```
rtk/v1/ACME-CORP!/HQ Office/AP@Lobby/state  # 包含特殊字符和空格
rtk/v1/a/b/c/state                          # 太簡潔，無意義
rtk/v1/acme_corporation_headquarters_building/office_room_101_lobby_access_point_device/state  # 太冗長
```

## 主題長度限制

### MQTT 協議限制
- 主題總長度: 最大65535字節
- 實際建議: 小於256字節

### RTK 建議限制
- tenant: 3-32字符
- site: 3-32字符  
- device_id: 6-64字符
- 總主題長度: 小於200字符

## 保留消息策略

### 需要保留的消息類型
- `state` - 設備狀態 (需要最新狀態)
- `attr` - 設備屬性 (需要設備信息)
- `lwt` - 遺言消息 (需要離線狀態)

### 不保留的消息類型
- `telemetry` - 遙測數據 (時序性數據)
- `evt` - 事件通知 (實時性事件)
- `cmd/*` - 命令相關 (一次性操作)

## 版本演進策略

### 向後兼容
- v1主題永久支持
- 新版本採用並行支持方式
- 設備可選擇支持的版本

### 未來版本示例
```
rtk/v2/{tenant}/{site}/{device_id}/{message_type}
rtk/v3/{tenant}/{site}/{device_id}/{message_type}
```

## 監控和調試

### 有用的訂閱模式

#### 監控所有狀態
```
rtk/v1/+/+/+/state
```

#### 監控特定租戶
```
rtk/v1/company_a/+/+/#
```

#### 監控所有事件
```
rtk/v1/+/+/+/evt/#
```

#### 監控命令執行
```
rtk/v1/+/+/+/cmd/req
rtk/v1/+/+/+/cmd/ack  
rtk/v1/+/+/+/cmd/res
```

## 相關文檔

- [MQTT Protocol Specification](MQTT_PROTOCOL_SPEC.md) - 完整協議規範
- [Commands Reference](COMMANDS_EVENTS_REFERENCE.md) - 命令和事件參考
- [Schema Reference](SCHEMA_REFERENCE.md) - JSON Schema定義
# RTK MQTT Protocol Specification

## 概述

RTK MQTT協議是一個專為IoT設備診斷和網絡管理設計的通信協議，基於MQTT 3.1.1標準構建。本協議支持設備狀態監控、命令執行、事件通知、拓撲發現和診斷數據收集。

## 協議版本

- **協議版本**: v1
- **MQTT版本**: 3.1.1
- **消息格式**: JSON
- **字符編碼**: UTF-8

## 主題結構

### 基本主題格式
```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}[/{sub_type}]
```

### 參數說明
- `tenant`: 租戶標識符 (例如: company_a, user_123)
- `site`: 站點標識符 (例如: office_main, home_network)  
- `device_id`: 設備唯一標識符 (通常為MAC地址)
- `message_type`: 消息類型 (state, telemetry, evt, attr, cmd, lwt, topology)
- `sub_type`: 子類型 (可選，用於細分消息類型)

### 消息類型分類

#### 1. 狀態消息 (state)
- **主題**: `rtk/v1/{tenant}/{site}/{device_id}/state`
- **QoS**: 1
- **Retained**: true
- **用途**: 設備健康狀態和基本信息摘要

#### 2. 遙測數據 (telemetry)
- **主題**: `rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}`
- **QoS**: 0-1
- **Retained**: false
- **用途**: 性能指標和測量數據

#### 3. 事件通知 (evt)
- **主題**: `rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}`
- **QoS**: 1
- **Retained**: false
- **用途**: 事件和告警通知

#### 4. 設備屬性 (attr)
- **主題**: `rtk/v1/{tenant}/{site}/{device_id}/attr`
- **QoS**: 1
- **Retained**: true
- **用途**: 設備靜態屬性和配置信息

#### 5. 命令流程 (cmd)
- **請求**: `rtk/v1/{tenant}/{site}/{device_id}/cmd/req`
- **確認**: `rtk/v1/{tenant}/{site}/{device_id}/cmd/ack`
- **響應**: `rtk/v1/{tenant}/{site}/{device_id}/cmd/res`
- **QoS**: 1
- **Retained**: false
- **用途**: 設備命令執行流程

#### 6. 遺言消息 (lwt)
- **主題**: `rtk/v1/{tenant}/{site}/{device_id}/lwt`
- **QoS**: 1
- **Retained**: true
- **用途**: 設備斷線檢測

#### 7. 拓撲更新 (topology)
- **主題**: `rtk/v1/{tenant}/{site}/{device_id}/topology/update`
- **QoS**: 1
- **Retained**: false
- **用途**: 網絡拓撲變更通知

## 消息格式規範

### 通用消息結構
```json
{
  "schema": "message_type.sub_type/version",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "data": {
    // 具體消息內容
  },
  "meta": {
    // 元數據 (可選)
  }
}
```

### 字段說明
- `schema`: 消息架構標識，格式為 `{type}.{subtype}/{version}`
- `ts`: Unix時間戳 (毫秒)
- `device_id`: 設備標識符
- `data`: 消息負載數據
- `meta`: 可選元數據 (如來源、優先級等)

## 命令執行協議

### 命令請求格式 (cmd/req)
```json
{
  "id": "cmd-1699123456789",
  "op": "wifi_scan",
  "schema": "cmd.wifi_scan/1.0",
  "args": {
    "scan_type": "active",
    "duration": 10
  },
  "timeout_ms": 30000,
  "expect": "result",
  "reply_to": null,
  "ts": 1699123456789
}
```

### 命令確認格式 (cmd/ack)
```json
{
  "id": "cmd-1699123456789",
  "schema": "cmd.ack/1.0",
  "status": "accepted",
  "ts": 1699123456789
}
```

### 命令響應格式 (cmd/res)
```json
{
  "id": "cmd-1699123456789",
  "schema": "cmd.wifi_scan.result/1.0",
  "status": "completed",
  "result": {
    // 命令執行結果
  },
  "error": null,
  "ts": 1699123456789
}
```

## 支持的命令類型

### 網絡診斷命令
- `cmd/req/speed_test` - 網絡速度測試
- `cmd/req/wan_connectivity` - WAN連接診斷
- `cmd/req/latency_test` - 延遲測試
- `cmd/req/dns_resolution` - DNS解析測試

### WiFi管理命令
- `cmd/req/wifi_scan` - WiFi頻道掃描
- `cmd/req/interference_analysis` - 干擾分析
- `cmd/req/wifi_scan_channels` - 綜合WiFi頻道掃描
- `cmd/req/wifi_analyze_interference` - 先進干擾檢測
- `cmd/req/wifi_spectrum_utilization` - 頻譜利用率分析
- `cmd/req/wifi_signal_strength_map` - 信號強度映射
- `cmd/req/wifi_coverage_analysis` - 覆蓋範圍分析
- `cmd/req/wifi_roaming_optimization` - 漫遊優化
- `cmd/req/wifi_throughput_analysis` - 吞吐量分析
- `cmd/req/wifi_latency_profiling` - 延遲分析

### 系統管理命令
- `cmd/req/restart` - 設備重啟
- `cmd/req/reboot` - 設備重新啟動
- `cmd/req/restart_service` - 服務重新啟動
- `cmd/req/get_system_info` - 獲取系統信息
- `cmd/req/update_config` - 更新配置
- `cmd/req/run_diagnostics` - 運行診斷
- `cmd/req/set_log_level` - 設置日誌級別
- `cmd/req/get_logs` - 獲取日誌
- `cmd/req/network_test` - 網絡測試
- `cmd/req/cancel_command` - 取消命令
- `cmd/req/device.status` - 設備狀態查詢
- `cmd/req/firmware_update` - 固件更新

### 其他命令分類
詳細的命令參考請參見 [COMMANDS_EVENTS_REFERENCE.md](COMMANDS_EVENTS_REFERENCE.md)

## QoS 策略

| 消息類型 | QoS Level | Retained | 說明 |
|---------|-----------|----------|------|
| state | 1 | true | 確保狀態消息可靠傳遞並保留 |
| telemetry | 0-1 | false | 根據重要性調整QoS |
| evt | 1 | false | 事件消息需要可靠傳遞 |
| attr | 1 | true | 屬性消息需要保留 |
| cmd/* | 1 | false | 命令消息需要可靠傳遞 |
| lwt | 1 | true | 遺言消息需要保留 |
| topology | 1 | false | 拓撲更新需要可靠傳遞 |

## 錯誤處理

### 錯誤代碼
- `1000`: 未知錯誤
- `1001`: 消息格式錯誤
- `1002`: 不支持的命令
- `1003`: 參數錯誤
- `1004`: 設備離線
- `1005`: 超時
- `1006`: 權限不足

### 錯誤響應格式
```json
{
  "id": "cmd-1699123456789",
  "schema": "cmd.error/1.0",
  "status": "failed",
  "error": {
    "code": 1002,
    "message": "Unsupported command",
    "details": "Command 'invalid_cmd' is not supported by this device"
  },
  "ts": 1699123456789
}
```

## 安全考量

### 認證和授權
- 支持MQTT用戶名/密碼認證
- 支持SSL/TLS加密傳輸
- 基於主題的訪問控制

### 數據保護
- 敏感數據加密傳輸
- 設備標識符匿名化選項
- 數據保留策略配置

## 兼容性

### MQTT Broker兼容性
- Eclipse Mosquitto 2.0+
- EMQ X 4.0+
- AWS IoT Core
- Azure IoT Hub

### 客戶端庫支持
- Paho MQTT (C/C++, Python, Java, JavaScript)
- Eclipse Paho (Go)
- MQTT.js (Node.js)

## 版本演進

### v1.0 (當前版本)
- 基本協議框架
- 核心消息類型支持
- 命令執行流程

### 未來版本計劃
- v1.1: 批量命令支持
- v1.2: 增強安全特性
- v2.0: 協議優化和新特性

## 相關文檔

- [Schema Reference](SCHEMA_REFERENCE.md) - JSON Schema詳細定義
- [Commands Reference](COMMANDS_EVENTS_REFERENCE.md) - 命令和事件參考
- [Topic Structure](TOPIC_STRUCTURE.md) - 主題結構詳解
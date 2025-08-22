# 上行訊息結構 (Device → Controller)

## 概述

上行訊息是設備主動向控制器發送的資料，包括狀態報告、遙測數據、事件通知和設備屬性等。這些訊息讓控制器能夠全面了解設備的健康狀況、效能指標和重要事件。

## 訊息類型總覽

| 訊息類型 | Topic 格式 | QoS | Retained | 用途 |
|---------|------------|-----|----------|------|
| `state` | `rtk/v1/{tenant}/{site}/{device_id}/state` | 1 | ✅ | 設備狀態摘要 |
| `telemetry/{metric}` | `rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}` | 0-1 | ❌ | 遙測數據 |
| `evt/{event_type}` | `rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}` | 1 | ❌ | 事件通知 |
| `attr` | `rtk/v1/{tenant}/{site}/{device_id}/attr` | 1 | ✅ | 設備屬性 |
| `topology/{type}` | `rtk/v1/{tenant}/{site}/{device_id}/topology/{type}` | 1 | ❌ | 拓撲資訊 |
| `lwt` | `rtk/v1/{tenant}/{site}/{device_id}/lwt` | 1 | ✅ | 生命狀態 |

## state 訊息 (狀態報告)

### 用途與時機
- **用途**: 設備整體健康狀態和關鍵指標摘要
- **發送頻率**: 每 30-60 秒或狀態變化時
- **Retained**: 是，確保訂閱者能立即獲得最新狀態

### Topic 格式
```
rtk/v1/{tenant}/{site}/{device_id}/state
```

### 核心欄位

| 欄位 | 型態 | 必要 | 說明 | 範例值 |
|------|------|------|------|--------|
| `health` | string | ✅ | 整體健康狀態 | `ok`, `warning`, `error`, `critical` |
| `uptime_s` | integer | ✅ | 運行時間（秒） | `86400` |
| `cpu_usage` | number | 建議 | CPU 使用率 (%) | `45.2` |
| `memory_usage` | number | 建議 | 記憶體使用率 (%) | `62.8` |
| `temperature_c` | number | 可選 | 設備溫度（攝氏度） | `42.5` |
| `connection_status` | string | 建議 | 網路連接狀態 | `connected`, `disconnected` |

### 範例訊息
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "health": "ok",
  "uptime_s": 86400,
  "cpu_usage": 45.2,
  "memory_usage": 62.8,
  "temperature_c": 42.5,
  "connection_status": "connected",
  "interfaces": {
    "eth0": {
      "status": "up",
      "ip": "192.168.1.100"
    },
    "wlan0": {
      "status": "up", 
      "ip": "192.168.1.101",
      "ssid": "office-wifi"
    }
  }
}
```

## telemetry 訊息 (遙測數據)

### 用途與時機
- **用途**: 定期收集的監控數據和效能指標
- **發送頻率**: 每 10-60 秒，依重要性調整
- **Retained**: 否，避免歷史數據干擾

### Topic 格式
```
rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}
```

### 常見 Metric 類型

#### 硬體診斷
```bash
rtk/v1/office/floor1/aabbccddeeff/telemetry/temperature
rtk/v1/office/floor1/aabbccddeeff/telemetry/cpu_usage
rtk/v1/office/floor1/aabbccddeeff/telemetry/memory_usage
rtk/v1/office/floor1/aabbccddeeff/telemetry/disk_usage
rtk/v1/office/floor1/aabbccddeeff/telemetry/fan_speed
```

#### 網路診斷
```bash
rtk/v1/office/floor1/aabbccddeeff/telemetry/interface.eth0.rx_bytes
rtk/v1/office/floor1/aabbccddeeff/telemetry/interface.eth0.tx_bytes
rtk/v1/office/floor1/aabbccddeeff/telemetry/ping_latency
rtk/v1/office/floor1/aabbccddeeff/telemetry/bandwidth_usage
```

#### WiFi 診斷
```bash
rtk/v1/office/floor1/aabbccddeeff/telemetry/wifi_clients
rtk/v1/office/floor1/aabbccddeeff/telemetry/wifi.rssi
rtk/v1/office/floor1/aabbccddeeff/telemetry/wifi.channel_utilization
rtk/v1/office/floor1/aabbccddeeff/telemetry/wifi.scan_result
```

### 範例訊息

#### 溫度遙測
```json
{
  "schema": "telemetry.temperature/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "sensor_id": "cpu_temp",
  "value": 42.5,
  "unit": "celsius",
  "location": "cpu_core_0"
}
```

#### 網路介面遙測
```json
{
  "schema": "telemetry.network_interface/1.0", 
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "interface": "eth0",
  "rx_bytes": 1048576000,
  "tx_bytes": 524288000,
  "rx_packets": 1000000,
  "tx_packets": 500000,
  "errors": 0,
  "drops": 0
}
```

#### WiFi 客戶端遙測
```json
{
  "schema": "telemetry.wifi_clients/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "total_clients": 12,
  "clients": [
    {
      "mac": "112233445566",
      "ip": "192.168.1.50",
      "hostname": "laptop-001",
      "rssi": -45,
      "tx_rate": 150,
      "rx_rate": 144,
      "connected_time_s": 3600
    }
  ]
}
```

## evt 訊息 (事件通知)

### 用途與時機
- **用途**: 重要事件、告警和狀態變化的即時通知
- **發送時機**: 事件發生時立即發送
- **Retained**: 否，事件具有時效性

### Topic 格式
```
rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}
```

### 事件分類

#### 系統事件
```bash
rtk/v1/office/floor1/aabbccddeeff/evt/system.error
rtk/v1/office/floor1/aabbccddeeff/evt/system.warning  
rtk/v1/office/floor1/aabbccddeeff/evt/system.recovery
rtk/v1/office/floor1/aabbccddeeff/evt/system.restart
```

#### 硬體事件
```bash
rtk/v1/office/floor1/aabbccddeeff/evt/hardware.fault
rtk/v1/office/floor1/aabbccddeeff/evt/hardware.overheat
rtk/v1/office/floor1/aabbccddeeff/evt/power.failure
rtk/v1/office/floor1/aabbccddeeff/evt/fan.failed
```

#### 網路事件
```bash
rtk/v1/office/floor1/aabbccddeeff/evt/network.disconnected
rtk/v1/office/floor1/aabbccddeeff/evt/network.latency_high
rtk/v1/office/floor1/aabbccddeeff/evt/interface.down
rtk/v1/office/floor1/aabbccddeeff/evt/dhcp.lease_expired
```

#### WiFi 事件
```bash
rtk/v1/office/floor1/aabbccddeeff/evt/wifi.roam_triggered
rtk/v1/office/floor1/aabbccddeeff/evt/wifi.scan_failed
rtk/v1/office/floor1/aabbccddeeff/evt/wifi.signal_weak
rtk/v1/office/floor1/aabbccddeeff/evt/wifi.ap_changed
```

### 範例訊息

#### WiFi 漫遊事件
```json
{
  "schema": "evt.wifi.roam_triggered/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "event_type": "roam_triggered",
  "severity": "info",
  "details": {
    "trigger_reason": "signal_strength_low",
    "current_rssi": -70,
    "threshold_rssi": -65,
    "current_bssid": "aabbccddeeff",
    "target_bssid": "bbccddeeff00",
    "scan_duration_ms": 500
  }
}
```

#### 系統錯誤事件
```json
{
  "schema": "evt.system.error/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "event_type": "system_error",
  "severity": "error",
  "error_code": "E001",
  "error_message": "Memory allocation failed",
  "component": "network_driver",
  "details": {
    "process_id": 1234,
    "memory_requested": 1048576,
    "available_memory": 524288
  }
}
```

## attr 訊息 (設備屬性)

### 用途與時機
- **用途**: 設備的靜態屬性和能力宣告
- **發送時機**: 設備啟動、屬性變更、能力更新
- **Retained**: 是，確保新訂閱者能獲得設備資訊

### Topic 格式
```
rtk/v1/{tenant}/{site}/{device_id}/attr
```

### 核心欄位

| 欄位 | 型態 | 必要 | 說明 |
|------|------|------|------|
| `model` | string | ✅ | 設備型號 |
| `manufacturer` | string | ✅ | 製造商 |
| `serial_number` | string | ✅ | 序號 |
| `hardware_version` | string | ✅ | 硬體版本 |
| `firmware_version` | string | ✅ | 韌體版本 |
| `device_type` | string | ✅ | 設備類型 |
| `capabilities` | object | 建議 | 設備能力 |

### 範例訊息
```json
{
  "schema": "attr/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "model": "RTK-AP-2024",
  "manufacturer": "RTK Systems",
  "serial_number": "RTK240813001",
  "hardware_version": "v1.2",
  "firmware_version": "2.1.3",
  "device_type": "wireless_access_point",
  "capabilities": {
    "wireless": {
      "standards": ["802.11ac", "802.11n"],
      "frequencies": ["2.4GHz", "5GHz"],
      "max_clients": 100
    },
    "diagnostics": {
      "speed_test": true,
      "site_survey": true,
      "client_monitoring": true
    },
    "management": {
      "remote_config": true,
      "firmware_update": true,
      "factory_reset": true
    },
    "llm_integration": {
      "supported": true,
      "tools": ["wifi.get_environment", "wifi.speedtest", "topology.discover"]
    }
  }
}
```

## topology 訊息 (拓撲資訊)

### 用途與時機
- **用途**: 網路拓撲發現和連接狀態資訊
- **發送時機**: 拓撲變化、定期掃描、請求觸發
- **Retained**: 否，拓撲資訊具有時效性

### 子類型

#### topology/discovery (拓撲發現)
```json
{
  "schema": "topology.discovery/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "discovered_devices": [
    {
      "mac": "112233445566",
      "ip": "192.168.1.50",
      "hostname": "laptop-001",
      "device_type": "client",
      "interface": "wlan0",
      "last_seen": 1699123456789
    }
  ],
  "network_info": {
    "subnet": "192.168.1.0/24",
    "gateway": "192.168.1.1",
    "dns_servers": ["8.8.8.8", "8.8.4.4"]
  }
}
```

#### topology/connections (連接狀態)
```json
{
  "schema": "topology.connections/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "arp_table": [
    {
      "ip": "192.168.1.50",
      "mac": "112233445566",
      "interface": "eth0",
      "state": "reachable"
    }
  ],
  "dhcp_leases": [
    {
      "ip": "192.168.1.50",
      "mac": "112233445566",
      "hostname": "laptop-001",
      "lease_start": 1699123456789,
      "lease_duration": 86400
    }
  ]
}
```

## lwt 訊息 (Last Will Testament)

### 用途與時機
- **用途**: 設備上下線狀態通知
- **發送時機**: 連線建立、正常斷線、異常斷線
- **Retained**: 是，確保狀態持久化

### 範例訊息

#### 上線狀態
```json
{
  "schema": "lwt/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "status": "online",
  "connection_info": {
    "client_id": "rtk-aabbccddeeff",
    "clean_session": false,
    "keepalive": 60
  }
}
```

#### 下線狀態
```json
{
  "schema": "lwt/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff", 
  "status": "offline",
  "reason": "unexpected_disconnect",
  "last_seen": 1699123456789
}
```

## 發送頻率指南

| 訊息類型 | 建議頻率 | 條件觸發 |
|---------|---------|---------|
| `state` | 30-60 秒 | 健康狀態變化 |
| `telemetry/temperature` | 60 秒 | 溫度變化 > 2°C |
| `telemetry/cpu_usage` | 30 秒 | CPU 使用率變化 > 10% |
| `telemetry/wifi_clients` | 60 秒 | 客戶端數量變化 |
| `evt/*` | 即時 | 事件發生時 |
| `attr` | 啟動時 | 屬性變更時 |
| `topology/*` | 5 分鐘 | 拓撲變化時 |

## QoS 設定建議

| 訊息類型 | QoS | 理由 |
|---------|-----|------|
| `state` | 1 | 狀態資訊重要，需要確保送達 |
| `telemetry/*` | 0 | 高頻數據，偶爾丟失可接受 |
| `evt/*` | 1 | 事件通知重要，不可丟失 |
| `attr` | 1 | 屬性資訊重要，需要確保送達 |
| `topology/*` | 1 | 拓撲資訊重要，需要確保送達 |
| `lwt` | 1 | 生命狀態重要，需要確保送達 |

---

**下一步**: 閱讀 [下行命令結構](09-downlink-commands.md) 了解控制器向設備發送命令的格式。
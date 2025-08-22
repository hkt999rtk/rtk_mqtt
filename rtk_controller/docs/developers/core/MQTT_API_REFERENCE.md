# RTK MQTT API 完整參考

## 概述

這是 RTK MQTT 協議的完整 API 參考文檔，涵蓋所有訊息類型、命令格式、事件結構和響應規範。本文檔為開發者提供精確的技術規格，確保協議實作的一致性。

📋 **JSON Schema 驗證**: 所有 API 格式都有對應的 JSON Schema 定義，位於 [`docs/spec/schemas/`](../../spec/schemas/) 目錄。

## 🏗️ API 架構

### 主題結構
```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}[/{sub_type}]
```

### 訊息分類
- **狀態訊息**: `state`, `attr`, `lwt`
- **遙測資料**: `telemetry/{metric}`
- **事件通知**: `evt/{event_type}`
- **命令控制**: `cmd/{req|ack|res}`
- **拓撲管理**: `topology/{update|discovery}`

## 📊 狀態訊息 API

### state - 設備狀態
**主題**: `rtk/v1/{tenant}/{site}/{device_id}/state`  
**QoS**: 1  
**Retained**: true  
**頻率**: 每 5 分鐘或狀態變化時

#### 請求格式
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "health": "ok",
    "uptime_s": 86400,
    "cpu_usage": 25.4,
    "memory_usage": 45.2,
    "connection_status": "connected",
    "device_specific": {
      "wifi": {
        "enabled": true,
        "ssid": "HomeNetwork",
        "channel": 6,
        "connected_clients": 12,
        "signal_strength": -45
      },
      "network": {
        "wan_connected": true,
      "lan_ip": "192.168.1.1",
      "wan_ip": "203.0.113.1",
      "throughput_mbps": 85.2
    }
  }
}
```

#### 參數說明
| 欄位 | 類型 | 必要 | 說明 |
|------|------|------|------|
| `schema` | string | ✅ | Schema 版本標識 |
| `ts` | integer | ✅ | Unix 時間戳 (毫秒) |
| `health` | string | ✅ | 設備健康狀態: ok/warning/error |
| `uptime_s` | integer | ✅ | 設備運行時間 (秒) |
| `cpu_usage` | number | ✅ | CPU 使用率 (%) |
| `memory_usage` | number | ✅ | 記憶體使用率 (%) |
| `connection_status` | string | ✅ | 連接狀態: connected/disconnected |
| `device_specific` | object | ❌ | 設備特定的狀態資料 |

### attr - 設備屬性
**主題**: `rtk/v1/{tenant}/{site}/{device_id}/attr`  
**QoS**: 1  
**Retained**: true  
**觸發**: 設備啟動或屬性變更時

#### 請求格式
```json
{
  "schema": "attr/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "device_type": "router",
    "manufacturer": "RTK Systems",
    "model": "RTK-R2024",
    "firmware_version": "2.1.3",
    "hardware_version": "rev-b",
    "serial_number": "RTK202400123",
    "mac_address": "aa:bb:cc:dd:ee:ff",
    "capabilities": ["wifi", "ethernet", "mesh", "qos", "diagnostics"],
    "supported_commands": [
      "speed_test", "wifi_scan", "topology_update", 
      "qos_config", "device_reset"
    ],
    "supported_events": [
      "wifi.connection_lost", "network.client_connect",
      "system.error", "mesh.node_join"
    ]
  }
}
```

### lwt - 遺言訊息
**主題**: `rtk/v1/{tenant}/{site}/{device_id}/lwt`  
**QoS**: 1  
**Retained**: true  
**觸發**: 連接建立/斷開時

#### 線上狀態
```json
{
  "schema": "lwt/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "status": "online"
}
```

#### 離線狀態
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

## 📈 遙測資料 API

### telemetry/{metric} - 遙測資料
**主題**: `rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}`  
**QoS**: 0  
**Retained**: false  
**頻率**: 每 30 秒或變化超過閾值時

#### 通用格式
```json
{
  "schema": "telemetry.{metric}/1.0",
  "ts": 1699123456789,
  "value": 45.2,
  "unit": "%",
  "tags": {
    "source": "internal_sensor",
    "location": "cpu_core_0"
  }
}
```

#### 常見遙測類型

##### CPU 使用率
**主題**: `telemetry/cpu_usage`
```json
{
  "schema": "telemetry.cpu_usage/1.0",
  "ts": 1699123456789,
  "value": 35.7,
  "unit": "%"
}
```

##### 網路吞吐量
**主題**: `telemetry/network_throughput`
```json
{
  "schema": "telemetry.network_throughput/1.0",
  "ts": 1699123456789,
  "value": 85.3,
  "unit": "mbps",
  "tags": {
    "direction": "download",
    "interface": "eth0"
  }
}
```

##### WiFi 訊號強度
**主題**: `telemetry/wifi_signal`
```json
{
  "schema": "telemetry.wifi_signal/1.0",
  "ts": 1699123456789,
  "value": -45,
  "unit": "dBm",
  "tags": {
    "ssid": "HomeNetwork",
    "band": "5ghz"
  }
}
```

## 🔔 事件通知 API

### evt/{event_type} - 事件通知
**主題**: `rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}`  
**QoS**: 1  
**Retained**: false  
**觸發**: 事件發生時立即發送

#### 通用事件格式
```json
{
  "schema": "evt.{event_type}/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "event_type": "network.client_connect",
  "severity": "info",
  "data": {
    // 事件特定資料
  }
}
```

#### WiFi 事件

##### wifi.connection_lost - WiFi 連接丟失
```json
{
  "schema": "evt.wifi.connection_lost/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "event_type": "wifi.connection_lost",
  "severity": "warning",
  "data": {
    "client_mac": "11:22:33:44:55:66",
    "ssid": "HomeNetwork",
    "disconnection_reason": "signal_weak",
    "last_signal_strength": -78,
    "connection_duration": 3600
  }
}
```

##### wifi.roam_triggered - WiFi 漫遊觸發
```json
{
  "schema": "evt.wifi.roam_triggered/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "event_type": "wifi.roam_triggered",
  "severity": "info",
  "data": {
    "client_mac": "11:22:33:44:55:66",
    "from_ap": "aabbccddeeff",
    "to_ap": "ffeeddccbbaa",
    "reason": "signal_optimization",
    "trigger_rssi": -70
  }
}
```

#### 網路事件

##### network.client_connect - 客戶端連接
```json
{
  "schema": "evt.network.client_connect/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "event_type": "network.client_connect",
  "severity": "info",
  "data": {
    "client_mac": "11:22:33:44:55:66",
    "client_ip": "192.168.1.100",
    "hostname": "laptop-001",
    "connection_type": "wifi",
    "auth_method": "wpa2"
  }
}
```

#### 系統事件

##### system.error - 系統錯誤
```json
{
  "schema": "evt.system.error/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "event_type": "system.error",
  "severity": "error",
  "data": {
    "error_code": "E_MEMORY_LOW",
    "error_message": "Available memory below 10%",
    "component": "system_monitor",
    "memory_usage": 92.5,
    "recommended_action": "restart_services"
  }
}
```

## 🎛️ 命令控制 API

### cmd/req - 命令請求
**主題**: `rtk/v1/{tenant}/{site}/{device_id}/cmd/req`  
**QoS**: 1  
**發送方**: Controller → Device

#### 通用命令格式
```json
{
  "schema": "cmd.{operation}/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-{operation}-{timestamp}",
    "op": "{operation}",
    "args": {
      // 命令特定參數
    },
    "timeout_ms": 30000,
    "trace": {
      "session_id": "session-abc123",
      "trace_id": "trace-def456"
    }
  }
}
```

### cmd/ack - 命令確認
**主題**: `rtk/v1/{tenant}/{site}/{device_id}/cmd/ack`  
**QoS**: 1  
**發送方**: Device → Controller

```json
{
  "schema": "cmd.ack/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-speed-test-123",
    "status": "received",
    "estimated_duration": 30
  }
}
```

### cmd/res - 命令結果
**主題**: `rtk/v1/{tenant}/{site}/{device_id}/cmd/res`  
**QoS**: 1  
**發送方**: Device → Controller

#### 成功結果
```json
{
  "schema": "cmd.result/1.0",
  "ts": 1699123456820,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-speed-test-123",
    "status": "completed",
    "execution_time_ms": 30150,
    "result": {
      // 命令特定結果
    }
  }
}
```

#### 錯誤結果
```json
{
  "schema": "cmd.error/1.0",
  "ts": 1699123456820,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-speed-test-123",
    "status": "error",
    "error_code": "E_NETWORK_UNREACHABLE",
    "error_message": "Cannot reach speed test server",
    "execution_time_ms": 5000
  }
}
```

## 🔧 診斷命令 API

### speed_test - 網路速度測試

#### 請求
```json
{
  "schema": "cmd.speed_test/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-speed-test-123",
    "op": "speed_test",
    "args": {
      "server": "auto",
      "duration": 30,
      "direction": "both"
    },
    "timeout_ms": 60000
  }
}
```

#### 結果
```json
{
  "schema": "cmd.speed_test.result/1.0",
  "ts": 1699123456820,
  "device_id": "aabbccddeeff", 
  "payload": {
    "id": "cmd-speed-test-123",
    "status": "completed",
    "result": {
      "download_mbps": 85.2,
      "upload_mbps": 12.4,
      "latency_ms": 15.3,
      "jitter_ms": 2.1,
      "packet_loss": 0.0,
      "server": "speedtest.example.com",
      "test_duration": 30
    }
  }
}
```

### wan_connectivity - WAN 連接診斷

#### 請求
```json
{
  "schema": "cmd.wan_connectivity/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-wan-test-123",
    "op": "wan_connectivity", 
    "args": {
      "dns_servers": ["8.8.8.8", "1.1.1.1"],
      "test_hosts": ["google.com", "cloudflare.com"],
      "timeout": 10
    },
    "timeout_ms": 30000
  }
}
```

#### 結果
```json
{
  "schema": "cmd.wan_connectivity.result/1.0",
  "ts": 1699123456820,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-wan-test-123", 
    "status": "completed",
    "result": {
      "wan_status": "connected",
      "dns_resolution": {
        "8.8.8.8": true,
        "1.1.1.1": true
      },
      "external_connectivity": {
        "google.com": {
          "reachable": true,
          "response_time_ms": 25.3
        },
        "cloudflare.com": {
          "reachable": true,
          "response_time_ms": 18.7
        }
      },
      "gateway_reachable": true,
      "public_ip": "203.0.113.1"
    }
  }
}
```

### wifi_scan - WiFi 掃描

#### 請求
```json
{
  "schema": "cmd.wifi_scan/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-wifi-scan-123",
    "op": "wifi_scan",
    "args": {
      "channels": [1, 6, 11],
      "scan_duration": 5,
      "include_hidden": true
    },
    "timeout_ms": 15000
  }
}
```

#### 結果
```json
{
  "schema": "cmd.wifi_scan.result/1.0", 
  "ts": 1699123456820,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-wifi-scan-123",
    "status": "completed",
    "result": {
      "scan_time": 1699123456820,
      "networks": [
        {
          "ssid": "HomeNetwork",
          "bssid": "aa:bb:cc:dd:ee:ff",
          "channel": 6,
          "frequency": 2437,
          "signal_strength": -45,
          "security": "WPA2",
          "bandwidth": "20MHz"
        },
        {
          "ssid": "NeighborWiFi",
          "bssid": "ff:ee:dd:cc:bb:aa", 
          "channel": 11,
          "frequency": 2462,
          "signal_strength": -68,
          "security": "WPA3",
          "bandwidth": "40MHz"
        }
      ],
      "interference_analysis": {
        "channel_utilization": {
          "1": 25.3,
          "6": 78.9,
          "11": 45.2
        },
        "recommended_channel": 1
      }
    }
  }
}
```

## 🌐 拓撲管理 API

### topology/update - 拓撲更新
**主題**: `rtk/v1/{tenant}/{site}/{device_id}/topology/update`  
**QoS**: 1  
**Retained**: false

```json
{
  "schema": "topology.update/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "discovered_devices": [
    {
      "mac_address": "11:22:33:44:55:66",
      "ip_address": "192.168.1.100",
      "hostname": "laptop-001",
      "device_type": "client",
      "manufacturer": "Apple",
      "connection_type": "wifi",
      "signal_strength": -52,
      "last_seen": 1699123456789
    }
  ],
  "connections": [
    {
      "from_device": "aabbccddeeff",
      "to_device": "11:22:33:44:55:66", 
      "connection_type": "wifi",
      "quality_score": 85.3,
      "bandwidth_mbps": 150
    }
  ],
  "changes": [
    {
      "change_type": "device_added",
      "device": "11:22:33:44:55:66",
      "timestamp": 1699123456789
    }
  ]
}
```

## 🔍 進階命令 API

### mesh_topology - Mesh 拓撲查詢

#### 請求
```json
{
  "schema": "cmd.mesh_topology/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-mesh-topo-123",
    "op": "mesh_topology",
    "args": {
      "include_metrics": true,
      "depth": 3
    },
    "timeout_ms": 15000
  }
}
```

#### 結果
```json
{
  "schema": "cmd.mesh_topology.result/1.0",
  "ts": 1699123456820,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-mesh-topo-123",
    "status": "completed", 
    "result": {
      "mesh_nodes": [
        {
          "node_id": "aabbccddeeff",
          "role": "root",
          "parent": null,
          "children": ["112233445566", "ffeeddccbbaa"],
          "hop_count": 0,
          "signal_strength": null
        },
        {
          "node_id": "112233445566",
          "role": "router",
          "parent": "aabbccddeeff", 
          "children": ["778899aabbcc"],
          "hop_count": 1,
          "signal_strength": -42
        }
      ],
      "backhaul_links": [
        {
          "from": "aabbccddeeff",
          "to": "112233445566",
          "type": "wireless",
          "band": "5ghz",
          "quality": 90.5,
          "throughput_mbps": 450
        }
      ],
      "metrics": {
        "total_nodes": 3,
        "max_hop_count": 2,
        "average_signal": -45.7
      }
    }
  }
}
```

### qos_config - QoS 配置

#### 請求
```json
{
  "schema": "cmd.qos_config/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-qos-config-123",
    "op": "qos_config",
    "args": {
      "operation": "apply",
      "policies": [
        {
          "name": "video_streaming",
          "priority": "high",
          "bandwidth_min": "10mbps",
          "bandwidth_max": "50mbps",
          "applications": ["netflix", "youtube", "video_call"]
        },
        {
          "name": "gaming", 
          "priority": "high",
          "latency_max": "20ms",
          "applications": ["gaming", "real_time"]
        }
      ]
    },
    "timeout_ms": 10000
  }
}
```

## 📝 API 使用最佳實踐

### 錯誤處理
```json
{
  "error_codes": {
    "E_INVALID_ARGS": "命令參數無效",
    "E_TIMEOUT": "命令執行超時",
    "E_NETWORK_ERROR": "網路錯誤",
    "E_PERMISSION_DENIED": "權限不足",
    "E_DEVICE_BUSY": "設備忙碌中",
    "E_UNSUPPORTED_OPERATION": "不支援的操作"
  }
}
```

### QoS 建議
| 訊息類型 | QoS 等級 | Retained | 原因 |
|----------|----------|----------|------|
| `state` | 1 | ✅ | 重要狀態資訊需要可靠傳輸 |
| `attr` | 1 | ✅ | 設備屬性需要持久保存 |
| `lwt` | 1 | ✅ | 連接狀態對系統運作至關重要 |
| `telemetry` | 0 | ❌ | 高頻率資料，允許部分遺失 |
| `evt` | 1 | ❌ | 事件通知重要但不需持久化 |
| `cmd/*` | 1 | ❌ | 命令控制需要可靠傳輸 |

### 頻率限制
- **狀態訊息**: 最多每分鐘 1 次
- **遙測資料**: 每個指標最多每 10 秒 1 次  
- **事件通知**: 無限制，但建議實作去重
- **命令響應**: 必須在超時時間內響應

### Schema 驗證
所有訊息都應該根據對應的 JSON Schema 進行驗證：
```bash
# 驗證狀態訊息
ajv validate -s docs/spec/schemas/state.json -d message.json

# 驗證命令格式
ajv validate -s docs/spec/schemas/cmd-speed-test.json -d command.json
```

## 🔗 相關資源

- **[協議規格文檔](../../spec/)** - 完整的協議規範
- **[JSON Schema 定義](../../spec/schemas/)** - 所有格式的 Schema 檔案
- **[整合指南](../INTEGRATION_GUIDE.md)** - 實際整合步驟
- **[測試工具](../tools/MQTT_TESTING_TOOLS.md)** - API 測試框架

---

此 API 參考文檔提供了 RTK MQTT 協議的完整技術規格，開發者可以根據此文檔實作標準相容的設備和應用程式。
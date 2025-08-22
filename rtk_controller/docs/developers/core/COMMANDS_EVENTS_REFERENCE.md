# RTK MQTT 命令和事件完整參考

## 概述

本文檔提供RTK MQTT協議中所有支持的命令和事件的完整參考。包括消息格式、參數說明、響應結構和使用示例。

## 命令分類

### 網絡診斷命令

#### cmd/req/speed_test - 網絡速度測試

**用途**: 執行網絡帶寬速度測試

**請求格式**:
```json
{
  "id": "cmd-speed-test-123",
  "op": "speed_test",
  "schema": "cmd.speed_test/1.0",
  "args": {
    "server": "auto",
    "duration": 30,
    "direction": "both"
  },
  "timeout_ms": 60000,
  "ts": 1699123456789
}
```

**參數說明**:
- `server`: 測試服務器 ("auto" 或具體服務器地址)
- `duration`: 測試持續時間 (秒)
- `direction`: 測試方向 ("download", "upload", "both")

**響應格式**:
```json
{
  "id": "cmd-speed-test-123",
  "schema": "cmd.speed_test.result/1.0",
  "status": "completed",
  "result": {
    "download_mbps": 85.2,
    "upload_mbps": 12.4,
    "latency_ms": 15.3,
    "jitter_ms": 2.1,
    "packet_loss": 0.0,
    "server": "speedtest.example.com",
    "test_duration": 30
  },
  "ts": 1699123456820
}
```

#### cmd/req/wan_connectivity - WAN連接診斷

**用途**: 檢測WAN連接狀態和外部網絡可達性

**請求格式**:
```json
{
  "id": "cmd-wan-test-123",
  "op": "wan_connectivity",
  "schema": "cmd.wan_connectivity/1.0",
  "args": {
    "dns_servers": ["8.8.8.8", "1.1.1.1"],
    "test_hosts": ["google.com", "cloudflare.com"],
    "timeout": 10
  },
  "timeout_ms": 30000,
  "ts": 1699123456789
}
```

**響應格式**:
```json
{
  "id": "cmd-wan-test-123",
  "schema": "cmd.wan_connectivity.result/1.0",
  "status": "completed",
  "result": {
    "wan_connected": true,
    "public_ip": "203.0.113.1",
    "gateway_reachable": true,
    "gateway_latency_ms": 5.2,
    "dns_resolution": {
      "8.8.8.8": {"reachable": true, "latency_ms": 8.1},
      "1.1.1.1": {"reachable": true, "latency_ms": 12.3}
    },
    "external_connectivity": {
      "google.com": {"reachable": true, "latency_ms": 25.4},
      "cloudflare.com": {"reachable": true, "latency_ms": 18.7}
    }
  },
  "ts": 1699123456820
}
```

#### cmd/req/latency_test - 延遲測試

**用途**: 測量到指定目標的網絡延遲

**請求格式**:
```json
{
  "id": "cmd-latency-123",
  "op": "latency_test",
  "schema": "cmd.latency_test/1.0",
  "args": {
    "targets": ["8.8.8.8", "gateway", "192.168.1.1"],
    "count": 10,
    "interval_ms": 1000
  },
  "timeout_ms": 30000,
  "ts": 1699123456789
}
```

**響應格式**:
```json
{
  "id": "cmd-latency-123",
  "schema": "cmd.latency_test.result/1.0",
  "status": "completed",
  "result": {
    "targets": [
      {
        "target": "8.8.8.8",
        "packets_sent": 10,
        "packets_received": 10,
        "packet_loss": 0.0,
        "min_latency_ms": 12.1,
        "max_latency_ms": 18.9,
        "avg_latency_ms": 15.2,
        "jitter_ms": 2.3
      }
    ]
  },
  "ts": 1699123456820
}
```

#### cmd/req/dns_resolution - DNS解析測試

**用途**: 測試DNS解析功能

**請求格式**:
```json
{
  "id": "cmd-dns-123",
  "op": "dns_resolution",
  "schema": "cmd.dns_resolution/1.0",
  "args": {
    "hostnames": ["google.com", "github.com"],
    "dns_servers": ["8.8.8.8", "system"],
    "timeout": 5
  },
  "timeout_ms": 15000,
  "ts": 1699123456789
}
```

### WiFi管理命令

#### cmd/req/wifi_scan - WiFi頻道掃描

**用途**: 掃描可用的WiFi網絡和頻道信息

**請求格式**:
```json
{
  "id": "cmd-wifi-scan-123",
  "op": "wifi_scan",
  "schema": "cmd.wifi_scan/1.0",
  "args": {
    "scan_type": "active",
    "duration": 10,
    "channels": [1, 6, 11, 36, 149],
    "include_hidden": true
  },
  "timeout_ms": 30000,
  "ts": 1699123456789
}
```

**參數說明**:
- `scan_type`: 掃描類型 ("active", "passive")
- `duration`: 掃描持續時間 (秒)
- `channels`: 指定掃描頻道 (空陣列表示掃描所有頻道)
- `include_hidden`: 是否包含隱藏網絡

**響應格式**:
```json
{
  "id": "cmd-wifi-scan-123",
  "schema": "cmd.wifi_scan.result/1.0",
  "status": "completed",
  "result": {
    "networks": [
      {
        "ssid": "HomeNetwork",
        "bssid": "aabbccddeeff",
        "channel": 6,
        "frequency": 2437,
        "signal_strength": -45,
        "security": "WPA2-PSK",
        "bandwidth": "20MHz",
        "hidden": false
      }
    ],
    "scan_duration": 10,
    "channels_scanned": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11]
  },
  "ts": 1699123456820
}
```

#### cmd/req/interference_analysis - 干擾分析

**用途**: 分析WiFi頻道干擾情況

**請求格式**:
```json
{
  "id": "cmd-interference-123",
  "op": "interference_analysis",
  "schema": "cmd.interference_analysis/1.0",
  "args": {
    "channels": [1, 6, 11, 36, 149],
    "duration": 30,
    "sensitivity": "high"
  },
  "timeout_ms": 45000,
  "ts": 1699123456789
}
```

**響應格式**:
```json
{
  "id": "cmd-interference-123",
  "schema": "cmd.interference_analysis.result/1.0",
  "status": "completed",
  "result": {
    "channel_analysis": [
      {
        "channel": 6,
        "frequency": 2437,
        "utilization": 65.2,
        "noise_floor": -95,
        "interference_sources": [
          {
            "type": "microwave",
            "strength": "medium",
            "frequency_range": "2450-2460"
          }
        ],
        "recommendation": "consider_channel_11"
      }
    ],
    "overall_assessment": "moderate_interference"
  },
  "ts": 1699123456820
}
```

### Mesh網絡命令

#### cmd/req/mesh_topology - Mesh拓撲查詢

**用途**: 獲取Mesh網絡拓撲信息

**請求格式**:
```json
{
  "id": "cmd-mesh-topo-123",
  "op": "mesh_topology",
  "schema": "cmd.mesh_topology/1.0",
  "args": {
    "include_metrics": true,
    "include_paths": true,
    "depth": 3
  },
  "timeout_ms": 20000,
  "ts": 1699123456789
}
```

**響應格式**:
```json
{
  "id": "cmd-mesh-topo-123",
  "schema": "cmd.mesh_topology.result/1.0",
  "status": "completed",
  "result": {
    "nodes": [
      {
        "node_id": "mesh-node-01",
        "role": "gateway",
        "parent": null,
        "children": ["mesh-node-02", "mesh-node-03"],
        "signal_strength": -30,
        "load": 25.4
      }
    ],
    "paths": [
      {
        "from": "mesh-node-02",
        "to": "gateway",
        "hops": 2,
        "path": ["mesh-node-02", "mesh-node-01", "gateway"],
        "quality": 85.2
      }
    ]
  },
  "ts": 1699123456820
}
```

#### cmd/req/backhaul_test - 回程連接測試

**用途**: 測試Mesh回程連接質量

**請求格式**:
```json
{
  "id": "cmd-backhaul-123",
  "op": "backhaul_test",
  "schema": "cmd.backhaul_test/1.0",
  "args": {
    "target_nodes": ["mesh-node-01", "mesh-node-02"],
    "test_duration": 30,
    "test_types": ["throughput", "latency", "stability"]
  },
  "timeout_ms": 45000,
  "ts": 1699123456789
}
```

### 拓撲管理命令

#### cmd/req/topology_update - 網絡拓撲更新

**用途**: 請求更新網絡拓撲信息

**請求格式**:
```json
{
  "id": "cmd-topo-update-123",
  "op": "topology_update",
  "schema": "cmd.topology_update/1.0",
  "args": {
    "force_discovery": true,
    "include_offline": false,
    "depth": 5
  },
  "timeout_ms": 30000,
  "ts": 1699123456789
}
```

#### cmd/req/client_list - 客戶端列表查詢

**用途**: 獲取連接的客戶端設備列表

**請求格式**:
```json
{
  "id": "cmd-clients-123",
  "op": "client_list",
  "schema": "cmd.client_list/1.0",
  "args": {
    "include_offline": true,
    "include_details": true,
    "filter_by_type": null
  },
  "timeout_ms": 15000,
  "ts": 1699123456789
}
```

### QoS管理命令

#### cmd/req/qos_config - QoS策略配置

**用途**: 配置服務質量(QoS)策略

**請求格式**:
```json
{
  "id": "cmd-qos-config-123",
  "op": "qos_config",
  "schema": "cmd.qos_config/1.0",
  "args": {
    "policy": "gaming",
    "rules": [
      {
        "name": "high_priority",
        "match": {"protocol": "udp", "port_range": "3000-4000"},
        "priority": "high",
        "bandwidth_limit": null
      }
    ],
    "apply_immediately": true
  },
  "timeout_ms": 10000,
  "ts": 1699123456789
}
```

#### cmd/req/traffic_stats - 流量統計查詢

**用途**: 獲取網絡流量統計信息

**請求格式**:
```json
{
  "id": "cmd-traffic-123",
  "op": "traffic_stats",
  "schema": "cmd.traffic_stats/1.0",
  "args": {
    "time_range": "last_hour",
    "granularity": "5min",
    "include_clients": true,
    "include_protocols": true
  },
  "timeout_ms": 10000,
  "ts": 1699123456789
}
```

### 系統管理命令

#### cmd/req/restart - 設備重啟

**用途**: 重啟設備

**請求格式**:
```json
{
  "id": "cmd-restart-123",
  "op": "restart",
  "schema": "cmd.restart/1.0",
  "args": {
    "delay_seconds": 5,
    "reason": "system_update"
  },
  "timeout_ms": 60000,
  "ts": 1699123456789
}
```

#### cmd/req/reboot - 設備重新啟動

**用途**: 重新啟動設備

**請求格式**:
```json
{
  "id": "cmd-reboot-123",
  "op": "reboot",
  "schema": "cmd.reboot/1.0",
  "args": {
    "force": false,
    "delay_seconds": 10
  },
  "timeout_ms": 120000,
  "ts": 1699123456789
}
```

#### cmd/req/get_system_info - 獲取系統信息

**用途**: 獲取設備系統信息

**請求格式**:
```json
{
  "id": "cmd-sysinfo-123",
  "op": "get_system_info",
  "schema": "cmd.get_system_info/1.0",
  "args": {
    "include": ["cpu", "memory", "disk", "network"],
    "detailed": true
  },
  "timeout_ms": 10000,
  "ts": 1699123456789
}
```

**響應格式**:
```json
{
  "id": "cmd-sysinfo-123",
  "schema": "cmd.get_system_info.result/1.0",
  "status": "completed",
  "result": {
    "cpu": {
      "usage_percent": 25.4,
      "load_average": [0.8, 1.2, 1.5],
      "cores": 4,
      "model": "ARMv8 Cortex-A72"
    },
    "memory": {
      "total_mb": 1024,
      "used_mb": 456,
      "free_mb": 568,
      "usage_percent": 44.5
    },
    "disk": {
      "total_mb": 8192,
      "used_mb": 2048,
      "free_mb": 6144,
      "usage_percent": 25.0
    },
    "network": {
      "interfaces": [
        {
          "name": "eth0",
          "ip": "192.168.1.100",
          "mac": "aabbccddeeff",
          "status": "up",
          "speed": "1000Mbps"
        }
      ]
    }
  },
  "ts": 1699123456820
}
```

#### cmd/req/firmware_update - 固件更新

**用途**: 執行固件更新

**請求格式**:
```json
{
  "id": "cmd-firmware-123",
  "op": "firmware_update",
  "schema": "cmd.firmware_update/1.0",
  "args": {
    "url": "https://updates.example.com/firmware-v2.1.0.bin",
    "version": "2.1.0",
    "checksum": "sha256:abcdef123456...",
    "auto_reboot": true,
    "backup_config": true
  },
  "timeout_ms": 300000,
  "ts": 1699123456789
}
```

#### cmd/req/device.status - 設備狀態查詢

**用途**: 查詢設備當前狀態

**請求格式**:
```json
{
  "id": "cmd-status-123",
  "op": "device.status",
  "schema": "cmd.device.status/1.0",
  "args": {},
  "timeout_ms": 5000,
  "ts": 1699123456789
}
```

## 事件分類

### WiFi事件

#### evt/wifi.connection_lost - WiFi連接丟失

**事件格式**:
```json
{
  "schema": "evt.wifi.connection_lost/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "data": {
    "client_mac": "112233445566",
    "ssid": "HomeNetwork",
    "reason": "signal_weak",
    "duration_connected": 1800,
    "signal_strength": -78
  }
}
```

#### evt/wifi.roam_triggered - WiFi漫遊觸發

**事件格式**:
```json
{
  "schema": "evt.wifi.roam_triggered/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "data": {
    "client_mac": "112233445566",
    "from_ap": "aabbccddeeff",
    "to_ap": "bbccddeeff00",
    "reason": "better_signal",
    "old_signal": -65,
    "new_signal": -45
  }
}
```

### 網絡事件

#### evt/network.link_up - 網絡連接建立

**事件格式**:
```json
{
  "schema": "evt.network.link_up/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "data": {
    "interface": "eth0",
    "speed": "1000Mbps",
    "duplex": "full",
    "ip_address": "192.168.1.100"
  }
}
```

#### evt/network.client_connect - 客戶端連接

**事件格式**:
```json
{
  "schema": "evt.network.client_connect/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "data": {
    "client_mac": "112233445566",
    "ip_address": "192.168.1.105",
    "connection_type": "wifi",
    "ssid": "HomeNetwork",
    "signal_strength": -45
  }
}
```

### 系統事件

#### evt/system.error - 系統錯誤

**事件格式**:
```json
{
  "schema": "evt.system.error/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "data": {
    "error_code": "E001",
    "error_message": "Memory usage high",
    "severity": "warning",
    "component": "system_monitor",
    "details": {
      "memory_usage": 85.2,
      "threshold": 80.0
    }
  }
}
```

### Mesh事件

#### evt/mesh.node_join - Mesh節點加入

**事件格式**:
```json
{
  "schema": "evt.mesh.node_join/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "data": {
    "node_id": "mesh-node-03",
    "parent_node": "mesh-node-01",
    "signal_strength": -55,
    "path_quality": 78.5
  }
}
```

## 錯誤處理

### 通用錯誤響應

```json
{
  "id": "cmd-example-123",
  "schema": "cmd.error/1.0",
  "status": "failed",
  "error": {
    "code": 1002,
    "message": "Command not supported",
    "details": "The device does not support the requested operation"
  },
  "ts": 1699123456820
}
```

### 錯誤代碼參考

| 代碼 | 名稱 | 描述 |
|------|------|------|
| 1000 | UNKNOWN_ERROR | 未知錯誤 |
| 1001 | INVALID_FORMAT | 消息格式錯誤 |
| 1002 | UNSUPPORTED_COMMAND | 不支持的命令 |
| 1003 | INVALID_PARAMETERS | 參數錯誤 |
| 1004 | DEVICE_OFFLINE | 設備離線 |
| 1005 | TIMEOUT | 操作超時 |
| 1006 | PERMISSION_DENIED | 權限不足 |
| 1007 | RESOURCE_BUSY | 資源忙碌 |
| 1008 | INSUFFICIENT_RESOURCES | 資源不足 |

## 最佳實踐

### 命令超時設置
- 快速查詢操作: 5-10秒
- 網絡測試: 30-60秒  
- 系統重啟: 60-120秒
- 固件更新: 300-600秒

### 重試策略
- 使用指數退避重試
- 最大重試次數: 3次
- 重試間隔: 1s, 2s, 4s

### 消息優先級
- 緊急命令: QoS 1, 立即處理
- 常規操作: QoS 1, 正常隊列
- 監控數據: QoS 0, 批量處理

## 相關文檔

- [MQTT Protocol Specification](MQTT_PROTOCOL_SPEC.md) - 協議完整規範
- [Schema Reference](SCHEMA_REFERENCE.md) - JSON Schema定義
- [Topic Structure](TOPIC_STRUCTURE.md) - 主題結構詳解
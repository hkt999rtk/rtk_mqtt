# 完整範例與應用場景

## 概述

本文檔提供 RTK MQTT 協議的完整使用範例，涵蓋常見的 IoT 設備整合場景、診斷流程和故障排除案例。這些範例展示了協議在實際環境中的應用方式。

## 場景一：智慧家庭網路診斷

### 1.1 場景描述
用戶反映「Wi-Fi 連上但不能上網」問題，需要進行全面的網路診斷。

### 1.2 設備拓撲
```
Internet ── Router (gateway-001) ── Switch (switch-001)
                │                        │
                │                   ─────┼─────
                │                   │         │
           Mesh Node A         Laptop     Smart TV
         (mesh-node-a)      (client-001) (client-002)
                │
         Smart Bulb
        (bulb-001)
```

### 1.3 完整診斷流程

#### 階段 1: 問題確認
```json
// 1. 檢查 WAN 連線狀態
{
  "id": "cmd-wan-check-001",
  "op": "diagnostics.wan_connectivity",
  "schema": "cmd.diagnostics.wan_connectivity/1.0",
  "trace": {
    "session_id": "diag-session-001",
    "trace_id": "wan-check-step-01"
  }
}

// 結果：WAN 連線正常
{
  "id": "cmd-wan-check-001",
  "status": "completed",
  "result": {
    "wan_status": "connected",
    "public_ip": "203.123.45.67",
    "gateway_reachable": true,
    "dns_resolution": true
  }
}
```

#### 階段 2: DHCP/DNS 檢查
```json
// 2. 檢查 DHCP 配置
{
  "id": "cmd-dhcp-check-001", 
  "op": "dhcpdns.get_config",
  "schema": "cmd.dhcpdns.get_config/1.0",
  "trace": {
    "session_id": "diag-session-001",
    "trace_id": "dhcp-check-step-02"
  }
}

// 3. 掃描 Rogue DHCP
{
  "id": "cmd-rogue-scan-001",
  "op": "dhcpdns.scan_rogue", 
  "schema": "cmd.dhcpdns.scan_rogue/1.0",
  "trace": {
    "session_id": "diag-session-001",
    "trace_id": "rogue-scan-step-03"
  }
}

// 發現問題：存在 Rogue DHCP
{
  "id": "cmd-rogue-scan-001",
  "status": "completed", 
  "result": {
    "rogue_dhcp_detected": true,
    "rogue_servers": [
      {
        "ip": "192.168.1.200",
        "mac": "001122334455",
        "hostname": "old-router"
      }
    ]
  }
}
```

#### 階段 3: 問題修復
```json
// 4. 統一 DHCP 權威
{
  "id": "cmd-dhcp-fix-001",
  "op": "dhcpdns.set",
  "schema": "cmd.dhcpdns.set/1.0", 
  "args": {
    "disable_other_dhcp": true,
    "dhcp_range": "192.168.1.50-192.168.1.200",
    "dns_servers": ["1.1.1.1", "8.8.8.8"]
  },
  "trace": {
    "session_id": "diag-session-001",
    "trace_id": "dhcp-fix-step-04"
  }
}

// 5. 重啟網路服務
{
  "id": "cmd-service-restart-001",
  "op": "system.restart_service",
  "schema": "cmd.system.restart_service/1.0",
  "args": {
    "services": ["dhcp", "dns"]
  },
  "trace": {
    "session_id": "diag-session-001",
    "trace_id": "service-restart-step-05"
  }
}
```

#### 階段 4: 驗證修復
```json
// 6. 再次檢查客戶端連線
{
  "id": "cmd-client-recheck-001",
  "op": "clients.list",
  "schema": "cmd.clients.list/1.0",
  "trace": {
    "session_id": "diag-session-001", 
    "trace_id": "client-recheck-step-06"
  }
}

// 修復成功
{
  "id": "cmd-client-recheck-001",
  "status": "completed",
  "result": {
    "all_clients_connected": true,
    "internet_access": true,
    "dhcp_conflicts": false
  }
}
```

## 場景二：網速緩慢問題診斷

### 2.1 場景描述
用戶抱怨「付費 100M 但只跑到 20M」，需要分析速度瓶頸。

### 2.2 診斷流程

#### 階段 1: 基準測試
```json
// 1. 路由器端速度測試
{
  "id": "cmd-speed-baseline-001",
  "op": "network.speedtest_full",
  "schema": "cmd.network.speedtest_full/1.0",
  "args": {
    "server": "auto",
    "duration": 30,
    "direction": "both"
  },
  "timeout_ms": 60000,
  "trace": {
    "session_id": "speed-diag-001",
    "trace_id": "baseline-test-step-01"
  }
}

// 結果：路由器速度正常
{
  "id": "cmd-speed-baseline-001",
  "status": "completed",
  "result": {
    "download_mbps": 95.2,
    "upload_mbps": 48.1,
    "latency_ms": 12.3,
    "jitter_ms": 1.8,
    "server": "taipei-speedtest.example.com"
  }
}
```

#### 階段 2: 客戶端分析
```json
// 2. 檢查問題客戶端
{
  "id": "cmd-client-analysis-001",
  "op": "clients.list",
  "schema": "cmd.clients.list/1.0",
  "args": {
    "include_performance": true
  },
  "trace": {
    "session_id": "speed-diag-001",
    "trace_id": "client-analysis-step-02"
  }
}

// 發現 WiFi 客戶端速度差
{
  "id": "cmd-client-analysis-001", 
  "status": "completed",
  "result": {
    "clients": [
      {
        "id": "laptop-001",
        "connection": "wifi",
        "rssi": -65,
        "link_speed_mbps": 25,
        "band": "2.4ghz"
      }
    ]
  }
}
```

#### 階段 3: WiFi 環境檢查
```json
// 3. 掃描 WiFi 環境
{
  "id": "cmd-wifi-env-001",
  "op": "wifi.get_environment", 
  "schema": "cmd.wifi.get_environment/1.0",
  "trace": {
    "session_id": "speed-diag-001",
    "trace_id": "wifi-env-step-03"
  }
}

// 發現頻道干擾
{
  "id": "cmd-wifi-env-001",
  "status": "completed",
  "result": {
    "current_channel": 6,
    "interference_level": "high", 
    "recommended_channels": [1, 11],
    "neighboring_aps": 8
  }
}
```

#### 階段 4: 問題修復
```json
// 4. 切換到較佳頻道
{
  "id": "cmd-wifi-channel-001",
  "op": "wifi.set_channel",
  "schema": "cmd.wifi.set_channel/1.0",
  "args": {
    "band": "2.4ghz",
    "channel": 11
  },
  "trace": {
    "session_id": "speed-diag-001",
    "trace_id": "channel-fix-step-04"
  }
}

// 5. 引導客戶端到 5GHz
{
  "id": "cmd-client-steer-001",
  "op": "wifi.client_steer",
  "schema": "cmd.wifi.client_steer/1.0",
  "args": {
    "client_id": "laptop-001",
    "target_band": "5ghz",
    "method": "band_steering"
  },
  "trace": {
    "session_id": "speed-diag-001", 
    "trace_id": "client-steer-step-05"
  }
}
```

## 場景三：Mesh 網路部署與最佳化

### 3.1 場景描述
三層樓房需要部署 Mesh 網路，要求無縫漫遊和最佳覆蓋。

### 3.2 拓撲規劃
```
3F: Mesh Node C (mesh-node-c)
    │ (wireless backhaul)
2F: Mesh Node B (mesh-node-b) ── Router (gateway-001)
    │ (ethernet backhaul)       │ (wired connection)
1F: Smart Devices              Gateway
```

### 3.3 部署流程

#### 階段 1: 拓撲發現
```json
// 1. 掃描現有網路拓撲
{
  "id": "cmd-topo-scan-001",
  "op": "topology.discover",
  "schema": "cmd.topology.discover/1.0",
  "args": {
    "method": "comprehensive",
    "subnet": "192.168.1.0/24"
  }
}
```

#### 階段 2: Mesh 配置
```json
// 2. 設定 Mesh 回程
{
  "id": "cmd-mesh-config-001",
  "op": "mesh.set_backhaul",
  "schema": "cmd.mesh.set_backhaul/1.0",
  "args": {
    "nodes": [
      {
        "node_id": "mesh-node-b",
        "backhaul_type": "ethernet",
        "priority": "primary"
      },
      {
        "node_id": "mesh-node-c", 
        "backhaul_type": "wireless",
        "parent": "mesh-node-b",
        "band": "5ghz_dedicated"
      }
    ]
  }
}

// 3. 啟用無縫漫遊
{
  "id": "cmd-roaming-config-001",
  "op": "wifi.set_roaming",
  "schema": "cmd.wifi.set_roaming/1.0",
  "args": {
    "enable_80211r": true,
    "enable_80211k": true, 
    "enable_80211v": true,
    "rssi_threshold": -70,
    "load_balance": true
  }
}
```

#### 階段 3: 驗證與最佳化
```json
// 4. 測試回程性能
{
  "id": "cmd-backhaul-test-001",
  "op": "mesh.backhaul_test",
  "schema": "cmd.mesh.backhaul_test/1.0",
  "args": {
    "test_duration": 30,
    "include_latency": true
  }
}

// 5. 漫遊測試
{
  "id": "cmd-roam-test-001",
  "op": "wifi.roam_probe", 
  "schema": "cmd.wifi.roam_probe/1.0",
  "args": {
    "test_device": "phone-001",
    "test_path": ["mesh-node-b", "mesh-node-c"]
  }
}
```

## 場景四：IoT 設備大規模部署

### 4.1 場景描述
辦公室部署 100+ IoT 感測器，需要自動發現和管理。

### 4.2 設備類型
- 溫濕度感測器 (50 個)
- 煙霧偵測器 (30 個)
- 智慧插座 (20 個)
- 門禁感測器 (10 個)

### 4.3 批量部署流程

#### 階段 1: 設備註冊模式
```json
// 1. 啟用設備自動發現
{
  "id": "cmd-discovery-mode-001",
  "op": "device.enable_discovery",
  "schema": "cmd.device.enable_discovery/1.0",
  "args": {
    "duration_minutes": 60,
    "auto_provision": true,
    "device_types": ["sensor", "actuator"]
  }
}
```

#### 階段 2: 批量配置
```json
// 2. 批量 WiFi 配置
{
  "id": "cmd-batch-wifi-001",
  "op": "batch.execute",
  "schema": "cmd.batch.execute/1.0",
  "args": {
    "changeset_id": "iot-deployment-001",
    "commands": [
      {
        "op": "net.wifi.config",
        "target_pattern": "sensor-*",
        "args": {
          "ssid": "IoT-Network",
          "password": "secure-password", 
          "security": "wpa2",
          "band": "2.4ghz"
        }
      }
    ]
  }
}
```

#### 階段 3: 監控配置
```json
// 3. 設定監控政策
{
  "id": "cmd-monitoring-001",
  "op": "monitoring.set_policy",
  "schema": "cmd.monitoring.set_policy/1.0",
  "args": {
    "device_groups": [
      {
        "pattern": "temp-sensor-*",
        "telemetry_interval": 300,
        "heartbeat_interval": 3600
      },
      {
        "pattern": "smoke-detector-*", 
        "telemetry_interval": 60,
        "heartbeat_interval": 300
      }
    ]
  }
}
```

## 場景五：企業網路安全診斷

### 5.1 場景描述
企業網路出現異常流量，需要進行安全性分析。

### 5.2 安全診斷流程

#### 階段 1: 流量分析
```json
// 1. 分析高流量設備
{
  "id": "cmd-traffic-analysis-001",
  "op": "traffic.get_top_talkers",
  "schema": "cmd.traffic.get_top_talkers/1.0",
  "args": {
    "time_window": "1h",
    "top_n": 10,
    "include_protocols": true
  }
}

// 2. 檢查異常連線
{
  "id": "cmd-anomaly-check-001",
  "op": "security.detect_anomalies",
  "schema": "cmd.security.detect_anomalies/1.0",
  "args": {
    "analysis_types": ["traffic", "connection", "protocol"],
    "baseline_hours": 24
  }
}
```

#### 階段 2: 隔離措施
```json
// 3. 隔離可疑設備
{
  "id": "cmd-quarantine-001",
  "op": "security.quarantine_device",
  "schema": "cmd.security.quarantine_device/1.0",
  "args": {
    "device_id": "suspicious-device-001",
    "isolation_level": "network_only",
    "reason": "abnormal_traffic_pattern"
  }
}
```

## 場景六：韌體批量更新

### 6.1 場景描述
50 台路由器需要安全地批量更新韌體版本。

### 6.2 更新流程

#### 階段 1: 更新前檢查
```json
// 1. 檢查設備狀態
{
  "id": "cmd-pre-update-check-001",
  "op": "firmware.pre_update_check",
  "schema": "cmd.firmware.pre_update_check/1.0",
  "args": {
    "target_version": "v2.1.3",
    "check_compatibility": true,
    "check_storage": true
  }
}
```

#### 階段 2: 分批更新
```json
// 2. 第一批更新 (10%)
{
  "id": "cmd-firmware-batch1-001",
  "op": "batch.execute",
  "schema": "cmd.batch.execute/1.0",
  "args": {
    "changeset_id": "firmware-update-batch1",
    "target_devices": ["router-001", "router-002", "router-003"],
    "commands": [
      {
        "op": "firmware.update",
        "args": {
          "url": "https://updates.example.com/fw-v2.1.3.bin",
          "checksum": "sha256:abcd1234...",
          "auto_reboot": true,
          "rollback_timeout": 300
        }
      }
    ],
    "rollback_on_failure": true
  }
}
```

#### 階段 3: 驗證與繼續
```json
// 3. 驗證更新成功
{
  "id": "cmd-update-verify-001",
  "op": "firmware.verify_update",
  "schema": "cmd.firmware.verify_update/1.0",
  "args": {
    "expected_version": "v2.1.3",
    "health_check": true
  }
}
```

## 場景七：QoS 政策最佳化

### 7.1 場景描述
家庭網路有多種設備競爭頻寬，需要智能 QoS 管理。

### 7.2 QoS 配置

#### 階段 1: 流量分類
```json
// 1. 分析流量模式
{
  "id": "cmd-traffic-pattern-001",
  "op": "traffic.analyze_patterns",
  "schema": "cmd.traffic.analyze_patterns/1.0",
  "args": {
    "analysis_period": "7d",
    "classify_applications": true
  }
}
```

#### 階段 2: 政策套用
```json
// 2. 套用 QoS 政策
{
  "id": "cmd-qos-policy-001",
  "op": "qos.apply_policy",
  "schema": "cmd.qos.apply_policy/1.0",
  "args": {
    "policy_name": "smart_home_balanced",
    "rules": [
      {
        "category": "video_call",
        "priority": "high",
        "bandwidth_guarantee": "2mbps"
      },
      {
        "category": "gaming", 
        "priority": "high",
        "latency_target": "20ms"
      },
      {
        "category": "streaming",
        "priority": "medium",
        "bandwidth_limit": "50mbps"
      },
      {
        "category": "backup",
        "priority": "low",
        "bandwidth_limit": "10mbps"
      }
    ]
  }
}
```

## 故障排除常見問題

### Q1: 設備連線後立即斷線
**症狀**: 設備頻繁出現連線/斷線循環

**診斷步驟**:
1. 檢查 LWT 訊息模式
2. 分析 MQTT Keep-Alive 設定
3. 檢查網路穩定性

**解決方案**:
```json
{
  "op": "mqtt.adjust_keepalive",
  "args": {
    "keepalive_seconds": 60,
    "clean_session": false
  }
}
```

### Q2: 大量設備同時上線造成網路壅塞
**症狀**: 設備批量啟動時網路回應緩慢

**診斷步驟**:
1. 分析連線時序
2. 檢查 DHCP 池容量
3. 監控頻寬使用

**解決方案**:
```json
{
  "op": "device.stagger_connection",
  "args": {
    "batch_size": 10,
    "interval_seconds": 5
  }
}
```

### Q3: 診斷命令執行超時
**症狀**: 複雜診斷命令無法在預期時間完成

**解決方案**:
```json
{
  "timeout_ms": 120000,
  "priority": "high",
  "expect": "result"
}
```

## 最佳實踐總結

### 1. 部署前規劃
- 進行網路環境評估
- 制定設備命名規範
- 規劃 IP 位址分配
- 設計監控策略

### 2. 分階段部署
- 小規模試點測試
- 逐步擴展範圍
- 持續監控效能
- 及時調整配置

### 3. 監控與維護
- 建立健康檢查機制
- 定期進行效能分析
- 主動預防潛在問題
- 保持韌體版本更新

### 4. 安全性考量
- 使用強密碼和憑證
- 定期更新存取權限
- 監控異常行為
- 建立應急回應計畫

---

**下一步**: 閱讀 [實作指南](12-implementation-guide.md) 了解如何開發和部署 RTK MQTT 協議的具體實作步驟。
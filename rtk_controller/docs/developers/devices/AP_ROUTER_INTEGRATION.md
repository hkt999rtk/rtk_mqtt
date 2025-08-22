# AP/Router 設備整合指南

## 概述

本文檔提供接入點(Access Point)和路由器(Router)設備整合RTK MQTT協議的完整指南，包括必要功能實作、消息格式、命令處理和最佳實踐。

## 設備角色定義

### AP/Router 在網絡中的角色
- **網絡核心設備**: 提供網絡連接和管理服務
- **拓撲中心點**: 連接多個客戶端設備
- **數據聚合點**: 收集和轉發網絡診斷數據
- **控制執行點**: 執行網絡配置和管理命令

### 設備能力要求
- WiFi網絡管理
- 客戶端連接追蹤
- 網絡拓撲發現
- QoS流量管理
- 診斷數據收集
- 遠程配置支持

## 必要功能實作

### 1. 基礎連接管理

#### MQTT連接配置
```json
{
  "mqtt": {
    "broker_host": "mqtt.rtk.local",
    "broker_port": 1883,
    "client_id": "ap-{mac_address}",
    "username": "device_user",
    "password": "device_password",
    "keep_alive": 60,
    "clean_session": false,
    "ssl": {
      "enabled": true,
      "ca_cert": "/path/to/ca.crt",
      "client_cert": "/path/to/client.crt",
      "client_key": "/path/to/client.key"
    }
  }
}
```

#### 遺言消息設置
```json
{
  "lwt": {
    "topic": "rtk/v1/{tenant}/{site}/{device_id}/lwt",
    "payload": {
      "schema": "lwt/1.0",
      "ts": 1699123456789,
      "device_id": "aabbccddeeff",
      "payload": {
        "status": "offline",
        "last_seen": 1699123456789
      }
    },
    "qos": 1,
    "retain": true
  }
}
```

### 2. 狀態報告實作

#### 設備狀態消息 (每5分鐘)
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

**發送到**: `rtk/v1/{tenant}/{site}/{device_id}/state`

#### 設備屬性信息 (啟動時和變更時)
```json
{
  "schema": "attr/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "device_type": "router",
    "manufacturer": "Example Corp",
    "model": "EX-R1000",
    "firmware_version": "1.2.3",
    "hardware_version": "A1",
    "serial_number": "ABC123456789",
    "mac_address": "aabbccddeeff",
    "capabilities": [
      "wifi",
      "ethernet", 
      "mesh",
      "qos",
      "firewall",
      "dhcp"
    ],
    "wifi_capabilities": {
      "bands": ["2.4GHz", "5GHz"],
      "max_clients": 100,
      "mesh_support": true,
      "channels_2_4": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11],
      "channels_5": [36, 40, 44, 48, 149, 153, 157, 161, 165]
    }
  }
}
```

**發送到**: `rtk/v1/{tenant}/{site}/{device_id}/attr`

### 3. 遙測數據收集

#### WiFi遙測 (每30秒)
```json
{
  "schema": "telemetry.wifi/1.0", 
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "interfaces": [
      {
        "name": "wlan0",
        "band": "2.4GHz",
        "channel": 6,
        "channel_utilization": 65.2,
        "noise_floor": -95,
        "tx_power": 20,
        "connected_clients": 8,
        "throughput": {
          "rx_mbps": 12.5,
          "tx_mbps": 8.3
        }
      },
      {
        "name": "wlan1", 
        "band": "5GHz",
        "channel": 149,
        "channel_utilization": 35.8,
        "noise_floor": -92,
        "tx_power": 23,
        "connected_clients": 4,
        "throughput": {
          "rx_mbps": 45.2,
          "tx_mbps": 38.7
        }
      }
    ]
  }
}
```

**發送到**: `rtk/v1/{tenant}/{site}/{device_id}/telemetry/wifi`

#### 客戶端遙測 (每2分鐘)
```json
{
  "schema": "telemetry.clients/1.0",
  "ts": 1699123456789, 
  "device_id": "aabbccddeeff",
  "payload": {
    "total_clients": 12,
    "wifi_clients": 10,
    "ethernet_clients": 2,
    "clients": [
      {
        "mac": "112233445566",
        "ip": "192.168.1.105",
        "connection_type": "wifi_5ghz",
        "signal_strength": -45,
        "connected_time": 1800,
        "rx_bytes": 1048576,
        "tx_bytes": 524288,
        "device_type": "smartphone"
      }
    ]
  }
}
```

**發送到**: `rtk/v1/{tenant}/{site}/{device_id}/telemetry/clients`

## 命令處理實作

### 命令監聽設置
```python
def setup_command_listener(mqtt_client):
    topic = f"rtk/v1/{tenant}/{site}/{device_id}/cmd/req"
    mqtt_client.subscribe(topic, qos=1)
    mqtt_client.message_callback_add(topic, handle_command)
```

### 命令處理流程

#### 1. WiFi掃描命令 (wifi_scan)

**接收命令**:
```json
{
  "schema": "cmd.wifi_scan/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-wifi-scan-123",
    "op": "wifi_scan",
    "args": {
      "scan_type": "active",
      "duration": 10,
      "channels": [1, 6, 11, 36, 149],
      "include_hidden": true
    },
    "timeout_ms": 30000
  }
}
```

**實作示例** (Python):
```python
def handle_wifi_scan(command):
    # 發送ACK
    send_ack(command["id"])
    
    try:
        # 執行WiFi掃描
        scan_result = perform_wifi_scan(
            scan_type=command["args"]["scan_type"],
            duration=command["args"]["duration"],
            channels=command["args"].get("channels", []),
            include_hidden=command["args"].get("include_hidden", True)
        )
        
        # 發送結果
        result = {
            "schema": "cmd.wifi_scan.result/1.0",
            "ts": int(time.time() * 1000),
            "device_id": get_device_id(),
            "payload": {
                "id": command["payload"]["id"],
                "status": "completed",
                "result": scan_result
            }
        }
        
        send_result(result)
        
    except Exception as e:
        # 發送錯誤
        error_result = {
            "schema": "cmd.error/1.0",
            "ts": int(time.time() * 1000),
            "device_id": get_device_id(),
            "payload": {
                "id": command["payload"]["id"],
                "status": "failed",
                "error": {
                    "code": 1000,
                    "message": str(e)
                }
            }
        }
        send_result(error_result)
```

**返回結果**:
```json
{
  "schema": "cmd.wifi_scan.result/1.0",
  "ts": 1699123456820,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-wifi-scan-123",
    "status": "completed",
    "result": {
      "networks": [
        {
          "ssid": "NeighborNetwork",
          "bssid": "bbccddeeff00",
          "channel": 6,
          "frequency": 2437,
          "signal_strength": -65,
          "security": "WPA2-PSK",
        "bandwidth": "20MHz",
        "hidden": false
      }
    ],
    "scan_duration": 10,
    "channels_scanned": [1, 6, 11, 36, 149]
  }
}
```

#### 2. 客戶端列表查詢 (client_list)

**實作示例**:
```python
def handle_client_list(command):
    send_ack(command["id"])
    
    try:
        clients = get_connected_clients(
            include_offline=command["args"].get("include_offline", False),
            include_details=command["args"].get("include_details", True)
        )
        
        result = {
            "schema": "cmd.client_list.result/1.0",
            "ts": int(time.time() * 1000),
            "device_id": get_device_id(),
            "payload": {
                "id": command["payload"]["id"],
                "status": "completed",
                "result": {
                    "clients": clients,
                    "total_count": len(clients),
                    "online_count": sum(1 for c in clients if c["online"]),
                    "timestamp": int(time.time() * 1000)
                }
            }
        }
        
        send_result(result)
        
    except Exception as e:
        send_error(command["id"], 1000, str(e))
```

#### 3. QoS配置 (qos_config)

**接收命令**:
```json
{
  "schema": "cmd.qos_config/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-qos-123",
    "op": "qos_config",
    "args": {
      "policy": "gaming",
      "rules": [
        {
          "name": "high_priority_gaming",
          "match": {
            "protocol": "udp",
            "port_range": "3000-4000"
          },
          "priority": "high",
          "bandwidth_limit": null
        }
      ],
      "apply_immediately": true
    },
    "timeout_ms": 10000
  }
}
```

**實作流程**:
1. 驗證QoS規則格式
2. 檢查硬件支持能力
3. 備份當前配置
4. 應用新配置
5. 驗證配置生效
6. 返回結果

## 事件生成實作

### WiFi客戶端事件

#### 客戶端連接事件
```python
def on_client_connected(client_info):
    event = {
        "schema": "evt.network.client_connect/1.0",
        "ts": int(time.time() * 1000),
        "device_id": get_device_id(),
        "payload": {
            "client_mac": client_info["mac"],
            "ip_address": client_info["ip"],
            "connection_type": client_info["type"],
            "ssid": client_info["ssid"],
            "signal_strength": client_info["signal"]
        }
    }
    
    topic = f"rtk/v1/{tenant}/{site}/{device_id}/evt/network"
    publish_event(topic, event)
```

#### 漫遊事件
```python  
def on_client_roamed(client_mac, old_ap, new_ap, reason):
    event = {
        "schema": "evt.wifi.roam_triggered/1.0",
        "ts": int(time.time() * 1000),
        "device_id": get_device_id(),
        "payload": {
            "client_mac": client_mac,
            "from_ap": old_ap,
            "to_ap": new_ap,
            "reason": reason,
            "old_signal": get_signal_strength(client_mac, old_ap),
            "new_signal": get_signal_strength(client_mac, new_ap)
        }
    }
    
    topic = f"rtk/v1/{tenant}/{site}/{device_id}/evt/wifi"
    publish_event(topic, event)
```

## 支持的命令列表

### WiFi管理命令
- `wifi_scan` - WiFi頻道掃描
- `interference_analysis` - 干擾分析
- `wifi_scan_channels` - 綜合WiFi頻道掃描和頻譜分析
- `wifi_analyze_interference` - 先進干擾檢測和源識別
- `wifi_spectrum_utilization` - WiFi頻譜利用率分析
- `wifi_signal_strength_map` - WiFi信號強度熱圖生成
- `wifi_coverage_analysis` - 綜合WiFi覆蓋範圍分析
- `wifi_roaming_optimization` - WiFi漫遊行為分析和優化
- `wifi_throughput_analysis` - 先進WiFi吞吐量分析
- `wifi_latency_profiling` - 綜合WiFi延遲分析和抖動測量

### 網絡管理命令
- `client_list` - 客戶端列表查詢
- `topology_update` - 網絡拓撲更新
- `topology_get_full` - 獲取完整網絡拓撲
- `speed_test` - 網絡速度測試
- `wan_connectivity` - WAN連接診斷
- `network_test` - 網絡測試

### QoS管理命令
- `qos_config` - QoS策略配置
- `qos_get_status` - 獲取當前QoS狀態
- `traffic_get_stats` - 獲取詳細流量統計
- `traffic_stats` - 流量統計查詢

### 系統管理命令
- `restart` - 設備重啟
- `reboot` - 設備重新啟動
- `restart_service` - 服務重新啟動
- `get_system_info` - 獲取系統信息
- `update_config` - 更新配置
- `run_diagnostics` - 運行診斷
- `set_log_level` - 設置日誌級別
- `get_logs` - 獲取日誌
- `cancel_command` - 取消命令
- `device.status` - 設備狀態查詢
- `firmware_update` - 固件更新

### 配置管理命令
- `config_update` - 配置更新
- `config_wifi_settings` - WiFi配置管理
- `config_qos_policies` - QoS策略管理
- `config_security_settings` - 安全配置管理
- `config_band_steering` - 智能頻段引導配置
- `config_auto_optimize` - 自動配置優化
- `config_validate_changes` - 配置變更驗證
- `config_rollback_safe` - 安全配置回滾
- `config_impact_analysis` - 配置變更影響分析

## 實作檢查清單

### 基本功能 ✓
- [ ] MQTT連接和重連機制
- [ ] 遺言消息配置
- [ ] SSL/TLS加密支持
- [ ] 設備標識符生成

### 消息發送 ✓
- [ ] 狀態消息定期發送 (5分鐘間隔)
- [ ] 屬性消息發送 (啟動時)
- [ ] 遙測數據收集和發送
- [ ] 事件即時通知

### 命令處理 ✓
- [ ] 命令監聽和解析
- [ ] ACK消息發送
- [ ] 命令執行和錯誤處理
- [ ] 結果消息返回
- [ ] 超時處理

### WiFi功能 ✓
- [ ] WiFi掃描功能
- [ ] 客戶端連接追蹤
- [ ] 信號強度監控
- [ ] 漫遊事件檢測
- [ ] 干擾分析能力

### 網絡功能 ✓
- [ ] 拓撲發現
- [ ] 速度測試
- [ ] WAN連接檢測
- [ ] 客戶端管理
- [ ] 流量統計

### QoS功能 ✓
- [ ] QoS策略配置
- [ ] 流量分類
- [ ] 頻寬控制
- [ ] 優先級管理

### 錯誤處理 ✓
- [ ] 命令格式驗證
- [ ] 錯誤代碼標準化
- [ ] 詳細錯誤信息
- [ ] 優雅降級處理

### 安全性 ✓
- [ ] 認證機制
- [ ] 加密通信
- [ ] 輸入驗證
- [ ] 權限檢查

## 性能優化建議

### 消息頻率控制
- 狀態消息: 5分鐘間隔
- WiFi遙測: 30秒間隔  
- 客戶端遙測: 2分鐘間隔
- 事件消息: 實時發送

### 數據壓縮
- 大型數據使用gzip壓縮
- 批量發送小數據
- 避免重複發送相同數據

### 連接管理
- 實現指數退避重連
- 使用持久會話
- 適當的Keep-Alive設置

## 故障排除

### 常見問題

#### 1. MQTT連接失敗
- 檢查網絡連接
- 驗證認證信息
- 確認防火牆設置
- 檢查SSL證書

#### 2. 命令執行超時
- 調整超時設置
- 檢查設備性能
- 優化命令執行邏輯
- 添加進度報告

#### 3. 數據丟失
- 增加QoS級別
- 檢查消息大小限制
- 驗證主題權限
- 監控網絡穩定性

### 調試工具
- MQTT客戶端工具 (MQTT Explorer)
- 網絡抓包 (Wireshark)
- 系統日誌監控
- 性能分析工具

## 相關文檔

- [MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md) - 完整協議規範
- [Commands Reference](../core/COMMANDS_EVENTS_REFERENCE.md) - 命令參考
- [Schema Reference](../core/SCHEMA_REFERENCE.md) - 消息格式定義
- [Quick Start Guide](../guides/QUICK_START_GUIDE.md) - 快速開始指南
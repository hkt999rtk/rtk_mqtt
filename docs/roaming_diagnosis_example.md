# 漫遊問題診斷完整 MQTT 範例

## 情境描述
設備 `office-ap-001` 在辦公室環境中，RSSI 降到 -75dBm 並持續 10 秒，但系統未觸發漫遊機制。

## MQTT 訊息流程

### 1. 事件觸發 (Device → Controller)

**Topic**: `rtk/v1/office/floor1/office-ap-001/evt/wifi.roam_miss`  
**Retained**: false

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
      "2g": {
        "bssid": "00:00:00:00:00:00",
        "rssi": 0,
        "channel": 0
      },
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

### 2. Controller 請求詳細診斷 (Controller → Device)

**Topic**: `rtk/v1/office/floor1/office-ap-001/cmd/req`

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

### 3. Device 命令確認 (Device → Controller)

**Topic**: `rtk/v1/office/floor1/office-ap-001/cmd/ack`

```json
{
  "id": "roam-diag-001",
  "ts": 1723526402200,
  "accepted": true,
  "err": null
}
```

### 4. Device 回傳詳細診斷結果 (Device → Controller)

**Topic**: `rtk/v1/office/floor1/office-ap-001/cmd/res`

```json
{
  "id": "roam-diag-001", 
  "ts": 1723526403500,
  "ok": true,
  "result": {
    "diagnosis_type": "wifi.roaming",
    "device_type": "wifi_access_point",
    "collection_time": 1723526403000,
    "data": {
      "roaming_analysis": {
        "trigger_reasons": [
          "poor_signal_quality",
          "scan_skip_detected"
        ],
        "skip_analysis": {
          "total_skips_10sec": 3,
          "skip_reasons": ["scan_in_progress", "channel_switch_delay"],
          "last_successful_scan": 1723526395000,
          "scan_interval_ms": 5000,
          "expected_scan_count": 2,
          "actual_scan_count": 0
        },
        "environmental_scan": {
          "total_ap_detected": 8,
          "same_ssid_ap_count": 3,
          "better_rssi_ap_count": 2,
          "roam_candidate_count": 2
        }
      },
      "current_connection": {
        "bssid": "aa:bb:cc:dd:ee:ff",
        "ssid": "OfficeWiFi-5G",
        "rssi": -75,
        "channel": 149,
        "bandwidth": "80MHz",
        "band": "5G",
        "connection_duration_ms": 1847500,
        "last_roam_time": 1723524553500
      },
      "roam_candidates": [
        {
          "band": "5G",
          "bssid": "11:22:33:44:55:66",
          "ssid": "OfficeWiFi-5G", 
          "rssi": -42,
          "channel": 36,
          "bandwidth": "80MHz",
          "estimated_throughput_mbps": 450,
          "load_percentage": 25,
          "roam_score": 85
        },
        {
          "band": "6G",
          "bssid": "77:88:99:aa:bb:cc",
          "ssid": "OfficeWiFi-6G",
          "rssi": -48,
          "channel": 37,
          "bandwidth": "160MHz", 
          "estimated_throughput_mbps": 980,
          "load_percentage": 15,
          "roam_score": 92
        }
      ],
      "rf_statistics": {
        "interference_level": "moderate",
        "noise_floor_dbm": -95,
        "channel_utilization_percent": 45,
        "retry_rate_percent": 12.5,
        "packet_loss_rate_percent": 2.1
      },
      "scan_history": [
        {
          "timestamp": 1723526395000,
          "duration_ms": 850,
          "channels_scanned": [36, 40, 44, 48, 149, 153, 157, 161],
          "ap_found": 8,
          "trigger_reason": "periodic_background"
        },
        {
          "timestamp": 1723526380000,
          "duration_ms": 1200,
          "channels_scanned": [36, 40, 44, 48, 149, 153, 157, 161],
          "ap_found": 7,
          "trigger_reason": "roam_evaluation"
        }
      ]
    }
  },
  "err": null
}
```

### 5. 狀態更新 (Device → Controller) - 定期發送

**Topic**: `rtk/v1/office/floor1/office-ap-001/state`  
**Retained**: true

```json
{
  "schema": "state/1.0",
  "ts": 1723526404000,
  "health": "warn",
  "fw": "1.2.3",
  "uptime_s": 4567,
  "cpu_usage": 35.2,
  "memory_usage": 42.8,
  "temperature_c": 38.5,
  "wifi_stats": {
    "current_rssi": -75,
    "connection_quality": "poor",
    "roaming_enabled": true,
    "last_roam_attempt": 0,
    "roam_candidate_count": 2
  },
  "net": {
    "interface": "wlan0",
    "ip": "10.0.1.23",
    "bytes_rx": 1048576,
    "bytes_tx": 524288,
    "packets_dropped": 127
  },
  "diagnosis": {
    "last_event": "wifi.roam_miss",
    "last_event_time": 1723526401000,
    "error_count": 1,
    "warning_count": 3
  }
}
```

## 診斷分析結果

基於這次診斷資料，可以得出以下結論：

### 根本原因分析
1. **掃描跳過問題**: 10秒內有3次掃描被跳過，導致無法及時發現更好的AP
2. **掃描觸發延遲**: 上次完整掃描距離事件發生已經21秒
3. **候選AP可用**: 環境中有2個更好的AP可供漫遊

### 建議動作
1. **立即動作**: 觸發一次完整的環境掃描
2. **配置調整**: 降低掃描間隔或調整掃描跳過條件
3. **持續監控**: 關注後續5分鐘內的漫遊行為

### Controller 後續動作建議

**觸發環境掃描命令**:
```json
{
  "id": "emergency-scan-001",
  "op": "wifi.scan",
  "schema": "cmd.wifi.scan/1.0",
  "args": {
    "scan_type": "full_active",
    "channels": [36, 40, 44, 48, 149, 153, 157, 161],
    "dwell_time_ms": 100,
    "priority": "high"
  },
  "timeout_ms": 10000,
  "expect": "result",
  "ts": 1723526405000
}
```
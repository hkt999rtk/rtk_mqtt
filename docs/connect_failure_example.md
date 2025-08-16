# WiFi 連線失敗診斷完整 MQTT 範例

![WiFi Connection Failure Diagnosis Sequence](connection_failure.png)

**詳細序列圖**：
![Connection Failure Sequence](connection_failure_sequence_simple.png)

## 情境描述
用戶設備 `laptop-005` 嘗試連接到企業 WiFi `CorporateNet-5G`，但在 WPA3 認證階段失敗。

## MQTT 訊息流程

### 1. 連線失敗事件觸發 (Device → Controller)

**Topic**: `rtk/v1/corporate/building-a/laptop-005/evt/wifi.connect_fail`  
**Retained**: false

```json
{
  "schema": "evt.wifi.connect_fail/1.0",
  "ts": 1723526501000,
  "severity": "error",
  "connection_info": {
    "role_type": "STA",
    "target_bssid": "cc:dd:ee:ff:00:11",
    "ssid": "CorporateNet-5G",
    "security_type": "WPA3_SAE"
  },
  "failure_details": {
    "join_status": "AUTH_TIMEOUT", 
    "join_rx_count": 18,
    "tx_fail_category": "AUTH",
    "auth_mode": "WPA3SAE",
    "auth_algo": "SAE",
    "four_way_status": 0,
    "response_status_code": 0,
    "failure_stage": "authentication"
  },
  "environment": {
    "rssi": -52,
    "channel": 44,
    "bandwidth": "80MHz",
    "interference_level": "low"
  }
}
```

### 2. Controller 請求詳細診斷 (Controller → Device)

**Topic**: `rtk/v1/corporate/building-a/laptop-005/cmd/req`

```json
{
  "id": "connect-diag-002",
  "op": "diagnosis.get",
  "schema": "cmd.diagnosis.get/1.0",
  "args": {
    "type": "wifi.connection",
    "detail_level": "full",
    "include_auth_details": true,
    "include_rf_analysis": true,
    "target_bssid": "cc:dd:ee:ff:00:11"
  },
  "timeout_ms": 20000,
  "expect": "result",
  "ts": 1723526502000
}
```

### 3. Device 命令確認 (Device → Controller)

**Topic**: `rtk/v1/corporate/building-a/laptop-005/cmd/ack`

```json
{
  "id": "connect-diag-002",
  "ts": 1723526502150,
  "accepted": true,
  "err": null
}
```

### 4. Device 回傳詳細連線失敗診斷 (Device → Controller)

**Topic**: `rtk/v1/corporate/building-a/laptop-005/cmd/res`

```json
{
  "id": "connect-diag-002",
  "ts": 1723526503800,
  "ok": true,
  "result": {
    "diagnosis_type": "wifi.connection",
    "device_type": "laptop_client",
    "collection_time": 1723526503000,
    "data": {
      "connection_attempt": {
        "start_time": 1723526498000,
        "end_time": 1723526501000,
        "total_duration_ms": 3000,
        "attempt_count": 1,
        "retry_count": 0
      },
      "target_ap_info": {
        "bssid": "cc:dd:ee:ff:00:11",
        "ssid": "CorporateNet-5G",
        "rssi": -52,
        "channel": 44,
        "bandwidth": "80MHz",
        "band": "5G",
        "security": {
          "type": "WPA3_SAE",
          "akm_suite": "SAE_SHA256",
          "pairwise_cipher": "CCMP_128",
          "group_cipher": "CCMP_128",
          "mgmt_group_cipher": "BIP_CMAC_128"
        },
        "capabilities": {
          "ht_supported": true,
          "vht_supported": true,
          "he_supported": true,
          "mfp_capable": true,
          "mfp_required": true
        }
      },
      "connection_stages": {
        "beacon_detection": {
          "status": "success",
          "duration_ms": 200,
          "beacons_received": 3,
          "probe_responses_received": 2
        },
        "authentication": {
          "status": "timeout",
          "duration_ms": 2500,
          "auth_algorithm": "SAE",
          "auth_transactions": [
            {
              "sequence": 1,
              "timestamp": 1723526498500,
              "direction": "tx", 
              "status": "sent",
              "tx_attempts": 3,
              "ack_received": true
            },
            {
              "sequence": 2,
              "timestamp": 0,
              "direction": "rx",
              "status": "timeout",
              "expected": true,
              "received": false
            }
          ],
          "sae_exchange": {
            "commit_sent": true,
            "commit_response_received": false,
            "confirm_sent": false,
            "confirm_response_received": false,
            "pwd_id": null,
            "rejected_groups": []
          }
        },
        "association": {
          "status": "not_attempted",
          "reason": "authentication_failed"
        },
        "four_way_handshake": {
          "status": "not_attempted",
          "reason": "authentication_failed"
        }
      },
      "failure_analysis": {
        "primary_cause": "sae_timeout",
        "contributing_factors": [
          "ap_sae_processing_delay",
          "possible_ap_overload"
        ],
        "recommendations": [
          "retry_with_different_ap",
          "check_ap_load_balance",
          "verify_sae_configuration"
        ]
      },
      "rf_environment": {
        "channel_info": {
          "primary_channel": 44,
          "secondary_channel": 48,
          "center_frequency": 5230,
          "bandwidth": "80MHz"
        },
        "signal_quality": {
          "rssi": -52,
          "snr": 28,
          "noise_floor": -95,
          "signal_strength": "good"
        },
        "interference": {
          "level": "low",
          "sources": ["microwave"],
          "channel_utilization": 25,
          "adjacent_channel_interference": false
        }
      },
      "device_state": {
        "wifi_driver_version": "22.140.1",
        "firmware_version": "ucode-36.177a0a2c",
        "regulatory_domain": "TW",
        "power_management": "disabled",
        "roaming_enabled": true,
        "sae_h2e_enabled": true,
        "pmf_enabled": true
      },
      "network_scan_results": [
        {
          "bssid": "cc:dd:ee:ff:00:11",
          "ssid": "CorporateNet-5G",
          "rssi": -52,
          "channel": 44,
          "security": "WPA3_SAE",
          "load": "medium"
        },
        {
          "bssid": "cc:dd:ee:ff:00:22", 
          "ssid": "CorporateNet-5G",
          "rssi": -58,
          "channel": 48,
          "security": "WPA3_SAE",
          "load": "low"
        },
        {
          "bssid": "cc:dd:ee:ff:00:33",
          "ssid": "CorporateNet-2G",
          "rssi": -45,
          "channel": 6,
          "security": "WPA2_PSK",
          "load": "high"
        }
      ]
    }
  },
  "err": null
}
```

### 5. 狀態更新反映連線問題 (Device → Controller)

**Topic**: `rtk/v1/corporate/building-a/laptop-005/state`  
**Retained**: true

```json
{
  "schema": "state/1.0",
  "ts": 1723526504000,
  "health": "error",
  "fw": "ucode-36.177a0a2c",
  "uptime_s": 8923,
  "cpu_usage": 15.3,
  "memory_usage": 68.4,
  "temperature_c": 45.2,
  "wifi_stats": {
    "connection_status": "disconnected",
    "last_connected_ssid": "HomeWiFi",
    "last_disconnect_time": 1723523891000,
    "connection_attempts": 1,
    "connection_failures": 1,
    "last_failure_reason": "auth_timeout"
  },
  "net": {
    "interface": "wlan0",
    "ip": null,
    "connection_type": "wifi",
    "connected": false
  },
  "diagnosis": {
    "last_event": "wifi.connect_fail",
    "last_event_time": 1723526501000,
    "error_count": 1,
    "warning_count": 0,
    "connectivity_status": "no_internet"
  }
}
```

## 診斷分析結果

### 失敗原因分析
1. **SAE 認證超時**: AP 未在預期時間內回應 SAE commit 訊息
2. **AP 負載問題**: 可能的 AP 處理延遲或過載
3. **配置相容性**: SAE 參數設定可能存在不相容

### 建議修復動作

#### 立即動作
1. **嘗試連線備用 AP**:
```json
{
  "id": "auto-retry-001",
  "op": "wifi.connect",
  "schema": "cmd.wifi.connect/1.0",
  "args": {
    "ssid": "CorporateNet-5G",
    "bssid": "cc:dd:ee:ff:00:22",
    "security": "WPA3_SAE",
    "priority": "high",
    "timeout_ms": 30000
  }
}
```

#### 長期解決方案
1. **SAE 參數調整**: 檢查 H2E (Hash-to-Element) 設定
2. **AP 負載平衡**: 建議 IT 部門檢查 AP 負載分佈
3. **韌體更新**: 確認裝置與 AP 韌體相容性

### Controller 監控建議

**設定連線重試策略**:
```json
{
  "id": "retry-policy-001",
  "op": "policy.set",
  "schema": "cmd.policy.set/1.0",
  "args": {
    "policy_type": "connection_retry",
    "max_retries": 3,
    "retry_interval_ms": 5000,
    "fallback_security": "WPA2_PSK",
    "prefer_less_loaded_ap": true
  }
}
```
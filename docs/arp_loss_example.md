# ARP 遺失診斷完整 MQTT 範例

![ARP Loss Diagnosis Sequence](arp_loss.png)

**詳細序列圖**：
![ARP Loss Sequence](arp_loss_sequence_simple.png)

## 情境描述
IoT 設備 `smart-camera-003` 在監控網路環境中，連續2次 ARP response 未收到，可能導致網路連線中斷。

## MQTT 訊息流程

### 1. ARP 遺失事件觸發 (Device → Controller)

**Topic**: `rtk/v1/factory/workshop/smart-camera-003/evt/wifi.arp_loss`  
**Retained**: false

```json
{
  "schema": "evt.wifi.arp_loss/1.0",
  "ts": 1723526601000,
  "severity": "warning",
  "trigger_info": {
    "consecutive_loss_count": 2,
    "cooldown_ms": 300000,
    "detection_window_ms": 15000
  },
  "network_info": {
    "current_bssid": "bb:cc:dd:ee:ff:00",
    "current_rssi": -67,
    "channel_hw_match": true,
    "scan_in_progress": false,
    "link_quality": "degraded"
  },
  "arp_statistics": {
    "source_ip": "192.168.10.203",
    "destination_ip": "192.168.10.1",
    "req_tx_fail_count": 2,
    "req_count": 5,
    "rsp_count": 3,
    "success_rate": 0.6
  },
  "immediate_indicators": {
    "packet_loss_detected": true,
    "latency_increase": true,
    "throughput_degraded": false
  }
}
```

### 2. Controller 請求詳細網路診斷 (Controller → Device)

**Topic**: `rtk/v1/factory/workshop/smart-camera-003/cmd/req`

```json
{
  "id": "arp-diag-003",
  "op": "diagnosis.get",
  "schema": "cmd.diagnosis.get/1.0",
  "args": {
    "type": "wifi.network",
    "detail_level": "full",
    "include_rf_stats": true,
    "include_traffic_analysis": true,
    "include_interference_scan": true,
    "analysis_window_sec": 60
  },
  "timeout_ms": 25000,
  "expect": "result", 
  "ts": 1723526602000
}
```

### 3. Device 命令確認 (Device → Controller)

**Topic**: `rtk/v1/factory/workshop/smart-camera-003/cmd/ack`

```json
{
  "id": "arp-diag-003",
  "ts": 1723526602300,
  "accepted": true,
  "estimated_completion_ms": 20000,
  "err": null
}
```

### 4. Device 回傳網路診斷分析 (Device → Controller)

**Topic**: `rtk/v1/factory/workshop/smart-camera-003/cmd/res`

```json
{
  "id": "arp-diag-003",
  "ts": 1723526606500,
  "ok": true,
  "result": {
    "diagnosis_type": "wifi.network",
    "device_type": "smart_camera",
    "collection_time": 1723526606000,
    "analysis_window": {
      "start_time": 1723526546000,
      "end_time": 1723526606000,
      "duration_sec": 60
    },
    "data": {
      "network_connectivity": {
        "gateway_ip": "192.168.10.1",
        "dns_servers": ["192.168.10.10", "8.8.8.8"],
        "dhcp_server": "192.168.10.1",
        "subnet_mask": "255.255.255.0",
        "connection_uptime_sec": 28947
      },
      "arp_analysis": {
        "target_analysis": {
          "ip": "192.168.10.1", 
          "mac": "aa:bb:cc:dd:ee:01",
          "role": "gateway",
          "reachability": "intermittent"
        },
        "recent_requests": [
          {
            "timestamp": 1723526598000,
            "target_ip": "192.168.10.1",
            "tx_attempts": 3,
            "response_received": true,
            "response_time_ms": 12
          },
          {
            "timestamp": 1723526595000,
            "target_ip": "192.168.10.1",
            "tx_attempts": 3,
            "response_received": false,
            "timeout_ms": 3000
          },
          {
            "timestamp": 1723526592000,
            "target_ip": "192.168.10.1",
            "tx_attempts": 3,
            "response_received": false,
            "timeout_ms": 3000
          }
        ],
        "statistics": {
          "total_requests": 15,
          "successful_responses": 10,
          "failed_requests": 5,
          "success_rate": 0.67,
          "average_response_time_ms": 18.5
        }
      },
      "ap_connection_info": {
        "bssid": "bb:cc:dd:ee:ff:00",
        "ssid": "Factory-IoT-5G",
        "channel": 149,
        "bandwidth": "80MHz",
        "band": "5G",
        "rssi": -67,
        "link_speed_mbps": 200,
        "connection_duration_sec": 28947
      },
      "rf_diagnostics": {
        "signal_quality": {
          "rssi": -67,
          "snr": 18,
          "noise_floor": -95,
          "signal_strength": "fair",
          "trend": "declining"
        },
        "channel_analysis": {
          "primary_channel": 149,
          "center_frequency": 5745,
          "channel_utilization": 65,
          "interference_sources": [
            {
              "type": "bluetooth",
              "strength": "weak",
              "frequency_overlap": false
            },
            {
              "type": "industrial_equipment", 
              "strength": "moderate",
              "frequency_overlap": true,
              "estimated_impact": "medium"
            }
          ]
        },
        "rf_calibration": {
          "last_calibration": 1723495200000,
          "calibration_status": "normal",
          "errors_detected": false,
          "next_calibration_due": 1723581600000
        }
      },
      "traffic_analysis": {
        "recent_5min": {
          "rx_bytes": 2457600,
          "tx_bytes": 1843200,
          "rx_packets": 1890,
          "tx_packets": 1432,
          "rx_errors": 23,
          "tx_errors": 8,
          "rx_dropped": 12,
          "tx_dropped": 3
        },
        "beacon_monitoring": {
          "beacons_expected": 300,
          "beacons_received": 287,
          "beacon_loss_rate": 0.043,
          "consecutive_missed_max": 3,
          "avg_beacon_interval_ms": 102.1
        },
        "packet_loss_analysis": {
          "overall_loss_rate": 0.021,
          "upstream_loss_rate": 0.018,
          "downstream_loss_rate": 0.024,
          "loss_pattern": "sporadic",
          "correlation_with_interference": true
        }
      },
      "baseband_statistics": {
        "rx_sensitivity": -85,
        "false_alarm_rate": 2.1,
        "cca_busy_time_percent": 35,
        "txop_utilization": 0.23,
        "retry_statistics": {
          "short_retries": 145,
          "long_retries": 28,
          "retry_rate": 0.12
        }
      },
      "environmental_factors": {
        "nearby_networks": [
          {
            "ssid": "Factory-Guest",
            "bssid": "bb:cc:dd:ee:ff:01",
            "channel": 153,
            "rssi": -72,
            "interference_potential": "low"
          },
          {
            "ssid": "Industrial-Control", 
            "bssid": "cc:dd:ee:ff:00:11",
            "channel": 149,
            "rssi": -58,
            "interference_potential": "high"
          }
        ],
        "channel_congestion": {
          "primary_channel_busy": 0.65,
          "secondary_channel_busy": 0.42,
          "recommendation": "consider_channel_switch"
        }
      }
    },
    "root_cause_analysis": {
      "primary_causes": [
        "channel_interference",
        "ap_load_fluctuation"
      ],
      "contributing_factors": [
        "industrial_equipment_interference",
        "co_channel_ap_competition",
        "declining_signal_strength"
      ],
      "probability_scores": {
        "rf_interference": 0.8,
        "ap_overload": 0.6,
        "hardware_issue": 0.2,
        "network_congestion": 0.7
      }
    },
    "recommendations": {
      "immediate_actions": [
        "monitor_signal_strength_trend",
        "consider_ap_roaming",
        "increase_arp_timeout"
      ],
      "short_term_fixes": [
        "request_channel_change",
        "optimize_ap_placement",
        "reduce_non_critical_traffic"
      ],
      "long_term_solutions": [
        "deploy_additional_ap",
        "implement_load_balancing",
        "upgrade_to_wifi6e"
      ]
    }
  },
  "err": null
}
```

### 5. 遙測資料持續監控 (Device → Controller)

**Topic**: `rtk/v1/factory/workshop/smart-camera-003/telemetry/wifi.connectivity`  
**Retained**: false

```json
{
  "schema": "telemetry.wifi.connectivity/1.0", 
  "ts": 1723526610000,
  "metrics": {
    "arp_success_rate": 0.72,
    "avg_response_time_ms": 15.3,
    "packet_loss_rate": 0.019,
    "rssi": -66,
    "link_quality": 68,
    "throughput_mbps": {
      "rx": 12.5,
      "tx": 8.3
    }
  },
  "trend_indicators": {
    "signal_improving": true,
    "connectivity_stabilizing": true,
    "interference_decreasing": false
  }
}
```

### 6. 狀態更新顯示網路品質改善 (Device → Controller)

**Topic**: `rtk/v1/factory/workshop/smart-camera-003/state`  
**Retained**: true

```json
{
  "schema": "state/1.0",
  "ts": 1723526611000,
  "health": "warn",
  "fw": "2.1.4",
  "uptime_s": 28954,
  "cpu_usage": 42.1,
  "memory_usage": 58.7,
  "temperature_c": 52.3,
  "wifi_stats": {
    "connection_status": "connected",
    "ssid": "Factory-IoT-5G",
    "rssi": -66,
    "link_quality": "fair",
    "packet_loss_rate": 0.019,
    "arp_success_rate": 0.72,
    "last_connectivity_issue": 1723526601000,
    "interference_level": "moderate"
  },
  "net": {
    "interface": "wlan0",
    "ip": "192.168.10.203",
    "gateway": "192.168.10.1",
    "dns": ["192.168.10.10", "8.8.8.8"],
    "bytes_rx": 48576000,
    "bytes_tx": 23456000,
    "connection_uptime_sec": 28947
  },
  "diagnosis": {
    "last_event": "wifi.arp_loss",
    "last_event_time": 1723526601000,
    "error_count": 0,
    "warning_count": 1,
    "network_stability": "improving"
  },
  "device_specific": {
    "camera_streaming": {
      "status": "active",
      "resolution": "1080p",
      "bitrate_kbps": 2500,
      "frames_dropped": 12,
      "stream_health": "good"
    }
  }
}
```

## 診斷分析結果

### 根本原因
1. **頻道干擾**: 工業設備在相同頻道產生干擾，導致封包遺失
2. **AP 負載波動**: 同頻道其他 AP 競爭造成的週期性干擾  
3. **訊號強度下降**: RSSI 從 -60dBm 降至 -67dBm

### 即時處理動作

**調整 ARP 參數**:
```json
{
  "id": "arp-tune-001",
  "op": "network.arp.config", 
  "schema": "cmd.network.arp.config/1.0",
  "args": {
    "timeout_ms": 5000,
    "retry_count": 5,
    "probe_interval_sec": 30,
    "adaptive_timeout": true
  }
}
```

**請求頻道切換評估**:
```json
{
  "id": "channel-eval-001",
  "op": "wifi.channel.evaluate",
  "schema": "cmd.wifi.channel.evaluate/1.0", 
  "args": {
    "current_channel": 149,
    "candidate_channels": [36, 40, 44, 48, 157, 161],
    "evaluation_duration_sec": 30,
    "include_interference_scan": true
  }
}
```

### 持續監控策略

**加強遙測頻率**:
```json
{
  "id": "monitor-enhance-001",
  "op": "telemetry.schedule",
  "schema": "cmd.telemetry.schedule/1.0",
  "args": {
    "metrics": ["wifi.connectivity", "wifi.interference"],
    "interval_sec": 10,
    "duration_sec": 1800,
    "alert_thresholds": {
      "arp_success_rate": 0.8,
      "packet_loss_rate": 0.05,
      "rssi_threshold": -70
    }
  }
}
```
# 下行命令結構 (Controller → Device)

## 概述

下行命令是控制器向設備發送的操作指令，包括配置變更、診斷請求、系統控制等。RTK MQTT 協議採用統一的三段式命令執行流程：請求 → 確認 → 結果。

## 命令執行流程

```
Controller                    Device
    |                           |
    |--- cmd/req -------------->|  1. 發送命令請求
    |                           |
    |<-- cmd/ack ---------------|  2. 立即確認接收 (< 1秒)
    |                           |
    |                           |  3. 執行命令
    |                           |
    |<-- cmd/res ---------------|  4. 返回執行結果
    |                           |
```

## Topic 結構總覽

| 階段 | Topic 格式 | 方向 | QoS | 用途 |
|------|------------|------|-----|------|
| 請求 | `rtk/v1/{tenant}/{site}/{device_id}/cmd/req` | Controller → Device | 1 | 命令請求 |
| 確認 | `rtk/v1/{tenant}/{site}/{device_id}/cmd/ack` | Device → Controller | 1 | 接收確認 |
| 結果 | `rtk/v1/{tenant}/{site}/{device_id}/cmd/res` | Device → Controller | 1 | 執行結果 |

## cmd/req (命令請求)

### 基本結構

```json
{
  "schema": "cmd.<operation>/1.0",
  "ts": 1699123456789,
  "id": "cmd-20241104-001",
  "op": "device.reboot",
  "args": {
    // 命令特定參數
  },
  "timeout_ms": 30000,
  "expect": "result",
  "priority": "normal",
  "trace": {
    "session_id": "llm-session-001",
    "trace_id": "diagnostic-step-05"
  }
}
```

### 核心欄位說明

| 欄位 | 型態 | 必要 | 說明 | 範例 |
|------|------|------|------|------|
| `id` | string | ✅ | 命令唯一識別碼 | `"cmd-20241104-001"` |
| `op` | string | ✅ | 命令操作名稱 | `"device.reboot"` |
| `args` | object | 可選 | 命令參數 | `{"delay_s": 5}` |
| `timeout_ms` | integer | 可選 | 執行超時時間 | `30000` |
| `expect` | string | 可選 | 期望回應類型 | `"ack"`, `"result"`, `"none"` |
| `priority` | string | 可選 | 命令優先級 | `"low"`, `"normal"`, `"high"`, `"urgent"` |

### 命令分類

#### 系統控制命令
```json
// 設備重啟
{
  "id": "cmd-reboot-001",
  "op": "device.reboot",
  "schema": "cmd.device.reboot/1.0",
  "args": {
    "delay_s": 5,
    "reason": "maintenance"
  }
}

// 韌體更新
{
  "id": "cmd-firmware-001", 
  "op": "firmware.update",
  "schema": "cmd.firmware.update/1.0",
  "args": {
    "url": "https://updates.example.com/fw-v1.2.3.bin",
    "checksum": "sha256:abc123...",
    "auto_reboot": true
  }
}
```

#### 網路配置命令
```json
// WiFi 配置
{
  "id": "cmd-wifi-001",
  "op": "net.wifi.config", 
  "schema": "cmd.net.wifi.config/1.0",
  "args": {
    "ssid": "new-network",
    "password": "new-password",
    "security": "wpa2",
    "channel": "auto"
  }
}

// IP 配置
{
  "id": "cmd-ip-001",
  "op": "net.ip.config",
  "schema": "cmd.net.ip.config/1.0", 
  "args": {
    "interface": "eth0",
    "mode": "static",
    "ip": "192.168.1.100",
    "netmask": "255.255.255.0",
    "gateway": "192.168.1.1"
  }
}
```

#### 診斷測試命令
```json
// 網路速度測試
{
  "id": "cmd-speedtest-001",
  "op": "diagnostics.speed_test",
  "schema": "cmd.diagnostics.speed_test/1.0",
  "args": {
    "server": "auto",
    "duration": 30,
    "direction": "both"
  },
  "timeout_ms": 60000
}

// 拓撲發現
{
  "id": "cmd-topology-001",
  "op": "topology.discover",
  "schema": "cmd.topology.discover/1.0",
  "args": {
    "method": "arp_scan",
    "subnet": "192.168.1.0/24",
    "timeout": 5000
  }
}
```

#### 設備控制命令
```json
// 燈光控制
{
  "id": "cmd-light-001",
  "op": "light.set",
  "schema": "cmd.light.set/1.0",
  "args": {
    "on": true,
    "brightness": 80,
    "color": {
      "r": 255,
      "g": 255, 
      "b": 255
    }
  }
}

// 電源控制
{
  "id": "cmd-power-001",
  "op": "power.control",
  "schema": "cmd.power.control/1.0",
  "args": {
    "outlet": 1,
    "state": "on",
    "schedule": {
      "enabled": true,
      "on_time": "08:00",
      "off_time": "18:00"
    }
  }
}
```

## cmd/ack (命令確認)

### 用途與時機
- **用途**: 設備確認已接收到命令請求
- **發送時機**: 收到 cmd/req 後立即發送 (< 1 秒)
- **重要性**: 讓控制器知道設備正常接收命令

### 確認訊息結構
```json
{
  "schema": "cmd.ack/1.0",
  "ts": 1699123456789,
  "id": "cmd-20241104-001",
  "accepted": true,
  "reason": null,
  "estimated_duration_ms": 5000
}
```

### 欄位說明

| 欄位 | 型態 | 必要 | 說明 |
|------|------|------|------|
| `id` | string | ✅ | 對應的命令 ID |
| `accepted` | boolean | ✅ | 是否接受執行命令 |
| `reason` | string | 可選 | 拒絕原因 (accepted=false 時) |
| `estimated_duration_ms` | integer | 可選 | 預估執行時間 |

### 確認狀態範例

#### 成功接受
```json
{
  "schema": "cmd.ack/1.0",
  "ts": 1699123456789,
  "id": "cmd-reboot-001",
  "accepted": true,
  "estimated_duration_ms": 5000
}
```

#### 拒絕執行
```json
{
  "schema": "cmd.ack/1.0", 
  "ts": 1699123456789,
  "id": "cmd-forbidden-001",
  "accepted": false,
  "reason": "insufficient_privileges",
  "error_code": "E403"
}
```

## cmd/res (命令結果)

### 用途與時機
- **用途**: 回報命令執行結果和相關資料
- **發送時機**: 命令執行完成後
- **內容**: 執行狀態、結果資料、錯誤資訊等

### 結果訊息結構
```json
{
  "schema": "cmd.<operation>.result/1.0",
  "ts": 1699123456789,
  "id": "cmd-20241104-001",
  "status": "completed",
  "result": {
    // 命令特定的結果資料
  },
  "execution": {
    "duration_ms": 2150,
    "steps_completed": 3,
    "steps_total": 3
  },
  "error": null
}
```

### 核心欄位說明

| 欄位 | 型態 | 必要 | 說明 |
|------|------|------|------|
| `id` | string | ✅ | 對應的命令 ID |
| `status` | string | ✅ | 執行狀態 |
| `result` | object | 可選 | 命令執行結果 |
| `execution` | object | 可選 | 執行過程資訊 |
| `error` | object | 可選 | 錯誤資訊 |

### 執行狀態類型

| 狀態 | 說明 | 下一步動作 |
|------|------|-----------|
| `completed` | 成功完成 | 無需動作 |
| `failed` | 執行失敗 | 檢查錯誤資訊 |
| `partial` | 部分成功 | 查看具體結果 |
| `timeout` | 執行超時 | 可考慮重試 |
| `cancelled` | 被取消 | 無需動作 |

### 結果範例

#### 成功執行 - 速度測試
```json
{
  "schema": "cmd.diagnostics.speed_test.result/1.0",
  "ts": 1699123456789,
  "id": "cmd-speedtest-001",
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
  "execution": {
    "duration_ms": 32150,
    "data_transferred_mb": 95.6
  }
}
```

#### 失敗執行 - 網路配置
```json
{
  "schema": "cmd.net.wifi.config.result/1.0",
  "ts": 1699123456789,
  "id": "cmd-wifi-001",
  "status": "failed",
  "result": null,
  "error": {
    "code": "E_WIFI_CONNECT_FAILED",
    "message": "Failed to connect to WiFi network",
    "details": {
      "ssid": "new-network",
      "reason": "authentication_failed",
      "signal_strength": -45,
      "attempts": 3
    }
  },
  "execution": {
    "duration_ms": 15000,
    "steps_completed": 2,
    "steps_total": 4
  }
}
```

#### 部分成功 - 拓撲發現
```json
{
  "schema": "cmd.topology.discover.result/1.0",
  "ts": 1699123456789,
  "id": "cmd-topology-001", 
  "status": "partial",
  "result": {
    "discovered_devices": [
      {
        "ip": "192.168.1.50",
        "mac": "112233445566",
        "hostname": "laptop-001",
        "response_time_ms": 2
      }
    ],
    "unreachable_ips": ["192.168.1.75", "192.168.1.100"],
    "scan_coverage": "95%"
  },
  "execution": {
    "duration_ms": 5000,
    "ips_scanned": 254,
    "responses_received": 24
  }
}
```

## 進階功能

### 變更集管理 (Changeset)
支援批量操作的原子性和回滾機制。

#### 變更集命令
```json
{
  "id": "cmd-changeset-001",
  "op": "net.wifi.config",
  "schema": "cmd.net.wifi.config/1.0",
  "args": {
    "ssid": "new-network",
    "changeset_id": "changeset-20241104-001",
    "changeset_timeout": 300
  }
}
```

#### 回滾命令
```json
{
  "id": "cmd-rollback-001",
  "op": "changeset.rollback",
  "schema": "cmd.changeset.rollback/1.0",
  "args": {
    "changeset_id": "changeset-20241104-001",
    "force": false
  }
}
```

### LLM 診斷追蹤
支援 AI 診斷系統的會話和步驟追蹤。

```json
{
  "id": "cmd-llm-diag-001",
  "op": "diagnostics.network_analysis",
  "schema": "cmd.diagnostics.network_analysis/1.0", 
  "trace": {
    "session_id": "llm-session-20241104-001",
    "trace_id": "network-analysis-step-03",
    "user_request": "檢查網路連線問題"
  },
  "args": {
    "analysis_type": "comprehensive",
    "include_topology": true
  }
}
```

## 錯誤處理

### 常見錯誤碼

| 錯誤碼 | 說明 | 處理建議 |
|--------|------|----------|
| `E_INVALID_COMMAND` | 無效命令 | 檢查命令格式 |
| `E_UNSUPPORTED_OPERATION` | 不支援的操作 | 檢查設備能力 |
| `E_INSUFFICIENT_PRIVILEGES` | 權限不足 | 檢查授權設定 |
| `E_DEVICE_BUSY` | 設備忙碌 | 稍後重試 |
| `E_TIMEOUT` | 執行超時 | 增加超時時間或分段執行 |
| `E_RESOURCE_UNAVAILABLE` | 資源不可用 | 檢查資源狀態 |

### 錯誤回應範例
```json
{
  "schema": "cmd.error/1.0",
  "ts": 1699123456789,
  "id": "cmd-invalid-001",
  "status": "failed",
  "error": {
    "code": "E_INVALID_COMMAND",
    "message": "Command operation 'invalid.op' is not supported",
    "details": {
      "received_op": "invalid.op",
      "supported_ops": ["device.reboot", "net.wifi.config", "diagnostics.speed_test"]
    }
  }
}
```

## 最佳實踐

### 1. 命令設計原則
- 使用語義化的操作名稱
- 保持參數結構的一致性  
- 提供合理的預設值
- 支援參數驗證

### 2. 超時時間設定
```json
{
  "diagnostics.speed_test": 60000,     // 速度測試 60 秒
  "firmware.update": 300000,           // 韌體更新 5 分鐘  
  "device.reboot": 30000,             // 設備重啟 30 秒
  "net.wifi.config": 15000,           // WiFi 配置 15 秒
  "topology.discover": 10000          // 拓撲發現 10 秒
}
```

### 3. 優先級管理
- `urgent`: 安全相關命令 (緊急停止、安全模式)
- `high`: 關鍵功能命令 (系統重啟、網路修復)
- `normal`: 一般操作命令 (配置變更、狀態查詢)
- `low`: 後台任務命令 (日誌收集、統計分析)

### 4. 命令批次化
對於需要多步驟執行的複雜操作，建議使用變更集機制：

```json
{
  "id": "cmd-batch-001",
  "op": "batch.execute",
  "schema": "cmd.batch.execute/1.0",
  "args": {
    "changeset_id": "maintenance-20241104",
    "commands": [
      {"op": "service.stop", "args": {"service": "web-server"}},
      {"op": "firmware.update", "args": {"url": "..."}},
      {"op": "device.reboot", "args": {"delay_s": 5}}
    ],
    "rollback_on_failure": true
  }
}
```

---

**下一步**: 閱讀 [診斷協議](10-diagnostics-protocol.md) 了解網路診斷和 LLM 整合的詳細規格。
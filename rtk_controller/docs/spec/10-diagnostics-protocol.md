# 診斷協議詳細規格

## 概述

RTK MQTT 協議整合了先進的 LLM（Large Language Model）自動化診斷系統，提供智能的網路問題分析和解決方案。本文檔詳細說明診斷協議的架構、工具分類、會話管理和典型診斷流程。

## LLM 診斷系統架構

### 三層架構設計

```
┌─────────────────┐    ┌───────────────────┐    ┌─────────────────┐
│   LLM Agent     │    │  RTK Controller   │    │   Network       │
│                 │    │                   │    │   Devices       │
│ ┌─────────────┐ │    │ ┌───────────────┐ │    │ ┌─────────────┐ │
│ │ Intent      │ │    │ │ MQTT Gateway  │ │    │ │ WiFi Router │ │
│ │ Recognition │ │    │ │ Device Mgmt   │ │    │ │ Access Point│ │
│ │             │ │    │ │ Session Mgmt  │ │    │ │ Smart Device│ │
│ └─────────────┘ │    │ └───────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌───────────────┐ │    │ ┌─────────────┐ │
│ │ Tool        │ │◄──►│ │ HTTP/gRPC API │ │◄──►│ │ MQTT Client │ │
│ │ Orchestration│ │    │ │ Tool Executor │ │    │ │ Protocol    │ │
│ └─────────────┘ │    │ └───────────────┘ │    │ │ Handler     │ │
└─────────────────┘    └───────────────────┘    │ └─────────────┘ │
                                               └─────────────────┘
```

### 架構層級說明

#### 1. LLM Agent Layer
- **意圖理解**: 解析用戶的自然語言描述
- **推理決策**: 基於設備狀態和問題症狀進行分析
- **工具編排**: 選擇和調用適當的診斷工具序列

#### 2. RTK Controller Layer
- **MQTT 閘道**: 轉換 HTTP/gRPC 請求為 MQTT 命令
- **設備管理**: 維護設備清單和能力資訊
- **會話管理**: 追蹤診斷會話的完整生命週期

#### 3. Device Layer
- **協議處理**: 實作 RTK MQTT 協議的設備端
- **診斷執行**: 執行具體的診斷和配置變更操作
- **狀態回報**: 實時回報設備狀態和診斷結果

## 工具分類系統

### Read 工具 (只讀操作)

**特性**:
- 不改變系統狀態
- 可並行執行
- 低風險操作

**常用工具**:

| 工具名稱 | 功能 | 回應時間 | 支援度 |
|---------|------|----------|--------|
| `topology.get_full` | 獲取完整網路拓撲圖 | 2-5s | 通用 |
| `wifi.get_environment` | 掃描週圍 WiFi 環境 | 3-8s | WiFi 設備 |
| `device.get_capabilities` | 查詢設備能力清單 | <1s | 通用 |
| `clients.list` | 列出連接的客戶端 | 1-3s | 網路設備 |
| `system.get_logs` | 獲取系統日誌 | 1-2s | 通用 |

**範例命令**:
```json
{
  "id": "cmd-read-001",
  "op": "wifi.get_environment",
  "schema": "cmd.wifi.get_environment/2.0",
  "trace": {
    "session_id": "llm-session-001",
    "trace_id": "wifi-scan-step-01"
  },
  "args": {
    "include_hidden": true,
    "scan_duration": 5000
  }
}
```

### Test 工具 (診斷測試)

**特性**:
- 執行測試但不永久改變配置
- 可能有短暫的系統影響
- 提供詳細的診斷資訊

**常用工具**:

| 工具名稱 | 功能 | 回應時間 | 資源消耗 |
|---------|------|----------|----------|
| `network.speedtest_full` | 完整網路速度測試 | 30-60s | 高頻寬 |
| `diagnostics.wan_connectivity` | WAN 連線診斷 | 5-10s | 低 |
| `diagnostics.latency_matrix` | 延遲矩陣測試 | 10-20s | 中等 |
| `mesh.backhaul_test` | Mesh 回程品質測試 | 15-30s | 高頻寬 |
| `dhcpdns.scan_rogue` | 掃描 Rogue DHCP | 10-15s | 中等 |

**範例命令**:
```json
{
  "id": "cmd-test-001",
  "op": "network.speedtest_full",
  "schema": "cmd.network.speedtest_full/1.0",
  "trace": {
    "session_id": "llm-session-001", 
    "trace_id": "speed-test-step-02"
  },
  "args": {
    "server": "auto",
    "duration": 30,
    "direction": "both",
    "parallel_streams": 4
  },
  "timeout_ms": 65000
}
```

### Act 工具 (配置變更)

**特性**:
- 會改變系統狀態
- 需要用戶明確授權
- 支援變更集管理和回滾

**安全機制**:
- 變更集 (Changeset) 管理
- 自動回滾機制
- 執行前確認
- 操作審計追蹤

**常用工具**:

| 工具名稱 | 功能 | 風險等級 | 回滾支援 |
|---------|------|----------|----------|
| `wifi.set_power` | RF 功率調整 | 中 | ✅ |
| `wifi.set_channel` | WiFi 頻道變更 | 中 | ✅ |
| `qos.apply_policy` | QoS 政策套用 | 低 | ✅ |
| `wifi.client_disconnect` | 強制客戶端重連 | 低 | ✅ |
| `system.restart_service` | 重啟系統服務 | 高 | ⚠️ |

**範例命令**:
```json
{
  "id": "cmd-act-001",
  "op": "wifi.set_channel",
  "schema": "cmd.wifi.set_channel/1.0",
  "trace": {
    "session_id": "llm-session-001",
    "trace_id": "channel-optimization-step-05"
  },
  "args": {
    "interface": "wlan0",
    "channel": 6,
    "changeset_id": "changeset-20241104-001",
    "auto_rollback_timeout": 300
  },
  "authorization": {
    "user_confirmed": true,
    "risk_acknowledged": true
  }
}
```

### 管理工具 (變更集管理)

**特性**:
- 支援批量操作
- 原子性執行
- 統一回滾機制

**常用工具**:

| 工具名稱 | 功能 | 用途 |
|---------|------|------|
| `changeset.create` | 建立變更集 | 開始批量操作 |
| `changeset.commit` | 提交變更集 | 確認變更生效 |
| `changeset.rollback` | 回滾變更集 | 撤銷所有變更 |
| `changeset.status` | 查詢變更集狀態 | 監控執行進度 |

## 會話管理

### 會話生命週期

```
創建會話 → 執行診斷 → 收集結果 → 提供建議 → 執行修復 → 完成會話
    ↓           ↓           ↓           ↓           ↓           ↓
session_id  trace_id_1  trace_id_2  trace_id_3  trace_id_4  session_end
```

### 會話結構

```json
{
  "session": {
    "session_id": "llm-diag-session-20241104-001",
    "created_at": 1699123456789,
    "user_request": "家裡 WiFi 很慢，尤其是晚上",
    "status": "active",
    "steps": [
      {
        "trace_id": "speed-baseline-step-01",
        "tool": "network.speedtest_full",
        "status": "completed",
        "started_at": 1699123456789,
        "completed_at": 1699123486789,
        "result": {
          "download_mbps": 15.2,
          "upload_mbps": 3.1,
          "expected_speed": 100
        }
      }
    ]
  }
}
```

### 追蹤範例

每個診斷步驟都包含追蹤資訊：

```json
{
  "trace": {
    "session_id": "llm-diag-session-20241104-001",
    "trace_id": "wifi-environment-scan-step-03",
    "parent_trace_id": "speed-analysis-step-02",
    "step_description": "掃描週圍 WiFi 環境以檢查頻道干擾"
  }
}
```

## 設備能力發現

### 能力宣告格式

設備通過 `attr` 訊息宣告其支援的診斷工具：

```json
{
  "schema": "attr/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "capabilities": {
    "device_type": "wifi_router",
    "llm_integration": {
      "supported": true,
      "version": "1.0"
    },
    "tools": {
      "wifi.get_environment": {
        "supported": true,
        "version": "2.0",
        "response_time_ms": 2000,
        "resource_usage": "low"
      },
      "network.speedtest_full": {
        "supported": true,
        "version": "1.0",
        "response_time_ms": 30000,
        "resource_usage": "high",
        "limitations": {
          "max_duration": 60,
          "concurrent_limit": 1
        }
      },
      "wifi.set_channel": {
        "supported": true,
        "version": "1.0",
        "permissions": ["config_change"],
        "rollback_supported": true
      }
    }
  }
}
```

### 動態能力調整

```json
{
  "capability_update": {
    "device_id": "aabbccddeeff",
    "timestamp": 1699123456789,
    "changes": [
      {
        "tool": "network.speedtest_full",
        "change_type": "availability",
        "old_value": true,
        "new_value": false,
        "reason": "high_cpu_load"
      }
    ]
  }
}
```

## 典型診斷流程

### A. 無網路連線問題 (no_internet)

**症狀**: "Wi-Fi 連上但不能上網"、"路由器沒有網路燈"

**診斷流程**:

1. **Read**: `diagnostics.wan_connectivity` - 檢查 WAN 連線狀態
```json
{
  "id": "cmd-no-internet-01",
  "op": "diagnostics.wan_connectivity",
  "trace": {"session_id": "session-001", "trace_id": "wan-check-01"}
}
```

2. **Read**: `dhcpdns.get_config` - 檢查 DHCP/DNS 配置
3. **Test**: `dhcpdns.scan_rogue` - 掃描 Rogue DHCP
4. **Read**: `clients.list` - 檢查是否全域問題
5. **Test**: `network.trace_route` - 追蹤到外網的路由
6. **Act**: `dhcpdns.set` - 統一 DHCP 伺服器（如發現衝突）

### B. 網速緩慢問題 (slow_speed)

**症狀**: "網速比付費方案慢很多"、"晚上特別慢"

**診斷流程**:

1. **Test**: `network.speedtest_full` - 基準速度測試
2. **Read**: `clients.list` - 檢查客戶端連線狀態
3. **Test**: `network.speedtest_full` - 客戶端速度對比
4. **Read**: `wifi.get_environment` - 檢查頻道干擾
5. **Read**: `traffic.get_top_talkers` - 找出高流量設備
6. **Act**: `wifi.set_channel` - 調整到較清淨頻道
7. **Act**: `qos.apply_policy` - 套用流量控制政策

### C. 不穩定連線問題 (unstable_disconnect)

**症狀**: "Wi-Fi 會自動斷線"、"路由器會自己重開"

**診斷流程**:

1. **Read**: `clients.health` - 檢查問題客戶端健康狀態
2. **Read**: `system.get_logs` - 查看系統錯誤日誌
3. **Test**: `diagnostics.latency_matrix` - 測試連線穩定性
4. **Read**: `system.get_hardware_info` - 檢查硬體狀態
5. **Act**: `wifi.set_power` - 調整功率避免過熱
6. **Act**: `wifi.client_disconnect` - 強制重連問題客戶端

### D. 訊號覆蓋不足 (weak_signal_coverage)

**症狀**: "房間 Wi-Fi 收不到"、"樓上訊號很弱"

**診斷流程**:

1. **Read**: `wifi.signal_map` - 生成訊號強度地圖
2. **Read**: `mesh.get_topology` - 檢查 Mesh 拓撲
3. **Test**: `wifi.roaming_test` - 測試漫遊效果
4. **Read**: `wifi.get_environment` - 分析頻道使用情況
5. **Act**: `wifi.set_power` - 調整發射功率
6. **Act**: `mesh.optimize_placement` - 最佳化節點位置建議

## 錯誤處理與復原

### LLM 診斷專用錯誤碼

| 錯誤碼 | 說明 | 處理方式 |
|--------|------|----------|
| `E_LLM_CONTEXT_INIT_FAILED` | LLM 上下文初始化失敗 | 重新初始化會話 |
| `E_LLM_TOOL_NOT_AVAILABLE` | 工具不可用 | 使用替代工具 |
| `E_LLM_SESSION_TIMEOUT` | 會話超時 | 創建新會話 |
| `E_LLM_CAPABILITY_MISMATCH` | 能力不匹配 | 降級執行 |
| `E_LLM_PERMISSION_DENIED` | 權限不足 | 請求用戶授權 |
| `E_LLM_CHANGESET_CONFLICT` | 變更集衝突 | 解決衝突後重試 |
| `E_LLM_ROLLBACK_FAILED` | 回滾失敗 | 手動介入修復 |

### 錯誤處理範例

```json
{
  "schema": "cmd.error/1.0",
  "ts": 1699123456789,
  "id": "cmd-failed-001",
  "status": "failed",
  "error": {
    "code": "E_LLM_TOOL_NOT_AVAILABLE",
    "message": "WiFi environment scan tool is currently unavailable",
    "details": {
      "requested_tool": "wifi.get_environment",
      "reason": "device_busy",
      "alternative_tools": ["wifi.basic_scan", "network.interface_status"],
      "retry_after_ms": 30000
    }
  },
  "recovery_action": {
    "type": "use_alternative",
    "alternative_tool": "wifi.basic_scan",
    "adjusted_expectations": "基本 WiFi 掃描結果，可能不如完整環境分析詳細"
  }
}
```

## 安全性考量

### 權限分級

| 權限等級 | 工具類型 | 授權要求 | 範例工具 |
|---------|---------|---------|---------|
| `read` | Read 工具 | 無需授權 | `topology.get_full` |
| `test` | Test 工具 | 無需授權 | `network.speedtest` |
| `config` | Act 工具 (配置) | 用戶確認 | `wifi.set_channel` |
| `admin` | Act 工具 (系統) | 管理員授權 | `system.restart` |

### Act 工具安全機制

```json
{
  "security_context": {
    "user_confirmation": {
      "required": true,
      "message": "即將調整 WiFi 頻道到 Channel 6，可能造成短暫斷線",
      "risk_level": "medium",
      "estimated_impact": "30 秒內客戶端重新連接"
    },
    "changeset_protection": {
      "enabled": true,
      "auto_rollback_timeout": 300,
      "rollback_conditions": ["connection_lost", "user_cancel"]
    },
    "audit_logging": {
      "enabled": true,
      "include_context": true,
      "retain_days": 90
    }
  }
}
```

### 會話隔離

```json
{
  "session_isolation": {
    "data_separation": true,
    "resource_limits": {
      "max_concurrent_tools": 3,
      "max_session_duration": 3600,
      "max_resource_usage": "medium"
    },
    "cleanup_policy": {
      "auto_cleanup_timeout": 7200,
      "cleanup_on_error": true
    }
  }
}
```

## 實作建議

### 工具開發指南

1. **統一介面**: 所有診斷工具都應實作統一的命令介面
2. **狀態回報**: 提供詳細的執行進度和結果資訊
3. **錯誤處理**: 包含詳細的錯誤資訊和恢復建議
4. **資源管理**: 合理控制資源使用，避免影響正常業務

### 效能最佳化

```python
class DiagnosticToolManager:
    def __init__(self):
        self.tool_cache = {}
        self.resource_monitor = ResourceMonitor()
        
    async def execute_tool(self, tool_name, args, trace_info):
        # 檢查資源可用性
        if not self.resource_monitor.can_execute(tool_name):
            raise ToolUnavailableError("Resource limit exceeded")
            
        # 並行執行 Read 工具
        if tool_name.startswith('read_'):
            return await self.execute_parallel(tool_name, args)
            
        # 序列執行 Act 工具
        if tool_name.startswith('act_'):
            return await self.execute_sequential(tool_name, args, trace_info)
```

### 監控和度量

```json
{
  "diagnostics_metrics": {
    "session_count": 145,
    "avg_session_duration": 180,
    "tool_success_rate": 0.94,
    "most_used_tools": [
      "network.speedtest_full",
      "wifi.get_environment", 
      "topology.get_full"
    ],
    "error_distribution": {
      "E_LLM_TOOL_NOT_AVAILABLE": 12,
      "E_LLM_SESSION_TIMEOUT": 8,
      "E_LLM_PERMISSION_DENIED": 3
    }
  }
}
```

---

**下一步**: 閱讀 [完整範例](11-examples.md) 查看各種診斷場景的詳細實作範例。
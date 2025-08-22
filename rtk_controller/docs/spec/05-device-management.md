# 設備管理與生命週期

## Device ID 規範

### 標識符格式
RTK MQTT 協議使用設備 MAC 地址作為唯一標識符，確保全域唯一性。

**格式要求**:
- **正則表達式**: `^[a-f0-9]{12}$`
- **字符數**: 固定 12 個字符
- **字符集**: 小寫十六進制 (a-f, 0-9)
- **分隔符**: 無 (移除冒號)

**有效範例**:
```
aabbccddeeff    # ✅ 正確格式
112233445566    # ✅ 正確格式  
778899aabbcc    # ✅ 正確格式
ddeeff001122    # ✅ 正確格式
```

**無效範例**:
```
aa:bb:cc:dd:ee:ff  # ❌ 包含冒號
AABBCCDDEEFF       # ❌ 大寫字母
aabbccddee         # ❌ 長度不足
aabbccddeeffgg     # ❌ 長度超出
```

### MAC 地址獲取方法

#### Linux 系統
```bash
# 獲取主要網路介面 MAC 地址
ip link show | grep -E "ether|link/ether" | head -1 | awk '{print $2}' | tr -d ':'

# 獲取特定介面 MAC 地址
cat /sys/class/net/eth0/address | tr -d ':'
```

#### Windows 系統
```cmd
# PowerShell
Get-NetAdapter | Select Name, MacAddress | Format-Table

# 格式化為協議要求格式
(Get-NetAdapter -Name "Ethernet").MacAddress -replace "-",""
```

#### 嵌入式系統
```c
// ESP32/Arduino 範例
#include "WiFi.h"

String getMacAddress() {
    String mac = WiFi.macAddress();
    mac.replace(":", "");
    mac.toLowerCase();
    return mac;
}
```

## 設備上下線偵測

### Last Will Testament (LWT) 機制

LWT 是 MQTT 協議的內建機制，當設備意外斷線時自動發布預設訊息。

#### LWT 設定要求

**Topic**: `rtk/v1/{tenant}/{site}/{device_id}/lwt`  
**QoS**: 1  
**Retained**: true  
**Payload**: 標準化的離線狀態訊息

#### LWT 訊息格式
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

### 上下線狀態管理

#### 設備上線流程
1. **建立 MQTT 連接**: 設置 LWT 訊息
2. **發布上線狀態**: 發送 online 狀態至 LWT topic
3. **發布設備屬性**: 發送 `attr` 訊息
4. **發布初始狀態**: 發送第一個 `state` 訊息
5. **開始正常操作**: 定期發送 telemetry 和 state

#### 設備下線流程

**正常下線**:
1. 發布 offline 狀態至 LWT topic
2. 關閉 MQTT 連接
3. 清理本地資源

**異常下線**:
- MQTT Broker 自動發布 LWT 訊息
- Controller 檢測到設備離線
- 觸發相關告警和處理流程

### 心跳與健康檢查

#### 狀態報告頻率
- **正常狀態**: 每 60 秒發送 `state` 訊息
- **問題狀態**: 每 30 秒發送 `state` 訊息  
- **嚴重狀態**: 每 10 秒發送 `state` 訊息

#### 連接保持策略
```json
{
  "keepalive": 60,
  "clean_session": false,
  "client_id": "rtk-{device_id}",
  "lwt": {
    "topic": "rtk/v1/{tenant}/{site}/{device_id}/lwt",
    "payload": "{\"status\":\"offline\",\"ts\":1699123456789}",
    "qos": 1,
    "retain": true
  }
}
```

## 設備能力發現

### 能力聲明機制
設備通過 `attr` 訊息聲明自身支援的功能和診斷能力。

#### 能力分類
```json
{
  "capabilities": {
    "device_type": "router",
    "diagnostics": {
      "network_tests": ["speed_test", "wan_test", "latency_test"],
      "wifi_analysis": ["site_survey", "channel_analysis", "client_monitoring"],
      "topology": ["arp_discovery", "route_analysis", "neighbor_detection"]
    },
    "controls": {
      "network": ["interface_control", "routing_config", "firewall_rules"],
      "wifi": ["ssid_management", "client_control", "power_adjustment"],
      "system": ["reboot", "factory_reset", "firmware_update"]
    },
    "sensors": ["temperature", "cpu_usage", "memory_usage", "power_consumption"],
    "llm_integration": {
      "supported": true,
      "session_tracking": true,
      "tool_categories": ["read", "test", "act"]
    }
  }
}
```

### 動態能力註冊
設備可以在運行時動態更新其能力聲明。

**觸發條件**:
- 韌體更新後
- 硬體配置變更
- 外掛模組加載/卸載
- 功能授權變更

**更新流程**:
1. 重新評估設備能力
2. 生成新的能力清單
3. 發布更新的 `attr` 訊息
4. 通知相關的控制器和監控系統

## 設備分群管理

### 群組定義
基於設備類型、位置或功能的邏輯分組。

#### 預定義群組類型
```json
{
  "groups": {
    "by_type": {
      "routers": ["aabbccddeeff", "112233445566"],
      "switches": ["778899aabbcc", "ddeeff001122"],
      "access_points": ["445566778899", "001122334455"]
    },
    "by_location": {
      "floor1": ["aabbccddeeff", "778899aabbcc"], 
      "floor2": ["112233445566", "ddeeff001122"]
    },
    "by_function": {
      "emergency_systems": ["778899aabbcc", "445566778899"],
      "lighting_controls": ["ddeeff001122", "001122334455"]
    }
  }
}
```

### 群組命令執行
```bash
# 群組命令 Topic
rtk/v1/{tenant}/{site}/group/{group_id}/cmd/req

# 個別設備回應路徑
rtk/v1/{tenant}/{site}/{device_id}/cmd/ack
rtk/v1/{tenant}/{site}/{device_id}/cmd/res
```

## 設備配置管理

### 配置版本控制
使用變更集 (changeset) 機制管理設備配置變更。

#### 變更集結構
```json
{
  "changeset_id": "change-1699123456789",
  "device_id": "aabbccddeeff", 
  "session_id": "session-abc123",
  "trace_id": "trace-def456",
  "changes": [
    {
      "operation": "set",
      "path": "network.wifi.ssid",
      "old_value": "old-network",
      "new_value": "new-network"
    }
  ],
  "rollback_info": {
    "supported": true,
    "automatic_timeout": 300
  }
}
```

### 配置回滾機制
支援配置變更的原子操作和自動回滾。

**回滾觸發條件**:
- 設備連接中斷超過指定時間
- 健康檢查失敗
- 手動觸發回滾命令

**回滾執行流程**:
1. 檢測回滾條件
2. 停止當前配置變更
3. 應用保存的舊配置
4. 驗證回滾結果
5. 發送回滾狀態通知

## 設備生命週期事件

### 關鍵生命週期階段

#### 初始化階段
- 設備首次連接
- 能力發現和註冊
- 初始配置載入

#### 運行階段  
- 正常狀態監控
- 定期資料收集
- 命令執行和回應

#### 維護階段
- 韌體更新
- 配置變更
- 診斷測試執行

#### 停用階段
- 計劃性下線
- 設備替換
- 最終清理

### 生命週期事件通知
```json
{
  "schema": "evt.lifecycle/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "event_type": "device_initialized",
  "lifecycle_stage": "initialization",
  "details": {
    "firmware_version": "1.2.3",
    "capabilities_registered": true,
    "initial_config_loaded": true
  }
}
```

---

**下一步**: 閱讀 [訊息格式](06-message-format.md) 了解共通 Payload 格式規範。
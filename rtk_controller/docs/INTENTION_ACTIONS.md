# RTK Controller 意圖類型與操作報告

本文件提供 RTK Controller 意圖分類系統的完整對應表，以及每種意圖類型所執行的 MQTT 請求和工具操作。

## 概述

RTK Controller 使用意圖驅動的工作流程系統，將 LLM 的參與限制在意圖分類上。分類完成後，預定義的確定性工作流程會根據識別出的意圖執行特定的工具序列。

## 意圖類別和工作流程

### 1. 覆蓋問題

#### 1.1 弱信號覆蓋 (`weak_signal_coverage_diagnosis`)

**意圖分類：**
- 主要：`coverage_issues`
- 次要：`weak_signal_coverage`
- 信心閾值：0.8

**用戶查詢範例 (來自 discussion/llm.md)：**
- "Wi-Fi is great in the living room but dead in my bedroom."
- "Why does my phone lose Wi-Fi when I move upstairs?"
- "Signal is very weak in the kitchen."
- "Do I need another router or extender?"

**工作流程步驟和工具調用：**

1. **資料收集階段 (並行執行)**
   - **WiFi 覆蓋掃描**
     - 工具：`wifi_signal_analysis`
     - 超時：30s
     - MQTT 主題：訂閱設備的信號強度報告
   
   - **信號強度映射**
     - 工具：`wifi_signal_analysis`
     - 超時：45s
     - MQTT 主題：從網狀網路節點收集覆蓋資料
   
   - **拓撲發現**
     - 工具：`network_topology_scan`
     - 參數：`include_offline: true`
     - 超時：20s
     - MQTT 主題：`rtk/v1/+/+/+/attr`、`rtk/v1/+/+/+/state`
   
   - **頻道分析**
     - 工具：`wifi_channel_analysis`
     - 參數：`band: all, scan_duration: 30`
     - 超時：35s
     - MQTT 主題：來自存取點的頻道利用率資料

2. **干擾檢測階段 (條件式順序執行)**
   - **分析干擾**
     - 工具：`device_interference_scan`
     - 超時：25s
     - 條件：當弱信號區域 > 0 時觸發
     - MQTT 主題：來自 WiFi 設備的干擾資料

3. **效能驗證階段 (條件式)**
   - **WAN 連線測試**
     - 工具：`wan_connectivity_test`
     - 超時：60s
     - 條件：當存在問題區域時觸發
     - MQTT 主題：來自邊緣設備的速度測試結果

#### 1.2 死角區域 (`coverage_optimization_diagnosis`)

**意圖分類：**
- 主要：`coverage_issues`
- 次要：`dead_zones`

**用戶查詢範例 (來自 discussion/llm.md)：**
- "Wi-Fi is great in the living room but dead in my bedroom." (與弱信號重疊)
- "Why does my phone lose Wi-Fi when I move upstairs?"

**相關工具：**
- `wifi_coverage_analysis`
- `mesh_node_positioning`
- `range_extension_recommendations`

#### 1.3 漫遊問題 (`roaming_diagnosis`)

**意圖分類：**
- 主要：`coverage_issues`
- 次要：`roaming_issues`

**用戶查詢範例 (來自 discussion/llm.md)：**
- "My phone doesn't switch to the closer Wi-Fi point."
- "Why does my laptop stay connected to the far AP?"
- "Roaming between mesh nodes doesn't work smoothly."
- "Signal is weak even though there's another AP nearby."

**相關工具：**
- `wifi_roaming_optimization`
- `handoff_analysis`
- `client_mobility_tracking`

### 2. 連線問題

#### 2.1 WAN 連線 (`wan_connectivity_diagnosis`)

**意圖分類：**
- 主要：`connectivity_issues`
- 次要：`wan_connectivity`

**用戶查詢範例 (來自 discussion/llm.md)：**
- "My Wi-Fi is connected but I can't get online."
- "The router shows no internet light."
- "Why does my laptop say 'No IP address'?"
- "Internet keeps dropping every few minutes."
- "Websites don't load unless I change DNS to Google."
- "Why does it say 'DHCP server not found'?"

**工作流程步驟：**

1. **WAN 狀態檢查**
   - 工具：`wan_connectivity_test`
   - 超時：30s
   - MQTT 主題：來自網關設備的 WAN 狀態

2. **DNS 解析測試**
   - 工具：`wan_connectivity_test`
   - 參數：`test_type: dns`
   - 超時：15s
   - MQTT 主題：DNS 解析結果

3. **速度測試 (條件式)**
   - 工具：`lan_performance_test`
   - 條件：WAN connectivity_status == "connected"
   - 超時：60s
   - MQTT 主題：頻寬測試結果

#### 2.2 設備離線 (`device_offline_diagnosis`)

**意圖分類：**
- 主要：`connectivity_issues`
- 次要：`device_offline`

**用戶查詢範例 (來自 discussion/llm.md)：**
- "My Alexa won't connect to Wi-Fi."
- "Why won't my smart bulb stay connected?"
- "The TV connects but says 'no internet'."
- "My printer keeps dropping off the Wi-Fi."
- "My device says it can't get an IP address."

**工作流程步驟：**

1. **拓撲掃描**
   - 工具：`network_topology_scan`
   - 參數：`include_offline: true, deep_scan: true`
   - 超時：45s
   - MQTT 主題：`rtk/v1/+/+/+/lwt`、`rtk/v1/+/+/+/state`

2. **設備 Ping 測試**
   - 工具：`network_health_check`
   - 參數：`target_devices: extracted_from_input`
   - 超時：30s
   - MQTT 主題：設備可達性狀態

#### 2.3 WiFi 斷線 (`wifi_connectivity_diagnosis`)

**意圖分類：**
- 主要：`connectivity_issues`
- 次要：`wifi_disconnection`

**用戶查詢範例 (來自 discussion/llm.md)：**
- "My Wi-Fi keeps disconnecting randomly."
- "The internet works for a while then drops out."
- "Why does my router reboot itself all the time?"
- "Sometimes the connection freezes when I stream."

**相關工具：**
- `wifi_association_analysis`
- `authentication_troubleshoot`
- `wireless_client_diagnostics`

### 3. 效能問題

#### 3.1 網速慢 (`performance_bottleneck_analysis`)

**意圖分類：**
- 主要：`performance_problems`
- 次要：`slow_internet`

**用戶查詢範例 (來自 discussion/llm.md)：**
- "Why is my internet so much slower than what I pay for?"
- "Speedtest only shows 20 Mbps but I have a 200 Mbps plan."
- "Wi-Fi is fine in the morning but super slow in the evening."
- "Why is the 2.4 GHz so slow compared to 5 GHz?"

**工作流程步驟：**

1. **基準測量 (並行)**
   - **WAN 速度測試**
     - 工具：`wan_connectivity_test`
     - 超時：60s
     - MQTT 主題：網際網路速度測試結果
   
   - **LAN 速度測試**
     - 工具：`lan_performance_test`
     - 超時：30s
     - MQTT 主題：區域網路效能資料
   
   - **QoS 分析**
     - 工具：`qos_analysis`
     - 超時：20s
     - MQTT 主題：流量優先級資料

2. **瓶頸識別**
   - 工具：`network_health_check`
   - 超時：30s
   - MQTT 主題：網路擁塞指示器

#### 3.2 高延遲 (`latency_analysis_diagnosis`)

**意圖分類：**
- 主要：`performance_problems`
- 次要：`high_latency`

**用戶查詢範例 (來自 discussion/llm.md)：**
- "Why does Zoom keep freezing?"
- "My online games lag a lot."
- "Ping is very high when I play at night."
- "Video calls always stutter even though speed seems okay."

**相關工具：**
- `ping_latency_analysis`
- `route_optimization`
- `traffic_path_analysis`

#### 3.3 頻寬瓶頸 (`bandwidth_analysis_diagnosis`)

**意圖分類：**
- 主要：`performance_problems`
- 次要：`bandwidth_bottleneck`

**用戶查詢範例 (來自 discussion/llm.md)：**
- "Why is my internet so much slower than what I pay for?" (與 slow_internet 重疊)
- "Wi-Fi is fine in the morning but super slow in the evening." (頻寬競爭)

**相關工具：**
- `bandwidth_utilization_analysis`
- `traffic_shaping_recommendations`
- `qos_policy_optimization`

### 4. 安全疑慮

#### 4.1 未授權存取 (`security_audit_diagnosis`)

**意圖分類：**
- 主要：`security_concerns`
- 次要：`unauthorized_access`

**相關工具：**
- `access_control_audit`
- `device_authentication_check`
- `suspicious_activity_detection`

#### 4.2 可疑流量 (`traffic_analysis_diagnosis`)

**意圖分類：**
- 主要：`security_concerns`
- 次要：`suspicious_traffic`

**相關工具：**
- `traffic_pattern_analysis`
- `anomaly_detection`
- `intrusion_detection_system`

### 5. 配置要求

#### 5.1 QoS 調整 (`qos_optimization_diagnosis`)

**意圖分類：**
- 主要：`configuration_requests`
- 次要：`qos_adjustment`

**相關工具：**
- `qos_policy_configuration`
- `traffic_prioritization_setup`
- `bandwidth_allocation_management`

#### 5.2 頻道優化 (`channel_optimization_diagnosis`)

**意圖分類：**
- 主要：`configuration_requests`
- 次要：`channel_optimization`

**相關工具：**
- `channel_selection_optimization`
- `interference_mitigation`
- `spectrum_analysis`

### 6. 一般網路診斷 (`general_network_diagnosis`)

**意圖分類：**
- 主要：`general`
- 次要：`network_diagnosis`
- 用於未分類或複雜問題的備用選項

**工作流程步驟：**

1. **網合掃描 (並行)**
   - **拓撲掃描**
     - 工具：`network_topology_scan`
     - 超時：30s
   
   - **健康檢查**
     - 工具：`network_health_check`
     - 超時：25s
   
   - **WiFi 分析**
     - 工具：`wifi_signal_analysis`
     - 超時：20s

2. **問題分析**
   - 工具：`wifi_advanced_diagnostics`
   - 超時：40s

## MQTT 主題結構

所有 MQTT 通信都遵循階層式主題結構：
```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}
```

### 訂閱主題 (輸入)
- `rtk/v1/+/+/+/state` - 設備健康摘要 (保留)
- `rtk/v1/+/+/+/evt/#` - 事件和警告
- `rtk/v1/+/+/+/lwt` - 連線狀態的遺嘱訊息
- `rtk/v1/+/+/+/cmd/ack` - 命令確認
- `rtk/v1/+/+/+/cmd/res` - 命令回應
- `rtk/v1/+/+/+/attr` - 設備屬性 (保留)

### 發佈主題 (輸出)
- `rtk/v1/{tenant}/{site}/{device_id}/cmd/req` - 命令請求
- `rtk/v1/{tenant}/{site}/{device_id}/telemetry/{metric}` - 效能指標
- `rtk/v1/{tenant}/{site}/{device_id}/topology/update` - 拓撲變更

### 排除於日誌之外
- `rtk/v1/+/+/+/telemetry/heartbeat` - 心跳訊息
- `rtk/v1/+/+/+/internal/#` - 系統內部訊息

## 可用的 LLM 工具分類

### WiFi 進階工具
1. `wifi_scan_channels` - 頻道掃描和分析
2. `wifi_analyze_interference` - 干擾檢測和緩解
3. `wifi_spectrum_utilization` - 頻譜使用率分析
4. `wifi_signal_strength_map` - 信號強度映射
5. `wifi_coverage_analysis` - 覆蓋優化分析
6. `wifi_roaming_optimization` - 漫遊行為優化
7. `wifi_throughput_analysis` - WiFi 效能分析
8. `wifi_latency_profiling` - 延遲特性分析

### 網狀網路工具
1. `mesh_get_topology` - 網狀網路拓撲擷取
2. `mesh_node_relationship` - 節點關係分析
3. `mesh_path_optimization` - 路徑優化分析
4. `mesh_backhaul_test` - 回程效能測試
5. `mesh_load_balancing` - 負載分配優化
6. `mesh_failover_simulation` - 故障轉移情境模擬

### 網路效能工具
1. `qos_get_status` - QoS 狀態和策略資訊
2. `traffic_get_stats` - 網路流量統計
3. `network_topology_scan` - 完整拓撲發現
4. `network_health_check` - 網路健康評估
5. `wan_connectivity_test` - WAN 連線測試
6. `lan_performance_test` - LAN 效能測量

### 配置管理工具
1. `config_wifi_settings` - WiFi 配置管理
2. `config_qos_policies` - QoS 策略配置
3. `config_security_settings` - 安全設定管理
4. `config_band_steering` - 頻段導向配置
5. `config_auto_optimize` - 自動優化設定
6. `config_validate_changes` - 配置驗證

### 拓撲工具
1. `topology_get_full` - 完整網路拓撲
2. `clients_list` - 已連線用戶端設備列表

## 風險緩解功能

### 意圖分類信心闾值
- **高信心度**：≥ 0.9 - 直接執行
- **中等信心度**：0.7-0.89 - 在監控下執行
- **低信心度**：0.5-0.69 - 請求用戶確認
- **備用方案**：< 0.5 - 使用一般網路診斷

### 工作流程執行安全
- **參數驗證**：所有工具參數在執行前都會驗證
- **超時保護**：每個工具調用都有可配置的超時
- **重試邏輯**：失敗的工具最多重試 2 次，並有退後機制
- **條件式執行**：步驟可以根據條件被跳過
- **優雅降級**：關鍵故障有備用工作流程

### 監控和指標
- **執行追蹤**：所有工作流程步驟都會被記錄和監控
- **效能指標**：追蹤工具執行時間和成功率
- **意圖準確性**：監控分類準確性以進行改進
- **故障分析**：分析失敗的執行以進行模式檢測

## 配置檔案

- **工作流程定義**：`configs/complete_workflows.yaml`
- **意圖分類**：`configs/intent_classification.yaml`
- **MCP 伺服器匯出**：`configs/mcp-server.yaml`
- **工具註冊**：定義在 `internal/llm/tools_*.go` 下的 Go 程式碼中

## 用戶查詢到意圖映射摘要

根據 discussion/llm.md，以下是用戶查詢如何映射到系統意圖：

### A. 沒有網際網路 → `wan_connectivity_diagnosis`
- "My Wi-Fi is connected but I can't get online."
- "The router shows no internet light."
- "Why does my laptop say 'No IP address'?"
- "Internet keeps dropping every few minutes."

### B. 網速慢 → `performance_bottleneck_analysis`
- "Why is my internet so much slower than what I pay for?"
- "Speedtest only shows 20 Mbps but I have a 200 Mbps plan."
- "Wi-Fi is fine in the morning but super slow in the evening."
- "Why is the 2.4 GHz so slow compared to 5 GHz?"

### C. 不穩定斷線 → `wifi_connectivity_diagnosis`
- "My Wi-Fi keeps disconnecting randomly."
- "The internet works for a while then drops out."
- "Why does my router reboot itself all the time?"
- "Sometimes the connection freezes when I stream."

### D. 弱信號覆蓋 → `weak_signal_coverage_diagnosis`
- "Wi-Fi is great in the living room but dead in my bedroom."
- "Why does my phone lose Wi-Fi when I move upstairs?"
- "Signal is very weak in the kitchen."
- "Do I need another router or extender?"

### E. 即時延遲 → `latency_analysis_diagnosis`
- "Why does Zoom keep freezing?"
- "My online games lag a lot."
- "Ping is very high when I play at night."
- "Video calls always stutter even though speed seems okay."

### F. 特定設備問題 → `device_offline_diagnosis`
- "My Alexa won't connect to Wi-Fi."
- "Why won't my smart bulb stay connected?"
- "The TV connects but says 'no internet'."
- "My printer keeps dropping off the Wi-Fi."

### G. 漫遊問題 → `roaming_diagnosis`
- "My phone doesn't switch to the closer Wi-Fi point."
- "Why does my laptop stay connected to the far AP?"
- "Roaming between mesh nodes doesn't work smoothly."
- "Signal is weak even though there's another AP nearby."

### H. 網狀網路回程問題 → `mesh_backhaul_diagnosis` (未來功能)
- "The upstairs mesh node shows poor connection."
- "Wi-Fi works near the main router but not through the satellite."
- "Why is the backhaul speed so low?"
- "The mesh keeps disconnecting from the main router."

### I. DHCP/DNS 進階問題 → `wan_connectivity_diagnosis` + `dns_diagnosis`
- "My device says it can't get an IP address."
- "Why does it say 'DHCP server not found'?"
- "Websites don't load unless I change DNS to Google."
- "Mesh nodes work on main router but not with Pi-hole DHCP."

## 使用範例

### CLI 自然語言介面
```bash
./rtk_controller --cli
> "WiFi signal is weak in the bedroom"
# Executes: weak_signal_coverage_diagnosis workflow

> "Internet is very slow"
# Executes: performance_bottleneck_analysis workflow

> "My laptop keeps disconnecting"
# Executes: wifi_connectivity_diagnosis workflow

> "Check if all devices are online"
# Executes: device_offline_diagnosis workflow
```

### 手動工作流程執行
```bash
# Execute specific workflow directly
./rtk_controller --cli
> workflow execute weak_signal_coverage_diagnosis

# List available workflows
> workflow list

# Get workflow details
> workflow describe wan_connectivity_diagnosis
```

這個意圖驅動的工作流程系統確保了確定性、可控制的網路診斷，同時保持了通過自然語言處理來處理多樣化用戶查詢的靈活性。
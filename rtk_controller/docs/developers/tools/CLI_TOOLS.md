# RTK MQTT CLI 工具指南

## 概述

本文檔提供RTK MQTT系統中命令列介面(CLI)工具的完整使用指南，包含所有可用命令、參數說明和使用範例。

## CLI工具架構

### 命令結構
```
rtk_controller --cli

RTK> [category] [command] [options]
```

### 主要類別
- `device` - 設備管理命令
- `topology` - 拓撲管理命令  
- `diagnostic` - 診斷工具命令
- `config` - 配置管理命令
- `system` - 系統管理命令
- `llm` - LLM整合命令

## 設備管理命令 (device)

### device list
列出所有已知設備

```bash
RTK> device list [options]
```

#### 選項
- `--format <format>` - 輸出格式: table, json, csv (預設: table)
- `--filter <filter>` - 過濾條件: online, offline, all (預設: all)
- `--sort <field>` - 排序欄位: name, type, last_seen (預設: name)

#### 範例
```bash
# 列出所有設備
RTK> device list

# 只顯示線上設備，JSON格式
RTK> device list --filter online --format json

# 按最後上線時間排序
RTK> device list --sort last_seen
```

#### 輸出範例
```
Device List (Total: 15, Online: 12, Offline: 3)
================================================================================
Device ID              Type        Status    Last Seen           Location
--------------------------------------------------------------------------------
router-main-001        router      online    2024-01-15 14:30:25 office/floor1
ap-lobby-002           ap          online    2024-01-15 14:29:58 office/lobby
sensor-temp-003        iot         offline   2024-01-15 12:15:42 office/server_room
switch-core-001        switch      online    2024-01-15 14:30:20 datacenter/rack1
```

### device info
顯示特定設備的詳細資訊

```bash
RTK> device info <device_id> [options]
```

#### 選項
- `--include-history` - 包含歷史資料
- `--format <format>` - 輸出格式 (預設: table)

#### 範例
```bash
# 顯示設備詳細資訊
RTK> device info router-main-001

# 包含歷史資料
RTK> device info router-main-001 --include-history
```

#### 輸出範例
```
Device Information: router-main-001
================================================================================
Basic Information:
  Device ID:         router-main-001
  Device Type:       router
  Manufacturer:      Cisco Systems
  Model:             ISR4321
  Firmware Version:  16.09.04
  Serial Number:     FCZ2140L0GK
  MAC Address:       a46c2a123456

Status:
  Health:            ok
  Connection Status: connected
  Uptime:            15d 8h 42m
  Last Seen:         2024-01-15 14:30:25
  Signal Strength:   -45 dBm

Network Configuration:
  IP Address:        192.168.1.1
  Subnet Mask:       255.255.255.0
  Gateway:           192.168.1.254
  DNS Servers:       8.8.8.8, 8.8.4.4

Capabilities:
  - routing
  - nat
  - firewall
  - vpn
  - qos

Recent Activity:
  - 2024-01-15 14:25:00: Configuration updated
  - 2024-01-15 13:45:30: Interface eth0 link up
  - 2024-01-15 12:30:15: QoS policy applied
```

### device command
向設備發送命令

```bash
RTK> device command <device_id> <operation> [args...]
```

#### 支援的操作
- `reboot` - 重新啟動設備
- `restart_service <service>` - 重啟特定服務
- `get_system_info` - 獲取系統資訊
- `update_config <config_json>` - 更新配置
- `run_diagnostics [test_type]` - 執行診斷測試

#### 範例
```bash
# 重新啟動設備
RTK> device command router-main-001 reboot

# 重啟網路服務
RTK> device command router-main-001 restart_service network

# 獲取系統資訊
RTK> device command router-main-001 get_system_info

# 更新配置
RTK> device command router-main-001 update_config '{"log_level": "debug"}'

# 執行速度測試
RTK> device command router-main-001 run_diagnostics speed_test
```

### device monitor
即時監控設備狀態

```bash
RTK> device monitor <device_id> [options]
```

#### 選項
- `--interval <seconds>` - 更新間隔 (預設: 5)
- `--metrics <metrics>` - 監控指標: cpu,memory,network,all (預設: all)
- `--duration <seconds>` - 監控持續時間 (預設: 無限)

#### 範例
```bash
# 監控設備狀態
RTK> device monitor router-main-001

# 每10秒更新，只監控CPU和記憶體
RTK> device monitor router-main-001 --interval 10 --metrics cpu,memory

# 監控5分鐘
RTK> device monitor router-main-001 --duration 300
```

## 拓撲管理命令 (topology)

### topology show
顯示網路拓撲

```bash
RTK> topology show [options]
```

#### 選項
- `--format <format>` - 輸出格式: tree, graph, json (預設: tree)
- `--depth <levels>` - 顯示深度 (預設: 無限)
- `--include-offline` - 包含離線設備
- `--filter <type>` - 過濾設備類型

#### 範例
```bash
# 顯示拓撲樹
RTK> topology show

# 圖形格式，深度2層
RTK> topology show --format graph --depth 2

# 只顯示路由器和交換機
RTK> topology show --filter router,switch
```

#### 輸出範例
```
Network Topology (Tree View)
================================================================================
Root: datacenter-gateway (router)
├─ core-switch-001 (switch)
│  ├─ floor1-switch-001 (switch)
│  │  ├─ ap-office-101 (ap) [12 clients]
│  │  ├─ ap-office-102 (ap) [8 clients]
│  │  └─ printer-laser-001 (iot)
│  └─ floor2-switch-001 (switch)
│     ├─ ap-office-201 (ap) [15 clients]
│     └─ sensor-hvac-001 (iot)
└─ backup-router-001 (router) [standby]
   └─ wan-connection-backup (link)

Summary:
  Total Devices: 9
  Active Links: 8
  Redundant Paths: 1
  Network Health: 95%
```

### topology refresh
重新發現網路拓撲

```bash
RTK> topology refresh [options]
```

#### 選項
- `--deep-scan` - 執行深度掃描
- `--timeout <seconds>` - 掃描超時時間 (預設: 60)
- `--discover-method <method>` - 發現方法: snmp, lldp, arp, all (預設: all)

#### 範例
```bash
# 重新整理拓撲
RTK> topology refresh

# 深度掃描，2分鐘超時
RTK> topology refresh --deep-scan --timeout 120
```

### topology path
顯示兩個設備間的路徑

```bash
RTK> topology path <source_device> <target_device> [options]
```

#### 選項
- `--show-metrics` - 顯示路徑指標
- `--alternative-paths` - 顯示替代路徑

#### 範例
```bash
# 顯示路徑
RTK> topology path router-main-001 ap-office-101

# 包含指標和替代路徑
RTK> topology path router-main-001 ap-office-101 --show-metrics --alternative-paths
```

## 診斷工具命令 (diagnostic)

### diagnostic speed_test
執行網路速度測試

```bash
RTK> diagnostic speed_test <device_id> [options]
```

#### 選項
- `--server <server>` - 測試伺服器: auto, ookla, custom (預設: auto)
- `--duration <seconds>` - 測試持續時間 (預設: 10)
- `--direction <dir>` - 測試方向: download, upload, both (預設: both)

#### 範例
```bash
# 基本速度測試
RTK> diagnostic speed_test router-main-001

# 使用Ookla伺服器，測試30秒
RTK> diagnostic speed_test router-main-001 --server ookla --duration 30

# 只測試下載速度
RTK> diagnostic speed_test router-main-001 --direction download
```

### diagnostic wifi_scan
WiFi頻道掃描

```bash
RTK> diagnostic wifi_scan <device_id> [options]
```

#### 選項
- `--channels <list>` - 指定頻道列表 (例: 1,6,11)
- `--band <band>` - 頻段: 2.4ghz, 5ghz, both (預設: both)
- `--duration <ms>` - 每頻道掃描時間 (預設: 500)

#### 範例
```bash
# 完整WiFi掃描
RTK> diagnostic wifi_scan ap-office-101

# 只掃描特定頻道
RTK> diagnostic wifi_scan ap-office-101 --channels 1,6,11

# 只掃描5GHz頻段
RTK> diagnostic wifi_scan ap-office-101 --band 5ghz
```

### diagnostic latency_test
延遲測試

```bash
RTK> diagnostic latency_test <device_id> [options]
```

#### 選項
- `--targets <hosts>` - 目標主機列表
- `--count <number>` - 測試次數 (預設: 20)
- `--interval <ms>` - 測試間隔 (預設: 1000)

#### 範例
```bash
# 延遲測試
RTK> diagnostic latency_test router-main-001

# 測試特定主機
RTK> diagnostic latency_test router-main-001 --targets 8.8.8.8,1.1.1.1

# 100次測試，間隔500ms
RTK> diagnostic latency_test router-main-001 --count 100 --interval 500
```

### diagnostic comprehensive
執行全面診斷

```bash
RTK> diagnostic comprehensive <device_id> [options]
```

#### 選項
- `--tests <test_list>` - 指定測試項目
- `--save-report <file>` - 儲存報告到檔案
- `--email-report <email>` - 發送報告到信箱

#### 範例
```bash
# 全面診斷
RTK> diagnostic comprehensive router-main-001

# 只執行特定測試
RTK> diagnostic comprehensive router-main-001 --tests speed,latency,wifi

# 儲存報告
RTK> diagnostic comprehensive router-main-001 --save-report router-001-report.json
```

## 配置管理命令 (config)

### config show
顯示配置

```bash
RTK> config show [section] [options]
```

#### 選項
- `--format <format>` - 輸出格式: yaml, json, table (預設: yaml)
- `--include-defaults` - 包含預設值

#### 範例
```bash
# 顯示所有配置
RTK> config show

# 顯示MQTT配置
RTK> config show mqtt

# JSON格式
RTK> config show --format json
```

### config set
設定配置值

```bash
RTK> config set <key> <value> [options]
```

#### 選項
- `--persist` - 永久儲存配置
- `--validate` - 驗證配置值

#### 範例
```bash
# 設定日誌等級
RTK> config set logging.level debug --persist

# 設定MQTT broker
RTK> config set mqtt.broker_host 192.168.1.100 --persist --validate
```

### config reload
重新載入配置

```bash
RTK> config reload [file]
```

#### 範例
```bash
# 重新載入預設配置
RTK> config reload

# 載入特定配置檔案
RTK> config reload /path/to/config.yaml
```

### config validate
驗證配置

```bash
RTK> config validate [file]
```

#### 範例
```bash
# 驗證當前配置
RTK> config validate

# 驗證特定檔案
RTK> config validate /path/to/config.yaml
```

## 系統管理命令 (system)

### system status
顯示系統狀態

```bash
RTK> system status [options]
```

#### 選項
- `--detailed` - 顯示詳細資訊
- `--refresh <seconds>` - 自動重新整理間隔

#### 範例
```bash
# 顯示系統狀態
RTK> system status

# 詳細資訊
RTK> system status --detailed

# 每5秒自動重新整理
RTK> system status --refresh 5
```

#### 輸出範例
```
RTK MQTT Controller System Status
================================================================================
Service Status:
  ✓ RTK Controller:      Running (PID: 1234)
  ✓ MQTT Broker:         Connected (localhost:1883)
  ✓ Database:           Online (BuntDB)
  ✓ Web Interface:      Active (Port: 8080)

System Resources:
  CPU Usage:            15.2%
  Memory Usage:         245MB / 4GB (6.1%)
  Disk Usage:           12GB / 100GB (12%)
  Network:              RX: 1.2MB/s, TX: 800KB/s

Active Connections:
  MQTT Clients:         24
  Device Connections:   18
  Web Sessions:         3

Statistics (Last 24h):
  Messages Processed:   1,234,567
  Commands Executed:    156
  Alerts Generated:     12
  Uptime:              15d 8h 42m

Health Score:           98/100
```

### system logs
查看系統日誌

```bash
RTK> system logs [options]
```

#### 選項
- `--level <level>` - 日誌等級: debug, info, warn, error (預設: info)
- `--tail <lines>` - 顯示最後N行 (預設: 50)
- `--follow` - 即時跟蹤日誌
- `--filter <pattern>` - 過濾條件

#### 範例
```bash
# 查看最近日誌
RTK> system logs

# 只顯示錯誤日誌
RTK> system logs --level error

# 即時跟蹤最後100行
RTK> system logs --tail 100 --follow

# 過濾包含"mqtt"的日誌
RTK> system logs --filter mqtt
```

### system backup
建立系統備份

```bash
RTK> system backup [options]
```

#### 選項
- `--output <file>` - 輸出檔案路徑
- `--include <components>` - 包含的組件: config, data, logs, all (預設: all)
- `--compress` - 壓縮備份檔案

#### 範例
```bash
# 建立完整備份
RTK> system backup

# 只備份配置
RTK> system backup --include config --output config-backup.tar.gz

# 壓縮備份
RTK> system backup --compress --output full-backup.tar.gz
```

### system restore
還原系統備份

```bash
RTK> system restore <backup_file> [options]
```

#### 選項
- `--components <list>` - 要還原的組件
- `--confirm` - 確認還原操作

#### 範例
```bash
# 還原完整備份
RTK> system restore full-backup.tar.gz --confirm

# 只還原配置
RTK> system restore config-backup.tar.gz --components config --confirm
```

## LLM整合命令 (llm)

### llm ask
向LLM諮詢網路問題

```bash
RTK> llm ask "<question>" [options]
```

#### 選項
- `--context <context>` - 提供額外上下文
- `--include-data` - 包含當前系統數據

#### 範例
```bash
# 詢問網路問題
RTK> llm ask "為什麼設備router-main-001的延遲突然增加?"

# 包含系統數據的詢問
RTK> llm ask "如何優化當前網路性能?" --include-data

# 提供額外上下文
RTK> llm ask "這個錯誤訊息代表什麼?" --context "Connection timeout to 192.168.1.100"
```

### llm analyze
LLM網路分析

```bash
RTK> llm analyze <analysis_type> [options]
```

#### 分析類型
- `network` - 網路效能分析
- `topology` - 拓撲結構分析
- `security` - 安全性分析
- `optimization` - 優化建議

#### 範例
```bash
# 網路效能分析
RTK> llm analyze network

# 拓撲結構分析
RTK> llm analyze topology

# 安全性分析
RTK> llm analyze security
```

### llm recommend
獲取LLM建議

```bash
RTK> llm recommend <recommendation_type> [target]
```

#### 建議類型
- `optimization` - 優化建議
- `troubleshooting` - 故障排除建議
- `configuration` - 配置建議

#### 範例
```bash
# 獲取優化建議
RTK> llm recommend optimization

# 特定設備的故障排除建議
RTK> llm recommend troubleshooting router-main-001

# 配置建議
RTK> llm recommend configuration
```

## 互動功能

### 自動完成
CLI支援Tab鍵自動完成命令和參數

```bash
RTK> dev<TAB>        # 自動完成為 "device"
RTK> device l<TAB>   # 自動完成為 "device list"
```

### 歷史記錄
使用上下箭頭瀏覽命令歷史

```bash
# 按上箭頭查看上一個命令
# 按下箭頭查看下一個命令
# Ctrl+R 搜尋歷史命令
```

### 管道和重定向
支援輸出重定向和管道操作

```bash
# 輸出到檔案
RTK> device list > devices.txt

# 管道到grep
RTK> system logs | grep error

# 附加到檔案
RTK> topology show >> network-topology.log
```

### 別名
支援命令別名設定

```bash
# 設定別名
RTK> alias dl "device list"
RTK> alias st "system status"

# 使用別名
RTK> dl --format json
RTK> st --detailed
```

## 腳本模式

### 批次執行
```bash
# 從檔案執行命令
RTK> source commands.txt

# 執行單行腳本
RTK> exec "device list; system status"
```

### 變數支援
```bash
# 設定變數
RTK> set DEVICE_ID router-main-001
RTK> set TEST_COUNT 10

# 使用變數
RTK> device info $DEVICE_ID
RTK> diagnostic latency_test $DEVICE_ID --count $TEST_COUNT
```

### 條件執行
```bash
# 條件執行
RTK> if device status router-main-001 == online then device command router-main-001 get_system_info

# 迴圈執行
RTK> for device in $(device list --format ids) do diagnostic speed_test $device
```

## 配置檔案

### CLI配置
```yaml
# ~/.rtk_cli_config.yaml
cli:
  prompt: "RTK> "
  history_size: 1000
  auto_complete: true
  colors: true
  aliases:
    dl: "device list"
    st: "system status"
    dt: "diagnostic comprehensive"
  
output:
  default_format: table
  paging: true
  page_size: 20
  
timeouts:
  command_timeout: 30
  connection_timeout: 10
```

### 環境變數
```bash
# RTK CLI環境變數
export RTK_CLI_CONFIG=~/.rtk_cli_config.yaml
export RTK_CLI_HISTORY=~/.rtk_cli_history
export RTK_CLI_LOG_LEVEL=info
export RTK_CONTROLLER_HOST=localhost
export RTK_CONTROLLER_PORT=8080
```

## 故障排除

### 常見問題

1. **連接失敗**
```bash
RTK> system status
Error: Failed to connect to RTK Controller

# 檢查連接設定
RTK> config show connection
```

2. **命令超時**
```bash
RTK> config set timeouts.command_timeout 60
RTK> config reload
```

3. **權限錯誤**
```bash
# 檢查認證設定
RTK> config show auth
```

### 除錯模式
```bash
# 啟用除錯模式
RTK> config set logging.level debug

# 查看詳細錯誤
RTK> system logs --level debug --follow
```

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Quick Start Guide](../guides/QUICK_START_GUIDE.md)
- [Troubleshooting Guide](../guides/TROUBLESHOOTING_GUIDE.md)
- [Network Diagnostics](../diagnostics/NETWORK_DIAGNOSTICS.md)
- [WiFi Diagnostics](../diagnostics/WIFI_DIAGNOSTICS.md)
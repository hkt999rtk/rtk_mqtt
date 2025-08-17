# RTK Controller 交互式 CLI 使用指南

## 啟動交互式 CLI

使用 `--cli` 參數啟動交互式命令行介面：

```bash
./rtk-controller --cli
```

## 主要功能

### 1. 系統狀態查詢
```bash
rtk> status                    # 顯示系統整體狀態
rtk> system info              # 顯示系統資訊
rtk> system health            # 顯示系統健康狀態
rtk> system stats             # 顯示系統統計
```

### 2. 設備管理
```bash
rtk> device list              # 列出所有設備
rtk> device show <device_id>  # 顯示設備詳細資訊
rtk> device status <device_id> # 顯示設備狀態
rtk> device history <device_id> # 顯示設備歷史記錄
rtk> device stats             # 顯示設備統計
```

### 3. 命令管理
```bash
rtk> command send <device_id> <operation> [timeout] # 發送命令
rtk> command list             # 列出命令歷史
rtk> command show <command_id> # 顯示命令詳情
rtk> command cancel <command_id> # 取消待執行命令
rtk> command stats            # 顯示命令統計
```

### 4. 事件監控
```bash
rtk> event list               # 列出事件
rtk> event show <event_id>    # 顯示事件詳情
rtk> event stats              # 顯示事件統計
rtk> event watch              # 即時監控事件（尚未實現）
```

### 5. 診斷管理
```bash
rtk> diagnosis run <device_id> # 執行設備診斷
rtk> diagnosis list           # 列出診斷結果
rtk> diagnosis show <diag_id> # 顯示診斷詳情
rtk> diagnosis analyzers      # 列出可用分析器
```

### 6. 配置管理
```bash
rtk> config show              # 顯示配置
rtk> config reload            # 重新載入配置
rtk> config set <key> <value> # 設定配置值
```

### 7. 日誌管理
```bash
rtk> log show                 # 顯示最近日誌
rtk> log tail                 # 實時查看日誌
rtk> log search <keyword>     # 搜索日誌
rtk> log download <seconds>   # 下載指定時間內的日誌
```

### 8. 測試工具
```bash
rtk> test mqtt                # 測試 MQTT 連接
rtk> test storage             # 測試存儲連接
rtk> test ping <device_id>    # Ping 設備
```

### 9. 通用命令
```bash
rtk> help                     # 顯示幫助資訊
rtk> help <command>           # 顯示特定命令的幫助
rtk> version                  # 顯示版本資訊
rtk> clear                    # 清除螢幕
rtk> exit                     # 退出 CLI
rtk> quit                     # 退出 CLI（同 exit）
```

## 特色功能

### 1. 自動補全
- 使用 Tab 鍵可以自動補全命令和子命令
- 支援多層級的命令補全

### 2. 命令歷史
- 使用上下箭頭鍵瀏覽命令歷史
- 命令歷史自動保存到 `/tmp/.rtk_cli_history`

### 3. 中斷處理
- 按 Ctrl+C 可以中斷當前輸入並開始新行
- 連續兩次 Ctrl+C 或輸入 EOF（Ctrl+D）退出 CLI

### 4. 錯誤處理
- 輸入錯誤命令時會提示可用命令
- 缺少參數時會顯示使用方法

## 使用範例

```bash
# 啟動交互式 CLI
$ ./rtk-controller --cli

RTK Controller Interactive CLI
==============================
Version: 1.0.0
Type 'help' for available commands, 'exit' to quit

# 查看系統狀態
rtk> status
RTK Controller System Status
============================
MQTT:          disconnected
MQTT Broker:   localhost:1883
Devices:       0 total, 0 online, 0 offline
Commands:      0 total, 0 pending, 0 completed

# 列出設備
rtk> device list
Device List
-----------
Total devices: 0

No devices found

# 測試 MQTT 連接
rtk> test mqtt
Testing MQTT connection...
❌ MQTT connection is down

# 查看系統健康狀態
rtk> system health
System Health Check
===================
❌ MQTT: Connection is down
✅ Storage: Available
✅ Device Manager: Running
✅ Command Manager: Running

Overall Status: ❌ Unhealthy

Issues:
  ⚠ MQTT connection is down

# 退出 CLI
rtk> exit
Bye!
```

## 配置要求

確保 `configs/controller.yaml` 配置文件存在且包含正確的設定：

```yaml
mqtt:
  broker: "localhost"
  port: 1883
  client_id: "rtk-controller"

storage:
  path: "data"

# ... 其他配置
```

## 故障排除

### 1. MQTT 連接失敗
- 確認 MQTT broker 正在運行
- 檢查配置文件中的 broker 地址和端口
- 檢查網路連接

### 2. 存儲錯誤
- 確認 `data` 目錄存在且有寫入權限
- 檢查磁碟空間

### 3. CLI 無法啟動
- 確認配置文件路徑正確
- 檢查日誌文件以獲取詳細錯誤信息

## 日誌位置

- 應用程式日誌：`logs/controller.log`
- CLI 命令歷史：`/tmp/.rtk_cli_history`
# RTK Controller 工程師使用手冊

## 概述

本手冊專為拿到 RTK Controller 可執行檔案的工程師和使用者設計，提供詳細的操作指導、測試程序和實例演練。如果你是系統管理員，請參閱 `RELEASE.md` 獲取安裝和部署指南。

## 快速開始 (5 分鐘)

### 1. 解壓縮和準備

```bash
# 解壓縮發行包
tar -xzf release_v1.0.0.tgz
cd rtk_controller_release

# 設定可執行權限 (Linux/macOS)
chmod +x bin/rtk_controller-*
chmod +x demo_cli.sh
```

### 2. 選擇你的平台

```bash
# Linux ARM64
export RTK_BIN="./bin/rtk_controller-linux-arm64"

# Linux x86_64
export RTK_BIN="./bin/rtk_controller-linux-amd64"

# macOS ARM64
export RTK_BIN="./bin/rtk_controller-darwin-arm64"

# Windows (PowerShell)
$RTK_BIN = ".\bin\rtk_controller-windows-amd64.exe"
```

### 3. 驗證安裝

```bash
# 檢查版本
$RTK_BIN --version

# 輸出範例：
# RTK Controller v1.0.0 (built at 2025-08-16_10:04:55)
```

### 4. 快速演示

```bash
# 執行功能演示
./demo_cli.sh

# 這會啟動互動式演示，展示主要功能
```

✅ **如果以上步驟都成功，你已經準備好開始使用 RTK Controller！**

## 配置設定

### 配置檔案結構

RTK Controller 使用 YAML 格式的配置檔案。主要配置檔案是 `configs/controller.yaml`：

```yaml
# configs/controller.yaml - 完整配置範例

# MQTT 連接設定
mqtt:
  broker: "localhost:1883"                    # MQTT broker 地址
  username: "rtk_user"                        # 使用者名稱
  password: "your_password"                   # 密碼
  client_id: "rtk_controller_001"             # 客戶端 ID
  keep_alive: 60                              # 心跳間隔 (秒)
  clean_session: true                         # 清除會話
  reconnect_interval: 30                      # 重連間隔 (秒)
  
  # TLS 設定 (可選)
  tls:
    enabled: false                            # 啟用 TLS
    ca_cert: "/path/to/ca.crt"               # CA 憑證
    client_cert: "/path/to/client.crt"       # 客戶端憑證
    client_key: "/path/to/client.key"        # 客戶端私鑰
    insecure_skip_verify: false              # 跳過憑證驗證

# Controller 設定
controller:
  tenant: "your_company"                      # 租戶名稱
  site: "main_site"                          # 站點名稱
  device_id: "controller_001"                # 設備 ID
  log_level: "info"                          # 日誌等級: debug, info, warn, error
  
  # 資料儲存設定
  storage:
    type: "buntdb"                           # 儲存類型
    path: "./data/controller.db"             # 資料庫路徑
    
  # 診斷設定
  diagnosis:
    enabled: true                            # 啟用診斷
    interval: 300                            # 診斷間隔 (秒)
    retention_days: 7                        # 資料保留天數

# 外掛程式設定
plugins:
  enabled: true                              # 啟用外掛程式
  directory: "./plugins"                     # 外掛程式目錄
  
# 日誌設定
logging:
  level: "info"                              # 日誌等級
  format: "json"                             # 日誌格式: json, text
  output: "file"                             # 輸出: stdout, file, both
  file_path: "./logs/controller.log"         # 日誌檔案路徑
  max_size: 100                              # 檔案大小限制 (MB)
  max_backups: 5                             # 備份檔案數量
  max_age: 30                                # 保留天數
```

### 最小配置

如果你只想快速開始，這是最小的配置檔案：

```yaml
# configs/minimal.yaml
mqtt:
  broker: "localhost:1883"
  username: "test_user"
  password: "test_password"

controller:
  tenant: "test_tenant"
  site: "test_site"
  log_level: "info"
```

### 環境變數覆蓋

你可以使用環境變數覆蓋配置：

```bash
# 設定環境變數
export RTK_MQTT_BROKER="mqtt://production-broker:1883"
export RTK_MQTT_USERNAME="prod_user"
export RTK_MQTT_PASSWORD="secure_password"
export RTK_TENANT="production"
export RTK_SITE="datacenter_01"
export RTK_LOG_LEVEL="warn"

# 啟動 Controller (環境變數會覆蓋配置檔案)
$RTK_BIN --config configs/controller.yaml
```

## 基本操作

### CLI 互動模式

RTK Controller 提供強大的互動式 CLI 介面：

```bash
# 啟動 CLI 模式
$RTK_BIN --cli --config configs/controller.yaml

# 你會看到類似這樣的提示符：
# RTK Controller CLI v1.0.0
# Type 'help' for available commands
# rtk>
```

### 核心命令

#### 1. 設備管理

```bash
# 在 CLI 中執行以下命令：

# 列出所有設備
rtk> device list

# 查看特定設備狀態
rtk> device status wifi_router_001

# 查看設備詳細資訊
rtk> device info wifi_router_001

# 搜尋設備
rtk> device search --type wifi_router
rtk> device search --status online

# 設備統計
rtk> device stats
```

#### 2. 命令管理

```bash
# 發送命令給設備
rtk> cmd send wifi_router_001 reboot

# 發送帶參數的命令
rtk> cmd send wifi_router_001 set_config '{"wifi_channel": 6}'

# 查看命令歷史
rtk> cmd history

# 查看待處理的命令
rtk> cmd pending

# 查看命令執行結果
rtk> cmd result <command_id>
```

#### 3. 診斷功能

```bash
# 執行設備診斷
rtk> diagnosis run wifi_router_001

# 查看診斷報告
rtk> diagnosis report wifi_router_001

# 查看所有診斷歷史
rtk> diagnosis list

# 匯出診斷資料
rtk> diagnosis export --format json --output ./reports/
```

#### 4. 系統監控

```bash
# 查看系統狀態
rtk> system status

# 查看 MQTT 連接狀態
rtk> system mqtt-status

# 查看統計資訊
rtk> system stats

# 查看日誌
rtk> system logs --lines 50
rtk> system logs --level error
```

#### 5. 配置管理

```bash
# 查看當前配置
rtk> config show

# 重新載入配置
rtk> config reload

# 驗證配置
rtk> config validate

# 設定單一參數
rtk> config set logging.level debug
```

#### 6. 說明和工具

```bash
# 顯示所有可用命令
rtk> help

# 顯示特定命令的說明
rtk> help device
rtk> help cmd send

# 清除螢幕
rtk> clear

# 退出 CLI
rtk> exit
# 或按 Ctrl+C
```

### 命令列模式

除了互動式 CLI，你也可以直接執行單一命令：

```bash
# 列出設備
$RTK_BIN --config configs/controller.yaml --cmd "device list"

# 查看系統狀態
$RTK_BIN --config configs/controller.yaml --cmd "system status"

# 執行診斷
$RTK_BIN --config configs/controller.yaml --cmd "diagnosis run wifi_router_001"
```

## 進階功能

### MQTT 主題結構

RTK Controller 使用標準化的 MQTT 主題結構：

```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}

範例：
- rtk/v1/company_a/office/wifi_router_001/state
- rtk/v1/company_a/office/wifi_router_001/telemetry/cpu_usage
- rtk/v1/company_a/office/wifi_router_001/evt/connection_lost
- rtk/v1/company_a/office/wifi_router_001/cmd/req
```

### 訊息格式

所有 MQTT 訊息都使用 JSON 格式：

```json
{
  "timestamp": "2025-08-16T10:30:00Z",
  "device_id": "wifi_router_001",
  "tenant": "company_a",
  "site": "office",
  "message_type": "state",
  "payload": {
    "status": "online",
    "cpu_usage": 45.2,
    "memory_usage": 67.8,
    "uptime": 86400
  }
}
```

### 外掛程式系統

RTK Controller 支援動態載入外掛程式：

```bash
# 列出可用外掛程式
rtk> plugin list

# 載入外掛程式
rtk> plugin load wifi_router_plugin

# 卸載外掛程式
rtk> plugin unload wifi_router_plugin

# 查看外掛程式資訊
rtk> plugin info wifi_router_plugin
```

### 自訂命令開發

你可以為 RTK Controller 開發自訂命令：

```bash
# 建立命令範本
rtk> dev create-command my_custom_command

# 測試自訂命令
rtk> dev test-command my_custom_command
```

## 測試指南

### 功能測試

#### 1. 連接測試

```bash
# 測試 MQTT 連接
$RTK_BIN --config configs/controller.yaml --test-connection

# 預期輸出：
# ✓ MQTT broker connection successful
# ✓ Authentication successful
# ✓ Topic subscription successful
```

#### 2. 基本 CLI 功能測試

```bash
# 執行 CLI 測試腳本
./test/scripts/test_cli_commands.sh

# 手動測試步驟：
$RTK_BIN --cli --config configs/controller.yaml

# 在 CLI 中執行：
rtk> help                    # 應該顯示命令清單
rtk> system status          # 應該顯示系統狀態
rtk> device list            # 應該顯示設備清單（可能為空）
rtk> config show            # 應該顯示當前配置
rtk> exit                   # 應該正常退出
```

#### 3. 設備模擬測試

```bash
# 啟動設備模擬器（如果可用）
$RTK_BIN --simulate-device wifi_router_001 &

# 在另一個終端中測試
$RTK_BIN --cli --config configs/controller.yaml

rtk> device list            # 應該看到模擬設備
rtk> device status wifi_router_001
rtk> cmd send wifi_router_001 ping
```

### 性能測試

```bash
# 執行性能測試
./test/scripts/performance_test.sh

# 測試內容包括：
# - MQTT 連接建立時間
# - 訊息處理吞吐量
# - 記憶體使用量
# - CPU 使用率
```

### 壓力測試

```bash
# 模擬大量設備
$RTK_BIN --simulate-devices 100 --config configs/controller.yaml &

# 監控性能
$RTK_BIN --cli --config configs/controller.yaml
rtk> system stats
rtk> device stats
```

### 故障模擬測試

```bash
# 模擬 MQTT 連接中斷
# 1. 啟動 controller
$RTK_BIN --cli --config configs/controller.yaml &

# 2. 停止 MQTT broker（如果你控制它）
# 3. 觀察 controller 的重連行為
# 4. 重新啟動 broker
# 5. 確認 controller 成功重連
```

## 故障排除

### 常見錯誤及解決方法

#### 1. 連接錯誤

**錯誤**: `Failed to connect to MQTT broker`

**排查步驟**:
```bash
# 檢查網路連接
ping your-broker-host

# 檢查埠號
telnet your-broker-host 1883

# 檢查配置
$RTK_BIN --config configs/controller.yaml --validate

# 測試連接
$RTK_BIN --config configs/controller.yaml --test-connection
```

#### 2. 認證錯誤

**錯誤**: `Authentication failed`

**解決方法**:
```bash
# 檢查用戶名和密碼
grep -A 10 "mqtt:" configs/controller.yaml

# 測試認證
mosquitto_pub -h your-broker-host -u your-username -P your-password -t test -m "test"
```

#### 3. 權限錯誤

**錯誤**: `Permission denied` 或 `Access denied`

**解決方法**:
```bash
# Linux/macOS: 檢查檔案權限
ls -la bin/rtk_controller-*
chmod +x bin/rtk_controller-*

# 檢查配置檔案權限
ls -la configs/controller.yaml

# 檢查資料目錄權限
mkdir -p data logs
chmod 755 data logs
```

#### 4. 配置錯誤

**錯誤**: `Invalid configuration` 或 `Configuration validation failed`

**解決方法**:
```bash
# 驗證 YAML 語法
python3 -c "import yaml; yaml.safe_load(open('configs/controller.yaml'))"

# 使用最小配置測試
$RTK_BIN --config configs/minimal.yaml --validate
```

### 除錯模式

```bash
# 啟用除錯日誌
$RTK_BIN --config configs/controller.yaml --debug

# 或設定環境變數
export RTK_LOG_LEVEL=debug
$RTK_BIN --config configs/controller.yaml
```

### 日誌分析

```bash
# 查看即時日誌
tail -f logs/controller.log

# 搜尋錯誤
grep -i error logs/controller.log

# 搜尋 MQTT 相關日誌
grep -i mqtt logs/controller.log

# 分析 JSON 格式日誌（如果使用 jq）
cat logs/controller.log | jq 'select(.level=="error")'
```

## 實例演練

### 演練 1：設置和基本操作

**目標**: 從零開始設置 RTK Controller 並執行基本操作

**步驟**:

1. **準備環境**
```bash
cd rtk_controller_release
export RTK_BIN="./bin/rtk_controller-linux-amd64"  # 選擇你的平台
```

2. **修改配置**
```bash
cp configs/controller.yaml configs/my_config.yaml
# 編輯 my_config.yaml，設定你的 MQTT broker 資訊
```

3. **驗證配置**
```bash
$RTK_BIN --config configs/my_config.yaml --validate
# 預期輸出: ✓ Configuration validation successful
```

4. **測試連接**
```bash
$RTK_BIN --config configs/my_config.yaml --test-connection
# 預期輸出: ✓ MQTT broker connection successful
```

5. **啟動 CLI 並探索**
```bash
$RTK_BIN --cli --config configs/my_config.yaml

# 在 CLI 中執行：
rtk> help
rtk> system status
rtk> config show
rtk> device list
rtk> exit
```

### 演練 2：設備監控

**目標**: 監控模擬設備並查看其狀態

**步驟**:

1. **啟動設備模擬器**（在一個終端中）
```bash
$RTK_BIN --simulate-device test_device_001 --config configs/my_config.yaml &
```

2. **監控設備**（在另一個終端中）
```bash
$RTK_BIN --cli --config configs/my_config.yaml

rtk> device list
# 應該看到 test_device_001

rtk> device status test_device_001
# 查看設備狀態

rtk> device info test_device_001
# 查看詳細資訊
```

3. **發送命令**
```bash
rtk> cmd send test_device_001 ping
rtk> cmd send test_device_001 get_status
rtk> cmd history
```

4. **查看診斷**
```bash
rtk> diagnosis run test_device_001
rtk> diagnosis report test_device_001
```

### 演練 3：故障診斷

**目標**: 模擬和診斷常見問題

**步驟**:

1. **模擬連接問題**
```bash
# 修改配置使用錯誤的 broker 地址
sed 's/localhost:1883/invalid-host:1883/' configs/my_config.yaml > configs/broken_config.yaml

$RTK_BIN --config configs/broken_config.yaml --test-connection
# 預期會失敗，觀察錯誤訊息
```

2. **模擬認證問題**
```bash
# 修改配置使用錯誤的認證資訊
sed 's/your_password/wrong_password/' configs/my_config.yaml > configs/auth_broken_config.yaml

$RTK_BIN --config configs/auth_broken_config.yaml --test-connection
# 預期會失敗，觀察錯誤訊息
```

3. **分析日誌**
```bash
# 啟用除錯模式
$RTK_BIN --config configs/broken_config.yaml --debug

# 查看詳細錯誤日誌
tail -f logs/controller.log | grep -i error
```

## 最佳實踐

### 配置管理

1. **使用版本控制**: 將配置檔案加入版本控制系統
2. **環境分離**: 為不同環境（開發、測試、生產）維護獨立配置
3. **敏感資訊**: 使用環境變數存儲密碼和金鑰
4. **配置驗證**: 部署前總是驗證配置檔案

### 監控和日誌

1. **日誌等級**: 生產環境使用 "info" 或 "warn"，除錯時使用 "debug"
2. **日誌輪轉**: 配置適當的日誌輪轉策略
3. **監控指標**: 定期檢查系統狀態和性能指標
4. **告警設置**: 為關鍵錯誤設置自動告警

### 安全考量

1. **最小權限**: RTK Controller 只給予必要的系統權限
2. **TLS 加密**: 生產環境總是啟用 TLS
3. **認證**: 使用強密碼和定期更換認證資訊
4. **網路隔離**: 限制 RTK Controller 的網路存取

### 效能最佳化

1. **資料庫**: 定期清理歷史資料
2. **記憶體**: 監控記憶體使用量，必要時調整配置
3. **連接**: 優化 MQTT 連接參數
4. **外掛程式**: 只載入必要的外掛程式

## 進階主題

### 自動化腳本

建立自動化腳本來簡化常見任務：

```bash
#!/bin/bash
# auto_deploy.sh - 自動部署腳本

RTK_BIN="./bin/rtk_controller-linux-amd64"
CONFIG_FILE="configs/production.yaml"

# 驗證配置
echo "驗證配置..."
$RTK_BIN --config $CONFIG_FILE --validate || exit 1

# 測試連接
echo "測試 MQTT 連接..."
$RTK_BIN --config $CONFIG_FILE --test-connection || exit 1

# 執行測試
echo "執行功能測試..."
./test/scripts/test_cli_commands.sh || exit 1

echo "部署成功！"
```

### 效能監控腳本

```bash
#!/bin/bash
# monitor.sh - 效能監控腳本

RTK_BIN="./bin/rtk_controller-linux-amd64"
CONFIG_FILE="configs/controller.yaml"

while true; do
    echo "=== $(date) ==="
    
    # 檢查系統狀態
    $RTK_BIN --config $CONFIG_FILE --cmd "system status"
    
    # 檢查設備統計
    $RTK_BIN --config $CONFIG_FILE --cmd "device stats"
    
    sleep 60
done
```

### 備份腳本

```bash
#!/bin/bash
# backup.sh - 資料備份腳本

BACKUP_DIR="./backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p $BACKUP_DIR

# 備份配置
cp -r configs $BACKUP_DIR/

# 備份資料庫
cp -r data $BACKUP_DIR/

# 備份日誌
cp -r logs $BACKUP_DIR/

echo "備份完成: $BACKUP_DIR"
```

## 常見問題 FAQ

**Q: RTK Controller 支援哪些作業系統？**
A: 支援 Linux (ARM64/x86_64)、macOS (ARM64) 和 Windows (x86_64)。

**Q: 如何更改日誌等級？**
A: 修改配置檔案中的 `logging.level` 或設定環境變數 `RTK_LOG_LEVEL`。

**Q: RTK Controller 可以同時連接多個 MQTT broker 嗎？**
A: 目前版本支援單一 broker 連接。多 broker 支援在未來版本中規劃。

**Q: 如何新增自訂設備類型？**
A: 使用外掛程式系統，參考 `plugins/examples/` 目錄中的範例。

**Q: RTK Controller 是否支援叢集部署？**
A: 可以部署多個實例，但目前不支援原生叢集功能。

**Q: 如何備份和恢復資料？**
A: 備份 `data/` 目錄和配置檔案。恢復時將檔案放回原位置即可。

**Q: 效能限制是什麼？**
A: 單一實例可以處理數千個設備，具體取決於硬體資源和網路條件。

## 結語

這份手冊涵蓋了 RTK Controller 的核心功能和使用方法。如果你遇到手冊中未涵蓋的問題，請：

1. 查看 `docs/` 目錄中的技術文檔
2. 檢查 `VERSION` 檔案中的版本說明
3. 聯繫技術支援，提供詳細的錯誤資訊和日誌

祝你使用 RTK Controller 順利！ 🚀
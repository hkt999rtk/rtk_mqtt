# RTK Controller 使用手冊

## 目錄

1. [系統簡介](#系統簡介)
2. [安裝與設定](#安裝與設定)
3. [啟動模式](#啟動模式)
4. [交互式 CLI 使用](#交互式-cli-使用)
5. [網絡拓撲管理](#網絡拓撲管理)
6. [網絡診斷功能](#網絡診斷功能)
7. [QoS 流量分析](#qos-流量分析)
8. [設備管理](#設備管理)
9. [命令管理](#命令管理)
10. [系統管理](#系統管理)
11. [Web Console](#web-console)
12. [配置說明](#配置說明)
13. [故障排除](#故障排除)
14. [進階功能](#進階功能)

## 系統簡介

RTK Controller 是一個功能完整的網絡管理系統，提供：
- 網絡拓撲自動發現與可視化
- 設備身份管理與分類
- 網絡性能診斷與監控
- QoS 策略分析與推薦
- MQTT 通訊與設備控制
- 交互式命令行介面
- Web 管理控制台

## 安裝與設定

### 系統需求

- **作業系統**: Linux、macOS、Windows
- **Go 版本**: 1.19 或更高
- **記憶體**: 至少 512MB
- **磁碟空間**: 至少 100MB

### 編譯安裝

```bash
# 下載源碼
git clone <repository_url>
cd rtk_controller

# 安裝依賴
go mod download

# 編譯執行檔
make build

# 或編譯所有平台版本
make build-all
```

### 目錄結構

安裝後會建立以下目錄：
```
rtk_controller/
├── rtk-controller      # 執行檔
├── configs/           # 配置文件
│   └── controller.yaml
├── data/              # 數據存儲
├── logs/              # 日誌文件
└── test/              # 測試腳本
```

## 啟動模式

### 1. 交互式 CLI 模式

最常用的模式，提供友善的命令行介面：

```bash
./rtk-controller --cli
```

啟動後會看到：
```
RTK Controller Interactive CLI
==============================
Version: 1.0.0
Type 'help' for available commands, 'exit' to quit

rtk> 
```

### 2. 服務模式

作為後台服務運行，提供 API 和 Web Console：

```bash
./rtk-controller --config configs/controller.yaml
```

### 3. 調試模式

啟用詳細日誌輸出：

```bash
./rtk-controller --cli --debug
```

## 交互式 CLI 使用

### 基本操作

#### 命令格式
```
rtk> <主命令> [子命令] [參數...]
```

#### 獲取幫助
```bash
rtk> help                    # 顯示所有可用命令
rtk> help topology          # 顯示 topology 命令的幫助
rtk> topology help          # 同上
```

#### 自動補全
- 按 **Tab** 鍵自動補全命令
- 支援多層級命令補全
- 補全包括命令、子命令和部分參數

#### 命令歷史
- **↑/↓** 箭頭鍵瀏覽歷史命令
- 歷史記錄保存在 `/tmp/.rtk_cli_history`
- 支援跨會話的歷史記錄

#### 快捷鍵
- **Ctrl+C**: 中斷當前輸入
- **Ctrl+D**: 退出 CLI（EOF）
- **Ctrl+L**: 清除螢幕

### 通用命令

```bash
rtk> version               # 顯示版本信息
rtk> status               # 顯示系統狀態
rtk> clear                # 清除螢幕
rtk> exit                 # 退出 CLI
rtk> quit                 # 退出 CLI（同 exit）
```

## 網絡拓撲管理

### 查看網絡拓撲

```bash
# 顯示完整拓撲圖
rtk> topology show

# 以 ASCII 圖形顯示
rtk> topology show ascii

# 顯示詳細信息
rtk> topology show detailed
```

### 設備列表與連接

```bash
# 列出所有設備
rtk> topology devices

# 顯示連接關係
rtk> topology connections

# 查看特定設備
rtk> topology device <device_id>
```

### WiFi 漫遊分析

```bash
# 查看漫遊歷史
rtk> topology roaming

# 分析漫遊模式
rtk> topology roaming analyze

# 查看特定設備的漫遊
rtk> topology roaming <device_id>
```

### 拓撲導出

```bash
# 導出為 DOT 格式（Graphviz）
rtk> topology export dot > network.dot

# 導出為 PlantUML
rtk> topology export plantuml > network.puml

# 導出為 JSON
rtk> topology export json > network.json
```

### 拓撲更新

```bash
# 手動刷新拓撲
rtk> topology refresh

# 從 MQTT 更新
rtk> topology update

# 清除拓撲數據
rtk> topology clear
```

## 網絡診斷功能

### 完整診斷測試

運行所有診斷測試：
```bash
rtk> diagnose full

# 或指定設備
rtk> diagnose full <device_id>
```

輸出示例：
```
Network Diagnostics Report
==========================
Device: router-main
Time:   2025-01-17 10:30:00

Speed Test:
-----------
  Download: 95.50 Mbps
  Upload:   45.20 Mbps
  Jitter:   2.50 ms
  Loss:     0.00%

WAN Status:
-----------
  Gateway:     ✓ OK (1.20 ms)
  External DNS: 8.50 ms
  Internet:    ✓ OK
  Public IP:   203.0.113.1

Latency Tests:
--------------
  8.8.8.8:        10.50 ms
  1.1.1.1:        12.30 ms
  google.com:     15.80 ms
```

### 速度測試

```bash
# 運行速度測試
rtk> test speedtest

# 使用特定服務器
rtk> test speedtest iperf3

# 指定測試時長
rtk> test speedtest --duration 30
```

### WAN 連接測試

```bash
# 測試 WAN 連接
rtk> test wan

# 測試特定 DNS
rtk> test wan --dns 8.8.8.8,1.1.1.1
```

### 延遲測試

```bash
# 測試預設目標
rtk> test latency

# 測試特定目標
rtk> test latency google.com cloudflare.com

# 詳細模式
rtk> test latency --verbose
```

### 定期診斷

```bash
# 設定每 30 分鐘運行一次
rtk> diagnose schedule 30m

# 查看排程狀態
rtk> diagnose schedules

# 停止排程
rtk> diagnose unschedule diagnostics

# 查看最後結果
rtk> diagnose last
```

## QoS 流量分析

### 流量統計

```bash
# 顯示當前流量統計
rtk> traffic show

# 查看設備流量歷史
rtk> traffic history <device_id>

# 顯示流量排行
rtk> traffic top

# 實時監控
rtk> traffic monitor
```

### 異常檢測

```bash
# 列出檢測到的異常
rtk> anomaly list

# 查看異常詳情
rtk> anomaly show <anomaly_id>

# 設定異常閾值
rtk> anomaly threshold 0.8

# 清除異常記錄
rtk> anomaly clear
```

### QoS 策略分析

```bash
# 分析並推薦 QoS 策略
rtk> qos analyze

# 查看推薦詳情
rtk> qos recommendations

# 應用推薦策略
rtk> qos apply <recommendation_id>

# 查看當前策略
rtk> qos policies
```

### 流量熱點

```bash
# 識別流量熱點設備
rtk> traffic hotspots

# 設定熱點閾值
rtk> traffic hotspot-threshold 80

# 導出熱點報告
rtk> traffic hotspot-report
```

## 設備管理

### 設備列表與查詢

```bash
# 列出所有設備
rtk> device list

# 查看設備詳情
rtk> device show <device_id>

# 搜尋設備
rtk> device search <keyword>

# 按類型篩選
rtk> device list --type router
```

### 設備身份管理

```bash
# 重命名設備
rtk> device rename <device_id> <new_name>

# 設定設備組
rtk> device group <device_id> <group_name>

# 添加標籤
rtk> device tag <device_id> <tag1> <tag2>

# 設定位置
rtk> device location <device_id> "Living Room"
```

### 設備狀態

```bash
# 查看設備狀態
rtk> device status <device_id>

# 查看歷史記錄
rtk> device history <device_id>

# 查看設備統計
rtk> device stats
```

## 命令管理

### 發送命令

```bash
# 發送重啟命令
rtk> command send <device_id> reboot

# 發送配置更新
rtk> command send <device_id> update_config

# 設定超時時間（秒）
rtk> command send <device_id> reboot 30
```

### 命令追蹤

```bash
# 列出所有命令
rtk> command list

# 查看命令詳情
rtk> command show <command_id>

# 查看待執行命令
rtk> command pending

# 取消命令
rtk> command cancel <command_id>
```

### 命令統計

```bash
# 顯示命令統計
rtk> command stats

# 查看成功率
rtk> command success-rate

# 查看失敗原因分析
rtk> command failures
```

## 系統管理

### 系統狀態

```bash
# 系統概況
rtk> system info

# 健康檢查
rtk> system health

# 性能統計
rtk> system stats

# 資源使用
rtk> system resources
```

### 配置管理

```bash
# 顯示當前配置
rtk> config show

# 重載配置
rtk> config reload

# 設定配置值
rtk> config set mqtt.broker "192.168.1.100"

# 驗證配置
rtk> config validate
```

### 日誌管理

```bash
# 查看最近日誌
rtk> log show

# 實時查看日誌
rtk> log tail

# 搜索日誌
rtk> log search "error"

# 下載日誌
rtk> log download 3600  # 下載最近 1 小時
```

### 測試工具

```bash
# 測試 MQTT 連接
rtk> test mqtt

# 測試存儲
rtk> test storage

# Ping 設備
rtk> test ping <device_id>

# 運行自檢
rtk> test self
```

## Web Console

### 啟動 Web Console

在服務模式下自動啟動：
```bash
./rtk-controller --config configs/controller.yaml
```

### 訪問介面

打開瀏覽器訪問：
```
http://localhost:8081
```

預設帳號密碼：
- 用戶名：`rtkadmin`
- 密碼：`admin123`

### REST API

API 端點基礎 URL：`http://localhost:8081/api/v1`

主要端點：
- `GET /api/v1/devices` - 獲取設備列表
- `GET /api/v1/topology` - 獲取拓撲信息
- `POST /api/v1/commands` - 發送命令
- `GET /api/v1/diagnostics` - 獲取診斷結果
- `WS /api/v1/ws` - WebSocket 連接

## 配置說明

### 主配置文件

配置文件位置：`configs/controller.yaml`

### MQTT 配置

```yaml
mqtt:
  broker: "localhost"        # MQTT broker 地址
  port: 1883                # MQTT 端口
  client_id: "rtk-controller" # 客戶端 ID
  username: ""              # 用戶名（可選）
  password: ""              # 密碼（可選）
  qos: 1                    # QoS 級別
  retain: false             # 保留消息
  clean_session: true       # 清潔會話
  keep_alive: 60            # 保活時間（秒）
```

### 診斷配置

```yaml
diagnostics:
  enable_speed_test: true    # 啟用速度測試
  enable_wan_test: true      # 啟用 WAN 測試
  enable_latency_test: true  # 啟用延遲測試
  test_interval: "30m"       # 測試間隔
  dns_servers:              # DNS 服務器列表
    - "8.8.8.8"
    - "1.1.1.1"
  speed_test_servers:       # 速度測試服務器
    - "speedtest.net"
    - "fast.com"
```

### QoS 配置

```yaml
qos:
  enable_anomaly_detection: true  # 啟用異常檢測
  anomaly_threshold: 0.8          # 異常閾值
  traffic_history_size: 100       # 歷史記錄大小
  hotspot_threshold: 0.7          # 熱點閾值
  analysis_interval: "5m"         # 分析間隔
```

### 存儲配置

```yaml
storage:
  path: "data"              # 存儲路徑
  backup_enabled: true      # 啟用備份
  backup_interval: "24h"    # 備份間隔
  max_backup_count: 7       # 最大備份數
```

### Web Console 配置

```yaml
console:
  host: "0.0.0.0"          # 監聽地址
  port: 8081               # 監聽端口
  auth:
    enabled: true          # 啟用認證
    default_username: "rtkadmin"
    default_password: "admin123"
    session_timeout: "30m" # 會話超時
```

### 日誌配置

```yaml
logging:
  level: "info"            # 日誌級別
  file: "logs/controller.log"
  max_size: 100            # MB
  max_backups: 10
  max_age: 30              # 天
  compress: true
```

## 故障排除

### 常見問題

#### 1. MQTT 連接失敗

**症狀**：
```
❌ MQTT connection is down
```

**解決方法**：
1. 確認 MQTT broker 正在運行
2. 檢查防火牆設置
3. 驗證配置文件中的連接參數
4. 查看日誌獲取詳細錯誤

#### 2. 存儲錯誤

**症狀**：
```
Failed to initialize storage
```

**解決方法**：
1. 確認 `data/` 目錄存在
2. 檢查目錄權限
3. 確保有足夠的磁碟空間
4. 刪除損壞的數據文件並重啟

#### 3. CLI 無法啟動

**症狀**：
```
Failed to start CLI
```

**解決方法**：
1. 檢查配置文件是否存在
2. 驗證配置文件語法
3. 確認端口未被占用
4. 查看日誌文件

#### 4. 診斷測試失敗

**症狀**：
```
Diagnostic test failed
```

**解決方法**：
1. 檢查網絡連接
2. 確認測試工具已安裝（iperf3、curl）
3. 驗證 DNS 設置
4. 檢查防火牆規則

### 日誌位置

- **應用日誌**: `logs/controller.log`
- **審計日誌**: `logs/audit.log`
- **性能日誌**: `logs/performance.log`
- **CLI 歷史**: `/tmp/.rtk_cli_history`

### 調試技巧

1. **啟用調試模式**：
   ```bash
   ./rtk-controller --cli --debug
   ```

2. **查看詳細日誌**：
   ```bash
   rtk> log tail
   ```

3. **運行自檢**：
   ```bash
   rtk> test self
   ```

4. **檢查系統健康**：
   ```bash
   rtk> system health
   ```

## 進階功能

### 批量操作

使用管道和腳本執行批量操作：

```bash
# 批量重命名設備
echo -e "device rename dev1 Router1\ndevice rename dev2 Router2" | ./rtk-controller --cli

# 從文件執行命令
./rtk-controller --cli < commands.txt
```

### 數據導出

```bash
# 導出完整配置
rtk> export config > backup.yaml

# 導出拓撲數據
rtk> topology export json > topology.json

# 導出診斷報告
rtk> diagnose report > report.txt
```

### 自動化腳本

創建自動化腳本 `auto_diagnose.sh`：
```bash
#!/bin/bash
./rtk-controller --cli << EOF
diagnose full
traffic show
anomaly list
exit
EOF
```

### 與其他系統集成

使用 REST API 進行集成：
```bash
# 獲取設備列表
curl http://localhost:8081/api/v1/devices

# 發送命令
curl -X POST http://localhost:8081/api/v1/commands \
  -H "Content-Type: application/json" \
  -d '{"device_id":"router1","operation":"reboot"}'
```

### 性能優化

1. **調整測試間隔**：
   ```yaml
   diagnostics:
     test_interval: "60m"  # 減少測試頻率
   ```

2. **限制歷史記錄**：
   ```yaml
   qos:
     traffic_history_size: 50  # 減少記憶體使用
   ```

3. **優化日誌**：
   ```yaml
   logging:
     level: "warn"  # 減少日誌輸出
   ```

## 附錄

### 命令快速參考

| 類別 | 命令 | 說明 |
|------|------|------|
| 拓撲 | `topology show` | 顯示網絡拓撲 |
| 拓撲 | `topology devices` | 列出所有設備 |
| 拓撲 | `topology roaming` | 查看漫遊記錄 |
| 診斷 | `diagnose full` | 完整診斷測試 |
| 診斷 | `test speedtest` | 速度測試 |
| 診斷 | `test wan` | WAN 測試 |
| 流量 | `traffic show` | 顯示流量統計 |
| 流量 | `anomaly list` | 列出異常 |
| QoS | `qos analyze` | 分析 QoS |
| 設備 | `device list` | 列出設備 |
| 設備 | `device show <id>` | 顯示設備詳情 |
| 命令 | `command send` | 發送命令 |
| 系統 | `system health` | 健康檢查 |
| 配置 | `config reload` | 重載配置 |

### 版本歷史

- **v1.0.0** (2025-01-17)
  - 初始版本發布
  - 完整 8 階段功能實現
  - 支援拓撲檢測、診斷、QoS

---

**文檔版本**: 1.0.0  
**最後更新**: 2025-01-17  
**作者**: RTK Controller Team
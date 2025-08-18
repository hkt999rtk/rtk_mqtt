# RTK Controller 使用手冊

> **工程師快速部署指南** - 本手冊專為拿到 release binary package 的工程師而設計

## 目錄

1. [快速開始](#快速開始)
2. [系統需求](#系統需求)
3. [安裝部署](#安裝部署)
4. [基本使用](#基本使用)
5. [配置說明](#配置說明)
6. [功能驗證](#功能驗證)
7. [故障排除](#故障排除)
8. [維護運行](#維護運行)

## 快速開始

### 1. 解壓縮發行包

```bash
# 解壓縮發行包
tar -xzf rtk_controller-[版本]_[平台].tar.gz
cd rtk_controller-[平台]/

# 檢查包內容
ls -la
```

發行包包含：
```
rtk_controller-[平台]/
├── bin/                    # 可執行檔案
│   └── rtk_controller-*    # 對應平台的執行檔
├── configs/                # 配置檔案
│   └── controller.yaml     # 主配置檔案
├── docs/                   # 技術文檔
├── test-tools/             # 測試工具 (可選)
├── test/scripts/           # 測試腳本
├── demo_cli.sh             # 功能演示腳本
├── MANUAL.md               # 本手冊
├── LICENSE                 # 許可證
└── VERSION                 # 版本資訊
```

### 2. 選擇對應平台執行檔

根據您的系統選擇正確的執行檔：

**Linux ARM64 (樹莓派、ARM 伺服器)**
```bash
cp bin/rtk_controller-linux-arm64 ./rtk_controller
chmod +x rtk_controller
```

**Linux x86_64 (標準 Linux 伺服器)**
```bash
cp bin/rtk_controller-linux-amd64 ./rtk_controller
chmod +x rtk_controller
```

**macOS ARM64 (Apple Silicon Mac)**
```bash
cp bin/rtk_controller-darwin-arm64 ./rtk_controller
chmod +x rtk_controller
```

**Windows x86_64**
```cmd
copy bin\rtk_controller-windows-amd64.exe rtk_controller.exe
```

### 3. 立即測試

```bash
# 檢查版本
./rtk_controller --version

# 啟動交互式 CLI
./rtk_controller --cli
```

成功啟動會顯示：
```
RTK Controller Interactive CLI
==============================
Version: [版本號]
Type 'help' for available commands, 'exit' to quit

rtk> 
```

## 系統需求

### 硬體需求
- **CPU**: 雙核心 1GHz 以上
- **記憶體**: 最少 512MB，建議 1GB 以上
- **磁碟空間**: 最少 100MB 可用空間
- **網路**: 支援 TCP/IP 網路連接

### 作業系統支援
- **Linux**: Ubuntu 18.04+, CentOS 7+, Debian 9+
- **macOS**: macOS 10.15+ (Catalina)
- **Windows**: Windows 10, Windows Server 2016+

### 網路需求
- **MQTT Broker**: 需要可連接的 MQTT broker (如 Mosquitto)
- **連接埠**: 
  - MQTT: 預設 1883 (可設定)
  - MQTT over TLS: 預設 8883 (可設定)

## 安裝部署

### 生產環境部署

#### 1. 建立專用用戶 (Linux/macOS)

```bash
# 建立 rtk 用戶
sudo useradd -m -s /bin/bash rtk
sudo passwd rtk

# 切換到 rtk 用戶
sudo su - rtk

# 建立工作目錄
mkdir -p ~/rtk_controller
cd ~/rtk_controller
```

#### 2. 部署執行檔

```bash
# 將發行包複製到部署目錄
tar -xzf rtk_controller-*.tar.gz --strip-components=1

# 設定執行權限
chmod +x bin/rtk_controller-*

# 建立符號連結
ln -sf bin/rtk_controller-[您的平台] rtk_controller
```

#### 3. 建立系統服務 (Linux)

建立 systemd 服務檔案：

```bash
sudo tee /etc/systemd/system/rtk-controller.service > /dev/null <<EOF
[Unit]
Description=RTK Controller Network Management System
After=network.target

[Service]
Type=simple
User=rtk
WorkingDirectory=/home/rtk/rtk_controller
ExecStart=/home/rtk/rtk_controller/rtk_controller --config configs/controller.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 啟用並啟動服務
sudo systemctl daemon-reload
sudo systemctl enable rtk-controller
sudo systemctl start rtk-controller

# 檢查服務狀態
sudo systemctl status rtk-controller
```

## 基本使用

### 交互式 CLI 模式

最適合初次使用和測試：

```bash
./rtk_controller --cli
```

#### 常用命令

```bash
# 顯示所有可用命令
rtk> help

# 檢查系統狀態
rtk> system status

# 查看網絡拓撲
rtk> topology show

# 列出設備
rtk> device list

# 執行網絡診斷
rtk> diagnostics run speed-test

# 查看 QoS 統計
rtk> qos show stats

# 重新載入配置
rtk> config reload

# 退出
rtk> exit
```

### 服務模式

適合生產環境長期運行：

```bash
# 前台運行 (測試用)
./rtk_controller --config configs/controller.yaml

# 後台運行
nohup ./rtk_controller --config configs/controller.yaml > logs/controller.log 2>&1 &
```

### 常用操作

#### 快速健康檢查
```bash
# 檢查連線狀態
./rtk_controller --cli --execute "system status"

# 檢查 MQTT 連線
./rtk_controller --cli --execute "mqtt status"
```

#### 匯出拓撲資料
```bash
# 匯出為 JSON
./rtk_controller --cli --execute "topology export --format json --output topology.json"
```

## 配置說明

主配置檔案：`configs/controller.yaml`

### 基本配置

```yaml
# MQTT 連線設定
mqtt:
  broker: "localhost"        # MQTT broker 地址
  port: 1883                # MQTT 連接埠
  client_id: "rtk-controller"
  username: ""              # MQTT 用戶名 (可選)
  password: ""              # MQTT 密碼 (可選)

# 資料存儲
storage:
  path: "data"              # 資料目錄

# 日誌設定
logging:
  level: "info"             # debug, info, warn, error
  file: "logs/controller.log"
```

### 進階配置

```yaml
# TLS 加密 (生產環境建議)
mqtt:
  tls:
    enabled: true
    cert_file: "certs/client.crt"
    key_file: "certs/client.key"
    ca_file: "certs/ca.crt"

# 診斷設定
diagnosis:
  enabled: true
  default_analyzers:
    - "builtin_wifi_analyzer"
```

### 環境變數覆蓋

```bash
# 覆蓋 MQTT broker 地址
export RTK_MQTT_BROKER=192.168.1.100
export RTK_MQTT_PORT=1883

# 啟動
./rtk_controller --config configs/controller.yaml
```

## 功能驗證

### 1. 基本功能測試

執行演示腳本：
```bash
./demo_cli.sh
```

### 2. 連線測試

```bash
./rtk_controller --cli --execute "mqtt connect"
```

### 3. 使用測試工具

如果發行包包含測試工具：

```bash
# 基本 MQTT 功能測試
./test-tools/mqtt_client

# 拓撲測試
./test-tools/test_topology_simple

# 診斷測試
./test-tools/test_diagnostics
```

### 4. 手動驗證步驟

1. **檢查服務啟動**
   ```bash
   ps aux | grep rtk_controller
   ```

2. **檢查日誌**
   ```bash
   tail -f logs/controller.log
   ```

3. **檢查資料目錄**
   ```bash
   ls -la data/
   ```

4. **檢查網路連線**
   ```bash
   netstat -tulpn | grep rtk_controller
   ```

## 故障排除

### 常見問題

#### 1. 執行檔無法啟動

**問題**: `Permission denied` 或 `Command not found`

**解決方案**:
```bash
# 檢查檔案權限
ls -la rtk_controller

# 設定執行權限
chmod +x rtk_controller

# 檢查是否為正確平台
file rtk_controller
```

#### 2. MQTT 連線失敗

**問題**: `Failed to connect to MQTT broker`

**解決方案**:
```bash
# 檢查 broker 是否運行
telnet [broker_ip] 1883

# 檢查配置檔案
cat configs/controller.yaml | grep -A 10 mqtt

# 測試基本連線
mosquitto_pub -h [broker_ip] -p 1883 -t test -m "hello"
```

#### 3. 權限錯誤

**問題**: `Permission denied` 存取 data/ 或 logs/ 目錄

**解決方案**:
```bash
# 建立目錄並設定權限
mkdir -p data logs
chmod 755 data logs

# 或以 root 身份執行 (不建議生產環境)
sudo ./rtk_controller --config configs/controller.yaml
```

#### 4. 配置檔案錯誤

**問題**: `Failed to parse config file`

**解決方案**:
```bash
# 驗證 YAML 語法
python3 -c "import yaml; yaml.safe_load(open('configs/controller.yaml'))"

# 重置為預設配置
cp configs/controller.yaml.example configs/controller.yaml
```

### 診斷命令

```bash
# 檢查系統資源
./rtk_controller --cli --execute "system info"

# 檢查配置
./rtk_controller --cli --execute "config show"

# 檢查 MQTT 狀態
./rtk_controller --cli --execute "mqtt status"

# 檢查儲存狀態
./rtk_controller --cli --execute "storage status"
```

### 日誌分析

```bash
# 檢查錯誤日誌
grep -i error logs/controller.log

# 檢查警告
grep -i warning logs/controller.log

# 即時監控
tail -f logs/controller.log | grep -E "(ERROR|WARN|FATAL)"
```

## 維護運行

### 日常維護

#### 1. 日誌輪轉

```bash
# 手動輪轉日誌
mv logs/controller.log logs/controller.log.$(date +%Y%m%d)
kill -USR1 $(pgrep rtk_controller)  # 重新開啟日誌檔案
```

#### 2. 資料備份

```bash
# 備份資料目錄
tar -czf backup/rtk_data_$(date +%Y%m%d).tar.gz data/

# 定期清理舊備份 (保留 7 天)
find backup/ -name "rtk_data_*.tar.gz" -mtime +7 -delete
```

#### 3. 效能監控

```bash
# 檢查記憶體使用
ps aux | grep rtk_controller | awk '{print $4"%", $6/1024"MB"}'

# 檢查 CPU 使用
top -p $(pgrep rtk_controller)
```

### 版本升級

```bash
# 停止服務
sudo systemctl stop rtk-controller

# 備份當前版本
cp rtk_controller rtk_controller.backup

# 替換執行檔
cp bin/rtk_controller-[新平台] rtk_controller

# 重新啟動
sudo systemctl start rtk-controller

# 檢查版本
./rtk_controller --version
```

### 安全建議

1. **不使用 root 權限運行**
2. **啟用 TLS 加密**
3. **定期更新密碼**
4. **監控異常連線**
5. **定期備份資料**

---

## 技術支援

**版本資訊**: 請執行 `./rtk_controller --version` 取得詳細版本資訊

**配置檔案**: 詳細配置說明請參考 `configs/controller.yaml` 中的註釋

**技術文檔**: 更多技術細節請參考 `docs/` 目錄

**問題回報**: 請提供以下資訊：
- 執行環境 (OS, 版本)
- 錯誤訊息
- 相關日誌
- 配置檔案 (去除敏感資訊)

---

**發行包版本**: 請查看 `VERSION` 檔案  
**授權許可**: 請查看 `LICENSE` 檔案  
**最後更新**: 2025-08-18
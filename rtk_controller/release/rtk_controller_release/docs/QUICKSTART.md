# RTK Controller 快速開始指南

## 前置需求

- Go 1.19 或更高版本
- Git

## 快速安裝和啟動

### 1. 編譯程式
```bash
# 在項目根目錄執行
go mod tidy
go build -o rtk-controller ./cmd/controller
```

### 2. 檢查編譯結果
```bash
./rtk-controller --version
```
應該看到類似輸出：
```
RTK Controller dev (built at unknown)
```

### 3. 啟動交互式 CLI
```bash
./rtk-controller --cli
```

你會看到：
```
RTK Controller Interactive CLI
==============================
Version: 1.0.0
Type 'help' for available commands, 'exit' to quit

rtk> 
```

### 4. 嘗試一些基本命令
```bash
# 查看幫助
rtk> help

# 查看系統狀態
rtk> status

# 查看系統健康狀態
rtk> system health

# 測試存儲連接
rtk> test storage

# 查看設備列表（應該是空的）
rtk> device list

# 查看命令統計
rtk> command stats

# 退出 CLI
rtk> exit
```

## 運行演示

執行預製的演示腳本：
```bash
./demo_cli.sh
```

這會自動運行一系列命令來展示 CLI 的基本功能。

## 啟動完整服務（可選）

如果你想啟動完整的服務模式（包括 Web Console 和 API）：

1. 確保配置檔案存在：
```bash
ls configs/controller.yaml
```

2. 啟動服務：
```bash
./rtk-controller --config configs/controller.yaml
```

注意：由於沒有 MQTT broker 運行，會看到連接錯誤，這是正常的。

## 常見問題

### Q: 看到 "MQTT connection is down" 錯誤
A: 這是正常的，因為沒有 MQTT broker 運行。如果你有 MQTT broker（如 mosquitto），可以修改 `configs/controller.yaml` 中的連接設定。

### Q: CLI 無法啟動
A: 確保：
- 配置檔案 `configs/controller.yaml` 存在
- 有寫入 `data/` 目錄的權限
- 檢查 `logs/controller.log` 中的錯誤信息

### Q: 如何退出 CLI？
A: 輸入 `exit` 或 `quit`，或按 Ctrl+D

## 下一步

- 閱讀 [CLI_USAGE.md](CLI_USAGE.md) 了解所有可用命令
- 查看 [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) 了解完整功能
- 修改 `configs/controller.yaml` 配置你的 MQTT broker

## 快速測試清單

運行以下命令確保基本功能正常：
```bash
# 1. 編譯
go build -o rtk-controller ./cmd/controller

# 2. 檢查版本
./rtk-controller --version

# 3. 運行演示
./demo_cli.sh

# 4. 手動測試關鍵命令
echo -e "status\nsystem health\ntest storage\nexit" | ./rtk-controller --cli
```

如果以上步驟都能成功運行，說明 RTK Controller 安裝正確！
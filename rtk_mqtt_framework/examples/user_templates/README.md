# RTK MQTT Framework 使用者範例

歡迎使用 RTK MQTT Framework 使用者範例集！這些範例提供了從基礎到進階的完整學習路徑，幫助您快速掌握 RTK MQTT Framework 的使用和最佳實踐。

## 📚 學習路徑

這些範例按照由淺入深的順序設計，建議依序學習：

### 🚀 [01_hello_world](01_hello_world/) - MQTT 基礎入門
**學習時間**: 30 分鐘  
**難度**: ⭐  

```
最簡單的 MQTT 客戶端範例，僅 20 行代碼
├── main.c          # 極簡 MQTT 發布範例
├── Makefile        # 獨立建構系統
└── README.md       # 詳細說明和故障排除
```

**您將學到**:
- RTK MQTT Framework 基本用法
- 連接 MQTT broker 和發布訊息
- 獨立 Makefile 的使用
- 基礎故障排除技巧

---

### 📊 [02_basic_sensor](02_basic_sensor/) - 感測器模擬
**學習時間**: 1-2 小時  
**難度**: ⭐⭐  

```
持續運行的感測器設備模擬
├── sensor.c        # 感測器模擬和週期發布
├── Makefile        # 進階建構系統 (debug/release/daemon)
├── config.json     # JSON 配置檔案範例
└── README.md       # 深入的代碼解析和最佳化指南
```

**您將學到**:
- 週期性遙測資料發布
- RTK 主題結構的使用
- JSON 配置檔案管理
- 信號處理和優雅退出
- 背景執行和程序管理

---

### 🏭 [03_complete_device](03_complete_device/) - 生產級設備
**學習時間**: 3-4 小時  
**難度**: ⭐⭐⭐⭐  

```
企業級 IoT 設備完整實作
├── device.c        # 多執行緒架構 (390+ 行)
├── Makefile        # 企業級建構系統
├── config.json     # 完整配置檔案
└── README.md       # 生產級開發指南
```

**您將學到**:
- 多執行緒架構設計 (感測器/命令/健康監控)
- 完整的錯誤處理和恢復機制
- 企業級建構系統 (靜態分析、記憶體檢查、打包)
- 配置檔案管理和日誌系統
- 健康監控和自動重連
- 優雅啟動和關閉程序

---

### 🌐 [04_cross_platform](04_cross_platform/) - 跨平台開發
**學習時間**: 2-3 小時  
**難度**: ⭐⭐⭐⭐⭐  

```
多平台相容的設備程式
├── cross_platform_device.c  # 平台抽象層實作
├── Makefile                 # 多平台建構系統
└── README.md                # 跨平台開發指南
```

**支援平台**:
- **POSIX** (Linux/macOS) - pthread, BSD socket
- **Windows** - WinAPI, Winsock2  
- **ARM FreeRTOS** - FreeRTOS Tasks, LwIP

**您將學到**:
- 平台抽象層設計模式
- 條件編譯和平台檢測
- 多平台建構系統配置
- 執行緒和同步的平台差異
- 嵌入式系統程式設計最佳實踐

## 🛠️ 使用方式

### 快速開始

```bash
# 1. 進入任何一個範例目錄
cd 01_hello_world

# 2. 檢查建構環境
make check-rtk

# 3. 建構和執行
make run
```

### 所有範例都支援的通用命令

```bash
make              # 基本建構
make run          # 建構並執行
make debug        # 除錯版本
make release      # 最佳化版本
make clean        # 清理建構產物
make help         # 顯示詳細說明
```

### 範例特有功能

| 功能 | 01_hello | 02_sensor | 03_complete | 04_cross |
|------|----------|-----------|-------------|----------|
| 基本建構 | ✅ | ✅ | ✅ | ✅ |
| 除錯模式 | ✅ | ✅ | ✅ | ✅ |
| 背景執行 | ❌ | ✅ | ✅ | ❌ |
| 配置檔案 | ❌ | ✅ | ✅ | ❌ |
| 多執行緒 | ❌ | ❌ | ✅ | ✅ |
| 跨平台建構 | ❌ | ❌ | ❌ | ✅ |
| 靜態分析 | ❌ | ❌ | ✅ | ❌ |
| 記憶體檢查 | ❌ | ❌ | ✅ | ❌ |
| 分發打包 | ❌ | ❌ | ✅ | ❌ |

## 📋 系統需求

### 基本需求 (所有範例)

```bash
# RTK MQTT Framework (已預先建構)
├── include/rtk_mqtt_framework/    # 標頭檔
└── lib/librtk_mqtt_framework.a    # 函式庫

# 編譯工具
gcc 4.8+                          # C 編譯器
make 3.8+                         # 建構工具

# 系統函式庫
pthread                           # 執行緒支援 (POSIX)
libm                             # 數學函式庫
```

### 跨平台建構需求 (僅 04_cross_platform)

```bash
# Windows 交叉編譯
gcc-mingw-w64                     # MinGW 工具鏈

# ARM FreeRTOS 交叉編譯  
gcc-arm-none-eabi                 # ARM GCC 工具鏈
/opt/freertos                     # FreeRTOS 源碼 (設定 FREERTOS_PATH)
```

### 測試和分析工具 (選用)

```bash
# 品質保證工具
valgrind                          # 記憶體檢查
cppcheck                          # 靜態分析  
clang-format                      # 代碼格式化

# MQTT 測試工具
mosquitto-clients                 # mosquitto_pub/sub
```

## 🧪 測試和驗證

### 快速驗證所有範例

```bash
#!/bin/bash
# 測試所有範例的腳本

examples=("01_hello_world" "02_basic_sensor" "03_complete_device" "04_cross_platform")

for example in "${examples[@]}"; do
    echo "測試 $example..."
    cd "$example"
    
    if make test 2>/dev/null; then
        echo "✅ $example 測試通過"
    else
        echo "❌ $example 測試失敗"
    fi
    
    cd ..
    echo ""
done
```

### MQTT 通信驗證

```bash
# 監聽所有範例的 MQTT 訊息
mosquitto_sub -h test.mosquitto.org -t "rtk/v1/+/+/+/+"

# 預期看到的主題格式:
# rtk/v1/demo_tenant/demo_site/hello_world_001/telemetry/message
# rtk/v1/demo_tenant/demo_site/basic_sensor_001/telemetry/temperature  
# rtk/v1/demo_tenant/demo_site/complete_device_001/state
# rtk/v1/demo_tenant/demo_site/cross_platform_Linux_001/evt/platform.info
```

## 🎯 選擇適合的範例

### 根據您的需求選擇

| 如果您想要... | 推薦範例 | 原因 |
|---------------|----------|------|
| 快速了解 RTK 框架 | `01_hello_world` | 最簡單，5 分鐘即可運行 |
| 建立感測器設備 | `02_basic_sensor` | 完整的感測器模擬和配置管理 |
| 開發生產級設備 | `03_complete_device` | 企業級功能和建構系統 |
| 支援多個平台 | `04_cross_platform` | 跨平台抽象層和建構支援 |
| 學習 MQTT 協議 | `01_hello_world` → `02_basic_sensor` | 循序漸進了解 RTK 主題結構 |
| 學習多執行緒程式設計 | `03_complete_device` | 生產級多執行緒架構範例 |
| 嵌入式系統開發 | `04_cross_platform` | FreeRTOS 支援和資源最佳化 |

### 根據您的經驗水平

| 經驗水平 | 建議學習路徑 | 預估時間 |
|----------|--------------|----------|
| **初學者** | `01` → `02` → 暫停複習 | 2-3 小時 |
| **中級開發者** | `01` → `02` → `03` | 4-6 小時 |
| **高級開發者** | 全部範例 + 自訂修改 | 6-8 小時 |
| **架構師/技術主管** | 重點關注 `03` 和 `04` | 3-4 小時 |

## 🔧 常見問題

### Q: 如何確認 RTK 框架已正確安裝？

**A**: 執行任何範例的檢查命令：
```bash
cd 01_hello_world
make check-rtk
```

### Q: 編譯時提示找不到標頭檔？

**A**: 檢查 RTK 框架路徑設定：
```bash
ls -la ../../include/rtk_mqtt_framework/
ls -la ../../lib/librtk_mqtt_framework.a

# 如果路徑不正確，調整 Makefile 中的路徑
RTK_INCLUDE_DIR = /path/to/your/rtk/include
RTK_LIB_DIR = /path/to/your/rtk/lib
```

### Q: 無法連接到 MQTT broker？

**A**: 檢查網路連接：
```bash
# 測試連接性
ping test.mosquitto.org
telnet test.mosquitto.org 1883

# 檢查防火牆設定
sudo ufw status

# 嘗試其他 broker
# 編輯源碼中的 broker 位址為您的私有 broker
```

### Q: 想使用自己的 MQTT broker？

**A**: 修改各範例中的 broker 設定：
```c
// 在源碼中找到並修改
rtk_mqtt_client_t* client = rtk_mqtt_client_create(
    "your.mqtt.broker.com",    // 您的 broker 位址
    1883,                      // 埠號
    "your_device_id"           // 設備 ID
);
```

### Q: 如何添加新的感測器類型？

**A**: 參考 `02_basic_sensor` 的模式：
```c
// 1. 添加感測器讀取函式
float read_your_sensor(void) {
    // 您的感測器邏輯
    return sensor_value;
}

// 2. 在發布函式中添加
if (rtk_mqtt_client_publish_telemetry(client, "your_metric", 
                                     read_your_sensor(), "unit") == RTK_SUCCESS) {
    printf("  ✅ 您的感測器: %.1f unit\n", read_your_sensor());
    success_count++;
}
```

## 🚀 進階使用

### 自訂範例開發

```bash
# 1. 複製最相似的範例
cp -r 02_basic_sensor my_custom_device

# 2. 修改專案資訊
cd my_custom_device
sed -i 's/basic_sensor/my_custom_device/g' Makefile
sed -i 's/basic_sensor/my_custom_device/g' *.c

# 3. 根據需求修改功能
nano sensor.c  # 修改感測器邏輯
nano config.json  # 調整配置
```

### 整合到您的專案

```bash
# 1. 複製獨立 Makefile 模板
cp 01_hello_world/Makefile your_project/

# 2. 調整 RTK 框架路徑
# 編輯 Makefile 中的路徑設定

# 3. 參考範例代碼模式
# 使用範例中的最佳實踐
```

### 建立分發版本

```bash
# 使用 03_complete_device 的打包功能
cd 03_complete_device
make package

# 查看生成的分發包
ls -la dist/
```

## 🔗 相關資源

### 官方文檔
- **[MANUAL.md](../../docs/MANUAL.md)** - 完整使用手冊
- **[README.md](../../README.md)** - 開發者指南  
- **[RTK MQTT 協議規範](../../docs/SPEC.md)** - 協議詳細說明

### 外部資源
- **[MQTT 協議規範](https://mqtt.org/)** - MQTT 官方文檔
- **[Eclipse Paho](https://www.eclipse.org/paho/)** - MQTT 客戶端實作
- **[test.mosquitto.org](https://test.mosquitto.org/)** - 免費測試 broker

### 社群和支援
- **問題回報**: 請在專案 repository 中提交 issue
- **功能建議**: 歡迎提交 pull request
- **技術討論**: 參考各範例的 README.md 中的故障排除章節

## 🎉 開始您的 RTK 之旅！

選擇一個範例開始您的學習之旅：

```bash
# 最簡單的開始方式
cd 01_hello_world
make run

# 看到 "Hello, RTK MQTT Framework!" 消息後，
# 您就正式開始了 RTK MQTT Framework 的學習！
```

祝您學習愉快！如果遇到任何問題，請參考各範例的詳細 README 或聯繫技術支援團隊。
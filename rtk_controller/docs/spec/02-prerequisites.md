# 前置要求與依賴程式庫

## 概述

在開始實作 RTK MQTT 協議前，請先確保您的開發環境已正確安裝以下經過測試驗證的第三方程式庫。這些程式庫提供了協議實作所需的核心功能，包括 JSON 處理和 MQTT 通信支援。

## 必要依賴程式庫

### 1. cJSON - JSON 解析程式庫

**用途**: 處理 MQTT payload 中的 JSON 訊息格式，包括編碼、解碼和驗證

**官方網站**: https://github.com/DaveGamble/cJSON  
**推薦版本**: v1.7.15 或更新版本  
**授權**: MIT License

#### 安裝方式

##### Linux 發行版 (推薦)
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install libcjson-dev

# CentOS/RHEL/Fedora
sudo yum install cjson-devel
# 或使用 dnf (較新版本)
sudo dnf install cjson-devel

# Alpine Linux
apk add cjson-dev
```

##### 從原始碼編譯 (適用於嵌入式系統)
```bash
git clone https://github.com/DaveGamble/cJSON.git
cd cJSON
mkdir build && cd build
cmake .. -DENABLE_CJSON_UTILS=On -DBUILD_SHARED_LIBS=On
make
sudo make install
sudo ldconfig  # 更新動態程式庫連結
```

##### FreeRTOS 整合
```c
// 將 cJSON 原始碼直接加入專案
#include "cJSON.h"

// 記憶體管理配置 (可選)
void cJSON_InitHooks(cJSON_Hooks* hooks);
```

#### 功能驗證
```bash
# 驗證安裝是否成功
pkg-config --cflags --libs libcjson
# 預期輸出: -I/usr/include -lcjson

# 簡單測試
echo '{"test": "value"}' | cJSON_test
```

### 2. libmosquitto - MQTT 客戶端程式庫

**用途**: 提供 MQTT 協議的完整實作，包括連接、發布、訂閱和 LWT 機制

**官方網站**: https://mosquitto.org/  
**推薦版本**: v2.0.15 或更新版本  
**授權**: EPL-2.0 / EDL-1.0

#### 安裝方式

##### Linux 發行版 (推薦)
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install libmosquitto-dev mosquitto-clients

# CentOS/RHEL/Fedora
sudo yum install mosquitto-devel
# 或使用 dnf
sudo dnf install mosquitto-devel

# Alpine Linux
apk add mosquitto-dev mosquitto-clients
```

##### 從原始碼編譯
```bash
# 下載並解壓縮
wget https://mosquitto.org/files/source/mosquitto-2.0.15.tar.gz
tar -xzf mosquitto-2.0.15.tar.gz
cd mosquitto-2.0.15

# 編譯配置 (可選 SSL 支援)
make WITH_TLS=yes WITH_WEBSOCKETS=yes
sudo make install
sudo ldconfig

# 創建配置目錄
sudo mkdir -p /etc/mosquitto
sudo cp mosquitto.conf /etc/mosquitto/
```

##### 嵌入式系統整合

**Realtek Ameba SDK**:
```c
// 直接使用 AmebaSDK 內建的 MQTT 實作
#include "mqtt/MQTTClient.h"
#include "mqtt/MQTTFreertos.h"
```

**ESP-IDF**:
```c
// 使用 ESP-MQTT 組件
#include "mqtt_client.h"
```

#### 功能驗證
```bash
# 驗證 mosquitto 客戶端工具
mosquitto_pub --help
mosquitto_sub --help

# 測試本地連接 (需要 mosquitto broker 運行)
mosquitto_pub -h localhost -t "test/topic" -m "Hello RTK MQTT"
mosquitto_sub -h localhost -t "test/topic"
```

## 平台相容性

### 支援的作業系統

| 平台 | cJSON | libmosquitto | 備註 |
|------|-------|-------------|------|
| Linux (Ubuntu 18.04+) | ✅ | ✅ | 主要開發平台 |
| Linux (CentOS/RHEL 7+) | ✅ | ✅ | 企業級部署 |
| Windows 10/11 | ✅ | ✅ | 需要 MinGW 或 MSVC |
| macOS (10.14+) | ✅ | ✅ | 使用 Homebrew 安裝 |
| FreeRTOS | ✅ | ⚠️ | 需要第三方 MQTT 實作 |
| ESP32/ESP8266 | ✅ | ✅ | ESP-IDF 框架支援 |
| Realtek Ameba | ✅ | ✅ | SDK 內建支援 |

### 編譯器要求

- **GCC**: 4.8+ (建議 7.0+)
- **Clang**: 3.4+ (建議 6.0+)  
- **MSVC**: 2015+ (Visual Studio 2015)
- **ARM GCC**: 6.0+ (嵌入式開發)

## 開發環境設定

### 1. 編譯參數配置

```makefile
# Makefile 範例
CFLAGS += -std=c99 -Wall -Wextra
CFLAGS += $(shell pkg-config --cflags libcjson)
LDFLAGS += $(shell pkg-config --libs libcjson)
LDFLAGS += -lmosquitto

# 除錯模式
ifdef DEBUG
    CFLAGS += -g -DDEBUG
endif

# 發布模式
ifdef RELEASE  
    CFLAGS += -O2 -DNDEBUG
endif
```

### 2. CMake 配置

```cmake
# CMakeLists.txt 範例
cmake_minimum_required(VERSION 3.10)
project(rtk_mqtt_client)

find_package(PkgConfig REQUIRED)
pkg_check_modules(CJSON REQUIRED libcjson)
find_library(MOSQUITTO_LIB mosquitto)

target_link_libraries(rtk_mqtt_client 
    ${CJSON_LIBRARIES} 
    ${MOSQUITTO_LIB}
)
```

### 3. 程式庫初始化測試

```c
#include <stdio.h>
#include <cjson/cJSON.h>
#include <mosquitto.h>

int main() {
    // 測試 cJSON
    cJSON *json = cJSON_CreateObject();
    cJSON_AddStringToObject(json, "test", "RTK MQTT");
    char *json_string = cJSON_Print(json);
    printf("cJSON test: %s\n", json_string);
    
    // 測試 libmosquitto
    int major, minor, revision;
    mosquitto_lib_version(&major, &minor, &revision);
    printf("libmosquitto version: %d.%d.%d\n", major, minor, revision);
    
    // 清理
    free(json_string);
    cJSON_Delete(json);
    
    return 0;
}
```

## 故障排除

### 常見問題

#### 1. cJSON 找不到標頭檔
```bash
# 解決方案: 檢查安裝路徑
find /usr -name "cjson.h" 2>/dev/null
export C_INCLUDE_PATH=/usr/include/cjson:$C_INCLUDE_PATH
```

#### 2. libmosquitto 連結錯誤
```bash
# 解決方案: 更新程式庫快取
sudo ldconfig
ldd your_program  # 檢查動態程式庫依賴
```

#### 3. 版本相容性問題
```bash
# 檢查已安裝版本
pkg-config --modversion libcjson
mosquitto_pub --help | head -1
```

### 效能調優建議

1. **記憶體管理**: 對於記憶體受限的嵌入式系統，考慮使用自定義的記憶體分配器
2. **編譯最佳化**: 生產環境使用 `-O2` 或 `-Os` 最佳化
3. **靜態連結**: 嵌入式系統可考慮靜態連結以減少執行時依賴

## 下一步

完成程式庫安裝後，請繼續閱讀 [術語定義](03-terminology.md) 了解協議的核心概念。

---

**重要提醒**:
- 所列程式庫已在測試環境中驗證相容性和穩定性
- 建議優先使用指定版本以確保最佳相容性  
- 安裝完成後請執行上述測試程式確認程式庫正常運作
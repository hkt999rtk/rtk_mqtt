# RTK MQTT Framework ARM FreeRTOS 靜態庫構建指南

## 概述

本文檔說明如何為 ARM Cortex-M 微控制器構建 RTK MQTT Framework 靜態庫，特別針對 FreeRTOS 嵌入式系統。

## 系統需求

### 開發環境
- CMake 3.10+
- ARM GNU 工具鏈 (arm-none-eabi-gcc)
- 支持的 ARM 微控制器: Cortex-M0/M3/M4/M7/M33

### ARM 工具鏈安裝

#### macOS (使用 Homebrew)
```bash
brew install arm-none-eabi-gcc
brew install arm-none-eabi-binutils
```

#### Ubuntu/Debian
```bash
sudo apt-get install gcc-arm-none-eabi
sudo apt-get install binutils-arm-none-eabi
```

#### Windows
下載並安裝 [GNU Arm Embedded Toolchain](https://developer.arm.com/tools-and-software/open-source-software/developer-tools/gnu-toolchain/gnu-rm)

## 構建選項

### 支持的 ARM 處理器

| CPU        | FPU 支持 | 特性 |
|------------|---------|------|
| cortex-m0  | 無      | 基本版本 |
| cortex-m3  | 無      | 增強性能 |
| cortex-m4  | 硬件    | DSP + FPU |
| cortex-m7  | 硬件    | 高性能 |
| cortex-m33 | 硬件    | ARMv8-M |

### 記憶體優化設定

針對嵌入式系統，框架提供多個記憶體優化選項：

```cmake
-DRTK_USE_LIGHTWEIGHT_JSON=ON      # 使用輕量級 JSON 解析器
-DRTK_MINIMAL_MEMORY=ON            # 啟用記憶體最小化模式
-DRTK_NO_DYNAMIC_ALLOCATION=ON     # 禁用動態記憶體分配
```

## 構建方法

### 方法 1: 使用 CMake 和工具鏈文件

```bash
# 1. 創建構建目錄
mkdir build-arm && cd build-arm

# 2. 配置 CMake (Cortex-M4 示例)
cmake -DCMAKE_TOOLCHAIN_FILE=../cmake/arm-none-eabi-toolchain.cmake \
      -DCMAKE_BUILD_TYPE=Release \
      -DRTK_TARGET_FREERTOS=ON \
      -DARM_CPU=cortex-m4 \
      -DRTK_USE_LIGHTWEIGHT_JSON=ON \
      -DBUILD_EXAMPLES=OFF \
      -DBUILD_TOOLS=OFF \
      -DBUILD_SHARED_LIBS=OFF \
      ..

# 3. 編譯
make -j$(nproc)
```

### 方法 2: 使用跨平台構建腳本

```bash
# 執行自動化構建腳本
./build-all-platforms.sh
```

### 方法 3: 手動編譯 (適用於特殊需求)

```bash
# 設定編譯器和標志
ARM_CC=arm-none-eabi-gcc
ARM_CFLAGS="-mcpu=cortex-m4 -mthumb -mfloat-abi=hard -mfpu=fpv4-sp-d16"
ARM_CFLAGS="$ARM_CFLAGS -ffunction-sections -fdata-sections -O2"
ARM_CFLAGS="$ARM_CFLAGS -DRTK_PLATFORM_FREERTOS=1"

# 編譯核心源碼
$ARM_CC $ARM_CFLAGS -I./framework/include -c framework/src/*.c

# 創建靜態庫
arm-none-eabi-ar rcs librtk_mqtt_framework_arm.a *.o
```

## 整合到 FreeRTOS 項目

### 1. 複製庫文件

將構建產生的檔案複製到您的 FreeRTOS 項目：

```
your_project/
├── lib/
│   └── librtk_mqtt_framework_arm.a
├── include/
│   └── rtk/
│       ├── rtk_mqtt_client.h
│       ├── rtk_device_plugin.h
│       └── ...
└── src/
    └── your_application.c
```

### 2. 更新 Makefile/CMake

#### Makefile 示例
```makefile
# 連結庫
LDFLAGS += -L./lib -lrtk_mqtt_framework_arm

# 包含路徑
CFLAGS += -I./include/rtk

# FreeRTOS 特定設定
CFLAGS += -DRTK_PLATFORM_FREERTOS=1
```

#### CMakeLists.txt 示例
```cmake
# 添加包含目錄
target_include_directories(your_target PRIVATE include/rtk)

# 連結 RTK 庫
target_link_libraries(your_target 
    PRIVATE 
    ${CMAKE_CURRENT_SOURCE_DIR}/lib/librtk_mqtt_framework_arm.a
)

# 添加編譯定義
target_compile_definitions(your_target PRIVATE RTK_PLATFORM_FREERTOS=1)
```

### 3. FreeRTOS 配置需求

確保您的 \`FreeRTOSConfig.h\` 包含以下設定：

```c
#define configUSE_MUTEXES                1
#define configUSE_RECURSIVE_MUTEXES      1
#define configUSE_COUNTING_SEMAPHORES    1
#define configSUPPORT_DYNAMIC_ALLOCATION 1
#define configTOTAL_HEAP_SIZE           (64 * 1024)  // 根據需要調整
```

## 記憶體需求

### 典型記憶體使用量

| 組件              | RAM (KB) | Flash (KB) |
|------------------|----------|------------|
| 核心框架         | 2-4      | 8-12       |
| MQTT 客戶端      | 4-8      | 15-20      |
| JSON 處理        | 1-2      | 5-8        |
| 設備插件 (示例)  | 1-3      | 3-5        |
| **總計**        | **8-17** | **31-45**  |

### 最小系統需求

- **Flash**: 64KB+ (包含應用程序)
- **RAM**: 32KB+ (包含 FreeRTOS 和應用程序)
- **堆疊**: 每個任務至少 2KB

## 支持的特性

### ✅ 完全支持的功能
- MQTT 通訊 (使用 Eclipse Paho MQTT C)
- JSON 消息編解碼
- 設備插件系統
- 診斷消息格式
- Topic 路徑構建
- 錯誤處理和重連

### ⚠️ 平台限制
- 沒有檔案系統操作
- 簡化的網路接口 (需要平台適配)
- 無動態插件載入 (靜態連結)

### ❌ 不支持的功能
- 共享庫 (.so/.dll)
- 完整的 POSIX 網路層
- 動態記憶體池 (可選)

## 範例使用

### 基本 MQTT 客戶端

```c
#include "rtk_mqtt_client.h"
#include "rtk_device_plugin.h"

// FreeRTOS 任務
void mqtt_task(void *pvParameters) {
    rtk_mqtt_client_t *client;
    
    // 初始化 MQTT 客戶端
    rtk_mqtt_client_config_t config = {
        .broker_host = "iot.example.com",
        .broker_port = 1883,
        .client_id = "device_001",
        .username = "user",
        .password = "pass"
    };
    
    client = rtk_mqtt_client_create(&config);
    if (!client) {
        // 錯誤處理
        return;
    }
    
    // 連接到 MQTT broker
    if (rtk_mqtt_client_connect(client) != RTK_MQTT_SUCCESS) {
        // 連接失敗處理
        return;
    }
    
    // 主循環
    while (1) {
        rtk_mqtt_client_loop(client);
        vTaskDelay(pdMS_TO_TICKS(100));
    }
}

// 應用程序入口點
int main(void) {
    // 初始化 FreeRTOS
    xTaskCreate(mqtt_task, "MQTT", 4096, NULL, 2, NULL);
    vTaskStartScheduler();
    
    return 0;
}
```

## 故障排除

### 常見問題

#### 1. 編譯錯誤: "stdint.h not found"
**解決方案**: 確保安裝了完整的 ARM 工具鏈，包括 newlib：
```bash
# 檢查工具鏈版本
arm-none-eabi-gcc --version

# 查找標準頭文件
find /usr -name "stdint.h" | grep arm-none-eabi
```

#### 2. 連結錯誤: 未定義的符號
**解決方案**: 確保所有依賴項都已正確連結：
```makefile
LDFLAGS += -lrtk_mqtt_framework_arm -lm -lc
```

#### 3. 執行時錯誤: 堆疊溢出
**解決方案**: 增加任務堆疊大小：
```c
xTaskCreate(mqtt_task, "MQTT", 8192, NULL, 2, NULL);  // 增加到 8KB
```

#### 4. 記憶體不足
**解決方案**: 啟用記憶體優化選項：
```c
#define RTK_USE_LIGHTWEIGHT_JSON 1
#define RTK_MINIMAL_MEMORY 1
```

## 版本資訊

- **RTK MQTT Framework**: 1.0.0
- **支持的 ARM 架構**: ARMv6-M, ARMv7-M, ARMv8-M
- **相容的 FreeRTOS 版本**: 10.0.0+
- **測試過的微控制器**: STM32F4, STM32F7, NXP LPC, Nordic nRF52

## 相關資源

- [RTK MQTT 診斷規格](../SPEC.md)
- [設備插件開發指南](../examples/)
- [FreeRTOS 官方文檔](https://www.freertos.org/Documentation/RTOS_book.html)
- [ARM Cortex-M 程式設計指南](https://developer.arm.com/documentation/)

---

**注意**: 此文檔描述了理想的構建流程。實際實施時可能需要根據特定硬體平台和工具鏈版本進行調整。
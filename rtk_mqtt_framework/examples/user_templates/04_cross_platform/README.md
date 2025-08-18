# 跨平台設備範例

這個範例展示如何建立可在多個平台運行的 RTK 設備程式，包括 POSIX 系統 (Linux/macOS)、Windows 系統、和 ARM FreeRTOS 嵌入式系統。通過平台抽象層和條件編譯技術，實現了統一的設備介面和平台特定的最佳化。

## 🎯 學習目標

- 掌握跨平台 C 程式設計技巧
- 學習平台抽象層設計模式
- 理解條件編譯和平台檢測
- 熟悉多平台建構系統配置
- 掌握執行緒和同步的平台差異
- 學習嵌入式系統程式設計最佳實踐

## 🏗️ 支援平台

### 1. POSIX 系統 (Linux/macOS)
- **執行緒**: pthread
- **同步**: pthread_mutex
- **網路**: BSD socket
- **記憶體**: malloc/free
- **時間**: time() / gettimeofday()

### 2. Windows 系統
- **執行緒**: CreateThread / WinAPI
- **同步**: CRITICAL_SECTION
- **網路**: Winsock2
- **記憶體**: HeapAlloc / GlobalMemoryStatusEx
- **時間**: GetTickCount / GetSystemTimes

### 3. ARM FreeRTOS 嵌入式系統
- **執行緒**: FreeRTOS Tasks (xTaskCreate)
- **同步**: FreeRTOS Semaphore
- **網路**: LwIP 或平台特定網路堆疊
- **記憶體**: FreeRTOS Heap (pvPortMalloc)
- **時間**: xTaskGetTickCount

## 🚀 快速開始

### 1. 檢查建構環境

```bash
# 檢查 RTK 框架
make check-rtk

# 檢查工具鏈
make check-toolchain

# 顯示建構資訊
make info
```

### 2. 建構特定平台

```bash
# Linux/macOS 版本
make posix

# Windows 版本 (需要 MinGW)
make windows

# ARM FreeRTOS 版本 (需要 ARM GCC)
make freertos

# 除錯版本
make posix debug

# 發布版本
make posix release
```

### 3. 執行和測試

```bash
# 執行 POSIX 版本
make posix run

# 執行測試
make test

# 建構所有可用平台
make build-all
```

## 🔧 平台特定配置

### POSIX 系統配置

```bash
# 基本建構
make TARGET_PLATFORM=posix BUILD_TYPE=release

# 需要的依賴
sudo apt-get install build-essential libpthread-stubs0-dev  # Ubuntu/Debian
brew install gcc  # macOS
```

### Windows 交叉編譯

```bash
# 安裝 MinGW 工具鏈
sudo apt-get install gcc-mingw-w64  # Ubuntu/Debian
brew install mingw-w64              # macOS

# 建構 Windows 版本
make TARGET_PLATFORM=windows BUILD_TYPE=release

# 執行 (需要 Wine 或 Windows 環境)
wine build/windows-release/cross_platform_device.exe
```

### ARM FreeRTOS 配置

```bash
# 安裝 ARM GCC 工具鏈
sudo apt-get install gcc-arm-none-eabi  # Ubuntu/Debian
brew install arm-none-eabi-gcc          # macOS

# 設置 FreeRTOS 路徑
export FREERTOS_PATH=/path/to/freertos

# 建構 FreeRTOS 版本
make TARGET_PLATFORM=freertos BUILD_TYPE=release

# 生成的 .elf 檔案需要燒錄到硬體
```

## 📋 平台抽象層架構

### 核心抽象類型

```c
// 執行緒類型統一定義
#ifdef RTK_PLATFORM_FREERTOS
    typedef TaskHandle_t rtk_thread_t;
#elif defined(RTK_PLATFORM_WINDOWS)
    typedef HANDLE rtk_thread_t;
#else  // POSIX
    typedef pthread_t rtk_thread_t;
#endif

// 同步原語統一介面
int rtk_mutex_init(rtk_mutex_t* mutex);
void rtk_mutex_lock(rtk_mutex_t* mutex);
void rtk_mutex_unlock(rtk_mutex_t* mutex);
void rtk_mutex_destroy(rtk_mutex_t* mutex);
```

### 時間和延遲抽象

```c
// 統一的時間介面
rtk_time_t rtk_get_time(void);
void rtk_sleep(int seconds);

// 平台特定實作
#ifdef RTK_PLATFORM_FREERTOS
    return xTaskGetTickCount();           // FreeRTOS ticks
#elif defined(RTK_PLATFORM_WINDOWS)
    return GetTickCount();                // Windows milliseconds
#else
    return time(NULL);                    // POSIX seconds
#endif
```

### 系統資源監控

```c
// 跨平台系統指標獲取
float get_cpu_usage(void);
float get_memory_usage(void);
const char* get_platform_name(void);

// FreeRTOS 記憶體監控
#ifdef RTK_PLATFORM_FREERTOS
size_t free_heap = xPortGetFreeHeapSize();
size_t total_heap = configTOTAL_HEAP_SIZE;
return ((float)(total_heap - free_heap) / total_heap) * 100.0f;
#endif
```

## 🧵 多執行緒架構設計

### 統一執行緒管理

```c
// 跨平台執行緒創建
int rtk_thread_create(rtk_thread_t* thread, rtk_thread_func_t func, void* arg) {
#ifdef RTK_PLATFORM_FREERTOS
    BaseType_t result = xTaskCreate(
        (TaskFunction_t)func,
        "RTK_Worker",
        configMINIMAL_STACK_SIZE * 4,
        arg,
        tskIDLE_PRIORITY + 1,
        thread
    );
    return (result == pdPASS) ? 0 : -1;
#elif defined(RTK_PLATFORM_WINDOWS)
    *thread = CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)func, arg, 0, NULL);
    return (*thread != NULL) ? 0 : -1;
#else  // POSIX
    return pthread_create(thread, NULL, (void*(*)(void*))func, arg);
#endif
}
```

### 同步機制統一

```c
// 跨平台互斥鎖實作
void rtk_mutex_lock(rtk_mutex_t* mutex) {
#ifdef RTK_PLATFORM_FREERTOS
    xSemaphoreTake(*mutex, portMAX_DELAY);
#elif defined(RTK_PLATFORM_WINDOWS)
    EnterCriticalSection(mutex);
#else  // POSIX
    pthread_mutex_lock(mutex);
#endif
}
```

## 📊 平台特定最佳化

### FreeRTOS 最佳化

```c
// 記憶體管理最佳化
#ifdef RTK_PLATFORM_FREERTOS
    // 使用 FreeRTOS 靜態分配
    static StaticTask_t sensor_task_buffer;
    static StackType_t sensor_task_stack[SENSOR_TASK_STACK_SIZE];
    
    xTaskCreateStatic(
        sensor_worker_thread,
        "SensorTask",
        SENSOR_TASK_STACK_SIZE,
        NULL,
        SENSOR_TASK_PRIORITY,
        sensor_task_stack,
        &sensor_task_buffer
    );
#endif

// 功耗管理
#ifdef RTK_PLATFORM_FREERTOS
    // 進入低功耗模式
    portENTER_CRITICAL();
    // ... 關鍵操作
    portEXIT_CRITICAL();
#endif
```

### Windows 最佳化

```c
// Windows 特定記憶體監控
#ifdef RTK_PLATFORM_WINDOWS
MEMORYSTATUSEX mem_info;
mem_info.dwLength = sizeof(MEMORYSTATUSEX);
if (GlobalMemoryStatusEx(&mem_info)) {
    return (float)mem_info.dwMemoryLoad;
}
#endif

// Windows 服務整合
#ifdef RTK_PLATFORM_WINDOWS
    // 可擴展為 Windows 服務
    SERVICE_TABLE_ENTRY DispatchTable[] = {
        { TEXT("RTKDevice"), (LPSERVICE_MAIN_FUNCTION)ServiceMain },
        { NULL, NULL }
    };
#endif
```

### POSIX 最佳化

```c
// Linux 系統資訊讀取
#ifdef __linux__
float get_cpu_usage_linux(void) {
    FILE* stat_file = fopen("/proc/stat", "r");
    if (stat_file) {
        // 解析 /proc/stat 獲取實際 CPU 使用率
        // ...
    }
}
#endif

// macOS 系統資訊
#ifdef __APPLE__
#include <sys/sysctl.h>
float get_memory_usage_macos(void) {
    // 使用 sysctl 獲取記憶體資訊
    // ...
}
#endif
```

## 🔧 建構系統詳解

### 平台檢測機制

```makefile
# 自動平台檢測
UNAME_S := $(shell uname -s 2>/dev/null || echo "Windows")
UNAME_M := $(shell uname -m 2>/dev/null || echo "x64")

# 平台特定編譯器選擇
ifeq ($(TARGET_PLATFORM),windows)
    CC = x86_64-w64-mingw32-gcc
    PLATFORM_CFLAGS = -DRTK_PLATFORM_WINDOWS
endif

ifeq ($(TARGET_PLATFORM),freertos)
    CC = arm-none-eabi-gcc
    PLATFORM_CFLAGS = -DRTK_PLATFORM_FREERTOS
    PLATFORM_CFLAGS += -mcpu=cortex-m4 -mthumb
endif
```

### 工具鏈驗證

```makefile
check-toolchain:
	@if ! command -v $(CC) > /dev/null 2>&1; then \
		echo "❌ 編譯器未找到: $(CC)"; \
		echo "安裝指南:"; \
		if [ "$(TARGET_PLATFORM)" = "windows" ]; then \
			echo "  sudo apt-get install gcc-mingw-w64"; \
		elif [ "$(TARGET_PLATFORM)" = "freertos" ]; then \
			echo "  sudo apt-get install gcc-arm-none-eabi"; \
		fi; \
		exit 1; \
	fi
```

## 🧪 測試和驗證

### 自動化測試套件

```bash
# 平台功能測試
make test

# 測試所有可用平台
make build-all

# 特定平台測試
make TARGET_PLATFORM=posix test
make TARGET_PLATFORM=windows test
```

### 手動驗證流程

#### 1. POSIX 系統測試
```bash
# 建構和執行
make posix run

# 監控系統資源
top -p $(pgrep cross_platform_device)

# 檢查執行緒
ps -T -p $(pgrep cross_platform_device)
```

#### 2. Windows 交叉編譯測試
```bash
# 建構 Windows 版本
make windows

# 使用 Wine 測試（Linux 上）
wine build/windows-release/cross_platform_device.exe

# 檢查依賴庫
x86_64-w64-mingw32-objdump -p build/windows-release/cross_platform_device.exe | grep DLL
```

#### 3. FreeRTOS 目標測試
```bash
# 建構 FreeRTOS 版本
make freertos

# 檢查 ELF 檔案
arm-none-eabi-objdump -h build/freertos-arm-release/cross_platform_device.elf

# 生成燒錄檔案
arm-none-eabi-objcopy -O ihex build/freertos-arm-release/cross_platform_device.elf firmware.hex
```

### MQTT 通信驗證

```bash
# 監聽跨平台設備訊息
mosquitto_sub -h test.mosquitto.org -t "rtk/v1/+/+/cross_platform_+/+"

# 平台識別訊息範例
# rtk/v1/demo_tenant/demo_site/cross_platform_Linux_001/evt/platform.info
# rtk/v1/demo_tenant/demo_site/cross_platform_Windows_001/telemetry/cpu_usage
# rtk/v1/demo_tenant/demo_site/cross_platform_FreeRTOS_001/state
```

## 🐛 故障排除

### 常見問題解決

#### Q: MinGW 交叉編譯失敗
**A**:
```bash
# 安裝完整的 MinGW 工具鏈
sudo apt-get install gcc-mingw-w64 g++-mingw-w64

# 檢查 MinGW 版本
x86_64-w64-mingw32-gcc --version

# 設置替代編譯器
make TARGET_PLATFORM=windows CC=i686-w64-mingw32-gcc
```

#### Q: ARM GCC 找不到 FreeRTOS 標頭檔
**A**:
```bash
# 下載 FreeRTOS 源碼
wget https://github.com/FreeRTOS/FreeRTOS/releases/download/...

# 設置 FreeRTOS 路徑
export FREERTOS_PATH=/path/to/freertos

# 或在 Makefile 中設定
make freertos FREERTOS_PATH=/opt/freertos
```

#### Q: 執行緒同步問題
**A**:
```bash
# 使用除錯版本
make posix debug run

# 檢查死鎖（Linux）
gdb --batch --ex run --ex bt --args ./build/linux-debug/cross_platform_device

# 使用 Valgrind 檢查執行緒問題
valgrind --tool=helgrind ./build/linux-debug/cross_platform_device
```

### 除錯技巧

#### 1. 平台特定日誌
```c
#ifdef DEBUG
    printf("[%s DEBUG] 執行緒狀態: %s\n", get_platform_name(), thread_status);
    
    #ifdef RTK_PLATFORM_FREERTOS
        printf("[FreeRTOS] 剩餘堆疊: %d bytes\n", uxTaskGetStackHighWaterMark(NULL));
    #endif
#endif
```

#### 2. 條件編譯驗證
```bash
# 檢查預處理器定義
gcc -E -dM cross_platform_device.c | grep RTK_PLATFORM

# 生成預處理檔案檢查
gcc -E cross_platform_device.c > preprocessed.c
```

#### 3. 記憶體和效能分析
```bash
# POSIX 系統記憶體分析
valgrind --tool=massif ./build/linux-debug/cross_platform_device

# ARM 目標大小分析
arm-none-eabi-size build/freertos-arm-release/cross_platform_device.elf

# Windows 目標分析
x86_64-w64-mingw32-size build/windows-release/cross_platform_device.exe
```

## ⚡ 效能最佳化策略

### 記憶體最佳化

```c
// 平台特定記憶體管理
#ifdef RTK_PLATFORM_FREERTOS
    // 使用 FreeRTOS 記憶體池
    static uint8_t memory_pool[1024];
    static size_t pool_offset = 0;
#else
    // 使用標準記憶體分配
    #define RTK_MALLOC malloc
    #define RTK_FREE free
#endif
```

### 網路效能調校

```c
// 平台特定網路最佳化
#ifdef RTK_PLATFORM_WINDOWS
    // Windows Winsock 最佳化
    int tcp_nodelay = 1;
    setsockopt(socket, IPPROTO_TCP, TCP_NODELAY, (char*)&tcp_nodelay, sizeof(tcp_nodelay));
#endif

#ifdef RTK_PLATFORM_FREERTOS
    // LwIP 緩衝區調整
    #define MQTT_BUFFER_SIZE 512  // 減少記憶體使用
#else
    #define MQTT_BUFFER_SIZE 2048  // 桌面系統可用更大緩衝區
#endif
```

### CPU 使用最佳化

```c
// 平台特定 CPU 最佳化
#ifdef RTK_PLATFORM_FREERTOS
    // 減少浮點運算
    int cpu_percent_int = (int)(cpu_usage * 100);
    
    // 使用 FreeRTOS tick 中斷進行週期性任務
    static TimerHandle_t sensor_timer;
#else
    // 桌面系統可使用精確浮點運算
    float precise_cpu = get_precise_cpu_usage();
#endif
```

## 📈 監控和運維

### 跨平台監控策略

```bash
# Linux 監控腳本
#!/bin/bash
while true; do
    if pgrep cross_platform_device > /dev/null; then
        echo "$(date): 設備運行正常"
        ps aux | grep cross_platform_device | head -1
    else
        echo "$(date): 設備已停止，重新啟動"
        ./build/linux-release/cross_platform_device &
    fi
    sleep 60
done
```

### Windows 服務整合
```c
#ifdef RTK_PLATFORM_WINDOWS
// Windows 服務主函式
void WINAPI ServiceMain(DWORD argc, LPTSTR *argv) {
    // 註冊服務控制處理程序
    status_handle = RegisterServiceCtrlHandler(service_name, ServiceCtrlHandler);
    
    // 啟動設備主邏輯
    initialize_device();
    // ...
}
#endif
```

### FreeRTOS 系統監控
```c
#ifdef RTK_PLATFORM_FREERTOS
void print_system_stats(void) {
    printf("堆疊使用情況:\n");
    printf("  感測器任務: %d bytes\n", 
           uxTaskGetStackHighWaterMark(sensor_task_handle));
    printf("  剩餘堆記憶體: %d bytes\n", xPortGetFreeHeapSize());
    printf("  系統運行時間: %d ticks\n", xTaskGetTickCount());
}
#endif
```

## 🔄 下一步學習

完成跨平台設備範例後，建議您：

1. **深入特定平台** - 專注學習目標平台的特有功能
2. **研究即時操作系統** - 深入 FreeRTOS 任務調度和中斷處理
3. **探索設備驅動開發** - 整合硬體感測器和執行器
4. **學習無線通信協議** - WiFi、LoRa、NB-IoT 整合
5. **研究邊緣計算** - 在設備端實作 AI 推理
6. **探索容器化部署** - Docker 多架構建構
7. **學習設備安全** - 加密、認證、安全啟動

## 📚 相關資源

- **[MANUAL.md](../../../docs/MANUAL.md)** - 完整使用手冊
- **[03_complete_device](../03_complete_device/)** - 前一個學習範例  
- **[RTK MQTT 協議規範](../../../docs/SPEC.md)** - 協議詳細說明
- **[建構系統指南](../../../README.md)** - 開發環境設置

### 平台特定資源

#### FreeRTOS 資源
- [FreeRTOS 官方文檔](https://www.freertos.org/Documentation/RTOS_book.html)
- [FreeRTOS+TCP 網路堆疊](https://www.freertos.org/FreeRTOS-Plus/FreeRTOS_Plus_TCP/)
- [ARM Cortex-M 開發指南](https://developer.arm.com/documentation/)

#### Windows 開發資源  
- [Windows API 參考](https://docs.microsoft.com/en-us/windows/win32/api/)
- [MinGW-w64 官網](http://mingw-w64.org/)
- [Windows 服務開發](https://docs.microsoft.com/en-us/windows/win32/services/)

#### POSIX 開發資源
- [POSIX 標準文檔](https://pubs.opengroup.org/onlinepubs/9699919799/)
- [Linux 系統程式設計](https://man7.org/linux/man-pages/)
- [macOS 開發文檔](https://developer.apple.com/documentation/)

## 🏆 跨平台開發最佳實踐總結

這個範例展示了企業級跨平台開發的關鍵技術：

✅ **平台抽象層設計** - 統一介面隔離平台差異  
✅ **條件編譯策略** - 高效的平台特定代碼組織  
✅ **多工具鏈支援** - 靈活的建構系統設計  
✅ **執行緒同步抽象** - 跨平台並行程式設計  
✅ **系統資源監控** - 平台特定的效能最佳化  
✅ **記憶體管理策略** - 嵌入式到桌面的適配  
✅ **網路通信抽象** - 統一的 MQTT 通信介面  
✅ **除錯和測試支援** - 多平台品質保證  
✅ **部署和運維友善** - 生產環境就緒  
✅ **文檔和範例完整** - 便於學習和維護  

通過這個範例，您將具備開發工業級跨平台 IoT 解決方案的完整技能。
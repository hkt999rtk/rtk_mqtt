# RTK MQTT Framework 使用手冊

RTK MQTT Framework 是一個專為 IoT 設備、網路設備和企業系統設計的跨平台 MQTT 診斷通訊框架。支援 POSIX (Linux/macOS)、Windows 和 ARM FreeRTOS 環境，零外部依賴。

**本手冊適用於想要將框架整合到自己專案中的工程師。** 如果您是框架開發者，請參閱 [README.md](README.md)。

---

## 第一部分：快速上手指南 (5分鐘開始)

### RTK MQTT Framework 是什麼？

RTK MQTT Framework 提供了一套完整的解決方案，讓您可以快速為 IoT 設備添加 MQTT 診斷功能：

- **零外部依賴**：所有 MQTT 和 JSON 函式庫都已內建
- **跨平台支援**：Linux、Windows、FreeRTOS 統一介面
- **即插即用**：下載發行包即可開始整合
- **漸進式學習**：從 20 行 Hello World 到生產級應用

### 系統需求

**所有平台共同需求**：
- 編譯器：GCC 4.9+、Clang 3.5+、或 Visual Studio 2017+
- 建構工具：Make 或 CMake 3.10+

**平台特定需求**：

#### Linux/macOS
```bash
# Ubuntu/Debian
sudo apt-get install build-essential

# macOS  
# 安裝 Xcode Command Line Tools
xcode-select --install
```

#### Windows
```powershell
# 選項1：Visual Studio (推薦)
# 安裝 Visual Studio 2017+ 並包含 C++ 工作負載

# 選項2：MinGW-w64
# 從 https://www.mingw-w64.org/downloads/ 下載安裝
```

#### ARM FreeRTOS
```bash
# 安裝 ARM GCC 工具鏈
# Ubuntu/Debian
sudo apt-get install gcc-arm-none-eabi

# macOS
brew install arm-none-eabi-gcc
```

### 第一個 Hello World 程式

1. **下載發行包**
   ```bash
   # 下載適合您平台的發行包
   wget https://releases.example.com/rtk_mqtt_framework_v1.0.0_linux-amd64.tar.gz
   tar -xzf rtk_mqtt_framework_v1.0.0_linux-amd64.tar.gz
   cd rtk_mqtt_framework_v1.0.0_linux-amd64
   ```

2. **查看發行包內容**
   ```bash
   ls -la
   # bin/          - 可執行檔案和工具
   # lib/          - 靜態和動態函式庫  
   # include/      - 標頭檔
   # examples/     - 整合範例
   # MANUAL.md     - 本使用手冊
   ```

3. **執行第一個範例**
   ```bash
   cd examples/user_templates/01_hello_world
   cat README.md              # 閱讀範例說明
   make                       # 編譯範例
   ./hello_rtk_mqtt          # 執行範例
   ```

4. **驗證安裝成功**
   ```bash
   # 如果看到以下輸出，表示安裝成功：
   # ✓ RTK MQTT Framework 初始化成功
   # ✓ 連接到 MQTT broker: test.mosquitto.org:1883
   # ✓ 發布訊息成功: Hello from RTK MQTT Framework!
   # ✓ 清理完成
   ```

---

## 第二部分：完整整合指南

### 如何整合到您的專案中

#### 方法1：使用發行包 (推薦)

1. **複製必要檔案**
   ```bash
   # 複製標頭檔到您的專案
   cp -r include/rtk_mqtt_framework /path/to/your/project/include/
   
   # 複製函式庫檔案
   cp lib/librtk_mqtt_framework.a /path/to/your/project/lib/
   ```

2. **修改您的 Makefile**
   ```makefile
   # 添加 RTK 框架
   RTK_INCLUDE_DIR = include/rtk_mqtt_framework
   RTK_LIB_DIR = lib
   
   CFLAGS += -I$(RTK_INCLUDE_DIR)
   LDFLAGS += -L$(RTK_LIB_DIR) -lrtk_mqtt_framework -lpthread -lm
   
   # 您的目標
   your_app: your_app.c
   	$(CC) $(CFLAGS) -o $@ $^ $(LDFLAGS)
   ```

3. **在您的代碼中使用**
   ```c
   #include <rtk_mqtt_client.h>
   #include <rtk_topic_builder.h>
   #include <rtk_message_codec.h>
   
   int main() {
       // 初始化 RTK 框架
       rtk_mqtt_client_t* client = rtk_mqtt_client_create("test.mosquitto.org", 1883, "my_device");
       
       // 連接到 MQTT broker
       if (rtk_mqtt_client_connect(client) == RTK_SUCCESS) {
           printf("✓ 連接成功!\n");
           
           // 發布設備狀態
           rtk_mqtt_client_publish_state(client, "online", "healthy");
           
           // 清理
           rtk_mqtt_client_destroy(client);
       }
       
       return 0;
   }
   ```

#### 方法2：系統安裝 (進階用戶)

```bash
# 安裝到系統路徑
sudo cp include/rtk_mqtt_framework/* /usr/local/include/
sudo cp lib/* /usr/local/lib/
sudo ldconfig

# 使用 pkg-config
pkg-config --cflags --libs rtk-mqtt-framework
```

### 平台特定整合指示

#### Linux 平台整合 (生產級完整指南)

**第一步：環境準備**
```bash
# Ubuntu/Debian 系統
sudo apt-get update
sudo apt-get install build-essential cmake pkg-config git

# CentOS/RHEL 系統  
sudo yum groupinstall "Development Tools"
sudo yum install cmake pkgconfig git

# 驗證安裝
gcc --version
make --version
cmake --version
```

**第二步：建立專案結構**
```bash
mkdir my_iot_project && cd my_iot_project

# 建立目錄結構
mkdir -p {src,include,lib,build,config}

# 複製 RTK 框架檔案
cp -r /path/to/rtk_framework/include/rtk_mqtt_framework include/
cp /path/to/rtk_framework/lib/librtk_mqtt_framework.a lib/
```

**第三步：建立 Makefile**
```makefile
# 專案配置
PROJECT_NAME = my_iot_device
VERSION = 1.0.0

# 編譯器設定
CC = gcc
CFLAGS = -std=c99 -Wall -Wextra -O2 -g
CPPFLAGS = -DVERSION=\"$(VERSION)\"

# RTK 框架設定
RTK_INCLUDE = include/rtk_mqtt_framework
RTK_LIB = lib

# 包含路徑和連結設定
INCLUDES = -I$(RTK_INCLUDE) -Iinclude
LDFLAGS = -L$(RTK_LIB) -lrtk_mqtt_framework -lpthread -lm

# 源檔案
SRCDIR = src
SOURCES = $(wildcard $(SRCDIR)/*.c)
OBJECTS = $(SOURCES:$(SRCDIR)/%.c=build/%.o)

# 建構目標
all: $(PROJECT_NAME)

$(PROJECT_NAME): $(OBJECTS)
	$(CC) $(OBJECTS) -o $@ $(LDFLAGS)
	@echo "✓ 編譯完成: $(PROJECT_NAME)"

build/%.o: $(SRCDIR)/%.c | build
	$(CC) $(CFLAGS) $(CPPFLAGS) $(INCLUDES) -c $< -o $@

build:
	mkdir -p build

# 測試目標
test: $(PROJECT_NAME)
	@echo "正在測試應用程式..."
	./$(PROJECT_NAME) --test

# 安裝目標
install: $(PROJECT_NAME)
	sudo cp $(PROJECT_NAME) /usr/local/bin/
	@echo "✓ 已安裝到 /usr/local/bin/"

# 清理目標
clean:
	rm -rf build $(PROJECT_NAME)

.PHONY: all test install clean
```

**第四步：編寫應用程式代碼**
```c
// src/main.c - Linux 生產級範例
#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include <unistd.h>
#include <rtk_mqtt_client.h>
#include <rtk_device_plugin.h>

// 全域變數
static rtk_mqtt_client_t* g_client = NULL;
static int g_running = 1;

// 信號處理函式
void signal_handler(int sig) {
    printf("\n收到信號 %d，正在優雅退出...\n", sig);
    g_running = 0;
}

// 設備狀態更新回調
void device_status_callback(rtk_device_state_t* state) {
    printf("設備狀態更新: %s\n", state->status);
}

int main(int argc, char* argv[]) {
    printf("=== Linux IoT 設備應用程式 v%s ===\n", VERSION);
    
    // 註冊信號處理
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    
    // 建立 MQTT 客戶端
    rtk_mqtt_client_config_t config = {
        .broker_host = "mqtt.example.com",
        .broker_port = 1883,
        .client_id = "linux_device_001",
        .username = getenv("MQTT_USERNAME"),
        .password = getenv("MQTT_PASSWORD"),
        .keepalive = 60,
        .qos = 1,
        .clean_session = 1
    };
    
    g_client = rtk_mqtt_client_create(&config);
    if (!g_client) {
        fprintf(stderr, "❌ 無法建立 MQTT 客戶端\n");
        return 1;
    }
    
    // 設定狀態回調
    rtk_mqtt_client_set_status_callback(g_client, device_status_callback);
    
    // 連接到 MQTT broker
    printf("正在連接到 %s:%d...\n", config.broker_host, config.broker_port);
    if (rtk_mqtt_client_connect(g_client) != RTK_SUCCESS) {
        fprintf(stderr, "❌ 連接失敗\n");
        rtk_mqtt_client_destroy(g_client);
        return 1;
    }
    
    printf("✓ 連接成功，開始主循環...\n");
    
    // 主處理循環
    int counter = 0;
    while (g_running) {
        // 處理 MQTT 訊息
        rtk_mqtt_client_loop(g_client);
        
        // 每30秒發送遙測資料
        if (counter % 30 == 0) {
            float cpu_temp = 45.5 + (rand() % 100) / 10.0; // 模擬 CPU 溫度
            rtk_mqtt_client_publish_telemetry(g_client, "cpu_temperature", cpu_temp, "celsius");
            printf("📊 發布遙測: CPU溫度 %.1f°C\n", cpu_temp);
        }
        
        sleep(1);
        counter++;
    }
    
    // 清理
    printf("正在清理資源...\n");
    rtk_mqtt_client_disconnect(g_client);
    rtk_mqtt_client_destroy(g_client);
    printf("✓ 應用程式正常退出\n");
    
    return 0;
}
```

**第五步：編譯和執行**
```bash
# 編譯應用程式
make

# 設定環境變數 (選用)
export MQTT_USERNAME="your_username"
export MQTT_PASSWORD="your_password"

# 執行應用程式
./my_iot_device

# 或以背景程序執行
nohup ./my_iot_device > device.log 2>&1 &
```

**第六步：系統服務整合**
```bash
# 建立 systemd 服務檔案
sudo tee /etc/systemd/system/my-iot-device.service > /dev/null <<EOF
[Unit]
Description=My IoT Device Service
After=network.target

[Service]
Type=simple
User=iot
WorkingDirectory=/opt/my-iot-device
ExecStart=/opt/my-iot-device/my_iot_device
Restart=always
RestartSec=5
Environment=MQTT_USERNAME=your_username
Environment=MQTT_PASSWORD=your_password

[Install]
WantedBy=multi-user.target
EOF

# 啟用和啟動服務
sudo systemctl daemon-reload
sudo systemctl enable my-iot-device.service
sudo systemctl start my-iot-device.service

# 檢查服務狀態
sudo systemctl status my-iot-device.service
```

#### Windows 平台整合 (企業級完整指南)

**第一步：開發環境設置**

**選項1：Visual Studio (推薦)**
```powershell
# 1. 下載並安裝 Visual Studio 2019 或更新版本
# 2. 安裝時選擇 "C++ 桌面開發" 工作負載
# 3. 確保包含 Windows 10/11 SDK

# 驗證安裝
cl
link
nmake
```

**選項2：MinGW-w64**
```powershell
# 下載 MinGW-w64 從官網
# 或使用 Chocolatey 安裝
choco install mingw

# 驗證安裝
gcc --version
mingw32-make --version
```

**第二步：建立專案結構**
```powershell
mkdir MyIoTProject
cd MyIoTProject

# 建立目錄結構
New-Item -ItemType Directory -Path src, include, lib, build, config, logs

# 複製 RTK 框架檔案
Copy-Item -Recurse "C:\path\to\rtk_framework\include\rtk_mqtt_framework" include\
Copy-Item "C:\path\to\rtk_framework\lib\rtk_mqtt_framework.lib" lib\
```

**第三步：建立 Visual Studio 專案**
```xml
<!-- MyIoTDevice.vcxproj - Visual Studio 專案檔 -->
<?xml version="1.0" encoding="utf-8"?>
<Project DefaultTargets="Build" xmlns="http://schemas.microsoft.com/developer/msbuild/2003">
  <PropertyGroup Label="Globals">
    <ProjectName>MyIoTDevice</ProjectName>
    <WindowsTargetPlatformVersion>10.0</WindowsTargetPlatformVersion>
  </PropertyGroup>
  
  <PropertyGroup Condition="'$(Configuration)|$(Platform)'=='Release|x64'">
    <OutDir>$(SolutionDir)build\$(Platform)\$(Configuration)\</OutDir>
    <IntDir>$(SolutionDir)build\$(Platform)\$(Configuration)\intermediate\</IntDir>
  </PropertyGroup>
  
  <ItemDefinitionGroup Condition="'$(Configuration)|$(Platform)'=='Release|x64'">
    <ClCompile>
      <AdditionalIncludeDirectories>include\rtk_mqtt_framework;include;%(AdditionalIncludeDirectories)</AdditionalIncludeDirectories>
      <PreprocessorDefinitions>WIN32;NDEBUG;_CONSOLE;RTK_PLATFORM_WINDOWS;%(PreprocessorDefinitions)</PreprocessorDefinitions>
      <WarningLevel>Level3</WarningLevel>
      <Optimization>MaxSpeed</Optimization>
    </ClCompile>
    <Link>
      <AdditionalLibraryDirectories>lib;%(AdditionalLibraryDirectories)</AdditionalLibraryDirectories>
      <AdditionalDependencies>rtk_mqtt_framework.lib;ws2_32.lib;kernel32.lib;%(AdditionalDependencies)</AdditionalDependencies>
      <SubSystem>Console</SubSystem>
    </Link>
  </ItemDefinitionGroup>
  
  <ItemGroup>
    <ClCompile Include="src\main.c" />
    <ClCompile Include="src\device_manager.c" />
  </ItemGroup>
</Project>
```

**第四步：建立 Makefile (MinGW)**
```makefile
# Makefile.windows - Windows MinGW 建構檔
PROJECT_NAME = MyIoTDevice
VERSION = 1.0.0

# MinGW 編譯器設定
CC = gcc
CFLAGS = -std=c99 -Wall -Wextra -O2 -g
CPPFLAGS = -DWIN32 -DRTK_PLATFORM_WINDOWS -DVERSION=\"$(VERSION)\"

# RTK 框架設定
RTK_INCLUDE = include/rtk_mqtt_framework
RTK_LIB = lib

# 包含路徑和連結設定
INCLUDES = -I$(RTK_INCLUDE) -Iinclude
LDFLAGS = -L$(RTK_LIB) -lrtk_mqtt_framework -lws2_32 -lkernel32

# Windows 特定設定
EXECUTABLE = $(PROJECT_NAME).exe
SRCDIR = src
SOURCES = $(wildcard $(SRCDIR)/*.c)
OBJECTS = $(SOURCES:$(SRCDIR)/%.c=build/%.o)

# 建構目標
all: $(EXECUTABLE)

$(EXECUTABLE): $(OBJECTS) | build
	$(CC) $(OBJECTS) -o $@ $(LDFLAGS)
	@echo ✓ 編譯完成: $(EXECUTABLE)

build/%.o: $(SRCDIR)/%.c | build
	$(CC) $(CFLAGS) $(CPPFLAGS) $(INCLUDES) -c $< -o $@

build:
	if not exist build mkdir build

# 測試目標
test: $(EXECUTABLE)
	@echo 正在測試應用程式...
	$(EXECUTABLE) --test

# 清理目標
clean:
	if exist build rmdir /s /q build
	if exist $(EXECUTABLE) del $(EXECUTABLE)

.PHONY: all test clean
```

**第五步：編寫 Windows 應用程式**
```c
// src/main.c - Windows 企業級範例
#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include <windows.h>
#include <rtk_mqtt_client.h>
#include <rtk_device_plugin.h>

// Windows 特定標頭
#include <winsvc.h>
#include <tchar.h>

// 全域變數
static rtk_mqtt_client_t* g_client = NULL;
static SERVICE_STATUS g_service_status = {0};
static SERVICE_STATUS_HANDLE g_status_handle = NULL;
static HANDLE g_service_stop_event = INVALID_HANDLE_VALUE;

// Windows 服務控制處理程序
VOID WINAPI ServiceCtrlHandler(DWORD CtrlCode) {
    switch (CtrlCode) {
        case SERVICE_CONTROL_STOP:
            ReportServiceStatus(SERVICE_STOP_PENDING, NO_ERROR, 0);
            SetEvent(g_service_stop_event);
            break;
        default:
            break;
    }
}

// 服務狀態報告
VOID ReportServiceStatus(DWORD dwCurrentState, DWORD dwWin32ExitCode, DWORD dwWaitHint) {
    static DWORD dwCheckPoint = 1;
    
    g_service_status.dwCurrentState = dwCurrentState;
    g_service_status.dwWin32ExitCode = dwWin32ExitCode;
    g_service_status.dwWaitHint = dwWaitHint;
    
    if (dwCurrentState == SERVICE_START_PENDING)
        g_service_status.dwControlsAccepted = 0;
    else
        g_service_status.dwControlsAccepted = SERVICE_ACCEPT_STOP;
    
    if ((dwCurrentState == SERVICE_RUNNING) || (dwCurrentState == SERVICE_STOPPED))
        g_service_status.dwCheckPoint = 0;
    else
        g_service_status.dwCheckPoint = dwCheckPoint++;
    
    SetServiceStatus(g_status_handle, &g_service_status);
}

// 主服務執行緒
DWORD WINAPI ServiceWorkerThread(LPVOID lpParam) {
    // 建立 MQTT 客戶端
    rtk_mqtt_client_config_t config = {
        .broker_host = "mqtt.enterprise.com",
        .broker_port = 1883,
        .client_id = "windows_device_001",
        .username = getenv("MQTT_USERNAME"),
        .password = getenv("MQTT_PASSWORD"),
        .keepalive = 60,
        .qos = 1,
        .clean_session = 1
    };
    
    g_client = rtk_mqtt_client_create(&config);
    if (!g_client) {
        ReportServiceStatus(SERVICE_STOPPED, ERROR_FUNCTION_FAILED, 0);
        return ERROR_FUNCTION_FAILED;
    }
    
    // 連接到 MQTT broker
    if (rtk_mqtt_client_connect(g_client) != RTK_SUCCESS) {
        rtk_mqtt_client_destroy(g_client);
        ReportServiceStatus(SERVICE_STOPPED, ERROR_NETWORK_UNREACHABLE, 0);
        return ERROR_NETWORK_UNREACHABLE;
    }
    
    // 服務運行中
    ReportServiceStatus(SERVICE_RUNNING, NO_ERROR, 0);
    
    // 主處理循環
    DWORD counter = 0;
    while (WaitForSingleObject(g_service_stop_event, 1000) != WAIT_OBJECT_0) {
        // 處理 MQTT 訊息
        rtk_mqtt_client_loop(g_client);
        
        // 每30秒發送系統資訊
        if (counter % 30 == 0) {
            MEMORYSTATUSEX memInfo;
            memInfo.dwLength = sizeof(MEMORYSTATUSEX);
            GlobalMemoryStatusEx(&memInfo);
            
            float memory_usage = (float)memInfo.dwMemoryLoad;
            rtk_mqtt_client_publish_telemetry(g_client, "memory_usage", memory_usage, "percent");
        }
        
        counter++;
    }
    
    // 清理
    rtk_mqtt_client_disconnect(g_client);
    rtk_mqtt_client_destroy(g_client);
    
    ReportServiceStatus(SERVICE_STOPPED, NO_ERROR, 0);
    return NO_ERROR;
}

// Windows 服務主程序
VOID WINAPI ServiceMain(DWORD argc, LPTSTR *argv) {
    // 註冊服務控制處理程序
    g_status_handle = RegisterServiceCtrlHandler(
        TEXT("MyIoTDeviceService"),
        ServiceCtrlHandler);
    
    if (g_status_handle == NULL) {
        return;
    }
    
    // 初始化服務狀態
    ZeroMemory(&g_service_status, sizeof(g_service_status));
    g_service_status.dwServiceType = SERVICE_WIN32_OWN_PROCESS;
    g_service_status.dwServiceSpecificExitCode = 0;
    
    ReportServiceStatus(SERVICE_START_PENDING, NO_ERROR, 3000);
    
    // 建立停止事件
    g_service_stop_event = CreateEvent(NULL, TRUE, FALSE, NULL);
    if (g_service_stop_event == NULL) {
        ReportServiceStatus(SERVICE_STOPPED, GetLastError(), 0);
        return;
    }
    
    // 啟動工作執行緒
    HANDLE hThread = CreateThread(NULL, 0, ServiceWorkerThread, NULL, 0, NULL);
    if (hThread == NULL) {
        ReportServiceStatus(SERVICE_STOPPED, GetLastError(), 0);
        return;
    }
    
    // 等待執行緒結束
    WaitForSingleObject(hThread, INFINITE);
    CloseHandle(hThread);
    CloseHandle(g_service_stop_event);
}

// 主程序入口點
int main(int argc, char* argv[]) {
    // 檢查是否以服務模式運行
    if (argc > 1 && strcmp(argv[1], "--service") == 0) {
        SERVICE_TABLE_ENTRY DispatchTable[] = {
            { TEXT("MyIoTDeviceService"), (LPSERVICE_MAIN_FUNCTION)ServiceMain },
            { NULL, NULL }
        };
        
        if (!StartServiceCtrlDispatcher(DispatchTable)) {
            printf("❌ 無法啟動服務調度程序\n");
            return 1;
        }
    } else {
        // 控制台模式 (開發/測試用)
        printf("=== Windows IoT 設備應用程式 v%s ===\n", VERSION);
        printf("提示：使用 --service 參數以服務模式運行\n\n");
        
        // 直接呼叫工作執行緒函式
        g_service_stop_event = CreateEvent(NULL, TRUE, FALSE, NULL);
        ServiceWorkerThread(NULL);
        CloseHandle(g_service_stop_event);
    }
    
    return 0;
}
```

**第六步：編譯和部署**
```powershell
# 使用 Visual Studio
MSBuild MyIoTDevice.vcxproj /p:Configuration=Release /p:Platform=x64

# 或使用 MinGW
mingw32-make -f Makefile.windows

# 安裝為 Windows 服務
sc create "MyIoTDeviceService" binPath= "C:\path\to\MyIoTDevice.exe --service"
sc config "MyIoTDeviceService" start= auto
sc start "MyIoTDeviceService"

# 檢查服務狀態
sc query "MyIoTDeviceService"
```

#### ARM FreeRTOS 平台整合 (嵌入式系統完整指南)

**第一步：工具鏈環境設置**
```bash
# Ubuntu/Debian 安裝 ARM 工具鏈
sudo apt-get install gcc-arm-none-eabi binutils-arm-none-eabi \
                     gdb-arm-none-eabi newlib-arm-none-eabi

# macOS 安裝 ARM 工具鏈
brew install arm-none-eabi-gcc arm-none-eabi-binutils

# 驗證安裝
arm-none-eabi-gcc --version
arm-none-eabi-size --version
arm-none-eabi-objdump --version
```

**第二步：FreeRTOS 專案結構**
```bash
mkdir freertos_iot_device && cd freertos_iot_device

# 建立標準 FreeRTOS 專案結構
mkdir -p {src,include,lib,FreeRTOS/{Source,Demo},hardware,config,scripts}

# 複製 RTK 框架檔案
cp -r /path/to/rtk_framework/include/rtk_mqtt_framework include/
cp /path/to/rtk_framework/lib/librtk_mqtt_framework_arm.a lib/

# 下載並設置 FreeRTOS (如果尚未有)
cd FreeRTOS
wget https://github.com/FreeRTOS/FreeRTOS/releases/download/V10.4.6/FreeRTOSv10.4.6.zip
unzip FreeRTOSv10.4.6.zip
cd ..
```

**第三步：建立 Makefile**
```makefile
# Makefile.freertos - ARM FreeRTOS 建構檔
PROJECT_NAME = freertos_iot_device
VERSION = 1.0.0

# ARM 工具鏈設定
PREFIX = arm-none-eabi-
CC = $(PREFIX)gcc
OBJCOPY = $(PREFIX)objcopy
OBJDUMP = $(PREFIX)objdump
SIZE = $(PREFIX)size

# 目標 MCU 設定 (Cortex-M4 範例)
ARCH = cortex-m4
FPU = fpv4-sp-d16
FLOAT_ABI = hard

# 編譯標志
CFLAGS = -mcpu=$(ARCH) -mthumb -mfloat-abi=$(FLOAT_ABI) -mfpu=$(FPU)
CFLAGS += -std=c99 -Wall -Wextra -O2 -g3
CFLAGS += -ffunction-sections -fdata-sections
CFLAGS += -DRTK_PLATFORM_FREERTOS -DARM_MATH_CM4 -D__FPU_PRESENT=1

# 連結標志
LDFLAGS = -mcpu=$(ARCH) -mthumb -mfloat-abi=$(FLOAT_ABI) -mfpu=$(FPU)
LDFLAGS += -Wl,--gc-sections -Wl,--print-memory-usage
LDFLAGS += -T linker_script.ld

# 路徑設定
RTK_INCLUDE = include/rtk_mqtt_framework
FREERTOS_DIR = FreeRTOS/Source
FREERTOS_INCLUDE = $(FREERTOS_DIR)/include
FREERTOS_PORT = $(FREERTOS_DIR)/portable/GCC/ARM_CM4F

# 包含路徑
INCLUDES = -I$(RTK_INCLUDE) -I$(FREERTOS_INCLUDE) -I$(FREERTOS_PORT)
INCLUDES += -Iinclude -Iconfig -Ihardware

# 源檔案
SOURCES = src/main.c src/device_tasks.c src/network_interface.c
SOURCES += hardware/system_init.c hardware/uart.c hardware/ethernet.c
SOURCES += $(FREERTOS_DIR)/tasks.c $(FREERTOS_DIR)/queue.c
SOURCES += $(FREERTOS_DIR)/list.c $(FREERTOS_DIR)/timers.c
SOURCES += $(FREERTOS_DIR)/portable/MemMang/heap_4.c
SOURCES += $(FREERTOS_PORT)/port.c

# 建構目標
OBJECTS = $(SOURCES:.c=.o)
ELF_FILE = $(PROJECT_NAME).elf
HEX_FILE = $(PROJECT_NAME).hex
BIN_FILE = $(PROJECT_NAME).bin

all: $(ELF_FILE) $(HEX_FILE) $(BIN_FILE) size

$(ELF_FILE): $(OBJECTS)
	$(CC) $(OBJECTS) -o $@ $(LDFLAGS) -Llib -lrtk_mqtt_framework_arm -lm
	@echo "✓ 連結完成: $@"

%.o: %.c
	$(CC) $(CFLAGS) $(INCLUDES) -c $< -o $@

$(HEX_FILE): $(ELF_FILE)
	$(OBJCOPY) -O ihex $< $@
	@echo "✓ 產生 Intel HEX: $@"

$(BIN_FILE): $(ELF_FILE)
	$(OBJCOPY) -O binary $< $@
	@echo "✓ 產生二進位檔: $@"

size: $(ELF_FILE)
	@echo "=== 記憶體使用情況 ==="
	$(SIZE) $<
	@echo "========================"

# 除錯目標
debug: $(ELF_FILE)
	arm-none-eabi-gdb $< -ex "target remote localhost:3333"

# 燒錄目標 (OpenOCD)
flash: $(HEX_FILE)
	openocd -f interface/stlink-v2.cfg -f target/stm32f4x.cfg \
		-c "program $(HEX_FILE) verify reset exit"

# 清理目標
clean:
	rm -f $(OBJECTS) $(ELF_FILE) $(HEX_FILE) $(BIN_FILE)

.PHONY: all size debug flash clean
```

**第四步：建立 FreeRTOS 配置**
```c
// config/FreeRTOSConfig.h - FreeRTOS 配置檔
#ifndef FREERTOS_CONFIG_H
#define FREERTOS_CONFIG_H

// === 核心配置 ===
#define configUSE_PREEMPTION                    1
#define configUSE_IDLE_HOOK                     0
#define configUSE_TICK_HOOK                     0
#define configCPU_CLOCK_HZ                      168000000UL  // STM32F4 範例
#define configTICK_RATE_HZ                      1000
#define configMAX_PRIORITIES                    7
#define configMINIMAL_STACK_SIZE                256
#define configTOTAL_HEAP_SIZE                   (64 * 1024)  // 64KB 堆記憶體

// === RTK MQTT 框架需求 ===
#define configUSE_MUTEXES                       1
#define configUSE_RECURSIVE_MUTEXES             1
#define configUSE_COUNTING_SEMAPHORES           1
#define configUSE_QUEUE_SETS                    1
#define configUSE_TIME_SLICING                  1
#define configUSE_NEWLIB_REENTRANT              1

// === 記憶體管理 ===
#define configSUPPORT_STATIC_ALLOCATION         1
#define configSUPPORT_DYNAMIC_ALLOCATION        1
#define configAPPLICATION_ALLOCATED_HEAP        0

// === 任務管理 ===
#define configUSE_16_BIT_TICKS                  0
#define configIDLE_SHOULD_YIELD                 1
#define configUSE_TASK_NOTIFICATIONS            1
#define configTASK_NOTIFICATION_ARRAY_ENTRIES   1

// === 開發除錯 ===
#define configUSE_TRACE_FACILITY                1
#define configUSE_STATS_FORMATTING_FUNCTIONS    1
#define configGENERATE_RUN_TIME_STATS           1
#define configCHECK_FOR_STACK_OVERFLOW          2

// === 網路配置 (lwIP) ===
#define configNUM_THREAD_LOCAL_STORAGE_POINTERS 2
#define configUSE_APPLICATION_TASK_TAG          1

// === 中斷優先級 ===
#define configKERNEL_INTERRUPT_PRIORITY         255
#define configMAX_SYSCALL_INTERRUPT_PRIORITY    191
#define configMAX_API_CALL_INTERRUPT_PRIORITY   191

// === 任務堆疊大小 ===
#define configIDLE_TASK_STACK_SIZE              configMINIMAL_STACK_SIZE
#define configTIMER_TASK_STACK_SIZE             (configMINIMAL_STACK_SIZE * 4)

// 包含標準定義
#include "stm32f4xx.h"  // 根據您的 MCU 調整

#endif /* FREERTOS_CONFIG_H */
```

**第五步：編寫 FreeRTOS 應用程式**
```c
// src/main.c - FreeRTOS MQTT 設備範例
#include "FreeRTOS.h"
#include "task.h"
#include "queue.h"
#include "semphr.h"

#include <rtk_mqtt_client.h>
#include <rtk_device_plugin.h>
#include "network_interface.h"
#include "system_init.h"

// === 任務優先級定義 ===
#define PRIORITY_MQTT_TASK          (tskIDLE_PRIORITY + 3)
#define PRIORITY_SENSOR_TASK        (tskIDLE_PRIORITY + 2)
#define PRIORITY_NETWORK_TASK       (tskIDLE_PRIORITY + 4)

// === 任務堆疊大小 ===
#define STACK_SIZE_MQTT_TASK        (4 * 1024)
#define STACK_SIZE_SENSOR_TASK      (2 * 1024)
#define STACK_SIZE_NETWORK_TASK     (6 * 1024)

// === 全域變數 ===
static rtk_mqtt_client_t* g_mqtt_client = NULL;
static QueueHandle_t g_sensor_queue = NULL;
static SemaphoreHandle_t g_network_mutex = NULL;

// === 感測器資料結構 ===
typedef struct {
    char metric_name[32];
    float value;
    char unit[16];
    TickType_t timestamp;
} sensor_data_t;

// === 網路任務 ===
void network_task(void *pvParameters) {
    printf("網路任務啟動\n");
    
    // 初始化網路介面
    if (network_interface_init() != 0) {
        printf("❌ 網路初始化失敗\n");
        vTaskDelete(NULL);
        return;
    }
    
    // 等待網路連接
    while (!network_interface_is_connected()) {
        printf("等待網路連接...\n");
        vTaskDelay(pdMS_TO_TICKS(1000));
    }
    
    printf("✓ 網路連接成功\n");
    
    // 建立 MQTT 客戶端
    rtk_mqtt_client_config_t config = {
        .broker_host = "mqtt.freertos.device.com",
        .broker_port = 1883,
        .client_id = "freertos_device_001",
        .keepalive = 60,
        .qos = 1,
        .clean_session = 1
    };
    
    g_mqtt_client = rtk_mqtt_client_create(&config);
    if (!g_mqtt_client) {
        printf("❌ 無法建立 MQTT 客戶端\n");
        vTaskDelete(NULL);
        return;
    }
    
    // 連接到 MQTT broker
    while (rtk_mqtt_client_connect(g_mqtt_client) != RTK_SUCCESS) {
        printf("⏳ 連接 MQTT broker 失敗，5秒後重試...\n");
        vTaskDelay(pdMS_TO_TICKS(5000));
    }
    
    printf("✓ MQTT 連接成功\n");
    
    // 網路處理循環
    while (1) {
        if (xSemaphoreTake(g_network_mutex, pdMS_TO_TICKS(100)) == pdTRUE) {
            rtk_mqtt_client_loop(g_mqtt_client);
            xSemaphoreGive(g_network_mutex);
        }
        
        vTaskDelay(pdMS_TO_TICKS(50));
    }
}

// === MQTT 處理任務 ===
void mqtt_task(void *pvParameters) {
    sensor_data_t sensor_data;
    
    printf("MQTT 任務啟動\n");
    
    // 等待 MQTT 客戶端就緒
    while (g_mqtt_client == NULL) {
        vTaskDelay(pdMS_TO_TICKS(500));
    }
    
    while (1) {
        // 從感測器佇列接收資料
        if (xQueueReceive(g_sensor_queue, &sensor_data, pdMS_TO_TICKS(1000)) == pdTRUE) {
            if (xSemaphoreTake(g_network_mutex, pdMS_TO_TICKS(500)) == pdTRUE) {
                // 發布遙測資料
                rtk_mqtt_client_publish_telemetry(
                    g_mqtt_client,
                    sensor_data.metric_name,
                    sensor_data.value,
                    sensor_data.unit
                );
                
                printf("📊 發布: %s = %.2f %s\n", 
                       sensor_data.metric_name, 
                       sensor_data.value, 
                       sensor_data.unit);
                
                xSemaphoreGive(g_network_mutex);
            }
        }
        
        // 每分鐘發送一次心跳
        static TickType_t last_heartbeat = 0;
        TickType_t current_time = xTaskGetTickCount();
        
        if ((current_time - last_heartbeat) > pdMS_TO_TICKS(60000)) {
            if (xSemaphoreTake(g_network_mutex, pdMS_TO_TICKS(500)) == pdTRUE) {
                rtk_mqtt_client_publish_state(g_mqtt_client, "online", "healthy");
                xSemaphoreGive(g_network_mutex);
                last_heartbeat = current_time;
                printf("💓 心跳發送\n");
            }
        }
    }
}

// === 感測器任務 ===
void sensor_task(void *pvParameters) {
    printf("感測器任務啟動\n");
    
    while (1) {
        sensor_data_t data;
        
        // 模擬感測器讀取
        snprintf(data.metric_name, sizeof(data.metric_name), "temperature");
        data.value = 25.0f + (rand() % 100) / 10.0f;  // 25-35°C
        snprintf(data.unit, sizeof(data.unit), "celsius");
        data.timestamp = xTaskGetTickCount();
        
        // 發送到 MQTT 任務
        if (xQueueSend(g_sensor_queue, &data, pdMS_TO_TICKS(100)) != pdTRUE) {
            printf("⚠️  感測器佇列滿了\n");
        }
        
        // 每10秒讀取一次
        vTaskDelay(pdMS_TO_TICKS(10000));
    }
}

// === 系統狀態監控任務 ===
void system_monitor_task(void *pvParameters) {
    printf("系統監控任務啟動\n");
    
    while (1) {
        // 檢查堆記憶體使用情況
        size_t free_heap = xPortGetFreeHeapSize();
        size_t min_free_heap = xPortGetMinimumEverFreeHeapSize();
        
        printf("🧠 記憶體: 可用=%d bytes, 最小可用=%d bytes\n", 
               (int)free_heap, (int)min_free_heap);
        
        // 如果記憶體不足，發出警告
        if (free_heap < 1024) {
            printf("⚠️  記憶體不足警告！\n");
            
            if (g_mqtt_client && (xSemaphoreTake(g_network_mutex, pdMS_TO_TICKS(500)) == pdTRUE)) {
                rtk_mqtt_client_publish_event(g_mqtt_client, "memory_warning", "Low memory detected");
                xSemaphoreGive(g_network_mutex);
            }
        }
        
        // 每30秒檢查一次
        vTaskDelay(pdMS_TO_TICKS(30000));
    }
}

// === 主程序 ===
int main(void) {
    // 硬體初始化
    system_init();
    
    printf("=== FreeRTOS IoT 設備啟動 ===\n");
    printf("版本: %s\n", VERSION);
    printf("FreeRTOS 版本: %s\n", tskKERNEL_VERSION_NUMBER);
    printf("系統時鐘: %lu Hz\n", configCPU_CLOCK_HZ);
    
    // 建立同步物件
    g_sensor_queue = xQueueCreate(10, sizeof(sensor_data_t));
    g_network_mutex = xSemaphoreCreateMutex();
    
    if (!g_sensor_queue || !g_network_mutex) {
        printf("❌ 無法建立同步物件\n");
        return -1;
    }
    
    // 建立任務
    BaseType_t result;
    
    result = xTaskCreate(network_task, "Network", STACK_SIZE_NETWORK_TASK, NULL, PRIORITY_NETWORK_TASK, NULL);
    if (result != pdPASS) {
        printf("❌ 無法建立網路任務\n");
        return -1;
    }
    
    result = xTaskCreate(mqtt_task, "MQTT", STACK_SIZE_MQTT_TASK, NULL, PRIORITY_MQTT_TASK, NULL);
    if (result != pdPASS) {
        printf("❌ 無法建立 MQTT 任務\n");
        return -1;
    }
    
    result = xTaskCreate(sensor_task, "Sensor", STACK_SIZE_SENSOR_TASK, NULL, PRIORITY_SENSOR_TASK, NULL);
    if (result != pdPASS) {
        printf("❌ 無法建立感測器任務\n");
        return -1;
    }
    
    result = xTaskCreate(system_monitor_task, "Monitor", configMINIMAL_STACK_SIZE * 2, NULL, tskIDLE_PRIORITY + 1, NULL);
    if (result != pdPASS) {
        printf("❌ 無法建立監控任務\n");
        return -1;
    }
    
    printf("✓ 所有任務建立成功，啟動排程器\n");
    
    // 啟動 FreeRTOS 排程器
    vTaskStartScheduler();
    
    // 不應該到達這裡
    printf("❌ 排程器意外停止\n");
    return -1;
}

// === FreeRTOS 回調函式 ===
void vApplicationStackOverflowHook(TaskHandle_t xTask, char *pcTaskName) {
    printf("❌ 堆疊溢出: %s\n", pcTaskName);
    for(;;);  // 停止系統
}

void vApplicationMallocFailedHook(void) {
    printf("❌ 記憶體分配失敗\n");
    for(;;);  // 停止系統
}

void vApplicationIdleHook(void) {
    // 空閒任務鉤子 - 可用於低功耗模式
    __WFI();  // 等待中斷
}
```

**第六步：編譯和燒錄**
```bash
# 編譯 FreeRTOS 專案
make -f Makefile.freertos

# 檢查記憶體使用
make size

# 燒錄到硬體 (使用 OpenOCD + ST-Link)
make flash

# 或手動燒錄
openocd -f interface/stlink-v2.cfg -f target/stm32f4x.cfg \
        -c "program freertos_iot_device.hex verify reset exit"

# 連接除錯器
make debug
```

**第七步：效能調校和最佳化**
```c
// config/rtk_freertos_config.h - RTK 框架 FreeRTOS 特定配置
#ifndef RTK_FREERTOS_CONFIG_H
#define RTK_FREERTOS_CONFIG_H

// === 記憶體最佳化 ===
#define RTK_USE_LIGHTWEIGHT_JSON        1
#define RTK_MINIMAL_MEMORY             1
#define RTK_MAX_TOPIC_LENGTH           128
#define RTK_MAX_MESSAGE_SIZE           512

// === 網路緩衝區設定 ===
#define RTK_NETWORK_BUFFER_SIZE        1024
#define RTK_MQTT_KEEPALIVE_DEFAULT     120
#define RTK_MQTT_TIMEOUT_MS            5000

// === 任務優先級 ===
#define RTK_NETWORK_TASK_PRIORITY      (configMAX_PRIORITIES - 1)
#define RTK_MQTT_TASK_PRIORITY         (configMAX_PRIORITIES - 2)

// === 除錯選項 ===
#ifdef DEBUG
#define RTK_DEBUG_ENABLED              1
#define RTK_DEBUG_NETWORK              1
#define RTK_DEBUG_MQTT                 1
#else
#define RTK_DEBUG_ENABLED              0
#endif

#endif /* RTK_FREERTOS_CONFIG_H */
```

---

## 第三部分：漸進式學習範例

框架提供了四個漸進式學習範例，從簡單到複雜，幫助您逐步掌握整合技巧。

### 01_hello_world：最小整合範例 (20行代碼)

**目標**：2分鐘內了解基本使用方式

```bash
cd examples/user_templates/01_hello_world
cat main.c        # 查看20行範例代碼
cat Makefile      # 查看獨立編譯設置  
make              # 編譯並執行
```

**學習重點**：
- RTK 框架初始化流程
- 基本 MQTT 連接
- 發布單一訊息
- 資源清理

### 02_basic_sensor：感測器模擬 (50行代碼)

**目標**：學習週期性遙測資料發布

```bash
cd examples/user_templates/02_basic_sensor
cat sensor.c      # 查看感測器模擬代碼
make              # 編譯
./basic_sensor    # 執行感測器模擬
```

**學習重點**：
- 週期性資料發布
- RTK 主題結構使用
- JSON 訊息格式
- 基本錯誤處理

### 03_complete_device：生產級設備範例

**目標**：了解生產環境最佳實務

```bash
cd examples/user_templates/03_complete_device
cat device.c      # 查看完整設備實作
cat config.json   # 查看配置檔案範例
make              # 編譯
./complete_device config.json
```

**學習重點**：
- 配置檔案管理
- 插件介面實作
- 信號處理
- 日誌記錄
- 生產級錯誤處理

### 04_cross_platform：跨平台整合

**目標**：學習不同平台的編譯差異

```bash
cd examples/user_templates/04_cross_platform
ls -la Makefile.*  # 查看平台特定 Makefile
cat README.md      # 查看跨平台說明

# Linux 編譯
make -f Makefile.linux

# Windows 編譯 (在 Windows 上)
make -f Makefile.windows

# FreeRTOS 編譯 (需要 ARM 工具鏈)
make -f Makefile.freertos
```

**學習重點**：
- 平台特定編譯設置
- 條件編譯使用
- 網路層抽象
- 記憶體管理差異

---

## 第四部分：進階主題

### 插件開發指南

RTK 框架支援插件架構，讓您可以為不同類型的設備開發專用功能。

#### 插件介面實作

```c
#include <rtk_device_plugin.h>

// 實作插件虛擬函式表
static rtk_device_plugin_vtable_t my_device_vtable = {
    .get_device_info = my_device_get_info,
    .initialize = my_device_initialize,
    .get_state = my_device_get_state,
    .cleanup = my_device_cleanup
};

// 插件入口點
RTK_PLUGIN_EXPORT const rtk_device_plugin_vtable_t* rtk_plugin_get_vtable() {
    return &my_device_vtable;
}

// 實作各個介面函式
static int my_device_get_info(rtk_device_info_t* info) {
    strncpy(info->device_type, "custom_sensor", sizeof(info->device_type));
    strncpy(info->manufacturer, "Your Company", sizeof(info->manufacturer));
    strncpy(info->model, "Model-123", sizeof(info->model));
    info->version_major = 1;
    info->version_minor = 0;
    return RTK_SUCCESS;
}

static int my_device_initialize(const rtk_plugin_config_t* config) {
    // 初始化您的設備特定功能
    printf("正在初始化自定義感測器...\n");
    return RTK_SUCCESS;
}

static int my_device_get_state(rtk_device_state_t* state) {
    // 讀取並回報設備狀態
    state->status = RTK_DEVICE_STATUS_ONLINE;
    state->health = RTK_DEVICE_HEALTH_HEALTHY;
    state->uptime = get_device_uptime();
    return RTK_SUCCESS;
}

static int my_device_cleanup(void) {
    // 清理資源
    printf("正在清理自定義感測器...\n");
    return RTK_SUCCESS;
}
```

#### 編譯插件

```makefile
# 插件 Makefile 範例
PLUGIN_NAME = my_device_plugin
RTK_INCLUDE_DIR = ../../include/rtk_mqtt_framework

# 編譯為共享函式庫
$(PLUGIN_NAME).so: $(PLUGIN_NAME).c
	$(CC) -shared -fPIC \
		-I$(RTK_INCLUDE_DIR) \
		-o $@ $< \
		-L../../lib -lrtk_mqtt_framework

# 測試插件
test: $(PLUGIN_NAME).so
	../../bin/plugin_demo -p ./$(PLUGIN_NAME).so -c config.json
```

### 配置管理

#### 配置檔案格式 (JSON)

```json
{
  "device": {
    "id": "sensor_001",
    "type": "temperature_sensor",
    "location": "office_room_1"
  },
  "mqtt": {
    "broker_host": "mqtt.example.com",
    "broker_port": 1883,
    "username": "device_user",
    "password": "device_pass",
    "client_id": "sensor_001",
    "keepalive": 60
  },
  "rtk": {
    "tenant": "my_company",
    "site": "office_building",
    "publish_interval": 30,
    "qos": 1
  },
  "logging": {
    "level": "info",
    "file": "/var/log/rtk_device.log"
  }
}
```

#### 配置載入範例

```c
#include <rtk_json_config.h>

// 載入並解析配置檔案
rtk_config_t* config = rtk_config_load_from_file("config.json");
if (config == NULL) {
    fprintf(stderr, "無法載入配置檔案\n");
    return -1;
}

// 存取配置值
const char* broker_host = rtk_config_get_string(config, "mqtt.broker_host");
int broker_port = rtk_config_get_int(config, "mqtt.broker_port");
int publish_interval = rtk_config_get_int(config, "rtk.publish_interval");

// 使用配置
printf("MQTT Broker: %s:%d\n", broker_host, broker_port);
printf("發布間隔: %d 秒\n", publish_interval);

// 清理
rtk_config_destroy(config);
```

### RTK MQTT 主題結構

RTK 框架使用標準化的主題階層結構：

```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}
```

#### 主題組件說明

- **tenant**: 租戶/公司識別符
- **site**: 站點/位置識別符  
- **device_id**: 設備唯一識別符
- **message_type**: 訊息類型

#### 支援的訊息類型

```c
// 設備狀態 (保留訊息)
rtk/v1/my_company/office/sensor_001/state

// 遙測資料
rtk/v1/my_company/office/sensor_001/telemetry/temperature
rtk/v1/my_company/office/sensor_001/telemetry/humidity
rtk/v1/my_company/office/sensor_001/telemetry/cpu_usage

// 事件和警報
rtk/v1/my_company/office/sensor_001/evt/sensor.high_temperature
rtk/v1/my_company/office/sensor_001/evt/system.startup
rtk/v1/my_company/office/sensor_001/evt/network.disconnected

// 設備屬性 (保留訊息)
rtk/v1/my_company/office/sensor_001/attr

// 命令介面
rtk/v1/my_company/office/sensor_001/cmd/req    # 命令請求
rtk/v1/my_company/office/sensor_001/cmd/ack    # 命令確認
rtk/v1/my_company/office/sensor_001/cmd/res    # 命令回應

// 遺言 (Last Will Testament)
rtk/v1/my_company/office/sensor_001/lwt
```

#### 主題建構範例

```c
#include <rtk_topic_builder.h>

// 初始化主題建構器
rtk_topic_builder_t* builder = rtk_topic_builder_create();
rtk_topic_builder_set_tenant(builder, "my_company");
rtk_topic_builder_set_site(builder, "office");
rtk_topic_builder_set_device_id(builder, "sensor_001");

// 建構不同類型的主題
char state_topic[256];
rtk_topic_builder_build_state(builder, state_topic, sizeof(state_topic));
// 結果: rtk/v1/my_company/office/sensor_001/state

char telemetry_topic[256];
rtk_topic_builder_build_telemetry(builder, "temperature", telemetry_topic, sizeof(telemetry_topic));
// 結果: rtk/v1/my_company/office/sensor_001/telemetry/temperature

char event_topic[256];
rtk_topic_builder_build_event(builder, "sensor.high_temperature", event_topic, sizeof(event_topic));
// 結果: rtk/v1/my_company/office/sensor_001/evt/sensor.high_temperature

// 清理
rtk_topic_builder_destroy(builder);
```

### 訊息格式規範

#### 設備狀態訊息

```json
{
  "status": "online",
  "health": "healthy",
  "uptime": 3600,
  "last_seen": 1692123456,
  "properties": {
    "temperature": 25.6,
    "humidity": 60.2,
    "battery_level": 85
  }
}
```

#### 遙測資料訊息

```json
{
  "metric": "temperature",
  "value": 25.6,
  "unit": "°C", 
  "timestamp": 1692123456,
  "labels": {
    "sensor": "internal",
    "location": "room_1"
  }
}
```

#### 事件訊息

```json
{
  "id": "evt_1692123456_001",
  "type": "sensor.high_temperature",
  "level": "warning",
  "message": "溫度超過閾值",
  "timestamp": 1692123456,
  "data": {
    "current_temperature": 35.2,
    "threshold": 30.0
  }
}
```

### 效能調校

#### 記憶體使用優化

```c
// 針對嵌入式系統的記憶體池配置
#ifdef RTK_PLATFORM_FREERTOS
    #define RTK_JSON_BUFFER_SIZE 1024    // 1KB 緩衝區
    #define RTK_JSON_MAX_DEPTH 8         // 最大巢狀深度
#else
    #define RTK_JSON_BUFFER_SIZE 4096    // 4KB 緩衝區  
    #define RTK_JSON_MAX_DEPTH 32        // 最大巢狀深度
#endif

// 初始化時設置記憶體限制
rtk_config_t config = {
    .memory_pool_size = RTK_JSON_BUFFER_SIZE,
    .max_json_depth = RTK_JSON_MAX_DEPTH,
    .max_concurrent_connections = 1
};
```

#### 網路效能優化

```c
// MQTT 連接參數調校
rtk_mqtt_config_t mqtt_config = {
    .keepalive = 60,              // 心跳間隔 (秒)
    .clean_session = 1,           // 清除會話
    .qos = 1,                     // QoS 等級
    .retain = 0,                  // 是否保留訊息
    .max_in_flight = 10,          // 最大飛行訊息數
    .message_timeout = 30,        // 訊息超時 (秒)
    .reconnect_interval = 5       // 重連間隔 (秒)
};
```

### 故障排除

#### 常見問題及解決方案

**問題1：編譯錯誤 - 找不到標頭檔**
```bash
# 錯誤：fatal error: rtk_mqtt_client.h: No such file or directory

# 解決方案：確認 include 路徑設置
export C_INCLUDE_PATH=/path/to/rtk_framework/include:$C_INCLUDE_PATH
# 或在 Makefile 中添加
CFLAGS += -I/path/to/rtk_framework/include/rtk_mqtt_framework
```

**問題2：連接失敗 - 無法連接到 MQTT broker**
```bash
# 錯誤：RTK_ERROR_CONNECTION_FAILED

# 解決方案：
# 1. 檢查網路連接
ping mqtt.example.com

# 2. 檢查防火牆設置
sudo ufw allow 1883

# 3. 使用測試 broker 進行驗證
# 在代碼中暫時使用：test.mosquitto.org:1883
```

**問題3：記憶體洩漏 - 長時間執行後記憶體不足**
```c
// 確保正確清理資源
void cleanup_rtk_resources() {
    if (mqtt_client) {
        rtk_mqtt_client_disconnect(mqtt_client);
        rtk_mqtt_client_destroy(mqtt_client);
        mqtt_client = NULL;
    }
    
    if (topic_builder) {
        rtk_topic_builder_destroy(topic_builder);
        topic_builder = NULL;
    }
    
    if (config) {
        rtk_config_destroy(config);
        config = NULL;
    }
}

// 在程式退出時調用
atexit(cleanup_rtk_resources);
```

**問題4：ARM 平台編譯失敗**
```bash
# 錯誤：arm-none-eabi-gcc: command not found

# 解決方案：安裝 ARM 工具鏈
# Ubuntu/Debian
sudo apt-get install gcc-arm-none-eabi binutils-arm-none-eabi

# macOS
brew install arm-none-eabi-gcc

# 驗證安裝
arm-none-eabi-gcc --version
```

#### 除錯模式啟用

```c
// 編譯時啟用除錯輸出
#define RTK_DEBUG 1
#include <rtk_mqtt_client.h>

// 設置日誌等級
rtk_log_set_level(RTK_LOG_LEVEL_DEBUG);
rtk_log_set_output(stdout);

// 啟用 MQTT 除錯
rtk_mqtt_client_set_debug(client, 1);
```

```bash
# 使用除錯編譯
gcc -DRTK_DEBUG=1 -g -O0 \
    -I./include/rtk_mqtt_framework \
    -o my_device_debug my_device.c \
    -L./lib -lrtk_mqtt_framework -lpthread -lm

# 使用 GDB 除錯
gdb ./my_device_debug
(gdb) set environment RTK_LOG_LEVEL=DEBUG
(gdb) run
```

### 常見整合模式

#### 模式1：簡單感測器設備

適用於單一功能感測器，定期回報資料。

```c
// simple_sensor.c
#include <rtk_mqtt_client.h>
#include <unistd.h>

int main() {
    rtk_mqtt_client_t* client = rtk_mqtt_client_create("mqtt.example.com", 1883, "sensor_001");
    
    if (rtk_mqtt_client_connect(client) == RTK_SUCCESS) {
        while (1) {
            // 讀取感測器資料
            float temperature = read_temperature_sensor();
            
            // 發布遙測資料
            rtk_mqtt_client_publish_telemetry(client, "temperature", temperature, "°C");
            
            // 等待30秒
            sleep(30);
        }
    }
    
    rtk_mqtt_client_destroy(client);
    return 0;
}
```

#### 模式2：多功能 IoT 設備

適用於具有多種感測器和控制功能的複雜設備。

```c
// complex_iot_device.c
#include <rtk_mqtt_client.h>
#include <pthread.h>

typedef struct {
    rtk_mqtt_client_t* mqtt_client;
    int running;
} device_context_t;

// 感測器資料收集執行緒
void* sensor_thread(void* arg) {
    device_context_t* ctx = (device_context_t*)arg;
    
    while (ctx->running) {
        // 收集多種感測器資料
        float temp = read_temperature();
        float humidity = read_humidity(); 
        int battery = read_battery_level();
        
        // 批次發布
        rtk_mqtt_client_publish_telemetry(ctx->mqtt_client, "temperature", temp, "°C");
        rtk_mqtt_client_publish_telemetry(ctx->mqtt_client, "humidity", humidity, "%");
        rtk_mqtt_client_publish_telemetry(ctx->mqtt_client, "battery", battery, "%");
        
        sleep(60);
    }
    return NULL;
}

// 命令處理執行緒
void* command_thread(void* arg) {
    device_context_t* ctx = (device_context_t*)arg;
    
    // 訂閱命令主題
    rtk_mqtt_client_subscribe_commands(ctx->mqtt_client, command_handler);
    
    while (ctx->running) {
        rtk_mqtt_client_process_messages(ctx->mqtt_client, 1000);
    }
    return NULL;
}

int main() {
    device_context_t ctx = {0};
    ctx.mqtt_client = rtk_mqtt_client_create("mqtt.example.com", 1883, "iot_device_001");
    ctx.running = 1;
    
    if (rtk_mqtt_client_connect(ctx.mqtt_client) == RTK_SUCCESS) {
        pthread_t sensor_tid, command_tid;
        
        // 啟動工作執行緒
        pthread_create(&sensor_tid, NULL, sensor_thread, &ctx);
        pthread_create(&command_tid, NULL, command_thread, &ctx);
        
        // 等待信號
        wait_for_shutdown_signal();
        
        // 清理
        ctx.running = 0;
        pthread_join(sensor_tid, NULL);
        pthread_join(command_tid, NULL);
    }
    
    rtk_mqtt_client_destroy(ctx.mqtt_client);
    return 0;
}
```

#### 模式3：邊緣閘道器

適用於聚合多個設備資料的邊緣運算裝置。

```c
// edge_gateway.c
#include <rtk_mqtt_client.h>

typedef struct {
    char device_id[64];
    rtk_mqtt_client_t* client;
} device_proxy_t;

// 設備代理清單
device_proxy_t device_proxies[MAX_DEVICES];
int device_count = 0;

// 註冊新設備
int register_device(const char* device_id) {
    if (device_count >= MAX_DEVICES) return -1;
    
    device_proxy_t* proxy = &device_proxies[device_count++];
    strncpy(proxy->device_id, device_id, sizeof(proxy->device_id));
    
    // 為每個設備建立獨立的 MQTT 客戶端
    proxy->client = rtk_mqtt_client_create("mqtt.example.com", 1883, device_id);
    rtk_mqtt_client_connect(proxy->client);
    
    printf("已註冊設備: %s\n", device_id);
    return 0;
}

// 轉發設備資料
void forward_device_data(const char* device_id, const char* metric, float value, const char* unit) {
    for (int i = 0; i < device_count; i++) {
        if (strcmp(device_proxies[i].device_id, device_id) == 0) {
            rtk_mqtt_client_publish_telemetry(device_proxies[i].client, metric, value, unit);
            break;
        }
    }
}

int main() {
    // 註冊連接的設備
    register_device("sensor_001");
    register_device("sensor_002");
    register_device("actuator_001");
    
    // 主迴圈：收集本地設備資料並轉發
    while (1) {
        // 從本地設備收集資料
        collect_local_device_data();
        
        // 轉發到雲端
        forward_all_device_data();
        
        sleep(30);
    }
    
    return 0;
}
```

---

## 總結

RTK MQTT Framework 提供了一個完整、易用的解決方案，讓您可以快速為 IoT 設備添加 MQTT 診斷功能。通過本手冊的指導，您應該能夠：

1. **快速上手**：在5分鐘內完成第一個 Hello World 程式
2. **漸進學習**：通過四個範例逐步掌握框架功能  
3. **平台整合**：在 Linux、Windows、FreeRTOS 上成功整合
4. **進階應用**：開發插件、配置管理、效能調校
5. **問題解決**：診斷和解決常見的整合問題

如果您遇到本手冊未涵蓋的問題，請參考：
- **技術文件**：`docs/` 目錄中的詳細規範
- **範例代碼**：`examples/` 目錄中的完整實作
- **發行說明**：`RELEASE.md` 中的版本資訊

祝您整合順利！
# FreeRTOS 網路架構設計文件

## 概述

RTK MQTT Framework 提供了統一的網路抽象層，支援多平台網路實作，包括 POSIX、Windows 和 FreeRTOS。本文件詳細說明 FreeRTOS 平台的網路套接字抽象設計。

## 網路堆疊支援

### 1. FreeRTOS+TCP (官方堆疊)
```c
#define RTK_USE_FREERTOS_TCP

// API 映射
#include "FreeRTOS_Sockets.h"
#include "FreeRTOS_IP.h"

Socket_t socket = FreeRTOS_socket(FREERTOS_AF_INET, FREERTOS_SOCK_STREAM, FREERTOS_IPPROTO_TCP);
FreeRTOS_connect(socket, &address, sizeof(address));
FreeRTOS_send(socket, data, length, 0);
FreeRTOS_recv(socket, buffer, size, 0);
FreeRTOS_closesocket(socket);
```

### 2. lwIP (輕量級 IP 堆疊) - 預設
```c
// 不定義 RTK_USE_FREERTOS_TCP (預設使用 lwIP)

// API 映射
#include "lwip/sockets.h"
#include "lwip/netdb.h"

int socket = lwip_socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
lwip_connect(socket, &address, sizeof(address));
lwip_send(socket, data, length, 0);
lwip_recv(socket, buffer, size, 0);
lwip_close(socket);
```

## 抽象層架構

### 核心抽象結構

```c
typedef struct rtk_network_interface {
    void* context;                                    // 平台特定上下文
    int (*connect)(rtk_network_interface_t*, const char*, uint16_t);
    int (*disconnect)(rtk_network_interface_t*);
    int (*send)(rtk_network_interface_t*, const void*, size_t, size_t*);
    int (*receive)(rtk_network_interface_t*, void*, size_t, size_t*);
    int (*is_connected)(rtk_network_interface_t*);
    int (*get_status)(rtk_network_interface_t*, rtk_network_status_t*);
    void (*cleanup)(rtk_network_interface_t*);
} rtk_network_interface_t;
```

### FreeRTOS 特定上下文

```c
typedef struct {
    SOCKET_TYPE sockfd;              // Socket 描述符 (Socket_t 或 int)
    int connected;                   // 連線狀態
    char remote_host[256];           // 遠端主機名稱
    uint16_t remote_port;            // 遠端埠號
    uint32_t connect_timeout_ms;     // 連線超時
    uint32_t send_timeout_ms;        // 發送超時
    uint32_t recv_timeout_ms;        // 接收超時
} freertos_network_context_t;
```

## 編譯時配置

### 1. CMake 配置
```cmake
# 啟用 FreeRTOS 平台
set(RTK_TARGET_FREERTOS ON)

# 選擇網路堆疊
option(RTK_USE_FREERTOS_TCP "Use FreeRTOS+TCP stack" OFF)
option(RTK_USE_LWIP "Use lwIP stack" ON)

# 編譯條件
if(RTK_TARGET_FREERTOS)
    if(RTK_USE_FREERTOS_TCP)
        target_compile_definitions(rtk_mqtt_framework PRIVATE RTK_USE_FREERTOS_TCP)
    endif()
endif()
```

### 2. 編譯器定義
```c
// 在 rtk_platform_compat.h 中
#ifdef RTK_TARGET_FREERTOS
    #define RTK_PLATFORM_FREERTOS
#endif

// 在 network_freertos.c 中
#ifdef RTK_PLATFORM_FREERTOS
    // FreeRTOS 網路實作
#endif
```

## API 映射機制

### 套接字類型統一
```c
#ifdef RTK_USE_FREERTOS_TCP
    #define SOCKET_TYPE Socket_t
    #define INVALID_SOCKET FREERTOS_INVALID_SOCKET
#else
    #define SOCKET_TYPE int
    #define INVALID_SOCKET -1
#endif
```

### 函式映射
```c
#ifdef RTK_USE_FREERTOS_TCP
    #define socket_create(f,t,p) FreeRTOS_socket(f,t,p)
    #define socket_connect(s,a,l) FreeRTOS_connect(s,a,l)
    #define socket_send(s,d,l,f) FreeRTOS_send(s,d,l,f)
    #define socket_recv(s,b,l,f) FreeRTOS_recv(s,b,l,f)
    #define socket_close(s) FreeRTOS_closesocket(s)
#else
    #define socket_create(f,t,p) lwip_socket(f,t,p)
    #define socket_connect(s,a,l) lwip_connect(s,a,l)
    #define socket_send(s,d,l,f) lwip_send(s,d,l,f)
    #define socket_recv(s,b,l,f) lwip_recv(s,b,l,f)
    #define socket_close(s) lwip_close(s)
#endif
```

## 實作細節

### 1. 連線建立
```c
static int freertos_network_connect(rtk_network_interface_t* iface, 
                                   const char* host, uint16_t port) {
    freertos_network_context_t* ctx = iface->context;
    
    // 1. 創建 socket
    ctx->sockfd = socket_create(ADDR_FAMILY, SOCK_TYPE_STREAM, PROTO_TCP);
    
    // 2. 設定 socket 選項 (超時等)
    freertos_set_socket_options(ctx->sockfd, ctx);
    
    // 3. DNS 解析
    uint32_t ip_addr;
    freertos_resolve_hostname(host, &ip_addr);
    
    // 4. 建立連線
    struct sockaddr_in server_addr;
    // ... 設定地址
    socket_connect(ctx->sockfd, &server_addr, sizeof(server_addr));
    
    ctx->connected = 1;
    return RTK_NETWORK_SUCCESS;
}
```

### 2. 資料傳輸
```c
static int freertos_network_send(rtk_network_interface_t* iface, 
                                const void* data, size_t len, size_t* sent) {
    freertos_network_context_t* ctx = iface->context;
    
    // 檢查連線狀態
    if (!ctx->connected || ctx->sockfd == INVALID_SOCKET) {
        return RTK_NETWORK_ERROR_NOT_CONNECTED;
    }
    
    // 發送資料
    int result = socket_send(ctx->sockfd, data, len, 0);
    if (result < 0) {
        ctx->connected = 0;  // 標記連線失效
        return RTK_NETWORK_ERROR_SOCKET_ERROR;
    }
    
    *sent = (size_t)result;
    return RTK_NETWORK_SUCCESS;
}
```

### 3. DNS 解析
```c
static int freertos_resolve_hostname(const char* hostname, uint32_t* ip_addr) {
#ifdef RTK_USE_FREERTOS_TCP
    // 使用 FreeRTOS+TCP DNS
    *ip_addr = FreeRTOS_inet_addr(hostname);
    if (*ip_addr == FREERTOS_INADDR_NONE) {
        *ip_addr = FreeRTOS_gethostbyname(hostname);
    }
#else
    // 使用 lwIP DNS
    struct hostent* host_entry = lwip_gethostbyname(hostname);
    if (host_entry) {
        *ip_addr = ((struct in_addr*)host_entry->h_addr)->s_addr;
    }
#endif
    return RTK_NETWORK_SUCCESS;
}
```

## 超時處理

### FreeRTOS+TCP 超時
```c
// 設定接收超時
TickType_t timeout_ticks = pdMS_TO_TICKS(timeout_ms);
FreeRTOS_setsockopt(sockfd, 0, FREERTOS_SO_RCVTIMEO, 
                   &timeout_ticks, sizeof(timeout_ticks));
```

### lwIP 超時
```c
// 設定接收超時
struct timeval timeout;
timeout.tv_sec = timeout_ms / 1000;
timeout.tv_usec = (timeout_ms % 1000) * 1000;
lwip_setsockopt(sockfd, SOL_SOCKET, SO_RCVTIMEO, 
               &timeout, sizeof(timeout));
```

## 記憶體管理

### 上下文分配
```c
int rtk_network_create_freertos(rtk_network_interface_t* iface) {
    // 使用 FreeRTOS 記憶體分配
    freertos_network_context_t* ctx = rtk_calloc(1, sizeof(freertos_network_context_t));
    
    // 初始化預設值
    ctx->sockfd = INVALID_SOCKET;
    ctx->connected = 0;
    ctx->connect_timeout_ms = 30000;
    ctx->send_timeout_ms = 10000;
    ctx->recv_timeout_ms = 10000;
    
    // 設定函式指標
    iface->context = ctx;
    iface->connect = freertos_network_connect;
    iface->disconnect = freertos_network_disconnect;
    // ...
    
    return RTK_NETWORK_SUCCESS;
}
```

## 錯誤處理

### 統一錯誤碼
```c
typedef enum {
    RTK_NETWORK_SUCCESS = 0,
    RTK_NETWORK_ERROR_INVALID_PARAM = -1,
    RTK_NETWORK_ERROR_CONNECTION_FAILED = -2,
    RTK_NETWORK_ERROR_TIMEOUT = -3,
    RTK_NETWORK_ERROR_MEMORY = -4,
    RTK_NETWORK_ERROR_NOT_CONNECTED = -5,
    RTK_NETWORK_ERROR_SOCKET_ERROR = -7,
    RTK_NETWORK_ERROR_DNS_FAILED = -8
} rtk_network_error_t;
```

### 連線狀態追蹤
```c
static int freertos_network_is_connected(rtk_network_interface_t* iface) {
    freertos_network_context_t* ctx = iface->context;
    
    if (!ctx->connected || ctx->sockfd == INVALID_SOCKET) {
        return 0;
    }
    
    // 可選：檢查 socket 健康狀態
    // (FreeRTOS+TCP 和 lwIP 可能有不同的檢查方式)
    
    return 1;
}
```

## 使用範例

### 初始化和使用
```c
#include "rtk_network_interface.h"

// 1. 創建網路介面
rtk_network_interface_t network_iface;
int result = rtk_network_create_freertos(&network_iface);

// 2. 設定超時參數
rtk_network_freertos_set_timeouts(&network_iface, 30000, 10000, 10000);

// 3. 連線到 MQTT 伺服器
result = network_iface.connect(&network_iface, "mqtt.broker.com", 1883);

// 4. 發送 MQTT 資料
size_t sent;
result = network_iface.send(&network_iface, mqtt_packet, packet_len, &sent);

// 5. 接收 MQTT 資料
size_t received;
result = network_iface.receive(&network_iface, buffer, buffer_size, &received);

// 6. 清理
network_iface.cleanup(&network_iface);
```

### 與 MQTT 客戶端整合
```c
// 在 rtk_mqtt_client.c 中
static int mqtt_client_init_network(rtk_mqtt_client_t* client) {
    #ifdef RTK_PLATFORM_FREERTOS
        return rtk_network_create_freertos(&client->network_interface);
    #elif defined(RTK_PLATFORM_WINDOWS)
        return rtk_network_create_windows(&client->network_interface);
    #else
        return rtk_network_create_posix(&client->network_interface);
    #endif
}
```

## 調試和診斷

### 連線狀態監控
```c
rtk_network_status_t status;
network_iface.get_status(&network_iface, &status);

printf("Connected: %d\n", status.connected);
printf("Remote Host: %s:%d\n", status.remote_host, status.remote_port);
printf("Socket FD: %d\n", status.socket_fd);
```

### 錯誤資訊獲取
```c
if (result != RTK_NETWORK_SUCCESS) {
    const char* error_msg = rtk_network_get_error_string(result);
    printf("Network error: %s\n", error_msg);
}
```

## 效能考量

### 緩衝區大小
- **發送緩衝區**: 建議 1-4KB，根據 MQTT 訊息大小調整
- **接收緩衝區**: 建議 2-8KB，支援大型 MQTT 訊息
- **FreeRTOS 堆疊大小**: 建議至少 4KB 給網路任務

### 記憶體使用
- **網路上下文**: ~300 bytes
- **Socket 緩衝區**: 依網路堆疊配置而定
- **DNS 查詢**: 暫時性記憶體使用

### 任務優先級
- **網路任務**: 建議中等優先級 (tskIDLE_PRIORITY + 3)
- **MQTT 任務**: 建議高於網路任務 (tskIDLE_PRIORITY + 4)

這個抽象層設計確保了 RTK MQTT Framework 能夠在各種 FreeRTOS 配置下穩定運行，同時保持代碼的可移植性和可維護性。
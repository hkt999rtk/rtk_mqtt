#include "rtk_network_interface.h"
#include "rtk_platform_compat.h"

#ifdef RTK_PLATFORM_FREERTOS

// 選擇網路堆疊
#ifdef RTK_USE_FREERTOS_TCP
    // 使用 FreeRTOS+TCP
    #include "FreeRTOS_Sockets.h"
    #include "FreeRTOS_IP.h"
    #define SOCKET_TYPE Socket_t
    #define INVALID_SOCKET FREERTOS_INVALID_SOCKET
    #define SOCKET_ERROR FREERTOS_SOCKET_ERROR
    #define socket_close(s) FreeRTOS_closesocket(s)
    #define socket_connect(s, addr, len) FreeRTOS_connect(s, addr, len)
    #define socket_send(s, data, len, flags) FreeRTOS_send(s, data, len, flags)
    #define socket_recv(s, buf, len, flags) FreeRTOS_recv(s, buf, len, flags)
    #define socket_create(family, type, protocol) FreeRTOS_socket(family, type, protocol)
    #define ADDR_FAMILY FREERTOS_AF_INET
    #define SOCK_TYPE_STREAM FREERTOS_SOCK_STREAM
    #define PROTO_TCP FREERTOS_IPPROTO_TCP
#else
    // 使用 lwIP (預設)
    #include "lwip/sockets.h"
    #include "lwip/netdb.h"
    #include "lwip/sys.h"
    #define SOCKET_TYPE int
    #define INVALID_SOCKET -1
    #define SOCKET_ERROR -1
    #define socket_close(s) lwip_close(s)
    #define socket_connect(s, addr, len) lwip_connect(s, addr, len)
    #define socket_send(s, data, len, flags) lwip_send(s, data, len, flags)
    #define socket_recv(s, buf, len, flags) lwip_recv(s, buf, len, flags)
    #define socket_create(family, type, protocol) lwip_socket(family, type, protocol)
    #define ADDR_FAMILY AF_INET
    #define SOCK_TYPE_STREAM SOCK_STREAM
    #define PROTO_TCP IPPROTO_TCP
#endif

#include <string.h>

/**
 * @file network_freertos.c
 * @brief FreeRTOS 網路介面實現
 * 
 * 支援兩種網路堆疊：
 * 1. FreeRTOS+TCP (官方堆疊)
 * 2. lwIP (更常用的開源堆疊)
 */

typedef struct {
    SOCKET_TYPE sockfd;
    int connected;
    char remote_host[256];
    uint16_t remote_port;
    uint32_t connect_timeout_ms;
    uint32_t send_timeout_ms;
    uint32_t recv_timeout_ms;
} freertos_network_context_t;

// === 內部輔助函式 ===

static int freertos_resolve_hostname(const char* hostname, uint32_t* ip_addr) {
    if (!hostname || !ip_addr) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
#ifdef RTK_USE_FREERTOS_TCP
    *ip_addr = FreeRTOS_inet_addr(hostname);
    if (*ip_addr == FREERTOS_INADDR_NONE) {
        // 嘗試 DNS 解析
        *ip_addr = FreeRTOS_gethostbyname(hostname);
        if (*ip_addr == 0) {
            return RTK_NETWORK_ERROR_DNS_FAILED;
        }
    }
#else
    // 使用 lwIP
    struct hostent* host_entry = lwip_gethostbyname(hostname);
    if (!host_entry) {
        return RTK_NETWORK_ERROR_DNS_FAILED;
    }
    *ip_addr = ((struct in_addr*)host_entry->h_addr)->s_addr;
#endif
    
    return RTK_NETWORK_SUCCESS;
}

static int freertos_set_socket_options(SOCKET_TYPE sockfd, freertos_network_context_t* ctx) {
    // 設定接收超時
    if (ctx->recv_timeout_ms > 0) {
#ifdef RTK_USE_FREERTOS_TCP
        TickType_t timeout_ticks = pdMS_TO_TICKS(ctx->recv_timeout_ms);
        FreeRTOS_setsockopt(sockfd, 0, FREERTOS_SO_RCVTIMEO, &timeout_ticks, sizeof(timeout_ticks));
#else
        struct timeval timeout;
        timeout.tv_sec = ctx->recv_timeout_ms / 1000;
        timeout.tv_usec = (ctx->recv_timeout_ms % 1000) * 1000;
        lwip_setsockopt(sockfd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));
#endif
    }
    
    // 設定發送超時
    if (ctx->send_timeout_ms > 0) {
#ifdef RTK_USE_FREERTOS_TCP
        TickType_t timeout_ticks = pdMS_TO_TICKS(ctx->send_timeout_ms);
        FreeRTOS_setsockopt(sockfd, 0, FREERTOS_SO_SNDTIMEO, &timeout_ticks, sizeof(timeout_ticks));
#else
        struct timeval timeout;
        timeout.tv_sec = ctx->send_timeout_ms / 1000;
        timeout.tv_usec = (ctx->send_timeout_ms % 1000) * 1000;
        lwip_setsockopt(sockfd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));
#endif
    }
    
    return RTK_NETWORK_SUCCESS;
}

// === 網路介面實作 ===

static int freertos_network_connect(rtk_network_interface_t* iface, const char* host, uint16_t port) {
    if (!iface || !host) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    freertos_network_context_t* ctx = (freertos_network_context_t*)iface->context;
    if (!ctx) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    // 如果已經連線，先關閉
    if (ctx->connected && ctx->sockfd != INVALID_SOCKET) {
        socket_close(ctx->sockfd);
        ctx->connected = 0;
    }
    
    // 創建 socket
    ctx->sockfd = socket_create(ADDR_FAMILY, SOCK_TYPE_STREAM, PROTO_TCP);
    if (ctx->sockfd == INVALID_SOCKET) {
        return RTK_NETWORK_ERROR_SOCKET_ERROR;
    }
    
    // 設定 socket 選項
    freertos_set_socket_options(ctx->sockfd, ctx);
    
    // 解析主機名
    uint32_t ip_addr;
    int result = freertos_resolve_hostname(host, &ip_addr);
    if (result != RTK_NETWORK_SUCCESS) {
        socket_close(ctx->sockfd);
        ctx->sockfd = INVALID_SOCKET;
        return result;
    }
    
    // 設定伺服器地址
#ifdef RTK_USE_FREERTOS_TCP
    struct freertos_sockaddr server_addr;
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = FREERTOS_AF_INET;
    server_addr.sin_port = FreeRTOS_htons(port);
    server_addr.sin_addr = ip_addr;
#else
    struct sockaddr_in server_addr;
    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(port);
    server_addr.sin_addr.s_addr = ip_addr;
#endif
    
    // 連線 (帶超時控制)
    TickType_t start_time = xTaskGetTickCount();
    TickType_t timeout_ticks = pdMS_TO_TICKS(ctx->connect_timeout_ms);
    
    result = socket_connect(ctx->sockfd, (struct sockaddr*)&server_addr, sizeof(server_addr));
    if (result != 0) {
        socket_close(ctx->sockfd);
        ctx->sockfd = INVALID_SOCKET;
        return RTK_NETWORK_ERROR_CONNECTION_FAILED;
    }
    
    ctx->connected = 1;
    strncpy(ctx->remote_host, host, sizeof(ctx->remote_host) - 1);
    ctx->remote_host[sizeof(ctx->remote_host) - 1] = '\0';
    ctx->remote_port = port;
    
    return RTK_NETWORK_SUCCESS;
}

static int freertos_network_disconnect(rtk_network_interface_t* iface) {
    if (!iface) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    freertos_network_context_t* ctx = (freertos_network_context_t*)iface->context;
    if (!ctx) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    if (ctx->sockfd != INVALID_SOCKET) {
        socket_close(ctx->sockfd);
        ctx->sockfd = INVALID_SOCKET;
    }
    
    ctx->connected = 0;
    return RTK_NETWORK_SUCCESS;
}

static int freertos_network_send(rtk_network_interface_t* iface, const void* data, size_t len, size_t* sent) {
    if (!iface || !data || !sent) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    freertos_network_context_t* ctx = (freertos_network_context_t*)iface->context;
    if (!ctx || !ctx->connected || ctx->sockfd == INVALID_SOCKET) {
        return RTK_NETWORK_ERROR_NOT_CONNECTED;
    }
    
    int result = socket_send(ctx->sockfd, data, len, 0);
    if (result < 0) {
        ctx->connected = 0;
        return RTK_NETWORK_ERROR_SOCKET_ERROR;
    }
    
    *sent = (size_t)result;
    return RTK_NETWORK_SUCCESS;
}

static int freertos_network_receive(rtk_network_interface_t* iface, void* buffer, size_t len, size_t* received) {
    if (!iface || !buffer || !received) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    freertos_network_context_t* ctx = (freertos_network_context_t*)iface->context;
    if (!ctx || !ctx->connected || ctx->sockfd == INVALID_SOCKET) {
        return RTK_NETWORK_ERROR_NOT_CONNECTED;
    }
    
    int result = socket_recv(ctx->sockfd, buffer, len, 0);
    if (result < 0) {
        ctx->connected = 0;
        return RTK_NETWORK_ERROR_SOCKET_ERROR;
    } else if (result == 0) {
        // 連線已關閉
        ctx->connected = 0;
        *received = 0;
        return RTK_NETWORK_ERROR_CONNECTION_FAILED;
    }
    
    *received = (size_t)result;
    return RTK_NETWORK_SUCCESS;
}

static int freertos_network_is_connected(rtk_network_interface_t* iface) {
    if (!iface) {
        return 0;
    }
    
    freertos_network_context_t* ctx = (freertos_network_context_t*)iface->context;
    if (!ctx) {
        return 0;
    }
    
    return (ctx->connected && ctx->sockfd != INVALID_SOCKET) ? 1 : 0;
}

static int freertos_network_get_status(rtk_network_interface_t* iface, rtk_network_status_t* status) {
    if (!iface || !status) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    freertos_network_context_t* ctx = (freertos_network_context_t*)iface->context;
    if (!ctx) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    memset(status, 0, sizeof(rtk_network_status_t));
    
    if (ctx->connected && ctx->sockfd != INVALID_SOCKET) {
        status->connected = 1;
        strncpy(status->remote_host, ctx->remote_host, sizeof(status->remote_host) - 1);
        status->remote_port = ctx->remote_port;
        status->socket_fd = (int)ctx->sockfd;
    }
    
    return RTK_NETWORK_SUCCESS;
}

static void freertos_network_cleanup(rtk_network_interface_t* iface) {
    if (!iface) {
        return;
    }
    
    freertos_network_disconnect(iface);
    
    if (iface->context) {
        rtk_free(iface->context);
        iface->context = NULL;
    }
}

// === 公開 API ===

int rtk_network_create_freertos(rtk_network_interface_t* iface) {
    if (!iface) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    // 分配上下文
    freertos_network_context_t* ctx = (freertos_network_context_t*)rtk_calloc(1, sizeof(freertos_network_context_t));
    if (!ctx) {
        return RTK_NETWORK_ERROR_MEMORY;
    }
    
    // 初始化上下文
    ctx->sockfd = INVALID_SOCKET;
    ctx->connected = 0;
    ctx->connect_timeout_ms = 30000;  // 30 秒
    ctx->send_timeout_ms = 10000;     // 10 秒
    ctx->recv_timeout_ms = 10000;     // 10 秒
    
    // 設定介面
    memset(iface, 0, sizeof(rtk_network_interface_t));
    iface->context = ctx;
    iface->connect = freertos_network_connect;
    iface->disconnect = freertos_network_disconnect;
    iface->send = freertos_network_send;
    iface->receive = freertos_network_receive;
    iface->is_connected = freertos_network_is_connected;
    iface->get_status = freertos_network_get_status;
    iface->cleanup = freertos_network_cleanup;
    
    return RTK_NETWORK_SUCCESS;
}

// === 配置 API ===

int rtk_network_freertos_set_timeouts(rtk_network_interface_t* iface, 
                                      uint32_t connect_timeout_ms,
                                      uint32_t send_timeout_ms, 
                                      uint32_t recv_timeout_ms) {
    if (!iface) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    freertos_network_context_t* ctx = (freertos_network_context_t*)iface->context;
    if (!ctx) {
        return RTK_NETWORK_ERROR_INVALID_PARAM;
    }
    
    ctx->connect_timeout_ms = connect_timeout_ms;
    ctx->send_timeout_ms = send_timeout_ms;
    ctx->recv_timeout_ms = recv_timeout_ms;
    
    // 如果 socket 已經存在，更新設定
    if (ctx->sockfd != INVALID_SOCKET) {
        freertos_set_socket_options(ctx->sockfd, ctx);
    }
    
    return RTK_NETWORK_SUCCESS;
}

#endif // RTK_PLATFORM_FREERTOS
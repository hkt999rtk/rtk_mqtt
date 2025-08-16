#ifndef RTK_NETWORK_INTERFACE_H
#define RTK_NETWORK_INTERFACE_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/**
 * @file rtk_network_interface.h
 * @brief 網路層抽象介面 - 支援多平台網路實作
 * 
 * 提供統一的網路操作介面，支援：
 * - POSIX Socket (Linux/macOS/Unix)
 * - Windows Socket (Winsock)
 * - FreeRTOS 網路堆疊
 * - 自定義網路實作
 */

// === 前向宣告 ===

typedef struct rtk_network_interface rtk_network_interface_t;
typedef struct rtk_network_config rtk_network_config_t;

// === 列舉定義 ===

/**
 * @brief 網路錯誤碼
 */
typedef enum {
    RTK_NETWORK_SUCCESS = 0,
    RTK_NETWORK_ERROR_INVALID_PARAM = -1,
    RTK_NETWORK_ERROR_CONNECTION_FAILED = -2,
    RTK_NETWORK_ERROR_TIMEOUT = -3,
    RTK_NETWORK_ERROR_MEMORY = -4,
    RTK_NETWORK_ERROR_NOT_CONNECTED = -5,
    RTK_NETWORK_ERROR_WOULD_BLOCK = -6,
    RTK_NETWORK_ERROR_SOCKET_ERROR = -7,
    RTK_NETWORK_ERROR_DNS_FAILED = -8,
    RTK_NETWORK_ERROR_SOCKET_CREATE = -9,
    RTK_NETWORK_ERROR_HOST_RESOLVE = -10,
    RTK_NETWORK_ERROR_CONNECT = -11,
    RTK_NETWORK_ERROR_SEND = -12,
    RTK_NETWORK_ERROR_RECV = -13,
    RTK_NETWORK_ERROR_UNKNOWN = -99
} rtk_network_error_t;

/**
 * @brief 網路連線類型
 */
typedef enum {
    RTK_NETWORK_TYPE_TCP = 0,
    RTK_NETWORK_TYPE_UDP = 1,
    RTK_NETWORK_TYPE_SSL = 2,
    RTK_NETWORK_TYPE_TLS = 3
} rtk_network_type_t;

/**
 * @brief 網路平台類型
 */
typedef enum {
    RTK_NETWORK_PLATFORM_POSIX = 0,
    RTK_NETWORK_PLATFORM_WINDOWS = 1,
    RTK_NETWORK_PLATFORM_FREERTOS = 2,
    RTK_NETWORK_PLATFORM_CUSTOM = 99
} rtk_network_platform_t;

// === 結構定義 ===

/**
 * @brief 網路配置參數
 */
struct rtk_network_config {
    rtk_network_type_t type;         /**< 連線類型 */
    int socket_timeout_ms;           /**< Socket 超時 (毫秒) */
    int connect_timeout_ms;          /**< 連線超時 (毫秒) */
    int recv_timeout_ms;             /**< 接收超時 (毫秒) */
    int send_timeout_ms;             /**< 發送超時 (毫秒) */
    
    int keep_alive;                  /**< 啟用 TCP Keep-Alive */
    int tcp_nodelay;                 /**< 啟用 TCP_NODELAY */
    int reuse_addr;                  /**< 啟用地址重用 */
    
    // SSL/TLS 設定 (如果支援)
    char ca_cert_file[256];          /**< CA 憑證檔案路徑 */
    char client_cert_file[256];      /**< 客戶端憑證檔案路徑 */
    char client_key_file[256];       /**< 客戶端私鑰檔案路徑 */
    int verify_cert;                 /**< 是否驗證伺服器憑證 */
    
    // 平台特定設定
    void* platform_data;            /**< 平台特定資料 */
};

/**
 * @brief 網路介面操作結構
 */
struct rtk_network_interface {
    const char* name;                /**< 介面名稱 */
    const char* version;             /**< 介面版本 */
    rtk_network_platform_t platform; /**< 平台類型 */
    
    // 生命週期管理
    int (*init)(const rtk_network_config_t* config);
    void (*cleanup)(void);
    
    // 連線管理
    int (*tcp_connect)(const char* host, int port);
    int (*tcp_disconnect)(void);
    int (*tcp_is_connected)(void);
    
    // 資料傳輸
    int (*tcp_write)(const void* data, size_t len);
    int (*tcp_read)(void* buffer, size_t len, int timeout_ms);
    int (*tcp_available)(void);      /**< 獲取可讀取的資料量 */
    
    // Socket 控制
    int (*set_blocking)(int blocking);
    int (*set_timeout)(int timeout_ms);
    int (*get_socket_fd)(void);      /**< 獲取 socket 檔案描述符 (如果適用) */
    
    // 狀態查詢
    const char* (*get_last_error)(void);
    int (*get_error_code)(void);
    
    // 進階功能 (可選)
    int (*ssl_connect)(const char* host, int port, const rtk_network_config_t* ssl_config);
    int (*udp_bind)(int port);
    int (*udp_send_to)(const void* data, size_t len, const char* host, int port);
    int (*udp_recv_from)(void* buffer, size_t len, char* host, int* port);
    
    // 私有資料
    void* private_data;              /**< 實作私有資料 */
};

// === 公開 API ===

/**
 * @brief 初始化網路介面系統
 * @param platform 平台類型
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_init(rtk_network_platform_t platform);

/**
 * @brief 清理網路介面系統
 */
void rtk_network_cleanup(void);

/**
 * @brief 設定網路介面
 * @param interface 網路介面實作
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_set_interface(const rtk_network_interface_t* interface);

/**
 * @brief 獲取當前網路介面
 * @return 網路介面指標，如果未設定則返回 NULL
 */
const rtk_network_interface_t* rtk_network_get_interface(void);

/**
 * @brief 註冊自定義網路介面
 * @param name 介面名稱
 * @param interface 網路介面實作
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_register_interface(const char* name, const rtk_network_interface_t* interface);

/**
 * @brief 按名稱查找網路介面
 * @param name 介面名稱
 * @return 網路介面指標，如果未找到則返回 NULL
 */
const rtk_network_interface_t* rtk_network_find_interface(const char* name);

// === 平台特定初始化函式 ===

/**
 * @brief 初始化 POSIX 網路介面 (Linux/macOS/Unix)
 * @param interface 要填充的介面結構
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_init_posix(rtk_network_interface_t* interface);

/**
 * @brief 初始化 Windows 網路介面
 * @param interface 要填充的介面結構
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_init_windows(rtk_network_interface_t* interface);

/**
 * @brief 初始化 FreeRTOS 網路介面
 * @param interface 要填充的介面結構
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_init_freertos(rtk_network_interface_t* interface);

// === 統一網路操作 API ===

/**
 * @brief 配置網路介面
 * @param config 配置參數
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_configure(const rtk_network_config_t* config);

/**
 * @brief 連接到遠端主機
 * @param host 主機名稱或 IP 位址
 * @param port 埠號
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_connect(const char* host, int port);

/**
 * @brief 斷開連線
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_disconnect(void);

/**
 * @brief 檢查是否已連線
 * @return 1 已連線，0 未連線
 */
int rtk_network_is_connected(void);

/**
 * @brief 發送資料
 * @param data 要發送的資料
 * @param len 資料長度
 * @return 實際發送的位元組數，負值表示錯誤
 */
int rtk_network_write(const void* data, size_t len);

/**
 * @brief 接收資料
 * @param buffer 接收緩衝區
 * @param len 緩衝區大小
 * @param timeout_ms 超時時間 (毫秒)
 * @return 實際接收的位元組數，0 表示連線關閉，負值表示錯誤
 */
int rtk_network_read(void* buffer, size_t len, int timeout_ms);

/**
 * @brief 獲取可讀取的資料量
 * @return 可讀取的位元組數，負值表示錯誤
 */
int rtk_network_available(void);

/**
 * @brief 設定阻塞模式
 * @param blocking 1 為阻塞模式，0 為非阻塞模式
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_set_blocking(int blocking);

/**
 * @brief 設定超時時間
 * @param timeout_ms 超時時間 (毫秒)
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_set_timeout(int timeout_ms);

// === 輔助函式 ===

/**
 * @brief 獲取錯誤碼的描述字串
 * @param error_code 錯誤碼
 * @return 錯誤描述字串
 */
const char* rtk_network_get_error_string(rtk_network_error_t error_code);

/**
 * @brief 獲取最後的錯誤描述
 * @return 錯誤描述字串
 */
const char* rtk_network_get_last_error(void);

/**
 * @brief 建立預設網路配置
 * @param config 配置結構指標
 * @param type 連線類型
 */
void rtk_network_create_default_config(rtk_network_config_t* config, rtk_network_type_t type);

/**
 * @brief 驗證網路配置
 * @param config 配置參數
 * @return RTK_NETWORK_SUCCESS 有效，其他值表示無效
 */
int rtk_network_validate_config(const rtk_network_config_t* config);

/**
 * @brief 解析主機名稱為 IP 位址
 * @param hostname 主機名稱
 * @param ip_buffer IP 位址緩衝區
 * @param buffer_size 緩衝區大小
 * @return RTK_NETWORK_SUCCESS 成功，其他值表示失敗
 */
int rtk_network_resolve_hostname(const char* hostname, char* ip_buffer, size_t buffer_size);

#ifdef __cplusplus
}
#endif

#endif // RTK_NETWORK_INTERFACE_H
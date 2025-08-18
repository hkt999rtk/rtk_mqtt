/**
 * @file rtk_pubsub_cpp_wrapper.cpp
 * @brief C++ wrapper for PubSubClient to provide C interface
 * 
 * This file bridges the C++-based PubSubClient library with the
 * C-based RTK MQTT Framework, providing a clean C interface.
 */

// Include PubSubClient first to get Arduino types
#include <PubSubClient.h>

#include "rtk_pubsub_cpp_wrapper.h"
#include <string.h>
#include <stdio.h>
#include <stdexcept>

// Define error codes used internally
#define RTK_PUBSUB_SUCCESS                  0
#define RTK_PUBSUB_ERROR_INVALID_PARAM     -1
#define RTK_PUBSUB_ERROR_NOT_INITIALIZED   -2
#define RTK_PUBSUB_ERROR_NOT_CONNECTED     -3
#define RTK_PUBSUB_ERROR_CONNECTION_FAILED -4
#define RTK_PUBSUB_ERROR_CONNECTION_LOST   -5
#define RTK_PUBSUB_ERROR_TIMEOUT           -6
#define RTK_PUBSUB_ERROR_MEMORY            -7
#define RTK_PUBSUB_ERROR_PROTOCOL          -8
#define RTK_PUBSUB_ERROR_AUTH              -9
#define RTK_PUBSUB_ERROR_PUBLISH_FAILED    -10
#define RTK_PUBSUB_ERROR_SUBSCRIBE_FAILED  -11
#define RTK_PUBSUB_ERROR_UNSUBSCRIBE_FAILED -12
#define RTK_PUBSUB_ERROR_LOOP_FAILED       -13
#define RTK_PUBSUB_ERROR_UNKNOWN           -99

// Our POSIX network client that implements Arduino's Client interface
class PosixNetworkClient : public Client {
private:
    int _socket;
    bool _connected;
    unsigned long _timeout;
    
public:
    PosixNetworkClient();
    virtual ~PosixNetworkClient();
    
    virtual int connect(IPAddress ip, uint16_t port);
    virtual int connect(const char *host, uint16_t port);
    virtual size_t write(uint8_t);
    virtual size_t write(const uint8_t *buf, size_t size);
    virtual int available();
    virtual int read();
    virtual int read(uint8_t *buf, size_t size);
    virtual int peek();
    virtual void flush();
    virtual void stop();
    virtual uint8_t connected();
    virtual operator bool();
    void setTimeout(unsigned long timeout);
};

#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <sys/select.h>

// PosixNetworkClient implementation
PosixNetworkClient::PosixNetworkClient() 
    : _socket(-1), _connected(false), _timeout(5000) {
}

PosixNetworkClient::~PosixNetworkClient() {
    stop();
}

int PosixNetworkClient::connect(IPAddress ip, uint16_t port) {
    char ip_str[16];
    snprintf(ip_str, sizeof(ip_str), "%u.%u.%u.%u", ip[0], ip[1], ip[2], ip[3]);
    return connect(ip_str, port);
}

int PosixNetworkClient::connect(const char* host, uint16_t port) {
    if (_connected) {
        stop();
    }
    
    _socket = socket(AF_INET, SOCK_STREAM, 0);
    if (_socket < 0) {
        return 0;
    }
    
    struct hostent* server = gethostbyname(host);
    if (server == NULL) {
        close(_socket);
        _socket = -1;
        return 0;
    }
    
    struct sockaddr_in serv_addr;
    memset(&serv_addr, 0, sizeof(serv_addr));
    serv_addr.sin_family = AF_INET;
    serv_addr.sin_port = htons(port);
    memcpy(&serv_addr.sin_addr.s_addr, server->h_addr, server->h_length);
    
    int flags = fcntl(_socket, F_GETFL, 0);
    if (flags != -1) {
        fcntl(_socket, F_SETFL, flags | O_NONBLOCK);
    }
    
    int result = ::connect(_socket, (struct sockaddr*)&serv_addr, sizeof(serv_addr));
    if (result == 0) {
        _connected = true;
        if (flags != -1) {
            fcntl(_socket, F_SETFL, flags);
        }
        return 1;
    }
    
    if (errno != EINPROGRESS) {
        close(_socket);
        _socket = -1;
        return 0;
    }
    
    fd_set write_fds;
    FD_ZERO(&write_fds);
    FD_SET(_socket, &write_fds);
    
    struct timeval tv;
    tv.tv_sec = _timeout / 1000;
    tv.tv_usec = (_timeout % 1000) * 1000;
    
    int select_result = select(_socket + 1, NULL, &write_fds, NULL, &tv);
    if (select_result > 0 && FD_ISSET(_socket, &write_fds)) {
        int error = 0;
        socklen_t len = sizeof(error);
        if (getsockopt(_socket, SOL_SOCKET, SO_ERROR, &error, &len) == 0 && error == 0) {
            _connected = true;
            if (flags != -1) {
                fcntl(_socket, F_SETFL, flags);
            }
            return 1;
        }
    }
    
    close(_socket);
    _socket = -1;
    return 0;
}

size_t PosixNetworkClient::write(uint8_t b) {
    return write(&b, 1);
}

size_t PosixNetworkClient::write(const uint8_t *buf, size_t size) {
    if (!_connected || _socket < 0) {
        return 0;
    }
    
    ssize_t sent = send(_socket, buf, size, MSG_NOSIGNAL);
    if (sent < 0) {
        if (errno == EPIPE || errno == ECONNRESET) {
            _connected = false;
        }
        return 0;
    }
    
    return (size_t)sent;
}

int PosixNetworkClient::available() {
    if (!_connected || _socket < 0) {
        return 0;
    }
    
    fd_set read_fds;
    FD_ZERO(&read_fds);
    FD_SET(_socket, &read_fds);
    
    struct timeval tv;
    tv.tv_sec = 0;
    tv.tv_usec = 0;
    
    int result = select(_socket + 1, &read_fds, NULL, NULL, &tv);
    if (result > 0 && FD_ISSET(_socket, &read_fds)) {
        char test_byte;
        ssize_t peek_result = recv(_socket, &test_byte, 1, MSG_PEEK | MSG_DONTWAIT);
        if (peek_result > 0) {
            return 1;
        } else if (peek_result == 0) {
            _connected = false;
            return 0;
        }
    }
    
    return 0;
}

int PosixNetworkClient::read() {
    if (!_connected || _socket < 0) {
        return -1;
    }
    
    uint8_t b;
    ssize_t received = recv(_socket, &b, 1, 0);
    if (received <= 0) {
        if (received == 0 || errno == ECONNRESET) {
            _connected = false;
        }
        return -1;
    }
    
    return (int)b;
}

int PosixNetworkClient::read(uint8_t *buf, size_t size) {
    if (!_connected || _socket < 0) {
        return 0;
    }
    
    ssize_t received = recv(_socket, buf, size, 0);
    if (received <= 0) {
        if (received == 0 || errno == ECONNRESET) {
            _connected = false;
        }
        return 0;
    }
    
    return (int)received;
}

int PosixNetworkClient::peek() {
    if (!_connected || _socket < 0) {
        return -1;
    }
    
    uint8_t b;
    ssize_t received = recv(_socket, &b, 1, MSG_PEEK);
    if (received <= 0) {
        if (received == 0 || errno == ECONNRESET) {
            _connected = false;
        }
        return -1;
    }
    
    return (int)b;
}

void PosixNetworkClient::flush() {
    // TCP sockets flush automatically
}

void PosixNetworkClient::stop() {
    if (_socket >= 0) {
        close(_socket);
        _socket = -1;
    }
    _connected = false;
}

uint8_t PosixNetworkClient::connected() {
    if (!_connected || _socket < 0) {
        return 0;
    }
    
    int error = 0;
    socklen_t len = sizeof(error);
    int result = getsockopt(_socket, SOL_SOCKET, SO_ERROR, &error, &len);
    
    if (result != 0 || error != 0) {
        _connected = false;
        return 0;
    }
    
    return 1;
}

PosixNetworkClient::operator bool() {
    return connected();
}

void PosixNetworkClient::setTimeout(unsigned long timeout) {
    _timeout = timeout;
}

// Global state structure
struct PubSubWrapper {
    PosixNetworkClient* network_client;
    PubSubClient* mqtt_client;
    bool initialized;
    bool connected;
    char last_error[256];
    
    // Configuration
    char broker_host[256];
    uint16_t broker_port;
    char client_id[128];
    char username[128];
    char password[128];
    
    // Callback handling
    rtk_mqtt_message_callback_t message_callback;
    void* callback_user_data;
};

static PubSubWrapper g_wrapper = {
    .network_client = nullptr,
    .mqtt_client = nullptr,
    .initialized = false,
    .connected = false,
    .last_error = {0},
    .broker_host = {0},
    .broker_port = 1883,
    .client_id = {0},
    .username = {0},
    .password = {0},
    .message_callback = nullptr,
    .callback_user_data = nullptr
};

// Internal callback function for PubSubClient
static void internal_mqtt_callback(char* topic, uint8_t* payload, unsigned int length) {
    if (g_wrapper.message_callback) {
        // Convert to our message structure
        rtk_cpp_mqtt_message_t message;
        strncpy(message.topic, topic, sizeof(message.topic) - 1);
        message.topic[sizeof(message.topic) - 1] = '\0';
        
        message.payload = payload;
        message.payload_len = length;
        message.qos = 0;  // PubSubClient doesn't provide QoS info in callback
        message.retained = false;  // PubSubClient doesn't provide retained flag
        
        g_wrapper.message_callback(&message, g_wrapper.callback_user_data);
    }
}

static void set_last_error(const char* error) {
    if (error) {
        strncpy(g_wrapper.last_error, error, sizeof(g_wrapper.last_error) - 1);
        g_wrapper.last_error[sizeof(g_wrapper.last_error) - 1] = '\0';
    }
}

extern "C" {

int rtk_pubsub_cpp_init(const char* broker_host, uint16_t broker_port, const char* client_id) {
    if (g_wrapper.initialized) {
        return RTK_PUBSUB_SUCCESS;
    }
    
    // Validate parameters
    if (!broker_host || !client_id) {
        set_last_error("Invalid parameters: broker_host and client_id cannot be NULL");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    // Store configuration
    strncpy(g_wrapper.broker_host, broker_host, sizeof(g_wrapper.broker_host) - 1);
    g_wrapper.broker_host[sizeof(g_wrapper.broker_host) - 1] = '\0';
    g_wrapper.broker_port = broker_port;
    strncpy(g_wrapper.client_id, client_id, sizeof(g_wrapper.client_id) - 1);
    g_wrapper.client_id[sizeof(g_wrapper.client_id) - 1] = '\0';
    
    try {
        // Create network client
        g_wrapper.network_client = new PosixNetworkClient();
        if (!g_wrapper.network_client) {
            set_last_error("Failed to create network client");
            return RTK_PUBSUB_ERROR_MEMORY;
        }
        
        // Create MQTT client
        g_wrapper.mqtt_client = new PubSubClient(*g_wrapper.network_client);
        if (!g_wrapper.mqtt_client) {
            delete g_wrapper.network_client;
            g_wrapper.network_client = nullptr;
            set_last_error("Failed to create MQTT client");
            return RTK_PUBSUB_ERROR_MEMORY;
        }
        
        // Configure MQTT client
        g_wrapper.mqtt_client->setServer(g_wrapper.broker_host, g_wrapper.broker_port);
        g_wrapper.mqtt_client->setCallback(internal_mqtt_callback);
        
        g_wrapper.initialized = true;
        printf("[PubSubCpp] ✓ 初始化完成 - Broker: %s:%d, 客戶端: %s\n", 
               broker_host, broker_port, client_id);
        
        return RTK_PUBSUB_SUCCESS;
        
    } catch (const std::exception& e) {
        set_last_error("C++ exception during initialization");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    } catch (...) {
        set_last_error("Unknown exception during initialization");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    }
}

void rtk_pubsub_cpp_cleanup(void) {
    if (!g_wrapper.initialized) {
        return;
    }
    
    // Disconnect if connected
    if (g_wrapper.connected) {
        rtk_pubsub_cpp_disconnect();
    }
    
    // Clean up C++ objects
    if (g_wrapper.mqtt_client) {
        delete g_wrapper.mqtt_client;
        g_wrapper.mqtt_client = nullptr;
    }
    
    if (g_wrapper.network_client) {
        delete g_wrapper.network_client;
        g_wrapper.network_client = nullptr;
    }
    
    // Reset state
    memset(&g_wrapper, 0, sizeof(g_wrapper));
    
    printf("[PubSubCpp] ✓ 清理完成\n");
}

int rtk_pubsub_cpp_connect(void) {
    if (!g_wrapper.initialized) {
        set_last_error("Not initialized");
        return RTK_PUBSUB_ERROR_NOT_INITIALIZED;
    }
    
    if (g_wrapper.connected) {
        return RTK_PUBSUB_SUCCESS;
    }
    
    printf("[PubSubCpp] 正在連接到 %s:%d...\n", 
           g_wrapper.broker_host, g_wrapper.broker_port);
    
    try {
        bool success = false;
        
        // Attempt connection with or without credentials
        if (strlen(g_wrapper.username) > 0) {
            success = g_wrapper.mqtt_client->connect(g_wrapper.client_id, 
                                                   g_wrapper.username, 
                                                   g_wrapper.password);
        } else {
            success = g_wrapper.mqtt_client->connect(g_wrapper.client_id);
        }
        
        if (success) {
            g_wrapper.connected = true;
            printf("[PubSubCpp] ✓ 連接成功\n");
            return RTK_PUBSUB_SUCCESS;
        } else {
            int state = g_wrapper.mqtt_client->state();
            switch (state) {
                case MQTT_CONNECTION_TIMEOUT:
                    set_last_error("Connection timeout");
                    return RTK_PUBSUB_ERROR_TIMEOUT;
                case MQTT_CONNECTION_LOST:
                    set_last_error("Connection lost");
                    return RTK_PUBSUB_ERROR_CONNECTION_LOST;
                case MQTT_CONNECT_FAILED:
                    set_last_error("Connection failed");
                    return RTK_PUBSUB_ERROR_CONNECTION_FAILED;
                case MQTT_CONNECT_BAD_PROTOCOL:
                    set_last_error("Bad protocol version");
                    return RTK_PUBSUB_ERROR_PROTOCOL;
                case MQTT_CONNECT_BAD_CLIENT_ID:
                    set_last_error("Bad client ID");
                    return RTK_PUBSUB_ERROR_INVALID_PARAM;
                case MQTT_CONNECT_UNAVAILABLE:
                    set_last_error("Server unavailable");
                    return RTK_PUBSUB_ERROR_CONNECTION_FAILED;
                case MQTT_CONNECT_BAD_CREDENTIALS:
                    set_last_error("Bad credentials");
                    return RTK_PUBSUB_ERROR_AUTH;
                case MQTT_CONNECT_UNAUTHORIZED:
                    set_last_error("Unauthorized");
                    return RTK_PUBSUB_ERROR_AUTH;
                default:
                    set_last_error("Unknown connection error");
                    return RTK_PUBSUB_ERROR_UNKNOWN;
            }
        }
        
    } catch (const std::exception& e) {
        set_last_error("C++ exception during connection");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    } catch (...) {
        set_last_error("Unknown exception during connection");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    }
}

int rtk_pubsub_cpp_disconnect(void) {
    if (!g_wrapper.initialized) {
        return RTK_PUBSUB_ERROR_NOT_INITIALIZED;
    }
    
    if (!g_wrapper.connected) {
        return RTK_PUBSUB_SUCCESS;
    }
    
    printf("[PubSubCpp] 正在斷開連接...\n");
    
    try {
        g_wrapper.mqtt_client->disconnect();
        g_wrapper.connected = false;
        printf("[PubSubCpp] ✓ 斷開連接完成\n");
        return RTK_PUBSUB_SUCCESS;
        
    } catch (const std::exception& e) {
        set_last_error("C++ exception during disconnection");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    } catch (...) {
        set_last_error("Unknown exception during disconnection");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    }
}

int rtk_pubsub_cpp_is_connected(void) {
    if (!g_wrapper.initialized || !g_wrapper.mqtt_client) {
        return 0;
    }
    
    try {
        bool connected = g_wrapper.mqtt_client->connected();
        g_wrapper.connected = connected;  // Update our state
        return connected ? 1 : 0;
        
    } catch (...) {
        g_wrapper.connected = false;
        return 0;
    }
}

int rtk_pubsub_cpp_publish(const char* topic, const void* payload, size_t payload_len, 
                          int qos, int retained) {
    if (!g_wrapper.initialized || !g_wrapper.mqtt_client) {
        set_last_error("Not initialized");
        return RTK_PUBSUB_ERROR_NOT_INITIALIZED;
    }
    
    if (!g_wrapper.connected) {
        set_last_error("Not connected");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    if (!topic || !payload) {
        set_last_error("Invalid parameters");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    printf("[PubSubCpp] 發布訊息到 '%s' (長度: %zu, QoS: %d, Retained: %s)\n", 
           topic, payload_len, qos, retained ? "是" : "否");
    
    try {
        bool success = g_wrapper.mqtt_client->publish(topic, 
                                                    (const uint8_t*)payload, 
                                                    (unsigned int)payload_len, 
                                                    retained ? true : false);
        
        if (success) {
            printf("[PubSubCpp] ✓ 訊息發布成功\n");
            return RTK_PUBSUB_SUCCESS;
        } else {
            set_last_error("Failed to publish message");
            printf("[PubSubCpp] ❌ 訊息發布失敗\n");
            return RTK_PUBSUB_ERROR_PUBLISH_FAILED;
        }
        
    } catch (const std::exception& e) {
        set_last_error("C++ exception during publish");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    } catch (...) {
        set_last_error("Unknown exception during publish");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    }
}

int rtk_pubsub_cpp_subscribe(const char* topic, int qos) {
    if (!g_wrapper.initialized || !g_wrapper.mqtt_client) {
        set_last_error("Not initialized");
        return RTK_PUBSUB_ERROR_NOT_INITIALIZED;
    }
    
    if (!g_wrapper.connected) {
        set_last_error("Not connected");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    if (!topic) {
        set_last_error("Invalid topic");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    printf("[PubSubCpp] 訂閱主題 '%s' (QoS: %d)\n", topic, qos);
    
    try {
        bool success = g_wrapper.mqtt_client->subscribe(topic, (uint8_t)qos);
        
        if (success) {
            printf("[PubSubCpp] ✓ 訂閱成功\n");
            return RTK_PUBSUB_SUCCESS;
        } else {
            set_last_error("Failed to subscribe to topic");
            printf("[PubSubCpp] ❌ 訂閱失敗\n");
            return RTK_PUBSUB_ERROR_SUBSCRIBE_FAILED;
        }
        
    } catch (const std::exception& e) {
        set_last_error("C++ exception during subscribe");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    } catch (...) {
        set_last_error("Unknown exception during subscribe");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    }
}

int rtk_pubsub_cpp_unsubscribe(const char* topic) {
    if (!g_wrapper.initialized || !g_wrapper.mqtt_client) {
        set_last_error("Not initialized");
        return RTK_PUBSUB_ERROR_NOT_INITIALIZED;
    }
    
    if (!g_wrapper.connected) {
        set_last_error("Not connected");
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    if (!topic) {
        set_last_error("Invalid topic");
        return RTK_PUBSUB_ERROR_INVALID_PARAM;
    }
    
    printf("[PubSubCpp] 取消訂閱主題 '%s'\n", topic);
    
    try {
        bool success = g_wrapper.mqtt_client->unsubscribe(topic);
        
        if (success) {
            printf("[PubSubCpp] ✓ 取消訂閱成功\n");
            return RTK_PUBSUB_SUCCESS;
        } else {
            set_last_error("Failed to unsubscribe from topic");
            printf("[PubSubCpp] ❌ 取消訂閱失敗\n");
            return RTK_PUBSUB_ERROR_UNSUBSCRIBE_FAILED;
        }
        
    } catch (const std::exception& e) {
        set_last_error("C++ exception during unsubscribe");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    } catch (...) {
        set_last_error("Unknown exception during unsubscribe");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    }
}

int rtk_pubsub_cpp_loop(void) {
    if (!g_wrapper.initialized || !g_wrapper.mqtt_client) {
        return RTK_PUBSUB_ERROR_NOT_INITIALIZED;
    }
    
    if (!g_wrapper.connected) {
        return RTK_PUBSUB_ERROR_NOT_CONNECTED;
    }
    
    try {
        bool success = g_wrapper.mqtt_client->loop();
        return success ? RTK_PUBSUB_SUCCESS : RTK_PUBSUB_ERROR_LOOP_FAILED;
        
    } catch (const std::exception& e) {
        set_last_error("C++ exception during loop");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    } catch (...) {
        set_last_error("Unknown exception during loop");
        return RTK_PUBSUB_ERROR_UNKNOWN;
    }
}

int rtk_pubsub_cpp_set_credentials(const char* username, const char* password) {
    if (username) {
        strncpy(g_wrapper.username, username, sizeof(g_wrapper.username) - 1);
        g_wrapper.username[sizeof(g_wrapper.username) - 1] = '\0';
    } else {
        g_wrapper.username[0] = '\0';
    }
    
    if (password) {
        strncpy(g_wrapper.password, password, sizeof(g_wrapper.password) - 1);
        g_wrapper.password[sizeof(g_wrapper.password) - 1] = '\0';
    } else {
        g_wrapper.password[0] = '\0';
    }
    
    printf("[PubSubCpp] 設定認證資訊 (用戶名: %s)\n", 
           strlen(g_wrapper.username) > 0 ? g_wrapper.username : "無");
    
    return RTK_PUBSUB_SUCCESS;
}

int rtk_pubsub_cpp_set_callback(rtk_mqtt_message_callback_t callback, void* user_data) {
    g_wrapper.message_callback = callback;
    g_wrapper.callback_user_data = user_data;
    
    printf("[PubSubCpp] 設定訊息回調函式\n");
    return RTK_PUBSUB_SUCCESS;
}

const char* rtk_pubsub_cpp_get_last_error(void) {
    return g_wrapper.last_error;
}

const char* rtk_pubsub_cpp_get_version(void) {
    return "RTK PubSubClient C++ Wrapper v1.0.0";
}

} // extern "C"
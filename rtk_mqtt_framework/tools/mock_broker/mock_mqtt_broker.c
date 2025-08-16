#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <signal.h>
#include <time.h>
#include <pthread.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

/**
 * @file mock_mqtt_broker.c
 * @brief 簡化的 MQTT Broker 模擬器
 * 
 * 提供基本的 MQTT 功能用於測試：
 * - 接受客戶端連線
 * - 處理 SUBSCRIBE/PUBLISH 訊息
 * - 模擬訊息轉發
 * - 記錄 RTK MQTT 診斷訊息
 */

// === MQTT 協議定義 ===

#define MQTT_CONNECT     1
#define MQTT_CONNACK     2
#define MQTT_PUBLISH     3
#define MQTT_PUBACK      4
#define MQTT_SUBSCRIBE   8
#define MQTT_SUBACK      9
#define MQTT_UNSUBSCRIBE 10
#define MQTT_UNSUBACK    11
#define MQTT_PINGREQ     12
#define MQTT_PINGRESP    13
#define MQTT_DISCONNECT  14

#define MAX_CLIENTS      32
#define MAX_TOPICS       128
#define BUFFER_SIZE      4096

// === 資料結構 ===

typedef struct mqtt_client {
    int socket_fd;
    char client_id[64];
    char remote_addr[32];
    int is_connected;
    time_t connect_time;
    int message_count;
    pthread_t thread_id;
} mqtt_client_t;

typedef struct mqtt_subscription {
    char topic[256];
    int client_fd;
    int qos;
} mqtt_subscription_t;

typedef struct mqtt_message {
    char topic[256];
    char payload[1024];
    int payload_len;
    int qos;
    time_t timestamp;
    char client_id[64];
} mqtt_message_t;

// === 全域狀態 ===

static mqtt_client_t g_clients[MAX_CLIENTS];
static mqtt_subscription_t g_subscriptions[MAX_TOPICS];
static int g_client_count = 0;
static int g_subscription_count = 0;
static int g_running = 1;
static int g_server_socket = -1;
static pthread_mutex_t g_mutex = PTHREAD_MUTEX_INITIALIZER;

// 統計資料
static struct {
    int total_connections;
    int total_messages;
    int total_publishes;
    int total_subscribes;
    time_t start_time;
} g_stats = {0};

// === 輔助函式 ===

static void signal_handler(int sig) {
    printf("\n[Mock-Broker] 收到信號 %d，正在關閉...\n", sig);
    g_running = 0;
    if (g_server_socket >= 0) {
        close(g_server_socket);
    }
}

static void log_rtk_message(const char* topic, const char* payload) {
    // 解析 RTK MQTT 主題格式: rtk/v1/{tenant}/{site}/{device_id}/{message_type}
    if (strncmp(topic, "rtk/v1/", 7) == 0) {
        char* topic_copy = strdup(topic);
        char* parts[6];
        int part_count = 0;
        
        char* token = strtok(topic_copy, "/");
        while (token && part_count < 6) {
            parts[part_count++] = token;
            token = strtok(NULL, "/");
        }
        
        if (part_count >= 5) {
            const char* tenant = parts[2];
            const char* site = parts[3];
            const char* device_id = parts[4];
            const char* message_type = parts[5];
            
            time_t now = time(NULL);
            struct tm* tm_info = localtime(&now);
            char time_str[32];
            strftime(time_str, sizeof(time_str), "%H:%M:%S", tm_info);
            
            printf("[%s] RTK 訊息 - %s/%s/%s (%s)\n", 
                   time_str, tenant, site, device_id, message_type);
            
            // 解析 JSON 載荷中的關鍵資訊
            if (strstr(payload, "\"schema\"")) {
                const char* schema_start = strstr(payload, "\"schema\":\"");
                if (schema_start) {
                    schema_start += 10;
                    const char* schema_end = strchr(schema_start, '"');
                    if (schema_end) {
                        int schema_len = schema_end - schema_start;
                        printf("          Schema: %.*s\n", schema_len, schema_start);
                    }
                }
            }
            
            if (strstr(payload, "\"health\"")) {
                const char* health_start = strstr(payload, "\"health\":\"");
                if (health_start) {
                    health_start += 10;
                    const char* health_end = strchr(health_start, '"');
                    if (health_end) {
                        int health_len = health_end - health_start;
                        printf("          Health: %.*s\n", health_len, health_start);
                    }
                }
            }
            
            if (strstr(payload, "\"severity\"")) {
                const char* severity_start = strstr(payload, "\"severity\":\"");
                if (severity_start) {
                    severity_start += 12;
                    const char* severity_end = strchr(severity_start, '"');
                    if (severity_end) {
                        int severity_len = severity_end - severity_start;
                        printf("          Severity: %.*s\n", severity_len, severity_start);
                    }
                }
            }
        }
        
        free(topic_copy);
    }
}

static int read_mqtt_length(int socket_fd) {
    unsigned char buffer[4];
    int bytes_read = 0;
    int length = 0;
    int multiplier = 1;
    
    do {
        if (recv(socket_fd, &buffer[bytes_read], 1, 0) <= 0) {
            return -1;
        }
        length += (buffer[bytes_read] & 0x7F) * multiplier;
        multiplier *= 128;
        bytes_read++;
    } while ((buffer[bytes_read - 1] & 0x80) && bytes_read < 4);
    
    return length;
}

static int write_mqtt_length(unsigned char* buffer, int length) {
    int bytes = 0;
    do {
        unsigned char byte = length % 128;
        length /= 128;
        if (length > 0) {
            byte |= 0x80;
        }
        buffer[bytes++] = byte;
    } while (length > 0);
    return bytes;
}

static void send_connack(int client_fd, int return_code) {
    unsigned char response[4] = {
        MQTT_CONNACK << 4,  // Fixed header
        2,                  // Remaining length
        0,                  // Connect acknowledge flags
        return_code         // Connect return code
    };
    
    send(client_fd, response, 4, 0);
    printf("[Mock-Broker] 發送 CONNACK (return_code: %d)\n", return_code);
}

static void send_suback(int client_fd, int packet_id, int qos) {
    unsigned char response[5] = {
        MQTT_SUBACK << 4,   // Fixed header
        3,                  // Remaining length
        (packet_id >> 8) & 0xFF,  // Packet ID high byte
        packet_id & 0xFF,   // Packet ID low byte
        qos                 // QoS level
    };
    
    send(client_fd, response, 5, 0);
    printf("[Mock-Broker] 發送 SUBACK (packet_id: %d, qos: %d)\n", packet_id, qos);
}

static void send_pingresp(int client_fd) {
    unsigned char response[2] = {
        MQTT_PINGRESP << 4, // Fixed header
        0                   // Remaining length
    };
    
    send(client_fd, response, 2, 0);
    printf("[Mock-Broker] 發送 PINGRESP\n");
}

static void forward_message(const char* topic, const char* payload, int payload_len, int sender_fd) {
    pthread_mutex_lock(&g_mutex);
    
    // 尋找匹配的訂閱
    for (int i = 0; i < g_subscription_count; i++) {
        mqtt_subscription_t* sub = &g_subscriptions[i];
        
        if (sub->client_fd == sender_fd) continue;  // 不轉發給發送者
        
        // 簡化的主題匹配 (支援 + 和 # 萬用字元)
        int match = 0;
        if (strcmp(sub->topic, topic) == 0) {
            match = 1;
        } else if (strchr(sub->topic, '+') || strchr(sub->topic, '#')) {
            // 簡化的萬用字元匹配
            if (strstr(sub->topic, "#")) {
                char* hash_pos = strstr(sub->topic, "#");
                int prefix_len = hash_pos - sub->topic;
                if (strncmp(sub->topic, topic, prefix_len) == 0) {
                    match = 1;
                }
            }
        }
        
        if (match) {
            // 建構 PUBLISH 訊息
            unsigned char header[256];
            int header_len = 0;
            
            header[header_len++] = MQTT_PUBLISH << 4;  // Fixed header
            
            int topic_len = strlen(topic);
            int remaining_len = 2 + topic_len + payload_len;
            header_len += write_mqtt_length(&header[header_len], remaining_len);
            
            // Topic length
            header[header_len++] = (topic_len >> 8) & 0xFF;
            header[header_len++] = topic_len & 0xFF;
            
            // Topic
            memcpy(&header[header_len], topic, topic_len);
            header_len += topic_len;
            
            // 發送標頭
            send(sub->client_fd, header, header_len, 0);
            
            // 發送載荷
            if (payload_len > 0) {
                send(sub->client_fd, payload, payload_len, 0);
            }
            
            printf("[Mock-Broker] 轉發訊息到客戶端 (fd: %d, topic: %s)\n", 
                   sub->client_fd, topic);
        }
    }
    
    pthread_mutex_unlock(&g_mutex);
}

static void handle_connect(int client_fd, unsigned char* buffer, int length) {
    // 簡化的 CONNECT 處理
    int pos = 0;
    
    // 跳過協議名稱和版本
    int protocol_name_len = (buffer[pos] << 8) | buffer[pos + 1];
    pos += 2 + protocol_name_len + 1 + 2;  // 名稱 + 版本 + 標誌 + Keep Alive
    
    // 讀取客戶端 ID
    int client_id_len = (buffer[pos] << 8) | buffer[pos + 1];
    pos += 2;
    
    char client_id[64] = {0};
    if (client_id_len > 0 && client_id_len < sizeof(client_id)) {
        memcpy(client_id, &buffer[pos], client_id_len);
    } else {
        snprintf(client_id, sizeof(client_id), "client_%d", client_fd);
    }
    
    printf("[Mock-Broker] 客戶端連線: %s (fd: %d)\n", client_id, client_fd);
    
    // 儲存客戶端資訊
    pthread_mutex_lock(&g_mutex);
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (g_clients[i].socket_fd == 0) {
            g_clients[i].socket_fd = client_fd;
            strncpy(g_clients[i].client_id, client_id, sizeof(g_clients[i].client_id) - 1);
            g_clients[i].is_connected = 1;
            g_clients[i].connect_time = time(NULL);
            g_clients[i].message_count = 0;
            g_client_count++;
            g_stats.total_connections++;
            break;
        }
    }
    pthread_mutex_unlock(&g_mutex);
    
    // 發送 CONNACK
    send_connack(client_fd, 0);  // 0 = 連線成功
}

static void handle_subscribe(int client_fd, unsigned char* buffer, int length) {
    int pos = 0;
    
    // 讀取封包 ID
    int packet_id = (buffer[pos] << 8) | buffer[pos + 1];
    pos += 2;
    
    while (pos < length) {
        // 讀取主題長度和內容
        int topic_len = (buffer[pos] << 8) | buffer[pos + 1];
        pos += 2;
        
        if (pos + topic_len >= length) break;
        
        char topic[256] = {0};
        memcpy(topic, &buffer[pos], topic_len);
        pos += topic_len;
        
        // 讀取 QoS
        int qos = buffer[pos++];
        
        printf("[Mock-Broker] 訂閱主題: %s (QoS: %d, fd: %d)\n", topic, qos, client_fd);
        
        // 儲存訂閱
        pthread_mutex_lock(&g_mutex);
        if (g_subscription_count < MAX_TOPICS) {
            mqtt_subscription_t* sub = &g_subscriptions[g_subscription_count++];
            strncpy(sub->topic, topic, sizeof(sub->topic) - 1);
            sub->client_fd = client_fd;
            sub->qos = qos;
            g_stats.total_subscribes++;
        }
        pthread_mutex_unlock(&g_mutex);
        
        // 發送 SUBACK
        send_suback(client_fd, packet_id, qos);
    }
}

static void handle_publish(int client_fd, unsigned char* buffer, int length) {
    int pos = 0;
    
    // 讀取主題長度和內容
    int topic_len = (buffer[pos] << 8) | buffer[pos + 1];
    pos += 2;
    
    char topic[256] = {0};
    memcpy(topic, &buffer[pos], topic_len);
    pos += topic_len;
    
    // 讀取載荷
    int payload_len = length - pos;
    char payload[1024] = {0};
    if (payload_len > 0) {
        memcpy(payload, &buffer[pos], payload_len);
    }
    
    printf("[Mock-Broker] 收到 PUBLISH: %s (%d bytes)\n", topic, payload_len);
    
    // 記錄 RTK 訊息
    if (strncmp(topic, "rtk/v1/", 7) == 0) {
        log_rtk_message(topic, payload);
    }
    
    // 更新統計
    pthread_mutex_lock(&g_mutex);
    g_stats.total_publishes++;
    g_stats.total_messages++;
    
    // 更新客戶端統計
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (g_clients[i].socket_fd == client_fd) {
            g_clients[i].message_count++;
            break;
        }
    }
    pthread_mutex_unlock(&g_mutex);
    
    // 轉發訊息給訂閱者
    forward_message(topic, payload, payload_len, client_fd);
}

static void* client_handler(void* arg) {
    int client_fd = *(int*)arg;
    free(arg);
    
    unsigned char buffer[BUFFER_SIZE];
    
    while (g_running) {
        // 讀取固定標頭
        ssize_t bytes_read = recv(client_fd, buffer, 1, 0);
        if (bytes_read <= 0) {
            break;  // 客戶端斷線
        }
        
        unsigned char message_type = (buffer[0] >> 4) & 0x0F;
        
        // 讀取剩餘長度
        int remaining_length = read_mqtt_length(client_fd);
        if (remaining_length < 0) {
            break;
        }
        
        // 讀取訊息內容
        if (remaining_length > 0) {
            bytes_read = recv(client_fd, buffer, remaining_length, 0);
            if (bytes_read != remaining_length) {
                break;
            }
        }
        
        // 處理訊息
        switch (message_type) {
            case MQTT_CONNECT:
                handle_connect(client_fd, buffer, remaining_length);
                break;
                
            case MQTT_SUBSCRIBE:
                handle_subscribe(client_fd, buffer, remaining_length);
                break;
                
            case MQTT_PUBLISH:
                handle_publish(client_fd, buffer, remaining_length);
                break;
                
            case MQTT_PINGREQ:
                send_pingresp(client_fd);
                break;
                
            case MQTT_DISCONNECT:
                printf("[Mock-Broker] 客戶端主動斷線 (fd: %d)\n", client_fd);
                goto cleanup;
                
            default:
                printf("[Mock-Broker] 未知訊息類型: %d\n", message_type);
                break;
        }
    }
    
cleanup:
    // 清理客戶端
    pthread_mutex_lock(&g_mutex);
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (g_clients[i].socket_fd == client_fd) {
            printf("[Mock-Broker] 客戶端斷線: %s\n", g_clients[i].client_id);
            memset(&g_clients[i], 0, sizeof(mqtt_client_t));
            g_client_count--;
            break;
        }
    }
    
    // 清理訂閱
    for (int i = 0; i < g_subscription_count; i++) {
        if (g_subscriptions[i].client_fd == client_fd) {
            memmove(&g_subscriptions[i], &g_subscriptions[i + 1], 
                    (g_subscription_count - i - 1) * sizeof(mqtt_subscription_t));
            g_subscription_count--;
            i--;  // 重新檢查這個位置
        }
    }
    pthread_mutex_unlock(&g_mutex);
    
    close(client_fd);
    return NULL;
}

static void print_status(void) {
    time_t now = time(NULL);
    int uptime = now - g_stats.start_time;
    
    printf("\n=== Mock MQTT Broker 狀態 ===\n");
    printf("運行時間: %d 秒\n", uptime);
    printf("連接客戶端: %d\n", g_client_count);
    printf("總連線次數: %d\n", g_stats.total_connections);
    printf("總訊息數: %d\n", g_stats.total_messages);
    printf("發佈訊息: %d\n", g_stats.total_publishes);
    printf("訂閱數: %d\n", g_stats.total_subscribes);
    
    if (g_client_count > 0) {
        printf("\n活躍客戶端:\n");
        pthread_mutex_lock(&g_mutex);
        for (int i = 0; i < MAX_CLIENTS; i++) {
            if (g_clients[i].socket_fd > 0) {
                int conn_time = now - g_clients[i].connect_time;
                printf("  %s (連線 %d 秒, 訊息: %d)\n", 
                       g_clients[i].client_id, 
                       conn_time, 
                       g_clients[i].message_count);
            }
        }
        pthread_mutex_unlock(&g_mutex);
    }
    
    printf("==============================\n\n");
}

int main(int argc, char* argv[]) {
    printf("Mock MQTT Broker v1.0.0\n");
    printf("=======================\n");
    
    // 解析命令列參數
    int port = 1883;  // 預設 MQTT 埠
    
    if (argc > 1) {
        port = atoi(argv[1]);
        if (port <= 0 || port > 65535) {
            printf("無效的埠號: %s\n", argv[1]);
            return 1;
        }
    }
    
    // 設定信號處理
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    
    // 建立伺服器 socket
    g_server_socket = socket(AF_INET, SOCK_STREAM, 0);
    if (g_server_socket < 0) {
        perror("socket");
        return 1;
    }
    
    // 設定 socket 選項
    int opt = 1;
    if (setsockopt(g_server_socket, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt)) < 0) {
        perror("setsockopt");
        close(g_server_socket);
        return 1;
    }
    
    // 綁定地址
    struct sockaddr_in server_addr = {0};
    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = INADDR_ANY;
    server_addr.sin_port = htons(port);
    
    if (bind(g_server_socket, (struct sockaddr*)&server_addr, sizeof(server_addr)) < 0) {
        perror("bind");
        close(g_server_socket);
        return 1;
    }
    
    // 開始監聽
    if (listen(g_server_socket, 10) < 0) {
        perror("listen");
        close(g_server_socket);
        return 1;
    }
    
    g_stats.start_time = time(NULL);
    
    printf("Mock MQTT Broker 已啟動，監聽埠: %d\n", port);
    printf("支援 RTK MQTT 診斷協議訊息記錄\n");
    printf("按 Ctrl+C 停止伺服器\n\n");
    
    // 建立狀態顯示執行緒
    pthread_t status_thread;
    pthread_create(&status_thread, NULL, (void*)(void*)print_status, NULL);
    
    // 主迴圈 - 接受連線
    while (g_running) {
        struct sockaddr_in client_addr;
        socklen_t client_len = sizeof(client_addr);
        
        int client_fd = accept(g_server_socket, (struct sockaddr*)&client_addr, &client_len);
        if (client_fd < 0) {
            if (g_running) {
                perror("accept");
            }
            continue;
        }
        
        char client_ip[INET_ADDRSTRLEN];
        inet_ntop(AF_INET, &client_addr.sin_addr, client_ip, sizeof(client_ip));
        printf("[Mock-Broker] 新客戶端連接: %s:%d (fd: %d)\n", 
               client_ip, ntohs(client_addr.sin_port), client_fd);
        
        // 建立客戶端處理執行緒
        int* client_fd_ptr = malloc(sizeof(int));
        *client_fd_ptr = client_fd;
        
        pthread_t client_thread;
        if (pthread_create(&client_thread, NULL, client_handler, client_fd_ptr) != 0) {
            perror("pthread_create");
            close(client_fd);
            free(client_fd_ptr);
            continue;
        }
        
        pthread_detach(client_thread);  // 自動清理執行緒
    }
    
    // 清理
    printf("\n[Mock-Broker] 正在關閉伺服器...\n");
    
    // 關閉所有客戶端連線
    pthread_mutex_lock(&g_mutex);
    for (int i = 0; i < MAX_CLIENTS; i++) {
        if (g_clients[i].socket_fd > 0) {
            close(g_clients[i].socket_fd);
        }
    }
    pthread_mutex_unlock(&g_mutex);
    
    close(g_server_socket);
    print_status();
    
    printf("Mock MQTT Broker 已關閉\n");
    return 0;
}
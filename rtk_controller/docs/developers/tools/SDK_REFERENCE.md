# RTK MQTT SDK 參考指南

## 概述

RTK MQTT SDK 提供多種程式語言的實作，幫助開發者快速整合 RTK MQTT 協議。本指南涵蓋 C/C++、Python、Go 和 JavaScript 的 SDK 使用方法，包含完整的 API 文檔和實作範例。

## 🚀 支援的程式語言

| 語言 | 狀態 | 適用場景 | 最低版本 |
|------|------|----------|----------|
| **C/C++** | ✅ 穩定 | 嵌入式設備、高效能應用 | C99, C++11 |
| **Python** | ✅ 穩定 | 快速原型、測試、Web 應用 | Python 3.7+ |
| **Go** | ✅ 穩定 | 微服務、高併發應用 | Go 1.19+ |
| **JavaScript** | ✅ 穩定 | Web 應用、Node.js 服務 | Node.js 16+ |

## 📦 C/C++ SDK

### 安裝和設置

#### 依賴程式庫
```bash
# Ubuntu/Debian
sudo apt-get install libcjson-dev libmosquitto-dev libssl-dev

# CentOS/RHEL  
sudo yum install cjson-devel mosquitto-devel openssl-devel

# macOS
brew install cjson mosquitto openssl
```

#### CMake 配置
```cmake
# CMakeLists.txt
cmake_minimum_required(VERSION 3.12)
project(rtk_mqtt_device)

find_package(PkgConfig REQUIRED)
pkg_check_modules(MOSQUITTO REQUIRED libmosquitto)
pkg_check_modules(CJSON REQUIRED libcjson)

add_executable(rtk_device main.c)
target_link_libraries(rtk_device ${MOSQUITTO_LIBRARIES} ${CJSON_LIBRARIES} ssl crypto)
target_include_directories(rtk_device PRIVATE ${MOSQUITTO_INCLUDE_DIRS} ${CJSON_INCLUDE_DIRS})
```

### 基礎 API

#### 客戶端初始化
```c
#include <mosquitto.h>
#include <cjson/cjson.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

typedef struct {
    struct mosquitto *mosq;
    char *device_id;
    char *tenant;
    char *site;
    char client_id[64];
    bool connected;
} rtk_mqtt_client_t;

int rtk_mqtt_init(rtk_mqtt_client_t *client, const char *device_id, 
                  const char *tenant, const char *site) {
    // 初始化 mosquitto 庫
    mosquitto_lib_init();
    
    // 設置客戶端資訊
    client->device_id = strdup(device_id);
    client->tenant = strdup(tenant);
    client->site = strdup(site);
    snprintf(client->client_id, sizeof(client->client_id), 
             "rtk-%s", device_id);
    
    // 創建 MQTT 客戶端
    client->mosq = mosquitto_new(client->client_id, true, client);
    if (!client->mosq) {
        fprintf(stderr, "Failed to create MQTT client\n");
        return -1;
    }
    
    // 設置回調函數
    mosquitto_connect_callback_set(client->mosq, on_connect);
    mosquitto_disconnect_callback_set(client->mosq, on_disconnect);
    mosquitto_message_callback_set(client->mosq, on_message);
    
    return 0;
}
```

#### 連接管理
```c
void on_connect(struct mosquitto *mosq, void *userdata, int result) {
    rtk_mqtt_client_t *client = (rtk_mqtt_client_t *)userdata;
    
    if (result == 0) {
        client->connected = true;
        printf("Connected to MQTT broker\n");
        
        // 訂閱命令主題
        char cmd_topic[256];
        snprintf(cmd_topic, sizeof(cmd_topic), 
                "rtk/v1/%s/%s/%s/cmd/req", 
                client->tenant, client->site, client->device_id);
        mosquitto_subscribe(mosq, NULL, cmd_topic, 1);
        
        // 發送上線狀態
        rtk_mqtt_publish_online_status(client);
        
        // 發送設備屬性
        rtk_mqtt_publish_attributes(client);
    } else {
        fprintf(stderr, "Connection failed: %s\n", mosquitto_connack_string(result));
    }
}

int rtk_mqtt_connect(rtk_mqtt_client_t *client, const char *host, int port) {
    // 設置 LWT
    char lwt_topic[256];
    snprintf(lwt_topic, sizeof(lwt_topic), 
            "rtk/v1/%s/%s/%s/lwt", 
            client->tenant, client->site, client->device_id);
    
    cJSON *lwt_json = cJSON_CreateObject();
    cJSON_AddStringToObject(lwt_json, "schema", "lwt/1.0");
    cJSON_AddNumberToObject(lwt_json, "ts", time(NULL) * 1000);
    cJSON_AddStringToObject(lwt_json, "device_id", client->device_id);
    cJSON_AddStringToObject(lwt_json, "status", "offline");
    cJSON_AddStringToObject(lwt_json, "reason", "unexpected_disconnect");
    
    char *lwt_payload = cJSON_Print(lwt_json);
    mosquitto_will_set(client->mosq, lwt_topic, strlen(lwt_payload), 
                      lwt_payload, 1, true);
    
    free(lwt_payload);
    cJSON_Delete(lwt_json);
    
    // 建立連接
    return mosquitto_connect(client->mosq, host, port, 60);
}
```

#### 狀態發布
```c
int rtk_mqtt_publish_state(rtk_mqtt_client_t *client, const char *health, 
                          double cpu_usage, double memory_usage) {
    char topic[256];
    snprintf(topic, sizeof(topic), "rtk/v1/%s/%s/%s/state", 
            client->tenant, client->site, client->device_id);
    
    cJSON *state_json = cJSON_CreateObject();
    cJSON_AddStringToObject(state_json, "schema", "state/1.0");
    cJSON_AddNumberToObject(state_json, "ts", time(NULL) * 1000);
    cJSON_AddStringToObject(state_json, "health", health);
    cJSON_AddNumberToObject(state_json, "cpu_usage", cpu_usage);
    cJSON_AddNumberToObject(state_json, "memory_usage", memory_usage);
    cJSON_AddStringToObject(state_json, "connection_status", "connected");
    
    char *payload = cJSON_Print(state_json);
    int result = mosquitto_publish(client->mosq, NULL, topic, 
                                  strlen(payload), payload, 1, true);
    
    free(payload);
    cJSON_Delete(state_json);
    return result;
}
```

#### 命令處理
```c
void on_message(struct mosquitto *mosq, void *userdata, 
               const struct mosquitto_message *message) {
    rtk_mqtt_client_t *client = (rtk_mqtt_client_t *)userdata;
    
    // 解析命令
    cJSON *cmd_json = cJSON_Parse(message->payload);
    if (!cmd_json) return;
    
    cJSON *payload = cJSON_GetObjectItem(cmd_json, "payload");
    cJSON *id = cJSON_GetObjectItem(payload, "id");
    cJSON *op = cJSON_GetObjectItem(payload, "op");
    
    if (!id || !op) {
        cJSON_Delete(cmd_json);
        return;
    }
    
    // 發送 ACK
    rtk_mqtt_send_command_ack(client, id->valuestring);
    
    // 執行命令
    cJSON *result = rtk_mqtt_execute_command(client, op->valuestring, 
                                           cJSON_GetObjectItem(payload, "args"));
    
    // 發送結果
    rtk_mqtt_send_command_result(client, id->valuestring, result);
    
    cJSON_Delete(cmd_json);
    cJSON_Delete(result);
}

cJSON *rtk_mqtt_execute_command(rtk_mqtt_client_t *client, 
                               const char *operation, cJSON *args) {
    cJSON *result = cJSON_CreateObject();
    
    if (strcmp(operation, "device.status") == 0) {
        cJSON_AddStringToObject(result, "status", "healthy");
        cJSON_AddNumberToObject(result, "uptime", time(NULL));
        
    } else if (strcmp(operation, "speed_test") == 0) {
        // 實作速度測試
        cJSON_AddNumberToObject(result, "download_mbps", 85.2);
        cJSON_AddNumberToObject(result, "upload_mbps", 12.4);
        cJSON_AddNumberToObject(result, "latency_ms", 15.3);
        
    } else {
        cJSON_AddStringToObject(result, "error", "Unknown command");
    }
    
    return result;
}
```

### 編譯範例
```bash
# 編譯基礎範例
gcc -o rtk_device main.c -lmosquitto -lcjson -lssl -lcrypto

# 使用 CMake
mkdir build && cd build
cmake ..
make
```

## 🐍 Python SDK

### 安裝
```bash
pip install paho-mqtt jsonschema
```

### 基礎使用
```python
import paho.mqtt.client as mqtt
import json
import time
import threading
from typing import Dict, Any, Callable

class RTKMQTTClient:
    def __init__(self, device_id: str, tenant: str, site: str, 
                 device_type: str = "sensor"):
        self.device_id = device_id
        self.tenant = tenant
        self.site = site
        self.device_type = device_type
        self.client_id = f"rtk-{device_id}"
        
        self.client = mqtt.Client(self.client_id)
        self.connected = False
        self.command_handlers = {}
        
        # 設置回調
        self.client.on_connect = self._on_connect
        self.client.on_disconnect = self._on_disconnect  
        self.client.on_message = self._on_message
        
    def connect(self, host: str, port: int = 1883, 
                username: str = None, password: str = None):
        """連接到 MQTT Broker"""
        # 設置認證
        if username and password:
            self.client.username_pw_set(username, password)
        
        # 設置 LWT
        lwt_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/lwt"
        lwt_payload = {
            "schema": "lwt/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "status": "offline",
            "reason": "unexpected_disconnect"
        }
        
        self.client.will_set(lwt_topic, json.dumps(lwt_payload), qos=1, retain=True)
        
        # 建立連接
        return self.client.connect(host, port, 60)
    
    def _on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            self.connected = True
            print(f"Connected to MQTT Broker")
            
            # 訂閱命令主題
            cmd_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/req"
            client.subscribe(cmd_topic, qos=1)
            
            # 發送上線狀態和設備屬性
            self.publish_online_status()
            self.publish_device_attributes()
            
        else:
            print(f"Connection failed with code {rc}")
    
    def publish_device_attributes(self, **kwargs):
        """發布設備屬性"""
        topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/attr"
        
        attributes = {
            "schema": "attr/1.0",
            "ts": int(time.time() * 1000),
            "device_type": self.device_type,
            "manufacturer": kwargs.get("manufacturer", "RTK Systems"),
            "model": kwargs.get("model", "RTK-Device-2024"),
            "firmware_version": kwargs.get("firmware_version", "1.0.0"),
            "hardware_version": kwargs.get("hardware_version", "rev-a"),
            "mac_address": self.device_id,
            "capabilities": kwargs.get("capabilities", ["telemetry", "commands"])
        }
        
        self.client.publish(topic, json.dumps(attributes), qos=1, retain=True)
    
    def publish_state(self, health: str = "ok", **metrics):
        """發布設備狀態"""
        topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/state"
        
        state = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "health": health,
            "connection_status": "connected",
            **metrics
        }
        
        self.client.publish(topic, json.dumps(state), qos=1, retain=True)
    
    def publish_telemetry(self, metric: str, value: float, unit: str = None):
        """發布遙測資料"""
        topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/telemetry/{metric}"
        
        telemetry = {
            "schema": f"telemetry.{metric}/1.0",
            "ts": int(time.time() * 1000),
            "value": value
        }
        
        if unit:
            telemetry["unit"] = unit
            
        self.client.publish(topic, json.dumps(telemetry), qos=0)
    
    def publish_event(self, event_type: str, data: Dict[str, Any], 
                     severity: str = "info"):
        """發布事件通知"""
        topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/evt/{event_type}"
        
        event = {
            "schema": f"evt.{event_type}/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "event_type": event_type,
            "severity": severity,
            "data": data
        }
        
        self.client.publish(topic, json.dumps(event), qos=1)
    
    def register_command_handler(self, operation: str, handler: Callable):
        """註冊命令處理器"""
        self.command_handlers[operation] = handler
    
    def _on_message(self, client, userdata, msg):
        try:
            command = json.loads(msg.payload.decode())
            payload = command['payload']
            cmd_id = payload['id']
            operation = payload['op']
            args = payload.get('args', {})
            
            # 發送 ACK
            self._send_command_ack(cmd_id)
            
            # 執行命令
            if operation in self.command_handlers:
                result = self.command_handlers[operation](args)
                self._send_command_result(cmd_id, "completed", result)
            else:
                self._send_command_error(cmd_id, f"Unknown operation: {operation}")
                
        except Exception as e:
            print(f"Error processing command: {e}")
    
    def _send_command_ack(self, cmd_id: str):
        topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/ack"
        ack = {
            "schema": "cmd.ack/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "payload": {
                "id": cmd_id,
                "status": "received"
            }
        }
        self.client.publish(topic, json.dumps(ack), qos=1)
    
    def start_loop(self):
        """啟動事件循環"""
        self.client.loop_start()
    
    def stop(self):
        """停止客戶端"""
        if self.connected:
            self.publish_offline_status()
        self.client.loop_stop()
        self.client.disconnect()

# 使用範例
def device_status_handler(args):
    return {
        "uptime": int(time.time()),
        "memory_usage": 45.2,
        "cpu_usage": 25.7
    }

def speed_test_handler(args):
    # 模擬速度測試
    duration = args.get('duration', 30)
    time.sleep(min(duration, 5))  # 簡化版測試
    
    return {
        "download_mbps": 85.2,
        "upload_mbps": 12.4,
        "latency_ms": 15.3,
        "test_duration": duration
    }

# 創建客戶端
client = RTKMQTTClient(
    device_id="aabbccddeeff",
    tenant="home", 
    site="main",
    device_type="router"
)

# 註冊命令處理器
client.register_command_handler("device.status", device_status_handler)
client.register_command_handler("speed_test", speed_test_handler)

# 連接和啟動
client.connect("localhost", 1883)
client.start_loop()

# 定期發送狀態和遙測
def monitoring_loop():
    while client.connected:
        client.publish_state("ok", cpu_usage=25.4, memory_usage=45.2)
        client.publish_telemetry("cpu_usage", 25.4, "%")
        time.sleep(300)  # 每 5 分鐘

threading.Thread(target=monitoring_loop, daemon=True).start()
```

## 🔧 Go SDK

### 模組初始化
```bash
go mod init rtk-mqtt-device
go get github.com/eclipse/paho.mqtt.golang
go get github.com/json-iterator/go
```

### 基礎實作
```go
package main

import (
    "encoding/json"
    "fmt"
    "time"
    "sync"
    
    mqtt "github.com/eclipse/paho.mqtt.golang"
    jsoniter "github.com/json-iterator/go"
)

type RTKMQTTClient struct {
    DeviceID   string
    Tenant     string
    Site       string
    DeviceType string
    
    client     mqtt.Client
    connected  bool
    handlers   map[string]CommandHandler
    mutex      sync.RWMutex
}

type CommandHandler func(args map[string]interface{}) (interface{}, error)

type StateMessage struct {
    Schema           string  `json:"schema"`
    Timestamp        int64   `json:"ts"`
    Health           string  `json:"health"`
    ConnectionStatus string  `json:"connection_status"`
    CPUUsage         float64 `json:"cpu_usage,omitempty"`
    MemoryUsage      float64 `json:"memory_usage,omitempty"`
}

func NewRTKMQTTClient(deviceID, tenant, site, deviceType string) *RTKMQTTClient {
    return &RTKMQTTClient{
        DeviceID:   deviceID,
        Tenant:     tenant,
        Site:       site,
        DeviceType: deviceType,
        handlers:   make(map[string]CommandHandler),
    }
}

func (c *RTKMQTTClient) Connect(broker string, port int) error {
    opts := mqtt.NewClientOptions()
    opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
    opts.SetClientID(fmt.Sprintf("rtk-%s", c.DeviceID))
    opts.SetKeepAlive(60 * time.Second)
    opts.SetCleanSession(false)
    
    // 設置 LWT
    lwtTopic := fmt.Sprintf("rtk/v1/%s/%s/%s/lwt", c.Tenant, c.Site, c.DeviceID)
    lwtPayload := map[string]interface{}{
        "schema":    "lwt/1.0",
        "ts":        time.Now().UnixMilli(),
        "device_id": c.DeviceID,
        "status":    "offline",
        "reason":    "unexpected_disconnect",
    }
    
    lwtBytes, _ := json.Marshal(lwtPayload)
    opts.SetWill(lwtTopic, string(lwtBytes), 1, true)
    
    // 設置回調
    opts.SetDefaultPublishHandler(c.onMessage)
    opts.SetOnConnectHandler(c.onConnect)
    opts.SetConnectionLostHandler(c.onDisconnect)
    
    c.client = mqtt.NewClient(opts)
    
    if token := c.client.Connect(); token.Wait() && token.Error() != nil {
        return token.Error()
    }
    
    return nil
}

func (c *RTKMQTTClient) onConnect(client mqtt.Client) {
    c.mutex.Lock()
    c.connected = true
    c.mutex.Unlock()
    
    fmt.Println("Connected to MQTT broker")
    
    // 訂閱命令主題
    cmdTopic := fmt.Sprintf("rtk/v1/%s/%s/%s/cmd/req", c.Tenant, c.Site, c.DeviceID)
    client.Subscribe(cmdTopic, 1, nil)
    
    // 發送上線狀態和設備屬性
    c.PublishOnlineStatus()
    c.PublishDeviceAttributes()
}

func (c *RTKMQTTClient) PublishState(health string, metrics map[string]interface{}) error {
    topic := fmt.Sprintf("rtk/v1/%s/%s/%s/state", c.Tenant, c.Site, c.DeviceID)
    
    state := StateMessage{
        Schema:           "state/1.0",
        Timestamp:        time.Now().UnixMilli(),
        Health:           health,
        ConnectionStatus: "connected",
    }
    
    if cpu, ok := metrics["cpu_usage"].(float64); ok {
        state.CPUUsage = cpu
    }
    if mem, ok := metrics["memory_usage"].(float64); ok {
        state.MemoryUsage = mem
    }
    
    payload, err := json.Marshal(state)
    if err != nil {
        return err
    }
    
    token := c.client.Publish(topic, 1, true, payload)
    token.Wait()
    return token.Error()
}

func (c *RTKMQTTClient) PublishTelemetry(metric string, value float64, unit string) error {
    topic := fmt.Sprintf("rtk/v1/%s/%s/%s/telemetry/%s", c.Tenant, c.Site, c.DeviceID, metric)
    
    telemetry := map[string]interface{}{
        "schema": fmt.Sprintf("telemetry.%s/1.0", metric),
        "ts":     time.Now().UnixMilli(),
        "value":  value,
    }
    
    if unit != "" {
        telemetry["unit"] = unit
    }
    
    payload, err := json.Marshal(telemetry)
    if err != nil {
        return err
    }
    
    token := c.client.Publish(topic, 0, false, payload)
    token.Wait()
    return token.Error()
}

func (c *RTKMQTTClient) RegisterCommandHandler(operation string, handler CommandHandler) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    c.handlers[operation] = handler
}

func (c *RTKMQTTClient) onMessage(client mqtt.Client, msg mqtt.Message) {
    var command map[string]interface{}
    if err := json.Unmarshal(msg.Payload(), &command); err != nil {
        fmt.Printf("Failed to parse command: %v\n", err)
        return
    }
    
    payload, ok := command["payload"].(map[string]interface{})
    if !ok {
        return
    }
    
    cmdID, _ := payload["id"].(string)
    operation, _ := payload["op"].(string)
    args, _ := payload["args"].(map[string]interface{})
    
    // 發送 ACK
    c.sendCommandAck(cmdID)
    
    // 執行命令
    c.mutex.RLock()
    handler, exists := c.handlers[operation]
    c.mutex.RUnlock()
    
    if exists {
        result, err := handler(args)
        if err != nil {
            c.sendCommandError(cmdID, err.Error())
        } else {
            c.sendCommandResult(cmdID, result)
        }
    } else {
        c.sendCommandError(cmdID, fmt.Sprintf("Unknown operation: %s", operation))
    }
}

// 命令處理器範例
func deviceStatusHandler(args map[string]interface{}) (interface{}, error) {
    return map[string]interface{}{
        "uptime":       time.Now().Unix(),
        "memory_usage": 45.2,
        "cpu_usage":    25.7,
        "status":       "healthy",
    }, nil
}

func speedTestHandler(args map[string]interface{}) (interface{}, error) {
    duration := 30
    if d, ok := args["duration"].(float64); ok {
        duration = int(d)
    }
    
    // 模擬測試延遲
    time.Sleep(time.Duration(min(duration, 5)) * time.Second)
    
    return map[string]interface{}{
        "download_mbps":  85.2,
        "upload_mbps":    12.4,
        "latency_ms":     15.3,
        "test_duration":  duration,
    }, nil
}

func main() {
    client := NewRTKMQTTClient("aabbccddeeff", "home", "main", "router")
    
    // 註冊命令處理器
    client.RegisterCommandHandler("device.status", deviceStatusHandler)
    client.RegisterCommandHandler("speed_test", speedTestHandler)
    
    // 連接
    if err := client.Connect("localhost", 1883); err != nil {
        panic(err)
    }
    
    // 啟動監控循環
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        for range ticker.C {
            metrics := map[string]interface{}{
                "cpu_usage":    25.4,
                "memory_usage": 45.2,
            }
            client.PublishState("ok", metrics)
            client.PublishTelemetry("cpu_usage", 25.4, "%")
        }
    }()
    
    // 保持運行
    select {}
}
```

## 🌐 JavaScript/Node.js SDK

### 安裝
```bash
npm install mqtt jsonschema
```

### 基礎實作
```javascript
const mqtt = require('mqtt');

class RTKMQTTClient {
    constructor(deviceId, tenant, site, deviceType = 'sensor') {
        this.deviceId = deviceId;
        this.tenant = tenant;
        this.site = site;
        this.deviceType = deviceType;
        this.clientId = `rtk-${deviceId}`;
        
        this.connected = false;
        this.commandHandlers = new Map();
        this.client = null;
    }
    
    async connect(host, port = 1883, options = {}) {
        const connectOptions = {
            clientId: this.clientId,
            keepalive: 60,
            clean: false,
            will: {
                topic: `rtk/v1/${this.tenant}/${this.site}/${this.deviceId}/lwt`,
                payload: JSON.stringify({
                    schema: 'lwt/1.0',
                    ts: Date.now(),
                    device_id: this.deviceId,
                    status: 'offline',
                    reason: 'unexpected_disconnect'
                }),
                qos: 1,
                retain: true
            },
            ...options
        };
        
        this.client = mqtt.connect(`mqtt://${host}:${port}`, connectOptions);
        
        return new Promise((resolve, reject) => {
            this.client.on('connect', () => {
                this.connected = true;
                console.log('Connected to MQTT broker');
                
                // 訂閱命令主題
                const cmdTopic = `rtk/v1/${this.tenant}/${this.site}/${this.deviceId}/cmd/req`;
                this.client.subscribe(cmdTopic, { qos: 1 });
                
                // 發送上線狀態和設備屬性
                this.publishOnlineStatus();
                this.publishDeviceAttributes();
                
                resolve();
            });
            
            this.client.on('error', reject);
            this.client.on('message', this._onMessage.bind(this));
            this.client.on('close', () => {
                this.connected = false;
                console.log('Disconnected from MQTT broker');
            });
        });
    }
    
    publishDeviceAttributes(attributes = {}) {
        const topic = `rtk/v1/${this.tenant}/${this.site}/${this.deviceId}/attr`;
        
        const payload = {
            schema: 'attr/1.0',
            ts: Date.now(),
            device_type: this.deviceType,
            manufacturer: attributes.manufacturer || 'RTK Systems',
            model: attributes.model || 'RTK-Device-2024',
            firmware_version: attributes.firmware_version || '1.0.0',
            hardware_version: attributes.hardware_version || 'rev-a',
            mac_address: this.deviceId,
            capabilities: attributes.capabilities || ['telemetry', 'commands']
        };
        
        this.client.publish(topic, JSON.stringify(payload), { qos: 1, retain: true });
    }
    
    publishState(health = 'ok', metrics = {}) {
        const topic = `rtk/v1/${this.tenant}/${this.site}/${this.deviceId}/state`;
        
        const payload = {
            schema: 'state/1.0',
            ts: Date.now(),
            health,
            connection_status: 'connected',
            ...metrics
        };
        
        this.client.publish(topic, JSON.stringify(payload), { qos: 1, retain: true });
    }
    
    publishTelemetry(metric, value, unit = null) {
        const topic = `rtk/v1/${this.tenant}/${this.site}/${this.deviceId}/telemetry/${metric}`;
        
        const payload = {
            schema: `telemetry.${metric}/1.0`,
            ts: Date.now(),
            value
        };
        
        if (unit) payload.unit = unit;
        
        this.client.publish(topic, JSON.stringify(payload), { qos: 0 });
    }
    
    publishEvent(eventType, data, severity = 'info') {
        const topic = `rtk/v1/${this.tenant}/${this.site}/${this.deviceId}/evt/${eventType}`;
        
        const payload = {
            schema: `evt.${eventType}/1.0`,
            ts: Date.now(),
            device_id: this.deviceId,
            event_type: eventType,
            severity,
            data
        };
        
        this.client.publish(topic, JSON.stringify(payload), { qos: 1 });
    }
    
    registerCommandHandler(operation, handler) {
        this.commandHandlers.set(operation, handler);
    }
    
    async _onMessage(topic, message) {
        try {
            const command = JSON.parse(message.toString());
            const { payload } = command;
            const { id: cmdId, op: operation, args = {} } = payload;
            
            // 發送 ACK
            this._sendCommandAck(cmdId);
            
            // 執行命令
            const handler = this.commandHandlers.get(operation);
            if (handler) {
                try {
                    const result = await handler(args);
                    this._sendCommandResult(cmdId, 'completed', result);
                } catch (error) {
                    this._sendCommandError(cmdId, error.message);
                }
            } else {
                this._sendCommandError(cmdId, `Unknown operation: ${operation}`);
            }
        } catch (error) {
            console.error('Error processing command:', error);
        }
    }
    
    _sendCommandAck(cmdId) {
        const topic = `rtk/v1/${this.tenant}/${this.site}/${this.deviceId}/cmd/ack`;
        const payload = {
            schema: 'cmd.ack/1.0',
            ts: Date.now(),
            device_id: this.deviceId,
            payload: {
                id: cmdId,
                status: 'received'
            }
        };
        
        this.client.publish(topic, JSON.stringify(payload), { qos: 1 });
    }
    
    _sendCommandResult(cmdId, status, result) {
        const topic = `rtk/v1/${this.tenant}/${this.site}/${this.deviceId}/cmd/res`;
        const payload = {
            schema: 'cmd.result/1.0',
            ts: Date.now(),
            device_id: this.deviceId,
            payload: {
                id: cmdId,
                status,
                result
            }
        };
        
        this.client.publish(topic, JSON.stringify(payload), { qos: 1 });
    }
    
    disconnect() {
        if (this.connected) {
            this.publishOfflineStatus();
            this.client.end();
        }
    }
}

// 使用範例
async function main() {
    const client = new RTKMQTTClient('aabbccddeeff', 'home', 'main', 'router');
    
    // 註冊命令處理器
    client.registerCommandHandler('device.status', async (args) => {
        return {
            uptime: Math.floor(Date.now() / 1000),
            memory_usage: 45.2,
            cpu_usage: 25.7,
            status: 'healthy'
        };
    });
    
    client.registerCommandHandler('speed_test', async (args) => {
        const duration = args.duration || 30;
        
        // 模擬測試延遲
        await new Promise(resolve => 
            setTimeout(resolve, Math.min(duration, 5) * 1000)
        );
        
        return {
            download_mbps: 85.2,
            upload_mbps: 12.4,
            latency_ms: 15.3,
            test_duration: duration
        };
    });
    
    // 連接
    await client.connect('localhost', 1883);
    
    // 啟動監控循環
    setInterval(() => {
        client.publishState('ok', {
            cpu_usage: 25.4,
            memory_usage: 45.2
        });
        
        client.publishTelemetry('cpu_usage', 25.4, '%');
    }, 5 * 60 * 1000); // 每 5 分鐘
    
    // 優雅關閉
    process.on('SIGINT', () => {
        console.log('Shutting down...');
        client.disconnect();
        process.exit(0);
    });
}

main().catch(console.error);
```

## 🔧 進階功能

### TLS/SSL 支援

#### Python
```python
import ssl

context = ssl.create_default_context(ssl.Purpose.SERVER_AUTH)
context.check_hostname = False
context.verify_mode = ssl.CERT_REQUIRED
context.load_verify_locations("ca.crt")
context.load_cert_chain("client.crt", "client.key")

client.tls_set_context(context)
```

#### Go
```go
import "crypto/tls"

tlsConfig := &tls.Config{
    InsecureSkipVerify: false,
    ClientAuth:         tls.RequireAndVerifyClientCert,
}

opts.SetTLSConfig(tlsConfig)
```

### 批次操作
```python
class BatchProcessor:
    def __init__(self, client):
        self.client = client
        self.batch = []
        
    def add_telemetry(self, metric, value, unit=None):
        self.batch.append(('telemetry', metric, value, unit))
        
    def add_event(self, event_type, data):
        self.batch.append(('event', event_type, data))
        
    def flush(self):
        for item in self.batch:
            if item[0] == 'telemetry':
                self.client.publish_telemetry(item[1], item[2], item[3])
            elif item[0] == 'event':
                self.client.publish_event(item[1], item[2])
        self.batch.clear()
```

## 📊 效能最佳化

### 連接池 (Python)
```python
class RTKMQTTConnectionPool:
    def __init__(self, max_connections=10):
        self.pool = queue.Queue(maxsize=max_connections)
        self.max_connections = max_connections
        
    def get_client(self):
        try:
            return self.pool.get_nowait()
        except queue.Empty:
            return self.create_client()
            
    def return_client(self, client):
        try:
            self.pool.put_nowait(client)
        except queue.Full:
            client.disconnect()
```

### 記憶體管理 (C++)
```cpp
class RTKMQTTClientManager {
private:
    std::unique_ptr<mosquitto, void(*)(mosquitto*)> client_;
    std::unordered_map<std::string, std::string> message_cache_;
    static constexpr size_t MAX_CACHE_SIZE = 1000;
    
public:
    void cleanup_old_messages() {
        if (message_cache_.size() > MAX_CACHE_SIZE) {
            message_cache_.clear();
        }
    }
};
```

## 🔗 相關資源

- **[API 完整參考](../core/MQTT_API_REFERENCE.md)** - 所有 API 格式規範
- **[整合指南](../INTEGRATION_GUIDE.md)** - 通用整合步驟
- **[測試工具](MQTT_TESTING_TOOLS.md)** - SDK 測試方法
- **[範例專案](https://github.com/rtk-mqtt/examples)** - 完整的範例代碼

---

這個 SDK 參考指南提供了四種主流程式語言的完整實作範例，開發者可以根據專案需求選擇合適的語言和 SDK 進行 RTK MQTT 整合。
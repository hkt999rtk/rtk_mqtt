# NIC 設備 MQTT 整合指南

## 概述

本文檔提供網路介面卡(NIC)設備與RTK MQTT協議整合的完整指南，包含訊息格式、命令處理和實作範例。

## NIC設備特性

### 設備類型識別
```json
{
  "device_type": "nic",
  "capabilities": ["ethernet", "wifi_client", "monitoring"],
  "interfaces": ["eth0", "wlan0"]
}
```

### 主要功能
- 網路連接狀態監控
- 流量統計收集
- 連接品質分析
- 漫遊事件檢測

## MQTT 主題結構

### 基本主題格式
```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}[/{sub_type}]
```

### NIC 專用主題
```
rtk/v1/{tenant}/{site}/{nic_mac}/state
rtk/v1/{tenant}/{site}/{nic_mac}/telemetry/network
rtk/v1/{tenant}/{site}/{nic_mac}/telemetry/wifi
rtk/v1/{tenant}/{site}/{nic_mac}/evt/connection
rtk/v1/{tenant}/{site}/{nic_mac}/evt/roaming
```

## 狀態訊息 (state)

### 訊息格式
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "health": "ok",
    "uptime_s": 86400,
    "connection_status": "connected",
    "active_interface": "wlan0",
    "ip_address": "192.168.1.100",
    "gateway": "192.168.1.1",
    "dns_servers": ["8.8.8.8", "8.8.4.4"]
  }
}
```

### 發送規則
- **QoS**: 1
- **Retained**: true
- **頻率**: 每5分鐘或狀態變化時

## 遙測數據 (telemetry)

### 網路遙測 (telemetry/network)
```json
{
  "schema": "telemetry.network/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "interface": "wlan0",
    "tx_bytes": 1048576,
    "rx_bytes": 2097152,
    "tx_packets": 1024,
    "rx_packets": 2048,
    "tx_errors": 0,
    "rx_errors": 0,
    "latency_ms": 15.5,
    "jitter_ms": 2.1,
    "packet_loss_rate": 0.01
  }
}
```

### WiFi 遙測 (telemetry/wifi)
```json
{
  "schema": "telemetry.wifi/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "ssid": "Office-WiFi",
    "bssid": "aabbccddeeff",
    "channel": 6,
    "frequency": 2437,
    "signal_strength": -45,
    "link_quality": 85,
    "tx_rate_mbps": 150,
    "rx_rate_mbps": 130,
    "security": "WPA2-PSK"
  }
}
```

## 事件通知 (evt)

### 連接事件 (evt/connection)
```json
{
  "schema": "evt.connection/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "event_type": "disconnected",
    "interface": "wlan0",
    "reason": "signal_lost",
    "previous_state": "connected",
    "duration_s": 3600
  }
}
```

### 漫遊事件 (evt/roaming)
```json
{
  "schema": "evt.roaming/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "event_type": "roam_completed",
    "from_bssid": "aabbccddeeff",
    "to_bssid": "aabbccddeeff",
    "roam_time_ms": 150,
    "trigger": "signal_strength",
    "success": true
  }
}
```

## 支援的命令

### 網路診斷命令

#### 速度測試
```json
{
  "schema": "cmd.speed_test/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-1699123456789",
    "op": "speed_test",
    "args": {
      "server": "auto",
      "duration": 10,
      "direction": "both"
    }
  }
}
```

#### 連線測試
```json
{
  "schema": "cmd.wan_connectivity/1.0",
  "ts": 1699123456790,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-1699123456790",
    "op": "wan_connectivity",
    "args": {
      "test_hosts": ["8.8.8.8", "google.com"],
      "timeout": 5
    }
  }
}
```

### WiFi 管理命令

#### WiFi 掃描
```json
{
  "schema": "cmd.wifi_scan/1.0",
  "ts": 1699123456791,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-1699123456791",
    "op": "wifi_scan",
    "args": {
      "scan_type": "active",
      "channels": [1, 6, 11],
      "duration": 10
    }
  }
}
```

### 系統管理命令

#### 重新啟動網路服務
```json
{
  "schema": "cmd.restart_service/1.0",
  "ts": 1699123456792,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-1699123456792",
    "op": "restart_service",
    "args": {
      "service": "network-manager"
    }
  }
}
```

## 實作範例

### Python 客戶端實作

```python
import paho.mqtt.client as mqtt
import json
import time
import psutil
import subprocess

class NICMQTTClient:
    def __init__(self, broker_host, device_id, tenant="demo", site="office"):
        self.client = mqtt.Client()
        self.broker_host = broker_host
        self.device_id = device_id
        self.tenant = tenant
        self.site = site
        
        # 設定回調函數
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        
    def on_connect(self, client, userdata, flags, rc):
        print(f"Connected with result code {rc}")
        # 訂閱命令主題
        cmd_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/req"
        client.subscribe(cmd_topic, qos=1)
        
    def on_message(self, client, userdata, msg):
        try:
            command = json.loads(msg.payload.decode())
            self.handle_command(command)
        except Exception as e:
            print(f"Error processing command: {e}")
            
    def handle_command(self, command):
        payload = command.get("payload", {})
        cmd_id = payload.get("id")
        operation = payload.get("op")
        
        # 發送確認
        self.send_ack(cmd_id)
        
        # 處理命令
        if operation == "speed_test":
            result = self.speed_test(payload.get("args", {}))
        elif operation == "wifi_scan":
            result = self.wifi_scan(payload.get("args", {}))
        elif operation == "wan_connectivity":
            result = self.wan_connectivity_test(payload.get("args", {}))
        else:
            result = {"error": "Unsupported command"}
            
        # 發送結果
        self.send_result(cmd_id, operation, result)
        
    def send_ack(self, cmd_id):
        ack_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/ack"
        ack_msg = {
            "schema": "cmd.ack/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "payload": {
                "id": cmd_id,
                "status": "accepted"
            }
        }
        self.client.publish(ack_topic, json.dumps(ack_msg), qos=1)
        
    def send_result(self, cmd_id, operation, result):
        res_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/res"
        res_msg = {
            "schema": f"cmd.{operation}.result/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "payload": {
                "id": cmd_id,
                "status": "completed" if "error" not in result else "failed",
                "result": result
            }
        }
        self.client.publish(res_topic, json.dumps(res_msg), qos=1)
        
    def speed_test(self, args):
        """執行網路速度測試"""
        try:
            # 簡化的速度測試實作
            return {
                "download_mbps": 100.5,
                "upload_mbps": 50.2,
                "latency_ms": 15.3,
                "server": args.get("server", "auto")
            }
        except Exception as e:
            return {"error": str(e)}
            
    def wifi_scan(self, args):
        """執行WiFi掃描"""
        try:
            # 使用iwlist或類似工具掃描WiFi
            result = subprocess.run(['iwlist', 'wlan0', 'scan'], 
                                  capture_output=True, text=True)
            # 解析掃描結果
            networks = []
            return {"networks": networks}
        except Exception as e:
            return {"error": str(e)}
            
    def wan_connectivity_test(self, args):
        """執行WAN連線測試"""
        try:
            test_hosts = args.get("test_hosts", ["8.8.8.8"])
            results = []
            for host in test_hosts:
                result = subprocess.run(['ping', '-c', '3', host], 
                                      capture_output=True, text=True)
                success = result.returncode == 0
                results.append({"host": host, "success": success})
            return {"connectivity_results": results}
        except Exception as e:
            return {"error": str(e)}
            
    def publish_state(self):
        """發布設備狀態"""
        state_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/state"
        state_msg = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "payload": {
                "health": "ok",
                "uptime_s": int(time.time() - psutil.boot_time()),
                "connection_status": "connected",
                "active_interface": "wlan0"
            }
        }
        self.client.publish(state_topic, json.dumps(state_msg), qos=1, retain=True)
        
    def publish_network_telemetry(self):
        """發布網路遙測數據"""
        telemetry_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/telemetry/network"
        stats = psutil.net_io_counters()
        telemetry_msg = {
            "schema": "telemetry.network/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "payload": {
                "interface": "wlan0",
                "tx_bytes": stats.bytes_sent,
                "rx_bytes": stats.bytes_recv,
                "tx_packets": stats.packets_sent,
                "rx_packets": stats.packets_recv
            }
        }
        self.client.publish(telemetry_topic, json.dumps(telemetry_msg), qos=0)
        
    def start(self):
        self.client.connect(self.broker_host, 1883, 60)
        self.client.loop_start()
        
        # 定期發布狀態和遙測數據
        while True:
            self.publish_state()
            self.publish_network_telemetry()
            time.sleep(30)

# 使用範例
if __name__ == "__main__":
    nic_client = NICMQTTClient("localhost", "aabbccddeeff")
    nic_client.start()
```

### C++ 客戶端實作

```cpp
#include <mosquitto.h>
#include <json/json.h>
#include <iostream>
#include <string>
#include <chrono>

class NICMQTTClient {
private:
    struct mosquitto *mosq;
    std::string device_id;
    std::string tenant;
    std::string site;
    
public:
    NICMQTTClient(const std::string& device_id, 
                  const std::string& tenant = "demo", 
                  const std::string& site = "office") 
        : device_id(device_id), tenant(tenant), site(site) {
        
        mosquitto_lib_init();
        mosq = mosquitto_new(device_id.c_str(), true, this);
        
        mosquitto_connect_callback_set(mosq, on_connect);
        mosquitto_message_callback_set(mosq, on_message);
    }
    
    ~NICMQTTClient() {
        mosquitto_destroy(mosq);
        mosquitto_lib_cleanup();
    }
    
    static void on_connect(struct mosquitto *mosq, void *obj, int result) {
        NICMQTTClient* client = static_cast<NICMQTTClient*>(obj);
        if (result == 0) {
            std::string cmd_topic = "rtk/v1/" + client->tenant + "/" + 
                                   client->site + "/" + client->device_id + "/cmd/req";
            mosquitto_subscribe(mosq, nullptr, cmd_topic.c_str(), 1);
        }
    }
    
    static void on_message(struct mosquitto *mosq, void *obj, 
                          const struct mosquitto_message *message) {
        NICMQTTClient* client = static_cast<NICMQTTClient*>(obj);
        client->handle_message(message);
    }
    
    void handle_message(const struct mosquitto_message *message) {
        Json::Reader reader;
        Json::Value command;
        
        std::string payload(static_cast<char*>(message->payload), message->payloadlen);
        if (reader.parse(payload, command)) {
            std::string cmd_id = command["id"].asString();
            std::string operation = command["op"].asString();
            
            // 發送確認
            send_ack(cmd_id);
            
            // 處理命令
            Json::Value result;
            if (operation == "speed_test") {
                result = handle_speed_test(command["args"]);
            } else if (operation == "wifi_scan") {
                result = handle_wifi_scan(command["args"]);
            } else {
                result["error"] = "Unsupported command";
            }
            
            // 發送結果
            send_result(cmd_id, operation, result);
        }
    }
    
    void send_ack(const std::string& cmd_id) {
        Json::Value ack;
        ack["id"] = cmd_id;
        ack["schema"] = "cmd.ack/1.0";
        ack["status"] = "accepted";
        ack["ts"] = std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()).count();
        
        Json::StreamWriterBuilder builder;
        std::string ack_str = Json::writeString(builder, ack);
        
        std::string topic = "rtk/v1/" + tenant + "/" + site + "/" + device_id + "/cmd/ack";
        mosquitto_publish(mosq, nullptr, topic.c_str(), ack_str.length(), 
                         ack_str.c_str(), 1, false);
    }
    
    void send_result(const std::string& cmd_id, const std::string& operation, 
                    const Json::Value& result) {
        Json::Value response;
        response["id"] = cmd_id;
        response["schema"] = "cmd." + operation + ".result/1.0";
        response["status"] = result.isMember("error") ? "failed" : "completed";
        response["result"] = result;
        response["ts"] = std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()).count();
        
        Json::StreamWriterBuilder builder;
        std::string response_str = Json::writeString(builder, response);
        
        std::string topic = "rtk/v1/" + tenant + "/" + site + "/" + device_id + "/cmd/res";
        mosquitto_publish(mosq, nullptr, topic.c_str(), response_str.length(), 
                         response_str.c_str(), 1, false);
    }
    
    Json::Value handle_speed_test(const Json::Value& args) {
        Json::Value result;
        result["download_mbps"] = 95.5;
        result["upload_mbps"] = 45.2;
        result["latency_ms"] = 18.7;
        result["server"] = args.get("server", "auto").asString();
        return result;
    }
    
    Json::Value handle_wifi_scan(const Json::Value& args) {
        Json::Value result;
        Json::Value networks(Json::arrayValue);
        // 實際實作應該呼叫系統WiFi掃描功能
        result["networks"] = networks;
        return result;
    }
    
    void connect(const std::string& host, int port = 1883) {
        mosquitto_connect(mosq, host.c_str(), port, 60);
        mosquitto_loop_start(mosq);
    }
    
    void publish_state() {
        Json::Value state;
        state["schema"] = "state/1.0";
        state["ts"] = std::chrono::duration_cast<std::chrono::milliseconds>(
            std::chrono::system_clock::now().time_since_epoch()).count();
        state["health"] = "ok";
        state["connection_status"] = "connected";
        
        Json::StreamWriterBuilder builder;
        std::string state_str = Json::writeString(builder, state);
        
        std::string topic = "rtk/v1/" + tenant + "/" + site + "/" + device_id + "/state";
        mosquitto_publish(mosq, nullptr, topic.c_str(), state_str.length(), 
                         state_str.c_str(), 1, true);
    }
};
```

## 測試與驗證

### 命令測試清單
- [ ] 速度測試命令 (`speed_test`)
- [ ] WAN連線測試 (`wan_connectivity`) 
- [ ] WiFi掃描 (`wifi_scan`)
- [ ] 網路服務重啟 (`restart_service`)
- [ ] 系統資訊獲取 (`get_system_info`)

### 遙測數據驗證
- [ ] 網路介面統計數據準確性
- [ ] WiFi信號強度測量
- [ ] 連接狀態變化檢測
- [ ] 漫遊事件觸發

### 整合測試場景
1. **正常運作流程**
   - 設備連接MQTT broker
   - 定期發布狀態和遙測數據
   - 響應管理命令

2. **網路中斷處理**
   - WiFi斷線事件檢測
   - 自動重連機制
   - Last Will Testament觸發

3. **漫遊場景測試**
   - 在多個AP間移動
   - 漫遊事件正確記錄
   - 連接品質變化追蹤

## 故障排除

### 常見問題

1. **MQTT連線失敗**
   - 檢查broker連線參數
   - 驗證認證資訊
   - 確認網路連通性

2. **命令執行失敗**
   - 驗證命令格式正確性
   - 檢查權限設定
   - 查看系統錯誤日誌

3. **遙測數據異常**
   - 確認網路介面名稱
   - 檢查系統API權限
   - 驗證數據單位轉換

### 除錯工具
```bash
# 監控MQTT流量
mosquitto_sub -h localhost -t "rtk/v1/+/+/+/#" -v

# 發送測試命令
mosquitto_pub -h localhost -t "rtk/v1/demo/office/test-nic/cmd/req" -m '{
  "schema": "cmd.speed_test/1.0",
  "ts": 1699123456789,
  "device_id": "test-nic",
  "payload": {
    "id": "test-cmd-123",
    "op": "speed_test",
    "args": {"duration": 5}
  }
}'

# 檢查網路介面狀態
ip link show
iwconfig
```

## 效能考量

### 資源使用
- CPU使用率: < 5%
- 記憶體使用: < 50MB
- 網路頻寬: < 1KB/s (正常遙測)

### 最佳化建議
- 調整遙測頻率平衡即時性和資源消耗
- 使用QoS 0進行非關鍵遙測數據
- 實作本地快取減少重複計算
- 批次發送事件降低網路負載

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Commands and Events Reference](../core/COMMANDS_EVENTS_REFERENCE.md)
- [Topic Structure Guide](../core/TOPIC_STRUCTURE.md)
- [Schema Reference](../core/SCHEMA_REFERENCE.md)
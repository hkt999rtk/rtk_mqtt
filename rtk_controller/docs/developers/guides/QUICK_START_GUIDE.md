# RTK MQTT 快速入門指南

## 概述

本指南幫助開發者快速開始使用RTK MQTT協議，從環境設置到第一個設備整合，提供step-by-step的實作步驟。

## 先決條件

### 系統需求
- **作業系統**: Linux, macOS, Windows
- **程式語言**: Python 3.7+, Go 1.19+, C/C++, JavaScript/Node.js
- **網路**: MQTT broker存取權限
- **硬體**: 支援網路的設備(可選)

### 必要軟體
```bash
# MQTT工具
sudo apt-get install mosquitto mosquitto-clients

# Python依賴
pip install paho-mqtt

# Go依賴  
go mod init rtk-mqtt-example
go get github.com/eclipse/paho.mqtt.golang
```

## 步驟1: 設置MQTT Broker

### 使用Mosquitto (建議用於開發)
```bash
# 安裝Mosquitto
sudo apt-get install mosquitto mosquitto-clients

# 啟動broker
sudo systemctl start mosquitto
sudo systemctl enable mosquitto

# 測試連線
mosquitto_pub -h localhost -t test -m "Hello RTK MQTT"
mosquitto_sub -h localhost -t test
```

### 使用RTK專用Broker
```bash
# 編譯RTK Broker
cd rtk_mqtt_broker
go build -o rtk_mqtt_broker .

# 配置broker
cp config/config.yaml.example config/config.yaml

# 啟動broker
./rtk_mqtt_broker -config config/config.yaml
```

### Broker配置檔案範例
```yaml
# config/config.yaml
server:
  host: "0.0.0.0"
  port: 1883
  websocket_port: 8083

auth:
  enabled: false
  users:
    - username: "rtk_device"
      password: "device_password"
    - username: "rtk_controller"
      password: "controller_password"

logging:
  level: "info"
  format: "json"

limits:
  max_clients: 1000
  max_subscriptions: 100
  max_message_size: 1048576
```

## 步驟2: 第一個RTK MQTT設備

### 基本設備模擬器 (Python)

建立檔案 `simple_device.py`:

```python
#!/usr/bin/env python3
import paho.mqtt.client as mqtt
import json
import time
import random
import uuid

class SimpleRTKDevice:
    def __init__(self, broker_host="localhost", device_id=None):
        # 設備識別
        self.device_id = device_id or f"device_{uuid.uuid4().hex[:8]}"
        self.tenant = "demo"
        self.site = "quickstart"
        
        # MQTT客戶端設置
        self.client = mqtt.Client(client_id=self.device_id)
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        self.broker_host = broker_host
        
        # 設備狀態
        self.health = "ok"
        self.uptime = 0
        
        print(f"Initialized RTK device: {self.device_id}")
        
    def on_connect(self, client, userdata, flags, rc):
        print(f"Connected to broker with result code {rc}")
        
        # 訂閱命令主題
        cmd_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/req"
        client.subscribe(cmd_topic, qos=1)
        print(f"Subscribed to: {cmd_topic}")
        
        # 發布設備屬性
        self.publish_attributes()
        
    def on_message(self, client, userdata, msg):
        try:
            command = json.loads(msg.payload.decode())
            print(f"Received command: {command}")
            self.handle_command(command)
        except Exception as e:
            print(f"Error processing command: {e}")
            
    def handle_command(self, command):
        cmd_id = command.get("id")
        operation = command.get("op")
        
        # 發送命令確認
        self.send_ack(cmd_id)
        
        # 處理命令
        if operation == "get_system_info":
            result = self.get_system_info()
        elif operation == "reboot":
            result = self.simulate_reboot()
        elif operation == "update_config":
            result = self.update_config(command.get("args", {}))
        else:
            result = {"error": "Unsupported command"}
            
        # 發送命令結果
        self.send_result(cmd_id, operation, result)
        
    def send_ack(self, cmd_id):
        ack_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/ack"
        ack_msg = {
            "id": cmd_id,
            "schema": "cmd.ack/1.0",
            "status": "accepted",
            "ts": int(time.time() * 1000)
        }
        self.client.publish(ack_topic, json.dumps(ack_msg), qos=1)
        print(f"Sent ACK for command: {cmd_id}")
        
    def send_result(self, cmd_id, operation, result):
        res_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/res"
        res_msg = {
            "id": cmd_id,
            "schema": f"cmd.{operation}.result/1.0",
            "status": "completed" if "error" not in result else "failed",
            "result": result,
            "ts": int(time.time() * 1000)
        }
        self.client.publish(res_topic, json.dumps(res_msg), qos=1)
        print(f"Sent result for command: {cmd_id}")
        
    def get_system_info(self):
        return {
            "device_id": self.device_id,
            "uptime_s": self.uptime,
            "health": self.health,
            "memory_usage": random.uniform(20, 80),
            "cpu_usage": random.uniform(10, 50),
            "version": "1.0.0"
        }
        
    def simulate_reboot(self):
        print("Simulating device reboot...")
        self.uptime = 0
        return {"reboot_initiated": True, "estimated_downtime_s": 30}
        
    def update_config(self, config):
        print(f"Updating config: {config}")
        return {"config_updated": True, "applied_settings": config}
        
    def publish_attributes(self):
        """發布設備屬性 (一次性)"""
        attr_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/attr"
        attr_msg = {
            "schema": "attr/1.0",
            "ts": int(time.time() * 1000),
            "device_type": "demo_device",
            "manufacturer": "RTK Demo",
            "model": "QuickStart-1.0",
            "firmware_version": "1.0.0",
            "capabilities": ["telemetry", "remote_control", "diagnostics"]
        }
        self.client.publish(attr_topic, json.dumps(attr_msg), qos=1, retain=True)
        print("Published device attributes")
        
    def publish_state(self):
        """發布設備狀態"""
        state_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/state"
        state_msg = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "health": self.health,
            "uptime_s": self.uptime,
            "connection_status": "connected"
        }
        self.client.publish(state_topic, json.dumps(state_msg), qos=1, retain=True)
        
    def publish_telemetry(self):
        """發布遙測數據"""
        telemetry_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/telemetry/system"
        telemetry_msg = {
            "schema": "telemetry.system/1.0",
            "ts": int(time.time() * 1000),
            "cpu_usage": random.uniform(10, 50),
            "memory_usage": random.uniform(20, 80),
            "temperature_c": random.uniform(35, 45),
            "load_average": random.uniform(0.1, 2.0)
        }
        self.client.publish(telemetry_topic, json.dumps(telemetry_msg), qos=0)
        
    def publish_event(self, event_type, data):
        """發布事件"""
        evt_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/evt/{event_type}"
        evt_msg = {
            "schema": f"evt.{event_type}/1.0",
            "ts": int(time.time() * 1000),
            **data
        }
        self.client.publish(evt_topic, json.dumps(evt_msg), qos=1)
        print(f"Published event: {event_type}")
        
    def start(self):
        """啟動設備"""
        # 連接到broker
        print(f"Connecting to broker at {self.broker_host}...")
        self.client.connect(self.broker_host, 1883, 60)
        self.client.loop_start()
        
        try:
            while True:
                # 定期發布狀態和遙測
                self.publish_state()
                self.publish_telemetry()
                
                # 增加運行時間
                self.uptime += 30
                
                # 模擬隨機事件
                if random.random() < 0.1:  # 10%機率
                    event_types = ["system", "network", "performance"]
                    event_type = random.choice(event_types)
                    event_data = {
                        "event_type": "status_change", 
                        "description": f"Random {event_type} event",
                        "severity": random.choice(["info", "warning"])
                    }
                    self.publish_event(event_type, event_data)
                
                time.sleep(30)
                
        except KeyboardInterrupt:
            print("\\nShutting down device...")
            self.client.disconnect()

if __name__ == "__main__":
    import sys
    
    broker_host = sys.argv[1] if len(sys.argv) > 1 else "localhost"
    device = SimpleRTKDevice(broker_host)
    device.start()
```

### 運行設備模擬器

```bash
# 基本運行
python3 simple_device.py

# 指定broker位址
python3 simple_device.py 192.168.1.100
```

## 步驟3: 監控設備訊息

### 訂閱所有RTK訊息
```bash
# 監控所有RTK訊息
mosquitto_sub -h localhost -t "rtk/v1/#" -v

# 只監控狀態訊息
mosquitto_sub -h localhost -t "rtk/v1/+/+/+/state" -v

# 監控特定設備
mosquitto_sub -h localhost -t "rtk/v1/demo/quickstart/device_12345678/#" -v
```

### 監控腳本 (monitor.py)
```python
#!/usr/bin/env python3
import paho.mqtt.client as mqtt
import json
from datetime import datetime

class RTKMonitor:
    def __init__(self, broker_host="localhost"):
        self.client = mqtt.Client()
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        self.broker_host = broker_host
        
    def on_connect(self, client, userdata, flags, rc):
        print(f"Monitor connected with result code {rc}")
        # 訂閱所有RTK訊息
        client.subscribe("rtk/v1/#", qos=0)
        
    def on_message(self, client, userdata, msg):
        try:
            timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            topic_parts = msg.topic.split('/')
            
            if len(topic_parts) >= 6:
                tenant = topic_parts[2]
                site = topic_parts[3] 
                device_id = topic_parts[4]
                msg_type = topic_parts[5]
                sub_type = topic_parts[6] if len(topic_parts) > 6 else ""
                
                payload = json.loads(msg.payload.decode())
                
                print(f"[{timestamp}] {tenant}/{site}/{device_id}")
                print(f"  Type: {msg_type}/{sub_type}")
                print(f"  Schema: {payload.get('schema', 'N/A')}")
                
                if msg_type == "state":
                    print(f"  Health: {payload.get('health', 'N/A')}")
                    print(f"  Uptime: {payload.get('uptime_s', 0)}s")
                elif msg_type == "telemetry":
                    print(f"  Data: {json.dumps(payload, indent=4)}")
                elif msg_type == "evt":
                    print(f"  Event: {payload.get('event_type', 'N/A')}")
                elif msg_type == "cmd":
                    if sub_type == "req":
                        print(f"  Command: {payload.get('op', 'N/A')}")
                    elif sub_type == "ack":
                        print(f"  ACK Status: {payload.get('status', 'N/A')}")
                    elif sub_type == "res":
                        print(f"  Result Status: {payload.get('status', 'N/A')}")
                        
                print("-" * 50)
                
        except Exception as e:
            print(f"Error processing message: {e}")
            
    def start(self):
        print(f"Starting RTK monitor, connecting to {self.broker_host}")
        self.client.connect(self.broker_host, 1883, 60)
        self.client.loop_forever()

if __name__ == "__main__":
    import sys
    broker_host = sys.argv[1] if len(sys.argv) > 1 else "localhost"
    monitor = RTKMonitor(broker_host)
    monitor.start()
```

## 步驟4: 發送命令到設備

### 命令發送腳本 (send_command.py)
```python
#!/usr/bin/env python3
import paho.mqtt.client as mqtt
import json
import time
import uuid

def send_rtk_command(broker_host, device_id, operation, args=None, tenant="demo", site="quickstart"):
    """發送RTK命令到指定設備"""
    
    client = mqtt.Client()
    client.connect(broker_host, 1883, 60)
    
    # 構建命令
    cmd_id = f"cmd-{int(time.time() * 1000)}"
    command = {
        "id": cmd_id,
        "op": operation,
        "schema": f"cmd.{operation}/1.0",
        "ts": int(time.time() * 1000)
    }
    
    if args:
        command["args"] = args
        
    # 發送命令
    cmd_topic = f"rtk/v1/{tenant}/{site}/{device_id}/cmd/req"
    client.publish(cmd_topic, json.dumps(command), qos=1)
    
    print(f"Sent command '{operation}' to device {device_id}")
    print(f"Command ID: {cmd_id}")
    print(f"Topic: {cmd_topic}")
    print(f"Payload: {json.dumps(command, indent=2)}")
    
    client.disconnect()
    return cmd_id

if __name__ == "__main__":
    import sys
    
    if len(sys.argv) < 4:
        print("Usage: python3 send_command.py <broker_host> <device_id> <operation> [args_json]")
        print("Examples:")
        print("  python3 send_command.py localhost device_12345678 get_system_info")
        print("  python3 send_command.py localhost device_12345678 reboot")
        print('  python3 send_command.py localhost device_12345678 update_config \'{"interval": 60}\'')
        sys.exit(1)
        
    broker_host = sys.argv[1]
    device_id = sys.argv[2]
    operation = sys.argv[3]
    args = json.loads(sys.argv[4]) if len(sys.argv) > 4 else None
    
    send_rtk_command(broker_host, device_id, operation, args)
```

### 常用命令範例
```bash
# 獲取系統資訊
python3 send_command.py localhost device_12345678 get_system_info

# 重新啟動設備
python3 send_command.py localhost device_12345678 reboot

# 更新配置
python3 send_command.py localhost device_12345678 update_config '{"sampling_interval": 60, "debug_mode": true}'

# 運行診斷
python3 send_command.py localhost device_12345678 run_diagnostics '{"test_type": "connectivity"}'
```

## 步驟5: 使用RTK Controller

### 下載並編譯Controller
```bash
# 取得代碼
git clone <rtk_controller_repo>
cd rtk_controller

# 編譯
make build

# 配置
cp configs/controller.yaml.example configs/controller.yaml
```

### 配置Controller
```yaml
# configs/controller.yaml
mqtt:
  broker_host: "localhost"
  broker_port: 1883
  client_id: "rtk_controller"
  
database:
  type: "buntdb"
  path: "./data"
  
diagnostics:
  enabled: true
  interval_s: 300
  
qos:
  enabled: true
  analysis_interval_s: 60
```

### 啟動Controller
```bash
# 啟動CLI模式
./build_dir/rtk_controller --cli

# 在CLI中執行命令
RTK> device list
RTK> topology show
RTK> diagnostic run speed_test device_12345678
```

## 步驟6: 開發自定義設備

### 設備開發模板

建立檔案 `custom_device_template.py`:

```python
#!/usr/bin/env python3
import paho.mqtt.client as mqtt
import json
import time
from abc import ABC, abstractmethod

class RTKDeviceBase(ABC):
    """RTK設備基礎類別"""
    
    def __init__(self, device_id, device_type, tenant="demo", site="custom"):
        self.device_id = device_id
        self.device_type = device_type
        self.tenant = tenant
        self.site = site
        
        # MQTT設置
        self.client = mqtt.Client(client_id=device_id)
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        
        # 設備狀態
        self.connected = False
        self.start_time = time.time()
        
    def on_connect(self, client, userdata, flags, rc):
        self.connected = (rc == 0)
        if self.connected:
            print(f"Device {self.device_id} connected")
            cmd_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/req"
            client.subscribe(cmd_topic, qos=1)
            self.publish_attributes()
            
    def on_message(self, client, userdata, msg):
        try:
            command = json.loads(msg.payload.decode())
            self.handle_command(command)
        except Exception as e:
            print(f"Command processing error: {e}")
            
    def handle_command(self, command):
        cmd_id = command.get("id")
        operation = command.get("op")
        args = command.get("args", {})
        
        # 發送確認
        self.send_ack(cmd_id)
        
        # 委派給子類處理
        result = self.process_command(operation, args)
        
        # 發送結果
        self.send_result(cmd_id, operation, result)
        
    @abstractmethod
    def process_command(self, operation, args):
        """子類必須實作的命令處理方法"""
        pass
        
    @abstractmethod
    def get_device_attributes(self):
        """子類必須實作的屬性獲取方法"""
        pass
        
    @abstractmethod
    def get_device_state(self):
        """子類必須實作的狀態獲取方法"""
        pass
        
    @abstractmethod
    def get_telemetry_data(self):
        """子類必須實作的遙測數據獲取方法"""
        pass
        
    def send_ack(self, cmd_id):
        ack_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/ack"
        ack_msg = {
            "id": cmd_id,
            "schema": "cmd.ack/1.0",
            "status": "accepted",
            "ts": int(time.time() * 1000)
        }
        self.client.publish(ack_topic, json.dumps(ack_msg), qos=1)
        
    def send_result(self, cmd_id, operation, result):
        res_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/res"
        status = "completed" if "error" not in result else "failed"
        res_msg = {
            "id": cmd_id,
            "schema": f"cmd.{operation}.result/1.0",
            "status": status,
            "result": result,
            "ts": int(time.time() * 1000)
        }
        self.client.publish(res_topic, json.dumps(res_msg), qos=1)
        
    def publish_attributes(self):
        attr_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/attr"
        attr_msg = {
            "schema": "attr/1.0",
            "ts": int(time.time() * 1000),
            **self.get_device_attributes()
        }
        self.client.publish(attr_topic, json.dumps(attr_msg), qos=1, retain=True)
        
    def publish_state(self):
        state_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/state"
        state_msg = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            **self.get_device_state()
        }
        self.client.publish(state_topic, json.dumps(state_msg), qos=1, retain=True)
        
    def publish_telemetry(self, metric_type="system"):
        telemetry_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/telemetry/{metric_type}"
        telemetry_msg = {
            "schema": f"telemetry.{metric_type}/1.0",
            "ts": int(time.time() * 1000),
            **self.get_telemetry_data()
        }
        self.client.publish(telemetry_topic, json.dumps(telemetry_msg), qos=0)
        
    def publish_event(self, event_type, event_data):
        evt_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/evt/{event_type}"
        evt_msg = {
            "schema": f"evt.{event_type}/1.0",
            "ts": int(time.time() * 1000),
            **event_data
        }
        self.client.publish(evt_topic, json.dumps(evt_msg), qos=1)
        
    def start(self, broker_host="localhost", broker_port=1883):
        print(f"Starting {self.device_type} device: {self.device_id}")
        self.client.connect(broker_host, broker_port, 60)
        self.client.loop_start()
        
        try:
            while True:
                if self.connected:
                    self.publish_state()
                    self.publish_telemetry()
                time.sleep(30)
        except KeyboardInterrupt:
            print("Shutting down device...")
            self.client.disconnect()

# 自定義設備實作範例
class CustomSensorDevice(RTKDeviceBase):
    def __init__(self, device_id):
        super().__init__(device_id, "custom_sensor")
        self.sensor_value = 0.0
        
    def get_device_attributes(self):
        return {
            "device_type": "custom_sensor",
            "manufacturer": "Custom Corp",
            "model": "CS-1000",
            "capabilities": ["temperature", "humidity", "remote_control"]
        }
        
    def get_device_state(self):
        return {
            "health": "ok",
            "uptime_s": int(time.time() - self.start_time),
            "sensor_status": "active"
        }
        
    def get_telemetry_data(self):
        import random
        self.sensor_value = random.uniform(20.0, 30.0)
        return {
            "temperature_c": self.sensor_value,
            "humidity_percent": random.uniform(40.0, 60.0),
            "battery_level": random.uniform(80.0, 100.0)
        }
        
    def process_command(self, operation, args):
        if operation == "get_system_info":
            return self.get_device_attributes()
        elif operation == "calibrate_sensor":
            offset = args.get("offset", 0.0)
            self.sensor_value += offset
            return {"calibration_applied": True, "offset": offset}
        else:
            return {"error": "Unsupported command"}

if __name__ == "__main__":
    device = CustomSensorDevice("custom_sensor_001")
    device.start()
```

## 故障排除

### 常見問題

1. **無法連接到MQTT Broker**
   ```bash
   # 檢查broker是否運行
   netstat -ln | grep 1883
   
   # 測試連接
   mosquitto_pub -h localhost -t test -m "test"
   ```

2. **設備訊息格式錯誤**
   ```bash
   # 檢查JSON格式
   echo '{"test": "data"}' | python3 -m json.tool
   
   # 驗證Schema
   # 參考: docs/developers/core/SCHEMA_REFERENCE.md
   ```

3. **命令無響應**
   ```bash
   # 檢查主題訂閱
   mosquitto_sub -h localhost -t "rtk/v1/+/+/+/cmd/#" -v
   
   # 檢查設備日誌
   # 確認設備有正確訂閱cmd/req主題
   ```

### 除錯技巧

1. **啟用詳細日誌**
   ```python
   import logging
   logging.basicConfig(level=logging.DEBUG)
   ```

2. **監控MQTT流量**
   ```bash
   # 監控所有訊息
   mosquitto_sub -h localhost -t "#" -v
   
   # 監控特定設備
   mosquitto_sub -h localhost -t "rtk/v1/demo/quickstart/+/#" -v
   ```

3. **驗證訊息Schema**
   - 參考 [Schema Reference](../core/SCHEMA_REFERENCE.md)
   - 使用JSON Schema驗證工具

## 下一步

完成快速入門後，建議：

1. **深入學習協議**
   - [MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
   - [Commands and Events Reference](../core/COMMANDS_EVENTS_REFERENCE.md)

2. **設備特定整合**
   - [AP/Router Integration](../devices/AP_ROUTER_INTEGRATION.md)
   - [IoT Device Integration](../devices/IOT_DEVICE_INTEGRATION.md)

3. **進階功能**
   - [Testing Integration Guide](TESTING_INTEGRATION.md)
   - [Deployment Guide](DEPLOYMENT_GUIDE.md)

4. **工具和診斷**
   - [Diagnostics Tools](../diagnostics/)
   - [Support Tools](../tools/)

## 參考資源

- [RTK MQTT Repository](https://github.com/your-org/rtk-mqtt)
- [MQTT.org](https://mqtt.org/)
- [Paho MQTT Clients](https://www.eclipse.org/paho/)
- [Mosquitto Broker](https://mosquitto.org/)
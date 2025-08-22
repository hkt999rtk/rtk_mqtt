# RTK MQTT 測試整合指南

## 概述

本指南提供RTK MQTT系統的完整測試策略，包含單元測試、整合測試、負載測試和驗收測試的實作方法和最佳實務。

📋 **JSON Schema 參考**: 測試中使用的所有 MQTT 訊息格式必須符合 [`docs/spec/schemas/`](../../spec/schemas/) 中定義的 JSON Schema 規範。

## 測試架構

### 測試層級

```
┌─────────────────────────────────────┐
│           驗收測試                    │
│   (End-to-End / User Acceptance)    │
├─────────────────────────────────────┤
│           整合測試                    │
│    (Integration / System Tests)     │
├─────────────────────────────────────┤
│           單元測試                    │
│       (Unit / Component Tests)      │
└─────────────────────────────────────┘
```

### 測試環境

#### 開發環境 (Development)
- 本地MQTT broker
- 模擬設備
- 單元測試套件

#### 測試環境 (Testing)  
- 專用MQTT broker
- 設備模擬器集群
- 自動化測試管道

#### 預生產環境 (Staging)
- 生產等級MQTT broker
- 真實設備子集
- 性能和負載測試

## 單元測試

### MQTT訊息格式測試

```python
# test_mqtt_messages.py
import unittest
import json
import jsonschema
from datetime import datetime

class TestMQTTMessageFormats(unittest.TestCase):
    
    def setUp(self):
        # 載入JSON Schema
        with open('schemas/state_schema.json', 'r') as f:
            self.state_schema = json.load(f)
        with open('schemas/telemetry_schema.json', 'r') as f:
            self.telemetry_schema = json.load(f)
    
    def test_state_message_format(self):
        """測試狀態訊息格式"""
        state_msg = {
            "schema": "state/1.0",
            "ts": int(datetime.now().timestamp() * 1000),
            "health": "ok",
            "uptime_s": 3600,
            "connection_status": "connected"
        }
        
        # 驗證JSON Schema
        try:
            jsonschema.validate(state_msg, self.state_schema)
        except jsonschema.ValidationError as e:
            self.fail(f"State message validation failed: {e}")
    
    def test_invalid_state_message(self):
        """測試無效的狀態訊息"""
        invalid_msg = {
            "schema": "state/1.0",
            "ts": "invalid_timestamp",  # 應該是數字
            "health": "unknown_status"   # 不在enum中
        }
        
        with self.assertRaises(jsonschema.ValidationError):
            jsonschema.validate(invalid_msg, self.state_schema)
    
    def test_telemetry_message_format(self):
        """測試遙測訊息格式"""
        telemetry_msg = {
            "schema": "telemetry.network/1.0",
            "ts": int(datetime.now().timestamp() * 1000),
            "interface": "eth0",
            "tx_bytes": 1048576,
            "rx_bytes": 2097152,
            "latency_ms": 15.5
        }
        
        try:
            jsonschema.validate(telemetry_msg, self.telemetry_schema)
        except jsonschema.ValidationError as e:
            self.fail(f"Telemetry message validation failed: {e}")

class TestTopicStructure(unittest.TestCase):
    
    def test_valid_topic_format(self):
        """測試有效的主題格式"""
        valid_topics = [
            "rtk/v1/demo/office/device_001/state",
            "rtk/v1/company_a/site_1/aabbccddeeff/telemetry/cpu",
            "rtk/v1/tenant_123/location_main/sensor_001/evt/threshold"
        ]
        
        topic_pattern = r"rtk/v1/[a-zA-Z0-9_]+/[a-zA-Z0-9_]+/[a-zA-Z0-9:_-]+/(state|telemetry|evt|attr|cmd)(/.*)?$"
        import re
        
        for topic in valid_topics:
            self.assertTrue(re.match(topic_pattern, topic), 
                          f"Topic format invalid: {topic}")
    
    def test_invalid_topic_format(self):
        """測試無效的主題格式"""
        invalid_topics = [
            "rtk/v2/demo/office/device_001/state",  # 錯誤版本
            "mqtt/v1/demo/office/device_001/state", # 錯誤協議
            "rtk/v1/demo/device_001/state",         # 缺少site
            "rtk/v1/demo/office/device_001/invalid" # 無效訊息類型
        ]
        
        topic_pattern = r"rtk/v1/[a-zA-Z0-9_]+/[a-zA-Z0-9_]+/[a-zA-Z0-9:_-]+/(state|telemetry|evt|attr|cmd)(/.*)?$"
        import re
        
        for topic in invalid_topics:
            self.assertFalse(re.match(topic_pattern, topic),
                           f"Topic should be invalid: {topic}")
```

### 命令處理測試

```python
# test_command_processing.py
import unittest
from unittest.mock import Mock, patch
import json
import time

class TestCommandProcessing(unittest.TestCase):
    
    def setUp(self):
        from rtk_device import RTKDevice
        self.device = RTKDevice("test_device", "test_type")
        self.device.client = Mock()
    
    def test_command_ack_response(self):
        """測試命令確認響應"""
        command = {
            "id": "cmd-123456789",
            "op": "get_system_info",
            "schema": "cmd.get_system_info/1.0",
            "ts": int(time.time() * 1000)
        }
        
        # 模擬命令處理
        self.device.handle_command(command)
        
        # 驗證ACK被發送
        self.device.client.publish.assert_any_call(
            "rtk/v1/demo/test/test_device/cmd/ack",
            unittest.mock.ANY,
            qos=1
        )
        
        # 驗證ACK訊息格式
        ack_call = [call for call in self.device.client.publish.call_args_list 
                   if "cmd/ack" in call[0][0]][0]
        ack_msg = json.loads(ack_call[0][1])
        
        self.assertEqual(ack_msg["id"], "cmd-123456789")
        self.assertEqual(ack_msg["schema"], "cmd.ack/1.0")
        self.assertEqual(ack_msg["status"], "accepted")
    
    def test_command_result_response(self):
        """測試命令結果響應"""
        command = {
            "id": "cmd-123456789",
            "op": "get_system_info",
            "schema": "cmd.get_system_info/1.0"
        }
        
        self.device.handle_command(command)
        
        # 驗證結果被發送
        self.device.client.publish.assert_any_call(
            "rtk/v1/demo/test/test_device/cmd/res",
            unittest.mock.ANY,
            qos=1
        )
        
        # 驗證結果訊息格式
        res_call = [call for call in self.device.client.publish.call_args_list 
                   if "cmd/res" in call[0][0]][0]
        res_msg = json.loads(res_call[0][1])
        
        self.assertEqual(res_msg["id"], "cmd-123456789")
        self.assertEqual(res_msg["schema"], "cmd.get_system_info.result/1.0")
        self.assertIn(res_msg["status"], ["completed", "failed"])
        self.assertIn("result", res_msg)
    
    def test_unsupported_command(self):
        """測試不支援的命令"""
        command = {
            "id": "cmd-123456789",
            "op": "unsupported_operation",
            "schema": "cmd.unsupported_operation/1.0"
        }
        
        self.device.handle_command(command)
        
        # 驗證錯誤響應
        res_call = [call for call in self.device.client.publish.call_args_list 
                   if "cmd/res" in call[0][0]][0]
        res_msg = json.loads(res_call[0][1])
        
        self.assertEqual(res_msg["status"], "failed")
        self.assertIn("error", res_msg["result"])
```

## 整合測試

### MQTT通信測試

```python
# test_mqtt_integration.py
import unittest
import paho.mqtt.client as mqtt
import json
import time
import threading
from queue import Queue, Empty

class TestMQTTIntegration(unittest.TestCase):
    
    def setUp(self):
        self.broker_host = "localhost"
        self.broker_port = 1883
        self.message_queue = Queue()
        
        # 建立測試用MQTT客戶端
        self.test_client = mqtt.Client("test_client")
        self.test_client.on_message = self.on_test_message
        self.test_client.connect(self.broker_host, self.broker_port, 60)
        self.test_client.loop_start()
        
    def tearDown(self):
        self.test_client.loop_stop()
        self.test_client.disconnect()
    
    def on_test_message(self, client, userdata, msg):
        """測試客戶端訊息處理"""
        message = {
            "topic": msg.topic,
            "payload": msg.payload.decode(),
            "qos": msg.qos,
            "retain": msg.retain
        }
        self.message_queue.put(message)
    
    def wait_for_message(self, timeout=5):
        """等待MQTT訊息"""
        try:
            return self.message_queue.get(timeout=timeout)
        except Empty:
            return None
    
    def test_device_state_publication(self):
        """測試設備狀態發布"""
        # 訂閱狀態主題
        state_topic = "rtk/v1/test/integration/test_device/state"
        self.test_client.subscribe(state_topic, qos=1)
        
        # 啟動測試設備
        from test_device import TestDevice
        device = TestDevice("test_device")
        device.start_async(self.broker_host)
        
        # 等待狀態訊息
        message = self.wait_for_message(timeout=10)
        self.assertIsNotNone(message, "No state message received")
        self.assertEqual(message["topic"], state_topic)
        
        # 驗證狀態訊息內容
        state_data = json.loads(message["payload"])
        self.assertEqual(state_data["schema"], "state/1.0")
        self.assertIn("health", state_data)
        self.assertIn("ts", state_data)
        
        device.stop()
    
    def test_command_execution_flow(self):
        """測試完整命令執行流程"""
        device_id = "test_cmd_device"
        
        # 訂閱命令響應主題
        ack_topic = f"rtk/v1/test/integration/{device_id}/cmd/ack"
        res_topic = f"rtk/v1/test/integration/{device_id}/cmd/res"
        
        self.test_client.subscribe(ack_topic, qos=1)
        self.test_client.subscribe(res_topic, qos=1)
        
        # 啟動測試設備
        from test_device import TestDevice
        device = TestDevice(device_id)
        device.start_async(self.broker_host)
        
        time.sleep(2)  # 等待設備準備就緒
        
        # 發送命令
        cmd_topic = f"rtk/v1/test/integration/{device_id}/cmd/req"
        command = {
            "id": "test-cmd-123",
            "op": "get_system_info",
            "schema": "cmd.get_system_info/1.0",
            "ts": int(time.time() * 1000)
        }
        
        self.test_client.publish(cmd_topic, json.dumps(command), qos=1)
        
        # 等待ACK
        ack_msg = self.wait_for_message(timeout=5)
        self.assertIsNotNone(ack_msg, "No ACK received")
        self.assertEqual(ack_msg["topic"], ack_topic)
        
        ack_data = json.loads(ack_msg["payload"])
        self.assertEqual(ack_data["id"], "test-cmd-123")
        self.assertEqual(ack_data["status"], "accepted")
        
        # 等待結果
        res_msg = self.wait_for_message(timeout=10)
        self.assertIsNotNone(res_msg, "No result received")
        self.assertEqual(res_msg["topic"], res_topic)
        
        res_data = json.loads(res_msg["payload"])
        self.assertEqual(res_data["id"], "test-cmd-123")
        self.assertEqual(res_data["status"], "completed")
        self.assertIn("result", res_data)
        
        device.stop()
    
    def test_qos_and_retain_flags(self):
        """測試QoS和Retain標誌"""
        device_id = "test_qos_device"
        state_topic = f"rtk/v1/test/integration/{device_id}/state"
        telemetry_topic = f"rtk/v1/test/integration/{device_id}/telemetry/system"
        
        self.test_client.subscribe(state_topic, qos=1)
        self.test_client.subscribe(telemetry_topic, qos=1)
        
        # 啟動設備
        from test_device import TestDevice
        device = TestDevice(device_id)
        device.start_async(self.broker_host)
        
        # 檢查狀態訊息的Retain標誌
        state_msg = self.wait_for_message(timeout=10)
        self.assertTrue(state_msg["retain"], "State message should be retained")
        self.assertEqual(state_msg["qos"], 1, "State message should have QoS 1")
        
        # 檢查遙測訊息的Retain標誌
        telemetry_msg = self.wait_for_message(timeout=10)
        self.assertFalse(telemetry_msg["retain"], "Telemetry message should not be retained")
        
        device.stop()
```

### 設備模擬器

```python
# test_device.py
import paho.mqtt.client as mqtt
import json
import time
import threading
import random

class TestDevice:
    """測試用設備模擬器"""
    
    def __init__(self, device_id, tenant="test", site="integration"):
        self.device_id = device_id
        self.tenant = tenant
        self.site = site
        self.running = False
        
        self.client = mqtt.Client(client_id=device_id)
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        
        self.thread = None
        self.start_time = time.time()
        
    def on_connect(self, client, userdata, flags, rc):
        if rc == 0:
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
        
        # 發送ACK
        self.send_ack(cmd_id)
        
        # 處理命令
        if operation == "get_system_info":
            result = {
                "device_id": self.device_id,
                "uptime_s": int(time.time() - self.start_time),
                "version": "test-1.0.0",
                "capabilities": ["testing", "simulation"]
            }
        elif operation == "reboot":
            result = {"reboot_initiated": True}
            self.start_time = time.time()  # 重置運行時間
        else:
            result = {"error": "Unsupported command in test device"}
            
        # 發送結果
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
            "device_type": "test_device",
            "manufacturer": "Test Corp",
            "model": "TD-1000",
            "capabilities": ["testing", "simulation", "remote_control"]
        }
        self.client.publish(attr_topic, json.dumps(attr_msg), qos=1, retain=True)
        
    def publish_state(self):
        state_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/state"
        state_msg = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "health": "ok",
            "uptime_s": int(time.time() - self.start_time),
            "connection_status": "connected"
        }
        self.client.publish(state_topic, json.dumps(state_msg), qos=1, retain=True)
        
    def publish_telemetry(self):
        telemetry_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/telemetry/system"
        telemetry_msg = {
            "schema": "telemetry.system/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "payload": {
                "cpu_usage": random.uniform(10, 80),
                "memory_usage": random.uniform(20, 90),
                "temperature_celsius": random.uniform(30, 50)
            }
        }
        self.client.publish(telemetry_topic, json.dumps(telemetry_msg), qos=0)
        
    def run_loop(self):
        """設備主循環"""
        while self.running:
            self.publish_state()
            self.publish_telemetry()
            time.sleep(5)
            
    def start_async(self, broker_host="localhost", broker_port=1883):
        """非同步啟動設備"""
        self.running = True
        self.client.connect(broker_host, broker_port, 60)
        self.client.loop_start()
        
        self.thread = threading.Thread(target=self.run_loop)
        self.thread.daemon = True
        self.thread.start()
        
    def stop(self):
        """停止設備"""
        self.running = False
        if self.thread:
            self.thread.join(timeout=5)
        self.client.loop_stop()
        self.client.disconnect()
```

## 負載測試

### 多設備模擬

```python
# load_test.py
import threading
import time
import json
from test_device import TestDevice
import random

class LoadTestSuite:
    """負載測試套件"""
    
    def __init__(self, broker_host="localhost"):
        self.broker_host = broker_host
        self.devices = []
        self.running = False
        
    def create_devices(self, count=100, device_prefix="load_test_device"):
        """創建多個測試設備"""
        for i in range(count):
            device_id = f"{device_prefix}_{i:04d}"
            device = TestDevice(device_id, tenant="load_test", site="performance")
            self.devices.append(device)
            
    def start_load_test(self, duration_minutes=10):
        """開始負載測試"""
        self.running = True
        
        print(f"Starting load test with {len(self.devices)} devices for {duration_minutes} minutes")
        
        # 啟動所有設備
        for device in self.devices:
            device.start_async(self.broker_host)
            time.sleep(0.1)  # 避免同時連接
            
        # 運行指定時間
        start_time = time.time()
        while time.time() - start_time < duration_minutes * 60 and self.running:
            self.send_random_commands()
            time.sleep(30)
            
        # 停止所有設備
        self.stop_load_test()
        
    def send_random_commands(self):
        """發送隨機命令到設備"""
        import paho.mqtt.client as mqtt
        
        command_client = mqtt.Client("load_test_commander")
        command_client.connect(self.broker_host, 1883, 60)
        
        commands = ["get_system_info", "reboot", "run_diagnostics"]
        
        # 向隨機設備發送命令
        for _ in range(10):  # 每次發送10個命令
            device = random.choice(self.devices)
            command = random.choice(commands)
            
            cmd_topic = f"rtk/v1/load_test/performance/{device.device_id}/cmd/req"
            cmd_msg = {
                "id": f"load-test-{int(time.time() * 1000)}-{random.randint(1000, 9999)}",
                "op": command,
                "schema": f"cmd.{command}/1.0",
                "ts": int(time.time() * 1000)
            }
            
            command_client.publish(cmd_topic, json.dumps(cmd_msg), qos=1)
            
        command_client.disconnect()
        
    def stop_load_test(self):
        """停止負載測試"""
        self.running = False
        
        print("Stopping load test...")
        for device in self.devices:
            device.stop()
            
        print("Load test completed")
        
    def run_connection_stress_test(self, device_count=500, connection_interval=0.05):
        """連接壓力測試"""
        print(f"Starting connection stress test with {device_count} devices")
        
        successful_connections = 0
        failed_connections = 0
        
        for i in range(device_count):
            try:
                device_id = f"stress_test_device_{i:04d}"
                device = TestDevice(device_id, tenant="stress_test", site="connections")
                device.start_async(self.broker_host)
                
                # 短暫等待確認連接
                time.sleep(0.1)
                successful_connections += 1
                
                # 立即斷開連接
                device.stop()
                
                if connection_interval > 0:
                    time.sleep(connection_interval)
                    
            except Exception as e:
                print(f"Connection failed for device {i}: {e}")
                failed_connections += 1
                
        print(f"Connection stress test completed:")
        print(f"  Successful: {successful_connections}")
        print(f"  Failed: {failed_connections}")
        print(f"  Success rate: {successful_connections / device_count * 100:.2f}%")

# 負載測試執行腳本
if __name__ == "__main__":
    import sys
    
    if len(sys.argv) < 2:
        print("Usage: python3 load_test.py <test_type> [options]")
        print("Test types:")
        print("  device_load <count> <duration_minutes>")
        print("  connection_stress <count> [interval]")
        sys.exit(1)
        
    test_type = sys.argv[1]
    
    load_test = LoadTestSuite()
    
    if test_type == "device_load":
        count = int(sys.argv[2]) if len(sys.argv) > 2 else 50
        duration = int(sys.argv[3]) if len(sys.argv) > 3 else 5
        
        load_test.create_devices(count)
        load_test.start_load_test(duration)
        
    elif test_type == "connection_stress":
        count = int(sys.argv[2]) if len(sys.argv) > 2 else 200
        interval = float(sys.argv[3]) if len(sys.argv) > 3 else 0.05
        
        load_test.run_connection_stress_test(count, interval)
        
    else:
        print(f"Unknown test type: {test_type}")
```

## 性能測試

### 延遲測試

```python
# latency_test.py
import paho.mqtt.client as mqtt
import json
import time
import statistics
from queue import Queue
import threading

class LatencyTest:
    """MQTT訊息延遲測試"""
    
    def __init__(self, broker_host="localhost"):
        self.broker_host = broker_host
        self.results = []
        self.response_queue = Queue()
        
    def test_command_response_latency(self, iterations=100):
        """測試命令響應延遲"""
        print(f"Testing command response latency ({iterations} iterations)...")
        
        # 設置響應監聽客戶端
        response_client = mqtt.Client("latency_test_listener")
        response_client.on_message = self.on_response_message
        response_client.connect(self.broker_host, 1883, 60)
        response_client.subscribe("rtk/v1/latency_test/+/+/cmd/res", qos=1)
        response_client.loop_start()
        
        # 設置命令發送客戶端
        command_client = mqtt.Client("latency_test_sender")
        command_client.connect(self.broker_host, 1883, 60)
        
        # 啟動測試設備
        from test_device import TestDevice
        test_device = TestDevice("latency_test_device", "latency_test", "performance")
        test_device.start_async(self.broker_host)
        
        time.sleep(2)  # 等待設備準備就緒
        
        latencies = []
        
        for i in range(iterations):
            cmd_id = f"latency-test-{i:04d}"
            start_time = time.time()
            
            # 發送命令
            cmd_topic = "rtk/v1/latency_test/performance/latency_test_device/cmd/req"
            command = {
                "id": cmd_id,
                "op": "get_system_info",
                "schema": "cmd.get_system_info/1.0",
                "ts": int(time.time() * 1000)
            }
            
            command_client.publish(cmd_topic, json.dumps(command), qos=1)
            
            # 等待響應
            try:
                response = self.response_queue.get(timeout=10)
                if response["id"] == cmd_id:
                    end_time = time.time()
                    latency_ms = (end_time - start_time) * 1000
                    latencies.append(latency_ms)
                    
            except Exception as e:
                print(f"Iteration {i} failed: {e}")
                
            time.sleep(0.1)  # 避免過於頻繁
            
        # 清理
        test_device.stop()
        response_client.loop_stop()
        response_client.disconnect()
        command_client.disconnect()
        
        # 計算統計
        if latencies:
            print(f"Latency Statistics (ms):")
            print(f"  Min: {min(latencies):.2f}")
            print(f"  Max: {max(latencies):.2f}")
            print(f"  Mean: {statistics.mean(latencies):.2f}")
            print(f"  Median: {statistics.median(latencies):.2f}")
            print(f"  95th percentile: {statistics.quantiles(latencies, n=20)[18]:.2f}")
            
        return latencies
        
    def on_response_message(self, client, userdata, msg):
        """處理響應訊息"""
        try:
            response = json.loads(msg.payload.decode())
            self.response_queue.put(response)
        except Exception as e:
            print(f"Response processing error: {e}")
            
    def test_throughput(self, duration_seconds=60, message_rate_per_second=10):
        """測試訊息吞吐量"""
        print(f"Testing throughput for {duration_seconds}s at {message_rate_per_second} msg/s")
        
        # 設置監聽客戶端
        listener_client = mqtt.Client("throughput_listener")
        received_count = [0]  # 使用列表以便在nested function中修改
        
        def on_throughput_message(client, userdata, msg):
            received_count[0] += 1
            
        listener_client.on_message = on_throughput_message
        listener_client.connect(self.broker_host, 1883, 60)
        listener_client.subscribe("rtk/v1/throughput_test/+/+/telemetry/+", qos=0)
        listener_client.loop_start()
        
        # 設置發送客戶端
        sender_client = mqtt.Client("throughput_sender")
        sender_client.connect(self.broker_host, 1883, 60)
        
        # 開始發送訊息
        start_time = time.time()
        sent_count = 0
        interval = 1.0 / message_rate_per_second
        
        while time.time() - start_time < duration_seconds:
            telemetry_topic = f"rtk/v1/throughput_test/performance/sender/telemetry/system"
            telemetry_msg = {
                "schema": "telemetry.system/1.0",
                "ts": int(time.time() * 1000),
                "device_id": "sender",
                "payload": {
                    "message_id": sent_count,
                    "cpu_usage": 50.0,
                    "memory_usage": 60.0
                }
            }
            
            sender_client.publish(telemetry_topic, json.dumps(telemetry_msg), qos=0)
            sent_count += 1
            
            time.sleep(interval)
            
        # 等待最後的訊息
        time.sleep(2)
        
        # 清理
        listener_client.loop_stop()
        listener_client.disconnect()
        sender_client.disconnect()
        
        # 計算結果
        actual_duration = time.time() - start_time
        sent_rate = sent_count / actual_duration
        received_rate = received_count[0] / actual_duration
        loss_rate = (sent_count - received_count[0]) / sent_count * 100
        
        print(f"Throughput Results:")
        print(f"  Duration: {actual_duration:.2f}s")
        print(f"  Messages sent: {sent_count}")
        print(f"  Messages received: {received_count[0]}")
        print(f"  Send rate: {sent_rate:.2f} msg/s")
        print(f"  Receive rate: {received_rate:.2f} msg/s")
        print(f"  Loss rate: {loss_rate:.2f}%")
        
        return {
            "sent_count": sent_count,
            "received_count": received_count[0],
            "duration": actual_duration,
            "send_rate": sent_rate,
            "receive_rate": received_rate,
            "loss_rate": loss_rate
        }

# 性能測試執行腳本
if __name__ == "__main__":
    import sys
    
    test = LatencyTest()
    
    if len(sys.argv) > 1 and sys.argv[1] == "throughput":
        duration = int(sys.argv[2]) if len(sys.argv) > 2 else 60
        rate = int(sys.argv[3]) if len(sys.argv) > 3 else 10
        test.test_throughput(duration, rate)
    else:
        iterations = int(sys.argv[1]) if len(sys.argv) > 1 else 100
        test.test_command_response_latency(iterations)
```

## 自動化測試管道

### GitHub Actions配置

```yaml
# .github/workflows/rtk_mqtt_tests.yml
name: RTK MQTT Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Python
      uses: actions/setup-python@v3
      with:
        python-version: '3.9'
        
    - name: Install dependencies
      run: |
        pip install -r requirements.txt
        pip install pytest pytest-cov jsonschema
        
    - name: Run unit tests
      run: |
        pytest tests/unit/ -v --cov=rtk_mqtt --cov-report=xml
        
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.xml

  integration-tests:
    runs-on: ubuntu-latest
    
    services:
      mosquitto:
        image: eclipse-mosquitto:2.0
        ports:
          - 1883:1883
        options: >-
          --health-cmd "mosquitto_pub -h localhost -t test -m test"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
          
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Python
      uses: actions/setup-python@v3
      with:
        python-version: '3.9'
        
    - name: Install dependencies
      run: |
        pip install -r requirements.txt
        pip install pytest
        
    - name: Wait for Mosquitto
      run: |
        sleep 10
        mosquitto_pub -h localhost -t test -m "test connection"
        
    - name: Run integration tests
      run: |
        pytest tests/integration/ -v
        
  load-tests:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    
    services:
      mosquitto:
        image: eclipse-mosquitto:2.0
        ports:
          - 1883:1883
          
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Python
      uses: actions/setup-python@v3
      with:
        python-version: '3.9'
        
    - name: Install dependencies
      run: |
        pip install -r requirements.txt
        
    - name: Run load tests
      run: |
        python3 tests/load/load_test.py device_load 50 2
        python3 tests/load/load_test.py connection_stress 100 0.1
        
    - name: Run performance tests
      run: |
        python3 tests/performance/latency_test.py 50
        python3 tests/performance/latency_test.py throughput 30 5
```

### Makefile測試目標

```makefile
# Makefile
.PHONY: test test-unit test-integration test-load test-performance

# 運行所有測試
test: test-unit test-integration

# 單元測試
test-unit:
	pytest tests/unit/ -v --cov=rtk_mqtt --cov-report=html
	@echo "Unit test coverage report: htmlcov/index.html"

# 整合測試
test-integration:
	pytest tests/integration/ -v

# 負載測試
test-load:
	python3 tests/load/load_test.py device_load 100 5
	python3 tests/load/load_test.py connection_stress 200 0.05

# 性能測試
test-performance:
	python3 tests/performance/latency_test.py 100
	python3 tests/performance/latency_test.py throughput 60 20

# 設置測試環境
setup-test-env:
	docker run -d --name test-mosquitto -p 1883:1883 eclipse-mosquitto:2.0
	sleep 5
	mosquitto_pub -h localhost -t test -m "setup complete"

# 清理測試環境
cleanup-test-env:
	docker stop test-mosquitto || true
	docker rm test-mosquitto || true

# 生成測試報告
test-report:
	pytest tests/ --html=test_report.html --self-contained-html
	@echo "Test report: test_report.html"
```

## 測試最佳實務

### 1. 測試隔離
- 每個測試使用獨立的MQTT主題
- 測試後清理資源
- 避免測試間的依賴關係

### 2. 模擬和存根
- 使用Mock物件模擬外部依賴
- 建立可重複的測試環境
- 隔離被測試的組件

### 3. 測試數據管理
- 使用測試專用的配置
- 避免使用生產數據
- 實作測試數據工廠

### 4. 持續整合
- 每次提交都執行測試
- 設置測試覆蓋率閾值
- 自動化測試環境部署

## 故障排除

### 常見測試問題

1. **MQTT連接失敗**
   ```bash
   # 檢查broker狀態
   docker ps | grep mosquitto
   netstat -ln | grep 1883
   ```

2. **測試超時**
   - 增加超時時間
   - 檢查網路延遲
   - 優化測試邏輯

3. **測試不穩定**
   - 增加重試機制
   - 改善測試同步
   - 減少測試依賴

### 除錯技巧

```python
# 啟用MQTT客戶端除錯
import paho.mqtt.client as mqtt
mqtt.Client.enable_logger = True

# 測試中加入診斷訊息
import logging
logging.basicConfig(level=logging.DEBUG)
```

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Quick Start Guide](QUICK_START_GUIDE.md)
- [Deployment Guide](DEPLOYMENT_GUIDE.md)
- [pytest Documentation](https://docs.pytest.org/)
- [Paho MQTT Testing](https://github.com/eclipse/paho.mqtt.python)
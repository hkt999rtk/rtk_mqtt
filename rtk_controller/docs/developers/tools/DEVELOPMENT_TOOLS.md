# RTK MQTT 開發工具指南

## 概述

本文檔提供 RTK MQTT 開發過程中的完整工具鏈，包含測試工具、除錯工具、效能分析工具和 IDE 整合，幫助開發者提升開發效率和代碼品質。

## 🛠️ 核心開發工具

### 1. RTK MQTT CLI 工具

#### 安裝
```bash
# 從源碼編譯
cd rtk_controller
make build
./build_dir/rtk_controller --cli

# 或下載預編譯版本
wget https://releases.rtk-mqtt.com/latest/rtk_controller-linux-amd64
chmod +x rtk_controller-linux-amd64
```

#### 基本命令
```bash
# 啟動互動式 CLI
./rtk_controller --cli

# 連接到 MQTT broker
rtk> connect --broker localhost:1883 --username admin --password secret

# 監聽所有訊息
rtk> monitor --pattern "rtk/v1/+/+/+/+"

# 發送測試命令
rtk> send-command --device aabbccddeeff --operation speed_test --args '{"duration":30}'

# 查看設備狀態
rtk> device-status --device aabbccddeeff

# 檢視拓撲
rtk> topology --site main --format json
```

#### 進階功能
```bash
# 批次執行腳本
rtk> script --file test_scenarios.rtk

# 匯出數據
rtk> export --format csv --output device_metrics.csv --time-range "last 24h"

# 性能測試
rtk> benchmark --devices 100 --duration 300 --operations speed_test,wifi_scan
```

### 2. MQTT 測試框架

#### 消息驗證器
```python
#!/usr/bin/env python3
# tools/rtk_message_validator.py

import paho.mqtt.client as mqtt
import json
import jsonschema
import argparse
import sys
from pathlib import Path

class RTKMessageValidator:
    def __init__(self, schema_dir="docs/spec/schemas"):
        self.schema_dir = Path(schema_dir)
        self.schemas = self._load_schemas()
        self.stats = {
            'total': 0, 'valid': 0, 'invalid': 0,
            'by_type': {}, 'errors': []
        }
    
    def _load_schemas(self):
        schemas = {}
        for schema_file in self.schema_dir.glob("*.json"):
            with open(schema_file) as f:
                schema = json.load(f)
                schemas[schema_file.stem] = schema
        return schemas
    
    def validate_message(self, topic, payload):
        self.stats['total'] += 1
        
        try:
            # 解析 JSON
            message = json.loads(payload)
            schema_name = message.get('schema', '').split('/')[0]
            
            # 統計消息類型
            msg_type = topic.split('/')[-1]
            self.stats['by_type'][msg_type] = self.stats['by_type'].get(msg_type, 0) + 1
            
            # Schema 驗證
            if schema_name in self.schemas:
                jsonschema.validate(message, self.schemas[schema_name])
                self.stats['valid'] += 1
                return True, None
            else:
                error = f"Unknown schema: {schema_name}"
                self.stats['errors'].append(error)
                self.stats['invalid'] += 1
                return False, error
                
        except json.JSONDecodeError as e:
            error = f"Invalid JSON: {e}"
            self.stats['errors'].append(error)
            self.stats['invalid'] += 1
            return False, error
        except jsonschema.ValidationError as e:
            error = f"Schema validation failed: {e.message}"
            self.stats['errors'].append(error)
            self.stats['invalid'] += 1
            return False, error
    
    def on_message(self, client, userdata, msg):
        is_valid, error = self.validate_message(msg.topic, msg.payload.decode())
        
        status = "✅ VALID" if is_valid else "❌ INVALID"
        print(f"{status} | {msg.topic} | {len(msg.payload)} bytes")
        
        if error:
            print(f"  Error: {error}")
    
    def print_stats(self):
        print("\n📊 Validation Statistics:")
        print(f"Total messages: {self.stats['total']}")
        print(f"Valid: {self.stats['valid']} ({self.stats['valid']/self.stats['total']*100:.1f}%)")
        print(f"Invalid: {self.stats['invalid']} ({self.stats['invalid']/self.stats['total']*100:.1f}%)")
        
        print("\n📈 By message type:")
        for msg_type, count in self.stats['by_type'].items():
            print(f"  {msg_type}: {count}")

def main():
    parser = argparse.ArgumentParser(description="RTK MQTT Message Validator")
    parser.add_argument("--broker", default="localhost", help="MQTT broker host")
    parser.add_argument("--port", type=int, default=1883, help="MQTT broker port")
    parser.add_argument("--topic", default="rtk/v1/+/+/+/+", help="Topic pattern to monitor")
    parser.add_argument("--schemas", default="docs/spec/schemas", help="Schema directory")
    
    args = parser.parse_args()
    
    validator = RTKMessageValidator(args.schemas)
    
    client = mqtt.Client("rtk_validator")
    client.on_message = validator.on_message
    
    try:
        client.connect(args.broker, args.port, 60)
        client.subscribe(args.topic)
        
        print(f"🔍 Monitoring {args.topic} on {args.broker}:{args.port}")
        print("Press Ctrl+C to stop\n")
        
        client.loop_forever()
        
    except KeyboardInterrupt:
        validator.print_stats()
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
```

#### 負載測試工具
```python
#!/usr/bin/env python3
# tools/rtk_load_tester.py

import asyncio
import aiofiles
import json
import time
import random
from concurrent.futures import ThreadPoolExecutor
import paho.mqtt.client as mqtt
import threading
import argparse

class RTKLoadTester:
    def __init__(self, broker_host, broker_port, num_devices=10):
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.num_devices = num_devices
        self.clients = []
        self.stats = {
            'messages_sent': 0,
            'messages_received': 0,
            'errors': 0,
            'start_time': None,
            'latencies': []
        }
        
    def create_device_client(self, device_id):
        client = mqtt.Client(f"load-test-{device_id}")
        client.user_data_set({
            'device_id': device_id,
            'stats': self.stats
        })
        
        client.on_connect = self._on_connect
        client.on_message = self._on_message
        client.on_publish = self._on_publish
        
        return client
    
    def _on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            device_id = userdata['device_id']
            # 訂閱命令主題
            cmd_topic = f"rtk/v1/test/load/{device_id}/cmd/req"
            client.subscribe(cmd_topic)
            print(f"Device {device_id} connected")
        else:
            self.stats['errors'] += 1
    
    def _on_message(self, client, userdata, msg):
        self.stats['messages_received'] += 1
        
        # 模擬命令處理
        try:
            command = json.loads(msg.payload.decode())
            self._handle_command(client, userdata, command)
        except Exception as e:
            self.stats['errors'] += 1
    
    def _on_publish(self, client, userdata, mid):
        self.stats['messages_sent'] += 1
    
    def _handle_command(self, client, userdata, command):
        device_id = userdata['device_id']
        payload = command.get('payload', {})
        cmd_id = payload.get('id')
        operation = payload.get('op')
        
        # 發送 ACK
        ack_topic = f"rtk/v1/test/load/{device_id}/cmd/ack"
        ack_payload = {
            "schema": "cmd.ack/1.0",
            "ts": int(time.time() * 1000),
            "device_id": device_id,
            "payload": {"id": cmd_id, "status": "received"}
        }
        client.publish(ack_topic, json.dumps(ack_payload))
        
        # 模擬命令執行時間
        time.sleep(random.uniform(0.1, 2.0))
        
        # 發送結果
        res_topic = f"rtk/v1/test/load/{device_id}/cmd/res"
        res_payload = {
            "schema": "cmd.result/1.0",
            "ts": int(time.time() * 1000),
            "device_id": device_id,
            "payload": {
                "id": cmd_id,
                "status": "completed",
                "result": {"test": "success"}
            }
        }
        client.publish(res_topic, json.dumps(res_payload))
    
    def start_load_test(self, duration=300):
        print(f"🚀 Starting load test with {self.num_devices} devices for {duration}s")
        
        # 創建設備客戶端
        for i in range(self.num_devices):
            device_id = f"loadtest{i:04d}"
            client = self.create_device_client(device_id)
            client.connect(self.broker_host, self.broker_port, 60)
            client.loop_start()
            self.clients.append(client)
        
        # 等待連接建立
        time.sleep(2)
        
        self.stats['start_time'] = time.time()
        
        # 定期發送狀態消息
        def send_status_messages():
            while time.time() - self.stats['start_time'] < duration:
                for i, client in enumerate(self.clients):
                    device_id = f"loadtest{i:04d}"
                    
                    # 發送狀態
                    state_topic = f"rtk/v1/test/load/{device_id}/state"
                    state_payload = {
                        "schema": "state/1.0",
                        "ts": int(time.time() * 1000),
                        "health": "ok",
                        "cpu_usage": random.uniform(10, 80),
                        "memory_usage": random.uniform(20, 90)
                    }
                    client.publish(state_topic, json.dumps(state_payload))
                    
                    # 發送遙測
                    telemetry_topic = f"rtk/v1/test/load/{device_id}/telemetry/cpu_usage"
                    telemetry_payload = {
                        "schema": "telemetry.cpu_usage/1.0",
                        "ts": int(time.time() * 1000),
                        "value": random.uniform(10, 80),
                        "unit": "%"
                    }
                    client.publish(telemetry_topic, json.dumps(telemetry_payload))
                
                time.sleep(5)  # 每5秒發送一次
        
        # 啟動狀態發送線程
        status_thread = threading.Thread(target=send_status_messages)
        status_thread.start()
        
        # 運行測試
        time.sleep(duration)
        
        # 清理
        for client in self.clients:
            client.loop_stop()
            client.disconnect()
        
        status_thread.join()
        self._print_results()
    
    def _print_results(self):
        duration = time.time() - self.stats['start_time']
        
        print(f"\n📊 Load Test Results ({duration:.1f}s)")
        print(f"Devices: {self.num_devices}")
        print(f"Messages sent: {self.stats['messages_sent']}")
        print(f"Messages received: {self.stats['messages_received']}")
        print(f"Errors: {self.stats['errors']}")
        print(f"Messages/second: {self.stats['messages_sent']/duration:.1f}")
        print(f"Error rate: {self.stats['errors']/self.stats['messages_sent']*100:.2f}%")

def main():
    parser = argparse.ArgumentParser(description="RTK MQTT Load Tester")
    parser.add_argument("--broker", default="localhost", help="MQTT broker host")
    parser.add_argument("--port", type=int, default=1883, help="MQTT broker port")
    parser.add_argument("--devices", type=int, default=10, help="Number of simulated devices")
    parser.add_argument("--duration", type=int, default=300, help="Test duration in seconds")
    
    args = parser.parse_args()
    
    tester = RTKLoadTester(args.broker, args.port, args.devices)
    tester.start_load_test(args.duration)

if __name__ == "__main__":
    main()
```

### 3. 協議分析工具

#### 流量分析器
```bash
#!/bin/bash
# tools/rtk_traffic_analyzer.sh

set -e

BROKER_HOST=${1:-localhost}
BROKER_PORT=${2:-1883}
DURATION=${3:-60}

echo "🔍 RTK MQTT Traffic Analyzer"
echo "Broker: $BROKER_HOST:$BROKER_PORT"
echo "Duration: ${DURATION}s"
echo ""

# 創建臨時文件
TEMP_FILE=$(mktemp)
trap "rm -f $TEMP_FILE" EXIT

# 使用 mosquitto_sub 收集數據
mosquitto_sub -h $BROKER_HOST -p $BROKER_PORT -t "rtk/v1/+/+/+/+" -v > $TEMP_FILE &
SUB_PID=$!

# 運行指定時間
sleep $DURATION

# 停止收集
kill $SUB_PID 2>/dev/null || true

# 分析數據
echo "📊 Traffic Analysis Results:"
echo ""

# 總消息數
TOTAL_MESSAGES=$(wc -l < $TEMP_FILE)
echo "Total messages: $TOTAL_MESSAGES"

# 按消息類型統計
echo ""
echo "📈 Messages by type:"
awk '{print $1}' $TEMP_FILE | awk -F'/' '{print $6}' | sort | uniq -c | sort -nr

# 按設備統計
echo ""
echo "📱 Messages by device:"
awk '{print $1}' $TEMP_FILE | awk -F'/' '{print $5}' | sort | uniq -c | sort -nr | head -10

# 按租戶統計
echo ""
echo "🏢 Messages by tenant:"
awk '{print $1}' $TEMP_FILE | awk -F'/' '{print $3}' | sort | uniq -c | sort -nr

# 消息大小分析
echo ""
echo "📏 Message size analysis:"
awk '{$1=""; print length($0)}' $TEMP_FILE | sort -n | awk '
{
    sizes[NR] = $1
    sum += $1
}
END {
    print "Total messages: " NR
    print "Average size: " sum/NR " bytes"
    print "Min size: " sizes[1] " bytes"
    print "Max size: " sizes[NR] " bytes"
    print "Median size: " sizes[int(NR/2)] " bytes"
}'

# 頻率分析
echo ""
echo "⚡ Message frequency:"
echo "Messages per second: $(echo "scale=2; $TOTAL_MESSAGES / $DURATION" | bc)"
```

### 4. 除錯工具

#### MQTT 調試器
```python
#!/usr/bin/env python3
# tools/rtk_debugger.py

import paho.mqtt.client as mqtt
import json
import time
import argparse
from datetime import datetime
import re

class RTKDebugger:
    def __init__(self, broker_host, broker_port):
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.client = mqtt.Client("rtk_debugger")
        self.filters = []
        
        # 設置回調
        self.client.on_connect = self._on_connect
        self.client.on_message = self._on_message
        
    def add_filter(self, pattern):
        """添加消息過濾器"""
        self.filters.append(re.compile(pattern))
        
    def _should_display(self, topic, payload):
        """檢查是否應該顯示此消息"""
        if not self.filters:
            return True
            
        for filter_pattern in self.filters:
            if filter_pattern.search(topic) or filter_pattern.search(payload):
                return True
        return False
    
    def _on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            print(f"🔗 Connected to {self.broker_host}:{self.broker_port}")
            # 訂閱所有 RTK 消息
            client.subscribe("rtk/v1/+/+/+/+")
        else:
            print(f"❌ Connection failed: {rc}")
    
    def _on_message(self, client, userdata, msg):
        try:
            timestamp = datetime.now().strftime("%H:%M:%S.%f")[:-3]
            topic = msg.topic
            payload = msg.payload.decode()
            
            if not self._should_display(topic, payload):
                return
            
            # 解析主題
            topic_parts = topic.split('/')
            if len(topic_parts) >= 6:
                protocol, version, tenant, site, device_id, msg_type = topic_parts[:6]
                
                # 嘗試解析 JSON
                try:
                    message_data = json.loads(payload)
                    formatted_payload = json.dumps(message_data, indent=2)
                except json.JSONDecodeError:
                    formatted_payload = payload
                
                # 顯示消息
                print(f"\n📡 [{timestamp}] {msg_type.upper()}")
                print(f"   Device: {device_id} | Tenant: {tenant} | Site: {site}")
                print(f"   Topic: {topic}")
                print(f"   Payload ({len(payload)} bytes):")
                
                # 根據消息類型使用不同顏色
                if msg_type == "state":
                    print(f"   \033[92m{formatted_payload}\033[0m")  # 綠色
                elif msg_type.startswith("cmd"):
                    print(f"   \033[94m{formatted_payload}\033[0m")  # 藍色
                elif msg_type.startswith("evt"):
                    print(f"   \033[93m{formatted_payload}\033[0m")  # 黃色
                elif msg_type == "lwt":
                    print(f"   \033[91m{formatted_payload}\033[0m")  # 紅色
                else:
                    print(f"   {formatted_payload}")
                    
        except Exception as e:
            print(f"   ⚠️  Error processing message: {e}")
    
    def start(self, filter_patterns=None):
        """啟動調試器"""
        if filter_patterns:
            for pattern in filter_patterns:
                self.add_filter(pattern)
                
        print(f"🔍 RTK MQTT Debugger started")
        print(f"   Broker: {self.broker_host}:{self.broker_port}")
        
        if self.filters:
            print(f"   Filters: {[f.pattern for f in self.filters]}")
            
        print(f"   Press Ctrl+C to stop\n")
        
        try:
            self.client.connect(self.broker_host, self.broker_port, 60)
            self.client.loop_forever()
        except KeyboardInterrupt:
            print(f"\n👋 Debugger stopped")
        except Exception as e:
            print(f"❌ Error: {e}")

def main():
    parser = argparse.ArgumentParser(description="RTK MQTT Debugger")
    parser.add_argument("--broker", default="localhost", help="MQTT broker host")
    parser.add_argument("--port", type=int, default=1883, help="MQTT broker port")
    parser.add_argument("--filter", action="append", help="Message filter patterns")
    parser.add_argument("--device", help="Filter by specific device ID")
    parser.add_argument("--type", help="Filter by message type (state,cmd,evt,etc)")
    
    args = parser.parse_args()
    
    debugger = RTKDebugger(args.broker, args.port)
    
    # 添加過濾器
    filters = args.filter or []
    
    if args.device:
        filters.append(f"/{args.device}/")
        
    if args.type:
        filters.append(f"/{args.type}")
    
    debugger.start(filters)

if __name__ == "__main__":
    main()
```

## 🔧 IDE 整合

### VS Code 擴展配置

#### 設置檔案 (.vscode/settings.json)
```json
{
    "files.associations": {
        "*.rtk": "yaml",
        "*.mqtt": "json"
    },
    "yaml.schemas": {
        "./docs/spec/schemas/*.json": [
            "configs/*.yaml",
            "test/*.yaml"
        ]
    },
    "json.schemas": [
        {
            "fileMatch": ["**/test/**/*.json"],
            "url": "./docs/spec/schemas/base.json"
        }
    ],
    "editor.rulers": [80, 120],
    "python.defaultInterpreterPath": "./venv/bin/python",
    "go.toolsManagement.autoUpdate": true
}
```

#### 任務配置 (.vscode/tasks.json)
```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "RTK: Build Controller",
            "type": "shell",
            "command": "make",
            "args": ["build"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": []
        },
        {
            "label": "RTK: Run Tests",
            "type": "shell",
            "command": "make",
            "args": ["test"],
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        },
        {
            "label": "RTK: Start Debugger",
            "type": "shell",
            "command": "python3",
            "args": ["tools/rtk_debugger.py", "--broker", "localhost"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": true,
                "panel": "new"
            }
        },
        {
            "label": "RTK: Validate Messages",
            "type": "shell",
            "command": "python3",
            "args": ["tools/rtk_message_validator.py"],
            "group": "test"
        }
    ]
}
```

#### 啟動配置 (.vscode/launch.json)
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug RTK Controller",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/controller",
            "args": ["--cli", "--debug"],
            "env": {},
            "console": "integratedTerminal"
        },
        {
            "name": "Debug Python Device",
            "type": "python",
            "request": "launch",
            "program": "${workspaceFolder}/examples/python_device.py",
            "args": ["--broker", "localhost"],
            "console": "integratedTerminal"
        }
    ]
}
```

### JetBrains IDE 配置

#### IntelliJ IDEA / GoLand
```xml
<!-- .idea/runConfigurations/RTK_Controller.xml -->
<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="RTK Controller" type="GoApplicationRunConfiguration">
    <module name="rtk_controller" />
    <working_directory value="$PROJECT_DIR$" />
    <kind value="PACKAGE" />
    <filePath value="$PROJECT_DIR$/cmd/controller/main.go" />
    <package value="github.com/rtk-mqtt/rtk_controller/cmd/controller" />
    <directory value="$PROJECT_DIR$" />
    <parameters value="--cli --debug" />
    <method v="2" />
  </configuration>
</component>
```

## 📊 效能分析工具

### 記憶體分析
```bash
#!/bin/bash
# tools/memory_profiler.sh

echo "🧠 RTK MQTT Memory Profiler"

# Go 應用記憶體分析
if command -v go >/dev/null 2>&1; then
    echo "📊 Go Memory Profile:"
    go tool pprof -top http://localhost:6060/debug/pprof/heap
fi

# Python 記憶體分析
if command -v python3 >/dev/null 2>&1; then
    echo "🐍 Python Memory Analysis:"
    python3 -m memory_profiler examples/python_device.py
fi

# 系統記憶體監控
echo "💾 System Memory Usage:"
ps aux | grep -E "(rtk|mqtt)" | awk '{print $2, $4, $11}' | sort -k2 -nr
```

### CPU 效能分析
```bash
#!/bin/bash
# tools/cpu_profiler.sh

echo "⚡ RTK MQTT CPU Profiler"

# 找到 RTK 相關進程
RTK_PIDS=$(pgrep -f "rtk")

if [ -z "$RTK_PIDS" ]; then
    echo "❌ No RTK processes found"
    exit 1
fi

echo "🔍 Found RTK processes: $RTK_PIDS"

# 監控 CPU 使用率
echo "📊 CPU Usage (60s monitoring):"
for pid in $RTK_PIDS; do
    echo "Process $pid:"
    top -p $pid -b -n 60 -d 1 | grep $pid | awk '{print $9}' | \
    awk '{sum+=$1; count++} END {print "Average CPU:", sum/count "%"}'
done

# Go 程式 CPU 分析
if command -v go >/dev/null 2>&1; then
    echo "🔥 Go CPU Profile:"
    go tool pprof -top http://localhost:6060/debug/pprof/profile?seconds=30
fi
```

## 🧪 自動化測試工具

### 整合測試套件
```python
#!/usr/bin/env python3
# tools/integration_test_suite.py

import unittest
import time
import json
import subprocess
import tempfile
import os
from pathlib import Path

class RTKIntegrationTestSuite(unittest.TestCase):
    
    @classmethod
    def setUpClass(cls):
        """啟動測試環境"""
        cls.temp_dir = tempfile.mkdtemp()
        cls.broker_process = None
        cls.controller_process = None
        
        # 啟動 MQTT broker
        cls._start_mqtt_broker()
        time.sleep(2)
        
        # 啟動 RTK controller
        cls._start_rtk_controller()
        time.sleep(3)
    
    @classmethod
    def tearDownClass(cls):
        """清理測試環境"""
        if cls.controller_process:
            cls.controller_process.terminate()
        if cls.broker_process:
            cls.broker_process.terminate()
    
    @classmethod
    def _start_mqtt_broker(cls):
        """啟動 MQTT broker"""
        try:
            cls.broker_process = subprocess.Popen([
                "mosquitto", "-v"
            ], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        except FileNotFoundError:
            cls.fail("Mosquitto not found. Please install mosquitto.")
    
    @classmethod
    def _start_rtk_controller(cls):
        """啟動 RTK controller"""
        config_file = os.path.join(cls.temp_dir, "test_config.yaml")
        with open(config_file, 'w') as f:
            f.write("""
mqtt:
  broker_host: localhost
  broker_port: 1883
  
logging:
  level: debug
""")
        
        try:
            cls.controller_process = subprocess.Popen([
                "./build_dir/rtk_controller",
                "--config", config_file
            ], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        except FileNotFoundError:
            cls.fail("RTK controller not found. Please run 'make build'.")
    
    def test_device_connection(self):
        """測試設備連接"""
        # 創建測試設備
        device_script = f"""
from rtk_mqtt_client import RTKMQTTClient
import time

client = RTKMQTTClient('test001', 'test', 'integration', 'sensor')
client.connect('localhost', 1883)
client.publish_device_attributes()
client.publish_state('ok')
time.sleep(5)
client.disconnect()
"""
        
        # 執行測試
        result = subprocess.run([
            "python3", "-c", device_script
        ], capture_output=True, text=True, timeout=30)
        
        self.assertEqual(result.returncode, 0, 
                        f"Device connection failed: {result.stderr}")
    
    def test_command_execution(self):
        """測試命令執行"""
        # TODO: 實作命令執行測試
        pass
    
    def test_message_validation(self):
        """測試消息格式驗證"""
        # TODO: 實作消息驗證測試
        pass

if __name__ == "__main__":
    unittest.main()
```

### 效能基準測試
```bash
#!/bin/bash
# tools/benchmark.sh

set -e

echo "🚀 RTK MQTT Performance Benchmark"

# 參數
BROKER_HOST=${1:-localhost}
BROKER_PORT=${2:-1883}
NUM_DEVICES=${3:-100}
DURATION=${4:-300}
RESULTS_DIR="benchmark_results_$(date +%Y%m%d_%H%M%S)"

mkdir -p $RESULTS_DIR

echo "📊 Benchmark Configuration:"
echo "  Broker: $BROKER_HOST:$BROKER_PORT"
echo "  Devices: $NUM_DEVICES"
echo "  Duration: ${DURATION}s"
echo "  Results: $RESULTS_DIR"
echo ""

# 啟動負載測試
echo "🔄 Starting load test..."
python3 tools/rtk_load_tester.py \
    --broker $BROKER_HOST \
    --port $BROKER_PORT \
    --devices $NUM_DEVICES \
    --duration $DURATION > $RESULTS_DIR/load_test.log

# 記憶體使用監控
echo "🧠 Monitoring memory usage..."
(
    while sleep 5; do
        ps aux | grep -E "(mosquitto|rtk)" | grep -v grep >> $RESULTS_DIR/memory_usage.log
    done
) &
MEMORY_PID=$!

# 等待測試完成
sleep $DURATION

# 停止監控
kill $MEMORY_PID 2>/dev/null || true

# 生成報告
echo "📋 Generating benchmark report..."
cat > $RESULTS_DIR/report.md << EOF
# RTK MQTT Benchmark Report

## Test Configuration
- Date: $(date)
- Broker: $BROKER_HOST:$BROKER_PORT
- Devices: $NUM_DEVICES
- Duration: ${DURATION}s

## Results
$(cat $RESULTS_DIR/load_test.log | tail -10)

## Resource Usage
### Memory Usage
$(tail -5 $RESULTS_DIR/memory_usage.log)

EOF

echo "✅ Benchmark completed. Results in $RESULTS_DIR/"
```

## 🔗 相關工具

### Makefile 整合
```makefile
# 添加到現有 Makefile

.PHONY: tools-install tools-test tools-benchmark

tools-install:
	@echo "Installing development tools..."
	pip3 install paho-mqtt jsonschema memory_profiler
	go install golang.org/x/tools/cmd/pprof@latest

tools-test:
	@echo "Running tool tests..."
	python3 tools/rtk_message_validator.py --broker localhost --duration 10
	python3 tools/integration_test_suite.py

tools-benchmark:
	@echo "Running performance benchmark..."
	./tools/benchmark.sh localhost 1883 50 60

tools-debug:
	@echo "Starting RTK debugger..."
	python3 tools/rtk_debugger.py --broker localhost
```

### Docker 開發環境
```dockerfile
# tools/Dockerfile.dev
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    mosquitto mosquitto-clients \
    python3 python3-pip \
    golang-go \
    build-essential \
    curl wget \
    && rm -rf /var/lib/apt/lists/*

# 安裝 Python 依賴
RUN pip3 install paho-mqtt jsonschema memory_profiler

# 安裝 Go 工具
RUN go install golang.org/x/tools/cmd/pprof@latest

WORKDIR /rtk_development

COPY tools/ ./tools/
COPY docs/ ./docs/

CMD ["/bin/bash"]
```

```bash
# 使用開發容器
docker build -f tools/Dockerfile.dev -t rtk-dev-tools .
docker run -it --rm -v $(pwd):/rtk_development rtk-dev-tools
```

## 📚 相關資源

- **[測試指南](../guides/TESTING_INTEGRATION.md)** - 完整測試策略
- **[故障排除](../guides/TROUBLESHOOTING_GUIDE.md)** - 常見問題解決
- **[API 參考](../core/MQTT_API_REFERENCE.md)** - 協議規格
- **[SDK 指南](SDK_REFERENCE.md)** - 多語言開發支援

---

這個開發工具指南提供了完整的工具鏈，幫助開發者在 RTK MQTT 專案中進行高效的開發、測試和除錯工作。
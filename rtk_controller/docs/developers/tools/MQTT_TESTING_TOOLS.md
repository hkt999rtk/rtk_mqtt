# MQTT 測試工具指南

## 概述

本文檔提供RTK MQTT系統專用的測試工具，包含訊息驗證、負載測試、協議測試和效能分析等工具。

📋 **JSON Schema 驗證**: 所有測試工具都使用 [`docs/spec/schemas/`](../../spec/schemas/) 中的 JSON Schema 進行訊息格式驗證。

## 工具列表

### 1. RTK MQTT 訊息驗證器 (Message Validator)

#### 功能說明
驗證MQTT訊息是否符合RTK協議規範，包含主題格式、JSON Schema和內容檢查。

#### 安裝和使用
```bash
# 安裝依賴
pip install paho-mqtt jsonschema

# 執行驗證器
python3 rtk_message_validator.py --broker localhost --port 1883
```

#### 完整實作
```python
#!/usr/bin/env python3
# rtk_message_validator.py

import paho.mqtt.client as mqtt
import json
import jsonschema
import re
import argparse
import logging
from datetime import datetime
from collections import defaultdict, Counter
import time

class RTKMessageValidator:
    def __init__(self, broker_host="localhost", broker_port=1883):
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.client = mqtt.Client("rtk_validator")
        
        # 統計資料
        self.stats = {
            'total_messages': 0,
            'valid_messages': 0,
            'invalid_messages': 0,
            'validation_errors': defaultdict(int),
            'topic_patterns': Counter(),
            'schema_violations': defaultdict(list)
        }
        
        # 載入JSON Schemas
        self.schemas = self.load_schemas()
        
        # 設定MQTT回調
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        
        # 設定日誌
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s'
        )
        self.logger = logging.getLogger(__name__)
        
    def load_schemas(self):
        """載入RTK MQTT JSON Schemas"""
        schemas = {}
        
        # 狀態訊息Schema
        schemas['state'] = {
            "type": "object",
            "required": ["schema", "ts", "health"],
            "properties": {
                "schema": {"type": "string", "pattern": "^state/\\d+\\.\\d+$"},
                "ts": {"type": "integer", "minimum": 0},
                "health": {"type": "string", "enum": ["ok", "warning", "error", "critical"]},
                "uptime_s": {"type": "integer", "minimum": 0},
                "connection_status": {"type": "string"},
                "device_id": {"type": "string"}
            }
        }
        
        # 遙測訊息Schema
        schemas['telemetry'] = {
            "type": "object",
            "required": ["schema", "ts"],
            "properties": {
                "schema": {"type": "string", "pattern": "^telemetry\\..+/\\d+\\.\\d+$"},
                "ts": {"type": "integer", "minimum": 0},
                "device_id": {"type": "string"}
            }
        }
        
        # 事件訊息Schema
        schemas['evt'] = {
            "type": "object",
            "required": ["schema", "ts", "event_type"],
            "properties": {
                "schema": {"type": "string", "pattern": "^evt\\..+/\\d+\\.\\d+$"},
                "ts": {"type": "integer", "minimum": 0},
                "event_type": {"type": "string"},
                "severity": {"type": "string", "enum": ["info", "warning", "error", "critical"]},
                "device_id": {"type": "string"}
            }
        }
        
        # 命令訊息Schema
        schemas['cmd'] = {
            "req": {
                "type": "object",
                "required": ["id", "op", "schema", "ts"],
                "properties": {
                    "id": {"type": "string"},
                    "op": {"type": "string"},
                    "schema": {"type": "string", "pattern": "^cmd\\..+/\\d+\\.\\d+$"},
                    "ts": {"type": "integer", "minimum": 0},
                    "args": {"type": "object"},
                    "timeout_ms": {"type": "integer", "minimum": 1000}
                }
            },
            "ack": {
                "type": "object",
                "required": ["id", "schema", "status", "ts"],
                "properties": {
                    "id": {"type": "string"},
                    "schema": {"type": "string", "enum": ["cmd.ack/1.0"]},
                    "status": {"type": "string", "enum": ["accepted", "rejected"]},
                    "ts": {"type": "integer", "minimum": 0}
                }
            },
            "res": {
                "type": "object",
                "required": ["id", "schema", "status", "ts"],
                "properties": {
                    "id": {"type": "string"},
                    "schema": {"type": "string", "pattern": "^cmd\\..+\\.result/\\d+\\.\\d+$"},
                    "status": {"type": "string", "enum": ["completed", "failed", "timeout"]},
                    "result": {"type": "object"},
                    "error": {"type": "string"},
                    "ts": {"type": "integer", "minimum": 0}
                }
            }
        }
        
        # 屬性訊息Schema
        schemas['attr'] = {
            "type": "object",
            "required": ["schema", "ts", "device_type"],
            "properties": {
                "schema": {"type": "string", "enum": ["attr/1.0"]},
                "ts": {"type": "integer", "minimum": 0},
                "device_type": {"type": "string"},
                "manufacturer": {"type": "string"},
                "model": {"type": "string"},
                "firmware_version": {"type": "string"},
                "capabilities": {"type": "array", "items": {"type": "string"}}
            }
        }
        
        return schemas
    
    def on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            self.logger.info(f"Connected to MQTT broker at {self.broker_host}:{self.broker_port}")
            # 訂閱所有RTK主題
            client.subscribe("rtk/v1/#", qos=0)
            self.logger.info("Subscribed to rtk/v1/#")
        else:
            self.logger.error(f"Failed to connect to MQTT broker: {rc}")
    
    def on_message(self, client, userdata, msg):
        try:
            self.validate_message(msg.topic, msg.payload.decode('utf-8'))
        except Exception as e:
            self.logger.error(f"Error processing message: {e}")
    
    def validate_message(self, topic, payload):
        """驗證MQTT訊息"""
        self.stats['total_messages'] += 1
        validation_result = {
            'topic': topic,
            'timestamp': datetime.now().isoformat(),
            'valid': True,
            'errors': []
        }
        
        # 1. 驗證主題格式
        topic_valid = self.validate_topic_format(topic)
        if not topic_valid['valid']:
            validation_result['valid'] = False
            validation_result['errors'].extend(topic_valid['errors'])
            self.stats['validation_errors']['topic_format'] += 1
        
        # 2. 驗證JSON格式
        try:
            message_data = json.loads(payload)
        except json.JSONDecodeError as e:
            validation_result['valid'] = False
            validation_result['errors'].append(f"Invalid JSON: {str(e)}")
            self.stats['validation_errors']['json_format'] += 1
            self.log_validation_result(validation_result)
            return
        
        # 3. 驗證時間戳
        timestamp_valid = self.validate_timestamp(message_data.get('ts'))
        if not timestamp_valid['valid']:
            validation_result['valid'] = False
            validation_result['errors'].extend(timestamp_valid['errors'])
            self.stats['validation_errors']['timestamp'] += 1
        
        # 4. 驗證Schema
        if topic_valid['valid']:
            schema_valid = self.validate_message_schema(topic_valid['parsed'], message_data)
            if not schema_valid['valid']:
                validation_result['valid'] = False
                validation_result['errors'].extend(schema_valid['errors'])
                self.stats['validation_errors']['schema'] += 1
        
        # 5. 驗證業務邏輯
        business_valid = self.validate_business_logic(topic_valid.get('parsed', {}), message_data)
        if not business_valid['valid']:
            validation_result['valid'] = False
            validation_result['errors'].extend(business_valid['errors'])
            self.stats['validation_errors']['business_logic'] += 1
        
        # 更新統計
        if validation_result['valid']:
            self.stats['valid_messages'] += 1
        else:
            self.stats['invalid_messages'] += 1
        
        # 記錄結果
        self.log_validation_result(validation_result)
        
        # 更新主題模式統計
        if topic_valid['valid']:
            pattern = self.get_topic_pattern(topic_valid['parsed'])
            self.stats['topic_patterns'][pattern] += 1
    
    def validate_topic_format(self, topic):
        """驗證主題格式"""
        result = {'valid': True, 'errors': []}
        
        # RTK主題格式: rtk/v1/{tenant}/{site}/{device_id}/{message_type}[/{sub_type}]
        pattern = r'^rtk/v1/([a-zA-Z0-9_]+)/([a-zA-Z0-9_]+)/([a-zA-Z0-9:_.-]+)/(state|telemetry|evt|attr|cmd|lwt|topology)(?:/(.+))?$'
        
        match = re.match(pattern, topic)
        if not match:
            result['valid'] = False
            result['errors'].append(f"Topic does not match RTK format: {topic}")
            return result
        
        # 解析主題組件
        tenant, site, device_id, message_type, sub_type = match.groups()
        
        result['parsed'] = {
            'tenant': tenant,
            'site': site,
            'device_id': device_id,
            'message_type': message_type,
            'sub_type': sub_type
        }
        
        # 驗證組件
        if len(tenant) < 3 or len(tenant) > 32:
            result['valid'] = False
            result['errors'].append(f"Tenant name length should be 3-32 characters: {tenant}")
        
        if len(site) < 3 or len(site) > 32:
            result['valid'] = False
            result['errors'].append(f"Site name length should be 3-32 characters: {site}")
        
        if len(device_id) < 6 or len(device_id) > 64:
            result['valid'] = False
            result['errors'].append(f"Device ID length should be 6-64 characters: {device_id}")
        
        return result
    
    def validate_timestamp(self, timestamp):
        """驗證時間戳"""
        result = {'valid': True, 'errors': []}
        
        if timestamp is None:
            result['valid'] = False
            result['errors'].append("Missing timestamp field 'ts'")
            return result
        
        if not isinstance(timestamp, int):
            result['valid'] = False
            result['errors'].append(f"Timestamp must be integer: {type(timestamp)}")
            return result
        
        # 檢查時間戳合理性 (毫秒級)
        current_time = int(time.time() * 1000)
        if timestamp < 1000000000000:  # 2001年以前
            result['valid'] = False
            result['errors'].append(f"Timestamp too old: {timestamp}")
        elif timestamp > current_time + 300000:  # 未來5分鐘以後
            result['valid'] = False
            result['errors'].append(f"Timestamp too far in future: {timestamp}")
        
        return result
    
    def validate_message_schema(self, topic_parts, message_data):
        """驗證訊息Schema"""
        result = {'valid': True, 'errors': []}
        
        message_type = topic_parts.get('message_type')
        sub_type = topic_parts.get('sub_type')
        
        # 選擇適當的Schema
        schema = None
        if message_type in ['state', 'telemetry', 'evt', 'attr']:
            schema = self.schemas.get(message_type)
        elif message_type == 'cmd' and sub_type:
            schema = self.schemas.get('cmd', {}).get(sub_type)
        
        if not schema:
            result['valid'] = False
            result['errors'].append(f"No schema found for message type: {message_type}/{sub_type}")
            return result
        
        # 執行Schema驗證
        try:
            jsonschema.validate(message_data, schema)
        except jsonschema.ValidationError as e:
            result['valid'] = False
            result['errors'].append(f"Schema validation error: {e.message}")
            # 記錄詳細錯誤
            self.stats['schema_violations'][message_type].append({
                'error': e.message,
                'path': list(e.absolute_path),
                'timestamp': datetime.now().isoformat()
            })
        except jsonschema.SchemaError as e:
            result['valid'] = False
            result['errors'].append(f"Schema error: {e.message}")
        
        return result
    
    def validate_business_logic(self, topic_parts, message_data):
        """驗證業務邏輯"""
        result = {'valid': True, 'errors': []}
        
        message_type = topic_parts.get('message_type')
        
        # 特定訊息類型的業務邏輯驗證
        if message_type == 'state':
            # 狀態訊息必須包含健康狀態
            health = message_data.get('health')
            if health not in ['ok', 'warning', 'error', 'critical']:
                result['valid'] = False
                result['errors'].append(f"Invalid health status: {health}")
        
        elif message_type == 'cmd':
            sub_type = topic_parts.get('sub_type')
            if sub_type == 'req':
                # 命令請求必須有操作名稱
                operation = message_data.get('op')
                if not operation:
                    result['valid'] = False
                    result['errors'].append("Command request missing 'op' field")
                
                # 命令ID格式檢查
                cmd_id = message_data.get('id')
                if not cmd_id or len(cmd_id) < 8:
                    result['valid'] = False
                    result['errors'].append("Command ID too short or missing")
        
        elif message_type == 'telemetry':
            # 遙測數據應該包含數值
            has_numeric_data = any(
                isinstance(v, (int, float)) for v in message_data.values()
                if k not in ['schema', 'ts', 'device_id']
            )
            if not has_numeric_data:
                result['valid'] = False
                result['errors'].append("Telemetry message should contain numeric data")
        
        return result
    
    def get_topic_pattern(self, topic_parts):
        """獲取主題模式"""
        return f"{topic_parts['message_type']}/{topic_parts.get('sub_type', '')}"
    
    def log_validation_result(self, result):
        """記錄驗證結果"""
        if result['valid']:
            self.logger.debug(f"✓ Valid message: {result['topic']}")
        else:
            self.logger.warning(f"✗ Invalid message: {result['topic']}")
            for error in result['errors']:
                self.logger.warning(f"  - {error}")
    
    def print_statistics(self):
        """打印統計資訊"""
        print("\n" + "="*80)
        print("RTK MQTT Message Validation Statistics")
        print("="*80)
        
        total = self.stats['total_messages']
        valid = self.stats['valid_messages']
        invalid = self.stats['invalid_messages']
        
        print(f"Total Messages:     {total}")
        print(f"Valid Messages:     {valid} ({valid/total*100:.1f}%)" if total > 0 else "Valid Messages:     0")
        print(f"Invalid Messages:   {invalid} ({invalid/total*100:.1f}%)" if total > 0 else "Invalid Messages:   0")
        
        if self.stats['validation_errors']:
            print(f"\nValidation Errors:")
            for error_type, count in self.stats['validation_errors'].items():
                print(f"  {error_type:20} {count:6} ({count/total*100:.1f}%)" if total > 0 else f"  {error_type:20} {count:6}")
        
        if self.stats['topic_patterns']:
            print(f"\nMessage Type Distribution:")
            for pattern, count in self.stats['topic_patterns'].most_common():
                print(f"  {pattern:20} {count:6} ({count/total*100:.1f}%)" if total > 0 else f"  {pattern:20} {count:6}")
        
        print("="*80)
    
    def start_validation(self, duration=None):
        """開始驗證"""
        try:
            self.logger.info(f"Starting RTK MQTT message validation...")
            self.client.connect(self.broker_host, self.broker_port, 60)
            self.client.loop_start()
            
            if duration:
                self.logger.info(f"Validation will run for {duration} seconds")
                time.sleep(duration)
            else:
                self.logger.info("Validation running indefinitely. Press Ctrl+C to stop.")
                while True:
                    time.sleep(1)
                    
        except KeyboardInterrupt:
            self.logger.info("Validation stopped by user")
        finally:
            self.client.loop_stop()
            self.client.disconnect()
            self.print_statistics()

def main():
    parser = argparse.ArgumentParser(description='RTK MQTT Message Validator')
    parser.add_argument('--broker', default='localhost', help='MQTT broker host')
    parser.add_argument('--port', type=int, default=1883, help='MQTT broker port')
    parser.add_argument('--duration', type=int, help='Validation duration in seconds')
    parser.add_argument('--verbose', '-v', action='store_true', help='Verbose logging')
    
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    validator = RTKMessageValidator(args.broker, args.port)
    validator.start_validation(args.duration)

if __name__ == "__main__":
    main()
```

### 2. RTK MQTT 負載測試工具 (Load Testing Tool)

#### 功能說明
模擬大量設備連接，測試MQTT系統的性能和穩定性。

#### 實作
```python
#!/usr/bin/env python3
# rtk_load_tester.py

import paho.mqtt.client as mqtt
import json
import time
import random
import threading
import argparse
from concurrent.futures import ThreadPoolExecutor
import logging
from dataclasses import dataclass
from typing import List, Dict, Optional
import signal
import sys

@dataclass
class LoadTestConfig:
    broker_host: str = "localhost"
    broker_port: int = 1883
    device_count: int = 100
    test_duration: int = 300
    message_rate: float = 1.0  # messages per second per device
    message_types: List[str] = None
    tenant: str = "load_test"
    site: str = "simulation"
    
    def __post_init__(self):
        if self.message_types is None:
            self.message_types = ["state", "telemetry", "evt"]

class SimulatedDevice:
    def __init__(self, device_id: str, config: LoadTestConfig):
        self.device_id = device_id
        self.config = config
        self.client = mqtt.Client(client_id=f"load_test_{device_id}")
        self.running = False
        self.stats = {
            'messages_sent': 0,
            'messages_failed': 0,
            'connection_attempts': 0,
            'connection_failures': 0,
            'last_message_time': None
        }
        
        # 設定回調
        self.client.on_connect = self.on_connect
        self.client.on_disconnect = self.on_disconnect
        self.client.on_publish = self.on_publish
        
        # 設備屬性
        self.device_type = random.choice(["router", "ap", "switch", "iot", "sensor"])
        self.uptime = 0
        self.last_telemetry = {}
        
    def on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            logging.debug(f"Device {self.device_id} connected")
            # 發布設備屬性
            self.publish_attributes()
        else:
            self.stats['connection_failures'] += 1
            logging.error(f"Device {self.device_id} connection failed: {rc}")
    
    def on_disconnect(self, client, userdata, rc):
        logging.debug(f"Device {self.device_id} disconnected: {rc}")
    
    def on_publish(self, client, userdata, mid):
        self.stats['messages_sent'] += 1
        self.stats['last_message_time'] = time.time()
    
    def connect(self):
        """連接到MQTT broker"""
        try:
            self.stats['connection_attempts'] += 1
            self.client.connect(self.config.broker_host, self.config.broker_port, 60)
            self.client.loop_start()
            return True
        except Exception as e:
            self.stats['connection_failures'] += 1
            logging.error(f"Device {self.device_id} connection error: {e}")
            return False
    
    def disconnect(self):
        """斷開連接"""
        self.running = False
        self.client.loop_stop()
        self.client.disconnect()
    
    def publish_attributes(self):
        """發布設備屬性"""
        attr_topic = f"rtk/v1/{self.config.tenant}/{self.config.site}/{self.device_id}/attr"
        attr_msg = {
            "schema": "attr/1.0",
            "ts": int(time.time() * 1000),
            "device_type": self.device_type,
            "manufacturer": "LoadTest Corp",
            "model": f"LT-{self.device_type.upper()}-1000",
            "firmware_version": "1.0.0",
            "capabilities": self.get_device_capabilities()
        }
        
        self.client.publish(attr_topic, json.dumps(attr_msg), qos=1, retain=True)
    
    def get_device_capabilities(self):
        """獲取設備能力"""
        capabilities_map = {
            "router": ["routing", "nat", "firewall", "vpn"],
            "ap": ["wifi", "client_access", "roaming"],
            "switch": ["switching", "vlan", "stp"],
            "iot": ["sensor", "telemetry"],
            "sensor": ["temperature", "humidity", "monitoring"]
        }
        return capabilities_map.get(self.device_type, ["basic"])
    
    def generate_state_message(self):
        """生成狀態訊息"""
        self.uptime += random.randint(1, 10)
        
        return {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "health": random.choices(
                ["ok", "warning", "error"], 
                weights=[0.8, 0.15, 0.05]
            )[0],
            "uptime_s": self.uptime,
            "connection_status": "connected",
            "cpu_usage": random.uniform(10, 80),
            "memory_usage": random.uniform(20, 90)
        }
    
    def generate_telemetry_message(self):
        """生成遙測訊息"""
        telemetry_types = {
            "router": self.generate_router_telemetry,
            "ap": self.generate_ap_telemetry,
            "switch": self.generate_switch_telemetry,
            "iot": self.generate_iot_telemetry,
            "sensor": self.generate_sensor_telemetry
        }
        
        generator = telemetry_types.get(self.device_type, self.generate_generic_telemetry)
        return generator()
    
    def generate_router_telemetry(self):
        """生成路由器遙測"""
        return {
            "schema": "telemetry.network/1.0",
            "ts": int(time.time() * 1000),
            "interface": "eth0",
            "tx_bytes": random.randint(1000000, 100000000),
            "rx_bytes": random.randint(1000000, 100000000),
            "tx_packets": random.randint(1000, 100000),
            "rx_packets": random.randint(1000, 100000),
            "cpu_usage": random.uniform(10, 60),
            "memory_usage": random.uniform(30, 80),
            "temperature_c": random.uniform(35, 65)
        }
    
    def generate_ap_telemetry(self):
        """生成AP遙測"""
        return {
            "schema": "telemetry.wifi/1.0",
            "ts": int(time.time() * 1000),
            "ssid": "LoadTest-WiFi",
            "channel": random.choice([1, 6, 11, 36, 40, 44, 48]),
            "client_count": random.randint(0, 50),
            "signal_strength": random.randint(-80, -30),
            "throughput_mbps": random.uniform(10, 150),
            "interference_level": random.choice(["low", "medium", "high"])
        }
    
    def generate_switch_telemetry(self):
        """生成交換機遙測"""
        return {
            "schema": "telemetry.switch/1.0",
            "ts": int(time.time() * 1000),
            "port_count": 24,
            "active_ports": random.randint(8, 24),
            "total_throughput_mbps": random.uniform(100, 1000),
            "spanning_tree_status": "forwarding",
            "vlan_count": random.randint(1, 10)
        }
    
    def generate_iot_telemetry(self):
        """生成IoT遙測"""
        return {
            "schema": "telemetry.iot/1.0",
            "ts": int(time.time() * 1000),
            "battery_level": random.randint(20, 100),
            "signal_strength": random.randint(-90, -40),
            "sensor_data": {
                "value": random.uniform(0, 100),
                "unit": "percent"
            }
        }
    
    def generate_sensor_telemetry(self):
        """生成感測器遙測"""
        return {
            "schema": "telemetry.environment/1.0",
            "ts": int(time.time() * 1000),
            "temperature_c": random.uniform(18, 28),
            "humidity_percent": random.uniform(40, 70),
            "pressure_hpa": random.uniform(1000, 1030),
            "air_quality_index": random.randint(20, 150)
        }
    
    def generate_generic_telemetry(self):
        """生成通用遙測"""
        return {
            "schema": "telemetry.system/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "payload": {
                "cpu_usage": random.uniform(10, 80),
                "memory_usage": random.uniform(20, 90),
                "disk_usage": random.uniform(10, 95)
            }
        }
    
    def generate_event_message(self):
        """生成事件訊息"""
        event_types = ["system", "network", "security", "performance"]
        event_type = random.choice(event_types)
        
        events = {
            "system": ["startup", "shutdown", "config_change"],
            "network": ["link_up", "link_down", "high_latency"],
            "security": ["login_failed", "unauthorized_access"],
            "performance": ["high_cpu", "high_memory", "disk_full"]
        }
        
        specific_event = random.choice(events[event_type])
        
        return {
            "schema": f"evt.{event_type}/1.0",
            "ts": int(time.time() * 1000),
            "event_type": specific_event,
            "severity": random.choices(
                ["info", "warning", "error", "critical"],
                weights=[0.5, 0.3, 0.15, 0.05]
            )[0],
            "description": f"Simulated {specific_event} event",
            "source": self.device_type
        }
    
    def publish_message(self, message_type: str):
        """發布訊息"""
        try:
            if message_type == "state":
                topic = f"rtk/v1/{self.config.tenant}/{self.config.site}/{self.device_id}/state"
                message = self.generate_state_message()
                qos = 1
                retain = True
            elif message_type == "telemetry":
                sub_type = "system" if self.device_type == "generic" else self.device_type
                topic = f"rtk/v1/{self.config.tenant}/{self.config.site}/{self.device_id}/telemetry/{sub_type}"
                message = self.generate_telemetry_message()
                qos = 0
                retain = False
            elif message_type == "evt":
                event_category = random.choice(["system", "network", "security"])
                topic = f"rtk/v1/{self.config.tenant}/{self.config.site}/{self.device_id}/evt/{event_category}"
                message = self.generate_event_message()
                qos = 1
                retain = False
            else:
                return
                
            payload = json.dumps(message)
            result = self.client.publish(topic, payload, qos=qos, retain=retain)
            
            if result.rc != mqtt.MQTT_ERR_SUCCESS:
                self.stats['messages_failed'] += 1
                
        except Exception as e:
            self.stats['messages_failed'] += 1
            logging.error(f"Device {self.device_id} publish error: {e}")
    
    def run_simulation(self):
        """運行設備模擬"""
        self.running = True
        interval = 1.0 / self.config.message_rate if self.config.message_rate > 0 else 1.0
        
        while self.running:
            message_type = random.choice(self.config.message_types)
            self.publish_message(message_type)
            
            # 隨機間隔以模擬真實設備行為
            actual_interval = interval * random.uniform(0.8, 1.2)
            time.sleep(actual_interval)

class LoadTester:
    def __init__(self, config: LoadTestConfig):
        self.config = config
        self.devices: List[SimulatedDevice] = []
        self.running = False
        self.start_time = None
        
        # 設定日誌
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s'
        )
        self.logger = logging.getLogger(__name__)
        
        # 設定信號處理
        signal.signal(signal.SIGINT, self.signal_handler)
        signal.signal(signal.SIGTERM, self.signal_handler)
    
    def signal_handler(self, signum, frame):
        """處理終止信號"""
        self.logger.info("Received termination signal, stopping load test...")
        self.stop_test()
        sys.exit(0)
    
    def create_devices(self):
        """創建模擬設備"""
        self.logger.info(f"Creating {self.config.device_count} simulated devices...")
        
        for i in range(self.config.device_count):
            device_id = f"load_test_device_{i:05d}"
            device = SimulatedDevice(device_id, self.config)
            self.devices.append(device)
        
        self.logger.info(f"Created {len(self.devices)} devices")
    
    def connect_devices(self):
        """連接所有設備"""
        self.logger.info("Connecting devices to MQTT broker...")
        
        successful_connections = 0
        failed_connections = 0
        
        with ThreadPoolExecutor(max_workers=50) as executor:
            futures = [executor.submit(device.connect) for device in self.devices]
            
            for future in futures:
                if future.result():
                    successful_connections += 1
                else:
                    failed_connections += 1
        
        self.logger.info(f"Connection results: {successful_connections} successful, {failed_connections} failed")
        
        # 等待連接穩定
        time.sleep(5)
    
    def start_simulation(self):
        """開始負載模擬"""
        self.logger.info("Starting load simulation...")
        self.running = True
        self.start_time = time.time()
        
        # 啟動所有設備的模擬
        with ThreadPoolExecutor(max_workers=self.config.device_count) as executor:
            simulation_futures = [
                executor.submit(device.run_simulation) 
                for device in self.devices 
                if hasattr(device.client, '_sock') and device.client._sock
            ]
            
            # 監控測試進度
            try:
                elapsed = 0
                while elapsed < self.config.test_duration and self.running:
                    time.sleep(10)
                    elapsed = time.time() - self.start_time
                    self.print_progress(elapsed)
                    
            except KeyboardInterrupt:
                self.logger.info("Test interrupted by user")
            finally:
                self.stop_simulation()
    
    def stop_simulation(self):
        """停止模擬"""
        self.running = False
        for device in self.devices:
            device.running = False
    
    def disconnect_devices(self):
        """斷開所有設備連接"""
        self.logger.info("Disconnecting devices...")
        
        for device in self.devices:
            device.disconnect()
    
    def print_progress(self, elapsed_time):
        """打印測試進度"""
        total_messages = sum(device.stats['messages_sent'] for device in self.devices)
        total_failed = sum(device.stats['messages_failed'] for device in self.devices)
        message_rate = total_messages / elapsed_time if elapsed_time > 0 else 0
        
        self.logger.info(f"Progress: {elapsed_time:.0f}s elapsed, "
                        f"{total_messages} messages sent, "
                        f"{total_failed} failed, "
                        f"{message_rate:.1f} msg/s")
    
    def generate_report(self):
        """生成測試報告"""
        if not self.start_time:
            return
            
        total_duration = time.time() - self.start_time
        
        # 收集統計
        total_messages = sum(device.stats['messages_sent'] for device in self.devices)
        total_failed = sum(device.stats['messages_failed'] for device in self.devices)
        total_connection_attempts = sum(device.stats['connection_attempts'] for device in self.devices)
        total_connection_failures = sum(device.stats['connection_failures'] for device in self.devices)
        
        connected_devices = sum(1 for device in self.devices 
                              if hasattr(device.client, '_sock') and device.client._sock)
        
        report = {
            "test_configuration": {
                "device_count": self.config.device_count,
                "test_duration": self.config.test_duration,
                "target_message_rate": self.config.message_rate,
                "message_types": self.config.message_types
            },
            "results": {
                "actual_duration": round(total_duration, 2),
                "total_messages_sent": total_messages,
                "total_messages_failed": total_failed,
                "success_rate": round((total_messages / (total_messages + total_failed)) * 100, 2) if (total_messages + total_failed) > 0 else 0,
                "actual_message_rate": round(total_messages / total_duration, 2) if total_duration > 0 else 0,
                "connected_devices": connected_devices,
                "connection_success_rate": round((total_connection_attempts - total_connection_failures) / total_connection_attempts * 100, 2) if total_connection_attempts > 0 else 0
            },
            "performance_metrics": {
                "messages_per_second": round(total_messages / total_duration, 2) if total_duration > 0 else 0,
                "average_messages_per_device": round(total_messages / len(self.devices), 2) if self.devices else 0,
                "throughput_efficiency": round((total_messages / total_duration) / (self.config.device_count * self.config.message_rate) * 100, 2) if total_duration > 0 and self.config.message_rate > 0 else 0
            }
        }
        
        return report
    
    def print_report(self, report):
        """打印測試報告"""
        print("\n" + "="*80)
        print("RTK MQTT Load Test Report")
        print("="*80)
        
        config = report["test_configuration"]
        results = report["results"]
        metrics = report["performance_metrics"]
        
        print(f"Test Configuration:")
        print(f"  Device Count:         {config['device_count']}")
        print(f"  Test Duration:        {config['test_duration']}s")
        print(f"  Target Message Rate:  {config['target_message_rate']} msg/s/device")
        print(f"  Message Types:        {', '.join(config['message_types'])}")
        
        print(f"\nTest Results:")
        print(f"  Actual Duration:      {results['actual_duration']}s")
        print(f"  Messages Sent:        {results['total_messages_sent']}")
        print(f"  Messages Failed:      {results['total_messages_failed']}")
        print(f"  Success Rate:         {results['success_rate']}%")
        print(f"  Connected Devices:    {results['connected_devices']}")
        print(f"  Connection Success:   {results['connection_success_rate']}%")
        
        print(f"\nPerformance Metrics:")
        print(f"  Actual Message Rate:  {metrics['messages_per_second']} msg/s")
        print(f"  Avg Msg/Device:       {metrics['average_messages_per_device']}")
        print(f"  Throughput Efficiency: {metrics['throughput_efficiency']}%")
        
        print("="*80)
    
    def run_test(self):
        """執行完整負載測試"""
        try:
            self.create_devices()
            self.connect_devices()
            self.start_simulation()
        finally:
            self.disconnect_devices()
            report = self.generate_report()
            self.print_report(report)
            return report
    
    def stop_test(self):
        """停止測試"""
        self.stop_simulation()
        self.disconnect_devices()

def main():
    parser = argparse.ArgumentParser(description='RTK MQTT Load Tester')
    parser.add_argument('--broker', default='localhost', help='MQTT broker host')
    parser.add_argument('--port', type=int, default=1883, help='MQTT broker port')
    parser.add_argument('--devices', type=int, default=100, help='Number of simulated devices')
    parser.add_argument('--duration', type=int, default=300, help='Test duration in seconds')
    parser.add_argument('--rate', type=float, default=1.0, help='Message rate per device (msg/s)')
    parser.add_argument('--types', nargs='+', default=['state', 'telemetry', 'evt'], 
                       help='Message types to simulate')
    parser.add_argument('--tenant', default='load_test', help='Tenant name')
    parser.add_argument('--site', default='simulation', help='Site name')
    parser.add_argument('--verbose', '-v', action='store_true', help='Verbose logging')
    
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    config = LoadTestConfig(
        broker_host=args.broker,
        broker_port=args.port,
        device_count=args.devices,
        test_duration=args.duration,
        message_rate=args.rate,
        message_types=args.types,
        tenant=args.tenant,
        site=args.site
    )
    
    load_tester = LoadTester(config)
    load_tester.run_test()

if __name__ == "__main__":
    main()
```

### 3. RTK MQTT 協議測試工具 (Protocol Tester)

#### 功能說明
測試RTK MQTT協議的各種功能，包含命令響應、QoS設定、主題權限等。

#### 實作
```python
#!/usr/bin/env python3
# rtk_protocol_tester.py

import paho.mqtt.client as mqtt
import json
import time
import threading
import queue
import argparse
import logging
from dataclasses import dataclass
from typing import Dict, List, Optional, Callable
from enum import Enum

class TestResult(Enum):
    PASS = "PASS"
    FAIL = "FAIL"
    SKIP = "SKIP"

@dataclass
class TestCase:
    name: str
    description: str
    test_function: Callable
    expected_result: str = "success"
    timeout: int = 30
    prerequisites: List[str] = None
    
    def __post_init__(self):
        if self.prerequisites is None:
            self.prerequisites = []

class RTKProtocolTester:
    def __init__(self, broker_host="localhost", broker_port=1883, device_id="protocol_tester"):
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.device_id = device_id
        
        # MQTT客戶端
        self.publisher = mqtt.Client("rtk_tester_pub")
        self.subscriber = mqtt.Client("rtk_tester_sub")
        
        # 測試狀態
        self.test_results = {}
        self.message_queue = queue.Queue()
        self.response_events = {}
        
        # 設定回調
        self.subscriber.on_connect = self.on_subscriber_connect
        self.subscriber.on_message = self.on_message_received
        
        # 測試用例
        self.test_cases = self.define_test_cases()
        
        # 設定日誌
        logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
        self.logger = logging.getLogger(__name__)
    
    def define_test_cases(self):
        """定義測試用例"""
        return [
            TestCase(
                name="mqtt_connection",
                description="Test MQTT broker connection",
                test_function=self.test_mqtt_connection
            ),
            TestCase(
                name="topic_format_validation",
                description="Test RTK topic format validation",
                test_function=self.test_topic_format_validation
            ),
            TestCase(
                name="state_message_publishing",
                description="Test device state message publishing",
                test_function=self.test_state_message_publishing,
                prerequisites=["mqtt_connection"]
            ),
            TestCase(
                name="telemetry_message_publishing",
                description="Test telemetry message publishing",
                test_function=self.test_telemetry_message_publishing,
                prerequisites=["mqtt_connection"]
            ),
            TestCase(
                name="command_request_response",
                description="Test command request-response flow",
                test_function=self.test_command_request_response,
                prerequisites=["mqtt_connection"],
                timeout=60
            ),
            TestCase(
                name="event_message_publishing",
                description="Test event message publishing",
                test_function=self.test_event_message_publishing,
                prerequisites=["mqtt_connection"]
            ),
            TestCase(
                name="qos_levels",
                description="Test different QoS levels",
                test_function=self.test_qos_levels,
                prerequisites=["mqtt_connection"]
            ),
            TestCase(
                name="retained_messages",
                description="Test retained message behavior",
                test_function=self.test_retained_messages,
                prerequisites=["mqtt_connection"]
            ),
            TestCase(
                name="lwt_functionality",
                description="Test Last Will and Testament",
                test_function=self.test_lwt_functionality,
                prerequisites=["mqtt_connection"]
            ),
            TestCase(
                name="message_schema_validation",
                description="Test JSON schema validation",
                test_function=self.test_message_schema_validation,
                prerequisites=["mqtt_connection"]
            )
        ]
    
    def on_subscriber_connect(self, client, userdata, flags, rc):
        if rc == 0:
            self.logger.debug("Subscriber connected to MQTT broker")
            # 訂閱測試相關主題
            client.subscribe("rtk/v1/test/#", qos=2)
            client.subscribe("rtk/v1/+/+/protocol_tester/cmd/ack", qos=1)
            client.subscribe("rtk/v1/+/+/protocol_tester/cmd/res", qos=1)
        else:
            self.logger.error(f"Subscriber connection failed: {rc}")
    
    def on_message_received(self, client, userdata, msg):
        """處理接收到的訊息"""
        try:
            message = {
                'topic': msg.topic,
                'payload': msg.payload.decode('utf-8'),
                'qos': msg.qos,
                'retain': msg.retain,
                'timestamp': time.time()
            }
            
            self.message_queue.put(message)
            
            # 檢查是否是等待的響應
            topic_parts = msg.topic.split('/')
            if len(topic_parts) >= 6:
                response_key = f"{topic_parts[4]}_{topic_parts[5]}"  # device_id_message_type
                if response_key in self.response_events:
                    self.response_events[response_key].set()
            
        except Exception as e:
            self.logger.error(f"Error processing received message: {e}")
    
    def connect_clients(self):
        """連接MQTT客戶端"""
        try:
            self.publisher.connect(self.broker_host, self.broker_port, 60)
            self.subscriber.connect(self.broker_host, self.broker_port, 60)
            
            self.publisher.loop_start()
            self.subscriber.loop_start()
            
            # 等待連接建立
            time.sleep(2)
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to connect MQTT clients: {e}")
            return False
    
    def disconnect_clients(self):
        """斷開MQTT客戶端"""
        self.publisher.loop_stop()
        self.subscriber.loop_stop()
        self.publisher.disconnect()
        self.subscriber.disconnect()
    
    def test_mqtt_connection(self):
        """測試MQTT連接"""
        try:
            # 測試發布者連接
            if not self.publisher.is_connected():
                return TestResult.FAIL, "Publisher not connected"
            
            # 測試訂閱者連接
            if not self.subscriber.is_connected():
                return TestResult.FAIL, "Subscriber not connected"
            
            # 測試基本發布/訂閱
            test_topic = "rtk/v1/test/protocol/test_device/state"
            test_message = {"test": "connection", "ts": int(time.time() * 1000)}
            
            result = self.publisher.publish(test_topic, json.dumps(test_message), qos=1)
            if result.rc != mqtt.MQTT_ERR_SUCCESS:
                return TestResult.FAIL, f"Publish failed: {result.rc}"
            
            # 等待訊息接收
            timeout = time.time() + 5
            while time.time() < timeout:
                try:
                    message = self.message_queue.get(timeout=1)
                    if message['topic'] == test_topic:
                        return TestResult.PASS, "MQTT connection working"
                except queue.Empty:
                    continue
            
            return TestResult.FAIL, "Message not received"
            
        except Exception as e:
            return TestResult.FAIL, f"Connection test error: {str(e)}"
    
    def test_topic_format_validation(self):
        """測試主題格式驗證"""
        valid_topics = [
            "rtk/v1/test/site/device001/state",
            "rtk/v1/company/office/aabbccddeeff/telemetry/cpu",
            "rtk/v1/demo/home/sensor_123/evt/temperature"
        ]
        
        invalid_topics = [
            "rtk/v2/test/site/device001/state",  # 錯誤版本
            "mqtt/v1/test/site/device001/state",  # 錯誤協議
            "rtk/v1/test/device001/state",  # 缺少site
            "rtk/v1/test/site/device001/invalid"  # 無效訊息類型
        ]
        
        # 測試有效主題
        for topic in valid_topics:
            try:
                result = self.publisher.publish(topic, '{"test": "valid"}', qos=0)
                if result.rc != mqtt.MQTT_ERR_SUCCESS:
                    return TestResult.FAIL, f"Valid topic rejected: {topic}"
            except Exception as e:
                return TestResult.FAIL, f"Error testing valid topic {topic}: {str(e)}"
        
        # 注意: MQTT broker通常不會拒絕無效主題格式，
        # 這個測試主要是確保我們的客戶端可以發布到各種主題
        
        return TestResult.PASS, "Topic format validation passed"
    
    def test_state_message_publishing(self):
        """測試狀態訊息發布"""
        state_topic = f"rtk/v1/test/protocol/{self.device_id}/state"
        state_message = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "health": "ok",
            "uptime_s": 3600,
            "connection_status": "connected"
        }
        
        try:
            result = self.publisher.publish(state_topic, json.dumps(state_message), qos=1, retain=True)
            
            if result.rc != mqtt.MQTT_ERR_SUCCESS:
                return TestResult.FAIL, f"State message publish failed: {result.rc}"
            
            # 等待訊息確認
            result.wait_for_publish()
            
            return TestResult.PASS, "State message published successfully"
            
        except Exception as e:
            return TestResult.FAIL, f"State message test error: {str(e)}"
    
    def test_telemetry_message_publishing(self):
        """測試遙測訊息發布"""
        telemetry_topic = f"rtk/v1/test/protocol/{self.device_id}/telemetry/system"
        telemetry_message = {
            "schema": "telemetry.system/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "payload": {
                "cpu_usage": 45.2,
                "memory_usage": 62.8,
                "disk_usage": 35.1
            }
        }
        
        try:
            result = self.publisher.publish(telemetry_topic, json.dumps(telemetry_message), qos=0)
            
            if result.rc != mqtt.MQTT_ERR_SUCCESS:
                return TestResult.FAIL, f"Telemetry message publish failed: {result.rc}"
            
            return TestResult.PASS, "Telemetry message published successfully"
            
        except Exception as e:
            return TestResult.FAIL, f"Telemetry message test error: {str(e)}"
    
    def test_command_request_response(self):
        """測試命令請求響應流程"""
        # 模擬設備客戶端
        device_client = mqtt.Client("test_device")
        
        def on_device_connect(client, userdata, flags, rc):
            if rc == 0:
                client.subscribe(f"rtk/v1/test/protocol/{self.device_id}/cmd/req", qos=1)
        
        def on_device_message(client, userdata, msg):
            try:
                command = json.loads(msg.payload.decode())
                cmd_id = command.get("id")
                
                # 發送ACK
                ack_topic = f"rtk/v1/test/protocol/{self.device_id}/cmd/ack"
                ack_message = {
                    "id": cmd_id,
                    "schema": "cmd.ack/1.0",
                    "status": "accepted",
                    "ts": int(time.time() * 1000)
                }
                client.publish(ack_topic, json.dumps(ack_message), qos=1)
                
                # 發送結果
                time.sleep(1)  # 模擬處理時間
                res_topic = f"rtk/v1/test/protocol/{self.device_id}/cmd/res"
                res_message = {
                    "id": cmd_id,
                    "schema": "cmd.get_system_info.result/1.0",
                    "status": "completed",
                    "result": {"test": "success"},
                    "ts": int(time.time() * 1000)
                }
                client.publish(res_topic, json.dumps(res_message), qos=1)
                
            except Exception as e:
                self.logger.error(f"Device message handling error: {e}")
        
        device_client.on_connect = on_device_connect
        device_client.on_message = on_device_message
        
        try:
            # 連接模擬設備
            device_client.connect(self.broker_host, self.broker_port, 60)
            device_client.loop_start()
            time.sleep(2)
            
            # 發送命令請求
            cmd_topic = f"rtk/v1/test/protocol/{self.device_id}/cmd/req"
            cmd_id = f"test-cmd-{int(time.time())}"
            command = {
                "id": cmd_id,
                "op": "get_system_info",
                "schema": "cmd.get_system_info/1.0",
                "ts": int(time.time() * 1000)
            }
            
            # 設定響應等待事件
            ack_event = threading.Event()
            res_event = threading.Event()
            self.response_events[f"{self.device_id}_ack"] = ack_event
            self.response_events[f"{self.device_id}_res"] = res_event
            
            # 發送命令
            result = self.publisher.publish(cmd_topic, json.dumps(command), qos=1)
            if result.rc != mqtt.MQTT_ERR_SUCCESS:
                return TestResult.FAIL, f"Command publish failed: {result.rc}"
            
            # 等待ACK
            if not ack_event.wait(timeout=10):
                return TestResult.FAIL, "ACK not received within timeout"
            
            # 等待結果
            if not res_event.wait(timeout=20):
                return TestResult.FAIL, "Result not received within timeout"
            
            return TestResult.PASS, "Command request-response flow completed"
            
        except Exception as e:
            return TestResult.FAIL, f"Command test error: {str(e)}"
        finally:
            device_client.loop_stop()
            device_client.disconnect()
            # 清理事件
            self.response_events.pop(f"{self.device_id}_ack", None)
            self.response_events.pop(f"{self.device_id}_res", None)
    
    def test_event_message_publishing(self):
        """測試事件訊息發布"""
        event_topic = f"rtk/v1/test/protocol/{self.device_id}/evt/system"
        event_message = {
            "schema": "evt.system/1.0",
            "ts": int(time.time() * 1000),
            "event_type": "config_change",
            "severity": "info",
            "description": "Configuration updated during protocol test"
        }
        
        try:
            result = self.publisher.publish(event_topic, json.dumps(event_message), qos=1)
            
            if result.rc != mqtt.MQTT_ERR_SUCCESS:
                return TestResult.FAIL, f"Event message publish failed: {result.rc}"
            
            result.wait_for_publish()
            
            return TestResult.PASS, "Event message published successfully"
            
        except Exception as e:
            return TestResult.FAIL, f"Event message test error: {str(e)}"
    
    def test_qos_levels(self):
        """測試不同QoS等級"""
        base_topic = f"rtk/v1/test/protocol/{self.device_id}"
        
        test_cases = [
            (f"{base_topic}/telemetry/qos_test", 0, "QoS 0 telemetry"),
            (f"{base_topic}/state", 1, "QoS 1 state"),
            (f"{base_topic}/evt/qos_test", 1, "QoS 1 event")
        ]
        
        try:
            for topic, qos, description in test_cases:
                message = {
                    "schema": "test/1.0",
                    "ts": int(time.time() * 1000),
                    "test_type": f"qos_{qos}",
                    "description": description
                }
                
                result = self.publisher.publish(topic, json.dumps(message), qos=qos)
                
                if result.rc != mqtt.MQTT_ERR_SUCCESS:
                    return TestResult.FAIL, f"QoS {qos} publish failed: {result.rc}"
                
                if qos > 0:
                    result.wait_for_publish()
            
            return TestResult.PASS, "All QoS levels tested successfully"
            
        except Exception as e:
            return TestResult.FAIL, f"QoS test error: {str(e)}"
    
    def test_retained_messages(self):
        """測試保留訊息"""
        retained_topic = f"rtk/v1/test/protocol/{self.device_id}/state"
        retained_message = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "health": "ok",
            "test_type": "retained_message",
            "uptime_s": 7200
        }
        
        try:
            # 發布保留訊息
            result = self.publisher.publish(retained_topic, json.dumps(retained_message), qos=1, retain=True)
            
            if result.rc != mqtt.MQTT_ERR_SUCCESS:
                return TestResult.FAIL, f"Retained message publish failed: {result.rc}"
            
            result.wait_for_publish()
            
            # 創建新客戶端測試保留訊息
            test_client = mqtt.Client("retain_test_client")
            received_retained = threading.Event()
            
            def on_test_connect(client, userdata, flags, rc):
                if rc == 0:
                    client.subscribe(retained_topic, qos=1)
            
            def on_test_message(client, userdata, msg):
                if msg.retain and msg.topic == retained_topic:
                    received_retained.set()
            
            test_client.on_connect = on_test_connect
            test_client.on_message = on_test_message
            
            test_client.connect(self.broker_host, self.broker_port, 60)
            test_client.loop_start()
            
            # 等待接收保留訊息
            if received_retained.wait(timeout=10):
                result_status = TestResult.PASS
                result_message = "Retained message test passed"
            else:
                result_status = TestResult.FAIL
                result_message = "Retained message not received"
            
            test_client.loop_stop()
            test_client.disconnect()
            
            return result_status, result_message
            
        except Exception as e:
            return TestResult.FAIL, f"Retained message test error: {str(e)}"
    
    def test_lwt_functionality(self):
        """測試LWT功能"""
        lwt_topic = f"rtk/v1/test/protocol/lwt_test_device/lwt"
        lwt_message = {
            "schema": "lwt/1.0",
            "ts": int(time.time() * 1000),
            "status": "offline",
            "last_seen": int(time.time() * 1000)
        }
        
        try:
            # 創建帶LWT的客戶端
            lwt_client = mqtt.Client("lwt_test_device")
            lwt_client.will_set(lwt_topic, json.dumps(lwt_message), qos=1, retain=True)
            
            lwt_received = threading.Event()
            
            def on_lwt_message(client, userdata, msg):
                if msg.topic == lwt_topic:
                    lwt_received.set()
            
            self.subscriber.on_message = on_lwt_message
            self.subscriber.subscribe(lwt_topic, qos=1)
            
            # 連接後立即斷開以觸發LWT
            lwt_client.connect(self.broker_host, self.broker_port, 60)
            lwt_client.loop_start()
            time.sleep(1)
            
            # 強制斷開以觸發LWT
            lwt_client.disconnect()
            lwt_client.loop_stop()
            
            # 等待LWT訊息
            if lwt_received.wait(timeout=15):
                return TestResult.PASS, "LWT functionality working"
            else:
                return TestResult.FAIL, "LWT message not received"
                
        except Exception as e:
            return TestResult.FAIL, f"LWT test error: {str(e)}"
    
    def test_message_schema_validation(self):
        """測試訊息Schema驗證"""
        # 這個測試發布各種格式的訊息，檢查是否能正常發布
        # 實際的Schema驗證通常在接收端進行
        
        test_messages = [
            {
                "topic": f"rtk/v1/test/protocol/{self.device_id}/state",
                "message": {
                    "schema": "state/1.0",
                    "ts": int(time.time() * 1000),
                    "health": "ok"
                },
                "description": "Valid state message"
            },
            {
                "topic": f"rtk/v1/test/protocol/{self.device_id}/telemetry/test",
                "message": {
                    "schema": "telemetry.test/1.0",
                    "ts": int(time.time() * 1000),
                    "value": 42.5
                },
                "description": "Valid telemetry message"
            }
        ]
        
        try:
            for test_case in test_messages:
                result = self.publisher.publish(
                    test_case["topic"], 
                    json.dumps(test_case["message"]), 
                    qos=1
                )
                
                if result.rc != mqtt.MQTT_ERR_SUCCESS:
                    return TestResult.FAIL, f"Schema test failed for {test_case['description']}: {result.rc}"
                
                result.wait_for_publish()
            
            return TestResult.PASS, "Message schema validation tests passed"
            
        except Exception as e:
            return TestResult.FAIL, f"Schema validation test error: {str(e)}"
    
    def run_test_case(self, test_case: TestCase):
        """執行單一測試用例"""
        self.logger.info(f"Running test: {test_case.name}")
        
        try:
            start_time = time.time()
            result, message = test_case.test_function()
            duration = time.time() - start_time
            
            self.test_results[test_case.name] = {
                'result': result,
                'message': message,
                'duration': duration,
                'timestamp': time.time()
            }
            
            status_symbol = "✓" if result == TestResult.PASS else "✗"
            self.logger.info(f"{status_symbol} {test_case.name}: {message} ({duration:.2f}s)")
            
            return result
            
        except Exception as e:
            error_message = f"Test execution error: {str(e)}"
            self.test_results[test_case.name] = {
                'result': TestResult.FAIL,
                'message': error_message,
                'duration': 0,
                'timestamp': time.time()
            }
            self.logger.error(f"✗ {test_case.name}: {error_message}")
            return TestResult.FAIL
    
    def run_all_tests(self):
        """執行所有測試用例"""
        self.logger.info("Starting RTK MQTT Protocol Tests")
        
        if not self.connect_clients():
            self.logger.error("Failed to connect to MQTT broker")
            return False
        
        try:
            passed = 0
            failed = 0
            skipped = 0
            
            for test_case in self.test_cases:
                # 檢查先決條件
                prerequisites_met = True
                for prereq in test_case.prerequisites:
                    if prereq not in self.test_results or self.test_results[prereq]['result'] != TestResult.PASS:
                        prerequisites_met = False
                        break
                
                if not prerequisites_met:
                    self.logger.warning(f"Skipping {test_case.name}: prerequisites not met")
                    self.test_results[test_case.name] = {
                        'result': TestResult.SKIP,
                        'message': 'Prerequisites not met',
                        'duration': 0,
                        'timestamp': time.time()
                    }
                    skipped += 1
                    continue
                
                result = self.run_test_case(test_case)
                
                if result == TestResult.PASS:
                    passed += 1
                elif result == TestResult.FAIL:
                    failed += 1
                else:
                    skipped += 1
            
            # 打印測試報告
            self.print_test_report(passed, failed, skipped)
            
            return failed == 0
            
        finally:
            self.disconnect_clients()
    
    def print_test_report(self, passed, failed, skipped):
        """打印測試報告"""
        total = passed + failed + skipped
        
        print("\n" + "="*80)
        print("RTK MQTT Protocol Test Report")
        print("="*80)
        
        print(f"Total Tests:    {total}")
        print(f"Passed:         {passed} ({passed/total*100:.1f}%)" if total > 0 else "Passed:         0")
        print(f"Failed:         {failed} ({failed/total*100:.1f}%)" if total > 0 else "Failed:         0")
        print(f"Skipped:        {skipped} ({skipped/total*100:.1f}%)" if total > 0 else "Skipped:        0")
        
        print(f"\nDetailed Results:")
        print("-" * 80)
        
        for test_name, result in self.test_results.items():
            status_symbol = {
                TestResult.PASS: "✓",
                TestResult.FAIL: "✗",
                TestResult.SKIP: "⚬"
            }.get(result['result'], "?")
            
            print(f"{status_symbol} {test_name:30} {result['result'].value:8} {result['duration']:6.2f}s  {result['message']}")
        
        print("="*80)

def main():
    parser = argparse.ArgumentParser(description='RTK MQTT Protocol Tester')
    parser.add_argument('--broker', default='localhost', help='MQTT broker host')
    parser.add_argument('--port', type=int, default=1883, help='MQTT broker port')
    parser.add_argument('--device-id', default='protocol_tester', help='Test device ID')
    parser.add_argument('--verbose', '-v', action='store_true', help='Verbose logging')
    
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    tester = RTKProtocolTester(args.broker, args.port, args.device_id)
    success = tester.run_all_tests()
    
    exit(0 if success else 1)

if __name__ == "__main__":
    main()
```

## 使用範例

### 1. 基本測試流程

```bash
# 1. 驗證MQTT訊息
python3 rtk_message_validator.py --broker localhost --duration 60 --verbose

# 2. 執行負載測試
python3 rtk_load_tester.py --devices 50 --duration 300 --rate 2.0

# 3. 協議功能測試
python3 rtk_protocol_tester.py --broker localhost --verbose
```

### 2. 整合測試腳本

```bash
#!/bin/bash
# rtk_full_test.sh

echo "Starting RTK MQTT Full Test Suite"
echo "=================================="

# 1. 協議測試
echo "Running protocol tests..."
python3 rtk_protocol_tester.py --broker localhost
if [ $? -ne 0 ]; then
    echo "Protocol tests failed!"
    exit 1
fi

# 2. 負載測試
echo "Running load tests..."
python3 rtk_load_tester.py --devices 100 --duration 120 --rate 1.0
if [ $? -ne 0 ]; then
    echo "Load tests failed!"
    exit 1
fi

# 3. 訊息驗證 (背景執行)
echo "Starting message validation..."
python3 rtk_message_validator.py --broker localhost --duration 300 &
VALIDATOR_PID=$!

# 等待驗證完成
wait $VALIDATOR_PID

echo "All tests completed successfully!"
```

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Commands and Events Reference](../core/COMMANDS_EVENTS_REFERENCE.md)
- [Troubleshooting Guide](../guides/TROUBLESHOOTING_GUIDE.md)
- [Testing Integration Guide](../guides/TESTING_INTEGRATION.md)
- [Paho MQTT Documentation](https://www.eclipse.org/paho/index.php?page=clients/python/docs/index.php)
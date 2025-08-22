# RTK MQTT 通用整合指南

## 概述

本指南提供 RTK MQTT 協議的通用整合模式和最佳實踐，適用於所有類型的設備和應用場景。無論你是開發 IoT 感測器、網路設備還是智慧家電，都能從中找到適用的整合模式。

## 🏗️ 整合架構模式

### 1. 基礎整合模式 (Basic Integration)

**適用場景**: 簡單的 IoT 設備、感測器
**複雜度**: ⭐
**功能**: 基本狀態報告和命令響應

```
Device ──── MQTT ──── Controller
      state/telemetry    cmd/req
      ←──── cmd/ack ─────┘
```

### 2. 進階整合模式 (Advanced Integration)

**適用場景**: 網路設備、智慧家電
**複雜度**: ⭐⭐⭐
**功能**: 完整診斷、事件處理、拓撲管理

```
Device ──── MQTT ──── Controller ──── LLM Engine
│     state/telemetry/events       │
│     ←──── cmd/req ────────────────┘
└──── topology/diagnostics ─────────┘
```

### 3. 企業整合模式 (Enterprise Integration)

**適用場景**: 大型網路基礎設施
**複雜度**: ⭐⭐⭐⭐⭐
**功能**: 多租戶、高可用、批次管理

```
Multiple Devices ──── Load Balancer ──── Controller Cluster
     │                     │                    │
   MQTT Pool         Message Queue        Database Cluster
```

## 📋 通用整合檢查清單

### 階段 1: 準備工作 (5 分鐘)

#### ✅ 環境需求確認
- [ ] MQTT Broker 可達性 (ping 測試)
- [ ] 網路連通性 (防火牆規則)
- [ ] 時間同步 (NTP 配置)
- [ ] 設備唯一標識 (MAC 地址獲取)

#### ✅ 基本配置
```json
{
  "mqtt": {
    "broker_host": "mqtt.rtk.local",
    "broker_port": 1883,
    "client_id": "{device_type}-{mac_address}",
    "keep_alive": 60,
    "clean_session": false,
    "qos_default": 1
  },
  "device": {
    "tenant": "your-org",
    "site": "main-office", 
    "device_id": "aabbccddeeff",
    "device_type": "router"
  }
}
```

### 階段 2: 基礎連接 (10 分鐘)

#### ✅ MQTT 連接建立
```python
import paho.mqtt.client as mqtt
import json
import time

class RTKMQTTClient:
    def __init__(self, config):
        self.config = config
        self.client = mqtt.Client(config['mqtt']['client_id'])
        self.connected = False
        
    def connect(self):
        # 設置 LWT
        lwt_topic = f"rtk/v1/{self.config['device']['tenant']}/{self.config['device']['site']}/{self.config['device']['device_id']}/lwt"
        lwt_payload = {
            "schema": "lwt/1.0",
            "ts": int(time.time() * 1000),
            "status": "offline",
            "last_seen": int(time.time() * 1000)
        }
        
        self.client.will_set(lwt_topic, json.dumps(lwt_payload), qos=1, retain=True)
        
        # 連接 callbacks
        self.client.on_connect = self._on_connect
        self.client.on_disconnect = self._on_disconnect
        self.client.on_message = self._on_message
        
        # 建立連接
        self.client.connect(
            self.config['mqtt']['broker_host'],
            self.config['mqtt']['broker_port'],
            self.config['mqtt']['keep_alive']
        )
        
    def _on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            self.connected = True
            print("Connected to MQTT Broker")
            
            # 訂閱命令主題
            cmd_topic = f"rtk/v1/{self.config['device']['tenant']}/{self.config['device']['site']}/{self.config['device']['device_id']}/cmd/req"
            client.subscribe(cmd_topic, qos=1)
            
            # 發送上線狀態
            self._publish_online_status()
            
    def _publish_online_status(self):
        lwt_topic = f"rtk/v1/{self.config['device']['tenant']}/{self.config['device']['site']}/{self.config['device']['device_id']}/lwt"
        online_payload = {
            "schema": "lwt/1.0",
            "ts": int(time.time() * 1000),
            "status": "online"
        }
        self.client.publish(lwt_topic, json.dumps(online_payload), qos=1, retain=True)
```

#### ✅ 設備屬性發布
```python
def publish_device_attributes(self):
    attr_topic = f"rtk/v1/{self.config['device']['tenant']}/{self.config['device']['site']}/{self.config['device']['device_id']}/attr"
    
    attr_payload = {
        "schema": "attr/1.0",
        "ts": int(time.time() * 1000),
        "device_type": self.config['device']['device_type'],
        "manufacturer": "YourCompany",
        "model": "Device-Model-123",
        "firmware_version": "1.0.0",
        "hardware_version": "rev-a",
        "serial_number": "SN123456789",
        "mac_address": self.config['device']['device_id'],
        "capabilities": ["wifi", "ethernet", "diagnostics"]
    }
    
    self.client.publish(attr_topic, json.dumps(attr_payload), qos=1, retain=True)
```

### 階段 3: 狀態管理 (15 分鐘)

#### ✅ 定期狀態報告
```python
def publish_device_state(self):
    state_topic = f"rtk/v1/{self.config['device']['tenant']}/{self.config['device']['site']}/{self.config['device']['device_id']}/state"
    
    state_payload = {
        "schema": "state/1.0",
        "ts": int(time.time() * 1000),
        "health": "ok",  # ok, warning, error
        "uptime_s": self._get_uptime(),
        "cpu_usage": self._get_cpu_usage(),
        "memory_usage": self._get_memory_usage(),
        "connection_status": "connected",
        "last_command_ts": self.last_command_ts
    }
    
    self.client.publish(state_topic, json.dumps(state_payload), qos=1, retain=True)

# 每 5 分鐘自動發送
def start_state_reporting(self):
    def state_loop():
        while self.connected:
            self.publish_device_state()
            time.sleep(300)  # 5 minutes
    
    import threading
    threading.Thread(target=state_loop, daemon=True).start()
```

#### ✅ Telemetry 數據發送
```python
def publish_telemetry(self, metric_name, value, unit=None):
    telemetry_topic = f"rtk/v1/{self.config['device']['tenant']}/{self.config['device']['site']}/{self.config['device']['device_id']}/telemetry/{metric_name}"
    
    telemetry_payload = {
        "schema": f"telemetry.{metric_name}/1.0",
        "ts": int(time.time() * 1000),
        "value": value,
        "unit": unit
    }
    
    self.client.publish(telemetry_topic, json.dumps(telemetry_payload), qos=0)

# 使用範例
client.publish_telemetry("cpu_usage", 45.2, "%")
client.publish_telemetry("temperature", 68.5, "celsius")
client.publish_telemetry("network_throughput", 85.3, "mbps")
```

### 階段 4: 命令處理 (20 分鐘)

#### ✅ 命令處理框架
```python
def _on_message(self, client, userdata, msg):
    try:
        # 解析命令
        command = json.loads(msg.payload.decode())
        
        # 發送 ACK
        self._send_command_ack(command['payload']['id'])
        
        # 處理命令
        result = self._execute_command(command)
        
        # 發送結果
        self._send_command_result(command['payload']['id'], result)
        
    except Exception as e:
        # 發送錯誤結果
        self._send_command_error(command['payload']['id'], str(e))

def _send_command_ack(self, command_id):
    ack_topic = f"rtk/v1/{self.config['device']['tenant']}/{self.config['device']['site']}/{self.config['device']['device_id']}/cmd/ack"
    
    ack_payload = {
        "schema": "cmd.ack/1.0",
        "ts": int(time.time() * 1000),
        "id": command_id,
        "status": "received"
    }
    
    self.client.publish(ack_topic, json.dumps(ack_payload), qos=1)

def _send_command_result(self, command_id, result):
    res_topic = f"rtk/v1/{self.config['device']['tenant']}/{self.config['device']['site']}/{self.config['device']['device_id']}/cmd/res"
    
    res_payload = {
        "schema": "cmd.result/1.0",
        "ts": int(time.time() * 1000),
        "id": command_id,
        "status": "completed",
        "result": result
    }
    
    self.client.publish(res_topic, json.dumps(res_payload), qos=1)
```

#### ✅ 命令執行器
```python
def _execute_command(self, command):
    op = command['payload']['op']
    args = command['payload'].get('args', {})
    
    # 命令路由
    if op == "device.status":
        return self._cmd_device_status(args)
    elif op == "speed_test":
        return self._cmd_speed_test(args)
    elif op == "restart":
        return self._cmd_restart(args)
    else:
        raise ValueError(f"Unknown command: {op}")

def _cmd_device_status(self, args):
    return {
        "uptime": self._get_uptime(),
        "cpu_usage": self._get_cpu_usage(),
        "memory_usage": self._get_memory_usage(),
        "network_status": "connected"
    }

def _cmd_speed_test(self, args):
    # 實際的速度測試實作
    duration = args.get('duration', 30)
    # ... 執行速度測試 ...
    return {
        "download_mbps": 85.2,
        "upload_mbps": 12.4,
        "latency_ms": 15.3,
        "test_duration": duration
    }
```

### 階段 5: 事件通知 (25 分鐘)

#### ✅ 事件發送機制
```python
def publish_event(self, event_type, event_data):
    event_topic = f"rtk/v1/{self.config['device']['tenant']}/{self.config['device']['site']}/{self.config['device']['device_id']}/evt/{event_type}"
    
    event_payload = {
        "schema": f"evt.{event_type}/1.0",
        "ts": int(time.time() * 1000),
        "device_id": self.config['device']['device_id'],
        "event_type": event_type,
        "data": event_data
    }
    
    self.client.publish(event_topic, json.dumps(event_payload), qos=1)

# 常見事件範例
def _on_network_change(self, interface, status):
    self.publish_event("network.link_change", {
        "interface": interface,
        "status": status,
        "timestamp": int(time.time() * 1000)
    })

def _on_error_detected(self, error_type, description):
    self.publish_event("system.error", {
        "error_type": error_type,
        "description": description,
        "severity": "high"
    })
```

## 🔧 進階整合模式

### 拓撲管理整合
```python
def publish_topology_update(self, topology_data):
    topology_topic = f"rtk/v1/{self.config['device']['tenant']}/{self.config['device']['site']}/{self.config['device']['device_id']}/topology/update"
    
    topology_payload = {
        "schema": "topology.update/1.0",
        "ts": int(time.time() * 1000),
        "discovered_devices": topology_data['devices'],
        "connections": topology_data['connections'],
        "changes": topology_data.get('changes', [])
    }
    
    self.client.publish(topology_topic, json.dumps(topology_payload), qos=1)
```

### 批次命令處理
```python
def _execute_batch_commands(self, commands):
    results = []
    for cmd in commands:
        try:
            result = self._execute_command(cmd)
            results.append({"command_id": cmd['id'], "status": "success", "result": result})
        except Exception as e:
            results.append({"command_id": cmd['id'], "status": "error", "error": str(e)})
    return results
```

## 🛡️ 錯誤處理和可靠性

### 重連機制
```python
def _on_disconnect(self, client, userdata, rc):
    self.connected = False
    print(f"Disconnected with result code {rc}")
    
    # 自動重連
    while not self.connected:
        try:
            time.sleep(5)
            self.client.reconnect()
        except Exception as e:
            print(f"Reconnection failed: {e}")

def _setup_reconnection(self):
    self.client.reconnect_delay_set(min_delay=1, max_delay=120)
    self.client.loop_start()  # 啟動自動重連
```

### 訊息持久化
```python
import sqlite3

class MessageBuffer:
    def __init__(self, db_path="rtk_mqtt_buffer.db"):
        self.conn = sqlite3.connect(db_path)
        self._create_tables()
    
    def _create_tables(self):
        self.conn.execute('''
            CREATE TABLE IF NOT EXISTS pending_messages (
                id INTEGER PRIMARY KEY,
                topic TEXT,
                payload TEXT,
                qos INTEGER,
                timestamp INTEGER
            )
        ''')
    
    def store_message(self, topic, payload, qos):
        self.conn.execute(
            "INSERT INTO pending_messages (topic, payload, qos, timestamp) VALUES (?, ?, ?, ?)",
            (topic, payload, qos, int(time.time() * 1000))
        )
        self.conn.commit()
    
    def get_pending_messages(self):
        cursor = self.conn.execute("SELECT topic, payload, qos FROM pending_messages")
        return cursor.fetchall()
    
    def clear_sent_messages(self):
        self.conn.execute("DELETE FROM pending_messages")
        self.conn.commit()
```

## 📊 效能最佳化

### 訊息頻率控制
```python
class RateLimiter:
    def __init__(self, max_messages_per_minute=60):
        self.max_messages = max_messages_per_minute
        self.messages = []
    
    def can_send(self):
        now = time.time()
        # 移除 1 分鐘前的記錄
        self.messages = [msg_time for msg_time in self.messages if now - msg_time < 60]
        
        if len(self.messages) < self.max_messages:
            self.messages.append(now)
            return True
        return False

# 使用範例
rate_limiter = RateLimiter(max_messages_per_minute=30)

def safe_publish(self, topic, payload, qos=1):
    if self.rate_limiter.can_send():
        self.client.publish(topic, payload, qos)
    else:
        # 暫存訊息或記錄警告
        self.message_buffer.store_message(topic, payload, qos)
```

### 記憶體管理
```python
def _cleanup_old_data(self):
    # 定期清理過期數據
    cutoff_time = time.time() - 3600  # 1 hour ago
    
    # 清理事件緩存
    self.event_cache = {k: v for k, v in self.event_cache.items() 
                       if v['timestamp'] > cutoff_time}
    
    # 清理命令歷史
    self.command_history = [cmd for cmd in self.command_history 
                           if cmd['timestamp'] > cutoff_time]
```

## 🔒 安全最佳實踐

### TLS/SSL 配置
```python
def _setup_ssl(self):
    self.client.tls_set(
        ca_certs="ca.crt",
        certfile="client.crt", 
        keyfile="client.key",
        cert_reqs=ssl.CERT_REQUIRED,
        tls_version=ssl.PROTOCOL_TLS,
        ciphers=None
    )
```

### 訊息加密
```python
from cryptography.fernet import Fernet

class MessageEncryption:
    def __init__(self, key=None):
        self.key = key or Fernet.generate_key()
        self.cipher = Fernet(self.key)
    
    def encrypt_payload(self, payload):
        return self.cipher.encrypt(payload.encode()).decode()
    
    def decrypt_payload(self, encrypted_payload):
        return self.cipher.decrypt(encrypted_payload.encode()).decode()
```

## 📈 監控和除錯

### 診斷資訊收集
```python
class DiagnosticsCollector:
    def __init__(self):
        self.metrics = {
            'messages_sent': 0,
            'messages_received': 0,
            'connection_errors': 0,
            'last_heartbeat': None
        }
    
    def record_message_sent(self):
        self.metrics['messages_sent'] += 1
    
    def record_connection_error(self):
        self.metrics['connection_errors'] += 1
    
    def get_diagnostics(self):
        return {
            **self.metrics,
            'uptime': time.time() - self.start_time,
            'memory_usage': self._get_memory_usage()
        }
```

## ✅ 整合驗證檢查清單

### 基礎功能驗證
- [ ] MQTT 連接成功建立
- [ ] LWT 訊息正確設置
- [ ] 設備屬性成功發布
- [ ] 狀態訊息定期發送
- [ ] 命令訂閱正常工作
- [ ] 命令 ACK/結果正確發送

### 可靠性驗證
- [ ] 網路中斷後自動重連
- [ ] 訊息緩存和重發機制
- [ ] 錯誤處理流程完整
- [ ] 記憶體使用控制在合理範圍

### 效能驗證
- [ ] 訊息延遲在可接受範圍 (<100ms)
- [ ] CPU 使用率穩定 (<20%)
- [ ] 記憶體使用無洩漏
- [ ] 網路頻寬使用合理

### 安全性驗證
- [ ] TLS 加密連接 (如適用)
- [ ] 認證機制正常
- [ ] 敏感資料加密
- [ ] 訪問權限控制

## 📚 相關資源

- **[設備專用指南](devices/)** - 選擇你的設備類型
- **[API 完整參考](core/MQTT_API_REFERENCE.md)** - 所有命令和事件
- **[測試指南](guides/TESTING_INTEGRATION.md)** - 測試策略和工具
- **[故障排除](guides/TROUBLESHOOTING_GUIDE.md)** - 常見問題解決

---

完成這個整合指南後，你的設備將具備完整的 RTK MQTT 功能，包括可靠的通信、智慧的診斷能力和強大的網路管理功能。
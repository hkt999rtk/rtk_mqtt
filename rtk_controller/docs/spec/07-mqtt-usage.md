# MQTT 使用時機與最佳實踐

## 概述

本文檔提供 RTK MQTT 協議的使用時機指南和最佳實踐，包括發布頻率、QoS 設定、訂閱模式、連接管理等重要考量。正確的 MQTT 使用策略能確保系統的可靠性、效能和擴展性。

## 發布頻率指南

### 基本原則
- **狀態優先**: 重要狀態變化應立即發布
- **頻率適中**: 避免過於頻繁造成網路負擔
- **條件觸發**: 結合定時發布和狀態變化觸發
- **分層傳輸**: 根據重要性調整發布頻率

### 訊息類型頻率表

| 訊息類型 | 建議頻率 | 觸發條件 | 重要性 |
|---------|---------|---------|--------|
| `state` | 30-60 秒 | 健康狀態變化 | 高 |
| `telemetry/critical` | 10-30 秒 | 閾值變化 | 高 |
| `telemetry/normal` | 60-300 秒 | 定時發送 | 中 |
| `telemetry/background` | 300-600 秒 | 定時發送 | 低 |
| `evt/*` | 即時 | 事件發生 | 高 |
| `attr` | 啟動時 | 屬性變更 | 中 |
| `topology/discovery` | 300 秒 | 拓撲變化 | 中 |
| `topology/connections` | 120 秒 | 連接狀態變化 | 中 |

### 詳細頻率建議

#### 狀態資料 (state)
```json
{
  "normal_frequency": "60s",
  "warning_frequency": "30s", 
  "error_frequency": "10s",
  "triggers": [
    "health_status_change",
    "critical_resource_threshold",
    "connection_status_change"
  ]
}
```

#### 遙測資料 (telemetry)
```json
{
  "critical_metrics": {
    "frequency": "10s",
    "metrics": ["temperature", "cpu_usage", "memory_usage"],
    "threshold_change": 5
  },
  "normal_metrics": {
    "frequency": "60s", 
    "metrics": ["disk_usage", "network_traffic", "wifi_clients"]
  },
  "background_metrics": {
    "frequency": "300s",
    "metrics": ["system_logs", "statistics", "historical_data"]
  }
}
```

#### 事件資料 (evt)
```json
{
  "immediate_events": [
    "system.error",
    "hardware.fault", 
    "network.disconnected",
    "security.breach"
  ],
  "deferred_events": [
    "system.warning",
    "performance.degraded",
    "maintenance.scheduled"
  ]
}
```

## QoS 設定策略

### QoS 等級選擇

| QoS | 說明 | 適用場景 | 效能影響 |
|-----|------|----------|----------|
| 0 | 最多傳送一次 | 高頻遙測資料 | 最低 |
| 1 | 至少傳送一次 | 狀態、事件、命令 | 中等 |
| 2 | 恰好傳送一次 | 關鍵命令、金融交易 | 最高 |

### 訊息類型 QoS 映射

```json
{
  "qos_mapping": {
    "state": 1,
    "telemetry/critical": 1,
    "telemetry/normal": 0,
    "telemetry/background": 0,
    "evt/critical": 1,
    "evt/warning": 1,
    "evt/info": 0,
    "attr": 1,
    "cmd/req": 1,
    "cmd/ack": 1,
    "cmd/res": 1,
    "topology": 1,
    "lwt": 1
  }
}
```

### QoS 選擇邏輯
```python
def select_qos(message_type, severity=None, frequency=None):
    """選擇適當的 QoS 等級"""
    
    # 關鍵訊息必須使用 QoS 1
    critical_types = ['state', 'cmd', 'evt', 'attr', 'lwt']
    if any(msg_type in message_type for msg_type in critical_types):
        return 1
    
    # 高頻遙測使用 QoS 0 以提高效能
    if 'telemetry' in message_type and frequency and frequency < 30:
        return 0
        
    # 嚴重事件使用 QoS 1
    if severity in ['error', 'critical', 'warning']:
        return 1
        
    # 預設使用 QoS 0
    return 0
```

## 訂閱模式設計

### Controller 訂閱模式

#### 全域監控訂閱
```bash
# 基本監控
rtk/v1/+/+/+/state              # 所有設備狀態
rtk/v1/+/+/+/lwt                # 所有設備上下線
rtk/v1/+/+/+/evt/#              # 所有事件

# 分類監控
rtk/v1/+/+/+/evt/system.#       # 系統事件
rtk/v1/+/+/+/evt/hardware.#     # 硬體事件
rtk/v1/+/+/+/evt/network.#      # 網路事件
rtk/v1/+/+/+/telemetry/cpu_usage # CPU 使用率
```

#### 租戶級監控訂閱
```bash
# 特定租戶
rtk/v1/office/+/+/state         # 辦公室所有設備
rtk/v1/factory/+/+/evt/#        # 工廠所有事件

# 特定場域
rtk/v1/office/floor1/+/evt/#    # 辦公室 1 樓事件
rtk/v1/factory/workshop-a/+/telemetry/temperature # A車間溫度
```

#### 設備類型監控訂閱
```bash
# 按設備類型
rtk/v1/+/+/router-+/topology/#  # 所有路由器拓撲
rtk/v1/+/+/ap-+/evt/wifi.#      # 所有 AP 的 WiFi 事件
rtk/v1/+/+/sensor-+/telemetry/# # 所有感測器遙測
```

### Device 訂閱模式

#### 基本訂閱 (必須)
```bash
# 設備專用命令通道
rtk/v1/{tenant}/{site}/{device_id}/cmd/req
```

#### 群組訂閱 (可選)
```bash
# 設備群組命令
rtk/v1/{tenant}/{site}/group/{group_id}/cmd/req

# 廣播命令
rtk/v1/{tenant}/broadcast/cmd/req
rtk/v1/broadcast/cmd/req
```

### 訂閱最佳實踐

#### 1. 避免過寬訂閱
```bash
# ❌ 避免 - 可能造成大量流量
rtk/v1/+/+/+/telemetry/#

# ✅ 建議 - 具體的訂閱
rtk/v1/office/+/+/telemetry/temperature
rtk/v1/+/+/+/evt/system.error
```

#### 2. 分層訂閱策略
```python
class SubscriptionManager:
    def __init__(self):
        self.subscriptions = {
            'critical': [
                'rtk/v1/+/+/+/evt/system.error',
                'rtk/v1/+/+/+/evt/hardware.fault',
                'rtk/v1/+/+/+/state'
            ],
            'important': [
                'rtk/v1/+/+/+/evt/network.#',
                'rtk/v1/+/+/+/topology/discovery'
            ],
            'monitoring': [
                'rtk/v1/office/+/+/telemetry/cpu_usage',
                'rtk/v1/factory/+/+/telemetry/temperature'
            ]
        }
    
    def subscribe_by_priority(self, client, priority_level):
        """按優先級訂閱"""
        if priority_level == 'critical':
            for topic in self.subscriptions['critical']:
                client.subscribe(topic, qos=1)
```

## 連接管理最佳實踐

### Keep Alive 設定

```json
{
  "device_types": {
    "high_frequency": {
      "keep_alive": 30,
      "devices": ["sensor", "monitor"],
      "reason": "頻繁通信，短間隔檢測"
    },
    "normal": {
      "keep_alive": 60, 
      "devices": ["router", "switch", "ap"],
      "reason": "標準設備，平衡可靠性與資源"
    },
    "low_power": {
      "keep_alive": 300,
      "devices": ["battery_sensor", "iot_device"],
      "reason": "省電考量，延長電池壽命"
    }
  }
}
```

### Clean Session 策略

```python
def get_clean_session_setting(device_type, use_case):
    """決定 Clean Session 設定"""
    
    # 需要訊息持久化的場景
    persistent_scenarios = [
        'critical_monitoring',
        'command_control',
        'data_logging'
    ]
    
    # 臨時連接場景
    temporary_scenarios = [
        'testing',
        'debugging', 
        'one_time_query'
    ]
    
    if use_case in persistent_scenarios:
        return False  # 保持會話狀態
    elif use_case in temporary_scenarios:
        return True   # 清除會話狀態
    else:
        # 根據設備類型決定
        return device_type in ['test_device', 'temporary_sensor']
```

### 重連策略

```python
class ReconnectionManager:
    def __init__(self):
        self.retry_intervals = [1, 2, 4, 8, 16, 32, 60]  # 指數退避
        self.max_retries = 10
        
    def calculate_retry_delay(self, attempt):
        """計算重連延遲時間"""
        if attempt < len(self.retry_intervals):
            return self.retry_intervals[attempt]
        return self.retry_intervals[-1]  # 最大間隔
        
    def should_retry(self, attempt, error_type):
        """判斷是否應該重試"""
        # 網路錯誤 - 應該重試
        if error_type in ['network_error', 'timeout', 'connection_refused']:
            return attempt < self.max_retries
            
        # 認證錯誤 - 不應重試
        if error_type in ['authentication_failed', 'authorization_denied']:
            return False
            
        return attempt < 3  # 其他錯誤最多重試 3 次
```

## 效能最佳化

### 訊息批次處理

```python
class MessageBatcher:
    def __init__(self, batch_size=10, flush_interval=5):
        self.batch_size = batch_size
        self.flush_interval = flush_interval
        self.message_queue = []
        self.last_flush = time.time()
        
    def add_message(self, topic, payload, qos=0):
        """添加訊息到批次"""
        self.message_queue.append({
            'topic': topic,
            'payload': payload,
            'qos': qos,
            'timestamp': time.time()
        })
        
        # 檢查是否需要刷新
        if (len(self.message_queue) >= self.batch_size or 
            time.time() - self.last_flush > self.flush_interval):
            self.flush_batch()
            
    def flush_batch(self):
        """發送批次訊息"""
        for message in self.message_queue:
            self.client.publish(
                message['topic'],
                message['payload'], 
                qos=message['qos']
            )
        
        self.message_queue.clear()
        self.last_flush = time.time()
```

### 流量控制

```python
class TrafficController:
    def __init__(self, max_messages_per_second=100):
        self.max_rate = max_messages_per_second
        self.message_count = 0
        self.window_start = time.time()
        
    def can_send_message(self):
        """檢查是否可以發送訊息"""
        current_time = time.time()
        
        # 重置計數器（每秒）
        if current_time - self.window_start >= 1.0:
            self.message_count = 0
            self.window_start = current_time
            
        # 檢查是否超過限制
        if self.message_count >= self.max_rate:
            return False
            
        self.message_count += 1
        return True
```

## 錯誤處理和恢復

### 連接狀態監控

```python
class ConnectionMonitor:
    def __init__(self, client):
        self.client = client
        self.last_ping = time.time()
        self.ping_interval = 30
        self.health_status = 'connected'
        
    def monitor_connection(self):
        """監控連接健康狀態"""
        current_time = time.time()
        
        if current_time - self.last_ping > self.ping_interval:
            try:
                # 發送 ping 或簡單的狀態訊息
                self.client.publish(
                    f"rtk/v1/{tenant}/{site}/{device_id}/ping",
                    json.dumps({"ts": int(current_time * 1000)}),
                    qos=0
                )
                self.last_ping = current_time
                self.health_status = 'connected'
                
            except Exception as e:
                self.health_status = 'disconnected'
                self.handle_connection_error(e)
                
    def handle_connection_error(self, error):
        """處理連接錯誤"""
        # 記錄錯誤
        logger.error(f"Connection error: {error}")
        
        # 嘗試重連
        self.attempt_reconnection()
```

### 訊息去重機制

```python
class MessageDeduplicator:
    def __init__(self, cache_size=1000, ttl_seconds=300):
        self.message_cache = {}
        self.cache_size = cache_size
        self.ttl = ttl_seconds
        
    def is_duplicate(self, message_id):
        """檢查訊息是否重複"""
        current_time = time.time()
        
        # 清理過期記錄
        self.cleanup_expired(current_time)
        
        # 檢查是否重複
        if message_id in self.message_cache:
            return True
            
        # 添加到快取
        self.message_cache[message_id] = current_time
        
        # 控制快取大小
        if len(self.message_cache) > self.cache_size:
            self.cleanup_oldest()
            
        return False
        
    def cleanup_expired(self, current_time):
        """清理過期記錄"""
        expired_keys = [
            key for key, timestamp in self.message_cache.items()
            if current_time - timestamp > self.ttl
        ]
        for key in expired_keys:
            del self.message_cache[key]
```

## 安全性考量

### 認證和授權

```python
class MQTTSecurityManager:
    def __init__(self):
        self.device_credentials = {}
        self.topic_permissions = {}
        
    def authenticate_device(self, client_id, username, password):
        """設備認證"""
        if client_id not in self.device_credentials:
            return False
            
        stored_creds = self.device_credentials[client_id]
        return (stored_creds['username'] == username and 
                stored_creds['password'] == password)
                
    def authorize_topic(self, client_id, topic, action):
        """主題授權檢查"""
        if client_id not in self.topic_permissions:
            return False
            
        permissions = self.topic_permissions[client_id]
        
        # 檢查設備是否有權限操作該主題
        for pattern in permissions.get(action, []):
            if self.topic_matches_pattern(topic, pattern):
                return True
                
        return False
        
    def topic_matches_pattern(self, topic, pattern):
        """檢查主題是否匹配權限模式"""
        # 實作主題匹配邏輯
        # 支援萬用字元 + 和 #
        pass
```

### TLS 配置

```python
def configure_tls(client, ca_cert_path, client_cert_path, client_key_path):
    """配置 TLS 安全連接"""
    client.tls_set(
        ca_certs=ca_cert_path,
        certfile=client_cert_path, 
        keyfile=client_key_path,
        cert_reqs=ssl.CERT_REQUIRED,
        tls_version=ssl.PROTOCOL_TLSv1_2
    )
    
    # 驗證主機名
    client.tls_insecure_set(False)
```

## 監控和除錯

### 效能指標收集

```python
class MQTTMetrics:
    def __init__(self):
        self.metrics = {
            'messages_sent': 0,
            'messages_received': 0,
            'bytes_sent': 0,
            'bytes_received': 0,
            'connection_errors': 0,
            'reconnections': 0,
            'average_latency_ms': 0
        }
        
    def record_message_sent(self, topic, payload_size):
        """記錄發送的訊息"""
        self.metrics['messages_sent'] += 1
        self.metrics['bytes_sent'] += payload_size
        
    def record_message_received(self, topic, payload_size):
        """記錄接收的訊息"""
        self.metrics['messages_received'] += 1
        self.metrics['bytes_received'] += payload_size
        
    def get_metrics_report(self):
        """生成指標報告"""
        return {
            'timestamp': time.time(),
            'metrics': self.metrics.copy(),
            'rates': {
                'messages_per_second': self.calculate_message_rate(),
                'bytes_per_second': self.calculate_byte_rate()
            }
        }
```

---

**下一步**: 閱讀 [診斷協議](10-diagnostics-protocol.md) 了解網路診斷和 LLM 整合功能的詳細規格。
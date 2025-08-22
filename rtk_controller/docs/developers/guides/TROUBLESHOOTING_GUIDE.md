# RTK MQTT 故障排除指南

## 概述

本指南提供RTK MQTT系統常見問題的診斷和解決方案，包含系統性的故障排除流程、診斷工具和解決步驟。

## 故障分類

### 連接問題
- MQTT Broker連接失敗
- 設備斷線問題
- 網路連通性問題
- TLS/SSL認證失敗

### 協議問題
- 訊息格式錯誤
- Schema驗證失敗
- 主題結構不正確
- QoS設定問題

### 性能問題
- 高延遲
- 訊息丟失
- 記憶體洩漏
- CPU使用率過高

### 配置問題
- 設定檔錯誤
- 環境變數問題
- 權限設定錯誤
- 版本相容性問題

## 診斷工具

### 系統診斷腳本

```bash
#!/bin/bash
# rtk_diagnostics.sh

set -e

echo "RTK MQTT System Diagnostics"
echo "============================"
echo "Timestamp: $(date)"
echo "Hostname: $(hostname)"
echo "User: $(whoami)"
echo ""

# 基本系統資訊
echo "1. System Information"
echo "---------------------"
echo "OS: $(uname -a)"
echo "Uptime: $(uptime)"
echo "Load Average: $(uptime | awk -F'load average:' '{print $2}')"
echo ""

# 網路連通性
echo "2. Network Connectivity"
echo "-----------------------"
# 檢查DNS解析
if nslookup google.com >/dev/null 2>&1; then
    echo "✓ DNS resolution working"
else
    echo "✗ DNS resolution failed"
fi

# 檢查外網連通性
if ping -c 1 8.8.8.8 >/dev/null 2>&1; then
    echo "✓ Internet connectivity working"
else
    echo "✗ Internet connectivity failed"
fi

# 檢查本地端口
echo "Active ports:"
netstat -ln | grep -E "(1883|8080|8083|9090|3000)" || echo "No RTK MQTT ports found"
echo ""

# MQTT Broker診斷
echo "3. MQTT Broker Status"
echo "---------------------"
check_mqtt_broker() {
    local host="${1:-localhost}"
    local port="${2:-1883}"
    
    if timeout 5 bash -c "</dev/tcp/$host/$port"; then
        echo "✓ MQTT broker reachable at $host:$port"
        
        # 測試發布/訂閱
        if command -v mosquitto_pub >/dev/null && command -v mosquitto_sub >/dev/null; then
            TEST_TOPIC="rtk/test/diagnostic/$(date +%s)"
            TEST_MESSAGE="diagnostic_test_$(date +%s)"
            
            # 背景訂閱
            timeout 10 mosquitto_sub -h "$host" -p "$port" -t "$TEST_TOPIC" -C 1 >/tmp/mqtt_test 2>&1 &
            SUB_PID=$!
            
            sleep 2
            
            # 發布測試訊息
            if mosquitto_pub -h "$host" -p "$port" -t "$TEST_TOPIC" -m "$TEST_MESSAGE"; then
                sleep 2
                if grep -q "$TEST_MESSAGE" /tmp/mqtt_test 2>/dev/null; then
                    echo "✓ MQTT publish/subscribe working"
                else
                    echo "✗ MQTT subscribe failed"
                fi
            else
                echo "✗ MQTT publish failed"
            fi
            
            kill $SUB_PID 2>/dev/null || true
            rm -f /tmp/mqtt_test
        else
            echo "! mosquitto_pub/sub not available for testing"
        fi
    else
        echo "✗ MQTT broker not reachable at $host:$port"
    fi
}

check_mqtt_broker "localhost" "1883"
echo ""

# RTK Controller診斷
echo "4. RTK Controller Status"
echo "------------------------"
check_rtk_controller() {
    local host="${1:-localhost}"
    local port="${2:-8080}"
    
    if curl -sf "http://$host:$port/health" >/dev/null 2>&1; then
        echo "✓ RTK Controller health endpoint responding"
        
        # 獲取詳細狀態
        HEALTH_DATA=$(curl -s "http://$host:$port/health" 2>/dev/null)
        if [ -n "$HEALTH_DATA" ]; then
            echo "Health data: $HEALTH_DATA"
        fi
        
        # 檢查metrics端點
        if curl -sf "http://$host:$port/metrics" >/dev/null 2>&1; then
            echo "✓ Metrics endpoint responding"
        else
            echo "! Metrics endpoint not available"
        fi
    else
        echo "✗ RTK Controller not responding at http://$host:$port"
    fi
}

check_rtk_controller "localhost" "8080"
echo ""

# 容器狀態 (如果使用Docker)
echo "5. Container Status"
echo "-------------------"
if command -v docker >/dev/null; then
    if docker ps | grep -E "(mosquitto|rtk)" >/dev/null; then
        echo "RTK MQTT containers:"
        docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "(mosquitto|rtk|NAME)"
        
        echo ""
        echo "Container resource usage:"
        docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep -E "(mosquitto|rtk|NAME)"
    else
        echo "No RTK MQTT containers found"
    fi
else
    echo "Docker not available"
fi
echo ""

# 日誌檢查
echo "6. Recent Log Analysis"
echo "----------------------"
check_logs() {
    local log_file="$1"
    local description="$2"
    
    if [ -f "$log_file" ]; then
        echo "$description:"
        echo "  Last modified: $(stat -c %y "$log_file" 2>/dev/null || stat -f %Sm "$log_file" 2>/dev/null || echo "unknown")"
        echo "  Size: $(du -h "$log_file" | cut -f1)"
        
        # 檢查最近的錯誤
        ERRORS=$(grep -i "error\|fail\|exception" "$log_file" 2>/dev/null | tail -3)
        if [ -n "$ERRORS" ]; then
            echo "  Recent errors:"
            echo "$ERRORS" | sed 's/^/    /'
        else
            echo "  No recent errors found"
        fi
    else
        echo "$description: Log file not found at $log_file"
    fi
}

# 檢查常見日誌位置
check_logs "/var/log/mosquitto/mosquitto.log" "Mosquitto Broker"
check_logs "/app/logs/rtk_controller.log" "RTK Controller"
check_logs "/var/log/syslog" "System Log (filtered)"

# Docker日誌檢查
if command -v docker >/dev/null; then
    echo ""
    echo "Docker container logs (last 5 lines):"
    for container in $(docker ps --format "{{.Names}}" | grep -E "(mosquitto|rtk)"); do
        echo "  $container:"
        docker logs "$container" --tail 5 2>&1 | sed 's/^/    /'
    done
fi
echo ""

# 配置檔案檢查
echo "7. Configuration Files"
echo "----------------------"
check_config() {
    local config_file="$1"
    local description="$2"
    
    if [ -f "$config_file" ]; then
        echo "$description:"
        echo "  File: $config_file"
        echo "  Size: $(du -h "$config_file" | cut -f1)"
        echo "  Permissions: $(ls -l "$config_file" | awk '{print $1, $3, $4}')"
        
        # 語法檢查 (針對YAML)
        if [[ "$config_file" == *.yaml ]] || [[ "$config_file" == *.yml ]]; then
            if command -v python3 >/dev/null; then
                if python3 -c "import yaml; yaml.safe_load(open('$config_file'))" 2>/dev/null; then
                    echo "  ✓ YAML syntax valid"
                else
                    echo "  ✗ YAML syntax error"
                fi
            fi
        fi
    else
        echo "$description: File not found at $config_file"
    fi
}

check_config "/app/configs/controller.yaml" "RTK Controller Config"
check_config "/mosquitto/config/mosquitto.conf" "Mosquitto Config"
check_config "/app/configs/dev.yaml" "Development Config"
echo ""

# 環境變數檢查
echo "8. Environment Variables"
echo "------------------------"
env | grep -E "^RTK_|^MQTT_" | sort || echo "No RTK/MQTT environment variables found"
echo ""

# 磁碟空間檢查
echo "9. Disk Space Analysis"
echo "----------------------"
df -h | head -1
df -h | grep -E "(/|/app|/mosquitto|/var/log)" || df -h | head -2
echo ""

# 記憶體分析
echo "10. Memory Analysis"
echo "-------------------"
free -h
echo ""
echo "Top memory consumers:"
ps aux --sort=-%mem | head -6
echo ""

# 網路連接分析
echo "11. Network Connections"
echo "-----------------------"
echo "MQTT-related connections:"
netstat -an | grep -E ":1883|:8080|:8083" | head -10
echo ""

# 最終建議
echo "12. Diagnostic Summary and Recommendations"
echo "-------------------------------------------"

# 檢查關鍵服務狀態
CRITICAL_ISSUES=0

if ! timeout 5 bash -c "</dev/tcp/localhost/1883" 2>/dev/null; then
    echo "🔴 CRITICAL: MQTT Broker not accessible"
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + 1))
fi

if ! curl -sf "http://localhost:8080/health" >/dev/null 2>&1; then
    echo "🔴 CRITICAL: RTK Controller not responding"
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + 1))
fi

DISK_USAGE=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
if [ "$DISK_USAGE" -gt 90 ]; then
    echo "🟡 WARNING: Disk usage critical (${DISK_USAGE}%)"
fi

MEMORY_USAGE=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
if [ "$MEMORY_USAGE" -gt 90 ]; then
    echo "🟡 WARNING: Memory usage high (${MEMORY_USAGE}%)"
fi

if [ $CRITICAL_ISSUES -eq 0 ]; then
    echo "✅ No critical issues detected"
else
    echo "⚠️  Found $CRITICAL_ISSUES critical issue(s) - immediate attention required"
fi

echo ""
echo "Diagnostic completed at $(date)"
echo "Report saved to: rtk_diagnostic_$(date +%Y%m%d_%H%M%S).log"
}

# 執行診斷並保存到檔案
rtk_diagnostics | tee "rtk_diagnostic_$(date +%Y%m%d_%H%M%S).log"
```

### MQTT訊息監控工具

```python
#!/usr/bin/env python3
# mqtt_monitor.py

import paho.mqtt.client as mqtt
import json
import time
import sys
from datetime import datetime
from collections import defaultdict, Counter
import argparse

class RTKMQTTMonitor:
    def __init__(self, broker_host="localhost", broker_port=1883):
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.client = mqtt.Client("rtk_monitor")
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        
        # 統計數據
        self.message_count = 0
        self.message_types = Counter()
        self.device_activity = defaultdict(lambda: {"last_seen": None, "message_count": 0})
        self.error_count = 0
        self.start_time = time.time()
        
        # 配置
        self.verbose = False
        self.show_payload = False
        self.filter_topic = None
        self.filter_device = None
        
    def on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            print(f"✓ Connected to MQTT broker at {self.broker_host}:{self.broker_port}")
            # 訂閱所有RTK主題
            client.subscribe("rtk/v1/#", qos=0)
            client.subscribe("$SYS/#", qos=0)  # 系統主題
        else:
            print(f"✗ Connection failed with code {rc}")
            
    def on_message(self, client, userdata, msg):
        try:
            self.process_message(msg)
        except Exception as e:
            self.error_count += 1
            if self.verbose:
                print(f"Error processing message: {e}")
                
    def process_message(self, msg):
        topic = msg.topic
        payload = msg.payload.decode('utf-8', errors='ignore')
        timestamp = datetime.now().strftime("%H:%M:%S")
        
        # 應用過濾器
        if self.filter_topic and self.filter_topic not in topic:
            return
            
        # 解析RTK主題
        if topic.startswith("rtk/v1/"):
            self.process_rtk_message(topic, payload, timestamp)
        elif topic.startswith("$SYS/"):
            self.process_system_message(topic, payload, timestamp)
            
        self.message_count += 1
        
    def process_rtk_message(self, topic, payload, timestamp):
        parts = topic.split('/')
        if len(parts) >= 6:
            tenant = parts[2]
            site = parts[3]
            device_id = parts[4]
            message_type = parts[5]
            sub_type = parts[6] if len(parts) > 6 else ""
            
            # 應用設備過濾器
            if self.filter_device and self.filter_device != device_id:
                return
                
            # 更新設備活動
            device_key = f"{tenant}/{site}/{device_id}"
            self.device_activity[device_key]["last_seen"] = timestamp
            self.device_activity[device_key]["message_count"] += 1
            
            # 統計訊息類型
            full_type = f"{message_type}/{sub_type}" if sub_type else message_type
            self.message_types[full_type] += 1
            
            # 顯示訊息
            if self.verbose:
                print(f"[{timestamp}] {device_key} -> {full_type}")
                
                if self.show_payload:
                    try:
                        # 嘗試格式化JSON
                        json_data = json.loads(payload)
                        print(f"  Payload: {json.dumps(json_data, indent=2)}")
                    except:
                        print(f"  Payload: {payload[:100]}...")
                        
                # 檢查訊息錯誤
                self.check_message_validity(topic, payload, json_data if 'json_data' in locals() else None)
                
    def process_system_message(self, topic, payload, timestamp):
        if self.verbose and "broker" in topic:
            print(f"[{timestamp}] SYSTEM: {topic} = {payload}")
            
    def check_message_validity(self, topic, payload, json_data):
        """檢查訊息有效性"""
        issues = []
        
        # 檢查JSON格式
        if json_data is None:
            try:
                json_data = json.loads(payload)
            except:
                issues.append("Invalid JSON format")
                return issues
                
        # 檢查必要欄位
        if "schema" not in json_data:
            issues.append("Missing 'schema' field")
            
        if "ts" not in json_data:
            issues.append("Missing 'ts' field")
        elif not isinstance(json_data["ts"], int):
            issues.append("'ts' field must be integer")
            
        # 檢查時間戳合理性
        if "ts" in json_data:
            msg_time = json_data["ts"] / 1000
            current_time = time.time()
            if abs(current_time - msg_time) > 300:  # 5分鐘容忍度
                issues.append("Timestamp too old or in future")
                
        # 檢查主題結構
        if not topic.startswith("rtk/v1/"):
            issues.append("Invalid topic prefix")
            
        if issues:
            print(f"  ⚠️  Issues found: {', '.join(issues)}")
            
        return issues
        
    def print_statistics(self):
        """打印統計資訊"""
        runtime = time.time() - self.start_time
        
        print("\n" + "="*60)
        print("RTK MQTT Monitor Statistics")
        print("="*60)
        print(f"Runtime: {runtime:.1f} seconds")
        print(f"Total messages: {self.message_count}")
        print(f"Message rate: {self.message_count/runtime:.2f} msg/s")
        print(f"Error count: {self.error_count}")
        
        print(f"\nTop message types:")
        for msg_type, count in self.message_types.most_common(10):
            percentage = (count / self.message_count * 100) if self.message_count > 0 else 0
            print(f"  {msg_type:20} {count:6} ({percentage:5.1f}%)")
            
        print(f"\nActive devices ({len(self.device_activity)}):")
        for device, activity in sorted(self.device_activity.items())[:10]:
            print(f"  {device:30} {activity['message_count']:6} messages, last: {activity['last_seen']}")
            
        if len(self.device_activity) > 10:
            print(f"  ... and {len(self.device_activity) - 10} more devices")
            
    def run(self, duration=None):
        """運行監控器"""
        try:
            print(f"Starting RTK MQTT monitor...")
            print(f"Broker: {self.broker_host}:{self.broker_port}")
            print(f"Filters: topic='{self.filter_topic}', device='{self.filter_device}'")
            print("Press Ctrl+C to stop\n")
            
            self.client.connect(self.broker_host, self.broker_port, 60)
            self.client.loop_start()
            
            if duration:
                time.sleep(duration)
            else:
                while True:
                    time.sleep(1)
                    
        except KeyboardInterrupt:
            print("\nShutting down monitor...")
        finally:
            self.client.loop_stop()
            self.client.disconnect()
            self.print_statistics()

def main():
    parser = argparse.ArgumentParser(description='RTK MQTT Message Monitor')
    parser.add_argument('--host', default='localhost', help='MQTT broker host')
    parser.add_argument('--port', type=int, default=1883, help='MQTT broker port')
    parser.add_argument('--verbose', '-v', action='store_true', help='Verbose output')
    parser.add_argument('--payload', '-p', action='store_true', help='Show message payloads')
    parser.add_argument('--topic', '-t', help='Filter by topic pattern')
    parser.add_argument('--device', '-d', help='Filter by device ID')
    parser.add_argument('--duration', type=int, help='Run for specified seconds')
    
    args = parser.parse_args()
    
    monitor = RTKMQTTMonitor(args.host, args.port)
    monitor.verbose = args.verbose
    monitor.show_payload = args.payload
    monitor.filter_topic = args.topic
    monitor.filter_device = args.device
    
    monitor.run(args.duration)

if __name__ == "__main__":
    main()
```

## 常見問題解決方案

### 1. MQTT連接問題

#### 問題: 無法連接到MQTT Broker

**症狀:**
```
Connection failed with result code 1
```

**診斷步驟:**
```bash
# 1. 檢查broker是否運行
sudo systemctl status mosquitto
# 或
docker ps | grep mosquitto

# 2. 檢查端口是否開放
netstat -ln | grep 1883
telnet localhost 1883

# 3. 檢查防火牆
sudo ufw status
sudo iptables -L | grep 1883

# 4. 檢查broker日誌
sudo tail -f /var/log/mosquitto/mosquitto.log
# 或
docker logs mosquitto
```

**解決方案:**
```bash
# 重啟Mosquitto服務
sudo systemctl restart mosquitto

# 檢查配置文件語法
sudo mosquitto -c /etc/mosquitto/mosquitto.conf -v

# 開放防火牆端口
sudo ufw allow 1883
sudo ufw allow 8083
```

#### 問題: TLS連接失敗

**症狀:**
```
SSL/TLS handshake failed
Certificate verification failed
```

**診斷步驟:**
```bash
# 1. 測試TLS連接
openssl s_client -connect localhost:8883 -servername localhost

# 2. 檢查憑證
openssl x509 -in /certs/server.crt -text -noout

# 3. 驗證憑證鏈
openssl verify -CAfile /certs/ca.crt /certs/server.crt
```

**解決方案:**
```bash
# 重新生成憑證
./generate_certs.sh

# 檢查憑證權限
chmod 644 /certs/*.crt
chmod 600 /certs/*.key

# 更新CA憑證儲存
sudo update-ca-certificates
```

### 2. 訊息格式問題

#### 問題: JSON Schema驗證失敗

**症狀:**
```
Schema validation error: 'ts' is a required property
Invalid message format
```

**診斷工具:**
```python
#!/usr/bin/env python3
# validate_message.py

import json
import jsonschema
import sys

def validate_rtk_message(message_str, schema_file):
    try:
        # 解析訊息
        message = json.loads(message_str)
        
        # 載入schema
        with open(schema_file, 'r') as f:
            schema = json.load(f)
            
        # 驗證
        jsonschema.validate(message, schema)
        print("✓ Message is valid")
        
    except json.JSONDecodeError as e:
        print(f"✗ JSON parsing error: {e}")
    except jsonschema.ValidationError as e:
        print(f"✗ Schema validation error: {e}")
    except FileNotFoundError:
        print(f"✗ Schema file not found: {schema_file}")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 validate_message.py '<json_message>' <schema_file>")
        sys.exit(1)
        
    validate_rtk_message(sys.argv[1], sys.argv[2])
```

**使用範例:**
```bash
# 驗證狀態訊息
python3 validate_message.py '{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "health": "ok",
  "uptime_s": 3600
}' schemas/state_schema.json
```

#### 問題: 主題結構錯誤

**症狀:**
```
Invalid topic format
Topic does not match RTK pattern
```

**檢查腳本:**
```bash
#!/bin/bash
# check_topic_format.sh

check_topic() {
    local topic="$1"
    local pattern="rtk/v1/[a-zA-Z0-9_]+/[a-zA-Z0-9_]+/[a-zA-Z0-9:_-]+/(state|telemetry|evt|attr|cmd|lwt|topology)"
    
    if [[ $topic =~ $pattern ]]; then
        echo "✓ Valid topic: $topic"
        
        # 解析組件
        IFS='/' read -ra PARTS <<< "$topic"
        echo "  Protocol: ${PARTS[0]}"
        echo "  Version: ${PARTS[1]}"
        echo "  Tenant: ${PARTS[2]}"
        echo "  Site: ${PARTS[3]}"
        echo "  Device: ${PARTS[4]}"
        echo "  Type: ${PARTS[5]}"
        if [ ${#PARTS[@]} -gt 6 ]; then
            echo "  Subtype: ${PARTS[6]}"
        fi
    else
        echo "✗ Invalid topic: $topic"
        echo "  Expected format: rtk/v1/{tenant}/{site}/{device_id}/{message_type}[/{sub_type}]"
    fi
}

# 測試範例
check_topic "rtk/v1/demo/office/device_001/state"
check_topic "rtk/v1/company/site1/aabbccddeeff/telemetry/cpu"
check_topic "invalid/topic/format"
```

### 3. 性能問題

#### 問題: 高延遲

**症狀:**
```
Command response time > 5 seconds
High message processing delay
```

**診斷工具:**
```python
#!/usr/bin/env python3
# latency_test.py

import paho.mqtt.client as mqtt
import json
import time
import threading
from queue import Queue

class LatencyTester:
    def __init__(self, broker_host="localhost"):
        self.broker_host = broker_host
        self.response_queue = Queue()
        self.results = []
        
    def test_round_trip_latency(self, iterations=10):
        # 設置響應監聽
        response_client = mqtt.Client("latency_response_listener")
        response_client.on_message = self.on_response
        response_client.connect(self.broker_host, 1883, 60)
        response_client.subscribe("rtk/v1/test/latency/+/cmd/res", qos=1)
        response_client.loop_start()
        
        # 設置命令發送
        command_client = mqtt.Client("latency_command_sender")
        command_client.connect(self.broker_host, 1883, 60)
        
        print(f"Testing latency with {iterations} iterations...")
        
        for i in range(iterations):
            cmd_id = f"latency-test-{i}"
            start_time = time.time()
            
            command = {
                "id": cmd_id,
                "op": "get_system_info",
                "schema": "cmd.get_system_info/1.0",
                "ts": int(time.time() * 1000)
            }
            
            command_client.publish(
                "rtk/v1/test/latency/test_device/cmd/req",
                json.dumps(command),
                qos=1
            )
            
            # 等待響應
            try:
                response = self.response_queue.get(timeout=10)
                if response["id"] == cmd_id:
                    latency = (time.time() - start_time) * 1000
                    self.results.append(latency)
                    print(f"  Iteration {i+1}: {latency:.2f}ms")
            except:
                print(f"  Iteration {i+1}: TIMEOUT")
                
            time.sleep(0.5)
            
        # 清理
        response_client.loop_stop()
        response_client.disconnect()
        command_client.disconnect()
        
        # 統計
        if self.results:
            avg = sum(self.results) / len(self.results)
            min_lat = min(self.results)
            max_lat = max(self.results)
            
            print(f"\nLatency Statistics:")
            print(f"  Average: {avg:.2f}ms")
            print(f"  Min: {min_lat:.2f}ms")
            print(f"  Max: {max_lat:.2f}ms")
            print(f"  Successful: {len(self.results)}/{iterations}")
            
    def on_response(self, client, userdata, msg):
        try:
            response = json.loads(msg.payload.decode())
            self.response_queue.put(response)
        except:
            pass

if __name__ == "__main__":
    tester = LatencyTester()
    tester.test_round_trip_latency(20)
```

**性能調優:**
```bash
# 系統層級優化
echo 'net.core.rmem_max = 16777216' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 16777216' >> /etc/sysctl.conf
sysctl -p

# Mosquitto優化
cat >> /etc/mosquitto/mosquitto.conf << EOF
max_connections 2000
max_queued_messages 1000
message_size_limit 1048576
keepalive_interval 60
EOF

# 重啟服務
sudo systemctl restart mosquitto
```

#### 問題: 記憶體洩漏

**症狀:**
```
Memory usage continuously increasing
OOM killer activated
```

**監控腳本:**
```bash
#!/bin/bash
# memory_monitor.sh

PROCESS_NAME="rtk_controller"
LOG_FILE="memory_usage.log"

echo "Monitoring memory usage for $PROCESS_NAME"
echo "Time,PID,RSS(MB),VSZ(MB),CPU%" > $LOG_FILE

while true; do
    TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
    
    # 查找進程
    PID=$(pgrep -f $PROCESS_NAME | head -1)
    
    if [ -n "$PID" ]; then
        # 獲取記憶體使用
        MEMORY_INFO=$(ps -p $PID -o pid,rss,vsz,pcpu --no-headers 2>/dev/null)
        
        if [ -n "$MEMORY_INFO" ]; then
            read PID RSS VSZ CPU <<< "$MEMORY_INFO"
            RSS_MB=$((RSS / 1024))
            VSZ_MB=$((VSZ / 1024))
            
            echo "$TIMESTAMP,$PID,$RSS_MB,$VSZ_MB,$CPU" >> $LOG_FILE
            echo "[$TIMESTAMP] PID:$PID RSS:${RSS_MB}MB VSZ:${VSZ_MB}MB CPU:${CPU}%"
            
            # 警告閾值
            if [ $RSS_MB -gt 1000 ]; then
                echo "WARNING: High memory usage detected!"
            fi
        else
            echo "[$TIMESTAMP] Process $PROCESS_NAME not found"
        fi
    else
        echo "[$TIMESTAMP] Process $PROCESS_NAME not running"
    fi
    
    sleep 60
done
```

### 4. 配置問題

#### 問題: 環境變數未設定

**症狀:**
```
Configuration error: required environment variable not set
Failed to load configuration
```

**檢查腳本:**
```bash
#!/bin/bash
# check_env.sh

echo "RTK MQTT Environment Check"
echo "=========================="

# 必要環境變數
REQUIRED_VARS=(
    "RTK_MQTT_BROKER"
    "RTK_DATA_DIR"
    "RTK_LOG_LEVEL"
)

# 可選環境變數
OPTIONAL_VARS=(
    "RTK_ENV"
    "RTK_JWT_SECRET"
    "RTK_MQTT_USERNAME"
    "RTK_MQTT_PASSWORD"
)

ERROR_COUNT=0

echo "Required environment variables:"
for var in "${REQUIRED_VARS[@]}"; do
    if [ -n "${!var}" ]; then
        echo "  ✓ $var = ${!var}"
    else
        echo "  ✗ $var = NOT SET"
        ERROR_COUNT=$((ERROR_COUNT + 1))
    fi
done

echo ""
echo "Optional environment variables:"
for var in "${OPTIONAL_VARS[@]}"; do
    if [ -n "${!var}" ]; then
        # 隱藏敏感資訊
        if [[ $var == *"PASSWORD"* ]] || [[ $var == *"SECRET"* ]]; then
            echo "  ✓ $var = [HIDDEN]"
        else
            echo "  ✓ $var = ${!var}"
        fi
    else
        echo "  - $var = NOT SET"
    fi
done

echo ""
if [ $ERROR_COUNT -eq 0 ]; then
    echo "✅ All required environment variables are set"
else
    echo "❌ $ERROR_COUNT required environment variables are missing"
    echo ""
    echo "To fix, run:"
    for var in "${REQUIRED_VARS[@]}"; do
        if [ -z "${!var}" ]; then
            echo "  export $var=<value>"
        fi
    done
fi
```

#### 問題: 配置檔案語法錯誤

**症狀:**
```
YAML parsing error
Configuration validation failed
```

**驗證腳本:**
```python
#!/usr/bin/env python3
# validate_config.py

import yaml
import sys
import os

def validate_yaml_config(config_file):
    """驗證YAML配置檔案"""
    try:
        with open(config_file, 'r') as f:
            config = yaml.safe_load(f)
        
        print(f"✓ YAML syntax is valid for {config_file}")
        
        # 檢查必要配置節
        required_sections = ['mqtt', 'database', 'logging']
        missing_sections = []
        
        for section in required_sections:
            if section not in config:
                missing_sections.append(section)
        
        if missing_sections:
            print(f"⚠️  Missing required sections: {', '.join(missing_sections)}")
        else:
            print("✓ All required sections present")
            
        # 檢查MQTT配置
        if 'mqtt' in config:
            mqtt_config = config['mqtt']
            required_mqtt = ['broker_host', 'broker_port']
            
            for field in required_mqtt:
                if field not in mqtt_config:
                    print(f"⚠️  Missing MQTT field: {field}")
                    
        # 檢查路徑存在性
        if 'database' in config and 'path' in config['database']:
            db_path = config['database']['path']
            db_dir = os.path.dirname(db_path)
            
            if not os.path.exists(db_dir):
                print(f"⚠️  Database directory does not exist: {db_dir}")
            else:
                print(f"✓ Database directory exists: {db_dir}")
                
        return True
        
    except yaml.YAMLError as e:
        print(f"✗ YAML syntax error in {config_file}:")
        print(f"  {e}")
        return False
        
    except FileNotFoundError:
        print(f"✗ Configuration file not found: {config_file}")
        return False
        
    except Exception as e:
        print(f"✗ Unexpected error validating {config_file}: {e}")
        return False

def main():
    if len(sys.argv) != 2:
        print("Usage: python3 validate_config.py <config_file>")
        sys.exit(1)
        
    config_file = sys.argv[1]
    
    if validate_yaml_config(config_file):
        print(f"\n✅ Configuration file {config_file} is valid")
        sys.exit(0)
    else:
        print(f"\n❌ Configuration file {config_file} has errors")
        sys.exit(1)

if __name__ == "__main__":
    main()
```

## 預防措施

### 1. 健康檢查自動化

```bash
#!/bin/bash
# automated_health_check.sh

SLACK_WEBHOOK="YOUR_SLACK_WEBHOOK_URL"
EMAIL_RECIPIENT="admin@yourcompany.com"

send_alert() {
    local message="$1"
    local severity="$2"
    
    echo "[$(date)] ALERT: $message"
    
    # Slack通知
    if [ -n "$SLACK_WEBHOOK" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"🚨 RTK MQTT Alert: $message\"}" \
            "$SLACK_WEBHOOK"
    fi
    
    # Email通知
    if command -v mail >/dev/null && [ -n "$EMAIL_RECIPIENT" ]; then
        echo "$message" | mail -s "RTK MQTT Alert - $severity" "$EMAIL_RECIPIENT"
    fi
}

# 檢查關鍵服務
check_critical_services() {
    # MQTT Broker
    if ! timeout 5 bash -c "</dev/tcp/localhost/1883" 2>/dev/null; then
        send_alert "MQTT Broker is not responding on port 1883" "CRITICAL"
        return 1
    fi
    
    # RTK Controller
    if ! curl -sf "http://localhost:8080/health" >/dev/null 2>&1; then
        send_alert "RTK Controller health check failed" "CRITICAL"
        return 1
    fi
    
    return 0
}

# 檢查資源使用
check_resource_usage() {
    # 磁碟空間
    DISK_USAGE=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
    if [ "$DISK_USAGE" -gt 90 ]; then
        send_alert "Disk usage critical: ${DISK_USAGE}%" "WARNING"
    fi
    
    # 記憶體使用
    MEMORY_USAGE=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
    if [ "$MEMORY_USAGE" -gt 90 ]; then
        send_alert "Memory usage critical: ${MEMORY_USAGE}%" "WARNING"
    fi
}

# 主檢查流程
main() {
    echo "Starting automated health check at $(date)"
    
    if check_critical_services && check_resource_usage; then
        echo "All checks passed"
    else
        echo "Some checks failed - alerts sent"
    fi
}

# 每5分鐘執行一次
# 添加到crontab: */5 * * * * /path/to/automated_health_check.sh
main
```

### 2. 日誌輪轉設定

```bash
# /etc/logrotate.d/rtk_mqtt
/app/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    postrotate
        # 重新載入日誌配置
        docker kill -s USR1 rtk-controller 2>/dev/null || true
    endscript
}

/mosquitto/log/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    postrotate
        # 重新載入Mosquitto日誌
        docker kill -s HUP mosquitto 2>/dev/null || true
    endscript
}
```

### 3. 監控指標配置

```yaml
# monitoring/alerts.yml
groups:
  - name: rtk_mqtt_preventive
    rules:
    - alert: DiskSpaceWarning
      expr: (node_filesystem_avail_bytes / node_filesystem_size_bytes) < 0.2
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Disk space running low"
        
    - alert: MemoryWarning
      expr: (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) < 0.1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Available memory running low"
        
    - alert: HighMessageLatency
      expr: histogram_quantile(0.95, rate(rtk_command_duration_seconds_bucket[5m])) > 2
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "High message latency detected"
```

## 緊急恢復程序

### 1. 快速重啟流程

```bash
#!/bin/bash
# emergency_restart.sh

echo "EMERGENCY RESTART PROCEDURE"
echo "=========================="
echo "This will restart all RTK MQTT services"
read -p "Continue? (y/N): " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Starting emergency restart..."
    
    # 停止服務
    echo "1. Stopping services..."
    docker-compose down
    
    # 等待清理
    echo "2. Waiting for cleanup..."
    sleep 10
    
    # 檢查端口釋放
    echo "3. Checking ports..."
    if netstat -ln | grep -E "(1883|8080)" >/dev/null; then
        echo "WARNING: Ports still in use, waiting longer..."
        sleep 20
    fi
    
    # 重啟服務
    echo "4. Starting services..."
    docker-compose up -d
    
    # 等待服務就緒
    echo "5. Waiting for services..."
    sleep 30
    
    # 驗證服務
    echo "6. Verifying services..."
    ./health_check.sh
    
    echo "Emergency restart completed"
else
    echo "Emergency restart cancelled"
fi
```

### 2. 配置回滾程序

```bash
#!/bin/bash
# rollback_config.sh

BACKUP_DIR="/backup/configs"
CONFIG_DIR="/app/configs"

# 列出可用備份
list_backups() {
    echo "Available configuration backups:"
    ls -la "$BACKUP_DIR"/config_backup_*.tar.gz 2>/dev/null | \
        awk '{print $9}' | \
        sed 's/.*config_backup_\(.*\)\.tar\.gz/\1/' | \
        sort -r | head -10
}

# 回滾到指定備份
rollback_to_backup() {
    local backup_date="$1"
    local backup_file="$BACKUP_DIR/config_backup_${backup_date}.tar.gz"
    
    if [ ! -f "$backup_file" ]; then
        echo "Error: Backup file not found: $backup_file"
        return 1
    fi
    
    echo "Rolling back to configuration from $backup_date"
    
    # 創建當前配置備份
    tar -czf "$BACKUP_DIR/config_backup_before_rollback_$(date +%Y%m%d_%H%M%S).tar.gz" -C "$CONFIG_DIR" .
    
    # 恢復備份
    rm -rf "$CONFIG_DIR"/*
    tar -xzf "$backup_file" -C "$CONFIG_DIR"
    
    echo "Configuration rolled back successfully"
    echo "Please restart services to apply changes"
}

# 主程序
if [ -z "$1" ]; then
    echo "Usage: $0 <backup_date>"
    echo ""
    list_backups
else
    rollback_to_backup "$1"
fi
```

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Quick Start Guide](QUICK_START_GUIDE.md)
- [Deployment Guide](DEPLOYMENT_GUIDE.md)
- [Testing Integration Guide](TESTING_INTEGRATION.md)
- [MQTT Broker Troubleshooting](https://mosquitto.org/documentation/troubleshooting/)
- [Docker Troubleshooting](https://docs.docker.com/config/troubleshooting/)
- [System Performance Analysis](https://github.com/brendangregg/perf-tools)
# RTK MQTT 認證流程

## 概述

RTK MQTT 認證流程確保所有整合的設備和應用程式符合協議標準，保證互操作性、安全性和效能品質。本文檔詳細說明認證要求、測試程序和申請流程。

## 🏆 認證級別

### RTK Basic 認證
**適用對象**: 基礎 IoT 設備、感測器
**要求**:
- 基本 MQTT 連接和訊息處理
- 標準狀態報告和屬性發布
- 基礎命令響應能力

### RTK Advanced 認證  
**適用對象**: 網路設備、智慧家電
**要求**:
- 完整的診斷功能支援
- 進階事件處理和拓撲管理
- QoS 和效能最佳化

### RTK Enterprise 認證
**適用對象**: 企業級解決方案、關鍵基礎設施
**要求**:
- 高可用性和容錯能力
- 企業級安全性功能
- 大規模部署支援

## 📋 認證要求

### 1. 協議相容性要求

#### 🔗 基礎連接 (所有級別)
- [ ] **MQTT 3.1.1 相容性**: 完整支援 MQTT 3.1.1 協議
- [ ] **客戶端識別**: 使用標準格式的客戶端 ID (`rtk-{device_id}`)
- [ ] **Keep Alive**: 支援 60 秒心跳間隔
- [ ] **LWT 機制**: 正確實作 Last Will Testament
- [ ] **QoS 支援**: 支援 QoS 0 和 QoS 1

#### 📡 訊息格式 (所有級別)
- [ ] **主題結構**: 遵循 `rtk/v1/{tenant}/{site}/{device_id}/{message_type}` 格式
- [ ] **JSON Schema**: 所有訊息符合官方 JSON Schema 規範
- [ ] **時間戳**: 使用 Unix 毫秒時間戳
- [ ] **Schema 版本**: 正確使用 schema 版本標識

#### 📊 狀態管理 (所有級別)
- [ ] **設備屬性**: 啟動時發布完整的 `attr` 訊息
- [ ] **狀態報告**: 定期發送 `state` 訊息 (≤ 5 分鐘間隔)
- [ ] **上線/離線**: 正確處理連接狀態變化
- [ ] **健康指標**: 提供 CPU、記憶體等基本指標

### 2. 功能性要求

#### ⚡ 基礎命令支援 (Basic+)
- [ ] **device.status**: 設備狀態查詢
- [ ] **restart**: 設備重啟 (如適用)
- [ ] **get_system_info**: 系統資訊查詢
- [ ] **命令確認**: 所有命令必須發送 ACK 和結果

#### 🔧 進階功能 (Advanced+)
- [ ] **診斷命令**: 支援相關的診斷操作
- [ ] **配置管理**: 支援遠端配置更新
- [ ] **事件通知**: 主動發送相關事件
- [ ] **拓撲資訊**: 提供網路拓撲資料 (網路設備)

#### 🏢 企業功能 (Enterprise)
- [ ] **批次操作**: 支援批次命令處理
- [ ] **變更集管理**: 支援原子性配置變更
- [ ] **安全模式**: 支援 TLS/SSL 加密
- [ ] **權限控制**: 實作存取權限管理

### 3. 效能要求

#### ⚡ 回應時間
| 操作類型 | Basic | Advanced | Enterprise |
|----------|-------|----------|------------|
| 命令 ACK | < 1s | < 500ms | < 200ms |
| 狀態查詢 | < 5s | < 3s | < 1s |
| 診斷測試 | < 30s | < 20s | < 10s |
| 配置變更 | < 10s | < 5s | < 3s |

#### 📈 吞吐量
| 指標 | Basic | Advanced | Enterprise |
|------|-------|----------|------------|
| 訊息/秒 | ≥ 10 | ≥ 100 | ≥ 1000 |
| 並發連接 | ≥ 100 | ≥ 1000 | ≥ 10000 |
| 資料保留 | 24h | 7d | 30d |

#### 💾 資源使用
| 資源 | Basic | Advanced | Enterprise |
|------|-------|----------|------------|
| RAM | < 64MB | < 256MB | < 1GB |
| CPU | < 20% | < 30% | < 50% |
| 網路 | < 1Mbps | < 10Mbps | < 100Mbps |

### 4. 安全性要求

#### 🔒 基礎安全 (Basic+)
- [ ] **認證支援**: 支援用戶名/密碼認證
- [ ] **資料驗證**: 輸入資料驗證和清理
- [ ] **錯誤處理**: 安全的錯誤處理，不洩露敏感資訊

#### 🛡️ 進階安全 (Advanced+)
- [ ] **TLS 支援**: 支援 TLS 1.2+ 加密連接
- [ ] **憑證驗證**: 支援客戶端憑證認證
- [ ] **存取控制**: 實作基於角色的存取控制

#### 🏰 企業安全 (Enterprise)
- [ ] **端到端加密**: 支援訊息層級加密
- [ ] **稽核日誌**: 完整的操作稽核記錄
- [ ] **合規性**: 符合相關安全合規要求

## 🧪 測試程序

### 自動化測試套件

#### 基礎相容性測試
```python
#!/usr/bin/env python3
# certification/basic_compatibility_test.py

import unittest
import time
import json
import paho.mqtt.client as mqtt
from datetime import datetime, timedelta

class RTKBasicCompatibilityTest(unittest.TestCase):
    
    def setUp(self):
        self.broker_host = "test.rtk-mqtt.com"
        self.broker_port = 1883
        self.device_id = "cert-test-001"
        self.tenant = "certification"
        self.site = "test"
        
        # 測試結果收集
        self.received_messages = []
        self.test_results = {}
        
    def test_01_connection_establishment(self):
        """測試 MQTT 連接建立"""
        client = mqtt.Client(f"rtk-{self.device_id}")
        
        # 設置 LWT
        lwt_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/lwt"
        lwt_payload = {
            "schema": "lwt/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.device_id,
            "status": "offline"
        }
        client.will_set(lwt_topic, json.dumps(lwt_payload), qos=1, retain=True)
        
        # 連接測試
        result = client.connect(self.broker_host, self.broker_port, 60)
        self.assertEqual(result, 0, "MQTT connection should succeed")
        
        client.disconnect()
        
    def test_02_device_attributes(self):
        """測試設備屬性發布"""
        client = mqtt.Client(f"rtk-{self.device_id}")
        client.connect(self.broker_host, self.broker_port, 60)
        
        attr_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/attr"
        attr_payload = {
            "schema": "attr/1.0",
            "ts": int(time.time() * 1000),
            "device_type": "test_device",
            "manufacturer": "RTK Test",
            "model": "Test-001",
            "firmware_version": "1.0.0",
            "capabilities": ["basic_commands"]
        }
        
        result = client.publish(attr_topic, json.dumps(attr_payload), qos=1, retain=True)
        self.assertTrue(result.is_published(), "Device attributes should be published")
        
        client.disconnect()
        
    def test_03_state_reporting(self):
        """測試狀態報告"""
        client = mqtt.Client(f"rtk-{self.device_id}")
        client.connect(self.broker_host, self.broker_port, 60)
        
        state_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/state"
        state_payload = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "health": "ok",
            "connection_status": "connected",
            "cpu_usage": 25.4,
            "memory_usage": 45.2
        }
        
        result = client.publish(state_topic, json.dumps(state_payload), qos=1, retain=True)
        self.assertTrue(result.is_published(), "State should be published")
        
        client.disconnect()
        
    def test_04_command_handling(self):
        """測試命令處理"""
        # 設置命令監聽
        client = mqtt.Client(f"rtk-{self.device_id}")
        client.on_message = self._on_command_message
        
        client.connect(self.broker_host, self.broker_port, 60)
        
        # 訂閱命令主題
        cmd_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/req"
        client.subscribe(cmd_topic, qos=1)
        
        # 模擬命令發送 (需要外部測試工具)
        client.loop_start()
        time.sleep(10)  # 等待命令
        client.loop_stop()
        
        # 驗證收到命令
        self.assertTrue(len(self.received_messages) > 0, "Should receive commands")
        
        client.disconnect()
        
    def _on_command_message(self, client, userdata, msg):
        """命令訊息處理"""
        try:
            command = json.loads(msg.payload.decode())
            self.received_messages.append(command)
            
            # 發送 ACK
            payload = command['payload']
            cmd_id = payload['id']
            
            ack_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/ack"
            ack_payload = {
                "schema": "cmd.ack/1.0",
                "ts": int(time.time() * 1000),
                "device_id": self.device_id,
                "payload": {
                    "id": cmd_id,
                    "status": "received"
                }
            }
            
            client.publish(ack_topic, json.dumps(ack_payload), qos=1)
            
        except Exception as e:
            self.fail(f"Command processing failed: {e}")

if __name__ == "__main__":
    unittest.main()
```

#### 效能測試
```python
#!/usr/bin/env python3
# certification/performance_test.py

import time
import threading
import statistics
from concurrent.futures import ThreadPoolExecutor
import paho.mqtt.client as mqtt

class RTKPerformanceTest:
    
    def __init__(self, broker_host, broker_port, device_id):
        self.broker_host = broker_host
        self.broker_port = broker_port
        self.device_id = device_id
        self.response_times = []
        self.message_count = 0
        
    def test_command_response_time(self, num_commands=100):
        """測試命令響應時間"""
        print(f"🚀 Testing command response time ({num_commands} commands)")
        
        response_times = []
        
        for i in range(num_commands):
            start_time = time.time()
            
            # 發送命令 (簡化版)
            self._send_test_command(f"cmd-perf-{i}")
            
            # 等待響應 (實際應該監聽 ACK/結果)
            time.sleep(0.1)  # 模擬響應時間
            
            response_time = time.time() - start_time
            response_times.append(response_time * 1000)  # 轉換為毫秒
        
        # 統計結果
        avg_time = statistics.mean(response_times)
        max_time = max(response_times)
        min_time = min(response_times)
        p95_time = statistics.quantiles(response_times, n=20)[18]  # 95th percentile
        
        print(f"📊 Response Time Statistics:")
        print(f"  Average: {avg_time:.2f}ms")
        print(f"  Min: {min_time:.2f}ms")
        print(f"  Max: {max_time:.2f}ms")
        print(f"  95th percentile: {p95_time:.2f}ms")
        
        return {
            'average': avg_time,
            'max': max_time,
            'min': min_time,
            'p95': p95_time
        }
    
    def test_message_throughput(self, duration=60):
        """測試訊息吞吐量"""
        print(f"📈 Testing message throughput ({duration}s)")
        
        start_time = time.time()
        message_count = 0
        
        def send_messages():
            nonlocal message_count
            client = mqtt.Client(f"perf-test-{self.device_id}")
            client.connect(self.broker_host, self.broker_port, 60)
            
            while time.time() - start_time < duration:
                # 發送測試訊息
                topic = f"rtk/v1/test/perf/{self.device_id}/telemetry/test"
                payload = {
                    "schema": "telemetry.test/1.0",
                    "ts": int(time.time() * 1000),
                    "value": message_count
                }
                
                client.publish(topic, json.dumps(payload), qos=0)
                message_count += 1
                
                time.sleep(0.01)  # 100 messages/second
            
            client.disconnect()
        
        # 啟動發送線程
        sender_thread = threading.Thread(target=send_messages)
        sender_thread.start()
        sender_thread.join()
        
        throughput = message_count / duration
        print(f"📊 Throughput: {throughput:.2f} messages/second")
        
        return throughput
    
    def test_resource_usage(self, duration=300):
        """測試資源使用"""
        print(f"💾 Testing resource usage ({duration}s)")
        
        # 啟動監控
        cpu_samples = []
        memory_samples = []
        
        def monitor_resources():
            import psutil
            process = psutil.Process()
            
            start_time = time.time()
            while time.time() - start_time < duration:
                cpu_samples.append(process.cpu_percent())
                memory_samples.append(process.memory_info().rss / 1024 / 1024)  # MB
                time.sleep(1)
        
        monitor_thread = threading.Thread(target=monitor_resources)
        monitor_thread.start()
        
        # 模擬正常負載
        self._simulate_normal_load(duration)
        
        monitor_thread.join()
        
        avg_cpu = statistics.mean(cpu_samples)
        max_cpu = max(cpu_samples) 
        avg_memory = statistics.mean(memory_samples)
        max_memory = max(memory_samples)
        
        print(f"📊 Resource Usage:")
        print(f"  CPU - Average: {avg_cpu:.2f}%, Max: {max_cpu:.2f}%")
        print(f"  Memory - Average: {avg_memory:.2f}MB, Max: {max_memory:.2f}MB")
        
        return {
            'cpu_avg': avg_cpu,
            'cpu_max': max_cpu,
            'memory_avg': avg_memory,
            'memory_max': max_memory
        }

def main():
    test = RTKPerformanceTest("test.rtk-mqtt.com", 1883, "perf-test-001")
    
    # 執行效能測試
    response_time_results = test.test_command_response_time(100)
    throughput_result = test.test_message_throughput(60)
    resource_results = test.test_resource_usage(300)
    
    # 評估結果
    print("\n🎯 Performance Evaluation:")
    
    # 響應時間評估
    if response_time_results['average'] < 1000:  # < 1s
        print("✅ Command response time: PASS")
    else:
        print("❌ Command response time: FAIL")
    
    # 吞吐量評估
    if throughput_result >= 10:  # ≥ 10 msg/s for Basic
        print("✅ Message throughput: PASS")
    else:
        print("❌ Message throughput: FAIL")
    
    # 資源使用評估
    if resource_results['cpu_avg'] < 20 and resource_results['memory_max'] < 64:
        print("✅ Resource usage: PASS")
    else:
        print("❌ Resource usage: FAIL")

if __name__ == "__main__":
    main()
```

### 安全性測試
```bash
#!/bin/bash
# certification/security_test.sh

set -e

DEVICE_ID="security-test-001"
BROKER_HOST="test.rtk-mqtt.com"
BROKER_PORT=1883

echo "🔒 RTK MQTT Security Certification Test"

# 測試 1: 認證機制
echo "🔐 Testing Authentication..."
mosquitto_pub -h $BROKER_HOST -p $BROKER_PORT \
    -t "rtk/v1/test/security/$DEVICE_ID/state" \
    -m '{"test": "no_auth"}' \
    && echo "❌ No authentication required - SECURITY RISK" \
    || echo "✅ Authentication required - PASS"

# 測試 2: 主題授權
echo "🚪 Testing Topic Authorization..."
mosquitto_pub -h $BROKER_HOST -p $BROKER_PORT \
    -u "test_user" -P "test_pass" \
    -t "rtk/v1/other/tenant/device/cmd/req" \
    -m '{"test": "cross_tenant"}' \
    && echo "❌ Cross-tenant access allowed - SECURITY RISK" \
    || echo "✅ Topic authorization working - PASS"

# 測試 3: TLS 支援
echo "🔒 Testing TLS Support..."
mosquitto_pub -h $BROKER_HOST -p 8883 \
    --cafile ca.crt \
    --cert client.crt \
    --key client.key \
    -t "rtk/v1/test/security/$DEVICE_ID/state" \
    -m '{"test": "tls"}' \
    && echo "✅ TLS connection successful - PASS" \
    || echo "❌ TLS connection failed - FAIL"

# 測試 4: 訊息注入防護
echo "🛡️ Testing Message Injection Protection..."
MALICIOUS_PAYLOAD='{"schema": "state/1.0", "ts": 1699123456789, "health": "ok", "malicious": "<script>alert(\"xss\")</script>"}'
mosquitto_pub -h $BROKER_HOST -p $BROKER_PORT \
    -u "test_user" -P "test_pass" \
    -t "rtk/v1/test/security/$DEVICE_ID/state" \
    -m "$MALICIOUS_PAYLOAD"

echo "Check broker logs for injection attempts..."

echo "🔒 Security test completed"
```

## 📝 認證申請流程

### 1. 申請準備
```markdown
## RTK MQTT 認證申請檢查清單

### 申請資訊
- [ ] 公司/組織名稱
- [ ] 產品名稱和版本
- [ ] 認證級別 (Basic/Advanced/Enterprise)
- [ ] 技術聯絡人資訊
- [ ] 預計認證完成時間

### 技術文檔
- [ ] 產品技術規格
- [ ] RTK MQTT 整合說明
- [ ] 支援的命令和事件清單
- [ ] 安全性功能說明
- [ ] 已知限制和問題

### 測試準備
- [ ] 測試環境設置完成
- [ ] 測試設備或模擬器準備就緒
- [ ] 網路連接和防火牆配置
- [ ] 測試數據和腳本準備
```

### 2. 提交申請
```bash
# 認證申請提交
curl -X POST https://certification.rtk-mqtt.com/api/applications \
  -H "Content-Type: application/json" \
  -d '{
    "company": "Your Company",
    "product": "Your Product",
    "version": "1.0.0",
    "certification_level": "Advanced",
    "contact_email": "tech@company.com",
    "expected_completion": "2024-03-01"
  }'
```

### 3. 測試執行

#### 自動化測試執行
```bash
#!/bin/bash
# certification/run_certification_tests.sh

CERTIFICATION_LEVEL=${1:-basic}
DEVICE_ID=${2:-cert-device-001}

echo "🎯 Running RTK MQTT Certification Tests"
echo "Level: $CERTIFICATION_LEVEL"
echo "Device: $DEVICE_ID"

# 基礎測試 (所有級別)
echo "📋 Running basic compatibility tests..."
python3 certification/basic_compatibility_test.py

# 效能測試
echo "⚡ Running performance tests..."
python3 certification/performance_test.py

# 安全性測試
echo "🔒 Running security tests..."
./certification/security_test.sh

# 進階測試 (Advanced+)
if [ "$CERTIFICATION_LEVEL" != "basic" ]; then
    echo "🔧 Running advanced functionality tests..."
    python3 certification/advanced_functionality_test.py
fi

# 企業級測試 (Enterprise)
if [ "$CERTIFICATION_LEVEL" = "enterprise" ]; then
    echo "🏢 Running enterprise feature tests..."
    python3 certification/enterprise_test.py
fi

# 生成測試報告
echo "📊 Generating certification report..."
python3 certification/generate_report.py --level $CERTIFICATION_LEVEL --device $DEVICE_ID
```

### 4. 報告生成
```python
#!/usr/bin/env python3
# certification/generate_report.py

import json
import argparse
from datetime import datetime

def generate_certification_report(level, device_id, test_results):
    """生成認證報告"""
    
    report = {
        "certification_info": {
            "device_id": device_id,
            "certification_level": level,
            "test_date": datetime.now().isoformat(),
            "rtk_mqtt_version": "1.2.0"
        },
        "test_results": test_results,
        "summary": {
            "total_tests": len(test_results),
            "passed": sum(1 for r in test_results if r["status"] == "PASS"),
            "failed": sum(1 for r in test_results if r["status"] == "FAIL"),
            "overall_status": "PASS" if all(r["status"] == "PASS" for r in test_results) else "FAIL"
        }
    }
    
    # 生成 HTML 報告
    html_report = f"""
    <!DOCTYPE html>
    <html>
    <head>
        <title>RTK MQTT Certification Report - {device_id}</title>
        <style>
            body {{ font-family: Arial, sans-serif; margin: 40px; }}
            .header {{ background: #f0f0f0; padding: 20px; border-radius: 5px; }}
            .pass {{ color: green; }}
            .fail {{ color: red; }}
            table {{ border-collapse: collapse; width: 100%; margin: 20px 0; }}
            th, td {{ border: 1px solid #ddd; padding: 8px; text-align: left; }}
            th {{ background-color: #f2f2f2; }}
        </style>
    </head>
    <body>
        <div class="header">
            <h1>RTK MQTT Certification Report</h1>
            <p><strong>Device ID:</strong> {device_id}</p>
            <p><strong>Certification Level:</strong> {level.title()}</p>
            <p><strong>Test Date:</strong> {report['certification_info']['test_date']}</p>
            <p><strong>Overall Status:</strong> 
               <span class="{'pass' if report['summary']['overall_status'] == 'PASS' else 'fail'}">
                   {report['summary']['overall_status']}
               </span>
            </p>
        </div>
        
        <h2>Test Results Summary</h2>
        <p>Total Tests: {report['summary']['total_tests']}</p>
        <p>Passed: <span class="pass">{report['summary']['passed']}</span></p>
        <p>Failed: <span class="fail">{report['summary']['failed']}</span></p>
        
        <h2>Detailed Test Results</h2>
        <table>
            <tr>
                <th>Test Category</th>
                <th>Test Name</th>
                <th>Status</th>
                <th>Details</th>
            </tr>
    """
    
    for result in test_results:
        status_class = "pass" if result["status"] == "PASS" else "fail"
        html_report += f"""
            <tr>
                <td>{result['category']}</td>
                <td>{result['name']}</td>
                <td><span class="{status_class}">{result['status']}</span></td>
                <td>{result.get('details', '')}</td>
            </tr>
        """
    
    html_report += """
        </table>
    </body>
    </html>
    """
    
    return report, html_report

def main():
    parser = argparse.ArgumentParser(description="Generate RTK MQTT Certification Report")
    parser.add_argument("--level", required=True, help="Certification level")
    parser.add_argument("--device", required=True, help="Device ID")
    parser.add_argument("--results", default="test_results.json", help="Test results file")
    
    args = parser.parse_args()
    
    # 載入測試結果
    try:
        with open(args.results, 'r') as f:
            test_results = json.load(f)
    except FileNotFoundError:
        print(f"Test results file {args.results} not found")
        return
    
    # 生成報告
    report, html_report = generate_certification_report(args.level, args.device, test_results)
    
    # 保存報告
    with open(f"certification_report_{args.device}.json", 'w') as f:
        json.dump(report, f, indent=2)
    
    with open(f"certification_report_{args.device}.html", 'w') as f:
        f.write(html_report)
    
    print(f"📊 Certification report generated:")
    print(f"  JSON: certification_report_{args.device}.json")
    print(f"  HTML: certification_report_{args.device}.html")
    print(f"  Overall Status: {report['summary']['overall_status']}")

if __name__ == "__main__":
    main()
```

## 🏅 認證結果和證書

### 認證證書範例
```
╔══════════════════════════════════════════════════════════════╗
║                    RTK MQTT CERTIFICATION                    ║
║                         CERTIFICATE                         ║
╠══════════════════════════════════════════════════════════════╣
║                                                              ║
║  This certifies that                                        ║
║                                                              ║
║  Product: Smart WiFi Router Model R2024                     ║
║  Manufacturer: TechCorp Ltd.                                 ║
║  Version: 2.1.0                                             ║
║                                                              ║
║  has successfully completed RTK MQTT                        ║
║  ADVANCED LEVEL CERTIFICATION                               ║
║                                                              ║
║  Certificate ID: RTK-ADV-2024-001                           ║
║  Issue Date: March 15, 2024                                 ║
║  Valid Until: March 15, 2025                               ║
║                                                              ║
║  This product is certified to be fully compatible          ║
║  with RTK MQTT Protocol v1.2.0 and meets all              ║
║  requirements for Advanced level certification.             ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
```

### 認證標誌使用
```html
<!-- RTK MQTT 認證標誌 -->
<div class="rtk-certification-badge">
  <img src="https://certification.rtk-mqtt.com/badges/advanced.svg" 
       alt="RTK MQTT Advanced Certified" />
  <p>RTK MQTT Advanced Certified</p>
</div>
```

## 📅 認證維護

### 年度複審
- **時機**: 證書到期前 30 天
- **要求**: 提交更新的測試報告
- **流程**: 簡化測試程序，重點檢查變更部分

### 版本更新認證
- **觸發條件**: 
  - MAJOR 版本更新: 完整重新認證
  - MINOR 版本更新: 增量測試
  - PATCH 版本更新: 自動延續認證

### 認證撤銷
**撤銷條件**:
- 發現嚴重安全漏洞
- 協議相容性問題
- 證書到期未續簽
- 違反認證協議

## 💰 認證費用

| 認證級別 | 申請費用 | 年度維護費 | 測試時間 |
|----------|----------|------------|----------|
| Basic | $500 | $100 | 1-2 週 |
| Advanced | $1,500 | $300 | 2-4 週 |
| Enterprise | $5,000 | $1,000 | 4-8 週 |

## 🔗 相關資源

- **[測試環境](https://test.rtk-mqtt.com)** - 認證測試平台
- **[認證申請](https://certification.rtk-mqtt.com)** - 線上申請系統
- **[技術支援](SUPPORT_RESOURCES.md)** - 認證技術支援
- **[發布指南](RELEASE_GUIDE.md)** - 版本發布流程

---

RTK MQTT 認證確保您的產品符合最高的品質和相容性標準，為用戶提供可靠的整合體驗。
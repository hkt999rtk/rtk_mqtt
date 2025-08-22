# Mesh 節點 MQTT 整合指南

## 概述

本文檔提供Mesh網路節點設備與RTK MQTT協議整合的完整指南，涵蓋Mesh拓撲管理、路由協議、節點間通信等核心功能。

## Mesh節點類型

### 根節點 (Root Node)
```json
{
  "device_type": "mesh_root",
  "mesh_role": "gateway",
  "capabilities": ["routing", "internet_access", "topology_management"],
  "supported_protocols": ["ieee802.11s", "batman-adv", "olsr"]
}
```

### 中繼節點 (Relay Node)
```json
{
  "device_type": "mesh_relay", 
  "mesh_role": "router",
  "capabilities": ["routing", "forwarding", "load_balancing"],
  "max_connections": 32
}
```

### 葉節點 (Leaf Node)
```json
{
  "device_type": "mesh_leaf",
  "mesh_role": "endpoint", 
  "capabilities": ["client_access", "local_services"],
  "client_capacity": 16
}
```

## MQTT 主題結構

### Mesh專用主題
```
rtk/v1/{tenant}/{site}/{node_id}/mesh/topology
rtk/v1/{tenant}/{site}/{node_id}/mesh/routing
rtk/v1/{tenant}/{site}/{node_id}/mesh/quality
rtk/v1/{tenant}/{site}/{node_id}/mesh/neighbors
rtk/v1/{tenant}/{site}/{node_id}/mesh/load
```

### 節點管理主題
```
rtk/v1/{tenant}/{site}/{node_id}/state
rtk/v1/{tenant}/{site}/{node_id}/telemetry/mesh
rtk/v1/{tenant}/{site}/{node_id}/evt/mesh
rtk/v1/{tenant}/{site}/{node_id}/cmd/req
```

## 狀態訊息 (state)

### Mesh節點狀態
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "health": "ok",
    "mesh_role": "relay",
    "mesh_id": "mesh_network_001",
    "node_status": "active",
    "connected_nodes": 5,
    "routing_table_size": 12,
    "uplink_quality": 85,
    "load_factor": 0.45
  }
}
```

### 根節點狀態
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "health": "ok",
    "mesh_role": "gateway",
    "internet_connectivity": true,
    "total_mesh_nodes": 15,
    "active_clients": 42,
    "network_throughput_mbps": 125.5,
    "gateway_load": 0.32
  }
}
```

## 遙測數據 (telemetry)

### Mesh拓撲遙測
```json
{
  "schema": "telemetry.mesh_topology/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "mesh_id": "mesh_network_001",
    "node_count": 15,
    "topology_map": {
      "nodes": [
        {
          "node_id": "node_001",
          "role": "gateway",
          "coordinates": {"x": 0, "y": 0},
          "status": "active"
        },
        {
          "node_id": "node_002", 
          "role": "relay",
          "coordinates": {"x": 100, "y": 50},
          "status": "active"
        }
      ],
      "links": [
        {
          "from": "node_001",
          "to": "node_002",
          "quality": 85,
          "bandwidth_mbps": 54,
          "latency_ms": 5.2
        }
      ]
    }
  }
}
```

### 路由表遙測
```json
{
  "schema": "telemetry.mesh_routing/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "routing_protocol": "batman-adv",
    "routing_table": [
      {
        "destination": "node_003",
        "next_hop": "node_002",
        "metric": 15,
        "interface": "mesh0",
        "last_seen": 1699123450000
      },
      {
        "destination": "node_004",
        "next_hop": "node_002", 
        "metric": 22,
        "interface": "mesh0",
        "last_seen": 1699123455000
      }
    ],
    "table_size": 12,
    "convergence_time_ms": 150
  }
}
```

### 鄰居節點遙測
```json
{
  "schema": "telemetry.mesh_neighbors/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "neighbor_count": 3,
    "neighbors": [
      {
        "node_id": "node_002",
        "signal_strength": -45,
        "link_quality": 95,
        "last_seen": 1699123456000,
        "tx_rate_mbps": 54,
        "rx_rate_mbps": 48,
        "packet_loss": 0.01
      },
      {
        "node_id": "node_005",
        "signal_strength": -65,
        "link_quality": 75,
        "last_seen": 1699123455000,
        "tx_rate_mbps": 24,
        "rx_rate_mbps": 22,
        "packet_loss": 0.05
      }
    ]
  }
}
```

### 負載均衡遙測
```json
{
  "schema": "telemetry.mesh_load/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "cpu_usage": 35.5,
    "memory_usage": 45.2,
    "forwarding_rate_pps": 1250,
    "queue_depth": 15,
    "buffer_utilization": 0.25,
    "active_flows": 45,
    "load_distribution": {
      "local_traffic": 0.3,
      "forwarded_traffic": 0.6,
      "control_traffic": 0.1
    }
  }
}
```

## 事件通知 (evt)

### 拓撲變化事件
```json
{
  "schema": "evt.mesh_topology/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "event_type": "node_joined",
    "node_id": "node_015",
    "mesh_role": "leaf",
    "connection_point": "node_003",
    "signal_strength": -55,
    "auto_configured": true,
    "topology_version": 15
  }
}
```

### 路由變化事件
```json
{
  "schema": "evt.mesh_routing/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "event_type": "route_changed",
    "destination": "node_008",
    "old_next_hop": "node_003",
    "new_next_hop": "node_002",
    "reason": "link_quality_improved",
    "convergence_time_ms": 200
  }
}
```

### 負載警報事件
```json
{
  "schema": "evt.mesh_load/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "event_type": "high_load_warning",
    "load_type": "cpu",
    "current_value": 85.5,
    "threshold": 80.0,
    "duration_s": 120,
    "affected_services": ["forwarding", "routing"],
    "suggested_action": "load_balancing"
  }
}
```

### 連接品質事件
```json
{
  "schema": "evt.mesh_quality/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "event_type": "link_degraded",
    "neighbor_node": "node_007",
    "signal_strength": -78,
    "link_quality": 45,
    "packet_loss": 0.15,
    "severity": "warning",
    "duration_s": 300
  }
}
```

## 支援的命令

### Mesh管理命令

#### 拓撲掃描
```json
{
  "schema": "cmd.mesh_topology_scan/1.0",
  "ts": 1699123456789,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-1699123456789",
    "op": "mesh_topology_scan",
    "args": {
      "scan_depth": 3,
      "include_inactive": false,
      "detailed_metrics": true
    }
  }
}
```

#### 路由表重建
```json
{
  "schema": "cmd.rebuild_routing_table/1.0",
  "ts": 1699123456790,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-1699123456790",
    "op": "rebuild_routing_table",
    "args": {
      "force_convergence": true,
      "timeout_s": 30
    }
  }
}
```

#### 負載均衡優化
```json
{
  "schema": "cmd.optimize_load_balance/1.0",
  "ts": 1699123456791,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-1699123456791",
    "op": "optimize_load_balance",
    "args": {
      "target_utilization": 0.7,
      "redistribution_method": "least_loaded"
    }
  }
}
```

#### 節點角色變更
```json
{
  "schema": "cmd.change_mesh_role/1.0",
  "ts": 1699123456792,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-1699123456792",
    "op": "change_mesh_role",
    "args": {
      "new_role": "relay",
      "migration_timeout_s": 60,
      "preserve_connections": true
    }
  }
}
```

### 連接管理命令

#### 強制鄰居重新發現
```json
{
  "schema": "cmd.rediscover_neighbors/1.0",
  "ts": 1699123456793,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-1699123456793",
    "op": "rediscover_neighbors",
    "args": {
      "scan_channels": [1, 6, 11],
      "scan_duration_s": 10
    }
  }
}
```

#### 連接品質優化
```json
{
  "schema": "cmd.optimize_connections/1.0",
  "ts": 1699123456794,
  "device_id": "aabbccddeeff",
  "payload": {
    "id": "cmd-1699123456794",
    "op": "optimize_connections",
    "args": {
      "min_quality_threshold": 70,
      "max_connections": 8,
      "prioritize_stability": true
    }
  }
}
```

## 實作範例

### Linux Mesh節點實作

```python
import paho.mqtt.client as mqtt
import json
import time
import subprocess
import threading
import socket
from collections import defaultdict

class MeshNodeMQTTClient:
    def __init__(self, broker_host, node_id, mesh_role="relay", tenant="demo", site="mesh"):
        self.client = mqtt.Client()
        self.broker_host = broker_host
        self.node_id = node_id
        self.mesh_role = mesh_role
        self.tenant = tenant
        self.site = site
        
        # Mesh狀態
        self.mesh_id = "mesh_network_001"
        self.routing_table = {}
        self.neighbors = {}
        self.load_metrics = {}
        
        # 設定MQTT回調
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        
        # 啟動background threads
        self.monitoring_active = True
        self.start_monitoring_threads()
        
    def on_connect(self, client, userdata, flags, rc):
        print(f"Mesh node {self.node_id} connected with result code {rc}")
        cmd_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.node_id}/cmd/req"
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
        args = payload.get("args", {})
        
        # 發送確認
        self.send_ack(cmd_id)
        
        # 處理Mesh專用命令
        try:
            if operation == "mesh_topology_scan":
                result = self.mesh_topology_scan(args)
            elif operation == "rebuild_routing_table":
                result = self.rebuild_routing_table(args)
            elif operation == "optimize_load_balance":
                result = self.optimize_load_balance(args)
            elif operation == "change_mesh_role":
                result = self.change_mesh_role(args)
            elif operation == "rediscover_neighbors":
                result = self.rediscover_neighbors(args)
            elif operation == "optimize_connections":
                result = self.optimize_connections(args)
            else:
                result = {"error": "Unsupported mesh command"}
                
            self.send_result(cmd_id, operation, result)
            
        except Exception as e:
            self.send_result(cmd_id, operation, {"error": str(e)})
            
    def send_ack(self, cmd_id):
        ack_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.node_id}/cmd/ack"
        ack_msg = {
            "schema": "cmd.ack/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.node_id,
            "payload": {
                "id": cmd_id,
                "status": "accepted"
            }
        }
        self.client.publish(ack_topic, json.dumps(ack_msg), qos=1)
        
    def send_result(self, cmd_id, operation, result):
        res_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.node_id}/cmd/res"
        status = "completed" if "error" not in result else "failed"
        res_msg = {
            "schema": f"cmd.{operation}.result/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.node_id,
            "payload": {
                "id": cmd_id,
                "status": status,
                "result": result
            }
        }
        self.client.publish(res_topic, json.dumps(res_msg), qos=1)
        
    def mesh_topology_scan(self, args):
        """執行Mesh拓撲掃描"""
        try:
            scan_depth = args.get("scan_depth", 3)
            detailed = args.get("detailed_metrics", True)
            
            # 使用batman-adv或其他mesh協議獲取拓撲
            result = subprocess.run(['batctl', 'o'], capture_output=True, text=True)
            
            topology = self.parse_topology_output(result.stdout, scan_depth, detailed)
            
            return {
                "topology_discovered": True,
                "node_count": len(topology.get("nodes", [])),
                "link_count": len(topology.get("links", [])),
                "topology": topology
            }
        except Exception as e:
            return {"error": f"Topology scan failed: {str(e)}"}
            
    def rebuild_routing_table(self, args):
        """重建路由表"""
        try:
            force = args.get("force_convergence", False)
            timeout = args.get("timeout_s", 30)
            
            if force:
                # 強制重新計算路由
                subprocess.run(['batctl', 'if', 'del', 'mesh0'], capture_output=True)
                time.sleep(2)
                subprocess.run(['batctl', 'if', 'add', 'mesh0'], capture_output=True)
            
            # 等待路由收斂
            start_time = time.time()
            while time.time() - start_time < timeout:
                if self.check_routing_convergence():
                    break
                time.sleep(1)
                
            new_table_size = len(self.get_routing_table())
            
            return {
                "routing_rebuilt": True,
                "convergence_time_s": time.time() - start_time,
                "new_table_size": new_table_size
            }
        except Exception as e:
            return {"error": f"Routing rebuild failed: {str(e)}"}
            
    def optimize_load_balance(self, args):
        """負載均衡優化"""
        try:
            target_util = args.get("target_utilization", 0.7)
            method = args.get("redistribution_method", "least_loaded")
            
            current_load = self.get_current_load()
            
            if current_load > target_util:
                # 執行負載重分配
                self.redistribute_load(method)
                
            new_load = self.get_current_load()
            
            return {
                "optimization_completed": True,
                "old_load": current_load,
                "new_load": new_load,
                "improvement": current_load - new_load
            }
        except Exception as e:
            return {"error": f"Load optimization failed: {str(e)}"}
            
    def rediscover_neighbors(self, args):
        """重新發現鄰居節點"""
        try:
            channels = args.get("scan_channels", [1, 6, 11])
            duration = args.get("scan_duration_s", 10)
            
            old_neighbors = len(self.neighbors)
            
            # 執行鄰居掃描
            for channel in channels:
                self.scan_channel_for_neighbors(channel, duration // len(channels))
                
            new_neighbors = len(self.neighbors)
            
            return {
                "rediscovery_completed": True,
                "old_neighbor_count": old_neighbors,
                "new_neighbor_count": new_neighbors,
                "discovered_neighbors": list(self.neighbors.keys())
            }
        except Exception as e:
            return {"error": f"Neighbor rediscovery failed: {str(e)}"}
            
    def get_routing_table(self):
        """獲取當前路由表"""
        try:
            result = subprocess.run(['batctl', 'r'], capture_output=True, text=True)
            return self.parse_routing_table(result.stdout)
        except:
            return {}
            
    def get_neighbors(self):
        """獲取鄰居節點信息"""
        try:
            result = subprocess.run(['batctl', 'n'], capture_output=True, text=True)
            return self.parse_neighbors(result.stdout)
        except:
            return {}
            
    def publish_state(self):
        """發布節點狀態"""
        state_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.node_id}/state"
        state_msg = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.node_id,
            "payload": {
                "health": "ok",
                "mesh_role": self.mesh_role,
                "mesh_id": self.mesh_id,
                "node_status": "active",
                "connected_nodes": len(self.neighbors),
                "routing_table_size": len(self.routing_table),
                "load_factor": self.get_current_load()
            }
        }
        self.client.publish(state_topic, json.dumps(state_msg), qos=1, retain=True)
        
    def publish_topology_telemetry(self):
        """發布拓撲遙測"""
        topology_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.node_id}/telemetry/mesh_topology"
        
        # 獲取完整拓撲視圖(僅根節點)
        if self.mesh_role == "gateway":
            topology_data = self.get_full_topology()
            telemetry_msg = {
                "schema": "telemetry.mesh_topology/1.0",
                "ts": int(time.time() * 1000),
                "device_id": self.node_id,
                "payload": {
                    "mesh_id": self.mesh_id,
                    "node_count": topology_data.get("node_count", 0),
                    "topology_map": topology_data.get("topology_map", {})
                }
            }
            self.client.publish(topology_topic, json.dumps(telemetry_msg), qos=0)
            
    def publish_neighbors_telemetry(self):
        """發布鄰居遙測"""
        neighbors_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.node_id}/telemetry/mesh_neighbors"
        
        neighbors_data = self.get_neighbors()
        telemetry_msg = {
            "schema": "telemetry.mesh_neighbors/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.node_id,
            "payload": {
                "neighbor_count": len(neighbors_data),
                "neighbors": list(neighbors_data.values())
            }
        }
        self.client.publish(neighbors_topic, json.dumps(telemetry_msg), qos=0)
        
    def start_monitoring_threads(self):
        """啟動監控線程"""
        def monitoring_loop():
            while self.monitoring_active:
                try:
                    # 更新路由表和鄰居信息
                    self.routing_table = self.get_routing_table()
                    self.neighbors = self.get_neighbors()
                    
                    # 檢查負載和連接品質
                    self.check_load_thresholds()
                    self.check_connection_quality()
                    
                except Exception as e:
                    print(f"Monitoring error: {e}")
                    
                time.sleep(10)
                
        self.monitoring_thread = threading.Thread(target=monitoring_loop)
        self.monitoring_thread.daemon = True
        self.monitoring_thread.start()
        
    def start(self):
        self.client.connect(self.broker_host, 1883, 60)
        self.client.loop_start()
        
        # 定期發布狀態和遙測
        while True:
            try:
                self.publish_state()
                self.publish_topology_telemetry()
                self.publish_neighbors_telemetry()
                time.sleep(30)
            except KeyboardInterrupt:
                print("Shutting down mesh node...")
                self.monitoring_active = False
                break
                
# 使用範例
if __name__ == "__main__":
    import sys
    
    if len(sys.argv) < 3:
        print("Usage: python mesh_node.py <node_id> <mesh_role>")
        sys.exit(1)
        
    node_id = sys.argv[1]
    mesh_role = sys.argv[2]
    
    mesh_node = MeshNodeMQTTClient("localhost", node_id, mesh_role)
    mesh_node.start()
```

### OpenWrt Mesh節點配置

```bash
#!/bin/bash
# mesh_node_setup.sh - OpenWrt Mesh節點設定腳本

# 安裝必要套件
opkg update
opkg install batctl-default kmod-batman-adv mosquitto-client

# 設定batman-adv mesh介面
uci set network.bat0=interface
uci set network.bat0.proto='batadv'
uci set network.bat0.mesh='bat0'
uci set network.bat0.routing_algo='BATMAN_IV'

# 設定mesh0介面
uci set network.mesh0=interface  
uci set network.mesh0.proto='none'
uci set network.mesh0.ifname='wlan0-1'

# 設定WiFi mesh
uci set wireless.mesh0=wifi-iface
uci set wireless.mesh0.device='radio0'
uci set wireless.mesh0.network='mesh0'
uci set wireless.mesh0.mode='mesh'
uci set wireless.mesh0.mesh_id='RTK_MESH'
uci set wireless.mesh0.encryption='sae'
uci set wireless.mesh0.key='mesh_password'

# 應用設定
uci commit
/etc/init.d/network restart

# 啟動batman-adv
echo 'bat0' > /sys/class/net/bat0/mesh/interfaces
batctl if add mesh0

# 設定MQTT發布腳本
cat > /usr/bin/mesh_mqtt_publisher.py << 'EOF'
#!/usr/bin/env python3

import paho.mqtt.client as mqtt
import json
import time
import subprocess
import os

class OpenWrtMeshNode:
    def __init__(self):
        self.node_id = self.get_node_id()
        self.client = mqtt.Client()
        
    def get_node_id(self):
        try:
            with open('/sys/class/net/wlan0/address', 'r') as f:
                return f.read().strip()
        except:
            return "unknown_node"
            
    def get_mesh_status(self):
        try:
            # 獲取batman-adv狀態
            result = subprocess.run(['batctl', 'o'], capture_output=True, text=True)
            neighbors = len([line for line in result.stdout.split('\n') if 'if' in line])
            
            return {
                "active": True,
                "neighbor_count": neighbors,
                "mesh_interface": "mesh0"
            }
        except:
            return {"active": False}
            
    def publish_status(self):
        mesh_status = self.get_mesh_status()
        
        state_msg = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "device_id": self.node_id,
            "payload": {
                "health": "ok" if mesh_status["active"] else "error",
                "mesh_role": "relay",
                "node_status": "active" if mesh_status["active"] else "inactive",
                "connected_nodes": mesh_status.get("neighbor_count", 0)
            }
        }
        
        topic = f"rtk/v1/demo/mesh/{self.node_id}/state"
        self.client.publish(topic, json.dumps(state_msg), qos=1, retain=True)
        
    def start(self):
        self.client.connect("192.168.1.100", 1883, 60)
        self.client.loop_start()
        
        while True:
            self.publish_status()
            time.sleep(60)

if __name__ == "__main__":
    node = OpenWrtMeshNode()
    node.start()
EOF

chmod +x /usr/bin/mesh_mqtt_publisher.py

# 建立systemd服務
cat > /etc/init.d/mesh_mqtt << 'EOF'
#!/bin/sh /etc/rc.common

START=99
STOP=15

start() {
    echo "Starting mesh MQTT publisher"
    python3 /usr/bin/mesh_mqtt_publisher.py &
}

stop() {
    echo "Stopping mesh MQTT publisher"
    killall python3
}
EOF

chmod +x /etc/init.d/mesh_mqtt
/etc/init.d/mesh_mqtt enable
```

## 測試與驗證

### Mesh功能測試清單
- [ ] 節點自動發現機制
- [ ] 路由協議收斂時間
- [ ] 拓撲變化處理
- [ ] 負載均衡效果
- [ ] 故障恢復能力

### 性能測試場景

1. **擴展性測試**
   - 逐步增加節點數量
   - 測量拓撲收斂時間
   - 監控路由表大小

2. **容錯性測試**
   - 隨機節點故障
   - 連接中斷恢復
   - 負載重分配

3. **負載測試**
   - 高流量轉發
   - 多跳路徑性能
   - QoS保證測試

### 診斷工具

```bash
# 檢查mesh狀態
batctl o                    # 查看鄰居節點
batctl r                    # 查看路由表  
batctl n                    # 查看鄰居詳情
batctl s                    # 查看統計信息

# 網路連通性測試
ping6 ff02::1%mesh0         # 多播ping
iperf3 -c <target_node>     # 頻寬測試

# MQTT測試
mosquitto_pub -h broker -t "rtk/v1/demo/mesh/test_node/cmd/req" -m '{
  "id": "test-001", 
  "op": "mesh_topology_scan",
  "args": {"scan_depth": 2}
}'
```

## 故障排除

### 常見問題

1. **節點無法加入Mesh**
   - 檢查mesh ID匹配
   - 驗證無線頻道設定
   - 確認加密金鑰正確

2. **路由收斂緩慢**
   - 調整batman-adv參數
   - 檢查網路拓撲複雜度
   - 監控控制封包丟失

3. **負載不均衡**
   - 分析流量模式
   - 調整負載均衡算法
   - 檢查節點能力差異

### 效能調優

```bash
# Batman-adv參數調優
echo 1000 > /sys/class/net/bat0/mesh/orig_interval
echo 3 > /sys/class/net/bat0/mesh/hop_penalty
echo 1 > /sys/class/net/bat0/mesh/distributed_arp_table

# 無線參數調優
iw dev mesh0 set mesh_param mesh_ttl 31
iw dev mesh0 set mesh_param mesh_element_ttl 31
iw dev mesh0 set mesh_param mesh_auto_open_plinks 1
```

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Commands and Events Reference](../core/COMMANDS_EVENTS_REFERENCE.md)
- [Topic Structure Guide](../core/TOPIC_STRUCTURE.md)
- [Schema Reference](../core/SCHEMA_REFERENCE.md)
- [Batman-adv Documentation](https://www.open-mesh.org/projects/batman-adv/wiki)
- [IEEE 802.11s Mesh Networking](https://standards.ieee.org/standard/802_11s-2011.html)
# 交換機 MQTT 整合指南

## 概述

本文檔提供網路交換機設備與RTK MQTT協議整合的完整指南，涵蓋VLAN管理、端口監控、生成樹協議、流量分析等核心功能。

## 交換機類型

### 管理型交換機 (Managed Switch)
```json
{
  "device_type": "managed_switch",
  "capabilities": ["vlan", "port_mirroring", "qos", "stp", "snmp"],
  "port_count": 24,
  "management_protocols": ["snmp", "telnet", "ssh", "web"],
  "switching_capacity_gbps": 48
}
```

### 智慧交換機 (Smart Switch)
```json
{
  "device_type": "smart_switch",
  "capabilities": ["basic_vlan", "port_management", "link_aggregation"],
  "port_count": 16,
  "management_protocols": ["web", "snmp"],
  "switching_capacity_gbps": 32
}
```

### PoE交換機 (PoE Switch)
```json
{
  "device_type": "poe_switch",
  "capabilities": ["poe_plus", "port_management", "power_monitoring"],
  "port_count": 48,
  "poe_budget_w": 740,
  "poe_standard": "802.3at",
  "management_protocols": ["snmp", "web"]
}
```

## MQTT 主題結構

### 基本主題格式
```
rtk/v1/{tenant}/{site}/{switch_id}/{message_type}[/{sub_type}]
```

### 交換機專用主題
```
rtk/v1/{tenant}/{site}/{switch_id}/state
rtk/v1/{tenant}/{site}/{switch_id}/telemetry/ports
rtk/v1/{tenant}/{site}/{switch_id}/telemetry/vlans
rtk/v1/{tenant}/{site}/{switch_id}/telemetry/traffic
rtk/v1/{tenant}/{site}/{switch_id}/telemetry/poe
rtk/v1/{tenant}/{site}/{switch_id}/evt/port
rtk/v1/{tenant}/{site}/{switch_id}/evt/stp
rtk/v1/{tenant}/{site}/{switch_id}/cmd/req
```

## 狀態訊息 (state)

### 基本交換機狀態
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "health": "ok",
  "uptime_s": 2592000,
  "cpu_usage": 15.5,
  "memory_usage": 35.2,
  "temperature_c": 42.5,
  "power_consumption_w": 85.2,
  "active_ports": 18,
  "total_ports": 24,
  "stp_root": true,
  "vlan_count": 5
}
```

### PoE交換機狀態
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "health": "ok",
  "poe_budget_used_w": 425.5,
  "poe_budget_total_w": 740,
  "poe_utilization": 0.575,
  "powered_devices": 15,
  "poe_status": "normal"
}
```

## 遙測數據 (telemetry)

### 端口遙測 (telemetry/ports)
```json
{
  "schema": "telemetry.ports/1.0",
  "ts": 1699123456789,
  "ports": [
    {
      "port_id": 1,
      "name": "GigabitEthernet1/0/1",
      "status": "up",
      "link_speed": "1000",
      "duplex": "full",
      "vlan_id": 100,
      "rx_bytes": 1048576000,
      "tx_bytes": 524288000,
      "rx_packets": 1000000,
      "tx_packets": 500000,
      "rx_errors": 0,
      "tx_errors": 0,
      "collisions": 0,
      "last_change": 1699120000000
    },
    {
      "port_id": 2,
      "name": "GigabitEthernet1/0/2", 
      "status": "down",
      "link_speed": "unknown",
      "duplex": "unknown",
      "vlan_id": null,
      "poe_enabled": true,
      "poe_power_w": 25.5,
      "poe_status": "delivering"
    }
  ]
}
```

### VLAN遙測 (telemetry/vlans)
```json
{
  "schema": "telemetry.vlans/1.0",
  "ts": 1699123456789,
  "vlans": [
    {
      "vlan_id": 100,
      "name": "Management",
      "status": "active",
      "port_count": 8,
      "tagged_ports": [1, 2, 3, 4],
      "untagged_ports": [5, 6, 7, 8],
      "rx_bytes": 5242880000,
      "tx_bytes": 3145728000,
      "broadcast_packets": 15000,
      "multicast_packets": 5000
    },
    {
      "vlan_id": 200,
      "name": "Guest", 
      "status": "active",
      "port_count": 12,
      "tagged_ports": [1, 2],
      "untagged_ports": [9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20],
      "rx_bytes": 10485760000,
      "tx_bytes": 8388608000
    }
  ]
}
```

### 流量遙測 (telemetry/traffic)
```json
{
  "schema": "telemetry.traffic/1.0",
  "ts": 1699123456789,
  "total_throughput_mbps": 450.5,
  "ingress_utilization": 0.45,
  "egress_utilization": 0.38,
  "packet_rate_pps": 125000,
  "broadcast_rate_pps": 500,
  "multicast_rate_pps": 1200,
  "error_rate": 0.001,
  "top_talkers": [
    {
      "source_mac": "aabbccddeeff",
      "port": 5,
      "bytes_per_sec": 52428800,
      "packets_per_sec": 50000
    }
  ],
  "protocol_distribution": {
    "tcp": 0.65,
    "udp": 0.25,
    "icmp": 0.05,
    "other": 0.05
  }
}
```

### PoE遙測 (telemetry/poe)
```json
{
  "schema": "telemetry.poe/1.0",
  "ts": 1699123456789,
  "total_power_budget_w": 740,
  "total_power_used_w": 425.5,
  "available_power_w": 314.5,
  "poe_ports": [
    {
      "port_id": 1,
      "poe_enabled": true,
      "power_consumption_w": 25.5,
      "voltage_v": 54.2,
      "current_ma": 470,
      "device_type": "pd_class_3",
      "status": "delivering"
    },
    {
      "port_id": 2,
      "poe_enabled": false,
      "power_consumption_w": 0,
      "status": "disabled"
    }
  ],
  "power_efficiency": 0.92,
  "thermal_shutdown_risk": false
}
```

## 事件通知 (evt)

### 端口事件 (evt/port)
```json
{
  "schema": "evt.port/1.0",
  "ts": 1699123456789,
  "event_type": "link_up",
  "port_id": 5,
  "port_name": "GigabitEthernet1/0/5",
  "link_speed": "1000",
  "duplex": "full",
  "connected_device_mac": "aabbccddeeff",
  "auto_negotiation": true,
  "previous_status": "down"
}
```

### 生成樹事件 (evt/stp)
```json
{
  "schema": "evt.stp/1.0",
  "ts": 1699123456789,
  "event_type": "topology_change",
  "instance_id": 0,
  "vlan_id": 100,
  "root_bridge_id": "8000.aabbccddeeff",
  "root_path_cost": 20000,
  "designated_root": true,
  "port_role_changes": [
    {
      "port_id": 1,
      "old_role": "blocking",
      "new_role": "forwarding"
    }
  ],
  "convergence_time_ms": 15000
}
```

### PoE事件 (evt/poe)
```json
{
  "schema": "evt.poe/1.0",
  "ts": 1699123456789,
  "event_type": "power_denied",
  "port_id": 10,
  "requested_power_w": 60,
  "available_power_w": 45,
  "reason": "insufficient_power",
  "device_class": "pd_class_4",
  "action_taken": "deny_power"
}
```

### 安全事件 (evt/security)
```json
{
  "schema": "evt.security/1.0",
  "ts": 1699123456789,
  "event_type": "mac_address_violation",
  "port_id": 8,
  "violating_mac": "ffffffffffff",
  "authorized_mac": "aabbccddeeff",
  "violation_count": 3,
  "action_taken": "port_shutdown",
  "security_level": "critical"
}
```

## 支援的命令

### 端口管理命令

#### 端口啟用/停用
```json
{
  "id": "cmd-1699123456789",
  "op": "set_port_state",
  "schema": "cmd.set_port_state/1.0",
  "args": {
    "port_id": 5,
    "state": "enable",
    "description": "Server connection"
  }
}
```

#### 端口速度設定
```json
{
  "id": "cmd-1699123456790",
  "op": "set_port_speed",
  "schema": "cmd.set_port_speed/1.0",
  "args": {
    "port_id": 3,
    "speed": "1000",
    "duplex": "full",
    "auto_negotiation": false
  }
}
```

### VLAN管理命令

#### 創建VLAN
```json
{
  "id": "cmd-1699123456791",
  "op": "create_vlan",
  "schema": "cmd.create_vlan/1.0",
  "args": {
    "vlan_id": 300,
    "name": "IoT_Network",
    "description": "IoT devices network"
  }
}
```

#### 設定端口VLAN
```json
{
  "id": "cmd-1699123456792",
  "op": "set_port_vlan",
  "schema": "cmd.set_port_vlan/1.0",
  "args": {
    "port_id": 10,
    "vlan_id": 300,
    "mode": "access"
  }
}
```

### PoE管理命令

#### PoE端口控制
```json
{
  "id": "cmd-1699123456793",
  "op": "set_poe_state",
  "schema": "cmd.set_poe_state/1.0",
  "args": {
    "port_id": 8,
    "poe_enabled": true,
    "power_limit_w": 30,
    "priority": "high"
  }
}
```

### 流量管理命令

#### 端口鏡像設定
```json
{
  "id": "cmd-1699123456794",
  "op": "set_port_mirror",
  "schema": "cmd.set_port_mirror/1.0",
  "args": {
    "source_ports": [1, 2, 3],
    "destination_port": 24,
    "direction": "both",
    "enabled": true
  }
}
```

#### QoS設定
```json
{
  "id": "cmd-1699123456795",
  "op": "set_qos_policy",
  "schema": "cmd.set_qos_policy/1.0",
  "args": {
    "port_id": 5,
    "priority_queue": 7,
    "rate_limit_mbps": 100,
    "burst_size_kb": 1000
  }
}
```

## 實作範例

### SNMP-based交換機監控

```python
import paho.mqtt.client as mqtt
import json
import time
from pysnmp.hlapi import *
import threading
from collections import defaultdict

class SwitchMQTTClient:
    def __init__(self, broker_host, switch_ip, community, device_id, tenant="demo", site="datacenter"):
        self.client = mqtt.Client()
        self.broker_host = broker_host
        self.switch_ip = switch_ip
        self.community = community
        self.device_id = device_id
        self.tenant = tenant
        self.site = site
        
        # SNMP OIDs
        self.oids = {
            'sysUpTime': '1.3.6.1.2.1.1.3.0',
            'ifNumber': '1.3.6.1.2.1.2.1.0',
            'ifDescr': '1.3.6.1.2.1.2.2.1.2',
            'ifOperStatus': '1.3.6.1.2.1.2.2.1.8',
            'ifSpeed': '1.3.6.1.2.1.2.2.1.5',
            'ifInOctets': '1.3.6.1.2.1.2.2.1.10',
            'ifOutOctets': '1.3.6.1.2.1.2.2.1.16',
            'dot1qVlanStaticName': '1.3.6.1.2.1.17.7.1.4.3.1.1',
            'pethPsePortPowerDeniedCounter': '1.3.6.1.2.1.105.1.1.1.1.5'
        }
        
        # 數據快取
        self.port_data = {}
        self.vlan_data = {}
        self.poe_data = {}
        
        # 設定MQTT回調
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        
        # 啟動SNMP監控線程
        self.monitoring_active = True
        self.start_monitoring()
        
    def on_connect(self, client, userdata, flags, rc):
        print(f"Switch {self.device_id} connected with result code {rc}")
        cmd_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/cmd/req"
        client.subscribe(cmd_topic, qos=1)
        
    def on_message(self, client, userdata, msg):
        try:
            command = json.loads(msg.payload.decode())
            self.handle_command(command)
        except Exception as e:
            print(f"Error processing command: {e}")
            
    def handle_command(self, command):
        cmd_id = command.get("id")
        operation = command.get("op")
        args = command.get("args", {})
        
        # 發送確認
        self.send_ack(cmd_id)
        
        try:
            if operation == "set_port_state":
                result = self.set_port_state(args)
            elif operation == "create_vlan":
                result = self.create_vlan(args)
            elif operation == "set_port_vlan":
                result = self.set_port_vlan(args)
            elif operation == "set_poe_state":
                result = self.set_poe_state(args)
            elif operation == "get_system_info":
                result = self.get_system_info()
            else:
                result = {"error": "Unsupported switch command"}
                
            self.send_result(cmd_id, operation, result)
            
        except Exception as e:
            self.send_result(cmd_id, operation, {"error": str(e)})
            
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
        
    def snmp_get(self, oid):
        """執行SNMP GET操作"""
        try:
            iterator = getCmd(
                SnmpEngine(),
                CommunityData(self.community),
                UdpTransportTarget((self.switch_ip, 161)),
                ContextData(),
                ObjectType(ObjectIdentity(oid))
            )
            
            errorIndication, errorStatus, errorIndex, varBinds = next(iterator)
            
            if errorIndication or errorStatus:
                return None
                
            return varBinds[0][1]
        except Exception as e:
            print(f"SNMP GET error: {e}")
            return None
            
    def snmp_walk(self, oid):
        """執行SNMP WALK操作"""
        try:
            results = []
            for (errorIndication, errorStatus, errorIndex, varBinds) in nextCmd(
                SnmpEngine(),
                CommunityData(self.community),
                UdpTransportTarget((self.switch_ip, 161)),
                ContextData(),
                ObjectType(ObjectIdentity(oid)),
                lexicographicMode=False):
                
                if errorIndication or errorStatus:
                    break
                    
                for varBind in varBinds:
                    results.append(varBind)
                    
            return results
        except Exception as e:
            print(f"SNMP WALK error: {e}")
            return []
            
    def get_port_status(self):
        """獲取所有端口狀態"""
        ports = []
        
        # 獲取端口描述
        descriptions = self.snmp_walk(self.oids['ifDescr'])
        # 獲取端口狀態
        statuses = self.snmp_walk(self.oids['ifOperStatus'])
        # 獲取端口速度
        speeds = self.snmp_walk(self.oids['ifSpeed'])
        
        for i, desc_var in enumerate(descriptions):
            port_id = i + 1
            description = str(desc_var[1])
            
            # 過濾非實體端口
            if 'Vlan' in description or 'Loopback' in description:
                continue
                
            status_val = int(statuses[i][1]) if i < len(statuses) else 2
            speed_val = int(speeds[i][1]) if i < len(speeds) else 0
            
            port_info = {
                "port_id": port_id,
                "name": description,
                "status": "up" if status_val == 1 else "down",
                "link_speed": str(speed_val // 1000000) if speed_val > 0 else "unknown",
                "duplex": "full",  # 簡化處理
                "last_change": int(time.time() * 1000)
            }
            
            ports.append(port_info)
            
        return ports
        
    def get_vlan_info(self):
        """獲取VLAN信息"""
        vlans = []
        vlan_names = self.snmp_walk(self.oids['dot1qVlanStaticName'])
        
        for vlan_var in vlan_names:
            vlan_id = int(str(vlan_var[0]).split('.')[-1])
            vlan_name = str(vlan_var[1])
            
            vlan_info = {
                "vlan_id": vlan_id,
                "name": vlan_name,
                "status": "active",
                "port_count": 0,  # 需要額外查詢
                "tagged_ports": [],
                "untagged_ports": []
            }
            
            vlans.append(vlan_info)
            
        return vlans
        
    def publish_state(self):
        """發布交換機狀態"""
        state_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/state"
        
        uptime_ticks = self.snmp_get(self.oids['sysUpTime'])
        uptime_s = int(uptime_ticks) // 100 if uptime_ticks else 0
        
        active_ports = len([p for p in self.port_data if p.get('status') == 'up'])
        
        state_msg = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "health": "ok",
            "uptime_s": uptime_s,
            "active_ports": active_ports,
            "total_ports": len(self.port_data),
            "vlan_count": len(self.vlan_data)
        }
        
        self.client.publish(state_topic, json.dumps(state_msg), qos=1, retain=True)
        
    def publish_port_telemetry(self):
        """發布端口遙測數據"""
        ports_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/telemetry/ports"
        
        telemetry_msg = {
            "schema": "telemetry.ports/1.0",
            "ts": int(time.time() * 1000),
            "ports": self.port_data
        }
        
        self.client.publish(ports_topic, json.dumps(telemetry_msg), qos=0)
        
    def publish_vlan_telemetry(self):
        """發布VLAN遙測數據"""
        vlan_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/telemetry/vlans"
        
        telemetry_msg = {
            "schema": "telemetry.vlans/1.0",
            "ts": int(time.time() * 1000),
            "vlans": self.vlan_data
        }
        
        self.client.publish(vlan_topic, json.dumps(telemetry_msg), qos=0)
        
    def monitor_ports(self):
        """監控端口狀態變化"""
        old_port_status = {}
        
        while self.monitoring_active:
            try:
                current_ports = self.get_port_status()
                
                for port in current_ports:
                    port_id = port['port_id']
                    current_status = port['status']
                    old_status = old_port_status.get(port_id, {}).get('status')
                    
                    # 檢測狀態變化
                    if old_status and old_status != current_status:
                        self.publish_port_event(port, old_status, current_status)
                        
                    old_port_status[port_id] = port
                    
                self.port_data = current_ports
                
            except Exception as e:
                print(f"Port monitoring error: {e}")
                
            time.sleep(10)
            
    def publish_port_event(self, port, old_status, new_status):
        """發布端口事件"""
        evt_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/evt/port"
        
        event_type = "link_up" if new_status == "up" else "link_down"
        
        evt_msg = {
            "schema": "evt.port/1.0",
            "ts": int(time.time() * 1000),
            "event_type": event_type,
            "port_id": port['port_id'],
            "port_name": port['name'],
            "link_speed": port['link_speed'],
            "previous_status": old_status
        }
        
        self.client.publish(evt_topic, json.dumps(evt_msg), qos=1)
        
    def start_monitoring(self):
        """啟動監控線程"""
        self.port_monitor_thread = threading.Thread(target=self.monitor_ports)
        self.port_monitor_thread.daemon = True
        self.port_monitor_thread.start()
        
    def start(self):
        self.client.connect(self.broker_host, 1883, 60)
        self.client.loop_start()
        
        # 定期發布狀態和遙測
        while True:
            try:
                self.publish_state()
                self.publish_port_telemetry()
                self.publish_vlan_telemetry()
                time.sleep(30)
            except KeyboardInterrupt:
                print("Shutting down switch monitoring...")
                self.monitoring_active = False
                break

# 使用範例
if __name__ == "__main__":
    switch_client = SwitchMQTTClient(
        broker_host="localhost",
        switch_ip="192.168.1.10", 
        community="public",
        device_id="switch_001"
    )
    switch_client.start()
```

### 簡化的交換機模擬器

```python
import random
import time
import json

class SwitchSimulator:
    def __init__(self, port_count=24):
        self.port_count = port_count
        self.ports = self.initialize_ports()
        self.vlans = self.initialize_vlans()
        
    def initialize_ports(self):
        ports = []
        for i in range(1, self.port_count + 1):
            port = {
                "port_id": i,
                "name": f"GigabitEthernet1/0/{i}",
                "status": random.choice(["up", "down"]),
                "link_speed": "1000" if random.random() > 0.2 else "100",
                "duplex": "full",
                "vlan_id": 1,
                "rx_bytes": random.randint(1000000, 100000000),
                "tx_bytes": random.randint(1000000, 100000000),
                "rx_packets": random.randint(10000, 1000000),
                "tx_packets": random.randint(10000, 1000000),
                "rx_errors": random.randint(0, 100),
                "tx_errors": random.randint(0, 100)
            }
            ports.append(port)
        return ports
        
    def initialize_vlans(self):
        return [
            {"vlan_id": 1, "name": "default", "status": "active", "port_count": 20},
            {"vlan_id": 100, "name": "Management", "status": "active", "port_count": 2},
            {"vlan_id": 200, "name": "Guest", "status": "active", "port_count": 2}
        ]
        
    def get_random_state(self):
        return {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "health": "ok",
            "uptime_s": random.randint(86400, 2592000),
            "cpu_usage": random.uniform(10, 30),
            "memory_usage": random.uniform(20, 50),
            "temperature_c": random.uniform(35, 50),
            "active_ports": len([p for p in self.ports if p["status"] == "up"]),
            "total_ports": self.port_count,
            "vlan_count": len(self.vlans)
        }
        
    def simulate_port_change(self):
        # 隨機改變某個端口狀態
        port = random.choice(self.ports)
        old_status = port["status"]
        port["status"] = "down" if old_status == "up" else "up"
        
        return {
            "schema": "evt.port/1.0",
            "ts": int(time.time() * 1000),
            "event_type": "link_up" if port["status"] == "up" else "link_down",
            "port_id": port["port_id"],
            "port_name": port["name"],
            "previous_status": old_status
        }

# 使用模擬器進行測試
simulator = SwitchSimulator()
print(json.dumps(simulator.get_random_state(), indent=2))
```

## 測試與驗證

### 交換機功能測試清單
- [ ] 端口狀態監控準確性
- [ ] VLAN配置正確性
- [ ] PoE功率管理
- [ ] 生成樹協議收斂
- [ ] 流量統計精確度

### 效能測試場景

1. **高密度端口測試**
   - 48端口同時up/down
   - 測量事件響應時間
   - 監控CPU和記憶體使用

2. **VLAN擴展測試**
   - 創建大量VLAN
   - 端口VLAN成員變更
   - 廣播風暴處理

3. **PoE負載測試**
   - 最大功率輸出測試
   - 功率不足處理
   - 熱保護機制

### 故障模擬

```bash
# 模擬端口故障
mosquitto_pub -h localhost -t "rtk/v1/demo/datacenter/switch_001/evt/port" -m '{
  "schema": "evt.port/1.0",
  "ts": 1699123456789,
  "event_type": "link_down",
  "port_id": 5,
  "port_name": "GigabitEthernet1/0/5"
}'

# 模擬PoE過載
mosquitto_pub -h localhost -t "rtk/v1/demo/datacenter/switch_001/evt/poe" -m '{
  "schema": "evt.poe/1.0", 
  "ts": 1699123456789,
  "event_type": "power_budget_exceeded",
  "total_budget_w": 740,
  "current_usage_w": 755
}'
```

## 故障排除

### 常見問題

1. **SNMP無回應**
   - 檢查community string
   - 驗證SNMP版本設定
   - 確認防火牆規則

2. **端口狀態不準確**
   - 檢查SNMP OID對應
   - 驗證輪詢間隔設定
   - 確認交換機型號支援

3. **PoE監控異常**
   - 驗證交換機PoE支援
   - 檢查PoE相關MIB
   - 確認功率計算公式

### 診斷工具

```bash
# SNMP測試
snmpget -v2c -c public 192.168.1.10 1.3.6.1.2.1.1.3.0
snmpwalk -v2c -c public 192.168.1.10 1.3.6.1.2.1.2.2.1.2

# 端口監控
mosquitto_sub -h localhost -t "rtk/v1/+/+/+/telemetry/ports" -v

# 事件監控  
mosquitto_sub -h localhost -t "rtk/v1/+/+/+/evt/#" -v
```

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Commands and Events Reference](../core/COMMANDS_EVENTS_REFERENCE.md)
- [Topic Structure Guide](../core/TOPIC_STRUCTURE.md)
- [Schema Reference](../core/SCHEMA_REFERENCE.md)
- [RFC 2863 - The Interfaces Group MIB](https://tools.ietf.org/html/rfc2863)
- [IEEE 802.1Q VLAN Standard](https://standards.ieee.org/standard/802_1Q-2018.html)
- [IEEE 802.3at PoE+ Standard](https://standards.ieee.org/standard/802_3at-2009.html)
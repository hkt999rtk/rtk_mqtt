# IoT 設備 MQTT 整合指南

## 概述

本文檔提供各種IoT設備與RTK MQTT協議整合的完整指南，涵蓋感測器、智慧家電、工業設備等不同類型的IoT設備。

## IoT設備分類

### 感測器設備
```json
{
  "device_type": "sensor",
  "sensor_types": ["temperature", "humidity", "motion", "light"],
  "capabilities": ["telemetry", "threshold_alerts"],
  "power_mode": "battery"
}
```

### 智慧家電
```json
{
  "device_type": "smart_appliance", 
  "appliance_type": "thermostat",
  "capabilities": ["control", "scheduling", "energy_monitoring"],
  "power_mode": "mains"
}
```

### 工業設備
```json
{
  "device_type": "industrial",
  "equipment_type": "pump_controller",
  "capabilities": ["monitoring", "control", "maintenance_alerts"],
  "protocols": ["modbus", "bacnet"]
}
```

## MQTT 主題結構

### 基本主題格式
```
rtk/v1/{tenant}/{site}/{device_id}/{message_type}[/{sub_type}]
```

### IoT 設備專用主題
```
rtk/v1/{tenant}/{site}/{device_id}/state
rtk/v1/{tenant}/{site}/{device_id}/telemetry/{sensor_type}
rtk/v1/{tenant}/{site}/{device_id}/evt/{event_type}
rtk/v1/{tenant}/{site}/{device_id}/cmd/req
rtk/v1/{tenant}/{site}/{device_id}/cmd/ack
rtk/v1/{tenant}/{site}/{device_id}/cmd/res
```

## 狀態訊息 (state)

### 感測器狀態
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "health": "ok",
  "battery_level": 85,
  "signal_strength": -65,
  "last_reading": 1699123456000,
  "sensor_status": {
    "temperature": "active",
    "humidity": "active",
    "motion": "inactive"
  }
}
```

### 智慧家電狀態
```json
{
  "schema": "state/1.0", 
  "ts": 1699123456789,
  "health": "ok",
  "power_state": "on",
  "operating_mode": "auto",
  "current_setpoint": 22.5,
  "system_status": "heating",
  "energy_usage_w": 150
}
```

### 工業設備狀態
```json
{
  "schema": "state/1.0",
  "ts": 1699123456789,
  "health": "ok", 
  "operational_state": "running",
  "runtime_hours": 1234.5,
  "maintenance_due": false,
  "alarm_status": "clear",
  "production_rate": 95.2
}
```

## 遙測數據 (telemetry)

### 環境感測器遙測
```json
{
  "schema": "telemetry.environment/1.0",
  "ts": 1699123456789,
  "temperature_c": 23.5,
  "humidity_percent": 65.2,
  "pressure_hpa": 1013.25,
  "air_quality_index": 42,
  "light_lux": 350,
  "uv_index": 3
}
```

### 運動感測器遙測
```json
{
  "schema": "telemetry.motion/1.0",
  "ts": 1699123456789,
  "motion_detected": true,
  "detection_count": 5,
  "last_motion": 1699123450000,
  "sensitivity_level": 7,
  "detection_zone": "main_area"
}
```

### 能耗監控遙測
```json
{
  "schema": "telemetry.energy/1.0",
  "ts": 1699123456789,
  "power_w": 125.5,
  "energy_kwh": 2.45,
  "voltage_v": 230.2,
  "current_a": 0.55,
  "power_factor": 0.95,
  "frequency_hz": 50.1
}
```

### 工業設備遙測
```json
{
  "schema": "telemetry.industrial/1.0",
  "ts": 1699123456789,
  "pressure_bar": 3.2,
  "flow_rate_lpm": 45.8,
  "temperature_c": 68.5,
  "vibration_mm_s": 2.1,
  "rpm": 1450,
  "efficiency_percent": 87.3
}
```

## 事件通知 (evt)

### 閾值警報事件
```json
{
  "schema": "evt.threshold/1.0",
  "ts": 1699123456789,
  "event_type": "threshold_exceeded",
  "sensor_type": "temperature",
  "current_value": 35.2,
  "threshold_value": 30.0,
  "threshold_type": "max",
  "severity": "warning",
  "duration_s": 120
}
```

### 設備故障事件
```json
{
  "schema": "evt.fault/1.0",
  "ts": 1699123456789,
  "event_type": "sensor_failure",
  "component": "humidity_sensor",
  "fault_code": "SENSOR_001",
  "description": "Humidity sensor reading out of range",
  "severity": "critical",
  "recommended_action": "Replace sensor"
}
```

### 維護事件
```json
{
  "schema": "evt.maintenance/1.0",
  "ts": 1699123456789,
  "event_type": "maintenance_due",
  "maintenance_type": "preventive",
  "component": "filter",
  "due_date": 1699200000000,
  "priority": "medium",
  "estimated_duration_hours": 2
}
```

## 支援的命令

### 設備控制命令

#### 設定感測器閾值
```json
{
  "id": "cmd-1699123456789",
  "op": "set_threshold",
  "schema": "cmd.set_threshold/1.0",
  "args": {
    "sensor_type": "temperature",
    "threshold_type": "max",
    "value": 30.0,
    "hysteresis": 1.0
  }
}
```

#### 更新設備配置
```json
{
  "id": "cmd-1699123456790",
  "op": "update_config", 
  "schema": "cmd.update_config/1.0",
  "args": {
    "sampling_interval": 60,
    "reporting_interval": 300,
    "power_save_mode": true
  }
}
```

#### 執行校準
```json
{
  "id": "cmd-1699123456791",
  "op": "calibrate_sensor",
  "schema": "cmd.calibrate_sensor/1.0",
  "args": {
    "sensor_type": "temperature",
    "reference_value": 25.0,
    "calibration_type": "offset"
  }
}
```

#### 控制家電設備
```json
{
  "id": "cmd-1699123456792",
  "op": "set_temperature",
  "schema": "cmd.set_temperature/1.0",
  "args": {
    "target_temperature": 22.5,
    "mode": "auto",
    "schedule_enabled": true
  }
}
```

## 實作範例

### Arduino/ESP32 實作

```cpp
#include <WiFi.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>
#include <DHT.h>

// DHT感測器設定
#define DHT_PIN 2
#define DHT_TYPE DHT22
DHT dht(DHT_PIN, DHT_TYPE);

// WiFi和MQTT設定
const char* ssid = "YOUR_WIFI_SSID";
const char* password = "YOUR_WIFI_PASSWORD";
const char* mqtt_server = "your.mqtt.broker.com";
const char* device_id = "sensor_001";
const char* tenant = "demo";
const char* site = "home";

WiFiClient espClient;
PubSubClient client(espClient);

void setup() {
  Serial.begin(115200);
  dht.begin();
  
  setup_wifi();
  client.setServer(mqtt_server, 1883);
  client.setCallback(callback);
}

void setup_wifi() {
  delay(10);
  Serial.println();
  Serial.print("Connecting to ");
  Serial.println(ssid);

  WiFi.begin(ssid, password);
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }

  Serial.println("");
  Serial.println("WiFi connected");
  Serial.println("IP address: ");
  Serial.println(WiFi.localIP());
}

void callback(char* topic, byte* payload, unsigned int length) {
  Serial.print("Message arrived [");
  Serial.print(topic);
  Serial.print("] ");
  
  String message;
  for (int i = 0; i < length; i++) {
    message += (char)payload[i];
  }
  Serial.println(message);
  
  // 解析命令
  DynamicJsonDocument doc(1024);
  deserializeJson(doc, message);
  
  String cmd_id = doc["id"];
  String operation = doc["op"];
  
  // 發送確認
  send_ack(cmd_id);
  
  // 處理命令
  if (operation == "set_threshold") {
    handle_set_threshold(doc["args"], cmd_id);
  } else if (operation == "update_config") {
    handle_update_config(doc["args"], cmd_id);
  } else {
    send_error(cmd_id, "Unsupported command");
  }
}

void send_ack(String cmd_id) {
  String ack_topic = "rtk/v1/" + String(tenant) + "/" + String(site) + "/" + String(device_id) + "/cmd/ack";
  
  DynamicJsonDocument doc(512);
  doc["id"] = cmd_id;
  doc["schema"] = "cmd.ack/1.0";
  doc["status"] = "accepted";
  doc["ts"] = millis();
  
  String ack_msg;
  serializeJson(doc, ack_msg);
  client.publish(ack_topic.c_str(), ack_msg.c_str(), true);
}

void handle_set_threshold(JsonObject args, String cmd_id) {
  String sensor_type = args["sensor_type"];
  float value = args["value"];
  
  // 儲存閾值設定
  // ... 實作閾值設定邏輯
  
  // 發送結果
  String res_topic = "rtk/v1/" + String(tenant) + "/" + String(site) + "/" + String(device_id) + "/cmd/res";
  
  DynamicJsonDocument doc(512);
  doc["id"] = cmd_id;
  doc["schema"] = "cmd.set_threshold.result/1.0";
  doc["status"] = "completed";
  doc["result"]["threshold_set"] = true;
  doc["result"]["sensor_type"] = sensor_type;
  doc["result"]["value"] = value;
  doc["ts"] = millis();
  
  String res_msg;
  serializeJson(doc, res_msg);
  client.publish(res_topic.c_str(), res_msg.c_str());
}

void publish_state() {
  String state_topic = "rtk/v1/" + String(tenant) + "/" + String(site) + "/" + String(device_id) + "/state";
  
  DynamicJsonDocument doc(512);
  doc["schema"] = "state/1.0";
  doc["ts"] = millis();
  doc["health"] = "ok";
  doc["battery_level"] = 85;
  doc["signal_strength"] = WiFi.RSSI();
  
  String state_msg;
  serializeJson(doc, state_msg);
  client.publish(state_topic.c_str(), state_msg.c_str(), true);
}

void publish_telemetry() {
  float humidity = dht.readHumidity();
  float temperature = dht.readTemperature();
  
  if (isnan(humidity) || isnan(temperature)) {
    Serial.println("Failed to read from DHT sensor!");
    return;
  }
  
  String telemetry_topic = "rtk/v1/" + String(tenant) + "/" + String(site) + "/" + String(device_id) + "/telemetry/environment";
  
  DynamicJsonDocument doc(512);
  doc["schema"] = "telemetry.environment/1.0";
  doc["ts"] = millis();
  doc["temperature_c"] = temperature;
  doc["humidity_percent"] = humidity;
  
  String telemetry_msg;
  serializeJson(doc, telemetry_msg);
  client.publish(telemetry_topic.c_str(), telemetry_msg.c_str());
}

void reconnect() {
  while (!client.connected()) {
    Serial.print("Attempting MQTT connection...");
    if (client.connect(device_id)) {
      Serial.println("connected");
      // 訂閱命令主題
      String cmd_topic = "rtk/v1/" + String(tenant) + "/" + String(site) + "/" + String(device_id) + "/cmd/req";
      client.subscribe(cmd_topic.c_str());
    } else {
      Serial.print("failed, rc=");
      Serial.print(client.state());
      Serial.println(" try again in 5 seconds");
      delay(5000);
    }
  }
}

void loop() {
  if (!client.connected()) {
    reconnect();
  }
  client.loop();
  
  static unsigned long lastMsg = 0;
  unsigned long now = millis();
  
  // 每30秒發送遙測數據
  if (now - lastMsg > 30000) {
    lastMsg = now;
    publish_telemetry();
  }
  
  // 每5分鐘發送狀態
  static unsigned long lastState = 0;
  if (now - lastState > 300000) {
    lastState = now;
    publish_state();
  }
}
```

### Python Raspberry Pi 實作

```python
import paho.mqtt.client as mqtt
import json
import time
import RPi.GPIO as GPIO
import w1thermsensor
from threading import Timer

class IoTMQTTClient:
    def __init__(self, broker_host, device_id, tenant="demo", site="home"):
        self.client = mqtt.Client()
        self.broker_host = broker_host
        self.device_id = device_id
        self.tenant = tenant
        self.site = site
        
        # 感測器設定
        self.temperature_sensor = w1thermsensor.W1ThermSensor()
        
        # GPIO設定
        GPIO.setmode(GPIO.BCM)
        GPIO.setup(18, GPIO.IN, pull_up_down=GPIO.PUD_UP)  # 運動感測器
        GPIO.setup(24, GPIO.OUT)  # 繼電器控制
        
        # 閾值設定
        self.temperature_threshold = 30.0
        self.threshold_hysteresis = 1.0
        
        # 設定MQTT回調
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        
    def on_connect(self, client, userdata, flags, rc):
        print(f"Connected with result code {rc}")
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
        
        # 處理命令
        try:
            if operation == "set_threshold":
                result = self.set_threshold(args)
            elif operation == "update_config":
                result = self.update_config(args)
            elif operation == "get_system_info":
                result = self.get_system_info()
            elif operation == "control_relay":
                result = self.control_relay(args)
            else:
                result = {"error": "Unsupported command"}
                
            # 發送結果
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
        
    def set_threshold(self, args):
        sensor_type = args.get("sensor_type")
        threshold_value = args.get("value")
        hysteresis = args.get("hysteresis", 1.0)
        
        if sensor_type == "temperature":
            self.temperature_threshold = threshold_value
            self.threshold_hysteresis = hysteresis
            
        return {
            "threshold_set": True,
            "sensor_type": sensor_type,
            "value": threshold_value,
            "hysteresis": hysteresis
        }
        
    def control_relay(self, args):
        state = args.get("state", "off")
        
        if state == "on":
            GPIO.output(24, GPIO.HIGH)
        else:
            GPIO.output(24, GPIO.LOW)
            
        return {"relay_state": state}
        
    def get_system_info(self):
        import platform
        import psutil
        
        return {
            "platform": platform.platform(),
            "cpu_usage": psutil.cpu_percent(),
            "memory_usage": psutil.virtual_memory().percent,
            "disk_usage": psutil.disk_usage('/').percent,
            "uptime": time.time() - psutil.boot_time()
        }
        
    def publish_state(self):
        state_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/state"
        state_msg = {
            "schema": "state/1.0",
            "ts": int(time.time() * 1000),
            "health": "ok",
            "sensor_status": {
                "temperature": "active",
                "motion": "active"
            },
            "relay_state": "on" if GPIO.input(24) else "off"
        }
        self.client.publish(state_topic, json.dumps(state_msg), qos=1, retain=True)
        
    def publish_telemetry(self):
        try:
            # 溫度感測器
            temperature = self.temperature_sensor.get_temperature()
            
            # 運動感測器
            motion_detected = not GPIO.input(18)  # 反相邏輯
            
            telemetry_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/telemetry/environment"
            telemetry_msg = {
                "schema": "telemetry.environment/1.0",
                "ts": int(time.time() * 1000),
                "temperature_c": temperature,
                "motion_detected": motion_detected
            }
            self.client.publish(telemetry_topic, json.dumps(telemetry_msg), qos=0)
            
            # 檢查溫度閾值
            self.check_temperature_threshold(temperature)
            
        except Exception as e:
            print(f"Error reading sensors: {e}")
            
    def check_temperature_threshold(self, current_temp):
        if current_temp > self.temperature_threshold:
            evt_topic = f"rtk/v1/{self.tenant}/{self.site}/{self.device_id}/evt/threshold"
            evt_msg = {
                "schema": "evt.threshold/1.0",
                "ts": int(time.time() * 1000),
                "event_type": "threshold_exceeded",
                "sensor_type": "temperature",
                "current_value": current_temp,
                "threshold_value": self.temperature_threshold,
                "threshold_type": "max",
                "severity": "warning"
            }
            self.client.publish(evt_topic, json.dumps(evt_msg), qos=1)
            
    def start(self):
        self.client.connect(self.broker_host, 1883, 60)
        self.client.loop_start()
        
        # 定期任務
        def publish_periodic():
            self.publish_state()
            self.publish_telemetry()
            Timer(30.0, publish_periodic).start()
            
        publish_periodic()
        
        try:
            while True:
                time.sleep(1)
        except KeyboardInterrupt:
            print("Shutting down...")
            GPIO.cleanup()
            self.client.disconnect()

# 使用範例
if __name__ == "__main__":
    iot_client = IoTMQTTClient("localhost", "rpi_sensor_001")
    iot_client.start()
```

## 電源管理策略

### 電池供電設備

#### 省電模式配置
```json
{
  "power_save_config": {
    "sleep_interval_s": 300,
    "wake_duration_s": 10,
    "deep_sleep_enabled": true,
    "low_battery_threshold": 15
  }
}
```

#### 電池狀態監控
```json
{
  "schema": "telemetry.battery/1.0",
  "ts": 1699123456789,
  "battery_level": 75,
  "voltage": 3.7,
  "charging": false,
  "estimated_runtime_hours": 48,
  "power_consumption_mw": 125
}
```

### 主電源設備

#### 功耗監控
```json
{
  "schema": "telemetry.power/1.0",
  "ts": 1699123456789,
  "power_consumption_w": 5.2,
  "voltage_v": 5.0,
  "current_ma": 1040,
  "power_state": "active"
}
```

## 測試與驗證

### 感測器測試清單
- [ ] 溫濕度讀取準確性
- [ ] 運動檢測靈敏度
- [ ] 光照度測量範圍
- [ ] 空氣品質指數計算

### 命令執行測試
- [ ] 閾值設定命令
- [ ] 配置更新命令
- [ ] 校準命令執行
- [ ] 設備控制命令

### 事件觸發測試
- [ ] 閾值超限事件
- [ ] 設備故障事件
- [ ] 電池低電量警報
- [ ] 維護提醒事件

### 整合測試場景

1. **感測器監控流程**
   - 定期收集感測器數據
   - 閾值檢查和警報
   - 數據品質驗證

2. **遠端控制流程**
   - 接收控制命令
   - 執行設備操作
   - 回報執行結果

3. **省電模式測試**
   - 深度睡眠功能
   - 定時喚醒機制
   - 低電量處理

## 故障排除

### 常見問題

1. **感測器讀取失敗**
   - 檢查感測器連接
   - 驗證GPIO設定
   - 確認驅動程式正確

2. **MQTT連線不穩定**
   - 網路訊號強度檢查
   - Keep-alive設定調整
   - 自動重連機制

3. **電池耗電過快**
   - 調整取樣頻率
   - 啟用省電模式
   - 檢查休眠機制

### 診斷命令
```bash
# 檢查感測器狀態
cat /sys/class/thermal/thermal_zone0/temp

# 監控GPIO狀態
gpio readall

# 檢查網路連線
ping -c 3 your.mqtt.broker.com

# 監控MQTT訊息
mosquitto_sub -h localhost -t "rtk/v1/+/+/+/#" -v
```

## 安全考量

### 設備認證
- 使用唯一設備憑證
- 定期更新認證資訊
- 實作設備證書驗證

### 數據加密
- MQTT over TLS
- 敏感數據欄位加密
- 金鑰管理最佳實務

### 存取控制
- 基於角色的權限控制
- 設備級別的主題權限
- 命令執行授權驗證

## 效能最佳化

### 數據傳輸最佳化
- 壓縮遙測數據
- 批次傳送非即時數據
- 差異化數據上傳

### 資源使用最佳化
- 記憶體池管理
- CPU負載平衡
- 儲存空間優化

## 參考資料

- [RTK MQTT Protocol Specification](../core/MQTT_PROTOCOL_SPEC.md)
- [Commands and Events Reference](../core/COMMANDS_EVENTS_REFERENCE.md)
- [Topic Structure Guide](../core/TOPIC_STRUCTURE.md)
- [Schema Reference](../core/SCHEMA_REFERENCE.md)
- [AP/Router Integration Guide](AP_ROUTER_INTEGRATION.md)
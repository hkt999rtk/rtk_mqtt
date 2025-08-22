# 配置參考指南

## 主配置文件結構

### wrapper.yaml

```yaml
# MQTT Wrapper 主配置文件
wrapper:
  # 基本資訊
  name: "RTK MQTT Wrapper"
  version: "1.0.0"
  
  # MQTT 連接設定
  mqtt:
    broker: "tcp://localhost:1883"
    client_id: "rtk_wrapper"
    username: ""
    password: ""
    keep_alive: 60
    clean_session: true
    
    # TLS 設定
    tls:
      enabled: false
      cert_file: ""
      key_file: ""
      ca_file: ""
      insecure_skip_verify: false
    
    # 連接重試設定
    reconnect:
      enabled: true
      initial_delay: "1s"
      max_delay: "30s"
      multiplier: 2
      max_attempts: 10
  
  # 日志設定
  logging:
    level: "info"  # trace, debug, info, warn, error
    format: "text" # text, json
    output: "stdout" # stdout, stderr, file path
    rotation:
      enabled: false
      max_size: "100MB"
      max_age: "7d"
      max_backups: 3
  
  # 監控和統計
  monitoring:
    enabled: true
    metrics_port: 8080
    health_check_port: 8081
    collect_interval: "30s"
  
  # Wrapper 註冊表
  registry:
    auto_discovery: true
    discovery_timeout: "5s"
    
    # 註冊的 wrapper 列表
    wrappers:
      - name: "homeassistant"
        enabled: true
        config_file: "wrappers/homeassistant.yaml"
        priority: 100
        
      - name: "tasmota"  
        enabled: true
        config_file: "wrappers/tasmota.yaml"
        priority: 90
        
      - name: "xiaomi"
        enabled: false
        config_file: "wrappers/xiaomi.yaml"
        priority: 80
        
      - name: "custom"
        enabled: false
        config_file: "wrappers/custom.yaml"
        priority: 50

  # 性能調優
  performance:
    worker_pool_size: 10
    message_buffer_size: 1000
    batch_processing:
      enabled: false
      batch_size: 50
      flush_interval: "100ms"
    
  # RTK 設定
  rtk:
    # 預設租戶和站點
    default_tenant: "home"
    default_site: "main"
    
    # Topic 前綴
    topic_prefix: "rtk/v1"
    
    # QoS 設定  
    qos:
      state_messages: 1
      telemetry_messages: 0
      event_messages: 1
      command_messages: 2
    
    # Retained 設定
    retained:
      state_messages: true
      attr_messages: true
      lwt_messages: true
      others: false
```

## Wrapper 專用配置

### Home Assistant Wrapper (homeassistant.yaml)

```yaml
# Home Assistant Wrapper 配置
homeassistant:
  name: "Home Assistant Wrapper"
  description: "Converts Home Assistant MQTT messages to RTK format"
  
  # 支援的設備類型
  supported_device_types:
    - light
    - switch
    - sensor
    - climate
    - cover
    - binary_sensor
    - fan
    - lock
  
  # Topic 模式匹配
  topic_patterns:
    # 上行模式（設備 → RTK）
    uplink:
      - pattern: "homeassistant/{device_class}/{device_name}/state"
        priority: 100
        device_id_extract: "{device_name}"
        
      - pattern: "homeassistant/{device_class}/{device_name}/attributes"  
        priority: 95
        device_id_extract: "{device_name}"
        message_type: "attr"
        
      # 支援巢狀命名
      - pattern: "homeassistant/{device_class}/{location}/{device_name}/state"
        priority: 90
        device_id_extract: "{location}_{device_name}"
    
    # 下行模式（RTK → 設備）  
    downlink:
      - pattern: "rtk/v1/{tenant}/{site}/{device_id}/cmd/req"
        priority: 100
        target_topic: "homeassistant/{device_class}/{mapped_name}/set"
  
  # Payload 處理規則
  payload_rules:
    # 狀態訊息轉換
    state_transform:
      # 布林值映射
      boolean_mapping:
        "on": true
        "off": false
        "ON": true
        "OFF": false
      
      # 數值單位轉換
      unit_conversions:
        temperature:
          from_unit: "°F"
          to_unit: "°C"
          formula: "(x - 32) * 5/9"
        
        brightness:
          from_range: [0, 255]
          to_range: [0, 100]
    
    # 屬性提取
    attribute_extraction:
      # 從 HA attributes 提取到 RTK payload
      brightness: "payload.brightness"
      color_temp: "payload.color_temp" 
      rgb_color: "payload.rgb_color"
      
      # 計算衍生屬性
      power_consumption:
        formula: "voltage * current"
        sources: ["voltage", "current"]
  
  # 設備映射表
  device_mapping:
    # HA entity_id → RTK device_id 映射
    "light.living_room": "living_room_light"
    "switch.bedroom_fan": "bedroom_fan_switch"
    "sensor.kitchen_temperature": "kitchen_temp_sensor"
    
  # 錯誤處理
  error_handling:
    invalid_json:
      action: "log_and_drop" # log_and_drop, forward_raw, retry
      max_retries: 3
    
    missing_fields:
      action: "fill_defaults"
      defaults:
        device_id: "unknown_device"
        timestamp: "current_time"
    
    transformation_errors:
      action: "log_and_forward_original"
```

### Tasmota Wrapper (tasmota.yaml)

```yaml
# Tasmota Wrapper 配置
tasmota:
  name: "Tasmota Wrapper"
  description: "Converts Tasmota MQTT messages to RTK format"
  
  # Tasmota 特定設定
  device_prefix: "tasmota"
  
  # Topic 模式
  topic_patterns:
    uplink:
      # STATE 訊息
      - pattern: "{device_prefix}/{device_name}/STATE" 
        priority: 100
        device_id_extract: "{device_name}"
        message_type: "state"
        
      # SENSOR 訊息  
      - pattern: "{device_prefix}/{device_name}/SENSOR"
        priority: 95
        device_id_extract: "{device_name}"
        message_type: "telemetry/sensor"
        
      # LWT 訊息
      - pattern: "{device_prefix}/{device_name}/LWT"
        priority: 90
        device_id_extract: "{device_name}"
        message_type: "lwt"
        
      # INFO 訊息
      - pattern: "{device_prefix}/{device_name}/INFO{info_num}"
        priority: 85
        device_id_extract: "{device_name}"
        message_type: "attr"
    
    downlink:
      # 命令模式
      - pattern: "rtk/v1/{tenant}/{site}/{device_id}/cmd/req"
        priority: 100
        target_topic: "{device_prefix}/{mapped_device}/cmnd/{command}"
  
  # Tasmota 狀態解析
  state_parsing:
    # 電源狀態
    power_states:
      field_path: "POWER"
      mapping:
        "ON": true
        "OFF": false
    
    # WiFi 資訊
    wifi_info:
      rssi: "Wifi.RSSI"
      ssid: "Wifi.SSId"
      channel: "Wifi.Channel"
    
    # 硬體資訊
    hardware_info:
      version: "Version"
      hardware: "Hardware" 
      uptime: "UptimeSec"
      heap: "Heap"
  
  # Sensor 數據解析
  sensor_parsing:
    # 溫濕度感測器
    temperature_humidity:
      temperature_path: "DHT22.Temperature"
      humidity_path: "DHT22.Humidity"
      
    # 能源監控
    energy:
      voltage_path: "ENERGY.Voltage"
      current_path: "ENERGY.Current"  
      power_path: "ENERGY.Power"
      total_path: "ENERGY.Total"
  
  # 命令轉換
  command_mapping:
    # RTK 命令 → Tasmota 命令
    "turn_on": 
      command: "POWER"
      value: "ON"
      
    "turn_off":
      command: "POWER"
      value: "OFF"
      
    "set_brightness":
      command: "Dimmer"
      value_path: "payload.brightness"
      
    "set_color":
      command: "Color"
      value_transform: "rgb_to_hex"
      value_path: "payload.rgb_color"
  
  # 設備類型檢測
  device_type_detection:
    rules:
      - condition:
          has_fields: ["POWER"]
        device_type: "switch"
        
      - condition:
          has_fields: ["POWER", "Dimmer"]  
        device_type: "light"
        
      - condition:
          has_fields: ["Temperature", "Humidity"]
        device_type: "sensor"
        
      - condition:
          has_fields: ["ENERGY"]
        device_type: "power_meter"
```

### Xiaomi Wrapper (xiaomi.yaml)

```yaml
# Xiaomi Wrapper 配置
xiaomi:
  name: "Xiaomi Wrapper"
  description: "Converts Xiaomi IoT messages to RTK format"
  
  # 支援的協議
  protocols:
    - miio
    - miot
    - zigbee
  
  # Topic 模式
  topic_patterns:
    uplink:
      # miio 協議
      - pattern: "miio/{device_id}/status"
        priority: 100
        protocol: "miio"
        
      # miot 協議  
      - pattern: "miot/{device_id}/properties"
        priority: 95
        protocol: "miot"
        
      # Zigbee 設備
      - pattern: "zigbee/{gateway_id}/{device_id}/report"
        priority: 90
        protocol: "zigbee"
        device_id_extract: "{gateway_id}_{device_id}"
    
    downlink:
      - pattern: "rtk/v1/{tenant}/{site}/{device_id}/cmd/req"
        target_topic_template: "{protocol}/{extracted_id}/cmd"
  
  # miio 協議設定
  miio:
    # 屬性映射
    property_mapping:
      # 空氣清淨機
      air_purifier:
        power: "power"
        mode: "mode" 
        aqi: "aqi"
        filter_life: "filter1_life"
        
      # 掃地機器人
      vacuum:
        state: "state"
        battery: "battery"  
        fan_speed: "fan_speed"
        clean_area: "clean_area"
    
    # 命令映射
    command_mapping:
      power_on: ["power", true]
      power_off: ["power", false]
      set_mode: ["mode", "${mode}"]
      start_clean: ["app_start", []]
  
  # miot 協議設定  
  miot:
    # 服務和屬性映射
    service_mapping:
      "2:1": # 空氣清淨機服務
        power: "2:1:1"
        mode: "2:1:2"
        fan_level: "2:1:11"
        
      "3:1": # 環境監測服務
        pm25: "3:1:4"
        temperature: "3:1:7"
        humidity: "3:1:8"
  
  # Zigbee 設定
  zigbee:
    # 設備類型映射
    device_types:
      "lumi.sensor_ht": "temperature_humidity_sensor"
      "lumi.sensor_motion": "motion_sensor" 
      "lumi.ctrl_ln2": "switch_2ch"
      
    # 屬性解析
    attribute_parsing:
      temperature: 
        zigbee_attr: "temperature"
        unit_convert: "0.01"  # 除以100
        
      humidity:
        zigbee_attr: "humidity" 
        unit_convert: "0.01"
        
      illuminance:
        zigbee_attr: "illuminance"
        unit_convert: "1"
```

### 自定義 Wrapper (custom.yaml)

```yaml
# 自定義 Wrapper 配置模板
custom:
  name: "Custom Device Wrapper"
  description: "Template for custom device integration"
  
  # 自定義設定
  vendor: "CustomVendor"
  product_line: "CustomProduct"
  
  # Topic 模式（可自定義）
  topic_patterns:
    uplink:
      - pattern: "custom/{device_type}/{device_id}/data"
        priority: 100
        
      - pattern: "vendor/{vendor_id}/{device_id}/status"
        priority: 90
    
    downlink:
      - pattern: "rtk/v1/{tenant}/{site}/{device_id}/cmd/req"
        target_topic: "custom/{device_type}/{device_id}/cmd"
  
  # 自定義轉換規則
  transform_rules:
    # JavaScript 式的轉換函數（偽代碼）
    uplink_transform: |
      function transform(message) {
        return {
          schema: "state/1.0",
          device_id: extractDeviceId(message.topic),
          payload: {
            custom_field1: message.payload.field1,
            custom_field2: convertUnit(message.payload.field2, "custom_unit")
          }
        };
      }
    
    downlink_transform: |
      function transform(rtkMessage) {
        return {
          topic: buildCustomTopic(rtkMessage.device_id),
          payload: {
            cmd: rtkMessage.payload.command,
            params: rtkMessage.payload.parameters
          }
        };
      }
  
  # 驗證規則
  validation:
    required_fields:
      uplink: ["device_id", "timestamp"]
      downlink: ["command"]
    
    field_types:
      device_id: "string"
      timestamp: "number"
      command: "string"
```

## 環境變數設定

```bash
# MQTT 連接
export RTK_WRAPPER_MQTT_BROKER="tcp://mqtt.example.com:1883"
export RTK_WRAPPER_MQTT_USERNAME="rtk_wrapper"  
export RTK_WRAPPER_MQTT_PASSWORD="wrapper_password"

# TLS 設定
export RTK_WRAPPER_TLS_ENABLED="true"
export RTK_WRAPPER_TLS_CERT_FILE="/etc/ssl/certs/wrapper.crt"
export RTK_WRAPPER_TLS_KEY_FILE="/etc/ssl/private/wrapper.key"

# 日志設定
export RTK_WRAPPER_LOG_LEVEL="info"
export RTK_WRAPPER_LOG_FORMAT="json"

# 監控設定  
export RTK_WRAPPER_METRICS_PORT="8080"
export RTK_WRAPPER_HEALTH_PORT="8081"

# RTK 設定
export RTK_WRAPPER_DEFAULT_TENANT="production"
export RTK_WRAPPER_DEFAULT_SITE="datacenter1"
```

## 配置驗證

### 配置檢查工具

創建配置驗證腳本 `scripts/validate_config.py`：

```python
#!/usr/bin/env python3
import yaml
import sys
import jsonschema

def validate_config(config_file):
    """驗證 wrapper 配置文件"""
    
    # 載入配置 schema
    with open('schemas/wrapper_config_schema.json', 'r') as f:
        schema = json.load(f)
    
    # 載入配置文件
    with open(config_file, 'r') as f:
        config = yaml.safe_load(f)
    
    try:
        # 驗證配置
        jsonschema.validate(config, schema)
        print(f"✓ Configuration {config_file} is valid")
        return True
    except jsonschema.ValidationError as e:
        print(f"✗ Configuration validation failed: {e.message}")
        return False

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: validate_config.py <config_file>")
        sys.exit(1)
    
    config_file = sys.argv[1]
    if not validate_config(config_file):
        sys.exit(1)
```

### 配置 Schema

創建配置驗證 schema `schemas/wrapper_config_schema.json`：

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["wrapper"],
  "properties": {
    "wrapper": {
      "type": "object",
      "required": ["name", "mqtt", "registry"],
      "properties": {
        "name": {"type": "string"},
        "version": {"type": "string"},
        "mqtt": {
          "type": "object",
          "required": ["broker"],
          "properties": {
            "broker": {"type": "string", "format": "uri"},
            "client_id": {"type": "string"},
            "username": {"type": "string"},
            "password": {"type": "string"},
            "keep_alive": {"type": "integer", "minimum": 1},
            "clean_session": {"type": "boolean"}
          }
        },
        "registry": {
          "type": "object", 
          "required": ["wrappers"],
          "properties": {
            "auto_discovery": {"type": "boolean"},
            "wrappers": {
              "type": "array",
              "items": {
                "type": "object",
                "required": ["name", "enabled", "config_file"],
                "properties": {
                  "name": {"type": "string"},
                  "enabled": {"type": "boolean"},
                  "config_file": {"type": "string"},
                  "priority": {"type": "integer", "minimum": 0}
                }
              }
            }
          }
        }
      }
    }
  }
}
```

## 範例配置文件

### 開發環境配置

```yaml
# configs/wrapper.dev.yaml - 開發環境配置
wrapper:
  name: "RTK MQTT Wrapper (Development)"
  version: "1.0.0-dev"
  
  mqtt:
    broker: "tcp://localhost:1883"
    client_id: "rtk_wrapper_dev"
    keep_alive: 30
    clean_session: true
  
  logging:
    level: "debug"
    format: "text"
    output: "stdout"
  
  monitoring:
    enabled: true
    metrics_port: 8080
    health_check_port: 8081
  
  registry:
    auto_discovery: true
    wrappers:
      - name: "homeassistant"
        enabled: true
        config_file: "wrappers/homeassistant.yaml"
        priority: 100
  
  rtk:
    default_tenant: "dev"
    default_site: "local"
```

### 生產環境配置

```yaml
# configs/wrapper.prod.yaml - 生產環境配置
wrapper:
  name: "RTK MQTT Wrapper (Production)"
  version: "1.0.0"
  
  mqtt:
    broker: "ssl://mqtt.prod.example.com:8883"
    client_id: "rtk_wrapper_prod"
    username: "${RTK_WRAPPER_MQTT_USERNAME}"
    password: "${RTK_WRAPPER_MQTT_PASSWORD}"
    keep_alive: 60
    clean_session: false
    
    tls:
      enabled: true
      cert_file: "/etc/ssl/certs/wrapper.crt"
      key_file: "/etc/ssl/private/wrapper.key"
      ca_file: "/etc/ssl/certs/ca.crt"
      insecure_skip_verify: false
      
    reconnect:
      enabled: true
      initial_delay: "2s"
      max_delay: "60s"
      multiplier: 2
      max_attempts: 20
  
  logging:
    level: "info"
    format: "json"
    output: "/var/log/rtk-wrapper/wrapper.log"
    rotation:
      enabled: true
      max_size: "100MB"
      max_age: "30d"
      max_backups: 7
  
  monitoring:
    enabled: true
    metrics_port: 8080
    health_check_port: 8081
    collect_interval: "10s"
  
  performance:
    worker_pool_size: 20
    message_buffer_size: 5000
    batch_processing:
      enabled: true
      batch_size: 100
      flush_interval: "50ms"
  
  registry:
    auto_discovery: false
    discovery_timeout: "10s"
    wrappers:
      - name: "homeassistant"
        enabled: true
        config_file: "wrappers/homeassistant.yaml"
        priority: 100
      - name: "tasmota"
        enabled: true  
        config_file: "wrappers/tasmota.yaml"
        priority: 90
  
  rtk:
    default_tenant: "production"
    default_site: "main"
    topic_prefix: "rtk/v1"
```
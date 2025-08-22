# MQTT Wrapper 實現範例

本文檔提供具體的 Wrapper 實現範例，包含多廠牌支援和智能註冊機制。

## 多廠牌智能註冊機制

### WrapperRegistry 實現

```go
// internal/registry/registry.go
package registry

import (
    "fmt"
    "regexp"
    "sort"
    "strings"
    "sync"
)

// WrapperRegistry 管理所有註冊的 Wrapper
type WrapperRegistry struct {
    // 按名稱註冊的 Wrapper
    wrappers map[string]DeviceWrapper
    
    // 上行路由規則（設備 → RTK）
    uplinkRoutes []RouteRule
    
    // 下行路由規則（RTK → 設備）  
    downlinkRoutes []RouteRule
    
    // 設備類型映射
    deviceTypeMapping map[string]string
    
    mu sync.RWMutex
}

// RouteRule 定義路由匹配規則
type RouteRule struct {
    Name        string             `json:"name"`         // Wrapper 名稱
    Priority    int                `json:"priority"`     // 優先級（數字越小優先級越高）
    TopicPattern *regexp.Regexp    `json:"-"`           // Topic 匹配模式
    TopicRegex  string             `json:"topic_regex"`  // Topic 正則表達式
    PayloadRules []PayloadRule     `json:"payload_rules"` // Payload 匹配規則
    DeviceTypes []string           `json:"device_types"` // 支援的設備類型
    Wrapper     DeviceWrapper      `json:"-"`           // Wrapper 實例
}

// PayloadRule 定義 Payload 內容匹配規則
type PayloadRule struct {
    Field     string      `json:"field"`      // JSON 欄位路徑 (支援巢狀，如 "device.manufacturer")
    Operation string      `json:"operation"`  // 操作：equals, contains, exists, regex
    Value     interface{} `json:"value"`      // 期望值
}

// 智能尋找上行 Wrapper（設備 → RTK）
func (r *WrapperRegistry) FindUplinkWrapper(topic string, payload *FlexiblePayload) DeviceWrapper {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    candidates := make([]RouteMatch, 0)
    
    // 評分所有可能的 Wrapper
    for _, rule := range r.uplinkRoutes {
        score := r.calculateMatchScore(rule, topic, payload)
        if score > 0 {
            candidates = append(candidates, RouteMatch{
                Wrapper: rule.Wrapper,
                Rule:    rule,
                Score:   score,
            })
        }
    }
    
    // 按分數排序，返回最佳匹配
    if len(candidates) > 0 {
        sort.Slice(candidates, func(i, j int) bool {
            return candidates[i].Score > candidates[j].Score
        })
        return candidates[0].Wrapper
    }
    
    return nil
}

// 計算匹配分數
func (r *WrapperRegistry) calculateMatchScore(rule RouteRule, topic string, payload *FlexiblePayload) int {
    score := 0
    
    // Topic 匹配得分
    if rule.TopicPattern.MatchString(topic) {
        score += 100
        
        // 更精確的 topic 匹配給更高分數
        specificity := strings.Count(rule.TopicRegex, "+") + strings.Count(rule.TopicRegex, "#")
        score += (10 - specificity) * 10 // 萬用字元越少，分數越高
    } else {
        return 0 // Topic 不匹配直接排除
    }
    
    // Payload 規則匹配得分
    for _, payloadRule := range rule.PayloadRules {
        if r.matchPayloadRule(payloadRule, payload) {
            score += 50
        } else {
            score -= 25 // Payload 規則不匹配扣分
        }
    }
    
    // 優先級調整（數字越小優先級越高，得分越高）
    score -= rule.Priority * 5
    
    return score
}
```

## 具體 Wrapper 實現範例

### 1. Home Assistant Wrapper

```go
// wrappers/homeassistant/wrapper.go
package homeassistant

type HomeAssistantWrapper struct {
    config HomeAssistantConfig
}

func (ha *HomeAssistantWrapper) SupportedDeviceTypes() []string {
    return []string{"light", "switch", "sensor", "climate", "cover"}
}

func (ha *HomeAssistantWrapper) TransformUplink(msg *WrapperMessage) (*RTKMessage, error) {
    // 提取 Home Assistant 特有欄位
    entityID, _ := msg.ParsedPayload.GetString("entity_id")
    state, _ := msg.ParsedPayload.GetString("state")
    
    // 從 entity_id 提取設備類型和 ID
    parts := strings.Split(entityID, ".")
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid entity_id format: %s", entityID)
    }
    
    deviceType := parts[0]  // light, switch, sensor, etc.
    deviceName := parts[1]  // living_room, bedroom, etc.
    
    // 轉換為 RTK 格式
    rtkPayload := map[string]interface{}{
        "health": "ok",
        "status": state,
        "entity_id": entityID,
        "device_type": deviceType,
    }
    
    // 處理屬性資訊
    if attributes, ok := msg.ParsedPayload.GetNested("attributes"); ok {
        if attrMap, ok := attributes.(map[string]interface{}); ok {
            for key, value := range attrMap {
                rtkPayload[key] = value
            }
        }
    }
    
    return &RTKMessage{
        Schema:   "state/1.0",
        TS:       msg.MQTTInfo.Timestamp,
        DeviceID: ha.generateDeviceID(entityID),
        Payload:  rtkPayload,
        Trace: &TraceInfo{
            ReqID: fmt.Sprintf("ha-%d", time.Now().UnixNano()),
        },
    }, nil
}

func (ha *HomeAssistantWrapper) TransformDownlink(msg *WrapperMessage) (*DeviceMessage, error) {
    // 從 RTK 命令轉換為 Home Assistant 服務調用
    var rtkMsg RTKMessage
    if err := json.Unmarshal(msg.MQTTInfo.Payload, &rtkMsg); err != nil {
        return nil, fmt.Errorf("failed to parse RTK message: %w", err)
    }
    
    // 提取命令資訊
    if cmd, ok := rtkMsg.Payload.(map[string]interface{})["command"]; ok {
        switch cmd {
        case "turn_on":
            return &DeviceMessage{
                Topic: fmt.Sprintf("homeassistant/%s/set", ha.extractEntityID(rtkMsg.DeviceID)),
                Payload: map[string]interface{}{
                    "state": "ON",
                },
                QoS: 1,
                Retained: false,
            }, nil
        case "turn_off":
            return &DeviceMessage{
                Topic: fmt.Sprintf("homeassistant/%s/set", ha.extractEntityID(rtkMsg.DeviceID)),
                Payload: map[string]interface{}{
                    "state": "OFF",
                },
                QoS: 1,
                Retained: false,
            }, nil
        }
    }
    
    return nil, fmt.Errorf("unsupported RTK command")
}

func (ha *HomeAssistantWrapper) generateDeviceID(entityID string) string {
    // 將 entity_id 轉換為符合 RTK 格式的 device_id
    // 例如: "light.living_room" -> "ha_light_living_room"
    return fmt.Sprintf("ha_%s", strings.ReplaceAll(entityID, ".", "_"))
}

func (ha *HomeAssistantWrapper) extractEntityID(deviceID string) string {
    // 從 RTK device_id 還原為 entity_id
    // 例如: "ha_light_living_room" -> "light.living_room"
    if strings.HasPrefix(deviceID, "ha_") {
        entityPart := strings.TrimPrefix(deviceID, "ha_")
        parts := strings.Split(entityPart, "_")
        if len(parts) >= 2 {
            return fmt.Sprintf("%s.%s", parts[0], strings.Join(parts[1:], "_"))
        }
    }
    return deviceID
}
```

### 2. Tasmota Wrapper

```go
// wrappers/tasmota/wrapper.go
package tasmota

type TasmotaWrapper struct {
    config TasmotaConfig
}

func (t *TasmotaWrapper) SupportedDeviceTypes() []string {
    return []string{"relay", "sensor", "dimmer", "plug"}
}

func (t *TasmotaWrapper) TransformUplink(msg *WrapperMessage) (*RTKMessage, error) {
    topic := msg.MQTTInfo.Topic
    
    // Tasmota STATE 訊息處理
    if strings.Contains(topic, "STATE") {
        return t.handleStateMessage(msg)
    }
    
    // Tasmota SENSOR 訊息處理
    if strings.Contains(topic, "SENSOR") {
        return t.handleSensorMessage(msg)
    }
    
    return nil, fmt.Errorf("unsupported tasmota topic: %s", topic)
}

func (t *TasmotaWrapper) handleStateMessage(msg *WrapperMessage) (*RTKMessage, error) {
    wifi, _ := msg.ParsedPayload.GetNested("Wifi", "SSId")
    power, _ := msg.ParsedPayload.GetString("POWER")
    uptime, _ := msg.ParsedPayload.GetFloat64("UptimeSec")
    
    return &RTKMessage{
        Schema:   "state/1.0",
        TS:       msg.MQTTInfo.Timestamp,
        DeviceID: t.extractDeviceID(msg.MQTTInfo.Topic),
        Payload: map[string]interface{}{
            "health":     "ok",
            "power_state": power,
            "wifi_ssid":  wifi,
            "uptime_sec": uptime,
        },
        Trace: &TraceInfo{
            ReqID: fmt.Sprintf("tasmota-%d", time.Now().UnixNano()),
        },
    }, nil
}

func (t *TasmotaWrapper) handleSensorMessage(msg *WrapperMessage) (*RTKMessage, error) {
    // 處理各種感測器數據
    sensorData := make(map[string]interface{})
    
    // 溫度感測器
    if temp, ok := msg.ParsedPayload.GetNested("DS18B20", "Temperature"); ok {
        sensorData["temperature"] = temp
    }
    
    // 濕度感測器  
    if humidity, ok := msg.ParsedPayload.GetNested("DHT22", "Humidity"); ok {
        sensorData["humidity"] = humidity
    }
    
    // 能源監控
    if energy, ok := msg.ParsedPayload.GetNested("ENERGY"); ok {
        if energyMap, ok := energy.(map[string]interface{}); ok {
            for key, value := range energyMap {
                sensorData[strings.ToLower(key)] = value
            }
        }
    }
    
    return &RTKMessage{
        Schema:   "telemetry.sensor/1.0",
        TS:       msg.MQTTInfo.Timestamp,
        DeviceID: t.extractDeviceID(msg.MQTTInfo.Topic),
        Payload:  sensorData,
    }, nil
}

func (t *TasmotaWrapper) TransformDownlink(msg *WrapperMessage) (*DeviceMessage, error) {
    // Tasmota 命令格式：cmnd/{device_topic}/{command}
    var rtkMsg RTKMessage
    if err := json.Unmarshal(msg.MQTTInfo.Payload, &rtkMsg); err != nil {
        return nil, fmt.Errorf("failed to parse RTK message: %w", err)
    }
    
    deviceTopic := t.extractDeviceTopic(rtkMsg.DeviceID)
    
    if cmd, ok := rtkMsg.Payload.(map[string]interface{})["command"]; ok {
        switch cmd {
        case "turn_on":
            return &DeviceMessage{
                Topic: fmt.Sprintf("cmnd/%s/POWER", deviceTopic),
                Payload: "ON",
                QoS: 1,
                Retained: false,
            }, nil
        case "turn_off":
            return &DeviceMessage{
                Topic: fmt.Sprintf("cmnd/%s/POWER", deviceTopic),
                Payload: "OFF",
                QoS: 1,
                Retained: false,
            }, nil
        case "set_brightness":
            if brightness, ok := rtkMsg.Payload.(map[string]interface{})["brightness"]; ok {
                return &DeviceMessage{
                    Topic: fmt.Sprintf("cmnd/%s/Dimmer", deviceTopic),
                    Payload: fmt.Sprintf("%v", brightness),
                    QoS: 1,
                    Retained: false,
                }, nil
            }
        }
    }
    
    return nil, fmt.Errorf("unsupported RTK command")
}

func (t *TasmotaWrapper) extractDeviceID(topic string) string {
    // 從 Tasmota topic 提取設備 ID
    // 例如: "tasmota/device01/STATE" -> "tasmota_device01"
    parts := strings.Split(topic, "/")
    if len(parts) >= 2 {
        return fmt.Sprintf("tasmota_%s", parts[1])
    }
    return "unknown_tasmota_device"
}

func (t *TasmotaWrapper) extractDeviceTopic(deviceID string) string {
    // 從 RTK device_id 還原為 Tasmota topic
    // 例如: "tasmota_device01" -> "device01"
    if strings.HasPrefix(deviceID, "tasmota_") {
        return strings.TrimPrefix(deviceID, "tasmota_")
    }
    return deviceID
}
```

### 3. Xiaomi/Mi Wrapper

```go
// wrappers/xiaomi/wrapper.go
package xiaomi

type XiaomiWrapper struct {
    config XiaomiConfig
}

func (x *XiaomiWrapper) SupportedDeviceTypes() []string {
    return []string{"gateway", "sensor", "switch", "motion", "door"}
}

func (x *XiaomiWrapper) TransformUplink(msg *WrapperMessage) (*RTKMessage, error) {
    // Xiaomi miio 協議處理
    if strings.Contains(msg.MQTTInfo.Topic, "miio") {
        return x.handleMiioMessage(msg)
    }
    
    // Xiaomi miot 協議處理
    if strings.Contains(msg.MQTTInfo.Topic, "miot") {
        return x.handleMiotMessage(msg)
    }
    
    return nil, fmt.Errorf("unsupported xiaomi protocol")
}

func (x *XiaomiWrapper) handleMiioMessage(msg *WrapperMessage) (*RTKMessage, error) {
    model, _ := msg.ParsedPayload.GetString("model")
    properties, _ := msg.ParsedPayload.GetNested("properties")
    
    rtkPayload := map[string]interface{}{
        "health": "ok",
        "model": model,
    }
    
    // 轉換 Xiaomi 屬性到 RTK 格式
    if props, ok := properties.(map[string]interface{}); ok {
        for key, value := range props {
            rtkPayload[key] = value
        }
    }
    
    return &RTKMessage{
        Schema:   "state/1.0",
        TS:       msg.MQTTInfo.Timestamp,
        DeviceID: x.generateDeviceID(model),
        Payload:  rtkPayload,
    }, nil
}

func (x *XiaomiWrapper) handleMiotMessage(msg *WrapperMessage) (*RTKMessage, error) {
    // 處理 miot 協議特有的格式
    // ... 實現細節
    return nil, nil
}

func (x *XiaomiWrapper) generateDeviceID(model string) string {
    // 根據型號生成設備 ID
    return fmt.Sprintf("xiaomi_%s", strings.ReplaceAll(model, ".", "_"))
}
```

## Wrapper 註冊範例

```go
// 註冊所有 Wrapper 的範例
func InitializeWrappers(registry *WrapperRegistry) error {
    // 1. Home Assistant Wrapper
    haWrapper := &homeassistant.HomeAssistantWrapper{}
    haConfig := WrapperConfig{
        UplinkRules: []RouteRule{
            {
                Name:       "homeassistant",
                Priority:   10,
                TopicRegex: "homeassistant/.*/.*/(state|availability)",
                PayloadRules: []PayloadRule{
                    {Field: "entity_id", Operation: "exists"},
                },
                DeviceTypes: []string{"light", "switch", "sensor"},
            },
        },
        DownlinkRules: []RouteRule{
            {
                Name:        "homeassistant", 
                Priority:    10,
                TopicRegex:  "rtk/v1/.*/.*/.*/cmd/req",
                DeviceTypes: []string{"light", "switch", "sensor"},
            },
        },
    }
    
    if err := registry.Register("homeassistant", haWrapper, haConfig); err != nil {
        return fmt.Errorf("failed to register Home Assistant wrapper: %w", err)
    }
    
    // 2. Tasmota Wrapper
    tasmotaWrapper := &tasmota.TasmotaWrapper{}
    tasmotaConfig := WrapperConfig{
        UplinkRules: []RouteRule{
            {
                Name:       "tasmota",
                Priority:   20,
                TopicRegex: "tasmota/.*/STATE|tasmota/.*/SENSOR",
                PayloadRules: []PayloadRule{
                    {Field: "StatusSNS", Operation: "exists"},
                    {Field: "Wifi.SSId", Operation: "exists"},
                },
                DeviceTypes: []string{"relay", "sensor", "dimmer"},
            },
        },
    }
    
    if err := registry.Register("tasmota", tasmotaWrapper, tasmotaConfig); err != nil {
        return fmt.Errorf("failed to register Tasmota wrapper: %w", err)
    }
    
    // 3. Xiaomi Wrapper
    xiaomiWrapper := &xiaomi.XiaomiWrapper{}
    xiaomiConfig := WrapperConfig{
        UplinkRules: []RouteRule{
            {
                Name:       "xiaomi",
                Priority:   30,
                TopicRegex: "miio/.*/.*/(report|heartbeat)",
                PayloadRules: []PayloadRule{
                    {Field: "model", Operation: "contains", Value: "xiaomi"},
                },
                DeviceTypes: []string{"gateway", "sensor", "switch"},
            },
        },
    }
    
    if err := registry.Register("xiaomi", xiaomiWrapper, xiaomiConfig); err != nil {
        return fmt.Errorf("failed to register Xiaomi wrapper: %w", err)
    }
    
    log.Printf("Successfully registered %d wrappers", len(registry.List()))
    return nil
}
```

## 自定義 Wrapper 開發模板

```go
// wrappers/template/wrapper.go
package template

// 自定義 Wrapper 開發模板
type CustomWrapper struct {
    config CustomConfig
}

func (c *CustomWrapper) Name() string {
    return "custom_wrapper"
}

func (c *CustomWrapper) Version() string {
    return "1.0.0"
}

func (c *CustomWrapper) SupportedDeviceTypes() []string {
    return []string{"custom_device"}
}

func (c *CustomWrapper) TransformUplink(msg *WrapperMessage) (*RTKMessage, error) {
    // 步驟 1: 解析設備特有的訊息格式
    // TODO: 根據你的設備格式實現解析邏輯
    
    // 步驟 2: 提取關鍵資訊
    deviceID := c.extractDeviceID(msg)
    status := c.extractStatus(msg)
    
    // 步驟 3: 轉換為 RTK 標準格式
    return &RTKMessage{
        Schema:   "state/1.0",
        TS:       msg.MQTTInfo.Timestamp,
        DeviceID: deviceID,
        Payload:  map[string]interface{}{
            "health": "ok",
            "status": status,
            // 添加其他自定義欄位
        },
    }, nil
}

func (c *CustomWrapper) TransformDownlink(msg *WrapperMessage) (*DeviceMessage, error) {
    // 步驟 1: 解析 RTK 命令
    var rtkMsg RTKMessage
    if err := json.Unmarshal(msg.MQTTInfo.Payload, &rtkMsg); err != nil {
        return nil, fmt.Errorf("failed to parse RTK message: %w", err)
    }
    
    // 步驟 2: 轉換為設備特有格式
    // TODO: 根據你的設備格式實現命令轉換邏輯
    
    // 步驟 3: 返回設備訊息
    return &DeviceMessage{
        Topic:     fmt.Sprintf("custom/%s/command", rtkMsg.DeviceID),
        Payload:   "custom_command_payload",
        QoS:       1,
        Retained:  false,
    }, nil
}

// 輔助方法
func (c *CustomWrapper) extractDeviceID(msg *WrapperMessage) string {
    // TODO: 實現設備 ID 提取邏輯
    return "custom_device_id"
}

func (c *CustomWrapper) extractStatus(msg *WrapperMessage) string {
    // TODO: 實現狀態提取邏輯
    return "unknown"
}

func (c *CustomWrapper) Initialize(config WrapperConfig) error {
    // TODO: 實現初始化邏輯
    return nil
}

func (c *CustomWrapper) Start(ctx context.Context) error {
    // TODO: 實現啟動邏輯
    return nil
}

func (c *CustomWrapper) Stop() error {
    // TODO: 實現停止邏輯
    return nil
}

func (c *CustomWrapper) HealthCheck() error {
    // TODO: 實現健康檢查邏輯
    return nil
}
```

## 下一步

請閱讀 [BUILD_INTEGRATION.md](BUILD_INTEGRATION.md) 了解如何整合到構建系統和部署。
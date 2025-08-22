# MQTT Wrapper 架構設計

## 目錄結構

```
rtk_controller/
├── wrapper/                    # 新建中介層目錄
│   ├── cmd/                    # 主程序入口
│   │   └── main.go
│   ├── internal/               # 內部實現
│   │   ├── config/             # 配置管理
│   │   │   ├── config.go
│   │   │   └── registry.go
│   │   ├── mqtt/               # MQTT 連接管理
│   │   │   ├── client.go
│   │   │   └── forwarder.go
│   │   ├── transformer/        # 訊息轉換核心
│   │   │   ├── transformer.go
│   │   │   ├── factory.go
│   │   │   └── base.go
│   │   ├── registry/           # Wrapper 註冊機制
│   │   │   ├── registry.go
│   │   │   └── types.go
│   │   └── monitoring/         # 監控和統計
│   │       ├── metrics.go
│   │       └── logger.go
│   ├── pkg/                    # 公共類型定義
│   │   ├── types/
│   │   │   ├── wrapper.go
│   │   │   ├── device.go
│   │   │   └── message.go
│   │   └── utils/
│   │       ├── topic.go
│   │       └── validation.go
│   ├── wrappers/              # 具體 Wrapper 實現
│   │   ├── example/           # 範例 Wrapper
│   │   │   ├── wrapper.go
│   │   │   └── transformer.go
│   │   ├── homeassistant/     # Home Assistant Wrapper
│   │   │   ├── wrapper.go
│   │   │   └── transformer.go
│   │   ├── tasmota/           # Tasmota 設備 Wrapper
│   │   │   ├── wrapper.go
│   │   │   └── transformer.go
│   │   └── custom/            # 自定義 Wrapper 範例
│   │       ├── wrapper.go
│   │       └── transformer.go
│   ├── configs/               # 配置文件
│   │   ├── wrapper.yaml       # 主配置
│   │   └── wrappers/          # 各 Wrapper 配置
│   │       ├── homeassistant.yaml
│   │       ├── tasmota.yaml
│   │       └── custom.yaml
│   ├── Makefile               # 建置腳本
│   ├── go.mod
│   ├── go.sum
│   └── README.md
```

## 核心接口設計

### DeviceWrapper 接口

```go
// pkg/types/wrapper.go
package types

import "context"

// DeviceWrapper 定義設備包裝器接口
type DeviceWrapper interface {
    // 基本資訊
    Name() string
    Version() string
    SupportedDeviceTypes() []string
    
    // 訊息轉換
    Transform(inbound *InboundMessage) (*OutboundMessage, error)
    TransformCommand(rtk *RTKCommand) (*NativeCommand, error)
    
    // 生命週期管理
    Initialize(config WrapperConfig) error
    Start(ctx context.Context) error
    Stop() error
    
    // 健康檢查
    HealthCheck() error
}
```

### MessageTransformer 接口（雙向轉換）

```go
// MessageTransformer 訊息轉換器接口（雙向轉換）
type MessageTransformer interface {
    // 上行轉換：設備格式 -> RTK 格式
    TransformUplink(wrapperMsg *WrapperMessage) (*RTKMessage, error)
    
    // 下行轉換：RTK 格式 -> 設備格式
    TransformDownlink(wrapperMsg *WrapperMessage) (*DeviceMessage, error)
    
    // 統一轉換入口（自動判斷方向）
    Transform(wrapperMsg *WrapperMessage) (interface{}, error)
    
    // 驗證訊息格式
    ValidateUplink(wrapperMsg *WrapperMessage) error
    ValidateDownlink(wrapperMsg *WrapperMessage) error
    
    // 支援的訊息格式檢測
    CanHandleUplink(topic string, payload *FlexiblePayload) bool   // 能否處理設備→RTK
    CanHandleDownlink(topic string, payload *FlexiblePayload) bool // 能否處理RTK→設備
}
```

## 訊息類型定義

### 雙向訊息處理結構

```go
// MQTT 原始訊息結構（直接對應 MQTT packet）
type MQTTRawMessage struct {
    Topic     string    `json:"topic"`        // MQTT topic
    Payload   []byte    `json:"payload"`      // MQTT payload（原始 bytes）
    QoS       byte      `json:"qos"`          // MQTT QoS
    Retained  bool      `json:"retained"`     // MQTT retained flag
    Timestamp int64     `json:"timestamp"`    // Wrapper 接收時間戳
}

// 訊息方向定義
type MessageDirection int

const (
    DirectionUplink   MessageDirection = iota // Device → RTK (Non-RTK → RTK)
    DirectionDownlink                         // RTK → Device (RTK → Non-RTK)
)

// 訊息來源類型
type MessageSource int

const (
    SourceDevice MessageSource = iota  // 來自設備（非RTK格式）
    SourceRTK                          // 來自RTK Controller（RTK格式）
)

// 統一的訊息處理結構
type WrapperMessage struct {
    // 基本 MQTT 資訊
    MQTTInfo MQTTRawMessage `json:"mqtt_info"`
    
    // 訊息方向和來源
    Direction MessageDirection `json:"direction"`
    Source    MessageSource    `json:"source"`
    
    // 解析後的 JSON payload（彈性處理）
    ParsedPayload *FlexiblePayload `json:"parsed_payload"`
    
    // 目標資訊（用於路由）
    Target *MessageTarget `json:"target,omitempty"`
    
    // Wrapper 添加的元資料
    Meta map[string]interface{} `json:"meta,omitempty"`
}
```

### 彈性 JSON 處理

```go
// 彈性 JSON 處理輔助類型（處理 MQTT payload 中的 JSON 內容）
type FlexiblePayload struct {
    Raw    json.RawMessage        `json:"-"`                     // 原始 JSON bytes
    Parsed map[string]interface{} `json:"-"`                     // 解析後的通用結構
}

// 提供彈性解析方法
func (fp *FlexiblePayload) UnmarshalJSON(data []byte) error {
    fp.Raw = data
    return json.Unmarshal(data, &fp.Parsed)
}

func (fp *FlexiblePayload) GetString(key string) (string, bool) {
    if val, exists := fp.Parsed[key]; exists {
        if str, ok := val.(string); ok {
            return str, true
        }
    }
    return "", false
}

func (fp *FlexiblePayload) GetFloat64(key string) (float64, bool) {
    if val, exists := fp.Parsed[key]; exists {
        if num, ok := val.(float64); ok {
            return num, true
        }
    }
    return 0, false
}

func (fp *FlexiblePayload) GetBool(key string) (bool, bool) {
    if val, exists := fp.Parsed[key]; exists {
        if b, ok := val.(bool); ok {
            return b, true
        }
    }
    return false, false
}

func (fp *FlexiblePayload) GetNested(path ...string) (interface{}, bool) {
    current := fp.Parsed
    for _, key := range path {
        if val, exists := current[key]; exists {
            if nested, ok := val.(map[string]interface{}); ok {
                current = nested
            } else {
                return val, true // 最後一個層級
            }
        } else {
            return nil, false
        }
    }
    return current, true
}
```

### 訊息創建工廠函數

```go
// 訊息創建工廠函數
func NewWrapperMessage(mqttMsg MQTTRawMessage, direction MessageDirection, source MessageSource) (*WrapperMessage, error) {
    var payload FlexiblePayload
    
    // 嘗試解析 MQTT payload 為 JSON
    if len(mqttMsg.Payload) > 0 {
        payload.Raw = json.RawMessage(mqttMsg.Payload)
        if err := json.Unmarshal(mqttMsg.Payload, &payload.Parsed); err != nil {
            return nil, fmt.Errorf("failed to parse MQTT payload as JSON: %w", err)
        }
    }
    
    return &WrapperMessage{
        MQTTInfo:      mqttMsg,
        Direction:     direction,
        Source:        source,
        ParsedPayload: &payload,
        Meta:          make(map[string]interface{}),
    }, nil
}

// 便利函數：創建上行訊息（設備 → RTK）
func NewUplinkMessage(mqttMsg MQTTRawMessage) (*WrapperMessage, error) {
    return NewWrapperMessage(mqttMsg, DirectionUplink, SourceDevice)
}

// 便利函數：創建下行訊息（RTK → 設備）
func NewDownlinkMessage(mqttMsg MQTTRawMessage) (*WrapperMessage, error) {
    return NewWrapperMessage(mqttMsg, DirectionDownlink, SourceRTK)
}

// 檢查是否為 RTK 格式的訊息
func IsRTKMessage(topic string) bool {
    return strings.HasPrefix(topic, "rtk/v1/")
}

// 自動判斷訊息方向和來源
func NewWrapperMessageAuto(mqttMsg MQTTRawMessage) (*WrapperMessage, error) {
    if IsRTKMessage(mqttMsg.Topic) {
        // RTK 格式訊息，下行方向
        return NewWrapperMessage(mqttMsg, DirectionDownlink, SourceRTK)
    } else {
        // 非 RTK 格式訊息，上行方向
        return NewWrapperMessage(mqttMsg, DirectionUplink, SourceDevice)
    }
}
```

## 多廠牌智能註冊機制

詳細的多廠牌註冊機制請參考 [IMPLEMENTATION.md](IMPLEMENTATION.md) 中的實現範例。

## MQTT 客戶端處理邏輯

```go
// internal/mqtt/client.go - MQTT 客戶端封裝
type WrapperClient struct {
    client mqtt.Client
    transformerRegistry *transformer.Registry
}

func (wc *WrapperClient) messageHandler(client mqtt.Client, msg mqtt.Message) {
    // 步驟 1: 創建原始 MQTT 訊息
    rawMQTT := MQTTRawMessage{
        Topic:     msg.Topic(),
        Payload:   msg.Payload(),
        QoS:       msg.Qos(),
        Retained:  msg.Retained(),
        Timestamp: time.Now().UnixMilli(),
    }
    
    // 步驟 2: 自動判斷方向並創建 WrapperMessage
    wrapperMsg, err := NewWrapperMessageAuto(rawMQTT)
    if err != nil {
        log.Printf("Failed to parse MQTT message: %v", err)
        return
    }
    
    // 步驟 3: 根據方向處理訊息
    if wrapperMsg.Direction == DirectionUplink {
        wc.handleUplinkMessage(wrapperMsg)
    } else {
        wc.handleDownlinkMessage(wrapperMsg)
    }
}

// 處理上行訊息（設備 → RTK）
func (wc *WrapperClient) handleUplinkMessage(wrapperMsg *WrapperMessage) {
    // 尋找合適的 Wrapper
    wrapper := wc.transformerRegistry.FindUplinkWrapper(wrapperMsg.MQTTInfo.Topic, wrapperMsg.ParsedPayload)
    if wrapper == nil {
        log.Printf("No uplink wrapper found for topic: %s", wrapperMsg.MQTTInfo.Topic)
        return
    }
    
    // 執行上行轉換
    rtkMsg, err := wrapper.TransformUplink(wrapperMsg)
    if err != nil {
        log.Printf("Uplink transform failed: %v", err)
        return
    }
    
    // 發布到 RTK 格式
    wc.publishRTKMessage(rtkMsg)
    log.Printf("Uplink: %s → %s", wrapperMsg.MQTTInfo.Topic, wc.buildRTKTopic(rtkMsg))
}

// 處理下行訊息（RTK → 設備）
func (wc *WrapperClient) handleDownlinkMessage(wrapperMsg *WrapperMessage) {
    // 提取 RTK 訊息中的設備資訊
    deviceInfo, err := wc.extractDeviceInfo(wrapperMsg)
    if err != nil {
        log.Printf("Failed to extract device info: %v", err)
        return
    }
    
    // 尋找合適的 Wrapper
    wrapper := wc.transformerRegistry.FindDownlinkWrapper(deviceInfo.DeviceType)
    if wrapper == nil {
        log.Printf("No downlink wrapper found for device type: %s", deviceInfo.DeviceType)
        return
    }
    
    // 執行下行轉換
    deviceMsg, err := wrapper.TransformDownlink(wrapperMsg)
    if err != nil {
        log.Printf("Downlink transform failed: %v", err)
        return
    }
    
    // 發布到設備格式
    wc.publishDeviceMessage(deviceMsg)
    log.Printf("Downlink: %s → %s", wrapperMsg.MQTTInfo.Topic, deviceMsg.Topic)
}
```

## 訊息發布機制

```go
func (wc *WrapperClient) publishRTKMessage(rtkMsg *RTKMessage) error {
    // 構建 RTK topic
    rtkTopic := fmt.Sprintf("rtk/v1/%s/%s/%s/%s", 
        wc.extractTenant(rtkMsg),
        wc.extractSite(rtkMsg), 
        rtkMsg.DeviceID,
        wc.extractMessageType(rtkMsg))
    
    // 序列化為 JSON
    payload, err := json.Marshal(rtkMsg)
    if err != nil {
        return fmt.Errorf("failed to marshal RTK message: %w", err)
    }
    
    // 決定 QoS 和 Retained 設定
    qos := wc.determineQoS(rtkMsg.Schema)
    retained := wc.determineRetained(rtkMsg.Schema)
    
    // 發布到 MQTT broker
    token := wc.client.Publish(rtkTopic, qos, retained, payload)
    if !token.WaitTimeout(5 * time.Second) {
        return fmt.Errorf("publish timeout for topic: %s", rtkTopic)
    }
    
    if token.Error() != nil {
        return fmt.Errorf("failed to publish RTK message: %w", token.Error())
    }
    
    log.Printf("Published RTK message: %s", rtkTopic)
    return nil
}

// 發布設備訊息（下行轉換後）
func (wc *WrapperClient) publishDeviceMessage(deviceMsg *DeviceMessage) error {
    // 序列化 payload
    payload, err := json.Marshal(deviceMsg.Payload)
    if err != nil {
        return fmt.Errorf("failed to marshal device message: %w", err)
    }
    
    // 發布到 MQTT broker
    token := wc.client.Publish(deviceMsg.Topic, deviceMsg.QoS, deviceMsg.Retained, payload)
    if !token.WaitTimeout(5 * time.Second) {
        return fmt.Errorf("publish timeout for topic: %s", deviceMsg.Topic)
    }
    
    if token.Error() != nil {
        return fmt.Errorf("failed to publish device message: %w", token.Error())
    }
    
    log.Printf("Published device message: %s", deviceMsg.Topic)
    return nil
}
```

## 下一步

請閱讀 [IMPLEMENTATION.md](IMPLEMENTATION.md) 了解具體的實現範例和多廠牌支援。
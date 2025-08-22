package mqtt

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"rtk_wrapper/internal/config"
	"rtk_wrapper/internal/registry"
	"rtk_wrapper/pkg/types"
)

// Client 定義 MQTT wrapper 客戶端
type Client struct {
	config        *config.MQTTConfig
	client        mqtt.Client
	registry      *registry.Registry
	msgHandler    MessageHandler
	ctx           context.Context
	cancel        context.CancelFunc
	subscriptions map[string]byte
	rtkConfig     *config.RTKConfig
}

// MessageHandler 定義訊息處理介面
type MessageHandler interface {
	HandleMessage(msg *types.WrapperMessage) error
}

// NewClient 創建新的 MQTT wrapper 客戶端
func NewClient(cfg *config.MQTTConfig, rtkCfg *config.RTKConfig, reg *registry.Registry, handler MessageHandler) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config:        cfg,
		registry:      reg,
		msgHandler:    handler,
		ctx:           ctx,
		cancel:        cancel,
		subscriptions: make(map[string]byte),
		rtkConfig:     rtkCfg,
	}

	// 創建 MQTT 客戶端配置
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetClientID(cfg.ClientID)
	opts.SetKeepAlive(time.Duration(cfg.KeepAlive) * time.Second)
	opts.SetCleanSession(cfg.CleanSession)

	// 設置認證
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}

	// 設置 TLS
	if cfg.TLS.Enabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		}

		if cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		opts.SetTLSConfig(tlsConfig)
	}

	// 設置回調函數
	opts.SetDefaultPublishHandler(client.defaultMessageHandler)
	opts.SetConnectionLostHandler(client.connectionLostHandler)
	opts.SetOnConnectHandler(client.onConnectHandler)

	// 設置自動重連
	if cfg.Reconnect.Enabled {
		opts.SetAutoReconnect(true)
		opts.SetConnectRetry(true)
		opts.SetConnectRetryInterval(cfg.Reconnect.InitialDelay)
		opts.SetMaxReconnectInterval(cfg.Reconnect.MaxDelay)
	}

	// 創建 MQTT 客戶端
	client.client = mqtt.NewClient(opts)

	return client, nil
}

// Start 啟動 MQTT 客戶端
func (c *Client) Start() error {
	log.Printf("Connecting to MQTT broker: %s", c.config.Broker)

	// 連接到 MQTT broker
	token := c.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("connection timeout")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	log.Printf("Connected to MQTT broker successfully")

	// 訂閱必要的主題
	if err := c.setupSubscriptions(); err != nil {
		return fmt.Errorf("failed to setup subscriptions: %w", err)
	}

	return nil
}

// Stop 停止 MQTT 客戶端
func (c *Client) Stop() error {
	log.Printf("Stopping MQTT client...")

	c.cancel()

	// 取消訂閱
	for topic := range c.subscriptions {
		if token := c.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
			log.Printf("Failed to unsubscribe from %s: %v", topic, token.Error())
		}
	}

	// 斷開連接
	c.client.Disconnect(1000)
	log.Printf("MQTT client stopped")

	return nil
}

// setupSubscriptions 設置訂閱
func (c *Client) setupSubscriptions() error {
	// 訂閱 RTK 下行主題（RTK → 設備）
	rtkTopic := fmt.Sprintf("%s/+/+/+/cmd/req", c.rtkConfig.TopicPrefix)
	if err := c.subscribe(rtkTopic, 1); err != nil {
		return fmt.Errorf("failed to subscribe to RTK commands: %w", err)
	}

	// 訂閱各種設備上行主題
	uplinkTopics := []string{
		// Home Assistant
		"homeassistant/+/+/state",
		"homeassistant/+/+/+/state",
		"homeassistant/+/+/attributes",

		// Tasmota
		"tasmota/+/STATE",
		"tasmota/+/SENSOR",
		"tasmota/+/LWT",

		// 自定義廠牌
		"custom/+/+/data",
		"vendor/+/+/status",

		// 通用模式
		"+/+/+/state",
		"+/+/status",
	}

	for _, topic := range uplinkTopics {
		if err := c.subscribe(topic, 0); err != nil {
			log.Printf("Failed to subscribe to %s: %v", topic, err)
			// 不要因為單個訂閱失敗而終止整個啟動過程
		}
	}

	return nil
}

// subscribe 訂閱主題
func (c *Client) subscribe(topic string, qos byte) error {
	log.Printf("Subscribing to topic: %s (QoS: %d)", topic, qos)

	token := c.client.Subscribe(topic, qos, c.handleMQTTMessage)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("subscription timeout for topic: %s", topic)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", topic, err)
	}

	c.subscriptions[topic] = qos
	return nil
}

// handleMQTTMessage 統一訊息處理器
func (c *Client) handleMQTTMessage(client mqtt.Client, msg mqtt.Message) {
	// 創建原始 MQTT 訊息
	rawMQTT := types.MQTTRawMessage{
		Topic:     msg.Topic(),
		Payload:   msg.Payload(),
		QoS:       msg.Qos(),
		Retained:  msg.Retained(),
		Timestamp: time.Now().UnixMilli(),
	}

	// 自動判斷方向並創建 WrapperMessage
	wrapperMsg, err := types.NewWrapperMessageAuto(rawMQTT)
	if err != nil {
		log.Printf("Failed to parse MQTT message: %v", err)
		return
	}

	// 根據方向處理訊息
	if wrapperMsg.Direction == types.DirectionUplink {
		c.handleUplinkMessage(wrapperMsg)
	} else {
		c.handleDownlinkMessage(wrapperMsg)
	}
}

// handleUplinkMessage 處理上行訊息（設備 → RTK）
func (c *Client) handleUplinkMessage(wrapperMsg *types.WrapperMessage) {
	// 尋找合適的 Wrapper
	transformer := c.registry.FindUplinkWrapper(wrapperMsg.MQTTInfo.Topic, wrapperMsg.ParsedPayload)
	if transformer == nil {
		log.Printf("No uplink wrapper found for topic: %s", wrapperMsg.MQTTInfo.Topic)
		return
	}

	// 執行上行轉換
	rtkMsg, err := transformer.TransformUplink(wrapperMsg)
	if err != nil {
		log.Printf("Uplink transform failed: %v", err)
		return
	}

	// 發布到 RTK 格式
	if err := c.publishRTKMessage(rtkMsg); err != nil {
		log.Printf("Failed to publish RTK message: %v", err)
		return
	}

	log.Printf("Uplink: %s → %s", wrapperMsg.MQTTInfo.Topic, c.buildRTKTopic(rtkMsg))
}

// handleDownlinkMessage 處理下行訊息（RTK → 設備）
func (c *Client) handleDownlinkMessage(wrapperMsg *types.WrapperMessage) {
	// 提取 RTK 訊息中的設備資訊
	deviceInfo, err := c.extractDeviceInfo(wrapperMsg)
	if err != nil {
		log.Printf("Failed to extract device info: %v", err)
		return
	}

	// 尋找合適的 Wrapper
	transformer := c.registry.FindDownlinkWrapper(deviceInfo.DeviceType)
	if transformer == nil {
		log.Printf("No downlink wrapper found for device type: %s", deviceInfo.DeviceType)
		return
	}

	// 執行下行轉換
	deviceMsg, err := transformer.TransformDownlink(wrapperMsg)
	if err != nil {
		log.Printf("Downlink transform failed: %v", err)
		return
	}

	// 發布到設備格式
	if err := c.publishDeviceMessage(deviceMsg); err != nil {
		log.Printf("Failed to publish device message: %v", err)
		return
	}

	log.Printf("Downlink: %s → %s", wrapperMsg.MQTTInfo.Topic, deviceMsg.Topic)
}

// publishRTKMessage 發布 RTK 格式訊息
func (c *Client) publishRTKMessage(rtkMsg *types.RTKMessage) error {
	// 構建 RTK topic
	rtkTopic := c.buildRTKTopic(rtkMsg)

	// 序列化為 JSON
	payload, err := json.Marshal(rtkMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal RTK message: %w", err)
	}

	// 決定 QoS 和 Retained 設定
	qos := c.determineQoS(rtkMsg.Schema)
	retained := c.determineRetained(rtkMsg.Schema)

	// 發布到 MQTT broker
	token := c.client.Publish(rtkTopic, qos, retained, payload)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish timeout for topic: %s", rtkTopic)
	}

	if token.Error() != nil {
		return fmt.Errorf("failed to publish RTK message: %w", token.Error())
	}

	return nil
}

// publishDeviceMessage 發布設備訊息
func (c *Client) publishDeviceMessage(deviceMsg *types.DeviceMessage) error {
	// 序列化 payload
	payload, err := json.Marshal(deviceMsg.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal device message: %w", err)
	}

	// 發布到 MQTT broker
	token := c.client.Publish(deviceMsg.Topic, deviceMsg.QoS, deviceMsg.Retained, payload)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish timeout for topic: %s", deviceMsg.Topic)
	}

	if token.Error() != nil {
		return fmt.Errorf("failed to publish device message: %w", token.Error())
	}

	return nil
}

// buildRTKTopic 構建 RTK 主題
func (c *Client) buildRTKTopic(rtkMsg *types.RTKMessage) string {
	// 從 schema 提取 message type
	parts := strings.Split(rtkMsg.Schema, "/")
	messageType := parts[0]

	return fmt.Sprintf("%s/%s/%s/%s/%s",
		c.rtkConfig.TopicPrefix,
		c.rtkConfig.DefaultTenant,
		c.rtkConfig.DefaultSite,
		rtkMsg.DeviceID,
		messageType)
}

// extractDeviceInfo 從 RTK 訊息中提取設備資訊
func (c *Client) extractDeviceInfo(wrapperMsg *types.WrapperMessage) (*DeviceInfo, error) {
	// 解析 RTK topic: rtk/v1/{tenant}/{site}/{device_id}/{message_type}
	topic := wrapperMsg.MQTTInfo.Topic
	parts := strings.Split(topic, "/")
	if len(parts) < 6 {
		return nil, fmt.Errorf("invalid RTK topic format: %s", topic)
	}

	deviceID := parts[4]

	// 從 payload 或 device_id 推斷設備類型
	deviceType := c.inferDeviceType(deviceID, wrapperMsg.ParsedPayload)

	return &DeviceInfo{
		DeviceID:   deviceID,
		DeviceType: deviceType,
		Tenant:     parts[2],
		Site:       parts[3],
	}, nil
}

// DeviceInfo 設備資訊
type DeviceInfo struct {
	DeviceID   string
	DeviceType string
	Tenant     string
	Site       string
}

// inferDeviceType 推斷設備類型
func (c *Client) inferDeviceType(deviceID string, payload *types.FlexiblePayload) string {
	// 從設備 ID 推斷（基於命名模式）
	deviceID = strings.ToLower(deviceID)

	if strings.Contains(deviceID, "light") {
		return "light"
	} else if strings.Contains(deviceID, "switch") {
		return "switch"
	} else if strings.Contains(deviceID, "sensor") {
		return "sensor"
	} else if strings.Contains(deviceID, "climate") {
		return "climate"
	} else if strings.Contains(deviceID, "cover") {
		return "cover"
	}

	// 從 payload 推斷（如果有可用資訊）
	if payload != nil {
		if _, hasBrightness := payload.GetFloat64("brightness"); hasBrightness {
			return "light"
		}
		if _, hasTemperature := payload.GetFloat64("temperature"); hasTemperature {
			return "climate"
		}
	}

	// 預設為通用設備
	return "device"
}

// determineQoS 根據 schema 決定 QoS
func (c *Client) determineQoS(schema string) byte {
	if strings.HasPrefix(schema, "state") {
		return c.rtkConfig.QoS.StateMessages
	} else if strings.HasPrefix(schema, "telemetry") {
		return c.rtkConfig.QoS.TelemetryMessages
	} else if strings.HasPrefix(schema, "evt") {
		return c.rtkConfig.QoS.EventMessages
	} else if strings.HasPrefix(schema, "cmd") {
		return c.rtkConfig.QoS.CommandMessages
	}
	return 0 // 預設 QoS 0
}

// determineRetained 根據 schema 決定 retained
func (c *Client) determineRetained(schema string) bool {
	if strings.HasPrefix(schema, "state") {
		return c.rtkConfig.Retained.StateMessages
	} else if strings.HasPrefix(schema, "attr") {
		return c.rtkConfig.Retained.AttrMessages
	} else if strings.HasPrefix(schema, "lwt") {
		return c.rtkConfig.Retained.LWTMessages
	}
	return c.rtkConfig.Retained.Others
}

// defaultMessageHandler 預設訊息處理器
func (c *Client) defaultMessageHandler(client mqtt.Client, msg mqtt.Message) {
	c.handleMQTTMessage(client, msg)
}

// connectionLostHandler 連接丟失處理器
func (c *Client) connectionLostHandler(client mqtt.Client, err error) {
	log.Printf("MQTT connection lost: %v", err)
}

// onConnectHandler 連接成功處理器
func (c *Client) onConnectHandler(client mqtt.Client) {
	log.Printf("MQTT connection established")
}

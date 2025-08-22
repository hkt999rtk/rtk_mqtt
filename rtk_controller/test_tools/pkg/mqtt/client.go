package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"rtk_test_tools/pkg/types"
)

// Client MQTT 客戶端封裝
type Client struct {
	client   mqtt.Client
	deviceID string
	tenant   string
	site     string
	verbose  bool
}

// NewClient 創建新的 MQTT 客戶端
func NewClient(config types.MQTTConfig, deviceID, tenant, site string, verbose bool) (*Client, error) {
	opts := mqtt.NewClientOptions()
	brokerURL := fmt.Sprintf("tcp://%s:%d", config.Broker, config.Port)
	opts.AddBroker(brokerURL)
	
	clientID := fmt.Sprintf("%s-%s", config.ClientIDPrefix, deviceID)
	opts.SetClientID(clientID)
	
	if config.Username != "" {
		opts.SetUsername(config.Username)
	}
	if config.Password != "" {
		opts.SetPassword(config.Password)
	}
	
	opts.SetKeepAlive(60 * time.Second)
	opts.SetDefaultPublishHandler(defaultMessageHandler)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(1 * time.Minute)
	
	// 設置 LWT (Last Will and Testament)
	lwtTopic := fmt.Sprintf("rtk/v1/%s/%s/%s/lwt", tenant, site, deviceID)
	lwtPayload := map[string]interface{}{
		"status": "offline",
		"ts":     time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"reason": "connection_lost",
	}
	lwtData, _ := json.Marshal(lwtPayload)
	opts.SetWill(lwtTopic, string(lwtData), 1, true)
	
	client := mqtt.NewClient(opts)
	
	return &Client{
		client:   client,
		deviceID: deviceID,
		tenant:   tenant,
		site:     site,
		verbose:  verbose,
	}, nil
}

// Connect 連接到 MQTT Broker
func (c *Client) Connect() error {
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %v", token.Error())
	}
	
	if c.verbose {
		log.Printf("[%s] Connected to MQTT broker", c.deviceID)
	}
	
	// 發送上線通知
	return c.publishOnlineStatus()
}

// Disconnect 斷開連接
func (c *Client) Disconnect() {
	// 發送離線通知
	c.publishOfflineStatus()
	
	c.client.Disconnect(250)
	if c.verbose {
		log.Printf("[%s] Disconnected from MQTT broker", c.deviceID)
	}
}

// PublishState 發布狀態消息
func (c *Client) PublishState(payload types.StatePayload) error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/state", c.tenant, c.site, c.deviceID)
	
	message := types.BaseMessage{
		Schema:    "state/1.0",
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		Payload:   payload,
	}
	
	return c.publish(topic, message, true) // retained = true
}

// PublishTelemetry 發布遙測消息
func (c *Client) PublishTelemetry(metric string, payload types.TelemetryPayload) error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/telemetry/%s", c.tenant, c.site, c.deviceID, metric)
	
	message := types.BaseMessage{
		Schema:    fmt.Sprintf("telemetry.%s/1.0", metric),
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		Payload:   payload,
	}
	
	return c.publish(topic, message, false)
}

// PublishEvent 發布事件消息
func (c *Client) PublishEvent(eventType string, payload types.EventPayload) error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/evt/%s", c.tenant, c.site, c.deviceID, eventType)
	
	message := types.BaseMessage{
		Schema:    fmt.Sprintf("evt.%s/1.0", eventType),
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		Payload:   payload,
	}
	
	return c.publish(topic, message, false)
}

// 內部方法：發布消息
func (c *Client) publish(topic string, message types.BaseMessage, retained bool) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}
	
	qos := byte(1)
	token := c.client.Publish(topic, qos, retained, data)
	token.Wait()
	
	if token.Error() != nil {
		return fmt.Errorf("failed to publish to %s: %v", topic, token.Error())
	}
	
	if c.verbose {
		log.Printf("[%s] Published to %s: %s", c.deviceID, topic, string(data))
	}
	
	return nil
}

// 發布上線狀態
func (c *Client) publishOnlineStatus() error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/lwt", c.tenant, c.site, c.deviceID)
	
	payload := map[string]interface{}{
		"status": "online",
		"ts":     time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	
	data, _ := json.Marshal(payload)
	token := c.client.Publish(topic, 1, true, data)
	token.Wait()
	
	return token.Error()
}

// 發布離線狀態  
func (c *Client) publishOfflineStatus() error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/lwt", c.tenant, c.site, c.deviceID)
	
	payload := map[string]interface{}{
		"status": "offline",
		"ts":     time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"reason": "normal_shutdown",
	}
	
	data, _ := json.Marshal(payload)
	token := c.client.Publish(topic, 1, true, data)
	token.Wait()
	
	return token.Error()
}

// 默認消息處理器
func defaultMessageHandler(client mqtt.Client, msg mqtt.Message) {
	// 通常設備不需要處理收到的消息，但可以在這裡記錄
}
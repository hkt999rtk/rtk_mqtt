package base

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

// MQTTClient MQTT 客戶端包裝器
type MQTTClient struct {
	client    mqtt.Client
	device    Device
	config    MQTTConfig
	logger    *logrus.Entry
	connected bool
}

// MQTTConfig MQTT 配置
type MQTTConfig struct {
	Broker       string        `yaml:"broker"`
	Port         int           `yaml:"port"`
	Username     string        `yaml:"username"`
	Password     string        `yaml:"password"`
	ClientID     string        `yaml:"client_id"`
	KeepAlive    time.Duration `yaml:"keep_alive"`
	CleanSession bool          `yaml:"clean_session"`
	QoS          byte          `yaml:"qos"`
	Retained     bool          `yaml:"retained"`
}

// RTKMessage RTK MQTT 標準訊息格式
type RTKMessage struct {
	Schema    string      `json:"schema"`
	Timestamp string      `json:"ts"`
	DeviceID  string      `json:"device_id"`
	Payload   interface{} `json:"payload"`
	Trace     *TraceInfo  `json:"trace,omitempty"`
}

// TraceInfo LLM 診斷追蹤資訊
type TraceInfo struct {
	ReqID         string `json:"req_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
}

// NewMQTTClient 建立新的 MQTT 客戶端
func NewMQTTClient(config MQTTConfig, device Device) *MQTTClient {
	logger := logrus.WithFields(logrus.Fields{
		"device_id": device.GetDeviceID(),
		"broker":    config.Broker,
	})

	// 設定 MQTT 客戶端選項
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.Broker, config.Port))
	opts.SetClientID(config.ClientID)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetKeepAlive(config.KeepAlive)
	opts.SetCleanSession(config.CleanSession)

	// 設定 LWT (Last Will Testament)
	lwtTopic := fmt.Sprintf("rtk/v1/%s/%s/%s/lwt",
		device.GetTenant(), device.GetSite(), device.GetDeviceID())

	lwtPayload := createLWTMessage(device)
	lwtBytes, _ := json.Marshal(lwtPayload)

	opts.SetWill(lwtTopic, string(lwtBytes), 1, true)

	// 設定連線回調
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logger.WithError(err).Warn("MQTT connection lost")
	})

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Info("MQTT connected")
	})

	client := mqtt.NewClient(opts)

	return &MQTTClient{
		client: client,
		device: device,
		config: config,
		logger: logger,
	}
}

// Connect 連接到 MQTT Broker
func (m *MQTTClient) Connect() error {
	m.logger.Info("Connecting to MQTT broker")

	token := m.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %v", token.Error())
	}

	m.connected = true

	// 發送上線訊息
	if err := m.publishOnlineStatus(); err != nil {
		m.logger.WithError(err).Warn("Failed to publish online status")
	}

	// 發送設備屬性
	if err := m.publishDeviceAttributes(); err != nil {
		m.logger.WithError(err).Warn("Failed to publish device attributes")
	}

	// 訂閱命令 topic
	if err := m.subscribeToCommands(); err != nil {
		m.logger.WithError(err).Warn("Failed to subscribe to commands")
	}

	return nil
}

// Disconnect 斷開 MQTT 連接
func (m *MQTTClient) Disconnect() {
	if !m.connected {
		return
	}

	m.logger.Info("Disconnecting from MQTT broker")

	// 發送下線訊息
	m.publishOfflineStatus()

	// 斷開連接
	m.client.Disconnect(250)
	m.connected = false
}

// IsConnected 檢查連接狀態
func (m *MQTTClient) IsConnected() bool {
	return m.connected && m.client.IsConnected()
}

// PublishState 發布設備狀態
func (m *MQTTClient) PublishState(payload StatePayload) error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/state",
		m.device.GetTenant(), m.device.GetSite(), m.device.GetDeviceID())

	message := RTKMessage{
		Schema:    "state/1.0",
		Timestamp: m.getCurrentTimestamp(),
		DeviceID:  m.device.GetDeviceID(),
		Payload:   payload,
	}

	return m.publishMessage(topic, message, true)
}

// PublishTelemetry 發布遙測資料
func (m *MQTTClient) PublishTelemetry(metric string, payload TelemetryPayload) error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/telemetry/%s",
		m.device.GetTenant(), m.device.GetSite(), m.device.GetDeviceID(), metric)

	message := RTKMessage{
		Schema:    fmt.Sprintf("telemetry.%s/1.0", metric),
		Timestamp: m.getCurrentTimestamp(),
		DeviceID:  m.device.GetDeviceID(),
		Payload:   payload,
	}

	return m.publishMessage(topic, message, false)
}

// PublishEvent 發布事件
func (m *MQTTClient) PublishEvent(eventType string, event Event) error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/evt/%s",
		m.device.GetTenant(), m.device.GetSite(), m.device.GetDeviceID(), eventType)

	message := RTKMessage{
		Schema:    fmt.Sprintf("evt.%s/1.0", eventType),
		Timestamp: m.getCurrentTimestamp(),
		DeviceID:  m.device.GetDeviceID(),
		Payload:   event,
	}

	return m.publishMessage(topic, message, false)
}

// PublishCommandAck 發布命令確認
func (m *MQTTClient) PublishCommandAck(commandID string) error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/cmd/ack",
		m.device.GetTenant(), m.device.GetSite(), m.device.GetDeviceID())

	ackPayload := map[string]interface{}{
		"command_id": commandID,
		"status":     "received",
		"timestamp":  m.getCurrentTimestamp(),
	}

	message := RTKMessage{
		Schema:    "cmd.ack/1.0",
		Timestamp: m.getCurrentTimestamp(),
		DeviceID:  m.device.GetDeviceID(),
		Payload:   ackPayload,
	}

	return m.publishMessage(topic, message, false)
}

// PublishCommandResult 發布命令執行結果
func (m *MQTTClient) PublishCommandResult(result CommandResult) error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/cmd/res",
		m.device.GetTenant(), m.device.GetSite(), m.device.GetDeviceID())

	message := RTKMessage{
		Schema:    "cmd.result/1.0",
		Timestamp: m.getCurrentTimestamp(),
		DeviceID:  m.device.GetDeviceID(),
		Payload:   result,
	}

	return m.publishMessage(topic, message, false)
}

// 內部方法
func (m *MQTTClient) publishMessage(topic string, message RTKMessage, retained bool) error {
	if !m.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	qos := m.config.QoS
	if retained {
		qos = 1 // 狀態訊息使用 QoS 1
	}

	token := m.client.Publish(topic, qos, retained, data)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish message: %v", token.Error())
	}

	m.logger.WithFields(logrus.Fields{
		"topic":    topic,
		"retained": retained,
		"qos":      qos,
	}).Debug("Message published")

	return nil
}

func (m *MQTTClient) publishOnlineStatus() error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/lwt",
		m.device.GetTenant(), m.device.GetSite(), m.device.GetDeviceID())

	lwtPayload := map[string]interface{}{
		"status":    "online",
		"timestamp": m.getCurrentTimestamp(),
		"uptime":    m.device.GetUptime().Seconds(),
	}

	message := RTKMessage{
		Schema:    "lwt/1.0",
		Timestamp: m.getCurrentTimestamp(),
		DeviceID:  m.device.GetDeviceID(),
		Payload:   lwtPayload,
	}

	return m.publishMessage(topic, message, true)
}

func (m *MQTTClient) publishOfflineStatus() error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/lwt",
		m.device.GetTenant(), m.device.GetSite(), m.device.GetDeviceID())

	lwtPayload := map[string]interface{}{
		"status":    "offline",
		"reason":    "normal_disconnect",
		"timestamp": m.getCurrentTimestamp(),
		"uptime":    m.device.GetUptime().Seconds(),
	}

	message := RTKMessage{
		Schema:    "lwt/1.0",
		Timestamp: m.getCurrentTimestamp(),
		DeviceID:  m.device.GetDeviceID(),
		Payload:   lwtPayload,
	}

	return m.publishMessage(topic, message, true)
}

func (m *MQTTClient) publishDeviceAttributes() error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/attr",
		m.device.GetTenant(), m.device.GetSite(), m.device.GetDeviceID())

	attrPayload := map[string]interface{}{
		"device_type":  m.device.GetDeviceType(),
		"mac_address":  m.device.GetMACAddress(),
		"ip_address":   m.device.GetIPAddress(),
		"network_info": m.device.GetNetworkInfo(),
	}

	message := RTKMessage{
		Schema:    "attr/1.0",
		Timestamp: m.getCurrentTimestamp(),
		DeviceID:  m.device.GetDeviceID(),
		Payload:   attrPayload,
	}

	return m.publishMessage(topic, message, true)
}

func (m *MQTTClient) subscribeToCommands() error {
	topic := fmt.Sprintf("rtk/v1/%s/%s/%s/cmd/req",
		m.device.GetTenant(), m.device.GetSite(), m.device.GetDeviceID())

	token := m.client.Subscribe(topic, 1, m.handleCommandMessage)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to commands: %v", token.Error())
	}

	m.logger.WithField("topic", topic).Info("Subscribed to commands")
	return nil
}

func (m *MQTTClient) handleCommandMessage(client mqtt.Client, msg mqtt.Message) {
	m.logger.WithField("topic", msg.Topic()).Debug("Received command message")

	var message RTKMessage
	if err := json.Unmarshal(msg.Payload(), &message); err != nil {
		m.logger.WithError(err).Error("Failed to unmarshal command message")
		return
	}

	// 解析命令
	var command Command
	if payloadBytes, err := json.Marshal(message.Payload); err == nil {
		if err := json.Unmarshal(payloadBytes, &command); err != nil {
			m.logger.WithError(err).Error("Failed to parse command payload")
			return
		}
	}

	// 發送命令確認
	if err := m.PublishCommandAck(command.ID); err != nil {
		m.logger.WithError(err).Error("Failed to publish command ack")
	}

	// 執行命令
	go m.executeCommand(command)
}

func (m *MQTTClient) executeCommand(command Command) {
	m.logger.WithField("command_type", command.Type).Info("Executing command")

	err := m.device.HandleCommand(command)

	result := CommandResult{
		CommandID: command.ID,
		Success:   err == nil,
	}

	if err != nil {
		result.Error = err.Error()
		result.Message = "Command execution failed"
	} else {
		result.Message = "Command executed successfully"
	}

	// 發送命令結果
	if err := m.PublishCommandResult(result); err != nil {
		m.logger.WithError(err).Error("Failed to publish command result")
	}
}

func (m *MQTTClient) getCurrentTimestamp() string {
	return fmt.Sprintf("%d", time.Now().UnixMilli())
}

func createLWTMessage(device Device) RTKMessage {
	lwtPayload := map[string]interface{}{
		"status":    "offline",
		"reason":    "unexpected_disconnect",
		"last_seen": time.Now().UnixMilli(),
	}

	return RTKMessage{
		Schema:    "lwt/1.0",
		Timestamp: fmt.Sprintf("%d", time.Now().UnixMilli()),
		DeviceID:  device.GetDeviceID(),
		Payload:   lwtPayload,
	}
}

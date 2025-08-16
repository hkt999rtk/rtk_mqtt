package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

// PahoClient wraps the Eclipse Paho MQTT client
type PahoClient struct {
	client   pahomqtt.Client
	config   *Config
	stats    *Statistics
	statsMux sync.RWMutex

	connectionHandler     ConnectionHandler
	defaultMessageHandler MessageHandler

	logger *logrus.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

// PahoFactory creates Paho MQTT clients
type PahoFactory struct{}

// CreateClient creates a new Paho MQTT client
func (f *PahoFactory) CreateClient(cfg *Config) (Client, error) {
	SetDefaults(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	
	client := &PahoClient{
		config: cfg,
		stats: &Statistics{
			LastErrorTimestamp: time.Now(),
		},
		logger: logrus.New(),
		ctx:    ctx,
		cancel: cancel,
	}

	// Create Paho client options
	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.BrokerHost, cfg.BrokerPort))
	opts.SetClientID(cfg.ClientID)
	opts.SetCleanSession(cfg.CleanSession)
	opts.SetKeepAlive(cfg.KeepAlive)
	opts.SetConnectTimeout(cfg.ConnectTimeout)

	// Set credentials if provided
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		if cfg.Password != "" {
			opts.SetPassword(cfg.Password)
		}
	}

	// Configure TLS if enabled
	if cfg.TLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
		}
		opts.SetTLSConfig(tlsConfig)
	}

	// Set Last Will Testament if configured
	if cfg.LWT != nil {
		opts.SetWill(cfg.LWT.Topic, cfg.LWT.Message, byte(cfg.LWT.QoS), cfg.LWT.Retained)
	}

	// Set connection lost handler
	opts.SetConnectionLostHandler(func(c pahomqtt.Client, err error) {
		client.handleConnectionLost(err)
	})

	// Set on connect handler
	opts.SetOnConnectHandler(func(c pahomqtt.Client) {
		client.handleConnected()
	})

	// Set default message handler
	opts.SetDefaultPublishHandler(func(c pahomqtt.Client, msg pahomqtt.Message) {
		client.handleMessage(msg)
	})

	// Create the Paho client
	client.client = pahomqtt.NewClient(opts)

	return client, nil
}

// GetName returns the factory name
func (f *PahoFactory) GetName() string {
	return "paho"
}

// GetVersion returns the factory version
func (f *PahoFactory) GetVersion() string {
	return "1.0.0"
}

// Connect connects to the MQTT broker
func (p *PahoClient) Connect(ctx context.Context) error {
	if p.client.IsConnected() {
		return ErrAlreadyConnected
	}

	p.logger.WithFields(logrus.Fields{
		"broker":    fmt.Sprintf("%s:%d", p.config.BrokerHost, p.config.BrokerPort),
		"client_id": p.config.ClientID,
	}).Info("Connecting to MQTT broker")

	token := p.client.Connect()
	
	// Wait for connection with timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.config.ConnectTimeout):
		return ErrConnectionFailed.WithCause(fmt.Errorf("connection timeout"))
	default:
		if token.Wait() && token.Error() != nil {
			return ErrConnectionFailed.WithCause(token.Error())
		}
	}

	p.stats.ConnectionUptime = time.Now().Sub(time.Now())
	return nil
}

// Disconnect disconnects from the MQTT broker
func (p *PahoClient) Disconnect() error {
	if !p.client.IsConnected() {
		return nil
	}

	p.logger.Info("Disconnecting from MQTT broker")
	p.client.Disconnect(250) // 250ms grace period
	
	p.statsMux.Lock()
	p.stats.ConnectionUptime = 0
	p.statsMux.Unlock()

	return nil
}

// IsConnected returns the connection status
func (p *PahoClient) IsConnected() bool {
	return p.client.IsConnected()
}

// GetStatus returns the current connection status
func (p *PahoClient) GetStatus() ConnectionStatus {
	if p.client.IsConnected() {
		return StatusConnected
	}
	return StatusDisconnected
}

// Publish publishes a message to the specified topic
func (p *PahoClient) Publish(ctx context.Context, topic string, payload []byte, opts *PublishOptions) error {
	if !p.client.IsConnected() {
		return ErrNotConnected
	}

	if opts == nil {
		opts = &PublishOptions{QoS: QoSAtMostOnce}
	}

	p.logger.WithFields(logrus.Fields{
		"topic":    topic,
		"qos":      opts.QoS,
		"retained": opts.Retained,
		"size":     len(payload),
	}).Debug("Publishing message")

	token := p.client.Publish(topic, byte(opts.QoS), opts.Retained, payload)
	
	if token.Wait() && token.Error() != nil {
		return ErrPublishFailed.WithCause(token.Error())
	}

	p.statsMux.Lock()
	p.stats.MessagesPublished++
	p.stats.BytesPublished += uint64(len(payload))
	p.statsMux.Unlock()

	return nil
}

// PublishMessage publishes a Message struct
func (p *PahoClient) PublishMessage(ctx context.Context, msg *Message) error {
	opts := &PublishOptions{
		QoS:      msg.QoS,
		Retained: msg.Retained,
		Headers:  msg.Headers,
	}
	return p.Publish(ctx, msg.Topic, msg.Payload, opts)
}

// Subscribe subscribes to a topic with a message handler
func (p *PahoClient) Subscribe(ctx context.Context, topic string, handler MessageHandler, opts *SubscribeOptions) error {
	if !p.client.IsConnected() {
		return ErrNotConnected
	}

	if opts == nil {
		opts = &SubscribeOptions{QoS: QoSAtMostOnce}
	}

	p.logger.WithFields(logrus.Fields{
		"topic": topic,
		"qos":   opts.QoS,
	}).Info("Subscribing to topic")

	token := p.client.Subscribe(topic, byte(opts.QoS), func(c pahomqtt.Client, msg pahomqtt.Message) {
		if handler != nil {
			rtkMsg := &Message{
				Topic:     msg.Topic(),
				Payload:   msg.Payload(),
				QoS:       QoS(msg.Qos()),
				Retained:  msg.Retained(),
				MessageID: msg.MessageID(),
				Timestamp: time.Now(),
			}
			
			if err := handler(p.ctx, rtkMsg); err != nil {
				p.logger.WithError(err).WithField("topic", topic).Error("Message handler error")
			}
		}
	})

	if token.Wait() && token.Error() != nil {
		return ErrSubscribeFailed.WithCause(token.Error())
	}

	return nil
}

// Unsubscribe unsubscribes from a topic
func (p *PahoClient) Unsubscribe(ctx context.Context, topic string) error {
	if !p.client.IsConnected() {
		return ErrNotConnected
	}

	p.logger.WithField("topic", topic).Info("Unsubscribing from topic")

	token := p.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

// SetConnectionHandler sets the connection state change handler
func (p *PahoClient) SetConnectionHandler(handler ConnectionHandler) {
	p.connectionHandler = handler
}

// SetDefaultMessageHandler sets the default message handler
func (p *PahoClient) SetDefaultMessageHandler(handler MessageHandler) {
	p.defaultMessageHandler = handler
}

// GetStatistics returns client statistics
func (p *PahoClient) GetStatistics() *Statistics {
	p.statsMux.RLock()
	defer p.statsMux.RUnlock()
	
	stats := *p.stats
	if p.client.IsConnected() {
		stats.ConnectionUptime = time.Since(time.Now().Add(-p.stats.ConnectionUptime))
	}
	
	return &stats
}

// GetLastError returns the last error
func (p *PahoClient) GetLastError() error {
	p.statsMux.RLock()
	defer p.statsMux.RUnlock()
	
	if p.stats.LastError != "" {
		return fmt.Errorf(p.stats.LastError)
	}
	return nil
}

// GetConfig returns the client configuration
func (p *PahoClient) GetConfig() *Config {
	return p.config
}

// UpdateConfig updates the client configuration
func (p *PahoClient) UpdateConfig(cfg *Config) error {
	// Note: Most configuration changes require a reconnection
	p.config = cfg
	return nil
}

// Start starts the client
func (p *PahoClient) Start(ctx context.Context) error {
	return p.Connect(ctx)
}

// Stop stops the client
func (p *PahoClient) Stop() error {
	p.cancel()
	return p.Disconnect()
}

// Private methods

func (p *PahoClient) handleConnectionLost(err error) {
	p.logger.WithError(err).Warn("MQTT connection lost")
	
	p.statsMux.Lock()
	p.stats.ReconnectCount++
	p.stats.LastError = err.Error()
	p.stats.LastErrorTimestamp = time.Now()
	p.statsMux.Unlock()

	if p.connectionHandler != nil {
		p.connectionHandler(false, err)
	}
}

func (p *PahoClient) handleConnected() {
	p.logger.Info("MQTT connection established")
	
	p.statsMux.Lock()
	p.stats.ConnectionUptime = time.Now().Sub(time.Now())
	p.statsMux.Unlock()

	if p.connectionHandler != nil {
		p.connectionHandler(true, nil)
	}
}

func (p *PahoClient) handleMessage(msg pahomqtt.Message) {
	p.statsMux.Lock()
	p.stats.MessagesReceived++
	p.stats.BytesReceived += uint64(len(msg.Payload()))
	p.statsMux.Unlock()

	if p.defaultMessageHandler != nil {
		rtkMsg := &Message{
			Topic:     msg.Topic(),
			Payload:   msg.Payload(),
			QoS:       QoS(msg.Qos()),
			Retained:  msg.Retained(),
			MessageID: msg.MessageID(),
			Timestamp: time.Now(),
		}
		
		if err := p.defaultMessageHandler(p.ctx, rtkMsg); err != nil {
			p.logger.WithError(err).Error("Default message handler error")
		}
	}
}

// Register the Paho factory on package initialization
func init() {
	RegisterFactory("paho", &PahoFactory{})
}
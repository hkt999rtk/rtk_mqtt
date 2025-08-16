package mqtt

import (
	"context"
	"time"
)

// QoS represents MQTT Quality of Service levels
type QoS byte

const (
	QoSAtMostOnce  QoS = 0 // At most once delivery
	QoSAtLeastOnce QoS = 1 // At least once delivery
	QoSExactlyOnce QoS = 2 // Exactly once delivery
)

// String returns the string representation of QoS
func (q QoS) String() string {
	switch q {
	case QoSAtMostOnce:
		return "AtMostOnce"
	case QoSAtLeastOnce:
		return "AtLeastOnce"
	case QoSExactlyOnce:
		return "ExactlyOnce"
	default:
		return "Unknown"
	}
}

// Message represents an MQTT message
type Message struct {
	Topic     string            `json:"topic"`
	Payload   []byte            `json:"payload"`
	QoS       QoS               `json:"qos"`
	Retained  bool              `json:"retained"`
	MessageID uint16            `json:"message_id,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Config represents MQTT client configuration
type Config struct {
	// Connection settings
	BrokerHost string `json:"broker_host" yaml:"broker_host" validate:"required"`
	BrokerPort int    `json:"broker_port" yaml:"broker_port" validate:"min=1,max=65535"`
	ClientID   string `json:"client_id" yaml:"client_id" validate:"required"`
	Username   string `json:"username" yaml:"username"`
	Password   string `json:"password" yaml:"password"`

	// Connection behavior
	KeepAlive      time.Duration `json:"keep_alive" yaml:"keep_alive"`
	CleanSession   bool          `json:"clean_session" yaml:"clean_session"`
	ConnectTimeout time.Duration `json:"connect_timeout" yaml:"connect_timeout"`
	RetryInterval  time.Duration `json:"retry_interval" yaml:"retry_interval"`
	MaxRetryCount  int           `json:"max_retry_count" yaml:"max_retry_count"`

	// TLS settings
	TLS    bool   `json:"tls" yaml:"tls"`
	CACert string `json:"ca_cert" yaml:"ca_cert"`
	Cert   string `json:"cert" yaml:"cert"`
	Key    string `json:"key" yaml:"key"`

	// Last Will Testament
	LWT *LWTConfig `json:"lwt,omitempty" yaml:"lwt,omitempty"`
}

// LWTConfig represents Last Will Testament configuration
type LWTConfig struct {
	Topic    string `json:"topic" yaml:"topic" validate:"required"`
	Message  string `json:"message" yaml:"message"`
	QoS      QoS    `json:"qos" yaml:"qos"`
	Retained bool   `json:"retained" yaml:"retained"`
}

// MessageHandler is a function type for handling incoming messages
type MessageHandler func(ctx context.Context, msg *Message) error

// ConnectionHandler is a function type for handling connection state changes
type ConnectionHandler func(connected bool, err error)

// PublishOptions contains options for publishing messages
type PublishOptions struct {
	QoS      QoS
	Retained bool
	Headers  map[string]string
}

// SubscribeOptions contains options for subscribing to topics
type SubscribeOptions struct {
	QoS QoS
}

// ConnectionStatus represents the current connection status
type ConnectionStatus int

const (
	StatusDisconnected ConnectionStatus = iota
	StatusConnecting
	StatusConnected
	StatusReconnecting
	StatusError
)

// String returns the string representation of connection status
func (s ConnectionStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "Disconnected"
	case StatusConnecting:
		return "Connecting"
	case StatusConnected:
		return "Connected"
	case StatusReconnecting:
		return "Reconnecting"
	case StatusError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Statistics contains MQTT client statistics
type Statistics struct {
	MessagesPublished  uint64        `json:"messages_published"`
	MessagesReceived   uint64        `json:"messages_received"`
	BytesPublished     uint64        `json:"bytes_published"`
	BytesReceived      uint64        `json:"bytes_received"`
	ConnectionUptime   time.Duration `json:"connection_uptime"`
	ReconnectCount     uint32        `json:"reconnect_count"`
	LastError          string        `json:"last_error,omitempty"`
	LastErrorTimestamp time.Time     `json:"last_error_timestamp,omitempty"`
}

// Error types
var (
	ErrNotConnected     = &Error{Code: "NOT_CONNECTED", Message: "MQTT client is not connected"}
	ErrAlreadyConnected = &Error{Code: "ALREADY_CONNECTED", Message: "MQTT client is already connected"}
	ErrInvalidConfig    = &Error{Code: "INVALID_CONFIG", Message: "Invalid MQTT configuration"}
	ErrConnectionFailed = &Error{Code: "CONNECTION_FAILED", Message: "Failed to connect to MQTT broker"}
	ErrPublishFailed    = &Error{Code: "PUBLISH_FAILED", Message: "Failed to publish message"}
	ErrSubscribeFailed  = &Error{Code: "SUBSCRIBE_FAILED", Message: "Failed to subscribe to topic"}
)

// Error represents an MQTT-specific error
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"cause,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithCause wraps the error with a cause
func (e *Error) WithCause(cause error) *Error {
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Cause:   cause,
	}
}
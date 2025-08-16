package mqtt

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Subscriber handles MQTT topic subscriptions
type Subscriber struct {
	client      *Client
	mu          sync.RWMutex
	subscribed  map[string]byte // topic -> qos
}

// NewSubscriber creates a new subscriber
func NewSubscriber(client *Client) *Subscriber {
	return &Subscriber{
		client:     client,
		subscribed: make(map[string]byte),
	}
}

// Subscribe subscribes to multiple topics
func (s *Subscriber) Subscribe(topics []string) error {
	if !s.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	// Prepare subscription map
	filters := make(map[string]byte)
	for _, topic := range topics {
		// Default QoS 1 for all subscriptions
		filters[topic] = 1
	}

	log.WithField("topics", topics).Info("Subscribing to MQTT topics")

	// Subscribe to all topics at once
	token := s.client.client.SubscribeMultiple(filters, nil)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", token.Error())
	}

	// Update subscribed topics
	s.mu.Lock()
	for topic, qos := range filters {
		s.subscribed[topic] = qos
	}
	s.mu.Unlock()

	log.WithField("count", len(topics)).Info("Successfully subscribed to topics")
	return nil
}

// SubscribeSingle subscribes to a single topic with specified QoS
func (s *Subscriber) SubscribeSingle(topic string, qos byte) error {
	if !s.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	log.WithFields(log.Fields{
		"topic": topic,
		"qos":   qos,
	}).Info("Subscribing to topic")

	token := s.client.client.Subscribe(topic, qos, nil)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, token.Error())
	}

	s.mu.Lock()
	s.subscribed[topic] = qos
	s.mu.Unlock()

	log.WithField("topic", topic).Info("Successfully subscribed to topic")
	return nil
}

// Unsubscribe unsubscribes from topics
func (s *Subscriber) Unsubscribe(topics []string) error {
	if !s.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	log.WithField("topics", topics).Info("Unsubscribing from topics")

	token := s.client.client.Unsubscribe(topics...)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe from topics: %w", token.Error())
	}

	// Update subscribed topics
	s.mu.Lock()
	for _, topic := range topics {
		delete(s.subscribed, topic)
	}
	s.mu.Unlock()

	log.WithField("count", len(topics)).Info("Successfully unsubscribed from topics")
	return nil
}

// GetSubscribedTopics returns currently subscribed topics
func (s *Subscriber) GetSubscribedTopics() map[string]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make(map[string]byte)
	for topic, qos := range s.subscribed {
		result[topic] = qos
	}
	return result
}

// IsSubscribed checks if a topic is currently subscribed
func (s *Subscriber) IsSubscribed(topic string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	_, exists := s.subscribed[topic]
	return exists
}
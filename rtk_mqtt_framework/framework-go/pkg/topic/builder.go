package topic

import (
	"fmt"
	"strings"
)

// Builder helps construct RTK MQTT topic paths
type Builder struct {
	version  string
	tenant   string
	site     string
	deviceID string
}

// NewBuilder creates a new topic builder
func NewBuilder(tenant, site, deviceID string) *Builder {
	return &Builder{
		version:  "v1",
		tenant:   tenant,
		site:     site,
		deviceID: deviceID,
	}
}

// SetVersion sets the API version (default: v1)
func (b *Builder) SetVersion(version string) *Builder {
	b.version = version
	return b
}

// Base returns the base topic path: rtk/{version}/{tenant}/{site}/{device_id}
func (b *Builder) Base() string {
	return fmt.Sprintf("rtk/%s/%s/%s/%s", b.version, b.tenant, b.site, b.deviceID)
}

// State returns the state topic: rtk/{version}/{tenant}/{site}/{device_id}/state
func (b *Builder) State() string {
	return b.Base() + "/state"
}

// Telemetry returns the telemetry topic for a specific metric
func (b *Builder) Telemetry(metric string) string {
	return fmt.Sprintf("%s/telemetry/%s", b.Base(), metric)
}

// Event returns the event topic for a specific event type
func (b *Builder) Event(eventType string) string {
	return fmt.Sprintf("%s/evt/%s", b.Base(), eventType)
}

// Attribute returns the attribute topic: rtk/{version}/{tenant}/{site}/{device_id}/attr
func (b *Builder) Attribute() string {
	return b.Base() + "/attr"
}

// CommandRequest returns the command request topic
func (b *Builder) CommandRequest() string {
	return b.Base() + "/cmd/req"
}

// CommandAck returns the command acknowledgment topic
func (b *Builder) CommandAck() string {
	return b.Base() + "/cmd/ack"
}

// CommandResponse returns the command response topic
func (b *Builder) CommandResponse() string {
	return b.Base() + "/cmd/res"
}

// LWT returns the Last Will Testament topic
func (b *Builder) LWT() string {
	return b.Base() + "/lwt"
}

// GroupCommand returns the group command topic
func (b *Builder) GroupCommand(groupID string) string {
	return fmt.Sprintf("rtk/%s/%s/%s/group/%s/cmd", b.version, b.tenant, b.site, groupID)
}

// Broadcast returns the broadcast topic for the site
func (b *Builder) Broadcast() string {
	return fmt.Sprintf("rtk/%s/%s/%s/broadcast", b.version, b.tenant, b.site)
}

// Parser helps parse RTK MQTT topic paths
type Parser struct{}

// NewParser creates a new topic parser
func NewParser() *Parser {
	return &Parser{}
}

// TopicInfo contains parsed topic information
type TopicInfo struct {
	Version    string
	Tenant     string
	Site       string
	DeviceID   string
	MessageType string
	SubType    string
	IsValid    bool
}

// Parse parses an RTK MQTT topic and extracts its components
func (p *Parser) Parse(topic string) *TopicInfo {
	info := &TopicInfo{}
	
	parts := strings.Split(topic, "/")
	if len(parts) < 5 {
		return info
	}
	
	// Check if it's an RTK topic
	if parts[0] != "rtk" {
		return info
	}
	
	info.Version = parts[1]
	info.Tenant = parts[2]
	info.Site = parts[3]
	info.DeviceID = parts[4]
	
	if len(parts) >= 6 {
		info.MessageType = parts[5]
		
		if len(parts) >= 7 {
			info.SubType = parts[6]
		}
	}
	
	info.IsValid = true
	return info
}

// IsState checks if the topic is a state topic
func (p *Parser) IsState(topic string) bool {
	info := p.Parse(topic)
	return info.IsValid && info.MessageType == "state"
}

// IsTelemetry checks if the topic is a telemetry topic
func (p *Parser) IsTelemetry(topic string) bool {
	info := p.Parse(topic)
	return info.IsValid && info.MessageType == "telemetry"
}

// IsEvent checks if the topic is an event topic
func (p *Parser) IsEvent(topic string) bool {
	info := p.Parse(topic)
	return info.IsValid && info.MessageType == "evt"
}

// IsCommand checks if the topic is a command topic
func (p *Parser) IsCommand(topic string) bool {
	info := p.Parse(topic)
	return info.IsValid && info.MessageType == "cmd"
}

// IsAttribute checks if the topic is an attribute topic
func (p *Parser) IsAttribute(topic string) bool {
	info := p.Parse(topic)
	return info.IsValid && info.MessageType == "attr"
}

// IsLWT checks if the topic is a Last Will Testament topic
func (p *Parser) IsLWT(topic string) bool {
	info := p.Parse(topic)
	return info.IsValid && info.MessageType == "lwt"
}

// Validator validates RTK topic patterns
type Validator struct{}

// NewValidator creates a new topic validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateTopicName validates a topic name according to MQTT and RTK standards
func (v *Validator) ValidateTopicName(topic string) error {
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	
	if len(topic) > 65535 {
		return fmt.Errorf("topic length exceeds maximum of 65535 characters")
	}
	
	// Check for invalid characters
	if strings.Contains(topic, "+") {
		return fmt.Errorf("topic contains invalid wildcard character '+'")
	}
	
	if strings.Contains(topic, "#") {
		return fmt.Errorf("topic contains invalid wildcard character '#'")
	}
	
	if strings.Contains(topic, "\x00") {
		return fmt.Errorf("topic contains null character")
	}
	
	return nil
}

// ValidateRTKTopic validates an RTK topic structure
func (v *Validator) ValidateRTKTopic(topic string) error {
	if err := v.ValidateTopicName(topic); err != nil {
		return err
	}
	
	parser := NewParser()
	info := parser.Parse(topic)
	
	if !info.IsValid {
		return fmt.Errorf("invalid RTK topic format")
	}
	
	if info.Version == "" {
		return fmt.Errorf("missing version in RTK topic")
	}
	
	if info.Tenant == "" {
		return fmt.Errorf("missing tenant in RTK topic")
	}
	
	if info.Site == "" {
		return fmt.Errorf("missing site in RTK topic")
	}
	
	if info.DeviceID == "" {
		return fmt.Errorf("missing device ID in RTK topic")
	}
	
	return nil
}

// Subscription helps manage topic subscriptions
type Subscription struct {
	patterns map[string]bool
}

// NewSubscription creates a new subscription manager
func NewSubscription() *Subscription {
	return &Subscription{
		patterns: make(map[string]bool),
	}
}

// Subscribe adds a subscription pattern
func (s *Subscription) Subscribe(pattern string) {
	s.patterns[pattern] = true
}

// Unsubscribe removes a subscription pattern
func (s *Subscription) Unsubscribe(pattern string) {
	delete(s.patterns, pattern)
}

// IsSubscribed checks if a pattern is subscribed
func (s *Subscription) IsSubscribed(pattern string) bool {
	return s.patterns[pattern]
}

// GetPatterns returns all subscription patterns
func (s *Subscription) GetPatterns() []string {
	patterns := make([]string, 0, len(s.patterns))
	for pattern := range s.patterns {
		patterns = append(patterns, pattern)
	}
	return patterns
}

// Matches checks if a topic matches any subscribed pattern
func (s *Subscription) Matches(topic string) bool {
	for pattern := range s.patterns {
		if s.matchPattern(pattern, topic) {
			return true
		}
	}
	return false
}

// matchPattern checks if a topic matches a subscription pattern
func (s *Subscription) matchPattern(pattern, topic string) bool {
	// Simple wildcard matching for MQTT topics
	// + matches a single level
	// # matches multiple levels
	
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")
	
	return s.matchParts(patternParts, topicParts)
}

func (s *Subscription) matchParts(pattern, topic []string) bool {
	if len(pattern) == 0 && len(topic) == 0 {
		return true
	}
	
	if len(pattern) == 0 || len(topic) == 0 {
		return false
	}
	
	if pattern[0] == "#" {
		return true // # matches everything
	}
	
	if pattern[0] == "+" || pattern[0] == topic[0] {
		return s.matchParts(pattern[1:], topic[1:])
	}
	
	return false
}

// Utility functions

// BuildDeviceStatePattern builds a subscription pattern for device state topics
func BuildDeviceStatePattern(tenant, site string) string {
	return fmt.Sprintf("rtk/v1/%s/%s/+/state", tenant, site)
}

// BuildDeviceTelemetryPattern builds a subscription pattern for device telemetry topics
func BuildDeviceTelemetryPattern(tenant, site string) string {
	return fmt.Sprintf("rtk/v1/%s/%s/+/telemetry/+", tenant, site)
}

// BuildDeviceEventPattern builds a subscription pattern for device event topics
func BuildDeviceEventPattern(tenant, site string) string {
	return fmt.Sprintf("rtk/v1/%s/%s/+/evt/+", tenant, site)
}

// BuildSiteWildcardPattern builds a subscription pattern for all topics in a site
func BuildSiteWildcardPattern(tenant, site string) string {
	return fmt.Sprintf("rtk/v1/%s/%s/#", tenant, site)
}

// BuildTenantWildcardPattern builds a subscription pattern for all topics in a tenant
func BuildTenantWildcardPattern(tenant string) string {
	return fmt.Sprintf("rtk/v1/%s/#", tenant)
}
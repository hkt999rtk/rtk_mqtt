package utils

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

// TopicMatches checks if a topic matches a pattern (supports MQTT wildcards)
func TopicMatches(pattern, topic string) bool {
	// Handle special cases
	if pattern == "#" {
		return true
	}
	if pattern == topic {
		return true
	}

	// Convert MQTT wildcards to regex
	regexPattern := mqttToRegex(pattern)
	matched, err := regexp.MatchString(regexPattern, topic)
	if err != nil {
		return false
	}
	return matched
}

// mqttToRegex converts MQTT wildcard pattern to regex
func mqttToRegex(pattern string) string {
	// Escape regex special characters except MQTT wildcards
	escaped := regexp.QuoteMeta(pattern)
	
	// Replace escaped MQTT wildcards with regex equivalents
	escaped = strings.ReplaceAll(escaped, `\+`, `[^/]+`)  // + matches single level
	escaped = strings.ReplaceAll(escaped, `\#`, `.*`)     // # matches multi-level
	
	// Anchor the pattern
	return "^" + escaped + "$"
}

// ExtractClientID extracts client ID from topic
func ExtractClientID(topic string) string {
	// Topic format: rtk/v1/{tenant}/{site}/{device_id}/...
	parts := strings.Split(topic, "/")
	if len(parts) >= 5 {
		return parts[4] // device_id is at index 4
	}
	return ""
}

// ExtractMessageType extracts message type from topic
func ExtractMessageType(topic string) string {
	// Topic format: rtk/v1/{tenant}/{site}/{device_id}/{message_type}/...
	parts := strings.Split(topic, "/")
	if len(parts) >= 6 {
		return parts[5] // message_type is at index 5
	}
	return ""
}

// ExtractTopicParts extracts all parts from RTK topic
func ExtractTopicParts(topic string) (tenant, site, deviceID, messageType string, subParts []string) {
	// Topic format: rtk/v1/{tenant}/{site}/{device_id}/{message_type}/{sub_parts...}
	parts := strings.Split(topic, "/")
	
	if len(parts) < 6 {
		return "", "", "", "", nil
	}
	
	tenant = parts[2]
	site = parts[3]
	deviceID = parts[4]
	messageType = parts[5]
	
	if len(parts) > 6 {
		subParts = parts[6:]
	}
	
	return
}

// DeviceMatches checks if a device ID matches a pattern
func DeviceMatches(pattern, deviceID string) bool {
	// Support simple wildcard matching
	if pattern == "*" {
		return true
	}
	if pattern == deviceID {
		return true
	}
	
	// Support prefix matching with *
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(deviceID, prefix)
	}
	
	// Support suffix matching with *
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(deviceID, suffix)
	}
	
	return false
}

// IsValidTopic validates if a topic follows RTK format
func IsValidTopic(topic string) bool {
	parts := strings.Split(topic, "/")
	
	// Minimum parts: rtk/v1/{tenant}/{site}/{device_id}/{message_type}
	if len(parts) < 6 {
		return false
	}
	
	// Check protocol prefix
	if parts[0] != "rtk" || parts[1] != "v1" {
		return false
	}
	
	// Check for empty parts
	for i := 2; i < 6; i++ {
		if parts[i] == "" {
			return false
		}
	}
	
	return true
}

// BuildTopic builds an RTK topic from components
func BuildTopic(tenant, site, deviceID, messageType string, subParts ...string) string {
	parts := []string{"rtk", "v1", tenant, site, deviceID, messageType}
	parts = append(parts, subParts...)
	return strings.Join(parts, "/")
}

// GenerateMessageID generates a unique message ID
func GenerateMessageID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return hex.EncodeToString([]byte(strings.Replace(string(rune(int64(1000000))), ".", "", -1)))
	}
	return hex.EncodeToString(bytes)
}
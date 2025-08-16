package topic

import (
	"testing"
)

func TestNewBuilder(t *testing.T) {
	tenant := "test_tenant"
	site := "test_site"
	deviceID := "test_device"

	builder := NewBuilder(tenant, site, deviceID)

	if builder == nil {
		t.Fatal("Expected builder to be created")
	}

	if builder.tenant != tenant {
		t.Errorf("Expected tenant %s, got %s", tenant, builder.tenant)
	}

	if builder.site != site {
		t.Errorf("Expected site %s, got %s", site, builder.site)
	}

	if builder.deviceID != deviceID {
		t.Errorf("Expected deviceID %s, got %s", deviceID, builder.deviceID)
	}

	if builder.version != "v1" {
		t.Errorf("Expected default version v1, got %s", builder.version)
	}
}

func TestBuilderSetVersion(t *testing.T) {
	builder := NewBuilder("tenant", "site", "device")
	
	result := builder.SetVersion("v2")
	
	if result != builder {
		t.Error("Expected SetVersion to return builder for chaining")
	}
	
	if builder.version != "v2" {
		t.Errorf("Expected version v2, got %s", builder.version)
	}
}

func TestBuilderBase(t *testing.T) {
	builder := NewBuilder("demo", "office", "00:11:22:33:44:55")
	
	expected := "rtk/v1/demo/office/00:11:22:33:44:55"
	result := builder.Base()
	
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestBuilderTopics(t *testing.T) {
	builder := NewBuilder("demo", "office", "00:11:22:33:44:55")
	
	tests := []struct {
		name     string
		method   func() string
		expected string
	}{
		{"State", builder.State, "rtk/v1/demo/office/00:11:22:33:44:55/state"},
		{"Attribute", builder.Attribute, "rtk/v1/demo/office/00:11:22:33:44:55/attr"},
		{"CommandRequest", builder.CommandRequest, "rtk/v1/demo/office/00:11:22:33:44:55/cmd/req"},
		{"CommandAck", builder.CommandAck, "rtk/v1/demo/office/00:11:22:33:44:55/cmd/ack"},
		{"CommandResponse", builder.CommandResponse, "rtk/v1/demo/office/00:11:22:33:44:55/cmd/res"},
		{"LWT", builder.LWT, "rtk/v1/demo/office/00:11:22:33:44:55/lwt"},
		{"Broadcast", builder.Broadcast, "rtk/v1/demo/office/broadcast"},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.method()
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestBuilderTelemetry(t *testing.T) {
	builder := NewBuilder("demo", "office", "00:11:22:33:44:55")
	
	metric := "temperature"
	expected := "rtk/v1/demo/office/00:11:22:33:44:55/telemetry/temperature"
	result := builder.Telemetry(metric)
	
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestBuilderEvent(t *testing.T) {
	builder := NewBuilder("demo", "office", "00:11:22:33:44:55")
	
	eventType := "wifi.roam_miss"
	expected := "rtk/v1/demo/office/00:11:22:33:44:55/evt/wifi.roam_miss"
	result := builder.Event(eventType)
	
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestBuilderGroupCommand(t *testing.T) {
	builder := NewBuilder("demo", "office", "00:11:22:33:44:55")
	
	groupID := "sensors"
	expected := "rtk/v1/demo/office/group/sensors/cmd"
	result := builder.GroupCommand(groupID)
	
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestNewParser(t *testing.T) {
	parser := NewParser()
	
	if parser == nil {
		t.Fatal("Expected parser to be created")
	}
}

func TestParserParse(t *testing.T) {
	parser := NewParser()
	
	tests := []struct {
		name       string
		topic      string
		expected   *TopicInfo
	}{
		{
			name:  "Valid state topic",
			topic: "rtk/v1/demo/office/00:11:22:33:44:55/state",
			expected: &TopicInfo{
				Version:     "v1",
				Tenant:      "demo",
				Site:        "office",
				DeviceID:    "00:11:22:33:44:55",
				MessageType: "state",
				SubType:     "",
				IsValid:     true,
			},
		},
		{
			name:  "Valid telemetry topic",
			topic: "rtk/v1/demo/office/00:11:22:33:44:55/telemetry/temperature",
			expected: &TopicInfo{
				Version:     "v1",
				Tenant:      "demo",
				Site:        "office",
				DeviceID:    "00:11:22:33:44:55",
				MessageType: "telemetry",
				SubType:     "temperature",
				IsValid:     true,
			},
		},
		{
			name:  "Valid event topic",
			topic: "rtk/v1/demo/office/00:11:22:33:44:55/evt/wifi.roam_miss",
			expected: &TopicInfo{
				Version:     "v1",
				Tenant:      "demo",
				Site:        "office",
				DeviceID:    "00:11:22:33:44:55",
				MessageType: "evt",
				SubType:     "wifi.roam_miss",
				IsValid:     true,
			},
		},
		{
			name:  "Invalid topic - not RTK",
			topic: "other/v1/demo/office/00:11:22:33:44:55/state",
			expected: &TopicInfo{
				IsValid: false,
			},
		},
		{
			name:  "Invalid topic - too short",
			topic: "rtk/v1/demo",
			expected: &TopicInfo{
				IsValid: false,
			},
		},
		{
			name:  "Minimal valid topic",
			topic: "rtk/v1/demo/office/00:11:22:33:44:55",
			expected: &TopicInfo{
				Version:     "v1",
				Tenant:      "demo",
				Site:        "office",
				DeviceID:    "00:11:22:33:44:55",
				MessageType: "",
				SubType:     "",
				IsValid:     true,
			},
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parser.Parse(test.topic)
			
			if result.IsValid != test.expected.IsValid {
				t.Errorf("Expected IsValid %v, got %v", test.expected.IsValid, result.IsValid)
			}
			
			if !test.expected.IsValid {
				return // Skip other checks for invalid topics
			}
			
			if result.Version != test.expected.Version {
				t.Errorf("Expected Version %s, got %s", test.expected.Version, result.Version)
			}
			
			if result.Tenant != test.expected.Tenant {
				t.Errorf("Expected Tenant %s, got %s", test.expected.Tenant, result.Tenant)
			}
			
			if result.Site != test.expected.Site {
				t.Errorf("Expected Site %s, got %s", test.expected.Site, result.Site)
			}
			
			if result.DeviceID != test.expected.DeviceID {
				t.Errorf("Expected DeviceID %s, got %s", test.expected.DeviceID, result.DeviceID)
			}
			
			if result.MessageType != test.expected.MessageType {
				t.Errorf("Expected MessageType %s, got %s", test.expected.MessageType, result.MessageType)
			}
			
			if result.SubType != test.expected.SubType {
				t.Errorf("Expected SubType %s, got %s", test.expected.SubType, result.SubType)
			}
		})
	}
}

func TestParserTypeCheckers(t *testing.T) {
	parser := NewParser()
	
	tests := []struct {
		name     string
		topic    string
		checker  func(string) bool
		expected bool
	}{
		{"IsState - valid", "rtk/v1/demo/office/00:11:22:33:44:55/state", parser.IsState, true},
		{"IsState - invalid", "rtk/v1/demo/office/00:11:22:33:44:55/telemetry/temp", parser.IsState, false},
		{"IsTelemetry - valid", "rtk/v1/demo/office/00:11:22:33:44:55/telemetry/temp", parser.IsTelemetry, true},
		{"IsTelemetry - invalid", "rtk/v1/demo/office/00:11:22:33:44:55/state", parser.IsTelemetry, false},
		{"IsEvent - valid", "rtk/v1/demo/office/00:11:22:33:44:55/evt/error", parser.IsEvent, true},
		{"IsEvent - invalid", "rtk/v1/demo/office/00:11:22:33:44:55/state", parser.IsEvent, false},
		{"IsCommand - valid", "rtk/v1/demo/office/00:11:22:33:44:55/cmd/req", parser.IsCommand, true},
		{"IsCommand - invalid", "rtk/v1/demo/office/00:11:22:33:44:55/state", parser.IsCommand, false},
		{"IsAttribute - valid", "rtk/v1/demo/office/00:11:22:33:44:55/attr", parser.IsAttribute, true},
		{"IsAttribute - invalid", "rtk/v1/demo/office/00:11:22:33:44:55/state", parser.IsAttribute, false},
		{"IsLWT - valid", "rtk/v1/demo/office/00:11:22:33:44:55/lwt", parser.IsLWT, true},
		{"IsLWT - invalid", "rtk/v1/demo/office/00:11:22:33:44:55/state", parser.IsLWT, false},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.checker(test.topic)
			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

func TestValidatorValidateTopicName(t *testing.T) {
	validator := NewValidator()
	
	tests := []struct {
		name      string
		topic     string
		expectErr bool
	}{
		{"Valid topic", "rtk/v1/demo/office/00:11:22:33:44:55/state", false},
		{"Empty topic", "", true},
		{"Too long topic", string(make([]byte, 65536)), true},
		{"Topic with +", "rtk/v1/demo/+/00:11:22:33:44:55/state", true},
		{"Topic with #", "rtk/v1/demo/office/#", true},
		{"Topic with null", "rtk/v1/demo\x00office/00:11:22:33:44:55/state", true},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateTopicName(test.topic)
			
			if test.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !test.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidatorValidateRTKTopic(t *testing.T) {
	validator := NewValidator()
	
	tests := []struct {
		name      string
		topic     string
		expectErr bool
	}{
		{"Valid RTK topic", "rtk/v1/demo/office/00:11:22:33:44:55/state", false},
		{"Invalid RTK format", "other/v1/demo/office/00:11:22:33:44:55/state", true},
		{"Missing components", "rtk/v1/demo", true},
		{"Empty topic", "", true},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateRTKTopic(test.topic)
			
			if test.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !test.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestSubscription(t *testing.T) {
	sub := NewSubscription()
	
	if sub == nil {
		t.Fatal("Expected subscription to be created")
	}
	
	// Test subscribe
	pattern := "rtk/v1/demo/office/+/state"
	sub.Subscribe(pattern)
	
	if !sub.IsSubscribed(pattern) {
		t.Error("Expected pattern to be subscribed")
	}
	
	patterns := sub.GetPatterns()
	if len(patterns) != 1 {
		t.Errorf("Expected 1 pattern, got %d", len(patterns))
	}
	
	if patterns[0] != pattern {
		t.Errorf("Expected pattern %s, got %s", pattern, patterns[0])
	}
	
	// Test unsubscribe
	sub.Unsubscribe(pattern)
	
	if sub.IsSubscribed(pattern) {
		t.Error("Expected pattern to be unsubscribed")
	}
	
	patterns = sub.GetPatterns()
	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns, got %d", len(patterns))
	}
}

func TestSubscriptionMatches(t *testing.T) {
	sub := NewSubscription()
	
	// Subscribe to patterns
	sub.Subscribe("rtk/v1/demo/office/+/state")
	sub.Subscribe("rtk/v1/demo/office/00:11:22:33:44:55/#")
	sub.Subscribe("rtk/v1/demo/office/aa:bb:cc:dd:ee:ff/telemetry/+")
	
	tests := []struct {
		name     string
		topic    string
		expected bool
	}{
		{"Matches single level wildcard", "rtk/v1/demo/office/00:11:22:33:44:55/state", true},
		{"Matches multi level wildcard", "rtk/v1/demo/office/00:11:22:33:44:55/telemetry/temperature", true},
		{"Matches specific telemetry", "rtk/v1/demo/office/aa:bb:cc:dd:ee:ff/telemetry/humidity", true},
		{"No match - wrong device", "rtk/v1/demo/office/99:88:77:66:55:44/telemetry/temperature", false},
		{"No match - wrong message type", "rtk/v1/demo/office/99:88:77:66:55:44/evt/error", false},
		{"No match - different tenant", "rtk/v1/other/office/00:11:22:33:44:55/state", false},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := sub.Matches(test.topic)
			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

func TestUtilityFunctions(t *testing.T) {
	tenant := "demo"
	site := "office"
	
	tests := []struct {
		name     string
		function func(string, string) string
		expected string
	}{
		{"BuildDeviceStatePattern", BuildDeviceStatePattern, "rtk/v1/demo/office/+/state"},
		{"BuildDeviceTelemetryPattern", BuildDeviceTelemetryPattern, "rtk/v1/demo/office/+/telemetry/+"},
		{"BuildDeviceEventPattern", BuildDeviceEventPattern, "rtk/v1/demo/office/+/evt/+"},
		{"BuildSiteWildcardPattern", BuildSiteWildcardPattern, "rtk/v1/demo/office/#"},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.function(tenant, site)
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestBuildTenantWildcardPattern(t *testing.T) {
	tenant := "demo"
	expected := "rtk/v1/demo/#"
	result := BuildTenantWildcardPattern(tenant)
	
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}
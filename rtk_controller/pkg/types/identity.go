package types

import (
	"time"
)

// DeviceIdentity represents device identity information
type DeviceIdentity struct {
	MacAddress    string    `json:"mac_address"`              // Primary key: MAC address
	FriendlyName  string    `json:"friendly_name"`            // User-friendly name like "客廳冷氣"
	DeviceType    string    `json:"device_type,omitempty"`    // phone, laptop, tv, iot, router, ap
	Manufacturer  string    `json:"manufacturer,omitempty"`   // Apple, Samsung, TP-Link
	Model         string    `json:"model,omitempty"`          // iPhone 15, Galaxy S24
	Location      string    `json:"location,omitempty"`       // 客廳, 主臥, 廚房
	Owner         string    `json:"owner,omitempty"`          // Kevin, Alice, 家用設備
	Category      string    `json:"category,omitempty"`       // personal, shared, infrastructure
	Tags          []string  `json:"tags,omitempty"`           // gaming, work, entertainment
	
	// Auto-detection related fields
	AutoDetected   bool                 `json:"auto_detected"`       // Was this identity auto-detected?
	DetectionRules []DetectionRuleMatch `json:"detection_rules,omitempty"` // Which rules matched
	Confidence     float64              `json:"confidence"`          // Confidence level (0.0-1.0)
	
	// Metadata
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	LastUpdated   time.Time `json:"last_updated"`
	UpdatedBy     string    `json:"updated_by"`              // user, auto_detection, import
	Notes         string    `json:"notes,omitempty"`
}

// DetectionRule represents a rule for automatic device detection
type DetectionRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`              // Higher priority rules are evaluated first
	
	// Matching criteria
	Conditions  []DetectionCondition   `json:"conditions"`
	
	// Action to take when matched
	Action      DetectionAction        `json:"action"`
	
	// Metadata
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// DetectionCondition represents a condition for device detection
type DetectionCondition struct {
	Type     string `json:"type"`     // mac_prefix, hostname_pattern, manufacturer, dhcp_vendor, user_agent
	Field    string `json:"field"`    // Which field to match against
	Operator string `json:"operator"` // equals, contains, starts_with, regex, in_list
	Value    string `json:"value"`    // The value to match
	Negate   bool   `json:"negate"`   // Negate the condition
}

// DetectionAction represents the action to take when a detection rule matches
type DetectionAction struct {
	SetFriendlyName  string   `json:"set_friendly_name,omitempty"`
	SetDeviceType    string   `json:"set_device_type,omitempty"`
	SetManufacturer  string   `json:"set_manufacturer,omitempty"`
	SetModel         string   `json:"set_model,omitempty"`
	SetLocation      string   `json:"set_location,omitempty"`
	SetOwner         string   `json:"set_owner,omitempty"`
	SetCategory      string   `json:"set_category,omitempty"`
	AddTags          []string `json:"add_tags,omitempty"`
	Confidence       float64  `json:"confidence"`                    // Confidence level for this rule
}

// DetectionRuleMatch represents a matched detection rule
type DetectionRuleMatch struct {
	RuleID       string    `json:"rule_id"`
	RuleName     string    `json:"rule_name"`
	MatchedField string    `json:"matched_field"`
	MatchedValue string    `json:"matched_value"`
	Confidence   float64   `json:"confidence"`
	MatchedAt    time.Time `json:"matched_at"`
}

// DeviceIdentityFilter represents filtering criteria for device identities
type DeviceIdentityFilter struct {
	DeviceType     string   `json:"device_type,omitempty"`
	Manufacturer   string   `json:"manufacturer,omitempty"`
	Location       string   `json:"location,omitempty"`
	Owner          string   `json:"owner,omitempty"`
	Category       string   `json:"category,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	AutoDetected   *bool    `json:"auto_detected,omitempty"`
	LastSeenAfter  *int64   `json:"last_seen_after,omitempty"`  // Unix timestamp
	LastSeenBefore *int64   `json:"last_seen_before,omitempty"` // Unix timestamp
}

// DeviceIdentityStats represents statistics about device identities
type DeviceIdentityStats struct {
	TotalDevices        int            `json:"total_devices"`
	AutoDetectedDevices int            `json:"auto_detected_devices"`
	ManualDevices       int            `json:"manual_devices"`
	DeviceTypeStats     map[string]int `json:"device_type_stats"`
	ManufacturerStats   map[string]int `json:"manufacturer_stats"`
	LocationStats       map[string]int `json:"location_stats"`
	CategoryStats       map[string]int `json:"category_stats"`
	LastUpdated         time.Time      `json:"last_updated"`
}

// DeviceDetectionCandidate represents a device that needs identity detection
type DeviceDetectionCandidate struct {
	MacAddress   string                 `json:"mac_address"`
	Hostname     string                 `json:"hostname,omitempty"`
	Manufacturer string                 `json:"manufacturer,omitempty"`    // From MAC OUI lookup
	DHCPVendor   string                 `json:"dhcp_vendor,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	DeviceInfo   map[string]interface{} `json:"device_info,omitempty"`     // Additional device information
	FirstSeen    time.Time              `json:"first_seen"`
	LastSeen     time.Time              `json:"last_seen"`
}

// IdentityImportEntry represents an entry for bulk identity import
type IdentityImportEntry struct {
	MacAddress   string   `json:"mac_address"`
	FriendlyName string   `json:"friendly_name"`
	DeviceType   string   `json:"device_type,omitempty"`
	Manufacturer string   `json:"manufacturer,omitempty"`
	Model        string   `json:"model,omitempty"`
	Location     string   `json:"location,omitempty"`
	Owner        string   `json:"owner,omitempty"`
	Category     string   `json:"category,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Notes        string   `json:"notes,omitempty"`
}

// IdentityExportEntry represents an entry for identity export
type IdentityExportEntry struct {
	MacAddress   string   `json:"mac_address"`
	FriendlyName string   `json:"friendly_name"`
	DeviceType   string   `json:"device_type,omitempty"`
	Manufacturer string   `json:"manufacturer,omitempty"`
	Model        string   `json:"model,omitempty"`
	Location     string   `json:"location,omitempty"`
	Owner        string   `json:"owner,omitempty"`
	Category     string   `json:"category,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	AutoDetected bool     `json:"auto_detected"`
	Confidence   float64  `json:"confidence"`
	FirstSeen    string   `json:"first_seen"`    // ISO 8601 format
	LastSeen     string   `json:"last_seen"`     // ISO 8601 format
	Notes        string   `json:"notes,omitempty"`
}
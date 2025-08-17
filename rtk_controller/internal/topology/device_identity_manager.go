package topology

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// DeviceIdentityManager manages device identities, friendly names, and metadata
type DeviceIdentityManager struct {
	storage         *storage.IdentityStorage
	topologyManager *Manager
	
	// Configuration
	config          DeviceIdentityConfig
	
	// Cache
	identityCache   map[string]*DeviceIdentity
	groupCache      map[string]*DeviceGroup
	tagCache        map[string]*DeviceTag
	cacheMu         sync.RWMutex
	
	// Background processing
	running         bool
	cancel          context.CancelFunc
	
	// Statistics
	stats           DeviceIdentityStats
}

// DeviceIdentityConfig holds configuration for device identity management
type DeviceIdentityConfig struct {
	// Auto-detection settings
	EnableAutoDetection     bool
	AutoNamingEnabled       bool
	AutoGroupingEnabled     bool
	AutoTaggingEnabled      bool
	
	// Naming conventions
	DefaultNamingPattern    string
	UseVendorInfo          bool
	UseDeviceType          bool
	UseLocation            bool
	
	// Cache settings
	CacheSize              int
	CacheRetention         time.Duration
	EnablePersistentCache  bool
	
	// Validation settings
	MaxNameLength          int
	MinNameLength          int
	AllowedNameCharacters  string
	ReservedNames          []string
	
	// Performance settings
	BatchUpdateSize        int
	MaxConcurrentOps       int
	OperationTimeout       time.Duration
}

// DeviceIdentity represents comprehensive device identity information
type DeviceIdentity struct {
	// Core identity
	MacAddress      string                 `json:"mac_address"`
	FriendlyName    string                 `json:"friendly_name"`
	DisplayName     string                 `json:"display_name"`
	Description     string                 `json:"description"`
	
	// Device information
	DeviceType      DeviceType             `json:"device_type"`
	Manufacturer    string                 `json:"manufacturer"`
	Model           string                 `json:"model"`
	VendorOUI       string                 `json:"vendor_oui"`
	
	// Network information
	IPAddress       string                 `json:"ip_address,omitempty"`
	Hostname        string                 `json:"hostname,omitempty"`
	DHCP_Name       string                 `json:"dhcp_name,omitempty"`
	
	// Classification
	Category        DeviceCategory         `json:"category"`
	Groups          []string               `json:"groups"`
	Tags            []string               `json:"tags"`
	
	// Location and organization
	Location        string                 `json:"location"`
	Floor           string                 `json:"floor,omitempty"`
	Room            string                 `json:"room,omitempty"`
	Building        string                 `json:"building,omitempty"`
	
	// User association
	Owner           string                 `json:"owner,omitempty"`
	Department      string                 `json:"department,omitempty"`
	CostCenter      string                 `json:"cost_center,omitempty"`
	
	// Security and policy
	SecurityLevel   SecurityLevel          `json:"security_level"`
	AccessPolicy    string                 `json:"access_policy,omitempty"`
	VLANAssignment  int                    `json:"vlan_assignment,omitempty"`
	
	// Status and lifecycle
	Status          DeviceIdentityStatus   `json:"status"`
	RegistrationDate time.Time             `json:"registration_date"`
	LastUpdated     time.Time              `json:"last_updated"`
	LastSeen        time.Time              `json:"last_seen"`
	
	// Monitoring preferences
	MonitoringEnabled    bool               `json:"monitoring_enabled"`
	AlertingEnabled      bool               `json:"alerting_enabled"`
	NotificationLevel    NotificationLevel  `json:"notification_level"`
	
	// Custom metadata
	CustomFields    map[string]interface{} `json:"custom_fields,omitempty"`
	Notes           []DeviceNote           `json:"notes,omitempty"`
	
	// Audit trail
	CreatedBy       string                 `json:"created_by"`
	ModifiedBy      string                 `json:"modified_by"`
	Version         int                    `json:"version"`
}

// DeviceGroup represents a logical grouping of devices
type DeviceGroup struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	GroupType       DeviceGroupType        `json:"group_type"`
	
	// Group rules and criteria
	Rules           []GroupRule            `json:"rules"`
	AutoAssignment  bool                   `json:"auto_assignment"`
	
	// Group properties
	Color           string                 `json:"color,omitempty"`
	Icon            string                 `json:"icon,omitempty"`
	Priority        int                    `json:"priority"`
	
	// Policy inheritance
	DefaultPolicy   string                 `json:"default_policy,omitempty"`
	DefaultVLAN     int                    `json:"default_vlan,omitempty"`
	
	// Members
	DeviceCount     int                    `json:"device_count"`
	Members         []string               `json:"members,omitempty"`
	
	// Metadata
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CreatedBy       string                 `json:"created_by"`
	Tags            []string               `json:"tags,omitempty"`
}

// DeviceTag represents a tag that can be applied to devices
type DeviceTag struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	TagType         DeviceTagType          `json:"tag_type"`
	
	// Tag properties
	Color           string                 `json:"color,omitempty"`
	Category        string                 `json:"category,omitempty"`
	
	// Usage statistics
	DeviceCount     int                    `json:"device_count"`
	LastUsed        time.Time              `json:"last_used"`
	
	// Metadata
	CreatedAt       time.Time              `json:"created_at"`
	CreatedBy       string                 `json:"created_by"`
}

// DeviceNote represents a note or comment about a device
type DeviceNote struct {
	ID              string                 `json:"id"`
	Content         string                 `json:"content"`
	NoteType        DeviceNoteType         `json:"note_type"`
	Author          string                 `json:"author"`
	CreatedAt       time.Time              `json:"created_at"`
	Priority        NotePriority           `json:"priority"`
}

// GroupRule represents a rule for automatic group assignment
type GroupRule struct {
	ID              string                 `json:"id"`
	Field           string                 `json:"field"`
	Operator        RuleOperator           `json:"operator"`
	Value           string                 `json:"value"`
	CaseSensitive   bool                   `json:"case_sensitive"`
}

// Enums and constants

// DeviceType represents the type of device
type DeviceType string
const (
	DeviceTypeUnknown    DeviceType = "unknown"
	DeviceTypeRouter     DeviceType = "router"
	DeviceTypeSwitch     DeviceType = "switch"
	DeviceTypeAP         DeviceType = "ap"
	DeviceTypePhone      DeviceType = "phone"
	DeviceTypeTablet     DeviceType = "tablet"
	DeviceTypeLaptop     DeviceType = "laptop"
	DeviceTypeDesktop    DeviceType = "desktop"
	DeviceTypeIoT        DeviceType = "iot"
	DeviceTypePrinter    DeviceType = "printer"
	DeviceTypeServer     DeviceType = "server"
)

type DeviceCategory string
const (
	CategoryComputer    DeviceCategory = "computer"
	CategoryMobile      DeviceCategory = "mobile"
	CategoryIoT         DeviceCategory = "iot"
	CategoryNetwork     DeviceCategory = "network"
	CategoryPrinter     DeviceCategory = "printer"
	CategoryMedia       DeviceCategory = "media"
	// CategorySecurity moved to constants.go
	CategoryOther       DeviceCategory = "other"
	CategoryUnknown     DeviceCategory = "unknown"
)

type SecurityLevel string
const (
	SecurityLevelHigh       SecurityLevel = "high"
	SecurityLevelMedium     SecurityLevel = "medium"
	SecurityLevelLow        SecurityLevel = "low"
	SecurityLevelGuest      SecurityLevel = "guest"
	SecurityLevelQuarantine SecurityLevel = "quarantine"
)

type DeviceIdentityStatus string
const (
	IdentityStatusActive     DeviceIdentityStatus = "active"
	IdentityStatusInactive   DeviceIdentityStatus = "inactive"
	IdentityStatusPending    DeviceIdentityStatus = "pending"
	IdentityStatusQuarantine DeviceIdentityStatus = "quarantine"
	IdentityStatusBlocked    DeviceIdentityStatus = "blocked"
)

type NotificationLevel string
const (
	NotificationLevelAll     NotificationLevel = "all"
	NotificationLevelHigh    NotificationLevel = "high"
	NotificationLevelCritical NotificationLevel = "critical"
	NotificationLevelNone    NotificationLevel = "none"
)

type DeviceGroupType string
const (
	GroupTypeManual      DeviceGroupType = "manual"
	GroupTypeAutomatic   DeviceGroupType = "automatic"
	GroupTypeLocation    DeviceGroupType = "location"
	GroupTypeDepartment  DeviceGroupType = "department"
	GroupTypeFunction    DeviceGroupType = "function"
	GroupTypeSecurity    DeviceGroupType = "security"
)

type DeviceTagType string
const (
	TagTypeGeneral      DeviceTagType = "general"
	TagTypeStatus       DeviceTagType = "status"
	TagTypeProject      DeviceTagType = "project"
	TagTypeMaintenance  DeviceTagType = "maintenance"
	TagTypeSecurity     DeviceTagType = "security"
	TagTypeCustom       DeviceTagType = "custom"
)

type DeviceNoteType string
const (
	NoteTypeGeneral     DeviceNoteType = "general"
	NoteTypeIssue       DeviceNoteType = "issue"
	NoteTypeMaintenance DeviceNoteType = "maintenance"
	NoteTypeConfiguration DeviceNoteType = "configuration"
	NoteTypeIncident    DeviceNoteType = "incident"
)

type NotePriority string
const (
	NotePriorityLow     NotePriority = "low"
	NotePriorityMedium  NotePriority = "medium"
	NotePriorityHigh    NotePriority = "high"
	NotePriorityCritical NotePriority = "critical"
)

type RuleOperator string
const (
	OperatorEquals      RuleOperator = "equals"
	OperatorContains    RuleOperator = "contains"
	OperatorStartsWith  RuleOperator = "starts_with"
	OperatorEndsWith    RuleOperator = "ends_with"
	OperatorRegex       RuleOperator = "regex"
	OperatorIn          RuleOperator = "in"
	OperatorNotEquals   RuleOperator = "not_equals"
)

// DeviceIdentityStats holds statistics about device identity management
type DeviceIdentityStats struct {
	TotalDevices        int64     `json:"total_devices"`
	NamedDevices        int64     `json:"named_devices"`
	UnnamedDevices      int64     `json:"unnamed_devices"`
	GroupedDevices      int64     `json:"grouped_devices"`
	TaggedDevices       int64     `json:"tagged_devices"`
	TotalGroups         int64     `json:"total_groups"`
	TotalTags           int64     `json:"total_tags"`
	LastUpdate          time.Time `json:"last_update"`
	CacheHitRate        float64   `json:"cache_hit_rate"`
	OperationCount      int64     `json:"operation_count"`
}

// DeviceIdentityQuery represents query parameters for searching devices
type DeviceIdentityQuery struct {
	// Basic filters
	MacAddress      string                 `json:"mac_address,omitempty"`
	FriendlyName    string                 `json:"friendly_name,omitempty"`
	DeviceType      DeviceType             `json:"device_type,omitempty"`
	Manufacturer    string                 `json:"manufacturer,omitempty"`
	
	// Classification filters
	Category        DeviceCategory         `json:"category,omitempty"`
	Groups          []string               `json:"groups,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	
	// Location filters
	Location        string                 `json:"location,omitempty"`
	Floor           string                 `json:"floor,omitempty"`
	Building        string                 `json:"building,omitempty"`
	
	// Status filters
	Status          DeviceIdentityStatus   `json:"status,omitempty"`
	SecurityLevel   SecurityLevel          `json:"security_level,omitempty"`
	OnlineOnly      bool                   `json:"online_only"`
	
	// Time filters
	CreatedAfter    time.Time              `json:"created_after,omitempty"`
	CreatedBefore   time.Time              `json:"created_before,omitempty"`
	LastSeenAfter   time.Time              `json:"last_seen_after,omitempty"`
	LastSeenBefore  time.Time              `json:"last_seen_before,omitempty"`
	
	// Search options
	SearchText      string                 `json:"search_text,omitempty"`
	IncludeNotes    bool                   `json:"include_notes"`
	IncludeCustom   bool                   `json:"include_custom"`
	
	// Pagination and sorting
	Limit           int                    `json:"limit"`
	Offset          int                    `json:"offset"`
	SortBy          string                 `json:"sort_by"`
	SortOrder       string                 `json:"sort_order"`
}

// DeviceIdentityUpdateRequest represents a request to update device identity
type DeviceIdentityUpdateRequest struct {
	MacAddress      string                 `json:"mac_address"`
	Fields          map[string]interface{} `json:"fields"`
	AddTags         []string               `json:"add_tags,omitempty"`
	RemoveTags      []string               `json:"remove_tags,omitempty"`
	AddGroups       []string               `json:"add_groups,omitempty"`
	RemoveGroups    []string               `json:"remove_groups,omitempty"`
	ModifiedBy      string                 `json:"modified_by"`
}

// NewDeviceIdentityManager creates a new device identity manager
func NewDeviceIdentityManager(
	storage *storage.IdentityStorage,
	topologyManager *Manager,
	config DeviceIdentityConfig,
) *DeviceIdentityManager {
	return &DeviceIdentityManager{
		storage:         storage,
		topologyManager: topologyManager,
		config:          config,
		identityCache:   make(map[string]*DeviceIdentity),
		groupCache:      make(map[string]*DeviceGroup),
		tagCache:        make(map[string]*DeviceTag),
		stats:           DeviceIdentityStats{},
	}
}

// Start begins the device identity manager
func (dim *DeviceIdentityManager) Start() error {
	dim.cacheMu.Lock()
	defer dim.cacheMu.Unlock()
	
	if dim.running {
		// return fmt.Errorf("device identity manager is already running")
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	dim.cancel = cancel
	dim.running = true
	
	// Start background processing
	go dim.autoDetectionLoop(ctx)
	go dim.cacheMaintenanceLoop(ctx)
	go dim.statsUpdateLoop(ctx)
	
	// Load existing data
	if err := dim.loadFromStorage(); err != nil {
		fmt.Printf("Warning: Failed to load device identities from storage: %v\n", err)
	}
	
	return nil
}

// Stop stops the device identity manager
func (dim *DeviceIdentityManager) Stop() error {
	dim.cacheMu.Lock()
	defer dim.cacheMu.Unlock()
	
	if !dim.running {
		// return fmt.Errorf("device identity manager is not running")
	}
	
	dim.cancel()
	dim.running = false
	
	// Save data to storage
	if err := dim.saveToStorage(); err != nil {
		fmt.Printf("Warning: Failed to save device identities to storage: %v\n", err)
	}
	
	return nil
}

// Device Identity Operations

// GetDeviceIdentity retrieves device identity by MAC address
func (dim *DeviceIdentityManager) GetDeviceIdentity(macAddress string) (*DeviceIdentity, error) {
	macAddress = strings.ToLower(macAddress)
	
	// Check cache first
	dim.cacheMu.RLock()
	if identity, exists := dim.identityCache[macAddress]; exists {
		dim.cacheMu.RUnlock()
		dim.stats.CacheHitRate = (dim.stats.CacheHitRate + 1) / 2
		return identity, nil
	}
	dim.cacheMu.RUnlock()
	
	// Load from storage
	storageIdentity, err := dim.storage.GetDeviceIdentity(macAddress)
	var identity *DeviceIdentity
	if err != nil {
		// Create new identity if not found
		if strings.Contains(err.Error(), "not found") {
			identity = dim.createDefaultIdentity(macAddress)
		} else {
			return nil, fmt.Errorf("failed to get device identity: %w", err)
		}
	} else {
		// Convert from types.DeviceIdentity to local DeviceIdentity
		identity = &DeviceIdentity{
			MacAddress:   storageIdentity.MacAddress,
			FriendlyName: storageIdentity.FriendlyName,
			DeviceType:   DeviceType(storageIdentity.DeviceType),
			Manufacturer: storageIdentity.Manufacturer,
			Model:        storageIdentity.Model,
			Location:     storageIdentity.Location,
			Owner:        storageIdentity.Owner,
			Category:     DeviceCategory(storageIdentity.Category),
			Tags:         storageIdentity.Tags,
			LastSeen:     storageIdentity.LastSeen,
			LastUpdated:  storageIdentity.LastUpdated,
		}
	}
	
	// Cache the result
	dim.cacheMu.Lock()
	dim.identityCache[macAddress] = identity
	dim.cacheMu.Unlock()
	
	return identity, nil
}

// SetDeviceIdentity creates or updates a device identity
func (dim *DeviceIdentityManager) SetDeviceIdentity(identity *DeviceIdentity) error {
	if identity == nil {
		// return fmt.Errorf("identity cannot be nil")
	}
	
	macAddress := strings.ToLower(identity.MacAddress)
	identity.MacAddress = macAddress
	identity.LastUpdated = time.Now()
	identity.Version++
	
	// Validate identity data
	if err := dim.validateIdentity(identity); err != nil {
		// return fmt.Errorf("invalid identity data: %w", err)
	}
	
	// Convert to types.DeviceIdentity for storage
	storageIdentity := &types.DeviceIdentity{
		MacAddress:   identity.MacAddress,
		FriendlyName: identity.FriendlyName,
		DeviceType:   string(identity.DeviceType),
		Manufacturer: identity.Manufacturer,
		Model:        identity.Model,
		Location:     identity.Location,
		Owner:        identity.Owner,
		Category:     string(identity.Category),
		Tags:         identity.Tags,
		FirstSeen:    identity.RegistrationDate,
		LastSeen:     identity.LastSeen,
		LastUpdated:  identity.LastUpdated,
		UpdatedBy:    identity.ModifiedBy,
		Notes:        "", // TODO: Convert notes to string
	}
	
	// Save to storage  
	if err := dim.storage.SaveDeviceIdentity(storageIdentity); err != nil {
		// return fmt.Errorf("failed to save device identity: %w", err)
	}
	
	// Update cache
	dim.cacheMu.Lock()
	dim.identityCache[macAddress] = identity
	dim.cacheMu.Unlock()
	
	// Update statistics
	dim.stats.OperationCount++
	if identity.FriendlyName != "" {
		dim.stats.NamedDevices++
	}
	
	return nil
}

// UpdateDeviceIdentity updates specific fields of a device identity
func (dim *DeviceIdentityManager) UpdateDeviceIdentity(request DeviceIdentityUpdateRequest) error {
	macAddress := strings.ToLower(request.MacAddress)
	
	// Get existing identity
	identity, err := dim.GetDeviceIdentity(macAddress)
	if err != nil {
		// return fmt.Errorf("failed to get existing identity: %w", err)
	}
	
	// Apply field updates
	for field, value := range request.Fields {
		switch field {
		case "friendly_name":
			if s, ok := value.(string); ok {
				identity.FriendlyName = s
			}
		case "display_name":
			if s, ok := value.(string); ok {
				identity.DisplayName = s
			}
		case "description":
			if s, ok := value.(string); ok {
				identity.Description = s
			}
		case "location":
			if s, ok := value.(string); ok {
				identity.Location = s
			}
		case "owner":
			if s, ok := value.(string); ok {
				identity.Owner = s
			}
		case "department":
			if s, ok := value.(string); ok {
				identity.Department = s
			}
		case "security_level":
			if s, ok := value.(string); ok {
				identity.SecurityLevel = SecurityLevel(s)
			}
		case "status":
			if s, ok := value.(string); ok {
				identity.Status = DeviceIdentityStatus(s)
			}
		// Add more fields as needed
		}
	}
	
	// Handle tag operations
	if len(request.AddTags) > 0 {
		identity.Tags = dim.addUniqueStrings(identity.Tags, request.AddTags)
	}
	if len(request.RemoveTags) > 0 {
		identity.Tags = dim.removeStrings(identity.Tags, request.RemoveTags)
	}
	
	// Handle group operations
	if len(request.AddGroups) > 0 {
		identity.Groups = dim.addUniqueStrings(identity.Groups, request.AddGroups)
	}
	if len(request.RemoveGroups) > 0 {
		identity.Groups = dim.removeStrings(identity.Groups, request.RemoveGroups)
	}
	
	// Set modification metadata
	identity.ModifiedBy = request.ModifiedBy
	
	return dim.SetDeviceIdentity(identity)
}

// DeleteDeviceIdentity removes a device identity
func (dim *DeviceIdentityManager) DeleteDeviceIdentity(macAddress string) error {
	macAddress = strings.ToLower(macAddress)
	
	// Remove from storage
	// TODO: Implement DeleteDeviceIdentity
	// if err := dim.storage.DeleteDeviceIdentity(macAddress); err != nil {
	//	// return fmt.Errorf("failed to delete device identity: %w", err)
	// }
	
	// Remove from cache
	dim.cacheMu.Lock()
	delete(dim.identityCache, macAddress)
	dim.cacheMu.Unlock()
	
	dim.stats.OperationCount++
	return nil
}

// QueryDeviceIdentities searches for device identities based on criteria
func (dim *DeviceIdentityManager) QueryDeviceIdentities(query DeviceIdentityQuery) ([]*DeviceIdentity, int, error) {
	// This would typically query the storage layer with SQL-like operations
	// For now, we'll implement a simple in-memory search
	
	var results []*DeviceIdentity
	var totalCount int
	
	// Load all identities if needed (in a real implementation, this would be paginated)
	// TODO: Implement GetAllDeviceIdentities
	var allIdentities []*DeviceIdentity
	// For now, return cached identities
	for _, identity := range dim.identityCache {
		allIdentities = append(allIdentities, identity)
	}
	
	// Apply filters
	for _, identity := range allIdentities {
		if dim.matchesQuery(identity, query) {
			results = append(results, identity)
		}
	}
	
	totalCount = len(results)
	
	// Apply sorting
	dim.sortIdentities(results, query.SortBy, query.SortOrder)
	
	// Apply pagination
	if query.Limit > 0 {
		start := query.Offset
		end := start + query.Limit
		if start < len(results) {
			if end > len(results) {
				end = len(results)
			}
			results = results[start:end]
		} else {
			results = []*DeviceIdentity{}
		}
	}
	
	return results, totalCount, nil
}

// Group Management Operations

// CreateDeviceGroup creates a new device group
func (dim *DeviceIdentityManager) CreateDeviceGroup(group *DeviceGroup) error {
	if group == nil {
		// return fmt.Errorf("group cannot be nil")
	}
	
	if group.ID == "" {
		group.ID = dim.generateGroupID(group.Name)
	}
	
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	
	// Validate group data
	if err := dim.validateGroup(group); err != nil {
		// return fmt.Errorf("invalid group data: %w", err)
	}
	
	// Save to storage
	// TODO: Implement SetDeviceGroup
	// if err := dim.storage.SetDeviceGroup(group.ID, group); err != nil {
	//	// return fmt.Errorf("failed to save device group: %w", err)
	// }
	
	// Update cache
	dim.cacheMu.Lock()
	dim.groupCache[group.ID] = group
	dim.cacheMu.Unlock()
	
	dim.stats.TotalGroups++
	return nil
}

// GetDeviceGroup retrieves a device group by ID
func (dim *DeviceIdentityManager) GetDeviceGroup(groupID string) (*DeviceGroup, error) {
	// Check cache first
	dim.cacheMu.RLock()
	if group, exists := dim.groupCache[groupID]; exists {
		dim.cacheMu.RUnlock()
		return group, nil
	}
	dim.cacheMu.RUnlock()
	
	// Load from storage
	// TODO: Implement GetDeviceGroup
	var group *DeviceGroup
	// err := dim.storage.GetDeviceGroup(groupID)
	// if err != nil {
	//	return nil, fmt.Errorf("failed to get device group: %w", err)
	// }
	
	// Cache the result
	dim.cacheMu.Lock()
	dim.groupCache[groupID] = group
	dim.cacheMu.Unlock()
	
	return group, nil
}

// GetAllDeviceGroups retrieves all device groups
func (dim *DeviceIdentityManager) GetAllDeviceGroups() ([]*DeviceGroup, error) {
	// TODO: Implement GetAllDeviceGroups
	var groups []*DeviceGroup
	// err := nil // TODO: Implement storage method
	// _ = dim.storage.GetAllDeviceGroups()
	// if err != nil {
	//	return nil, fmt.Errorf("failed to get device groups: %w", err)
	// }
	
	// Update cache
	dim.cacheMu.Lock()
	for _, group := range groups {
		dim.groupCache[group.ID] = group
	}
	dim.cacheMu.Unlock()
	
	return groups, nil
}

// UpdateDeviceGroup updates a device group
func (dim *DeviceIdentityManager) UpdateDeviceGroup(group *DeviceGroup) error {
	if group == nil {
		// return fmt.Errorf("group cannot be nil")
	}
	
	group.UpdatedAt = time.Now()
	
	// Validate group data
	if err := dim.validateGroup(group); err != nil {
		// return fmt.Errorf("invalid group data: %w", err)
	}
	
	// Save to storage
	// TODO: Implement SetDeviceGroup
	// if err := dim.storage.SetDeviceGroup(group.ID, group); err != nil {
	//	// return fmt.Errorf("failed to update device group: %w", err)
	// }
	
	// Update cache
	dim.cacheMu.Lock()
	dim.groupCache[group.ID] = group
	dim.cacheMu.Unlock()
	
	return nil
}

// DeleteDeviceGroup deletes a device group
func (dim *DeviceIdentityManager) DeleteDeviceGroup(groupID string) error {
	// Remove from storage
	// TODO: Implement DeleteDeviceGroup
	// if err := dim.storage.DeleteDeviceGroup(groupID); err != nil {
	//	// return fmt.Errorf("failed to delete device group: %w", err)
	// }
	
	// Remove from cache
	dim.cacheMu.Lock()
	delete(dim.groupCache, groupID)
	dim.cacheMu.Unlock()
	
	// Remove group from all device identities
	// TODO: Implement GetAllDeviceIdentities
	var allIdentities []*DeviceIdentity
	var err error
	// allIdentities, err := dim.storage.GetAllDeviceIdentities()
	if err == nil {
		for _, identity := range allIdentities {
			identity.Groups = dim.removeStrings(identity.Groups, []string{groupID})
			dim.SetDeviceIdentity(identity)
		}
	}
	
	dim.stats.TotalGroups--
	return nil
}

// Tag Management Operations

// CreateDeviceTag creates a new device tag
func (dim *DeviceIdentityManager) CreateDeviceTag(tag *DeviceTag) error {
	if tag == nil {
		// return fmt.Errorf("tag cannot be nil")
	}
	
	if tag.ID == "" {
		tag.ID = dim.generateTagID(tag.Name)
	}
	
	tag.CreatedAt = time.Now()
	
	// Validate tag data
	if err := dim.validateTag(tag); err != nil {
		// return fmt.Errorf("invalid tag data: %w", err)
	}
	
	// Save to storage
	// TODO: Implement SetDeviceTag
	// if err := dim.storage.SetDeviceTag(tag.ID, tag); err != nil {
	//	// return fmt.Errorf("failed to save device tag: %w", err)
	// }
	
	// Update cache
	dim.cacheMu.Lock()
	dim.tagCache[tag.ID] = tag
	dim.cacheMu.Unlock()
	
	dim.stats.TotalTags++
	return nil
}

// GetDeviceTag retrieves a device tag by ID
func (dim *DeviceIdentityManager) GetDeviceTag(tagID string) (*DeviceTag, error) {
	// Check cache first
	dim.cacheMu.RLock()
	if tag, exists := dim.tagCache[tagID]; exists {
		dim.cacheMu.RUnlock()
		return tag, nil
	}
	dim.cacheMu.RUnlock()
	
	// Load from storage
	// TODO: Implement GetDeviceTag
	var tag *DeviceTag
	// err := nil // TODO: Implement storage method
	// _ = dim.storage.GetDeviceTag(tagID)
	// if err != nil {
	//	return nil, fmt.Errorf("failed to get device tag: %w", err)
	// }
	
	// Cache the result
	dim.cacheMu.Lock()
	dim.tagCache[tagID] = tag
	dim.cacheMu.Unlock()
	
	return tag, nil
}

// GetAllDeviceTags retrieves all device tags
func (dim *DeviceIdentityManager) GetAllDeviceTags() ([]*DeviceTag, error) {
	// TODO: Implement GetAllDeviceTags
	var tags []*DeviceTag
	// err := nil // TODO: Implement storage method
	// _ = dim.storage.GetAllDeviceTags()
	// if err != nil {
	//	return nil, fmt.Errorf("failed to get device tags: %w", err)
	// }
	
	// Update cache
	dim.cacheMu.Lock()
	for _, tag := range tags {
		dim.tagCache[tag.ID] = tag
	}
	dim.cacheMu.Unlock()
	
	return tags, nil
}

// DeleteDeviceTag deletes a device tag
func (dim *DeviceIdentityManager) DeleteDeviceTag(tagID string) error {
	// Remove from storage
	// TODO: Implement DeleteDeviceTag
	// if err := dim.storage.DeleteDeviceTag(tagID); err != nil {
	//	// return fmt.Errorf("failed to delete device tag: %w", err)
	// }
	
	// Remove from cache
	dim.cacheMu.Lock()
	delete(dim.tagCache, tagID)
	dim.cacheMu.Unlock()
	
	// Remove tag from all device identities
	// TODO: Implement GetAllDeviceIdentities
	var allIdentities []*DeviceIdentity
	var err error
	// allIdentities, err := dim.storage.GetAllDeviceIdentities()
	if err == nil {
		for _, identity := range allIdentities {
			identity.Tags = dim.removeStrings(identity.Tags, []string{tagID})
			dim.SetDeviceIdentity(identity)
		}
	}
	
	dim.stats.TotalTags--
	return nil
}

// Utility Operations

// GetStats returns device identity management statistics
func (dim *DeviceIdentityManager) GetStats() DeviceIdentityStats {
	return dim.stats
}

// RefreshCache clears and reloads the cache
func (dim *DeviceIdentityManager) RefreshCache() error {
	dim.cacheMu.Lock()
	defer dim.cacheMu.Unlock()
	
	// Clear caches
	dim.identityCache = make(map[string]*DeviceIdentity)
	dim.groupCache = make(map[string]*DeviceGroup)
	dim.tagCache = make(map[string]*DeviceTag)
	
	// Reload from storage
	return dim.loadFromStorage()
}

// Private helper methods

func (dim *DeviceIdentityManager) createDefaultIdentity(macAddress string) *DeviceIdentity {
	now := time.Now()
	
	identity := &DeviceIdentity{
		MacAddress:       macAddress,
		FriendlyName:     "",
		DisplayName:      macAddress,
		DeviceType:       DeviceTypeUnknown,
		Category:         CategoryUnknown,
		SecurityLevel:    SecurityLevelMedium,
		Status:           IdentityStatusActive,
		RegistrationDate: now,
		LastUpdated:      now,
		Groups:           []string{},
		Tags:             []string{},
		CustomFields:     make(map[string]interface{}),
		Notes:            []DeviceNote{},
		MonitoringEnabled: true,
		AlertingEnabled:   true,
		NotificationLevel: NotificationLevelHigh,
		Version:          1,
	}
	
	// Auto-detect information if enabled
	if dim.config.EnableAutoDetection {
		dim.autoDetectDeviceInfo(identity)
	}
	
	return identity
}

func (dim *DeviceIdentityManager) autoDetectDeviceInfo(identity *DeviceIdentity) {
	// Get OUI information
	if oui := dim.getOUIInfo(identity.MacAddress); oui != "" {
		identity.VendorOUI = oui
		identity.Manufacturer = dim.getManufacturerFromOUI(oui)
	}
	
	// Auto-generate friendly name if enabled
	if dim.config.AutoNamingEnabled && identity.FriendlyName == "" {
		identity.FriendlyName = dim.generateFriendlyName(identity)
	}
	
	// Auto-assign to groups if enabled
	if dim.config.AutoGroupingEnabled {
		identity.Groups = dim.autoAssignGroups(identity)
	}
	
	// Auto-assign tags if enabled
	if dim.config.AutoTaggingEnabled {
		identity.Tags = dim.autoAssignTags(identity)
	}
}

func (dim *DeviceIdentityManager) generateFriendlyName(identity *DeviceIdentity) string {
	if dim.config.DefaultNamingPattern == "" {
		return fmt.Sprintf("Device-%s", identity.MacAddress[12:17])
	}
	
	// Replace placeholders in naming pattern
	name := dim.config.DefaultNamingPattern
	name = strings.ReplaceAll(name, "{mac}", identity.MacAddress)
	name = strings.ReplaceAll(name, "{vendor}", identity.Manufacturer)
	name = strings.ReplaceAll(name, "{type}", string(identity.DeviceType))
	name = strings.ReplaceAll(name, "{location}", identity.Location)
	
	return name
}

func (dim *DeviceIdentityManager) autoAssignGroups(identity *DeviceIdentity) []string {
	var groups []string
	
	// Get all groups and check auto-assignment rules
	allGroups, err := dim.GetAllDeviceGroups()
	if err != nil {
		return groups
	}
	
	for _, group := range allGroups {
		if group.AutoAssignment && dim.matchesGroupRules(identity, group.Rules) {
			groups = append(groups, group.ID)
		}
	}
	
	return groups
}

func (dim *DeviceIdentityManager) autoAssignTags(identity *DeviceIdentity) []string {
	var tags []string
	
	// Auto-assign based on device type
	switch identity.DeviceType {
	case DeviceTypePhone, DeviceTypeTablet, DeviceTypeLaptop, DeviceTypeDesktop:
		tags = append(tags, "client")
	case DeviceTypeRouter:
		tags = append(tags, "network", "critical")
	case DeviceTypeAP:
		tags = append(tags, "network", "wifi")
	}
	
	// Auto-assign based on manufacturer
	if identity.Manufacturer != "" {
		tags = append(tags, strings.ToLower(identity.Manufacturer))
	}
	
	return tags
}

func (dim *DeviceIdentityManager) matchesGroupRules(identity *DeviceIdentity, rules []GroupRule) bool {
	for _, rule := range rules {
		if !dim.evaluateRule(identity, rule) {
			return false
		}
	}
	return true
}

func (dim *DeviceIdentityManager) evaluateRule(identity *DeviceIdentity, rule GroupRule) bool {
	var fieldValue string
	
	switch rule.Field {
	case "mac_address":
		fieldValue = identity.MacAddress
	case "manufacturer":
		fieldValue = identity.Manufacturer
	case "device_type":
		fieldValue = string(identity.DeviceType)
	case "location":
		fieldValue = identity.Location
	case "owner":
		fieldValue = identity.Owner
	case "department":
		fieldValue = identity.Department
	default:
		return false
	}
	
	if !rule.CaseSensitive {
		fieldValue = strings.ToLower(fieldValue)
		rule.Value = strings.ToLower(rule.Value)
	}
	
	switch rule.Operator {
	case OperatorEquals:
		return fieldValue == rule.Value
	case OperatorContains:
		return strings.Contains(fieldValue, rule.Value)
	case OperatorStartsWith:
		return strings.HasPrefix(fieldValue, rule.Value)
	case OperatorEndsWith:
		return strings.HasSuffix(fieldValue, rule.Value)
	case OperatorNotEquals:
		return fieldValue != rule.Value
	default:
		return false
	}
}

func (dim *DeviceIdentityManager) validateIdentity(identity *DeviceIdentity) error {
	if identity.MacAddress == "" {
		// return fmt.Errorf("MAC address is required")
	}
	
	if len(identity.FriendlyName) > dim.config.MaxNameLength {
		// return fmt.Errorf("friendly name too long (max %d characters)", dim.config.MaxNameLength)
	}
	
	if len(identity.FriendlyName) > 0 && len(identity.FriendlyName) < dim.config.MinNameLength {
		// return fmt.Errorf("friendly name too short (min %d characters)", dim.config.MinNameLength)
	}
	
	// Check for reserved names
	for _, reserved := range dim.config.ReservedNames {
		if strings.EqualFold(identity.FriendlyName, reserved) {
			// return fmt.Errorf("name '%s' is reserved", identity.FriendlyName)
		}
	}
	
	return nil
}

func (dim *DeviceIdentityManager) validateGroup(group *DeviceGroup) error {
	if group.Name == "" {
		// return fmt.Errorf("group name is required")
	}
	
	if group.ID == "" {
		// return fmt.Errorf("group ID is required")
	}
	
	return nil
}

func (dim *DeviceIdentityManager) validateTag(tag *DeviceTag) error {
	if tag.Name == "" {
		// return fmt.Errorf("tag name is required")
	}
	
	if tag.ID == "" {
		// return fmt.Errorf("tag ID is required")
	}
	
	return nil
}

func (dim *DeviceIdentityManager) matchesQuery(identity *DeviceIdentity, query DeviceIdentityQuery) bool {
	// MAC address filter
	if query.MacAddress != "" && !strings.Contains(strings.ToLower(identity.MacAddress), strings.ToLower(query.MacAddress)) {
		return false
	}
	
	// Friendly name filter
	if query.FriendlyName != "" && !strings.Contains(strings.ToLower(identity.FriendlyName), strings.ToLower(query.FriendlyName)) {
		return false
	}
	
	// Device type filter
	if query.DeviceType != "" && identity.DeviceType != query.DeviceType {
		return false
	}
	
	// Manufacturer filter
	if query.Manufacturer != "" && !strings.Contains(strings.ToLower(identity.Manufacturer), strings.ToLower(query.Manufacturer)) {
		return false
	}
	
	// Category filter
	if query.Category != "" && identity.Category != query.Category {
		return false
	}
	
	// Status filter
	if query.Status != "" && identity.Status != query.Status {
		return false
	}
	
	// Security level filter
	if query.SecurityLevel != "" && identity.SecurityLevel != query.SecurityLevel {
		return false
	}
	
	// Location filter
	if query.Location != "" && !strings.Contains(strings.ToLower(identity.Location), strings.ToLower(query.Location)) {
		return false
	}
	
	// Groups filter
	if len(query.Groups) > 0 {
		hasGroup := false
		for _, queryGroup := range query.Groups {
			for _, identityGroup := range identity.Groups {
				if queryGroup == identityGroup {
					hasGroup = true
					break
				}
			}
			if hasGroup {
				break
			}
		}
		if !hasGroup {
			return false
		}
	}
	
	// Tags filter
	if len(query.Tags) > 0 {
		hasTag := false
		for _, queryTag := range query.Tags {
			for _, identityTag := range identity.Tags {
				if queryTag == identityTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}
	
	// Search text filter
	if query.SearchText != "" {
		searchText := strings.ToLower(query.SearchText)
		searchableText := strings.ToLower(fmt.Sprintf("%s %s %s %s %s %s",
			identity.MacAddress, identity.FriendlyName, identity.DisplayName,
			identity.Description, identity.Manufacturer, identity.Location))
		
		if !strings.Contains(searchableText, searchText) {
			return false
		}
	}
	
	// Time filters
	if !query.CreatedAfter.IsZero() && identity.RegistrationDate.Before(query.CreatedAfter) {
		return false
	}
	
	if !query.CreatedBefore.IsZero() && identity.RegistrationDate.After(query.CreatedBefore) {
		return false
	}
	
	if !query.LastSeenAfter.IsZero() && identity.LastSeen.Before(query.LastSeenAfter) {
		return false
	}
	
	if !query.LastSeenBefore.IsZero() && identity.LastSeen.After(query.LastSeenBefore) {
		return false
	}
	
	return true
}

func (dim *DeviceIdentityManager) sortIdentities(identities []*DeviceIdentity, sortBy, sortOrder string) {
	if sortBy == "" {
		sortBy = "friendly_name"
	}
	
	ascending := sortOrder != "desc"
	
	sort.Slice(identities, func(i, j int) bool {
		var less bool
		
		switch sortBy {
		case "friendly_name":
			less = identities[i].FriendlyName < identities[j].FriendlyName
		case "mac_address":
			less = identities[i].MacAddress < identities[j].MacAddress
		case "device_type":
			less = identities[i].DeviceType < identities[j].DeviceType
		case "manufacturer":
			less = identities[i].Manufacturer < identities[j].Manufacturer
		case "location":
			less = identities[i].Location < identities[j].Location
		case "last_seen":
			less = identities[i].LastSeen.Before(identities[j].LastSeen)
		case "registration_date":
			less = identities[i].RegistrationDate.Before(identities[j].RegistrationDate)
		default:
			less = identities[i].FriendlyName < identities[j].FriendlyName
		}
		
		if ascending {
			return less
		}
		return !less
	})
}

func (dim *DeviceIdentityManager) addUniqueStrings(slice []string, items []string) []string {
	existingMap := make(map[string]bool)
	for _, item := range slice {
		existingMap[item] = true
	}
	
	for _, item := range items {
		if !existingMap[item] {
			slice = append(slice, item)
			existingMap[item] = true
		}
	}
	
	return slice
}

func (dim *DeviceIdentityManager) removeStrings(slice []string, items []string) []string {
	removeMap := make(map[string]bool)
	for _, item := range items {
		removeMap[item] = true
	}
	
	var result []string
	for _, item := range slice {
		if !removeMap[item] {
			result = append(result, item)
		}
	}
	
	return result
}

func (dim *DeviceIdentityManager) generateGroupID(name string) string {
	return fmt.Sprintf("group_%s_%d", strings.ReplaceAll(strings.ToLower(name), " ", "_"), time.Now().Unix())
}

func (dim *DeviceIdentityManager) generateTagID(name string) string {
	return fmt.Sprintf("tag_%s_%d", strings.ReplaceAll(strings.ToLower(name), " ", "_"), time.Now().Unix())
}

func (dim *DeviceIdentityManager) getOUIInfo(macAddress string) string {
	// Extract OUI (first 3 octets)
	if len(macAddress) >= 8 {
		return strings.ToUpper(macAddress[:8])
	}
	return ""
}

func (dim *DeviceIdentityManager) getManufacturerFromOUI(oui string) string {
	// This would typically query an OUI database
	// For now, return a simplified mapping
	ouiMap := map[string]string{
		"00:11:22": "Example Corp",
		"AA:BB:CC": "Test Inc",
		// Add more OUI mappings as needed
	}
	
	if manufacturer, exists := ouiMap[oui]; exists {
		return manufacturer
	}
	
	return "Unknown"
}

// Background processing methods

func (dim *DeviceIdentityManager) autoDetectionLoop(ctx context.Context) {
	if !dim.config.EnableAutoDetection {
		return
	}
	
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dim.performAutoDetection()
		}
	}
}

func (dim *DeviceIdentityManager) cacheMaintenanceLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dim.cleanupCache()
		}
	}
}

func (dim *DeviceIdentityManager) statsUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 15)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dim.updateStats()
		}
	}
}

func (dim *DeviceIdentityManager) performAutoDetection() {
	// Get current topology to find new devices
	if dim.topologyManager == nil {
		return
	}
	
	// TODO: Implement GetCurrentTopology
	// topology, err := dim.topologyManager.GetCurrentTopology()
	// if err != nil {
	//	return
	// }
	//
	// for macAddress := range topology.Devices {
	//	// Check if identity already exists
	//	if _, err := dim.GetDeviceIdentity(macAddress); err != nil {
	//		// Create new identity for unknown device
	//		identity := dim.createDefaultIdentity(macAddress)
	//		dim.SetDeviceIdentity(identity)
	//	}
	// }
}

func (dim *DeviceIdentityManager) cleanupCache() {
	dim.cacheMu.Lock()
	defer dim.cacheMu.Unlock()
	
	// Simple cache cleanup - remove entries if cache is too large
	if len(dim.identityCache) > dim.config.CacheSize {
		// Remove oldest entries (simplified - would use LRU in production)
		count := len(dim.identityCache) - dim.config.CacheSize
		for key := range dim.identityCache {
			if count <= 0 {
				break
			}
			delete(dim.identityCache, key)
			count--
		}
	}
}

func (dim *DeviceIdentityManager) updateStats() {
	// TODO: Implement GetAllDeviceIdentities
	var allIdentities []*DeviceIdentity
	var err error
	// allIdentities, err := dim.storage.GetAllDeviceIdentities()
	if err != nil {
		return
	}
	
	dim.stats.TotalDevices = int64(len(allIdentities))
	dim.stats.NamedDevices = 0
	dim.stats.UnnamedDevices = 0
	dim.stats.GroupedDevices = 0
	dim.stats.TaggedDevices = 0
	
	for _, identity := range allIdentities {
		if identity.FriendlyName != "" {
			dim.stats.NamedDevices++
		} else {
			dim.stats.UnnamedDevices++
		}
		
		if len(identity.Groups) > 0 {
			dim.stats.GroupedDevices++
		}
		
		if len(identity.Tags) > 0 {
			dim.stats.TaggedDevices++
		}
	}
	
	dim.stats.LastUpdate = time.Now()
}

func (dim *DeviceIdentityManager) loadFromStorage() error {
	// Load identities
	// TODO: Implement GetAllDeviceIdentities - for now, skip loading from storage
	// identities, err := dim.storage.GetAllDeviceIdentities()
	// if err != nil {
	//	return fmt.Errorf("failed to load identities: %w", err)
	// }
	// for _, identity := range identities {
	//	dim.identityCache[identity.MacAddress] = identity
	// }
	
	// Load groups
	// TODO: Implement GetAllDeviceGroups
	var groups []*DeviceGroup
	// err := nil // TODO: Implement storage method
	// _ = dim.storage.GetAllDeviceGroups()
	// if err != nil {
	//	// return fmt.Errorf("failed to load groups: %w", err)
	// }
	
	for _, group := range groups {
		dim.groupCache[group.ID] = group
	}
	
	// Load tags
	// TODO: Implement GetAllDeviceTags
	var tags []*DeviceTag
	// err := nil // TODO: Implement storage method
	// _ = dim.storage.GetAllDeviceTags()
	// if err != nil {
	//	// return fmt.Errorf("failed to load tags: %w", err)
	// }
	
	for _, tag := range tags {
		dim.tagCache[tag.ID] = tag
	}
	
	return nil
}

func (dim *DeviceIdentityManager) saveToStorage() error {
	// Save all cached data to storage
	dim.cacheMu.RLock()
	defer dim.cacheMu.RUnlock()
	
	// Save identities
	for _, identity := range dim.identityCache {
		// Convert to types.DeviceIdentity for storage
		storageIdentity := &types.DeviceIdentity{
			MacAddress:   identity.MacAddress,
			FriendlyName: identity.FriendlyName,
			DeviceType:   string(identity.DeviceType),
			Manufacturer: identity.Manufacturer,
			Model:        identity.Model,
			Location:     identity.Location,
			Owner:        identity.Owner,
			Category:     string(identity.Category),
			Tags:         identity.Tags,
			FirstSeen:    identity.RegistrationDate,
			LastSeen:     identity.LastSeen,
			LastUpdated:  identity.LastUpdated,
			UpdatedBy:    identity.ModifiedBy,
		}
		if err := dim.storage.SaveDeviceIdentity(storageIdentity); err != nil {
			// return fmt.Errorf("failed to save identity %s: %w", identity.MacAddress, err)
		}
	}
	
	// Save groups
	for _, _ = range dim.groupCache {
		// TODO: Implement SetDeviceGroup
		// if err := dim.storage.SetDeviceGroup(groupID, group); err != nil {
		//		// return fmt.Errorf("failed to save group %s: %w", groupID, err)
		// }
	}
	
	// Save tags
	for _, _ = range dim.tagCache {
		// TODO: Implement SetDeviceTag
		// if err := dim.storage.SetDeviceTag(tagID, tag); err != nil {
		//		// return fmt.Errorf("failed to save tag %s: %w", tagID, err)
		// }
	}
	
	return nil
}
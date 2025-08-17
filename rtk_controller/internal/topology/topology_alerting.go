package topology

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"rtk_controller/internal/storage"
)

// TopologyAlertingSystem manages alerts for topology changes and network issues
type TopologyAlertingSystem struct {
	// Core components
	topologyManager      *Manager
	qualityMonitor      *ConnectionQualityMonitor
	roamingDetector     *RoamingDetector
	realtimeUpdater     *RealtimeTopologyUpdater
	connectionTracker   *ConnectionHistoryTracker
	storage             *storage.TopologyStorage
	identityStorage     *storage.IdentityStorage
	
	// Alert management
	activeAlerts        map[string]*TopologyAlert
	alertHistory        []TopologyAlert
	alertRules          []AlertRule
	suppressions        map[string]*AlertSuppression
	escalations         map[string]*AlertEscalation
	mu                  sync.RWMutex
	
	// Configuration
	config AlertingConfig
	
	// Background processing
	running bool
	cancel  context.CancelFunc
	
	// Statistics
	stats AlertingStats
}

// TopologyAlert represents a topology-related alert
type TopologyAlert struct {
	ID                  string
	Type                TopologyAlertType
	Severity            AlertSeverity
	Status              AlertStatus
	Category            AlertCategory
	
	// Source information
	SourceType          AlertSourceType
	DeviceID            string
	MacAddress          string
	FriendlyName        string
	Location            string
	
	// Alert details
	Title               string
	Description         string
	Message             string
	Impact              AlertImpact
	Urgency             AlertUrgency
	
	// Timing
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastOccurrence      time.Time
	ResolvedAt          time.Time
	AcknowledgedAt      time.Time
	
	// Escalation and handling
	Frequency           int
	Escalated           bool
	EscalationLevel     int
	AssignedTo          string
	AcknowledgedBy      string
	ResolvedBy          string
	
	// Context and analysis
	Context             AlertContext
	TriggerConditions   []TriggerCondition
	AffectedDevices     []string
	RelatedAlerts       []string
	RootCause           string
	RecommendedActions  []string
	
	// Notification status
	NotificationsSent   []NotificationRecord
	SuppressedUntil     time.Time
	
	// Metadata
	Tags                []string
	CustomFields        map[string]interface{}
}

// AlertRule defines conditions for generating alerts
type AlertRule struct {
	ID                  string
	Name                string
	Description         string
	Enabled             bool
	Category            AlertCategory
	
	// Conditions
	Conditions          []RuleCondition
	ConditionLogic      ConditionLogic // AND, OR
	
	// Alert properties
	AlertType           TopologyAlertType
	Severity            AlertSeverity
	Priority            AlertPriority
	
	// Timing and frequency
	Cooldown            time.Duration
	MaxFrequency        int           // Max alerts per hour
	TimeWindow          time.Duration // Window for frequency calculation
	
	// Actions
	Actions             []AlertActionType
	AutoResolve         bool
	AutoResolveDelay    time.Duration
	
	// Escalation
	EscalationChain     []EscalationStep
	
	// Filtering
	DeviceFilter        DeviceFilter
	TimeFilter          TimeFilter
	
	// Metadata
	CreatedBy           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// RuleCondition defines a condition within an alert rule
type RuleCondition struct {
	Type                AlertConditionType
	Field               string
	Operator            ComparisonOperator
	Value               interface{}
	Threshold           float64
	TimeWindow          time.Duration
	AggregationFunction AggregationType
	Weight              float64
}

// AlertSuppression represents a suppression rule
type AlertSuppression struct {
	ID                  string
	Name                string
	Description         string
	Active              bool
	
	// Criteria
	AlertTypes          []TopologyAlertType
	DeviceIDs           []string
	MacAddresses        []string
	Categories          []AlertCategory
	
	// Timing
	StartTime           time.Time
	EndTime             time.Time
	Duration            time.Duration
	Recurring           bool
	RecurrencePattern   string
	
	// Metadata
	CreatedBy           string
	Reason              string
}

// AlertEscalation manages alert escalation
type AlertEscalation struct {
	AlertID             string
	CurrentLevel        int
	EscalationChain     []EscalationStep
	LastEscalation      time.Time
	NextEscalation      time.Time
	MaxLevel            int
	Completed           bool
}

// EscalationStep defines an escalation step
type EscalationStep struct {
	Level               int
	Delay               time.Duration
	NotificationTargets []NotificationTarget
	Actions             []EscalationAction
	Conditions          []EscalationCondition
}

// NotificationTarget defines where to send notifications
type NotificationTarget struct {
	Type                NotificationType
	Target              string // email, webhook URL, etc.
	Priority            NotificationPriority
	Template            string
	Enabled             bool
}

// NotificationRecord tracks sent notifications
type NotificationRecord struct {
	Type                NotificationType
	Target              string
	SentAt              time.Time
	Status              NotificationStatus
	Error               string
	Response            string
}

// AlertContext provides context about the alert
type AlertContext struct {
	NetworkState        NetworkStateContext
	DeviceContext       DeviceStateContext
	QualityContext      QualityStateContext
	RoamingContext      RoamingStateContext
	SystemContext       SystemStateContext
	HistoricalContext   HistoricalStateContext
}

// TriggerCondition describes what triggered the alert
type TriggerCondition struct {
	Type                string
	Field               string
	ExpectedValue       interface{}
	ActualValue         interface{}
	Threshold           float64
	ComparisonResult    string
	Confidence          float64
}

// DeviceFilter filters alerts by device characteristics
type DeviceFilter struct {
	DeviceIDs           []string
	MacAddresses        []string
	DeviceTypes         []string
	Locations           []string
	Tags                []string
	IncludePattern      string
	ExcludePattern      string
}

// TimeFilter filters alerts by time
type TimeFilter struct {
	TimeRanges          []AlertTimeRange
	DaysOfWeek          []time.Weekday
	ExcludedDates       []time.Time
	Timezone            string
}

// TimeRange defines a time range
type AlertTimeRange struct {
	StartTime           string // HH:MM format
	EndTime             string // HH:MM format
}

// Enums for alerting system

type TopologyAlertType string
const (
	// Device alerts
	AlertDeviceAdded            TopologyAlertType = "device_added"
	AlertDeviceRemoved          TopologyAlertType = "device_removed"
	AlertDeviceOffline          TopologyAlertType = "device_offline"
	AlertDeviceOnline           TopologyAlertType = "device_online"
	AlertDeviceMisconfigured    TopologyAlertType = "device_misconfigured"
	AlertDevicePerformanceIssue TopologyAlertType = "device_performance_issue"
	
	// Connection alerts
	AlertConnectionLost         TopologyAlertType = "connection_lost"
	AlertConnectionEstablished  TopologyAlertType = "connection_established"
	AlertConnectionDegraded     TopologyAlertType = "connection_degraded"
	// AlertConnectionUnstable already defined in connection_quality_monitor.go
	
	// Network topology alerts
	AlertTopologyChanged        TopologyAlertType = "topology_changed"
	AlertNetworkSplit           TopologyAlertType = "network_split"
	AlertNetworkMerged          TopologyAlertType = "network_merged"
	AlertLoopDetected           TopologyAlertType = "loop_detected"
	AlertSinglePointOfFailure   TopologyAlertType = "single_point_of_failure"
	
	// Roaming alerts
	AlertExcessiveRoaming       TopologyAlertType = "excessive_roaming"
	AlertRoamingFailure         TopologyAlertType = "roaming_failure"
	AlertPingPongRoaming        TopologyAlertType = "ping_pong_roaming"
	AlertStuckClient            TopologyAlertType = "stuck_client"
	
	// Quality alerts
	AlertQualityDegraded        TopologyAlertType = "quality_degraded"
	// AlertSignalWeak already defined in connection_quality_monitor.go
	// AlertHighLatency already defined in connection_quality_monitor.go
	AlertPacketLoss             TopologyAlertType = "packet_loss"
	AlertBandwidthSaturation    TopologyAlertType = "bandwidth_saturation"
	
	// Security alerts
	AlertUnauthorizedDevice     TopologyAlertType = "unauthorized_device"
	AlertAnomalousTraffic       TopologyAlertType = "anomalous_traffic"
	AlertSuspiciousActivity     TopologyAlertType = "suspicious_activity"
	
	// System alerts
	AlertSystemOverloaded       TopologyAlertType = "system_overloaded"
	AlertMonitoringFailure      TopologyAlertType = "monitoring_failure"
	AlertConfigurationError     TopologyAlertType = "configuration_error"
)

type AlertStatus string
const (
	StatusOpen        AlertStatus = "open"
	StatusAcknowledged AlertStatus = "acknowledged"
	StatusInProgress  AlertStatus = "in_progress"
	// StatusResolved moved to constants.go
	StatusClosed      AlertStatus = "closed"
	StatusSuppressed  AlertStatus = "suppressed"
)

type AlertCategory string
const (
	CategoryAvailability  AlertCategory = "availability"
	// CategoryPerformance defined in constants.go
	// CategorySecurity defined in constants.go
	// CategoryConfiguration defined in network_diagnostics.go
	CategoryCapacity      AlertCategory = "capacity"
	CategoryCompliance    AlertCategory = "compliance"
)

type AlertSourceType string
const (
	SourceTopologyManager    AlertSourceType = "topology_manager"
	SourceQualityMonitor     AlertSourceType = "quality_monitor"
	SourceRoamingDetector    AlertSourceType = "roaming_detector"
	SourceConnectionTracker  AlertSourceType = "connection_tracker"
	SourceManualTrigger      AlertSourceType = "manual_trigger"
	SourceScheduledCheck     AlertSourceType = "scheduled_check"
)

type AlertImpact string
const (
	// ImpactNone already defined in roaming_anomaly_detector.go
	// ImpactLow moved to constants.go
	// ImpactMedium moved to constants.go
	// ImpactHigh moved to constants.go
	// ImpactCritical moved to constants.go
)

type AlertUrgency string
const (
	UrgencyLow        AlertUrgency = "low"
	UrgencyMedium     AlertUrgency = "medium"
	UrgencyHigh       AlertUrgency = "high"
	UrgencyImmediate  AlertUrgency = "immediate"
)

type AlertPriority string
const (
	PriorityP1 AlertPriority = "P1" // Critical
	PriorityP2 AlertPriority = "P2" // High
	PriorityP3 AlertPriority = "P3" // Medium
	PriorityP4 AlertPriority = "P4" // Low
	PriorityP5 AlertPriority = "P5" // Informational
)

type AlertConditionType string
const (
	ConditionMetricThreshold    AlertConditionType = "metric_threshold"
	ConditionEventOccurrence    AlertConditionType = "event_occurrence"
	ConditionPatternDetection   AlertConditionType = "pattern_detection"
	ConditionAnomalyDetection   AlertConditionType = "anomaly_detection"
	ConditionStatusChange       AlertConditionType = "status_change"
	ConditionRateOfChange       AlertConditionType = "rate_of_change"
	ConditionCorrelation        AlertConditionType = "correlation"
)

type ComparisonOperator string
const (
	// OperatorEquals defined in device_identity_manager.go
// 	OperatorNotEquals           ComparisonOperator = "not_equals"
	OperatorGreaterThan         ComparisonOperator = "greater_than"
	OperatorGreaterThanOrEqual  ComparisonOperator = "greater_than_or_equal"
	OperatorLessThan            ComparisonOperator = "less_than"
	OperatorLessThanOrEqual     ComparisonOperator = "less_than_or_equal"
// 	OperatorContains            ComparisonOperator = "contains"
	OperatorNotContains         ComparisonOperator = "not_contains"
	OperatorMatches             ComparisonOperator = "matches"
	OperatorNotMatches          ComparisonOperator = "not_matches"
)

type ConditionLogic string
const (
	LogicAND ConditionLogic = "AND"
	LogicOR  ConditionLogic = "OR"
)

type AggregationType string
const (
	AggregationAverage AggregationType = "average"
	AggregationSum     AggregationType = "sum"
	AggregationMin     AggregationType = "min"
	AggregationMax     AggregationType = "max"
	AggregationCount   AggregationType = "count"
	AggregationRate    AggregationType = "rate"
)

type NotificationType string
const (
	NotificationEmail    NotificationType = "email"
	NotificationSMS      NotificationType = "sms"
	NotificationWebhook  NotificationType = "webhook"
	NotificationSlack    NotificationType = "slack"
	NotificationPagerDuty NotificationType = "pagerduty"
	NotificationSNMP     NotificationType = "snmp"
	NotificationSyslog   NotificationType = "syslog"
	NotificationMQTT     NotificationType = "mqtt"
)

type NotificationPriority string
const (
	NotificationLow    NotificationPriority = "low"
	NotificationNormal NotificationPriority = "normal"
	NotificationHigh   NotificationPriority = "high"
	NotificationUrgent NotificationPriority = "urgent"
)

type NotificationStatus string
const (
	NotificationPending   NotificationStatus = "pending"
	NotificationSent      NotificationStatus = "sent"
	NotificationDelivered NotificationStatus = "delivered"
	NotificationFailed    NotificationStatus = "failed"
	NotificationRetrying  NotificationStatus = "retrying"
)

type AlertActionType string
const (
	ActionNotify           AlertActionType = "notify"
	ActionEscalate         AlertActionType = "escalate"
	ActionExecuteScript    AlertActionType = "execute_script"
	ActionCreateTicket     AlertActionType = "create_ticket"
	ActionSendSNMP         AlertActionType = "send_snmp"
	ActionLogEvent         AlertActionType = "log_event"
	ActionTriggerWorkflow  AlertActionType = "trigger_workflow"
)

type EscalationAction string
const (
	EscalationNotifyManager     EscalationAction = "notify_manager"
	EscalationCreateIncident    EscalationAction = "create_incident"
	EscalationPageOnCall        EscalationAction = "page_on_call"
	EscalationExecuteRunbook    EscalationAction = "execute_runbook"
)

type EscalationCondition string
const (
	EscalationTimeElapsed       EscalationCondition = "time_elapsed"
	EscalationNotAcknowledged   EscalationCondition = "not_acknowledged"
	EscalationNotResolved       EscalationCondition = "not_resolved"
	EscalationSeverityIncreased EscalationCondition = "severity_increased"
)

// Context structures
type NetworkStateContext struct {
	TotalDevices        int
	OnlineDevices       int
	OfflineDevices      int
	NetworkLoad         float64
	TopologyComplexity  float64
	ConnectivityHealth  float64
}

type DeviceStateContext struct {
	DeviceType          string
	DeviceStatus        string
	DeviceUptime        time.Duration
	DeviceLoad          float64
	DeviceErrors        int
	DeviceCapabilities  []string
}

type QualityStateContext struct {
	AverageQuality      float64
	QualityTrend        string
	AffectedMetrics     []string
	SeverityDistribution map[string]int
}

type RoamingStateContext struct {
	RoamingFrequency    float64
	AnomalousRoaming    int
	RoamingPatterns     []string
	AffectedClients     int
}

type SystemStateContext struct {
	SystemLoad          float64
	MemoryUsage         float64
	DiskUsage           float64
	NetworkUtilization  float64
	ProcessingErrors    int
}

type HistoricalStateContext struct {
	SimilarAlerts       int
	PatternFrequency    float64
	SeasonalTrends      map[string]float64
	ResolutionHistory   []ResolutionRecord
}

type ResolutionRecord struct {
	AlertType           TopologyAlertType
	ResolutionTime      time.Duration
	ResolutionMethod    string
	Effectiveness       float64
}

// AlertingConfig holds alerting system configuration
type AlertingConfig struct {
	// Processing intervals
	AlertProcessingInterval    time.Duration
	EscalationCheckInterval    time.Duration
	NotificationRetryInterval  time.Duration
	AlertCleanupInterval       time.Duration
	
	// Alert management
	MaxActiveAlerts            int
	AlertHistoryRetention      time.Duration
	DefaultAlertTimeout        time.Duration
	AutoResolveEnabled         bool
	
	// Notification settings
	NotificationTimeout        time.Duration
	NotificationRetries        int
	NotificationBatchSize      int
	NotificationRateLimit      int // per minute
	
	// Escalation settings
	DefaultEscalationDelay     time.Duration
	MaxEscalationLevels        int
	EscalationEnabled          bool
	AutoEscalationEnabled      bool
	
	// Suppression settings
	SuppressionEnabled         bool
	MaintenanceWindowSuppress  bool
	DuplicateAlertSuppression  bool
	DuplicateTimeWindow        time.Duration
	
	// Correlation settings
	AlertCorrelationEnabled    bool
	CorrelationTimeWindow      time.Duration
	CorrelationDistance        float64
	
	// Performance settings
	BatchProcessingSize        int
	ConcurrentNotifications    int
	AlertQueueSize            int
	
	// Integration settings
	WebhookTimeout            time.Duration
	WebhookRetries            int
	EmailEnabled              bool
	SlackEnabled              bool
	PagerDutyEnabled          bool
}

// AlertingStats holds alerting system statistics
type AlertingStats struct {
	TotalAlerts           int64
	ActiveAlerts          int64
	ResolvedAlerts        int64
	EscalatedAlerts       int64
	SuppressedAlerts      int64
	NotificationsSent     int64
	NotificationsFailed   int64
	AverageResolutionTime time.Duration
	AlertsByType          map[TopologyAlertType]int64
	AlertsBySeverity      map[AlertSeverity]int64
	LastProcessingTime    time.Time
	ProcessingErrors      int64
}

// NewTopologyAlertingSystem creates a new topology alerting system
func NewTopologyAlertingSystem(
	topologyManager *Manager,
	qualityMonitor *ConnectionQualityMonitor,
	roamingDetector *RoamingDetector,
	realtimeUpdater *RealtimeTopologyUpdater,
	connectionTracker *ConnectionHistoryTracker,
	storage *storage.TopologyStorage,
	identityStorage *storage.IdentityStorage,
	config AlertingConfig,
) *TopologyAlertingSystem {
	
	return &TopologyAlertingSystem{
		topologyManager:   topologyManager,
		qualityMonitor:   qualityMonitor,
		roamingDetector:  roamingDetector,
		realtimeUpdater:  realtimeUpdater,
		connectionTracker: connectionTracker,
		storage:          storage,
		identityStorage:  identityStorage,
		activeAlerts:     make(map[string]*TopologyAlert),
		alertHistory:     []TopologyAlert{},
		alertRules:       []AlertRule{},
		suppressions:     make(map[string]*AlertSuppression),
		escalations:      make(map[string]*AlertEscalation),
		config:          config,
		stats:           AlertingStats{
			AlertsByType:     make(map[TopologyAlertType]int64),
			AlertsBySeverity: make(map[AlertSeverity]int64),
		},
	}
}

// Start begins the alerting system
func (tas *TopologyAlertingSystem) Start() error {
	tas.mu.Lock()
	defer tas.mu.Unlock()
	
	if tas.running {
		return fmt.Errorf("topology alerting system is already running")
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	tas.cancel = cancel
	tas.running = true
	
	log.Printf("Starting topology alerting system")
	
	// Initialize default alert rules
	tas.initializeDefaultRules()
	
	// Start background processing
	go tas.alertProcessingLoop(ctx)
	go tas.escalationProcessingLoop(ctx)
	go tas.notificationProcessingLoop(ctx)
	go tas.cleanupLoop(ctx)
	
	// Subscribe to real-time updates
	if tas.realtimeUpdater != nil {
		tas.subscribeToTopologyUpdates()
	}
	
	return nil
}

// Stop stops the alerting system
func (tas *TopologyAlertingSystem) Stop() error {
	tas.mu.Lock()
	defer tas.mu.Unlock()
	
	if !tas.running {
		return fmt.Errorf("topology alerting system is not running")
	}
	
	tas.cancel()
	tas.running = false
	
	log.Printf("Topology alerting system stopped")
	return nil
}

// CreateAlert creates a new topology alert
func (tas *TopologyAlertingSystem) CreateAlert(
	alertType TopologyAlertType,
	severity AlertSeverity,
	deviceID string,
	macAddress string,
	title string,
	description string,
	context AlertContext,
) (*TopologyAlert, error) {
	
	tas.mu.Lock()
	defer tas.mu.Unlock()
	
	// Check for duplicates
	if tas.config.DuplicateAlertSuppression {
		if existing := tas.findDuplicateAlert(alertType, deviceID, macAddress); existing != nil {
			// Update existing alert
			existing.Frequency++
			existing.LastOccurrence = time.Now()
			existing.UpdatedAt = time.Now()
			return existing, nil
		}
	}
	
	// Check if alert is suppressed
	if tas.isAlertSuppressed(alertType, deviceID, macAddress) {
		tas.stats.SuppressedAlerts++
		return nil, fmt.Errorf("alert is suppressed")
	}
	
	// Get friendly name
	friendlyName := macAddress
	if macAddress != "" {
		if identity, err := tas.identityStorage.GetDeviceIdentity(macAddress); err == nil {
			friendlyName = identity.FriendlyName
		}
	}
	
	// Create alert
	now := time.Now()
	alert := &TopologyAlert{
		ID:                fmt.Sprintf("alert_%d_%s", now.UnixMilli(), deviceID),
		Type:              alertType,
		Severity:          severity,
		Status:            StatusOpen,
		Category:          tas.categorizeAlert(alertType),
		SourceType:        SourceTopologyManager,
		DeviceID:          deviceID,
		MacAddress:        macAddress,
		FriendlyName:      friendlyName,
		Title:             title,
		Description:       description,
		Message:           tas.formatAlertMessage(alertType, title, description),
		Impact:            tas.calculateImpact(severity, alertType),
		Urgency:           tas.calculateUrgency(severity, alertType),
		CreatedAt:         now,
		UpdatedAt:         now,
		LastOccurrence:    now,
		Frequency:         1,
		Context:           context,
		TriggerConditions: []TriggerCondition{},
		AffectedDevices:   []string{deviceID},
		RelatedAlerts:     []string{},
		RecommendedActions: tas.generateRecommendations(alertType, severity),
		NotificationsSent: []NotificationRecord{},
		Tags:              []string{},
		CustomFields:      make(map[string]interface{}),
	}
	
	// Store alert
	tas.activeAlerts[alert.ID] = alert
	tas.alertHistory = append(tas.alertHistory, *alert)
	
	// Update statistics
	tas.stats.TotalAlerts++
	tas.stats.ActiveAlerts++
	tas.stats.AlertsByType[alertType]++
	tas.stats.AlertsBySeverity[severity]++
	
	log.Printf("Created alert: %s - %s (%s)", alert.ID, alert.Title, alert.Severity)
	
	// Process alert actions
	go tas.processAlertActions(alert)
	
	// Publish to real-time updates
	if tas.realtimeUpdater != nil {
		updateEvent := TopologyUpdateEvent{
			Type:      EventAnomalyDetected,
			DeviceID:  deviceID,
			Timestamp: now,
			Priority:  PriorityHigh,
			Metadata: map[string]interface{}{
				"alert_id":       alert.ID,
				"alert_type":     alertType,
				"alert_severity": severity,
				"alert_title":    title,
			},
		}
		
		go tas.realtimeUpdater.PublishUpdate(updateEvent)
	}
	
	return alert, nil
}

// ResolveAlert resolves an active alert
func (tas *TopologyAlertingSystem) ResolveAlert(alertID string, resolvedBy string, reason string) error {
	tas.mu.Lock()
	defer tas.mu.Unlock()
	
	alert, exists := tas.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}
	
	now := time.Now()
	alert.Status = StatusResolved
	alert.ResolvedAt = now
	alert.ResolvedBy = resolvedBy
	alert.UpdatedAt = now
	
	// Add resolution reason to custom fields
	if alert.CustomFields == nil {
		alert.CustomFields = make(map[string]interface{})
	}
	alert.CustomFields["resolution_reason"] = reason
	
	// Remove from active alerts
	delete(tas.activeAlerts, alertID)
	
	// Update statistics
	tas.stats.ActiveAlerts--
	tas.stats.ResolvedAlerts++
	
	// Calculate resolution time
	resolutionTime := alert.ResolvedAt.Sub(alert.CreatedAt)
	if tas.stats.AverageResolutionTime == 0 {
		tas.stats.AverageResolutionTime = resolutionTime
	} else {
		tas.stats.AverageResolutionTime = (tas.stats.AverageResolutionTime + resolutionTime) / 2
	}
	
	log.Printf("Resolved alert: %s by %s (reason: %s)", alertID, resolvedBy, reason)
	
	// Remove escalation if exists
	if escalation, exists := tas.escalations[alertID]; exists {
		escalation.Completed = true
	}
	
	return nil
}

// AcknowledgeAlert acknowledges an alert
func (tas *TopologyAlertingSystem) AcknowledgeAlert(alertID string, acknowledgedBy string) error {
	tas.mu.Lock()
	defer tas.mu.Unlock()
	
	alert, exists := tas.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}
	
	now := time.Now()
	alert.Status = StatusAcknowledged
	alert.AcknowledgedAt = now
	alert.AcknowledgedBy = acknowledgedBy
	alert.UpdatedAt = now
	
	log.Printf("Acknowledged alert: %s by %s", alertID, acknowledgedBy)
	return nil
}

// GetActiveAlerts returns all active alerts
func (tas *TopologyAlertingSystem) GetActiveAlerts() []*TopologyAlert {
	tas.mu.RLock()
	defer tas.mu.RUnlock()
	
	alerts := make([]*TopologyAlert, 0, len(tas.activeAlerts))
	for _, alert := range tas.activeAlerts {
		alertCopy := *alert
		alerts = append(alerts, &alertCopy)
	}
	
	return alerts
}

// GetAlertHistory returns alert history with optional filtering
func (tas *TopologyAlertingSystem) GetAlertHistory(
	since time.Time,
	alertType TopologyAlertType,
	severity AlertSeverity,
	deviceID string,
) []TopologyAlert {
	
	tas.mu.RLock()
	defer tas.mu.RUnlock()
	
	var filteredAlerts []TopologyAlert
	
	for _, alert := range tas.alertHistory {
		// Apply filters
		if !alert.CreatedAt.After(since) {
			continue
		}
		if alertType != "" && alert.Type != alertType {
			continue
		}
		if severity != "" && alert.Severity != severity {
			continue
		}
		if deviceID != "" && alert.DeviceID != deviceID {
			continue
		}
		
		filteredAlerts = append(filteredAlerts, alert)
	}
	
	return filteredAlerts
}

// AddAlertRule adds a new alert rule
func (tas *TopologyAlertingSystem) AddAlertRule(rule AlertRule) error {
	tas.mu.Lock()
	defer tas.mu.Unlock()
	
	// Validate rule
	if err := tas.validateAlertRule(rule); err != nil {
		return fmt.Errorf("invalid alert rule: %w", err)
	}
	
	tas.alertRules = append(tas.alertRules, rule)
	
	log.Printf("Added alert rule: %s", rule.Name)
	return nil
}

// CreateSuppression creates a new alert suppression
func (tas *TopologyAlertingSystem) CreateSuppression(suppression AlertSuppression) error {
	tas.mu.Lock()
	defer tas.mu.Unlock()
	
	tas.suppressions[suppression.ID] = &suppression
	
	log.Printf("Created alert suppression: %s", suppression.Name)
	return nil
}

// GetStats returns alerting system statistics
func (tas *TopologyAlertingSystem) GetStats() AlertingStats {
	tas.mu.RLock()
	defer tas.mu.RUnlock()
	
	return tas.stats
}

// Private methods

func (tas *TopologyAlertingSystem) initializeDefaultRules() {
	// Device offline rule
	deviceOfflineRule := AlertRule{
		ID:          "device_offline",
		Name:        "Device Offline Detection",
		Description: "Alert when a device goes offline",
		Enabled:     true,
		Category:    CategoryAvailability,
		Conditions: []RuleCondition{
			{
				Type:     ConditionStatusChange,
				Field:    "device_status",
				Operator: ComparisonOperator("equals"),
				Value:    "offline",
			},
		},
		ConditionLogic: LogicAND,
		AlertType:      AlertDeviceOffline,
		Severity:       SeverityError,
		Priority:       PriorityP2,
		Cooldown:       5 * time.Minute,
		Actions:        []AlertActionType{ActionNotify, ActionLogEvent},
		AutoResolve:    true,
		AutoResolveDelay: 10 * time.Minute,
	}
	
	// Quality degradation rule
	qualityDegradationRule := AlertRule{
		ID:          "quality_degradation",
		Name:        "Connection Quality Degradation",
		Description: "Alert when connection quality degrades significantly",
		Enabled:     true,
		Category:    CategoryPerformance,
		Conditions: []RuleCondition{
			{
				Type:                ConditionMetricThreshold,
				Field:               "quality_score",
				Operator:            OperatorLessThan,
				Threshold:           0.5,
				TimeWindow:          5 * time.Minute,
				AggregationFunction: AggregationAverage,
			},
		},
		ConditionLogic: LogicAND,
		AlertType:      AlertQualityDegraded,
		Severity:       SeverityWarning,
		Priority:       PriorityP3,
		Cooldown:       10 * time.Minute,
		Actions:        []AlertActionType{ActionNotify},
	}
	
	// Excessive roaming rule
	excessiveRoamingRule := AlertRule{
		ID:          "excessive_roaming",
		Name:        "Excessive Roaming Detection",
		Description: "Alert when a device roams excessively",
		Enabled:     true,
		Category:    CategoryPerformance,
		Conditions: []RuleCondition{
			{
				Type:                ConditionMetricThreshold,
				Field:               "roaming_frequency",
				Operator:            OperatorGreaterThan,
				Threshold:           5.0,
				TimeWindow:          time.Hour,
				AggregationFunction: AggregationCount,
			},
		},
		ConditionLogic: LogicAND,
		AlertType:      AlertExcessiveRoaming,
		Severity:       SeverityWarning,
		Priority:       PriorityP3,
		Cooldown:       30 * time.Minute,
		Actions:        []AlertActionType{ActionNotify},
	}
	
	tas.alertRules = []AlertRule{
		deviceOfflineRule,
		qualityDegradationRule,
		excessiveRoamingRule,
	}
	
	log.Printf("Initialized %d default alert rules", len(tas.alertRules))
}

func (tas *TopologyAlertingSystem) alertProcessingLoop(ctx context.Context) {
	ticker := time.NewTicker(tas.config.AlertProcessingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tas.processAlertRules()
		}
	}
}

func (tas *TopologyAlertingSystem) escalationProcessingLoop(ctx context.Context) {
	if !tas.config.EscalationEnabled {
		return
	}
	
	ticker := time.NewTicker(tas.config.EscalationCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tas.processEscalations()
		}
	}
}

func (tas *TopologyAlertingSystem) notificationProcessingLoop(ctx context.Context) {
	ticker := time.NewTicker(tas.config.NotificationRetryInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tas.retryFailedNotifications()
		}
	}
}

func (tas *TopologyAlertingSystem) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(tas.config.AlertCleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tas.cleanupOldAlerts()
		}
	}
}

func (tas *TopologyAlertingSystem) processAlertRules() {
	tas.mu.RLock()
	rules := make([]AlertRule, len(tas.alertRules))
	copy(rules, tas.alertRules)
	tas.mu.RUnlock()
	
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		
		if tas.shouldProcessRule(rule) {
			tas.evaluateRule(rule)
		}
	}
	
	tas.stats.LastProcessingTime = time.Now()
}

func (tas *TopologyAlertingSystem) shouldProcessRule(rule AlertRule) bool {
	// Check cooldown
	if rule.Cooldown > 0 {
		// Find last alert of this type
		for _, alert := range tas.activeAlerts {
			if alert.Type == rule.AlertType && 
			   time.Since(alert.LastOccurrence) < rule.Cooldown {
				return false
			}
		}
	}
	
	// Check frequency limits
	if rule.MaxFrequency > 0 && rule.TimeWindow > 0 {
		count := 0
		since := time.Now().Add(-rule.TimeWindow)
		
		for _, alert := range tas.alertHistory {
			if alert.Type == rule.AlertType && alert.CreatedAt.After(since) {
				count++
			}
		}
		
		if count >= rule.MaxFrequency {
			return false
		}
	}
	
	return true
}

func (tas *TopologyAlertingSystem) evaluateRule(rule AlertRule) {
	// This is a simplified rule evaluation
	// In a real implementation, this would be much more sophisticated
	
	switch rule.AlertType {
	case AlertDeviceOffline:
		tas.checkDeviceOfflineRule(rule)
	case AlertQualityDegraded:
		tas.checkQualityDegradationRule(rule)
	case AlertExcessiveRoaming:
		tas.checkExcessiveRoamingRule(rule)
	default:
		log.Printf("Unknown rule type: %s", rule.AlertType)
	}
}

func (tas *TopologyAlertingSystem) checkDeviceOfflineRule(rule AlertRule) {
	// Check for offline devices using topology manager
	// This is a placeholder implementation
	
	// In a real implementation, this would check actual device states
	// and create alerts for devices that have gone offline
}

func (tas *TopologyAlertingSystem) checkQualityDegradationRule(rule AlertRule) {
	// Check for quality degradation using quality monitor
	if tas.qualityMonitor == nil {
		return
	}
	
	allQuality := tas.qualityMonitor.GetAllConnectionQuality()
	
	for _, metrics := range allQuality {
		if metrics.OverallQuality.Overall < 0.5 {
			// Create quality degradation alert
			context := tas.buildAlertContext(metrics.DeviceID, metrics.MacAddress)
			
			tas.CreateAlert(
				AlertQualityDegraded,
				SeverityWarning,
				metrics.DeviceID,
				metrics.MacAddress,
				"Connection Quality Degraded",
				fmt.Sprintf("Quality score: %.2f", metrics.OverallQuality.Overall),
				context,
			)
		}
	}
}

func (tas *TopologyAlertingSystem) checkExcessiveRoamingRule(rule AlertRule) {
	// Check for excessive roaming using roaming detector
	if tas.roamingDetector == nil {
		return
	}
	
	stats := tas.roamingDetector.GetStats()
	
	// Check if excessive roaming clients count is above threshold
	if stats.ExcessiveRoamingClients > 0 {
		context := tas.buildGenericAlertContext()
		
		tas.CreateAlert(
			AlertExcessiveRoaming,
			SeverityWarning,
			"",
			"",
			"Excessive Roaming Detected",
			fmt.Sprintf("%d clients showing excessive roaming behavior", stats.ExcessiveRoamingClients),
			context,
		)
	}
}

func (tas *TopologyAlertingSystem) processAlertActions(alert *TopologyAlert) {
	// Find applicable rules for this alert
	for _, rule := range tas.alertRules {
		if rule.AlertType == alert.Type && rule.Enabled {
			tas.executeAlertActions(alert, rule.Actions)
			break
		}
	}
}

func (tas *TopologyAlertingSystem) executeAlertActions(alert *TopologyAlert, actions []AlertActionType) {
	for _, action := range actions {
		switch action {
		case ActionNotify:
			tas.sendNotifications(alert)
		case ActionEscalate:
			tas.initiateEscalation(alert)
		case ActionLogEvent:
			tas.logAlertEvent(alert)
		case ActionCreateTicket:
			tas.createTicket(alert)
		default:
			log.Printf("Unknown action: %s", action)
		}
	}
}

func (tas *TopologyAlertingSystem) sendNotifications(alert *TopologyAlert) {
	// Send notifications based on alert severity and configuration
	
	targets := tas.getNotificationTargets(alert)
	
	for _, target := range targets {
		record := NotificationRecord{
			Type:   target.Type,
			Target: target.Target,
			SentAt: time.Now(),
			Status: NotificationPending,
		}
		
		err := tas.sendNotification(target, alert)
		if err != nil {
			record.Status = NotificationFailed
			record.Error = err.Error()
			tas.stats.NotificationsFailed++
		} else {
			record.Status = NotificationSent
			tas.stats.NotificationsSent++
		}
		
		alert.NotificationsSent = append(alert.NotificationsSent, record)
	}
}

func (tas *TopologyAlertingSystem) sendNotification(target NotificationTarget, alert *TopologyAlert) error {
	switch target.Type {
	case NotificationEmail:
		return tas.sendEmailNotification(target.Target, alert)
	case NotificationWebhook:
		return tas.sendWebhookNotification(target.Target, alert)
	case NotificationSlack:
		return tas.sendSlackNotification(target.Target, alert)
	case NotificationMQTT:
		return tas.sendMQTTNotification(target.Target, alert)
	default:
		return fmt.Errorf("unsupported notification type: %s", target.Type)
	}
}

func (tas *TopologyAlertingSystem) getNotificationTargets(alert *TopologyAlert) []NotificationTarget {
	// Return default notification targets based on severity
	// In a real implementation, this would be configurable
	
	targets := []NotificationTarget{}
	
	switch alert.Severity {
	case SeverityCritical:
		targets = append(targets, NotificationTarget{
			Type:     NotificationEmail,
			Target:   "admin@example.com",
			Priority: NotificationUrgent,
			Enabled:  true,
		})
		targets = append(targets, NotificationTarget{
			Type:     NotificationSlack,
			Target:   "#alerts",
			Priority: NotificationHigh,
			Enabled:  true,
		})
	case SeverityError:
		targets = append(targets, NotificationTarget{
			Type:     NotificationEmail,
			Target:   "support@example.com",
			Priority: NotificationHigh,
			Enabled:  true,
		})
	case SeverityWarning:
		targets = append(targets, NotificationTarget{
			Type:     NotificationSlack,
			Target:   "#monitoring",
			Priority: NotificationNormal,
			Enabled:  true,
		})
	}
	
	return targets
}

func (tas *TopologyAlertingSystem) sendEmailNotification(email string, alert *TopologyAlert) error {
	// Placeholder email notification
	log.Printf("Sending email notification to %s for alert: %s", email, alert.Title)
	return nil
}

func (tas *TopologyAlertingSystem) sendWebhookNotification(url string, alert *TopologyAlert) error {
	// Placeholder webhook notification
	log.Printf("Sending webhook notification to %s for alert: %s", url, alert.Title)
	return nil
}

func (tas *TopologyAlertingSystem) sendSlackNotification(channel string, alert *TopologyAlert) error {
	// Placeholder Slack notification
	log.Printf("Sending Slack notification to %s for alert: %s", channel, alert.Title)
	return nil
}

func (tas *TopologyAlertingSystem) sendMQTTNotification(topic string, alert *TopologyAlert) error {
	// Placeholder MQTT notification
	log.Printf("Sending MQTT notification to %s for alert: %s", topic, alert.Title)
	return nil
}

func (tas *TopologyAlertingSystem) initiateEscalation(alert *TopologyAlert) {
	if !tas.config.EscalationEnabled {
		return
	}
	
	// Find escalation chain for this alert type
	var escalationChain []EscalationStep
	for _, rule := range tas.alertRules {
		if rule.AlertType == alert.Type && len(rule.EscalationChain) > 0 {
			escalationChain = rule.EscalationChain
			break
		}
	}
	
	if len(escalationChain) == 0 {
		return
	}
	
	escalation := &AlertEscalation{
		AlertID:         alert.ID,
		CurrentLevel:    0,
		EscalationChain: escalationChain,
		LastEscalation:  time.Now(),
		NextEscalation:  time.Now().Add(escalationChain[0].Delay),
		MaxLevel:        len(escalationChain),
		Completed:       false,
	}
	
	tas.escalations[alert.ID] = escalation
	
	alert.Escalated = true
	alert.EscalationLevel = 0
	
	log.Printf("Initiated escalation for alert: %s", alert.ID)
}

func (tas *TopologyAlertingSystem) processEscalations() {
	now := time.Now()
	
	tas.mu.Lock()
	defer tas.mu.Unlock()
	
	for alertID, escalation := range tas.escalations {
		if escalation.Completed {
			continue
		}
		
		if now.After(escalation.NextEscalation) {
			tas.executeEscalationStep(alertID, escalation)
		}
	}
}

func (tas *TopologyAlertingSystem) executeEscalationStep(alertID string, escalation *AlertEscalation) {
	if escalation.CurrentLevel >= len(escalation.EscalationChain) {
		escalation.Completed = true
		return
	}
	
	step := escalation.EscalationChain[escalation.CurrentLevel]
	
	// Send escalation notifications
	alert, exists := tas.activeAlerts[alertID]
	if exists {
		for _, target := range step.NotificationTargets {
			tas.sendNotification(target, alert)
		}
		
		alert.EscalationLevel = escalation.CurrentLevel
		tas.stats.EscalatedAlerts++
	}
	
	// Execute escalation actions
	for _, action := range step.Actions {
		tas.executeEscalationAction(action, alertID)
	}
	
	// Schedule next escalation
	escalation.CurrentLevel++
	escalation.LastEscalation = time.Now()
	
	if escalation.CurrentLevel < len(escalation.EscalationChain) {
		nextStep := escalation.EscalationChain[escalation.CurrentLevel]
		escalation.NextEscalation = time.Now().Add(nextStep.Delay)
	} else {
		escalation.Completed = true
	}
	
	log.Printf("Executed escalation step %d for alert: %s", escalation.CurrentLevel-1, alertID)
}

func (tas *TopologyAlertingSystem) executeEscalationAction(action EscalationAction, alertID string) {
	switch action {
	case EscalationNotifyManager:
		log.Printf("Notifying manager for escalated alert: %s", alertID)
	case EscalationCreateIncident:
		log.Printf("Creating incident for escalated alert: %s", alertID)
	case EscalationPageOnCall:
		log.Printf("Paging on-call for escalated alert: %s", alertID)
	case EscalationExecuteRunbook:
		log.Printf("Executing runbook for escalated alert: %s", alertID)
	}
}

func (tas *TopologyAlertingSystem) logAlertEvent(alert *TopologyAlert) {
	log.Printf("ALERT EVENT: [%s] %s - %s (%s)", alert.Severity, alert.Type, alert.Title, alert.DeviceID)
}

func (tas *TopologyAlertingSystem) createTicket(alert *TopologyAlert) {
	// Placeholder ticket creation
	log.Printf("Creating ticket for alert: %s", alert.Title)
}

func (tas *TopologyAlertingSystem) retryFailedNotifications() {
	// Retry failed notifications
	tas.mu.Lock()
	defer tas.mu.Unlock()
	
	for _, alert := range tas.activeAlerts {
		for i, record := range alert.NotificationsSent {
			if record.Status == NotificationFailed && 
			   time.Since(record.SentAt) > tas.config.NotificationRetryInterval {
				
				// Retry notification
				targets := tas.getNotificationTargets(alert)
				for _, target := range targets {
					if target.Target == record.Target && target.Type == record.Type {
						err := tas.sendNotification(target, alert)
						if err == nil {
							alert.NotificationsSent[i].Status = NotificationSent
							alert.NotificationsSent[i].SentAt = time.Now()
							tas.stats.NotificationsSent++
						}
						break
					}
				}
			}
		}
	}
}

func (tas *TopologyAlertingSystem) cleanupOldAlerts() {
	now := time.Now()
	cutoff := now.Add(-tas.config.AlertHistoryRetention)
	
	tas.mu.Lock()
	defer tas.mu.Unlock()
	
	// Clean up old alert history
	var filteredHistory []TopologyAlert
	for _, alert := range tas.alertHistory {
		if alert.CreatedAt.After(cutoff) {
			filteredHistory = append(filteredHistory, alert)
		}
	}
	
	removed := len(tas.alertHistory) - len(filteredHistory)
	tas.alertHistory = filteredHistory
	
	if removed > 0 {
		log.Printf("Cleaned up %d old alerts from history", removed)
	}
}

func (tas *TopologyAlertingSystem) subscribeToTopologyUpdates() {
	// Subscribe to real-time topology updates to trigger alerts
	filter := UpdateFilter{
		EventTypes: []UpdateEventType{
			EventDeviceAdded,
			EventDeviceRemoved,
			EventDeviceOffline,
			EventDeviceOnline,
			EventConnectionAdded,
			EventConnectionRemoved,
			EventTopologyChanged,
			EventAnomalyDetected,
		},
		IncludeDetails: true,
	}
	
	subscription, err := tas.realtimeUpdater.Subscribe(
		"alerting_system",
		filter,
		DeliveryWebhook,
		"internal://alerting",
	)
	
	if err != nil {
		log.Printf("Failed to subscribe to topology updates: %v", err)
		return
	}
	
	log.Printf("Subscribed to topology updates: %s", subscription.ID)
}

func (tas *TopologyAlertingSystem) findDuplicateAlert(
	alertType TopologyAlertType,
	deviceID string,
	macAddress string,
) *TopologyAlert {
	
	cutoff := time.Now().Add(-tas.config.DuplicateTimeWindow)
	
	for _, alert := range tas.activeAlerts {
		if alert.Type == alertType &&
		   alert.DeviceID == deviceID &&
		   alert.MacAddress == macAddress &&
		   alert.LastOccurrence.After(cutoff) {
			return alert
		}
	}
	
	return nil
}

func (tas *TopologyAlertingSystem) isAlertSuppressed(
	alertType TopologyAlertType,
	deviceID string,
	macAddress string,
) bool {
	
	if !tas.config.SuppressionEnabled {
		return false
	}
	
	now := time.Now()
	
	for _, suppression := range tas.suppressions {
		if !suppression.Active {
			continue
		}
		
		// Check time window
		if !suppression.StartTime.IsZero() && now.Before(suppression.StartTime) {
			continue
		}
		if !suppression.EndTime.IsZero() && now.After(suppression.EndTime) {
			continue
		}
		
		// Check alert type
		if len(suppression.AlertTypes) > 0 {
			found := false
			for _, t := range suppression.AlertTypes {
				if t == alertType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Check device ID
		if len(suppression.DeviceIDs) > 0 {
			found := false
			for _, id := range suppression.DeviceIDs {
				if id == deviceID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Check MAC address
		if len(suppression.MacAddresses) > 0 {
			found := false
			for _, mac := range suppression.MacAddresses {
				if mac == macAddress {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Alert is suppressed
		return true
	}
	
	return false
}

func (tas *TopologyAlertingSystem) categorizeAlert(alertType TopologyAlertType) AlertCategory {
	switch alertType {
	case AlertDeviceOffline, AlertDeviceOnline, AlertConnectionLost, AlertConnectionEstablished:
		return CategoryAvailability
	case AlertQualityDegraded, AlertConnectionDegraded, 
		 AlertPacketLoss, AlertBandwidthSaturation:
		return AlertCategory("performance")
	case AlertUnauthorizedDevice, AlertAnomalousTraffic, AlertSuspiciousActivity:
		return AlertCategory("security")
	case AlertDeviceMisconfigured, AlertConfigurationError:
		return AlertCategory("configuration")
	case AlertSystemOverloaded:
		return CategoryCapacity
	default:
		return CategoryAvailability
	}
}

func (tas *TopologyAlertingSystem) calculateImpact(severity AlertSeverity, alertType TopologyAlertType) AlertImpact {
	switch severity {
	case SeverityCritical:
		return ImpactCritical
	case SeverityError:
		return ImpactHigh
	case SeverityWarning:
		return ImpactMedium
	case SeverityInfo:
		return ImpactLow
	default:
		return ImpactLow
	}
}

func (tas *TopologyAlertingSystem) calculateUrgency(severity AlertSeverity, alertType TopologyAlertType) AlertUrgency {
	switch alertType {
	case AlertDeviceOffline, AlertConnectionLost, AlertNetworkSplit:
		return UrgencyHigh
	case AlertQualityDegraded, AlertConnectionDegraded:
		return UrgencyMedium
	case AlertDeviceAdded, AlertDeviceOnline:
		return UrgencyLow
	default:
		switch severity {
		case SeverityCritical:
			return UrgencyImmediate
		case SeverityError:
			return UrgencyHigh
		case SeverityWarning:
			return UrgencyMedium
		default:
			return UrgencyLow
		}
	}
}

func (tas *TopologyAlertingSystem) generateRecommendations(alertType TopologyAlertType, severity AlertSeverity) []string {
	recommendations := []string{}
	
	switch alertType {
	case AlertDeviceOffline:
		recommendations = append(recommendations, "Check device power and network connectivity")
		recommendations = append(recommendations, "Verify network configuration")
		recommendations = append(recommendations, "Check for hardware failures")
	case AlertQualityDegraded:
		recommendations = append(recommendations, "Check signal strength and interference")
		recommendations = append(recommendations, "Verify network bandwidth allocation")
		recommendations = append(recommendations, "Consider device placement optimization")
	case AlertExcessiveRoaming:
		recommendations = append(recommendations, "Check WiFi coverage and signal overlap")
		recommendations = append(recommendations, "Review roaming thresholds and settings")
		recommendations = append(recommendations, "Analyze interference sources")
	// TODO: Add AlertConnectionUnstable to TopologyAlertType
	// case AlertConnectionUnstable:
	//	recommendations = append(recommendations, "Check network hardware health")
	//	recommendations = append(recommendations, "Verify cable connections and quality")
	//	recommendations = append(recommendations, "Monitor for interference sources")
	default:
		recommendations = append(recommendations, "Review device and network configuration")
		recommendations = append(recommendations, "Check system logs for additional details")
	}
	
	return recommendations
}

func (tas *TopologyAlertingSystem) formatAlertMessage(alertType TopologyAlertType, title string, description string) string {
	return fmt.Sprintf("[%s] %s: %s", alertType, title, description)
}

func (tas *TopologyAlertingSystem) buildAlertContext(deviceID string, macAddress string) AlertContext {
	context := AlertContext{
		SystemContext: tas.buildSystemContext(),
	}
	
	// Add device-specific context if available
	if deviceID != "" {
		context.DeviceContext = tas.buildDeviceContext(deviceID)
	}
	
	// Add quality context if quality monitor is available
	if tas.qualityMonitor != nil && deviceID != "" && macAddress != "" {
		if metrics, exists := tas.qualityMonitor.GetConnectionQuality(deviceID, macAddress); exists {
			context.QualityContext = QualityStateContext{
				AverageQuality: metrics.OverallQuality.Overall,
				QualityTrend:   string(metrics.TrendAnalysis.TrendDirection),
				AffectedMetrics: []string{"signal", "throughput", "latency"},
			}
		}
	}
	
	return context
}

func (tas *TopologyAlertingSystem) buildGenericAlertContext() AlertContext {
	return AlertContext{
		SystemContext: tas.buildSystemContext(),
		NetworkState: NetworkStateContext{
			NetworkLoad:        0.5, // TODO: Get actual network load
			ConnectivityHealth: 0.8, // TODO: Calculate connectivity health
		},
	}
}

func (tas *TopologyAlertingSystem) buildSystemContext() SystemStateContext {
	return SystemStateContext{
		SystemLoad:         0.5, // TODO: Get actual system load
		MemoryUsage:        0.3, // TODO: Get actual memory usage
		DiskUsage:          0.2, // TODO: Get actual disk usage
		NetworkUtilization: 0.4, // TODO: Get actual network utilization
		ProcessingErrors:   int(tas.stats.ProcessingErrors),
	}
}

func (tas *TopologyAlertingSystem) buildDeviceContext(deviceID string) DeviceStateContext {
	// Build device context - placeholder implementation
	return DeviceStateContext{
		DeviceType:   "unknown",
		DeviceStatus: "unknown",
		DeviceUptime: time.Hour,
		DeviceLoad:   0.5,
	}
}

func (tas *TopologyAlertingSystem) validateAlertRule(rule AlertRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("at least one condition is required")
	}
	if rule.AlertType == "" {
		return fmt.Errorf("alert type is required")
	}
	
	return nil
}
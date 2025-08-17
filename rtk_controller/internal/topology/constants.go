package topology

// Severity levels for alerts and diagnostics
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityError    = "error"
	SeverityCritical = "critical"
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
)

// Priority levels
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
)

// Status values
const (
	StatusPending   = "pending"
	StatusActive    = "active"
	StatusResolved  = "resolved"
	StatusCompleted = "completed"
)

// Impact levels
const (
	ImpactLow      = "low"
	ImpactMedium   = "medium"
	ImpactHigh     = "high"
	ImpactCritical = "critical"
)

// Category types
const (
	CategorySecurity     = "security"
	CategoryPerformance  = "performance"
	CategoryConnectivity = "connectivity"
	CategoryMaintenance  = "maintenance"
)
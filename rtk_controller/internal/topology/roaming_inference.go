package topology

import (
	"fmt"
	"log"
	"time"

	"rtk_controller/internal/storage"
)

// RoamingInferenceEngine infers roaming events from WiFi client telemetry
type RoamingInferenceEngine struct {
	// Data sources
	wifiCollector     *WiFiClientCollector
	connectionTracker *ConnectionHistoryTracker
	identityStorage   *storage.IdentityStorage

	// Inference state
	clientStates   map[string]*InferenceClientState
	inferenceRules []InferenceRule

	// Configuration
	config RoamingInferenceConfig

	// Statistics
	stats RoamingInferenceStats
}

// InferenceClientState tracks client state for roaming inference
type InferenceClientState struct {
	MacAddress       string
	CurrentAP        string
	CurrentSSID      string
	LastSignalRSSI   int
	LastUpdateTime   time.Time
	SignalHistory    []SignalPoint
	APHistory        []APConnection
	InferenceScore   float64
	PendingInference *PendingRoamingEvent
}

// SignalPoint represents a signal measurement point
type SignalPoint struct {
	Timestamp time.Time
	RSSI      int
	AP        string
	SSID      string
	Quality   float64
}

// APConnection represents a connection to an AP
type APConnection struct {
	AP          string
	SSID        string
	StartTime   time.Time
	EndTime     time.Time
	AverageRSSI int
	Stability   float64
	Quality     float64
}

// PendingRoamingEvent represents a potential roaming event being evaluated
type PendingRoamingEvent struct {
	FromAP         string
	ToAP           string
	FromSSID       string
	ToSSID         string
	StartTime      time.Time
	SignalBefore   int
	SignalAfter    int
	Confidence     float64
	TriggerFactors []string
	EvaluationTime time.Time
}

// InferenceRule defines rules for roaming inference
type InferenceRule struct {
	ID          string
	Name        string
	Description string
	Weight      float64
	Conditions  []InferenceCondition
	Action      InferenceAction
}

// InferenceCondition defines conditions for inference rules
type InferenceCondition struct {
	Type     ConditionType
	Field    string
	Operator string
	Value    interface{}
	Weight   float64
}

// InferenceAction defines actions when rules match
type InferenceAction struct {
	Type       ActionType
	Confidence float64
	Trigger    RoamingTrigger
	Metadata   map[string]interface{}
}

// Enums for inference engine
type ConditionType string

const (
	ConditionSignalStrength ConditionType = "signal_strength"
	ConditionTimeDelta      ConditionType = "time_delta"
	ConditionAPChange       ConditionType = "ap_change"
	ConditionSSIDChange     ConditionType = "ssid_change"
	ConditionSignalTrend    ConditionType = "signal_trend"
	ConditionFrequency      ConditionType = "frequency"
)

type ActionType string

const (
	ActionInferRoaming   ActionType = "infer_roaming"
	ActionRejectRoaming  ActionType = "reject_roaming"
	ActionPendEvaluation ActionType = "pend_evaluation"
	ActionSignalAnomaly  ActionType = "signal_anomaly"
)

// RoamingInferenceConfig holds inference engine configuration
type RoamingInferenceConfig struct {
	// Signal thresholds
	WeakSignalThreshold   int // RSSI threshold for weak signal
	StrongSignalThreshold int // RSSI threshold for strong signal
	SignalDeltaThreshold  int // Minimum RSSI change for roaming
	SignalStabilityWindow time.Duration

	// Timing thresholds
	MinRoamingGap    time.Duration // Minimum time between roamings
	MaxRoamingGap    time.Duration // Maximum time for roaming detection
	EvaluationWindow time.Duration // Window for evaluating roaming
	HistoryWindow    time.Duration // How far back to look

	// Inference parameters
	MinConfidenceThreshold float64 // Minimum confidence to infer roaming
	SignalHistorySize      int     // Number of signal points to keep
	APHistorySize          int     // Number of AP connections to keep

	// Quality thresholds
	MinConnectionQuality float64
	StabilityThreshold   float64

	// Rules configuration
	EnableSignalBasedRules  bool
	EnableTimeBasedRules    bool
	EnablePatternBasedRules bool
	EnableAnomalyDetection  bool
}

// RoamingInferenceStats holds inference statistics
type RoamingInferenceStats struct {
	TotalInferences      int64
	SuccessfulInferences int64
	RejectedInferences   int64
	PendingEvaluations   int64
	SignalAnomalies      int64
	AverageConfidence    float64
	LastInference        time.Time
	ProcessingErrors     int64
}

// NewRoamingInferenceEngine creates a new roaming inference engine
func NewRoamingInferenceEngine(
	wifiCollector *WiFiClientCollector,
	connectionTracker *ConnectionHistoryTracker,
	identityStorage *storage.IdentityStorage,
	config RoamingInferenceConfig,
) *RoamingInferenceEngine {

	engine := &RoamingInferenceEngine{
		wifiCollector:     wifiCollector,
		connectionTracker: connectionTracker,
		identityStorage:   identityStorage,
		clientStates:      make(map[string]*InferenceClientState),
		config:            config,
		stats:             RoamingInferenceStats{},
	}

	// Initialize inference rules
	engine.initializeInferenceRules()

	return engine
}

// ProcessWiFiClientUpdate processes WiFi client updates for roaming inference
func (rie *RoamingInferenceEngine) ProcessWiFiClientUpdate(
	apDeviceID string,
	clientMAC string,
	ssid string,
	rssi int,
	timestamp time.Time,
) (*RoamingAnalysisEvent, error) {

	// Get or create client state
	clientState, exists := rie.clientStates[clientMAC]
	if !exists {
		clientState = &InferenceClientState{
			MacAddress:    clientMAC,
			SignalHistory: []SignalPoint{},
			APHistory:     []APConnection{},
		}
		rie.clientStates[clientMAC] = clientState
	}

	// Add signal point
	signalPoint := SignalPoint{
		Timestamp: timestamp,
		RSSI:      rssi,
		AP:        apDeviceID,
		SSID:      ssid,
		Quality:   rie.calculateSignalQuality(rssi),
	}

	clientState.SignalHistory = append(clientState.SignalHistory, signalPoint)

	// Limit signal history size
	if len(clientState.SignalHistory) > rie.config.SignalHistorySize {
		clientState.SignalHistory = clientState.SignalHistory[len(clientState.SignalHistory)-rie.config.SignalHistorySize:]
	}

	// Check for AP change (potential roaming)
	previousAP := clientState.CurrentAP
	currentAP := apDeviceID

	if previousAP != "" && previousAP != currentAP {
		// Potential roaming detected
		roamingEvent, err := rie.inferRoamingEvent(clientState, previousAP, currentAP, ssid, timestamp)
		if err != nil {
			rie.stats.ProcessingErrors++
			return nil, fmt.Errorf("failed to infer roaming event: %w", err)
		}

		if roamingEvent != nil {
			rie.stats.SuccessfulInferences++
			rie.stats.LastInference = timestamp

			log.Printf("Roaming inferred: %s -> %s for client %s (confidence: %.2f)",
				previousAP, currentAP, clientMAC, roamingEvent.Confidence)

			return roamingEvent, nil
		}
	}

	// Update client state
	clientState.CurrentAP = currentAP
	clientState.CurrentSSID = ssid
	clientState.LastSignalRSSI = rssi
	clientState.LastUpdateTime = timestamp

	// Update AP history
	rie.updateAPHistory(clientState, apDeviceID, ssid, timestamp)

	// Calculate inference score
	clientState.InferenceScore = rie.calculateInferenceScore(clientState)

	return nil, nil
}

// InferRoamingFromHistory infers roaming events from historical data
func (rie *RoamingInferenceEngine) InferRoamingFromHistory(
	clientMAC string,
	startTime time.Time,
	endTime time.Time,
) ([]RoamingAnalysisEvent, error) {

	// Get WiFi client state history
	clientState, exists := rie.clientStates[clientMAC]
	if !exists {
		return nil, fmt.Errorf("no client state found for %s", clientMAC)
	}

	var inferredEvents []RoamingAnalysisEvent

	// Analyze signal history for roaming patterns
	events := rie.analyzeSignalHistoryForRoaming(clientState, startTime, endTime)
	inferredEvents = append(inferredEvents, events...)

	return inferredEvents, nil
}

// GetInferenceStats returns inference engine statistics
func (rie *RoamingInferenceEngine) GetInferenceStats() RoamingInferenceStats {
	// Calculate average confidence
	if rie.stats.SuccessfulInferences > 0 {
		totalConfidence := 0.0
		for _, clientState := range rie.clientStates {
			totalConfidence += clientState.InferenceScore
		}
		rie.stats.AverageConfidence = totalConfidence / float64(len(rie.clientStates))
	}

	return rie.stats
}

// Private methods

func (rie *RoamingInferenceEngine) initializeInferenceRules() {
	// Initialize built-in inference rules

	// Rule 1: Signal-driven roaming
	rie.inferenceRules = append(rie.inferenceRules, InferenceRule{
		ID:          "signal_driven_roaming",
		Name:        "Signal-Driven Roaming",
		Description: "Infer roaming when signal strength improves significantly",
		Weight:      0.8,
		Conditions: []InferenceCondition{
			{
				Type:     ConditionSignalStrength,
				Field:    "signal_delta",
				Operator: "greater_than",
				Value:    rie.config.SignalDeltaThreshold,
				Weight:   0.6,
			},
			{
				Type:     ConditionTimeDelta,
				Field:    "time_gap",
				Operator: "less_than",
				Value:    rie.config.MaxRoamingGap,
				Weight:   0.4,
			},
		},
		Action: InferenceAction{
			Type:       ActionInferRoaming,
			Confidence: 0.8,
			Trigger:    TriggerBetterSignal,
		},
	})

	// Rule 2: Weak signal roaming
	rie.inferenceRules = append(rie.inferenceRules, InferenceRule{
		ID:          "weak_signal_roaming",
		Name:        "Weak Signal Roaming",
		Description: "Infer roaming when leaving an AP with weak signal",
		Weight:      0.7,
		Conditions: []InferenceCondition{
			{
				Type:     ConditionSignalStrength,
				Field:    "previous_signal",
				Operator: "less_than",
				Value:    rie.config.WeakSignalThreshold,
				Weight:   0.7,
			},
			{
				Type:     ConditionSignalStrength,
				Field:    "current_signal",
				Operator: "greater_than",
				Value:    rie.config.WeakSignalThreshold,
				Weight:   0.3,
			},
		},
		Action: InferenceAction{
			Type:       ActionInferRoaming,
			Confidence: 0.7,
			Trigger:    TriggerWeakSignal,
		},
	})

	// Rule 3: Rapid AP switching (ping-pong)
	rie.inferenceRules = append(rie.inferenceRules, InferenceRule{
		ID:          "ping_pong_detection",
		Name:        "Ping-Pong Detection",
		Description: "Detect and reject ping-pong roaming",
		Weight:      0.9,
		Conditions: []InferenceCondition{
			{
				Type:     ConditionTimeDelta,
				Field:    "time_gap",
				Operator: "less_than",
				Value:    rie.config.MinRoamingGap,
				Weight:   0.5,
			},
			{
				Type:     ConditionAPChange,
				Field:    "ap_return",
				Operator: "equals",
				Value:    true,
				Weight:   0.5,
			},
		},
		Action: InferenceAction{
			Type:       ActionRejectRoaming,
			Confidence: 0.9,
		},
	})
}

func (rie *RoamingInferenceEngine) inferRoamingEvent(
	clientState *InferenceClientState,
	fromAP string,
	toAP string,
	toSSID string,
	timestamp time.Time,
) (*RoamingAnalysisEvent, error) {

	// Get previous signal strength
	previousSignal := clientState.LastSignalRSSI
	currentSignal := rie.getCurrentSignal(clientState, toAP)

	// Create inference context
	context := rie.createInferenceContext(clientState, fromAP, toAP, timestamp)

	// Apply inference rules
	var totalConfidence float64
	var totalWeight float64
	var triggers []RoamingTrigger
	var shouldReject bool

	for _, rule := range rie.inferenceRules {
		matches, confidence := rie.evaluateRule(rule, context)
		if matches {
			totalConfidence += confidence * rule.Weight
			totalWeight += rule.Weight

			if rule.Action.Type == ActionRejectRoaming {
				shouldReject = true
				break
			}

			if rule.Action.Trigger != "" {
				triggers = append(triggers, rule.Action.Trigger)
			}
		}
	}

	// Reject if any rejection rule matched
	if shouldReject {
		rie.stats.RejectedInferences++
		log.Printf("Roaming inference rejected for client %s: %s -> %s",
			clientState.MacAddress, fromAP, toAP)
		return nil, nil
	}

	// Calculate final confidence
	finalConfidence := 0.5 // Base confidence
	if totalWeight > 0 {
		finalConfidence = totalConfidence / totalWeight
	}

	// Check confidence threshold
	if finalConfidence < rie.config.MinConfidenceThreshold {
		log.Printf("Roaming inference confidence too low: %.2f < %.2f",
			finalConfidence, rie.config.MinConfidenceThreshold)
		return nil, nil
	}

	// Get friendly name
	friendlyName := clientState.MacAddress
	if identity, err := rie.identityStorage.GetDeviceIdentity(clientState.MacAddress); err == nil {
		friendlyName = identity.FriendlyName
	}

	// Get previous SSID
	fromSSID := rie.getPreviousSSID(clientState, fromAP)

	// Determine primary trigger
	primaryTrigger := TriggerUnknown
	if len(triggers) > 0 {
		primaryTrigger = triggers[0]
	}

	// Determine roaming type
	roamingType := rie.determineRoamingType(previousSignal, currentSignal, fromSSID, toSSID)

	// Create roaming analysis event
	event := &RoamingAnalysisEvent{
		ID:           fmt.Sprintf("inferred_%d_%s", timestamp.UnixMilli(), clientState.MacAddress),
		MacAddress:   clientState.MacAddress,
		FriendlyName: friendlyName,
		FromAP:       fromAP,
		ToAP:         toAP,
		FromSSID:     fromSSID,
		ToSSID:       toSSID,
		Timestamp:    timestamp,
		RoamingType:  roamingType,
		Trigger:      primaryTrigger,
		SignalBefore: previousSignal,
		SignalAfter:  currentSignal,
		Confidence:   finalConfidence,
		Context:      rie.buildRoamingContext(),
	}

	// Calculate event quality
	rie.calculateEventQualityInference(event)

	rie.stats.TotalInferences++

	return event, nil
}

func (rie *RoamingInferenceEngine) createInferenceContext(
	clientState *InferenceClientState,
	fromAP string,
	toAP string,
	timestamp time.Time,
) map[string]interface{} {

	context := make(map[string]interface{})

	// Basic information
	context["client_mac"] = clientState.MacAddress
	context["from_ap"] = fromAP
	context["to_ap"] = toAP
	context["timestamp"] = timestamp

	// Signal information
	context["previous_signal"] = clientState.LastSignalRSSI
	context["current_signal"] = rie.getCurrentSignal(clientState, toAP)
	context["signal_delta"] = context["current_signal"].(int) - context["previous_signal"].(int)

	// Timing information
	if !clientState.LastUpdateTime.IsZero() {
		context["time_gap"] = timestamp.Sub(clientState.LastUpdateTime)
	}

	// History information
	context["ap_history"] = clientState.APHistory
	context["signal_history"] = clientState.SignalHistory

	// Check for AP return (ping-pong)
	context["ap_return"] = rie.checkAPReturn(clientState, fromAP, toAP)

	return context
}

func (rie *RoamingInferenceEngine) evaluateRule(
	rule InferenceRule,
	context map[string]interface{},
) (bool, float64) {

	var totalWeight float64
	var matchedWeight float64

	for _, condition := range rule.Conditions {
		totalWeight += condition.Weight

		if rie.evaluateCondition(condition, context) {
			matchedWeight += condition.Weight
		}
	}

	// Rule matches if most conditions are met
	matches := matchedWeight/totalWeight >= 0.5
	confidence := matchedWeight / totalWeight

	return matches, confidence
}

func (rie *RoamingInferenceEngine) evaluateCondition(
	condition InferenceCondition,
	context map[string]interface{},
) bool {

	value := context[condition.Field]
	if value == nil {
		return false
	}

	switch condition.Type {
	case ConditionSignalStrength:
		return rie.evaluateNumericCondition(value, condition.Operator, condition.Value)
	case ConditionTimeDelta:
		return rie.evaluateTimeCondition(value, condition.Operator, condition.Value)
	case ConditionAPChange:
		return rie.evaluateBooleanCondition(value, condition.Operator, condition.Value)
	default:
		return false
	}
}

func (rie *RoamingInferenceEngine) evaluateNumericCondition(
	value interface{},
	operator string,
	expected interface{},
) bool {

	numValue, ok1 := value.(int)
	numExpected, ok2 := expected.(int)

	if !ok1 || !ok2 {
		return false
	}

	switch operator {
	case "greater_than":
		return numValue > numExpected
	case "less_than":
		return numValue < numExpected
	case "equals":
		return numValue == numExpected
	case "greater_equal":
		return numValue >= numExpected
	case "less_equal":
		return numValue <= numExpected
	default:
		return false
	}
}

func (rie *RoamingInferenceEngine) evaluateTimeCondition(
	value interface{},
	operator string,
	expected interface{},
) bool {

	timeValue, ok1 := value.(time.Duration)
	timeExpected, ok2 := expected.(time.Duration)

	if !ok1 || !ok2 {
		return false
	}

	switch operator {
	case "greater_than":
		return timeValue > timeExpected
	case "less_than":
		return timeValue < timeExpected
	case "equals":
		return timeValue == timeExpected
	default:
		return false
	}
}

func (rie *RoamingInferenceEngine) evaluateBooleanCondition(
	value interface{},
	operator string,
	expected interface{},
) bool {

	boolValue, ok1 := value.(bool)
	boolExpected, ok2 := expected.(bool)

	if !ok1 || !ok2 {
		return false
	}

	switch operator {
	case "equals":
		return boolValue == boolExpected
	case "not_equals":
		return boolValue != boolExpected
	default:
		return false
	}
}

func (rie *RoamingInferenceEngine) analyzeSignalHistoryForRoaming(
	clientState *InferenceClientState,
	startTime time.Time,
	endTime time.Time,
) []RoamingAnalysisEvent {

	var events []RoamingAnalysisEvent

	// Look for signal patterns that indicate roaming
	var previousAP string
	var previousTime time.Time

	for _, point := range clientState.SignalHistory {
		if point.Timestamp.Before(startTime) || point.Timestamp.After(endTime) {
			continue
		}

		if previousAP != "" && previousAP != point.AP {
			// Potential roaming event
			timeDelta := point.Timestamp.Sub(previousTime)

			if timeDelta > rie.config.MinRoamingGap && timeDelta < rie.config.MaxRoamingGap {
				// Infer roaming event
				event := RoamingAnalysisEvent{
					ID:          fmt.Sprintf("historical_%d_%s", point.Timestamp.UnixMilli(), clientState.MacAddress),
					MacAddress:  clientState.MacAddress,
					FromAP:      previousAP,
					ToAP:        point.AP,
					Timestamp:   point.Timestamp,
					SignalAfter: point.RSSI,
					RoamingType: RoamingTypeSignalDriven,
					Trigger:     TriggerUnknown,
					Confidence:  0.6, // Lower confidence for historical inference
				}

				events = append(events, event)
			}
		}

		previousAP = point.AP
		previousTime = point.Timestamp
	}

	return events
}

func (rie *RoamingInferenceEngine) calculateSignalQuality(rssi int) float64 {
	// Convert RSSI to quality score (0-1)

	if rssi >= rie.config.StrongSignalThreshold {
		return 1.0
	} else if rssi <= -90 {
		return 0.0
	} else {
		// Linear interpolation
		return float64(rssi+90) / float64(rie.config.StrongSignalThreshold+90)
	}
}

func (rie *RoamingInferenceEngine) calculateInferenceScore(clientState *InferenceClientState) float64 {
	score := 0.5 // Base score

	// Factor in signal quality
	if len(clientState.SignalHistory) > 0 {
		lastPoint := clientState.SignalHistory[len(clientState.SignalHistory)-1]
		score += lastPoint.Quality * 0.3
	}

	// Factor in connection stability
	if len(clientState.APHistory) > 0 {
		lastConnection := clientState.APHistory[len(clientState.APHistory)-1]
		score += lastConnection.Stability * 0.2
	}

	return score
}

func (rie *RoamingInferenceEngine) updateAPHistory(
	clientState *InferenceClientState,
	ap string,
	ssid string,
	timestamp time.Time,
) {

	// Check if this is a continuation of the current connection
	if len(clientState.APHistory) > 0 {
		lastConnection := &clientState.APHistory[len(clientState.APHistory)-1]

		if lastConnection.AP == ap && lastConnection.SSID == ssid {
			// Update existing connection
			lastConnection.EndTime = timestamp
			return
		}

		// End previous connection
		lastConnection.EndTime = timestamp
	}

	// Start new connection
	newConnection := APConnection{
		AP:        ap,
		SSID:      ssid,
		StartTime: timestamp,
		EndTime:   timestamp,
	}

	clientState.APHistory = append(clientState.APHistory, newConnection)

	// Limit AP history size
	if len(clientState.APHistory) > rie.config.APHistorySize {
		clientState.APHistory = clientState.APHistory[1:]
	}
}

func (rie *RoamingInferenceEngine) getCurrentSignal(clientState *InferenceClientState, ap string) int {
	// Get current signal strength for the specified AP

	for i := len(clientState.SignalHistory) - 1; i >= 0; i-- {
		point := clientState.SignalHistory[i]
		if point.AP == ap {
			return point.RSSI
		}
	}

	return -100 // Default weak signal
}

func (rie *RoamingInferenceEngine) getPreviousSSID(clientState *InferenceClientState, ap string) string {
	// Get SSID for the specified AP from history

	for i := len(clientState.APHistory) - 1; i >= 0; i-- {
		connection := clientState.APHistory[i]
		if connection.AP == ap {
			return connection.SSID
		}
	}

	return ""
}

func (rie *RoamingInferenceEngine) checkAPReturn(
	clientState *InferenceClientState,
	fromAP string,
	toAP string,
) bool {

	// Check if client is returning to a recently used AP (ping-pong)
	if len(clientState.APHistory) < 2 {
		return false
	}

	// Look at recent AP history
	recentAPs := make(map[string]time.Time)
	cutoff := time.Now().Add(-rie.config.MinRoamingGap * 3)

	for _, connection := range clientState.APHistory {
		if connection.EndTime.After(cutoff) {
			recentAPs[connection.AP] = connection.EndTime
		}
	}

	// Check if toAP was recently used
	_, wasRecentlyUsed := recentAPs[toAP]
	return wasRecentlyUsed
}

func (rie *RoamingInferenceEngine) determineRoamingType(
	previousSignal int,
	currentSignal int,
	fromSSID string,
	toSSID string,
) RoamingType {

	signalDelta := currentSignal - previousSignal

	if fromSSID != toSSID {
		return RoamingTypeBandSteering
	} else if previousSignal < rie.config.WeakSignalThreshold {
		return RoamingTypeSignalDriven
	} else if signalDelta > 15 {
		return RoamingTypeSignalDriven
	} else {
		return RoamingTypeLoadBalancing
	}
}

func (rie *RoamingInferenceEngine) calculateEventQualityInference(event *RoamingAnalysisEvent) {
	// Calculate event quality based on inference analysis

	signalImprovement := event.SignalAfter - event.SignalBefore

	if signalImprovement > 15 && event.Confidence > 0.8 {
		event.Quality = QualityExcellent
	} else if signalImprovement > 5 && event.Confidence > 0.6 {
		event.Quality = QualityGood
	} else if event.Confidence > 0.4 {
		event.Quality = QualityFair
	} else {
		event.Quality = QualityPoor
	}
}

func (rie *RoamingInferenceEngine) buildRoamingContext() RoamingContext {
	// Build context information for the roaming event

	now := time.Now()

	return RoamingContext{
		NetworkLoad:       0.5, // TODO: Get actual network load
		APUtilization:     make(map[string]float64),
		InterferenceLevel: 0,
		ClientCount:       len(rie.clientStates),
		TimeOfDay:         fmt.Sprintf("%02d:00", now.Hour()),
		DayOfWeek:         now.Weekday().String(),
	}
}

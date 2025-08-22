package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

// IntentClassifier handles classification of user input to workflow intents
type IntentClassifier struct {
	intents              map[string]IntentDefinition
	intentToWorkflow     map[string]string // "primary/secondary" -> workflow_id
	config               *EngineConfig
	classificationConfig *IntentClassificationConfig
	mutex                sync.RWMutex
	classificationPrompt string
}

// IntentClassificationConfig represents the configuration for intent classification
type IntentClassificationConfig struct {
	IntentCategories struct {
		PrimaryCategories   []string                       `yaml:"primary_categories"`
		SecondaryCategories map[string][]string            `yaml:"secondary_categories"`
	} `yaml:"intent_categories"`
	
	ClassificationPrompt string `yaml:"classification_prompt"`
	FallbackPrompt       string `yaml:"fallback_prompt"`
	
	ParameterPatterns struct {
		Locations []ParameterPattern `yaml:"locations"`
		Devices   []ParameterPattern `yaml:"devices"`
		Severity  []ParameterPattern `yaml:"severity"`
	} `yaml:"parameter_patterns"`
	
	ConfidenceThresholds struct {
		High     float64 `yaml:"high"`
		Medium   float64 `yaml:"medium"`
		Low      float64 `yaml:"low"`
		Fallback float64 `yaml:"fallback"`
	} `yaml:"confidence_thresholds"`
	
	IntentWorkflowMapping map[string]map[string]string `yaml:"intent_workflow_mapping"`
	
	Fallback struct {
		DefaultWorkflow    string  `yaml:"default_workflow"`
		MinConfidence      float64 `yaml:"min_confidence"`
		MaxRetries         int     `yaml:"max_retries"`
		EnableManualOverride bool  `yaml:"enable_manual_override"`
	} `yaml:"fallback"`
}

// ParameterPattern defines a pattern for extracting parameters
type ParameterPattern struct {
	Pattern  string   `yaml:"pattern"`
	Examples []string `yaml:"examples,omitempty"`
	Level    string   `yaml:"level,omitempty"` // for severity patterns
}

// IntentDefinition defines an intent category
type IntentDefinition struct {
	Primary     string         `yaml:"primary" json:"primary"`
	Secondary   string         `yaml:"secondary" json:"secondary"`
	Description string         `yaml:"description" json:"description"`
	Keywords    []string       `yaml:"keywords" json:"keywords"`
	Examples    []string       `yaml:"examples" json:"examples"`
	WorkflowID  string         `yaml:"workflow_id" json:"workflow_id"`
	Parameters  []ParameterDef `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// ParameterDef defines extractable parameters for an intent
type ParameterDef struct {
	Name        string   `yaml:"name" json:"name"`
	Type        string   `yaml:"type" json:"type"` // string, number, boolean
	Description string   `yaml:"description" json:"description"`
	Required    bool     `yaml:"required" json:"required"`
	Examples    []string `yaml:"examples,omitempty" json:"examples,omitempty"`
}

// NewIntentClassifier creates a new intent classifier
func NewIntentClassifier(config *EngineConfig) *IntentClassifier {
	return &IntentClassifier{
		intents:              make(map[string]IntentDefinition),
		intentToWorkflow:     make(map[string]string),
		config:               config,
		classificationPrompt: buildDefaultClassificationPrompt(),
	}
}

// LoadIntentDefinitions loads intent definitions from configuration
func (ic *IntentClassifier) LoadIntentDefinitions(configPath string) error {
	ic.mutex.Lock()
	defer ic.mutex.Unlock()

	// Load from YAML configuration file
	config, err := ic.loadIntentClassificationConfig(configPath)
	if err != nil {
		// Fallback to built-in intent definitions
		intents := ic.getBuiltInIntentDefinitions()
		for _, intent := range intents {
			key := fmt.Sprintf("%s/%s", intent.Primary, intent.Secondary)
			ic.intents[key] = intent
			ic.intentToWorkflow[key] = intent.WorkflowID
		}
		return fmt.Errorf("failed to load intent configuration, using built-in definitions: %w", err)
	}

	ic.classificationConfig = config

	// Clear existing mappings
	ic.intents = make(map[string]IntentDefinition)
	ic.intentToWorkflow = make(map[string]string)

	// Build intent definitions from configuration
	for primaryIntent, secondaryIntents := range config.IntentWorkflowMapping {
		for secondaryIntent, workflowID := range secondaryIntents {
			intent := IntentDefinition{
				Primary:    primaryIntent,
				Secondary:  secondaryIntent,
				WorkflowID: workflowID,
			}
			
			key := fmt.Sprintf("%s/%s", primaryIntent, secondaryIntent)
			ic.intents[key] = intent
			ic.intentToWorkflow[key] = workflowID
		}
	}

	// Update classification prompt
	ic.classificationPrompt = ic.buildClassificationPromptFromConfig(config)

	return nil
}

// loadIntentClassificationConfig loads the intent classification configuration from file
func (ic *IntentClassifier) loadIntentClassificationConfig(configPath string) (*IntentClassificationConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read intent classification config file: %w", err)
	}

	var config IntentClassificationConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse intent classification config: %w", err)
	}

	return &config, nil
}

// buildClassificationPromptFromConfig builds the classification prompt from configuration
func (ic *IntentClassifier) buildClassificationPromptFromConfig(config *IntentClassificationConfig) string {
	// Format the prompt template with actual categories
	primaryCategoriesStr := strings.Join(config.IntentCategories.PrimaryCategories, ", ")
	
	var secondaryCategoriesStr []string
	for primary, secondaries := range config.IntentCategories.SecondaryCategories {
		secondaryCategoriesStr = append(secondaryCategoriesStr, 
			fmt.Sprintf("%s: %s", primary, strings.Join(secondaries, ", ")))
	}
	
	prompt := strings.ReplaceAll(config.ClassificationPrompt, "{primary_categories}", primaryCategoriesStr)
	prompt = strings.ReplaceAll(prompt, "{secondary_categories}", strings.Join(secondaryCategoriesStr, "\n  "))
	
	return prompt
}

// Risk Mitigation Helper Methods (MIDDLE_LAYER_PLAN.md Section 8)

// handleManualOverride handles manual intent specification (Risk Mitigation 8.1)
func (ic *IntentClassifier) handleManualOverride(req *IntentRequest) (*IntentResponse, error) {
	override := req.ManualIntentOverride
	
	// Validate the manual override
	intentKey := fmt.Sprintf("%s/%s", override.PrimaryIntent, override.SecondaryIntent)
	workflowID, exists := ic.intentToWorkflow[intentKey]
	
	if !exists {
		return nil, fmt.Errorf("invalid manual intent override: %s/%s not found", 
			override.PrimaryIntent, override.SecondaryIntent)
	}
	
	return &IntentResponse{
		PrimaryIntent:   override.PrimaryIntent,
		SecondaryIntent: override.SecondaryIntent,
		Confidence:      1.0, // Manual override has maximum confidence
		Parameters:      override.Parameters,
		WorkflowID:      workflowID,
		Reasoning:       fmt.Sprintf("Manual intent override: %s", override.Reason),
	}, nil
}

// getMaxRetries returns maximum retry attempts from configuration
func (ic *IntentClassifier) getMaxRetries() int {
	if ic.classificationConfig != nil {
		return ic.classificationConfig.Fallback.MaxRetries
	}
	return 2 // default
}

// getMinConfidence returns minimum confidence threshold
func (ic *IntentClassifier) getMinConfidence() float64 {
	if ic.classificationConfig != nil {
		return ic.classificationConfig.Fallback.MinConfidence
	}
	return ic.config.ConfidenceThreshold
}

// isHighConfidence checks if confidence is high enough for immediate acceptance
func (ic *IntentClassifier) isHighConfidence(confidence float64) bool {
	if ic.classificationConfig != nil {
		return confidence >= ic.classificationConfig.ConfidenceThresholds.High
	}
	return confidence >= 0.9
}

// isMediumConfidence checks if confidence is medium (proceed with caution)
func (ic *IntentClassifier) isMediumConfidence(confidence float64) bool {
	if ic.classificationConfig != nil {
		return confidence >= ic.classificationConfig.ConfidenceThresholds.Medium
	}
	return confidence >= 0.7
}

// enrichContextForRetry adds additional context for LLM retry attempts
func (ic *IntentClassifier) enrichContextForRetry(ctx context.Context, userInput string, attempt int) context.Context {
	// Add retry context information
	enrichedCtx := context.WithValue(ctx, "retry_attempt", attempt)
	enrichedCtx = context.WithValue(enrichedCtx, "previous_failures", attempt-1)
	enrichedCtx = context.WithValue(enrichedCtx, "retry_prompt_enhancement", 
		fmt.Sprintf("This is retry attempt %d. Please be more specific in classification.", attempt))
	return enrichedCtx
}

// extractParametersWithPatterns extracts parameters using configured patterns
func (ic *IntentClassifier) extractParametersWithPatterns(userInput string) map[string]interface{} {
	params := make(map[string]interface{})
	
	if ic.classificationConfig == nil {
		return ic.extractParameters(userInput)
	}
	
	// Use configured patterns to extract parameters
	// This is a simplified implementation - could be extended with regex patterns
	lowerInput := strings.ToLower(userInput)
	
	// Extract locations
	if strings.Contains(lowerInput, "bedroom") || strings.Contains(lowerInput, "bed room") {
		params["location1"] = "bedroom"
	}
	if strings.Contains(lowerInput, "living room") || strings.Contains(lowerInput, "livingroom") {
		params["location1"] = "living room"
	}
	if strings.Contains(lowerInput, "kitchen") {
		params["location1"] = "kitchen"
	}
	
	// Extract device types
	if strings.Contains(lowerInput, "phone") || strings.Contains(lowerInput, "smartphone") {
		params["device_type"] = "smartphone"
	}
	if strings.Contains(lowerInput, "laptop") || strings.Contains(lowerInput, "computer") {
		params["device_type"] = "laptop"
	}
	
	// Extract severity
	if strings.Contains(lowerInput, "very slow") || strings.Contains(lowerInput, "extremely slow") {
		params["severity"] = "critical"
	} else if strings.Contains(lowerInput, "slow") || strings.Contains(lowerInput, "poor") {
		params["severity"] = "moderate"
	}
	
	return params
}

// createFallbackResponse creates a fallback response with dynamic parameters (Risk Mitigation 8.2)
func (ic *IntentClassifier) createFallbackResponse(req *IntentRequest) *IntentResponse {
	fallbackWorkflow := ic.config.FallbackWorkflow
	
	if ic.classificationConfig != nil {
		fallbackWorkflow = ic.classificationConfig.Fallback.DefaultWorkflow
	}
	
	// Extract as many parameters as possible for dynamic injection
	params := ic.extractParametersWithPatterns(req.UserInput)
	params["original_input"] = req.UserInput
	params["fallback_reason"] = "Unable to classify with sufficient confidence"
	
	// Include context if available
	if req.Context != nil {
		for k, v := range req.Context {
			params["context_"+k] = v
		}
	}
	
	return &IntentResponse{
		PrimaryIntent:   "general",
		SecondaryIntent: "network_diagnosis",
		Confidence:      0.3, // Low confidence indicates fallback
		Parameters:      params,
		WorkflowID:      fallbackWorkflow,
		Reasoning:       "Fallback to general workflow due to low classification confidence",
	}
}

// ClassifyIntent classifies user input to determine intent and workflow
// Implements risk mitigation mechanisms from MIDDLE_LAYER_PLAN.md Section 8
func (ic *IntentClassifier) ClassifyIntent(ctx context.Context, req *IntentRequest) (*IntentResponse, error) {
	ic.mutex.RLock()
	defer ic.mutex.RUnlock()

	// Risk Mitigation 8.1: Confidence threshold mechanism
	var result *IntentResponse
	var err error

	// Check for manual intent override (Risk Mitigation 8.1)
	if req.ManualIntentOverride != nil {
		return ic.handleManualOverride(req)
	}

	// Try LLM-based classification with retry mechanism
	maxRetries := ic.getMaxRetries()
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err = ic.classifyWithLLM(ctx, req.UserInput)
		
		if err == nil {
			// Apply confidence threshold checks
			if ic.isHighConfidence(result.Confidence) {
				result.Reasoning = fmt.Sprintf("High confidence LLM classification (%.2f)", result.Confidence)
				return result, nil
			}
			
			if ic.isMediumConfidence(result.Confidence) {
				// Medium confidence - add warning but proceed
				result.Reasoning = fmt.Sprintf("Medium confidence LLM classification (%.2f) - consider verification", result.Confidence)
				return result, nil
			}
		}
		
		// If not the last attempt, retry with modified prompt
		if attempt < maxRetries {
			ctx = ic.enrichContextForRetry(ctx, req.UserInput, attempt)
		}
	}

	// Fallback to rule-based classification
	ruleIntent, ruleConfidence := ic.classifyWithRules(req.UserInput)

	// Risk Mitigation 8.1: Apply confidence thresholds
	if ruleConfidence >= ic.getMinConfidence() {
		intentKey := fmt.Sprintf("%s/%s", ruleIntent.Primary, ruleIntent.Secondary)
		workflowID, exists := ic.intentToWorkflow[intentKey]
		
		if exists {
			return &IntentResponse{
				PrimaryIntent:   ruleIntent.Primary,
				SecondaryIntent: ruleIntent.Secondary,
				Confidence:      ruleConfidence,
				Parameters:      ic.extractParametersWithPatterns(req.UserInput),
				WorkflowID:      workflowID,
				Reasoning:       fmt.Sprintf("Rule-based classification fallback (%.2f confidence)", ruleConfidence),
			}, nil
		}
	}

	// Risk Mitigation 8.2: Fallback to general workflow with dynamic parameters
	return ic.createFallbackResponse(req), nil
}

// ValidateIntent validates an intent response
func (ic *IntentClassifier) ValidateIntent(intent *IntentResponse) error {
	if intent == nil {
		return fmt.Errorf("intent response is nil")
	}

	if intent.PrimaryIntent == "" {
		return fmt.Errorf("primary intent cannot be empty")
	}

	if intent.SecondaryIntent == "" {
		return fmt.Errorf("secondary intent cannot be empty")
	}

	if intent.Confidence < 0 || intent.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got %f", intent.Confidence)
	}

	return nil
}

// classifyWithRules performs rule-based intent classification
func (ic *IntentClassifier) classifyWithRules(userInput string) (IntentDefinition, float64) {
	userInputLower := strings.ToLower(userInput)
	bestMatch := IntentDefinition{
		Primary:    "general",
		Secondary:  "network_diagnosis",
		WorkflowID: ic.config.FallbackWorkflow,
	}
	bestScore := 0.0

	for _, intent := range ic.intents {
		score := ic.calculateKeywordMatchScore(userInputLower, intent.Keywords)
		if score > bestScore {
			bestScore = score
			bestMatch = intent
		}
	}

	// Apply confidence scaling
	confidence := bestScore * 0.8 // Scale down to account for rule-based uncertainty
	if confidence > 0.95 {
		confidence = 0.95 // Cap at 95% for rule-based classification
	}

	return bestMatch, confidence
}

// calculateKeywordMatchScore calculates matching score based on keywords
func (ic *IntentClassifier) calculateKeywordMatchScore(userInput string, keywords []string) float64 {
	if len(keywords) == 0 {
		return 0.0
	}

	matches := 0
	for _, keyword := range keywords {
		if strings.Contains(userInput, strings.ToLower(keyword)) {
			matches++
		}
	}

	return float64(matches) / float64(len(keywords))
}

// extractParameters extracts parameters from user input
func (ic *IntentClassifier) extractParameters(userInput string) map[string]interface{} {
	params := make(map[string]interface{})

	// Simple parameter extraction for locations
	// This could be enhanced with NLP or regex patterns
	userInputLower := strings.ToLower(userInput)

	// Extract location references
	locationKeywords := []string{"living room", "bedroom", "kitchen", "bathroom", "garage", "office", "basement", "attic", "hallway", "dining room"}
	var locations []string

	for _, location := range locationKeywords {
		if strings.Contains(userInputLower, location) {
			locations = append(locations, location)
		}
	}

	if len(locations) >= 1 {
		params["location1"] = locations[0]
	}
	if len(locations) >= 2 {
		params["location2"] = locations[1]
	}

	// Extract severity indicators
	if strings.Contains(userInputLower, "dead") || strings.Contains(userInputLower, "no signal") {
		params["severity"] = "critical"
	} else if strings.Contains(userInputLower, "slow") || strings.Contains(userInputLower, "weak") {
		params["severity"] = "moderate"
	} else if strings.Contains(userInputLower, "sometimes") || strings.Contains(userInputLower, "occasionally") {
		params["severity"] = "minor"
	}

	// Extract device types
	deviceKeywords := map[string]string{
		"phone":      "smartphone",
		"smartphone": "smartphone",
		"laptop":     "laptop",
		"computer":   "laptop",
		"tablet":     "tablet",
		"tv":         "smart_tv",
		"smart tv":   "smart_tv",
	}

	for keyword, deviceType := range deviceKeywords {
		if strings.Contains(userInputLower, keyword) {
			params["device_type"] = deviceType
			break
		}
	}

	return params
}

// getBuiltInIntentDefinitions returns built-in intent definitions
func (ic *IntentClassifier) getBuiltInIntentDefinitions() []IntentDefinition {
	return []IntentDefinition{
		{
			Primary:     "connectivity_issues",
			Secondary:   "wan_connectivity",
			Description: "WAN connectivity problems - no internet access",
			Keywords:    []string{"no internet", "can't connect", "no connection", "wan down", "internet down"},
			WorkflowID:  "wan_connectivity_diagnosis",
			Parameters: []ParameterDef{
				{Name: "severity", Type: "string", Description: "Severity of the connectivity issue"},
			},
		},
		{
			Primary:     "connectivity_issues",
			Secondary:   "wifi_disconnection",
			Description: "WiFi disconnection and stability issues",
			Keywords:    []string{"wifi disconnect", "keeps disconnecting", "unstable wifi", "wifi drops"},
			WorkflowID:  "wifi_stability_diagnosis",
			Parameters: []ParameterDef{
				{Name: "device_type", Type: "string", Description: "Type of device experiencing issues"},
				{Name: "frequency", Type: "string", Description: "How often disconnections occur"},
			},
		},
		{
			Primary:     "coverage_issues",
			Secondary:   "weak_signal_coverage",
			Description: "WiFi weak signal and coverage problems",
			Keywords:    []string{"weak signal", "poor coverage", "dead zone", "no wifi", "wifi weak", "signal strength"},
			WorkflowID:  "weak_signal_coverage_diagnosis",
			Parameters: []ParameterDef{
				{Name: "location1", Type: "string", Description: "First location for comparison", Required: true},
				{Name: "location2", Type: "string", Description: "Second location for comparison"},
				{Name: "severity", Type: "string", Description: "Severity of coverage issue"},
			},
		},
		{
			Primary:     "coverage_issues",
			Secondary:   "dead_zones",
			Description: "Complete coverage dead zones",
			Keywords:    []string{"dead zone", "no signal", "no coverage", "completely dead"},
			WorkflowID:  "dead_zone_analysis",
		},
		{
			Primary:     "performance_problems",
			Secondary:   "slow_internet",
			Description: "Internet speed and throughput issues",
			Keywords:    []string{"slow internet", "slow speed", "low bandwidth", "sluggish", "slow download", "slow upload"},
			WorkflowID:  "performance_bottleneck_analysis",
			Parameters: []ParameterDef{
				{Name: "speed_type", Type: "string", Description: "Type of speed issue (download/upload/both)"},
				{Name: "device_type", Type: "string", Description: "Device experiencing slow speeds"},
			},
		},
		{
			Primary:     "performance_problems",
			Secondary:   "high_latency",
			Description: "Network latency and lag issues",
			Keywords:    []string{"lag", "latency", "delay", "slow response", "ping issues"},
			WorkflowID:  "latency_diagnosis",
		},
		{
			Primary:     "device_issues",
			Secondary:   "device_offline",
			Description: "Devices showing offline or not responding",
			Keywords:    []string{"device offline", "not responding", "can't reach", "device down"},
			WorkflowID:  "device_connectivity_diagnosis",
			Parameters: []ParameterDef{
				{Name: "device_id", Type: "string", Description: "ID or name of the offline device"},
			},
		},
	}
}

// buildDefaultClassificationPrompt builds the default LLM classification prompt
func buildDefaultClassificationPrompt() string {
	return `You are a network diagnostic intent classifier. Analyze user input and return standardized intent classification.

User Input: "{user_input}"

Available Intent Categories:
{intent_categories}

Return JSON format:
{
  "primary_intent": "main_category_of_the_issue",
  "secondary_intent": "specific_subcategory", 
  "confidence": 0.95,
  "parameters": {
    "location1": "extracted_location_1_from_user_input",
    "location2": "extracted_location_2_from_user_input",
    "severity": "extracted_severity_level",
    "device_type": "extracted_device_type"
  },
  "reasoning": "brief_explanation_of_classification"
}`
}

// LLMClient interface for LLM API calls
type LLMClient interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// SimpleLLMClient is a placeholder implementation
type SimpleLLMClient struct {
	// This would connect to actual LLM API (OpenAI, Anthropic, etc.)
	endpoint string
	apiKey   string
}

// NewSimpleLLMClient creates a new LLM client
func NewSimpleLLMClient(endpoint, apiKey string) *SimpleLLMClient {
	return &SimpleLLMClient{
		endpoint: endpoint,
		apiKey:   apiKey,
	}
}

// Complete makes API call to LLM service
func (client *SimpleLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	// Placeholder implementation - in production this would make HTTP requests
	// to actual LLM APIs like OpenAI GPT-4, Anthropic Claude, etc.

	// For now, simulate LLM response for common patterns
	userInput := strings.ToLower(prompt)

	if strings.Contains(userInput, "weak") && (strings.Contains(userInput, "signal") || strings.Contains(userInput, "wifi")) {
		return `{
			"primary_intent": "coverage_issues",
			"secondary_intent": "weak_signal_coverage", 
			"confidence": 0.92,
			"parameters": {
				"severity": "moderate"
			},
			"reasoning": "User mentions weak WiFi signal, indicating coverage issues"
		}`, nil
	}

	if strings.Contains(userInput, "slow") && strings.Contains(userInput, "internet") {
		return `{
			"primary_intent": "performance_problems",
			"secondary_intent": "slow_internet",
			"confidence": 0.89,
			"parameters": {
				"speed_type": "both"
			},
			"reasoning": "User reports slow internet speeds"
		}`, nil
	}

	if strings.Contains(userInput, "disconnect") || strings.Contains(userInput, "connection") {
		return `{
			"primary_intent": "connectivity_issues",
			"secondary_intent": "wan_connectivity",
			"confidence": 0.85,
			"parameters": {},
			"reasoning": "User reports connectivity issues"
		}`, nil
	}

	// Fallback to general diagnosis
	return `{
		"primary_intent": "general",
		"secondary_intent": "network_diagnosis",
		"confidence": 0.70,
		"parameters": {},
		"reasoning": "Unable to classify specific intent, using general diagnosis"
	}`, nil
}

// classifyWithLLM performs LLM-based intent classification
func (ic *IntentClassifier) classifyWithLLM(ctx context.Context, userInput string) (*IntentResponse, error) {
	// Build prompt with available intents
	intentCategories := ic.buildIntentCategoriesString()
	prompt := strings.ReplaceAll(ic.classificationPrompt, "{user_input}", userInput)
	prompt = strings.ReplaceAll(prompt, "{intent_categories}", intentCategories)

	// Use simple LLM client (in production, this would be a real LLM API)
	client := NewSimpleLLMClient("", "") // Placeholder

	response, err := client.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}

	// Parse JSON response
	var intentResp IntentResponse
	if err := json.Unmarshal([]byte(response), &intentResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Validate and find corresponding workflow
	if err := ic.ValidateIntent(&intentResp); err != nil {
		return nil, fmt.Errorf("intent validation failed: %w", err)
	}

	// Get workflow ID for this intent
	intentKey := fmt.Sprintf("%s/%s", intentResp.PrimaryIntent, intentResp.SecondaryIntent)
	workflowID, exists := ic.intentToWorkflow[intentKey]
	if !exists {
		workflowID = ic.config.FallbackWorkflow
	}

	intentResp.WorkflowID = workflowID
	return &intentResp, nil
}

// buildIntentCategoriesString builds a string representation of available intent categories
func (ic *IntentClassifier) buildIntentCategoriesString() string {
	var categories []string

	for _, intent := range ic.intents {
		category := fmt.Sprintf("- %s/%s: %s", intent.Primary, intent.Secondary, intent.Description)
		categories = append(categories, category)
	}

	return strings.Join(categories, "\n")
}

// GetAvailableIntents returns all available intent definitions
func (ic *IntentClassifier) GetAvailableIntents() map[string]IntentDefinition {
	ic.mutex.RLock()
	defer ic.mutex.RUnlock()

	// Return a copy to prevent external modification
	intents := make(map[string]IntentDefinition)
	for k, v := range ic.intents {
		intents[k] = v
	}
	return intents
}

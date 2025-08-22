package workflow

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"rtk_controller/internal/storage"

	"gopkg.in/yaml.v2"
)

// WorkflowRegistry manages workflow definitions and registration
type WorkflowRegistry struct {
	workflows        map[string]*Workflow
	intentToWorkflow map[string]string // "primary/secondary" -> workflow_id
	storage          storage.Storage
	mutex            sync.RWMutex
}

// NewWorkflowRegistry creates a new workflow registry
func NewWorkflowRegistry(storage storage.Storage) *WorkflowRegistry {
	return &WorkflowRegistry{
		workflows:        make(map[string]*Workflow),
		intentToWorkflow: make(map[string]string),
		storage:          storage,
	}
}

// WorkflowConfig represents the YAML configuration structure
type WorkflowConfig struct {
	EngineConfig struct {
		DefaultTimeout         string  `yaml:"default_timeout"`
		MaxConcurrentWorkflows int     `yaml:"max_concurrent_workflows"`
		RetryAttempts          int     `yaml:"retry_attempts"`
		ConfidenceThreshold    float64 `yaml:"confidence_threshold"`
		FallbackWorkflow       string  `yaml:"fallback_workflow"`
	} `yaml:"engine_config"`
	Intents struct {
		ClassificationModel string  `yaml:"classification_model"`
		ConfidenceThreshold float64 `yaml:"confidence_threshold"`
		FallbackWorkflow    string  `yaml:"fallback_workflow"`
	} `yaml:"intents"`
	Workflows []Workflow `yaml:"workflows"`
}

// LoadWorkflows loads workflows from YAML configuration file
func (wr *WorkflowRegistry) LoadWorkflows(configPath string) error {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	// Use default config path if not provided
	if configPath == "" {
		configPath = "configs/workflows.yaml"
	}

	// Try to load from file first, fallback to built-in if file doesn't exist
	var workflows []*Workflow
	var err error

	workflows, err = wr.loadWorkflowsFromFile(configPath)
	if err != nil {
		// Fallback to built-in workflows
		fmt.Printf("Warning: Could not load workflows from %s, using built-in workflows: %v\n", configPath, err)
		workflows = wr.getBuiltInWorkflows()
	}

	// Clear existing workflows
	wr.workflows = make(map[string]*Workflow)
	wr.intentToWorkflow = make(map[string]string)

	// Register workflows
	for _, workflow := range workflows {
		if err := wr.registerWorkflow(workflow); err != nil {
			return fmt.Errorf("failed to register workflow %s: %w", workflow.ID, err)
		}
	}

	return nil
}

// loadWorkflowsFromFile loads workflows from YAML file
func (wr *WorkflowRegistry) loadWorkflowsFromFile(configPath string) ([]*Workflow, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow config file: %w", err)
	}

	var config WorkflowConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse workflow config: %w", err)
	}

	// Convert to workflow pointers
	workflows := make([]*Workflow, len(config.Workflows))
	for i := range config.Workflows {
		workflows[i] = &config.Workflows[i]
	}

	return workflows, nil
}

// GetWorkflow retrieves a workflow by ID
func (wr *WorkflowRegistry) GetWorkflow(id string) (*Workflow, error) {
	wr.mutex.RLock()
	defer wr.mutex.RUnlock()

	workflow, exists := wr.workflows[id]
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", id)
	}

	// Return a copy to prevent external modification
	workflowCopy := *workflow
	return &workflowCopy, nil
}

// GetWorkflowForIntent returns workflow ID for given intent
func (wr *WorkflowRegistry) GetWorkflowForIntent(primary, secondary string) (string, bool) {
	wr.mutex.RLock()
	defer wr.mutex.RUnlock()

	key := fmt.Sprintf("%s/%s", primary, secondary)
	workflowID, exists := wr.intentToWorkflow[key]
	return workflowID, exists
}

// ListWorkflows returns all registered workflow IDs
func (wr *WorkflowRegistry) ListWorkflows() []string {
	wr.mutex.RLock()
	defer wr.mutex.RUnlock()

	workflows := make([]string, 0, len(wr.workflows))
	for id := range wr.workflows {
		workflows = append(workflows, id)
	}
	return workflows
}

// RegisterWorkflow registers a new workflow
func (wr *WorkflowRegistry) RegisterWorkflow(workflow *Workflow) error {
	wr.mutex.Lock()
	defer wr.mutex.Unlock()

	return wr.registerWorkflow(workflow)
}

// registerWorkflow registers a workflow (internal method, assumes lock is held)
func (wr *WorkflowRegistry) registerWorkflow(workflow *Workflow) error {
	if workflow == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	if workflow.ID == "" {
		return fmt.Errorf("workflow ID cannot be empty")
	}

	// Validate workflow
	if err := wr.ValidateWorkflow(workflow); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	// Check for duplicate
	if _, exists := wr.workflows[workflow.ID]; exists {
		return fmt.Errorf("workflow %s already exists", workflow.ID)
	}

	// Set metadata timestamps
	now := time.Now()
	if workflow.Metadata.CreatedAt.IsZero() {
		workflow.Metadata.CreatedAt = now
	}
	workflow.Metadata.UpdatedAt = now

	// Register workflow
	wr.workflows[workflow.ID] = workflow

	// Register intent mapping
	if workflow.Intent.Primary != "" && workflow.Intent.Secondary != "" {
		intentKey := fmt.Sprintf("%s/%s", workflow.Intent.Primary, workflow.Intent.Secondary)
		wr.intentToWorkflow[intentKey] = workflow.ID
	}

	// Store in persistent storage if available
	if wr.storage != nil {
		wr.storeWorkflow(workflow)
	}

	return nil
}

// ValidateWorkflow validates a workflow definition
func (wr *WorkflowRegistry) ValidateWorkflow(workflow *Workflow) error {
	if workflow == nil {
		return fmt.Errorf("workflow cannot be nil")
	}

	if workflow.ID == "" {
		return fmt.Errorf("workflow ID is required")
	}

	if workflow.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(workflow.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	// Validate steps
	stepIDs := make(map[string]bool)
	for i, step := range workflow.Steps {
		if err := wr.validateWorkflowStep(step, stepIDs); err != nil {
			return fmt.Errorf("step %d validation failed: %w", i, err)
		}
	}

	// Validate intent mapping
	if workflow.Intent.Primary == "" || workflow.Intent.Secondary == "" {
		return fmt.Errorf("workflow intent mapping is incomplete")
	}

	return nil
}

// validateWorkflowStep validates a single workflow step
func (wr *WorkflowRegistry) validateWorkflowStep(step WorkflowStep, stepIDs map[string]bool) error {
	if step.ID == "" {
		return fmt.Errorf("step ID is required")
	}

	// Check for duplicate step IDs
	if stepIDs[step.ID] {
		return fmt.Errorf("duplicate step ID: %s", step.ID)
	}
	stepIDs[step.ID] = true

	// Validate step type
	switch step.Type {
	case StepTypeTool:
		if step.ToolName == "" {
			return fmt.Errorf("tool_call step must have tool_name")
		}
	case StepTypeParallel, StepTypeSequential:
		if len(step.SubSteps) == 0 {
			return fmt.Errorf("%s step must have sub_steps", step.Type)
		}
		// Validate sub-steps recursively
		for i, subStep := range step.SubSteps {
			if err := wr.validateWorkflowStep(subStep, stepIDs); err != nil {
				return fmt.Errorf("sub-step %d validation failed: %w", i, err)
			}
		}
	case StepTypeCondition:
		if step.Condition == nil {
			return fmt.Errorf("condition step must have condition")
		}
		if err := wr.validateStepCondition(*step.Condition); err != nil {
			return fmt.Errorf("condition validation failed: %w", err)
		}
	default:
		return fmt.Errorf("unknown step type: %s", step.Type)
	}

	// Validate timeout
	if step.Timeout != nil && *step.Timeout <= 0 {
		return fmt.Errorf("step timeout must be positive")
	}

	// Validate retry configuration
	if step.Retry != nil {
		if step.Retry.MaxAttempts < 1 {
			return fmt.Errorf("retry max_attempts must be at least 1")
		}
		if step.Retry.BackoffMs < 0 {
			return fmt.Errorf("retry backoff_ms cannot be negative")
		}
	}

	return nil
}

// validateStepCondition validates a step condition
func (wr *WorkflowRegistry) validateStepCondition(condition StepCondition) error {
	if condition.Field == "" {
		return fmt.Errorf("condition field is required")
	}

	if condition.Operator == "" {
		return fmt.Errorf("condition operator is required")
	}

	// Validate operator
	validOperators := []string{"eq", "ne", "gt", "lt", "gte", "lte", "exists", "not_exists", "contains"}
	validOperator := false
	for _, op := range validOperators {
		if condition.Operator == op {
			validOperator = true
			break
		}
	}

	if !validOperator {
		return fmt.Errorf("invalid condition operator: %s", condition.Operator)
	}

	// Validate value for operators that require it
	if condition.Operator != "exists" && condition.Operator != "not_exists" && condition.Value == nil {
		return fmt.Errorf("condition value is required for operator: %s", condition.Operator)
	}

	return nil
}

// storeWorkflow stores workflow in persistent storage
func (wr *WorkflowRegistry) storeWorkflow(workflow *Workflow) {
	if wr.storage == nil {
		return
	}

	key := fmt.Sprintf("workflow_def:%s", workflow.ID)
	data, err := yaml.Marshal(workflow)
	if err != nil {
		// Log error but don't fail registration
		return
	}

	wr.storage.Set(key, string(data))
}

// loadWorkflowFromStorage loads workflow from persistent storage
func (wr *WorkflowRegistry) loadWorkflowFromStorage(workflowID string) (*Workflow, error) {
	if wr.storage == nil {
		return nil, fmt.Errorf("storage not available")
	}

	key := fmt.Sprintf("workflow_def:%s", workflowID)
	data, err := wr.storage.Get(key)
	if err != nil {
		return nil, fmt.Errorf("workflow not found in storage: %w", err)
	}

	var workflow Workflow
	if err := yaml.Unmarshal([]byte(data), &workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow: %w", err)
	}

	return &workflow, nil
}

// GetAllWorkflows returns all registered workflows
func (wr *WorkflowRegistry) GetAllWorkflows() map[string]*Workflow {
	wr.mutex.RLock()
	defer wr.mutex.RUnlock()

	// Return a copy to prevent external modification
	workflows := make(map[string]*Workflow)
	for id, workflow := range wr.workflows {
		workflowCopy := *workflow
		workflows[id] = &workflowCopy
	}

	return workflows
}

// getBuiltInWorkflows returns built-in workflow definitions
func (wr *WorkflowRegistry) getBuiltInWorkflows() []*Workflow {
	now := time.Now()
	return []*Workflow{
		{
			ID:          "weak_signal_coverage_diagnosis",
			Name:        "WiFi Weak Signal Coverage Diagnosis",
			Description: "Comprehensive WiFi coverage analysis identifying weak signal areas, interference sources, and coverage gaps with optimization recommendations",
			Intent: IntentMapping{
				Primary:   "coverage_issues",
				Secondary: "weak_signal_coverage",
			},
			Steps: []WorkflowStep{
				{
					ID:   "parallel_data_collection",
					Name: "Parallel Data Collection",
					Type: StepTypeParallel,
					SubSteps: []WorkflowStep{
						{
							ID:       "topology_scan",
							Name:     "Network Topology Scan",
							Type:     StepTypeTool,
							ToolName: "topology.get_full",
							Parameters: map[string]interface{}{
								"include_wireless": true,
								"depth":            "full",
							},
						},
						{
							ID:       "wifi_analysis",
							Name:     "WiFi Channel Analysis",
							Type:     StepTypeTool,
							ToolName: "wifi.analyze_channels",
							Parameters: map[string]interface{}{
								"scan_duration":     30,
								"include_neighbors": true,
							},
						},
						{
							ID:       "signal_strength_map",
							Name:     "Signal Strength Mapping",
							Type:     StepTypeTool,
							ToolName: "wifi.signal_strength_survey",
							Parameters: map[string]interface{}{
								"locations": []string{"${location1}", "${location2}"},
								"duration":  60,
							},
						},
						{
							ID:       "interference_scan",
							Name:     "Interference Detection",
							Type:     StepTypeTool,
							ToolName: "wifi.interference_detection",
							Parameters: map[string]interface{}{
								"scan_duration":   45,
								"frequency_bands": []string{"2.4GHz", "5GHz"},
							},
						},
					},
				},
				{
					ID:   "conditional_deep_analysis",
					Name: "Conditional Deep Analysis",
					Type: StepTypeCondition,
					Condition: &StepCondition{
						Field:    "signal_strength_map.min_rssi",
						Operator: "lt",
						Value:    -70,
					},
					SubSteps: []WorkflowStep{
						{
							ID:       "detailed_interference_analysis",
							Name:     "Detailed Interference Analysis",
							Type:     StepTypeTool,
							ToolName: "wifi.detailed_interference_analysis",
							Parameters: map[string]interface{}{
								"focus_areas":    []string{"${location1}"},
								"analysis_depth": "comprehensive",
							},
						},
						{
							ID:       "channel_utilization_analysis",
							Name:     "Channel Utilization Analysis",
							Type:     StepTypeTool,
							ToolName: "wifi.channel_utilization",
							Parameters: map[string]interface{}{
								"duration": 300,
							},
						},
					},
				},
				{
					ID:       "coverage_optimization",
					Name:     "Coverage Optimization Analysis",
					Type:     StepTypeTool,
					ToolName: "wifi.coverage_optimization",
					Parameters: map[string]interface{}{
						"weak_areas":           "${signal_strength_map.weak_areas}",
						"interference_sources": "${interference_scan.sources}",
						"topology":             "${topology_scan.wireless_topology}",
					},
				},
				{
					ID:       "performance_validation",
					Name:     "Performance Validation",
					Type:     StepTypeTool,
					ToolName: "network.targeted_speedtest",
					Parameters: map[string]interface{}{
						"locations":  []string{"${location1}", "${location2}"},
						"test_types": []string{"throughput", "latency", "jitter"},
					},
				},
			},
			Metadata: WorkflowMetadata{
				Version:      "1.0.0",
				Author:       "RTK Controller",
				CreatedAt:    now,
				UpdatedAt:    now,
				Tags:         []string{"wifi", "coverage", "diagnosis", "optimization"},
				Requirements: []string{"wifi.analyze_channels", "wifi.signal_strength_survey", "wifi.interference_detection"},
			},
		},
		{
			ID:          "wan_connectivity_diagnosis",
			Name:        "WAN Connectivity Diagnosis",
			Description: "Systematic diagnosis of WAN connectivity issues including ISP connection, DNS resolution, and routing problems",
			Intent: IntentMapping{
				Primary:   "connectivity_issues",
				Secondary: "wan_connectivity",
			},
			Steps: []WorkflowStep{
				{
					ID:       "connectivity_test",
					Name:     "WAN Connectivity Test",
					Type:     StepTypeTool,
					ToolName: "diagnostics.wan_connectivity",
					Parameters: map[string]interface{}{
						"timeout":  30,
						"test_dns": true,
					},
				},
				{
					ID:   "conditional_dns_analysis",
					Name: "Conditional DNS Analysis",
					Type: StepTypeCondition,
					Condition: &StepCondition{
						Field:    "connectivity_test.dns_resolution",
						Operator: "eq",
						Value:    false,
					},
					SubSteps: []WorkflowStep{
						{
							ID:       "dns_resolution_test",
							Name:     "DNS Resolution Analysis",
							Type:     StepTypeTool,
							ToolName: "diagnostics.dns_resolution",
							Parameters: map[string]interface{}{
								"test_domains": []string{"google.com", "cloudflare.com", "8.8.8.8"},
								"dns_servers":  []string{"auto", "8.8.8.8", "1.1.1.1"},
							},
						},
					},
				},
				{
					ID:   "conditional_routing_analysis",
					Name: "Conditional Routing Analysis",
					Type: StepTypeCondition,
					Condition: &StepCondition{
						Field:    "connectivity_test.ping_success",
						Operator: "eq",
						Value:    false,
					},
					SubSteps: []WorkflowStep{
						{
							ID:       "routing_table_analysis",
							Name:     "Routing Table Analysis",
							Type:     StepTypeTool,
							ToolName: "network.routing_analysis",
							Parameters: map[string]interface{}{
								"target":             "8.8.8.8",
								"include_traceroute": true,
							},
						},
					},
				},
				{
					ID:       "isp_connectivity_test",
					Name:     "ISP Connectivity Test",
					Type:     StepTypeTool,
					ToolName: "diagnostics.isp_connectivity",
					Parameters: map[string]interface{}{
						"gateway":       "${connectivity_test.gateway}",
						"test_external": true,
					},
				},
				{
					ID:       "network_path_analysis",
					Name:     "Network Path Analysis",
					Type:     StepTypeTool,
					ToolName: "network.path_analysis",
					Parameters: map[string]interface{}{
						"destination": "8.8.8.8",
						"protocol":    "both",
					},
				},
			},
			Metadata: WorkflowMetadata{
				Version:      "1.0.0",
				Author:       "RTK Controller",
				CreatedAt:    now,
				UpdatedAt:    now,
				Tags:         []string{"wan", "connectivity", "diagnosis", "isp"},
				Requirements: []string{"diagnostics.wan_connectivity", "diagnostics.dns_resolution", "network.routing_analysis"},
			},
		},
		{
			ID:          "performance_bottleneck_analysis",
			Name:        "Network Performance Bottleneck Analysis",
			Description: "Comprehensive network performance analysis identifying bottlenecks in throughput, latency, and quality of service",
			Intent: IntentMapping{
				Primary:   "performance_problems",
				Secondary: "slow_internet",
			},
			Steps: []WorkflowStep{
				{
					ID:   "baseline_performance_tests",
					Name: "Baseline Performance Tests",
					Type: StepTypeParallel,
					SubSteps: []WorkflowStep{
						{
							ID:       "speedtest_comprehensive",
							Name:     "Comprehensive Speed Test",
							Type:     StepTypeTool,
							ToolName: "network.speedtest_full",
							Parameters: map[string]interface{}{
								"servers":            []string{"auto", "nearest"},
								"test_duration":      60,
								"concurrent_streams": 4,
							},
						},
						{
							ID:       "latency_analysis",
							Name:     "Latency Analysis",
							Type:     StepTypeTool,
							ToolName: "network.latency_analysis",
							Parameters: map[string]interface{}{
								"targets":      []string{"8.8.8.8", "1.1.1.1", "gateway"},
								"packet_count": 100,
							},
						},
						{
							ID:       "bandwidth_utilization",
							Name:     "Bandwidth Utilization Check",
							Type:     StepTypeTool,
							ToolName: "network.bandwidth_utilization",
							Parameters: map[string]interface{}{
								"duration":   120,
								"interfaces": []string{"wan", "lan"},
							},
						},
					},
				},
				{
					ID:   "conditional_qos_analysis",
					Name: "Conditional QoS Analysis",
					Type: StepTypeCondition,
					Condition: &StepCondition{
						Field:    "speedtest_comprehensive.download_speed",
						Operator: "lt",
						Value:    "${expected_speed_mbps}",
					},
					SubSteps: []WorkflowStep{
						{
							ID:       "qos_policy_analysis",
							Name:     "QoS Policy Analysis",
							Type:     StepTypeTool,
							ToolName: "qos.analyze_policies",
							Parameters: map[string]interface{}{
								"include_traffic_classes": true,
							},
						},
						{
							ID:       "traffic_classification",
							Name:     "Traffic Classification Analysis",
							Type:     StepTypeTool,
							ToolName: "qos.traffic_classification",
							Parameters: map[string]interface{}{
								"duration":     180,
								"detail_level": "comprehensive",
							},
						},
					},
				},
				{
					ID:       "device_performance_analysis",
					Name:     "Device-Specific Performance Analysis",
					Type:     StepTypeTool,
					ToolName: "network.device_performance_analysis",
					Parameters: map[string]interface{}{
						"device_type":    "${device_type}",
						"test_scenarios": []string{"single_stream", "multi_stream", "mixed_traffic"},
					},
				},
				{
					ID:       "network_congestion_analysis",
					Name:     "Network Congestion Analysis",
					Type:     StepTypeTool,
					ToolName: "network.congestion_analysis",
					Parameters: map[string]interface{}{
						"interfaces":  []string{"wan", "lan", "wifi"},
						"time_window": 300,
					},
				},
			},
			Metadata: WorkflowMetadata{
				Version:      "1.0.0",
				Author:       "RTK Controller",
				CreatedAt:    now,
				UpdatedAt:    now,
				Tags:         []string{"performance", "speed", "analysis", "bottleneck", "qos"},
				Requirements: []string{"network.speedtest_full", "network.latency_analysis", "qos.analyze_policies"},
			},
		},
		{
			ID:          "device_connectivity_diagnosis",
			Name:        "Device Connectivity Diagnosis",
			Description: "Diagnose connectivity issues for specific network devices including wireless association, IP assignment, and traffic flow",
			Intent: IntentMapping{
				Primary:   "device_issues",
				Secondary: "device_offline",
			},
			Steps: []WorkflowStep{
				{
					ID:       "device_discovery",
					Name:     "Device Discovery and Status",
					Type:     StepTypeTool,
					ToolName: "topology.device_discovery",
					Parameters: map[string]interface{}{
						"device_id":       "${device_id}",
						"include_history": true,
					},
				},
				{
					ID:   "conditional_wireless_analysis",
					Name: "Conditional Wireless Analysis",
					Type: StepTypeCondition,
					Condition: &StepCondition{
						Field:    "device_discovery.connection_type",
						Operator: "eq",
						Value:    "wireless",
					},
					SubSteps: []WorkflowStep{
						{
							ID:       "wireless_association_analysis",
							Name:     "Wireless Association Analysis",
							Type:     StepTypeTool,
							ToolName: "wifi.device_association_analysis",
							Parameters: map[string]interface{}{
								"device_mac":             "${device_discovery.mac_address}",
								"include_signal_history": true,
							},
						},
						{
							ID:       "wireless_performance_test",
							Name:     "Wireless Performance Test",
							Type:     StepTypeTool,
							ToolName: "wifi.device_performance_test",
							Parameters: map[string]interface{}{
								"device_id":     "${device_id}",
								"test_duration": 60,
							},
						},
					},
				},
				{
					ID:       "connectivity_path_trace",
					Name:     "Connectivity Path Trace",
					Type:     StepTypeTool,
					ToolName: "network.device_connectivity_trace",
					Parameters: map[string]interface{}{
						"device_ip":   "${device_discovery.ip_address}",
						"trace_route": true,
					},
				},
				{
					ID:       "dhcp_lease_analysis",
					Name:     "DHCP Lease Analysis",
					Type:     StepTypeTool,
					ToolName: "network.dhcp_lease_analysis",
					Parameters: map[string]interface{}{
						"device_mac":      "${device_discovery.mac_address}",
						"include_history": true,
					},
				},
				{
					ID:       "traffic_flow_analysis",
					Name:     "Traffic Flow Analysis",
					Type:     StepTypeTool,
					ToolName: "network.device_traffic_analysis",
					Parameters: map[string]interface{}{
						"device_id":         "${device_id}",
						"duration":          120,
						"include_protocols": true,
					},
				},
			},
			Metadata: WorkflowMetadata{
				Version:      "1.0.0",
				Author:       "RTK Controller",
				CreatedAt:    now,
				UpdatedAt:    now,
				Tags:         []string{"device", "connectivity", "diagnosis", "wireless", "dhcp"},
				Requirements: []string{"topology.device_discovery", "wifi.device_association_analysis", "network.dhcp_lease_analysis"},
			},
		},
		{
			ID:          "general_network_diagnosis",
			Name:        "General Network Diagnosis",
			Description: "General-purpose network diagnosis workflow for unclassified issues providing comprehensive network health assessment",
			Intent: IntentMapping{
				Primary:   "general",
				Secondary: "network_diagnosis",
			},
			Steps: []WorkflowStep{
				{
					ID:       "network_overview",
					Name:     "Network Overview",
					Type:     StepTypeTool,
					ToolName: "topology.get_full",
					Parameters: map[string]interface{}{
						"include_metrics": true,
						"depth":           "comprehensive",
					},
				},
				{
					ID:       "basic_connectivity_test",
					Name:     "Basic Connectivity Test",
					Type:     StepTypeTool,
					ToolName: "network.basic_connectivity_test",
					Parameters: map[string]interface{}{
						"targets": []string{"gateway", "8.8.8.8", "1.1.1.1"},
					},
				},
				{
					ID:       "wifi_health_check",
					Name:     "WiFi Health Check",
					Type:     StepTypeTool,
					ToolName: "wifi.health_check",
					Parameters: map[string]interface{}{
						"include_all_bands": true,
						"scan_neighbors":    true,
					},
				},
				{
					ID:       "performance_snapshot",
					Name:     "Performance Snapshot",
					Type:     StepTypeTool,
					ToolName: "network.performance_snapshot",
					Parameters: map[string]interface{}{
						"quick_test": true,
					},
				},
				{
					ID:       "device_status_summary",
					Name:     "Device Status Summary",
					Type:     StepTypeTool,
					ToolName: "topology.device_status_summary",
					Parameters: map[string]interface{}{
						"include_offline_devices": true,
					},
				},
			},
			Metadata: WorkflowMetadata{
				Version:      "1.0.0",
				Author:       "RTK Controller",
				CreatedAt:    now,
				UpdatedAt:    now,
				Tags:         []string{"general", "diagnosis", "fallback", "health_check"},
				Requirements: []string{"topology.get_full", "network.basic_connectivity_test", "wifi.health_check"},
			},
		},
	}
}

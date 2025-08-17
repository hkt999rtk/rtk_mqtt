package topology

import (
	"time"
)

// NetworkDiagnosticsData represents comprehensive diagnostic information for network analysis
type NetworkDiagnosticsData struct {
	// Basic information
	ID           string    `json:"id"`
	GeneratedAt  time.Time `json:"generated_at"`
	DeviceID     string    `json:"device_id"`
	DeviceName   string    `json:"device_name,omitempty"`
	TestSuite    string    `json:"test_suite"`
	
	// Test execution context
	TestContext DiagnosticTestContext `json:"test_context"`
	
	// Speed test results
	SpeedTestResults *SpeedTestResults `json:"speed_test_results,omitempty"`
	
	// WAN connectivity tests
	WANTestResults *WANConnectivityResults `json:"wan_test_results,omitempty"`
	
	// Network latency and jitter measurements
// 	LatencyResults *LatencyJitterResults `json:"latency_results,omitempty"`
// 	
// 	// Packet loss analysis
// 	PacketLossResults *PacketLossResults `json:"packet_loss_results,omitempty"`
// 	
// 	// DNS resolution tests
// 	DNSTestResults *DNSTestResults `json:"dns_test_results,omitempty"`
// 	
// 	// HTTP/HTTPS connectivity tests
// 	HTTPTestResults *HTTPConnectivityResults `json:"http_test_results,omitempty"`
// 	
// 	// Network interface diagnostics
// 	InterfaceResults *NetworkInterfaceResults `json:"interface_results,omitempty"`
// 	
// 	// Route table analysis
// 	RouteTableResults *RouteTableAnalysis `json:"route_table_results,omitempty"`
// 	
// 	// ARP table analysis
// 	ARPTableResults *ARPTableAnalysis `json:"arp_table_results,omitempty"`
// 	
// 	// Port connectivity tests
// 	PortTestResults *PortConnectivityResults `json:"port_test_results,omitempty"`
// 	
// 	// Bandwidth utilization analysis
// 	BandwidthAnalysis *BandwidthUtilizationAnalysis `json:"bandwidth_analysis,omitempty"`
// 	
// 	// Quality of Service measurements
// 	QoSAnalysis *QoSAnalysisResults `json:"qos_analysis,omitempty"`
// 	
// 	// Security-related diagnostics
// 	SecurityAnalysis *NetworkSecurityAnalysis `json:"security_analysis,omitempty"`
// 	
// 	// Performance baseline comparison
// 	BaselineComparison *PerformanceBaselineComparison `json:"baseline_comparison,omitempty"`
// 	
// 	// Test execution statistics
	ExecutionStats DiagnosticExecutionStats `json:"execution_stats"`
}

// DiagnosticTestContext provides context for test execution
type DiagnosticTestContext struct {
	TestMode        string            `json:"test_mode"`        // manual, scheduled, triggered
	TriggerReason   string            `json:"trigger_reason,omitempty"`
	TestParameters  map[string]string `json:"test_parameters,omitempty"`
	ScheduleID      string            `json:"schedule_id,omitempty"`
	UserID          string            `json:"user_id,omitempty"`
	ClientIP        string            `json:"client_ip,omitempty"`
	TestLocation    string            `json:"test_location,omitempty"`
	TestEnvironment TestEnvironment   `json:"test_environment"`
}

// TestEnvironment captures environmental factors during testing
type TestEnvironment struct {
	NetworkLoad      float64           `json:"network_load"`
	ConcurrentUsers  int               `json:"concurrent_users"`
	TimeOfDay        string            `json:"time_of_day"`
	WeekDay          string            `json:"week_day"`
	SystemLoad       SystemLoadMetrics `json:"system_load"`
	ExternalFactors  []string          `json:"external_factors,omitempty"`
}

// SystemLoadMetrics represents system performance during tests
type SystemLoadMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskIO      float64 `json:"disk_io"`
	NetworkIO   float64 `json:"network_io"`
	LoadAverage []float64 `json:"load_average"` // 1min, 5min, 15min
}

// SpeedTestResults contains comprehensive speed test measurements
type SpeedTestResults struct {
	TestType        string                    `json:"test_type"`        // iperf3, speedtest-cli, custom
	TestDuration    time.Duration             `json:"test_duration"`
	ServerInfo      SpeedTestServerInfo       `json:"server_info"`
	DownloadResults SpeedTestMeasurement      `json:"download_results"`
	UploadResults   SpeedTestMeasurement      `json:"upload_results"`
	BidirectionalResults *SpeedTestMeasurement `json:"bidirectional_results,omitempty"`
	MultiStreamResults []SpeedTestStream      `json:"multi_stream_results,omitempty"`
	ProtocolAnalysis SpeedTestProtocolAnalysis `json:"protocol_analysis"`
	QualityMetrics  SpeedTestQualityMetrics   `json:"quality_metrics"`
	TestStability   SpeedTestStabilityMetrics `json:"test_stability"`
}

// SpeedTestServerInfo describes the test server
type SpeedTestServerInfo struct {
	ServerID   string  `json:"server_id"`
	ServerName string  `json:"server_name"`
	Location   string  `json:"location"`
	Country    string  `json:"country"`
	ISP        string  `json:"isp"`
	Distance   float64 `json:"distance_km"`
	Latency    float64 `json:"latency_ms"`
	Port       int     `json:"port"`
	Protocol   string  `json:"protocol"`
}

// SpeedTestMeasurement represents detailed speed measurements
type SpeedTestMeasurement struct {
	AverageSpeed     float64            `json:"average_speed_mbps"`
	MaxSpeed         float64            `json:"max_speed_mbps"`
	MinSpeed         float64            `json:"min_speed_mbps"`
	MedianSpeed      float64            `json:"median_speed_mbps"`
	SpeedVariability float64            `json:"speed_variability"`
	Percentiles      SpeedPercentiles   `json:"percentiles"`
	ThroughputCurve  []ThroughputPoint  `json:"throughput_curve"`
	Retransmissions  int                `json:"retransmissions"`
	BytesTransferred int64              `json:"bytes_transferred"`
	TotalTime        time.Duration      `json:"total_time"`
	WindowScaling    bool               `json:"window_scaling"`
	TCPCongestion    string             `json:"tcp_congestion"`
}

// SpeedPercentiles provides percentile analysis of speed measurements
type SpeedPercentiles struct {
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

// ThroughputPoint represents a point in time during speed test
type ThroughputPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Speed     float64   `json:"speed_mbps"`
	Bytes     int64     `json:"bytes"`
	RTT       float64   `json:"rtt_ms"`
}

// SpeedTestStream represents individual stream in multi-stream test
type SpeedTestStream struct {
	StreamID      int                  `json:"stream_id"`
	StartTime     time.Time            `json:"start_time"`
	EndTime       time.Time            `json:"end_time"`
	Measurement   SpeedTestMeasurement `json:"measurement"`
	StreamMetrics StreamPerformanceMetrics `json:"stream_metrics"`
}

// StreamPerformanceMetrics provides detailed stream analysis
type StreamPerformanceMetrics struct {
	WindowSize       int     `json:"window_size"`
	MSS              int     `json:"mss"`
	RetransmitRate   float64 `json:"retransmit_rate"`
	CongestionEvents int     `json:"congestion_events"`
	ZeroWindowEvents int     `json:"zero_window_events"`
	DuplicateACKs    int     `json:"duplicate_acks"`
}

// SpeedTestProtocolAnalysis analyzes protocol-level performance
type SpeedTestProtocolAnalysis struct {
	TCPVersion      string                `json:"tcp_version"`
	SSLVersion      string                `json:"ssl_version,omitempty"`
	CipherSuite     string                `json:"cipher_suite,omitempty"`
	MTUDiscovery    bool                  `json:"mtu_discovery"`
	PathMTU         int                   `json:"path_mtu"`
	ECNSupport      bool                  `json:"ecn_support"`
	SACKSupport     bool                  `json:"sack_support"`
	TimestampOption bool                  `json:"timestamp_option"`
	ProtocolEfficiency ProtocolEfficiency `json:"protocol_efficiency"`
}

// ProtocolEfficiency measures protocol overhead and efficiency
type ProtocolEfficiency struct {
	HeaderOverhead      float64 `json:"header_overhead_percent"`
	RetransmissionRate  float64 `json:"retransmission_rate"`
	OutOfOrderPackets   int     `json:"out_of_order_packets"`
	DuplicatePackets    int     `json:"duplicate_packets"`
	ProtocolUtilization float64 `json:"protocol_utilization"`
}

// SpeedTestQualityMetrics measures connection quality during speed tests
type SpeedTestQualityMetrics struct {
	Jitter           float64 `json:"jitter_ms"`
	PacketLoss       float64 `json:"packet_loss_percent"`
	LatencyVariation float64 `json:"latency_variation_ms"`
	BufferBloat      float64 `json:"buffer_bloat_ms"`
	QualityScore     float64 `json:"quality_score"`
	MOS              float64 `json:"mos_score"` // Mean Opinion Score
}

// SpeedTestStabilityMetrics measures test stability and consistency
type SpeedTestStabilityMetrics struct {
	CoefficientOfVariation float64   `json:"coefficient_of_variation"`
	StabilityScore         float64   `json:"stability_score"`
	Outliers               int       `json:"outliers"`
	TrendAnalysis          TrendData `json:"trend_analysis"`
	ConsistencyRating      string    `json:"consistency_rating"` // excellent, good, fair, poor
}

// TrendData analyzes performance trends during the test
type TrendData struct {
	Direction    string  `json:"direction"`    // increasing, decreasing, stable
	Slope        float64 `json:"slope"`
	Correlation  float64 `json:"correlation"`
	Seasonality  bool    `json:"seasonality"`
	TrendStrength string `json:"trend_strength"` // strong, moderate, weak
}

// WANConnectivityResults tests external connectivity
type WANConnectivityResults struct {
	ISPGatewayTest     GatewayConnectivityTest `json:"isp_gateway_test"`
	DNSConnectivity    DNSConnectivityTest     `json:"dns_connectivity"`
// 	InternetReachability InternetReachabilityTest `json:"internet_reachability"`
// 	ISPPerformance     ISPPerformanceAnalysis  `json:"isp_performance"`
// 	GeographicTest     GeographicConnectivityTest `json:"geographic_test"`
// 	ContentDelivery    CDNPerformanceTest      `json:"content_delivery"`
// 	WANQuality         WANQualityMetrics       `json:"wan_quality"`
}

// GatewayConnectivityTest tests ISP gateway connectivity
type GatewayConnectivityTest struct {
	GatewayIP         string        `json:"gateway_ip"`
	PingResults       PingTestResults `json:"ping_results"`
	TraceRouteResults TraceRouteResults `json:"traceroute_results"`
	Reachability      bool          `json:"reachability"`
	ResponseTime      time.Duration `json:"response_time"`
	HopCount          int           `json:"hop_count"`
	PathStability     PathStabilityMetrics `json:"path_stability"`
}

// PingTestResults provides detailed ping analysis
type PingTestResults struct {
	PacketsSent     int           `json:"packets_sent"`
	PacketsReceived int           `json:"packets_received"`
	PacketLoss      float64       `json:"packet_loss_percent"`
	MinRTT          time.Duration `json:"min_rtt"`
	MaxRTT          time.Duration `json:"max_rtt"`
	AvgRTT          time.Duration `json:"avg_rtt"`
	StdDevRTT       time.Duration `json:"stddev_rtt"`
	Jitter          time.Duration `json:"jitter"`
	RTTDistribution []RTTMeasurement `json:"rtt_distribution"`
	PacketSizes     []int         `json:"packet_sizes"`
	TimeBasedAnalysis TimeBasedPingAnalysis `json:"time_based_analysis"`
}

// RTTMeasurement represents individual RTT measurement
type RTTMeasurement struct {
	SequenceNumber int           `json:"sequence_number"`
	Timestamp      time.Time     `json:"timestamp"`
	RTT            time.Duration `json:"rtt"`
	PacketSize     int           `json:"packet_size"`
	TTL            int           `json:"ttl"`
}

// TimeBasedPingAnalysis analyzes ping performance over time
type TimeBasedPingAnalysis struct {
	SampleInterval  time.Duration     `json:"sample_interval"`
	TrendDirection  string           `json:"trend_direction"`
	Seasonality     bool             `json:"seasonality"`
	PerformancePattern string        `json:"performance_pattern"`
	OutageDetection []OutagePeriod   `json:"outage_detection"`
	QualityWindows  []QualityWindow  `json:"quality_windows"`
}

// OutagePeriod represents detected connectivity outages
type OutagePeriod struct {
	StartTime  time.Time     `json:"start_time"`
	EndTime    time.Time     `json:"end_time"`
	Duration   time.Duration `json:"duration"`
	Severity   string        `json:"severity"` // complete, partial, degraded
	RecoveryTime time.Duration `json:"recovery_time"`
}

// QualityWindow represents periods of different quality levels
type QualityWindow struct {
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	QualityLevel string        `json:"quality_level"` // excellent, good, fair, poor
	Metrics      WindowMetrics `json:"metrics"`
}

// WindowMetrics provides metrics for a quality window
type WindowMetrics struct {
	AverageRTT   time.Duration `json:"average_rtt"`
	PacketLoss   float64       `json:"packet_loss"`
	Jitter       time.Duration `json:"jitter"`
	Stability    float64       `json:"stability"`
}

// TraceRouteResults provides network path analysis
type TraceRouteResults struct {
	TargetHost     string      `json:"target_host"`
	TotalHops      int         `json:"total_hops"`
	CompletePath   bool        `json:"complete_path"`
	PathHops       []HopInfo   `json:"path_hops"`
	PathAnalysis   PathAnalysis `json:"path_analysis"`
	RouteStability RouteStability `json:"route_stability"`
}

// HopInfo represents information about a single hop
type HopInfo struct {
	HopNumber    int           `json:"hop_number"`
	IPAddress    string        `json:"ip_address"`
	Hostname     string        `json:"hostname,omitempty"`
	RTT1         time.Duration `json:"rtt1"`
	RTT2         time.Duration `json:"rtt2"`
	RTT3         time.Duration `json:"rtt3"`
	AverageRTT   time.Duration `json:"average_rtt"`
	Location     string        `json:"location,omitempty"`
	ISP          string        `json:"isp,omitempty"`
	ASN          string        `json:"asn,omitempty"`
	HopType      string        `json:"hop_type"` // router, switch, firewall, gateway
}

// PathAnalysis analyzes the complete network path
type PathAnalysis struct {
	BottleneckHop    int           `json:"bottleneck_hop"`
	HighestLatency   time.Duration `json:"highest_latency"`
	LatencyIncrease  []time.Duration `json:"latency_increase"`
	GeographicSpread GeographicSpread `json:"geographic_spread"`
	ASPathAnalysis   ASPathAnalysis  `json:"as_path_analysis"`
	PathEfficiency   PathEfficiency  `json:"path_efficiency"`
}

// GeographicSpread analyzes geographic distribution of path
type GeographicSpread struct {
	Countries    []string `json:"countries"`
	Continents   []string `json:"continents"`
	TotalDistance float64  `json:"total_distance_km"`
	GeographicDiversity string `json:"geographic_diversity"` // local, regional, international
}

// ASPathAnalysis analyzes Autonomous System path
type ASPathAnalysis struct {
	ASPath          []string `json:"as_path"`
	ASPathLength    int      `json:"as_path_length"`
	UniqueASCount   int      `json:"unique_as_count"`
	TierClassification []string `json:"tier_classification"`
	PeeringAnalysis PeeringAnalysis `json:"peering_analysis"`
}

// PeeringAnalysis analyzes peering relationships
type PeeringAnalysis struct {
	PeeringTypes    []string `json:"peering_types"`
	TransitRelations int     `json:"transit_relations"`
	PeerRelations   int      `json:"peer_relations"`
	IXPTransits     []string `json:"ixp_transits,omitempty"`
}

// PathEfficiency measures routing efficiency
type PathEfficiency struct {
	OptimalHops      int     `json:"optimal_hops"`
	ActualHops       int     `json:"actual_hops"`
	EfficiencyRatio  float64 `json:"efficiency_ratio"`
	PathOverhead     int     `json:"path_overhead"`
	RoutingQuality   string  `json:"routing_quality"` // optimal, good, suboptimal, poor
}

// RouteStability measures path stability over time
type RouteStability struct {
	StabilityPeriod  time.Duration `json:"stability_period"`
	PathChanges      int          `json:"path_changes"`
	FlappingHops     []int        `json:"flapping_hops"`
	StabilityScore   float64      `json:"stability_score"`
	ConsistentHops   int          `json:"consistent_hops"`
}

// PathStabilityMetrics measures gateway path stability
type PathStabilityMetrics struct {
	ConsistentPath   bool    `json:"consistent_path"`
	PathChanges      int     `json:"path_changes"`
	StabilityRating  string  `json:"stability_rating"`
	LastPathChange   time.Time `json:"last_path_change,omitempty"`
}

// DNSConnectivityTest tests DNS resolution
type DNSConnectivityTest struct {
	PrimaryDNS    DNSServerTest `json:"primary_dns"`
	SecondaryDNS  DNSServerTest `json:"secondary_dns"`
	PublicDNS     map[string]DNSServerTest `json:"public_dns"` // Google, Cloudflare, etc.
	LocalDNS      DNSServerTest `json:"local_dns"`
	DNSChain      DNSChainAnalysis `json:"dns_chain"`
	DNSSecurity   DNSSecurityAnalysis `json:"dns_security"`
}

// DNSServerTest tests individual DNS server
type DNSServerTest struct {
	ServerIP       string        `json:"server_ip"`
	ServerName     string        `json:"server_name"`
	Reachability   bool          `json:"reachability"`
	ResponseTime   time.Duration `json:"response_time"`
	QuerySuccess   bool          `json:"query_success"`
	ProtocolSupport DNSProtocolSupport `json:"protocol_support"`
	Performance    DNSPerformanceMetrics `json:"performance"`
}

// DNSProtocolSupport shows supported DNS features
type DNSProtocolSupport struct {
	UDP            bool `json:"udp"`
	TCP            bool `json:"tcp"`
	DOH            bool `json:"doh"`  // DNS over HTTPS
	DOT            bool `json:"dot"`  // DNS over TLS
	DNSSEC         bool `json:"dnssec"`
	IPv6           bool `json:"ipv6"`
	EDNS           bool `json:"edns"`
}

// DNSPerformanceMetrics measures DNS server performance
type DNSPerformanceMetrics struct {
	AverageQueryTime time.Duration `json:"average_query_time"`
	QuerySuccess     float64       `json:"query_success_rate"`
	TimeoutRate      float64       `json:"timeout_rate"`
	CacheHitRate     float64       `json:"cache_hit_rate,omitempty"`
	ConcurrentQueries int          `json:"concurrent_queries"`
}

// DNSChainAnalysis analyzes DNS resolution chain
type DNSChainAnalysis struct {
	TotalHops        int               `json:"total_hops"`
	AuthorityChain   []DNSAuthority    `json:"authority_chain"`
	ResolutionTime   time.Duration     `json:"total_resolution_time"`
	CacheAnalysis    DNSCacheAnalysis  `json:"cache_analysis"`
	ChainEfficiency  ChainEfficiency   `json:"chain_efficiency"`
}

// DNSAuthority represents DNS authority in resolution chain
type DNSAuthority struct {
	Server       string        `json:"server"`
	Zone         string        `json:"zone"`
	RecordType   string        `json:"record_type"`
	TTL          int           `json:"ttl"`
	ResponseTime time.Duration `json:"response_time"`
	Authoritative bool         `json:"authoritative"`
}

// DNSCacheAnalysis analyzes DNS caching behavior
type DNSCacheAnalysis struct {
	CacheHits      int     `json:"cache_hits"`
	CacheMisses    int     `json:"cache_misses"`
	CacheHitRatio  float64 `json:"cache_hit_ratio"`
	TTLDistribution []TTLBucket `json:"ttl_distribution"`
	CacheEfficiency string  `json:"cache_efficiency"`
}

// TTLBucket represents TTL distribution bucket
type TTLBucket struct {
	TTLRange string `json:"ttl_range"`
	Count    int    `json:"count"`
}

// ChainEfficiency measures DNS resolution efficiency
type ChainEfficiency struct {
	OptimalHops     int     `json:"optimal_hops"`
	ActualHops      int     `json:"actual_hops"`
	EfficiencyRatio float64 `json:"efficiency_ratio"`
	UnnecessaryHops int     `json:"unnecessary_hops"`
}

// DNSSecurityAnalysis analyzes DNS security features
type DNSSecurityAnalysis struct {
	DNSSECValidation bool              `json:"dnssec_validation"`
	DangerousDomains []DangerousDomain `json:"dangerous_domains"`
	FilteringActive  bool              `json:"filtering_active"`
	SecurityScore    float64           `json:"security_score"`
	Threats          []SecurityThreat  `json:"threats"`
}

// DangerousDomain represents detected malicious domains
type DangerousDomain struct {
	Domain      string    `json:"domain"`
	ThreatType  string    `json:"threat_type"`
	Confidence  float64   `json:"confidence"`
	DetectedAt  time.Time `json:"detected_at"`
	Source      string    `json:"source"`
}

// SecurityThreat represents security threat detected during DNS tests
type SecurityThreat struct {
	ThreatID    string    `json:"threat_id"`
	ThreatType  string    `json:"threat_type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	DetectedAt  time.Time `json:"detected_at"`
	Mitigated   bool      `json:"mitigated"`
}

// DiagnosticExecutionStats tracks test execution statistics
type DiagnosticExecutionStats struct {
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	TotalDuration   time.Duration `json:"total_duration"`
	TestsExecuted   int           `json:"tests_executed"`
	TestsSucceeded  int           `json:"tests_succeeded"`
	TestsFailed     int           `json:"tests_failed"`
	TestsSkipped    int           `json:"tests_skipped"`
	DataPointsCollected int       `json:"data_points_collected"`
	ErrorsEncountered []TestError `json:"errors_encountered"`
	ResourceUsage   ResourceUsage `json:"resource_usage"`
	Performance     ExecutionPerformance `json:"performance"`
}

// TestError represents errors during test execution
type TestError struct {
	TestName    string    `json:"test_name"`
	ErrorType   string    `json:"error_type"`
	ErrorMessage string   `json:"error_message"`
	Timestamp   time.Time `json:"timestamp"`
	Recoverable bool      `json:"recoverable"`
	RetryCount  int       `json:"retry_count"`
}

// ResourceUsage tracks resource consumption during tests
type ResourceUsage struct {
	CPUTime      time.Duration `json:"cpu_time"`
	MemoryPeak   int64         `json:"memory_peak_bytes"`
	MemoryAverage int64        `json:"memory_average_bytes"`
	NetworkBytes int64         `json:"network_bytes"`
	DiskIO       int64         `json:"disk_io_bytes"`
	FileDescriptors int        `json:"file_descriptors"`
}

// ExecutionPerformance measures test execution performance
type ExecutionPerformance struct {
	TestsPerSecond    float64       `json:"tests_per_second"`
	AverageTestTime   time.Duration `json:"average_test_time"`
	ParallelEfficiency float64      `json:"parallel_efficiency"`
	ResourceEfficiency float64      `json:"resource_efficiency"`
	BottleneckAnalysis BottleneckAnalysis `json:"bottleneck_analysis"`
}

// BottleneckAnalysis identifies performance bottlenecks
type BottleneckAnalysis struct {
	PrimaryBottleneck   string             `json:"primary_bottleneck"`
	BottleneckSeverity  string             `json:"bottleneck_severity"`
	ImpactedTests       []string           `json:"impacted_tests"`
	RecommendedActions  []string           `json:"recommended_actions"`
	PerformanceProfile  PerformanceProfile `json:"performance_profile"`
}

// PerformanceProfile characterizes system performance
type PerformanceProfile struct {
	CPUBound     bool `json:"cpu_bound"`
	MemoryBound  bool `json:"memory_bound"`
	NetworkBound bool `json:"network_bound"`
	IOBound      bool `json:"io_bound"`
}

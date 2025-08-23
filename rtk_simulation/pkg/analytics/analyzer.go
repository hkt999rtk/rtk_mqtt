package analytics

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// LogAnalyzer 日誌分析器
type LogAnalyzer struct {
	entries    []LogEntry
	patterns   map[string]*Pattern
	statistics *LogStatistics
	insights   []Insight
	running    bool
	mu         sync.RWMutex
	logger     *logrus.Entry
	config     *AnalyzerConfig
}

// LogEntry 日誌條目
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Component string
	DeviceID  string
	Message   string
	Fields    map[string]interface{}
	Pattern   string
	Anomaly   bool
}

// Pattern 日誌模式
type Pattern struct {
	ID       string
	Name     string
	Regex    *regexp.Regexp
	Category string
	Severity string
	Count    int
	LastSeen time.Time
	Examples []string
}

// LogStatistics 日誌統計
type LogStatistics struct {
	TotalEntries       int
	EntriesByLevel     map[string]int
	EntriesByComponent map[string]int
	EntriesByDevice    map[string]int
	ErrorRate          float64
	WarningRate        float64
	TopPatterns        []PatternStat
	TimeDistribution   map[string]int // 按小時分佈
	AnomalyCount       int
	LastUpdate         time.Time
}

// PatternStat 模式統計
type PatternStat struct {
	PatternID  string
	Name       string
	Count      int
	Percentage float64
	Trend      string // increasing, decreasing, stable
}

// Insight 洞察
type Insight struct {
	ID          string
	Type        string // anomaly, trend, correlation, recommendation
	Severity    string
	Title       string
	Description string
	Evidence    []string
	Timestamp   time.Time
	ActionItems []string
}

// AnalyzerConfig 分析器配置
type AnalyzerConfig struct {
	MaxEntries        int
	AnalysisInterval  time.Duration
	PatternThreshold  int
	AnomalyDetection  bool
	InsightGeneration bool
}

// TimeSeriesData 時間序列數據
type TimeSeriesData struct {
	Timestamp time.Time
	Value     float64
	Label     string
}

// NewLogAnalyzer 創建新的日誌分析器
func NewLogAnalyzer(config *AnalyzerConfig) *LogAnalyzer {
	if config == nil {
		config = &AnalyzerConfig{
			MaxEntries:        10000,
			AnalysisInterval:  30 * time.Second,
			PatternThreshold:  5,
			AnomalyDetection:  true,
			InsightGeneration: true,
		}
	}

	la := &LogAnalyzer{
		entries:  make([]LogEntry, 0),
		patterns: make(map[string]*Pattern),
		insights: make([]Insight, 0),
		config:   config,
		logger:   logrus.WithField("component", "log_analyzer"),
		statistics: &LogStatistics{
			EntriesByLevel:     make(map[string]int),
			EntriesByComponent: make(map[string]int),
			EntriesByDevice:    make(map[string]int),
			TimeDistribution:   make(map[string]int),
			TopPatterns:        make([]PatternStat, 0),
		},
	}

	// 初始化預定義模式
	la.initializePatterns()

	return la
}

// initializePatterns 初始化預定義模式
func (la *LogAnalyzer) initializePatterns() {
	patterns := []struct {
		id       string
		name     string
		regex    string
		category string
		severity string
	}{
		{"conn_fail", "Connection Failed", `connection.*failed|failed.*connect`, "network", "error"},
		{"timeout", "Timeout", `timeout|timed out`, "network", "warning"},
		{"auth_fail", "Authentication Failed", `auth.*fail|authentication.*failed`, "security", "error"},
		{"memory_leak", "Memory Leak", `memory.*leak|out of memory`, "performance", "critical"},
		{"high_cpu", "High CPU Usage", `cpu.*high|high.*cpu|cpu.*usage.*[89]\d%`, "performance", "warning"},
		{"disk_full", "Disk Full", `disk.*full|no space left`, "storage", "critical"},
		{"packet_loss", "Packet Loss", `packet.*loss|packets.*dropped`, "network", "warning"},
		{"service_down", "Service Down", `service.*down|service.*unavailable`, "availability", "error"},
		{"config_error", "Configuration Error", `config.*error|invalid.*config`, "configuration", "error"},
		{"restart", "Service Restart", `restart|rebooting|starting up`, "lifecycle", "info"},
	}

	for _, p := range patterns {
		regex, err := regexp.Compile(p.regex)
		if err != nil {
			la.logger.WithError(err).Warnf("Failed to compile pattern: %s", p.name)
			continue
		}

		la.patterns[p.id] = &Pattern{
			ID:       p.id,
			Name:     p.name,
			Regex:    regex,
			Category: p.category,
			Severity: p.severity,
			Examples: make([]string, 0),
		}
	}
}

// Start 啟動分析器
func (la *LogAnalyzer) Start(ctx context.Context) error {
	la.mu.Lock()
	defer la.mu.Unlock()

	if la.running {
		return fmt.Errorf("log analyzer is already running")
	}

	la.running = true
	la.logger.Info("Starting log analyzer")

	// 啟動分析循環
	go la.analysisLoop(ctx)
	go la.insightGenerationLoop(ctx)
	go la.cleanupLoop(ctx)

	return nil
}

// Stop 停止分析器
func (la *LogAnalyzer) Stop() error {
	la.mu.Lock()
	defer la.mu.Unlock()

	if !la.running {
		return fmt.Errorf("log analyzer is not running")
	}

	la.running = false
	la.logger.Info("Stopping log analyzer")
	return nil
}

// AddLogEntry 添加日誌條目
func (la *LogAnalyzer) AddLogEntry(timestamp time.Time, level, component, deviceID, message string, fields map[string]interface{}) {
	la.mu.Lock()
	defer la.mu.Unlock()

	entry := LogEntry{
		Timestamp: timestamp,
		Level:     level,
		Component: component,
		DeviceID:  deviceID,
		Message:   message,
		Fields:    fields,
	}

	// 模式匹配
	entry.Pattern = la.matchPattern(message)

	// 異常檢測
	if la.config.AnomalyDetection {
		entry.Anomaly = la.detectAnomaly(entry)
	}

	la.entries = append(la.entries, entry)

	// 限制條目數量
	if len(la.entries) > la.config.MaxEntries {
		la.entries = la.entries[len(la.entries)-la.config.MaxEntries:]
	}

	// 更新統計
	la.updateStatistics(entry)
}

// matchPattern 匹配模式
func (la *LogAnalyzer) matchPattern(message string) string {
	messageLower := strings.ToLower(message)

	for id, pattern := range la.patterns {
		if pattern.Regex.MatchString(messageLower) {
			pattern.Count++
			pattern.LastSeen = time.Now()

			// 保存示例
			if len(pattern.Examples) < 5 {
				pattern.Examples = append(pattern.Examples, message)
			}

			return id
		}
	}

	return ""
}

// detectAnomaly 檢測異常
func (la *LogAnalyzer) detectAnomaly(entry LogEntry) bool {
	// 簡單的異常檢測邏輯
	// 可以擴展為更複雜的機器學習模型

	// 檢查錯誤級別的突增
	if entry.Level == "error" || entry.Level == "critical" {
		// 檢查最近是否有類似錯誤
		recentErrors := 0
		cutoff := time.Now().Add(-5 * time.Minute)

		for i := len(la.entries) - 1; i >= 0 && i > len(la.entries)-100; i-- {
			if la.entries[i].Timestamp.After(cutoff) && la.entries[i].Level == entry.Level {
				recentErrors++
			}
		}

		if recentErrors > 10 {
			return true
		}
	}

	// 檢查罕見模式
	if entry.Pattern != "" {
		if pattern, exists := la.patterns[entry.Pattern]; exists {
			if pattern.Count < la.config.PatternThreshold {
				return true
			}
		}
	}

	return false
}

// updateStatistics 更新統計
func (la *LogAnalyzer) updateStatistics(entry LogEntry) {
	stats := la.statistics

	stats.TotalEntries++
	stats.EntriesByLevel[entry.Level]++
	stats.EntriesByComponent[entry.Component]++

	if entry.DeviceID != "" {
		stats.EntriesByDevice[entry.DeviceID]++
	}

	// 時間分佈（按小時）
	hour := entry.Timestamp.Format("15")
	stats.TimeDistribution[hour]++

	if entry.Anomaly {
		stats.AnomalyCount++
	}

	// 計算錯誤率
	if stats.TotalEntries > 0 {
		errorCount := stats.EntriesByLevel["error"] + stats.EntriesByLevel["critical"]
		stats.ErrorRate = float64(errorCount) / float64(stats.TotalEntries) * 100

		warningCount := stats.EntriesByLevel["warning"]
		stats.WarningRate = float64(warningCount) / float64(stats.TotalEntries) * 100
	}

	stats.LastUpdate = time.Now()
}

// analysisLoop 分析循環
func (la *LogAnalyzer) analysisLoop(ctx context.Context) {
	ticker := time.NewTicker(la.config.AnalysisInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			la.performAnalysis()
		}
	}
}

// performAnalysis 執行分析
func (la *LogAnalyzer) performAnalysis() {
	la.mu.Lock()
	defer la.mu.Unlock()

	// 更新模式統計
	la.updatePatternStatistics()

	// 檢測趨勢
	la.detectTrends()

	// 查找相關性
	la.findCorrelations()
}

// updatePatternStatistics 更新模式統計
func (la *LogAnalyzer) updatePatternStatistics() {
	patternStats := make([]PatternStat, 0)

	for id, pattern := range la.patterns {
		if pattern.Count > 0 {
			percentage := float64(pattern.Count) / float64(la.statistics.TotalEntries) * 100

			stat := PatternStat{
				PatternID:  id,
				Name:       pattern.Name,
				Count:      pattern.Count,
				Percentage: percentage,
				Trend:      la.calculateTrend(id),
			}

			patternStats = append(patternStats, stat)
		}
	}

	// 排序（按計數降序）
	sort.Slice(patternStats, func(i, j int) bool {
		return patternStats[i].Count > patternStats[j].Count
	})

	// 保留前 10 個
	if len(patternStats) > 10 {
		patternStats = patternStats[:10]
	}

	la.statistics.TopPatterns = patternStats
}

// calculateTrend 計算趨勢
func (la *LogAnalyzer) calculateTrend(patternID string) string {
	// 簡化的趨勢計算
	// 實際應用中可以使用更複雜的時間序列分析

	pattern := la.patterns[patternID]
	if pattern.Count < 10 {
		return "stable"
	}

	// 檢查最近的出現頻率
	recentCount := 0
	cutoff := time.Now().Add(-10 * time.Minute)

	for i := len(la.entries) - 1; i >= 0 && la.entries[i].Timestamp.After(cutoff); i-- {
		if la.entries[i].Pattern == patternID {
			recentCount++
		}
	}

	avgRate := float64(pattern.Count) / 60.0  // 假設 60 分鐘的數據
	recentRate := float64(recentCount) / 10.0 // 最近 10 分鐘

	if recentRate > avgRate*1.5 {
		return "increasing"
	} else if recentRate < avgRate*0.5 {
		return "decreasing"
	}

	return "stable"
}

// detectTrends 檢測趨勢
func (la *LogAnalyzer) detectTrends() {
	// 檢測錯誤率趨勢
	if la.statistics.ErrorRate > 5 {
		la.generateInsight("high_error_rate", "anomaly", "High Error Rate Detected",
			fmt.Sprintf("Error rate is %.2f%%, which is above normal threshold", la.statistics.ErrorRate))
	}

	// 檢測異常數量趨勢
	if la.statistics.AnomalyCount > 10 {
		la.generateInsight("high_anomaly_count", "anomaly", "Multiple Anomalies Detected",
			fmt.Sprintf("Detected %d anomalies in recent logs", la.statistics.AnomalyCount))
	}
}

// findCorrelations 查找相關性
func (la *LogAnalyzer) findCorrelations() {
	// 查找設備和錯誤的相關性
	for deviceID, count := range la.statistics.EntriesByDevice {
		errorCount := 0
		for _, entry := range la.entries {
			if entry.DeviceID == deviceID && (entry.Level == "error" || entry.Level == "critical") {
				errorCount++
			}
		}

		if count > 0 {
			errorRate := float64(errorCount) / float64(count) * 100
			if errorRate > 20 {
				la.generateInsight(
					fmt.Sprintf("device_error_%s", deviceID),
					"correlation",
					fmt.Sprintf("High Error Rate for Device %s", deviceID),
					fmt.Sprintf("Device %s has %.2f%% error rate", deviceID, errorRate),
				)
			}
		}
	}
}

// insightGenerationLoop 洞察生成循環
func (la *LogAnalyzer) insightGenerationLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if la.config.InsightGeneration {
				la.generateInsights()
			}
		}
	}
}

// generateInsights 生成洞察
func (la *LogAnalyzer) generateInsights() {
	la.mu.Lock()
	defer la.mu.Unlock()

	// 基於模式生成洞察
	for _, stat := range la.statistics.TopPatterns {
		if stat.Trend == "increasing" && stat.Count > 20 {
			pattern := la.patterns[stat.PatternID]
			la.generateInsight(
				fmt.Sprintf("pattern_trend_%s", stat.PatternID),
				"trend",
				fmt.Sprintf("Increasing Pattern: %s", pattern.Name),
				fmt.Sprintf("Pattern '%s' is showing an increasing trend with %d occurrences", pattern.Name, stat.Count),
			)
		}
	}
}

// generateInsight 生成單個洞察
func (la *LogAnalyzer) generateInsight(id, insightType, title, description string) {
	// 檢查是否已存在相同的洞察
	for _, insight := range la.insights {
		if insight.ID == id && time.Since(insight.Timestamp) < 10*time.Minute {
			return
		}
	}

	insight := Insight{
		ID:          id,
		Type:        insightType,
		Severity:    la.determineSeverity(insightType),
		Title:       title,
		Description: description,
		Timestamp:   time.Now(),
		Evidence:    la.collectEvidence(id),
		ActionItems: la.generateActionItems(insightType),
	}

	la.insights = append(la.insights, insight)

	la.logger.WithFields(logrus.Fields{
		"insight_id":   id,
		"insight_type": insightType,
		"title":        title,
	}).Info("New insight generated")
}

// determineSeverity 確定嚴重性
func (la *LogAnalyzer) determineSeverity(insightType string) string {
	switch insightType {
	case "anomaly":
		return "high"
	case "correlation":
		return "medium"
	case "trend":
		return "medium"
	case "recommendation":
		return "low"
	default:
		return "info"
	}
}

// collectEvidence 收集證據
func (la *LogAnalyzer) collectEvidence(insightID string) []string {
	evidence := make([]string, 0)

	// 收集相關的日誌條目作為證據
	for i := len(la.entries) - 1; i >= 0 && len(evidence) < 5; i-- {
		entry := la.entries[i]
		if entry.Anomaly || entry.Level == "error" || entry.Level == "critical" {
			evidence = append(evidence, fmt.Sprintf("[%s] %s: %s",
				entry.Timestamp.Format("15:04:05"), entry.Level, entry.Message))
		}
	}

	return evidence
}

// generateActionItems 生成行動項目
func (la *LogAnalyzer) generateActionItems(insightType string) []string {
	actions := make([]string, 0)

	switch insightType {
	case "anomaly":
		actions = append(actions, "Investigate the root cause of anomalies")
		actions = append(actions, "Check system health and resource usage")
		actions = append(actions, "Review recent configuration changes")
	case "correlation":
		actions = append(actions, "Analyze the correlation pattern")
		actions = append(actions, "Implement preventive measures")
	case "trend":
		actions = append(actions, "Monitor the trend closely")
		actions = append(actions, "Prepare scaling or optimization plans")
	}

	return actions
}

// cleanupLoop 清理循環
func (la *LogAnalyzer) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			la.cleanupOldData()
		}
	}
}

// cleanupOldData 清理舊數據
func (la *LogAnalyzer) cleanupOldData() {
	la.mu.Lock()
	defer la.mu.Unlock()

	// 清理舊洞察
	cutoff := time.Now().Add(-24 * time.Hour)
	newInsights := make([]Insight, 0)

	for _, insight := range la.insights {
		if insight.Timestamp.After(cutoff) {
			newInsights = append(newInsights, insight)
		}
	}

	la.insights = newInsights

	// 重置過期的模式計數
	for _, pattern := range la.patterns {
		if time.Since(pattern.LastSeen) > 1*time.Hour {
			pattern.Count = 0
			pattern.Examples = make([]string, 0)
		}
	}
}

// GetStatistics 獲取統計資訊
func (la *LogAnalyzer) GetStatistics() *LogStatistics {
	la.mu.RLock()
	defer la.mu.RUnlock()

	// 返回統計副本
	stats := &LogStatistics{
		TotalEntries:       la.statistics.TotalEntries,
		EntriesByLevel:     make(map[string]int),
		EntriesByComponent: make(map[string]int),
		EntriesByDevice:    make(map[string]int),
		ErrorRate:          la.statistics.ErrorRate,
		WarningRate:        la.statistics.WarningRate,
		TopPatterns:        append([]PatternStat{}, la.statistics.TopPatterns...),
		TimeDistribution:   make(map[string]int),
		AnomalyCount:       la.statistics.AnomalyCount,
		LastUpdate:         la.statistics.LastUpdate,
	}

	for k, v := range la.statistics.EntriesByLevel {
		stats.EntriesByLevel[k] = v
	}
	for k, v := range la.statistics.EntriesByComponent {
		stats.EntriesByComponent[k] = v
	}
	for k, v := range la.statistics.EntriesByDevice {
		stats.EntriesByDevice[k] = v
	}
	for k, v := range la.statistics.TimeDistribution {
		stats.TimeDistribution[k] = v
	}

	return stats
}

// GetInsights 獲取洞察
func (la *LogAnalyzer) GetInsights(limit int) []Insight {
	la.mu.RLock()
	defer la.mu.RUnlock()

	if limit <= 0 || limit > len(la.insights) {
		limit = len(la.insights)
	}

	// 返回最新的洞察
	start := len(la.insights) - limit
	if start < 0 {
		start = 0
	}

	result := make([]Insight, limit)
	copy(result, la.insights[start:])
	return result
}

// QueryLogs 查詢日誌
func (la *LogAnalyzer) QueryLogs(filter LogFilter) []LogEntry {
	la.mu.RLock()
	defer la.mu.RUnlock()

	results := make([]LogEntry, 0)

	for _, entry := range la.entries {
		if la.matchFilter(entry, filter) {
			results = append(results, entry)
		}
	}

	return results
}

// LogFilter 日誌過濾器
type LogFilter struct {
	StartTime   time.Time
	EndTime     time.Time
	Level       string
	Component   string
	DeviceID    string
	Pattern     string
	AnomalyOnly bool
	Limit       int
}

// matchFilter 匹配過濾器
func (la *LogAnalyzer) matchFilter(entry LogEntry, filter LogFilter) bool {
	// 時間範圍
	if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
		return false
	}
	if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
		return false
	}

	// 級別
	if filter.Level != "" && entry.Level != filter.Level {
		return false
	}

	// 組件
	if filter.Component != "" && entry.Component != filter.Component {
		return false
	}

	// 設備 ID
	if filter.DeviceID != "" && entry.DeviceID != filter.DeviceID {
		return false
	}

	// 模式
	if filter.Pattern != "" && entry.Pattern != filter.Pattern {
		return false
	}

	// 僅異常
	if filter.AnomalyOnly && !entry.Anomaly {
		return false
	}

	return true
}

// ExportReport 導出報告
func (la *LogAnalyzer) ExportReport() map[string]interface{} {
	la.mu.RLock()
	defer la.mu.RUnlock()

	report := map[string]interface{}{
		"generated_at":    time.Now(),
		"statistics":      la.GetStatistics(),
		"insights":        la.GetInsights(10),
		"top_errors":      la.getTopErrors(),
		"device_health":   la.getDeviceHealth(),
		"recommendations": la.generateRecommendations(),
	}

	return report
}

// getTopErrors 獲取主要錯誤
func (la *LogAnalyzer) getTopErrors() []map[string]interface{} {
	errors := make([]map[string]interface{}, 0)

	for _, entry := range la.entries {
		if entry.Level == "error" || entry.Level == "critical" {
			errors = append(errors, map[string]interface{}{
				"timestamp": entry.Timestamp,
				"level":     entry.Level,
				"message":   entry.Message,
				"device_id": entry.DeviceID,
			})
		}

		if len(errors) >= 10 {
			break
		}
	}

	return errors
}

// getDeviceHealth 獲取設備健康狀態
func (la *LogAnalyzer) getDeviceHealth() map[string]string {
	health := make(map[string]string)

	for deviceID, count := range la.statistics.EntriesByDevice {
		errorCount := 0
		for _, entry := range la.entries {
			if entry.DeviceID == deviceID && (entry.Level == "error" || entry.Level == "critical") {
				errorCount++
			}
		}

		if count > 0 {
			errorRate := float64(errorCount) / float64(count) * 100
			if errorRate > 20 {
				health[deviceID] = "unhealthy"
			} else if errorRate > 10 {
				health[deviceID] = "degraded"
			} else {
				health[deviceID] = "healthy"
			}
		}
	}

	return health
}

// generateRecommendations 生成建議
func (la *LogAnalyzer) generateRecommendations() []string {
	recommendations := make([]string, 0)

	if la.statistics.ErrorRate > 5 {
		recommendations = append(recommendations, "High error rate detected. Consider reviewing system logs and recent changes.")
	}

	if la.statistics.AnomalyCount > 10 {
		recommendations = append(recommendations, "Multiple anomalies detected. Perform a system health check.")
	}

	for _, pattern := range la.statistics.TopPatterns {
		if pattern.Trend == "increasing" && pattern.Count > 50 {
			recommendations = append(recommendations,
				fmt.Sprintf("Pattern '%s' is increasing rapidly. Investigate the root cause.", pattern.Name))
		}
	}

	return recommendations
}

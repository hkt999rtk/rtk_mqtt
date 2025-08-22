package registry

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"

	"rtk_wrapper/pkg/types"
)

// Registry 定義 Wrapper 註冊表
type Registry struct {
	mu               sync.RWMutex
	wrappers         map[string]types.DeviceWrapper
	transformers     map[string]types.MessageTransformer
	uplinkRoutes     []RouteRule
	downlinkRoutes   []RouteRule
	autoDiscovery    bool
	discoveryTimeout int
}

// RouteRule 定義路由規則
type RouteRule struct {
	WrapperName  string
	Priority     int
	TopicPattern *regexp.Regexp
	PayloadRules []PayloadRule
	DeviceTypes  []string
}

// PayloadRule 定義 payload 匹配規則
type PayloadRule struct {
	FieldPath     string      // JSON 欄位路徑，如 "state" 或 "attributes.brightness"
	ExpectedType  string      // 預期類型：string, number, boolean, object, array
	ExpectedValue interface{} // 預期值（可選）
	Required      bool        // 是否必須存在
}

// NewRegistry 創建新的註冊表
func NewRegistry(autoDiscovery bool, discoveryTimeout int) *Registry {
	return &Registry{
		wrappers:         make(map[string]types.DeviceWrapper),
		transformers:     make(map[string]types.MessageTransformer),
		uplinkRoutes:     make([]RouteRule, 0),
		downlinkRoutes:   make([]RouteRule, 0),
		autoDiscovery:    autoDiscovery,
		discoveryTimeout: discoveryTimeout,
	}
}

// RegisterWrapper 註冊設備 wrapper
func (r *Registry) RegisterWrapper(name string, wrapper types.DeviceWrapper, transformer types.MessageTransformer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.wrappers[name]; exists {
		return fmt.Errorf("wrapper %s already registered", name)
	}

	r.wrappers[name] = wrapper
	r.transformers[name] = transformer

	log.Printf("Registered wrapper: %s (version: %s)", name, wrapper.Version())
	return nil
}

// RegisterUplinkRoute 註冊上行路由規則
func (r *Registry) RegisterUplinkRoute(wrapperName string, priority int, topicPattern string, payloadRules []PayloadRule, deviceTypes []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 編譯 topic 模式
	pattern, err := r.compileTopicPattern(topicPattern)
	if err != nil {
		return fmt.Errorf("invalid topic pattern %s: %w", topicPattern, err)
	}

	rule := RouteRule{
		WrapperName:  wrapperName,
		Priority:     priority,
		TopicPattern: pattern,
		PayloadRules: payloadRules,
		DeviceTypes:  deviceTypes,
	}

	r.uplinkRoutes = append(r.uplinkRoutes, rule)

	// 按優先級排序
	sort.Slice(r.uplinkRoutes, func(i, j int) bool {
		return r.uplinkRoutes[i].Priority > r.uplinkRoutes[j].Priority
	})

	log.Printf("Registered uplink route for %s: %s (priority: %d)", wrapperName, topicPattern, priority)
	return nil
}

// RegisterDownlinkRoute 註冊下行路由規則
func (r *Registry) RegisterDownlinkRoute(wrapperName string, priority int, topicPattern string, deviceTypes []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 編譯 topic 模式
	pattern, err := r.compileTopicPattern(topicPattern)
	if err != nil {
		return fmt.Errorf("invalid topic pattern %s: %w", topicPattern, err)
	}

	rule := RouteRule{
		WrapperName:  wrapperName,
		Priority:     priority,
		TopicPattern: pattern,
		DeviceTypes:  deviceTypes,
	}

	r.downlinkRoutes = append(r.downlinkRoutes, rule)

	// 按優先級排序
	sort.Slice(r.downlinkRoutes, func(i, j int) bool {
		return r.downlinkRoutes[i].Priority > r.downlinkRoutes[j].Priority
	})

	log.Printf("Registered downlink route for %s: %s (priority: %d)", wrapperName, topicPattern, priority)
	return nil
}

// FindUplinkWrapper 尋找適合的上行 wrapper
func (r *Registry) FindUplinkWrapper(topic string, payload *types.FlexiblePayload) types.MessageTransformer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.uplinkRoutes {
		if r.matchRoute(route, topic, payload) {
			if transformer, exists := r.transformers[route.WrapperName]; exists {
				return transformer
			}
		}
	}

	return nil
}

// FindDownlinkWrapper 尋找適合的下行 wrapper（根據設備類型）
func (r *Registry) FindDownlinkWrapper(deviceType string) types.MessageTransformer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.downlinkRoutes {
		for _, supportedType := range route.DeviceTypes {
			if supportedType == deviceType || supportedType == "*" {
				if transformer, exists := r.transformers[route.WrapperName]; exists {
					return transformer
				}
			}
		}
	}

	return nil
}

// GetWrappers 獲取所有註冊的 wrappers
func (r *Registry) GetWrappers() map[string]types.DeviceWrapper {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]types.DeviceWrapper)
	for name, wrapper := range r.wrappers {
		result[name] = wrapper
	}
	return result
}

// GetTransformers 獲取所有註冊的 transformers
func (r *Registry) GetTransformers() map[string]types.MessageTransformer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]types.MessageTransformer)
	for name, transformer := range r.transformers {
		result[name] = transformer
	}
	return result
}

// UnregisterWrapper 取消註冊 wrapper
func (r *Registry) UnregisterWrapper(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.wrappers[name]; !exists {
		return fmt.Errorf("wrapper %s not found", name)
	}

	delete(r.wrappers, name)
	delete(r.transformers, name)

	// 移除相關路由規則
	r.uplinkRoutes = r.removeRoutesByWrapper(r.uplinkRoutes, name)
	r.downlinkRoutes = r.removeRoutesByWrapper(r.downlinkRoutes, name)

	log.Printf("Unregistered wrapper: %s", name)
	return nil
}

// compileTopicPattern 編譯 topic 模式為正則表達式
func (r *Registry) compileTopicPattern(pattern string) (*regexp.Regexp, error) {
	// 轉換 MQTT wildcard 為正則表達式
	// + 匹配單個層級，# 匹配多個層級
	escaped := regexp.QuoteMeta(pattern)

	// 處理 {variable} 模式
	escaped = regexp.MustCompile(`\\{[^}]+\\}`).ReplaceAllString(escaped, `([^/]+)`)

	// 處理 MQTT wildcards
	escaped = strings.ReplaceAll(escaped, `\+`, `[^/]+`)
	escaped = strings.ReplaceAll(escaped, `\#`, `.+`)

	// 添加行首行尾錨點
	regexPattern := fmt.Sprintf("^%s$", escaped)

	return regexp.Compile(regexPattern)
}

// matchRoute 檢查路由規則是否匹配
func (r *Registry) matchRoute(route RouteRule, topic string, payload *types.FlexiblePayload) bool {
	// 檢查 topic 模式
	if !route.TopicPattern.MatchString(topic) {
		return false
	}

	// 檢查 payload 規則
	if payload != nil && len(route.PayloadRules) > 0 {
		return r.matchPayloadRules(route.PayloadRules, payload)
	}

	return true
}

// matchPayloadRules 檢查 payload 規則是否匹配
func (r *Registry) matchPayloadRules(rules []PayloadRule, payload *types.FlexiblePayload) bool {
	for _, rule := range rules {
		if !r.matchPayloadRule(rule, payload) {
			return false
		}
	}
	return true
}

// matchPayloadRule 檢查單個 payload 規則
func (r *Registry) matchPayloadRule(rule PayloadRule, payload *types.FlexiblePayload) bool {
	// 解析欄位路徑
	pathParts := strings.Split(rule.FieldPath, ".")
	value, exists := payload.GetNested(pathParts...)

	// 檢查必填欄位
	if rule.Required && !exists {
		return false
	}
	if !exists {
		return true // 非必填欄位，不存在時視為匹配
	}

	// 檢查類型
	if rule.ExpectedType != "" && !r.checkType(value, rule.ExpectedType) {
		return false
	}

	// 檢查值
	if rule.ExpectedValue != nil && value != rule.ExpectedValue {
		return false
	}

	return true
}

// checkType 檢查值的類型
func (r *Registry) checkType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	default:
		return true // 未知類型，不檢查
	}
}

// removeRoutesByWrapper 移除指定 wrapper 的路由規則
func (r *Registry) removeRoutesByWrapper(routes []RouteRule, wrapperName string) []RouteRule {
	filtered := make([]RouteRule, 0)
	for _, route := range routes {
		if route.WrapperName != wrapperName {
			filtered = append(filtered, route)
		}
	}
	return filtered
}

// Stats 獲取註冊表統計資訊
func (r *Registry) Stats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RegistryStats{
		TotalWrappers:     len(r.wrappers),
		TotalTransformers: len(r.transformers),
		UplinkRoutes:      len(r.uplinkRoutes),
		DownlinkRoutes:    len(r.downlinkRoutes),
		AutoDiscovery:     r.autoDiscovery,
		DiscoveryTimeout:  r.discoveryTimeout,
	}
}

// RegistryStats 定義註冊表統計資訊
type RegistryStats struct {
	TotalWrappers     int  `json:"total_wrappers"`
	TotalTransformers int  `json:"total_transformers"`
	UplinkRoutes      int  `json:"uplink_routes"`
	DownlinkRoutes    int  `json:"downlink_routes"`
	AutoDiscovery     bool `json:"auto_discovery"`
	DiscoveryTimeout  int  `json:"discovery_timeout"`
}

// GetRegistryStats 獲取註冊表統計（用於監控系統）
func (r *Registry) GetRegistryStats() map[string]interface{} {
	stats := r.Stats()

	return map[string]interface{}{
		"registered_wrappers":     stats.TotalWrappers,
		"registered_transformers": stats.TotalTransformers,
		"uplink_routes":           stats.UplinkRoutes,
		"downlink_routes":         stats.DownlinkRoutes,
		"auto_discovery":          stats.AutoDiscovery,
		"discovery_timeout":       stats.DiscoveryTimeout,
	}
}

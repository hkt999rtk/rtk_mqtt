package scenarios

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// EventBus 事件總線
type EventBus struct {
	subscribers map[string][]EventHandler
	events      chan Event
	history     []Event
	running     bool
	mu          sync.RWMutex
	logger      *logrus.Entry
	config      *EventConfig
}

// Event 事件
type Event struct {
	ID        string
	Type      string // device, sensor, system, user, automation
	Source    string
	Target    string
	Name      string
	Data      interface{}
	Timestamp time.Time
	Priority  int
	TTL       time.Duration // Time to live
}

// EventHandler 事件處理器
type EventHandler struct {
	ID       string
	Name     string
	Filter   EventFilter
	Handler  func(Event) error
	Async    bool
	Priority int
}

// EventFilter 事件過濾器
type EventFilter struct {
	Types       []string
	Sources     []string
	Names       []string
	MinPriority int
}

// EventConfig 事件配置
type EventConfig struct {
	QueueSize         int
	HistorySize       int
	ProcessTimeout    time.Duration
	RetryCount        int
	EnablePersistence bool
}

// EventChain 事件鏈
type EventChain struct {
	ID          string
	Name        string
	Description string
	Steps       []EventChainStep
	CurrentStep int
	State       string // pending, running, completed, failed
	StartTime   time.Time
	EndTime     time.Time
}

// EventChainStep 事件鏈步驟
type EventChainStep struct {
	ID           string
	Name         string
	TriggerEvent Event
	Actions      []EventAction
	NextSteps    []string
	Conditions   []EventCondition
	Timeout      time.Duration
}

// EventAction 事件動作
type EventAction struct {
	Type       string // publish, call, wait, log
	Target     string
	Parameters map[string]interface{}
}

// EventCondition 事件條件
type EventCondition struct {
	Type     string
	Property string
	Operator string
	Value    interface{}
}

// EventProcessor 事件處理器
type EventProcessor struct {
	eventBus     *EventBus
	chains       map[string]*EventChain
	activeChains map[string]*EventChain
	processors   map[string]func(Event) error
	running      bool
	mu           sync.RWMutex
	logger       *logrus.Entry
}

// EventStatistics 事件統計
type EventStatistics struct {
	TotalEvents     int64
	ProcessedEvents int64
	FailedEvents    int64
	DroppedEvents   int64
	AverageLatency  time.Duration
	EventsByType    map[string]int64
	EventsBySource  map[string]int64
	LastUpdate      time.Time
	mu              sync.RWMutex
}

// NewEventBus 創建新的事件總線
func NewEventBus(queueSize int) *EventBus {
	if queueSize <= 0 {
		queueSize = 1000
	}

	return &EventBus{
		subscribers: make(map[string][]EventHandler),
		events:      make(chan Event, queueSize),
		history:     make([]Event, 0),
		logger:      logrus.WithField("component", "event_bus"),
		config: &EventConfig{
			QueueSize:      queueSize,
			HistorySize:    100,
			ProcessTimeout: 5 * time.Second,
			RetryCount:     3,
		},
	}
}

// Start 啟動事件總線
func (eb *EventBus) Start(ctx context.Context) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.running {
		return fmt.Errorf("event bus is already running")
	}

	eb.running = true
	eb.logger.Info("Starting event bus")

	// 啟動事件處理循環
	go eb.processEvents(ctx)
	go eb.cleanupHistory(ctx)

	return nil
}

// Stop 停止事件總線
func (eb *EventBus) Stop() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if !eb.running {
		return fmt.Errorf("event bus is not running")
	}

	eb.running = false
	close(eb.events)
	eb.logger.Info("Stopping event bus")
	return nil
}

// Subscribe 訂閱事件
func (eb *EventBus) Subscribe(pattern string, handler EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if pattern == "" {
		pattern = "*"
	}

	if _, exists := eb.subscribers[pattern]; !exists {
		eb.subscribers[pattern] = make([]EventHandler, 0)
	}

	eb.subscribers[pattern] = append(eb.subscribers[pattern], handler)

	eb.logger.WithFields(logrus.Fields{
		"pattern":      pattern,
		"handler_id":   handler.ID,
		"handler_name": handler.Name,
	}).Debug("Event handler subscribed")

	return nil
}

// Unsubscribe 取消訂閱
func (eb *EventBus) Unsubscribe(pattern string, handlerID string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers, exists := eb.subscribers[pattern]
	if !exists {
		return fmt.Errorf("no subscribers for pattern %s", pattern)
	}

	newHandlers := make([]EventHandler, 0)
	for _, h := range handlers {
		if h.ID != handlerID {
			newHandlers = append(newHandlers, h)
		}
	}

	eb.subscribers[pattern] = newHandlers
	return nil
}

// Publish 發布事件
func (eb *EventBus) Publish(event Event) error {
	eb.mu.RLock()
	if !eb.running {
		eb.mu.RUnlock()
		return fmt.Errorf("event bus is not running")
	}
	eb.mu.RUnlock()

	// 設置時間戳
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 設置ID
	if event.ID == "" {
		event.ID = fmt.Sprintf("event_%d", time.Now().UnixNano())
	}

	// 嘗試發送事件
	select {
	case eb.events <- event:
		eb.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"event_name": event.Name,
		}).Debug("Event published")
		return nil
	case <-time.After(100 * time.Millisecond):
		eb.logger.WithField("event_id", event.ID).Warn("Event queue full, dropping event")
		return fmt.Errorf("event queue full")
	}
}

// processEvents 處理事件
func (eb *EventBus) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eb.events:
			if !ok {
				return
			}
			eb.handleEvent(event)
		}
	}
}

// handleEvent 處理單個事件
func (eb *EventBus) handleEvent(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// 添加到歷史
	eb.addToHistory(event)

	// 檢查TTL
	if event.TTL > 0 && time.Since(event.Timestamp) > event.TTL {
		eb.logger.WithField("event_id", event.ID).Debug("Event expired, skipping")
		return
	}

	// 查找匹配的處理器
	handlers := eb.findMatchingHandlers(event)

	// 按優先級排序處理器
	eb.sortHandlersByPriority(handlers)

	// 執行處理器
	for _, handler := range handlers {
		if handler.Async {
			go eb.executeHandler(handler, event)
		} else {
			eb.executeHandler(handler, event)
		}
	}
}

// findMatchingHandlers 查找匹配的處理器
func (eb *EventBus) findMatchingHandlers(event Event) []EventHandler {
	handlers := make([]EventHandler, 0)

	// 通配符訂閱者
	if wildcardHandlers, exists := eb.subscribers["*"]; exists {
		for _, h := range wildcardHandlers {
			if eb.matchesFilter(event, h.Filter) {
				handlers = append(handlers, h)
			}
		}
	}

	// 類型匹配訂閱者
	if typeHandlers, exists := eb.subscribers[event.Type]; exists {
		for _, h := range typeHandlers {
			if eb.matchesFilter(event, h.Filter) {
				handlers = append(handlers, h)
			}
		}
	}

	// 特定模式訂閱者
	pattern := fmt.Sprintf("%s.%s", event.Type, event.Name)
	if patternHandlers, exists := eb.subscribers[pattern]; exists {
		for _, h := range patternHandlers {
			if eb.matchesFilter(event, h.Filter) {
				handlers = append(handlers, h)
			}
		}
	}

	return handlers
}

// matchesFilter 匹配過濾器
func (eb *EventBus) matchesFilter(event Event, filter EventFilter) bool {
	// 檢查類型
	if len(filter.Types) > 0 {
		matched := false
		for _, t := range filter.Types {
			if t == event.Type {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 檢查來源
	if len(filter.Sources) > 0 {
		matched := false
		for _, s := range filter.Sources {
			if s == event.Source {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 檢查名稱
	if len(filter.Names) > 0 {
		matched := false
		for _, n := range filter.Names {
			if n == event.Name {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 檢查優先級
	if filter.MinPriority > 0 && event.Priority < filter.MinPriority {
		return false
	}

	return true
}

// sortHandlersByPriority 按優先級排序處理器
func (eb *EventBus) sortHandlersByPriority(handlers []EventHandler) {
	// 簡單的冒泡排序
	for i := 0; i < len(handlers)-1; i++ {
		for j := 0; j < len(handlers)-i-1; j++ {
			if handlers[j].Priority < handlers[j+1].Priority {
				handlers[j], handlers[j+1] = handlers[j+1], handlers[j]
			}
		}
	}
}

// executeHandler 執行處理器
func (eb *EventBus) executeHandler(handler EventHandler, event Event) {
	defer func() {
		if r := recover(); r != nil {
			eb.logger.WithFields(logrus.Fields{
				"handler_id": handler.ID,
				"event_id":   event.ID,
				"panic":      r,
			}).Error("Handler panic recovered")
		}
	}()

	// 設置超時
	done := make(chan error, 1)
	go func() {
		done <- handler.Handler(event)
	}()

	select {
	case err := <-done:
		if err != nil {
			eb.logger.WithError(err).WithFields(logrus.Fields{
				"handler_id": handler.ID,
				"event_id":   event.ID,
			}).Error("Handler execution failed")
		}
	case <-time.After(eb.config.ProcessTimeout):
		eb.logger.WithFields(logrus.Fields{
			"handler_id": handler.ID,
			"event_id":   event.ID,
		}).Warn("Handler execution timeout")
	}
}

// addToHistory 添加到歷史
func (eb *EventBus) addToHistory(event Event) {
	if len(eb.history) >= eb.config.HistorySize {
		// 移除最舊的事件
		eb.history = eb.history[1:]
	}
	eb.history = append(eb.history, event)
}

// cleanupHistory 清理歷史
func (eb *EventBus) cleanupHistory(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			eb.mu.Lock()
			// 保留最近的事件
			if len(eb.history) > eb.config.HistorySize {
				eb.history = eb.history[len(eb.history)-eb.config.HistorySize:]
			}
			eb.mu.Unlock()
		}
	}
}

// GetHistory 獲取歷史
func (eb *EventBus) GetHistory(limit int) []Event {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if limit <= 0 || limit > len(eb.history) {
		limit = len(eb.history)
	}

	result := make([]Event, limit)
	copy(result, eb.history[len(eb.history)-limit:])
	return result
}

// QueueSize 獲取隊列大小
func (eb *EventBus) QueueSize() int {
	return len(eb.events)
}

// NewEventProcessor 創建事件處理器
func NewEventProcessor(eventBus *EventBus) *EventProcessor {
	return &EventProcessor{
		eventBus:     eventBus,
		chains:       make(map[string]*EventChain),
		activeChains: make(map[string]*EventChain),
		processors:   make(map[string]func(Event) error),
		logger:       logrus.WithField("component", "event_processor"),
	}
}

// RegisterChain 註冊事件鏈
func (ep *EventProcessor) RegisterChain(chain *EventChain) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if _, exists := ep.chains[chain.ID]; exists {
		return fmt.Errorf("chain %s already exists", chain.ID)
	}

	ep.chains[chain.ID] = chain
	ep.logger.WithField("chain_id", chain.ID).Info("Event chain registered")
	return nil
}

// TriggerChain 觸發事件鏈
func (ep *EventProcessor) TriggerChain(chainID string, initialEvent Event) error {
	ep.mu.RLock()
	chain, exists := ep.chains[chainID]
	ep.mu.RUnlock()

	if !exists {
		return fmt.Errorf("chain %s not found", chainID)
	}

	// 創建活動鏈實例
	activeChain := &EventChain{
		ID:          fmt.Sprintf("%s_%d", chainID, time.Now().UnixNano()),
		Name:        chain.Name,
		Description: chain.Description,
		Steps:       chain.Steps,
		CurrentStep: 0,
		State:       "running",
		StartTime:   time.Now(),
	}

	ep.mu.Lock()
	ep.activeChains[activeChain.ID] = activeChain
	ep.mu.Unlock()

	// 執行鏈
	go ep.executeChain(activeChain, initialEvent)

	return nil
}

// executeChain 執行事件鏈
func (ep *EventProcessor) executeChain(chain *EventChain, event Event) {
	defer func() {
		ep.mu.Lock()
		chain.State = "completed"
		chain.EndTime = time.Now()
		ep.mu.Unlock()
	}()

	for chain.CurrentStep < len(chain.Steps) {
		step := chain.Steps[chain.CurrentStep]

		// 檢查條件
		if !ep.checkStepConditions(step, event) {
			ep.logger.WithField("step_id", step.ID).Debug("Step conditions not met, skipping")
			chain.CurrentStep++
			continue
		}

		// 執行動作
		for _, action := range step.Actions {
			err := ep.executeAction(action, event)
			if err != nil {
				ep.logger.WithError(err).WithField("action_type", action.Type).Error("Failed to execute action")
				chain.State = "failed"
				return
			}
		}

		// 處理下一步
		if len(step.NextSteps) > 0 {
			// 找到下一步的索引
			nextStepID := step.NextSteps[0]
			for i, s := range chain.Steps {
				if s.ID == nextStepID {
					chain.CurrentStep = i
					break
				}
			}
		} else {
			chain.CurrentStep++
		}
	}
}

// checkStepConditions 檢查步驟條件
func (ep *EventProcessor) checkStepConditions(step EventChainStep, event Event) bool {
	for _, condition := range step.Conditions {
		if !ep.evaluateCondition(condition, event) {
			return false
		}
	}
	return true
}

// evaluateCondition 評估條件
func (ep *EventProcessor) evaluateCondition(condition EventCondition, event Event) bool {
	// 簡化的條件評估
	switch condition.Type {
	case "event_type":
		return event.Type == condition.Value
	case "event_name":
		return event.Name == condition.Value
	case "priority":
		return ep.comparePriority(event.Priority, condition.Value.(int), condition.Operator)
	default:
		return true
	}
}

// comparePriority 比較優先級
func (ep *EventProcessor) comparePriority(current, target int, operator string) bool {
	switch operator {
	case "gt":
		return current > target
	case "lt":
		return current < target
	case "eq":
		return current == target
	case "ge":
		return current >= target
	case "le":
		return current <= target
	default:
		return false
	}
}

// executeAction 執行動作
func (ep *EventProcessor) executeAction(action EventAction, event Event) error {
	switch action.Type {
	case "publish":
		// 發布新事件
		newEvent := Event{
			Type:   action.Target,
			Source: "event_chain",
			Name:   action.Parameters["name"].(string),
			Data:   action.Parameters["data"],
		}
		return ep.eventBus.Publish(newEvent)

	case "call":
		// 調用處理器
		if processor, exists := ep.processors[action.Target]; exists {
			return processor(event)
		}
		return fmt.Errorf("processor %s not found", action.Target)

	case "wait":
		// 等待
		if duration, ok := action.Parameters["duration"].(time.Duration); ok {
			time.Sleep(duration)
		}
		return nil

	case "log":
		// 記錄日誌
		ep.logger.WithFields(logrus.Fields{
			"event_id": event.ID,
			"message":  action.Parameters["message"],
		}).Info("Event chain log")
		return nil

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// RegisterProcessor 註冊處理器
func (ep *EventProcessor) RegisterProcessor(name string, processor func(Event) error) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	ep.processors[name] = processor
	ep.logger.WithField("processor_name", name).Debug("Processor registered")
}

// GetActiveChains 獲取活動鏈
func (ep *EventProcessor) GetActiveChains() []*EventChain {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	chains := make([]*EventChain, 0, len(ep.activeChains))
	for _, chain := range ep.activeChains {
		chains = append(chains, chain)
	}
	return chains
}

// CreateEventStatistics 創建事件統計
func CreateEventStatistics() *EventStatistics {
	return &EventStatistics{
		EventsByType:   make(map[string]int64),
		EventsBySource: make(map[string]int64),
		LastUpdate:     time.Now(),
	}
}

// RecordEvent 記錄事件
func (es *EventStatistics) RecordEvent(event Event, success bool, latency time.Duration) {
	es.mu.Lock()
	defer es.mu.Unlock()

	es.TotalEvents++

	if success {
		es.ProcessedEvents++
	} else {
		es.FailedEvents++
	}

	// 更新平均延遲
	if es.ProcessedEvents > 0 {
		es.AverageLatency = (es.AverageLatency*time.Duration(es.ProcessedEvents-1) + latency) / time.Duration(es.ProcessedEvents)
	}

	// 按類型統計
	es.EventsByType[event.Type]++

	// 按來源統計
	es.EventsBySource[event.Source]++

	es.LastUpdate = time.Now()
}

// GetStatistics 獲取統計
func (es *EventStatistics) GetStatistics() map[string]interface{} {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return map[string]interface{}{
		"total_events":     es.TotalEvents,
		"processed_events": es.ProcessedEvents,
		"failed_events":    es.FailedEvents,
		"dropped_events":   es.DroppedEvents,
		"average_latency":  es.AverageLatency.Milliseconds(),
		"events_by_type":   es.EventsByType,
		"events_by_source": es.EventsBySource,
		"last_update":      es.LastUpdate,
	}
}

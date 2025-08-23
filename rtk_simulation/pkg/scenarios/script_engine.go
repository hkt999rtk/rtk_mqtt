package scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"rtk_simulation/pkg/devices/base"
)

// ScriptEngine 腳本引擎
type ScriptEngine struct {
	scripts        map[string]*Script
	runningScripts map[string]*ScriptExecution
	variables      map[string]interface{}
	functions      map[string]ScriptFunction
	devices        map[string]base.Device
	eventBus       *EventBus
	running        bool
	mu             sync.RWMutex
	logger         *logrus.Entry
	config         *ScriptConfig
}

// Script 腳本
type Script struct {
	ID          string
	Name        string
	Description string
	Version     string
	Language    string // yaml, json, simple
	Content     string
	Steps       []ScriptStep
	Variables   map[string]interface{}
	Parameters  []ScriptParameter
	Triggers    []ScriptTrigger
	Schedule    *ScriptSchedule
	Enabled     bool
}

// ScriptStep 腳本步驟
type ScriptStep struct {
	ID           string
	Name         string
	Type         string // action, condition, loop, parallel, sequence
	Action       *ScriptAction
	Condition    *ScriptCondition
	Loop         *ScriptLoop
	Steps        []ScriptStep // 子步驟
	Delay        time.Duration
	Timeout      time.Duration
	ErrorHandler string // continue, stop, retry
}

// ScriptAction 腳本動作
type ScriptAction struct {
	Type       string // device, scene, wait, log, variable, function
	Target     string
	Method     string
	Parameters map[string]interface{}
	Result     string // 變數名稱，用於儲存結果
}

// ScriptCondition 腳本條件
type ScriptCondition struct {
	Type       string // if, switch
	Expression string
	TrueSteps  []ScriptStep
	FalseSteps []ScriptStep
	Cases      map[string][]ScriptStep // for switch
}

// ScriptLoop 腳本循環
type ScriptLoop struct {
	Type      string        // for, while, foreach
	Count     int           // for loop
	Condition string        // while loop
	Variable  string        // foreach loop
	Items     []interface{} // foreach items
	Steps     []ScriptStep
}

// ScriptParameter 腳本參數
type ScriptParameter struct {
	Name         string
	Type         string // string, number, boolean, object, array
	Required     bool
	DefaultValue interface{}
	Description  string
}

// ScriptTrigger 腳本觸發器
type ScriptTrigger struct {
	Type   string // event, time, manual
	Event  string
	Cron   string // for time trigger
	Active bool
}

// ScriptSchedule 腳本排程
type ScriptSchedule struct {
	Type       string // once, recurring, cron
	StartTime  time.Time
	EndTime    time.Time
	Interval   time.Duration
	CronExpr   string
	TimeZone   string
	DaysOfWeek []string
}

// ScriptExecution 腳本執行
type ScriptExecution struct {
	ScriptID    string
	ExecutionID string
	StartTime   time.Time
	EndTime     time.Time
	State       string // running, paused, completed, failed
	CurrentStep string
	Variables   map[string]interface{}
	Stack       []string // 執行堆疊
	Output      []string
	Errors      []error
	Context     context.Context
	Cancel      context.CancelFunc
}

// ScriptFunction 腳本函數
type ScriptFunction func(args map[string]interface{}) (interface{}, error)

// ScriptConfig 腳本配置
type ScriptConfig struct {
	MaxConcurrentScripts int
	MaxExecutionTime     time.Duration
	EnableSandbox        bool
	AllowExternalCalls   bool
	LogLevel             string
}

// Timeline 時間軸
type Timeline struct {
	ID          string
	Name        string
	Description string
	Duration    time.Duration
	Events      []TimelineEvent
	CurrentTime time.Duration
	State       string // stopped, playing, paused
	Speed       float64
	Loop        bool
}

// TimelineEvent 時間軸事件
type TimelineEvent struct {
	Time     time.Duration
	Type     string // script, action, marker
	ScriptID string
	Action   *ScriptAction
	Label    string
	Data     map[string]interface{}
}

// NewScriptEngine 創建新的腳本引擎
func NewScriptEngine(config *ScriptConfig) *ScriptEngine {
	if config == nil {
		config = &ScriptConfig{
			MaxConcurrentScripts: 10,
			MaxExecutionTime:     30 * time.Minute,
			EnableSandbox:        true,
			AllowExternalCalls:   false,
			LogLevel:             "info",
		}
	}

	se := &ScriptEngine{
		scripts:        make(map[string]*Script),
		runningScripts: make(map[string]*ScriptExecution),
		variables:      make(map[string]interface{}),
		functions:      make(map[string]ScriptFunction),
		devices:        make(map[string]base.Device),
		config:         config,
		logger:         logrus.WithField("component", "script_engine"),
	}

	// 註冊內建函數
	se.registerBuiltinFunctions()

	return se
}

// registerBuiltinFunctions 註冊內建函數
func (se *ScriptEngine) registerBuiltinFunctions() {
	// 數學函數
	se.RegisterFunction("add", func(args map[string]interface{}) (interface{}, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return a + b, nil
	})

	se.RegisterFunction("multiply", func(args map[string]interface{}) (interface{}, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return a * b, nil
	})

	// 字串函數
	se.RegisterFunction("concat", func(args map[string]interface{}) (interface{}, error) {
		a, _ := args["a"].(string)
		b, _ := args["b"].(string)
		return a + b, nil
	})

	// 時間函數
	se.RegisterFunction("now", func(args map[string]interface{}) (interface{}, error) {
		return time.Now(), nil
	})

	se.RegisterFunction("sleep", func(args map[string]interface{}) (interface{}, error) {
		duration, _ := args["duration"].(float64)
		time.Sleep(time.Duration(duration) * time.Second)
		return nil, nil
	})

	// 設備函數
	se.RegisterFunction("getDeviceState", func(args map[string]interface{}) (interface{}, error) {
		deviceID, _ := args["device_id"].(string)
		if device, exists := se.devices[deviceID]; exists {
			return device.GetHealth(), nil
		}
		return nil, fmt.Errorf("device %s not found", deviceID)
	})

	// 日誌函數
	se.RegisterFunction("log", func(args map[string]interface{}) (interface{}, error) {
		message, _ := args["message"].(string)
		level, _ := args["level"].(string)

		switch level {
		case "debug":
			se.logger.Debug(message)
		case "info":
			se.logger.Info(message)
		case "warn":
			se.logger.Warn(message)
		case "error":
			se.logger.Error(message)
		default:
			se.logger.Info(message)
		}
		return nil, nil
	})
}

// Start 啟動腳本引擎
func (se *ScriptEngine) Start(ctx context.Context) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	if se.running {
		return fmt.Errorf("script engine is already running")
	}

	se.running = true
	se.logger.Info("Starting script engine")

	// 啟動管理循環
	go se.schedulerLoop(ctx)
	go se.executionMonitor(ctx)

	return nil
}

// Stop 停止腳本引擎
func (se *ScriptEngine) Stop() error {
	se.mu.Lock()
	defer se.mu.Unlock()

	if !se.running {
		return fmt.Errorf("script engine is not running")
	}

	// 停止所有運行中的腳本
	for _, exec := range se.runningScripts {
		if exec.Cancel != nil {
			exec.Cancel()
		}
	}

	se.running = false
	se.logger.Info("Stopping script engine")
	return nil
}

// LoadScript 載入腳本
func (se *ScriptEngine) LoadScript(script *Script) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	// 解析腳本內容
	if script.Language != "simple" {
		steps, err := se.parseScript(script.Content, script.Language)
		if err != nil {
			return fmt.Errorf("failed to parse script: %w", err)
		}
		script.Steps = steps
	}

	se.scripts[script.ID] = script
	se.logger.WithField("script_id", script.ID).Info("Script loaded")
	return nil
}

// LoadScriptFromFile 從檔案載入腳本
func (se *ScriptEngine) LoadScriptFromFile(filename string) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read script file: %w", err)
	}

	// 判斷檔案類型
	var script Script
	if err := yaml.Unmarshal(content, &script); err != nil {
		// 嘗試 JSON
		if err := json.Unmarshal(content, &script); err != nil {
			return fmt.Errorf("failed to parse script file: %w", err)
		}
		script.Language = "json"
	} else {
		script.Language = "yaml"
	}

	script.Content = string(content)
	return se.LoadScript(&script)
}

// parseScript 解析腳本
func (se *ScriptEngine) parseScript(content string, language string) ([]ScriptStep, error) {
	var steps []ScriptStep

	switch language {
	case "yaml":
		if err := yaml.Unmarshal([]byte(content), &steps); err != nil {
			return nil, err
		}
	case "json":
		if err := json.Unmarshal([]byte(content), &steps); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported script language: %s", language)
	}

	return steps, nil
}

// ExecuteScript 執行腳本
func (se *ScriptEngine) ExecuteScript(scriptID string, parameters map[string]interface{}) (*ScriptExecution, error) {
	se.mu.RLock()
	script, exists := se.scripts[scriptID]
	se.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("script %s not found", scriptID)
	}

	if !script.Enabled {
		return nil, fmt.Errorf("script %s is disabled", scriptID)
	}

	// 檢查並發限制
	se.mu.RLock()
	runningCount := len(se.runningScripts)
	se.mu.RUnlock()

	if runningCount >= se.config.MaxConcurrentScripts {
		return nil, fmt.Errorf("maximum concurrent scripts limit reached")
	}

	// 創建執行實例
	ctx, cancel := context.WithTimeout(context.Background(), se.config.MaxExecutionTime)

	execution := &ScriptExecution{
		ScriptID:    scriptID,
		ExecutionID: fmt.Sprintf("%s_%d", scriptID, time.Now().UnixNano()),
		StartTime:   time.Now(),
		State:       "running",
		Variables:   make(map[string]interface{}),
		Stack:       make([]string, 0),
		Output:      make([]string, 0),
		Errors:      make([]error, 0),
		Context:     ctx,
		Cancel:      cancel,
	}

	// 初始化變數
	for k, v := range script.Variables {
		execution.Variables[k] = v
	}
	for k, v := range parameters {
		execution.Variables[k] = v
	}

	se.mu.Lock()
	se.runningScripts[execution.ExecutionID] = execution
	se.mu.Unlock()

	se.logger.WithFields(logrus.Fields{
		"script_id":    scriptID,
		"execution_id": execution.ExecutionID,
	}).Info("Script execution started")

	// 執行腳本
	go se.executeSteps(script.Steps, execution)

	return execution, nil
}

// executeSteps 執行步驟
func (se *ScriptEngine) executeSteps(steps []ScriptStep, execution *ScriptExecution) {
	defer func() {
		if r := recover(); r != nil {
			se.logger.WithFields(logrus.Fields{
				"execution_id": execution.ExecutionID,
				"panic":        r,
			}).Error("Script execution panic")
			execution.State = "failed"
		}

		execution.EndTime = time.Now()
		if execution.State == "running" {
			execution.State = "completed"
		}

		se.logger.WithField("execution_id", execution.ExecutionID).Info("Script execution finished")
	}()

	for _, step := range steps {
		// 檢查是否被取消
		select {
		case <-execution.Context.Done():
			execution.State = "cancelled"
			return
		default:
		}

		// 更新當前步驟
		execution.CurrentStep = step.ID

		// 執行延遲
		if step.Delay > 0 {
			time.Sleep(step.Delay)
		}

		// 執行步驟
		err := se.executeStep(step, execution)
		if err != nil {
			execution.Errors = append(execution.Errors, err)

			switch step.ErrorHandler {
			case "stop":
				execution.State = "failed"
				return
			case "retry":
				// 重試一次
				err = se.executeStep(step, execution)
				if err != nil {
					execution.State = "failed"
					return
				}
			case "continue":
				// 繼續執行
			}
		}
	}
}

// executeStep 執行單個步驟
func (se *ScriptEngine) executeStep(step ScriptStep, execution *ScriptExecution) error {
	se.logger.WithFields(logrus.Fields{
		"step_id":   step.ID,
		"step_type": step.Type,
	}).Debug("Executing step")

	switch step.Type {
	case "action":
		return se.executeAction(step.Action, execution)

	case "condition":
		return se.executeCondition(step.Condition, execution)

	case "loop":
		return se.executeLoop(step.Loop, execution)

	case "parallel":
		return se.executeParallel(step.Steps, execution)

	case "sequence":
		se.executeSteps(step.Steps, execution)
		return nil

	default:
		return fmt.Errorf("unknown step type: %s", step.Type)
	}
}

// executeAction 執行動作
func (se *ScriptEngine) executeAction(action *ScriptAction, execution *ScriptExecution) error {
	if action == nil {
		return nil
	}

	// 替換變數
	params := se.substituteVariables(action.Parameters, execution.Variables)

	switch action.Type {
	case "device":
		return se.executeDeviceAction(action.Target, action.Method, params)

	case "scene":
		// 執行場景
		return nil

	case "wait":
		if duration, ok := params["duration"].(float64); ok {
			time.Sleep(time.Duration(duration) * time.Second)
		}
		return nil

	case "log":
		if message, ok := params["message"].(string); ok {
			execution.Output = append(execution.Output, message)
			se.logger.Info(message)
		}
		return nil

	case "variable":
		if action.Result != "" {
			execution.Variables[action.Result] = params["value"]
		}
		return nil

	case "function":
		if fn, exists := se.functions[action.Method]; exists {
			result, err := fn(params)
			if err != nil {
				return err
			}
			if action.Result != "" {
				execution.Variables[action.Result] = result
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeDeviceAction 執行設備動作
func (se *ScriptEngine) executeDeviceAction(target, method string, params map[string]interface{}) error {
	se.mu.RLock()
	device, exists := se.devices[target]
	se.mu.RUnlock()

	if !exists {
		// 嘗試按類型查找
		se.mu.RLock()
		for _, d := range se.devices {
			if d.GetDeviceType() == target {
				device = d
				break
			}
		}
		se.mu.RUnlock()

		if device == nil {
			return fmt.Errorf("device %s not found", target)
		}
	}

	cmd := base.Command{
		ID:         fmt.Sprintf("script_%d", time.Now().UnixNano()),
		Type:       method,
		Parameters: params,
		Timeout:    5 * time.Second,
	}

	return device.HandleCommand(cmd)
}

// executeCondition 執行條件
func (se *ScriptEngine) executeCondition(condition *ScriptCondition, execution *ScriptExecution) error {
	if condition == nil {
		return nil
	}

	// 評估條件
	result := se.evaluateExpression(condition.Expression, execution.Variables)

	if result {
		se.executeSteps(condition.TrueSteps, execution)
	} else {
		se.executeSteps(condition.FalseSteps, execution)
	}

	return nil
}

// executeLoop 執行循環
func (se *ScriptEngine) executeLoop(loop *ScriptLoop, execution *ScriptExecution) error {
	if loop == nil {
		return nil
	}

	switch loop.Type {
	case "for":
		for i := 0; i < loop.Count; i++ {
			execution.Variables["index"] = i
			se.executeSteps(loop.Steps, execution)
		}

	case "while":
		for se.evaluateExpression(loop.Condition, execution.Variables) {
			se.executeSteps(loop.Steps, execution)
		}

	case "foreach":
		for i, item := range loop.Items {
			execution.Variables["index"] = i
			execution.Variables[loop.Variable] = item
			se.executeSteps(loop.Steps, execution)
		}
	}

	return nil
}

// executeParallel 並行執行
func (se *ScriptEngine) executeParallel(steps []ScriptStep, execution *ScriptExecution) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(steps))

	for _, step := range steps {
		wg.Add(1)
		go func(s ScriptStep) {
			defer wg.Done()
			if err := se.executeStep(s, execution); err != nil {
				errors <- err
			}
		}(step)
	}

	wg.Wait()
	close(errors)

	// 收集錯誤
	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

// evaluateExpression 評估表達式
func (se *ScriptEngine) evaluateExpression(expression string, variables map[string]interface{}) bool {
	// 簡化的表達式評估
	// 實際應用中可以使用更複雜的表達式引擎
	return true
}

// substituteVariables 替換變數
func (se *ScriptEngine) substituteVariables(params map[string]interface{}, variables map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range params {
		if str, ok := v.(string); ok && len(str) > 2 && str[0] == '$' && str[1] == '{' && str[len(str)-1] == '}' {
			// 變數引用
			varName := str[2 : len(str)-1]
			if value, exists := variables[varName]; exists {
				result[k] = value
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}

// schedulerLoop 排程循環
func (se *ScriptEngine) schedulerLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			se.checkScheduledScripts()
		}
	}
}

// checkScheduledScripts 檢查排程腳本
func (se *ScriptEngine) checkScheduledScripts() {
	se.mu.RLock()
	defer se.mu.RUnlock()

	now := time.Now()

	for _, script := range se.scripts {
		if !script.Enabled || script.Schedule == nil {
			continue
		}

		// 檢查排程
		if se.shouldRunScript(script.Schedule, now) {
			go se.ExecuteScript(script.ID, nil)
		}
	}
}

// shouldRunScript 判斷是否應該執行腳本
func (se *ScriptEngine) shouldRunScript(schedule *ScriptSchedule, now time.Time) bool {
	switch schedule.Type {
	case "once":
		return now.After(schedule.StartTime) && now.Before(schedule.StartTime.Add(1*time.Minute))

	case "recurring":
		if now.Before(schedule.StartTime) || (!schedule.EndTime.IsZero() && now.After(schedule.EndTime)) {
			return false
		}
		// 簡化的週期性檢查
		return true

	case "cron":
		// 需要 cron 表達式解析器
		return false

	default:
		return false
	}
}

// executionMonitor 執行監控
func (se *ScriptEngine) executionMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			se.cleanupCompletedExecutions()
		}
	}
}

// cleanupCompletedExecutions 清理完成的執行
func (se *ScriptEngine) cleanupCompletedExecutions() {
	se.mu.Lock()
	defer se.mu.Unlock()

	for id, exec := range se.runningScripts {
		if exec.State == "completed" || exec.State == "failed" || exec.State == "cancelled" {
			if time.Since(exec.EndTime) > 5*time.Minute {
				delete(se.runningScripts, id)
			}
		}
	}
}

// RegisterDevice 註冊設備
func (se *ScriptEngine) RegisterDevice(deviceID string, device base.Device) {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.devices[deviceID] = device
	se.logger.WithField("device_id", deviceID).Debug("Device registered for scripts")
}

// RegisterFunction 註冊函數
func (se *ScriptEngine) RegisterFunction(name string, fn ScriptFunction) {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.functions[name] = fn
	se.logger.WithField("function_name", name).Debug("Function registered")
}

// SetVariable 設置變數
func (se *ScriptEngine) SetVariable(name string, value interface{}) {
	se.mu.Lock()
	defer se.mu.Unlock()

	se.variables[name] = value
}

// GetVariable 獲取變數
func (se *ScriptEngine) GetVariable(name string) interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()

	return se.variables[name]
}

// StopScript 停止腳本
func (se *ScriptEngine) StopScript(executionID string) error {
	se.mu.RLock()
	exec, exists := se.runningScripts[executionID]
	se.mu.RUnlock()

	if !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	if exec.Cancel != nil {
		exec.Cancel()
	}

	exec.State = "cancelled"
	return nil
}

// GetRunningScripts 獲取運行中的腳本
func (se *ScriptEngine) GetRunningScripts() []*ScriptExecution {
	se.mu.RLock()
	defer se.mu.RUnlock()

	scripts := make([]*ScriptExecution, 0, len(se.runningScripts))
	for _, exec := range se.runningScripts {
		scripts = append(scripts, exec)
	}
	return scripts
}

// GetScriptOutput 獲取腳本輸出
func (se *ScriptEngine) GetScriptOutput(executionID string) ([]string, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	exec, exists := se.runningScripts[executionID]
	if !exists {
		return nil, fmt.Errorf("execution %s not found", executionID)
	}

	return exec.Output, nil
}

// GetStatistics 獲取統計資訊
func (se *ScriptEngine) GetStatistics() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()

	stats := map[string]interface{}{
		"total_scripts":    len(se.scripts),
		"running_scripts":  len(se.runningScripts),
		"enabled_scripts":  0,
		"total_functions":  len(se.functions),
		"global_variables": len(se.variables),
	}

	// 統計啟用的腳本
	for _, script := range se.scripts {
		if script.Enabled {
			stats["enabled_scripts"] = stats["enabled_scripts"].(int) + 1
		}
	}

	// 統計執行狀態
	stateCount := make(map[string]int)
	for _, exec := range se.runningScripts {
		stateCount[exec.State]++
	}
	stats["execution_states"] = stateCount

	return stats
}

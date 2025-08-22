package monitoring

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"rtk_wrapper/pkg/types"
)

// LogLevel 定義日誌等級
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
	LogLevelPanic LogLevel = "panic"
)

// LogFormat 定義日誌格式
type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// LogOutput 定義日誌輸出
type LogOutput string

const (
	LogOutputStdout LogOutput = "stdout"
	LogOutputStderr LogOutput = "stderr"
	LogOutputFile   LogOutput = "file"
)

// LogConfig 日誌配置
type LogConfig struct {
	Level      LogLevel  `yaml:"level" json:"level"`
	Format     LogFormat `yaml:"format" json:"format"`
	Output     LogOutput `yaml:"output" json:"output"`
	FilePath   string    `yaml:"file_path" json:"file_path"`
	MaxSize    int       `yaml:"max_size" json:"max_size"` // MB
	MaxAge     int       `yaml:"max_age" json:"max_age"`   // 天
	MaxBackups int       `yaml:"max_backups" json:"max_backups"`
	Compress   bool      `yaml:"compress" json:"compress"`
}

// StructuredLogger 結構化日誌記錄器
type StructuredLogger struct {
	logger      *logrus.Logger
	config      LogConfig
	mu          sync.RWMutex
	metrics     *MetricsCollector
	startTime   time.Time
	logBuffer   []LogEntry
	bufferSize  int
	contextPool sync.Pool
}

// LogEntry 日誌條目
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// LogContext 日誌上下文
type LogContext struct {
	WrapperName  string
	MessageID    string
	DeviceID     string
	Direction    types.MessageDirection
	Topic        string
	PayloadSize  int64
	ProcessingID string
	TraceID      string
	SessionID    string
	UserData     map[string]interface{}
}

// NewStructuredLogger 創建結構化日誌記錄器
func NewStructuredLogger(config LogConfig, metrics *MetricsCollector) (*StructuredLogger, error) {
	logger := logrus.New()

	// 設定日誌等級
	level, err := logrus.ParseLevel(string(config.Level))
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	logger.SetLevel(level)

	// 設定日誌格式
	if config.Format == LogFormatJSON {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
			PrettyPrint:     false,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	// 設定日誌輸出
	var writer io.Writer
	switch config.Output {
	case LogOutputStdout:
		writer = os.Stdout
	case LogOutputStderr:
		writer = os.Stderr
	case LogOutputFile:
		if config.FilePath == "" {
			config.FilePath = "logs/wrapper.log"
		}

		// 確保目錄存在
		if err := os.MkdirAll(filepath.Dir(config.FilePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		writer = &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.MaxSize,
			MaxAge:     config.MaxAge,
			MaxBackups: config.MaxBackups,
			Compress:   config.Compress,
		}
	default:
		writer = os.Stdout
	}
	logger.SetOutput(writer)

	sl := &StructuredLogger{
		logger:     logger,
		config:     config,
		metrics:    metrics,
		startTime:  time.Now(),
		bufferSize: 1000,
		logBuffer:  make([]LogEntry, 0, 1000),
		contextPool: sync.Pool{
			New: func() interface{} {
				return &LogContext{
					UserData: make(map[string]interface{}),
				}
			},
		},
	}

	return sl, nil
}

// WithContext 創建帶上下文的日誌記錄器
func (sl *StructuredLogger) WithContext(ctx *LogContext) *ContextLogger {
	return &ContextLogger{
		logger:  sl,
		context: ctx,
	}
}

// NewLogContext 創建新的日誌上下文
func (sl *StructuredLogger) NewLogContext() *LogContext {
	ctx := sl.contextPool.Get().(*LogContext)
	// 重置上下文
	ctx.WrapperName = ""
	ctx.MessageID = ""
	ctx.DeviceID = ""
	ctx.Direction = types.DirectionUplink
	ctx.Topic = ""
	ctx.PayloadSize = 0
	ctx.ProcessingID = ""
	ctx.TraceID = ""
	ctx.SessionID = ""
	for k := range ctx.UserData {
		delete(ctx.UserData, k)
	}
	return ctx
}

// ReleaseLogContext 釋放日誌上下文
func (sl *StructuredLogger) ReleaseLogContext(ctx *LogContext) {
	sl.contextPool.Put(ctx)
}

// Debug 記錄調試日誌
func (sl *StructuredLogger) Debug(msg string, fields ...map[string]interface{}) {
	sl.logWithFields(logrus.DebugLevel, msg, fields...)
}

// Info 記錄信息日誌
func (sl *StructuredLogger) Info(msg string, fields ...map[string]interface{}) {
	sl.logWithFields(logrus.InfoLevel, msg, fields...)
}

// Warn 記錄警告日誌
func (sl *StructuredLogger) Warn(msg string, fields ...map[string]interface{}) {
	sl.logWithFields(logrus.WarnLevel, msg, fields...)
}

// Error 記錄錯誤日誌
func (sl *StructuredLogger) Error(msg string, err error, fields ...map[string]interface{}) {
	allFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			allFields[k] = v
		}
	}
	if err != nil {
		allFields["error"] = err.Error()
	}
	sl.logWithFields(logrus.ErrorLevel, msg, allFields)
}

// Fatal 記錄致命錯誤日誌
func (sl *StructuredLogger) Fatal(msg string, err error, fields ...map[string]interface{}) {
	allFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			allFields[k] = v
		}
	}
	if err != nil {
		allFields["error"] = err.Error()
	}
	sl.logWithFields(logrus.FatalLevel, msg, allFields)
}

// logWithFields 帶字段記錄日誌
func (sl *StructuredLogger) logWithFields(level logrus.Level, msg string, fields ...map[string]interface{}) {
	entry := sl.logger.WithFields(logrus.Fields{})

	// 添加字段
	for _, f := range fields {
		for k, v := range f {
			entry = entry.WithField(k, v)
		}
	}

	// 添加調用者信息
	if sl.config.Level == LogLevelDebug {
		if pc, file, line, ok := runtime.Caller(2); ok {
			funcName := runtime.FuncForPC(pc).Name()
			entry = entry.WithField("caller", fmt.Sprintf("%s:%d %s", filepath.Base(file), line, funcName))
		}
	}

	// 記錄日誌
	entry.Log(level, msg)

	// 更新統計
	if sl.metrics != nil {
		sl.metrics.RecordLogMessage(level.String())
	}

	// 緩存重要日誌
	if level >= logrus.WarnLevel {
		sl.bufferLogEntry(level, msg, fields...)
	}
}

// bufferLogEntry 緩存日誌條目
func (sl *StructuredLogger) bufferLogEntry(level logrus.Level, msg string, fields ...map[string]interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	allFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			allFields[k] = v
		}
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   msg,
		Fields:    allFields,
	}

	// 如果有錯誤，提取到單獨字段
	if errField, exists := allFields["error"]; exists {
		if errStr, ok := errField.(string); ok {
			entry.Error = errStr
		}
	}

	// 添加到緩衝區
	if len(sl.logBuffer) >= sl.bufferSize {
		// 移除最舊的條目
		sl.logBuffer = sl.logBuffer[1:]
	}
	sl.logBuffer = append(sl.logBuffer, entry)
}

// GetRecentLogs 獲取最近的日誌
func (sl *StructuredLogger) GetRecentLogs(limit int) []LogEntry {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	if limit <= 0 || limit > len(sl.logBuffer) {
		limit = len(sl.logBuffer)
	}

	logs := make([]LogEntry, limit)
	start := len(sl.logBuffer) - limit
	copy(logs, sl.logBuffer[start:])

	return logs
}

// SetLevel 動態設置日誌等級
func (sl *StructuredLogger) SetLevel(level LogLevel) error {
	logrusLevel, err := logrus.ParseLevel(string(level))
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.logger.SetLevel(logrusLevel)
	sl.config.Level = level

	sl.Info("Log level changed", map[string]interface{}{
		"new_level": string(level),
	})

	return nil
}

// GetLogStats 獲取日誌統計
func (sl *StructuredLogger) GetLogStats() map[string]interface{} {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	return map[string]interface{}{
		"current_level":   string(sl.config.Level),
		"format":          string(sl.config.Format),
		"output":          string(sl.config.Output),
		"buffered_logs":   len(sl.logBuffer),
		"uptime":          time.Since(sl.startTime).String(),
		"buffer_capacity": sl.bufferSize,
	}
}

// ContextLogger 帶上下文的日誌記錄器
type ContextLogger struct {
	logger  *StructuredLogger
	context *LogContext
}

// Debug 帶上下文的調試日誌
func (cl *ContextLogger) Debug(msg string, fields ...map[string]interface{}) {
	cl.logWithContext(logrus.DebugLevel, msg, fields...)
}

// Info 帶上下文的信息日誌
func (cl *ContextLogger) Info(msg string, fields ...map[string]interface{}) {
	cl.logWithContext(logrus.InfoLevel, msg, fields...)
}

// Warn 帶上下文的警告日誌
func (cl *ContextLogger) Warn(msg string, fields ...map[string]interface{}) {
	cl.logWithContext(logrus.WarnLevel, msg, fields...)
}

// Error 帶上下文的錯誤日誌
func (cl *ContextLogger) Error(msg string, err error, fields ...map[string]interface{}) {
	allFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			allFields[k] = v
		}
	}
	if err != nil {
		allFields["error"] = err.Error()
	}
	cl.logWithContext(logrus.ErrorLevel, msg, allFields)
}

// logWithContext 帶上下文記錄日誌
func (cl *ContextLogger) logWithContext(level logrus.Level, msg string, fields ...map[string]interface{}) {
	// 合併上下文字段
	contextFields := map[string]interface{}{}

	if cl.context.WrapperName != "" {
		contextFields["wrapper"] = cl.context.WrapperName
	}
	if cl.context.MessageID != "" {
		contextFields["message_id"] = cl.context.MessageID
	}
	if cl.context.DeviceID != "" {
		contextFields["device_id"] = cl.context.DeviceID
	}
	// Always include direction info
	contextFields["direction"] = cl.context.Direction.String()
	if cl.context.Topic != "" {
		contextFields["topic"] = cl.context.Topic
	}
	if cl.context.PayloadSize > 0 {
		contextFields["payload_size"] = cl.context.PayloadSize
	}
	if cl.context.ProcessingID != "" {
		contextFields["processing_id"] = cl.context.ProcessingID
	}
	if cl.context.TraceID != "" {
		contextFields["trace_id"] = cl.context.TraceID
	}
	if cl.context.SessionID != "" {
		contextFields["session_id"] = cl.context.SessionID
	}

	// 添加用戶數據
	for k, v := range cl.context.UserData {
		contextFields[fmt.Sprintf("user_%s", k)] = v
	}

	// 合併所有字段
	allFields := []map[string]interface{}{contextFields}
	allFields = append(allFields, fields...)

	cl.logger.logWithFields(level, msg, allFields...)
}

// MessageLogger 訊息處理專用日誌記錄器
type MessageLogger struct {
	*ContextLogger
}

// LogMessageReceived 記錄訊息接收
func (ml *MessageLogger) LogMessageReceived(topic string, payloadSize int) {
	ml.Info("Message received", map[string]interface{}{
		"topic":        topic,
		"payload_size": payloadSize,
	})
}

// LogMessageProcessed 記錄訊息處理完成
func (ml *MessageLogger) LogMessageProcessed(duration time.Duration) {
	ml.Info("Message processed", map[string]interface{}{
		"processing_duration": duration.String(),
		"duration_ms":         duration.Milliseconds(),
	})
}

// LogTransformError 記錄轉換錯誤
func (ml *MessageLogger) LogTransformError(err error, originalTopic string) {
	ml.Error("Message transformation failed", err, map[string]interface{}{
		"original_topic": originalTopic,
	})
}

// LogRoutingDecision 記錄路由決策
func (ml *MessageLogger) LogRoutingDecision(selectedWrapper string, score int) {
	ml.Debug("Routing decision made", map[string]interface{}{
		"selected_wrapper": selectedWrapper,
		"match_score":      score,
	})
}

// NewMessageLogger 創建訊息日誌記錄器
func (sl *StructuredLogger) NewMessageLogger(ctx *LogContext) *MessageLogger {
	return &MessageLogger{
		ContextLogger: sl.WithContext(ctx),
	}
}

// GetDefaultLogConfig 獲取預設日誌配置
func GetDefaultLogConfig() LogConfig {
	return LogConfig{
		Level:      LogLevelInfo,
		Format:     LogFormatText,
		Output:     LogOutputStdout,
		FilePath:   "logs/wrapper.log",
		MaxSize:    100,
		MaxAge:     7,
		MaxBackups: 3,
		Compress:   true,
	}
}

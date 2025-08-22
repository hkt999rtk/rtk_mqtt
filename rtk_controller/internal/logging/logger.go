package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"rtk_controller/internal/config"
)

// Logger wraps logrus with additional functionality
type Logger struct {
	*logrus.Logger
	config  config.LoggingConfig
	fileLog *lumberjack.Logger
}

// NewLogger creates a new structured logger
func NewLogger(cfg config.LoggingConfig) (*Logger, error) {
	logger := &Logger{
		Logger: logrus.New(),
		config: cfg,
	}

	// Setup log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
		logger.WithField("invalid_level", cfg.Level).Warn("Invalid log level, using INFO")
	}
	logger.SetLevel(level)

	// Setup formatter
	switch cfg.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
			},
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	}

	// Setup output
	if err := logger.setupOutput(); err != nil {
		return nil, fmt.Errorf("failed to setup logger output: %w", err)
	}

	return logger, nil
}

// setupOutput configures log output destinations
func (l *Logger) setupOutput() error {
	var writers []io.Writer

	// Always include stdout/stderr
	writers = append(writers, os.Stdout)

	// Add file output if configured
	if l.config.File != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(l.config.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Setup rotating file logger
		l.fileLog = &lumberjack.Logger{
			Filename:   l.config.File,
			MaxSize:    l.config.MaxSize,    // MB
			MaxBackups: l.config.MaxBackups, // number of backups
			MaxAge:     l.config.MaxAge,     // days
			Compress:   l.config.Compress,   // compress rotated files
		}

		writers = append(writers, l.fileLog)
	}

	// Set multi-writer output
	if len(writers) > 1 {
		l.SetOutput(io.MultiWriter(writers...))
	} else {
		l.SetOutput(writers[0])
	}

	return nil
}

// UpdateConfig updates logger configuration
func (l *Logger) UpdateConfig(cfg config.LoggingConfig) error {
	l.config = cfg

	// Update log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
		l.WithField("invalid_level", cfg.Level).Warn("Invalid log level, using INFO")
	}
	l.SetLevel(level)

	// Update formatter if needed
	switch cfg.Format {
	case "json":
		l.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	case "text":
		l.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	}

	// Update output
	return l.setupOutput()
}

// Close closes any file handles
func (l *Logger) Close() error {
	if l.fileLog != nil {
		return l.fileLog.Close()
	}
	return nil
}

// GetLogLevel returns current log level as string
func (l *Logger) GetLogLevel() string {
	return l.GetLevel().String()
}

// SetLogLevel sets log level from string
func (l *Logger) SetLogLevel(level string) error {
	parsed, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	l.SetLevel(parsed)
	l.WithField("new_level", level).Info("Log level updated")

	return nil
}

// AuditLogger provides audit logging functionality
type AuditLogger struct {
	logger *Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(cfg config.LoggingConfig) (*AuditLogger, error) {
	// Create dedicated audit log configuration
	auditCfg := cfg
	if cfg.File != "" {
		// Change file extension to .audit.log
		ext := filepath.Ext(cfg.File)
		base := cfg.File[:len(cfg.File)-len(ext)]
		auditCfg.File = base + ".audit" + ext
	}

	logger, err := NewLogger(auditCfg)
	if err != nil {
		return nil, err
	}

	return &AuditLogger{logger: logger}, nil
}

// LogAction logs an audit action
func (a *AuditLogger) LogAction(ctx context.Context, action, user, resource string, details map[string]interface{}) {
	entry := a.logger.WithFields(logrus.Fields{
		"audit_action":   action,
		"audit_user":     user,
		"audit_resource": resource,
		"audit_time":     time.Now().UTC(),
	})

	// Add context fields if available
	if ctx != nil {
		if reqID := ctx.Value("request_id"); reqID != nil {
			entry = entry.WithField("request_id", reqID)
		}
		if clientIP := ctx.Value("client_ip"); clientIP != nil {
			entry = entry.WithField("client_ip", clientIP)
		}
	}

	// Add details
	if details != nil {
		for k, v := range details {
			entry = entry.WithField("detail_"+k, v)
		}
	}

	entry.Info("Audit log entry")
}

// LogCommandExecution logs command execution for audit
func (a *AuditLogger) LogCommandExecution(ctx context.Context, user, deviceID, command string, args map[string]interface{}, success bool, error string) {
	details := map[string]interface{}{
		"device_id": deviceID,
		"command":   command,
		"args":      args,
		"success":   success,
	}

	if error != "" {
		details["error"] = error
	}

	action := "command_execute"
	if !success {
		action = "command_failed"
	}

	a.LogAction(ctx, action, user, "device:"+deviceID, details)
}

// LogConfigChange logs configuration changes for audit
func (a *AuditLogger) LogConfigChange(ctx context.Context, user, key string, oldValue, newValue interface{}) {
	details := map[string]interface{}{
		"config_key":  key,
		"old_value":   oldValue,
		"new_value":   newValue,
		"change_time": time.Now().UTC(),
	}

	a.LogAction(ctx, "config_change", user, "config:"+key, details)
}

// LogAuthentication logs authentication attempts
func (a *AuditLogger) LogAuthentication(ctx context.Context, user, method string, success bool, reason string) {
	details := map[string]interface{}{
		"auth_method": method,
		"success":     success,
	}

	if reason != "" {
		details["reason"] = reason
	}

	action := "auth_success"
	if !success {
		action = "auth_failure"
	}

	a.LogAction(ctx, action, user, "auth:"+method, details)
}

// Close closes audit logger
func (a *AuditLogger) Close() error {
	return a.logger.Close()
}

// PerformanceLogger provides performance monitoring logging
type PerformanceLogger struct {
	logger *Logger
}

// NewPerformanceLogger creates a new performance logger
func NewPerformanceLogger(cfg config.LoggingConfig) (*PerformanceLogger, error) {
	// Create dedicated performance log configuration
	perfCfg := cfg
	if cfg.File != "" {
		ext := filepath.Ext(cfg.File)
		base := cfg.File[:len(cfg.File)-len(ext)]
		perfCfg.File = base + ".perf" + ext
	}

	logger, err := NewLogger(perfCfg)
	if err != nil {
		return nil, err
	}

	return &PerformanceLogger{logger: logger}, nil
}

// LogOperationDuration logs operation execution time
func (p *PerformanceLogger) LogOperationDuration(operation string, duration time.Duration, metadata map[string]interface{}) {
	entry := p.logger.WithFields(logrus.Fields{
		"operation":        operation,
		"duration_ms":      duration.Milliseconds(),
		"duration_ns":      duration.Nanoseconds(),
		"performance_time": time.Now().UTC(),
	})

	if metadata != nil {
		for k, v := range metadata {
			entry = entry.WithField(k, v)
		}
	}

	// Log as warning if duration exceeds thresholds
	if duration > 5*time.Second {
		entry.Warn("Slow operation detected")
	} else if duration > 1*time.Second {
		entry.Info("Operation completed")
	} else {
		entry.Debug("Operation completed")
	}
}

// LogMemoryUsage logs memory usage statistics
func (p *PerformanceLogger) LogMemoryUsage(component string, allocMB, sysMB float64) {
	p.logger.WithFields(logrus.Fields{
		"component": component,
		"alloc_mb":  allocMB,
		"sys_mb":    sysMB,
		"perf_time": time.Now().UTC(),
		"perf_type": "memory",
	}).Debug("Memory usage")
}

// Close closes performance logger
func (p *PerformanceLogger) Close() error {
	return p.logger.Close()
}

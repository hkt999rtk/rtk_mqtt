package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SessionManager 管理 MCP 診斷會話
type SessionManager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
	config   SessionConfig
	logger   *logrus.Logger

	// 清理工作
	stopCh  chan struct{}
	started bool
}

// Session 表示一個 MCP 診斷會話
type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id,omitempty"`
	DeviceID  string                 `json:"device_id,omitempty"`
	Intent    string                 `json:"intent,omitempty"`
	Status    SessionStatus          `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	ToolCalls []SessionToolCall      `json:"tool_calls"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SessionStatus 會話狀態
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusExpired   SessionStatus = "expired"
	SessionStatusCancelled SessionStatus = "cancelled"
)

// SessionToolCall 會話中的工具調用記錄
type SessionToolCall struct {
	ID          string                 `json:"id"`
	ToolName    string                 `json:"tool_name"`
	Arguments   map[string]interface{} `json:"arguments"`
	Result      interface{}            `json:"result,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
}

// NewSessionManager 建立會話管理器
func NewSessionManager(config SessionConfig, logger *logrus.Logger) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		config:   config,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

// Start 啟動會話管理器
func (sm *SessionManager) Start(ctx context.Context) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.started {
		return fmt.Errorf("session manager already started")
	}

	// 啟動清理工作
	if sm.config.AutoCleanup {
		go sm.cleanupWorker(ctx)
	}

	sm.started = true
	sm.logger.Info("Session manager started")

	return nil
}

// Stop 停止會話管理器
func (sm *SessionManager) Stop() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if !sm.started {
		return nil
	}

	close(sm.stopCh)
	sm.started = false

	sm.logger.Info("Session manager stopped")
	return nil
}

// CreateSession 建立新會話
func (sm *SessionManager) CreateSession(ctx context.Context, options *SessionOptions) (*Session, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// 檢查會話數量限制
	if len(sm.sessions) >= sm.config.MaxConcurrent {
		return nil, fmt.Errorf("maximum concurrent sessions exceeded (%d)", sm.config.MaxConcurrent)
	}

	// 建立新會話
	now := time.Now()
	session := &Session{
		ID:        uuid.New().String(),
		Status:    SessionStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(sm.config.Timeout),
		ToolCalls: make([]SessionToolCall, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 設定選項
	if options != nil {
		session.UserID = options.UserID
		session.DeviceID = options.DeviceID
		session.Intent = options.Intent

		if options.Metadata != nil {
			for k, v := range options.Metadata {
				session.Metadata[k] = v
			}
		}
	}

	// 儲存會話
	sm.sessions[session.ID] = session

	sm.logger.WithFields(logrus.Fields{
		"session_id": session.ID,
		"user_id":    session.UserID,
		"device_id":  session.DeviceID,
		"intent":     session.Intent,
	}).Info("Session created")

	return session, nil
}

// GetSession 取得會話
func (sm *SessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	// 檢查會話是否過期
	if time.Now().After(session.ExpiresAt) && session.Status == SessionStatusActive {
		session.Status = SessionStatusExpired
		session.UpdatedAt = time.Now()
	}

	return session, nil
}

// UpdateSession 更新會話
func (sm *SessionManager) UpdateSession(sessionID string, updates map[string]interface{}) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// 更新會話資訊
	session.UpdatedAt = time.Now()

	if intent, ok := updates["intent"].(string); ok {
		session.Intent = intent
	}

	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		for k, v := range metadata {
			session.Metadata[k] = v
		}
	}

	sm.logger.WithField("session_id", sessionID).Debug("Session updated")

	return nil
}

// CloseSession 關閉會話
func (sm *SessionManager) CloseSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// 更新會話狀態
	session.Status = SessionStatusCompleted
	session.UpdatedAt = time.Now()

	// 從活動會話中移除
	delete(sm.sessions, sessionID)

	sm.logger.WithField("session_id", sessionID).Info("Session closed")

	return nil
}

// CancelSession 取消會話
func (sm *SessionManager) CancelSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// 更新會話狀態
	session.Status = SessionStatusCancelled
	session.UpdatedAt = time.Now()

	// 從活動會話中移除
	delete(sm.sessions, sessionID)

	sm.logger.WithField("session_id", sessionID).Info("Session cancelled")

	return nil
}

// ListSessions 列出所有活動會話
func (sm *SessionManager) ListSessions() []*Session {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// AddToolCall 新增工具調用記錄
func (sm *SessionManager) AddToolCall(sessionID, toolName string, arguments map[string]interface{}) (string, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	if session.Status != SessionStatusActive {
		return "", fmt.Errorf("session %s is not active", sessionID)
	}

	// 建立工具調用記錄
	toolCall := SessionToolCall{
		ID:        uuid.New().String(),
		ToolName:  toolName,
		Arguments: arguments,
		StartedAt: time.Now(),
	}

	// 新增到會話中
	session.ToolCalls = append(session.ToolCalls, toolCall)
	session.UpdatedAt = time.Now()

	sm.logger.WithFields(logrus.Fields{
		"session_id":   sessionID,
		"tool_call_id": toolCall.ID,
		"tool_name":    toolName,
	}).Debug("Tool call added to session")

	return toolCall.ID, nil
}

// CompleteToolCall 完成工具調用
func (sm *SessionManager) CompleteToolCall(sessionID, toolCallID string, result interface{}, err error) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// 找到對應的工具調用
	for i := range session.ToolCalls {
		if session.ToolCalls[i].ID == toolCallID {
			now := time.Now()
			session.ToolCalls[i].CompletedAt = &now
			session.ToolCalls[i].Result = result
			session.ToolCalls[i].Success = (err == nil)

			if err != nil {
				session.ToolCalls[i].Error = err.Error()
			}

			session.UpdatedAt = now

			sm.logger.WithFields(logrus.Fields{
				"session_id":   sessionID,
				"tool_call_id": toolCallID,
				"success":      session.ToolCalls[i].Success,
			}).Debug("Tool call completed")

			return nil
		}
	}

	return fmt.Errorf("tool call %s not found in session %s", toolCallID, sessionID)
}

// GetSessionStats 取得會話統計
func (sm *SessionManager) GetSessionStats() SessionStats {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	stats := SessionStats{
		TotalActive: len(sm.sessions),
	}

	for _, session := range sm.sessions {
		stats.TotalToolCalls += len(session.ToolCalls)

		// 統計成功/失敗的工具調用
		for _, toolCall := range session.ToolCalls {
			if toolCall.CompletedAt != nil {
				if toolCall.Success {
					stats.SuccessfulToolCalls++
				} else {
					stats.FailedToolCalls++
				}
			}
		}
	}

	return stats
}

// SessionStats 會話統計資訊
type SessionStats struct {
	TotalActive         int `json:"total_active"`
	TotalToolCalls      int `json:"total_tool_calls"`
	SuccessfulToolCalls int `json:"successful_tool_calls"`
	FailedToolCalls     int `json:"failed_tool_calls"`
}

// cleanupWorker 定期清理過期會話
func (sm *SessionManager) cleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(sm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopCh:
			return
		case <-ticker.C:
			sm.cleanupExpiredSessions()
		}
	}
}

// cleanupExpiredSessions 清理過期會話
func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	expiredCount := 0

	for sessionID, session := range sm.sessions {
		if now.After(session.ExpiresAt) && session.Status == SessionStatusActive {
			session.Status = SessionStatusExpired
			session.UpdatedAt = now
			delete(sm.sessions, sessionID)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		sm.logger.WithField("count", expiredCount).Info("Expired sessions cleaned up")
	}
}

package command

import (
	"fmt"
	"time"

	"rtk_controller/pkg/types"

	log "github.com/sirupsen/logrus"
)

// Executor provides high-level command operations
type Executor struct {
	manager *Manager
}

// NewExecutor creates a new command executor
func NewExecutor(manager *Manager) *Executor {
	return &Executor{
		manager: manager,
	}
}

// RebootDevice sends a reboot command to a device
func (e *Executor) RebootDevice(tenant, site, deviceID string, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"force": false,
		"delay": 0,
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "reboot", args, timeoutSeconds)
}

// RestartService sends a service restart command to a device
func (e *Executor) RestartService(tenant, site, deviceID, serviceName string, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"service": serviceName,
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "restart_service", args, timeoutSeconds)
}

// GetSystemInfo requests system information from a device
func (e *Executor) GetSystemInfo(tenant, site, deviceID string, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"include": []string{"cpu", "memory", "disk", "network"},
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "get_system_info", args, timeoutSeconds)
}

// UpdateConfig sends a configuration update command to a device
func (e *Executor) UpdateConfig(tenant, site, deviceID string, configData map[string]interface{}, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"config": configData,
		"restart_required": false,
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "update_config", args, timeoutSeconds)
}

// RunDiagnostics sends a diagnostics command to a device
func (e *Executor) RunDiagnostics(tenant, site, deviceID string, diagnosticType string, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"type": diagnosticType,
		"level": "basic",
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "run_diagnostics", args, timeoutSeconds)
}

// SetLogLevel changes the log level of a device
func (e *Executor) SetLogLevel(tenant, site, deviceID, logLevel string, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"level": logLevel,
		"persistent": true,
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "set_log_level", args, timeoutSeconds)
}

// GetLogs requests log files from a device
func (e *Executor) GetLogs(tenant, site, deviceID string, logType string, lines int, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"type": logType,
		"lines": lines,
		"format": "json",
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "get_logs", args, timeoutSeconds)
}

// FirmwareUpdate sends a firmware update command to a device
func (e *Executor) FirmwareUpdate(tenant, site, deviceID, firmwareURL, version string, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"url": firmwareURL,
		"version": version,
		"verify_checksum": true,
		"auto_reboot": false,
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "firmware_update", args, timeoutSeconds)
}

// WiFiScan requests a WiFi scan from a device
func (e *Executor) WiFiScan(tenant, site, deviceID string, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"scan_type": "active",
		"duration": 10,
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "wifi_scan", args, timeoutSeconds)
}

// NetworkTest runs network connectivity tests on a device
func (e *Executor) NetworkTest(tenant, site, deviceID string, testType string, target string, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"test_type": testType, // ping, traceroute, speedtest
		"target": target,
		"count": 5,
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "network_test", args, timeoutSeconds)
}

// WaitForCommand waits for a command to complete with polling
func (e *Executor) WaitForCommand(commandID string, pollIntervalSeconds int, maxWaitSeconds int) (*types.DeviceCommand, error) {
	startTime := time.Now()
	maxDuration := time.Duration(maxWaitSeconds) * time.Second
	pollInterval := time.Duration(pollIntervalSeconds) * time.Second
	
	for {
		// Check if max wait time exceeded
		if time.Since(startTime) > maxDuration {
			return nil, fmt.Errorf("timeout waiting for command %s", commandID)
		}
		
		// Get current command status
		command, err := e.manager.GetCommand(commandID)
		if err != nil {
			return nil, fmt.Errorf("failed to get command status: %w", err)
		}
		
		// Check if command is completed
		switch command.Status {
		case "completed", "failed", "timeout":
			return command, nil
		case "pending", "sent", "ack":
			// Command still in progress, continue polling
			log.WithFields(log.Fields{
				"command_id": commandID,
				"status": command.Status,
			}).Debug("Command still in progress")
		}
		
		// Wait before next poll
		time.Sleep(pollInterval)
	}
}

// BatchExecute executes multiple commands on the same device sequentially
func (e *Executor) BatchExecute(tenant, site, deviceID string, commands []BatchCommand, timeoutSeconds int) ([]*types.DeviceCommand, error) {
	var results []*types.DeviceCommand
	
	for i, batchCmd := range commands {
		log.WithFields(log.Fields{
			"device_id": deviceID,
			"operation": batchCmd.Operation,
			"batch_index": i,
		}).Info("Executing batch command")
		
		command, err := e.manager.SendCommand(tenant, site, deviceID, batchCmd.Operation, batchCmd.Args, timeoutSeconds)
		if err != nil {
			return results, fmt.Errorf("failed to execute batch command %d (%s): %w", i, batchCmd.Operation, err)
		}
		
		results = append(results, command)
		
		// Wait for command completion if required
		if batchCmd.WaitForCompletion {
			completedCmd, err := e.WaitForCommand(command.ID, 2, timeoutSeconds)
			if err != nil {
				return results, fmt.Errorf("batch command %d (%s) failed: %w", i, batchCmd.Operation, err)
			}
			
			if completedCmd.Status == "failed" {
				return results, fmt.Errorf("batch command %d (%s) failed: %s", i, batchCmd.Operation, completedCmd.Error)
			}
			
			// Update result with completed command
			results[len(results)-1] = completedCmd
		}
		
		// Apply delay if specified
		if batchCmd.DelaySeconds > 0 {
			time.Sleep(time.Duration(batchCmd.DelaySeconds) * time.Second)
		}
	}
	
	return results, nil
}

// BatchCommand represents a command in a batch execution
type BatchCommand struct {
	Operation         string                 `json:"operation"`
	Args              map[string]interface{} `json:"args"`
	WaitForCompletion bool                   `json:"wait_for_completion"`
	DelaySeconds      int                    `json:"delay_seconds"`
}

// GetCommandHistory returns command history for a device
func (e *Executor) GetCommandHistory(deviceID string, limit int, offset int) ([]*types.DeviceCommand, int, error) {
	return e.manager.ListCommands(deviceID, "", limit, offset)
}

// GetPendingCommands returns pending commands for a device
func (e *Executor) GetPendingCommands(deviceID string) ([]*types.DeviceCommand, error) {
	commands, _, err := e.manager.ListCommands(deviceID, "pending", 100, 0)
	if err != nil {
		return nil, err
	}
	
	// Also get sent and ack commands
	sentCommands, _, err := e.manager.ListCommands(deviceID, "sent", 100, 0)
	if err != nil {
		return commands, nil // Return what we have
	}
	commands = append(commands, sentCommands...)
	
	ackCommands, _, err := e.manager.ListCommands(deviceID, "ack", 100, 0)
	if err != nil {
		return commands, nil // Return what we have
	}
	commands = append(commands, ackCommands...)
	
	return commands, nil
}

// CancelCommand attempts to cancel a pending command (implementation depends on device support)
func (e *Executor) CancelCommand(tenant, site, deviceID, commandID string, timeoutSeconds int) (*types.DeviceCommand, error) {
	args := map[string]interface{}{
		"command_id": commandID,
	}
	
	return e.manager.SendCommand(tenant, site, deviceID, "cancel_command", args, timeoutSeconds)
}
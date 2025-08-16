package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"

	"rtk_controller/internal/config"
	"rtk_controller/internal/device"
	"rtk_controller/internal/command"
	"rtk_controller/internal/diagnosis"
	"rtk_controller/internal/mqtt"
	"rtk_controller/internal/storage"
)

// InteractiveCLI provides an interactive command line interface
type InteractiveCLI struct {
	config           *config.Config
	mqttClient       *mqtt.Client
	storage          storage.Storage
	deviceManager    *device.Manager
	commandManager   *command.Manager
	diagnosisManager *diagnosis.Manager
	
	// Command history and state
	history []string
	running bool
}

// NewInteractiveCLI creates a new interactive CLI
func NewInteractiveCLI(
	config *config.Config,
	mqttClient *mqtt.Client,
	storage storage.Storage,
	deviceManager *device.Manager,
	commandManager *command.Manager,
	diagnosisManager *diagnosis.Manager,
) *InteractiveCLI {
	return &InteractiveCLI{
		config:           config,
		mqttClient:       mqttClient,
		storage:          storage,
		deviceManager:    deviceManager,
		commandManager:   commandManager,
		diagnosisManager: diagnosisManager,
		history:          make([]string, 0),
		running:          false,
	}
}

// Start starts the interactive CLI
func (cli *InteractiveCLI) Start() {
	cli.running = true
	
	fmt.Println("RTK Controller Interactive CLI")
	fmt.Println("==============================")
	fmt.Printf("Version: 1.0.0\n")
	fmt.Printf("Type 'help' for available commands, 'exit' to quit\n\n")
	
	// Setup readline with auto-completion
	completer := cli.createCompleter()
	l, err := readline.NewEx(&readline.Config{
		Prompt:          "rtk> ",
		HistoryFile:     "/tmp/.rtk_cli_history",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		return
	}
	defer l.Close()
	
	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				fmt.Println("\nBye!")
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			fmt.Println("\nBye!")
			break
		}
		
		cli.executor(strings.TrimSpace(line))
	}
}

// Stop stops the interactive CLI
func (cli *InteractiveCLI) Stop() {
	cli.running = false
}

// executor handles command execution
func (cli *InteractiveCLI) executor(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}
	
	// Add to history
	cli.history = append(cli.history, input)
	
	// Parse command
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}
	
	command := parts[0]
	args := parts[1:]
	
	switch command {
	case "help", "h":
		cli.showHelp(args)
	case "exit", "quit", "q":
		fmt.Println("Bye!")
		os.Exit(0)
	case "clear", "cls":
		fmt.Print("\033[H\033[2J")
	case "version", "ver":
		cli.showVersion()
	case "status":
		cli.showStatus(args)
	case "device", "dev":
		cli.handleDeviceCommand(args)
	case "command", "cmd":
		cli.handleCommandCommand(args)
	case "event", "evt":
		cli.handleEventCommand(args)
	case "diagnosis", "diag":
		cli.handleDiagnosisCommand(args)
	case "config", "cfg":
		cli.handleConfigCommand(args)
	case "log":
		cli.handleLogCommand(args)
	case "test":
		cli.handleTestCommand(args)
	case "system", "sys":
		cli.handleSystemCommand(args)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Type 'help' for available commands")
	}
}

// createCompleter creates a readline completer
func (cli *InteractiveCLI) createCompleter() readline.AutoCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("help"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
		readline.PcItem("clear"),
		readline.PcItem("version"),
		readline.PcItem("status"),
		readline.PcItem("device",
			readline.PcItem("list"),
			readline.PcItem("show"),
			readline.PcItem("status"),
			readline.PcItem("history"),
			readline.PcItem("stats"),
		),
		readline.PcItem("command",
			readline.PcItem("send"),
			readline.PcItem("list"),
			readline.PcItem("show"),
			readline.PcItem("cancel"),
			readline.PcItem("stats"),
		),
		readline.PcItem("event",
			readline.PcItem("list"),
			readline.PcItem("show"),
			readline.PcItem("stats"),
			readline.PcItem("watch"),
		),
		readline.PcItem("diagnosis",
			readline.PcItem("run"),
			readline.PcItem("list"),
			readline.PcItem("show"),
			readline.PcItem("analyzers"),
		),
		readline.PcItem("config",
			readline.PcItem("show"),
			readline.PcItem("reload"),
			readline.PcItem("set"),
		),
		readline.PcItem("log",
			readline.PcItem("show"),
			readline.PcItem("tail"),
			readline.PcItem("search"),
			readline.PcItem("download"),
		),
		readline.PcItem("test",
			readline.PcItem("mqtt"),
			readline.PcItem("storage"),
			readline.PcItem("ping"),
		),
		readline.PcItem("system",
			readline.PcItem("info"),
			readline.PcItem("health"),
			readline.PcItem("stats"),
		),
	)
}

// Command handlers

func (cli *InteractiveCLI) showHelp(args []string) {
	if len(args) == 0 {
		fmt.Println("Available commands:")
		fmt.Println("  help [command]     - Show help information")
		fmt.Println("  exit/quit          - Exit the CLI")
		fmt.Println("  clear              - Clear the screen")
		fmt.Println("  version            - Show version information")
		fmt.Println("  status             - Show system status")
		fmt.Println("  device <subcommand> - Device management")
		fmt.Println("  command <subcommand> - Command management")
		fmt.Println("  event <subcommand> - Event monitoring")
		fmt.Println("  diagnosis <subcommand> - Diagnosis management")
		fmt.Println("  config <subcommand> - Configuration management")
		fmt.Println("  log <subcommand>   - Log management")
		fmt.Println("  test <subcommand>  - Test commands")
		fmt.Println("  system <subcommand> - System information")
		fmt.Println()
		fmt.Println("Use 'help <command>' for detailed help on a specific command")
		return
	}
	
	switch args[0] {
	case "device":
		fmt.Println("Device management commands:")
		fmt.Println("  device list [--status=online|offline] [--limit=N] - List devices")
		fmt.Println("  device show <device_id> - Show device details")
		fmt.Println("  device status <device_id> - Show device status")
		fmt.Println("  device history <device_id> - Show device history")
		fmt.Println("  device stats - Show device statistics")
	case "command":
		fmt.Println("Command management commands:")
		fmt.Println("  command send <device_id> <operation> [args...] - Send command")
		fmt.Println("  command list [--device=<id>] [--status=<status>] - List commands")
		fmt.Println("  command show <command_id> - Show command details")
		fmt.Println("  command cancel <command_id> - Cancel pending command")
		fmt.Println("  command stats - Show command statistics")
	case "event":
		fmt.Println("Event monitoring commands:")
		fmt.Println("  event list [--device=<id>] [--type=<type>] - List events")
		fmt.Println("  event show <event_id> - Show event details")
		fmt.Println("  event stats - Show event statistics")
		fmt.Println("  event watch - Watch real-time events")
	default:
		fmt.Printf("No help available for command: %s\n", args[0])
	}
}

func (cli *InteractiveCLI) showVersion() {
	fmt.Println("RTK Controller CLI")
	fmt.Println("Version: 1.0.0")
	fmt.Printf("Build Date: %s\n", time.Now().Format("2006-01-02"))
	fmt.Println("Go Version: " + "1.19+")
}

func (cli *InteractiveCLI) showStatus(args []string) {
	fmt.Println("RTK Controller System Status")
	fmt.Println("============================")
	
	// MQTT Status
	mqttStatus := "disconnected"
	if cli.mqttClient != nil && cli.mqttClient.IsConnected() {
		mqttStatus = "connected"
	}
	fmt.Printf("MQTT:          %s\n", mqttStatus)
	if cli.config != nil {
		fmt.Printf("MQTT Broker:   %s:%d\n", cli.config.MQTT.Broker, cli.config.MQTT.Port)
	}
	
	// Device Manager Status
	if cli.deviceManager != nil {
		deviceStats := cli.deviceManager.GetStats()
		fmt.Printf("Devices:       %d total, %d online, %d offline\n", 
			deviceStats.TotalDevices, deviceStats.OnlineDevices, deviceStats.OfflineDevices)
	}
	
	// Command Manager Status
	if cli.commandManager != nil {
		cmdStats := cli.commandManager.GetStats()
		fmt.Printf("Commands:      %d total, %d pending, %d completed\n",
			cmdStats.TotalCommands, cmdStats.PendingCommands, cmdStats.CompletedCommands)
	}
	
	fmt.Println()
	fmt.Printf("Uptime:        %s\n", "Runtime information would go here")
	fmt.Printf("Last Updated:  %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

func (cli *InteractiveCLI) handleDeviceCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Device subcommands: list, show, status, history, stats")
		return
	}
	
	switch args[0] {
	case "list":
		cli.listDevices(args[1:])
	case "show":
		cli.showDevice(args[1:])
	case "status":
		cli.showDeviceStatus(args[1:])
	case "history":
		cli.showDeviceHistory(args[1:])
	case "stats":
		cli.showDeviceStats()
	default:
		fmt.Printf("Unknown device subcommand: %s\n", args[0])
	}
}

func (cli *InteractiveCLI) listDevices(args []string) {
	fmt.Println("Device List")
	fmt.Println("-----------")
	
	if cli.deviceManager == nil {
		fmt.Println("Device manager not available")
		return
	}
	
	devices, total, err := cli.deviceManager.ListDevices(nil, 50, 0)
	if err != nil {
		fmt.Printf("Error listing devices: %v\n", err)
		return
	}
	
	fmt.Printf("Total devices: %d\n\n", total)
	
	if len(devices) == 0 {
		fmt.Println("No devices found")
		return
	}
	
	fmt.Printf("%-20s %-15s %-10s %-10s %-20s\n", "DEVICE ID", "TYPE", "HEALTH", "STATUS", "LAST SEEN")
	fmt.Println(strings.Repeat("-", 80))
	
	for _, device := range devices {
		status := "offline"
		if device.Online {
			status = "online"
		}
		
		lastSeen := time.UnixMilli(device.LastSeen).Format("2006-01-02 15:04:05")
		fmt.Printf("%-20s %-15s %-10s %-10s %-20s\n", 
			device.ID, device.DeviceType, device.Health, status, lastSeen)
	}
}

func (cli *InteractiveCLI) showDevice(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: device show <device_id>")
		return
	}
	
	deviceID := args[0]
	device, err := cli.deviceManager.GetDevice("", "", deviceID)
	if err != nil {
		fmt.Printf("Error getting device: %v\n", err)
		return
	}
	
	fmt.Printf("Device Details: %s\n", deviceID)
	fmt.Println(strings.Repeat("=", 30))
	fmt.Printf("ID:           %s\n", device.ID)
	fmt.Printf("Tenant:       %s\n", device.Tenant)
	fmt.Printf("Site:         %s\n", device.Site)
	fmt.Printf("Type:         %s\n", device.DeviceType)
	fmt.Printf("Health:       %s\n", device.Health)
	fmt.Printf("Online:       %t\n", device.Online)
	fmt.Printf("Version:      %s\n", device.Version)
	fmt.Printf("Uptime:       %d seconds\n", device.UptimeS)
	fmt.Printf("Last Seen:    %s\n", time.UnixMilli(device.LastSeen).Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated At:   %s\n", device.UpdatedAt.Format("2006-01-02 15:04:05"))
	
	if len(device.Components) > 0 {
		fmt.Println("\nComponents:")
		for key, value := range device.Components {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
	
	if len(device.Attributes) > 0 {
		fmt.Println("\nAttributes:")
		for key, value := range device.Attributes {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}

func (cli *InteractiveCLI) showDeviceStatus(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: device status <device_id>")
		return
	}
	
	deviceID := args[0]
	device, err := cli.deviceManager.GetDevice("", "", deviceID)
	if err != nil {
		fmt.Printf("Error getting device: %v\n", err)
		return
	}
	
	fmt.Printf("Device Status: %s\n", deviceID)
	fmt.Println(strings.Repeat("=", 25))
	
	status := "🔴 OFFLINE"
	if device.Online {
		status = "🟢 ONLINE"
	}
	
	fmt.Printf("Status:   %s\n", status)
	fmt.Printf("Health:   %s\n", device.Health)
	fmt.Printf("Uptime:   %d seconds\n", device.UptimeS)
	fmt.Printf("Version:  %s\n", device.Version)
	fmt.Printf("Last Seen: %s\n", time.UnixMilli(device.LastSeen).Format("2006-01-02 15:04:05"))
}

func (cli *InteractiveCLI) showDeviceHistory(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: device history <device_id>")
		return
	}
	
	deviceID := args[0]
	fmt.Printf("Device History: %s\n", deviceID)
	fmt.Println(strings.Repeat("=", 25))
	fmt.Println("History feature is not yet implemented")
}

func (cli *InteractiveCLI) showDeviceStats() {
	if cli.deviceManager == nil {
		fmt.Println("Device manager not available")
		return
	}
	
	stats := cli.deviceManager.GetStats()
	
	fmt.Println("Device Statistics")
	fmt.Println("=================")
	fmt.Printf("Total Devices:    %d\n", stats.TotalDevices)
	fmt.Printf("Online Devices:   %d\n", stats.OnlineDevices)
	fmt.Printf("Offline Devices:  %d\n", stats.OfflineDevices)
	fmt.Printf("Last Updated:     %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05"))
	
	if len(stats.HealthStats) > 0 {
		fmt.Println("\nHealth Distribution:")
		for health, count := range stats.HealthStats {
			fmt.Printf("  %s: %d\n", health, count)
		}
	}
	
	if len(stats.DeviceTypeStats) > 0 {
		fmt.Println("\nDevice Type Distribution:")
		for deviceType, count := range stats.DeviceTypeStats {
			fmt.Printf("  %s: %d\n", deviceType, count)
		}
	}
}

func (cli *InteractiveCLI) handleCommandCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Command subcommands: send, list, show, cancel, stats")
		return
	}
	
	switch args[0] {
	case "send":
		cli.sendCommand(args[1:])
	case "list":
		cli.listCommands(args[1:])
	case "show":
		cli.showCommand(args[1:])
	case "cancel":
		cli.cancelCommand(args[1:])
	case "stats":
		cli.showCommandStats()
	default:
		fmt.Printf("Unknown command subcommand: %s\n", args[0])
	}
}

func (cli *InteractiveCLI) sendCommand(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: command send <device_id> <operation> [timeout_seconds]")
		fmt.Println("Example: command send device1 reboot 30")
		return
	}
	
	deviceID := args[0]
	operation := args[1]
	timeout := 30 // default timeout
	
	if len(args) > 2 {
		fmt.Sscanf(args[2], "%d", &timeout)
	}
	
	fmt.Printf("Sending command '%s' to device '%s' (timeout: %ds)...\n", operation, deviceID, timeout)
	
	cmd, err := cli.commandManager.SendCommand("", "", deviceID, operation, map[string]interface{}{}, timeout)
	if err != nil {
		fmt.Printf("Error sending command: %v\n", err)
		return
	}
	
	fmt.Printf("Command sent successfully!\n")
	fmt.Printf("Command ID: %s\n", cmd.ID)
	fmt.Printf("Status: %s\n", cmd.Status)
}

func (cli *InteractiveCLI) listCommands(args []string) {
	fmt.Println("Command List")
	fmt.Println("------------")
	
	commands, total, err := cli.commandManager.ListCommands("", "", 20, 0)
	if err != nil {
		fmt.Printf("Error listing commands: %v\n", err)
		return
	}
	
	fmt.Printf("Total commands: %d\n\n", total)
	
	if len(commands) == 0 {
		fmt.Println("No commands found")
		return
	}
	
	fmt.Printf("%-25s %-15s %-12s %-10s %-20s\n", "COMMAND ID", "DEVICE ID", "OPERATION", "STATUS", "CREATED AT")
	fmt.Println(strings.Repeat("-", 85))
	
	for _, cmd := range commands {
		createdAt := cmd.CreatedAt.Format("2006-01-02 15:04:05")
		fmt.Printf("%-25s %-15s %-12s %-10s %-20s\n", 
			cmd.ID, cmd.DeviceID, cmd.Operation, cmd.Status, createdAt)
	}
}

func (cli *InteractiveCLI) showCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: command show <command_id>")
		return
	}
	
	commandID := args[0]
	cmd, err := cli.commandManager.GetCommand(commandID)
	if err != nil {
		fmt.Printf("Error getting command: %v\n", err)
		return
	}
	
	fmt.Printf("Command Details: %s\n", commandID)
	fmt.Println(strings.Repeat("=", 30))
	fmt.Printf("ID:           %s\n", cmd.ID)
	fmt.Printf("Device ID:    %s\n", cmd.DeviceID)
	fmt.Printf("Operation:    %s\n", cmd.Operation)
	fmt.Printf("Status:       %s\n", cmd.Status)
	fmt.Printf("Timeout:      %d ms\n", cmd.TimeoutMS)
	fmt.Printf("Created At:   %s\n", cmd.CreatedAt.Format("2006-01-02 15:04:05"))
	
	if cmd.SentAt != nil {
		fmt.Printf("Sent At:      %s\n", cmd.SentAt.Format("2006-01-02 15:04:05"))
	}
	
	if cmd.CompletedAt != nil {
		fmt.Printf("Completed At: %s\n", cmd.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	
	if cmd.Error != "" {
		fmt.Printf("Error:        %s\n", cmd.Error)
	}
	
	if len(cmd.Args) > 0 {
		fmt.Println("\nArguments:")
		for key, value := range cmd.Args {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
	
	if len(cmd.Result) > 0 {
		fmt.Println("\nResult:")
		for key, value := range cmd.Result {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}

func (cli *InteractiveCLI) cancelCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: command cancel <command_id>")
		return
	}
	
	commandID := args[0]
	fmt.Printf("Cancelling command: %s\n", commandID)
	fmt.Println("Command cancellation is not yet implemented")
}

func (cli *InteractiveCLI) showCommandStats() {
	if cli.commandManager == nil {
		fmt.Println("Command manager not available")
		return
	}
	
	stats := cli.commandManager.GetStats()
	
	fmt.Println("Command Statistics")
	fmt.Println("==================")
	fmt.Printf("Total Commands:     %d\n", stats.TotalCommands)
	fmt.Printf("Pending Commands:   %d\n", stats.PendingCommands)
	fmt.Printf("Completed Commands: %d\n", stats.CompletedCommands)
	fmt.Printf("Failed Commands:    %d\n", stats.FailedCommands)
	fmt.Printf("Timeout Commands:   %d\n", stats.TimeoutCommands)
	fmt.Printf("Last Updated:       %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05"))
	
	if len(stats.StatusStats) > 0 {
		fmt.Println("\nStatus Distribution:")
		for status, count := range stats.StatusStats {
			fmt.Printf("  %s: %d\n", status, count)
		}
	}
}

func (cli *InteractiveCLI) handleEventCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Event subcommands: list, show, stats, watch")
		return
	}
	
	switch args[0] {
	case "list":
		fmt.Println("Event listing is not yet implemented")
	case "show":
		fmt.Println("Event details are not yet implemented")
	case "stats":
		fmt.Println("Event statistics are not yet implemented")
	case "watch":
		fmt.Println("Real-time event watching is not yet implemented")
	default:
		fmt.Printf("Unknown event subcommand: %s\n", args[0])
	}
}

func (cli *InteractiveCLI) handleDiagnosisCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Diagnosis subcommands: run, list, show, analyzers")
		return
	}
	
	switch args[0] {
	case "run":
		fmt.Println("Running diagnosis is not yet implemented")
	case "list":
		fmt.Println("Diagnosis listing is not yet implemented")
	case "show":
		fmt.Println("Diagnosis details are not yet implemented")
	case "analyzers":
		fmt.Println("Available analyzers listing is not yet implemented")
	default:
		fmt.Printf("Unknown diagnosis subcommand: %s\n", args[0])
	}
}

func (cli *InteractiveCLI) handleConfigCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Config subcommands: show, reload, set")
		return
	}
	
	switch args[0] {
	case "show":
		fmt.Println("Configuration display is not yet implemented")
	case "reload":
		fmt.Println("Configuration reload is not yet implemented")
	case "set":
		fmt.Println("Configuration setting is not yet implemented")
	default:
		fmt.Printf("Unknown config subcommand: %s\n", args[0])
	}
}

func (cli *InteractiveCLI) handleLogCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Log subcommands: show, tail, search, download")
		return
	}
	
	switch args[0] {
	case "show":
		fmt.Println("Log display is not yet implemented")
	case "tail":
		fmt.Println("Log tailing is not yet implemented")
	case "search":
		fmt.Println("Log search is not yet implemented")
	case "download":
		fmt.Println("Log download is not yet implemented")
	default:
		fmt.Printf("Unknown log subcommand: %s\n", args[0])
	}
}

func (cli *InteractiveCLI) handleTestCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Test subcommands: mqtt, storage, ping")
		return
	}
	
	switch args[0] {
	case "mqtt":
		cli.testMQTT()
	case "storage":
		cli.testStorage()
	case "ping":
		fmt.Println("Device ping is not yet implemented")
	default:
		fmt.Printf("Unknown test subcommand: %s\n", args[0])
	}
}

func (cli *InteractiveCLI) testMQTT() {
	fmt.Println("Testing MQTT connection...")
	
	if cli.mqttClient == nil {
		fmt.Println("❌ MQTT client not initialized")
		return
	}
	
	if cli.mqttClient.IsConnected() {
		fmt.Println("✅ MQTT connection is active")
		fmt.Printf("   Broker: %s:%d\n", cli.config.MQTT.Broker, cli.config.MQTT.Port)
	} else {
		fmt.Println("❌ MQTT connection is down")
	}
}

func (cli *InteractiveCLI) testStorage() {
	fmt.Println("Testing storage connection...")
	
	if cli.storage == nil {
		fmt.Println("❌ Storage not initialized")
		return
	}
	
	// Try a simple operation
	err := cli.storage.View(func(tx storage.Transaction) error {
		_, _ = tx.Get("test_key")
		return nil // Ignore key not found error
	})
	
	if err != nil && !strings.Contains(err.Error(), "not found") {
		fmt.Printf("❌ Storage test failed: %v\n", err)
	} else {
		fmt.Println("✅ Storage connection is working")
	}
}

func (cli *InteractiveCLI) handleSystemCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("System subcommands: info, health, stats")
		return
	}
	
	switch args[0] {
	case "info":
		cli.showSystemInfo()
	case "health":
		cli.showSystemHealth()
	case "stats":
		cli.showSystemStats()
	default:
		fmt.Printf("Unknown system subcommand: %s\n", args[0])
	}
}

func (cli *InteractiveCLI) showSystemInfo() {
	fmt.Println("System Information")
	fmt.Println("==================")
	fmt.Printf("Version:       1.0.0\n")
	fmt.Printf("Build Date:    %s\n", time.Now().Format("2006-01-02"))
	fmt.Printf("Configuration: %s\n", "controller.yaml")
	fmt.Printf("Storage Path:  %s\n", cli.config.Storage.Path)
}

func (cli *InteractiveCLI) showSystemHealth() {
	fmt.Println("System Health Check")
	fmt.Println("===================")
	
	overall := "✅ Healthy"
	issues := []string{}
	
	// Check MQTT
	if cli.mqttClient == nil || !cli.mqttClient.IsConnected() {
		overall = "❌ Unhealthy"
		issues = append(issues, "MQTT connection is down")
	} else {
		fmt.Println("✅ MQTT: Connected")
	}
	
	// Check storage
	if cli.storage == nil {
		overall = "❌ Unhealthy"
		issues = append(issues, "Storage is not available")
	} else {
		fmt.Println("✅ Storage: Available")
	}
	
	// Check managers
	if cli.deviceManager == nil {
		overall = "❌ Unhealthy"
		issues = append(issues, "Device manager is not available")
	} else {
		fmt.Println("✅ Device Manager: Running")
	}
	
	if cli.commandManager == nil {
		overall = "❌ Unhealthy"
		issues = append(issues, "Command manager is not available")
	} else {
		fmt.Println("✅ Command Manager: Running")
	}
	
	fmt.Printf("\nOverall Status: %s\n", overall)
	
	if len(issues) > 0 {
		fmt.Println("\nIssues:")
		for _, issue := range issues {
			fmt.Printf("  ⚠ %s\n", issue)
		}
	}
}

func (cli *InteractiveCLI) showSystemStats() {
	fmt.Println("System Statistics")
	fmt.Println("=================")
	
	// Device stats
	if cli.deviceManager != nil {
		deviceStats := cli.deviceManager.GetStats()
		fmt.Printf("Devices:       %d total, %d online, %d offline\n", 
			deviceStats.TotalDevices, deviceStats.OnlineDevices, deviceStats.OfflineDevices)
	}
	
	// Command stats
	if cli.commandManager != nil {
		cmdStats := cli.commandManager.GetStats()
		fmt.Printf("Commands:      %d total, %d pending, %d completed\n",
			cmdStats.TotalCommands, cmdStats.PendingCommands, cmdStats.CompletedCommands)
	}
	
	fmt.Printf("Last Updated:  %s\n", time.Now().Format("2006-01-02 15:04:05"))
}
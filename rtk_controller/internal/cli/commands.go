package cli

import (
	"time"

	"github.com/spf13/cobra"

	"rtk_controller/internal/config"
	"rtk_controller/internal/storage"
	"rtk_controller/internal/device"
	"rtk_controller/internal/command"
	"rtk_controller/internal/mqtt"
)

// CLI represents the command line interface
type CLI struct {
	config         *config.Config
	storage        storage.Storage
	deviceManager  *device.Manager
	commandManager *command.Manager
	mqttClient     *mqtt.Client
	rootCmd        *cobra.Command
}

// NewCLI creates a new CLI instance
func NewCLI(cfg *config.Config, storage storage.Storage, deviceMgr *device.Manager, cmdMgr *command.Manager, mqttClient *mqtt.Client) *CLI {
	cli := &CLI{
		config:         cfg,
		storage:        storage,
		deviceManager:  deviceMgr,
		commandManager: cmdMgr,
		mqttClient:     mqttClient,
	}

	cli.setupCommands()
	return cli
}

// Execute executes the CLI
func (c *CLI) Execute() error {
	return c.rootCmd.Execute()
}

// setupCommands sets up all CLI commands
func (c *CLI) setupCommands() {
	c.rootCmd = &cobra.Command{
		Use:   "rtk-cli",
		Short: "RTK Controller CLI",
		Long:  "Command line interface for RTK MQTT Controller",
	}

	// Add command groups
	c.addDeviceCommands()
	c.addCommandCommands()
	c.addEventCommands()
	c.addDiagnoseCommands()
	c.addSystemCommands()
	c.addTestCommands()
	c.addLogCommands()
	c.addConfigCommands()
}

// addDeviceCommands adds device management commands
func (c *CLI) addDeviceCommands() {
	deviceCmd := &cobra.Command{
		Use:   "device",
		Short: "Device management commands",
	}

	// device list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all devices",
		RunE:  c.deviceList,
	}
	listCmd.Flags().String("tenant", "", "Filter by tenant")
	listCmd.Flags().String("site", "", "Filter by site")
	listCmd.Flags().String("status", "", "Filter by status (online/offline)")
	listCmd.Flags().String("format", "table", "Output format (table/json)")

	// device show
	showCmd := &cobra.Command{
		Use:   "show <device_id>",
		Short: "Show device details",
		Args:  cobra.ExactArgs(1),
		RunE:  c.deviceShow,
	}
	showCmd.Flags().String("format", "table", "Output format (table/json)")

	// device status
	statusCmd := &cobra.Command{
		Use:   "status <device_id>",
		Short: "Get device current status",
		Args:  cobra.ExactArgs(1),
		RunE:  c.deviceStatus,
	}
	statusCmd.Flags().Bool("refresh", false, "Force refresh device status")

	// device history
	historyCmd := &cobra.Command{
		Use:   "history <device_id>",
		Short: "Show device history",
		Args:  cobra.ExactArgs(1),
		RunE:  c.deviceHistory,
	}
	historyCmd.Flags().Int("limit", 100, "Limit number of records")
	historyCmd.Flags().String("since", "", "Show records since timestamp")

	// device search
	searchCmd := &cobra.Command{
		Use:   "search <pattern>",
		Short: "Search devices by pattern",
		Args:  cobra.ExactArgs(1),
		RunE:  c.deviceSearch,
	}

	deviceCmd.AddCommand(listCmd, showCmd, statusCmd, historyCmd, searchCmd)
	c.rootCmd.AddCommand(deviceCmd)
}

// addCommandCommands adds command management commands
func (c *CLI) addCommandCommands() {
	// send command
	sendCmd := &cobra.Command{
		Use:   "send <device_id> <operation> [args...]",
		Short: "Send command to device",
		Args:  cobra.MinimumNArgs(2),
		RunE:  c.sendCommand,
	}
	sendCmd.Flags().Duration("timeout", 30*time.Second, "Command timeout")
	sendCmd.Flags().Bool("async", false, "Send command asynchronously")

	// cmd group
	cmdCmd := &cobra.Command{
		Use:   "cmd",
		Short: "Command management",
	}

	cmdStatusCmd := &cobra.Command{
		Use:   "status <command_id>",
		Short: "Get command status",
		Args:  cobra.ExactArgs(1),
		RunE:  c.commandStatus,
	}

	cmdListCmd := &cobra.Command{
		Use:   "list",
		Short: "List commands",
		RunE:  c.commandList,
	}
	cmdListCmd.Flags().String("device", "", "Filter by device ID")
	cmdListCmd.Flags().String("status", "", "Filter by status")
	cmdListCmd.Flags().Int("limit", 50, "Limit number of records")

	cmdHistoryCmd := &cobra.Command{
		Use:   "history",
		Short: "Show command history",
		RunE:  c.commandHistory,
	}
	cmdHistoryCmd.Flags().Int("limit", 50, "Limit number of records")

	cmdCancelCmd := &cobra.Command{
		Use:   "cancel <command_id>",
		Short: "Cancel command",
		Args:  cobra.ExactArgs(1),
		RunE:  c.commandCancel,
	}

	cmdCmd.AddCommand(cmdStatusCmd, cmdListCmd, cmdHistoryCmd, cmdCancelCmd)
	c.rootCmd.AddCommand(sendCmd, cmdCmd)
}

// addEventCommands adds event management commands
func (c *CLI) addEventCommands() {
	eventsCmd := &cobra.Command{
		Use:   "events",
		Short: "Event management commands",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List events",
		RunE:  c.eventsList,
	}
	listCmd.Flags().String("type", "", "Filter by event type")
	listCmd.Flags().String("severity", "", "Filter by severity")
	listCmd.Flags().Int("limit", 20, "Limit number of events")
	listCmd.Flags().String("device", "", "Filter by device ID")

	showCmd := &cobra.Command{
		Use:   "show <event_id>",
		Short: "Show event details",
		Args:  cobra.ExactArgs(1),
		RunE:  c.eventsShow,
	}

	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show event statistics",
		RunE:  c.eventsStats,
	}
	statsCmd.Flags().String("period", "24h", "Time period for statistics")
	statsCmd.Flags().String("group-by", "device_id", "Group statistics by field")

	followCmd := &cobra.Command{
		Use:   "follow",
		Short: "Follow events in real-time",
		RunE:  c.eventsFollow,
	}
	followCmd.Flags().String("filter", "", "Event filter pattern")

	eventsCmd.AddCommand(listCmd, showCmd, statsCmd, followCmd)
	c.rootCmd.AddCommand(eventsCmd)
}

// addDiagnoseCommands adds diagnosis commands
func (c *CLI) addDiagnoseCommands() {
	diagnoseCmd := &cobra.Command{
		Use:   "diagnose <device_id>",
		Short: "Run diagnosis on device",
		Args:  cobra.ExactArgs(1),
		RunE:  c.diagnoseDevice,
	}
	diagnoseCmd.Flags().String("type", "", "Diagnosis type (wifi/network/system)")
	diagnoseCmd.Flags().Bool("include-history", false, "Include historical data")
	diagnoseCmd.Flags().String("detail-level", "normal", "Detail level (basic/normal/full)")

	analyzerCmd := &cobra.Command{
		Use:   "analyzer",
		Short: "Analyzer management",
	}

	analyzerListCmd := &cobra.Command{
		Use:   "list",
		Short: "List analyzers",
		RunE:  c.analyzerList,
	}

	analyzerInfoCmd := &cobra.Command{
		Use:   "info <analyzer_name>",
		Short: "Show analyzer information",
		Args:  cobra.ExactArgs(1),
		RunE:  c.analyzerInfo,
	}

	analyzerTestCmd := &cobra.Command{
		Use:   "test <analyzer_name>",
		Short: "Test analyzer",
		Args:  cobra.ExactArgs(1),
		RunE:  c.analyzerTest,
	}

	analyzerEnableCmd := &cobra.Command{
		Use:   "enable <analyzer_name>",
		Short: "Enable analyzer",
		Args:  cobra.ExactArgs(1),
		RunE:  c.analyzerEnable,
	}

	analyzerDisableCmd := &cobra.Command{
		Use:   "disable <analyzer_name>",
		Short: "Disable analyzer",
		Args:  cobra.ExactArgs(1),
		RunE:  c.analyzerDisable,
	}

	analyzerCmd.AddCommand(analyzerListCmd, analyzerInfoCmd, analyzerTestCmd, analyzerEnableCmd, analyzerDisableCmd)
	c.rootCmd.AddCommand(diagnoseCmd, analyzerCmd)
}

// addSystemCommands adds system management commands
func (c *CLI) addSystemCommands() {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show system status",
		RunE:  c.systemStatus,
	}
	statusCmd.Flags().String("component", "", "Show specific component status")

	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show system statistics",
		RunE:  c.systemStats,
	}
	statsCmd.Flags().String("period", "1h", "Time period for statistics")

	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Show system health",
		RunE:  c.systemHealth,
	}

	c.rootCmd.AddCommand(statusCmd, statsCmd, healthCmd)
}

// addTestCommands adds test commands
func (c *CLI) addTestCommands() {
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test system components",
	}

	mqttTestCmd := &cobra.Command{
		Use:   "mqtt",
		Short: "Test MQTT connectivity",
		RunE:  c.testMQTT,
	}
	mqttTestCmd.Flags().String("topic", "", "Test topic")

	deviceTestCmd := &cobra.Command{
		Use:   "device <device_id>",
		Short: "Test device connectivity",
		Args:  cobra.ExactArgs(1),
		RunE:  c.testDevice,
	}
	deviceTestCmd.Flags().Duration("timeout", 10*time.Second, "Test timeout")

	analyzerTestCmd := &cobra.Command{
		Use:   "analyzer <analyzer_name>",
		Short: "Test analyzer",
		Args:  cobra.ExactArgs(1),
		RunE:  c.testAnalyzer,
	}

	connectivityTestCmd := &cobra.Command{
		Use:   "connectivity",
		Short: "Test system connectivity",
		RunE:  c.testConnectivity,
	}

	testCmd.AddCommand(mqttTestCmd, deviceTestCmd, analyzerTestCmd, connectivityTestCmd)
	c.rootCmd.AddCommand(testCmd)
}

// addLogCommands adds log management commands
func (c *CLI) addLogCommands() {
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Log management commands",
	}

	tailCmd := &cobra.Command{
		Use:   "tail",
		Short: "Tail system logs",
		RunE:  c.logsTail,
	}
	tailCmd.Flags().String("level", "", "Filter by log level")
	tailCmd.Flags().String("component", "", "Filter by component")
	tailCmd.Flags().Int("lines", 100, "Number of lines to show")

	searchCmd := &cobra.Command{
		Use:   "search <pattern>",
		Short: "Search logs",
		Args:  cobra.ExactArgs(1),
		RunE:  c.logsSearch,
	}

	downloadCmd := &cobra.Command{
		Use:   "download <seconds>",
		Short: "Download recent MQTT message logs",
		Args:  cobra.ExactArgs(1),
		RunE:  c.logsDownload,
	}

	logsCmd.AddCommand(tailCmd, searchCmd, downloadCmd)
	c.rootCmd.AddCommand(logsCmd)
}

// addConfigCommands adds configuration management commands
func (c *CLI) addConfigCommands() {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show configuration",
		RunE:  c.configShow,
	}
	showCmd.Flags().String("section", "", "Show specific section")

	reloadCmd := &cobra.Command{
		Use:   "reload",
		Short: "Reload configuration",
		RunE:  c.configReload,
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE:  c.configSet,
	}

	configCmd.AddCommand(showCmd, reloadCmd, setCmd)
	c.rootCmd.AddCommand(configCmd)
}

// Command implementations will be in separate files for better organization
package cli

import (
	"fmt"
	"os"

	"rtk_controller/internal/config"
	"rtk_controller/internal/storage"
	"rtk_controller/internal/device"
	"rtk_controller/internal/command"
	"rtk_controller/internal/mqtt"

	log "github.com/sirupsen/logrus"
)

// Handler handles CLI command execution (legacy interface)
type Handler struct {
	config *config.Config
	cli    *CLI
}

// NewHandler creates a new CLI handler
func NewHandler(config *config.Config) *Handler {
	return &Handler{
		config: config,
	}
}

// NewHandlerWithServices creates a new CLI handler with full services
func NewHandlerWithServices(config *config.Config, storage storage.Storage, deviceMgr *device.Manager, cmdMgr *command.Manager, mqttClient *mqtt.Client) *Handler {
	cli := NewCLI(config, storage, deviceMgr, cmdMgr, mqttClient)
	return &Handler{
		config: config,
		cli:    cli,
	}
}

// Execute executes a CLI command (legacy interface)
func (h *Handler) Execute(command string, args []string) error {
	log.WithFields(log.Fields{
		"command": command,
		"args":    args,
	}).Info("Executing CLI command")

	// If we have full CLI available, use it
	if h.cli != nil {
		// Set up os.Args for cobra
		originalArgs := os.Args
		defer func() { os.Args = originalArgs }()
		
		os.Args = append([]string{"rtk-cli"}, command)
		os.Args = append(os.Args, args...)
		
		return h.cli.Execute()
	}

	// Fallback to simple command handling
	switch command {
	case "version":
		fmt.Println("RTK Controller CLI v1.0.0")
		return nil
	case "status":
		fmt.Println("Controller status: limited functionality in CLI mode")
		fmt.Println("Start controller normally for full CLI features")
		return nil
	case "help":
		h.printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command: %s (use 'help' for available commands)", command)
	}
}

// ExecuteInteractive starts interactive CLI mode
func (h *Handler) ExecuteInteractive() error {
	if h.cli == nil {
		return fmt.Errorf("full CLI services not available")
	}
	
	return h.cli.Execute()
}

func (h *Handler) printHelp() {
	fmt.Println("RTK Controller CLI")
	fmt.Println("==================")
	fmt.Println()
	fmt.Println("Available commands in limited mode:")
	fmt.Println("  version  - Show version information")
	fmt.Println("  status   - Show basic status")
	fmt.Println("  help     - Show this help")
	fmt.Println()
	fmt.Println("For full CLI functionality, start the controller normally:")
	fmt.Println("  ./rtk-controller --config configs/controller.yaml")
	fmt.Println()
	fmt.Println("Then use the Web Console or connect via CLI tools.")
}
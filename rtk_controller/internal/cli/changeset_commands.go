package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rtk_controller/internal/changeset"
	"rtk_controller/pkg/types"
)

// handleChangesetCommand handles changeset management commands
func (cli *InteractiveCLI) handleChangesetCommand(args []string, changesetManager *changeset.SimpleManager) {
	if changesetManager == nil {
		fmt.Println("Changeset manager not available")
		return
	}

	if len(args) == 0 {
		cli.showChangesetHelp()
		return
	}

	switch args[0] {
	case "create":
		cli.createChangeset(args[1:], changesetManager)
	case "list":
		cli.listChangesets(changesetManager)
	case "show":
		if len(args) < 2 {
			fmt.Println("Usage: changeset show <changeset_id>")
			return
		}
		cli.showChangeset(args[1], changesetManager)
	case "execute", "exec":
		if len(args) < 2 {
			fmt.Println("Usage: changeset execute <changeset_id>")
			return
		}
		cli.executeChangeset(args[1], changesetManager)
	case "rollback":
		if len(args) < 2 {
			fmt.Println("Usage: changeset rollback <changeset_id>")
			return
		}
		cli.rollbackChangeset(args[1], changesetManager)
	case "add":
		if len(args) < 3 {
			fmt.Println("Usage: changeset add <changeset_id> <device_id> <operation> [args...]")
			return
		}
		cli.addCommandToChangeset(args[1:], changesetManager)
	default:
		fmt.Printf("Unknown changeset subcommand: %s\n", args[0])
		cli.showChangesetHelp()
	}
}

// showChangesetHelp displays help for changeset commands
func (cli *InteractiveCLI) showChangesetHelp() {
	fmt.Println("Changeset Management Commands:")
	fmt.Println("  changeset create [description]         - Create new changeset")
	fmt.Println("  changeset list                         - List all changesets")
	fmt.Println("  changeset show <id>                    - Show changeset details")
	fmt.Println("  changeset execute <id>                 - Execute changeset")
	fmt.Println("  changeset rollback <id>                - Rollback changeset")
	fmt.Println("  changeset add <id> <device> <op> [args] - Add command to changeset")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  changeset create \"Update WiFi settings\"")
	fmt.Println("  changeset add cs-123 device1 configure_wifi --ssid=NewSSID")
	fmt.Println("  changeset execute cs-123")
}

// createChangeset creates a new changeset
func (cli *InteractiveCLI) createChangeset(args []string, changesetManager *changeset.SimpleManager) {
	var description string
	if len(args) > 0 {
		description = strings.Join(args, " ")
	} else {
		description = "Changeset created via CLI"
	}

	options := &types.ChangesetOptions{
		Description: description,
		CreatedBy:   "cli-user",
		Metadata: map[string]interface{}{
			"source":      "interactive_cli",
			"created_via": "changeset_command",
		},
	}

	changeset, err := changesetManager.CreateChangeset(context.Background(), options)
	if err != nil {
		fmt.Printf("❌ Failed to create changeset: %v\n", err)
		return
	}

	fmt.Printf("✅ Created changeset: %s\n", changeset.ID)
	fmt.Printf("   Description: %s\n", changeset.Description)
	fmt.Printf("   Status: %s\n", changeset.Status)
	fmt.Printf("   Created: %s\n", changeset.CreatedAt.Format(time.RFC3339))
}

// listChangesets lists all changesets
func (cli *InteractiveCLI) listChangesets(changesetManager *changeset.SimpleManager) {
	// This would require implementing a ListChangesets method in the manager
	fmt.Println("📋 Changeset Listing:")
	fmt.Println("(Full listing functionality not yet implemented)")
	fmt.Println("This would show all changesets with their status and details")
}

// showChangeset shows details of a changeset
func (cli *InteractiveCLI) showChangeset(changesetID string, changesetManager *changeset.SimpleManager) {
	changeset, err := changesetManager.GetChangeset(changesetID)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve changeset: %v\n", err)
		return
	}

	fmt.Printf("Changeset Details: %s\n", changeset.ID)
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Description: %s\n", changeset.Description)
	fmt.Printf("Status: %s\n", changeset.Status)
	fmt.Printf("Created: %s\n", changeset.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Created by: %s\n", changeset.CreatedBy)

	if changeset.ExecutedAt != nil {
		fmt.Printf("Executed: %s\n", changeset.ExecutedAt.Format(time.RFC3339))
	}
	if changeset.RolledBackAt != nil {
		fmt.Printf("Rolled back: %s\n", changeset.RolledBackAt.Format(time.RFC3339))
	}

	if len(changeset.Commands) > 0 {
		fmt.Printf("\nCommands (%d):\n", len(changeset.Commands))
		fmt.Println(strings.Repeat("-", 20))
		for i, cmd := range changeset.Commands {
			fmt.Printf("%d. %s -> %s: %s\n", i+1, cmd.DeviceID, cmd.Operation, cmd.ID)
		}
	}

	if len(changeset.Results) > 0 {
		fmt.Printf("\nResults (%d):\n", len(changeset.Results))
		fmt.Println(strings.Repeat("-", 15))
		for i, result := range changeset.Results {
			status := "✅ Success"
			if !result.Success {
				status = "❌ Failed"
			}
			fmt.Printf("%d. %s - %s\n", i+1, result.CommandID, status)
		}
	}

	if len(changeset.Metadata) > 0 {
		fmt.Println("\nMetadata:")
		fmt.Println(strings.Repeat("-", 10))
		for key, value := range changeset.Metadata {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}

// executeChangeset executes a changeset
func (cli *InteractiveCLI) executeChangeset(changesetID string, changesetManager *changeset.SimpleManager) {
	fmt.Printf("🚀 Executing changeset: %s\n", changesetID)
	fmt.Println("This may take some time depending on the number of commands...")
	fmt.Println()

	startTime := time.Now()
	err := changesetManager.ExecuteChangeset(context.Background(), changesetID)
	executionTime := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Changeset execution failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Changeset executed successfully in %v\n", executionTime)
	fmt.Println("Use 'changeset show' to see detailed results")
}

// rollbackChangeset rolls back a changeset
func (cli *InteractiveCLI) rollbackChangeset(changesetID string, changesetManager *changeset.SimpleManager) {
	fmt.Printf("🔄 Rolling back changeset: %s\n", changesetID)
	fmt.Println("This may take some time depending on the number of rollback commands...")
	fmt.Println()

	startTime := time.Now()
	err := changesetManager.RollbackChangeset(context.Background(), changesetID)
	rollbackTime := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Changeset rollback failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Changeset rolled back successfully in %v\n", rollbackTime)
	fmt.Println("Use 'changeset show' to see updated status")
}

// addCommandToChangeset adds a command to an existing changeset
func (cli *InteractiveCLI) addCommandToChangeset(args []string, changesetManager *changeset.SimpleManager) {
	if len(args) < 3 {
		fmt.Println("Usage: changeset add <changeset_id> <device_id> <operation> [args...]")
		return
	}

	changesetID := args[0]
	deviceID := args[1]
	operation := args[2]
	cmdArgs := args[3:]

	// Parse command arguments into a map
	cmdParams := make(map[string]interface{})
	for _, arg := range cmdArgs {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			key := strings.TrimPrefix(parts[0], "--")
			value := parts[1]
			cmdParams[key] = value
		}
	}

	// Create command
	command := &types.Command{
		DeviceID:    deviceID,
		Operation:   operation,
		Args:        cmdParams,
		Timeout:     30 * time.Second, // default timeout
		Status:      types.CommandStatusPending,
		CreatedAt:   time.Now(),
		Expectation: "result", // expect result by default
	}

	err := changesetManager.AddCommandToChangeset(changesetID, command)
	if err != nil {
		fmt.Printf("❌ Failed to add command to changeset: %v\n", err)
		return
	}

	fmt.Printf("✅ Added command to changeset %s\n", changesetID)
	fmt.Printf("   Device: %s\n", deviceID)
	fmt.Printf("   Operation: %s\n", operation)
	if len(cmdParams) > 0 {
		fmt.Printf("   Parameters: %v\n", cmdParams)
	}
}

// Additional helper functions for advanced changeset operations could go here
// For example: batch operations, changeset templates, etc.

package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"rtk_controller/pkg/types"
)

// sendCommand sends a command to a device
func (c *CLI) sendCommand(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: send <device_id> <operation> [args...]")
	}

	deviceID := args[0]
	operation := args[1]
	commandArgs := args[2:]

	timeout, _ := cmd.Flags().GetDuration("timeout")
	async, _ := cmd.Flags().GetBool("async")

	// Parse command arguments
	cmdParams := make(map[string]interface{})
	for _, arg := range commandArgs {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			key := strings.TrimPrefix(parts[0], "--")
			value := parts[1]
			
			// Try to parse as different types
			if val, err := strconv.ParseBool(value); err == nil {
				cmdParams[key] = val
			} else if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				cmdParams[key] = val
			} else if val, err := strconv.ParseFloat(value, 64); err == nil {
				cmdParams[key] = val
			} else {
				cmdParams[key] = value
			}
		}
	}

	// Parse device ID to extract tenant, site, deviceID
	deviceParts := strings.Split(deviceID, "-")
	tenant := "default"
	site := "default"
	actualDeviceID := deviceID
	
	if len(deviceParts) >= 3 {
		tenant = deviceParts[0]
		site = deviceParts[1]
		actualDeviceID = strings.Join(deviceParts[2:], "-")
	}

	// Send command using the command manager interface
	timeoutSeconds := int(timeout.Seconds())
	command, err := c.commandManager.SendCommand(tenant, site, actualDeviceID, operation, cmdParams, timeoutSeconds)
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	commandID := command.ID

	fmt.Printf("Command sent successfully\n")
	fmt.Printf("Command ID: %s\n", commandID)
	fmt.Printf("Device:     %s\n", deviceID)
	fmt.Printf("Operation:  %s\n", operation)

	if len(cmdParams) > 0 {
		fmt.Printf("Arguments:\n")
		for key, value := range cmdParams {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	if !async {
		fmt.Printf("\nWaiting for response...\n")
		return c.waitForCommandResult(commandID, timeout)
	}

	fmt.Printf("\nCommand sent asynchronously. Use 'cmd status %s' to check progress.\n", commandID)
	return nil
}

// commandStatus gets command status
func (c *CLI) commandStatus(cmd *cobra.Command, args []string) error {
	commandID := args[0]

	command, err := c.commandManager.GetCommand(commandID)
	if err != nil {
		return fmt.Errorf("failed to get command: %w", err)
	}

	if command == nil {
		return fmt.Errorf("command not found: %s", commandID)
	}

	return c.printCommandStatus(command)
}

// commandList lists commands
func (c *CLI) commandList(cmd *cobra.Command, args []string) error {
	fmt.Println("Command listing functionality requires integration with command storage")
	fmt.Println("This would show recent commands and their status")
	return nil
}

// commandHistory shows command history
func (c *CLI) commandHistory(cmd *cobra.Command, args []string) error {
	fmt.Println("Command history functionality requires integration with command storage")
	fmt.Println("This would show historical commands and their results")
	return nil
}

// commandCancel cancels a command
func (c *CLI) commandCancel(cmd *cobra.Command, args []string) error {
	commandID := args[0]
	fmt.Printf("Command cancel functionality for %s requires integration with command manager\n", commandID)
	return nil
}

// Helper methods

func (c *CLI) waitForCommandResult(commandID string, timeout time.Duration) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ticker.C:
			command, err := c.commandManager.GetCommand(commandID)
			if err != nil {
				return fmt.Errorf("failed to check command status: %w", err)
			}

			if command == nil {
				return fmt.Errorf("command not found: %s", commandID)
			}

			switch command.Status {
			case "completed":
				fmt.Printf("✓ Command completed successfully\n")
				if command.Result != nil {
					fmt.Printf("\nResult:\n")
					return c.printJSON(command.Result)
				}
				return nil

			case "failed":
				fmt.Printf("✗ Command failed\n")
				if command.Error != "" {
					fmt.Printf("Error: %s\n", command.Error)
				}
				return nil

			case "timeout":
				fmt.Printf("⏰ Command timed out\n")
				return nil

			case "sent":
				fmt.Printf("📤 Command acknowledged by device\n")

			case "pending":
				fmt.Printf("⏳ Command pending...\n")
			}

		case <-timeoutTimer.C:
			fmt.Printf("⏰ Timeout waiting for command result\n")
			return nil
		}
	}
}

func (c *CLI) printCommandStatus(command *types.DeviceCommand) error {
	fmt.Printf("Command Status\n")
	fmt.Printf("==============\n\n")
	fmt.Printf("Command ID:    %s\n", command.ID)
	fmt.Printf("Device:        %s\n", command.DeviceID)
	fmt.Printf("Operation:     %s\n", command.Operation)
	fmt.Printf("Status:        %s\n", command.Status)
	fmt.Printf("Created:       %s\n", command.CreatedAt.Format(time.RFC3339))

	if command.SentAt != nil {
		fmt.Printf("Sent:          %s\n", command.SentAt.Format(time.RFC3339))
	}
	if command.CompletedAt != nil {
		fmt.Printf("Completed:     %s\n", command.CompletedAt.Format(time.RFC3339))
	}

	fmt.Printf("Timeout:       %d ms\n", command.TimeoutMS)

	if len(command.Args) > 0 {
		fmt.Printf("\nArguments:\n")
		for key, value := range command.Args {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	if command.Result != nil {
		fmt.Printf("\nResult:\n")
		return c.printJSON(command.Result)
	}

	if command.Error != "" {
		fmt.Printf("\nError: %s\n", command.Error)
	}

	return nil
}

func (c *CLI) printCommandsTable(commands []types.DeviceCommand) error {
	if len(commands) == 0 {
		fmt.Println("No commands found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "COMMAND_ID\tDEVICE\tOPERATION\tSTATUS\tCREATED\tDURATION")

	for _, command := range commands {
		duration := "unknown"
		if command.CompletedAt != nil {
			duration = command.CompletedAt.Sub(command.CreatedAt).Round(time.Millisecond).String()
		} else {
			duration = time.Since(command.CreatedAt).Round(time.Millisecond).String()
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			command.ID,
			command.DeviceID,
			command.Operation,
			command.Status,
			command.CreatedAt.Format("15:04:05"),
			duration,
		)
	}

	return w.Flush()
}
package cli

import (
	"fmt"
	"os"
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"rtk_controller/pkg/utils"
)

// systemStatus shows system status
func (c *CLI) systemStatus(cmd *cobra.Command, args []string) error {
	component, _ := cmd.Flags().GetString("component")

	if component != "" {
		return c.printComponentStatus(component)
	}

	return c.printSystemStatus()
}

// systemStats shows system statistics
func (c *CLI) systemStats(cmd *cobra.Command, args []string) error {
	period, _ := cmd.Flags().GetString("period")

	duration, err := time.ParseDuration(period)
	if err != nil {
		return fmt.Errorf("invalid period: %w", err)
	}

	return c.printSystemStats(duration)
}

// systemHealth shows system health
func (c *CLI) systemHealth(cmd *cobra.Command, args []string) error {
	return c.printSystemHealth()
}

// configShow shows configuration
func (c *CLI) configShow(cmd *cobra.Command, args []string) error {
	section, _ := cmd.Flags().GetString("section")

	config := c.config

	if section != "" {
		switch section {
		case "mqtt":
			return c.printJSON(config.MQTT)
		// API and Console sections removed - using CLI only
		case "storage":
			return c.printJSON(config.Storage)
		case "diagnosis":
			return c.printJSON(config.Diagnosis)
		case "schema":
			return c.printJSON(config.Schema)
		case "logging":
			return c.printJSON(config.Logging)
		default:
			return fmt.Errorf("unknown configuration section: %s", section)
		}
	}

	return c.printJSON(config)
}

// configReload reloads configuration
func (c *CLI) configReload(cmd *cobra.Command, args []string) error {
	// Note: This would need to be implemented with a config manager
	fmt.Println("Configuration reload functionality not yet implemented")
	fmt.Println("This requires integration with the configuration manager")
	return nil
}

// configSet sets configuration value
func (c *CLI) configSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Note: This would need to be implemented with a config manager
	fmt.Printf("Setting configuration: %s = %s\n", key, value)
	fmt.Println("Configuration update functionality not yet implemented")
	fmt.Println("This requires integration with the configuration manager")
	return nil
}

// Helper methods

func (c *CLI) printSystemStatus() error {
	fmt.Printf("RTK Controller System Status\n")
	fmt.Printf("============================\n\n")

	// MQTT Status
	mqttStatus := "disconnected"
	if c.mqttClient != nil && c.mqttClient.IsConnected() {
		mqttStatus = "connected"
	}
	fmt.Printf("MQTT:          %s\n", mqttStatus)
	if c.config != nil {
		fmt.Printf("MQTT Broker:   %s:%d\n", c.config.MQTT.Broker, c.config.MQTT.Port)
	}

	// Device Manager Status
	if c.deviceManager != nil {
		deviceStats := c.deviceManager.GetStats()
		fmt.Printf("Devices:       %d total, %d online, %d offline\n",
			deviceStats.TotalDevices, deviceStats.OnlineDevices, deviceStats.OfflineDevices)
	}

	// Command Manager Status
	if c.commandManager != nil {
		cmdStats := c.commandManager.GetStats()
		fmt.Printf("Commands:      %d total, %d pending, %d completed\n",
			cmdStats.TotalCommands, cmdStats.PendingCommands, cmdStats.CompletedCommands)
	}

	// System Info
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("\nSystem Information\n")
	fmt.Printf("==================\n")
	fmt.Printf("Go Version:    %s\n", runtime.Version())
	fmt.Printf("OS/Arch:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Goroutines:    %d\n", runtime.NumGoroutine())
	fmt.Printf("Memory:        %s allocated, %s system\n",
		utils.FormatBytes(m.Alloc), utils.FormatBytes(m.Sys))

	return nil
}

func (c *CLI) printComponentStatus(component string) error {
	fmt.Printf("Component Status: %s\n", component)
	fmt.Printf("========================\n\n")

	switch component {
	case "mqtt":
		if c.mqttClient == nil {
			fmt.Println("MQTT client not initialized")
			return nil
		}

		status := "disconnected"
		if c.mqttClient.IsConnected() {
			status = "connected"
		}

		fmt.Printf("Status:        %s\n", status)
		fmt.Printf("Broker:        %s:%d\n", c.config.MQTT.Broker, c.config.MQTT.Port)
		fmt.Printf("Client ID:     %s\n", c.config.MQTT.ClientID)
		fmt.Printf("TLS Enabled:   %t\n", c.config.MQTT.TLS.Enabled)

		if logger := c.mqttClient.GetMessageLogger(); logger != nil {
			stats := logger.GetStats()
			fmt.Printf("\nMessage Logging:\n")
			fmt.Printf("Total Messages: %d\n", stats.TotalMessages)
			fmt.Printf("Purged:         %d\n", stats.PurgedMessages)
		}

	case "storage":
		fmt.Printf("Storage Type:  BuntDB\n")
		if c.config != nil {
			fmt.Printf("Storage Path:  %s\n", c.config.Storage.Path)
		}
		fmt.Println("Status:        operational")

	case "devices":
		if c.deviceManager == nil {
			fmt.Println("Device manager not initialized")
			return nil
		}

		stats := c.deviceManager.GetStats()
		fmt.Printf("Total Devices: %d\n", stats.TotalDevices)
		fmt.Printf("Online:        %d\n", stats.OnlineDevices)
		fmt.Printf("Offline:       %d\n", stats.OfflineDevices)

	case "commands":
		if c.commandManager == nil {
			fmt.Println("Command manager not initialized")
			return nil
		}

		stats := c.commandManager.GetStats()
		fmt.Printf("Total:         %d\n", stats.TotalCommands)
		fmt.Printf("Pending:       %d\n", stats.PendingCommands)
		fmt.Printf("Completed:     %d\n", stats.CompletedCommands)
		fmt.Printf("Failed:        %d\n", stats.FailedCommands)

	default:
		return fmt.Errorf("unknown component: %s", component)
	}

	return nil
}

func (c *CLI) printSystemStats(period time.Duration) error {
	fmt.Printf("System Statistics (last %s)\n", period.String())
	fmt.Printf("===============================\n\n")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Device statistics
	if c.deviceManager != nil {
		stats := c.deviceManager.GetStats()
		fmt.Fprintln(w, "METRIC\tVALUE\tUNIT")
		fmt.Fprintf(w, "Total Devices\t%d\tcount\n", stats.TotalDevices)
		fmt.Fprintf(w, "Online Devices\t%d\tcount\n", stats.OnlineDevices)
		fmt.Fprintf(w, "Offline Devices\t%d\tcount\n", stats.OfflineDevices)
		// fmt.Fprintf(w, "Total Events\t%d\tcount\n", stats.TotalEvents)
	}

	// Command statistics
	if c.commandManager != nil {
		stats := c.commandManager.GetStats()
		fmt.Fprintf(w, "Total Commands\t%d\tcount\n", stats.TotalCommands)
		fmt.Fprintf(w, "Pending Commands\t%d\tcount\n", stats.PendingCommands)
		fmt.Fprintf(w, "Completed Commands\t%d\tcount\n", stats.CompletedCommands)
		fmt.Fprintf(w, "Failed Commands\t%d\tcount\n", stats.FailedCommands)
	}

	// System statistics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(w, "Memory Allocated\t%d\tbytes\n", m.Alloc)
	fmt.Fprintf(w, "Memory System\t%d\tbytes\n", m.Sys)
	fmt.Fprintf(w, "Goroutines\t%d\tcount\n", runtime.NumGoroutine())

	return w.Flush()
}

func (c *CLI) printSystemHealth() error {
	fmt.Printf("System Health Check\n")
	fmt.Printf("===================\n\n")

	overall := "healthy"
	issues := []string{}

	// Check MQTT connection
	if c.mqttClient == nil || !c.mqttClient.IsConnected() {
		overall = "unhealthy"
		issues = append(issues, "MQTT connection is down")
	} else {
		fmt.Printf("✓ MQTT connection: healthy\n")
	}

	// Check storage
	if c.storage == nil {
		overall = "unhealthy"
		issues = append(issues, "Storage is not available")
	} else {
		fmt.Printf("✓ Storage: healthy\n")
	}

	// Check device manager
	if c.deviceManager == nil {
		overall = "unhealthy"
		issues = append(issues, "Device manager is not available")
	} else {
		fmt.Printf("✓ Device manager: healthy\n")
	}

	// Check command manager
	if c.commandManager == nil {
		overall = "unhealthy"
		issues = append(issues, "Command manager is not available")
	} else {
		fmt.Printf("✓ Command manager: healthy\n")
	}

	// Check memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memUsagePercent := float64(m.Alloc) / float64(m.Sys) * 100
	if memUsagePercent > 80 {
		overall = "warning"
		issues = append(issues, fmt.Sprintf("High memory usage: %.1f%%", memUsagePercent))
	} else {
		fmt.Printf("✓ Memory usage: %.1f%% (healthy)\n", memUsagePercent)
	}

	fmt.Printf("\nOverall Status: %s\n", overall)

	if len(issues) > 0 {
		fmt.Printf("\nIssues:\n")
		for _, issue := range issues {
			fmt.Printf("  ⚠ %s\n", issue)
		}
	}

	return nil
}

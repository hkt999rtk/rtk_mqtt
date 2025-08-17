package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rtk_controller/internal/storage"
	"rtk_controller/pkg/types"
)

// DeviceStatus represents device monitoring status
type DeviceStatus struct {
	DeviceID   string
	Online     bool
	CPU        float64
	Memory     float64
	LastUpdate time.Time
}

// ConnectionStatus represents connection monitoring status
type ConnectionStatus struct {
	ConnectionID string
	Latency      float64
	PacketLoss   float64
	Bandwidth    float64
}

func main() {
	fmt.Println("🔍 RTK Controller - Real-time Topology Monitor")
	fmt.Println("============================================")
	fmt.Println()

	// Create storage
	store, err := storage.NewBuntDB("data")
	if err != nil {
		fmt.Printf("Error: Failed to open storage: %v\n", err)
		return
	}
	defer store.Close()

	topologyStorage := storage.NewTopologyStorage(store)

	// Load topology
	topology, err := topologyStorage.GetTopology("default", "office-1")
	if err != nil {
		fmt.Println("No topology found. Please run topology_demo.sh first.")
		return
	}

	fmt.Printf("Monitoring %d devices and %d connections\n", 
		len(topology.Devices), len(topology.Connections))
	fmt.Println("\nPress Ctrl+C to stop monitoring\n")

	// Setup signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring loop
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	iteration := 0
	for {
		select {
		case <-sigChan:
			fmt.Println("\n\n✅ Monitoring stopped")
			return
		case <-ticker.C:
			iteration++
			clearScreen()
			displayMonitoring(topology, iteration)
			
			// Simulate topology changes
			if iteration%5 == 0 {
				simulateTopologyChange(topology, topologyStorage)
			}
		}
	}
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func displayMonitoring(topology *types.NetworkTopology, iteration int) {
	now := time.Now().Format("15:04:05")
	
	fmt.Printf("🔍 Topology Monitor | Time: %s | Iteration: %d\n", now, iteration)
	fmt.Println("═══════════════════════════════════════════════════")
	
	// Device Status
	fmt.Println("\n📊 Device Status:")
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("%-20s %-10s %-8s %-8s %-10s\n", 
		"Device", "Status", "CPU%", "Mem%", "Uptime")
	fmt.Println("─────────────────────────────────────────────")
	
	for _, device := range topology.Devices {
		// Simulate metrics
		status := "🟢 Online"
		cpu := 20 + rand.Float64()*30
		mem := 30 + rand.Float64()*40
		uptime := time.Since(time.Unix(device.LastSeen, 0))
		
		// Randomly simulate issues
		if rand.Float64() < 0.1 {
			status = "🔴 Offline"
			cpu = 0
			mem = 0
		} else if cpu > 40 {
			status = "🟡 Warning"
		}
		
		fmt.Printf("%-20s %-10s %6.1f%% %6.1f%% %10s\n",
			device.Hostname, status, cpu, mem, formatDuration(uptime))
	}
	
	// Connection Health
	fmt.Println("\n🔗 Connection Health:")
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("%-30s %-10s %-10s %-12s\n", 
		"Link", "Latency", "Loss", "Bandwidth")
	fmt.Println("─────────────────────────────────────────────")
	
	for _, conn := range topology.Connections {
		// Get device names
		fromDevice := "unknown"
		toDevice := "unknown"
		if d, exists := topology.Devices[conn.FromDeviceID]; exists {
			fromDevice = d.Hostname
		}
		if d, exists := topology.Devices[conn.ToDeviceID]; exists {
			toDevice = d.Hostname
		}
		
		link := fmt.Sprintf("%s → %s", fromDevice, toDevice)
		if len(link) > 28 {
			link = link[:28] + ".."
		}
		
		// Simulate metrics
		latency := conn.Metrics.Latency + rand.Float64()*0.5
		loss := rand.Float64() * 0.5 // 0-0.5% loss
		bandwidth := conn.Metrics.Bandwidth * (0.7 + rand.Float64()*0.3)
		
		// Format based on health
		latencyStr := fmt.Sprintf("%.2f ms", latency)
		lossStr := fmt.Sprintf("%.2f%%", loss)
		bwStr := fmt.Sprintf("%.0f Mbps", bandwidth)
		
		if latency > 1 {
			latencyStr = "⚠️ " + latencyStr
		}
		if loss > 0.1 {
			lossStr = "⚠️ " + lossStr
		}
		
		fmt.Printf("%-30s %-10s %-10s %-12s\n",
			link, latencyStr, lossStr, bwStr)
	}
	
	// Summary Statistics
	fmt.Println("\n📈 Summary Statistics:")
	fmt.Println("─────────────────────────────────────────────")
	
	onlineCount := 0
	for _, device := range topology.Devices {
		if device.Online {
			onlineCount++
		}
	}
	
	fmt.Printf("• Total Devices:     %d\n", len(topology.Devices))
	fmt.Printf("• Online Devices:    %d (%.0f%%)\n", 
		onlineCount, float64(onlineCount)/float64(len(topology.Devices))*100)
	fmt.Printf("• Total Connections: %d\n", len(topology.Connections))
	fmt.Printf("• Network Health:    ")
	
	healthScore := float64(onlineCount) / float64(len(topology.Devices)) * 100
	if healthScore >= 95 {
		fmt.Println("🟢 Excellent")
	} else if healthScore >= 80 {
		fmt.Println("🟡 Good")
	} else {
		fmt.Println("🔴 Critical")
	}
	
	// Recent Events
	fmt.Println("\n📝 Recent Events:")
	fmt.Println("─────────────────────────────────────────────")
	
	// Simulate events
	events := generateRandomEvents(iteration)
	for _, event := range events {
		fmt.Println(event)
	}
}

func simulateTopologyChange(topology *types.NetworkTopology, storage *storage.TopologyStorage) {
	// Randomly toggle device status
	for _, device := range topology.Devices {
		if rand.Float64() < 0.1 { // 10% chance to toggle
			device.Online = !device.Online
			device.LastSeen = time.Now().Unix()
		}
	}
	
	// Update connection metrics
	for i := range topology.Connections {
		conn := &topology.Connections[i]
		// Vary metrics slightly
		conn.Metrics.Latency = conn.Metrics.Latency * (0.9 + rand.Float64()*0.2)
		conn.Metrics.Bandwidth = conn.Metrics.Bandwidth * (0.95 + rand.Float64()*0.1)
	}
	
	topology.UpdatedAt = time.Now()
	storage.SaveTopology(topology)
}

func generateRandomEvents(iteration int) []string {
	events := []string{}
	timestamp := time.Now().Format("15:04:05")
	
	eventTemplates := []string{
		"[%s] ℹ️  Device health check completed",
		"[%s] ✓ Topology sync successful",
		"[%s] 📊 Metrics collection cycle #%d",
		"[%s] 🔄 Connection state refreshed",
	}
	
	// Always show at least one event
	events = append(events, fmt.Sprintf(eventTemplates[2], timestamp, iteration))
	
	// Randomly add more events
	if rand.Float64() < 0.3 {
		events = append(events, fmt.Sprintf(eventTemplates[0], timestamp))
	}
	if rand.Float64() < 0.2 {
		events = append(events, fmt.Sprintf(eventTemplates[1], timestamp))
	}
	if iteration%10 == 0 {
		events = append(events, fmt.Sprintf("[%s] 🎯 Milestone: %d monitoring cycles", timestamp, iteration))
	}
	
	return events
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}
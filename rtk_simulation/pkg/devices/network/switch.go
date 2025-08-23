package network

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	// "github.com/sirupsen/logrus"
	// "rtk_simulation/pkg/config"
	"rtk_simulation/pkg/devices/base"
)

type Switch struct {
	*base.BaseDevice

	ports           []Port
	vlanConfig      []VLANConfig
	spanningTree    SpanningTreeConfig
	portMirroring   PortMirroringConfig
	macAddressTable map[string]MACEntry
	trafficStats    TrafficStatistics
	mu              sync.RWMutex
}

type Port struct {
	PortNumber      int     `json:"port_number"`
	Status          string  `json:"status"`
	Speed           string  `json:"speed"`
	Duplex          string  `json:"duplex"`
	VLANID          int     `json:"vlan_id"`
	ConnectedMAC    string  `json:"connected_mac,omitempty"`
	LinkUtilization float64 `json:"link_utilization"`
	ErrorCount      int64   `json:"error_count"`
}

type VLANConfig struct {
	VLANID      int    `json:"vlan_id"`
	Name        string `json:"name"`
	Ports       []int  `json:"ports"`
	TaggedPorts []int  `json:"tagged_ports"`
}

type SpanningTreeConfig struct {
	Enabled    bool   `json:"enabled"`
	Priority   int    `json:"priority"`
	RootBridge bool   `json:"root_bridge"`
	Protocol   string `json:"protocol"`
}

type PortMirroringConfig struct {
	Enabled     bool  `json:"enabled"`
	SourcePorts []int `json:"source_ports"`
	TargetPort  int   `json:"target_port"`
}

type MACEntry struct {
	MACAddress string    `json:"mac_address"`
	Port       int       `json:"port"`
	VLANID     int       `json:"vlan_id"`
	LearnedAt  time.Time `json:"learned_at"`
	Static     bool      `json:"static"`
}

type TrafficStatistics struct {
	TotalPackets     int64   `json:"total_packets"`
	TotalBytes       int64   `json:"total_bytes"`
	BroadcastPackets int64   `json:"broadcast_packets"`
	UnicastPackets   int64   `json:"unicast_packets"`
	MulticastPackets int64   `json:"multicast_packets"`
	ErrorPackets     int64   `json:"error_packets"`
	Throughput       float64 `json:"throughput"`
}

func NewSwitch(deviceConfig base.DeviceConfig, mqttConfig base.MQTTConfig) (*Switch, error) {
	baseDevice := base.NewBaseDevice(deviceConfig)

	sw := &Switch{
		BaseDevice:      baseDevice,
		macAddressTable: make(map[string]MACEntry),
	}

	if err := sw.initializePorts(deviceConfig); err != nil {
		return nil, err
	}

	sw.initializeVLANs(deviceConfig)
	sw.initializeSpanningTree(deviceConfig)

	return sw, nil
}

func (s *Switch) initializePorts(config base.DeviceConfig) error {
	// 從配置中獲取端口數量，預設為 24
	portCount := 24
	if config.Extra != nil {
		if pc, ok := config.Extra["port_count"].(float64); ok {
			portCount = int(pc)
		} else if pc, ok := config.Extra["port_count"].(int); ok {
			portCount = pc
		}
	}

	// 限制端口數量範圍
	if portCount < 4 {
		portCount = 4
	} else if portCount > 48 {
		portCount = 48
	}

	s.ports = make([]Port, portCount)
	for i := 0; i < portCount; i++ {
		s.ports[i] = Port{
			PortNumber:      i + 1,
			Status:          "up",
			Speed:           "1000Mbps",
			Duplex:          "full",
			VLANID:          1,
			LinkUtilization: 0.0,
			ErrorCount:      0,
		}
	}

	return nil
}

// Helper functions
func (s *Switch) getRandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min) + min
}

func (s *Switch) getRandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (s *Switch) getRandomBool() bool {
	return rand.Intn(2) == 1
}

func (s *Switch) initializeVLANs(config base.DeviceConfig) {
	s.vlanConfig = []VLANConfig{
		{
			VLANID: 1,
			Name:   "default",
			Ports:  []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24},
		},
	}
}

func (s *Switch) initializeSpanningTree(config base.DeviceConfig) {
	s.spanningTree = SpanningTreeConfig{
		Enabled:    true,
		Priority:   32768,
		RootBridge: false,
		Protocol:   "RSTP",
	}
}

func (s *Switch) Start(ctx context.Context) error {
	if err := s.BaseDevice.Start(ctx); err != nil {
		return err
	}

	go s.runPortMonitoring(ctx)
	go s.runMACLearning(ctx)
	go s.runSpanningTree(ctx)
	go s.runTrafficAnalysis(ctx)

	// Switch started successfully
	return nil
}

func (s *Switch) runPortMonitoring(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updatePortStatistics()
		}
	}
}

func (s *Switch) runMACLearning(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.simulateMACLearning()
		}
	}
}

func (s *Switch) runSpanningTree(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processSpanningTree()
		}
	}
}

func (s *Switch) runTrafficAnalysis(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateTrafficStatistics()
		}
	}
}

func (s *Switch) updatePortStatistics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.ports {
		port := &s.ports[i]

		if port.Status == "up" {
			port.LinkUtilization = s.getRandomFloat(0, 80)

			if s.getRandomFloat(0, 100) < 1 {
				port.ErrorCount++
			}
		} else {
			port.LinkUtilization = 0
		}
	}
}

func (s *Switch) simulateMACLearning() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.macAddressTable) < 100 {
		mac := fmt.Sprintf("00:11:22:33:%02x:%02x",
			s.getRandomInt(0, 255),
			s.getRandomInt(0, 255))

		port := s.getRandomInt(1, len(s.ports))

		s.macAddressTable[mac] = MACEntry{
			MACAddress: mac,
			Port:       port,
			VLANID:     1,
			LearnedAt:  time.Now(),
			Static:     false,
		}
	}
}

func (s *Switch) processSpanningTree() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.spanningTree.Enabled {
		for i := range s.ports {
			if s.ports[i].Status == "down" {
				continue
			}

			if s.getRandomFloat(0, 100) < 5 {
				if s.getRandomBool() {
					s.ports[i].Status = "blocking"
				} else {
					s.ports[i].Status = "forwarding"
				}
			}
		}
	}
}

func (s *Switch) updateTrafficStatistics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	increment := int64(s.getRandomInt(100, 10000))
	s.trafficStats.TotalPackets += increment
	s.trafficStats.TotalBytes += increment * int64(s.getRandomInt(64, 1518))

	s.trafficStats.UnicastPackets += int64(float64(increment) * 0.7)
	s.trafficStats.BroadcastPackets += int64(float64(increment) * 0.2)
	s.trafficStats.MulticastPackets += int64(float64(increment) * 0.1)

	if s.getRandomFloat(0, 100) < 1 {
		s.trafficStats.ErrorPackets += int64(s.getRandomInt(1, 10))
	}

	s.trafficStats.Throughput = float64(s.trafficStats.TotalBytes) / (1024 * 1024)
}

func (s *Switch) GenerateStatePayload() base.StatePayload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	baseState := s.BaseDevice.GenerateStatePayload()

	portStatus := make(map[string]interface{})
	activePorts := 0
	for _, port := range s.ports {
		if port.Status != "down" {
			activePorts++
		}
		portStatus[fmt.Sprintf("port_%d", port.PortNumber)] = port.Status
	}

	if baseState.Extra == nil {
		baseState.Extra = make(map[string]interface{})
	}

	baseState.Extra["ports"] = map[string]interface{}{
		"total_ports":  len(s.ports),
		"active_ports": activePorts,
		"port_status":  portStatus,
	}

	baseState.Extra["mac_table_size"] = len(s.macAddressTable)
	baseState.Extra["spanning_tree"] = s.spanningTree.Enabled
	baseState.Extra["vlans"] = len(s.vlanConfig)

	return baseState
}

func (s *Switch) GenerateTelemetryData() map[string]base.TelemetryPayload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	telemetry := s.BaseDevice.GenerateTelemetryData()

	telemetry["port_utilization"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     s.getAveragePortUtilization(),
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "port_utilization",
		},
	}

	telemetry["throughput"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     s.trafficStats.Throughput,
		"unit":      "mbps",
		"tags": map[string]string{
			"metric": "throughput",
		},
	}

	telemetry["mac_table_usage"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     float64(len(s.macAddressTable)),
		"unit":      "entries",
		"tags": map[string]string{
			"metric": "mac_table_usage",
		},
	}

	telemetry["error_rate"] = base.TelemetryPayload{
		"timestamp": time.Now(),
		"value":     s.getErrorRate(),
		"unit":      "percent",
		"tags": map[string]string{
			"metric": "error_rate",
		},
	}

	return telemetry
}

func (s *Switch) getAveragePortUtilization() float64 {
	if len(s.ports) == 0 {
		return 0
	}

	total := 0.0
	for _, port := range s.ports {
		if port.Status != "down" {
			total += port.LinkUtilization
		}
	}

	return total / float64(len(s.ports))
}

func (s *Switch) getErrorRate() float64 {
	if s.trafficStats.TotalPackets == 0 {
		return 0
	}

	return float64(s.trafficStats.ErrorPackets) / float64(s.trafficStats.TotalPackets) * 100
}

func (s *Switch) HandleCommand(cmd base.Command) error {
	// Handling command: cmd.Action

	switch cmd.Action {
	case "get_port_status":
		return s.sendPortStatus(cmd)
	case "set_port_config":
		return s.setPortConfig(cmd)
	case "get_mac_table":
		return s.sendMACTable(cmd)
	case "clear_mac_table":
		return s.clearMACTable(cmd)
	case "get_vlan_config":
		return s.sendVLANConfig(cmd)
	case "set_vlan_config":
		return s.setVLANConfig(cmd)
	default:
		return s.BaseDevice.HandleCommand(cmd)
	}
}

func (s *Switch) sendPortStatus(cmd base.Command) error {
	s.mu.RLock()
	portsData, _ := json.Marshal(s.ports)
	s.mu.RUnlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      string(portsData),
	}

	return s.PublishCommandResponse(response)
}

func (s *Switch) setPortConfig(cmd base.Command) error {
	var portConfig struct {
		PortNumber int    `json:"port_number"`
		Status     string `json:"status,omitempty"`
		Speed      string `json:"speed,omitempty"`
		VLANID     int    `json:"vlan_id,omitempty"`
	}

	if err := json.Unmarshal([]byte(cmd.Payload), &portConfig); err != nil {
		return fmt.Errorf("invalid port config: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if portConfig.PortNumber < 1 || portConfig.PortNumber > len(s.ports) {
		return fmt.Errorf("invalid port number: %d", portConfig.PortNumber)
	}

	port := &s.ports[portConfig.PortNumber-1]

	if portConfig.Status != "" {
		port.Status = portConfig.Status
	}
	if portConfig.Speed != "" {
		port.Speed = portConfig.Speed
	}
	if portConfig.VLANID > 0 {
		port.VLANID = portConfig.VLANID
	}

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "Port configuration updated",
	}

	return s.PublishCommandResponse(response)
}

func (s *Switch) sendMACTable(cmd base.Command) error {
	s.mu.RLock()
	macTableData, _ := json.Marshal(s.macAddressTable)
	s.mu.RUnlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      string(macTableData),
	}

	return s.PublishCommandResponse(response)
}

func (s *Switch) clearMACTable(cmd base.Command) error {
	s.mu.Lock()
	s.macAddressTable = make(map[string]MACEntry)
	s.mu.Unlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "MAC table cleared",
	}

	return s.PublishCommandResponse(response)
}

func (s *Switch) sendVLANConfig(cmd base.Command) error {
	s.mu.RLock()
	vlanData, _ := json.Marshal(s.vlanConfig)
	s.mu.RUnlock()

	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      string(vlanData),
	}

	return s.PublishCommandResponse(response)
}

func (s *Switch) setVLANConfig(cmd base.Command) error {
	var vlanConfig VLANConfig
	if err := json.Unmarshal([]byte(cmd.Payload), &vlanConfig); err != nil {
		return fmt.Errorf("invalid VLAN config: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, vlan := range s.vlanConfig {
		if vlan.VLANID == vlanConfig.VLANID {
			s.vlanConfig[i] = vlanConfig
			goto updated
		}
	}

	s.vlanConfig = append(s.vlanConfig, vlanConfig)

updated:
	response := base.CommandResponse{
		CommandID: cmd.ID,
		Status:    "success",
		Data:      "VLAN configuration updated",
	}

	return s.PublishCommandResponse(response)
}

package types

import (
	"time"
)

// DeviceRole represents the role of a device in the network
type DeviceRole string

const (
	RoleGateway     DeviceRole = "gateway"
	RoleAccessPoint DeviceRole = "access_point"
	RoleSwitch      DeviceRole = "switch"
	RoleClient      DeviceRole = "client"
	RoleBridge      DeviceRole = "bridge"
	RoleRouter      DeviceRole = "router"
)

// NetworkTopology represents the complete network topology
type NetworkTopology struct {
	ID          string                    `json:"id"`          // 拓撲圖ID
	Tenant      string                    `json:"tenant"`      // 租戶ID
	Site        string                    `json:"site"`        // 場域ID
	Devices     map[string]*NetworkDevice `json:"devices"`     // 設備清單 (device_id -> NetworkDevice)
	Connections []DeviceConnection        `json:"connections"` // 連接關係
	Gateway     *GatewayInfo              `json:"gateway"`     // 閘道資訊
	UpdatedAt   time.Time                 `json:"updated_at"`  // 最後更新時間
}

// NetworkDevice represents a device in the network topology
type NetworkDevice struct {
	DeviceID     string                  `json:"device_id"`
	DeviceType   string                  `json:"device_type"` // router, ap, switch, iot, client, bridge
	PrimaryMAC   string                  `json:"primary_mac"` // 主要 MAC 地址
	Hostname     string                  `json:"hostname"`
	Manufacturer string                  `json:"manufacturer"`
	Model        string                  `json:"model"`
	Location     string                  `json:"location"`
	Role         DeviceRole              `json:"role"`       // gateway, access_point, switch, client, bridge
	Interfaces   map[string]NetworkIface `json:"interfaces"` // interface_name -> NetworkIface
	RoutingInfo  *RoutingInfo            `json:"routing_info,omitempty"`
	BridgeInfo   *BridgeInfo             `json:"bridge_info,omitempty"`
	Capabilities []string                `json:"capabilities"` // routing, bridge, ap, client, nat, dhcp
	LastSeen     int64                   `json:"last_seen"`
	Online       bool                    `json:"online"`
}

// NetworkIface represents a network interface
type NetworkIface struct {
	Name        string          `json:"name"`         // eth0, wlan0, br0
	Type        string          `json:"type"`         // ethernet, wifi, bridge, loopback, tunnel
	MacAddress  string          `json:"mac_address"`  // 此介面的 MAC
	IPAddresses []IPAddressInfo `json:"ip_addresses"` // 支援多 IP
	Status      string          `json:"status"`       // up, down, dormant
	MTU         int             `json:"mtu"`
	Speed       int             `json:"speed"`  // Mbps, WiFi 為協商速度
	Duplex      string          `json:"duplex"` // full, half (乙太網路用)

	// WiFi 專用欄位
	WiFiMode string `json:"wifi_mode,omitempty"` // AP, STA, Monitor, Mesh
	SSID     string `json:"ssid,omitempty"`
	BSSID    string `json:"bssid,omitempty"`
	Channel  int    `json:"channel,omitempty"`
	Band     string `json:"band,omitempty"` // 2.4G, 5G, 6G
	RSSI     int    `json:"rssi,omitempty"`
	Security string `json:"security,omitempty"`

	// 橋接介面專用欄位
	BridgedIfaces []string `json:"bridged_ifaces,omitempty"` // 被橋接的介面列表

	// 統計資訊
	TxBytes    int64 `json:"tx_bytes"`
	RxBytes    int64 `json:"rx_bytes"`
	TxPackets  int64 `json:"tx_packets"`
	RxPackets  int64 `json:"rx_packets"`
	TxErrors   int64 `json:"tx_errors"`
	RxErrors   int64 `json:"rx_errors"`
	LastUpdate int64 `json:"last_update"`
}

// IPAddressInfo represents IP address information
type IPAddressInfo struct {
	Address    string   `json:"address"` // IP 地址
	Network    string   `json:"network"` // 網段，如 192.168.1.0/24
	Type       string   `json:"type"`    // static, dhcp, link_local
	Gateway    string   `json:"gateway,omitempty"`
	DNSServers []string `json:"dns_servers,omitempty"`
}

// RoutingInfo represents routing information for a device
type RoutingInfo struct {
	RoutingTable      []RouteEntry    `json:"routing_table"`
	NATRules          []NATRule       `json:"nat_rules,omitempty"`
	ForwardingEnabled bool            `json:"forwarding_enabled"`
	DHCPServer        *DHCPServerInfo `json:"dhcp_server,omitempty"`
}

// RouteEntry represents a routing table entry
type RouteEntry struct {
	Destination string `json:"destination"` // 目的網段
	Gateway     string `json:"gateway"`     // 下一跳
	Interface   string `json:"interface"`   // 出介面
	Metric      int    `json:"metric"`      // 路由權重
	Type        string `json:"type"`        // static, dynamic, connected
}

// NATRule represents a NAT rule
type NATRule struct {
	Type      string `json:"type"` // SNAT, DNAT, MASQUERADE
	SourceNet string `json:"source_net"`
	DestNet   string `json:"dest_net"`
	Interface string `json:"interface"`
	Protocol  string `json:"protocol"` // tcp, udp, all
}

// DHCPServerInfo represents DHCP server information
type DHCPServerInfo struct {
	Enabled      bool        `json:"enabled"`
	IPRange      string      `json:"ip_range"` // 192.168.1.100-192.168.1.200
	SubnetMask   string      `json:"subnet_mask"`
	LeaseTime    int         `json:"lease_time"` // seconds
	Gateway      string      `json:"gateway"`
	DNSServers   []string    `json:"dns_servers"`
	ActiveLeases []DHCPLease `json:"active_leases"`
}

// DHCPLease represents a DHCP lease
type DHCPLease struct {
	MacAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	Hostname   string `json:"hostname,omitempty"`
	LeaseStart int64  `json:"lease_start"` // Unix timestamp
	LeaseEnd   int64  `json:"lease_end"`   // Unix timestamp
}

// BridgeInfo represents bridge information
type BridgeInfo struct {
	BridgeTable []BridgeTableEntry `json:"bridge_table"`
	STPEnabled  bool               `json:"stp_enabled"` // Spanning Tree Protocol
	BridgeID    string             `json:"bridge_id"`
	RootBridge  bool               `json:"root_bridge"`
}

// BridgeTableEntry represents a bridge table entry
type BridgeTableEntry struct {
	MacAddress string `json:"mac_address"`
	Interface  string `json:"interface"`
	VlanID     int    `json:"vlan_id,omitempty"`
	IsLocal    bool   `json:"is_local"`
	Age        int    `json:"age"` // seconds
}

// GatewayInfo represents gateway information
type GatewayInfo struct {
	DeviceID       string   `json:"device_id"`
	IPAddress      string   `json:"ip_address"`
	ExternalIP     string   `json:"external_ip,omitempty"`
	ISPInfo        string   `json:"isp_info,omitempty"`
	DNSServers     []string `json:"dns_servers"`
	ConnectionType string   `json:"connection_type"` // ethernet, pppoe, dhcp
	LastCheck      int64    `json:"last_check"`
}

// DeviceConnection represents a connection between two devices
type DeviceConnection struct {
	ID             string            `json:"id"`
	FromDeviceID   string            `json:"from_device_id"`
	ToDeviceID     string            `json:"to_device_id"`
	FromInterface  string            `json:"from_interface"`
	ToInterface    string            `json:"to_interface"`
	ConnectionType string            `json:"connection_type"` // ethernet, wifi, bridge, route
	IsDirectLink   bool              `json:"is_direct_link"`
	Metrics        ConnectionMetrics `json:"metrics"`
	LastSeen       int64             `json:"last_seen"`
	Discovered     int64             `json:"discovered"`
}

// ConnectionMetrics represents connection quality metrics
type ConnectionMetrics struct {
	RSSI       int     `json:"rssi,omitempty"` // WiFi 訊號強度
	LinkSpeed  int     `json:"link_speed"`     // 連線速度 (Mbps)
	Bandwidth  int     `json:"bandwidth"`      // 可用頻寬 (Mbps)
	Latency    float64 `json:"latency"`        // 延遲 (ms)
	TxBytes    int64   `json:"tx_bytes"`       // 傳送位元組
	RxBytes    int64   `json:"rx_bytes"`       // 接收位元組
	LastUpdate int64   `json:"last_update"`
}

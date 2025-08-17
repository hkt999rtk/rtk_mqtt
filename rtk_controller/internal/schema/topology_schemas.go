package schema

// Topology Discovery Schema - for network topology discovery messages
const topologyDiscoverySchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"title": "RTK Topology Discovery Message",
	"type": "object",
	"required": ["schema", "timestamp", "device_id", "device_info"],
	"properties": {
		"schema": {
			"type": "string",
			"const": "topology.discovery/1.0"
		},
		"timestamp": {
			"type": "integer",
			"description": "Unix timestamp in milliseconds"
		},
		"device_id": {
			"type": "string",
			"description": "Unique device identifier"
		},
		"device_info": {
			"type": "object",
			"required": ["device_type", "primary_mac"],
			"properties": {
				"device_type": {
					"type": "string",
					"enum": ["router", "ap", "switch", "iot", "client", "bridge"]
				},
				"primary_mac": {
					"type": "string",
					"pattern": "^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"
				},
				"hostname": {
					"type": "string"
				},
				"manufacturer": {
					"type": "string"
				},
				"model": {
					"type": "string"
				},
				"location": {
					"type": "string"
				},
				"role": {
					"type": "string",
					"enum": ["gateway", "access_point", "switch", "client", "bridge", "router"]
				},
				"capabilities": {
					"type": "array",
					"items": {
						"type": "string",
						"enum": ["routing", "bridge", "ap", "client", "nat", "dhcp", "mesh"]
					}
				}
			}
		},
		"interfaces": {
			"type": "array",
			"items": {
				"type": "object",
				"required": ["name", "type", "mac_address", "status"],
				"properties": {
					"name": {
						"type": "string",
						"description": "Interface name (e.g., eth0, wlan0)"
					},
					"type": {
						"type": "string",
						"enum": ["ethernet", "wifi", "bridge", "loopback", "tunnel"]
					},
					"mac_address": {
						"type": "string",
						"pattern": "^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"
					},
					"ip_addresses": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"address": {"type": "string"},
								"network": {"type": "string"},
								"type": {
									"type": "string",
									"enum": ["static", "dhcp", "link_local"]
								},
								"gateway": {"type": "string"},
								"dns_servers": {
									"type": "array",
									"items": {"type": "string"}
								}
							}
						}
					},
					"status": {
						"type": "string",
						"enum": ["up", "down", "dormant"]
					},
					"mtu": {"type": "integer"},
					"speed": {"type": "integer", "description": "Speed in Mbps"},
					"duplex": {
						"type": "string",
						"enum": ["full", "half"]
					},
					"wifi_mode": {
						"type": "string",
						"enum": ["AP", "STA", "Monitor", "Mesh"]
					},
					"ssid": {"type": "string"},
					"bssid": {"type": "string"},
					"channel": {"type": "integer"},
					"band": {
						"type": "string",
						"enum": ["2.4G", "5G", "6G"]
					},
					"rssi": {"type": "integer"},
					"security": {"type": "string"},
					"bridged_ifaces": {
						"type": "array",
						"items": {"type": "string"}
					},
					"statistics": {
						"type": "object",
						"properties": {
							"tx_bytes": {"type": "integer"},
							"rx_bytes": {"type": "integer"},
							"tx_packets": {"type": "integer"},
							"rx_packets": {"type": "integer"},
							"tx_errors": {"type": "integer"},
							"rx_errors": {"type": "integer"}
						}
					}
				}
			}
		},
		"routing_info": {
			"type": "object",
			"properties": {
				"routing_table": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"destination": {"type": "string"},
							"gateway": {"type": "string"},
							"interface": {"type": "string"},
							"metric": {"type": "integer"},
							"type": {
								"type": "string",
								"enum": ["static", "dynamic", "connected"]
							}
						}
					}
				},
				"nat_rules": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"type": {
								"type": "string",
								"enum": ["SNAT", "DNAT", "MASQUERADE"]
							},
							"source_net": {"type": "string"},
							"dest_net": {"type": "string"},
							"interface": {"type": "string"},
							"protocol": {
								"type": "string",
								"enum": ["tcp", "udp", "all"]
							}
						}
					}
				},
				"forwarding_enabled": {"type": "boolean"},
				"dhcp_server": {
					"type": "object",
					"properties": {
						"enabled": {"type": "boolean"},
						"ip_range": {"type": "string"},
						"subnet_mask": {"type": "string"},
						"lease_time": {"type": "integer"},
						"gateway": {"type": "string"},
						"dns_servers": {
							"type": "array",
							"items": {"type": "string"}
						},
						"active_leases": {
							"type": "array",
							"items": {
								"type": "object",
								"properties": {
									"mac_address": {"type": "string"},
									"ip_address": {"type": "string"},
									"hostname": {"type": "string"},
									"lease_start": {"type": "integer"},
									"lease_end": {"type": "integer"}
								}
							}
						}
					}
				}
			}
		},
		"bridge_info": {
			"type": "object",
			"properties": {
				"bridge_table": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"mac_address": {"type": "string"},
							"interface": {"type": "string"},
							"vlan_id": {"type": "integer"},
							"is_local": {"type": "boolean"},
							"age": {"type": "integer"}
						}
					}
				},
				"stp_enabled": {"type": "boolean"},
				"bridge_id": {"type": "string"},
				"root_bridge": {"type": "boolean"}
			}
		}
	}
}`

// Topology Connections Schema - for device connection information
const topologyConnectionsSchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"title": "RTK Topology Connections Message",
	"type": "object",
	"required": ["schema", "timestamp", "device_id", "connections"],
	"properties": {
		"schema": {
			"type": "string",
			"const": "topology.connections/1.0"
		},
		"timestamp": {
			"type": "integer",
			"description": "Unix timestamp in milliseconds"
		},
		"device_id": {
			"type": "string",
			"description": "Unique device identifier"
		},
		"connections": {
			"type": "array",
			"items": {
				"type": "object",
				"required": ["connected_device", "connection_type", "local_interface"],
				"properties": {
					"connected_device": {
						"type": "object",
						"required": ["device_id", "mac_address"],
						"properties": {
							"device_id": {"type": "string"},
							"mac_address": {"type": "string"},
							"hostname": {"type": "string"},
							"device_type": {"type": "string"}
						}
					},
					"connection_type": {
						"type": "string",
						"enum": ["ethernet", "wifi", "bridge", "route"]
					},
					"local_interface": {"type": "string"},
					"remote_interface": {"type": "string"},
					"is_direct_link": {"type": "boolean"},
					"metrics": {
						"type": "object",
						"properties": {
							"rssi": {"type": "integer"},
							"link_speed": {"type": "integer"},
							"bandwidth": {"type": "integer"},
							"latency": {"type": "number"},
							"tx_bytes": {"type": "integer"},
							"rx_bytes": {"type": "integer"}
						}
					},
					"last_seen": {"type": "integer"},
					"discovered": {"type": "integer"}
				}
			}
		},
		"gateway_info": {
			"type": "object",
			"properties": {
				"is_gateway": {"type": "boolean"},
				"ip_address": {"type": "string"},
				"external_ip": {"type": "string"},
				"isp_info": {"type": "string"},
				"dns_servers": {
					"type": "array",
					"items": {"type": "string"}
				},
				"connection_type": {
					"type": "string",
					"enum": ["ethernet", "pppoe", "dhcp"]
				}
			}
		}
	}
}`

// WiFi Clients Schema - for WiFi client connection information
const wifiClientsSchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"title": "RTK WiFi Clients Message",
	"type": "object",
	"required": ["schema", "timestamp", "device_id", "interface", "clients"],
	"properties": {
		"schema": {
			"type": "string",
			"const": "telemetry.wifi_clients/1.0"
		},
		"timestamp": {
			"type": "integer",
			"description": "Unix timestamp in milliseconds"
		},
		"device_id": {
			"type": "string",
			"description": "Access Point device identifier"
		},
		"interface": {
			"type": "string",
			"description": "WiFi interface name (e.g., wlan0, wlan1)"
		},
		"ap_info": {
			"type": "object",
			"properties": {
				"ssid": {"type": "string"},
				"bssid": {"type": "string"},
				"channel": {"type": "integer"},
				"band": {
					"type": "string",
					"enum": ["2.4G", "5G", "6G"]
				},
				"supported_channels": {
					"type": "array",
					"items": {"type": "integer"}
				},
				"max_clients": {"type": "integer"},
				"security": {"type": "string"}
			}
		},
		"clients": {
			"type": "array",
			"items": {
				"type": "object",
				"required": ["mac_address", "connected_at"],
				"properties": {
					"mac_address": {
						"type": "string",
						"pattern": "^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"
					},
					"ip_address": {"type": "string"},
					"hostname": {"type": "string"},
					"connected_at": {"type": "integer"},
					"last_seen": {"type": "integer"},
					"rssi": {"type": "integer"},
					"channel": {"type": "integer"},
					"band": {
						"type": "string",
						"enum": ["2.4G", "5G", "6G"]
					},
					"tx_rate": {"type": "integer"},
					"rx_rate": {"type": "integer"},
					"tx_bytes": {"type": "integer"},
					"rx_bytes": {"type": "integer"},
					"tx_packets": {"type": "integer"},
					"rx_packets": {"type": "integer"},
					"signal_strength": {"type": "integer"},
					"noise_level": {"type": "integer"},
					"connection_time": {"type": "integer"},
					"capabilities": {
						"type": "array",
						"items": {"type": "string"}
					}
				}
			}
		}
	}
}`

// Device Identity Schema - for device identity management
const deviceIdentitySchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"title": "RTK Device Identity Message",
	"type": "object",
	"required": ["schema", "timestamp", "mac_address"],
	"properties": {
		"schema": {
			"type": "string",
			"const": "device.identity/1.0"
		},
		"timestamp": {
			"type": "integer",
			"description": "Unix timestamp in milliseconds"
		},
		"mac_address": {
			"type": "string",
			"pattern": "^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"
		},
		"friendly_name": {"type": "string"},
		"device_type": {
			"type": "string",
			"enum": ["phone", "laptop", "tv", "iot", "router", "ap", "switch", "tablet", "gaming", "appliance"]
		},
		"manufacturer": {"type": "string"},
		"model": {"type": "string"},
		"location": {"type": "string"},
		"owner": {"type": "string"},
		"category": {
			"type": "string",
			"enum": ["personal", "shared", "infrastructure"]
		},
		"tags": {
			"type": "array",
			"items": {"type": "string"}
		},
		"auto_detected": {"type": "boolean"},
		"detection_rules": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"rule_id": {"type": "string"},
					"rule_name": {"type": "string"},
					"matched_field": {"type": "string"},
					"matched_value": {"type": "string"},
					"confidence": {"type": "number"},
					"matched_at": {"type": "integer"}
				}
			}
		},
		"confidence": {"type": "number"},
		"first_seen": {"type": "integer"},
		"last_seen": {"type": "integer"},
		"notes": {"type": "string"}
	}
}`

// Network Diagnostics Schema - for network testing and diagnostics
const networkDiagnosticsSchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"title": "RTK Network Diagnostics Message",
	"type": "object",
	"required": ["schema", "timestamp", "device_id"],
	"properties": {
		"schema": {
			"type": "string",
			"const": "diagnostics.network/1.0"
		},
		"timestamp": {
			"type": "integer",
			"description": "Unix timestamp in milliseconds"
		},
		"device_id": {
			"type": "string",
			"description": "Device performing the diagnostics"
		},
		"speed_test": {
			"type": "object",
			"properties": {
				"download_mbps": {"type": "number"},
				"upload_mbps": {"type": "number"},
				"jitter": {"type": "number"},
				"packet_loss": {"type": "number"},
				"test_server": {"type": "string"},
				"test_duration": {"type": "integer"},
				"status": {
					"type": "string",
					"enum": ["running", "completed", "failed"]
				},
				"error": {"type": "string"}
			}
		},
		"latency_test": {
			"type": "object",
			"properties": {
				"targets": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"target": {"type": "string"},
							"type": {
								"type": "string",
								"enum": ["gateway", "dns", "external"]
							},
							"avg_latency": {"type": "number"},
							"min_latency": {"type": "number"},
							"max_latency": {"type": "number"},
							"packet_loss": {"type": "number"},
							"packets_sent": {"type": "integer"},
							"packets_received": {"type": "integer"},
							"status": {
								"type": "string",
								"enum": ["success", "failed", "timeout"]
							}
						}
					}
				},
				"overall_status": {"type": "string"}
			}
		},
		"wan_test": {
			"type": "object",
			"properties": {
				"isp_gateway_reachable": {"type": "boolean"},
				"isp_gateway_latency": {"type": "number"},
				"external_dns_latency": {"type": "number"},
				"wan_connected": {"type": "boolean"},
				"public_ip": {"type": "string"},
				"isp_info": {"type": "string"}
			}
		},
		"connectivity_test": {
			"type": "object",
			"properties": {
				"internal_reachability": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"device_id": {"type": "string"},
							"ip_address": {"type": "string"},
							"reachable": {"type": "boolean"},
							"latency": {"type": "number"},
							"method": {
								"type": "string",
								"enum": ["ping", "tcp_connect"]
							}
						}
					}
				},
				"external_reachability": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"target": {"type": "string"},
							"type": {
								"type": "string",
								"enum": ["dns", "web", "speedtest_server"]
							},
							"reachable": {"type": "boolean"},
							"latency": {"type": "number"},
							"http_status": {"type": "integer"}
						}
					}
				}
			}
		}
	}
}`

// QoS Information Schema - for quality of service and traffic analysis
const qosInfoSchema = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"title": "RTK QoS Information Message",
	"type": "object",
	"required": ["schema", "timestamp", "device_id"],
	"properties": {
		"schema": {
			"type": "string",
			"const": "telemetry.qos/1.0"
		},
		"timestamp": {
			"type": "integer",
			"description": "Unix timestamp in milliseconds"
		},
		"device_id": {
			"type": "string",
			"description": "Device reporting QoS information"
		},
		"enabled": {"type": "boolean"},
		"bandwidth_caps": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"rule_id": {"type": "string"},
					"target": {"type": "string"},
					"upload_limit": {"type": "integer"},
					"download_limit": {"type": "integer"},
					"priority": {"type": "integer"},
					"enabled": {"type": "boolean"}
				}
			}
		},
		"traffic_shaping": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"rule_id": {"type": "string"},
					"protocol": {
						"type": "string",
						"enum": ["tcp", "udp", "icmp"]
					},
					"ports": {
						"type": "array",
						"items": {"type": "integer"}
					},
					"action": {
						"type": "string",
						"enum": ["allow", "block", "throttle", "prioritize"]
					},
					"priority": {"type": "integer"},
					"bytes_matched": {"type": "integer"}
				}
			}
		},
		"priority_queues": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"queue_id": {"type": "string"},
					"priority": {"type": "integer"},
					"bandwidth_percent": {"type": "number"},
					"current_load": {"type": "number"},
					"packets_queued": {"type": "integer"},
					"packets_dropped": {"type": "integer"}
				}
			}
		},
		"active_connections": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"connection_id": {"type": "string"},
					"source_ip": {"type": "string"},
					"source_port": {"type": "integer"},
					"dest_ip": {"type": "string"},
					"dest_port": {"type": "integer"},
					"protocol": {"type": "string"},
					"state": {"type": "string"},
					"bytes_sent": {"type": "integer"},
					"bytes_received": {"type": "integer"},
					"duration": {"type": "integer"},
					"queue_id": {"type": "string"}
				}
			}
		},
		"traffic_stats": {
			"type": "object",
			"properties": {
				"total_bandwidth": {"type": "number"},
				"used_bandwidth": {"type": "number"},
				"device_traffic": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"device_id": {"type": "string"},
							"device_mac": {"type": "string"},
							"friendly_name": {"type": "string"},
							"upload_mbps": {"type": "number"},
							"download_mbps": {"type": "number"},
							"total_bytes": {"type": "integer"},
							"active_connections": {"type": "integer"},
							"bandwidth_percent": {"type": "number"}
						}
					}
				},
				"top_talkers": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"device_id": {"type": "string"},
							"friendly_name": {"type": "string"},
							"total_mbps": {"type": "number"},
							"traffic_type": {
								"type": "string",
								"enum": ["upload", "download", "total"]
							},
							"rank": {"type": "integer"}
						}
					}
				},
				"protocol_distribution": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"protocol": {"type": "string"},
							"total_mbps": {"type": "number"},
							"percentage": {"type": "number"},
							"packet_count": {"type": "integer"}
						}
					}
				}
			}
		}
	}
}`
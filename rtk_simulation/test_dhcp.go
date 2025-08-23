package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type DHCPPoolConfig struct {
	StartIP string   `yaml:"start_ip"`
	EndIP   string   `yaml:"end_ip"`
	Gateway string   `yaml:"gateway"`
	DNS     []string `yaml:"dns"`
}

type NetworkSettings struct {
	DHCPPool DHCPPoolConfig `yaml:"dhcp_pool"`
}

type Config struct {
	Network NetworkSettings `yaml:"network"`
}

func main() {
	data, err := os.ReadFile("configs/home_basic.yaml")
	if err != nil {
		panic(err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("StartIP: %s\n", cfg.Network.DHCPPool.StartIP)
	fmt.Printf("EndIP: %s\n", cfg.Network.DHCPPool.EndIP)
	fmt.Printf("Gateway: %s\n", cfg.Network.DHCPPool.Gateway)
	fmt.Printf("DNS: %v\n", cfg.Network.DHCPPool.DNS)
}

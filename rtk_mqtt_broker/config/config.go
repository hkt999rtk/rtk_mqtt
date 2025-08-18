package config

import (
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Security SecurityConfig `yaml:"security"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	Port        int    `yaml:"port"`
	Host        string `yaml:"host"`
	MaxClients  int    `yaml:"max_clients"`
	EnableStats bool   `yaml:"enable_stats"`
}

type SecurityConfig struct {
	EnableAuth bool                `yaml:"enable_auth"`
	Users      []map[string]string `yaml:"users"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func LoadConfig(configPath string) (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:        1883,
			Host:        "0.0.0.0",
			MaxClients:  1000,
			EnableStats: true,
		},
		Security: SecurityConfig{
			EnableAuth: false,
			Users:      []map[string]string{},
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
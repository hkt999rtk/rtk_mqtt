package utils

import (
	"encoding/json"
	"os"
)

// Config represents the main application configuration
type Config struct {
	Server       ServerConfig            `json:"server"`
	LLMProviders []LLMProviderConfig     `json:"llm_providers"`
	MCPServers   []MCPServerConfig       `json:"mcp_servers"`
	WeatherAPI   WeatherAPIConfig        `json:"weatherapi"`
	HTTPServer   HTTPServerConfig        `json:"http_server"`
	
	// Legacy support
	LMStudio     LMStudioConfig          `json:"lmstudio,omitempty"`
}

type ServerConfig struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type LMStudioConfig struct {
	BaseURL     string  `json:"base_url"`
	APIKey      string  `json:"api_key,omitempty"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type WeatherAPIConfig struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

type HTTPServerConfig struct {
	Port    int    `json:"port"`
	Host    string `json:"host"`
	Enabled bool   `json:"enabled"`
}

type MCPServerConfig struct {
	Name    string `json:"name"`
	Type    string `json:"type"`    // local, http, websocket
	URL     string `json:"url,omitempty"`
	Enabled bool   `json:"enabled"`
}

// LoadConfig loads configuration from the specified file path
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	// Set default values
	if cfg.LMStudio.BaseURL == "" {
		cfg.LMStudio.BaseURL = "http://localhost:1234"
	}
	if cfg.LMStudio.Model == "" {
		cfg.LMStudio.Model = "local-model"
	}
	if cfg.LMStudio.MaxTokens == 0 {
		cfg.LMStudio.MaxTokens = 2048
	}
	if cfg.LMStudio.Temperature == 0 {
		cfg.LMStudio.Temperature = 0.7
	}

	return &cfg, nil
}

// GetDefaultConfig returns default configuration
func GetDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Name:        "CES2026 MCP Server",
			Version:     "1.0.0",
			Description: "MCP Server with multi-provider LLM support",
		},
		LLMProviders: []LLMProviderConfig{
			{
				Name:    "lmstudio",
				Type:    "lmstudio",
				BaseURL: "http://localhost:1234",
				Models:  []string{"local-model"},
				Enabled: true,
			},
		},
		MCPServers: []MCPServerConfig{
			{
				Name:    "local",
				Type:    "local",
				Enabled: true,
			},
		},
		WeatherAPI: WeatherAPIConfig{
			APIKey:  "47e95e09c9a3420d846103147233103",
			BaseURL: "https://api.weatherapi.com/v1",
		},
		HTTPServer: HTTPServerConfig{
			Port:    8080,
			Host:    "0.0.0.0",
			Enabled: true,
		},
		// Legacy support
		LMStudio: LMStudioConfig{
			BaseURL:     "http://localhost:1234",
			Model:       "local-model",
			MaxTokens:   2048,
			Temperature: 0.7,
		},
	}
}
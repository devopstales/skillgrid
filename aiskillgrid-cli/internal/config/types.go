package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type ToolsConfig struct {
	Agents []string `yaml:"agents"`
	Tools  []string `yaml:"tools"`
}

type MCPServerConfig struct {
	Type    string   `yaml:"type"`
	URL     string   `yaml:"url,omitempty"`
	Command []string `yaml:"command,omitempty"`
}

type MCPConfig struct {
	Servers map[string]MCPServerConfig `yaml:"servers"`
}

func LoadToolsYAML(path string) (*ToolsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ToolsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func LoadMCPYAML(path string) (*MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg MCPConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

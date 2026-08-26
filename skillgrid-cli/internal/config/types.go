package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type McpServer struct {
	Type    string   `yaml:"type"`
	URL     string   `yaml:"url,omitempty"`
	Command []string `yaml:"command,omitempty"`
	Enabled bool     `yaml:"enabled,omitempty"`
}

type ToolsConfig struct {
	Agents []string `yaml:"agents"`
	Tools  []string `yaml:"tools"`
}

type SkillEntry struct {
	Repo  string `yaml:"repo"`
	Skill string `yaml:"skill"`
	Agent string `yaml:"agent"`
}

type SkillsConfig struct {
	Skills []SkillEntry `yaml:"skills"`
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

func LoadSkillsYAML(path string) (*SkillsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SkillsConfig{}, nil
		}
		return nil, err
	}
	var cfg SkillsConfig
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

type indexingProfile struct {
	Profile string `yaml:"profile"`
}

// LoadIndexingProfile returns the profile field from config.d/indexing.yaml.
// Missing file yields an empty profile without error.
func LoadIndexingProfile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var cfg indexingProfile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	return cfg.Profile, nil
}

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Tunnels map[string]Tunnel `yaml:"tunnels"`
}

type Tunnel struct {
	Host         string   `yaml:"host"`
	Ports        []string `yaml:"ports"`
	User         string   `yaml:"user,omitempty"`
	IdentityFile string   `yaml:"identity_file,omitempty"`
}

func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".tunnrc")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("~/.tunnrc not found. Please create one!")
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

func (c *Config) FilterTunnels(names []string) map[string]Tunnel {
	if len(names) == 0 {
		return c.Tunnels
	}

	filtered := make(map[string]Tunnel)
	for _, name := range names {
		if tunnel, exists := c.Tunnels[name]; exists {
			filtered[name] = tunnel
		}
	}
	return filtered
}
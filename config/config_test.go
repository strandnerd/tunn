package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", homeDir)

	configContent := `
tunnels:
  api:
    host: myserver
    ports:
      - 3000:3000
      - 4000:4001
    user: apiuser
    identity_file: ~/.ssh/id_rsa
  db:
    host: database
    ports:
      - 3306:3306
`

	configPath := filepath.Join(tmpDir, ".tunnrc")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Tunnels) != 2 {
		t.Errorf("Expected 2 tunnels, got %d", len(cfg.Tunnels))
	}

	apiTunnel, exists := cfg.Tunnels["api"]
	if !exists {
		t.Fatal("API tunnel not found")
	}

	if apiTunnel.Host != "myserver" {
		t.Errorf("Expected host 'myserver', got %s", apiTunnel.Host)
	}

	if len(apiTunnel.Ports) != 2 {
		t.Errorf("Expected 2 ports for API tunnel, got %d", len(apiTunnel.Ports))
	}

	if apiTunnel.User != "apiuser" {
		t.Errorf("Expected user 'apiuser', got %s", apiTunnel.User)
	}

	if apiTunnel.IdentityFile != "~/.ssh/id_rsa" {
		t.Errorf("Expected identity file '~/.ssh/id_rsa', got %s", apiTunnel.IdentityFile)
	}

	dbTunnel, exists := cfg.Tunnels["db"]
	if !exists {
		t.Fatal("DB tunnel not found")
	}

	if dbTunnel.User != "" {
		t.Errorf("Expected empty user for DB tunnel, got %s", dbTunnel.User)
	}

	if dbTunnel.IdentityFile != "" {
		t.Errorf("Expected empty identity file for DB tunnel, got %s", dbTunnel.IdentityFile)
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", homeDir)

	_, err := Load()
	if err == nil {
		t.Fatal("Expected error for missing config file")
	}

	if err.Error() != "~/.tunnrc not found. Please create one!" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestFilterTunnels(t *testing.T) {
	cfg := &Config{
		Tunnels: map[string]Tunnel{
			"api": {
				Host:  "server1",
				Ports: []string{"3000:3000"},
			},
			"db": {
				Host:  "server2",
				Ports: []string{"5432:5432"},
			},
			"cache": {
				Host:  "server3",
				Ports: []string{"6379:6379"},
			},
		},
	}

	tests := []struct {
		name     string
		filter   []string
		expected int
	}{
		{"No filter", []string{}, 3},
		{"Single filter", []string{"api"}, 1},
		{"Multiple filters", []string{"api", "db"}, 2},
		{"Non-existent filter", []string{"nonexistent"}, 0},
		{"Mixed filters", []string{"api", "nonexistent", "cache"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := cfg.FilterTunnels(tt.filter)
			if len(filtered) != tt.expected {
				t.Errorf("Expected %d tunnels, got %d", tt.expected, len(filtered))
			}

			for _, name := range tt.filter {
				if _, exists := cfg.Tunnels[name]; exists {
					if _, filteredExists := filtered[name]; !filteredExists {
						t.Errorf("Expected tunnel %s to be in filtered result", name)
					}
				}
			}
		})
	}
}
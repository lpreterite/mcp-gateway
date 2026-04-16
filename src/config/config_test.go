package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPathsHelp(t *testing.T) {
	help := ConfigPathsHelp()
	if help == "" {
		t.Error("ConfigPathsHelp() returned empty string")
	}
	// 验证返回的路径格式正确
	if filepath.Ext(help) != ".json" {
		t.Errorf("ConfigPathsHelp() should return JSON file path, got: %s", help)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	configContent := `{
		"gateway": {
			"host": "127.0.0.1",
			"port": 9999
		},
		"pool": {
			"minConnections": 2,
			"maxConnections": 10,
			"acquireTimeout": 5000,
			"idleTimeout": 30000
		},
		"servers": [
			{
				"name": "test-server",
				"type": "local",
				"enabled": true,
				"command": ["echo", "hello"]
			}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Gateway.Host != "127.0.0.1" {
		t.Errorf("Expected Gateway.Host=127.0.0.1, got %s", cfg.Gateway.Host)
	}

	if cfg.Gateway.Port != 9999 {
		t.Errorf("Expected Gateway.Port=9999, got %d", cfg.Gateway.Port)
	}

	if cfg.Pool.MinConnections != 2 {
		t.Errorf("Expected Pool.MinConnections=2, got %d", cfg.Pool.MinConnections)
	}

	if len(cfg.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(cfg.Servers))
	}

	if cfg.Servers[0].Name != "test-server" {
		t.Errorf("Expected server name 'test-server', got %s", cfg.Servers[0].Name)
	}
}

func TestServerConfigEnabled(t *testing.T) {
	cfg := &ServerConfig{
		Name:    "test",
		Enabled: true,
		Command: []string{"echo", "test"},
	}

	if !cfg.Enabled {
		t.Error("ServerConfig.Enabled should be true")
	}

	if len(cfg.Command) != 2 {
		t.Errorf("Expected Command length 2, got %d", len(cfg.Command))
	}
}

func TestPoolConfigDefaults(t *testing.T) {
	cfg := &PoolConfig{
		MinConnections: 1,
		MaxConnections: 5,
	}

	if cfg.MinConnections < 0 {
		t.Error("MinConnections should not be negative")
	}

	if cfg.MaxConnections < cfg.MinConnections {
		t.Error("MaxConnections should be >= MinConnections")
	}
}

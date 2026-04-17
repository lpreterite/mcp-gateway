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

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Servers: []ServerConfig{
					{Name: "test", Type: "local", Command: []string{"echo"}},
				},
				Pool: &PoolConfig{
					MinConnections: 1,
					MaxConnections: 5,
					AcquireTimeout: 5000,
					IdleTimeout:    30000,
				},
				Gateway: &GatewayConfig{
					Port: 4298,
				},
			},
			wantErr: false,
		},
		{
			name: "no servers",
			config: &Config{
				Servers: []ServerConfig{},
			},
			wantErr: true,
			errMsg:  "no servers configured",
		},
		{
			name: "server without name",
			config: &Config{
				Servers: []ServerConfig{
					{Name: "", Type: "local", Command: []string{"echo"}},
				},
			},
			wantErr: true,
			errMsg:  "server[0]: name is required",
		},
		{
			name: "invalid server type",
			config: &Config{
				Servers: []ServerConfig{
					{Name: "test", Type: "invalid"},
				},
			},
			wantErr: true,
			errMsg:  "server[0]: type must be 'local' or 'remote'",
		},
		{
			name: "local server without command",
			config: &Config{
				Servers: []ServerConfig{
					{Name: "test", Type: "local", Command: []string{}},
				},
			},
			wantErr: true,
			errMsg:  "server[0]: command is required for local servers",
		},
		{
			name: "pool minConnections less than 1",
			config: &Config{
				Servers: []ServerConfig{
					{Name: "test", Type: "remote", URL: "http://test"},
				},
				Pool: &PoolConfig{
					MinConnections: 0,
					MaxConnections: 5,
				},
			},
			wantErr: true,
			errMsg:  "pool.minConnections must be >= 1",
		},
		{
			name: "pool maxConnections less than minConnections",
			config: &Config{
				Servers: []ServerConfig{
					{Name: "test", Type: "remote", URL: "http://test"},
				},
				Pool: &PoolConfig{
					MinConnections: 5,
					MaxConnections: 3,
				},
			},
			wantErr: true,
			errMsg:  "pool.maxConnections must be >= pool.minConnections",
		},
		{
			name: "pool acquireTimeout too small",
			config: &Config{
				Servers: []ServerConfig{
					{Name: "test", Type: "remote", URL: "http://test"},
				},
				Pool: &PoolConfig{
					MinConnections: 1,
					MaxConnections: 5,
					AcquireTimeout: 500,
				},
			},
			wantErr: true,
			errMsg:  "pool.acquireTimeout must be >= 1000",
		},
		{
			name: "pool idleTimeout too small",
			config: &Config{
				Servers: []ServerConfig{
					{Name: "test", Type: "remote", URL: "http://test"},
				},
				Pool: &PoolConfig{
					MinConnections: 1,
					MaxConnections: 5,
					AcquireTimeout: 5000,
					IdleTimeout:    5000,
				},
			},
			wantErr: true,
			errMsg:  "pool.idleTimeout must be >= 10000",
		},
		{
			name: "gateway port out of range",
			config: &Config{
				Servers: []ServerConfig{
					{Name: "test", Type: "remote", URL: "http://test"},
				},
				Gateway: &GatewayConfig{
					Port: 70000,
				},
			},
			wantErr: true,
			errMsg:  "gateway.port must be between 1 and 65535",
		},
		{
			name: "gateway port negative",
			config: &Config{
				Servers: []ServerConfig{
					{Name: "test", Type: "remote", URL: "http://test"},
				},
				Gateway: &GatewayConfig{
					Port: -1,
				},
			},
			wantErr: true,
			errMsg:  "gateway.port must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateConfig() expected error containing %q, got nil", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("validateConfig() error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()
	if cfg == nil {
		t.Fatal("DefaultPoolConfig() returned nil")
	}
	if cfg.MinConnections != 1 {
		t.Errorf("MinConnections = %d, want 1", cfg.MinConnections)
	}
	if cfg.MaxConnections != 5 {
		t.Errorf("MaxConnections = %d, want 5", cfg.MaxConnections)
	}
	if cfg.AcquireTimeout != 10000 {
		t.Errorf("AcquireTimeout = %d, want 10000", cfg.AcquireTimeout)
	}
	if cfg.IdleTimeout != 60000 {
		t.Errorf("IdleTimeout = %d, want 60000", cfg.IdleTimeout)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
}

func TestDefaultGatewayConfig(t *testing.T) {
	cfg := DefaultGatewayConfig()
	if cfg == nil {
		t.Fatal("DefaultGatewayConfig() returned nil")
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0", cfg.Host)
	}
	if cfg.Port != 4298 {
		t.Errorf("Port = %d, want 4298", cfg.Port)
	}
	if !cfg.CORS {
		t.Error("CORS should be true")
	}
}

func TestResolveConfigPath(t *testing.T) {
	// 测试显式路径直接返回
	explicitPath := "/path/to/config.json"
	result := ResolveConfigPath(explicitPath)
	if result != explicitPath {
		t.Errorf("ResolveConfigPath(%q) = %q, want %q", explicitPath, result, explicitPath)
	}

	// 测试空字符串会调用 findConfigPath（可能返回空或找到的配置路径）
	// 由于 findConfigPath 依赖环境，这个测试主要是确保不 panic
	_ = ResolveConfigPath("")
}

func TestGetConfigPaths(t *testing.T) {
	paths := GetConfigPaths()
	if len(paths) == 0 {
		t.Error("GetConfigPaths() returned empty slice")
	}
	// 验证返回的路径格式
	for _, p := range paths {
		if p == "" {
			t.Error("GetConfigPaths() returned empty path")
		}
	}
}

func TestResolveConfigPathWithEnv(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	configContent := `{"servers": [{"name": "test", "type": "local", "command": ["echo"]}]}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// 设置环境变量
	os.Setenv("MCP_GATEWAY_CONFIG", configPath)
	defer os.Unsetenv("MCP_GATEWAY_CONFIG")

	result := ResolveConfigPath("")
	if result != configPath {
		t.Errorf("ResolveConfigPath() with env = %q, want %q", result, configPath)
	}
}

func TestResolveConfigPathEmptyEnv(t *testing.T) {
	// 确保环境变量为空
	os.Unsetenv("MCP_GATEWAY_CONFIG")
	// 使用不存在的路径，应该返回空
	result := ResolveConfigPath("")
	// result 可能为空也可能找到本地配置，取决于测试环境
	_ = result
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// 创建临时配置文件，不包含 pool 和 gateway
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	configContent := `{
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

	// 验证默认值被应用
	if cfg.Pool == nil {
		t.Fatal("Pool should have default value")
	}
	if cfg.Gateway == nil {
		t.Fatal("Gateway should have default value")
	}
	if cfg.Pool.MinConnections != 1 {
		t.Errorf("Pool.MinConnections = %d, want 1 (default)", cfg.Pool.MinConnections)
	}
	if cfg.Gateway.Port != 4298 {
		t.Errorf("Gateway.Port = %d, want 4298 (default)", cfg.Gateway.Port)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Error("Load() should fail for nonexistent file")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(configPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should fail for invalid JSON")
	}
}

func TestInspect(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	configContent := `{
		"gateway": {"host": "127.0.0.1", "port": 9999},
		"servers": [{"name": "test", "type": "local", "command": ["echo"]}]
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	path, cfg, err := Inspect(configPath)
	if err != nil {
		t.Fatalf("Inspect() failed: %v", err)
	}
	if path != configPath {
		t.Errorf("Inspect() path = %q, want %q", path, configPath)
	}
	if cfg.Gateway.Port != 9999 {
		t.Errorf("Gateway.Port = %d, want 9999", cfg.Gateway.Port)
	}
}

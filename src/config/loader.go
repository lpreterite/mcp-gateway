package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// DefaultConfigPath 默认配置路径
const (
	DefaultConfigDir  = ".config/mcp-gateway"
	DefaultConfigFile = "config.json"
)

// DefaultPoolConfig 返回默认连接池配置
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MinConnections: 1,
		MaxConnections: 5,
		AcquireTimeout: 10000,
		IdleTimeout:    60000,
		MaxRetries:     3,
	}
}

// DefaultGatewayConfig 返回默认网关配置
func DefaultGatewayConfig() *GatewayConfig {
	return &GatewayConfig{
		Host: "0.0.0.0",
		Port: 4298,
		CORS: true,
	}
}

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 自动查找配置
		configPath = findConfigPath()
		if configPath == "" {
			return nil, fmt.Errorf("config file not found. Please create one of:\n" +
				"  - ~/.config/mcp-gateway/config.json (global install)\n" +
				"  - ./config/servers.json (local development)\n" +
				"  - Or set MCP_GATEWAY_CONFIG environment variable")
		}
		v.SetConfigFile(configPath)
	}

	// 设置配置类型
	v.SetConfigType("json")

	// 读取配置
	slog.Info("Loading config from", "path", configPath)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 应用默认值
	if cfg.Pool == nil {
		cfg.Pool = DefaultPoolConfig()
	}
	if cfg.Gateway == nil {
		cfg.Gateway = DefaultGatewayConfig()
	}

	// 验证配置
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// findConfigPath 自动查找配置文件路径
func findConfigPath() string {
	// 1. MCP_GATEWAY_CONFIG 环境变量
	if envPath := os.Getenv("MCP_GATEWAY_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// 2. ~/.config/mcp-gateway/config.json
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalConfig := filepath.Join(homeDir, DefaultConfigDir, DefaultConfigFile)
		if _, err := os.Stat(globalConfig); err == nil {
			return globalConfig
		}
	}

	// 3. Homebrew etc 目录 (macOS)
	if runtime.GOOS == "darwin" {
		for _, prefix := range []string{"/opt/homebrew", "/usr/local"} {
			brewConfig := filepath.Join(prefix, "etc/mcp-gateway", DefaultConfigFile)
			if _, err := os.Stat(brewConfig); err == nil {
				return brewConfig
			}
		}
	}

	// 4. ./config/servers.json (本地开发)
	localConfig := filepath.Join("config", "servers.json")
	if _, err := os.Stat(localConfig); err == nil {
		return localConfig
	}

	return ""
}

// validateConfig 验证配置
func validateConfig(cfg *Config) error {
	if len(cfg.Servers) == 0 {
		return fmt.Errorf("no servers configured")
	}

	for i, server := range cfg.Servers {
		if server.Name == "" {
			return fmt.Errorf("server[%d]: name is required", i)
		}
		if server.Type != "local" && server.Type != "remote" {
			return fmt.Errorf("server[%d]: type must be 'local' or 'remote'", i)
		}
		if server.Type == "local" && len(server.Command) == 0 {
			return fmt.Errorf("server[%d]: command is required for local servers", i)
		}
	}

	if cfg.Pool != nil {
		if cfg.Pool.MinConnections < 1 {
			return fmt.Errorf("pool.minConnections must be >= 1")
		}
		if cfg.Pool.MaxConnections < cfg.Pool.MinConnections {
			return fmt.Errorf("pool.maxConnections must be >= pool.minConnections")
		}
		if cfg.Pool.AcquireTimeout < 1000 {
			return fmt.Errorf("pool.acquireTimeout must be >= 1000")
		}
		if cfg.Pool.IdleTimeout < 10000 {
			return fmt.Errorf("pool.idleTimeout must be >= 10000")
		}
	}

	if cfg.Gateway != nil {
		if cfg.Gateway.Port < 1 || cfg.Gateway.Port > 65535 {
			return fmt.Errorf("gateway.port must be between 1 and 65535")
		}
	}

	return nil
}

// GetConfigPaths 返回所有可能的配置路径（用于显示帮助信息）
func GetConfigPaths() []string {
	var paths []string

	// MCP_GATEWAY_CONFIG 环境变量
	if envPath := os.Getenv("MCP_GATEWAY_CONFIG"); envPath != "" {
		paths = append(paths, fmt.Sprintf("  - MCP_GATEWAY_CONFIG: %s", envPath))
	}

	// ~/.config/mcp-gateway/config.json
	if homeDir, err := os.UserHomeDir(); err == nil {
		globalConfig := filepath.Join(homeDir, DefaultConfigDir, DefaultConfigFile)
		paths = append(paths, fmt.Sprintf("  - %s", globalConfig))
	}

	// Homebrew etc 目录 (macOS)
	if runtime.GOOS == "darwin" {
		for _, prefix := range []string{"/opt/homebrew", "/usr/local"} {
			brewConfig := filepath.Join(prefix, "etc/mcp-gateway", DefaultConfigFile)
			paths = append(paths, fmt.Sprintf("  - %s (Homebrew)", brewConfig))
		}
	}

	// ./config/servers.json
	wd, _ := os.Getwd()
	localConfig := filepath.Join(wd, "config", "servers.json")
	paths = append(paths, fmt.Sprintf("  - %s", localConfig))

	return paths
}

// String 返回配置路径优先级说明
func ConfigPathsHelp() string {
	return "Config file lookup order:\n" + strings.Join(GetConfigPaths(), "\n")
}

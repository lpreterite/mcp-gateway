package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	_, cfg, err := load(configPath, true)
	return cfg, err
}

// Inspect 检查配置文件并返回解析结果，不输出日志。
func Inspect(configPath string) (string, *Config, error) {
	return load(configPath, false)
}

// ResolveConfigPath 返回显式路径或自动探测到的配置路径。
func ResolveConfigPath(configPath string) string {
	if configPath != "" {
		return configPath
	}
	return findConfigPath()
}

func load(configPath string, enableLog bool) (string, *Config, error) {
	configPath = ResolveConfigPath(configPath)
	if configPath == "" {
		return "", nil, fmt.Errorf("config file not found. Please create one of:\n" +
			"  - ~/.config/mcp-gateway/config.json (global install)\n" +
			"  - ./config/servers.json (local development)\n" +
			"  - Or set MCP_GATEWAY_CONFIG environment variable")
	}

	if enableLog {
		slog.Info("Loading config from", "path", configPath)
	}

	// 直接读取文件并使用 json.Unmarshal 以保持大小写
	data, err := os.ReadFile(configPath)
	if err != nil {
		return configPath, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return configPath, nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	if enableLog {
		for _, s := range cfg.Servers {
			if len(s.Env) > 0 {
				keys := make([]string, 0, len(s.Env))
				for k := range s.Env {
					keys = append(keys, k)
				}
				slog.Info("Server env loaded (standard JSON)", "server", s.Name, "keys", keys)
			}
		}
	}

	if cfg.Pool == nil {
		cfg.Pool = DefaultPoolConfig()
	}
	if cfg.Gateway == nil {
		cfg.Gateway = DefaultGatewayConfig()
	}

	if err := validateConfig(&cfg); err != nil {
		return configPath, nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return configPath, &cfg, nil
}

// findConfigPath 自动查找配置文件路径，遵循 Unix 惯例优先级
func findConfigPath() string {
	// 1. 显式环境变量优先级最高
	if envPath := os.Getenv("MCP_GATEWAY_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// 2. XDG 标准 / 用户家目录 (Unix 惯例)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userConfig := filepath.Join(homeDir, DefaultConfigDir, DefaultConfigFile)
		if _, err := os.Stat(userConfig); err == nil {
			return userConfig
		}
	}

	// 3. 本地开发路径 (./config/...)
	localConfig := filepath.Join("config", "servers.json")
	if _, err := os.Stat(localConfig); err == nil {
		return localConfig
	}

	// 4. 系统全局配置 / Homebrew 备份 (最后手段)
	if runtime.GOOS == "darwin" {
		for _, prefix := range []string{"/opt/homebrew", "/usr/local"} {
			brewConfig := filepath.Join(prefix, "etc/mcp-gateway", DefaultConfigFile)
			if _, err := os.Stat(brewConfig); err == nil {
				return brewConfig
			}
		}
	} else if runtime.GOOS == "linux" {
		linuxConfig := filepath.Join("/etc/mcp-gateway", DefaultConfigFile)
		if _, err := os.Stat(linuxConfig); err == nil {
			return linuxConfig
		}
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

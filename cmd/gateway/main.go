package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/packy/mcp-gateway/internal/config"
	"github.com/packy/mcp-gateway/internal/gateway"
	"github.com/packy/mcp-gateway/internal/stdio"
	"github.com/urfave/cli/v2"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
)

func main() {
	// 设置 slog 格式
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// CLI 应用
	app := &cli.App{
		Name:    "mcp-gateway",
		Version: version,
		Usage:   "MCP Gateway - Centralized MCP server management with connection pooling and HTTP/SSE transport",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to config file",
			},
			&cli.StringFlag{
				Name:  "host",
				Usage: "Listen address",
				Value: "0.0.0.0",
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Listen port",
				Value:   4298,
			},
			&cli.BoolFlag{
				Name:  "stdio",
				Usage: "Run in stdio mode (for Claude Desktop)",
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "Log level (debug, info, warn, error)",
				Value: "info",
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	// 设置日志级别
	logLevel := c.String("log-level")
	var slogLevel slog.Level
	switch logLevel {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	// 重新配置 slog
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slogLevel,
	})))

	// 加载配置
	configPath := c.String("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		// 如果没有配置文件，显示配置路径帮助
		if configPath == "" {
			slog.Error("Failed to load config",
				"error", err,
				"hint", config.ConfigPathsHelp(),
			)
		} else {
			slog.Error("Failed to load config",
				"error", err,
				"path", configPath,
			)
		}
		return fmt.Errorf("config error: %w", err)
	}

	// 覆盖配置中的 host/port
	if host := c.String("host"); host != "" {
		cfg.Gateway.Host = host
	}
	if port := c.Int("port"); port > 0 {
		cfg.Gateway.Port = port
	}

	// Stdio 模式
	if c.Bool("stdio") {
		return runStdioMode(cfg)
	}

	// HTTP/SSE 模式
	return runServerMode(cfg)
}

func runServerMode(cfg *config.Config) error {
	slog.Info("Starting MCP Gateway",
		"version", version,
		"buildTime", buildTime,
	)

	server := gateway.NewServer(cfg)
	return server.Start()
}

func runStdioMode(cfg *config.Config) error {
	slog.Info("Starting MCP Gateway in stdio mode",
		"version", version,
	)

	server := stdio.NewServer(cfg)
	return server.Start()
}

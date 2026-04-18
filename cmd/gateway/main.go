package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	svc "github.com/kardianos/service"
	"github.com/lpreterite/mcp-gateway/src/config"
	"github.com/lpreterite/mcp-gateway/src/gateway"
	"github.com/lpreterite/mcp-gateway/src/gwservice"
	"github.com/lpreterite/mcp-gateway/src/stdio"
	"github.com/lpreterite/mcp-gateway/src/utils"
	"github.com/urfave/cli/v2"
)

var (
	version   = "1.2.2"
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
		Commands: []*cli.Command{
			{
				Name:  "config",
				Usage: "Config management (info, init)",
				Subcommands: []*cli.Command{
					{
						Name:  "info",
						Usage: "Show current config status and lookup paths",
						Action: func(c *cli.Context) error {
							fmt.Println("MCP Gateway Config Status:")
							active := config.ConfigPathsHelp()
							fmt.Println(active)
							return nil
						},
					},
					{
						Name:  "init",
						Usage: "Initialize user config (~/.config/mcp-gateway/config.json)",
						Action: func(c *cli.Context) error {
							homeDir, _ := os.UserHomeDir()
							targetDir := filepath.Join(homeDir, ".config/mcp-gateway")
							targetFile := filepath.Join(targetDir, "config.json")

							if _, err := os.Stat(targetFile); err == nil {
								fmt.Printf("Config already exists at %s\n", targetFile)
								return nil
							}

							if err := os.MkdirAll(targetDir, 0755); err != nil {
								return fmt.Errorf("failed to create config directory: %w", err)
							}
							defaultConfig := `{
  "gateway": {
    "host": "0.0.0.0",
    "port": 4298,
    "cors": true
  },
  "pool": {
    "minConnections": 1,
    "maxConnections": 5,
    "acquireTimeout": 5000,
    "idleTimeout": 30000
  },
  "servers": [
    {
      "name": "example",
      "type": "local",
      "command": ["echo", "hello"],
      "enabled": true,
      "poolSize": 1
    }
  ],
  "mapping": {}
}`
							err := os.WriteFile(targetFile, []byte(defaultConfig), 0644)
							if err != nil {
								return err
							}
							fmt.Printf("✓ Config initialized at %s\n", targetFile)
							return nil
						},
					},
				},
			},
			{
				Name:  "service",
				Usage: "Service management (install, uninstall, start, stop, restart, status)",
				Subcommands: []*cli.Command{
					{
						Name:  "install",
						Usage: "Install as a system service",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "config",
								Aliases: []string{"c"},
								Usage:   "Path to config file to use in service",
							},
						},
						Action: func(c *cli.Context) error {
							facade := gwservice.NewFacade(c.String("config"))
							result, err := facade.Install()
							if err != nil {
								return err
							}
							fmt.Printf("Service installed: %s\n", result.ServiceName)
							if result.ConfigPath != "" {
								fmt.Printf("Config path: %s\n", result.ConfigPath)
							}
							if result.InstallPath != "" {
								fmt.Printf("Install path: %s\n", result.InstallPath)
							}
							return nil
						},
					},
					{
						Name:  "uninstall",
						Usage: "Uninstall the system service",
						Action: func(c *cli.Context) error {
							return gwservice.NewFacade(c.String("config")).Uninstall()
						},
					},
					{
						Name:  "start",
						Usage: "Start the system service",
						Action: func(c *cli.Context) error {
							return gwservice.NewFacade(c.String("config")).Start()
						},
					},
					{
						Name:  "stop",
						Usage: "Stop the system service",
						Action: func(c *cli.Context) error {
							return gwservice.NewFacade(c.String("config")).Stop()
						},
					},
					{
						Name:  "restart",
						Usage: "Restart the system service",
						Action: func(c *cli.Context) error {
							return gwservice.NewFacade(c.String("config")).Restart()
						},
					},
					{
						Name:  "status",
						Usage: "Check the system service status",
						Action: func(c *cli.Context) error {
							report := gwservice.NewFacade(c.String("config")).Status()
							fmt.Println(report.Format())
							return nil
						},
					},
				},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if cmdErr, ok := err.(*gwservice.CommandError); ok {
			os.Exit(int(cmdErr.Code))
		}
		os.Exit(int(gwservice.ExitServiceCommandFail))
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
		return fmt.Errorf("config error: %w", err)
	}

	// 覆盖配置中的 host/port
	if host := c.String("host"); host != "" {
		cfg.Gateway.Host = host
	}
	if port := c.Int("port"); port > 0 {
		cfg.Gateway.Port = port
	}

	// 如果是非交互式模式（即作为服务运行），重定向日志
	if !svc.Interactive() {
		logDir := utils.GetDefaultLogDir()
		logFile := filepath.Join(logDir, "mcp-gateway.log")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			slog.Warn("Failed to create log directory", "error", err)
		}
		f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err == nil {
			os.Stdout = f
			os.Stderr = f
			slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
				Level: slogLevel,
			})))
			slog.Info("Running as service, logs redirected", "path", logFile)
		}
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

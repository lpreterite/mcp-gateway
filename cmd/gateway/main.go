package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	svc "github.com/kardianos/service"
	"github.com/lpreterite/mcp-gateway/src/config"
	"github.com/lpreterite/mcp-gateway/src/gateway"
	"github.com/lpreterite/mcp-gateway/src/service"
	"github.com/lpreterite/mcp-gateway/src/stdio"
	"github.com/urfave/cli/v2"
)

var (
	version   = "1.0.3"
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
							s, err := service.NewManager(c.String("config"))
							if err != nil {
								return err
							}
							return s.Install()
						},
					},
					{
						Name:  "uninstall",
						Usage: "Uninstall the system service",
						Action: func(c *cli.Context) error {
							s, err := service.NewManager("")
							if err != nil {
								return err
							}
							return s.Uninstall()
						},
					},
					{
						Name:  "start",
						Usage: "Start the system service",
						Action: func(c *cli.Context) error {
							s, err := service.NewManager("")
							if err != nil {
								return err
							}
							return s.Start()
						},
					},
					{
						Name:  "stop",
						Usage: "Stop the system service",
						Action: func(c *cli.Context) error {
							s, err := service.NewManager("")
							if err != nil {
								return err
							}
							return s.Stop()
						},
					},
					{
						Name:  "restart",
						Usage: "Restart the system service",
						Action: func(c *cli.Context) error {
							s, err := service.NewManager("")
							if err != nil {
								return err
							}
							return s.Restart()
						},
					},
					{
						Name:  "status",
						Usage: "Check the system service status",
						Action: func(c *cli.Context) error {
							s, err := service.NewManager("")
							if err != nil {
								return err
							}
							status, err := s.Status()
							if err != nil {
								return err
							}
							switch status {
							case svc.StatusRunning:
								fmt.Println("Service is running")
							case svc.StatusStopped:
								fmt.Println("Service is stopped")
							default:
								fmt.Println("Service status unknown")
							}
							return nil
						},
					},
				},
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

	// 如果是通过服务启动的，处理服务运行逻辑
	// 注意：kardianos/service 在运行二进制时，如果它已经在运行中，它会接管
	// 但在这里我们先尝试加载配置并直接运行

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

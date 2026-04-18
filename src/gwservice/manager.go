package gwservice

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
	"github.com/lpreterite/mcp-gateway/src/config"
	"github.com/lpreterite/mcp-gateway/src/gateway"
	"github.com/lpreterite/mcp-gateway/src/utils"
)

type program struct {
	cfg *config.Config
	svr *gateway.Server
}

func (p *program) Start(s service.Service) error {
	slog.Info("Service starting")
	go p.run()
	return nil
}

func (p *program) run() {
	if p.cfg == nil {
		slog.Error("Cannot start gateway server: configuration is nil")
		return
	}

	// 强制尝试创建日志文件，无论是否交互式
	logDir := utils.GetDefaultLogDir()
	logFile := filepath.Join(logDir, "mcp-gateway.log")

	// 确保目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		slog.Warn("Failed to create log directory", "error", err)
	}

	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		// 注意：在服务运行期间保持文件打开
		os.Stdout = f
		os.Stderr = f
		// 重新设置 slog 到文件
		slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))
		slog.Info("Logging redirected to file", "path", logFile)
	}

	slog.Info("Initializing gateway server...")
	p.svr = gateway.NewServer(p.cfg)
	slog.Info("Gateway server initialized, starting listener...")

	if err := p.svr.Start(); err != nil {
		slog.Error("Service failed to start", "error", err)
	}
}

func (p *program) Stop(s service.Service) error {
	slog.Info("Service stopping")
	if p.svr != nil {
		return p.svr.Stop()
	}
	return nil
}

// GetConfig 返回服务配置
func GetConfig(configPath string) (*service.Config, error) {
	exePath, err := utils.GetExecutablePath()
	if err != nil {
		return nil, err
	}

	// 准备运行参数
	arguments := []string{}
	if configPath != "" {
		absConfig, _ := filepath.Abs(configPath)
		arguments = append(arguments, "--config", absConfig)
	}

	// 检测 PATH
	detectedPath := utils.DetectSystemPaths()

	options := service.KeyValue{}
	// 在 macOS 上默认使用用户级服务 (LaunchAgents)
	options["UserService"] = true
	// 自动启动
	options["RunAtLoad"] = true

	return &service.Config{
		Name:        "mcp-gateway",
		DisplayName: "MCP Gateway Service",
		Description: "Centralized MCP server management with connection pooling and HTTP/SSE transport",
		Executable:  exePath,
		Arguments:   arguments,
		EnvVars: map[string]string{
			"PATH": detectedPath,
		},
		Option: options,
	}, nil
}

func newService(configPath string, cfg *config.Config) (service.Service, error) {
	svcConfig, err := GetConfig(configPath)
	if err != nil {
		return nil, err
	}

	prg := &program{
		cfg: cfg,
	}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// NewManager 创建运行时服务管理器，会校验配置并将其注入服务程序。
func NewManager(configPath string) (service.Service, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return newService(configPath, cfg)
}

// NewControlManager 创建控制用服务句柄，不要求当前配置可成功加载。
func NewControlManager(configPath string) (service.Service, error) {
	return newService(configPath, nil)
}

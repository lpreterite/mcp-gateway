package service

import (
	"fmt"
	"log/slog"
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
	p.svr = gateway.NewServer(p.cfg)
	if err := p.svr.Start(); err != nil {
		slog.Error("Service failed to start", "error", err)
	}
}

func (p *program) Stop(s service.Service) error {
	slog.Info("Service stopping")
	// 这里可以添加更优雅的停止逻辑
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

	return &service.Config{
		Name:        "mcp-gateway",
		DisplayName: "MCP Gateway Service",
		Description: "Centralized MCP server management with connection pooling and HTTP/SSE transport",
		Executable:  exePath,
		Arguments:   arguments,
		EnvVars: map[string]string{
			"PATH": detectedPath,
		},
	}, nil
}

// NewManager 创建服务管理器
func NewManager(configPath string) (service.Service, error) {
	svcConfig, err := GetConfig(configPath)
	if err != nil {
		return nil, err
	}

	// 加载配置供程序运行使用
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
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

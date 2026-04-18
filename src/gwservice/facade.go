package gwservice

import (
	"fmt"

	"github.com/kardianos/service"
	"github.com/lpreterite/mcp-gateway/src/config"
)

type PlatformAdapter interface {
	Start(f *Facade, report ServiceStatusReport) error
	Stop(f *Facade, report ServiceStatusReport) error
	Restart(f *Facade, report ServiceStatusReport) error
}

type Facade struct {
	configPath string
	adapter    PlatformAdapter
}

func NewFacade(configPath string) *Facade {
	return &Facade{
		configPath: configPath,
		adapter:    newFacadePlatformAdapter(),
	}
}

func (f *Facade) Install() (*InstallResult, error) {
	cfg, err := config.Load(f.configPath)
	if err != nil {
		return nil, &CommandError{Code: ExitConfigError, Message: fmt.Sprintf("failed to load config: %v", err)}
	}

	s, err := newService(f.configPath, cfg)
	if err != nil {
		return nil, err
	}
	if err := s.Install(); err != nil {
		return nil, &CommandError{Code: ExitServiceCommandFail, Message: err.Error()}
	}

	report := DiagnoseStatus(f.configPath)
	result := &InstallResult{
		ServiceName: "mcp-gateway",
		ConfigPath:  report.ConfigPath,
		InstallPath: report.InstallDetail,
	}
	return result, nil
}

func (f *Facade) Uninstall() error {
	s, err := NewControlManager(f.configPath)
	if err != nil {
		return &CommandError{Code: ExitServiceCommandFail, Message: err.Error()}
	}
	if err := s.Uninstall(); err != nil {
		return &CommandError{Code: ExitServiceCommandFail, Message: err.Error()}
	}
	return nil
}

func (f *Facade) Start() error {
	report := DiagnoseStatus(f.configPath)
	if report.ConfigStatus != StateValid {
		return &CommandError{Code: ExitConfigError, Message: fmt.Sprintf("cannot start service: %s", report.ConfigDetail)}
	}
	return f.adapter.Start(f, report)
}

func (f *Facade) Stop() error {
	report := DiagnoseStatus(f.configPath)
	return f.adapter.Stop(f, report)
}

func (f *Facade) Restart() error {
	report := DiagnoseStatus(f.configPath)
	if report.ConfigStatus != StateValid {
		return &CommandError{Code: ExitConfigError, Message: fmt.Sprintf("cannot restart service: %s", report.ConfigDetail)}
	}
	return f.adapter.Restart(f, report)
}

func (f *Facade) Status() ServiceStatusReport {
	return DiagnoseStatus(f.configPath)
}

func (f *Facade) controlService() (service.Service, error) {
	return NewControlManager(f.configPath)
}

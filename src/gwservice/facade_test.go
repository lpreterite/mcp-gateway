package gwservice

import (
	"errors"
	"runtime"
	"testing"

	"github.com/kardianos/service"
)

// Mock PlatformAdapter 用于测试
type mockPlatformAdapter struct {
	startErr      error
	stopErr       error
	restartErr    error
	startCalled   bool
	stopCalled    bool
	restartCalled bool
}

func (m *mockPlatformAdapter) Start(f *Facade, report ServiceStatusReport) error {
	m.startCalled = true
	return m.startErr
}

func (m *mockPlatformAdapter) Stop(f *Facade, report ServiceStatusReport) error {
	m.stopCalled = true
	return m.stopErr
}

func (m *mockPlatformAdapter) Restart(f *Facade, report ServiceStatusReport) error {
	m.restartCalled = true
	return m.restartErr
}

// mockControlManager 模拟 service.Service
type mockControlManager struct {
	installErr   error
	uninstallErr error
	startErr     error
	stopErr      error
	restartErr   error
}

func (m *mockControlManager) Install() error {
	return m.installErr
}

func (m *mockControlManager) Uninstall() error {
	return m.uninstallErr
}

func (m *mockControlManager) Start() error {
	return m.startErr
}

func (m *mockControlManager) Stop() error {
	return m.stopErr
}

func (m *mockControlManager) Restart() error {
	return m.restartErr
}

func (m *mockControlManager) Run() error {
	return nil
}

func (m *mockControlManager) Status() (service.Status, error) {
	return service.StatusUnknown, nil
}

func (m *mockControlManager) Logger() (service.Logger, error) {
	return nil, nil
}

func (m *mockControlManager) System() (service.System, error) {
	return nil, nil
}

func TestNewFacade(t *testing.T) {
	f := NewFacade("/path/to/config")
	if f == nil {
		t.Fatal("expected non-nil Facade")
	}
	if f.configPath != "/path/to/config" {
		t.Errorf("expected configPath '/path/to/config', got '%s'", f.configPath)
	}
	if f.adapter == nil {
		t.Error("expected non-nil adapter")
	}
}

func TestFacadeStatus(t *testing.T) {
	f := NewFacade("/nonexistent/config.json")
	report := f.Status()

	// Status 应该返回有效的报告
	if report.ConfigStatus == "" {
		t.Error("expected non-empty ConfigStatus")
	}
	if report.SuggestedAction.Code == "" {
		t.Error("expected non-empty SuggestedAction.Code")
	}
}

func TestFacadeStartWithInvalidConfig(t *testing.T) {
	f := NewFacade("/nonexistent/config.json")
	err := f.Start()

	// 配置无效时应该返回错误
	if err == nil {
		t.Error("expected error when config is invalid")
	}

	var cmdErr *CommandError
	if errors.As(err, &cmdErr) {
		if cmdErr.Code != ExitConfigError {
			t.Errorf("expected ExitConfigError, got %d", cmdErr.Code)
		}
	} else {
		t.Error("expected CommandError type")
	}
}

func TestFacadeRestartWithInvalidConfig(t *testing.T) {
	f := NewFacade("/nonexistent/config.json")
	err := f.Restart()

	// 配置无效时应该返回错误
	if err == nil {
		t.Error("expected error when config is invalid")
	}

	var cmdErr *CommandError
	if errors.As(err, &cmdErr) {
		if cmdErr.Code != ExitConfigError {
			t.Errorf("expected ExitConfigError, got %d", cmdErr.Code)
		}
	} else {
		t.Error("expected CommandError type")
	}
}

func TestFacadeStop(t *testing.T) {
	f := NewFacade("/nonexistent/config.json")
	// Stop 在配置无效时也应该能执行（因为它不依赖配置）
	err := f.Stop()
	// Stop 可能返回 nil 或者错误，取决于 adapter 实现
	_ = err
}

func TestFacadeUninstallWithInvalidConfig(t *testing.T) {
	f := NewFacade("/nonexistent/config.json")
	err := f.Uninstall()

	// NewControlManager 在配置不存在时也可能失败
	// 这取决于 NewControlManager 的实现
	if err != nil {
		var cmdErr *CommandError
		if errors.As(err, &cmdErr) {
			// 期望是 CommandError 类型
			if cmdErr.Code != ExitServiceCommandFail {
				t.Errorf("expected ExitServiceCommandFail, got %d", cmdErr.Code)
			}
		}
	}
}

func TestFacadeControlService(t *testing.T) {
	f := NewFacade("/nonexistent/config.json")
	// controlService 是内部方法，测试其基本行为
	svc, err := f.controlService()
	if err != nil {
		// 配置无效时可能返回错误
		t.Logf("controlService returned error (expected for invalid config): %v", err)
		return
	}
	if svc == nil {
		t.Error("expected non-nil service when config is valid")
	}
}

func TestPlatformAdapterDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	adapter := platformAdapter()
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}

	// 验证是 darwinAdapter 类型
	_, ok := adapter.(*darwinAdapter)
	if !ok {
		t.Errorf("expected *darwinAdapter, got %T", adapter)
	}
}

func TestPlatformAdapterLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-specific test on non-linux platform")
	}

	adapter := platformAdapter()
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}

	// 验证是 linuxAdapter 类型
	_, ok := adapter.(*linuxAdapter)
	if !ok {
		t.Errorf("expected *linuxAdapter, got %T", adapter)
	}
}

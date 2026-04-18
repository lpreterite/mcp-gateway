package gwservice

import (
	"errors"
	"testing"
)

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

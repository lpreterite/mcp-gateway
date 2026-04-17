package gwservice

import (
	"runtime"
	"testing"
)

func TestDiagnoseStatusReturnsReport(t *testing.T) {
	// 使用不存在的配置文件路径测试诊断功能
	report := DiagnoseStatus("/nonexistent/path/config.json")

	// 验证返回的是有效的报告结构
	if report.ConfigPath == "" {
		t.Error("expected ConfigPath to be set even for invalid path")
	}

	// 验证状态字段都有有效值
	if report.ConfigStatus == "" {
		t.Error("expected ConfigStatus to be set")
	}
	if report.InstallStatus == "" {
		t.Error("expected InstallStatus to be set")
	}
	if report.RegistrationStatus == "" {
		t.Error("expected RegistrationStatus to be set")
	}
	if report.ProcessStatus == "" {
		t.Error("expected ProcessStatus to be set")
	}
	if report.HealthStatus == "" {
		t.Error("expected HealthStatus to be set")
	}

	// 验证建议操作有有效值
	if report.SuggestedAction.Code == "" {
		t.Error("expected SuggestedAction.Code to be set")
	}
	if report.SuggestedAction.Message == "" {
		t.Error("expected SuggestedAction.Message to be set")
	}
}

func TestDiagnoseStatusWithInvalidConfigSetsCorrectStates(t *testing.T) {
	report := DiagnoseStatus("/nonexistent/path/config.json")

	// 配置无效时，ConfigStatus 应该是 invalid 或 valid 但带错误信息
	if report.ConfigStatus == StateValid && report.ConfigDetail == "" {
		t.Error("valid config should have config detail with path")
	}

	// 配置无效时，其他层应该是 unknown
	if report.ConfigStatus == StateInvalid {
		if report.InstallStatus != StateUnknown {
			t.Errorf("expected InstallStatus to be unknown when config invalid, got %s", report.InstallStatus)
		}
		if report.RegistrationStatus != StateUnknown {
			t.Errorf("expected RegistrationStatus to be unknown when config invalid, got %s", report.RegistrationStatus)
		}
	}
}

func TestSuggestedActionForMissingInstall(t *testing.T) {
	report := DiagnoseStatus("/nonexistent/path/config.json")

	// 当安装状态为 missing 时，应该建议安装服务
	if report.InstallStatus == StateMissing {
		if report.SuggestedAction.Code != ActionInstallService {
			t.Errorf("expected action code ActionInstallService, got %v", report.SuggestedAction.Code)
		}
	}
}

func TestDetectInstallDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	status, detail := detectInstall()

	// 应该返回已知状态
	if status != StatePresent && status != StateMissing && status != StateUnknown {
		t.Errorf("unexpected install status: %s", status)
	}

	// detail 应该非空
	if detail == "" {
		t.Error("expected non-empty install detail")
	}
}

func TestDetectInstallLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-specific test on non-linux platform")
	}

	status, detail := detectInstall()

	// 应该返回已知状态
	if status != StatePresent && status != StateMissing && status != StateUnknown {
		t.Errorf("unexpected install status: %s", status)
	}

	// detail 应该非空
	if detail == "" {
		t.Error("expected non-empty install detail")
	}
}

func TestDetectRegistrationDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	status, detail := detectRegistration()

	// 应该返回已知状态
	if status != StateLoaded && status != StateMissing && status != StateUnknown {
		t.Errorf("unexpected registration status: %s", status)
	}

	// detail 应该非空
	if detail == "" {
		t.Error("expected non-empty registration detail")
	}
}

func TestDetectRegistrationLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-specific test on non-linux platform")
	}

	status, detail := detectRegistration()

	// 应该返回已知状态
	if status != StateLoaded && status != StateMissing && status != StateUnknown {
		t.Errorf("unexpected registration status: %s", status)
	}

	// detail 应该非空
	if detail == "" {
		t.Error("expected non-empty registration detail")
	}
}

func TestSuggestedRegistrationFix(t *testing.T) {
	action := suggestedRegistrationFix()

	if action.Code != ActionReloadRegistration {
		t.Errorf("expected action code ActionReloadRegistration, got %v", action.Code)
	}

	if action.Message == "" {
		t.Error("expected non-empty action message")
	}
}

func TestParseLaunchctlField(t *testing.T) {
	text := `
	state = running
	pid = 12345
	label = com.example.test
	`

	state := parseLaunchctlField(text, "state")
	if state != "running" {
		t.Errorf("expected 'running', got '%s'", state)
	}

	pid := parseLaunchctlField(text, "pid")
	if pid != "12345" {
		t.Errorf("expected '12345', got '%s'", pid)
	}

	label := parseLaunchctlField(text, "label")
	if label != "com.example.test" {
		t.Errorf("expected 'com.example.test', got '%s'", label)
	}

	// 测试不存在的字段
	nonexistent := parseLaunchctlField(text, "nonexistent")
	if nonexistent != "" {
		t.Errorf("expected empty string for nonexistent field, got '%s'", nonexistent)
	}
}

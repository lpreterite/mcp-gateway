package gwservice

import (
	"testing"
)

func TestServiceStateConstants(t *testing.T) {
	// 验证状态常量定义正确
	states := []ServiceState{
		StateUnknown, StateValid, StateInvalid, StatePresent, StateMissing,
		StateLoaded, StateRunning, StateStopped, StateNotRunning,
		StateReachable, StateUnreachable,
	}

	expected := []string{
		"unknown", "valid", "invalid", "present", "missing",
		"loaded", "running", "stopped", "not_running",
		"reachable", "unreachable",
	}

	for i, state := range states {
		if string(state) != expected[i] {
			t.Errorf("expected state %s, got %s", expected[i], state)
		}
	}
}

func TestRunTargetStateConstants(t *testing.T) {
	if string(RunTargetRunning) != "running" {
		t.Errorf("expected RunTargetRunning to be 'running', got %s", RunTargetRunning)
	}
	if string(RunTargetStopped) != "stopped" {
		t.Errorf("expected RunTargetStopped to be 'stopped', got %s", RunTargetStopped)
	}
}

func TestSuggestedActionCodeConstants(t *testing.T) {
	codes := []SuggestedActionCode{
		ActionNone, ActionFixConfig, ActionInstallService, ActionStartService,
		ActionReloadRegistration, ActionWaitReady,
	}

	expected := []string{
		"none", "fix_config", "install_service", "start_service",
		"reload_registration", "wait_ready",
	}

	for i, code := range codes {
		if string(code) != expected[i] {
			t.Errorf("expected action code %s, got %s", expected[i], code)
		}
	}
}

func TestExitCodeConstants(t *testing.T) {
	codes := []ExitCode{
		ExitOK, ExitConfigError, ExitInstallMissing,
		ExitRegistrationError, ExitRuntimeError,
		ExitHealthError, ExitServiceCommandFail,
	}

	expected := []int{0, 10, 20, 30, 40, 50, 60}

	for i, code := range codes {
		if int(code) != expected[i] {
			t.Errorf("expected exit code %d, got %d", expected[i], code)
		}
	}
}

func TestCommandError(t *testing.T) {
	err := &CommandError{
		Code:    ExitConfigError,
		Message: "test error message",
	}

	if err.Error() != "test error message" {
		t.Errorf("expected error message 'test error message', got %s", err.Error())
	}

	if err.Code != ExitConfigError {
		t.Errorf("expected exit code %d, got %d", ExitConfigError, err.Code)
	}
}

func TestInstallResult(t *testing.T) {
	result := &InstallResult{
		ServiceName: "mcp-gateway",
		ConfigPath:  "/path/to/config",
		InstallPath: "/path/to/install",
	}

	if result.ServiceName != "mcp-gateway" {
		t.Errorf("expected service name 'mcp-gateway', got %s", result.ServiceName)
	}
	if result.ConfigPath != "/path/to/config" {
		t.Errorf("expected config path '/path/to/config', got %s", result.ConfigPath)
	}
	if result.InstallPath != "/path/to/install" {
		t.Errorf("expected install path '/path/to/install', got %s", result.InstallPath)
	}
}

func TestSuggestedAction(t *testing.T) {
	action := SuggestedAction{
		Code:    ActionFixConfig,
		Message: "please fix config",
	}

	if action.Code != ActionFixConfig {
		t.Errorf("expected action code ActionFixConfig, got %v", action.Code)
	}
	if action.Message != "please fix config" {
		t.Errorf("expected message 'please fix config', got %s", action.Message)
	}
}

func TestServiceStatusReportFormat(t *testing.T) {
	report := ServiceStatusReport{
		ConfigStatus:       StateValid,
		ConfigDetail:       "/path/to/config.json",
		InstallStatus:      StatePresent,
		InstallDetail:      "/path/to/plist",
		RegistrationStatus: StateLoaded,
		RegistrationDetail: "gui/501/mcp-gateway",
		ProcessStatus:      StateRunning,
		ProcessDetail:      "launchctl state=running, pid=12345",
		HealthStatus:       StateReachable,
		HealthDetail:       "TCP 端口可连接: 127.0.0.1:4298",
		SuggestedAction: SuggestedAction{
			Code:    ActionNone,
			Message: "none",
		},
	}

	formatted := report.Format()

	// 验证格式化输出包含关键信息
	expectedLines := []string{
		"Config: valid (/path/to/config.json)",
		"Install: present (/path/to/plist)",
		"Registration: loaded (gui/501/mcp-gateway)",
		"Process: running (launchctl state=running, pid=12345)",
		"Health: reachable (TCP 端口可连接: 127.0.0.1:4298)",
		"Suggested action: none",
	}

	for _, expected := range expectedLines {
		if !containsLine(formatted, expected) {
			t.Errorf("expected formatted output to contain: %s\nGot:\n%s", expected, formatted)
		}
	}
}

func TestServiceStatusReportFormatWithEmptyDetails(t *testing.T) {
	report := ServiceStatusReport{
		ConfigStatus:       StateUnknown,
		ConfigDetail:       "",
		InstallStatus:      StateUnknown,
		InstallDetail:      "",
		RegistrationStatus: StateUnknown,
		RegistrationDetail: "",
		ProcessStatus:      StateUnknown,
		ProcessDetail:      "",
		HealthStatus:       StateUnknown,
		HealthDetail:       "",
		SuggestedAction: SuggestedAction{
			Code:    ActionNone,
			Message: "none",
		},
	}

	formatted := report.Format()

	// 验证只有状态名，没有括号
	expectedLines := []string{
		"Config: unknown",
		"Install: unknown",
		"Registration: unknown",
		"Process: unknown",
		"Health: unknown",
		"Suggested action: none",
	}

	for _, expected := range expectedLines {
		if !containsLine(formatted, expected) {
			t.Errorf("expected formatted output to contain: %s\nGot:\n%s", expected, formatted)
		}
	}
}

func TestFormatLine(t *testing.T) {
	// 测试带详情
	result := formatLine(StateValid, "/path/to/config")
	expected := "valid (/path/to/config)"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}

	// 测试空详情
	result = formatLine(StateUnknown, "")
	expected = "unknown"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func containsLine(text, line string) bool {
	lines := splitLines(text)
	for _, l := range lines {
		if l == line {
			return true
		}
	}
	return false
}

func splitLines(text string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

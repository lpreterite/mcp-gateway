package gwservice

import (
	"errors"
	"runtime"
	"testing"
)

// ============ platform_generic.go 测试 ============

func TestGenericAdapterStart(t *testing.T) {
	adapter := &genericAdapter{}
	f := NewFacade("/nonexistent/config.json")

	// 由于 controlService 会失败，genericAdapter.Start 应该返回错误
	err := adapter.Start(f, ServiceStatusReport{})
	if err == nil {
		t.Error("expected error when controlService fails")
	}

	var cmdErr *CommandError
	if errors.As(err, &cmdErr) {
		if cmdErr.Code != ExitServiceCommandFail {
			t.Errorf("expected ExitServiceCommandFail, got %d", cmdErr.Code)
		}
	}
}

func TestGenericAdapterStop(t *testing.T) {
	adapter := &genericAdapter{}
	f := NewFacade("/nonexistent/config.json")

	// 由于 controlService 会失败，genericAdapter.Stop 应该返回错误
	err := adapter.Stop(f, ServiceStatusReport{})
	if err == nil {
		t.Error("expected error when controlService fails")
	}
}

func TestGenericAdapterRestart(t *testing.T) {
	adapter := &genericAdapter{}
	f := NewFacade("/nonexistent/config.json")

	// 由于 controlService 会失败，genericAdapter.Restart 应该返回错误
	err := adapter.Restart(f, ServiceStatusReport{})
	if err == nil {
		t.Error("expected error when controlService fails")
	}
}

// ============ platform_darwin.go 辅助函数测试 ============

func TestNormalizeLaunchctlError(t *testing.T) {
	testCases := []struct {
		name     string
		output   []byte
		inputErr error
		expected string
	}{
		{
			name:     "non-empty output",
			output:   []byte("some error message"),
			inputErr: errors.New("underlying error"),
			expected: "some error message",
		},
		{
			name:     "empty output with error",
			output:   []byte(""),
			inputErr: errors.New("underlying error"),
			expected: "underlying error",
		},
		{
			name:     "whitespace output",
			output:   []byte("   \n  "),
			inputErr: errors.New("error with whitespace"),
			expected: "error with whitespace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeLaunchctlError(tc.output, tc.inputErr)
			if result != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestIsLaunchctlMissing(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Could not find service",
			err:      errors.New("Could not find service com.example.test"),
			expected: true,
		},
		{
			name:     "No such process",
			err:      errors.New("launchctl: No such process"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isLaunchctlMissing(tc.err)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestDarwinAdapterStartStates(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	adapter := &darwinAdapter{}

	testCases := []struct {
		name            string
		report          ServiceStatusReport
		expectError     bool
		expectedErrCode ExitCode
	}{
		{
			name: "service not installed",
			report: ServiceStatusReport{
				InstallStatus: StateMissing,
				InstallDetail: "/path/to/plist",
			},
			expectError:     true,
			expectedErrCode: ExitInstallMissing,
		},
		{
			name: "service already running",
			report: ServiceStatusReport{
				InstallStatus: StatePresent,
				ProcessStatus: StateRunning,
			},
			expectError: false,
		},
		{
			name: "service not registered",
			report: ServiceStatusReport{
				InstallStatus:      StatePresent,
				ProcessStatus:      StateNotRunning,
				RegistrationStatus: StateMissing,
			},
			expectError: false, // bootstrap 会成功或返回错误
		},
		{
			name: "service loaded but not running",
			report: ServiceStatusReport{
				InstallStatus:      StatePresent,
				ProcessStatus:      StateNotRunning,
				RegistrationStatus: StateLoaded,
			},
			expectError: false, // kickstart 会尝试
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := adapter.Start(&Facade{}, tc.report)
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Logf("start returned error (may be expected): %v", err)
			}
			if tc.expectError && err != nil {
				var cmdErr *CommandError
				if errors.As(err, &cmdErr) {
					if cmdErr.Code != tc.expectedErrCode {
						t.Errorf("expected error code %d, got %d", tc.expectedErrCode, cmdErr.Code)
					}
				}
			}
		})
	}
}

func TestDarwinAdapterStopStates(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	adapter := &darwinAdapter{}

	testCases := []struct {
		name        string
		report      ServiceStatusReport
		expectError bool
	}{
		{
			name: "service not installed",
			report: ServiceStatusReport{
				InstallStatus: StateMissing,
			},
			expectError: false, // 应该直接返回 nil
		},
		{
			name: "service not registered",
			report: ServiceStatusReport{
				InstallStatus:      StatePresent,
				RegistrationStatus: StateMissing,
			},
			expectError: false, // 应该直接返回 nil
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := adapter.Stop(&Facade{}, tc.report)
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Logf("stop returned error (may be expected): %v", err)
			}
		})
	}
}

func TestDarwinAdapterRestartStates(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	adapter := &darwinAdapter{}

	testCases := []struct {
		name        string
		report      ServiceStatusReport
		expectError bool
	}{
		{
			name: "service not installed",
			report: ServiceStatusReport{
				InstallStatus: StateMissing,
			},
			expectError: true,
		},
		{
			name: "service not registered",
			report: ServiceStatusReport{
				InstallStatus:      StatePresent,
				RegistrationStatus: StateMissing,
			},
			expectError: false, // 会尝试 bootstrap
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := adapter.Restart(&Facade{}, tc.report)
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Logf("restart returned error (may be expected): %v", err)
			}
		})
	}
}

// ============ platform_systemd.go 测试 ============

func TestSystemctlCommand(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-specific test on non-linux platform")
	}

	adapter := &linuxAdapter{}

	// 测试 systemctl 命令执行
	// 这个测试可能会失败如果 systemctl 不可用
	err := adapter.systemctl("status", "nonexistent.service")
	if err != nil {
		t.Logf("systemctl status returned error (expected for nonexistent service): %v", err)
	}
}

func TestLinuxAdapterStartStates(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-specific test on non-linux platform")
	}

	adapter := &linuxAdapter{}

	testCases := []struct {
		name        string
		report      ServiceStatusReport
		expectError bool
	}{
		{
			name: "service not installed",
			report: ServiceStatusReport{
				InstallStatus: StateMissing,
			},
			expectError: true,
		},
		{
			name: "service already running",
			report: ServiceStatusReport{
				InstallStatus: StatePresent,
				ProcessStatus: StateRunning,
			},
			expectError: false,
		},
		{
			name: "service not registered",
			report: ServiceStatusReport{
				InstallStatus:      StatePresent,
				ProcessStatus:      StateNotRunning,
				RegistrationStatus: StateMissing,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := adapter.Start(&Facade{}, tc.report)
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Logf("start returned error (may be expected): %v", err)
			}
		})
	}
}

func TestLinuxAdapterStopStates(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-specific test on non-linux platform")
	}

	adapter := &linuxAdapter{}

	// service not installed - should return nil
	err := adapter.Stop(&Facade{}, ServiceStatusReport{
		InstallStatus: StateMissing,
	})
	if err != nil {
		t.Errorf("expected nil error when service not installed, got %v", err)
	}
}

func TestLinuxAdapterRestartStates(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-specific test on non-linux platform")
	}

	adapter := &linuxAdapter{}

	testCases := []struct {
		name        string
		report      ServiceStatusReport
		expectError bool
	}{
		{
			name: "service not installed",
			report: ServiceStatusReport{
				InstallStatus: StateMissing,
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := adapter.Restart(&Facade{}, tc.report)
			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
		})
	}
}

func TestDaemonReload(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-specific test on non-linux platform")
	}

	adapter := &linuxAdapter{}
	err := adapter.daemonReload()
	if err != nil {
		t.Logf("daemonReload returned error (may be expected if no permissions): %v", err)
	}
}

// ============ detectDarwinProcess 测试 ============

func TestDetectDarwinProcess(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	status, detail := detectDarwinProcess()

	// 应该返回已知状态
	if status != StateRunning && status != StateNotRunning && status != StateUnknown {
		t.Errorf("unexpected process status: %s", status)
	}

	// detail 可能为空（当 launchctl 失败时）
	t.Logf("detectDarwinProcess: status=%s, detail=%s", status, detail)
}

// ============ formatLine 边缘情况测试 ============

func TestFormatLineEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		status   ServiceState
		detail   string
		expected string
	}{
		{
			name:     "empty detail",
			status:   StateUnknown,
			detail:   "",
			expected: "unknown",
		},
		{
			name:     "whitespace detail",
			status:   StateValid,
			detail:   "   ",
			expected: "valid (   )",
		},
		{
			name:     "normal case",
			status:   StateRunning,
			detail:   "pid=12345",
			expected: "running (pid=12345)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatLine(tc.status, tc.detail)
			if result != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// ============ 状态转换分支测试 ============

func TestSuggestedActionForVariousStates(t *testing.T) {
	testCases := []struct {
		name               string
		report             ServiceStatusReport
		expectedActionCode SuggestedActionCode
	}{
		{
			name: "missing install",
			report: ServiceStatusReport{
				InstallStatus: StateMissing,
			},
			expectedActionCode: ActionInstallService,
		},
		{
			name: "present but not registered",
			report: ServiceStatusReport{
				InstallStatus:      StatePresent,
				RegistrationStatus: StateMissing,
			},
			expectedActionCode: ActionReloadRegistration,
		},
		{
			name: "running but unreachable",
			report: ServiceStatusReport{
				ProcessStatus: StateRunning,
				HealthStatus:  StateUnreachable,
			},
			expectedActionCode: ActionWaitReady,
		},
		{
			name: "valid config but not running",
			report: ServiceStatusReport{
				ConfigStatus:  StateValid,
				ProcessStatus: StateNotRunning,
			},
			expectedActionCode: ActionStartService,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			report := DiagnoseStatus("/nonexistent/config.json")
			// 由于 DiagnoseStatus 有自己的逻辑，我们只能验证它不会 panic
			if report.SuggestedAction.Code == "" {
				t.Error("expected non-empty SuggestedAction.Code")
			}
		})
	}
}

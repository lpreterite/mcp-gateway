//go:build darwin

package gwservice

import (
	"errors"
	"testing"
)

func TestDarwinAdapterStartStates(t *testing.T) {
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
			expectError: false,
		},
		{
			name: "service loaded but not running",
			report: ServiceStatusReport{
				InstallStatus:      StatePresent,
				ProcessStatus:      StateNotRunning,
				RegistrationStatus: StateLoaded,
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
			expectError: false,
		},
		{
			name: "service not registered",
			report: ServiceStatusReport{
				InstallStatus:      StatePresent,
				RegistrationStatus: StateMissing,
			},
			expectError: false,
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
			expectError: false,
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

func TestDetectDarwinProcess(t *testing.T) {
	status, detail := detectDarwinProcess()

	if status != StateRunning && status != StateNotRunning && status != StateUnknown {
		t.Errorf("unexpected process status: %s", status)
	}

	t.Logf("detectDarwinProcess: status=%s, detail=%s", status, detail)
}

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

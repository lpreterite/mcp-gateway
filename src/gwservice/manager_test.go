package gwservice

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lpreterite/mcp-gateway/src/config"
)

func TestGetConfig(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// 写入有效的测试配置
	configContent := `{
		"gateway": {"host": "127.0.0.1", "port": 4298},
		"servers": [{"name": "test", "type": "local", "command": ["echo", "test"], "enabled": true}]
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := GetConfig(configPath)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil service.Config")
	}

	if cfg.Name != "mcp-gateway" {
		t.Errorf("expected Name 'mcp-gateway', got '%s'", cfg.Name)
	}
	if cfg.DisplayName != "MCP Gateway Service" {
		t.Errorf("expected DisplayName 'MCP Gateway Service', got '%s'", cfg.DisplayName)
	}
	if cfg.Executable == "" {
		t.Error("expected non-empty Executable path")
	}
	if len(cfg.Arguments) == 0 {
		t.Error("expected non-empty Arguments")
	}
	if cfg.Arguments[0] != "--config" {
		t.Errorf("expected first argument '--config', got '%s'", cfg.Arguments[0])
	}
}

func TestGetConfigWithEmptyPath(t *testing.T) {
	cfg, err := GetConfig("")
	if err != nil {
		t.Fatalf("GetConfig with empty path failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil service.Config")
	}

	// 空路径时不应该有 --config 参数
	for _, arg := range cfg.Arguments {
		if arg == "--config" {
			t.Error("expected no --config argument when configPath is empty")
		}
	}
}

func TestNewManagerWithInvalidPath(t *testing.T) {
	_, err := NewManager("/nonexistent/path/config.json")

	if err == nil {
		t.Error("expected error for nonexistent config path")
	}
}

func TestNewControlManager(t *testing.T) {
	// NewControlManager 不要求配置有效
	svc, err := NewControlManager("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("NewControlManager failed: %v", err)
	}

	if svc == nil {
		t.Error("expected non-nil service")
	}
}

// 验证 CommandError 实现了 error 接口
func TestCommandErrorImplementsError(t *testing.T) {
	cmdErr := &CommandError{
		Code:    ExitConfigError,
		Message: "test",
	}

	var err error = cmdErr
	if err.Error() != "test" {
		t.Errorf("expected error message 'test', got '%s'", err.Error())
	}
}

func TestCommandErrorWithDifferentCodes(t *testing.T) {
	testCases := []struct {
		code    ExitCode
		message string
	}{
		{ExitOK, "success"},
		{ExitConfigError, "config error"},
		{ExitInstallMissing, "install missing"},
		{ExitRegistrationError, "registration error"},
		{ExitRuntimeError, "runtime error"},
		{ExitHealthError, "health error"},
		{ExitServiceCommandFail, "service command failed"},
	}

	for _, tc := range testCases {
		err := &CommandError{Code: tc.code, Message: tc.message}
		if err.Error() != tc.message {
			t.Errorf("expected message '%s', got '%s'", tc.message, err.Error())
		}
		if err.Code != tc.code {
			t.Errorf("expected code %d, got %d", tc.code, err.Code)
		}
	}
}

func TestNewService(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// 写入有效的测试配置
	configContent := `{
		"gateway": {"host": "127.0.0.1", "port": 4298},
		"servers": [{"name": "test", "type": "local", "command": ["echo", "test"], "enabled": true}]
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := loadConfigForTest(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	svc, err := newService(configPath, cfg)
	if err != nil {
		t.Fatalf("newService failed: %v", err)
	}

	if svc == nil {
		t.Error("expected non-nil service")
	}
}

func TestNewServiceWithNilConfig(t *testing.T) {
	// 使用 nil 配置创建服务（用于控制命令）
	svc, err := newService("/nonexistent/path.json", nil)
	if err != nil {
		t.Fatalf("newService with nil config failed: %v", err)
	}

	if svc == nil {
		t.Error("expected non-nil service")
	}
}

// loadConfigForTest 是测试辅助函数，用于加载配置
func loadConfigForTest(path string) (*config.Config, error) {
	return config.Load(path)
}

// 验证 InstallResult 结构
func TestInstallResultFields(t *testing.T) {
	result := &InstallResult{
		ServiceName: "test-service",
		ConfigPath:  "/path/to/config",
		InstallPath: "/path/to/plist",
	}

	if result.ServiceName != "test-service" {
		t.Errorf("expected ServiceName 'test-service', got '%s'", result.ServiceName)
	}
	if result.ConfigPath != "/path/to/config" {
		t.Errorf("expected ConfigPath '/path/to/config', got '%s'", result.ConfigPath)
	}
	if result.InstallPath != "/path/to/plist" {
		t.Errorf("expected InstallPath '/path/to/plist', got '%s'", result.InstallPath)
	}
}

// 验证 errors.As 能正确处理 CommandError
func TestErrorsAsCommandError(t *testing.T) {
	originalErr := &CommandError{Code: ExitConfigError, Message: "config is invalid"}

	wrappedErr := errors.New(originalErr.Error())

	var cmdErr *CommandError
	if errors.As(wrappedErr, &cmdErr) {
		t.Error("expected errors.As to return false for wrapped error")
	}
}

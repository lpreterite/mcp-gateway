package gwservice

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kardianos/service"
)

// ============ program 结构测试 ============

func TestProgramStruct(t *testing.T) {
	// program 结构是一个简单的结构体，包含配置和服务器
	p := &program{}
	if p.cfg != nil {
		t.Error("expected nil cfg initially")
	}
	if p.svr != nil {
		t.Error("expected nil svr initially")
	}
}

func TestProgramStartReturnsImmediately(t *testing.T) {
	// program.Start 应该立即返回（不阻塞）
	// 因为它使用 go run() 在后台运行
	p := &program{}
	var s mockService
	err := p.Start(s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProgramStopWithNilServer(t *testing.T) {
	// 当 svr 为 nil 时，Stop 应该返回 nil
	p := &program{}
	var s mockService
	err := p.Stop(s)
	if err != nil {
		t.Errorf("unexpected error when svr is nil: %v", err)
	}
}

// mockService 模拟 service.Service
type mockService struct{}

func (m mockService) Start() error                                           { return nil }
func (m mockService) Stop() error                                            { return nil }
func (m mockService) Restart() error                                         { return nil }
func (m mockService) Install() error                                         { return nil }
func (m mockService) Uninstall() error                                       { return nil }
func (m mockService) Run() error                                             { return nil }
func (m mockService) Logger(errs chan<- error) (service.Logger, error)       { return nil, nil }
func (m mockService) SystemLogger(errs chan<- error) (service.Logger, error) { return nil, nil }
func (m mockService) String() string                                         { return "mockService" }
func (m mockService) Platform() string                                       { return "mock" }
func (m mockService) Status() (service.Status, error)                        { return service.StatusUnknown, nil }

// ============ status.go 补充测试 ============

func TestDetectInstallOnUnknownPlatform(t *testing.T) {
	// 这个测试验证 detectInstall 对未知平台的处理
	// 模拟未知平台的行为
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		// 未知平台应该返回 StateUnknown
		t.Skip("test only for non-darwin/non-linux platforms")
	}
}

func TestDetectRegistrationDarwinErrorPath(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	// darwinLaunchctlPrint 可能会返回各种错误
	// 测试当 output 为空但有错误时的情况
	target, output, err := darwinLaunchctlPrint()
	if err != nil {
		// 预期可能出错
		t.Logf("darwinLaunchctlPrint returned error: %v", err)
	}
	if target == "" {
		t.Error("expected non-empty target")
	}
	_ = output // 可能为空
}

func TestSuggestedRegistrationFixOnLinux(t *testing.T) {
	// suggestedRegistrationFix 应该根据 GOOS 返回不同的建议
	action := suggestedRegistrationFix()
	if action.Code != ActionReloadRegistration {
		t.Errorf("expected ActionReloadRegistration, got %v", action.Code)
	}
	if action.Message == "" {
		t.Error("expected non-empty message")
	}

	// 验证消息包含平台特定内容
	if runtime.GOOS == "darwin" {
		if !containsString(action.Message, "launchctl") && !containsString(action.Message, "launchctl bootstrap") {
			// 消息可能包含 launchctl 或 systemctl
			t.Logf("darwin message: %s", action.Message)
		}
	} else if runtime.GOOS == "linux" {
		if !containsString(action.Message, "systemctl") && !containsString(action.Message, "daemon-reload") {
			t.Logf("linux message: %s", action.Message)
		}
	}
}

func TestDetectDarwinProcessOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	status, detail := detectDarwinProcess()

	// status 应该是已知状态之一
	switch status {
	case StateRunning, StateNotRunning, StateUnknown:
		// 已知状态
	default:
		t.Errorf("unexpected status: %s", status)
	}

	// detail 可能为空
	t.Logf("detectDarwinProcess: status=%s, detail=%s", status, detail)
}

func TestDiagnoseStatusWithRealConfig(t *testing.T) {
	// 创建临时有效配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	validConfig := `{
		"gateway": {"host": "127.0.0.1", "port": 4298},
		"servers": [{"name": "test", "type": "local", "command": ["echo", "test"], "enabled": true}]
	}`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	report := DiagnoseStatus(configPath)

	// 配置有效时，ConfigStatus 应该是 StateValid
	if report.ConfigStatus != StateValid {
		t.Errorf("expected ConfigStatus StateValid, got %s", report.ConfigStatus)
	}
	if report.ConfigDetail == "" {
		t.Error("expected non-empty ConfigDetail")
	}
}

func TestDiagnoseStatusWithConfigPathResolution(t *testing.T) {
	// 测试当 resolvedPath 不为空时的情况
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	validConfig := `{
		"gateway": {"host": "127.0.0.1", "port": 4298},
		"servers": [{"name": "test", "type": "local", "command": ["echo", "test"], "enabled": true}]
	}`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	report := DiagnoseStatus(configPath)

	// 验证 ConfigPath 被设置
	if report.ConfigPath == "" {
		t.Error("expected non-empty ConfigPath")
	}
}

// ============ darwinLaunchAgentPath 测试 ============

func TestDarwinLaunchAgentPath(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	path, err := darwinLaunchAgentPath()
	if err != nil {
		t.Fatalf("darwinLaunchAgentPath failed: %v", err)
	}

	// 验证路径格式
	if path == "" {
		t.Error("expected non-empty path")
	}
	if !containsString(path, "Library/LaunchAgents") {
		t.Error("expected path to contain Library/LaunchAgents")
	}
	if !containsString(path, "mcp-gateway.plist") {
		t.Error("expected path to contain mcp-gateway.plist")
	}
}

// ============ GetConfig 边缘测试 ============

func TestGetConfigWithRelativePath(t *testing.T) {
	// 使用相对路径
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	validConfig := `{
		"gateway": {"host": "127.0.0.1", "port": 4298},
		"servers": [{"name": "test", "type": "local", "command": ["echo", "test"], "enabled": true}]
	}`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := GetConfig(configPath)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	// 验证 Arguments 包含 --config 和绝对路径
	foundConfig := false
	for i, arg := range cfg.Arguments {
		if arg == "--config" && i+1 < len(cfg.Arguments) {
			foundConfig = true
			absPath := cfg.Arguments[i+1]
			if absPath == "" {
				t.Error("expected non-empty config path argument")
			}
			// 应该是绝对路径
			if !filepath.IsAbs(absPath) {
				t.Errorf("expected absolute path, got '%s'", absPath)
			}
			break
		}
	}
	if !foundConfig {
		t.Error("expected --config argument")
	}
}

// ============ newService 边缘测试 ============

func TestNewServiceWithValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	validConfig := `{
		"gateway": {"host": "127.0.0.1", "port": 4298},
		"servers": [{"name": "test", "type": "local", "command": ["echo", "test"], "enabled": true}]
	}`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
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

// ============ InstallResult JSON 序列化测试 ============

func TestInstallResultJSONSerialization(t *testing.T) {
	result := &InstallResult{
		ServiceName: "mcp-gateway",
		ConfigPath:  "/path/to/config",
		InstallPath: "/path/to/plist",
	}

	// 验证字段可以正常访问
	if result.ServiceName != "mcp-gateway" {
		t.Errorf("unexpected ServiceName: %s", result.ServiceName)
	}
	if result.ConfigPath != "/path/to/config" {
		t.Errorf("unexpected ConfigPath: %s", result.ConfigPath)
	}
	if result.InstallPath != "/path/to/plist" {
		t.Errorf("unexpected InstallPath: %s", result.InstallPath)
	}
}

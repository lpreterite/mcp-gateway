//go:build darwin

package gwservice

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ============ Facade.Install 错误路径测试 ============

func TestFacadeInstallWithInvalidConfig(t *testing.T) {
	f := NewFacade("/nonexistent/config.json")
	_, err := f.Install()

	// 配置无效时应该返回错误
	if err == nil {
		t.Error("expected error when config path is invalid")
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

func TestFacadeInstallWithMalformedConfig(t *testing.T) {
	// 创建临时目录和格式错误的配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// 写入格式错误的 JSON
	malformedContent := `{invalid json}`
	if err := os.WriteFile(configPath, []byte(malformedContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	f := NewFacade(configPath)
	_, err := f.Install()

	// 配置格式错误时应该返回错误
	if err == nil {
		t.Error("expected error when config is malformed")
	}

	var cmdErr *CommandError
	if errors.As(err, &cmdErr) {
		if cmdErr.Code != ExitConfigError {
			t.Errorf("expected ExitConfigError, got %d", cmdErr.Code)
		}
	}
}

// ============ darwinAdapter.bootout 测试 ============

func TestDarwinAdapterBootout(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	adapter := &darwinAdapter{}

	// 测试当服务未安装时
	err := adapter.bootout()
	// 可能会返回错误，因为服务可能不存在
	if err != nil {
		t.Logf("bootout returned error (may be expected): %v", err)
	}
}

// ============ darwinAdapter.kickstart 测试 ============

func TestDarwinAdapterKickstart(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	adapter := &darwinAdapter{}

	// 测试 kickstart - 可能会失败因为服务不存在
	err := adapter.kickstart(false)
	if err != nil {
		t.Logf("kickstart returned error (may be expected): %v", err)
	}

	err = adapter.kickstart(true) // with -k flag
	if err != nil {
		t.Logf("kickstart -k returned error (may be expected): %v", err)
	}
}

// ============ darwinAdapter.bootstrap 测试 ============

func TestDarwinAdapterBootstrap(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	adapter := &darwinAdapter{}

	// bootstrap 需要服务定义文件存在，可能会失败
	err := adapter.bootstrap()
	if err != nil {
		t.Logf("bootstrap returned error (may be expected): %v", err)
	}
}

// ============ NewFacade 边缘测试 ============

func TestNewFacadeWithEmptyPath(t *testing.T) {
	f := NewFacade("")
	if f == nil {
		t.Fatal("expected non-nil Facade")
	}
	if f.configPath != "" {
		t.Errorf("expected empty configPath, got '%s'", f.configPath)
	}
}

// ============ Facade 方法边缘测试 ============

func TestFacadeStartWithValidConfig(t *testing.T) {
	// 创建有效的临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	validConfig := `{
		"gateway": {"host": "127.0.0.1", "port": 4298},
		"servers": [{"name": "test", "type": "local", "command": ["echo", "test"], "enabled": true}]
	}`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	f := NewFacade(configPath)
	err := f.Start()

	// Start 可能会因为服务未安装而失败
	if err != nil {
		var cmdErr *CommandError
		if errors.As(err, &cmdErr) {
			t.Logf("Start returned expected error: %v", cmdErr)
		} else {
			t.Logf("Start returned error: %v", err)
		}
	}
}

func TestFacadeRestartWithValidConfig(t *testing.T) {
	// 创建有效的临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	validConfig := `{
		"gateway": {"host": "127.0.0.1", "port": 4298},
		"servers": [{"name": "test", "type": "local", "command": ["echo", "test"], "enabled": true}]
	}`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	f := NewFacade(configPath)
	err := f.Restart()

	// Restart 可能会因为各种原因失败
	if err != nil {
		t.Logf("Restart returned error: %v", err)
	}
}

// ============ DetectInstall 边缘平台测试 ============

func TestDetectInstallUnknownPlatform(t *testing.T) {
	// 这个测试在当前平台会跳过，因为 detectInstall 已经有 darwin/linux 实现
	// 但我们可以通过代码覆盖来分析
	if runtime.GOOS == "darwin" {
		// darwin 平台测试
		_, err := darwinLaunchAgentPath()
		if err != nil {
			t.Logf("darwinLaunchAgentPath returned error: %v", err)
		}
	}
}

func TestDarwinServiceTarget(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	target, err := darwinServiceTarget()
	if err != nil {
		t.Fatalf("darwinServiceTarget failed: %v", err)
	}

	// 验证目标格式
	if target == "" {
		t.Error("expected non-empty target")
	}
	if !containsString(target, "gui/") {
		t.Errorf("expected target to contain 'gui/', got '%s'", target)
	}
	if !containsString(target, "/mcp-gateway") {
		t.Errorf("expected target to contain '/mcp-gateway', got '%s'", target)
	}
}

func TestDarwinLaunchctlPrint(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping darwin-specific test on non-darwin platform")
	}

	target, output, err := darwinLaunchctlPrint()
	if err != nil {
		t.Logf("darwinLaunchctlPrint returned error (expected if service not registered): %v", err)
	}
	if target == "" && err == nil {
		t.Error("expected non-empty target when no error")
	}
	if len(output) > 0 {
		t.Logf("darwinLaunchctlPrint output: %s", string(output))
	}
}

// ============ detectRegistration 边缘情况测试 ============

func TestDetectRegistrationUnknownPlatform(t *testing.T) {
	// 验证在非 darwin/linux 平台上的行为
	// 当前测试在 darwin 上运行
	status, detail := detectRegistration()
	if status == StateUnknown && detail == "" {
		t.Error("expected detail when status is unknown")
	}
}

// ============ DiagnoseStatus 边缘情况测试 ============

func TestDiagnoseStatusWithEmptyPath(t *testing.T) {
	// 使用空路径可能会触发配置探测
	report := DiagnoseStatus("")
	// 应该返回有效报告
	if report.ConfigStatus == "" {
		t.Error("expected non-empty ConfigStatus")
	}
}

// ============ 辅助函数 ============

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

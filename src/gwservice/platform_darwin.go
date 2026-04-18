//go:build darwin

package gwservice

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type darwinAdapter struct{}

func (a *darwinAdapter) Start(_ *Facade, report ServiceStatusReport) error {
	if report.InstallStatus != StatePresent {
		return &CommandError{Code: ExitInstallMissing, Message: fmt.Sprintf("cannot start service: service definition not installed (%s)", report.InstallDetail)}
	}
	if report.ProcessStatus == StateRunning {
		return nil
	}
	if report.RegistrationStatus == StateMissing {
		return a.bootstrap()
	}
	if report.RegistrationStatus == StateLoaded {
		if err := a.kickstart(false); err != nil {
			return &CommandError{Code: ExitRegistrationError, Message: fmt.Sprintf("launchctl kickstart failed: %v", err)}
		}
		return nil
	}
	return &CommandError{Code: ExitRegistrationError, Message: fmt.Sprintf("cannot start service: registration status is %s", report.RegistrationStatus)}
}

func (a *darwinAdapter) Stop(_ *Facade, report ServiceStatusReport) error {
	if report.InstallStatus != StatePresent {
		return nil
	}
	if report.RegistrationStatus == StateMissing {
		return nil
	}
	if err := a.bootout(); err != nil && !isLaunchctlMissing(err) {
		return &CommandError{Code: ExitServiceCommandFail, Message: fmt.Sprintf("launchctl bootout failed: %v", err)}
	}
	return nil
}

func (a *darwinAdapter) Restart(_ *Facade, report ServiceStatusReport) error {
	if report.InstallStatus != StatePresent {
		return &CommandError{Code: ExitInstallMissing, Message: fmt.Sprintf("cannot restart service: service definition not installed (%s)", report.InstallDetail)}
	}
	if report.RegistrationStatus == StateMissing {
		return a.bootstrap()
	}
	if err := a.kickstart(true); err == nil {
		return nil
	}
	if err := a.bootout(); err != nil && !isLaunchctlMissing(err) {
		return &CommandError{Code: ExitRegistrationError, Message: fmt.Sprintf("launchctl bootout failed: %v", err)}
	}
	return a.bootstrap()
}

func (a *darwinAdapter) bootstrap() error {
	plistPath, err := darwinLaunchAgentPath()
	if err != nil {
		return err
	}
	target, err := darwinServiceTarget()
	if err != nil {
		return err
	}
	domain := strings.TrimSuffix(target, "/mcp-gateway")
	cmd := exec.Command("launchctl", "bootstrap", domain, plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return &CommandError{Code: ExitRegistrationError, Message: normalizeLaunchctlError(output, err)}
	}
	return nil
}

func (a *darwinAdapter) bootout() error {
	target, err := darwinServiceTarget()
	if err != nil {
		return err
	}
	cmd := exec.Command("launchctl", "bootout", target)
	if output, err := cmd.CombinedOutput(); err != nil {
		return &CommandError{Code: ExitRegistrationError, Message: normalizeLaunchctlError(output, err)}
	}
	return nil
}

func (a *darwinAdapter) kickstart(kill bool) error {
	target, err := darwinServiceTarget()
	if err != nil {
		return err
	}
	args := []string{"kickstart"}
	if kill {
		args = append(args, "-k")
	}
	args = append(args, target)
	cmd := exec.Command("launchctl", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return &CommandError{Code: ExitRegistrationError, Message: normalizeLaunchctlError(output, err)}
	}
	return nil
}

func darwinLaunchAgentPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "Library/LaunchAgents", "mcp-gateway.plist"), nil
}

func darwinServiceTarget() (string, error) {
	uidOutput, err := exec.Command("id", "-u").Output()
	if err != nil {
		return "", fmt.Errorf("无法获取 uid: %w", err)
	}
	uid := strings.TrimSpace(string(uidOutput))
	return fmt.Sprintf("gui/%s/mcp-gateway", uid), nil
}

func darwinLaunchctlPrint() (string, []byte, error) {
	target, err := darwinServiceTarget()
	if err != nil {
		return "", nil, err
	}
	// 验证 target 参数防止命令注入
	if !isValidLaunchctlArg(target) {
		return "", nil, fmt.Errorf("invalid launchctl target: %q", target)
	}
	cmd := exec.Command("launchctl", "print", target)
	output, runErr := cmd.CombinedOutput()
	return target, output, runErr
}

func normalizeLaunchctlError(output []byte, err error) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		text = err.Error()
	}
	return text
}

// isValidLaunchctlArg 验证 launchctl 参数不包含危险字符
func isValidLaunchctlArg(arg string) bool {
	if arg == "" {
		return false
	}
	// 检查危险字符：; | & $ ` \n 等
	dangerous := []string{";", "|", "&", "$", "`", "\\", "\n", "\r", "'", "\"", "!", ">", "<"}
	for _, d := range dangerous {
		if strings.Contains(arg, d) {
			return false
		}
	}
	return true
}

func isLaunchctlMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Could not find service") || strings.Contains(msg, "No such process")
}

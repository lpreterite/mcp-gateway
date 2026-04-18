//go:build windows

package gwservice

import (
	"fmt"
	"os/exec"
	"strings"
)

type windowsAdapter struct{}

func (a *windowsAdapter) Start(_ *Facade, report ServiceStatusReport) error {
	if report.InstallStatus != StatePresent {
		return &CommandError{Code: ExitInstallMissing, Message: fmt.Sprintf("cannot start service: service definition not installed (%s)", report.InstallDetail)}
	}
	if report.ProcessStatus == StateRunning {
		return nil
	}
	return a.sc("start", "mcp-gateway")
}

func (a *windowsAdapter) Stop(_ *Facade, report ServiceStatusReport) error {
	if report.InstallStatus != StatePresent {
		return nil
	}
	return a.sc("stop", "mcp-gateway")
}

func (a *windowsAdapter) Restart(_ *Facade, report ServiceStatusReport) error {
	if report.InstallStatus != StatePresent {
		return &CommandError{Code: ExitInstallMissing, Message: fmt.Sprintf("cannot restart service: service definition not installed (%s)", report.InstallDetail)}
	}
	if err := a.sc("stop", "mcp-gateway"); err != nil {
		// 忽略 stop 错误，继续尝试 start
	}
	return a.sc("start", "mcp-gateway")
}

func (a *windowsAdapter) sc(args ...string) error {
	// 验证所有参数
	for _, arg := range args {
		if !isValidScArg(arg) {
			return &CommandError{Code: ExitServiceCommandFail, Message: fmt.Sprintf("invalid sc argument: %q", arg)}
		}
	}
	cmd := exec.Command("sc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			text = err.Error()
		}
		return &CommandError{Code: ExitServiceCommandFail, Message: fmt.Sprintf("sc %s failed: %s", strings.Join(args, " "), text)}
	}
	return nil
}

// isValidScArg 验证 sc 参数不包含危险字符
func isValidScArg(arg string) bool {
	if arg == "" {
		return false
	}
	// 检查危险字符：; | & $ ` \n 等
	dangerous := []string{";", "|", "&", "$", "`", "\n", "\r", "'", "\"", "!", ">", "<"}
	for _, d := range dangerous {
		if strings.Contains(arg, d) {
			return false
		}
	}
	return true
}

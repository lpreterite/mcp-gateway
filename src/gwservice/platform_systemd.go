//go:build linux

package gwservice

import (
	"fmt"
	"os/exec"
	"strings"
)

type linuxAdapter struct{}

func (a *linuxAdapter) Start(_ *Facade, report ServiceStatusReport) error {
	if report.InstallStatus != StatePresent {
		return &CommandError{Code: ExitInstallMissing, Message: fmt.Sprintf("cannot start service: service definition not installed (%s)", report.InstallDetail)}
	}
	if report.ProcessStatus == StateRunning {
		return nil
	}
	if report.RegistrationStatus == StateMissing {
		if err := a.daemonReload(); err != nil {
			return err
		}
	}
	return a.systemctl("start", "mcp-gateway.service")
}

func (a *linuxAdapter) Stop(_ *Facade, report ServiceStatusReport) error {
	if report.InstallStatus != StatePresent {
		return nil
	}
	return a.systemctl("stop", "mcp-gateway.service")
}

func (a *linuxAdapter) Restart(_ *Facade, report ServiceStatusReport) error {
	if report.InstallStatus != StatePresent {
		return &CommandError{Code: ExitInstallMissing, Message: fmt.Sprintf("cannot restart service: service definition not installed (%s)", report.InstallDetail)}
	}
	if report.RegistrationStatus == StateMissing {
		if err := a.daemonReload(); err != nil {
			return err
		}
		return a.systemctl("start", "mcp-gateway.service")
	}
	if err := a.systemctl("restart", "mcp-gateway.service"); err == nil {
		return nil
	}
	if err := a.daemonReload(); err != nil {
		return err
	}
	return a.systemctl("restart", "mcp-gateway.service")
}

func (a *linuxAdapter) daemonReload() error {
	return a.systemctl("daemon-reload")
}

// isValidSystemctlArg 验证 systemctl 参数不包含危险字符
func isValidSystemctlArg(arg string) bool {
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

func (a *linuxAdapter) systemctl(args ...string) error {
	// 验证所有参数
	for _, arg := range args {
		if !isValidSystemctlArg(arg) {
			return &CommandError{Code: ExitServiceCommandFail, Message: fmt.Sprintf("invalid systemctl argument: %q", arg)}
		}
	}
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			text = err.Error()
		}
		return &CommandError{Code: ExitServiceCommandFail, Message: fmt.Sprintf("systemctl %s failed: %s", strings.Join(args, " "), text)}
	}
	return nil
}

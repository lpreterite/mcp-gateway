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

func (a *linuxAdapter) systemctl(args ...string) error {
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

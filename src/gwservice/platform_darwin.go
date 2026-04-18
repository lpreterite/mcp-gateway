//go:build darwin

package gwservice

import (
	"fmt"
	"os/exec"
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

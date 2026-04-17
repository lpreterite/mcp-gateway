package gwservice

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/lpreterite/mcp-gateway/src/config"
)

type ServiceStatusReport struct {
	ConfigPath         string
	ConfigStatus       ServiceState
	ConfigDetail       string
	InstallStatus      ServiceState
	InstallDetail      string
	RegistrationStatus ServiceState
	RegistrationDetail string
	ProcessStatus      ServiceState
	ProcessDetail      string
	HealthStatus       ServiceState
	HealthDetail       string
	SuggestedAction    SuggestedAction
}

func DiagnoseStatus(configPath string) ServiceStatusReport {
	report := ServiceStatusReport{
		ConfigStatus:       StateUnknown,
		InstallStatus:      StateUnknown,
		RegistrationStatus: StateUnknown,
		ProcessStatus:      StateUnknown,
		HealthStatus:       StateUnknown,
		SuggestedAction: SuggestedAction{
			Code:    ActionNone,
			Message: "none",
		},
	}

	resolvedPath, cfg, err := config.Inspect(configPath)
	report.ConfigPath = resolvedPath
	if err != nil {
		report.ConfigStatus = StateInvalid
		report.ConfigDetail = err.Error()
		report.InstallStatus = StateUnknown
		report.InstallDetail = "配置不可用，跳过服务定义检查"
		report.RegistrationStatus = StateUnknown
		report.RegistrationDetail = "配置不可用，跳过平台注册检查"
		report.ProcessStatus = StateUnknown
		report.ProcessDetail = "配置不可用，跳过进程检查"
		report.HealthStatus = StateUnknown
		report.HealthDetail = "配置不可用，跳过端口检查"
		report.SuggestedAction = SuggestedAction{Code: ActionFixConfig, Message: "修复配置文件后再执行 service status"}
		return report
	}

	report.ConfigStatus = StateValid
	if resolvedPath != "" {
		report.ConfigDetail = resolvedPath
	} else {
		report.ConfigDetail = "使用了内置路径探测，但未解析出最终路径"
	}

	if installStatus, installDetail := detectInstall(); installStatus != StateUnknown {
		report.InstallStatus = installStatus
		report.InstallDetail = installDetail
	}

	if registrationStatus, registrationDetail := detectRegistration(); registrationStatus != StateUnknown {
		report.RegistrationStatus = registrationStatus
		report.RegistrationDetail = registrationDetail
	}

	if report.ProcessStatus == StateUnknown {
		report.ProcessStatus = StateNotRunning
		report.ProcessDetail = "未做独立进程扫描，使用健康探测补充判断"
	}

	if runtime.GOOS == "darwin" && report.RegistrationStatus == StateLoaded {
		if processStatus, processDetail := detectDarwinProcess(); processStatus != StateUnknown {
			report.ProcessStatus = processStatus
			report.ProcessDetail = processDetail
		}
	}

	if cfg != nil && cfg.Gateway != nil {
		addr := net.JoinHostPort(cfg.Gateway.Host, fmt.Sprintf("%d", cfg.Gateway.Port))
		if cfg.Gateway.Host == "0.0.0.0" {
			addr = net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", cfg.Gateway.Port))
		}

		conn, err := net.DialTimeout("tcp", addr, defaultDialTimeout)
		if err == nil {
			_ = conn.Close()
			report.ProcessStatus = StateRunning
			report.ProcessDetail = fmt.Sprintf("检测到监听地址 %s", addr)
			report.HealthStatus = StateReachable
			report.HealthDetail = fmt.Sprintf("TCP 端口可连接: %s", addr)
		} else {
			report.HealthStatus = StateUnreachable
			report.HealthDetail = fmt.Sprintf("TCP 连接失败: %s (%v)", addr, err)
		}
	}

	if report.InstallStatus == StateMissing {
		report.SuggestedAction = SuggestedAction{Code: ActionInstallService, Message: "服务尚未安装，建议先执行 mcp-gateway service install"}
	} else if report.InstallStatus == StatePresent && report.RegistrationStatus == StateMissing {
		report.SuggestedAction = suggestedRegistrationFix()
	} else if report.ProcessStatus == StateRunning && report.HealthStatus == StateUnreachable {
		report.SuggestedAction = SuggestedAction{Code: ActionWaitReady, Message: "服务进程已运行，但网关端口尚未就绪；请等待初始化完成后重试"}
	} else if report.ConfigStatus == StateValid && report.ProcessStatus != StateRunning {
		report.SuggestedAction = SuggestedAction{Code: ActionStartService, Message: "执行 mcp-gateway service start 尝试拉起服务"}
	}

	return report
}

func (r ServiceStatusReport) Format() string {
	lines := []string{
		fmt.Sprintf("Config: %s", formatLine(r.ConfigStatus, r.ConfigDetail)),
		fmt.Sprintf("Install: %s", formatLine(r.InstallStatus, r.InstallDetail)),
		fmt.Sprintf("Registration: %s", formatLine(r.RegistrationStatus, r.RegistrationDetail)),
		fmt.Sprintf("Process: %s", formatLine(r.ProcessStatus, r.ProcessDetail)),
		fmt.Sprintf("Health: %s", formatLine(r.HealthStatus, r.HealthDetail)),
		fmt.Sprintf("Suggested action: %s", r.SuggestedAction.Message),
	}

	return strings.Join(lines, "\n")
}

func formatLine(status ServiceState, detail string) string {
	if detail == "" {
		return string(status)
	}
	return fmt.Sprintf("%s (%s)", status, detail)
}

func detectInstall() (ServiceState, string) {
	switch runtime.GOOS {
	case "darwin":
		plistPath, err := darwinLaunchAgentPath()
		if err != nil {
			return StateUnknown, err.Error()
		}
		if _, err := os.Stat(plistPath); err == nil {
			return StatePresent, plistPath
		}
		return StateMissing, plistPath
	case "linux":
		paths := []string{"/etc/systemd/system/mcp-gateway.service", "/usr/lib/systemd/system/mcp-gateway.service"}
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				return StatePresent, path
			}
		}
		return StateMissing, strings.Join(paths, ", ")
	default:
		return StateUnknown, "当前平台未实现安装层探测"
	}
}

func detectRegistration() (ServiceState, string) {
	switch runtime.GOOS {
	case "darwin":
		target, output, err := darwinLaunchctlPrint()
		if err != nil {
			text := strings.TrimSpace(string(output))
			if text == "" {
				text = err.Error()
			}
			if strings.Contains(text, "Could not find service") {
				return StateMissing, target
			}
			return StateUnknown, text
		}
		return StateLoaded, target
	case "linux":
		cmd := exec.Command("systemctl", "show", "mcp-gateway.service", "--property=LoadState,ActiveState,SubState")
		output, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(output))
		if err != nil {
			if strings.Contains(text, "could not be found") || strings.Contains(text, "not-found") {
				return StateMissing, "mcp-gateway.service"
			}
			if text == "" {
				text = err.Error()
			}
			return StateUnknown, text
		}
		if strings.Contains(text, "LoadState=not-found") {
			return StateMissing, text
		}
		if strings.Contains(text, "ActiveState=failed") {
			return StateLoaded, text
		}
		if strings.Contains(text, "ActiveState=inactive") {
			return StateLoaded, text
		}
		if strings.Contains(text, "ActiveState=active") {
			return StateLoaded, text
		}
		return StateLoaded, text
	default:
		return StateUnknown, "当前平台未实现注册层探测"
	}
}

func suggestedRegistrationFix() SuggestedAction {
	if runtime.GOOS == "darwin" {
		return SuggestedAction{Code: ActionReloadRegistration, Message: "服务定义已存在但未加载，建议执行 mcp-gateway service start 或用 launchctl bootstrap 重新加载"}
	}
	if runtime.GOOS == "linux" {
		return SuggestedAction{Code: ActionReloadRegistration, Message: "服务定义已存在但未被 systemd 加载，建议执行 systemctl daemon-reload 后再启动"}
	}
	return SuggestedAction{Code: ActionReloadRegistration, Message: "检查服务管理器中的注册状态后重新启动服务"}
}

func detectDarwinProcess() (ServiceState, string) {
	_, output, err := darwinLaunchctlPrint()
	if err != nil {
		return StateUnknown, ""
	}
	text := string(output)
	state := parseLaunchctlField(text, "state")
	pidText := parseLaunchctlField(text, "pid")
	if state != "running" {
		return StateNotRunning, fmt.Sprintf("launchctl state=%s", state)
	}
	if pidText != "" {
		if pid, convErr := strconv.Atoi(pidText); convErr == nil && pid > 0 {
			return StateRunning, fmt.Sprintf("launchctl state=running, pid=%d", pid)
		}
	}
	return StateRunning, "launchctl state=running"
}

func parseLaunchctlField(text, field string) string {
	prefix := field + " = "
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}

const defaultDialTimeout = 2_000_000_000

package gwservice

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func darwinLaunchAgentPath() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("darwinLaunchAgentPath is only available on darwin")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "Library/LaunchAgents", "mcp-gateway.plist"), nil
}

func darwinServiceTarget() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("darwinServiceTarget is only available on darwin")
	}
	uidOutput, err := exec.Command("id", "-u").Output()
	if err != nil {
		return "", fmt.Errorf("无法获取 uid: %w", err)
	}
	uid := strings.TrimSpace(string(uidOutput))
	return fmt.Sprintf("gui/%s/mcp-gateway", uid), nil
}

func darwinLaunchctlPrint() (string, []byte, error) {
	if runtime.GOOS != "darwin" {
		return "", nil, fmt.Errorf("darwinLaunchctlPrint is only available on darwin")
	}
	target, err := darwinServiceTarget()
	if err != nil {
		return "", nil, err
	}
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

func isValidLaunchctlArg(arg string) bool {
	if arg == "" {
		return false
	}
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

package gwservice

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

func isLaunchctlMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Could not find service") || strings.Contains(msg, "No such process")
}

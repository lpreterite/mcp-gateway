package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectSystemPaths 尝试检测常见的二进制路径，返回连接好的 PATH 字符串
func DetectSystemPaths() string {
	paths := []string{}

	// 1. 动态检测 Homebrew 前缀
	brewPrefix := "/opt/homebrew"
	if out, err := exec.Command("brew", "--prefix").Output(); err == nil {
		brewPrefix = strings.TrimSpace(string(out))
	}

	if info, err := os.Stat(filepath.Join(brewPrefix, "bin")); err == nil && info.IsDir() {
		paths = append(paths, filepath.Join(brewPrefix, "bin"))
	}
	if info, err := os.Stat(filepath.Join(brewPrefix, "sbin")); err == nil && info.IsDir() {
		paths = append(paths, filepath.Join(brewPrefix, "sbin"))
	}

	// 2. 常见的其他 Homebrew/包管理器路径
	commonPrefixes := []string{"/opt/homebrew", "/usr/local", "/home/linuxbrew/.linuxbrew"}
	for _, p := range commonPrefixes {
		if p == brewPrefix {
			continue
		}
		bin := filepath.Join(p, "bin")
		if info, err := os.Stat(bin); err == nil && info.IsDir() {
			paths = append(paths, bin)
		}
		sbin := filepath.Join(p, "sbin")
		if info, err := os.Stat(sbin); err == nil && info.IsDir() {
			paths = append(paths, sbin)
		}
	}

	// 3. 检测 Node.js (nvm)
	nvmDir := os.Getenv("NVM_DIR")
	if nvmDir == "" {
		nvmDir = filepath.Join(os.Getenv("HOME"), ".nvm")
	}
	nodeVersionsDir := filepath.Join(nvmDir, "versions/node")
	if entries, err := os.ReadDir(nodeVersionsDir); err == nil {
		var latest string
		for _, entry := range entries {
			if entry.IsDir() {
				latest = entry.Name() // 简单取最后一个，实际生产中可做版本排序
			}
		}
		if latest != "" {
			paths = append(paths, filepath.Join(nodeVersionsDir, latest, "bin"))
		}
	}

	// 4. 标准系统路径
	systemPaths := []string{"/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"}
	paths = append(paths, systemPaths...)

	// 去重
	uniquePaths := []string{}
	seen := make(map[string]bool)
	for _, p := range paths {
		if !seen[p] {
			uniquePaths = append(uniquePaths, p)
			seen[p] = true
		}
	}

	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	return strings.Join(uniquePaths, sep)
}

// GetDefaultLogDir 返回平台的默认日志目录
func GetDefaultLogDir() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(os.Getenv("HOME"), "Library/Logs")
	}
	if runtime.GOOS == "linux" {
		return "/var/log"
	}
	return os.TempDir()
}

// GetExecutablePath 返回当前可执行文件的绝对路径
func GetExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}

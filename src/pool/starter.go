package pool

import (
	"context"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ProcessStarter 进程启动器接口
type ProcessStarter interface {
	Start(ctx context.Context, command string, args []string, env map[string]string) (Process, error)
}

// Process 进程接口
type Process interface {
	Wait() error
	Stdin() io.Writer
	Stdout() io.Reader
	Stderr() io.Reader
	Kill() error
}

// DefaultProcessStarter 默认进程启动器实现
type DefaultProcessStarter struct{}

// Start 启动一个新进程
func (s *DefaultProcessStarter) Start(ctx context.Context, command string, args []string, env map[string]string) (Process, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = buildChildEnv(env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &defaultProcess{cmd: cmd, stdin: stdin, stdout: stdout, stderr: stderr}, nil
}

// buildChildEnv 构建子进程环境变量，确保关键变量存在
func buildChildEnv(extra map[string]string) []string {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		if idx := strings.IndexByte(e, '='); idx > 0 {
			envMap[e[:idx]] = e[idx+1:]
		}
	}

	ensureEssentialEnv(envMap)

	for k, v := range extra {
		envMap[k] = v
	}

	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, k+"="+v)
	}
	return result
}

var (
	loginShellEnv     map[string]string
	loginShellEnvOnce sync.Once
)

func fetchLoginShellEnv() map[string]string {
	shell := "/bin/zsh"
	shellArg := "-l"
	if runtime.GOOS != "darwin" {
		shell = "/bin/bash"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, shellArg, "-c", "env")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	envMap := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		if idx := strings.IndexByte(line, '='); idx > 0 {
			envMap[line[:idx]] = line[idx+1:]
		}
	}

	if len(envMap) == 0 {
		return nil
	}

	return envMap
}

func ensureEssentialEnv(envMap map[string]string) {
	loginShellEnvOnce.Do(func() {
		loginShellEnv = fetchLoginShellEnv()
	})

	for k, v := range loginShellEnv {
		if _, ok := envMap[k]; !ok {
			envMap[k] = v
		}
	}

	if _, ok := envMap["HOME"]; !ok {
		if u, err := user.Current(); err == nil && u.HomeDir != "" {
			envMap["HOME"] = u.HomeDir
		}
	}

	if _, ok := envMap["USER"]; !ok {
		if u, err := user.Current(); err == nil && u.Username != "" {
			envMap["USER"] = u.Username
		}
	}

	if _, ok := envMap["PATH"]; !ok {
		envMap["PATH"] = "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
		if runtime.GOOS == "darwin" {
			homebrewPaths := []string{"/opt/homebrew/bin", "/opt/homebrew/sbin", "/usr/local/bin"}
			for _, p := range homebrewPaths {
				if info, err := os.Stat(p); err == nil && info.IsDir() {
					envMap["PATH"] = p + ":" + envMap["PATH"]
				}
			}
			if home := envMap["HOME"]; home != "" {
				nvmNodeBin := filepath.Join(home, ".nvm/versions/node")
				if entries, err := os.ReadDir(nvmNodeBin); err == nil {
					var latest string
					for _, entry := range entries {
						if entry.IsDir() {
							latest = entry.Name()
						}
					}
					if latest != "" {
						nvmBin := filepath.Join(nvmNodeBin, latest, "bin")
						envMap["PATH"] = nvmBin + ":" + envMap["PATH"]
					}
				}
			}
		}
	}

	if _, ok := envMap["TMPDIR"]; !ok {
		switch runtime.GOOS {
		case "darwin":
			envMap["TMPDIR"] = "/tmp"
		case "windows":
			if tmp := os.Getenv("TEMP"); tmp != "" {
				envMap["TMPDIR"] = tmp
			} else {
				envMap["TMPDIR"] = os.TempDir()
			}
		default:
			envMap["TMPDIR"] = "/tmp"
		}
	}

	if _, ok := envMap["LANG"]; !ok {
		envMap["LANG"] = "en_US.UTF-8"
	}

	if _, ok := envMap["TERM"]; !ok {
		envMap["TERM"] = "xterm-256color"
	}

	if runtime.GOOS == "darwin" {
		if _, ok := envMap["XDG_STATE_HOME"]; !ok {
			if home := envMap["HOME"]; home != "" {
				envMap["XDG_STATE_HOME"] = filepath.Join(home, ".local/state")
			}
		}
		if _, ok := envMap["XDG_CACHE_HOME"]; !ok {
			if home := envMap["HOME"]; home != "" {
				envMap["XDG_CACHE_HOME"] = filepath.Join(home, ".cache")
			}
		}
		if _, ok := envMap["XDG_CONFIG_HOME"]; !ok {
			if home := envMap["HOME"]; home != "" {
				envMap["XDG_CONFIG_HOME"] = filepath.Join(home, ".config")
			}
		}
	}
}

// defaultProcess 默认进程实现
type defaultProcess struct {
	cmd    *exec.Cmd
	stdin  io.Writer
	stdout io.Reader
	stderr io.Reader
}

func (p *defaultProcess) Wait() error       { return p.cmd.Wait() }
func (p *defaultProcess) Stdin() io.Writer  { return p.stdin }
func (p *defaultProcess) Stdout() io.Reader { return p.stdout }
func (p *defaultProcess) Stderr() io.Reader { return p.stderr }
func (p *defaultProcess) Kill() error       { return p.cmd.Process.Kill() }

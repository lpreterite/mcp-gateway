package pool

import (
	"context"
	"io"
	"os/exec"
)

// ProcessStarter 进程启动器接口
type ProcessStarter interface {
	Start(ctx context.Context, command string, args []string) (Process, error)
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
func (s *DefaultProcessStarter) Start(ctx context.Context, command string, args []string) (Process, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	// 获取 stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	// 获取 stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// 获取 stderr pipe
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &defaultProcess{cmd: cmd, stdin: stdin, stdout: stdout, stderr: stderr}, nil
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

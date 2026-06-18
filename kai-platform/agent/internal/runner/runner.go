package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	WorkDir  string
	RepoDir  string
	AgentDir string
	Timeout  time.Duration
}

type Result struct {
	Success  bool
	ExitCode int
	Duration time.Duration
}

type LineHandler func(line string, source string)

type Policy struct {
	AllowedTools    []string `json:"allowed_tools"`
	AllowedCommands []string `json:"allowed_commands"`
	AllowedDirs     []string `json:"allowed_dirs"`
}

type WriteConfigInput struct {
	ConfigBlob string  `json:"config_blob"`
	Policy     *Policy `json:"policy,omitempty"`
}

type Runner interface {
	Run(ctx context.Context, prompt string, onLine LineHandler) (*Result, error)
	Stop()
	WriteConfig(input WriteConfigInput) error
}

var registry = make(map[string]func(Config) Runner)

func Register(name string, factory func(Config) Runner) {
	registry[name] = factory
}

func New(runnerType string, cfg Config) (Runner, error) {
	factory, ok := registry[runnerType]
	if !ok {
		return nil, fmt.Errorf("unknown runner: %q", runnerType)
	}
	return factory(cfg), nil
}

type baseRunner struct {
	cfg    Config
	cmd    *exec.Cmd
	cancel context.CancelFunc
	env    []string
}

func (r *baseRunner) Run(ctx context.Context, name string, args []string, prompt string, onLine LineHandler) (*Result, error) {
	summary := fmt.Sprintf("[runner] executing: %s %s", name, strings.Join(args, " "))
	if len(summary) > 200 {
		summary = summary[:197] + "..."
	}
	onLine(summary, "system")

	runCtx, cancel := context.WithTimeout(ctx, r.cfg.Timeout)
	defer cancel()
	r.cancel = cancel

	cmd := exec.CommandContext(runCtx, name, args...)
	cmd.Dir = r.cfg.RepoDir
	if r.env != nil {
		cmd.Env = r.env
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", name, err)
	}

	r.cmd = cmd
	start := time.Now()

	done := make(chan struct{}, 2)
	go streamLines(stdout, "stdout", onLine, done)
	go streamLines(stderr, "stderr", onLine, done)

	<-done
	<-done

	err = cmd.Wait()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("%s execution error: %w", name, err)
		}
	}

	return &Result{
		Success:  err == nil,
		ExitCode: exitCode,
		Duration: time.Since(start),
	}, nil
}

func (r *baseRunner) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}

func streamLines(reader io.Reader, source string, handler LineHandler, done chan struct{}) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)
	for scanner.Scan() {
		handler(scanner.Text(), source)
	}
	done <- struct{}{}
}

func TruncatePrompt(prompt string) string {
	firstLine := strings.SplitN(strings.TrimSpace(prompt), "\n", 2)[0]
	if len(firstLine) > 72 {
		return firstLine[:69] + "..."
	}
	return firstLine
}

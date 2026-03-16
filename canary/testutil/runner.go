// Package testutil provides utilities for running canary tests against the compiled Stripe CLI binary.
package testutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultTimeout is the default timeout for command execution.
const DefaultTimeout = 30 * time.Second

// Runner executes the Stripe CLI binary with isolated configuration.
type Runner struct {
	// BinaryPath is the path to the stripe binary.
	BinaryPath string

	// ConfigDir is an isolated config directory for this test.
	// If empty, a temp directory will be created.
	ConfigDir string

	// Env contains additional environment variables to set.
	Env map[string]string

	// Timeout is the command execution timeout.
	// If zero, DefaultTimeout is used.
	Timeout time.Duration
}

// Result contains the output of a command execution.
type Result struct {
	// Stdout is the standard output.
	Stdout string

	// Stderr is the standard error output.
	Stderr string

	// ExitCode is the process exit code.
	ExitCode int
}

// RunnerOption is a functional option for configuring a Runner at construction time.
type RunnerOption func(*Runner)

// WithConfigDir returns a RunnerOption that sets the config directory.
func WithConfigDir(dir string) RunnerOption {
	return func(r *Runner) {
		r.ConfigDir = dir
	}
}

// WithEnv returns a RunnerOption that merges additional environment variables.
func WithEnv(env map[string]string) RunnerOption {
	return func(r *Runner) {
		for k, v := range env {
			r.Env[k] = v
		}
	}
}

// WithTimeout returns a RunnerOption that sets the command execution timeout.
func WithTimeout(timeout time.Duration) RunnerOption {
	return func(r *Runner) {
		r.Timeout = timeout
	}
}

// NewRunner creates a new Runner with the binary path from STRIPE_CLI_BINARY env var.
// Options are applied after initializing defaults.
// Returns an error if the env var is not set or the binary doesn't exist.
func NewRunner(opts ...RunnerOption) (*Runner, error) {
	binaryPath := os.Getenv("STRIPE_CLI_BINARY")
	if binaryPath == "" {
		return nil, fmt.Errorf("STRIPE_CLI_BINARY environment variable not set")
	}

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("binary not found at %s", binaryPath)
	}

	r := &Runner{
		BinaryPath: binaryPath,
		Timeout:    DefaultTimeout,
		Env:        make(map[string]string),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

// WithConfigDir creates a copy of the runner with an isolated config directory.
func (r *Runner) WithConfigDir(dir string) *Runner {
	newRunner := *r
	newRunner.ConfigDir = dir
	newRunner.Env = make(map[string]string)
	for k, v := range r.Env {
		newRunner.Env[k] = v
	}
	return &newRunner
}

// WithEnv creates a copy of the runner with additional environment variables.
func (r *Runner) WithEnv(env map[string]string) *Runner {
	newRunner := *r
	newRunner.Env = make(map[string]string)
	for k, v := range r.Env {
		newRunner.Env[k] = v
	}
	for k, v := range env {
		newRunner.Env[k] = v
	}
	return &newRunner
}

// WithTimeout creates a copy of the runner with a different timeout.
func (r *Runner) WithTimeout(timeout time.Duration) *Runner {
	newRunner := *r
	newRunner.Env = make(map[string]string)
	for k, v := range r.Env {
		newRunner.Env[k] = v
	}
	newRunner.Timeout = timeout
	return &newRunner
}

// Run executes the stripe binary with the given arguments.
func (r *Runner) Run(args ...string) (*Result, error) {
	timeout := r.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return r.RunContext(ctx, args...)
}

// RunContext executes the stripe binary with the given context and arguments.
func (r *Runner) RunContext(ctx context.Context, args ...string) (*Result, error) {
	cmd := exec.CommandContext(ctx, r.BinaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Build environment
	cmd.Env = os.Environ()

	// Add config dir if specified
	if r.ConfigDir != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("XDG_CONFIG_HOME=%s", r.ConfigDir))
		// For macOS/Windows compatibility
		cmd.Env = append(cmd.Env, fmt.Sprintf("STRIPE_CONFIG_DIR=%s", filepath.Join(r.ConfigDir, "stripe")))
	}

	// Add custom environment variables
	for k, v := range r.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Disable telemetry by default
	cmd.Env = append(cmd.Env, "STRIPE_CLI_TELEMETRY_OPTOUT=1")

	err := cmd.Run()

	result := &Result{
		Stdout:   normalizeLineEndings(stdout.String()),
		Stderr:   normalizeLineEndings(stderr.String()),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			return result, fmt.Errorf("command timed out after %v", r.Timeout)
		} else {
			return result, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return result, nil
}

// normalizeLineEndings converts Windows CRLF to Unix LF for cross-platform consistency.
func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// CreateTempConfigDir creates a temporary directory for isolated configuration.
// The caller is responsible for cleaning up the directory.
// This also creates the stripe subdirectory and an empty config.toml file
// so that config commands work properly.
func CreateTempConfigDir(prefix string) (string, error) {
	dir, err := os.MkdirTemp("", fmt.Sprintf("stripe-canary-%s-", prefix))
	if err != nil {
		return "", err
	}

	// Create stripe config subdirectory
	stripeDir := filepath.Join(dir, "stripe")
	if err := os.MkdirAll(stripeDir, 0755); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("failed to create stripe config dir: %w", err)
	}

	// Create empty config.toml so config commands work
	configFile := filepath.Join(stripeDir, "config.toml")
	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("failed to create config.toml: %w", err)
	}

	return dir, nil
}

// HasAPIKey returns true if STRIPE_API_KEY is set in the environment.
func HasAPIKey() bool {
	return os.Getenv("STRIPE_API_KEY") != ""
}

// GetAPIKey returns the STRIPE_API_KEY from the environment.
func GetAPIKey() string {
	return os.Getenv("STRIPE_API_KEY")
}

// BackgroundProcess represents a running CLI command in the background.
type BackgroundProcess struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	done   chan error
	mu     sync.Mutex
}

// RunBackground starts the CLI in background and returns immediately.
// The returned BackgroundProcess can be used to wait for output, get current output,
// or stop the process.
func (r *Runner) RunBackground(args ...string) (*BackgroundProcess, error) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, r.BinaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Build environment
	cmd.Env = os.Environ()

	// Add config dir if specified
	if r.ConfigDir != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("XDG_CONFIG_HOME=%s", r.ConfigDir))
		cmd.Env = append(cmd.Env, fmt.Sprintf("STRIPE_CONFIG_DIR=%s", filepath.Join(r.ConfigDir, "stripe")))
	}

	// Add custom environment variables
	for k, v := range r.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Disable telemetry by default
	cmd.Env = append(cmd.Env, "STRIPE_CLI_TELEMETRY_OPTOUT=1")

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	bp := &BackgroundProcess{
		cmd:    cmd,
		cancel: cancel,
		stdout: &stdout,
		stderr: &stderr,
		done:   make(chan error, 1),
	}

	// Wait for command completion in background
	go func() {
		bp.done <- cmd.Wait()
	}()

	return bp, nil
}

// WaitForOutput waits until combined output contains the expected string or timeout.
func (bp *BackgroundProcess) WaitForOutput(contains string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	checkInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		bp.mu.Lock()
		combined := bp.stdout.String() + bp.stderr.String()
		bp.mu.Unlock()

		if strings.Contains(combined, contains) {
			return nil
		}

		// Check if process has exited
		select {
		case err := <-bp.done:
			// Process exited, check one more time
			bp.mu.Lock()
			combined = bp.stdout.String() + bp.stderr.String()
			bp.mu.Unlock()
			if strings.Contains(combined, contains) {
				// Put the error back for Stop() to retrieve
				bp.done <- err
				return nil
			}
			return fmt.Errorf("process exited before output appeared: %v (output: %s)", err, combined)
		default:
			// Process still running, continue waiting
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("timeout waiting for output containing %q", contains)
}

// GetOutput returns the current stdout and stderr contents.
func (bp *BackgroundProcess) GetOutput() (stdout, stderr string) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.stdout.String(), bp.stderr.String()
}

// Stop kills the process and returns the final result.
func (bp *BackgroundProcess) Stop() (*Result, error) {
	// Cancel the context to kill the process
	bp.cancel()

	// Wait for the process to exit
	var exitErr error
	select {
	case exitErr = <-bp.done:
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout waiting for process to stop")
	}

	bp.mu.Lock()
	result := &Result{
		Stdout:   normalizeLineEndings(bp.stdout.String()),
		Stderr:   normalizeLineEndings(bp.stderr.String()),
		ExitCode: 0,
	}
	bp.mu.Unlock()

	if exitErr != nil {
		if exitError, ok := exitErr.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
		// Don't return error for expected signal termination
	}

	return result, nil
}

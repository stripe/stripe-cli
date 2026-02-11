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

// NewRunner creates a new Runner with the binary path from STRIPE_CLI_BINARY env var.
// Returns an error if the env var is not set or the binary doesn't exist.
func NewRunner() (*Runner, error) {
	binaryPath := os.Getenv("STRIPE_CLI_BINARY")
	if binaryPath == "" {
		return nil, fmt.Errorf("STRIPE_CLI_BINARY environment variable not set")
	}

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("binary not found at %s", binaryPath)
	}

	return &Runner{
		BinaryPath: binaryPath,
		Timeout:    DefaultTimeout,
		Env:        make(map[string]string),
	}, nil
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
func CreateTempConfigDir(prefix string) (string, error) {
	return os.MkdirTemp("", fmt.Sprintf("stripe-canary-%s-", prefix))
}

// HasAPIKey returns true if STRIPE_API_KEY is set in the environment.
func HasAPIKey() bool {
	return os.Getenv("STRIPE_API_KEY") != ""
}

// GetAPIKey returns the STRIPE_API_KEY from the environment.
func GetAPIKey() string {
	return os.Getenv("STRIPE_API_KEY")
}

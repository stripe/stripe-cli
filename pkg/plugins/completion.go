package plugins

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
)

const completionTimeout = 3 * time.Second

// GetPluginCompletions invokes a plugin binary with Cobra's __complete protocol
// to get dynamic shell completions. It returns completions and a shell directive.
// On any error, it returns nil completions and ShellCompDirectiveError.
func GetPluginCompletions(ctx context.Context, cfg config.IConfig, fs afero.Fs, pluginName string, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	logger := log.WithFields(log.Fields{
		"prefix": "plugins.completion.GetPluginCompletions",
	})

	plugin, err := LookUpPlugin(ctx, cfg, fs, pluginName)
	if err != nil {
		logger.Debugf("Could not look up plugin %s: %s", pluginName, err)
		return nil, cobra.ShellCompDirectiveError
	}

	binaryPath, err := resolvePluginBinary(cfg, &plugin)
	if err != nil {
		logger.Debugf("Could not resolve binary for plugin %s: %s", pluginName, err)
		return nil, cobra.ShellCompDirectiveError
	}

	// Build the __complete command args
	completeArgs := make([]string, 0, len(args)+2)
	completeArgs = append(completeArgs, "__complete")
	completeArgs = append(completeArgs, args...)
	completeArgs = append(completeArgs, toComplete)

	// Determine how to invoke the binary (directly or via Node.js)
	cmdPath, cmdArgs := buildPluginCommand(cfg, &plugin, binaryPath, completeArgs)

	// Set timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, completionTimeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, cmdPath, cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		logger.Debugf("Plugin completion command failed for %s: %s (stderr: %s)", pluginName, err, stderr.String())
		return nil, cobra.ShellCompDirectiveError
	}

	completions, directive := parseCompletionOutput(stdout.String())
	return completions, directive
}

// resolvePluginBinary finds the installed binary path for a plugin.
func resolvePluginBinary(cfg config.IConfig, plugin *Plugin) (string, error) {
	var version string

	if PluginsPath != "" {
		version = "local.build.dev"
	} else {
		pluginDir := filepath.Join(getPluginsDir(cfg), plugin.Shortname, "*.*.*")
		existingLocalPlugin, err := filepath.Glob(pluginDir)
		if err != nil {
			return "", err
		}

		if len(existingLocalPlugin) == 0 {
			return "", fmt.Errorf("plugin %s is not installed", plugin.Shortname)
		}

		version = filepath.Base(existingLocalPlugin[0])
	}

	pluginDir := plugin.getPluginInstallPath(cfg, version)
	binaryPath := filepath.Join(pluginDir, plugin.Binary)
	binaryPath += GetBinaryExtension()

	return binaryPath, nil
}

// buildPluginCommand determines the command path and args to invoke a plugin,
// handling Node.js runtime detection.
func buildPluginCommand(cfg config.IConfig, plugin *Plugin, binaryPath string, args []string) (string, []string) {
	version := plugin.LookUpLatestVersion()
	release := plugin.getReleaseForVersion(version)

	if release != nil {
		if nodeVersion, requiresNode := GetRuntimeRequirement(*release); requiresNode {
			nodePath := GetNodeBinaryPath(cfg, nodeVersion)
			if nodePath != "" {
				// Invoke via node: node <binaryPath> <args...>
				return nodePath, append([]string{binaryPath}, args...)
			}
		}
	}

	// Direct execution
	return binaryPath, args
}

// parseCompletionOutput parses the stdout of a Cobra __complete command.
// The format is:
//
//	completion1\tdescription1
//	completion2\tdescription2
//	:<directive>
//
// Returns the completion strings and the ShellCompDirective.
// On parse failure, returns nil and ShellCompDirectiveError.
func parseCompletionOutput(output string) ([]string, cobra.ShellCompDirective) {
	output = strings.TrimRight(output, "\n")
	if output == "" {
		return nil, cobra.ShellCompDirectiveError
	}

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		return nil, cobra.ShellCompDirectiveError
	}

	// Last line must be :<directive>
	lastLine := lines[len(lines)-1]
	if !strings.HasPrefix(lastLine, ":") {
		return nil, cobra.ShellCompDirectiveError
	}

	directiveStr := lastLine[1:]
	directiveInt, err := strconv.Atoi(directiveStr)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	completions := lines[:len(lines)-1]
	if len(completions) == 0 {
		completions = nil
	}

	return completions, cobra.ShellCompDirective(directiveInt)
}

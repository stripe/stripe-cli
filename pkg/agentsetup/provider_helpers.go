package agentsetup

import (
	"context"
	"os/exec"
)

// detectAgentExecutable checks if the executable for the agent is available. defaultPluginStatus is the
// default status to report for the plugin when the executable exists.
func detectAgentExecutable(providerConfig ProviderConfig, defaultPluginStatus string) Status {
	scanner := providerConfig.Scanner.withDefaults()

	status := Status{
		Client:      providerConfig.Client,
		DisplayName: providerConfig.DisplayName,
		Status:      StatusNotDetected,
	}

	path, err := scanner.LookPath(providerConfig.BinaryName)
	if err != nil {
		return status
	}

	status.Detected = true
	status.ExecutablePath = path
	status.Status = defaultPluginStatus
	return status
}

// getPlanByStatus returns the common install plan for providers
func getPlanByStatus(status Status, force bool, installCommand []string, reinstallCommand []string) Plan {
	switch {
	case status.Status == StatusError:
		return Plan{Action: ActionNone}
	case !status.Detected:
		return Plan{Action: ActionNone}
	case status.Plugin.Installed && force:
		return Plan{Action: ActionReinstall, Command: reinstallCommand}
	case status.Plugin.Installed:
		return Plan{Action: ActionNone}
	default:
		return Plan{Action: ActionInstall, Command: installCommand}
	}
}

func runCommandOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

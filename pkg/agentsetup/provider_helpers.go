package agentsetup

// detectExecutable initializes a provider status and checks whether its
// executable is available. detectedStatus is the status to report when the
// executable exists but provider-specific detection has not completed yet.
func detectExecutable(scanner Scanner, client, displayName, binaryName, detectedStatus string) (Status, bool) {
	status := Status{
		Client:      client,
		DisplayName: displayName,
		Status:      StatusNotDetected,
	}

	path, err := scanner.LookPath(binaryName)
	if err != nil {
		return status, false
	}

	status.Detected = true
	status.ExecutablePath = path
	status.Status = detectedStatus
	return status, true
}

// standardPlan returns the common install plan for CLI-configurable providers
func standardPlan(status Status, force bool, installCommand []string, reinstallCommand []string) Plan {
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

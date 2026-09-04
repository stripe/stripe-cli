package agentsetup

import (
	"context"
	"io"
	"path/filepath"
	"runtime"
)

const (
	ClientGrok               = "grok"
	GrokDisplayName          = "Grok Build"
	ClientKimi               = "kimi"
	KimiDisplayName          = "Kimi Code"
	ClientGitHubCopilot      = "github-copilot"
	GitHubCopilotDisplayName = "GitHub Copilot CLI"
	ClientVSCode             = "vscode"
	VSCodeDisplayName        = "Visual Studio Code"
)

type skillsClientProvider struct {
	id          string
	displayName string
	binaries    []string
	scanner     Scanner
	fallbacks   func(Scanner) []string
}

// NewGrokProvider returns a Grok Build setup provider.
func NewGrokProvider(scanner Scanner) Provider {
	return newSkillsClientProvider(ClientGrok, GrokDisplayName, []string{"grok"}, scanner)
}

// NewKimiProvider returns a Kimi Code setup provider.
func NewKimiProvider(scanner Scanner) Provider {
	provider := newSkillsClientProvider(ClientKimi, KimiDisplayName, []string{"kimi"}, scanner)
	provider.fallbacks = kimiFallbackPaths
	return provider
}

// NewGitHubCopilotProvider returns a GitHub Copilot CLI setup provider.
func NewGitHubCopilotProvider(scanner Scanner) Provider {
	return newSkillsClientProvider(ClientGitHubCopilot, GitHubCopilotDisplayName, []string{"copilot"}, scanner)
}

// NewVSCodeProvider returns a Visual Studio Code setup provider.
func NewVSCodeProvider(scanner Scanner) Provider {
	return newSkillsClientProvider(ClientVSCode, VSCodeDisplayName, []string{"code", "code-insiders"}, scanner)
}

func newSkillsClientProvider(id, displayName string, binaries []string, scanner Scanner) skillsClientProvider {
	return skillsClientProvider{
		id:          id,
		displayName: displayName,
		binaries:    binaries,
		scanner:     scanner,
	}
}

func (p skillsClientProvider) ID() string { return p.id }

func (p skillsClientProvider) Detect() Status {
	status := Status{
		Client:      p.id,
		DisplayName: p.displayName,
		Setup:       SetupSkills,
		Status:      StatusNotDetected,
	}

	scanner := p.scanner.withDefaults()
	for _, binary := range p.binaries {
		path, err := scanner.LookPath(binary)
		if err != nil {
			continue
		}
		status.Detected = true
		status.ExecutablePath = path
		status.Status = StatusMissing
		return status
	}
	if p.fallbacks != nil {
		for _, path := range p.fallbacks(scanner) {
			info, err := scanner.Stat(path)
			if err != nil || info.IsDir() || runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
				continue
			}
			status.Detected = true
			status.ExecutablePath = path
			status.Status = StatusMissing
			return status
		}
	}
	return status
}

func kimiFallbackPaths(scanner Scanner) []string {
	home, err := scanner.HomeDir()
	if err != nil {
		return nil
	}
	kimiHome := scanner.Getenv("KIMI_CODE_HOME")
	if kimiHome == "" {
		kimiHome = filepath.Join(home, ".kimi-code")
	}
	binary := "kimi"
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	return []string{
		filepath.Join(kimiHome, "bin", binary),
		filepath.Join(home, ".local", "bin", binary),
		filepath.Join(home, ".kimi", "bin", binary),
	}
}

func (p skillsClientProvider) Plan(status Status, _ bool) Plan {
	if !status.Detected {
		return Plan{Action: ActionNone}
	}
	return Plan{
		Action:  ActionInstallSkills,
		Command: []string{"stripe", "agent", "setup"},
	}
}

func (p skillsClientProvider) Apply(_ context.Context, _ io.Writer, _ Plan) error {
	return nil
}

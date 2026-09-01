package installmethod

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	dpkgFile  = "/var/lib/dpkg/info/stripe.list"
	yumFile   = "/etc/yum.repos.d/stripe.repo"
	dockerEnv = "/.dockerenv"
)

// marker returns the path detection looks for a stamped method at, given the
// executable it found. Built with the same expression detection uses rather than
// written out, because filepath separates with backslashes on Windows: a
// hand-written forward-slash key is never found there.
func marker(exe string) string {
	return filepath.Join(filepath.Dir(exe), methodFile)
}

// stubs describes a simulated host: the value of STRIPE_INSTALL_METHOD, where the
// executable is and what it resolves to, which files exist, and what a stamped
// method file holds.
type stubs struct {
	installMethod string
	exe           string
	exeErr        bool
	links         map[string]string
	files         map[string]string
	present       []string
}

func (s stubs) env() Env {
	return Env{
		Getenv: func(string) string { return s.installMethod },
		Executable: func() (string, error) {
			if s.exeErr {
				return "", errors.New("executable not found")
			}
			return s.exe, nil
		},
		EvalSymlinks: func(path string) (string, error) {
			if target, ok := s.links[path]; ok {
				return target, nil
			}
			return path, nil
		},
		Stat: func(path string) error {
			for _, present := range s.present {
				if present == path {
					return nil
				}
			}
			return errors.New("no such file")
		},
		ReadFile: func(path string) ([]byte, error) {
			if contents, ok := s.files[path]; ok {
				return []byte(contents), nil
			}
			return nil, errors.New("no such file")
		},
	}
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		host     stubs
		expected string
	}{
		{
			name:     "npm_global via env",
			host:     stubs{installMethod: "npm_global", exe: "/usr/local/bin/stripe"},
			expected: NPMGlobal,
		},
		{
			name:     "npm_run via env",
			host:     stubs{installMethod: "npm_run", exe: "/any/path/stripe"},
			expected: NPMRun,
		},
		{
			name:     "npx via env",
			host:     stubs{installMethod: "npx", exe: "/any/path/stripe"},
			expected: NPX,
		},
		{
			name:     "env wins over every other signal",
			host:     stubs{installMethod: "npx", exe: "/opt/homebrew/Cellar/stripe/1.0/bin/stripe", present: []string{dpkgFile}},
			expected: NPX,
		},
		{
			name:     "homebrew cellar",
			host:     stubs{exe: "/opt/homebrew/Cellar/stripe/1.0/bin/stripe"},
			expected: Homebrew,
		},
		{
			name:     "homebrew apple silicon prefix, symlink unresolved",
			host:     stubs{exe: "/opt/homebrew/bin/stripe"},
			expected: Homebrew,
		},
		{
			// The Intel Mac case: os.Executable reports the launch path, which names
			// neither Cellar nor Homebrew, so only the resolved link identifies it.
			name: "homebrew intel mac, identified by resolving the symlink",
			host: stubs{
				exe:   "/usr/local/bin/stripe",
				links: map[string]string{"/usr/local/bin/stripe": "/usr/local/Cellar/stripe-cli/1.50.0/bin/stripe"},
			},
			expected: Homebrew,
		},
		{
			name:     "linuxbrew prefix",
			host:     stubs{exe: "/home/linuxbrew/.linuxbrew/bin/stripe"},
			expected: Homebrew,
		},
		{
			name:     "scoop apps directory",
			host:     stubs{exe: "C:/Users/foo/scoop/apps/stripe/current/stripe.exe"},
			expected: Scoop,
		},
		{
			name:     "scoop shim",
			host:     stubs{exe: "C:/Users/foo/scoop/shims/stripe.exe"},
			expected: Scoop,
		},
		{
			name:     "winget package directory",
			host:     stubs{exe: `C:\Users\foo\AppData\Local\Microsoft\WinGet\Packages\Stripe.StripeCLI_x\stripe.exe`},
			expected: Winget,
		},
		{
			name: "winget launcher, identified by resolving the link",
			host: stubs{
				exe: `C:\Users\foo\AppData\Local\Microsoft\WinGet\Links\stripe.exe`,
				links: map[string]string{
					`C:\Users\foo\AppData\Local\Microsoft\WinGet\Links\stripe.exe`: `C:\Users\foo\AppData\Local\Microsoft\WinGet\Packages\Stripe.StripeCLI_x\stripe.exe`,
				},
			},
			expected: Winget,
		},
		{
			name: "install script, from the method it stamped",
			host: stubs{
				exe:   "/home/user/.stripe/bin/stripe",
				files: map[string]string{marker("/home/user/.stripe/bin/stripe"): "script\n"},
			},
			expected: Script,
		},
		{
			// STRIPE_INSTALL_DIR lets the scripts install anywhere, so the stamped
			// method is the only thing that identifies a non-default location.
			name: "install script in a custom directory",
			host: stubs{
				exe:   "/opt/tools/bin/stripe",
				files: map[string]string{marker("/opt/tools/bin/stripe"): "script"},
			},
			expected: Script,
		},
		{
			name: "stamped method beats a path heuristic",
			host: stubs{
				exe:   "/opt/homebrew/bin/stripe",
				files: map[string]string{marker("/opt/homebrew/bin/stripe"): "script"},
			},
			expected: Script,
		},
		{
			name: "stamped method from an unrecognized distributor is still reported",
			host: stubs{
				exe:   "/opt/acme/bin/stripe",
				files: map[string]string{marker("/opt/acme/bin/stripe"): "acme_internal"},
			},
			expected: "acme_internal",
		},
		{
			name: "implausible stamped method is ignored",
			host: stubs{
				exe:     "/opt/acme/bin/stripe",
				files:   map[string]string{marker("/opt/acme/bin/stripe"): "Deploy Script <v2>"},
				present: []string{dpkgFile},
			},
			expected: APT,
		},
		{
			name: "oversized stamped method is ignored",
			host: stubs{
				exe:   "/opt/acme/bin/stripe",
				files: map[string]string{marker("/opt/acme/bin/stripe"): "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			},
			expected: Unknown,
		},
		{
			name:     "apt from the dpkg file list",
			host:     stubs{exe: "/usr/bin/stripe", present: []string{dpkgFile}},
			expected: APT,
		},
		{
			name:     "yum from the stripe repo",
			host:     stubs{exe: "/usr/bin/stripe", present: []string{yumFile}},
			expected: YUM,
		},
		{
			name:     "dpkg beats the weaker yum repo signal",
			host:     stubs{exe: "/usr/bin/stripe", present: []string{dpkgFile, yumFile}},
			expected: APT,
		},
		{
			name:     "docker from the official image",
			host:     stubs{exe: "/bin/stripe", present: []string{dockerEnv}},
			expected: Docker,
		},
		{
			// Our deb installs fine inside a container, and there apt is the answer.
			name:     "apt inside a container reports apt, not docker",
			host:     stubs{exe: "/usr/bin/stripe", present: []string{dpkgFile, dockerEnv}},
			expected: APT,
		},
		{
			name:     "unknown when nothing matches",
			host:     stubs{exe: "/usr/bin/stripe"},
			expected: Unknown,
		},
		{
			name:     "unknown when the executable cannot be found",
			host:     stubs{exeErr: true},
			expected: Unknown,
		},
		{
			name:     "system signals still apply when the executable cannot be found",
			host:     stubs{exeErr: true, present: []string{dpkgFile}},
			expected: APT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, Detect(tt.host.env()))
		})
	}
}

func TestDetectReadsOnlyItsOwnEnvVar(t *testing.T) {
	// The detected method is reported to telemetry, and this runs on every command
	// in an environment that routinely holds keys and session identifiers. Detection
	// reads one variable by name, so assert that nothing else in the environment can
	// reach the reported value, however many variables are set.
	//
	// Deliberately not shaped like a real credential: secret scanning reads test
	// files too, and a convincing literal blocks the push.
	const secret = "sensitive-value-that-must-not-be-reported"

	host := stubs{exe: "/usr/bin/stripe"}
	env := host.env()
	env.Getenv = func(name string) string {
		if name == "STRIPE_INSTALL_METHOD" {
			return ""
		}
		return secret
	}

	require.Equal(t, Unknown, Detect(env))
}

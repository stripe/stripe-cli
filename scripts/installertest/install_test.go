package installertest

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Run the actual installers with local downloads and a test signing identity.
// No network requests, PATH changes, or production signing secrets are needed.
func TestInstallRequiresApprovedArchive(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 3072)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	testPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	pinned, err := os.ReadFile("../../pkg/autoupdate/release-key.pub")
	require.NoError(t, err)
	block, _ := pem.Decode(pinned)
	require.NotNil(t, block)
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	pinnedKey := parsed.(*rsa.PublicKey)

	for _, interpreter := range []string{"sh", "pwsh", "powershell"} {
		t.Run(interpreter, func(t *testing.T) {
			if interpreter == "sh" && runtime.GOOS == "windows" {
				t.Skip("shell installer runs on macOS and Linux")
			}
			command, err := exec.LookPath(interpreter)
			if err != nil {
				if runtime.GOOS == "windows" && interpreter == "powershell" {
					t.Fatal("Windows PowerShell is required on Windows")
				}
				t.Skip(interpreter + " is not installed")
			}
			isShell := interpreter == "sh"
			scriptFile := "../install.ps1"
			if isShell {
				scriptFile = "../install.sh"
			}
			source, err := os.ReadFile(scriptFile)
			require.NoError(t, err)
			script := string(source)
			if isShell {
				require.Contains(t, script, strings.TrimSpace(string(pinned)), "installer must pin the reviewed release key")
				script = strings.Replace(script, strings.TrimSpace(string(pinned)), strings.TrimSpace(testPEM), 1)
				require.True(t, strings.HasSuffix(script, "\nmain\n"), "locate the installer entry point before injecting local downloads")
				script = strings.TrimSuffix(script, "main\n") + shellHarness + "\nmain\n"
			} else {
				modulus := base64.StdEncoding.EncodeToString(pinnedKey.N.Bytes())
				require.Contains(t, script, modulus, "installer must pin the reviewed release key")
				require.Contains(t, script, `$parameters.Exponent = [Convert]::FromBase64String("AQAB")`)
				script = strings.Replace(script, modulus, base64.StdEncoding.EncodeToString(key.N.Bytes()), 1)
				require.Contains(t, script, "# Main\n", "locate the installer entry point before injecting local downloads")
				script = strings.Replace(script, "# Main\n", powershellHarness+"\n# Main\n", 1)
			}

			for _, scenario := range []string{"approved", "altered checksums", "wrong signature", "missing signature", "wrong version", "duplicate checksum", "altered archive"} {
				t.Run(scenario, func(t *testing.T) {
					dir := t.TempDir()
					archiveName, binaryName := "stripe_1.55.7_windows_x86_64.zip", "stripe.exe"
					if isShell {
						archiveName, binaryName = "stripe_1.55.7_linux_x86_64.tar.gz", "stripe"
					}
					archive := testArchive(t, binaryName, isShell)
					digest := sha256.Sum256(archive)
					checksums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(digest[:]), archiveName)
					switch scenario {
					case "wrong version":
						checksums = strings.ReplaceAll(checksums, "1.55.7", "1.55.6")
					case "duplicate checksum":
						checksums += checksums
					}
					hash := sha256.Sum256([]byte(checksums))
					sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
					require.NoError(t, err)
					switch scenario {
					case "altered checksums":
						checksums = strings.ReplaceAll(checksums, hex.EncodeToString(digest[:]), strings.Repeat("0", 64))
					case "wrong signature":
						sig[0] ^= 1
					case "altered archive":
						archive = testArchiveContent(t, binaryName, isShell, "unapproved binary")
					}
					for name, content := range map[string][]byte{"archive": archive, "checksums": []byte(checksums), "signature": []byte(base64.StdEncoding.EncodeToString(sig) + "\n")} {
						require.NoError(t, os.WriteFile(filepath.Join(dir, name), content, 0600))
					}
					if scenario == "missing signature" {
						require.NoError(t, os.Remove(filepath.Join(dir, "signature")))
					}
					installer := filepath.Join(dir, filepath.Base(scriptFile))
					require.NoError(t, os.WriteFile(installer, []byte(script), 0600))
					installDir := filepath.Join(dir, "installed")
					require.NoError(t, os.Mkdir(installDir, 0700))
					installed := filepath.Join(installDir, binaryName)
					require.NoError(t, os.WriteFile(installed, []byte("existing binary"), 0700))
					args := []string{installer}
					if !isShell {
						args = []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", installer}
					}
					cmd := exec.Command(command, args...)
					cmd.Env = installerEnv(dir, installDir)
					output, runErr := cmd.CombinedOutput()
					got, err := os.ReadFile(installed)
					require.NoError(t, err)
					if scenario == "approved" {
						require.NoError(t, runErr, "%s", output)
						require.Equal(t, "approved binary", string(got))
					} else {
						require.Error(t, runErr, "%s", output)
						require.Equal(t, "existing binary", string(got), "%s", output)
					}
				})
			}
		})
	}
}

func installerEnv(dir, installDir string) []string {
	// Keep PATH and runtime configuration, isolating installer state and identity.
	replaced := regexp.MustCompile(`(?i)^(HOME|USERPROFILE|STRIPE_INSTALL_DIR|FIXTURES|GITHUB_TOKEN|VERSION)=`)
	env := []string{}
	for _, entry := range os.Environ() {
		if !replaced.MatchString(entry) {
			env = append(env, entry)
		}
	}
	return append(env, "HOME="+dir, "USERPROFILE="+dir, "STRIPE_INSTALL_DIR="+installDir, "FIXTURES="+dir)
}

func testArchive(t *testing.T, name string, shell bool) []byte {
	return testArchiveContent(t, name, shell, "approved binary")
}

func testArchiveContent(t *testing.T, name string, shell bool, content string) []byte {
	t.Helper()
	var data bytes.Buffer
	if shell {
		gz := gzip.NewWriter(&data)
		tw := tar.NewWriter(gz)
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(content))}))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
		require.NoError(t, tw.Close())
		require.NoError(t, gz.Close())
	} else {
		zw := zip.NewWriter(&data)
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
		require.NoError(t, zw.Close())
	}
	return data.Bytes()
}

const shellHarness = `
detect_platform() { OS=linux; OS_LABEL=linux; ARCH_LABEL=x86_64; CHECKSUMS_FILE=stripe-linux-checksums.txt; }
get_latest_version() { VERSION=1.55.7; LATEST_VERSION=1.55.7; }
http_download() {
  case "$1" in
    *.sig) cp "$FIXTURES/signature" "$2" ;;
    *checksums.txt) cp "$FIXTURES/checksums" "$2" ;;
    *) cp "$FIXTURES/archive" "$2" ;;
  esac
}
setup_path() { :; }
send_telemetry() { :; }
print_success() { :; }
`

const powershellHarness = `
function Detect-Platform { $script:ArchLabel = "x86_64" }
function Get-LatestVersion { $script:Version = "1.55.7"; $script:LatestVersion = "1.55.7" }
function Invoke-WebRequest($Uri, $OutFile, [switch]$UseBasicParsing) {
    $name = if ($Uri.EndsWith(".sig")) { "signature" } elseif ($Uri.EndsWith("checksums.txt")) { "checksums" } else { "archive" }
    Copy-Item -LiteralPath (Join-Path $env:FIXTURES $name) -Destination $OutFile
}
function Setup-Path {}
function Print-Success {}
`

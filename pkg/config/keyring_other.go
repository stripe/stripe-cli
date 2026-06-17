//go:build !darwin

package config

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	zkr "github.com/zalando/go-keyring"
)

type zalandoStore struct {
	service string
}

func (s *zalandoStore) Get(key string) ([]byte, error) {
	val, err := zkr.Get(s.service, key)
	if err == zkr.ErrNotFound {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (s *zalandoStore) Set(key string, data []byte, description string) error {
	return zkr.Set(s.service, key, string(data))
}

func (s *zalandoStore) Remove(key string) error {
	err := zkr.Delete(s.service, key)
	if err == zkr.ErrNotFound {
		return ErrKeyNotFound
	}
	return err
}


// wslWinCredStore bridges to Windows Credential Manager from WSL via
// powershell.exe. Scripts are passed via stdin and secrets via environment
// variables so that nothing sensitive appears in /proc/<pid>/cmdline or
// PowerShell script block logs.
type wslWinCredStore struct {
	service string
}

func (s *wslWinCredStore) targetName(key string) string {
	return fmt.Sprintf("%s:%s", s.service, key)
}

// runPS executes a PowerShell script via stdin with optional extra env vars.
// Neither the script text nor secrets appear in process arguments.
func runPS(script string, env []string) ([]byte, error) {
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-")
	cmd.Stdin = strings.NewReader(script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(defaultWSLEnv(), env...)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("powershell.exe: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	return bytes.TrimSpace(stdout.Bytes()), nil
}

// defaultWSLEnv returns the minimal environment needed for WSL interop.
func defaultWSLEnv() []string {
	keep := []string{"PATH", "WSLENV", "WSL_INTEROP", "HOME", "USER", "LOGNAME", "TERM"}
	env := make([]string, 0, len(keep))
	for _, k := range keep {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func (s *wslWinCredStore) Get(key string) ([]byte, error) {
	// Target is passed via env var so it doesn't appear in script block logs.
	script := `
$target = $env:_STRIPE_CRED_TARGET
[System.Environment]::SetEnvironmentVariable('_STRIPE_CRED_TARGET', $null, 'Process')

Add-Type -Namespace 'SC' -Name 'Cred' -MemberDefinition '
    [DllImport("advapi32.dll", SetLastError=true, CharSet=CharSet.Unicode)]
    public static extern bool CredRead(string target, int type, int flags, out IntPtr cred);
    [DllImport("advapi32.dll")]
    public static extern void CredFree(IntPtr cred);
    [StructLayout(LayoutKind.Sequential, CharSet=CharSet.Unicode)]
    public struct CREDENTIAL {
        public int Flags; public int Type;
        public string TargetName; public string Comment;
        public long LastWritten;
        public int CredentialBlobSize;
        public IntPtr CredentialBlob;
        public int Persist; public int AttributeCount;
        public IntPtr Attributes; public string TargetAlias;
        public string UserName;
    }
'
$ptr = [IntPtr]::Zero
$ok = [SC.Cred]::CredRead($target, 1, 0, [ref]$ptr)
if (-not $ok) { exit 1 }
$c = [System.Runtime.InteropServices.Marshal]::PtrToStructure($ptr, [Type][SC.Cred+CREDENTIAL])
$pass = [System.Runtime.InteropServices.Marshal]::PtrToStringUni($c.CredentialBlob, $c.CredentialBlobSize / 2)
[SC.Cred]::CredFree($ptr)
Write-Host -NoNewline $pass
`

	out, err := runPS(script, []string{"_STRIPE_CRED_TARGET=" + s.targetName(key)})
	if err != nil {
		return nil, ErrKeyNotFound
	}
	if len(out) == 0 {
		return nil, ErrKeyNotFound
	}
	return out, nil
}

func (s *wslWinCredStore) Set(key string, data []byte, description string) error {
	// Credentials passed via env vars — never in script text or process args.
	script := `
$target = $env:_STRIPE_CRED_TARGET
$user   = $env:_STRIPE_CRED_USER
$pass   = $env:_STRIPE_CRED_PASS

[System.Environment]::SetEnvironmentVariable('_STRIPE_CRED_TARGET', $null, 'Process')
[System.Environment]::SetEnvironmentVariable('_STRIPE_CRED_USER', $null, 'Process')
[System.Environment]::SetEnvironmentVariable('_STRIPE_CRED_PASS', $null, 'Process')

Add-Type -Namespace 'SC' -Name 'CredW' -MemberDefinition '
    [DllImport("advapi32.dll", SetLastError=true, CharSet=CharSet.Unicode)]
    public static extern bool CredWrite(ref CREDENTIAL cred, int flags);
    [StructLayout(LayoutKind.Sequential, CharSet=CharSet.Unicode)]
    public struct CREDENTIAL {
        public int Flags; public int Type;
        public string TargetName; public string Comment;
        public long LastWritten;
        public int CredentialBlobSize;
        public IntPtr CredentialBlob;
        public int Persist; public int AttributeCount;
        public IntPtr Attributes; public string TargetAlias;
        public string UserName;
    }
'

$passBytes = [System.Text.Encoding]::Unicode.GetBytes($pass)
$pass = $null

$blob = [System.Runtime.InteropServices.Marshal]::AllocHGlobal($passBytes.Length)
[System.Runtime.InteropServices.Marshal]::Copy($passBytes, 0, $blob, $passBytes.Length)

$cred = New-Object SC.CredW+CREDENTIAL
$cred.Type = 1  # CRED_TYPE_GENERIC
$cred.TargetName = $target
$cred.UserName = $user
$cred.CredentialBlobSize = $passBytes.Length
$cred.CredentialBlob = $blob
$cred.Persist = 2  # CRED_PERSIST_LOCAL_MACHINE

$ok = [SC.CredW]::CredWrite([ref]$cred, 0)
[System.Runtime.InteropServices.Marshal]::FreeHGlobal($blob)
[Array]::Clear($passBytes, 0, $passBytes.Length)

if (-not $ok) { exit 1 }
`

	_, err := runPS(script, []string{
		"_STRIPE_CRED_TARGET=" + s.targetName(key),
		"_STRIPE_CRED_USER=" + key,
		"_STRIPE_CRED_PASS=" + string(data),
	})
	return err
}

func (s *wslWinCredStore) Remove(key string) error {
	script := `
$target = $env:_STRIPE_CRED_TARGET
[System.Environment]::SetEnvironmentVariable('_STRIPE_CRED_TARGET', $null, 'Process')

Add-Type -Namespace 'SC' -Name 'CredD' -MemberDefinition '
    [DllImport("advapi32.dll", SetLastError=true, CharSet=CharSet.Unicode)]
    public static extern bool CredDelete(string target, int type, int flags);
'
$ok = [SC.CredD]::CredDelete($target, 1, 0)
if (-not $ok) { exit 1 }
`

	_, err := runPS(script, []string{"_STRIPE_CRED_TARGET=" + s.targetName(key)})
	if err != nil {
		return ErrKeyNotFound
	}
	return nil
}


func newSecureStore() SecureStore {
	if runtime.GOOS == "linux" && isWSL() {
		return &wslWinCredStore{service: KeyManagementService}
	}
	return &zalandoStore{service: KeyManagementService}
}

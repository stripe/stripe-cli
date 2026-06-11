//go:build !darwin

package config

import (
	"bytes"
	"fmt"
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

func (s *zalandoStore) Keys() ([]string, error) {
	return []string{}, nil
}

// wslWinCredStore bridges to Windows Credential Manager from WSL via
// cmdkey.exe (write/delete) and powershell.exe (read). This avoids
// maintaining custom encryption and uses the host OS's native credential
// store — the same one zalando/go-keyring uses on native Windows.
type wslWinCredStore struct {
	service string
}

func (s *wslWinCredStore) targetName(key string) string {
	return fmt.Sprintf("%s:%s", s.service, key)
}

func (s *wslWinCredStore) Get(key string) ([]byte, error) {
	target := s.targetName(key)

	// PowerShell script: try Get-StoredCredential first (CredentialManager
	// module), fall back to raw Win32 CredRead via P/Invoke.
	psScript := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
    $cred = Get-StoredCredential -Target '%s' -ErrorAction Stop
    if ($cred) {
        $n = $cred.GetNetworkCredential()
        Write-Host -NoNewline $n.Password
    } else {
        exit 1
    }
} catch {
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
    $ok = [SC.Cred]::CredRead('%s', 1, 0, [ref]$ptr)
    if (-not $ok) { exit 1 }
    $c = [System.Runtime.InteropServices.Marshal]::PtrToStructure($ptr, [Type][SC.Cred+CREDENTIAL])
    $pass = [System.Runtime.InteropServices.Marshal]::PtrToStringUni($c.CredentialBlob, $c.CredentialBlobSize / 2)
    [SC.Cred]::CredFree($ptr)
    Write-Host -NoNewline $pass
}
`, target, target)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psScript)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "not recognized") {
			return nil, fmt.Errorf("powershell.exe not available in WSL: %w", err)
		}
		return nil, ErrKeyNotFound
	}

	secret := stdout.String()
	if secret == "" {
		return nil, ErrKeyNotFound
	}

	return []byte(secret), nil
}

func (s *wslWinCredStore) Set(key string, data []byte, description string) error {
	target := s.targetName(key)

	cmd := exec.Command("cmdkey.exe",
		fmt.Sprintf("/generic:%s", target),
		fmt.Sprintf("/user:%s", key),
		fmt.Sprintf("/pass:%s", string(data)),
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmdkey.exe failed: %v (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	return nil
}

func (s *wslWinCredStore) Remove(key string) error {
	target := s.targetName(key)

	cmd := exec.Command("cmdkey.exe", fmt.Sprintf("/delete:%s", target))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return ErrKeyNotFound
	}

	return nil
}

func (s *wslWinCredStore) Keys() ([]string, error) {
	return []string{}, nil
}

func newSecureStore() SecureStore {
	if runtime.GOOS == "linux" && isWSL() {
		return &wslWinCredStore{service: KeyManagementService}
	}
	return &zalandoStore{service: KeyManagementService}
}

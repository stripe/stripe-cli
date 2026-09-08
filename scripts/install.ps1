#!/usr/bin/env pwsh
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Invoke-WebRequest's progress bar dominates download time on Windows PowerShell
# 5.1 (often by an order of magnitude). Suppress it.
$ProgressPreference = "SilentlyContinue"

$InstallDir = if ($env:STRIPE_INSTALL_DIR) { $env:STRIPE_INSTALL_DIR } else { Join-Path $env:USERPROFILE ".stripe\bin" }
$GitHubRepo = "stripe/stripe-cli"

function Get-ApiHeaders {
    $headers = @{ "User-Agent" = "stripe-installer" }
    # Unauthenticated api.github.com allows 60 requests/hour/IP, which the install
    # test workflow exhausts on its own. Mirrors install.sh's http_get().
    if ($env:GITHUB_TOKEN) {
        $headers["Authorization"] = "Bearer $env:GITHUB_TOKEN"
    }
    return $headers
}

function Detect-Platform {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
    switch ($arch) {
        "x64"   { $script:ArchLabel = "x86_64" }
        "x86"   { $script:ArchLabel = "i386" }
        "arm64" {
            # No windows/arm64 binary is published — .goreleaser/windows.yml builds
            # amd64 and 386 only. Windows on ARM runs x64 under emulation, so install
            # the x86_64 build instead of failing on a 404.
            $script:ArchLabel = "x86_64"
        }
        default { throw "Unsupported architecture: $arch. Supported: x64, x86, arm64." }
    }
    Write-Host "Detected: windows $script:ArchLabel"
}

function Get-LatestVersion {
    $releaseUrl = "https://api.github.com/repos/$GitHubRepo/releases/latest"
    try {
        $release = Invoke-RestMethod -Uri $releaseUrl -Headers (Get-ApiHeaders) -UseBasicParsing
        $script:Version = $release.tag_name -replace "^v", ""
    } catch {
        throw "Could not determine the latest version: $($_.Exception.Message)`nCheck your internet connection, or set GITHUB_TOKEN if you are being rate limited."
    }

    if (-not $script:Version) {
        throw "Could not parse a version from the GitHub release response."
    }
    Write-Host "Latest version: v$script:Version"
}

function Download-And-Verify {
    $archive = "stripe_${script:Version}_windows_${script:ArchLabel}.zip"
    $baseUrl = "https://github.com/$GitHubRepo/releases/download/v${script:Version}"

    $script:TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "stripe-install-$([System.Guid]::NewGuid().ToString('N'))"
    New-Item -ItemType Directory -Path $script:TmpDir | Out-Null

    $archivePath = Join-Path $script:TmpDir $archive
    $checksumsPath = Join-Path $script:TmpDir "checksums.txt"

    Write-Host "Downloading stripe v${script:Version}..."
    # No auth header here, deliberately: these URLs redirect to a signed object
    # store URL that rejects a second set of credentials. install.sh does the same.
    Invoke-WebRequest -Uri "$baseUrl/$archive" -OutFile $archivePath -UseBasicParsing
    Invoke-WebRequest -Uri "$baseUrl/stripe-windows-checksums.txt" -OutFile $checksumsPath -UseBasicParsing

    Write-Host "Verifying checksum..."
    $expected = $null
    foreach ($line in Get-Content $checksumsPath) {
        # Format is "<sha256>  <filename>". Compare the filename exactly rather than
        # regex-matching the whole line, whose dots would be metacharacters.
        $parts = $line.Trim() -split "\s+", 2
        if ($parts.Count -eq 2 -and $parts[1].Trim() -eq $archive) {
            $expected = $parts[0].ToLower()
            break
        }
    }
    if (-not $expected) {
        throw "Checksum entry not found for $archive"
    }

    $actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLower()

    if ($actual -ne $expected) {
        throw "Checksum verification failed.`n  Expected: $expected`n  Actual:   $actual`nThe downloaded file may be corrupted. Please try again."
    }
    Write-Host "Checksum verified."

    Expand-Archive -Path $archivePath -DestinationPath $script:TmpDir -Force
}

function Install-Binary {
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $src = Join-Path $script:TmpDir "stripe.exe"
    $dest = Join-Path $InstallDir "stripe.exe"
    $old = "$dest.old"

    # Left behind by an earlier upgrade whose binary was still running.
    if (Test-Path $old) {
        Remove-Item $old -Force -ErrorAction SilentlyContinue
    }

    # Windows locks a running image: stripe.exe cannot be overwritten while it is
    # executing, but it can be renamed. Move it aside so re-running the installer
    # over an in-use binary works.
    $movedAside = $false
    if (Test-Path $dest) {
        Move-Item -Path $dest -Destination $old -Force
        $movedAside = $true
    }

    try {
        Move-Item -Path $src -Destination $dest -Force
    } catch {
        # Restore the previous binary rather than leaving the user with no stripe.exe.
        if ($movedAside -and -not (Test-Path $dest)) {
            Move-Item -Path $old -Destination $dest -Force
        }
        throw
    }

    # Best effort: fails harmlessly if the old binary is still running, and the next
    # install cleans it up.
    if (Test-Path $old) {
        Remove-Item $old -Force -ErrorAction SilentlyContinue
    }

    # Record how the CLI was installed, so that an out-of-date binary can name the
    # command that upgrades it. STRIPE_INSTALL_DIR makes the location configurable,
    # so the CLI cannot infer this from its own path. Read by pkg/installmethod.
    Set-Content -Path (Join-Path $InstallDir ".stripe-install-method") -Value "script" -NoNewline

    # Warn about existing scoop/winget installs
    $scoopStripe = Join-Path $env:USERPROFILE "scoop\shims\stripe.exe"
    if (Test-Path $scoopStripe) {
        Write-Host ""
        Write-Host "Note: stripe is also installed via Scoop at $scoopStripe"
        Write-Host "You may want to run 'scoop uninstall stripe' to avoid confusion."
    }
}

function Send-EnvironmentChange {
    # Writing the registry directly does not notify running processes, so new
    # terminals would not see the PATH change until sign-out.
    # [Environment]::SetEnvironmentVariable does this for us; doing it by hand is the
    # cost of preserving the value's registry type below.
    try {
        if (-not ("Win32.NativeMethods" -as [type])) {
            Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition @'
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
'@
        }
        $result = [UIntPtr]::Zero
        # HWND_BROADCAST, WM_SETTINGCHANGE, SMTO_ABORTIFHUNG, 5s timeout
        [Win32.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1A, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
    } catch {
        # Purely a convenience; the success message already tells the user to restart
        # their terminal.
    }
}

function Setup-Path {
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment", $true)
    if (-not $key) {
        throw "Could not open HKCU\Environment to update PATH."
    }

    try {
        # DoNotExpandEnvironmentNames keeps entries like %USERPROFILE%\bin intact;
        # reading them expanded and writing them back would bake in the expansion.
        $current = [string]$key.GetValue("Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)

        $entries = @($current -split ";" | Where-Object { $_ -ne "" })
        if ($entries -contains $InstallDir) {
            return
        }

        # [Environment]::SetEnvironmentVariable(..., "User") always writes REG_SZ,
        # which silently breaks %VAR% expansion for every other PATH entry (and
        # truncates past ~2047 chars). Preserve whatever type is already there.
        $kind = [Microsoft.Win32.RegistryValueKind]::ExpandString
        try { $kind = $key.GetValueKind("Path") } catch { }

        $newPath = if ([string]::IsNullOrEmpty($current)) { $InstallDir } else { "$InstallDir;$current" }
        $key.SetValue("Path", $newPath, $kind)
    } finally {
        $key.Dispose()
    }

    Send-EnvironmentChange

    $env:Path = "$InstallDir;$env:Path"
    Write-Host "Added $InstallDir to user PATH."
    $script:PathUpdated = $true
}

function Print-Success {
    Write-Host ""
    Write-Host "stripe v${script:Version} installed to $InstallDir\stripe.exe"
    Write-Host ""
    if ($script:PathUpdated) {
        Write-Host "Restart your terminal for PATH changes to take effect, then:"
    }
    Write-Host "  stripe login    - authenticate with your Stripe account"
    Write-Host "  stripe --help   - see available commands"
}

function Cleanup {
    if ($script:TmpDir -and (Test-Path $script:TmpDir)) {
        Remove-Item -Recurse -Force $script:TmpDir
    }
}

# Main
$script:Version = ""
$script:ArchLabel = ""
$script:TmpDir = ""
$script:PathUpdated = $false

# Failures throw rather than calling exit: a terminating error still yields a
# non-zero exit code under `pwsh -File`, but does not tear down the caller's
# session the way `exit` does when this script is run via `irm ... | iex`.
try {
    Detect-Platform
    Get-LatestVersion
    Download-And-Verify
    Install-Binary
    Setup-Path
    Print-Success
} finally {
    Cleanup
}

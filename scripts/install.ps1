#!/usr/bin/env pwsh
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$InstallDir = if ($env:STRIPE_INSTALL_DIR) { $env:STRIPE_INSTALL_DIR } else { Join-Path $env:USERPROFILE ".stripe\bin" }
$GitHubRepo = "stripe/stripe-cli"

function Detect-Platform {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
    switch ($arch) {
        "x64"   { $script:ArchLabel = "x86_64" }
        "x86"   { $script:ArchLabel = "i386" }
        "arm64" { $script:ArchLabel = "arm64" }
        default {
            Write-Error "Unsupported architecture: $arch. Supported: x64, x86, arm64."
            exit 1
        }
    }
    Write-Host "Detected: windows $script:ArchLabel"
}

function Get-LatestVersion {
    $releaseUrl = "https://api.github.com/repos/$GitHubRepo/releases/latest"
    try {
        $release = Invoke-RestMethod -Uri $releaseUrl -Headers @{ "User-Agent" = "stripe-installer" }
        $script:Version = $release.tag_name -replace "^v", ""
    } catch {
        Write-Error "Could not determine latest version. Check your internet connection or try again later."
        exit 1
    }

    if (-not $script:Version) {
        Write-Error "Could not parse version from GitHub release."
        exit 1
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
    Invoke-WebRequest -Uri "$baseUrl/$archive" -OutFile $archivePath -UseBasicParsing
    Invoke-WebRequest -Uri "$baseUrl/stripe-windows-checksums.txt" -OutFile $checksumsPath -UseBasicParsing

    Write-Host "Verifying checksum..."
    $checksumContent = Get-Content $checksumsPath
    $expectedLine = $checksumContent | Where-Object { $_ -match $archive }
    if (-not $expectedLine) {
        Write-Error "Checksum entry not found for $archive"
        exit 1
    }
    $expected = ($expectedLine -split "\s+")[0]

    $actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLower()

    if ($actual -ne $expected) {
        Write-Error "Checksum verification failed.`n  Expected: $expected`n  Actual:   $actual`nThe downloaded file may be corrupted. Please try again."
        exit 1
    }
    Write-Host "Checksum verified."

    Expand-Archive -Path $archivePath -DestinationPath $script:TmpDir -Force
}

function Install-Binary {
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir | Out-Null
    }

    $src = Join-Path $script:TmpDir "stripe.exe"
    $dest = Join-Path $InstallDir "stripe.exe"
    Move-Item -Path $src -Destination $dest -Force

    # Warn about existing scoop/winget installs
    $scoopStripe = Join-Path $env:USERPROFILE "scoop\shims\stripe.exe"
    if (Test-Path $scoopStripe) {
        Write-Host ""
        Write-Host "Note: stripe is also installed via Scoop at $scoopStripe"
        Write-Host "You may want to run 'scoop uninstall stripe' to avoid confusion."
    }
}

function Setup-Path {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -split ";" | Where-Object { $_ -eq $InstallDir }) {
        return
    }

    [Environment]::SetEnvironmentVariable("Path", "$InstallDir;$userPath", "User")
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

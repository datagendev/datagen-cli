#Requires -Version 5.1
<#
.SYNOPSIS
    Install the DataGen CLI on Windows.
.DESCRIPTION
    Downloads the latest datagen-windows-amd64.exe from GitHub Releases
    and installs it to the user's local app data directory.
.PARAMETER Version
    Specific version to install (e.g. "v0.3.1"). Defaults to "latest".
.PARAMETER InstallDir
    Custom installation directory. Defaults to "$env:LOCALAPPDATA\datagen".
.EXAMPLE
    irm https://cli.datagen.dev/install.ps1 | iex
.EXAMPLE
    & .\install.ps1 -Version v0.3.1
#>
param(
    [string]$Version = $env:DATAGEN_VERSION,
    [string]$InstallDir = $env:DATAGEN_INSTALL_DIR
)

$ErrorActionPreference = "Stop"

$Repo = "datagendev/datagen-cli"
$Binary = "datagen"
$Asset = "datagen-windows-amd64.exe"

if (-not $Version) { $Version = "latest" }
if (-not $InstallDir) { $InstallDir = Join-Path $env:LOCALAPPDATA "datagen" }

function Write-Status($msg) {
    Write-Host $msg
}

function Get-DownloadUrl {
    if ($Version -eq "latest") {
        return "https://github.com/$Repo/releases/latest/download/$Asset"
    }
    return "https://github.com/$Repo/releases/download/$Version/$Asset"
}

function Add-ToUserPath($dir) {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -split ";" | Where-Object { $_ -eq $dir }) {
        return $false
    }
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$dir", "User")
    return $true
}

# Create install directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$url = Get-DownloadUrl
$dest = Join-Path $InstallDir "$Binary.exe"

Write-Status "Downloading $Asset ($Version)..."
try {
    Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing
} catch {
    Write-Error "Failed to download from $url`n$($_.Exception.Message)"
    exit 1
}

Write-Status "Installed: $dest"

# Add to user PATH if not already present
$added = Add-ToUserPath $InstallDir
if ($added) {
    Write-Status ""
    Write-Status "Added $InstallDir to your user PATH."
    Write-Status "Restart your terminal for PATH changes to take effect."
} else {
    Write-Status "$InstallDir is already in your PATH."
}

# Also add to current session PATH so it works immediately
if (-not ($env:Path -split ";" | Where-Object { $_ -eq $InstallDir })) {
    $env:Path = "$env:Path;$InstallDir"
}

Write-Status ""
Write-Status "Verify:"
Write-Status "  datagen --help"

# Install aiskillgrid from GitHub Releases onto PATH (Windows PowerShell).
param(
  [string]$Repo = $(if ($env:AISKILLGRID_REPO) { $env:AISKILLGRID_REPO } else { "aiskillgrid/aiskillgrid" }),
  [string]$Version = $(if ($env:AISKILLGRID_VERSION) { $env:AISKILLGRID_VERSION } else { "latest" }),
  [string]$InstallDir = $(if ($env:AISKILLGRID_INSTALL_DIR) { $env:AISKILLGRID_INSTALL_DIR } else { Join-Path $env:USERPROFILE ".local\bin" })
)

$ErrorActionPreference = "Stop"
$Name = "aiskillgrid"
$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { throw "unsupported arch" }

if ($Version -eq "latest") {
  $rel = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
  $tag = $rel.tag_name
  $Version = $tag.TrimStart("v")
}

$asset = "$Name-$Version-windows-$Arch.exe"
$url = "https://github.com/$Repo/releases/download/v$Version/$asset"
$tmp = Join-Path $env:TEMP "$Name.exe"

Write-Host "Downloading $url"
try {
  Invoke-WebRequest -Uri $url -OutFile $tmp
} catch {
  $url = "https://github.com/$Repo/releases/download/$Version/$asset"
  Write-Host "Retrying $url"
  Invoke-WebRequest -Uri $url -OutFile $tmp
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$dest = Join-Path $InstallDir "$Name.exe"
Move-Item -Force -Path $tmp -Destination $dest
Write-Host "Installed $dest"
if (-not ($env:PATH -split ";" | Where-Object { $_ -eq $InstallDir })) {
  Write-Host "Add to PATH: $InstallDir"
}
Write-Host "Next: aiskillgrid sync; aiskillgrid install"

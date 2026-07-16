# Fix Flutter Android builds when your Windows user folder has spaces
# (e.g. C:\Users\ThinkPad X1 Carbon\...)
#
# Run once from an elevated or normal PowerShell:
#   .\scripts\fix-android-path.ps1
#
# Then build from C:\Rozy:
#   cd C:\Rozy
#   make driver-prod

$ErrorActionPreference = "Stop"

$realRoot = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
if (-not (Test-Path (Join-Path $realRoot "mobile\pubspec.yaml"))) {
    $realRoot = Split-Path $PSScriptRoot -Parent
}
if (-not (Test-Path (Join-Path $realRoot "mobile\pubspec.yaml"))) {
    throw "Could not find Rozy repo root."
}

$links = @(
    @{ Link = "C:\Rozy"; Target = $realRoot }
)

$flutterInPath = Get-Command flutter -ErrorAction SilentlyContinue
if ($flutterInPath) {
    $flutterHome = Split-Path (Split-Path $flutterInPath.Source -Parent) -Parent
    if ($flutterHome -match " ") {
        $links += @{ Link = "C:\flutter"; Target = $flutterHome }
    }
}

foreach ($item in $links) {
    if (Test-Path $item.Link) {
        Write-Host "Already exists: $($item.Link)" -ForegroundColor Yellow
        continue
    }
    cmd /c mklink /J "$($item.Link)" "$($item.Target)" | Out-Null
    Write-Host "Created junction: $($item.Link) -> $($item.Target)" -ForegroundColor Green
}

if ($env:USERPROFILE -match " ") {
    if (-not $env:PUB_CACHE -or $env:PUB_CACHE -match " ") {
        $pubCache = "C:\pub-cache"
        New-Item -ItemType Directory -Force -Path $pubCache | Out-Null
        Write-Host ""
        Write-Host "Add this to your user environment variables (System Properties -> Environment):" -ForegroundColor Cyan
        Write-Host "  PUB_CACHE = $pubCache"
        Write-Host ""
        $env:PUB_CACHE = $pubCache
    }
}

Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Close and reopen terminal (after setting PUB_CACHE if needed)"
Write-Host "  2. cd C:\Rozy\mobile"
Write-Host "  3. flutter clean && flutter pub get"
Write-Host "  4. cd C:\Rozy && make driver-prod"
Write-Host ""
Write-Host "Quick test on PC without Android build:" -ForegroundColor Cyan
Write-Host "  make driver-prod -- -d windows"
Write-Host "  make passenger-prod -- -d chrome"
Write-Host ""

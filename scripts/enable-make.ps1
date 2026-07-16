# One-time setup: type "make run" from anywhere inside the Rozy repo (PowerShell).
# Run:  .\scripts\enable-make.ps1

$functionDef = @'
function global:make {
    param(
        [Parameter(Position = 0, ValueFromRemainingArguments = $true)]
        [string[]]$Args
    )
    $dir = (Get-Location).Path
    while ($dir) {
        $script = Join-Path $dir "make.ps1"
        if (Test-Path $script) {
            & $script @Args
            return
        }
        $parent = Split-Path $dir -Parent
        if (-not $parent -or $parent -eq $dir) { break }
        $dir = $parent
    }
    Write-Error "make.ps1 not found. cd into the Rozy repo first."
}
'@

$marker = "# Rozy make helper"
$block = @"

$marker
$functionDef
"@

function Install-ToProfile([string]$profilePath) {
    if (-not $profilePath) { return }
    $dir = Split-Path $profilePath -Parent
    if (-not (Test-Path $dir)) {
        New-Item -Path $dir -ItemType Directory -Force | Out-Null
    }
    if (Test-Path $profilePath) {
        $content = Get-Content $profilePath -Raw -ErrorAction SilentlyContinue
        if ($content -and $content -match [regex]::Escape($marker)) {
            Write-Host "Already enabled: $profilePath" -ForegroundColor Yellow
            return
        }
        Add-Content -Path $profilePath -Value $block
    } else {
        Set-Content -Path $profilePath -Value $block.TrimStart()
    }
    Write-Host "Enabled make in: $profilePath" -ForegroundColor Green
}

$profiles = @(
    $PROFILE.CurrentUserCurrentHost,
    $PROFILE.CurrentUserAllHosts
) | Select-Object -Unique

foreach ($p in $profiles) {
    Install-ToProfile $p
}

Write-Host ""
Write-Host "Restart PowerShell (or open a new terminal), then:" -ForegroundColor Cyan
Write-Host "  cd `"C:\Users\ThinkPad X1 Carbon\Rozy\backend`""
Write-Host "  make run"
Write-Host "  make migrate"
Write-Host "  make help"
Write-Host ""

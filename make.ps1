# Rozy dev commands for Windows (PowerShell)
# Usage: make run | make migrate | make admin | make help
# From repo root or backend/

param(
    [Parameter(Position = 0)]
    [string]$Target = "help",
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$RemainingArgs
)

$ErrorActionPreference = "Stop"

function Get-GoExe {
    $cmd = Get-Command go -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $default = "C:\Program Files\Go\bin\go.exe"
    if (Test-Path $default) { return $default }
    throw "Go not found. Install from https://go.dev/dl/ and restart your terminal."
}

function Get-NpmExe {
    $cmd = Get-Command npm -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $default = "C:\Program Files\nodejs\npm.cmd"
    if (Test-Path $default) { return $default }
    throw "npm not found. Install Node.js from https://nodejs.org/ and restart your terminal."
}

function Ensure-NodeOnPath {
    $nodeDir = Split-Path (Get-NpmExe) -Parent
    if ($env:PATH -notlike "*$nodeDir*") {
        $env:PATH = "$nodeDir;$env:PATH"
    }
}

function Get-FlutterExe {
    $cmd = Get-Command flutter -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    throw "flutter not found. Install Flutter and add it to PATH."
}

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Root = $ScriptDir
$Backend = Join-Path $Root "backend"
$Admin = Join-Path $Root "admin"
$Mobile = Join-Path $Root "mobile"

function Show-Help {
    Write-Host ""
    Write-Host "Rozy dev commands (run from repo root or backend/):" -ForegroundColor Cyan
    Write-Host "  make run            - start Go API server (:8080)"
    Write-Host "  make migrate        - apply database migrations"
    Write-Host "  make migrate-down   - roll back one migration"
    Write-Host "  make test           - run Go unit tests"
    Write-Host "  make test-api       - integration test APIs (API must be running)"
    Write-Host "  make build          - build API binary"
    Write-Host "  make admin          - start React admin dashboard (:5173)"
    Write-Host "  make redis          - start local Redis via Docker"
    Write-Host "  make passenger      - run Flutter passenger app (local API)"
    Write-Host "  make driver         - run Flutter driver app (local API)"
    Write-Host "  make passenger-prod - run passenger app (Render API)"
    Write-Host "  make driver-prod    - run driver app (Render API)"
    Write-Host "  make dev            - show full-stack startup tips"
    Write-Host "  make help           - show this help"
    Write-Host ""
    Write-Host "PowerShell: use .\make.cmd run  (or run scripts\enable-make.ps1 once for bare make run)" -ForegroundColor DarkGray
    Write-Host "CMD:        use make run" -ForegroundColor DarkGray
    Write-Host "OTP codes print in the API terminal (SMS_PROVIDER=console)." -ForegroundColor DarkGray
    Write-Host ""
}

switch ($Target.ToLower()) {
    "help" { Show-Help }

    "run" {
        $go = Get-GoExe
        Push-Location $Backend
        try { & $go run ./cmd/api }
        finally { Pop-Location }
    }

    "migrate" {
        $go = Get-GoExe
        Push-Location $Backend
        try { & $go run ./cmd/migrate }
        finally { Pop-Location }
    }

    "migrate-down" {
        $go = Get-GoExe
        Push-Location $Backend
        try { & $go run ./cmd/migrate --down }
        finally { Pop-Location }
    }

    "test" {
        $go = Get-GoExe
        Push-Location $Backend
        try { & $go test ./... }
        finally { Pop-Location }
    }

    "test-api" {
        $go = Get-GoExe
        Push-Location $Backend
        try { & $go run ./cmd/testapi }
        finally { Pop-Location }
    }

    "build" {
        $go = Get-GoExe
        $out = Join-Path $Backend "bin\rozy-api.exe"
        New-Item -ItemType Directory -Force -Path (Split-Path $out) | Out-Null
        Push-Location $Backend
        try { & $go build -o $out ./cmd/api }
        finally { Pop-Location }
        Write-Host "Built $out" -ForegroundColor Green
    }

    "admin" {
        Ensure-NodeOnPath
        $npm = Get-NpmExe
        Push-Location $Admin
        try {
            if (-not (Test-Path "node_modules")) {
                Write-Host "Installing admin dependencies..." -ForegroundColor Yellow
                & $npm install
            }
            & $npm run dev
        }
        finally { Pop-Location }
    }

    "redis" {
        $docker = Get-Command docker -ErrorAction SilentlyContinue
        if (-not $docker) {
            throw "Docker not found. Install Docker Desktop or skip Redis (API uses Postgres fallback)."
        }
        docker run -d --name rozy-redis -p 6379:6379 redis:7-alpine 2>$null
        if ($LASTEXITCODE -ne 0) {
            docker start rozy-redis
        }
        Write-Host "Redis running on localhost:6379" -ForegroundColor Green
    }

    "passenger" {
        $flutter = Get-FlutterExe
        Push-Location $Mobile
        try {
            & $flutter pub get
            if ($RemainingArgs) {
                & $flutter run -t lib/main_passenger.dart @RemainingArgs
            } else {
                & $flutter run -t lib/main_passenger.dart
            }
        }
        finally { Pop-Location }
    }

    "driver" {
        $flutter = Get-FlutterExe
        Push-Location $Mobile
        try {
            & $flutter pub get
            if ($RemainingArgs) {
                & $flutter run -t lib/main_driver.dart @RemainingArgs
            } else {
                & $flutter run -t lib/main_driver.dart
            }
        }
        finally { Pop-Location }
    }

    "passenger-prod" {
        $flutter = Get-FlutterExe
        Push-Location $Mobile
        try {
            & $flutter pub get
            $defines = @(
                "--dart-define=API_BASE_URL=https://rozy.onrender.com/v1"
            )
            if ($RemainingArgs) {
                & $flutter run -t lib/main_passenger.dart @defines @RemainingArgs
            } else {
                & $flutter run -t lib/main_passenger.dart @defines
            }
        }
        finally { Pop-Location }
    }

    "driver-prod" {
        $flutter = Get-FlutterExe
        Push-Location $Mobile
        try {
            & $flutter pub get
            $defines = @(
                "--dart-define=API_BASE_URL=https://rozy.onrender.com/v1"
            )
            if ($RemainingArgs) {
                & $flutter run -t lib/main_driver.dart @defines @RemainingArgs
            } else {
                & $flutter run -t lib/main_driver.dart @defines
            }
        }
        finally { Pop-Location }
    }

    "dev" {
        Write-Host ""
        Write-Host "Full stack startup:" -ForegroundColor Cyan
        Write-Host "  1. make redis      (optional, faster dispatch)"
        Write-Host "  2. make migrate    (first time / after schema changes)"
        Write-Host "  3. make run        (API on http://localhost:8080, tester at /dev)"
        Write-Host "  4. make admin      (dashboard on http://localhost:5173)"
        Write-Host "  5. make passenger  or  make driver"
        Write-Host ""
    }

    default {
        Write-Host "Unknown target: $Target" -ForegroundColor Red
        Show-Help
        exit 1
    }
}

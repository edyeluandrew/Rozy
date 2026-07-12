# Rozy dev shortcuts (Windows PowerShell)
# Usage: .\scripts\dev.ps1 run

param(
    [Parameter(Position = 0)]
    [ValidateSet('run', 'migrate', 'migrate-down', 'test', 'test-api', 'build', 'admin', 'redis', 'help')]
    [string]$Command = 'help'
)

$Backend = Join-Path $PSScriptRoot '..\backend'
$Admin = Join-Path $PSScriptRoot '..\admin'

switch ($Command) {
    'run' { Set-Location $Backend; go run ./cmd/api }
    'migrate' { Set-Location $Backend; go run ./cmd/migrate }
    'migrate-down' { Set-Location $Backend; go run ./cmd/migrate --down }
    'test' { Set-Location $Backend; go test ./... }
    'test-api' { Set-Location $Backend; go run ./cmd/testapi }
    'build' { Set-Location $Backend; go build -o bin/rozy-api.exe ./cmd/api }
    'admin' { Set-Location $Admin; npm run dev }
    'redis' { docker run -d --name rozy-redis -p 6379:6379 redis:7-alpine; if ($LASTEXITCODE -ne 0) { docker start rozy-redis } }
    default {
        Write-Host 'Commands: run, migrate, migrate-down, test, test-api, build, admin, redis'
    }
}

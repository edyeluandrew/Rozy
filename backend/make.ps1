# Wrapper so backend/ can call repo-root make.ps1
& (Join-Path (Split-Path $PSScriptRoot -Parent) "make.ps1") @args

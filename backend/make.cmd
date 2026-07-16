@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0..\make.ps1" %*
exit /b %ERRORLEVEL%

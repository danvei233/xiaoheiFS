@echo off
setlocal enabledelayedexpansion

REM This script lives under script/. Jump to repo root first.
cd /d "%~dp0\.."

set "OUT_DIR=build\windows"
set "OUT_STATIC=%OUT_DIR%\static"

echo [1/3] Build frontend...
pushd "frontend" >nul
call npm ci || exit /b 1
call npm run build || exit /b 1
popd >nul

echo [2/3] Copy dist to %OUT_STATIC%...
if exist "%OUT_DIR%" rmdir /s /q "%OUT_DIR%"
mkdir "%OUT_STATIC%" || exit /b 1
xcopy "frontend\dist\*" "%OUT_STATIC%\\" /E /I /Y >nul

echo [3/3] Build backend (windows)...
pushd "backend" >nul
go build -o "..\%OUT_DIR%\server.exe" .\cmd\server || exit /b 1
popd >nul

echo Done.
endlocal
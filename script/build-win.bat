@echo off
setlocal enabledelayedexpansion

REM This script lives under script/. Jump to repo root first.
cd /d "%~dp0\.."

set "OUT_DIR=build\windows"
set "OUT_STATIC=%OUT_DIR%\static"
set "OUT_STATIC_ADMIN=%OUT_DIR%\static-admin"

echo [1/4] Build frontend (public site)...
pushd "frontend" >nul
call npm ci || exit /b 1
call npm run build || exit /b 1
popd >nul

echo [2/4] Reset %OUT_DIR% and copy public static...
if exist "%OUT_DIR%" rmdir /s /q "%OUT_DIR%"
mkdir "%OUT_STATIC%" || exit /b 1
mkdir "%OUT_STATIC_ADMIN%" || exit /b 1
xcopy "frontend\dist\*" "%OUT_STATIC%\\" /E /I /Y >nul

echo [3/4] Build adminweb (admin SPA)...
pushd "adminweb" >nul
call corepack enable >nul 2>&1
call pnpm install --frozen-lockfile || exit /b 1
set "VITE_BUILD_OUT_DIR=%CD%\..\%OUT_STATIC_ADMIN%"
call pnpm build || exit /b 1
set "VITE_BUILD_OUT_DIR="
popd >nul

echo [4/4] Build backend (windows)...
pushd "backend" >nul
go env -w CGO_ENABLED=0
go build -o "..\%OUT_DIR%\server.exe" .\cmd\server || exit /b 1
popd >nul

echo Done.
endlocal

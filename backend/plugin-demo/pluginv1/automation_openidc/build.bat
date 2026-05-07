@echo off
setlocal enabledelayedexpansion

REM 编译 openidc_default 插件（Windows amd64）
REM 在 backend/ 目录下执行：plugin-demo\pluginv1\automation_openidc\build.bat

cd /d "%~dp0\..\..\..\"
echo [当前目录] %CD%

set "PLUGIN_SRC=./plugin-demo/pluginv1/automation_openidc"
set "OUT_DIR=plugins\automation\openidc_default\bin\windows_amd64"

echo [1/1] 编译 openidc_default (windows/amd64)...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags="-s -w" -o "%OUT_DIR%\plugin.exe" %PLUGIN_SRC% || exit /b 1

echo 编译完成：%OUT_DIR%\plugin.exe
endlocal

@echo off
setlocal

REM Build demo plugins for current Windows amd64 into plugins/**/bin/windows_amd64/plugin.exe
cd /d %~dp0
cd /d ..

set ORIG_GOOS=%GOOS%
set ORIG_GOARCH=%GOARCH%
set ORIG_CGO=%CGO_ENABLED%

set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

call :build_one "plugins\\payment\\ezpay\\bin\\windows_amd64" "./plugin-demo/pluginv1/payment_ezpay"
call :build_one "plugins\\payment\\wechatpay_v3\\bin\\windows_amd64" "./plugin-demo/pluginv1/payment_wechatpay_v3"
call :build_one "plugins\\payment\\alipay_open\\bin\\windows_amd64" "./plugin-demo/pluginv1/payment_alipay_open"
call :build_one "plugins\\sms\\alisms\\bin\\windows_amd64" "./plugin-demo/pluginv1/sms_alisms_mock"
call :build_one "plugins\\sms\\tencent_sms\\bin\\windows_amd64" "./plugin-demo/pluginv1/sms_tencent_mock"
call :build_one "plugins\\sms\\duanxinbao\\bin\\windows_amd64" "./plugin-demo/pluginv1/sms_duanxinbao"
call :build_one "plugins\\kyc\\aliyun_kyc\\bin\\windows_amd64" "./plugin-demo/pluginv1/kyc_aliyun_mock"
call :build_one "plugins\\kyc\\tencent_kyc\\bin\\windows_amd64" "./plugin-demo/pluginv1/kyc_tencent_mock"
call :build_one "plugins\\kyc\\mangzhu_realname\\bin\\windows_amd64" "./plugin-demo/pluginv1/kyc_mangzhu_realname"
call :build_one "plugins\\automation\\lightboat\\bin\\windows_amd64" "./plugin-demo/pluginv1/automation_lightboat"

set GOOS=%ORIG_GOOS%
set GOARCH=%ORIG_GOARCH%
set CGO_ENABLED=%ORIG_CGO%

echo OK: built demo plugins into ./plugins/**/bin/windows_amd64/plugin.exe
exit /b 0

:build_one
set OUTDIR=%~1
set PKG=%~2
if not exist %OUTDIR% mkdir %OUTDIR%
go build -o %OUTDIR%\\plugin.exe %PKG%
if errorlevel 1 (
  echo Build failed: %PKG%
  exit /b 1
)
exit /b 0

@echo off
REM Wrapper for Windows build artifacts: outputs to .\build\windows\
call "%~dp0build\\windows\\build-win.bat" %*

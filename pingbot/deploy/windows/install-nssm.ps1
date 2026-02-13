param(
  [string]$NssmPath = "C:\nssm\nssm.exe",
  [string]$PingbotExe = "C:\pingbot\pingbot.exe",
  [string]$ConfigPath = "C:\ProgramData\pingbot\config.yaml"
)

if (!(Test-Path $NssmPath)) { throw "nssm not found: $NssmPath" }
if (!(Test-Path $PingbotExe)) { throw "pingbot.exe not found: $PingbotExe" }

New-Item -ItemType Directory -Force -Path (Split-Path $ConfigPath -Parent) | Out-Null

& $NssmPath install pingbot $PingbotExe "-config `"$ConfigPath`""
& $NssmPath set pingbot Start SERVICE_AUTO_START
& $NssmPath start pingbot
Write-Host "pingbot installed and started."

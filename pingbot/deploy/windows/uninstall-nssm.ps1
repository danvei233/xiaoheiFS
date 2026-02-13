param(
  [string]$NssmPath = "C:\nssm\nssm.exe"
)

if (!(Test-Path $NssmPath)) { throw "nssm not found: $NssmPath" }

& $NssmPath stop pingbot | Out-Null
& $NssmPath remove pingbot confirm | Out-Null
Write-Host "pingbot service removed."

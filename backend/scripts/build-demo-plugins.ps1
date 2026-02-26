$ErrorActionPreference = "Stop"

Push-Location (Split-Path -Parent $MyInvocation.MyCommand.Path)
Pop-Location | Out-Null

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $root

New-Item -ItemType Directory -Force "plugins/payment/ezpay" | Out-Null
New-Item -ItemType Directory -Force "plugins/payment/wechatpay_v3" | Out-Null
New-Item -ItemType Directory -Force "plugins/payment/alipay_open" | Out-Null
New-Item -ItemType Directory -Force "plugins/sms/alisms" | Out-Null
New-Item -ItemType Directory -Force "plugins/sms/tencent_sms" | Out-Null
New-Item -ItemType Directory -Force "plugins/sms/duanxinbao" | Out-Null
New-Item -ItemType Directory -Force "plugins/kyc/aliyun_kyc" | Out-Null
New-Item -ItemType Directory -Force "plugins/kyc/tencent_kyc" | Out-Null
New-Item -ItemType Directory -Force "plugins/kyc/mangzhu_realname" | Out-Null
New-Item -ItemType Directory -Force "plugins/automation/lightboat" | Out-Null
New-Item -ItemType Directory -Force "plugins/automation/xiaohei_proxy" | Out-Null
New-Item -ItemType Directory -Force "plugins/automation/mofang_openapi" | Out-Null
New-Item -ItemType Directory -Force "plugins/automation/openidc_default" | Out-Null

$targets = @(
  @{ goos = "windows"; goarch = "amd64"; ext = ".exe" },
  @{ goos = "linux"; goarch = "amd64"; ext = "" },
  @{ goos = "darwin"; goarch = "amd64"; ext = "" },
  @{ goos = "darwin"; goarch = "arm64"; ext = "" }
)

$plugins = @(
  @{ id = "payment/ezpay"; pkg = "./plugin-demo/pluginv1/payment_ezpay" },
  @{ id = "payment/wechatpay_v3"; pkg = "./plugin-demo/pluginv1/payment_wechatpay_v3" },
  @{ id = "payment/alipay_open"; pkg = "./plugin-demo/pluginv1/payment_alipay_open" },
  @{ id = "sms/alisms"; pkg = "./plugin-demo/pluginv1/sms_alisms_mock" },
  @{ id = "sms/tencent_sms"; pkg = "./plugin-demo/pluginv1/sms_tencent_mock" },
  @{ id = "sms/duanxinbao"; pkg = "./plugin-demo/pluginv1/sms_duanxinbao" },
  @{ id = "kyc/aliyun_kyc"; pkg = "./plugin-demo/pluginv1/kyc_aliyun_mock" },
  @{ id = "kyc/tencent_kyc"; pkg = "./plugin-demo/pluginv1/kyc_tencent_mock" },
  @{ id = "kyc/mangzhu_realname"; pkg = "./plugin-demo/pluginv1/kyc_mangzhu_realname" },
  @{ id = "automation/lightboat"; pkg = "./plugin-demo/pluginv1/automation_lightboat" },
  @{ id = "automation/xiaohei_proxy"; pkg = "./plugin-demo/pluginv1/automation_xiaohei_proxy" },
  @{ id = "automation/mofang_openapi"; pkg = "./plugin-demo/pluginv1/automation_mofang_openapi" },
  @{ id = "automation/openidc_default"; pkg = "./plugin-demo/pluginv1/automation_openidc" }
)

$origGOOS = $env:GOOS
$origGOARCH = $env:GOARCH
$origCGO = $env:CGO_ENABLED
$env:CGO_ENABLED = "0"

foreach ($p in $plugins) {
  foreach ($t in $targets) {
    $key = "{0}_{1}" -f $t.goos, $t.goarch
    $outDir = Join-Path "plugins/$($p.id)" "bin/$key"
    New-Item -ItemType Directory -Force $outDir | Out-Null
    $outFile = Join-Path $outDir ("plugin" + $t.ext)
    $env:GOOS = $t.goos
    $env:GOARCH = $t.goarch
    go build -o $outFile $p.pkg
  }
}

$env:GOOS = $origGOOS
$env:GOARCH = $origGOARCH
$env:CGO_ENABLED = $origCGO

Write-Host "OK: built demo plugins into ./plugins/**/bin/<goos>_<goarch>/plugin(.exe)"

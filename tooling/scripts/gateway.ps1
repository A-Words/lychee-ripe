[CmdletBinding()]
param(
    [string]$Config = "tooling/configs/gateway.yaml"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "cache-env.ps1")

go run ./services/gateway/cmd/gateway --config $Config

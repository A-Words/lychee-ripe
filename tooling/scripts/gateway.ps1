[CmdletBinding()]
param(
    [string]$Config = "tooling/configs/gateway.yaml"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

go run ./services/gateway/cmd/gateway --config $Config

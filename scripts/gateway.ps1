[CmdletBinding()]
param(
    [string]$Config = "configs/gateway.yaml"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

go run ./gateway/cmd/gateway --config $Config

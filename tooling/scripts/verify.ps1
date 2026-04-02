[CmdletBinding()]
param(
    [ValidateSet("cpu", "cu128")]
    [string]$Target = "cpu"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "[check] Running tests..."
Push-Location "services/inference-api"
try {
    uv run --extra $Target python -m pytest -q
}
finally {
    Pop-Location
}
go test ./services/gateway/...
bun run --filter @lychee-ripe/orchard-console typecheck
bun run --filter @lychee-ripe/orchard-console test
bun run --filter @lychee-ripe/orchard-console generate

Write-Host "[check] Verifying required config examples..."
if (-not (Test-Path "tooling/configs/model.yaml.example")) {
    throw "Missing tooling/configs/model.yaml.example"
}
if (-not (Test-Path "tooling/configs/service.yaml.example")) {
    throw "Missing tooling/configs/service.yaml.example"
}
if (-not (Test-Path "tooling/configs/gateway.yaml.example")) {
    throw "Missing tooling/configs/gateway.yaml.example"
}

Write-Host "[check] OK"

[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "[check] Running tests..."
uv run pytest -q

Write-Host "[check] Verifying required config examples..."
if (-not (Test-Path "configs/model.yaml.example")) {
    throw "Missing configs/model.yaml.example"
}
if (-not (Test-Path "configs/service.yaml.example")) {
    throw "Missing configs/service.yaml.example"
}

Write-Host "[check] OK"

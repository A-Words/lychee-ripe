[CmdletBinding()]
param(
    [Alias("Host")]
    [string]$ListenHost = "127.0.0.1",
    [int]$Port = 8000,
    [ValidateSet("cpu", "cu128")]
    [string]$Target = "cpu"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "cache-env.ps1")

Push-Location "services/inference-api"
try {
    uv run --extra $Target python -m uvicorn app.main:app --reload --host $ListenHost --port $Port
}
finally {
    Pop-Location
}

[CmdletBinding()]
param(
    [string]$Host = "127.0.0.1",
    [int]$Port = 3000
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Push-Location "clients/orchard-console"
try {
    bun run dev -- --host $Host --port $Port
}
finally {
    Pop-Location
}

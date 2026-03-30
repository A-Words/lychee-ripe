[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Push-Location "clients/orchard-console"
try {
    bun run tauri:dev
}
finally {
    Pop-Location
}

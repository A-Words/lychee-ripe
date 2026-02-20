[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Push-Location "frontend"
try {
    bun run tauri:dev
}
finally {
    Pop-Location
}

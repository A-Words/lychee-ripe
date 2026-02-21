[CmdletBinding()]
param(
    [ValidateSet("cpu", "cu128", "auto")]
    [string]$Target = "auto"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Resolve-AutoTarget {
    $smi = Get-Command "nvidia-smi" -ErrorAction SilentlyContinue
    if ($null -eq $smi) {
        return "cpu"
    }

    $output = & nvidia-smi -L 2>$null
    if ($LASTEXITCODE -eq 0 -and ($output -match '^GPU \d+:')) {
        return "cu128"
    }

    return "cpu"
}

$resolvedTarget = $Target
if ($Target -eq "auto") {
    $resolvedTarget = Resolve-AutoTarget
}

$sourceLock = "uv.lock.$resolvedTarget"
if (-not (Test-Path $sourceLock)) {
    throw "Missing lock file: $sourceLock"
}

$uv = Get-Command "uv" -ErrorAction SilentlyContinue
if ($null -eq $uv) {
    throw "Missing command: uv"
}

Copy-Item -Path $sourceLock -Destination "uv.lock" -Force
Write-Host "[switch-lock] target=$resolvedTarget, source=$sourceLock"

uv sync --frozen
Write-Host "[switch-lock] synced with $sourceLock"

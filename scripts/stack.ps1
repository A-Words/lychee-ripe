[CmdletBinding()]
param(
    [string]$AppHost = "127.0.0.1",
    [int]$AppPort = 8000,
    [string]$GatewayConfig = "configs/gateway.yaml"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$appProc = $null
$gatewayProc = $null

try {
    $appArgs = @(
        "run", "uvicorn", "app.main:app",
        "--reload",
        "--host", $AppHost,
        "--port", "$AppPort"
    )
    $appProc = Start-Process -FilePath "uv" -ArgumentList $appArgs -PassThru

    $gatewayArgs = @("run", "./gateway/cmd/gateway", "--config", $GatewayConfig)
    $gatewayProc = Start-Process -FilePath "go" -ArgumentList $gatewayArgs -PassThru

    Write-Host "[stack] app pid=$($appProc.Id), gateway pid=$($gatewayProc.Id)"

    while (-not $appProc.HasExited -and -not $gatewayProc.HasExited) {
        Start-Sleep -Seconds 1
    }

    if ($appProc.HasExited) {
        Write-Host "[stack] app exited with code $($appProc.ExitCode)"
        if (-not $gatewayProc.HasExited) {
            Stop-Process -Id $gatewayProc.Id -Force
        }
        exit $appProc.ExitCode
    }

    Write-Host "[stack] gateway exited with code $($gatewayProc.ExitCode)"
    if (-not $appProc.HasExited) {
        Stop-Process -Id $appProc.Id -Force
    }
    exit $gatewayProc.ExitCode
}
finally {
    if ($null -ne $appProc -and -not $appProc.HasExited) {
        Stop-Process -Id $appProc.Id -Force -ErrorAction SilentlyContinue
    }
    if ($null -ne $gatewayProc -and -not $gatewayProc.HasExited) {
        Stop-Process -Id $gatewayProc.Id -Force -ErrorAction SilentlyContinue
    }
}

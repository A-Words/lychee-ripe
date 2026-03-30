[CmdletBinding()]
param(
    [string]$AppHost = "127.0.0.1",
    [int]$AppPort = 8000,
    [string]$GatewayConfig = "tooling/configs/gateway.yaml",
    [string]$FrontendHost = "127.0.0.1",
    [int]$FrontendPort = 3000
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$appProc = $null
$gatewayProc = $null
$frontendProc = $null

try {
    $appArgs = @(
        "run", "--project", "services/inference-api", "python", "-m", "uvicorn", "app.main:app",
        "--reload",
        "--host", $AppHost,
        "--port", "$AppPort"
    )
    $appProc = Start-Process -FilePath "uv" -ArgumentList $appArgs -PassThru

    $gatewayArgs = @("run", "./services/gateway/cmd/gateway", "--config", $GatewayConfig)
    $gatewayProc = Start-Process -FilePath "go" -ArgumentList $gatewayArgs -PassThru

    $frontendArgs = @("run", "dev", "--", "--host", $FrontendHost, "--port", "$FrontendPort")
    $frontendProc = Start-Process -FilePath "bun" -ArgumentList $frontendArgs -WorkingDirectory "clients/orchard-console" -PassThru

    Write-Host "[stack] inference-api pid=$($appProc.Id), gateway pid=$($gatewayProc.Id), orchard-console pid=$($frontendProc.Id)"

    while (-not $appProc.HasExited -and -not $gatewayProc.HasExited -and -not $frontendProc.HasExited) {
        Start-Sleep -Seconds 1
    }

    if ($appProc.HasExited) {
        Write-Host "[stack] inference-api exited with code $($appProc.ExitCode)"
        if (-not $gatewayProc.HasExited) {
            Stop-Process -Id $gatewayProc.Id -Force
        }
        if (-not $frontendProc.HasExited) {
            Stop-Process -Id $frontendProc.Id -Force
        }
        exit $appProc.ExitCode
    }

    if ($gatewayProc.HasExited) {
        Write-Host "[stack] gateway exited with code $($gatewayProc.ExitCode)"
        if (-not $appProc.HasExited) {
            Stop-Process -Id $appProc.Id -Force
        }
        if (-not $frontendProc.HasExited) {
            Stop-Process -Id $frontendProc.Id -Force
        }
        exit $gatewayProc.ExitCode
    }

    Write-Host "[stack] orchard-console exited with code $($frontendProc.ExitCode)"
    if (-not $appProc.HasExited) {
        Stop-Process -Id $appProc.Id -Force
    }
    if (-not $gatewayProc.HasExited) {
        Stop-Process -Id $gatewayProc.Id -Force
    }
    exit $frontendProc.ExitCode
}
finally {
    if ($null -ne $appProc -and -not $appProc.HasExited) {
        Stop-Process -Id $appProc.Id -Force -ErrorAction SilentlyContinue
    }
    if ($null -ne $gatewayProc -and -not $gatewayProc.HasExited) {
        Stop-Process -Id $gatewayProc.Id -Force -ErrorAction SilentlyContinue
    }
    if ($null -ne $frontendProc -and -not $frontendProc.HasExited) {
        Stop-Process -Id $frontendProc.Id -Force -ErrorAction SilentlyContinue
    }
}

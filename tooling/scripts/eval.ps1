[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Data,
    [string]$Exp = "lychee_v1",
    [int]$Imgsz = 640,
    [string]$Device = "cpu",
    [string]$Output = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$modelPath = Join-Path "mlops/artifacts/models" "$Exp/weights/best.pt"
if (-not (Test-Path $modelPath)) {
    throw "Model checkpoint not found: $modelPath"
}

if ([string]::IsNullOrWhiteSpace($Output)) {
    $Output = Join-Path "mlops/artifacts/metrics" "${Exp}-eval_metrics.json"
}

uv run --project services/inference-api python mlops/training/eval.py --model $modelPath --data $Data --imgsz $Imgsz --device $Device --output $Output

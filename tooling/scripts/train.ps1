[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Data,
    [string]$Model = "mlops/pretrained/yolo26n.pt",
    [int]$Epochs = 100,
    [int]$Imgsz = 640,
    [int]$Batch = 16,
    [string]$Device = "cpu",
    [string]$Name = "lychee_v1",
    [switch]$ExportOnnx
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$trainArgs = @(
    "run", "--project", "services/inference-api", "python", "mlops/training/train.py",
    "--data", $Data,
    "--model", $Model,
    "--epochs", $Epochs,
    "--imgsz", $Imgsz,
    "--batch", $Batch,
    "--device", $Device,
    "--project", "mlops/artifacts/models",
    "--name", $Name
)

if ($ExportOnnx) {
    $trainArgs += "--export-onnx"
}

uv @trainArgs

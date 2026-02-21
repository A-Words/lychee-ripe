[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Data,
    [string]$Model = "yolo26n.pt",
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
    "run", "python", "training/train.py",
    "--data", $Data,
    "--model", $Model,
    "--epochs", $Epochs,
    "--imgsz", $Imgsz,
    "--batch", $Batch,
    "--device", $Device,
    "--project", "artifacts/models",
    "--name", $Name
)

if ($ExportOnnx) {
    $trainArgs += "--export-onnx"
}

uv @trainArgs

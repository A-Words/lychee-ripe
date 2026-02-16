[CmdletBinding()]
param(
    [string]$Host = "127.0.0.1",
    [int]$Port = 8000
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

uv run uvicorn app.main:app --reload --host $Host --port $Port

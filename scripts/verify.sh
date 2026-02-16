#!/usr/bin/env sh
set -eu

echo "[check] Running tests..."
uv run pytest -q
go test ./gateway/...

echo "[check] Verifying required config examples..."
[ -f "configs/model.yaml.example" ] || { echo "Missing configs/model.yaml.example" >&2; exit 1; }
[ -f "configs/service.yaml.example" ] || { echo "Missing configs/service.yaml.example" >&2; exit 1; }
[ -f "configs/gateway.yaml.example" ] || { echo "Missing configs/gateway.yaml.example" >&2; exit 1; }

echo "[check] OK"

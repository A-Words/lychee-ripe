#!/usr/bin/env sh
set -eu

echo "[check] Running tests..."
(cd services/inference-api && uv run python -m pytest -q)
go test ./services/gateway/...
bun run --filter @lychee-ripe/orchard-console typecheck
bun run --filter @lychee-ripe/orchard-console test
bun run --filter @lychee-ripe/orchard-console generate

echo "[check] Verifying required config examples..."
[ -f "tooling/configs/model.yaml.example" ] || { echo "Missing tooling/configs/model.yaml.example" >&2; exit 1; }
[ -f "tooling/configs/service.yaml.example" ] || { echo "Missing tooling/configs/service.yaml.example" >&2; exit 1; }
[ -f "tooling/configs/gateway.yaml.example" ] || { echo "Missing tooling/configs/gateway.yaml.example" >&2; exit 1; }

echo "[check] OK"

#!/usr/bin/env sh
set -eu

TARGET="${TARGET:-cpu}"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --target)
      TARGET="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

case "$TARGET" in
  cpu|cu128)
    ;;
  *)
    echo "Invalid --target '$TARGET'. Expected cpu|cu128." >&2
    exit 1
    ;;
esac

echo "[check] Running tests..."
(cd services/inference-api && uv run --extra "$TARGET" python -m pytest -q)
go test ./services/gateway/...
bun run --filter @lychee-ripe/orchard-console typecheck
bun run --filter @lychee-ripe/orchard-console test
bun run --filter @lychee-ripe/orchard-console generate

echo "[check] Verifying required config examples..."
[ -f "tooling/configs/model.yaml.example" ] || { echo "Missing tooling/configs/model.yaml.example" >&2; exit 1; }
[ -f "tooling/configs/service.yaml.example" ] || { echo "Missing tooling/configs/service.yaml.example" >&2; exit 1; }
[ -f "tooling/configs/gateway.yaml.example" ] || { echo "Missing tooling/configs/gateway.yaml.example" >&2; exit 1; }

echo "[check] OK"

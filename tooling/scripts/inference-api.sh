#!/usr/bin/env sh
set -eu

. "$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)/cache-env.sh"

HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-8000}"
TARGET="${TARGET:-cpu}"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --host)
      HOST="$2"
      shift 2
      ;;
    --port)
      PORT="$2"
      shift 2
      ;;
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

cd services/inference-api
uv run --extra "$TARGET" python -m uvicorn app.main:app --reload --host "$HOST" --port "$PORT"

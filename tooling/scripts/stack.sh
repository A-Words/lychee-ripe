#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
APP_HOST="${APP_HOST:-127.0.0.1}"
APP_PORT="${APP_PORT:-8000}"
TARGET="${TARGET:-cpu}"
GATEWAY_CONFIG="${GATEWAY_CONFIG:-tooling/configs/gateway.yaml}"
FRONTEND_HOST="${FRONTEND_HOST:-127.0.0.1}"
FRONTEND_PORT="${FRONTEND_PORT:-3000}"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --app-host)
      APP_HOST="$2"
      shift 2
      ;;
    --app-port)
      APP_PORT="$2"
      shift 2
      ;;
    --target)
      TARGET="$2"
      shift 2
      ;;
    --gateway-config)
      GATEWAY_CONFIG="$2"
      shift 2
      ;;
    --frontend-host)
      FRONTEND_HOST="$2"
      shift 2
      ;;
    --frontend-port)
      FRONTEND_PORT="$2"
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

APP_PID=""
GW_PID=""
FE_PID=""

cleanup() {
  if [ -n "$APP_PID" ] && kill -0 "$APP_PID" 2>/dev/null; then
    kill "$APP_PID" 2>/dev/null || true
  fi
  if [ -n "$GW_PID" ] && kill -0 "$GW_PID" 2>/dev/null; then
    kill "$GW_PID" 2>/dev/null || true
  fi
  if [ -n "$FE_PID" ] && kill -0 "$FE_PID" 2>/dev/null; then
    kill "$FE_PID" 2>/dev/null || true
  fi
  if [ -n "$APP_PID" ]; then
    wait "$APP_PID" 2>/dev/null || true
  fi
  if [ -n "$GW_PID" ]; then
    wait "$GW_PID" 2>/dev/null || true
  fi
  if [ -n "$FE_PID" ]; then
    wait "$FE_PID" 2>/dev/null || true
  fi
}

trap cleanup INT TERM EXIT

(cd "$ROOT_DIR/services/inference-api" && uv run --extra "$TARGET" python -m uvicorn app.main:app --reload --host "$APP_HOST" --port "$APP_PORT") &
APP_PID="$!"
(cd "$ROOT_DIR/services/gateway" && go run ./cmd/gateway --config "$GATEWAY_CONFIG") &
GW_PID="$!"
(cd "$ROOT_DIR/clients/orchard-console" && bun run dev -- --host "$FRONTEND_HOST" --port "$FRONTEND_PORT") &
FE_PID="$!"

echo "[stack] inference-api pid=$APP_PID, gateway pid=$GW_PID, orchard-console pid=$FE_PID"

EXIT_CODE=0
while :; do
  if ! kill -0 "$APP_PID" 2>/dev/null; then
    wait "$APP_PID" || EXIT_CODE=$?
    echo "[stack] inference-api exited"
    break
  fi

  if ! kill -0 "$GW_PID" 2>/dev/null; then
    wait "$GW_PID" || EXIT_CODE=$?
    echo "[stack] gateway exited"
    break
  fi

  if ! kill -0 "$FE_PID" 2>/dev/null; then
    wait "$FE_PID" || EXIT_CODE=$?
    echo "[stack] orchard-console exited"
    break
  fi

  sleep 1
done

trap - INT TERM EXIT
cleanup
exit "$EXIT_CODE"

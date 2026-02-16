#!/usr/bin/env sh
set -eu

APP_HOST="${APP_HOST:-127.0.0.1}"
APP_PORT="${APP_PORT:-8000}"
GATEWAY_CONFIG="${GATEWAY_CONFIG:-configs/gateway.yaml}"

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
    --gateway-config)
      GATEWAY_CONFIG="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

APP_PID=""
GW_PID=""

cleanup() {
  if [ -n "$APP_PID" ] && kill -0 "$APP_PID" 2>/dev/null; then
    kill "$APP_PID" 2>/dev/null || true
  fi
  if [ -n "$GW_PID" ] && kill -0 "$GW_PID" 2>/dev/null; then
    kill "$GW_PID" 2>/dev/null || true
  fi
  if [ -n "$APP_PID" ]; then
    wait "$APP_PID" 2>/dev/null || true
  fi
  if [ -n "$GW_PID" ]; then
    wait "$GW_PID" 2>/dev/null || true
  fi
}

trap cleanup INT TERM EXIT

uv run uvicorn app.main:app --reload --host "$APP_HOST" --port "$APP_PORT" &
APP_PID="$!"
go run ./gateway/cmd/gateway --config "$GATEWAY_CONFIG" &
GW_PID="$!"

echo "[stack] app pid=$APP_PID, gateway pid=$GW_PID"

EXIT_CODE=0
while :; do
  if ! kill -0 "$APP_PID" 2>/dev/null; then
    wait "$APP_PID" || EXIT_CODE=$?
    echo "[stack] app exited"
    break
  fi

  if ! kill -0 "$GW_PID" 2>/dev/null; then
    wait "$GW_PID" || EXIT_CODE=$?
    echo "[stack] gateway exited"
    break
  fi

  sleep 1
done

trap - INT TERM EXIT
cleanup
exit "$EXIT_CODE"

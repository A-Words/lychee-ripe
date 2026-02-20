#!/usr/bin/env sh
set -eu

HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-3000}"

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
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

cd frontend
bun run dev -- --host "$HOST" --port "$PORT"

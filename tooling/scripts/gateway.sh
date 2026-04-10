#!/usr/bin/env sh
set -eu

. "$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)/cache-env.sh"

CONFIG="${CONFIG:-tooling/configs/gateway.yaml}"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --config)
      CONFIG="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

go run ./services/gateway/cmd/gateway --config "$CONFIG"

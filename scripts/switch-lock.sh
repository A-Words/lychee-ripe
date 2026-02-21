#!/usr/bin/env sh
set -eu

TARGET="auto"

usage() {
  echo "Usage: sh scripts/switch-lock.sh --target cpu|cu128|auto" >&2
}

resolve_auto_target() {
  if ! command -v nvidia-smi >/dev/null 2>&1; then
    echo "cpu"
    return
  fi

  if output="$(nvidia-smi -L 2>/dev/null)"; then
    if printf '%s\n' "$output" | grep -Eq '^GPU [0-9]+:'; then
      echo "cu128"
      return
    fi
  fi

  echo "cpu"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --target)
      TARGET="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

case "$TARGET" in
  cpu|cu128)
    RESOLVED_TARGET="$TARGET"
    ;;
  auto)
    RESOLVED_TARGET="$(resolve_auto_target)"
    ;;
  *)
    echo "Invalid --target '$TARGET'. Expected cpu|cu128|auto." >&2
    usage
    exit 1
    ;;
esac

SOURCE_LOCK="uv.lock.${RESOLVED_TARGET}"
if [ ! -f "$SOURCE_LOCK" ]; then
  echo "Missing lock file: $SOURCE_LOCK" >&2
  exit 1
fi

if ! command -v uv >/dev/null 2>&1; then
  echo "Missing command: uv" >&2
  exit 1
fi

cp "$SOURCE_LOCK" "uv.lock"
echo "[switch-lock] target=${RESOLVED_TARGET}, source=${SOURCE_LOCK}"

uv sync --frozen
echo "[switch-lock] synced with ${SOURCE_LOCK}"

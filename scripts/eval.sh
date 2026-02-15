#!/usr/bin/env sh
set -eu

DATA=""
EXP="lychee_v1"
IMGSZ="640"
DEVICE="0"
OUTPUT=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --data)
      DATA="$2"
      shift 2
      ;;
    --exp)
      EXP="$2"
      shift 2
      ;;
    --imgsz)
      IMGSZ="$2"
      shift 2
      ;;
    --device)
      DEVICE="$2"
      shift 2
      ;;
    --output)
      OUTPUT="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [ -z "$DATA" ]; then
  echo "Missing required argument: --data <path>" >&2
  exit 1
fi

MODEL_PATH="artifacts/models/$EXP/weights/best.pt"
if [ ! -f "$MODEL_PATH" ]; then
  echo "Model checkpoint not found: $MODEL_PATH" >&2
  exit 1
fi

if [ -z "$OUTPUT" ]; then
  OUTPUT="artifacts/metrics/${EXP}-eval_metrics.json"
fi

uv run python training/eval.py \
  --model "$MODEL_PATH" \
  --data "$DATA" \
  --imgsz "$IMGSZ" \
  --device "$DEVICE" \
  --output "$OUTPUT"

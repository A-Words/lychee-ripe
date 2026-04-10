#!/usr/bin/env sh
set -eu

. "$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)/cache-env.sh"

DATA=""
MODEL="mlops/pretrained/yolo26n.pt"
EPOCHS="100"
IMGSZ="640"
BATCH="16"
DEVICE="cpu"
NAME="lychee_v1"
TARGET="cpu"
EXPORT_ONNX="0"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --data)
      DATA="$2"
      shift 2
      ;;
    --model)
      MODEL="$2"
      shift 2
      ;;
    --epochs)
      EPOCHS="$2"
      shift 2
      ;;
    --imgsz)
      IMGSZ="$2"
      shift 2
      ;;
    --batch)
      BATCH="$2"
      shift 2
      ;;
    --device)
      DEVICE="$2"
      shift 2
      ;;
    --name)
      NAME="$2"
      shift 2
      ;;
    --target)
      TARGET="$2"
      shift 2
      ;;
    --export-onnx)
      EXPORT_ONNX="1"
      shift 1
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

case "$TARGET" in
  cpu|cu128)
    ;;
  *)
    echo "Invalid --target '$TARGET'. Expected cpu|cu128." >&2
    exit 1
    ;;
esac

if [ "$EXPORT_ONNX" = "1" ]; then
  uv run --project services/inference-api --extra "$TARGET" python mlops/training/train.py \
    --data "$DATA" \
    --model "$MODEL" \
    --epochs "$EPOCHS" \
    --imgsz "$IMGSZ" \
    --batch "$BATCH" \
    --device "$DEVICE" \
    --project "mlops/artifacts/models" \
    --name "$NAME" \
    --export-onnx
else
  uv run --project services/inference-api --extra "$TARGET" python mlops/training/train.py \
    --data "$DATA" \
    --model "$MODEL" \
    --epochs "$EPOCHS" \
    --imgsz "$IMGSZ" \
    --batch "$BATCH" \
    --device "$DEVICE" \
    --project "mlops/artifacts/models" \
    --name "$NAME"
fi

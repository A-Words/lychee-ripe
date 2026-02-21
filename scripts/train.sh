#!/usr/bin/env sh
set -eu

DATA=""
MODEL="yolo26n.pt"
EPOCHS="100"
IMGSZ="640"
BATCH="16"
DEVICE="cpu"
NAME="lychee_v1"
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

if [ "$EXPORT_ONNX" = "1" ]; then
  uv run python training/train.py \
    --data "$DATA" \
    --model "$MODEL" \
    --epochs "$EPOCHS" \
    --imgsz "$IMGSZ" \
    --batch "$BATCH" \
    --device "$DEVICE" \
    --project "artifacts/models" \
    --name "$NAME" \
    --export-onnx
else
  uv run python training/train.py \
    --data "$DATA" \
    --model "$MODEL" \
    --epochs "$EPOCHS" \
    --imgsz "$IMGSZ" \
    --batch "$BATCH" \
    --device "$DEVICE" \
    --project "artifacts/models" \
    --name "$NAME"
fi

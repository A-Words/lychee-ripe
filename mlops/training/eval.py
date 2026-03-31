from __future__ import annotations

import argparse
import json
from pathlib import Path

from lychee_common.device import resolve_torch_device

SCRIPT_DIR = Path(__file__).resolve().parent
MLOPS_DIR = SCRIPT_DIR.parent
REPO_ROOT = MLOPS_DIR.parent


def is_explicit_relative_path(raw_path: str) -> bool:
    return raw_path in {'.', '..'} or raw_path.startswith('./') or raw_path.startswith('../') or raw_path.startswith('.\\') or raw_path.startswith('..\\')


def resolve_input_path(raw_path: str) -> Path:
    path = Path(raw_path)
    if path.is_absolute():
        return path

    candidates = [
        Path.cwd() / path,
        SCRIPT_DIR / path,
        MLOPS_DIR / path,
        REPO_ROOT / path,
    ]
    for candidate in candidates:
        if candidate.exists():
            return candidate

    if is_explicit_relative_path(raw_path):
        return (Path.cwd() / path).resolve()
    return (REPO_ROOT / path).resolve()


def resolve_output_path(raw_path: str) -> Path:
    path = Path(raw_path)
    if path.is_absolute():
        return path

    if is_explicit_relative_path(raw_path):
        return (Path.cwd() / path).resolve()
    return (REPO_ROOT / path).resolve()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description='Evaluate YOLO model for lychee ripeness')
    parser.add_argument('--model', required=True, help='Path to .pt checkpoint')
    parser.add_argument('--data', required=True, help='Path to data YAML')
    parser.add_argument('--imgsz', type=int, default=640)
    parser.add_argument('--device', default='cpu')
    parser.add_argument('--output', default='mlops/artifacts/metrics/eval_metrics.json')
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    resolved_device, device_warning = resolve_torch_device(args.device)
    if device_warning:
        print(f'[device] {device_warning}')

    from ultralytics import YOLO

    model_path = resolve_input_path(args.model)
    data_path = resolve_input_path(args.data)
    model = YOLO(str(model_path))
    metrics = model.val(data=str(data_path), imgsz=args.imgsz, device=resolved_device)

    out_path = resolve_output_path(args.output)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    payload = {
        'mAP50': float(getattr(metrics.box, 'map50', 0.0)),
        'mAP50_95': float(getattr(metrics.box, 'map', 0.0)),
    }
    out_path.write_text(json.dumps(payload, indent=2), encoding='utf-8')
    print(f'Metrics written to: {out_path}')


if __name__ == '__main__':
    main()

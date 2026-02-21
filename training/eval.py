from __future__ import annotations

import argparse
import json
from pathlib import Path

from app.settings import resolve_torch_device


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description='Evaluate YOLO model for lychee ripeness')
    parser.add_argument('--model', required=True, help='Path to .pt checkpoint')
    parser.add_argument('--data', required=True, help='Path to data YAML')
    parser.add_argument('--imgsz', type=int, default=640)
    parser.add_argument('--device', default='cpu')
    parser.add_argument('--output', default='artifacts/metrics/eval_metrics.json')
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    resolved_device, device_warning = resolve_torch_device(args.device)
    if device_warning:
        print(f'[device] {device_warning}')

    from ultralytics import YOLO

    model = YOLO(args.model)
    metrics = model.val(data=args.data, imgsz=args.imgsz, device=resolved_device)

    out_path = Path(args.output)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    payload = {
        'mAP50': float(getattr(metrics.box, 'map50', 0.0)),
        'mAP50_95': float(getattr(metrics.box, 'map', 0.0)),
    }
    out_path.write_text(json.dumps(payload, indent=2), encoding='utf-8')
    print(f'Metrics written to: {out_path}')


if __name__ == '__main__':
    main()

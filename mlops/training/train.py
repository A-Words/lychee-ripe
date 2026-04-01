from __future__ import annotations

import argparse
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
    parser = argparse.ArgumentParser(description='Train YOLO stable baseline for lychee ripeness')
    parser.add_argument('--data', required=True, help='Path to data YAML')
    parser.add_argument('--model', default='mlops/pretrained/yolo26n.pt', help='Pretrained model checkpoint')
    parser.add_argument('--epochs', type=int, default=100)
    parser.add_argument('--imgsz', type=int, default=640)
    parser.add_argument('--batch', type=int, default=16)
    parser.add_argument('--device', default='cpu')
    parser.add_argument('--project', default='mlops/artifacts/models')
    parser.add_argument('--name', default='lychee_v1')
    parser.add_argument('--export-onnx', action='store_true')
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    resolved_device, device_warning = resolve_torch_device(args.device)
    if device_warning:
        print(f'[device] {device_warning}')

    from ultralytics import YOLO

    data_path = resolve_input_path(args.data)
    model_path = resolve_input_path(args.model)
    project_path = resolve_output_path(args.project)

    model = YOLO(str(model_path))
    results = model.train(
        data=str(data_path),
        epochs=args.epochs,
        imgsz=args.imgsz,
        batch=args.batch,
        device=resolved_device,
        project=str(project_path),
        name=args.name,
    )

    save_dir = Path(results.save_dir)
    print(f'Training done: {save_dir}')

    best_pt = save_dir / 'weights' / 'best.pt'
    if best_pt.exists():
        print(f'Best checkpoint: {best_pt}')

    if args.export_onnx and best_pt.exists():
        export_model = YOLO(str(best_pt))
        export_path = export_model.export(format='onnx')
        print(f'ONNX exported: {export_path}')


if __name__ == '__main__':
    main()

# lychee-ripe

Lychee detection and ripeness classification service.

## Features
- FastAPI service with `v1` endpoints
- Image inference (`POST /v1/infer/image`)
- Stream inference (`WS /v1/infer/stream`)
- Session-level ripeness statistics and harvest suggestion
- Configurable YOLO runtime version via `configs/model.yaml` (`yolo_version`)

## Quick start
```bash
uv sync
uv run uvicorn app.main:app --reload
```

## Dev commands
```bash
uv run pytest -q
uv run python training/train.py --data path/to/data.yaml --model yolov8n.pt
uv run python training/eval.py --model artifacts/train/lychee_v1/weights/best.pt --data path/to/data.yaml
```

## Dataset layout
- Recommended path in this repo: `data/lichi/`
- Keep raw datasets out of git (already ignored via `.gitignore`).
- Current label mapping (4 classes): `0=green`, `1=half`, `2=red`, `3=young`.

## Export requirements (optional compatibility)
```bash
uv export --no-hashes -o requirements.txt
```

## Config
- `configs/model.yaml.example` (copy to `configs/model.yaml` for local use)
- `configs/service.yaml.example` (copy to `configs/service.yaml` for local use)
- `model_path` precedence: if non-empty, use it directly; if empty, fallback to `${yolo_version}.pt`

Override paths with env vars:
- `LYCHEE_MODEL_CONFIG`
- `LYCHEE_SERVICE_CONFIG`

## Local config files (not committed)
- Copy `configs/model.yaml.example` to `configs/model.yaml`.
- Copy `configs/service.yaml.example` to `configs/service.yaml`.
- Any `configs/*.yaml` is ignored by git; only `configs/*.yaml.example` is tracked.
- Edit local files for machine-specific settings (e.g. `device: "cuda:0"`).
- Startup fails fast if either local config file is missing.
- Start service (defaults to these local files):
```powershell
uv run uvicorn app.main:app --reload
```

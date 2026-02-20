# lychee-ripe

Lychee detection and ripeness classification service.

## Features
- FastAPI service with `v1` endpoints
- Image inference (`POST /v1/infer/image`)
- Stream inference (`WS /v1/infer/stream`)
- Session-level ripeness statistics and harvest suggestion
- Configurable YOLO runtime version via `configs/model.yaml` (`yolo_version`)
- Optional Go gateway layer for external API, auth/rate limit, and orchestration
- Nuxt 4 + Nuxt UI frontend for live camera inference overlays (Web + Tauri desktop shell)

## Quick start
```bash
uv sync
uv run uvicorn app.main:app --reload
```

## Dev commands
```bash
uv run pytest -q
uv run python training/train.py --data path/to/data.yaml --model yolo26n.pt
uv run python training/eval.py --model artifacts/models/lychee_v1/weights/best.pt --data path/to/data.yaml
go run ./gateway/cmd/gateway
go test ./gateway/...
bun install --cwd frontend
bun run --cwd frontend dev
bun run --cwd frontend typecheck
bun run --cwd frontend test
bun run --cwd frontend generate
bun run --cwd frontend tauri:dev
```

## Script shortcuts (sh)
```bash
sh scripts/app.sh --host 127.0.0.1 --port 8000
sh scripts/gateway.sh --config configs/gateway.yaml
sh scripts/stack.sh --app-host 127.0.0.1 --app-port 8000 --gateway-config configs/gateway.yaml
sh scripts/frontend.sh --host 127.0.0.1 --port 3000
sh scripts/desktop.sh
sh scripts/train.sh --data data/lichi/data.yaml --name lychee_v1
sh scripts/eval.sh --data data/lichi/data.yaml --exp lychee_v1
sh scripts/verify.sh
```

## Script shortcuts (PowerShell)
```powershell
powershell -ExecutionPolicy Bypass -File scripts/app.ps1 -Host 127.0.0.1 -Port 8000
powershell -ExecutionPolicy Bypass -File scripts/gateway.ps1 -Config configs/gateway.yaml
powershell -ExecutionPolicy Bypass -File scripts/stack.ps1 -AppHost 127.0.0.1 -AppPort 8000 -GatewayConfig configs/gateway.yaml
powershell -ExecutionPolicy Bypass -File scripts/frontend.ps1 -Host 127.0.0.1 -Port 3000
powershell -ExecutionPolicy Bypass -File scripts/desktop.ps1
powershell -ExecutionPolicy Bypass -File scripts/train.ps1 -Data data/lichi/data.yaml -Name lychee_v1
powershell -ExecutionPolicy Bypass -File scripts/eval.ps1 -Data data/lichi/data.yaml -Exp lychee_v1
powershell -ExecutionPolicy Bypass -File scripts/verify.ps1
```

## Project structure
- `app/`: FastAPI inference service and API endpoints
- `gateway/`: Go gateway service for external API access and request orchestration
- `training/`: training and evaluation scripts
- `tests/`: unit, integration, and performance tests
- `frontend/`: frontend visualization client
- `shared/`: shared contracts and constants between backend/frontend
- `configs/`: local and example configuration files
- `data/`: dataset workspace (`raw/`, `processed/`, `samples/`, `lichi/`)
- `artifacts/`: model artifacts, metrics, and runtime logs
- `scripts/`: automation scripts for dev, train, eval, and checks
- `docker/`: container build assets
- `docs/`: project documentation

## Dataset layout
- Recommended path in this repo: `data/lichi/`
- Keep raw datasets out of git (already ignored via `.gitignore`).
- Current label mapping (4 classes): `0=green`, `1=half`, `2=red`, `3=young`.

## Dataset source
- Zhiqing, Zhao (2025), "lichi-maturity", Mendeley Data, V1, doi: `10.17632/c3rk9gv4w9.1`

## Export requirements (optional compatibility)
```bash
uv export --no-hashes -o requirements.txt
```

## Config
- `configs/model.yaml.example` (copy to `configs/model.yaml` for local use)
- `configs/service.yaml.example` (copy to `configs/service.yaml` for local use)
- `configs/gateway.yaml.example` (copy to `configs/gateway.yaml` for local use)
- `frontend/.env.example` (copy to `frontend/.env` for local frontend config)
- `model_path` precedence: if non-empty, use it directly; if empty, fallback to `${yolo_version}.pt`

Override paths with env vars:
- `LYCHEE_MODEL_CONFIG`
- `LYCHEE_SERVICE_CONFIG`
- `LYCHEE_GATEWAY_CONFIG`
- `NUXT_PUBLIC_GATEWAY_BASE` (frontend gateway URL, default: `http://127.0.0.1:9000`)

## Frontend quick start (Web)
```bash
uv sync
go run ./gateway/cmd/gateway
bun install --cwd frontend
bun run --cwd frontend dev
```

Open `http://127.0.0.1:3000`, click **Start**, grant camera permission, and verify detection boxes and ripeness labels are overlaid on video.

## Frontend quick start (Desktop / Tauri)
```bash
bun install --cwd frontend
bun run --cwd frontend tauri:dev
```

Notes:
- This repo currently ships the Tauri 2 project structure in `frontend/src-tauri`.
- You need Rust/Cargo and Tauri desktop dependencies installed locally for `tauri:dev`.
- Desktop build packaging (`tauri build`) is intentionally out of scope in this first phase.

## First-phase frontend behavior and limits
- Default stream profile: `640x360`, ~`5 FPS`, JPEG frames.
- Data path is fixed: `frontend -> gateway -> app`.
- Gateway auth is expected to stay disabled for local first-phase integration (`configs/gateway.yaml.example`).
- The frontend currently focuses on live overlay + current frame stats + session summary; no history playback/charting yet.

## API contract and service boundary
- Keep OpenAPI at `shared/schemas/openapi.yaml` as the single source of truth for external API fields.
- Recommended call path: `frontend -> gateway -> app`.
- Frontend should use gateway APIs only and avoid direct calls to `app`.

## Local config files (not committed)
- Copy `configs/model.yaml.example` to `configs/model.yaml`.
- Copy `configs/service.yaml.example` to `configs/service.yaml`.
- Copy `configs/gateway.yaml.example` to `configs/gateway.yaml`.
- Any `configs/*.yaml` is ignored by git; only `configs/*.yaml.example` is tracked.
- Edit local files for machine-specific settings (e.g. `device: "cuda:0"`).
- Startup fails fast if either local config file is missing.
- Start service (defaults to these local files):
```powershell
uv run uvicorn app.main:app --reload
```

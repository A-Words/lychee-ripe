from __future__ import annotations

import os
from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI

from app.api.v1.endpoints import router as v1_router
from app.inference.factory import build_detector
from app.inference.pipeline import InferencePipeline
from app.settings import (
    ServiceConfig,
    load_model_config,
    load_service_config,
)


def _ensure_config_file(path: Path, env_var: str) -> None:
    if path.exists():
        return

    example_path = Path(f"{path}.example")
    if example_path.exists():
        hint = f" Copy '{example_path.as_posix()}' to '{path.as_posix()}'."
    else:
        hint = ""

    raise RuntimeError(
        f"Missing config file: '{path.as_posix()}'. "
        f"Set {env_var} to an existing file.{hint}"
    )


@asynccontextmanager
async def lifespan(app: FastAPI):
    model_cfg_path = Path(os.getenv('LYCHEE_MODEL_CONFIG', 'configs/model.yaml'))
    service_cfg_path = Path(os.getenv('LYCHEE_SERVICE_CONFIG', 'configs/service.yaml'))
    _ensure_config_file(model_cfg_path, "LYCHEE_MODEL_CONFIG")
    _ensure_config_file(service_cfg_path, "LYCHEE_SERVICE_CONFIG")

    model_cfg = load_model_config(model_cfg_path)
    service_cfg: ServiceConfig = load_service_config(service_cfg_path)

    detector = build_detector(model_cfg)
    try:
        detector.load()
        detector.warmup()
    except Exception:
        # Keep service booted in degraded mode for health visibility.
        pass

    app.state.service_cfg = service_cfg
    app.state.pipeline = InferencePipeline(
        detector=detector,
        model_version=model_cfg.model_version,
        schema_version=service_cfg.schema_version,
    )

    yield


app = FastAPI(title='lychee-ripe', version='0.1.0', lifespan=lifespan)
app.include_router(v1_router, prefix='/v1', tags=['v1'])

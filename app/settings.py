from __future__ import annotations

import re
from pathlib import Path

from pydantic import BaseModel, Field

try:
    import yaml  # type: ignore
except Exception:  # pragma: no cover
    yaml = None

DEFAULT_SCHEMA_VERSION = "v1"
_CUDA_DEVICE_PATTERN = re.compile(r"^\d+(,\d+)*$")


class ModelConfig(BaseModel):
    yolo_version: str = "yolo26n"
    model_version: str = "1.0.0"
    model_path: str = ""
    conf_threshold: float = Field(default=0.25, ge=0.0, le=1.0)
    nms_iou: float = Field(default=0.45, ge=0.0, le=1.0)
    device: str = "auto"


class ServiceConfig(BaseModel):
    app_name: str = "lychee-ripe"
    api_prefix: str = "/v1"
    schema_version: str = DEFAULT_SCHEMA_VERSION
    max_upload_mb: int = 10


def _torch_cuda_runtime() -> tuple[bool, int]:
    try:
        import torch
    except Exception:
        return False, 0

    try:
        return bool(torch.cuda.is_available()), int(torch.cuda.device_count())
    except Exception:
        return False, 0


def _requests_cuda(device: str) -> bool:
    normalized = device.strip().lower()
    if not normalized or normalized in {"cpu", "mps"}:
        return False
    if normalized.startswith("cuda"):
        return True
    return bool(_CUDA_DEVICE_PATTERN.fullmatch(normalized))


def resolve_torch_device(device: str) -> tuple[str, str | None]:
    requested = (device or "").strip() or "cpu"
    cuda_available, cuda_device_count = _torch_cuda_runtime()

    if requested.lower() == "auto":
        if cuda_available and cuda_device_count > 0:
            return "cuda:0", None
        return (
            "cpu",
            (
                "Requested device='auto' but CUDA is unavailable "
                f"(torch.cuda.is_available()={cuda_available}, "
                f"torch.cuda.device_count()={cuda_device_count}). Falling back to CPU."
            ),
        )

    if _requests_cuda(requested) and (not cuda_available or cuda_device_count <= 0):
        return (
            "cpu",
            (
                f"Requested device='{requested}' but CUDA is unavailable "
                f"(torch.cuda.is_available()={cuda_available}, "
                f"torch.cuda.device_count()={cuda_device_count}). Falling back to CPU."
            ),
        )

    return requested, None


def _parse_simple_yaml(text: str) -> dict:
    data: dict[str, object] = {}
    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line or line.startswith('#') or ':' not in line:
            continue
        key, value = line.split(':', 1)
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        low = value.lower()
        if low in {'true', 'false'}:
            data[key] = low == 'true'
            continue
        try:
            if '.' in value:
                data[key] = float(value)
            else:
                data[key] = int(value)
            continue
        except ValueError:
            data[key] = value
    return data


def _load_yaml(path: Path) -> dict:
    if not path.exists():
        return {}
    text = path.read_text(encoding='utf-8')
    if yaml is not None:
        data = yaml.safe_load(text) or {}
        if not isinstance(data, dict):
            raise ValueError(f"Config file must contain a YAML mapping: {path}")
        return data
    return _parse_simple_yaml(text)


def load_model_config(path: Path) -> ModelConfig:
    return ModelConfig.model_validate(_load_yaml(path))


def load_service_config(path: Path) -> ServiceConfig:
    return ServiceConfig.model_validate(_load_yaml(path))

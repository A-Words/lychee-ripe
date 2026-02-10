from __future__ import annotations

from pathlib import Path

from pydantic import BaseModel, Field

try:
    import yaml  # type: ignore
except Exception:  # pragma: no cover
    yaml = None

DEFAULT_SCHEMA_VERSION = "v1"


class ModelConfig(BaseModel):
    yolo_version: str = "yolo11n"
    model_version: str = "1.0.0"
    model_path: str = ""
    conf_threshold: float = Field(default=0.25, ge=0.0, le=1.0)
    nms_iou: float = Field(default=0.45, ge=0.0, le=1.0)
    device: str = "cpu"


class ServiceConfig(BaseModel):
    app_name: str = "lychee-ripe"
    api_prefix: str = "/v1"
    schema_version: str = DEFAULT_SCHEMA_VERSION
    max_upload_mb: int = 10


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

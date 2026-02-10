from __future__ import annotations

from app.inference.adapters.base import DetectorAdapter
from app.inference.adapters.yolo_stable import YoloStableAdapter
from app.settings import ModelConfig


def build_detector(cfg: ModelConfig) -> DetectorAdapter:
    return YoloStableAdapter(cfg)

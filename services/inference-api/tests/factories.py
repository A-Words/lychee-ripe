from __future__ import annotations

import numpy as np

from app.inference.adapters.base import DetectorAdapter, RawDetection
from app.inference.pipeline import InferencePipeline


class FakeDetector(DetectorAdapter):
    name = "fake"

    def __init__(
        self,
        *,
        detections: list[RawDetection] | None = None,
        ripeness: str = "half",
        loaded: bool = True,
    ) -> None:
        self._loaded = loaded
        self._detections = detections or [build_raw_detection()]
        self._ripeness = ripeness

    @property
    def loaded(self) -> bool:
        return self._loaded

    def load(self) -> None:
        self._loaded = True

    def warmup(self) -> None:
        return

    def predict(self, frame: np.ndarray) -> list[RawDetection]:
        return list(self._detections)

    def ripeness_from_class_id(self, class_id: int) -> str:
        return self._ripeness


def build_raw_detection(
    *,
    bbox: tuple[float, float, float, float] = (10, 10, 100, 100),
    class_id: int = 1,
    confidence: float = 0.9,
) -> RawDetection:
    return RawDetection(bbox=bbox, class_id=class_id, confidence=confidence)


def build_frame(
    *,
    height: int = 240,
    width: int = 320,
    channels: int = 3,
    fill_value: int = 0,
) -> np.ndarray:
    return np.full((height, width, channels), fill_value, dtype=np.uint8)


def build_pipeline(
    *,
    detector: DetectorAdapter | None = None,
    model_version: str = "1.0.0",
    schema_version: str = "v1",
) -> InferencePipeline:
    return InferencePipeline(detector or FakeDetector(), model_version=model_version, schema_version=schema_version)

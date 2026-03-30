from __future__ import annotations

import numpy as np

from app.inference.adapters.base import DetectorAdapter, RawDetection
from app.inference.pipeline import InferencePipeline


class FakeDetector(DetectorAdapter):
    name = 'fake'

    def __init__(self) -> None:
        self._loaded = True

    @property
    def loaded(self) -> bool:
        return self._loaded

    def load(self) -> None:
        self._loaded = True

    def warmup(self) -> None:
        return

    def predict(self, frame: np.ndarray):
        return [RawDetection(bbox=(10, 10, 100, 100), class_id=1, confidence=0.9)]

    def ripeness_from_class_id(self, class_id: int) -> str:
        return 'half'


def test_infer_image_success() -> None:
    pipeline = InferencePipeline(FakeDetector(), model_version='1.0.0', schema_version='v1')
    frame = np.zeros((240, 320, 3), dtype=np.uint8)
    result, inference_ms = pipeline.infer_image(frame)

    assert result.frame_index == 0
    assert result.frame_summary.total == 1
    assert result.detections[0].ripeness == 'half'
    assert inference_ms >= 0

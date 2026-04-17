from __future__ import annotations

from tests.factories import FakeDetector, build_frame
from app.inference.pipeline import InferencePipeline


def test_infer_image_success() -> None:
    pipeline = InferencePipeline(FakeDetector(), model_version='1.0.0', schema_version='v1')
    frame = build_frame()
    result, inference_ms = pipeline.infer_image(frame)

    assert result.frame_index == 0
    assert result.frame_summary.total == 1
    assert result.detections[0].ripeness == 'half'
    assert inference_ms >= 0

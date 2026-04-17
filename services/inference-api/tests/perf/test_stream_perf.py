from __future__ import annotations

import pytest

from tests.factories import build_frame, build_raw_detection


@pytest.mark.perf
def test_stream_contract(test_client, install_pipeline, decode_image_to_frame, sample_image_bytes, fake_detector_factory) -> None:
    decode_image_to_frame(build_frame(height=120, width=120))
    install_pipeline(
        detector=fake_detector_factory(
            detections=[build_raw_detection(bbox=(1, 1, 20, 20), class_id=1, confidence=0.95)],
            ripeness="half",
        )
    )

    with test_client.websocket_connect("/v1/infer/stream") as ws:
        for _ in range(3):
            ws.send_bytes(sample_image_bytes)
            msg = ws.receive_json()
            assert msg["type"] == "frame"
        ws.send_text("eos")
        summary = ws.receive_json()
        assert summary["type"] == "summary"

from __future__ import annotations

from tests.factories import build_raw_detection


def test_health_and_image_infer(test_client, install_pipeline, decode_image_to_frame, sample_image_bytes, fake_detector_factory) -> None:
    decode_image_to_frame()
    install_pipeline(
        detector=fake_detector_factory(
            detections=[build_raw_detection(bbox=(5, 5, 30, 30), class_id=2, confidence=0.8)],
            ripeness="red",
        )
    )

    health = test_client.get("/v1/health")
    assert health.status_code == 200
    assert health.json()["status"] == "ok"

    resp = test_client.post("/v1/infer/image", files={"file": ("x.jpg", sample_image_bytes, "image/jpeg")})
    assert resp.status_code == 200
    body = resp.json()
    assert body["result"]["frame_summary"]["red"] == 1

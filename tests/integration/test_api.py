from __future__ import annotations

import numpy as np
from fastapi.testclient import TestClient

from app.inference.adapters.base import DetectorAdapter, RawDetection
from app.inference.pipeline import InferencePipeline
from app.main import app


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
        return [RawDetection(bbox=(5, 5, 30, 30), class_id=2, confidence=0.8)]

    def ripeness_from_class_id(self, class_id: int) -> str:
        return 'red'


def test_health_and_image_infer(monkeypatch) -> None:
    from app.api.v1 import endpoints

    monkeypatch.setenv('LYCHEE_MODEL_CONFIG', 'configs/model.yaml.example')
    monkeypatch.setenv('LYCHEE_SERVICE_CONFIG', 'configs/service.yaml.example')
    monkeypatch.setattr(endpoints, '_decode_image_bytes', lambda _: np.zeros((64, 64, 3), dtype=np.uint8))

    with TestClient(app) as client:
        app.state.pipeline = InferencePipeline(FakeDetector(), model_version='1.0.0', schema_version='v1')

        health = client.get('/v1/health')
        assert health.status_code == 200
        assert health.json()['status'] == 'ok'

        resp = client.post('/v1/infer/image', files={'file': ('x.jpg', b'raw-bytes', 'image/jpeg')})
        assert resp.status_code == 200
        body = resp.json()
        assert body['result']['frame_summary']['red'] == 1

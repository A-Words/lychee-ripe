from __future__ import annotations

from collections.abc import Callable, Generator

import numpy as np
import pytest
from fastapi.testclient import TestClient

from app.inference.pipeline import InferencePipeline
from app.main import app
from app.paths import resolve_repo_path
from tests.factories import FakeDetector, build_frame, build_pipeline


@pytest.fixture
def config_env(monkeypatch: pytest.MonkeyPatch) -> dict[str, str]:
    model_config = str(resolve_repo_path("tooling/configs/model.yaml.example"))
    service_config = str(resolve_repo_path("tooling/configs/service.yaml.example"))
    monkeypatch.setenv("LYCHEE_MODEL_CONFIG", model_config)
    monkeypatch.setenv("LYCHEE_SERVICE_CONFIG", service_config)
    return {
        "model_config": model_config,
        "service_config": service_config,
    }


@pytest.fixture
def sample_image_bytes() -> bytes:
    return b"test-image-bytes"


@pytest.fixture
def fake_detector_factory() -> Callable[..., FakeDetector]:
    def factory(**kwargs) -> FakeDetector:
        return FakeDetector(**kwargs)

    return factory


@pytest.fixture
def decode_image_to_frame(monkeypatch: pytest.MonkeyPatch) -> Callable[[np.ndarray | None], np.ndarray]:
    from app.api.v1 import endpoints

    def factory(frame: np.ndarray | None = None) -> np.ndarray:
        resolved_frame = frame if frame is not None else build_frame()
        monkeypatch.setattr(endpoints, "_decode_image_bytes", lambda _: resolved_frame.copy())
        return resolved_frame

    return factory


@pytest.fixture
def test_client(config_env: dict[str, str]) -> Generator[TestClient, None, None]:
    with TestClient(app) as client:
        yield client


@pytest.fixture
def install_pipeline() -> Callable[..., InferencePipeline]:
    def factory(*, detector: FakeDetector | None = None, model_version: str = "1.0.0", schema_version: str = "v1") -> InferencePipeline:
        pipeline = build_pipeline(detector=detector, model_version=model_version, schema_version=schema_version)
        app.state.pipeline = pipeline
        return pipeline

    return factory

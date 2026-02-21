import app.settings as settings


def test_model_config_default_device_is_auto() -> None:
    cfg = settings.ModelConfig()
    assert cfg.device == "auto"


def test_resolve_torch_device_keeps_cpu_when_no_cuda(monkeypatch) -> None:
    monkeypatch.setattr(settings, "_torch_cuda_runtime", lambda: (False, 0))
    resolved, warning = settings.resolve_torch_device("cpu")
    assert resolved == "cpu"
    assert warning is None


def test_resolve_torch_device_falls_back_for_cuda_index(monkeypatch) -> None:
    monkeypatch.setattr(settings, "_torch_cuda_runtime", lambda: (False, 0))
    resolved, warning = settings.resolve_torch_device("0")
    assert resolved == "cpu"
    assert warning is not None
    assert "Falling back to CPU" in warning


def test_resolve_torch_device_supports_auto(monkeypatch) -> None:
    monkeypatch.setattr(settings, "_torch_cuda_runtime", lambda: (True, 1))
    resolved, warning = settings.resolve_torch_device("auto")
    assert resolved == "cuda:0"
    assert warning is None

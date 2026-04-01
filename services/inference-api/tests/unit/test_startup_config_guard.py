from pathlib import Path

import pytest

from app.main import _ensure_config_file


def test_ensure_config_file_raises_with_example_hint(tmp_path: Path) -> None:
    cfg = tmp_path / "model.yaml"
    example = tmp_path / "model.yaml.example"
    example.write_text("x: 1\n", encoding="utf-8")

    with pytest.raises(RuntimeError) as exc_info:
        _ensure_config_file(cfg, "LYCHEE_MODEL_CONFIG")

    msg = str(exc_info.value)
    assert "Missing config file" in msg
    assert "LYCHEE_MODEL_CONFIG" in msg
    assert "model.yaml.example" in msg


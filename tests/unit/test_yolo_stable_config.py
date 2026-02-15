from app.inference.adapters.yolo_stable import YoloStableAdapter
from app.settings import ModelConfig


def test_model_source_uses_explicit_path() -> None:
    cfg = ModelConfig(yolo_version="yolo26n", model_path="weights/custom.pt")
    adapter = YoloStableAdapter(cfg)
    assert adapter._resolve_model_source() == "weights/custom.pt"


def test_model_source_falls_back_to_yolo_version() -> None:
    cfg = ModelConfig(yolo_version="yolo26n", model_path="")
    adapter = YoloStableAdapter(cfg)
    assert adapter._resolve_model_source() == "yolo26n.pt"


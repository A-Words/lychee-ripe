from __future__ import annotations

from typing import Sequence

import numpy as np

from app.inference.adapters.base import DetectorAdapter, RawDetection
from app.settings import ModelConfig


class YoloStableAdapter(DetectorAdapter):
    name = "yolo_stable"

    def __init__(self, cfg: ModelConfig) -> None:
        self.cfg = cfg
        self.name = cfg.yolo_version
        self._model = None
        self._loaded = False
        self._class_map = {
            0: "green",
            1: "half",
            2: "red",
            3: "young",
        }

    @property
    def loaded(self) -> bool:
        return self._loaded

    def _resolve_model_source(self) -> str:
        configured_path = (self.cfg.model_path or "").strip()
        if configured_path:
            return configured_path

        version = self.cfg.yolo_version.strip()
        if not version:
            raise ValueError("yolo_version must be set when model_path is empty")
        if version.endswith(".pt"):
            return version
        return f"{version}.pt"

    def load(self) -> None:
        try:
            from ultralytics import YOLO
        except Exception as exc:  # pragma: no cover
            raise RuntimeError("Ultralytics is required for YoloStableAdapter") from exc

        self._model = YOLO(self._resolve_model_source())
        self._loaded = True

    def warmup(self) -> None:
        if not self.loaded:
            return
        dummy = np.zeros((640, 640, 3), dtype=np.uint8)
        _ = self.predict(dummy)

    def predict(self, frame: np.ndarray) -> Sequence[RawDetection]:
        if not self.loaded or self._model is None:
            raise RuntimeError("Model is not loaded")

        results = self._model.predict(
            source=frame,
            conf=self.cfg.conf_threshold,
            iou=self.cfg.nms_iou,
            device=self.cfg.device,
            verbose=False,
        )

        detections: list[RawDetection] = []
        for result in results:
            if result.boxes is None:
                continue
            for box in result.boxes:
                xyxy = box.xyxy[0].tolist()
                conf = float(box.conf[0].item())
                cls_id = int(box.cls[0].item())
                detections.append(
                    RawDetection(
                        bbox=(float(xyxy[0]), float(xyxy[1]), float(xyxy[2]), float(xyxy[3])),
                        class_id=cls_id,
                        confidence=max(0.0, min(1.0, conf)),
                    )
                )
        return detections

    def ripeness_from_class_id(self, class_id: int) -> str:
        if class_id not in self._class_map:
            return "green"
        return self._class_map[class_id]

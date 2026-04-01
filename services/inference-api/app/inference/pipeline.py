from __future__ import annotations

import time
from dataclasses import dataclass

import numpy as np

from app.inference.adapters.base import DetectorAdapter, RawDetection
from app.inference.aggregator import SessionAggregator
from app.inference.tracker import ByteTrackManager
from app.schemas.common import Detection, FrameResult, ModelMeta


def _sanitize_bbox(bbox: tuple[float, float, float, float], width: int, height: int) -> tuple[float, float, float, float]:
    x1, y1, x2, y2 = bbox
    x1 = max(0.0, min(float(width - 1), x1))
    y1 = max(0.0, min(float(height - 1), y1))
    x2 = max(0.0, min(float(width - 1), x2))
    y2 = max(0.0, min(float(height - 1), y2))
    if x2 < x1:
        x1, x2 = x2, x1
    if y2 < y1:
        y1, y2 = y2, y1
    return (x1, y1, x2, y2)


@dataclass
class StreamSession:
    tracker: ByteTrackManager
    aggregator: SessionAggregator
    frame_index: int = 0


class InferencePipeline:
    def __init__(self, detector: DetectorAdapter, model_version: str, schema_version: str) -> None:
        self.detector = detector
        self.model_version = model_version
        self.schema_version = schema_version

    def model_meta(self) -> ModelMeta:
        return ModelMeta(
            model_version=self.model_version,
            schema_version=self.schema_version,
            adapter=self.detector.name,
            loaded=self.detector.loaded,
        )

    def create_stream_session(self) -> StreamSession:
        return StreamSession(tracker=ByteTrackManager(), aggregator=SessionAggregator())

    def infer_image(self, frame: np.ndarray) -> tuple[FrameResult, float]:
        session = self.create_stream_session()
        start = time.perf_counter()
        result = self._infer_frame(frame, session, timestamp_ms=0, use_track=False)
        elapsed_ms = (time.perf_counter() - start) * 1000.0
        return result, elapsed_ms

    def infer_stream_frame(self, frame: np.ndarray, session: StreamSession, timestamp_ms: int) -> FrameResult:
        return self._infer_frame(frame, session, timestamp_ms=timestamp_ms, use_track=True)

    def _infer_frame(self, frame: np.ndarray, session: StreamSession, timestamp_ms: int, use_track: bool) -> FrameResult:
        if frame.ndim != 3:
            raise ValueError("Expected BGR frame with shape [H, W, C]")
        if not self.detector.loaded:
            raise RuntimeError("Detector is not loaded")

        raw_dets = list(self.detector.predict(frame))
        height, width = frame.shape[:2]
        if use_track:
            tracked = session.tracker.update(raw_dets)
        else:
            tracked = []

        track_map = {id(t.det): t.track_id for t in tracked}
        detections: list[Detection] = []
        ripeness_list: list[str] = []
        track_ids: list[int | None] = []

        for det in raw_dets:
            ripeness = self.detector.ripeness_from_class_id(det.class_id)
            sanitized_bbox = _sanitize_bbox(det.bbox, width, height)
            track_id = track_map.get(id(det)) if use_track else None
            detections.append(
                Detection(
                    bbox=sanitized_bbox,
                    ripeness=ripeness,
                    confidence=det.confidence,
                    track_id=track_id,
                )
            )
            ripeness_list.append(ripeness)
            track_ids.append(track_id)

        session.aggregator.update_session(ripeness_list, track_ids)
        frame_summary = session.aggregator.frame_summary(ripeness_list)

        result = FrameResult(
            frame_index=session.frame_index,
            timestamp_ms=timestamp_ms,
            detections=detections,
            frame_summary=frame_summary,
        )
        session.frame_index += 1
        return result

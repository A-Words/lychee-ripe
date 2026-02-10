from __future__ import annotations

from dataclasses import dataclass

from app.inference.adapters.base import RawDetection


def _iou(a: tuple[float, float, float, float], b: tuple[float, float, float, float]) -> float:
    ax1, ay1, ax2, ay2 = a
    bx1, by1, bx2, by2 = b
    inter_x1 = max(ax1, bx1)
    inter_y1 = max(ay1, by1)
    inter_x2 = min(ax2, bx2)
    inter_y2 = min(ay2, by2)
    iw = max(0.0, inter_x2 - inter_x1)
    ih = max(0.0, inter_y2 - inter_y1)
    inter = iw * ih
    if inter <= 0:
        return 0.0
    area_a = max(0.0, ax2 - ax1) * max(0.0, ay2 - ay1)
    area_b = max(0.0, bx2 - bx1) * max(0.0, by2 - by1)
    denom = area_a + area_b - inter
    if denom <= 0:
        return 0.0
    return inter / denom


@dataclass(slots=True)
class TrackedDetection:
    det: RawDetection
    track_id: int


class ByteTrackManager:
    """A lightweight track manager with ByteTrack-like matching behavior."""

    def __init__(self, iou_threshold: float = 0.3, max_missing: int = 20) -> None:
        self.iou_threshold = iou_threshold
        self.max_missing = max_missing
        self._tracks: dict[int, tuple[float, float, float, float]] = {}
        self._missing: dict[int, int] = {}
        self._next_id = 1

    def update(self, detections: list[RawDetection]) -> list[TrackedDetection]:
        outputs: list[TrackedDetection] = []
        assigned_tracks: set[int] = set()

        for det in detections:
            best_id = None
            best_iou = 0.0
            for track_id, bbox in self._tracks.items():
                if track_id in assigned_tracks:
                    continue
                iou = _iou(det.bbox, bbox)
                if iou > best_iou and iou >= self.iou_threshold:
                    best_iou = iou
                    best_id = track_id

            if best_id is None:
                track_id = self._next_id
                self._next_id += 1
            else:
                track_id = best_id

            self._tracks[track_id] = det.bbox
            self._missing[track_id] = 0
            assigned_tracks.add(track_id)
            outputs.append(TrackedDetection(det=det, track_id=track_id))

        stale = []
        for track_id in list(self._tracks):
            if track_id not in assigned_tracks:
                self._missing[track_id] = self._missing.get(track_id, 0) + 1
                if self._missing[track_id] > self.max_missing:
                    stale.append(track_id)

        for track_id in stale:
            self._tracks.pop(track_id, None)
            self._missing.pop(track_id, None)

        return outputs

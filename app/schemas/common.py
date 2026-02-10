from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field

RipenessLabel = Literal["green", "half", "red", "young"]
HarvestSuggestion = Literal["not_ready", "partially_ready", "ready", "overripe_risk"]


class Detection(BaseModel):
    bbox: tuple[float, float, float, float]
    class_name: Literal["lychee"] = "lychee"
    ripeness: RipenessLabel
    confidence: float = Field(ge=0.0, le=1.0)
    track_id: int | None = None


class FrameSummary(BaseModel):
    total: int = 0
    green: int = 0
    half: int = 0
    red: int = 0
    young: int = 0


class FrameResult(BaseModel):
    frame_index: int = Field(ge=0)
    timestamp_ms: int = Field(ge=0)
    detections: list[Detection]
    frame_summary: FrameSummary


class RipenessRatio(BaseModel):
    green: float = 0.0
    half: float = 0.0
    red: float = 0.0
    young: float = 0.0


class SessionSummary(BaseModel):
    total_detected: int = 0
    ripeness_ratio: RipenessRatio
    harvest_suggestion: HarvestSuggestion = "not_ready"


class ModelMeta(BaseModel):
    model_version: str
    schema_version: str
    adapter: str
    loaded: bool

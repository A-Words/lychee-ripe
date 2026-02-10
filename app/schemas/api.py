from __future__ import annotations

from pydantic import BaseModel

from app.schemas.common import FrameResult, ModelMeta, SessionSummary


class ImageInferResponse(BaseModel):
    model_version: str
    schema_version: str
    inference_ms: float
    result: FrameResult


class HealthResponse(BaseModel):
    status: str
    model: ModelMeta


class CurrentModelResponse(BaseModel):
    model_version: str
    schema_version: str
    adapter: str
    loaded: bool


class StreamFrameEnvelope(BaseModel):
    type: str = "frame"
    model_version: str
    schema_version: str
    result: FrameResult


class StreamSummaryEnvelope(BaseModel):
    type: str = "summary"
    model_version: str
    schema_version: str
    summary: SessionSummary

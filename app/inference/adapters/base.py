from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import Sequence

import numpy as np


@dataclass(slots=True)
class RawDetection:
    bbox: tuple[float, float, float, float]
    class_id: int
    confidence: float


class DetectorAdapter(ABC):
    name: str = "base"

    @abstractmethod
    def load(self) -> None:
        raise NotImplementedError

    @abstractmethod
    def warmup(self) -> None:
        raise NotImplementedError

    @abstractmethod
    def predict(self, frame: np.ndarray) -> Sequence[RawDetection]:
        raise NotImplementedError

    @abstractmethod
    def ripeness_from_class_id(self, class_id: int) -> str:
        raise NotImplementedError

    @property
    @abstractmethod
    def loaded(self) -> bool:
        raise NotImplementedError

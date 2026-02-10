from __future__ import annotations

from dataclasses import dataclass, field

from app.schemas.common import FrameSummary, RipenessRatio, SessionSummary


@dataclass
class SessionAggregator:
    seen_track_ids: set[int] = field(default_factory=set)
    total_unique: int = 0
    counts: dict[str, int] = field(default_factory=lambda: {"green": 0, "half": 0, "red": 0, "young": 0})

    def frame_summary(self, ripeness_list: list[str]) -> FrameSummary:
        summary = FrameSummary(total=len(ripeness_list))
        for r in ripeness_list:
            if r == "green":
                summary.green += 1
            elif r == "half":
                summary.half += 1
            elif r == "red":
                summary.red += 1
            elif r == "young":
                summary.young += 1
        return summary

    def update_session(self, ripeness_list: list[str], track_ids: list[int | None]) -> None:
        for ripeness, track_id in zip(ripeness_list, track_ids):
            if track_id is not None:
                if track_id in self.seen_track_ids:
                    continue
                self.seen_track_ids.add(track_id)
            self.total_unique += 1
            if ripeness in self.counts:
                self.counts[ripeness] += 1

    def build_summary(self) -> SessionSummary:
        total = self.total_unique
        if total == 0:
            ratios = RipenessRatio()
        else:
            ratios = RipenessRatio(
                green=self.counts["green"] / total,
                half=self.counts["half"] / total,
                red=self.counts["red"] / total,
                young=self.counts["young"] / total,
            )

        suggestion = "not_ready"
        if ratios.red >= 0.7 and ratios.young < 0.15:
            suggestion = "ready"
        elif (ratios.red + ratios.half) >= 0.4:
            suggestion = "partially_ready"

        return SessionSummary(
            total_detected=total,
            ripeness_ratio=ratios,
            harvest_suggestion=suggestion,
        )

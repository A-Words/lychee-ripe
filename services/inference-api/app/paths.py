from __future__ import annotations

from pathlib import Path


def resolve_repo_path(relative_path: str) -> Path:
    relative = Path(relative_path)
    candidates: list[Path] = [Path.cwd() / relative]

    source_path = Path(__file__).resolve()
    for parent in source_path.parents:
        candidates.append(parent / relative)

    ordered_candidates: list[Path] = []
    seen: set[Path] = set()
    for candidate in candidates:
        if candidate not in seen:
            seen.add(candidate)
            ordered_candidates.append(candidate)

    for candidate in ordered_candidates:
        if candidate.exists():
            return candidate

    for candidate in ordered_candidates:
        if candidate.parent.exists():
            return candidate

    return ordered_candidates[0]

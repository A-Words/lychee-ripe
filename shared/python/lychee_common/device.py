from __future__ import annotations

import re
from collections.abc import Callable

_CUDA_DEVICE_PATTERN = re.compile(r"^\d+(,\d+)*$")


def _torch_cuda_runtime() -> tuple[bool, int]:
    try:
        import torch
    except Exception:
        return False, 0

    try:
        return bool(torch.cuda.is_available()), int(torch.cuda.device_count())
    except Exception:
        return False, 0


def _requests_cuda(device: str) -> bool:
    normalized = device.strip().lower()
    if not normalized or normalized in {"cpu", "mps"}:
        return False
    if normalized.startswith("cuda"):
        return True
    return bool(_CUDA_DEVICE_PATTERN.fullmatch(normalized))


def resolve_torch_device(
    device: str,
    cuda_runtime: Callable[[], tuple[bool, int]] | None = None,
) -> tuple[str, str | None]:
    requested = (device or "").strip() or "cpu"
    runtime = cuda_runtime or _torch_cuda_runtime
    cuda_available, cuda_device_count = runtime()

    if requested.lower() == "auto":
        if cuda_available and cuda_device_count > 0:
            return "cuda:0", None
        return (
            "cpu",
            (
                "Requested device='auto' but CUDA is unavailable "
                f"(torch.cuda.is_available()={cuda_available}, "
                f"torch.cuda.device_count()={cuda_device_count}). Falling back to CPU."
            ),
        )

    if _requests_cuda(requested) and (not cuda_available or cuda_device_count <= 0):
        return (
            "cpu",
            (
                f"Requested device='{requested}' but CUDA is unavailable "
                f"(torch.cuda.is_available()={cuda_available}, "
                f"torch.cuda.device_count()={cuda_device_count}). Falling back to CPU."
            ),
        )

    return requested, None

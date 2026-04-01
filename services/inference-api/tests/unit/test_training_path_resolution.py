from __future__ import annotations

import importlib
import os
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[4]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

train_module = importlib.import_module("mlops.training.train")
eval_module = importlib.import_module("mlops.training.eval")


def test_train_resolve_output_path_stays_in_repo_for_workspace_relative_path() -> None:
    resolved = train_module.resolve_output_path("mlops/artifacts/models")
    assert resolved == (REPO_ROOT / "mlops" / "artifacts" / "models").resolve()


def test_eval_resolve_output_path_stays_in_repo_for_workspace_relative_path() -> None:
    resolved = eval_module.resolve_output_path("mlops/artifacts/metrics/lychee_v1-eval_metrics.json")
    assert resolved == (REPO_ROOT / "mlops" / "artifacts" / "metrics" / "lychee_v1-eval_metrics.json").resolve()


def test_train_resolve_output_path_keeps_explicit_relative_path_inside_repo() -> None:
    prev_cwd = Path.cwd()
    try:
        os.chdir(REPO_ROOT / "mlops" / "training")
        resolved = train_module.resolve_output_path("../artifacts/models")
    finally:
        os.chdir(prev_cwd)

    assert resolved == (REPO_ROOT / "mlops" / "artifacts" / "models").resolve()


def test_train_resolve_input_path_prefers_existing_repo_root_dataset() -> None:
    resolved = train_module.resolve_input_path("mlops/data/lichi/data.yaml")
    assert resolved == (REPO_ROOT / "mlops" / "data" / "lichi" / "data.yaml").resolve()

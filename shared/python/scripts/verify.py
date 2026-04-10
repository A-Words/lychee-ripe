from __future__ import annotations

import importlib
import sys
import tomllib
from pathlib import Path

workspace_root = Path(__file__).resolve().parents[1]
pyproject_path = workspace_root / "pyproject.toml"
pyproject = tomllib.loads(pyproject_path.read_text(encoding="utf-8"))
project = pyproject.get("project", {})

if project.get("name") != "lychee-common":
    raise SystemExit("shared/python project.name must remain 'lychee-common'")

sys.path.insert(0, str(workspace_root))

importlib.import_module("lychee_common")
importlib.import_module("lychee_common.device")

print("shared/python verify passed")

from __future__ import annotations

from pathlib import Path

import yaml
from app.paths import resolve_repo_path

OPENAPI_PATH = resolve_repo_path("shared/contracts/schemas/openapi.yaml")
PRD_PATH = resolve_repo_path("docs/prd.md")


def _load_openapi() -> dict:
    return yaml.safe_load(OPENAPI_PATH.read_text(encoding="utf-8"))


def _collect_refs(node: object, out: list[str]) -> None:
    if isinstance(node, dict):
        ref = node.get("$ref")
        if isinstance(ref, str):
            out.append(ref)
        for value in node.values():
            _collect_refs(value, out)
        return
    if isinstance(node, list):
        for item in node:
            _collect_refs(item, out)


def test_openapi_yaml_is_parsable() -> None:
    doc = _load_openapi()
    assert isinstance(doc, dict)
    assert doc.get("openapi") == "3.1.0"


def test_openapi_refs_have_no_broken_links() -> None:
    doc = _load_openapi()
    refs: list[str] = []
    _collect_refs(doc, refs)

    components = doc.get("components", {})
    schemas = components.get("schemas", {})
    parameters = components.get("parameters", {})

    missing: list[str] = []
    for ref in refs:
        if ref.startswith("#/components/schemas/"):
            name = ref.split("/")[-1]
            if name not in schemas:
                missing.append(ref)
        elif ref.startswith("#/components/parameters/"):
            name = ref.split("/")[-1]
            if name not in parameters:
                missing.append(ref)

    assert not missing


def test_new_paths_exist_and_operation_ids_are_unique() -> None:
    doc = _load_openapi()
    paths = doc["paths"]

    assert "/v1/batches" in paths
    assert "/v1/batches/{batch_id}" in paths
    assert "/v1/trace/{trace_code}" in paths
    assert "/v1/dashboard/overview" in paths
    assert "/v1/batches/reconcile" in paths

    operation_ids: list[str] = []
    for methods in paths.values():
        for op in methods.values():
            if isinstance(op, dict) and "operationId" in op:
                operation_ids.append(op["operationId"])
    assert len(operation_ids) == len(set(operation_ids))


def test_create_batch_response_codes_and_security() -> None:
    doc = _load_openapi()
    operation = doc["paths"]["/v1/batches"]["post"]
    responses = operation["responses"]

    assert {"201", "202", "409"}.issubset(set(responses.keys()))
    assert operation["security"] == [{"CookieAuth": []}, {"BearerAuth": []}]


def test_trace_endpoint_is_public() -> None:
    doc = _load_openapi()
    operation = doc["paths"]["/v1/trace/{trace_code}"]["get"]
    assert operation["security"] == []


def test_write_endpoints_declare_session_or_bearer_auth() -> None:
    doc = _load_openapi()
    secure_ops = [
        doc["paths"]["/v1/batches"]["post"],
        doc["paths"]["/v1/batches/{batch_id}"]["get"],
        doc["paths"]["/v1/dashboard/overview"]["get"],
        doc["paths"]["/v1/batches/reconcile"]["post"],
    ]
    for op in secure_ops:
        assert op["security"] == [{"CookieAuth": []}, {"BearerAuth": []}]


def test_batch_summary_contains_unripe_fields() -> None:
    doc = _load_openapi()
    batch_summary = doc["components"]["schemas"]["BatchSummary"]
    properties = batch_summary["properties"]

    assert "unripe_count" in properties
    assert "unripe_ratio" in properties
    assert "unripe_handling" in properties


def test_unripe_handling_enum_is_frozen() -> None:
    doc = _load_openapi()
    enum_values = doc["components"]["schemas"]["UnripeHandling"]["enum"]
    assert enum_values == ["sorted_out"]


def test_verify_status_enum_values() -> None:
    doc = _load_openapi()
    enum_values = doc["components"]["schemas"]["VerifyStatus"]["enum"]
    assert enum_values == ["pass", "fail", "pending", "recorded"]


def test_prd_field_names_match_contract() -> None:
    doc = _load_openapi()
    prd_text = PRD_PATH.read_text(encoding="utf-8")

    required_tokens = [
        "confirm_unripe",
        "summary.unripe_count",
        "summary.unripe_ratio",
        "summary.unripe_handling",
        "verify_status",
    ]
    for token in required_tokens:
        assert token in prd_text

    req_props = doc["components"]["schemas"]["BatchCreateRequest"]["properties"]
    summary_props = doc["components"]["schemas"]["BatchSummary"]["properties"]
    verify_enum = doc["components"]["schemas"]["VerifyStatus"]["enum"]

    assert "confirm_unripe" in req_props
    assert "unripe_count" in summary_props
    assert "unripe_ratio" in summary_props
    assert "unripe_handling" in summary_props
    assert verify_enum == ["pass", "fail", "pending", "recorded"]

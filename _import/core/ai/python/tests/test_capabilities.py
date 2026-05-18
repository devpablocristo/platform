import json
from pathlib import Path
from typing import Any

import pytest
from jsonschema import Draft202012Validator
from jsonschema.exceptions import ValidationError as JSONSchemaValidationError

from runtime import CapabilityManifest, ToolDeclaration, validate_capability_manifest


CONTRACT_ROOT = Path(__file__).resolve().parents[2] / "contracts" / "capabilities" / "v1"
EXAMPLES_ROOT = CONTRACT_ROOT / "examples"


def _load_example(name: str) -> dict[str, Any]:
    return json.loads((EXAMPLES_ROOT / name).read_text())


def _schema_validator() -> Draft202012Validator:
    schema = json.loads((CONTRACT_ROOT / "capability-manifest.schema.json").read_text())
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema)


@pytest.mark.parametrize("name", ["valid_read_only.json", "valid_write_governed.json"])
def test_capability_manifest_schema_accepts_valid_examples(name: str) -> None:
    _schema_validator().validate(_load_example(name))


@pytest.mark.parametrize(
    "name",
    [
        "invalid_duplicate_tool.json",
        "invalid_write_missing_action_type.json",
        "invalid_read_side_effect.json",
        "invalid_invalid_enum.json",
    ],
)
def test_capability_manifest_schema_rejects_invalid_examples(name: str) -> None:
    with pytest.raises(JSONSchemaValidationError):
        _schema_validator().validate(_load_example(name))


@pytest.mark.parametrize("name", ["valid_read_only.json", "valid_write_governed.json"])
def test_capability_manifest_models_accept_valid_examples(name: str) -> None:
    manifest = validate_capability_manifest(_load_example(name))

    assert isinstance(manifest, CapabilityManifest)
    assert manifest.schema_version == "capability_manifest.v1"
    assert manifest.tools


@pytest.mark.parametrize(
    "name",
    [
        "invalid_duplicate_tool.json",
        "invalid_write_missing_action_type.json",
        "invalid_read_side_effect.json",
        "invalid_invalid_enum.json",
        "invalid_secret_config.json",
    ],
)
def test_capability_manifest_models_reject_invalid_examples(name: str) -> None:
    with pytest.raises((ValueError, TypeError)):
        validate_capability_manifest(_load_example(name))


def test_capability_tool_to_tool_declaration_preserves_llm_surface() -> None:
    manifest = validate_capability_manifest(_load_example("valid_write_governed.json"))

    tool = manifest.tools[0].to_tool_declaration()

    assert isinstance(tool, ToolDeclaration)
    assert tool.name == "pymes.sales.create"
    assert tool.description == manifest.tools[0].description
    assert tool.parameters["type"] == "object"
    assert manifest.tools_for_llm() == [tool]

"""Capability manifest contracts for ecosystem AI."""

from __future__ import annotations

import re
from collections.abc import Mapping
from typing import Any, Literal, Self

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from runtime.domain.models import ToolDeclaration

CAPABILITY_MANIFEST_SCHEMA_VERSION = "capability_manifest.v1"

TenantScope = Literal["global", "org", "project"]
CapabilityMode = Literal["read", "write"]
RiskClass = Literal["low", "medium", "high", "critical"]

_SLUG_PATTERN = re.compile(r"^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$")
_TOOL_PATTERN = re.compile(r"^[a-z][a-z0-9_-]*(?:\.[a-z][a-z0-9_-]*)+$")
_SEMVER_PATTERN = re.compile(r"^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$")
_FORBIDDEN_CONFIG_KEY_TOKENS = (
    "apikey",
    "basicauth",
    "baseurl",
    "credential",
    "dsn",
    "endpoint",
    "host",
    "password",
    "secret",
    "token",
    "url",
)


class CapabilityAgentDescriptor(BaseModel):
    """Light routing descriptor exposed by a capability manifest."""

    model_config = ConfigDict(extra="forbid")

    name: str
    description: str

    @field_validator("name")
    @classmethod
    def _validate_name(cls, value: str) -> str:
        return _validate_slug("agent.name", value)

    @field_validator("description")
    @classmethod
    def _validate_description(cls, value: str) -> str:
        return _validate_required_text("agent.description", value)


class CapabilityAuthz(BaseModel):
    """Tenant-local role and module requirements."""

    model_config = ConfigDict(extra="forbid")

    required_roles: list[str] = Field(default_factory=list)
    required_modules: list[str] = Field(default_factory=list)

    @field_validator("required_roles", "required_modules")
    @classmethod
    def _validate_string_list(cls, value: list[str]) -> list[str]:
        return _validate_string_list(value)


class CapabilityExecutor(BaseModel):
    """Logical execution reference. This is not a URL."""

    model_config = ConfigDict(extra="forbid")

    executor_ref: str

    @field_validator("executor_ref")
    @classmethod
    def _validate_executor_ref(cls, value: str) -> str:
        return _validate_required_text("executor_ref", value)


class CapabilityGovernance(BaseModel):
    """Nexus Governance metadata for a capability tool."""

    model_config = ConfigDict(extra="forbid")

    requires_approval: bool
    action_type: str | None = None
    target_system: str | None = None

    @field_validator("action_type", "target_system")
    @classmethod
    def _validate_optional_text(cls, value: str | None) -> str | None:
        if value is None:
            return None
        return _validate_required_text("governance field", value)


class CapabilityTool(BaseModel):
    """Tool contract exposed by a product capability."""

    model_config = ConfigDict(extra="forbid")

    name: str
    description: str
    mode: CapabilityMode
    side_effect: bool
    risk_class: RiskClass
    input_schema: dict[str, Any]
    output_schema: dict[str, Any] | None = None
    required_roles: list[str] = Field(default_factory=list)
    required_modules: list[str] = Field(default_factory=list)
    evidence_fields: list[str] = Field(default_factory=list)
    executor_ref: str
    governance: CapabilityGovernance | None = None

    @field_validator("name")
    @classmethod
    def _validate_tool_name(cls, value: str) -> str:
        value = value.strip()
        if not _TOOL_PATTERN.fullmatch(value):
            raise ValueError("tool.name must use dot notation")
        return value

    @field_validator("description", "executor_ref")
    @classmethod
    def _validate_text(cls, value: str) -> str:
        return _validate_required_text("tool field", value)

    @field_validator("input_schema", "output_schema")
    @classmethod
    def _validate_schema_object(cls, value: dict[str, Any] | None) -> dict[str, Any] | None:
        if value is None:
            return None
        if value.get("type") != "object":
            raise ValueError("JSON Schema must be an object with type=object")
        return value

    @field_validator("required_roles", "required_modules", "evidence_fields")
    @classmethod
    def _validate_lists(cls, value: list[str]) -> list[str]:
        return _validate_string_list(value)

    @model_validator(mode="after")
    def _validate_mode_rules(self) -> Self:
        if self.mode == "read":
            if self.side_effect:
                raise ValueError("mode=read requires side_effect=false")
            if self.governance is not None and self.governance.requires_approval:
                raise ValueError("read tools must not require review")
        if self.mode == "write":
            if not self.side_effect:
                raise ValueError("mode=write requires side_effect=true")
            if not self.evidence_fields:
                raise ValueError("write tools require evidence_fields")
            if self.governance is None:
                raise ValueError("write tools require governance")
            if not self.governance.requires_approval:
                raise ValueError("write tools require governance.requires_approval=true")
            if not self.governance.action_type:
                raise ValueError("write tools require governance.action_type")
        return self

    def to_tool_declaration(self) -> ToolDeclaration:
        """Convert this capability tool into the runtime LLM tool declaration."""

        return ToolDeclaration(
            name=self.name,
            description=self.description,
            parameters=dict(self.input_schema),
        )

    def authz(self) -> CapabilityAuthz:
        return CapabilityAuthz(required_roles=list(self.required_roles), required_modules=list(self.required_modules))

    def executor(self) -> CapabilityExecutor:
        return CapabilityExecutor(executor_ref=self.executor_ref)


class CapabilityManifest(BaseModel):
    """Versioned product capability package."""

    model_config = ConfigDict(extra="forbid")

    schema_version: Literal["capability_manifest.v1"]
    id: str
    product: str
    version: str
    tenant_scope: TenantScope
    name: str
    description: str
    agents: list[CapabilityAgentDescriptor]
    tools: list[CapabilityTool]

    @field_validator("id", "product")
    @classmethod
    def _validate_slug_fields(cls, value: str) -> str:
        return _validate_slug("manifest field", value)

    @field_validator("version")
    @classmethod
    def _validate_version(cls, value: str) -> str:
        value = value.strip()
        if not _SEMVER_PATTERN.fullmatch(value):
            raise ValueError("version must be semver without leading v")
        return value

    @field_validator("name", "description")
    @classmethod
    def _validate_text_fields(cls, value: str) -> str:
        return _validate_required_text("manifest field", value)

    @field_validator("agents", "tools")
    @classmethod
    def _validate_non_empty(cls, value: list[Any]) -> list[Any]:
        if not value:
            raise ValueError("list must contain at least one item")
        return value

    @model_validator(mode="after")
    def _validate_hard_rules(self) -> Self:
        agent_names: set[str] = set()
        for agent in self.agents:
            if agent.name in agent_names:
                raise ValueError(f"duplicate agent name {agent.name!r}")
            agent_names.add(agent.name)

        tool_names: set[str] = set()
        for tool in self.tools:
            if tool.name in tool_names:
                raise ValueError(f"duplicate tool name {tool.name!r}")
            tool_names.add(tool.name)

        _reject_forbidden_config(self.model_dump(mode="json", exclude_none=True), "manifest")
        return self

    def tools_for_llm(self) -> list[ToolDeclaration]:
        return [tool.to_tool_declaration() for tool in self.tools]


def validate_capability_manifest(payload: Mapping[str, Any] | CapabilityManifest) -> CapabilityManifest:
    """Validate and return a capability manifest."""

    if isinstance(payload, CapabilityManifest):
        manifest = payload
        _reject_forbidden_config(manifest.model_dump(mode="json", exclude_none=True), "manifest")
        return manifest
    return CapabilityManifest.model_validate(dict(payload))


def _validate_slug(field: str, value: str) -> str:
    value = value.strip()
    if not _SLUG_PATTERN.fullmatch(value):
        raise ValueError(f"{field} must be a stable slug")
    return value


def _validate_required_text(field: str, value: str) -> str:
    value = value.strip()
    if not value:
        raise ValueError(f"{field} is required")
    return value


def _validate_string_list(values: list[str]) -> list[str]:
    for value in values:
        if not value.strip():
            raise ValueError("list values cannot be empty")
    return values


def _reject_forbidden_config(value: Any, path: str) -> None:
    if isinstance(value, Mapping):
        for key, child in value.items():
            if _is_forbidden_config_key(str(key)):
                raise ValueError(f"capability manifest contains forbidden configuration key {key!r} at {path}")
            _reject_forbidden_config(child, f"{path}.{key}")
        return
    if isinstance(value, list):
        for index, child in enumerate(value):
            _reject_forbidden_config(child, f"{path}[{index}]")
        return
    if isinstance(value, str) and _looks_like_runtime_address(value):
        raise ValueError(f"capability manifest contains runtime address at {path}")


def _is_forbidden_config_key(key: str) -> bool:
    normalized = key.strip().lower().replace("_", "").replace("-", "").replace(".", "").replace(" ", "")
    return any(token in normalized for token in _FORBIDDEN_CONFIG_KEY_TOKENS)


def _looks_like_runtime_address(value: str) -> bool:
    return "://" in value.strip().lower()


__all__ = [
    "CAPABILITY_MANIFEST_SCHEMA_VERSION",
    "CapabilityAgentDescriptor",
    "CapabilityAuthz",
    "CapabilityExecutor",
    "CapabilityGovernance",
    "CapabilityManifest",
    "CapabilityMode",
    "CapabilityTool",
    "RiskClass",
    "TenantScope",
    "validate_capability_manifest",
]

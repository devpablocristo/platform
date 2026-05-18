"""Request and authentication context helpers."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(slots=True)
class AuthContext:
    tenant_id: str
    actor: str
    role: str
    scopes: list[str]
    mode: str
    authorization: str | None = None
    api_key: str | None = None
    api_actor: str | None = None
    api_role: str | None = None
    api_scopes: str | None = None

    @property
    def org_id(self) -> str:
        return self.tenant_id

"""Tipos de dominio para memory — agnósticos de DB y framework."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from typing import Any


@dataclass
class Turn:
    """Un turno de conversación (mensaje del usuario, respuesta del asistente, tool call, etc.)."""

    role: str
    content: str
    tool_calls: list[dict[str, Any]] = field(default_factory=list)
    tool_call_id: str | None = None
    timestamp: datetime | None = None
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class Conversation:
    """Una conversación completa con su historial de turnos."""

    id: str
    org_id: str
    turns: list[Turn] = field(default_factory=list)
    mode: str = "internal"
    channel: str = "console"
    title: str = ""
    user_id: str | None = None
    contact_id: str | None = None
    contact_name: str = ""
    tokens_input: int = 0
    tokens_output: int = 0
    tool_calls_count: int = 0
    metadata: dict[str, Any] = field(default_factory=dict)
    created_at: datetime | None = None
    updated_at: datetime | None = None

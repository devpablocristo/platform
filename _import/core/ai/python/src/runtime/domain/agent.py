"""Primitivas de sub-agentes: registro, descriptor y agente concreto.

Un SubAgent encapsula todo lo necesario para ejecutar orchestrate():
tools, handlers, system prompt y límites. El AgentRegistry es un
diccionario tipado que permite registrar y consultar sub-agentes.

Estas primitivas son agnósticas al producto — cualquier proyecto que
use el runtime puede componer sub-agentes sin depender de pymes.
"""

from __future__ import annotations

import threading
from dataclasses import dataclass, field
from typing import Any

from runtime.domain.models import ToolDeclaration
from runtime.services.orchestrator import OrchestratorLimits, ToolHandler


@dataclass(frozen=True)
class SubAgentDescriptor:
    """Descriptor inmutable para el router. No lleva handlers ni tools pesados."""

    name: str
    description: str


@dataclass
class SubAgent:
    """Agente concreto: descriptor + tools + prompt + límites."""

    descriptor: SubAgentDescriptor
    tools: list[ToolDeclaration]
    tool_handlers: dict[str, ToolHandler]
    system_prompt: str
    limits: OrchestratorLimits | None = None
    metadata: dict[str, Any] = field(default_factory=dict)

    @property
    def name(self) -> str:
        return self.descriptor.name

    @property
    def description(self) -> str:
        return self.descriptor.description


class AgentRegistryError(Exception):
    """Error al registrar o consultar un sub-agente."""


class AgentRegistry:
    """Registro thread-safe de sub-agentes.

    Uso típico:
        registry = AgentRegistry()
        registry.register(clientes_agent)
        registry.register(ventas_agent)

        agent = registry.get("ventas")
        descriptors = registry.descriptors()  # para el router
    """

    def __init__(self) -> None:
        self._agents: dict[str, SubAgent] = {}
        self._lock = threading.Lock()

    def register(self, agent: SubAgent) -> None:
        """Registra un sub-agente. Lanza si el nombre ya existe."""
        with self._lock:
            if agent.name in self._agents:
                raise AgentRegistryError(f"agent '{agent.name}' already registered")
            self._agents[agent.name] = agent

    def get(self, name: str) -> SubAgent | None:
        """Devuelve el sub-agente por nombre, o None."""
        return self._agents.get(name)

    def descriptors(self) -> list[SubAgentDescriptor]:
        """Devuelve los descriptors de todos los agentes registrados (para el router)."""
        return [agent.descriptor for agent in self._agents.values()]

    def names(self) -> list[str]:
        """Nombres registrados."""
        return list(self._agents.keys())

    def __len__(self) -> int:
        return len(self._agents)

    def __contains__(self, name: str) -> bool:
        return name in self._agents

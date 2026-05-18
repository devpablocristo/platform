"""Tests para SubAgent, AgentRegistry y AgentRouter."""

import sys
import os

# Exponer src para importar el runtime real del módulo.
src = os.path.join(os.path.dirname(__file__), '..', 'src')
sys.path.insert(0, src)

import pytest
from runtime.domain.models import ToolDeclaration
from runtime.domain.agent import AgentRegistry, AgentRegistryError, SubAgent, SubAgentDescriptor
from runtime.services.orchestrator import OrchestratorLimits
from runtime.services.agent_router import _extract_agent_name


def _make_agent(name: str, description: str = "test") -> SubAgent:
    return SubAgent(
        descriptor=SubAgentDescriptor(name=name, description=description),
        tools=[ToolDeclaration(name=f"{name}_tool", description="test tool")],
        tool_handlers={f"{name}_tool": lambda: {}},
        system_prompt=f"System prompt for {name}",
        limits=OrchestratorLimits(max_tool_calls=3),
    )


def test_register_and_get():
    reg = AgentRegistry()
    agent = _make_agent("ventas", "Ventas y presupuestos")
    reg.register(agent)
    assert reg.get("ventas") is agent
    assert reg.get("unknown") is None
    assert len(reg) == 1
    assert "ventas" in reg


def test_register_duplicate_raises():
    reg = AgentRegistry()
    reg.register(_make_agent("ventas"))
    with pytest.raises(AgentRegistryError):
        reg.register(_make_agent("ventas"))


def test_descriptors():
    reg = AgentRegistry()
    reg.register(_make_agent("clientes", "Gestion de clientes"))
    reg.register(_make_agent("ventas", "Ventas y presupuestos"))
    descs = reg.descriptors()
    assert len(descs) == 2
    names = {d.name for d in descs}
    assert names == {"clientes", "ventas"}


def test_names():
    reg = AgentRegistry()
    reg.register(_make_agent("a"))
    reg.register(_make_agent("b"))
    assert set(reg.names()) == {"a", "b"}


def test_sub_agent_properties():
    agent = _make_agent("ventas", "desc")
    assert agent.name == "ventas"
    assert agent.description == "desc"


def test_extract_agent_name_exact():
    valid = {"clientes", "ventas", "compras", "general"}
    assert _extract_agent_name("ventas", valid, "general") == "ventas"
    assert _extract_agent_name("  clientes  ", valid, "general") == "clientes"
    assert _extract_agent_name("VENTAS", valid, "general") == "ventas"


def test_extract_agent_name_partial():
    valid = {"clientes", "ventas", "compras", "general"}
    assert _extract_agent_name("I think ventas is the best", valid, "general") == "ventas"


def test_extract_agent_name_fallback():
    valid = {"clientes", "ventas"}
    assert _extract_agent_name("unknown_agent", valid, "general") == "general"
    assert _extract_agent_name("", valid, "general") == "general"


def test_empty_registry():
    reg = AgentRegistry()
    assert len(reg) == 0
    assert reg.names() == []
    assert reg.descriptors() == []
    assert "anything" not in reg

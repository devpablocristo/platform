"""Tests para runtime.memory — tipos, context window y protocolo."""

import sys
import os
from datetime import datetime, UTC

# Hack: registramos runtime como package vacío para evitar que su __init__.py
# cargue todas las dependencias pesadas (httpserver, etc.)
src = os.path.join(os.path.dirname(__file__), '..', 'src')
sys.path.insert(0, src)

import types as _t
_runtime_pkg = _t.ModuleType('runtime')
_runtime_pkg.__path__ = [os.path.join(src, 'runtime')]
_runtime_pkg.__package__ = 'runtime'
sys.modules['runtime'] = _runtime_pkg

# Ahora sí, importamos solo memory (que no depende de auth, api, etc.)
from runtime.memory.types import Conversation, Turn
from runtime.memory.context import build_context_window, turn_to_message


def _make_conversation(n_turns: int = 5) -> Conversation:
    turns = []
    for i in range(n_turns):
        role = "user" if i % 2 == 0 else "assistant"
        turns.append(Turn(role=role, content=f"message {i}", timestamp=datetime.now(UTC)))
    return Conversation(
        id="conv-1",
        org_id="org-1",
        turns=turns,
        mode="internal",
        title="Test conversation",
    )


def test_conversation_creation():
    conv = _make_conversation(3)
    assert conv.id == "conv-1"
    assert conv.org_id == "org-1"
    assert len(conv.turns) == 3
    assert conv.turns[0].role == "user"
    assert conv.turns[1].role == "assistant"


def test_turn_to_message():
    turn = Turn(role="user", content="hola", tool_calls=[{"name": "get_stock", "arguments": {"sku": "ABC"}}])
    msg = turn_to_message(turn)
    assert msg.role == "user"
    assert msg.content == "hola"
    assert len(msg.tool_calls) == 1
    assert msg.tool_calls[0].name == "get_stock"


def test_build_context_window_basic():
    conv = _make_conversation(10)
    messages = build_context_window(conv, max_turns=4)
    assert len(messages) == 4
    assert messages[0].content == "message 6"
    assert messages[-1].content == "message 9"


def test_build_context_window_with_system_prompt():
    conv = _make_conversation(3)
    messages = build_context_window(conv, system_prompt="Sos un asistente.")
    assert len(messages) == 4
    assert messages[0].role == "system"
    assert messages[0].content == "Sos un asistente."
    assert messages[1].role == "user"


def test_build_context_window_max_tokens():
    conv = _make_conversation(20)
    messages = build_context_window(conv, max_tokens=20)
    assert len(messages) < 20
    assert all(m.role in ("user", "assistant") for m in messages)


def test_empty_conversation():
    conv = Conversation(id="empty", org_id="org-1")
    messages = build_context_window(conv)
    assert messages == []


def test_empty_conversation_with_system_prompt():
    conv = Conversation(id="empty", org_id="org-1")
    messages = build_context_window(conv, system_prompt="Hola")
    assert len(messages) == 1
    assert messages[0].role == "system"

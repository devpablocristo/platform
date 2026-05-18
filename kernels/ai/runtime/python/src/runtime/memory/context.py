"""Construcción de ventana de contexto para el LLM a partir de una conversación."""

from __future__ import annotations

from runtime.domain.models import Message

from .types import Conversation, Turn


def turn_to_message(turn: Turn) -> Message:
    """Convierte un Turn de memory a un Message del dominio LLM."""
    tool_calls_data = []
    for tc in turn.tool_calls:
        from runtime.domain.models import ToolCall
        tool_calls_data.append(ToolCall(name=tc.get("name", ""), arguments=tc.get("arguments", {})))
    return Message(
        role=turn.role,
        content=turn.content,
        tool_calls=tool_calls_data if tool_calls_data else [],
        tool_call_id=turn.tool_call_id,
    )


def build_context_window(
    conversation: Conversation,
    *,
    max_turns: int = 40,
    max_tokens: int | None = None,
    system_prompt: str | None = None,
) -> list[Message]:
    """Arma la lista de Messages para enviar al LLM.

    - Toma los últimos `max_turns` turnos de la conversación.
    - Si `max_tokens` está definido, trunca desde el inicio hasta entrar en budget
      (estimación: 4 chars ≈ 1 token).
    - Antepone el system prompt si se provee.
    """
    turns = conversation.turns[-max_turns:] if max_turns else conversation.turns

    if max_tokens is not None:
        chars_budget = max_tokens * 4
        selected: list[Turn] = []
        total_chars = 0
        for turn in reversed(turns):
            turn_chars = len(turn.content) + sum(len(str(tc)) for tc in turn.tool_calls)
            if total_chars + turn_chars > chars_budget and selected:
                break
            selected.append(turn)
            total_chars += turn_chars
        turns = list(reversed(selected))

    messages = [turn_to_message(t) for t in turns]

    if system_prompt:
        messages.insert(0, Message(role="system", content=system_prompt))

    return messages

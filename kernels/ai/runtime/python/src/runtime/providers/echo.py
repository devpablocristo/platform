"""Development provider implementations."""

from __future__ import annotations

from collections.abc import AsyncIterator

from runtime.domain.models import ChatChunk, Message, ToolDeclaration


class EchoProvider:
    """Provider de desarrollo que replica el ultimo mensaje de usuario."""

    async def chat(
        self,
        messages: list[Message],
        *,
        tools: list[ToolDeclaration] | None = None,
        temperature: float | None = None,
        max_tokens: int | None = None,
    ) -> AsyncIterator[ChatChunk]:
        del tools, temperature, max_tokens
        last_user = next((message.content for message in reversed(messages) if message.role == "user"), "")
        if not last_user:
            yield ChatChunk(type="text", text="[echo] no user message received")
        else:
            yield ChatChunk(type="text", text=f"[echo] {last_user}")

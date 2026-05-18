"""Ollama LLM provider — usa la API compatible con OpenAI de Ollama."""

from __future__ import annotations

import json
from collections.abc import AsyncIterator
from typing import Any

import httpx

from runtime.logging import get_logger
from runtime.types import ChatChunk, Message, ToolCall, ToolDeclaration

logger = get_logger(__name__)


class OllamaProvider:
    """Provider que conecta a Ollama via su endpoint /api/chat."""

    def __init__(
        self,
        base_url: str = "http://localhost:11434",
        model: str = "llama3.2:1b",
        timeout: float = 120.0,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.model = model
        self.timeout = timeout

    async def chat(
        self,
        messages: list[Message],
        *,
        tools: list[ToolDeclaration] | None = None,
        temperature: float | None = None,
        max_tokens: int | None = None,
    ) -> AsyncIterator[ChatChunk]:
        payload: dict[str, Any] = {
            "model": self.model,
            "messages": [self._to_ollama_message(m) for m in messages],
            "stream": False,
        }
        options: dict[str, Any] = {}
        if temperature is not None:
            options["temperature"] = temperature
        if max_tokens is not None:
            options["num_predict"] = max_tokens
        if options:
            payload["options"] = options
        if tools:
            payload["tools"] = [self._to_ollama_tool(t) for t in tools]

        try:
            async with httpx.AsyncClient(timeout=self.timeout) as client:
                resp = await client.post(f"{self.base_url}/api/chat", json=payload)
                resp.raise_for_status()
                data = resp.json()
        except Exception as exc:
            logger.exception("ollama_chat_failed", model=self.model, error=str(exc))
            raise RuntimeError(f"ollama request failed: {exc}") from exc

        msg = data.get("message", {})
        content = msg.get("content", "")
        tool_calls = msg.get("tool_calls", [])

        if tool_calls:
            for tc in tool_calls:
                fn = tc.get("function", {})
                yield ChatChunk(
                    type="tool_call",
                    tool_call=ToolCall(
                        name=fn.get("name", ""),
                        arguments=fn.get("arguments", {}),
                    ),
                )

        if content:
            yield ChatChunk(type="text", text=content)

        yield ChatChunk(type="done")
        logger.info("ollama_chat_completed", model=self.model)

    def _to_ollama_message(self, message: Message) -> dict[str, Any]:
        role = message.role
        if role == "tool":
            return {"role": "tool", "content": message.content or ""}
        return {"role": role, "content": message.content or ""}

    def _to_ollama_tool(self, tool: ToolDeclaration) -> dict[str, Any]:
        return {
            "type": "function",
            "function": {
                "name": tool.name,
                "description": tool.description,
                "parameters": tool.parameters,
            },
        }

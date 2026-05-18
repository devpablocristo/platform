"""Application service for provider orchestration."""

from __future__ import annotations

import asyncio
import json
from collections.abc import AsyncIterator, Awaitable, Callable
from dataclasses import dataclass
from typing import Any

from runtime.domain.models import ChatChunk, LLMProvider, Message, ToolCall, ToolDeclaration

ToolHandler = Callable[..., Awaitable[dict[str, Any]]]


@dataclass(frozen=True)
class OrchestratorLimits:
    max_tool_calls: int = 10
    tool_timeout_seconds: float = 10.0
    total_timeout_seconds: float = 60.0
    timeout_message: str = "request timed out"


async def orchestrate(
    llm: LLMProvider,
    messages: list[Message],
    tools: list[ToolDeclaration],
    tool_handlers: dict[str, ToolHandler],
    context: dict[str, Any] | None = None,
    limits: OrchestratorLimits | None = None,
) -> AsyncIterator[ChatChunk]:
    """Execute a simple LLM -> tool calls -> tool results loop."""

    effective_limits = limits or OrchestratorLimits()
    payload = list(messages)
    tool_calls_count = 0
    started_at = asyncio.get_running_loop().time()
    base_context = dict(context or {})

    while tool_calls_count < effective_limits.max_tool_calls:
        if asyncio.get_running_loop().time() - started_at > effective_limits.total_timeout_seconds:
            yield ChatChunk(type="text", text=effective_limits.timeout_message)
            break

        pending_tool_calls: list[ToolCall] = []
        text_buffer: list[str] = []

        async for chunk in llm.chat(payload, tools=tools):
            if chunk.type == "text" and chunk.text is not None:
                text_buffer.append(chunk.text)
                yield chunk
            elif chunk.type == "tool_call" and chunk.tool_call is not None:
                pending_tool_calls.append(chunk.tool_call)

        if not pending_tool_calls:
            break

        payload.append(Message(role="assistant", content="".join(text_buffer), tool_calls=pending_tool_calls))

        for tool_call in pending_tool_calls:
            tool_calls_count += 1
            # Gemini/Vertex a veces prefija el nombre de la tool con `default_api.`.
            resolved_name = tool_call.name
            handler = tool_handlers.get(resolved_name)
            if handler is None and "." in resolved_name:
                resolved_name = resolved_name.rsplit(".", 1)[-1]
                handler = tool_handlers.get(resolved_name)
            yield ChatChunk(type="tool_call", tool_call=tool_call)

            if handler is None:
                result: dict[str, Any] = {"error": f"tool {tool_call.name} not found"}
            else:
                try:
                    result = await asyncio.wait_for(
                        handler(**base_context, **tool_call.arguments),
                        timeout=effective_limits.tool_timeout_seconds,
                    )
                except asyncio.TimeoutError:
                    result = {"error": f"timeout executing tool {tool_call.name}"}
                except Exception as exc:  # noqa: BLE001
                    result = {"error": str(exc)}

            payload.append(
                Message(
                    role="tool",
                    content=json.dumps(result, ensure_ascii=False, default=str),
                    tool_call_id=tool_call.name,
                )
            )
            yield ChatChunk(type="tool_result", tool_call=ToolCall(name=tool_call.name, arguments=result))

    yield ChatChunk(type="done")

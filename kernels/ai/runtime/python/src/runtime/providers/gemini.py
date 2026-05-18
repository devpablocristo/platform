"""Google Gemini LLM provider con retry y circuit breaker."""

from __future__ import annotations

import asyncio
import random
from collections.abc import AsyncIterator
from typing import Any

from google import genai
from google.genai import types

from runtime.logging import get_logger
from runtime.resilience import CircuitBreaker, CircuitBreakerOpenError
from runtime.types import ChatChunk, Message, ToolCall, ToolDeclaration

logger = get_logger(__name__)


class GeminiProvider:
    """Provider de Google Gemini con retry exponencial y circuit breaker."""

    def __init__(
        self,
        api_key: str | None = None,
        model: str = "gemini-2.0-flash",
        circuit_breaker: CircuitBreaker | None = None,
        max_retries: int = 3,
        retry_base_delay_seconds: float = 0.2,
        *,
        vertex_project: str | None = None,
        vertex_location: str | None = None,
    ) -> None:
        if vertex_project:
            self.client = genai.Client(
                vertexai=True,
                project=vertex_project,
                location=vertex_location or "us-central1",
            )
        else:
            if not api_key:
                raise ValueError("GeminiProvider requires api_key or vertex_project")
            self.client = genai.Client(api_key=api_key)
        self.model = model
        self.circuit_breaker = circuit_breaker or CircuitBreaker()
        self._max_retries = max_retries
        self._retry_base_delay = retry_base_delay_seconds

    async def chat(
        self,
        messages: list[Message],
        tools: list[ToolDeclaration] | None = None,
        temperature: float = 0.3,
        max_tokens: int = 2048,
    ) -> AsyncIterator[ChatChunk]:
        try:
            chunks = await self.circuit_breaker.call(
                self._collect_chunks_with_retry,
                messages,
                tools,
                temperature,
                max_tokens,
            )
        except CircuitBreakerOpenError:
            logger.warning("llm_circuit_open", model=self.model, state=self.circuit_breaker.state.value)
            raise RuntimeError("llm temporarily unavailable")
        except Exception as exc:  # noqa: BLE001
            logger.exception("llm_chat_failed", model=self.model, error=str(exc))
            raise

        logger.info("llm_chat_completed", model=self.model, chunks=len(chunks))
        for chunk in chunks:
            yield chunk

    async def _collect_chunks_with_retry(
        self,
        messages: list[Message],
        tools: list[ToolDeclaration] | None,
        temperature: float,
        max_tokens: int,
    ) -> list[ChatChunk]:
        last_error: Exception | None = None
        for attempt in range(1, self._max_retries + 1):
            try:
                return await self._collect_chunks(messages, tools, temperature, max_tokens)
            except Exception as exc:  # noqa: BLE001
                last_error = exc
                if attempt >= self._max_retries:
                    raise
                delay = self._retry_delay_seconds(attempt)
                logger.warning(
                    "llm_retrying",
                    model=self.model,
                    attempt=attempt,
                    delay_seconds=round(delay, 3),
                    error=str(exc),
                )
                await asyncio.sleep(delay)
        if last_error is not None:
            raise last_error
        raise RuntimeError("llm request failed without error")

    async def _collect_chunks(
        self,
        messages: list[Message],
        tools: list[ToolDeclaration] | None,
        temperature: float,
        max_tokens: int,
    ) -> list[ChatChunk]:
        return await asyncio.to_thread(self._collect_chunks_sync, messages, tools, temperature, max_tokens)

    def _collect_chunks_sync(
        self,
        messages: list[Message],
        tools: list[ToolDeclaration] | None,
        temperature: float,
        max_tokens: int,
    ) -> list[ChatChunk]:
        system_parts = [message.content for message in messages if message.role == "system"]
        system_instruction = "\n\n".join(system_parts) if system_parts else None

        config = types.GenerateContentConfig(
            temperature=temperature,
            max_output_tokens=max_tokens,
            system_instruction=system_instruction,
        )
        if tools:
            config.tools = [types.Tool(function_declarations=[self._to_gemini_tool(tool) for tool in tools])]

        non_system = [message for message in messages if message.role != "system"]
        response = self.client.models.generate_content_stream(
            model=self.model,
            contents=self._to_gemini_messages(non_system),
            config=config,
        )

        chunks: list[ChatChunk] = []
        for chunk in response:
            if not chunk.candidates:
                continue
            candidate = chunk.candidates[0]
            if not candidate.content or not candidate.content.parts:
                continue
            for part in candidate.content.parts:
                if getattr(part, "function_call", None):
                    function_call = part.function_call
                    chunks.append(
                        ChatChunk(
                            type="tool_call",
                            tool_call=ToolCall(
                                name=function_call.name,
                                arguments=dict(function_call.args or {}),
                            ),
                        )
                    )
                elif getattr(part, "text", None):
                    chunks.append(ChatChunk(type="text", text=part.text))

        chunks.append(ChatChunk(type="done"))
        return chunks

    def _retry_delay_seconds(self, attempt: int) -> float:
        base = self._retry_base_delay * (2 ** (attempt - 1))
        return base + random.uniform(0, base / 2)

    def _to_gemini_messages(self, messages: list[Message]) -> list[types.Content]:
        converted: list[types.Content] = []
        pending_tool_parts: list[types.Part] = []

        def flush_tools() -> None:
            if pending_tool_parts:
                converted.append(types.Content(role="user", parts=list(pending_tool_parts)))
                pending_tool_parts.clear()

        for message in messages:
            if message.role == "tool":
                pending_tool_parts.append(
                    types.Part.from_function_response(
                        name=message.tool_call_id or "unknown",
                        response={"result": message.content},
                    )
                )
                continue

            flush_tools()

            if message.role == "assistant":
                parts: list[types.Part] = []
                if message.content:
                    parts.append(types.Part.from_text(text=message.content))
                if message.tool_calls:
                    for tool_call in message.tool_calls:
                        parts.append(
                            types.Part.from_function_call(
                                name=tool_call.name,
                                args=tool_call.arguments,
                            )
                        )
                if parts:
                    converted.append(types.Content(role="model", parts=parts))
            else:
                converted.append(
                    types.Content(
                        role="user",
                        parts=[types.Part.from_text(text=message.content or "")],
                    )
                )

        flush_tools()
        return converted

    def _to_gemini_tool(self, tool: ToolDeclaration) -> types.FunctionDeclaration:
        return types.FunctionDeclaration(
            name=tool.name,
            description=tool.description,
            parameters=self._to_schema(tool.parameters),
        )

    def _to_schema(self, schema: dict[str, Any]) -> types.Schema:
        schema_type = str(schema.get("type", "object")).upper()
        properties = schema.get("properties", {})
        required = schema.get("required", [])
        converted_properties: dict[str, types.Schema] = {}
        for key, value in properties.items():
            prop_type = str(value.get("type", "string")).upper()
            if prop_type == "ARRAY":
                items_schema = value.get("items", {"type": "object"})
                converted_properties[key] = types.Schema(
                    type="ARRAY",
                    description=value.get("description"),
                    items=self._to_schema(items_schema),
                )
            elif prop_type == "OBJECT" and value.get("properties"):
                converted_properties[key] = self._to_schema(value)
            else:
                converted_properties[key] = types.Schema(
                    type=prop_type,
                    description=value.get("description"),
                )
        return types.Schema(type=schema_type, properties=converted_properties, required=required)

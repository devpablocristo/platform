"""Orquestador multi-agente: rutea al sub-agente correcto y ejecuta orchestrate().

Flujo:
    1. route() elige el sub-agente (LLM classifier)
    2. Se busca el SubAgent en el registry
    3. Se prepara el payload con el system prompt del agente
    4. Se llama orchestrate() con los tools del agente
    5. Si es "general", orchestrate() sin tools (conversación pura)
"""

from __future__ import annotations

from collections.abc import AsyncIterator
from typing import Any

from runtime.domain.agent import AgentRegistry
from runtime.domain.models import ChatChunk, LLMProvider, Message
from runtime.logging import get_logger
from runtime.services.agent_router import GENERAL_AGENT, route
from runtime.services.orchestrator import OrchestratorLimits, orchestrate

logger = get_logger(__name__)

_GENERAL_SYSTEM_PROMPT = """\
You are a general assistant for a software product.
Answer greetings and general questions clearly and concisely.
If the user asks for a concrete domain action, tell them they can ask for it directly.
Do not mention tools unless it helps clarify what the user can ask next."""


async def run_routed_agent(
    llm: LLMProvider,
    registry: AgentRegistry,
    user_message: str,
    history: list[Message] | None = None,
    context: dict[str, Any] | None = None,
    general_system_prompt: str = _GENERAL_SYSTEM_PROMPT,
    general_limits: OrchestratorLimits | None = None,
) -> AsyncIterator[ChatChunk]:
    """Rutea al sub-agente correcto y ejecuta orchestrate().

    Yields:
        ChatChunk con type="route" primero (nombre del agente seleccionado),
        seguido de los chunks normales de orchestrate().
    """
    descriptors = registry.descriptors()
    selected = await route(
        llm=llm,
        message=user_message,
        history=history,
        descriptors=descriptors,
        fallback=GENERAL_AGENT,
    )

    # Emitir qué agente se seleccionó
    yield ChatChunk(type="route", text=selected)

    agent = registry.get(selected)

    if agent is None:
        # Fallback conversacional: sin tools
        messages = [
            Message(role="system", content=general_system_prompt),
            *(history or []),
            Message(role="user", content=user_message),
        ]
        async for chunk in orchestrate(
            llm=llm,
            messages=messages,
            tools=[],
            tool_handlers={},
            context=context,
            limits=general_limits or OrchestratorLimits(
                max_tool_calls=0,
                total_timeout_seconds=30.0,
            ),
        ):
            yield chunk
        return

    # Sub-agente encontrado: ejecutar con sus tools
    messages = [
        Message(role="system", content=agent.system_prompt),
        *(history or []),
        Message(role="user", content=user_message),
    ]

    logger.info(
        "sub_agent_executing",
        agent=agent.name,
        tools_count=len(agent.tools),
        message_preview=user_message[:40],
    )

    async for chunk in orchestrate(
        llm=llm,
        messages=messages,
        tools=agent.tools,
        tool_handlers=agent.tool_handlers,
        context=context,
        limits=agent.limits,
    ):
        yield chunk

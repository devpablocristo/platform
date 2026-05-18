"""Router basado en LLM: clasifica el mensaje del usuario y elige el sub-agente.

Reemplaza routers basados en keywords. Usa una sola llamada al LLM sin tools,
con temperatura 0 para determinismo. Si falla, devuelve el fallback.
"""

from __future__ import annotations

from runtime.domain.agent import SubAgentDescriptor
from runtime.domain.models import ChatChunk, LLMProvider, Message
from runtime.logging import get_logger

logger = get_logger(__name__)

GENERAL_AGENT = "general"

_ROUTER_SYSTEM_PROMPT = """\
You are a routing assistant for a business management platform.
Given the user message, choose which specialized agent should handle this request.

Available agents:
{agent_list}

Rules:
- If the message is a greeting, general question, or does not match any agent, respond with "{fallback}".
- Respond with ONLY the agent name. No explanation, no punctuation, no quotes.
"""


def _build_router_prompt(
    descriptors: list[SubAgentDescriptor],
    fallback: str = GENERAL_AGENT,
) -> str:
    """Construye el system prompt del router con los agentes disponibles."""
    lines = [f"- {d.name}: {d.description}" for d in descriptors]
    return _ROUTER_SYSTEM_PROMPT.format(
        agent_list="\n".join(lines),
        fallback=fallback,
    )


def _extract_agent_name(response: str, valid_names: set[str], fallback: str) -> str:
    """Extrae el nombre del agente de la respuesta del LLM. Fallback si no matchea."""
    cleaned = response.strip().lower().strip('"\'`.').split("\n")[0].strip()
    if cleaned in valid_names:
        return cleaned
    # Intenta match parcial (el LLM puede agregar texto extra)
    for name in valid_names:
        if name in cleaned:
            return name
    return fallback


async def route(
    llm: LLMProvider,
    message: str,
    history: list[Message] | None = None,
    descriptors: list[SubAgentDescriptor] | None = None,
    fallback: str = GENERAL_AGENT,
) -> str:
    """Usa el LLM para decidir qué sub-agente debe manejar el mensaje.

    Args:
        llm: Provider de LLM.
        message: Mensaje del usuario.
        history: Últimos mensajes de la conversación (contexto).
        descriptors: Descriptors de los sub-agentes disponibles.
        fallback: Nombre del agente por defecto si no se puede clasificar.

    Returns:
        Nombre del sub-agente seleccionado.
    """
    if not descriptors:
        return fallback

    system_prompt = _build_router_prompt(descriptors, fallback)
    valid_names = {d.name for d in descriptors} | {fallback}

    # Contexto: últimos mensajes + mensaje actual
    tail = list(history or [])[-6:]
    messages: list[Message] = [
        Message(role="system", content=system_prompt),
        *tail,
        Message(role="user", content=message),
    ]

    try:
        response_parts: list[str] = []
        async for chunk in llm.chat(
            messages,
            tools=None,
            temperature=0.0,
            max_tokens=20,
        ):
            if chunk.type == "text" and chunk.text:
                response_parts.append(chunk.text)

        raw_response = "".join(response_parts)
        selected = _extract_agent_name(raw_response, valid_names, fallback)
        logger.info(
            "agent_routed",
            selected=selected,
            raw_response=raw_response.strip()[:60],
            message_preview=message[:40],
        )
        return selected

    except Exception as exc:  # noqa: BLE001
        logger.warning("agent_router_failed", error=str(exc), fallback=fallback)
        return fallback

"""Interface de persistencia de conversaciones — el producto provee la implementación."""

from __future__ import annotations

from typing import Protocol

from .types import Conversation, Turn


class MemoryStore(Protocol):
    """Puerto de persistencia. La implementación concreta (PostgreSQL, Redis, etc.)
    la provee el producto que consume runtime, no esta librería.

    Todos los métodos son async para soportar I/O sin bloquear.
    """

    async def create_conversation(
        self,
        org_id: str,
        mode: str,
        *,
        user_id: str | None = None,
        channel: str = "console",
        contact_id: str | None = None,
        contact_name: str = "",
        title: str = "",
    ) -> Conversation:
        """Crea una conversación nueva y la devuelve con ID asignado."""
        ...

    async def get_conversation(self, org_id: str, conversation_id: str) -> Conversation | None:
        """Busca una conversación por org + id. None si no existe."""
        ...

    async def list_conversations(
        self,
        org_id: str,
        *,
        mode: str | None = None,
        user_id: str | None = None,
        channel: str | None = None,
        limit: int = 50,
    ) -> list[Conversation]:
        """Lista conversaciones de una org, filtradas opcionalmente."""
        ...

    async def append_turns(
        self,
        org_id: str,
        conversation_id: str,
        turns: list[Turn],
        *,
        tokens_input: int = 0,
        tokens_output: int = 0,
        tool_calls_count: int = 0,
    ) -> Conversation | None:
        """Agrega turnos a una conversación existente. None si no existe."""
        ...

    async def delete_conversation(self, org_id: str, conversation_id: str) -> bool:
        """Elimina una conversación. True si existía."""
        ...

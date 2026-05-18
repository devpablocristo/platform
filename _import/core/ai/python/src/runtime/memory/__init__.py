"""Memory primitives for AI conversations."""

from .types import Conversation, Turn
from .store import MemoryStore
from .context import build_context_window
from .operational import (
    append_unique_list,
    build_operational_memory_view,
    capture_operational_turn,
    consolidate_operational_memory,
    ensure_operational_memory,
    normalize_memory_text,
    remove_resolved_open_loops,
    upsert_memory_fact,
)

__all__ = [
    "Conversation",
    "Turn",
    "MemoryStore",
    "build_context_window",
    "append_unique_list",
    "build_operational_memory_view",
    "capture_operational_turn",
    "consolidate_operational_memory",
    "ensure_operational_memory",
    "normalize_memory_text",
    "remove_resolved_open_loops",
    "upsert_memory_fact",
]

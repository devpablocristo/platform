"""Optional FastAPI adapters for reusable AI runtime components."""

from __future__ import annotations

from typing import Any

from .sse import EventSourceResponse

__all__ = ["EventSourceResponse", "create_app"]


def __getattr__(name: str) -> Any:
    """Evita importar `create_app` (y httpserver) al cargar `runtime.api.events` o `runtime.api.sse`."""
    if name == "create_app":
        from .app import create_app

        return create_app
    raise AttributeError(f"module {__name__!r} has no attribute {name!r}")

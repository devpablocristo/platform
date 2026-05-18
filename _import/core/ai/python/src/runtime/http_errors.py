"""Puente a `httpserver.errors`: import perezoso y fallback si el paquete no está instalado (p. ej. tests mínimos)."""

from __future__ import annotations

from typing import Any


def error_payload(
    code: str,
    message: str,
    request_id: str,
    details: dict[str, Any] | None = None,
) -> dict[str, Any]:
    try:
        from httpserver.errors import error_payload as _impl

        return _impl(code, message, request_id, details)
    except ModuleNotFoundError:
        return {
            "error": {
                "code": code,
                "message": message,
                "details": details or {},
                "request_id": request_id,
            }
        }

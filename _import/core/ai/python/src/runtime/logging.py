"""Logging y contexto de request (contextvars + structlog para eventos con kwargs)."""

from __future__ import annotations

import logging
from contextvars import ContextVar
from typing import Any

import structlog

request_id_var: ContextVar[str] = ContextVar("request_id", default="")
tenant_id_var: ContextVar[str] = ContextVar("tenant_id", default="")
user_id_var: ContextVar[str] = ContextVar("user_id", default="")


def configure_logging(level: str = "INFO", *, json_logs: bool = False) -> None:
    lvl = getattr(logging, level.upper(), logging.INFO)
    processors: list[Any] = [
        structlog.contextvars.merge_contextvars,
        structlog.processors.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
    ]
    if json_logs:
        processors.append(structlog.processors.JSONRenderer())
    else:
        processors.append(structlog.dev.ConsoleRenderer(colors=False))
    structlog.configure(
        processors=processors,
        wrapper_class=structlog.make_filtering_bound_logger(lvl),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=False,
    )


def bind_request_context(request_id: str, tenant_id: str = "", user_id: str = "") -> None:
    request_id_var.set(request_id)
    tenant_id_var.set(tenant_id)
    user_id_var.set(user_id)


def update_request_context(tenant_id: str = "", user_id: str = "", org_id: str = "") -> None:
    # org_id es alias de tenant_id (convención Pymes / multi-tenant)
    tid = tenant_id or org_id
    if tid:
        tenant_id_var.set(tid)
    if user_id:
        user_id_var.set(user_id)


def clear_request_context() -> None:
    request_id_var.set("")
    tenant_id_var.set("")
    user_id_var.set("")


def get_request_id() -> str:
    return request_id_var.get()


def get_logger(name: str) -> Any:
    return structlog.get_logger(name)


def context_fields() -> dict[str, Any]:
    return {
        "request_id": request_id_var.get(),
        "tenant_id": tenant_id_var.get(),
        "user_id": user_id_var.get(),
    }


def format_log_event(event: str, /, **fields: Any) -> str:
    parts = [event]
    for key, value in fields.items():
        parts.append(f"{key}={value}")
    return " ".join(parts)

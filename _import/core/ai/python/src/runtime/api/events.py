"""Helpers for SSE-compatible event payloads."""

from __future__ import annotations

import json
from typing import Any


def to_sse_event(event: str, payload: dict[str, Any]) -> dict[str, str]:
    return {"event": event, "data": json.dumps(payload, ensure_ascii=False)}

from __future__ import annotations

import httpx
import pytest

from runtime.clients.http_backend import HTTPBackendClient
from runtime.contexts import AuthContext
from runtime.logging import bind_request_context, clear_request_context


@pytest.fixture(autouse=True)
def clear_request_context_fixture() -> None:
    clear_request_context()
    yield
    clear_request_context()


@pytest.mark.asyncio
async def test_request_propagates_headers_and_internal_token() -> None:
    captured: dict[str, str] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["authorization"] = request.headers.get("Authorization", "")
        captured["org_id"] = request.headers.get("X-Org-ID", "")
        captured["request_id"] = request.headers.get("X-Request-ID", "")
        captured["internal"] = request.headers.get("X-Internal-Service-Token", "")
        return httpx.Response(200, json={"ok": True})

    bind_request_context("req-123")
    client = HTTPBackendClient(
        "https://backend.test",
        "internal-secret",
        client=httpx.AsyncClient(
            base_url="https://backend.test",
            transport=httpx.MockTransport(handler),
        ),
    )
    auth = AuthContext(
        tenant_id="org-123",
        actor="user-123",
        role="admin",
        scopes=["sales:read"],
        mode="internal",
        authorization="Bearer token-123",
    )

    payload = await client.request("GET", "/status", auth=auth, include_internal=True)

    assert payload == {"ok": True}
    assert captured["authorization"] == "Bearer token-123"
    assert captured["org_id"] == "org-123"
    assert captured["request_id"] == "req-123"
    assert captured["internal"] == "internal-secret"


@pytest.mark.asyncio
async def test_request_propagates_api_key_and_scopes() -> None:
    captured: dict[str, str] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["api_key"] = request.headers.get("X-API-Key", "")
        captured["scopes"] = request.headers.get("X-Scopes", "")
        captured["org_id"] = request.headers.get("X-Org-ID", "")
        return httpx.Response(200, json={"ok": True})

    client = HTTPBackendClient(
        "https://backend.test",
        "internal-secret",
        client=httpx.AsyncClient(
            base_url="https://backend.test",
            transport=httpx.MockTransport(handler),
        ),
    )
    auth = AuthContext(
        tenant_id="org-123",
        actor="api_key:key-123",
        role="service",
        scopes=["sales:read", "accounts:read"],
        mode="api_key",
        api_key="psk_local_admin",
        api_scopes="sales:read,accounts:read",
    )

    payload = await client.request("GET", "/status", auth=auth)

    assert payload == {"ok": True}
    assert captured["api_key"] == "psk_local_admin"
    assert captured["scopes"] == "sales:read,accounts:read"
    assert captured["org_id"] == "org-123"


@pytest.mark.asyncio
async def test_request_retries_retryable_5xx() -> None:
    attempts = {"count": 0}

    def handler(request: httpx.Request) -> httpx.Response:
        del request
        attempts["count"] += 1
        if attempts["count"] == 1:
            return httpx.Response(502, json={"error": "bad gateway"})
        return httpx.Response(200, json={"status": "ok"})

    client = HTTPBackendClient(
        "https://backend.test",
        "internal-secret",
        base_backoff_seconds=0,
        client=httpx.AsyncClient(
            base_url="https://backend.test",
            transport=httpx.MockTransport(handler),
        ),
    )

    payload = await client.request("GET", "/healthz")

    assert payload == {"status": "ok"}
    assert attempts["count"] == 2

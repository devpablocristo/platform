from __future__ import annotations

import json
import httpx
import pytest

from runtime.clients.review import ReviewClient, ReviewRequester


@pytest.mark.asyncio
async def test_submit_request_uses_custom_requester_identity() -> None:
    captured: dict[str, object] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["headers"] = dict(request.headers)
        captured["json"] = json.loads(request.read().decode("utf-8"))
        return httpx.Response(
            200,
            json={
                "request_id": "req-123",
                "decision": "allow",
                "risk_level": "low",
                "decision_reason": "ok",
                "status": "completed",
                "approval": {"id": "approval-1", "expires_at": "2026-01-01T00:00:00Z"},
            },
        )

    client = ReviewClient(
        "https://review.test",
        "test-api-key",
        requester=ReviewRequester(
            requester_id="pymes-ai",
            requester_name="Pymes AI Service",
        ),
        client=httpx.AsyncClient(
            base_url="https://review.test",
            transport=httpx.MockTransport(handler),
        ),
    )

    response = await client.submit_request(
        action_type="sale.create",
        target_system="pymes",
        target_resource="sale-1",
        params={"amount": 1500},
        reason="Create sale",
    )

    assert response.request_id == "req-123"
    assert response.approval_id == "approval-1"
    payload = captured["json"]
    assert isinstance(payload, dict)
    assert payload["requester_id"] == "pymes-ai"
    assert payload["requester_name"] == "Pymes AI Service"
    assert payload["target_system"] == "pymes"
    headers = captured["headers"]
    assert isinstance(headers, dict)
    assert "idempotency-key" in {key.lower() for key in headers}

    await client.close()


@pytest.mark.asyncio
async def test_submit_request_falls_back_to_require_approval() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        del request
        return httpx.Response(503, json={"error": "unavailable"})

    client = ReviewClient(
        "https://review.test",
        "test-api-key",
        client=httpx.AsyncClient(
            base_url="https://review.test",
            transport=httpx.MockTransport(handler),
        ),
    )

    response = await client.submit_request(action_type="sale.create")

    assert response.decision == "require_approval"
    assert response.status == "fallback"
    assert response.risk_level == "unknown"

    await client.close()

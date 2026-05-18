from __future__ import annotations

import unittest

from fastapi import FastAPI, Request
from fastapi.testclient import TestClient

from runtime.contexts import AuthContext
from runtime.rate_limit import RateLimitMiddleware, RateLimitSettings


class RateLimitTests(unittest.TestCase):
    def test_public_rate_limit(self) -> None:
        app = FastAPI()
        app.add_middleware(RateLimitMiddleware, settings=RateLimitSettings(external_rpm=1, internal_rpm=10))

        @app.get("/v1/public/ping")
        async def ping() -> dict[str, str]:
            return {"ok": "1"}

        client = TestClient(app)
        first = client.get("/v1/public/ping")
        second = client.get("/v1/public/ping")
        self.assertEqual(first.status_code, 200)
        self.assertEqual(second.status_code, 429)

    def test_internal_rate_limit_uses_tenant(self) -> None:
        app = FastAPI()
        app.add_middleware(RateLimitMiddleware, settings=RateLimitSettings(external_rpm=10, internal_rpm=1))

        @app.middleware("http")
        async def inject_auth(request: Request, call_next):  # type: ignore[no-untyped-def]
            request.state.auth = AuthContext(tenant_id="acme", actor="bot", role="service", scopes=[], mode="internal")
            return await call_next(request)

        @app.get("/v1/chat")
        async def chat() -> dict[str, str]:
            return {"ok": "1"}

        client = TestClient(app)
        first = client.get("/v1/chat")
        second = client.get("/v1/chat")
        self.assertEqual(first.status_code, 200)
        self.assertEqual(second.status_code, 429)


if __name__ == "__main__":
    unittest.main()

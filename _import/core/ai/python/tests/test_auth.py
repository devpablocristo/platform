from __future__ import annotations

import unittest

from fastapi import FastAPI
from fastapi.testclient import TestClient

from runtime.auth import APIKeyVerifier, AuthMiddleware, AuthSettings, BearerVerifier
from runtime.contexts import AuthContext


class StaticBearer(BearerVerifier):
    async def verify_bearer(self, token: str) -> AuthContext | None:
        if token == "token-1":
            return AuthContext(tenant_id="acme", actor="user-1", role="admin", scopes=["chat:run"], mode="bearer")
        return None


class StaticAPIKey(APIKeyVerifier):
    async def verify_api_key(self, key: str) -> AuthContext | None:
        if key == "key-1":
            return AuthContext(tenant_id="acme", actor="api_key:key-1", role="service", scopes=["chat:run"], mode="api_key")
        return None


class AuthTests(unittest.TestCase):
    def test_bearer_auth(self) -> None:
        app = FastAPI()
        app.add_middleware(AuthMiddleware, settings=AuthSettings(), bearer_verifier=StaticBearer(), api_key_verifier=StaticAPIKey())

        @app.get("/v1/chat")
        async def chat() -> dict[str, str]:
            return {"ok": "1"}

        client = TestClient(app)
        response = client.get("/v1/chat", headers={"Authorization": "Bearer token-1"})
        self.assertEqual(response.status_code, 200)

    def test_api_key_auth(self) -> None:
        app = FastAPI()
        app.add_middleware(AuthMiddleware, settings=AuthSettings(allow_api_key=True), bearer_verifier=StaticBearer(), api_key_verifier=StaticAPIKey())

        @app.get("/v1/chat")
        async def chat() -> dict[str, str]:
            return {"ok": "1"}

        client = TestClient(app)
        response = client.get("/v1/chat", headers={"X-API-KEY": "key-1"})
        self.assertEqual(response.status_code, 200)

    def test_missing_auth(self) -> None:
        app = FastAPI()
        app.add_middleware(AuthMiddleware, settings=AuthSettings(), bearer_verifier=StaticBearer(), api_key_verifier=StaticAPIKey())

        @app.get("/v1/chat")
        async def chat() -> dict[str, str]:
            return {"ok": "1"}

        client = TestClient(app)
        response = client.get("/v1/chat")
        self.assertEqual(response.status_code, 401)


if __name__ == "__main__":
    unittest.main()

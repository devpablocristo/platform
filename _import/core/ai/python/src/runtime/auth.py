"""Authentication middleware with injected verifiers."""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from typing import Protocol

from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import JSONResponse, Response
from starlette.types import ASGIApp

from runtime.contexts import AuthContext
from runtime.http_errors import error_payload
from runtime.logging import update_request_context as default_update_request_context


class BearerVerifier(Protocol):
    async def verify_bearer(self, token: str) -> AuthContext | None: ...


class APIKeyVerifier(Protocol):
    async def verify_api_key(self, key: str) -> AuthContext | None: ...


@dataclass(slots=True)
class AuthSettings:
    allow_api_key: bool = True


class AuthMiddleware(BaseHTTPMiddleware):
    def __init__(
        self,
        app: ASGIApp,
        settings: AuthSettings | None = None,
        bearer_verifier: BearerVerifier | None = None,
        api_key_verifier: APIKeyVerifier | None = None,
        update_request_context: Callable[..., None] | None = None,
        public_prefixes: tuple[str, ...] = ("/v1/public/",),
        protected_prefixes: tuple[str, ...] = ("/v1/chat",),
        health_prefixes: tuple[str, ...] = ("/healthz", "/readyz"),
    ) -> None:
        super().__init__(app)
        self.settings = settings or AuthSettings()
        self.bearer_verifier = bearer_verifier
        self.api_key_verifier = api_key_verifier
        self.update_request_context = update_request_context or default_update_request_context
        self.public_prefixes = public_prefixes
        self.protected_prefixes = protected_prefixes
        self.health_prefixes = health_prefixes

    async def dispatch(self, request: Request, call_next: Callable[[Request], Awaitable[Response]]) -> Response:
        if request.method == "OPTIONS":
            return await call_next(request)
        path = request.url.path
        request_id = getattr(request.state, "request_id", "")
        if any(path.startswith(prefix) for prefix in self.health_prefixes + self.public_prefixes):
            return await call_next(request)
        if not any(path.startswith(prefix) for prefix in self.protected_prefixes):
            return await call_next(request)

        authz = request.headers.get("Authorization", "")
        api_key = request.headers.get("X-API-KEY", "")

        if authz.lower().startswith("bearer ") and self.bearer_verifier is not None:
            token = authz[7:].strip()
            auth = await self.bearer_verifier.verify_bearer(token)
            if auth is not None:
                request.state.auth = auth
                self.update_request_context(tenant_id=auth.tenant_id, user_id=auth.actor)
                return await call_next(request)

        if api_key and self.settings.allow_api_key and self.api_key_verifier is not None:
            auth = await self.api_key_verifier.verify_api_key(api_key)
            if auth is not None:
                request.state.auth = auth
                self.update_request_context(tenant_id=auth.tenant_id, user_id=auth.actor)
                return await call_next(request)

        return JSONResponse(status_code=401, content=error_payload("unauthorized", "unauthorized", request_id))

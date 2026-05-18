"""HTTP middleware for request context propagation."""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from uuid import uuid4

from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response
from starlette.types import ASGIApp

from runtime.logging import bind_request_context, clear_request_context


class RequestContextMiddleware(BaseHTTPMiddleware):
    def __init__(self, app: ASGIApp, header_name: str = "X-Request-Id") -> None:
        super().__init__(app)
        self.header_name = header_name

    async def dispatch(self, request: Request, call_next: Callable[[Request], Awaitable[Response]]) -> Response:
        request_id = request.headers.get(self.header_name, "").strip() or str(uuid4())
        request.state.request_id = request_id
        bind_request_context(request_id=request_id)

        try:
            response = await call_next(request)
        finally:
            clear_request_context()

        response.headers[self.header_name] = request_id
        return response

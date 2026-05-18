"""Generic HTTP backend client with auth propagation and retryable transport handling."""

from __future__ import annotations

import asyncio
import time
from typing import Any

import httpx

from runtime.contexts import AuthContext
from runtime.logging import format_log_event, get_logger, get_request_id

logger = get_logger(__name__)


class HTTPBackendClient:
    def __init__(
        self,
        base_url: str,
        internal_token: str,
        *,
        timeout: float = 10.0,
        max_attempts: int = 3,
        base_backoff_seconds: float = 0.2,
        client: httpx.AsyncClient | None = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.internal_token = internal_token
        self.timeout = timeout
        self.max_attempts = max(1, max_attempts)
        self.base_backoff_seconds = max(0.0, base_backoff_seconds)
        self._owns_client = client is None
        self._client = client or httpx.AsyncClient(base_url=self.base_url, timeout=self.timeout)

    async def close(self) -> None:
        if self._owns_client:
            await self._client.aclose()

    def _headers(self, auth: AuthContext | None, *, include_internal: bool = False) -> dict[str, str]:
        headers: dict[str, str] = {}
        request_id = get_request_id()
        if request_id:
            headers["X-Request-ID"] = request_id
        if include_internal and self.internal_token:
            headers["X-Internal-Service-Token"] = self.internal_token

        if auth is None:
            return headers

        if auth.authorization:
            headers["Authorization"] = auth.authorization
        elif auth.api_key:
            headers["X-API-Key"] = auth.api_key
        elif auth.api_scopes:
            headers["X-Scopes"] = auth.api_scopes
        elif auth.scopes:
            headers["X-Scopes"] = ",".join(auth.scopes)
        if auth.api_key and auth.api_scopes:
            headers["X-Scopes"] = auth.api_scopes
        elif auth.api_key and auth.scopes:
            headers["X-Scopes"] = ",".join(auth.scopes)
        if auth.org_id:
            headers["X-Org-ID"] = auth.org_id
        return headers

    async def request(
        self,
        method: str,
        path: str,
        auth: AuthContext | None = None,
        *,
        include_internal: bool = False,
        **kwargs: Any,
    ) -> dict[str, Any]:
        headers = self._headers(auth, include_internal=include_internal)
        last_error: Exception | None = None

        for attempt in range(self.max_attempts):
            started_at = time.perf_counter()
            try:
                response = await self._client.request(method, path, headers=headers, **kwargs)
                if response.status_code >= 500 and attempt < self.max_attempts - 1:
                    logger.warning(
                        format_log_event(
                            "backend_retryable_status",
                            method=method,
                            path=path,
                            status_code=response.status_code,
                            attempt=attempt + 1,
                        )
                    )
                    await self._sleep_before_retry(attempt)
                    continue

                logger.info(
                    format_log_event(
                        "backend_request",
                        method=method,
                        path=path,
                        status_code=response.status_code,
                        duration_ms=round((time.perf_counter() - started_at) * 1000, 2),
                    )
                )
                response.raise_for_status()
                if response.headers.get("content-type", "").startswith("application/json"):
                    return response.json()
                return {"raw": response.text}
            except httpx.HTTPStatusError:
                logger.warning(
                    format_log_event(
                        "backend_http_error",
                        method=method,
                        path=path,
                        attempt=attempt + 1,
                        duration_ms=round((time.perf_counter() - started_at) * 1000, 2),
                    )
                )
                raise
            except (httpx.TimeoutException, httpx.NetworkError, httpx.RemoteProtocolError) as exc:
                last_error = exc
                logger.warning(
                    format_log_event(
                        "backend_transport_error",
                        method=method,
                        path=path,
                        error=str(exc),
                        attempt=attempt + 1,
                        duration_ms=round((time.perf_counter() - started_at) * 1000, 2),
                    )
                )
                if attempt == self.max_attempts - 1:
                    raise
                await self._sleep_before_retry(attempt)

        if last_error is not None:
            raise last_error
        raise RuntimeError("backend request failed without error")

    async def _sleep_before_retry(self, attempt: int) -> None:
        if self.base_backoff_seconds <= 0:
            return
        await asyncio.sleep(self.base_backoff_seconds * (attempt + 1))

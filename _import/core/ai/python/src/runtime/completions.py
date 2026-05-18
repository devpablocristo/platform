"""Reusable JSON completion clients for application services."""

from __future__ import annotations

import contextvars
import hashlib
import json
import threading
import time
from contextlib import contextmanager
from dataclasses import dataclass
from math import ceil
from typing import Any, Protocol, TypeVar

import httpx
from pydantic import BaseModel
from tenacity import Retrying, stop_after_attempt, wait_exponential_jitter

from runtime.config.llm import normalize_provider, resolve_model_name
from runtime.logging import get_logger

HANDLED_LLM_TRANSPORT_ERRORS = (ValueError, TypeError, OSError)
HANDLED_LLM_CONTENT_ERRORS = (KeyError, IndexError, TypeError, AttributeError, ValueError)
HANDLED_HTTP_DETAIL_ERRORS = (ValueError, TypeError, KeyError)
SchemaT = TypeVar("SchemaT", bound=BaseModel)


class LLMError(RuntimeError):
    pass


class LLMRateLimitError(LLMError):
    pass


class LLMBudgetExceededError(LLMError):
    pass


@dataclass(frozen=True)
class LLMCompletion:
    provider: str
    model: str
    content: str
    raw: dict[str, Any] | None = None


class CompletionSettings(Protocol):
    llm_provider: str
    llm_model: str | None
    llm_api_key: str | None
    llm_base_url: str | None
    llm_timeout_ms: int
    llm_max_retries: int
    llm_max_output_tokens: int
    llm_max_calls_per_request: int
    llm_budget_tokens_per_request: int
    llm_rate_limit_rps: float


@dataclass
class _RequestBudgetState:
    calls_left: int
    tokens_left: int


class _GlobalRateLimiter:
    def __init__(self, rps: float) -> None:
        self.rps = max(0.1, float(rps))
        self._lock = threading.Lock()
        self._last_call_monotonic = 0.0

    def acquire(self) -> None:
        min_interval = 1.0 / self.rps
        now = time.monotonic()
        with self._lock:
            if now - self._last_call_monotonic < min_interval:
                raise LLMRateLimitError("llm_global_rate_limit_exceeded")
            self._last_call_monotonic = now


_REQUEST_BUDGET: contextvars.ContextVar[_RequestBudgetState | None] = contextvars.ContextVar(
    "llm_request_budget",
    default=None,
)


class JSONCompletionClient:
    def __init__(self, settings: CompletionSettings, *, logger_name: str = "runtime.llm") -> None:
        self.settings = settings
        self.logger = get_logger(logger_name)
        self.provider = normalize_provider(getattr(settings, "llm_provider", "stub"))
        self.model = resolve_model_name(self.provider, getattr(settings, "llm_model", None))
        self._rate_limiter = _GlobalRateLimiter(rps=float(getattr(self.settings, "llm_rate_limit_rps", 2.0)))

    @contextmanager
    def request_scope(self):
        state = _RequestBudgetState(
            calls_left=int(getattr(self.settings, "llm_max_calls_per_request", 3)),
            tokens_left=int(getattr(self.settings, "llm_budget_tokens_per_request", 2500)),
        )
        token = _REQUEST_BUDGET.set(state)
        try:
            yield
        finally:
            _REQUEST_BUDGET.reset(token)

    def complete_json(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        raise NotImplementedError

    def _enforce_limits(self, *, system_prompt: str, user_prompt: str) -> None:
        self._rate_limiter.acquire()
        budget = _REQUEST_BUDGET.get()
        is_scoped = budget is not None
        if budget is None:
            budget = _RequestBudgetState(
                calls_left=int(getattr(self.settings, "llm_max_calls_per_request", 3)),
                tokens_left=int(getattr(self.settings, "llm_budget_tokens_per_request", 2500)),
            )
        if budget.calls_left <= 0:
            raise LLMBudgetExceededError("llm_budget_calls_exceeded")
        estimated_tokens = _estimate_tokens(system_prompt) + _estimate_tokens(user_prompt)
        estimated_tokens += int(getattr(self.settings, "llm_max_output_tokens", 700))
        if budget.tokens_left < estimated_tokens:
            raise LLMBudgetExceededError("llm_budget_tokens_exceeded")
        budget.calls_left -= 1
        budget.tokens_left -= estimated_tokens
        if is_scoped:
            _REQUEST_BUDGET.set(budget)

    def _timeout(self) -> httpx.Timeout:
        timeout_seconds = max(float(getattr(self.settings, "llm_timeout_ms", 5000)) / 1000.0, 0.001)
        return httpx.Timeout(timeout_seconds)

    def _log_request(self, *, provider: str, system_prompt: str, user_prompt: str) -> None:
        digest = hashlib.sha256((system_prompt + "\n" + user_prompt).encode("utf-8")).hexdigest()[:12]
        self.logger.info(
            json.dumps(
                {
                    "event": "llm.request",
                    "provider": provider,
                    "model": self.model,
                    "prompt_sha": digest,
                    "system_len": len(system_prompt),
                    "user_len": len(user_prompt),
                }
            )
        )

    def _retrying(self, *, max_attempts: int | None = None, max_wait_seconds: float = 8.0) -> Retrying:
        attempt_limit = max(int(max_attempts or getattr(self.settings, "llm_max_retries", 3)), 1)
        return Retrying(
            stop=stop_after_attempt(attempt_limit),
            wait=wait_exponential_jitter(initial=0.5, max=max_wait_seconds),
            reraise=True,
        )


class StubLLMClient(JSONCompletionClient):
    def complete_json(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        self._enforce_limits(system_prompt=system_prompt, user_prompt=user_prompt)
        _ = system_prompt
        if "COPILOT_EXPLAIN_V2" in user_prompt:
            payload = {
                "human_readable": "Stub explanation for the requested analysis.",
                "audit_focused": "Stub technical explanation for audit and review.",
                "what_to_watch_next": "Stub follow-up items to monitor next.",
            }
        else:
            payload = {
                "classification": {"severity": "high", "actionability": "act", "confidence": 0.85},
                "decision_summary": {
                    "recommended_outcome": "propose_actions",
                    "primary_reason": "Stub: sufficient signal was detected to recommend an action plan.",
                },
                "proposed_plan": [
                    {
                        "step": 1,
                        "action": "Stub: gather more detail about the main contributing factor.",
                        "tool": "request_cost_breakdown",
                        "tool_args": {"feature": "cost_total", "time_window": "all"},
                        "rationale": "Stub: validate the main driver before executing changes.",
                        "reversible": True,
                    }
                ],
                "risks_and_uncertainties": ["Stub: output generated without a real LLM response."],
                "explanation": {
                    "human_readable": "Stub: summary of what happened and what is suggested.",
                    "audit_focused": "Stub: reasoning trace for review purposes.",
                    "what_to_watch_next": "Stub: metrics or events to watch next.",
                },
            }
        return LLMCompletion(provider="stub", model=self.model, content=json.dumps(payload), raw=None)


class OpenAIChatCompletionsClient(JSONCompletionClient):
    def __init__(self, settings: CompletionSettings, *, logger_name: str = "runtime.llm") -> None:
        super().__init__(settings, logger_name=logger_name)
        api_key = str(getattr(settings, "llm_api_key", "") or "").strip()
        if not api_key:
            raise ValueError("LLM_API_KEY is required when LLM_PROVIDER != stub")
        self.api_key = api_key
        self.base_url = str(getattr(settings, "llm_base_url", "") or "https://api.openai.com/v1").rstrip("/")

    def complete_json(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        retrying = self._retrying()
        for attempt in retrying:
            with attempt:
                return self._complete_json_once(system_prompt=system_prompt, user_prompt=user_prompt)
        raise LLMError("LLM response could not be obtained after retries")

    def _complete_json_once(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        self._enforce_limits(system_prompt=system_prompt, user_prompt=user_prompt)
        self._log_request(provider="openai", system_prompt=system_prompt, user_prompt=user_prompt)
        headers = {"Authorization": f"Bearer {self.api_key}"}
        body: dict[str, Any] = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
            "temperature": 0,
            "max_tokens": int(getattr(self.settings, "llm_max_output_tokens", 700)),
            "response_format": {"type": "json_object"},
        }

        try:
            with httpx.Client(timeout=self._timeout()) as client:
                response = client.post(f"{self.base_url}/chat/completions", headers=headers, json=body)
                response.raise_for_status()
                data = response.json()
        except httpx.HTTPStatusError as exc:
            detail = _safe_http_error_detail(exc.response)
            raise LLMError(f"LLM HTTP {exc.response.status_code}: {detail}") from exc
        except httpx.HTTPError as exc:
            raise LLMError(f"LLM HTTP error: {exc}") from exc
        except HANDLED_LLM_TRANSPORT_ERRORS as exc:
            raise LLMError(f"LLM error: {exc}") from exc

        try:
            content = data["choices"][0]["message"]["content"]
        except HANDLED_LLM_CONTENT_ERRORS as exc:
            raise LLMError("Invalid LLM response: missing choices[0].message.content") from exc

        return LLMCompletion(provider="openai", model=self.model, content=content, raw=data)


class GoogleAIStudioGenerateContentClient(JSONCompletionClient):
    def __init__(self, settings: CompletionSettings, *, logger_name: str = "runtime.llm") -> None:
        super().__init__(settings, logger_name=logger_name)
        api_key = str(getattr(settings, "llm_api_key", "") or "").strip()
        if not api_key:
            raise ValueError("LLM_API_KEY is required when LLM_PROVIDER != stub")
        self.api_key = api_key
        self.base_url = str(getattr(settings, "llm_base_url", "") or "https://generativelanguage.googleapis.com/v1beta").rstrip("/")

    def complete_json(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        retrying = self._retrying()
        for attempt in retrying:
            with attempt:
                return self._complete_json_once(system_prompt=system_prompt, user_prompt=user_prompt)
        raise LLMError("LLM response could not be obtained after retries")

    def _complete_json_once(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        self._enforce_limits(system_prompt=system_prompt, user_prompt=user_prompt)
        self._log_request(provider="google_ai_studio", system_prompt=system_prompt, user_prompt=user_prompt)
        headers = {"x-goog-api-key": self.api_key}
        model_path = self.model if self.model.startswith("models/") else f"models/{self.model}"
        body: dict[str, Any] = {
            "systemInstruction": {"parts": [{"text": system_prompt}]},
            "contents": [{"role": "user", "parts": [{"text": user_prompt}]}],
            "generationConfig": {
                "temperature": 0,
                "responseMimeType": "application/json",
                "maxOutputTokens": int(getattr(self.settings, "llm_max_output_tokens", 700)),
            },
        }

        try:
            with httpx.Client(timeout=self._timeout()) as client:
                response = client.post(f"{self.base_url}/{model_path}:generateContent", headers=headers, json=body)
                response.raise_for_status()
                data = response.json()
        except httpx.HTTPStatusError as exc:
            detail = _safe_http_error_detail(exc.response)
            raise LLMError(f"LLM HTTP {exc.response.status_code}: {detail}") from exc
        except httpx.HTTPError as exc:
            raise LLMError(f"LLM HTTP error: {exc}") from exc
        except HANDLED_LLM_TRANSPORT_ERRORS as exc:
            raise LLMError(f"LLM error: {exc}") from exc

        try:
            candidates = data.get("candidates") or []
            if not candidates:
                raise LLMError("Invalid LLM response: missing candidates[0]")
            parts = candidates[0]["content"].get("parts") or []
            texts = [part.get("text", "") for part in parts if isinstance(part, dict)]
            content = "".join(texts).strip()
            if not content:
                raise LLMError("Invalid LLM response: empty content")
        except KeyError as exc:
            raise LLMError("Invalid LLM response: missing candidates[0].content.parts[*].text") from exc

        return LLMCompletion(provider="google_ai_studio", model=self.model, content=content, raw=data)


class VertexGenerateContentClient(JSONCompletionClient):
    def __init__(self, settings: CompletionSettings, *, logger_name: str = "runtime.llm") -> None:
        super().__init__(settings, logger_name=logger_name)
        project = str(getattr(settings, "llm_project", "") or "").strip()
        if not project:
            raise ValueError("LLM_PROJECT is required for LLM_PROVIDER=vertex")
        location = str(getattr(settings, "llm_location", "") or "us-central1").strip() or "us-central1"
        self.project = project
        self.location = location

    def complete_json(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        retrying = self._retrying()
        for attempt in retrying:
            with attempt:
                return self._complete_json_once(system_prompt=system_prompt, user_prompt=user_prompt)
        raise LLMError("LLM response could not be obtained after retries")

    def _complete_json_once(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        from google import genai
        from google.genai import types

        self._enforce_limits(system_prompt=system_prompt, user_prompt=user_prompt)
        self._log_request(provider="vertex", system_prompt=system_prompt, user_prompt=user_prompt)

        try:
            client = genai.Client(vertexai=True, project=self.project, location=self.location)
            config = types.GenerateContentConfig(
                temperature=0,
                max_output_tokens=int(getattr(self.settings, "llm_max_output_tokens", 700)),
                response_mime_type="application/json",
                system_instruction=system_prompt,
            )
            response = client.models.generate_content(
                model=self.model,
                contents=user_prompt,
                config=config,
            )
        except HANDLED_LLM_TRANSPORT_ERRORS as exc:
            raise LLMError(f"LLM error: {exc}") from exc
        except Exception as exc:  # noqa: BLE001
            raise LLMError(f"LLM error: {exc}") from exc

        content = (getattr(response, "text", "") or "").strip()
        if not content:
            raise LLMError("Invalid LLM response: empty content")

        raw = getattr(response, "model_dump", lambda: {})() if hasattr(response, "model_dump") else {}
        return LLMCompletion(provider="vertex", model=self.model, content=content, raw=raw)


class OllamaChatClient(JSONCompletionClient):
    def __init__(self, settings: CompletionSettings, *, logger_name: str = "runtime.llm") -> None:
        super().__init__(settings, logger_name=logger_name)
        self.base_url = str(getattr(settings, "llm_base_url", "") or "http://localhost:11434").rstrip("/")

    def complete_json(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        retrying = self._retrying(max_attempts=1, max_wait_seconds=4.0)
        for attempt in retrying:
            with attempt:
                return self._complete_json_once(system_prompt=system_prompt, user_prompt=user_prompt)
        raise LLMError("LLM response could not be obtained after retries")

    def _complete_json_once(self, *, system_prompt: str, user_prompt: str) -> LLMCompletion:
        self._enforce_limits(system_prompt=system_prompt, user_prompt=user_prompt)
        timeout_seconds = min(float(getattr(self.settings, "llm_timeout_ms", 5000)) / 1000.0, 30.0)
        self._log_request(provider="ollama", system_prompt=system_prompt, user_prompt=user_prompt)
        body: dict[str, Any] = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
            "stream": False,
            "options": {"temperature": 0, "num_predict": int(getattr(self.settings, "llm_max_output_tokens", 700))},
            "format": "json",
        }

        try:
            with httpx.Client(timeout=httpx.Timeout(max(timeout_seconds, 0.001))) as client:
                response = client.post(f"{self.base_url}/api/chat", json=body)
                response.raise_for_status()
                data = response.json()
        except httpx.HTTPStatusError as exc:
            detail = _safe_http_error_detail(exc.response)
            raise LLMError(f"LLM HTTP {exc.response.status_code}: {detail}") from exc
        except httpx.HTTPError as exc:
            raise LLMError(f"LLM HTTP error: {exc}") from exc
        except HANDLED_LLM_TRANSPORT_ERRORS as exc:
            raise LLMError(f"LLM error: {exc}") from exc

        try:
            content = str((data.get("message") or {}).get("content") or "").strip()
            if not content:
                content = str(data.get("response") or "").strip()
            if not content:
                raise LLMError("Invalid LLM response: empty content")
        except HANDLED_LLM_CONTENT_ERRORS as exc:
            raise LLMError("Invalid LLM response: missing message.content/response") from exc

        return LLMCompletion(provider="ollama", model=self.model, content=content, raw=data)


def build_llm_client(settings: CompletionSettings, *, logger_name: str = "runtime.llm") -> JSONCompletionClient:
    provider = normalize_provider(getattr(settings, "llm_provider", "stub"))
    if provider == "stub":
        return StubLLMClient(settings, logger_name=logger_name)
    if provider == "openai":
        return OpenAIChatCompletionsClient(settings, logger_name=logger_name)
    if provider in {"google", "google_ai_studio", "gemini"}:
        return GoogleAIStudioGenerateContentClient(settings, logger_name=logger_name)
    if provider in {"vertex", "vertex_ai"}:
        return VertexGenerateContentClient(settings, logger_name=logger_name)
    if provider == "ollama":
        return OllamaChatClient(settings, logger_name=logger_name)
    raise ValueError(f"Unsupported LLM_PROVIDER: {getattr(settings, 'llm_provider', '')}")


def validate_json_completion(content: str, schema: type[SchemaT]) -> SchemaT:
    payload = json.loads(content)
    return schema.model_validate(payload)


def _safe_http_error_detail(response: httpx.Response) -> str:
    try:
        payload = response.json()
        if isinstance(payload, dict) and isinstance(payload.get("error"), dict):
            message = payload["error"].get("message")
            if isinstance(message, str) and message.strip():
                return message.strip()[:300]
    except HANDLED_HTTP_DETAIL_ERRORS:
        pass
    try:
        return (response.text or "").strip()[:300] or "unknown_error"
    except HANDLED_LLM_TRANSPORT_ERRORS:
        return "unknown_error"


def _estimate_tokens(text: str) -> int:
    if not text:
        return 0
    return max(1, ceil(len(text) / 4))

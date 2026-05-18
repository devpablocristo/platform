"""Reusable helpers for LLM runtime configuration."""

from __future__ import annotations

DEFAULT_MODELS: dict[str, str] = {
    "stub": "stub",
    "google": "gemini-flash-latest",
    "google_ai_studio": "gemini-flash-latest",
    "gemini": "gemini-flash-latest",
    "vertex": "gemini-2.0-flash",
    "vertex_ai": "gemini-2.0-flash",
    "ollama": "llama3.1",
    "openai": "gpt-4o-mini",
}

PROVIDERS_WITHOUT_API_KEY = frozenset({"stub", "ollama", "vertex", "vertex_ai"})


def normalize_provider(provider: str | None) -> str:
    normalized = str(provider or "stub").strip().lower()
    if not normalized:
        return "stub"
    return normalized


def default_model_for_provider(provider: str | None) -> str:
    normalized = normalize_provider(provider)
    return DEFAULT_MODELS.get(normalized, "gpt-4o-mini")


def resolve_model_name(provider: str | None, model: str | None) -> str:
    explicit_model = str(model or "").strip()
    if explicit_model:
        return explicit_model
    return default_model_for_provider(provider)


def provider_requires_api_key(provider: str | None) -> bool:
    return normalize_provider(provider) not in PROVIDERS_WITHOUT_API_KEY


def validate_provider_api_key(
    provider: str | None,
    api_key: str | None,
    *,
    enabled: bool = True,
    error_message: str = "LLM_API_KEY is required for the configured provider",
) -> None:
    if not enabled:
        return
    if not provider_requires_api_key(provider):
        return
    if str(api_key or "").strip():
        return
    raise ValueError(error_message)

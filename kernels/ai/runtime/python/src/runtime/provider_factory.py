"""Provider factory — extensible registry + simple create_provider helper."""

from __future__ import annotations

from typing import Any

from runtime.registry.provider_factory import ProviderFactory, ProviderFactoryError
from runtime.types import EchoProvider


def create_provider(settings: Any) -> Any:
    """Factory simple: crea un LLM provider a partir de settings.llm_provider.

    Soporta: echo, gemini. Extensible via ProviderFactory para más providers.
    """
    provider_name = getattr(settings, "llm_provider", "echo").strip().lower()

    if provider_name == "echo":
        return EchoProvider()

    if provider_name == "gemini":
        api_key = getattr(settings, "gemini_api_key", "").strip()
        if not api_key:
            raise ValueError("gemini_api_key is required for gemini provider")
        from runtime.providers.gemini import GeminiProvider

        model = getattr(settings, "gemini_model", "gemini-2.0-flash").strip()
        return GeminiProvider(api_key=api_key, model=model)

    if provider_name == "ollama":
        from runtime.providers.ollama import OllamaProvider

        base_url = getattr(settings, "ollama_base_url", "http://localhost:11434").strip()
        model = getattr(settings, "ollama_model", "llama3.2:1b").strip()
        timeout = float(getattr(settings, "ollama_timeout_seconds", 240.0))
        return OllamaProvider(base_url=base_url, model=model, timeout=timeout)

    raise ProviderFactoryError(f"unknown provider: {provider_name}")


__all__ = ["ProviderFactory", "ProviderFactoryError", "create_provider"]

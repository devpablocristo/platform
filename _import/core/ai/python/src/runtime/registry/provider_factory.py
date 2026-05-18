"""Provider registry for runtime composition."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from runtime.providers.base import LLMProvider

ProviderBuilder = Callable[..., LLMProvider]


class ProviderFactoryError(RuntimeError):
    """Raised when provider registration or resolution fails."""


class ProviderFactory:
    def __init__(self) -> None:
        self._builders: dict[str, ProviderBuilder] = {}

    def register(self, name: str, builder: ProviderBuilder) -> None:
        normalized = name.strip().lower()
        if not normalized:
            raise ProviderFactoryError("provider name is required")
        if normalized in self._builders:
            raise ProviderFactoryError(f"provider {normalized} already registered")
        self._builders[normalized] = builder

    def build(self, name: str, **kwargs: Any) -> LLMProvider:
        normalized = name.strip().lower()
        builder = self._builders.get(normalized)
        if builder is None:
            raise ProviderFactoryError(f"provider {normalized} is not registered")
        return builder(**kwargs)

    def names(self) -> list[str]:
        return sorted(self._builders)

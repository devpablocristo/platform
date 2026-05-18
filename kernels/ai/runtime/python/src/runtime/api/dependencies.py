"""Dependency providers for FastAPI adapters."""

from __future__ import annotations

from fastapi import Request

from runtime.auth import AuthSettings
from runtime.config.settings import APISettings
from runtime.rate_limit import RateLimitSettings


def get_settings(request: Request) -> APISettings:
    settings = getattr(request.app.state, "settings", None)
    if isinstance(settings, APISettings):
        return settings
    return APISettings()


def build_auth_settings(settings: APISettings) -> AuthSettings:
    return AuthSettings(allow_api_key=settings.allow_api_key)


def build_rate_limit_settings(settings: APISettings) -> RateLimitSettings:
    return RateLimitSettings(external_rpm=settings.external_rpm, internal_rpm=settings.internal_rpm)

"""FastAPI router for reusable runtime health endpoints."""

from __future__ import annotations

from fastapi import APIRouter, Depends

from runtime.api.dependencies import get_settings
from runtime.api.schemas import HealthResponse
from runtime.config.settings import APISettings

router = APIRouter()


@router.get("/healthz", response_model=HealthResponse)
async def healthz(settings: APISettings = Depends(get_settings)) -> HealthResponse:
    return HealthResponse(status="ok", service=settings.service_name, version=settings.api_version)


@router.get("/readyz", response_model=HealthResponse)
async def readyz(settings: APISettings = Depends(get_settings)) -> HealthResponse:
    return HealthResponse(status="ok", service=settings.service_name, version=settings.api_version)

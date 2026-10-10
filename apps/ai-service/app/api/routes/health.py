"""Liveness and readiness endpoints (handler layer)."""

from fastapi import APIRouter

from app import __version__
from app.services import health as health_service

router = APIRouter()


@router.get("/healthz")
def healthz() -> dict:
    """Liveness: the process is up. Never touches dependencies."""
    return {"status": "ok", "version": __version__}


@router.get("/readyz")
def readyz() -> dict:
    """Readiness: delegates to the health use case."""
    checks = health_service.readiness()
    status = "ready" if all(value == "ok" for value in checks.values()) else "not_ready"
    return {"status": status, "version": __version__, "checks": checks}

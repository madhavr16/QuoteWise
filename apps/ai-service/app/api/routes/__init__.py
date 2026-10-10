"""Route registration: liveness/readiness plus the reserved internal
API boundary (implemented in Milestone 3+)."""

from fastapi import APIRouter

from app.api.routes import health, internal

router = APIRouter()
router.include_router(health.router)
router.include_router(internal.router)

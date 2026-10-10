"""FastAPI application factory and lifecycle (server layer).

Milestone 1 scope: health/readiness endpoints, startup configuration
validation, and structured logging. Document parsing, extraction,
embeddings, and RAG arrive in later milestones (see
docs/implementation-plan.md).
"""

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI

from app import __version__
from app.api.routes import router as api_router
from app.config import get_settings
from app.logging_setup import configure_logging
from app.server.errors import register_exception_handlers

logger = logging.getLogger("quotewise.ai_service")


@asynccontextmanager
async def lifespan(_: FastAPI):
    settings = get_settings()
    logger.info(
        "ai_service_started (environment=%s, version=%s)",
        settings.app_env,
        __version__,
    )
    yield
    logger.info("ai_service_stopped")


def create_app() -> FastAPI:
    """Build the application. Fails fast on invalid configuration."""
    settings = get_settings()
    configure_logging(settings.log_level)

    app = FastAPI(
        title="QuoteWise AI Service",
        version=__version__,
        lifespan=lifespan,
    )
    app.include_router(api_router)
    register_exception_handlers(app)
    return app

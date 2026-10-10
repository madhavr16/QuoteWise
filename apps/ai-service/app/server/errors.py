"""Consistent error responses (server layer).

Mirrors the Go API error envelope (``{"error": {"code", "message"}}``)
so clients of both services parse errors identically. Messages are
sanitized; diagnostics belong in server-side logs.
"""

from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException as StarletteHTTPException

_STATUS_CODES = {
    400: "bad_request",
    401: "unauthorized",
    403: "forbidden",
    404: "not_found",
    405: "method_not_allowed",
    409: "conflict",
    422: "validation_error",
    429: "rate_limited",
    500: "internal_error",
    501: "not_implemented",
    503: "service_unavailable",
}


def error_body(code: str, message: str) -> dict:
    """Build the shared error envelope body."""
    return {"error": {"code": code, "message": message}}


def register_exception_handlers(app: FastAPI) -> None:
    """Attach consistent error handlers to the application."""

    @app.exception_handler(StarletteHTTPException)
    async def _http_exception(_: Request, exc: StarletteHTTPException) -> JSONResponse:
        code = _STATUS_CODES.get(exc.status_code, "error")
        return JSONResponse(
            status_code=exc.status_code,
            content=error_body(code, str(exc.detail)),
        )

    @app.exception_handler(RequestValidationError)
    async def _validation_error(_: Request, __: RequestValidationError) -> JSONResponse:
        # Field-level details stay in server-side logs; clients get a
        # stable, sanitized shape.
        return JSONResponse(
            status_code=422,
            content=error_body("validation_error", "Request validation failed."),
        )

    @app.exception_handler(Exception)
    async def _unhandled(_: Request, ___: Exception) -> JSONResponse:
        # The exception is still logged by the server; the client only
        # sees a sanitized body.
        return JSONResponse(
            status_code=500,
            content=error_body("internal_error", "An unexpected error occurred."),
        )

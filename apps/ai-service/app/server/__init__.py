"""Server layer: application factory, middleware, and error handlers.

Routes are registered from the API (handler) layer; use cases live in
app.services; configuration in app.config. This package wires the
layers together and contains no business rules.
"""

from app.server.app import create_app

__all__ = ["create_app"]

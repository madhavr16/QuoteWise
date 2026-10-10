"""Composition root: builds the application via the server layer.

Run with: uvicorn app.main:app --host 0.0.0.0 --port 8000
"""

from app.server.app import create_app

app = create_app()

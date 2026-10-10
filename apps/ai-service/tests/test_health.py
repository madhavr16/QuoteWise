"""Tests for health endpoints and the error envelope."""

from fastapi.testclient import TestClient

from app import __version__
from app.main import app

client = TestClient(app)


def test_healthz_is_ok():
    response = client.get("/healthz")
    assert response.status_code == 200
    body = response.json()
    assert body["status"] == "ok"
    assert body["version"] == __version__


def test_readyz_is_ready():
    response = client.get("/readyz")
    assert response.status_code == 200
    body = response.json()
    assert body["status"] == "ready"
    assert body["checks"]["config"] == "ok"


def test_internal_routes_are_501_stubs():
    for method in ("get", "post", "put", "patch", "delete"):
        response = getattr(client, method)("/internal/v1/documents/extract")
        assert response.status_code == 501
        assert response.json()["error"]["code"] == "not_implemented"


def test_unknown_route_returns_error_envelope():
    response = client.get("/does-not-exist")
    assert response.status_code == 404
    assert response.json()["error"]["code"] == "not_found"


def test_method_not_allowed_returns_error_envelope():
    response = client.post("/healthz")
    assert response.status_code == 405
    assert response.json()["error"]["code"] == "method_not_allowed"

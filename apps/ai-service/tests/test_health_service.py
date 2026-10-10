"""Service layer: health readiness use case."""

from app.services import health


def test_readiness_reports_config_ok():
    checks = health.readiness()
    assert checks == {"config": "ok"}

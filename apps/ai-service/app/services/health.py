"""Health/readiness use cases.

Milestone 1: readiness is "startup configuration was validated".
Later milestones add provider and storage checks as they become real
dependencies of this service.
"""


def readiness() -> dict[str, str]:
    """Return named readiness checks mapped to "ok"/"fail"."""
    return {"config": "ok"}

"""Tests for configuration validation."""

import pytest
from pydantic import ValidationError

from app.config import Settings


def test_defaults():
    settings = Settings(_env_file=None)
    assert settings.app_env == "development"
    assert settings.log_level == "info"


def test_invalid_app_env_is_rejected():
    with pytest.raises(ValidationError):
        Settings(_env_file=None, APP_ENV="staging")


def test_log_level_is_case_insensitive():
    settings = Settings(_env_file=None, LOG_LEVEL="INFO")
    assert settings.log_level == "info"

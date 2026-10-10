"""Validated service configuration, loaded from the environment.

The service fails fast at startup when configuration is invalid, so
misconfiguration surfaces before the service accepts traffic.
"""

from functools import lru_cache
from typing import Literal

from pydantic import field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Runtime settings for the AI service."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    app_env: Literal["development", "production", "test"] = "development"
    log_level: Literal["debug", "info", "warn", "error"] = "info"

    @field_validator("app_env", "log_level", mode="before")
    @classmethod
    def _lowercase(cls, value: object) -> object:
        """Accept APP_ENV/LOG_LEVEL in any case (e.g. compose values)."""
        if isinstance(value, str):
            return value.strip().lower()
        return value


@lru_cache
def get_settings() -> Settings:
    """Return the cached, validated settings instance."""
    return Settings()

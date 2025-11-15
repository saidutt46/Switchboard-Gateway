"""Configuration management for Admin API."""

from pydantic_settings import BaseSettings
from functools import lru_cache


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""
    
    # Application
    app_name: str = "Switchboard Admin API"
    app_version: str = "1.0.0"
    environment: str = "development"
    
    # Database
    database_url: str = "postgresql://switchboard:switchboard@localhost:5432/switchboard"
    
    # Redis
    redis_url: str = "redis://localhost:6379/0"
    
    # Server
    host: str = "0.0.0.0"
    port: int = 8000
    
    # CORS (for frontend later)
    cors_origins: list[str] = ["*"]
    
    # Logging
    log_level: str = "INFO"
    
    class Config:
        env_file = ".env"
        case_sensitive = False


@lru_cache()
def get_settings() -> Settings:
    """Get cached settings instance."""
    return Settings()
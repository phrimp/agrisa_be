import os
from typing import Optional
from pydantic import BaseSettings, Field


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""
    
    # API Configuration
    app_name: str = Field(default="Agrisa Satellite Data Service", env="APP_NAME")
    app_version: str = Field(default="1.0.0", env="APP_VERSION")
    debug: bool = Field(default=False, env="DEBUG")
    host: str = Field(default="0.0.0.0", env="HOST")
    port: int = Field(default=8000, env="PORT")
    
    # Google Earth Engine Configuration
    # Service account key file path (for authentication)
    gee_service_account_key: Optional[str] = Field(default=None, env="GEE_SERVICE_ACCOUNT_KEY")
    gee_project_id: str = Field(default="", env="GEE_PROJECT_ID")
    
    # Image Processing Configuration
    default_image_scale: int = Field(default=30, env="DEFAULT_IMAGE_SCALE")  # meters per pixel
    max_image_pixels: int = Field(default=10000000, env="MAX_IMAGE_PIXELS")  # 10M pixel limit
    
    # Cache Configuration
    cache_expiry_hours: int = Field(default=24, env="CACHE_EXPIRY_HOURS")
    
    # Coordinate Validation
    vietnam_bounds: dict = {
        "north": 23.393395,
        "south": 8.559611,
        "east": 109.469922,
        "west": 102.148224
    }
    
    # Logging
    log_level: str = Field(default="INFO", env="LOG_LEVEL")
    
    class Config:
        env_file = None  # Explicitly no env file as requested
        case_sensitive = False


# Global settings instance
settings = Settings()


def get_settings() -> Settings:
    """Get application settings."""
    return settings

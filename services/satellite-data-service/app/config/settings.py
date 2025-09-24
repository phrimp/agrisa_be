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
    
    # Database Configuration
    database_url: str = Field(
        default="postgresql+asyncpg://postgres:postgres@postgres:9406/agrisa",
        env="DATABASE_URL"
    )
    
    # MinIO Configuration
    minio_endpoint: str = Field(default="localhost:9000", env="MINIO_ENDPOINT")
    minio_access_key: str = Field(default="minioadmin", env="MINIO_ACCESS_KEY")
    minio_secret_key: str = Field(default="minioadmin", env="MINIO_SECRET_KEY")
    minio_secure: bool = Field(default=False, env="MINIO_SECURE")
    minio_bucket_name: str = Field(default="satellite-data", env="MINIO_BUCKET_NAME")
    
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
        env_file = None
        case_sensitive = False


# Global settings instance
settings = Settings()


def get_settings() -> Settings:
    """Get application settings."""
    return settings

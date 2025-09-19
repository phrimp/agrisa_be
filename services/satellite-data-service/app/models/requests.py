from typing import Optional
from pydantic import BaseModel, Field, validator
from datetime import datetime


class SatelliteImageRequest(BaseModel):
    """Request model for satellite image retrieval."""
    
    latitude: float = Field(..., description="Latitude coordinate", ge=-90, le=90)
    longitude: float = Field(..., description="Longitude coordinate", ge=-180, le=180)
    
    # Optional parameters
    start_date: Optional[str] = Field(
        default=None, 
        description="Start date for image filtering (YYYY-MM-DD format)"
    )
    end_date: Optional[str] = Field(
        default=None, 
        description="End date for image filtering (YYYY-MM-DD format)"
    )
    scale: Optional[int] = Field(
        default=30, 
        description="Image resolution in meters per pixel",
        ge=10, 
        le=1000
    )
    dimensions: Optional[str] = Field(
        default="512x512",
        description="Image dimensions in pixels (e.g., '512x512')"
    )
    
    @validator('start_date', 'end_date')
    def validate_date_format(cls, v):
        if v is not None:
            try:
                datetime.strptime(v, '%Y-%m-%d')
            except ValueError:
                raise ValueError('Date must be in YYYY-MM-DD format')
        return v
    
    @validator('dimensions')
    def validate_dimensions(cls, v):
        if v:
            try:
                width, height = v.split('x')
                width, height = int(width), int(height)
                if width <= 0 or height <= 0 or width > 2048 or height > 2048:
                    raise ValueError('Invalid dimensions')
            except (ValueError, AttributeError):
                raise ValueError('Dimensions must be in format "WIDTHxHEIGHT" (e.g., "512x512")')
        return v


class CoordinateValidationRequest(BaseModel):
    """Request model for coordinate validation."""
    
    latitude: float = Field(..., description="Latitude coordinate")
    longitude: float = Field(..., description="Longitude coordinate")



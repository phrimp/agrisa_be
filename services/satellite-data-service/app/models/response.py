from typing import Optional, Dict, Any, List
from pydantic import BaseModel
from datetime import datetime


class SatelliteImageResponse(BaseModel):
    """Response model for satellite image data."""
    
    success: bool
    message: str
    data: Optional[Dict[str, Any]] = None
    
    # Image information
    image_url: Optional[str] = None
    thumbnail_url: Optional[str] = None
    
    # Metadata
    coordinates: Dict[str, float]
    acquisition_date: Optional[str] = None
    satellite_source: Optional[str] = None
    cloud_cover: Optional[float] = None
    scale_meters: int
    dimensions: str
    
    # Processing info
    processing_time_ms: Optional[int] = None
    cached: bool = False


class ErrorResponse(BaseModel):
    """Standard error response model."""
    
    success: bool = False
    error_code: str
    message: str
    details: Optional[Dict[str, Any]] = None
    timestamp: datetime = datetime.utcnow()


class HealthCheckResponse(BaseModel):
    """Health check response model."""
    
    status: str
    service: str
    version: str
    timestamp: datetime
    dependencies: Dict[str, str]  # service_name -> status


class CoordinateValidationResponse(BaseModel):
    """Response model for coordinate validation."""
    
    valid: bool
    latitude: float
    longitude: float
    within_vietnam: bool
    message: Optional[str] = None

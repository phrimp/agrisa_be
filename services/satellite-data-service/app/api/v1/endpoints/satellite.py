from fastapi import APIRouter, HTTPException, Depends, Query
from typing import Optional
import logging

from app.models.requests import SatelliteImageRequest, CoordinateValidationRequest
from app.models.responses import (
    SatelliteImageResponse, 
    CoordinateValidationResponse, 
    HealthCheckResponse,
    ErrorResponse
)
from app.services.satellite_service import SatelliteService
from app.utils.validation import validate_coordinates, validate_vietnam_bounds
from datetime import datetime

logger = logging.getLogger(__name__)

router = APIRouter(
    prefix="/satellite",
    tags=["satellite"]
)

# Dependency injection for satellite service
def get_satellite_service() -> SatelliteService:
    return SatelliteService()


@router.get("/health", response_model=HealthCheckResponse)
async def health_check(
    satellite_service: SatelliteService = Depends(get_satellite_service)
) -> HealthCheckResponse:
    """Check health status of satellite service and dependencies."""
    try:
        dependencies = satellite_service.get_service_health()
        
        return HealthCheckResponse(
            status="healthy",
            service="Agrisa Satellite Data Service",
            version="1.0.0",
            timestamp=datetime.utcnow(),
            dependencies=dependencies
        )
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        raise HTTPException(status_code=500, detail=f"Health check failed: {str(e)}")


@router.post("/image", response_model=SatelliteImageResponse)
async def get_satellite_image(
    request: SatelliteImageRequest,
    satellite_service: SatelliteService = Depends(get_satellite_service)
) -> SatelliteImageResponse:
    """
    Get satellite image for specified coordinates.
    
    This endpoint retrieves the most recent cloud-free satellite image
    for the given coordinates using Google Earth Engine.
    """
    try:
        logger.info(f"Satellite image request for coordinates: {request.latitude}, {request.longitude}")
        
        # Get satellite image through service layer
        response = satellite_service.get_satellite_image(request)
        
        # Return response directly (service layer handles errors)
        return response
        
    except Exception as e:
        logger.error(f"Unexpected error in satellite image endpoint: {e}")
        raise HTTPException(
            status_code=500, 
            detail=f"Internal server error: {str(e)}"
        )


@router.get("/image", response_model=SatelliteImageResponse)
async def get_satellite_image_simple(
    latitude: float = Query(..., description="Latitude coordinate", ge=-90, le=90),
    longitude: float = Query(..., description="Longitude coordinate", ge=-180, le=180),
    start_date: Optional[str] = Query(None, description="Start date (YYYY-MM-DD)"),
    end_date: Optional[str] = Query(None, description="End date (YYYY-MM-DD)"),
    scale: int = Query(30, description="Resolution in meters per pixel", ge=10, le=1000),
    dimensions: str = Query("512x512", description="Image dimensions (e.g., '512x512')"),
    satellite_service: SatelliteService = Depends(get_satellite_service)
) -> SatelliteImageResponse:
    """
    Get satellite image using query parameters (simplified endpoint).
    
    This is a convenience endpoint that accepts parameters as query strings
    instead of requiring a JSON request body.
    """
    try:
        # Create request object from query parameters
        request = SatelliteImageRequest(
            latitude=latitude,
            longitude=longitude,
            start_date=start_date,
            end_date=end_date,
            scale=scale,
            dimensions=dimensions
        )
        
        logger.info(f"Simple satellite image request for coordinates: {latitude}, {longitude}")
        
        # Process through service layer
        response = satellite_service.get_satellite_image(request)
        
        return response
        
    except Exception as e:
        logger.error(f"Error in simple satellite image endpoint: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Failed to retrieve satellite image: {str(e)}"
        )


@router.post("/validate-coordinates", response_model=CoordinateValidationResponse)
async def validate_coordinates_endpoint(
    request: CoordinateValidationRequest
) -> CoordinateValidationResponse:
    """
    Validate if coordinates are within acceptable ranges and Vietnam boundaries.
    
    This endpoint can be used to check coordinates before making 
    satellite image requests.
    """
    try:
        # Basic coordinate validation
        is_valid, error_msg = validate_coordinates(request.latitude, request.longitude)
        
        if not is_valid:
            return CoordinateValidationResponse(
                valid=False,
                latitude=request.latitude,
                longitude=request.longitude,
                within_vietnam=False,
                message=error_msg
            )
        
        # Vietnam bounds validation
        within_vietnam, warning_msg = validate_vietnam_bounds(request.latitude, request.longitude)
        
        return CoordinateValidationResponse(
            valid=True,
            latitude=request.latitude,
            longitude=request.longitude,
            within_vietnam=within_vietnam,
            message=warning_msg if not within_vietnam else "Valid coordinates within Vietnam"
        )
        
    except Exception as e:
        logger.error(f"Error validating coordinates: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Coordinate validation failed: {str(e)}"
        )


@router.get("/regions", response_model=dict)
async def get_supported_regions(
    satellite_service: SatelliteService = Depends(get_satellite_service)
) -> dict:
    """
    Get information about supported regions and satellite capabilities.
    
    Returns information about Vietnam boundaries, supported satellites,
    and recommended image scales for different use cases.
    """
    try:
        return satellite_service.get_supported_regions()
    except Exception as e:
        logger.error(f"Error getting supported regions: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Failed to get region information: {str(e)}"
        )

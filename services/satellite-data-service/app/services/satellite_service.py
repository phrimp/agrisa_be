import logging
import time
from typing import Dict, Any, Optional
from app.services.earth_engine_service import EarthEngineService
from app.models.requests import SatelliteImageRequest
from app.models.responses import SatelliteImageResponse, ErrorResponse
from app.config.settings import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()


class SatelliteService:
    """Business logic service for satellite data operations."""
    
    def __init__(self):
        self.earth_engine = EarthEngineService()
    
    def validate_coordinates(self, latitude: float, longitude: float) -> Dict[str, Any]:
        """
        Validate if coordinates are within Vietnam boundaries.
        
        Args:
            latitude: Latitude coordinate
            longitude: Longitude coordinate
            
        Returns:
            Validation result dictionary
        """
        bounds = settings.vietnam_bounds
        
        within_vietnam = (
            bounds["south"] <= latitude <= bounds["north"] and
            bounds["west"] <= longitude <= bounds["east"]
        )
        
        return {
            "valid": True,  # Coordinates are valid numbers
            "within_vietnam": within_vietnam,
            "latitude": latitude,
            "longitude": longitude,
            "message": "Coordinates are outside Vietnam" if not within_vietnam else "Valid coordinates"
        }
    
    def get_satellite_image(self, request: SatelliteImageRequest) -> SatelliteImageResponse:
        """
        Get satellite image for the specified coordinates.
        
        Args:
            request: Satellite image request parameters
            
        Returns:
            Satellite image response with metadata
        """
        start_time = time.time()
        
        try:
            # Validate coordinates
            coord_validation = self.validate_coordinates(request.latitude, request.longitude)
            if not coord_validation["within_vietnam"]:
                logger.warning(f"Request for coordinates outside Vietnam: {request.latitude}, {request.longitude}")
            
            # Check if Earth Engine is available
            if not self.earth_engine.is_available():
                return SatelliteImageResponse(
                    success=False,
                    message="Google Earth Engine service is not available",
                    coordinates={"latitude": request.latitude, "longitude": request.longitude},
                    scale_meters=request.scale or settings.default_image_scale,
                    dimensions=request.dimensions or "512x512"
                )
            
            # Get satellite image from Earth Engine
            image_data = self.earth_engine.get_satellite_image(
                latitude=request.latitude,
                longitude=request.longitude,
                start_date=request.start_date,
                end_date=request.end_date,
                scale=request.scale or settings.default_image_scale,
                dimensions=request.dimensions or "512x512"
            )
            
            # Calculate processing time
            processing_time_ms = int((time.time() - start_time) * 1000)
            
            # Build response
            response = SatelliteImageResponse(
                success=True,
                message="Satellite image retrieved successfully",
                data={
                    "bounds": image_data.get("bounds"),
                    "properties": image_data.get("properties", {})
                },
                image_url=image_data.get("image_url"),
                coordinates={
                    "latitude": request.latitude, 
                    "longitude": request.longitude
                },
                acquisition_date=image_data.get("acquisition_date"),
                satellite_source=image_data.get("satellite_source"),
                cloud_cover=image_data.get("cloud_cover"),
                scale_meters=request.scale or settings.default_image_scale,
                dimensions=request.dimensions or "512x512",
                processing_time_ms=processing_time_ms,
                cached=False  # Basic version doesn't implement caching
            )
            
            logger.info(f"Successfully retrieved satellite image for {request.latitude}, {request.longitude}")
            return response
            
        except ValueError as e:
            logger.error(f"Validation error: {e}")
            return SatelliteImageResponse(
                success=False,
                message=f"Invalid request: {str(e)}",
                coordinates={"latitude": request.latitude, "longitude": request.longitude},
                scale_meters=request.scale or settings.default_image_scale,
                dimensions=request.dimensions or "512x512"
            )
            
        except Exception as e:
            logger.error(f"Error processing satellite image request: {e}")
            return SatelliteImageResponse(
                success=False,
                message=f"Failed to retrieve satellite image: {str(e)}",
                coordinates={"latitude": request.latitude, "longitude": request.longitude},
                scale_meters=request.scale or settings.default_image_scale,
                dimensions=request.dimensions or "512x512"
            )
    
    def get_service_health(self) -> Dict[str, Any]:
        """
        Get health status of the satellite service and its dependencies.
        
        Returns:
            Health status dictionary
        """
        earth_engine_status = self.earth_engine.test_connection()
        
        return {
            "satellite_service": "healthy",
            "google_earth_engine": earth_engine_status.get("status", "unknown"),
            "earth_engine_details": earth_engine_status
        }
    
    def get_supported_regions(self) -> Dict[str, Any]:
        """
        Get information about supported regions for satellite imagery.
        
        Returns:
            Supported regions information
        """
        return {
            "primary_region": "Vietnam",
            "bounds": settings.vietnam_bounds,
            "supported_satellites": [
                "Sentinel-2 (10-20m resolution, 5-day revisit)",
                "Landsat 8/9 (30m resolution, 16-day revisit)"
            ],
            "recommended_scales": {
                "agriculture": "10-30 meters",
                "regional_analysis": "30-100 meters",
                "overview": "100+ meters"
            }
        }

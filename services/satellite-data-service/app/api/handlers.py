from fastapi import APIRouter, HTTPException
from typing import Dict, Any, List
from app.services.google_earth_service import GoogleEarthEngineService
import logging

logger = logging.getLogger(__name__)

router = APIRouter()

@router.get("/satellite/public/gee/image_test")
async def test_gee_image() -> Dict[str, Any]:
    """
    Test Google Earth Engine service with hardcoded coordinates.
    Returns raw GEE response without processing.
    """
    try:
        # Hardcoded test coordinates (small area in Vietnam - WGS84)
        test_coordinates = [
            [105.8342, 21.0285],  # Hanoi area
            [105.8442, 21.0285],
            [105.8442, 21.0385],
            [105.8342, 21.0385],
            [105.8342, 21.0285]   # Close the polygon
        ]
        
        # Initialize service
        gee_service = GoogleEarthEngineService()
        
        # Call service with test parameters
        result = gee_service.get_satellite_image_for_farm(
            coordinates=test_coordinates,
            coordinate_crs="EPSG:4326",  # WGS84
            start_date="2025-01-01",
            end_date="2025-01-31",
            satellite="LANDSAT_8",
            max_cloud_cover=10.0
        )
        
        return {
            "status": "success",
            "message": "Google Earth Engine service test completed",
            "data": result
        }
        
    except Exception as e:
        logger.error(f"GEE test failed: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Google Earth Engine test failed: {str(e)}"
        )



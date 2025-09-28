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
            [106.6660, 11.8778],  # 11°52'40.1"N 106°39'57.6"E
            [106.6633, 11.8781],  # 11°52'41.1"N 106°39'48.0"E
            [106.6633, 11.8772],  # 11°52'37.9"N 106°39'47.8"E
            [106.6662, 11.8770],  # 11°52'37.1"N 106°39'58.3"E
            [106.6660, 11.8778],  # Close the polygon
        ]

        # Initialize service
        gee_service = GoogleEarthEngineService()

        # Call service with test parameters
        result = gee_service.get_farm_thumbnails(
            test_coordinates,
            "EPSG:4326",
            "2025-01-01",
            "2025-01-31",
            "LANDSAT_8",
            max_cloud_cover=10,
        )

        return {
            "status": "success",
            "message": "Google Earth Engine service test completed",
            "data": result,
        }

    except Exception as e:
        logger.error(f"GEE test failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"Google Earth Engine test failed: {str(e)}"
        )

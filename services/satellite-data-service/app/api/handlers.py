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
            [105.47811, 9.96866],   # 9°58'07.2"N 105°28'41.2"E
            [105.44447, 9.99925],   # 9°59'57.3"N 105°26'40.1"E
            [105.42661, 9.96794],   # 9°58'04.6"N 105°25'35.8"E
            [105.43919, 9.96033],   # 9°57'37.2"N 105°26'21.1"E
            [105.47811, 9.96866] 
        ]

        # Initialize service
        gee_service = GoogleEarthEngineService()

        # Call service with test parameters
        result = gee_service.get_farm_thumbnails(
            test_coordinates,
            "EPSG:4326",
            "2025-02-01",
            "2025-02-28",
            "SENTINEL_2",
            max_cloud_cover=90,
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

@router.get("/satellite/public/gee/dynamic_world_raw")
async def test_dynamic_world_raw() -> Dict[str, Any]:
    """
    Get raw Dynamic World data without processing for inspection.
    Returns unprocessed Google Earth Engine responses.
    """
    try:
        # Hardcoded test coordinates (agricultural area in Mekong Delta, Vietnam)
        test_coordinates = [
            [105.47811, 9.96866],   # 9°58'07.2"N 105°28'41.2"E
            [105.44447, 9.99925],   # 9°59'57.3"N 105°26'40.1"E
            [105.42661, 9.96794],   # 9°58'04.6"N 105°25'35.8"E
            [105.43919, 9.96033],   # 9°57'37.2"N 105°26'21.1"E
            [105.47811, 9.96866] 
        ]

        # Initialize service
        gee_service = GoogleEarthEngineService()

        # Call raw Dynamic World analysis
        result = gee_service.get_dynamic_world_raw_data(
            test_coordinates,
            "EPSG:4326",
            "2025-01-01",
            "2025-02-28",
            max_images=3  # Limit to 3 images to avoid timeout
        )

        return {
            "status": "success",
            "message": "Raw Dynamic World data retrieved successfully",
            "data": result,
        }

    except Exception as e:
        logger.error(f"Raw Dynamic World test failed: {e}")
        raise HTTPException(
            status_code=500, 
            detail=f"Raw Dynamic World data retrieval failed: {str(e)}"
        )

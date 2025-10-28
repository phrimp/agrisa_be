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
            "2025-06-01",
            "2025-06-30",
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

@router.get("/satellite/public/gee/image_test_sar")
async def test_gee_image_sar() -> Dict[str, Any]:
    """
    Test Google Earth Engine service with hardcoded coordinates.
    Returns raw GEE response without processing.
    """
    try:
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
            "2025-06-01",
            "2025-06-30",
            "SENTINEL_2",
            max_cloud_cover=90,
            force_sar_backup=True
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

@router.get("/satellite/public/ndvi")
async def get_ndvi(
    coordinates: str,
    start_date: str = "2024-01-01",
    end_date: str = "2024-12-31",
    max_cloud_cover: float = 30.0,
    max_images: int = 10,
    crs: str = "EPSG:4326",
    include_components: bool = False
) -> Dict[str, Any]:
    """
    Get NDVI (Normalized Difference Vegetation Index) data for an area.
    Returns list of ALL available images with individual NDVI statistics.

    Query Parameters:
        coordinates: JSON string of coordinates array, e.g. "[[105.47811,9.96866],[105.44447,9.99925],[105.42661,9.96794],[105.43919,9.96033],[105.47811,9.96866]]"
        start_date: Start date in 'YYYY-MM-DD' format (default: 2024-01-01)
        end_date: End date in 'YYYY-MM-DD' format (default: 2024-12-31)
        max_cloud_cover: Maximum cloud coverage percentage 0-100 (default: 30.0)
        max_images: Maximum number of images to return (default: 10)
        crs: Coordinate Reference System (default: EPSG:4326)
        include_components: Include raw component band data (B8-NIR, B4-Red) before calculation (default: False)

    Returns:
        List of all available images with NDVI statistics, thumbnails, and GeoTIFF download URLs.
        When include_components=true, also includes raw reflectance values for NIR and Red bands.

    Example:
        GET /satellite/public/ndvi?coordinates=[[105.47811,9.96866],[105.44447,9.99925],[105.42661,9.96794],[105.43919,9.96033],[105.47811,9.96866]]&start_date=2024-01-01&end_date=2024-12-31&max_images=5&include_components=true
    """
    try:
        # Parse coordinates from JSON string
        import json
        coords_list = json.loads(coordinates)

        if not isinstance(coords_list, list) or len(coords_list) < 3:
            raise ValueError("Coordinates must be a list of at least 3 points forming a polygon")

        # Initialize service
        gee_service = GoogleEarthEngineService()

        # Get NDVI data for ALL images
        result = gee_service.get_ndvi_data(
            coordinates=coords_list,
            coordinate_crs=crs,
            start_date=start_date,
            end_date=end_date,
            max_cloud_cover=max_cloud_cover,
            max_images=max_images,
            include_components=include_components
        )

        return {
            "status": "success",
            "message": f"NDVI data retrieved for {result['summary']['images_processed']} images",
            "data": result
        }

    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON coordinates: {e}")
        raise HTTPException(
            status_code=400,
            detail=f"Invalid coordinates format. Must be valid JSON array: {str(e)}"
        )
    except ValueError as e:
        logger.error(f"Invalid input: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"NDVI retrieval failed: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Failed to retrieve NDVI data: {str(e)}"
        )

@router.get("/satellite/public/ndmi")
async def get_ndmi(
    coordinates: str,
    start_date: str = "2024-01-01",
    end_date: str = "2024-12-31",
    max_cloud_cover: float = 30.0,
    max_images: int = 10,
    crs: str = "EPSG:4326",
    include_components: bool = False
) -> Dict[str, Any]:
    """
    Get NDMI (Normalized Difference Moisture Index) data for an area at 10m resolution.
    Returns list of ALL available images with individual NDMI statistics.

    Query Parameters:
        coordinates: JSON string of coordinates array, e.g. "[[105.47811,9.96866],[105.44447,9.99925],[105.42661,9.96794],[105.43919,9.96033],[105.47811,9.96866]]"
        start_date: Start date in 'YYYY-MM-DD' format (default: 2024-01-01)
        end_date: End date in 'YYYY-MM-DD' format (default: 2024-12-31)
        max_cloud_cover: Maximum cloud coverage percentage 0-100 (default: 30.0)
        max_images: Maximum number of images to return (default: 10)
        crs: Coordinate Reference System (default: EPSG:4326)
        include_components: Include raw component band data (B8-NIR, B11-SWIR) before calculation (default: False)

    Returns:
        List of all available images with NDMI statistics, moisture status, irrigation recommendations, thumbnails, and GeoTIFF download URLs.
        When include_components=true, also includes raw reflectance values for NIR and SWIR bands.

    Example:
        GET /satellite/public/ndmi?coordinates=[[105.47811,9.96866],[105.44447,9.99925],[105.42661,9.96794],[105.43919,9.96033],[105.47811,9.96866]]&start_date=2024-01-01&end_date=2024-12-31&max_images=5&include_components=true
    """
    try:
        # Parse coordinates from JSON string
        import json
        coords_list = json.loads(coordinates)

        if not isinstance(coords_list, list) or len(coords_list) < 3:
            raise ValueError("Coordinates must be a list of at least 3 points forming a polygon")

        # Initialize service
        gee_service = GoogleEarthEngineService()

        # Get NDMI data for ALL images
        result = gee_service.get_ndmi_data(
            coordinates=coords_list,
            coordinate_crs=crs,
            start_date=start_date,
            end_date=end_date,
            max_cloud_cover=max_cloud_cover,
            max_images=max_images,
            include_components=include_components
        )

        return {
            "status": "success",
            "message": f"NDMI data retrieved for {result['summary']['images_processed']} images",
            "data": result
        }

    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON coordinates: {e}")
        raise HTTPException(
            status_code=400,
            detail=f"Invalid coordinates format. Must be valid JSON array: {str(e)}"
        )
    except ValueError as e:
        logger.error(f"Invalid input: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"NDMI retrieval failed: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Failed to retrieve NDMI data: {str(e)}"
        )


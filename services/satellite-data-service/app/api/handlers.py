from fastapi import APIRouter, HTTPException, Query
from typing import Dict, Any, List, Optional
from app.services.google_earth_service import GoogleEarthEngineService
from app.services.gee_boundary_detection import GEEBoundaryDetectionService
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
            [105.47811, 9.96866],  # 9°58'07.2"N 105°28'41.2"E
            [105.44447, 9.99925],  # 9°59'57.3"N 105°26'40.1"E
            [105.42661, 9.96794],  # 9°58'04.6"N 105°25'35.8"E
            [105.43919, 9.96033],  # 9°57'37.2"N 105°26'21.1"E
            [105.47811, 9.96866],
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
            [105.47811, 9.96866],  # 9°58'07.2"N 105°28'41.2"E
            [105.44447, 9.99925],  # 9°59'57.3"N 105°26'40.1"E
            [105.42661, 9.96794],  # 9°58'04.6"N 105°25'35.8"E
            [105.43919, 9.96033],  # 9°57'37.2"N 105°26'21.1"E
            [105.47811, 9.96866],
        ]

        # Initialize service
        gee_service = GoogleEarthEngineService()

        # Call raw Dynamic World analysis
        result = gee_service.get_dynamic_world_raw_data(
            test_coordinates,
            "EPSG:4326",
            "2025-01-01",
            "2025-02-28",
            max_images=3,  # Limit to 3 images to avoid timeout
        )

        return {
            "status": "success",
            "message": "Raw Dynamic World data retrieved successfully",
            "data": result,
        }

    except Exception as e:
        logger.error(f"Raw Dynamic World test failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"Raw Dynamic World data retrieval failed: {str(e)}"
        )


@router.get("/satellite/public/gee/image_test_sar")
async def test_gee_image_sar() -> Dict[str, Any]:
    """
    Test Google Earth Engine service with hardcoded coordinates.
    Returns raw GEE response without processing.
    """
    try:
        test_coordinates = [
            [105.47811, 9.96866],  # 9°58'07.2"N 105°28'41.2"E
            [105.44447, 9.99925],  # 9°59'57.3"N 105°26'40.1"E
            [105.42661, 9.96794],  # 9°58'04.6"N 105°25'35.8"E
            [105.43919, 9.96033],  # 9°57'37.2"N 105°26'21.1"E
            [105.47811, 9.96866],
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
            force_sar_backup=True,
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


@router.get("/satellite/public/ndvi/batch")
async def get_ndvi_batch(
    coordinates: str,
    start_date: str = "2024-01-01",
    end_date: str = "2024-12-31",
    max_cloud_cover: float = 30.0,
    max_images: int = 10,
    crs: str = "EPSG:4326",
    include_components: bool = False,
) -> Dict[str, Any]:
    """
    Get NDVI data using BATCH PROCESSING (FASTEST - 20-30× faster).

    This endpoint uses Google Earth Engine server-side batch operations combined
    with thread pool parallelization for maximum performance.

    Performance comparison for 10 images:
    - Sequential: ~20-23 seconds
    - Parallel: ~2-3 seconds
    - Batch: ~0.5-1 second (THIS ENDPOINT)

    Args:
        coordinates: JSON string of [lon,lat] coordinates forming a closed polygon
        start_date: Start date (YYYY-MM-DD)
        end_date: End date (YYYY-MM-DD)
        max_cloud_cover: Maximum cloud cover percentage (0-100)
        max_images: Maximum number of images to return
        crs: Coordinate reference system (default: EPSG:4326)
        include_components: Include raw B8/B4 band statistics
    """
    try:
        import json

        coords_list = json.loads(coordinates)

        gee_service = GoogleEarthEngineService()
        result = await gee_service.get_ndvi_data_batched(
            coords_list,
            crs,
            start_date,
            end_date,
            max_cloud_cover,
            max_images,
            include_components,
        )

        return {"status": "success", "data": result}

    except json.JSONDecodeError:
        raise HTTPException(status_code=400, detail="Invalid coordinates JSON format")
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"NDVI batch calculation failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"NDVI batch calculation failed: {str(e)}"
        )


@router.get("/satellite/public/ndvi")
async def get_ndvi(
    coordinates: str,
    start_date: str = "2024-01-01",
    end_date: str = "2024-12-31",
    max_cloud_cover: float = 30.0,
    max_images: int = 10,
    crs: str = "EPSG:4326",
    include_components: bool = False,
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
            raise ValueError(
                "Coordinates must be a list of at least 3 points forming a polygon"
            )

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
            include_components=include_components,
        )

        return {
            "status": "success",
            "message": f"NDVI data retrieved for {result['summary']['images_processed']} images",
            "data": result,
        }

    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON coordinates: {e}")
        raise HTTPException(
            status_code=400,
            detail=f"Invalid coordinates format. Must be valid JSON array: {str(e)}",
        )
    except ValueError as e:
        logger.error(f"Invalid input: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"NDVI retrieval failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"Failed to retrieve NDVI data: {str(e)}"
        )


@router.get("/satellite/public/ndmi/batch")
async def get_ndmi_batch(
    coordinates: str,
    start_date: str = "2024-01-01",
    end_date: str = "2024-12-31",
    max_cloud_cover: float = 30.0,
    max_images: int = 10,
    crs: str = "EPSG:4326",
    include_components: bool = False,
) -> Dict[str, Any]:
    """
    Get NDMI data using BATCH PROCESSING (FASTEST - 20-30× faster).

    This endpoint uses Google Earth Engine server-side batch operations combined
    with thread pool parallelization for maximum performance.

    Performance comparison for 10 images:
    - Sequential: ~20-23 seconds
    - Parallel: ~2-3 seconds
    - Batch: ~0.5-1 second (THIS ENDPOINT)

    Args:
        coordinates: JSON string of [lon,lat] coordinates forming a closed polygon
        start_date: Start date (YYYY-MM-DD)
        end_date: End date (YYYY-MM-DD)
        max_cloud_cover: Maximum cloud cover percentage (0-100)
        max_images: Maximum number of images to return
        crs: Coordinate reference system (default: EPSG:4326)
        include_components: Include raw B8/B11 band statistics
    """
    try:
        import json

        coords_list = json.loads(coordinates)

        gee_service = GoogleEarthEngineService()
        result = await gee_service.get_ndmi_data_batched(
            coords_list,
            crs,
            start_date,
            end_date,
            max_cloud_cover,
            max_images,
            include_components,
        )

        return {"status": "success", "data": result}

    except json.JSONDecodeError:
        raise HTTPException(status_code=400, detail="Invalid coordinates JSON format")
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"NDMI batch calculation failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"NDMI batch calculation failed: {str(e)}"
        )


@router.get("/satellite/public/ndmi")
async def get_ndmi(
    coordinates: str,
    start_date: str = "2024-01-01",
    end_date: str = "2024-12-31",
    max_cloud_cover: float = 30.0,
    max_images: int = 10,
    crs: str = "EPSG:4326",
    include_components: bool = False,
) -> Dict[str, Any]:
    """
    Get NDMI (Normalized Difference Moisture Index) data for an area at 10m resolution.
    Returns list of ALL available images with individual NDMI statistics.

    NOW WITH PARALLEL PROCESSING: 8-10× faster than sequential processing.

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
            raise ValueError(
                "Coordinates must be a list of at least 3 points forming a polygon"
            )

        # Initialize service
        gee_service = GoogleEarthEngineService()

        # Get NDMI data for ALL images (async with parallel processing)
        result = await gee_service.get_ndmi_data(
            coordinates=coords_list,
            coordinate_crs=crs,
            start_date=start_date,
            end_date=end_date,
            max_cloud_cover=max_cloud_cover,
            max_images=max_images,
            include_components=include_components,
        )

        return {
            "status": "success",
            "message": f"NDMI data retrieved for {result['summary']['images_processed']} images",
            "data": result,
        }

    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON coordinates: {e}")
        raise HTTPException(
            status_code=400,
            detail=f"Invalid coordinates format. Must be valid JSON array: {str(e)}",
        )
    except ValueError as e:
        logger.error(f"Invalid input: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"NDMI retrieval failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"Failed to retrieve NDMI data: {str(e)}"
        )


# ===== PURE GEE BOUNDARY DETECTION ENDPOINTS =====


@router.get("/satellite/public/boundary/detect-from-point")
async def detect_boundary_from_point(
    latitude: float = Query(..., description="Point latitude (WGS84)", ge=-90, le=90),
    longitude: float = Query(
        ..., description="Point longitude (WGS84)", ge=-180, le=180
    ),
    buffer_distance: int = Query(
        500, description="Search radius in meters", ge=50, le=2000
    ),
    start_date: str = Query("2024-01-01", description="Start date (YYYY-MM-DD)"),
    end_date: str = Query("2024-12-31", description="End date (YYYY-MM-DD)"),
    max_cloud_cover: float = Query(
        30.0, description="Maximum cloud coverage (%)", ge=0, le=100
    ),
    ndvi_threshold: float = Query(
        0.4, description="NDVI threshold for vegetation", ge=0, le=1
    ),
    min_field_area: float = Query(
        0.1, description="Minimum field area (hectares)", ge=0.01, le=1000
    ),
) -> Dict[str, Any]:
    """
    **PURE GEE SOLUTION: Detect farm boundary from a single point coordinate.**

    This endpoint uses only Google Earth Engine (Sentinel-2 at 10m resolution)
    to automatically detect agricultural field boundaries. No third-party APIs required.

    **Algorithm:**
    1. Create buffer around input point
    2. Load cloud-free Sentinel-2 composite
    3. Calculate NDVI to identify vegetation
    4. Apply morphological operations for cleanup
    5. Connected component analysis to identify fields
    6. Extract field polygon containing the point
    7. Return GeoJSON boundary with visualizations

    **Query Parameters:**
    - `latitude`: Point latitude in WGS84 (required)
    - `longitude`: Point longitude in WGS84 (required)
    - `buffer_distance`: Search radius in meters (default: 500m, range: 50-2000m)
    - `start_date`: Start date for imagery (default: 2024-01-01)
    - `end_date`: End date for imagery (default: 2024-12-31)
    - `max_cloud_cover`: Maximum cloud coverage % (default: 30%, range: 0-100%)
    - `ndvi_threshold`: NDVI threshold for crops (default: 0.4, range: 0-1)
    - `min_field_area`: Minimum field area in hectares (default: 0.1, range: 0.01-1000)

    **Returns:**
    - **boundary**: GeoJSON Feature with detected field polygon
    - **area**: Field area in hectares
    - **confidence_score**: Detection confidence (0-1 scale)
    - **visualizations**: Natural color, NDVI, boundary outline thumbnail URLs
    - **vegetation_metrics**: NDVI statistics and interpretation

    **Example:**
    ```
    GET /satellite/public/boundary/detect-from-point?latitude=9.97&longitude=105.45&buffer_distance=500
    ```

    **Response Structure:**
    ```json
    {
      "status": "success",
      "data": {
        "boundary": {
          "type": "Feature",
          "geometry": {
            "type": "Polygon",
            "coordinates": [[[lon, lat], ...]]
          },
          "properties": {
            "area": {"value": 2.5, "unit": "hectares"},
            "confidence_score": {"value": 0.85, "interpretation": "High confidence"}
          }
        },
        "visualizations": {
          "natural_color": {"url": "...", "description": "..."},
          "ndvi": {"url": "...", "description": "..."},
          "boundary_outline": {"url": "...", "description": "..."}
        }
      }
    }
    ```
    """
    try:
        logger.info(
            f"Detecting boundary from point: ({latitude}, {longitude}), buffer: {buffer_distance}m"
        )

        # Initialize GEE boundary detection service
        gee_boundary_service = GEEBoundaryDetectionService()

        # Detect boundary
        result = gee_boundary_service.detect_farm_boundary_from_point(
            latitude=latitude,
            longitude=longitude,
            buffer_distance=buffer_distance,
            start_date=start_date,
            end_date=end_date,
            max_cloud_cover=max_cloud_cover,
            ndvi_threshold=ndvi_threshold,
            min_field_area=min_field_area,
        )

        return {
            "status": "success",
            "message": f"Boundary detected: {result['boundary']['properties']['area']['value']} hectares",
            "data": result,
        }

    except ValueError as e:
        logger.error(f"Invalid input or no boundary found: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Boundary detection failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"Failed to detect boundary: {str(e)}"
        )


@router.get("/satellite/public/boundary/detect-multiple")
async def detect_multiple_boundaries(
    north: float = Query(..., description="North bound latitude", ge=-90, le=90),
    south: float = Query(..., description="South bound latitude", ge=-90, le=90),
    east: float = Query(..., description="East bound longitude", ge=-180, le=180),
    west: float = Query(..., description="West bound longitude", ge=-180, le=180),
    start_date: str = Query("2024-01-01", description="Start date (YYYY-MM-DD)"),
    end_date: str = Query("2024-12-31", description="End date (YYYY-MM-DD)"),
    max_cloud_cover: float = Query(
        30.0, description="Maximum cloud coverage (%)", ge=0, le=100
    ),
    ndvi_threshold: float = Query(
        0.4, description="NDVI threshold for vegetation", ge=0, le=1
    ),
    min_field_area: float = Query(
        0.1, description="Minimum field area (hectares)", ge=0.01, le=1000
    ),
    max_fields: int = Query(50, description="Maximum fields to return", ge=1, le=500),
) -> Dict[str, Any]:
    """
    **PURE GEE SOLUTION: Detect all farm boundaries within a region.**

    Detects multiple agricultural field boundaries within a bounding box.
    Useful for regional agricultural mapping and analysis.

    **Query Parameters:**
    - `north`, `south`, `east`, `west`: Bounding box coordinates (WGS84) (required)
    - `start_date`: Start date for imagery (default: 2024-01-01)
    - `end_date`: End date for imagery (default: 2024-12-31)
    - `max_cloud_cover`: Maximum cloud coverage % (default: 30%)
    - `ndvi_threshold`: NDVI threshold for crops (default: 0.4)
    - `min_field_area`: Minimum field area in hectares (default: 0.1)
    - `max_fields`: Maximum fields to return (default: 50, max: 500)

    **Returns:**
    - **boundaries**: GeoJSON FeatureCollection with all detected fields
    - **summary**: Total fields count and total area
    - **visualization**: Thumbnail showing all boundaries

    **Example:**
    ```
    GET /satellite/public/boundary/detect-multiple?north=10.0&south=9.9&east=105.5&west=105.4
    ```
    """
    try:
        # Validate bounds
        if north <= south:
            raise ValueError("North bound must be greater than south bound")
        if east <= west:
            raise ValueError("East bound must be greater than west bound")

        logger.info(
            f"Detecting multiple boundaries in ROI: N{north} S{south} E{east} W{west}"
        )

        # Initialize service
        gee_boundary_service = GEEBoundaryDetectionService()

        # Detect boundaries
        result = gee_boundary_service.detect_multiple_boundaries_in_roi(
            north=north,
            south=south,
            east=east,
            west=west,
            start_date=start_date,
            end_date=end_date,
            max_cloud_cover=max_cloud_cover,
            ndvi_threshold=ndvi_threshold,
            min_field_area=min_field_area,
            max_fields=max_fields,
        )

        return {
            "status": "success",
            "message": f"Detected {result['summary']['total_fields']} fields, total area: {result['summary']['total_area']['value']} ha",
            "data": result,
        }

    except ValueError as e:
        logger.error(f"Invalid input: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Multiple boundary detection failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"Failed to detect boundaries: {str(e)}"
        )


@router.get("/satellite/public/boundary/imagery")
async def get_boundary_imagery(
    coordinates: str = Query(
        ...,
        description='Polygon coordinates as JSON array: "[[lon,lat],[lon,lat],...]"',
    ),
    start_date: str = Query("2024-01-01", description="Start date (YYYY-MM-DD)"),
    end_date: str = Query("2024-12-31", description="End date (YYYY-MM-DD)"),
    max_cloud_cover: float = Query(
        30.0, description="Maximum cloud coverage (%)", ge=0, le=100
    ),
    max_images: int = Query(
        None, description="Maximum number of images to return (default: unlimited - returns all images)", ge=1
    ),
    crs: str = Query("EPSG:4326", description="Coordinate reference system"),
    buffer_meters: float = Query(
        0.0,
        description="Buffer distance in meters to expand viewing area (0-5000m, 0=no buffer). Useful for small farms to show context.",
        ge=0,
        le=5000
    ),
) -> Dict[str, Any]:
    """
    **PURE GEE SOLUTION: Get natural color imagery for all images of farm boundary.**

    Get Sentinel-2 imagery (10m resolution) for a given farm boundary polygon.
    Returns ALL available images with ONLY natural color (RGB) visualization.

    **Query Parameters:**
    - `coordinates`: JSON string of polygon coordinates (required)
      Example: `"[[105.47811,9.96866],[105.44447,9.99925],[105.42661,9.96794],[105.47811,9.96866]]"`
    - `start_date`: Start date for imagery (default: 2024-01-01)
    - `end_date`: End date for imagery (default: 2024-12-31)
    - `max_cloud_cover`: Maximum cloud coverage % (default: 30%)
    - `max_images`: Maximum number of images to return (default: unlimited - returns ALL images)
    - `crs`: Coordinate reference system (default: EPSG:4326)
    - `buffer_meters`: Buffer distance in meters to expand viewing area (default: 0, range: 0-5000m)
      Use this for small farms where exact boundary creates too zoomed-in view.
      Recommended: 250-400m for farms < 0.1ha, 150-250m for 0.1-1ha, 75-150m for 1-5ha

    **Returns:**
    - **summary**: Total images found and processed
    - **farm_info**: Boundary polygon and area
    - **images**: Array of all images with:
      - image_index: Image order (0-based)
      - image_id: Google Earth Engine image ID
      - product_id: Sentinel-2 product ID
      - acquisition_date: Image capture date (YYYY-MM-DD)
      - cloud_cover: Cloud coverage percentage
      - visualization.natural_color: Natural color thumbnail URL

    **Example:**
    ```
    # Get all images (unlimited)
    GET /satellite/public/boundary/imagery?coordinates=[[105.47811,9.96866],[105.44447,9.99925],[105.42661,9.96794],[105.47811,9.96866]]

    # Limit to 5 images
    GET /satellite/public/boundary/imagery?coordinates=[[105.47811,9.96866],[105.44447,9.99925],[105.42661,9.96794],[105.47811,9.96866]]&max_images=5
    ```

    **Response Structure:**
    ```json
    {
      "status": "success",
      "data": {
        "summary": {
          "total_images": 5,
          "images_processed": 5
        },
        "farm_info": {
          "boundary": {...},
          "area": {"value": 2.5, "unit": "hectares"}
        },
        "images": [
          {
            "image_index": 0,
            "acquisition_date": "2024-06-15",
            "cloud_cover": {"value": 5.2, "unit": "percentage"},
            "visualization": {
              "natural_color": {
                "url": "https://earthengine.googleapis.com/...",
                "description": "Natural color (RGB) - 10m resolution"
              }
            }
          },
          ...
        ]
      }
    }
    ```
    """
    try:
        # Parse coordinates
        import json

        coords_list = json.loads(coordinates)

        if not isinstance(coords_list, list) or len(coords_list) < 3:
            raise ValueError(
                "Coordinates must be a list of at least 3 points forming a polygon"
            )

        logger.info(f"Getting imagery for {len(coords_list)} point boundary (all images, natural color only, buffer: {buffer_meters}m)")

        # Initialize service
        gee_boundary_service = GEEBoundaryDetectionService()

        # Get imagery for all images
        result = gee_boundary_service.get_farm_imagery_by_boundary(
            coordinates=coords_list,
            coordinate_crs=crs,
            start_date=start_date,
            end_date=end_date,
            max_cloud_cover=max_cloud_cover,
            max_images=max_images,
            buffer_meters=buffer_meters,
        )

        return {
            "status": "success",
            "message": f"Retrieved {result['summary']['images_processed']} images for {result['farm_info']['area']['value']} hectares",
            "data": result,
        }

    except json.JSONDecodeError as e:
        logger.error(f"Invalid JSON coordinates: {e}")
        raise HTTPException(
            status_code=400,
            detail=f"Invalid coordinates format. Must be valid JSON array: {str(e)}",
        )
    except ValueError as e:
        logger.error(f"Invalid input: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Imagery retrieval failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"Failed to retrieve imagery: {str(e)}"
        )

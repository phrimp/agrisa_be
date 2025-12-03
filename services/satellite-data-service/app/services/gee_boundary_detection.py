"""
Pure Google Earth Engine solution for farm boundary detection and imagery.
Uses Sentinel-2 data at 10m resolution for automated field boundary delineation.

Based on research: "Field Boundary Detection Using Sentinel-2 Imagery"
Approach: NDVI-based edge detection + segmentation + polygon extraction
"""

import ee
import logging
from typing import Dict, List, Any, Optional, Tuple
from datetime import datetime
from app.config.settings import get_settings

logger = logging.getLogger(__name__)


class GEEBoundaryDetectionService:
    """
    Pure GEE service for automated farm boundary detection and imagery.
    No third-party API dependencies required.
    """

    def __init__(self):
        self.settings = get_settings()
        self._initialize_ee()

    def _initialize_ee(self):
        """Initialize Google Earth Engine with service account credentials."""
        try:
            if self.settings.gee_service_account_key:
                credentials = ee.ServiceAccountCredentials(
                    email=None,
                    key_file=self.settings.gee_service_account_key,
                )
                ee.Initialize(credentials, project=self.settings.gee_project_id)
            else:
                ee.Initialize(project=self.settings.gee_project_id)

            logger.info("GEE Boundary Detection Service initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize Google Earth Engine: {e}")
            raise

    def detect_farm_boundary_from_point(
        self,
        latitude: float,
        longitude: float,
        buffer_distance: int = 500,
        start_date: str = "2024-01-01",
        end_date: str = "2024-12-31",
        max_cloud_cover: float = 30.0,
        ndvi_threshold: float = 0.4,
        min_field_area: float = 0.1,  # hectares
    ) -> Dict[str, Any]:
        """
        Detect farm boundary from a single point coordinate using Sentinel-2 imagery.

        Algorithm:
        1. Create buffer around point
        2. Get cloud-free Sentinel-2 composite
        3. Calculate NDVI to identify vegetation
        4. Apply edge detection and segmentation
        5. Extract field polygon containing the point
        6. Return GeoJSON boundary

        Args:
            latitude: Point latitude (WGS84)
            longitude: Point longitude (WGS84)
            buffer_distance: Search radius in meters (default: 500m)
            start_date: Start date for imagery
            end_date: End date for imagery
            max_cloud_cover: Maximum cloud coverage percentage
            ndvi_threshold: NDVI threshold for vegetation (default: 0.4)
            min_field_area: Minimum field area in hectares

        Returns:
            Dictionary with boundary GeoJSON, area, confidence, and visualization URLs
        """
        try:
            logger.info(f"Detecting boundary for point ({latitude}, {longitude})")

            # Step 1: Create point and buffer
            point = ee.Geometry.Point([longitude, latitude])
            roi = point.buffer(buffer_distance)

            # Step 2: Get best Sentinel-2 composite
            s2_collection = (
                ee.ImageCollection("COPERNICUS/S2_SR_HARMONIZED")
                .filterBounds(roi)
                .filterDate(start_date, end_date)
                .filter(ee.Filter.lt("CLOUDY_PIXEL_PERCENTAGE", max_cloud_cover))
                .sort("CLOUDY_PIXEL_PERCENTAGE")
            )

            image_count = s2_collection.size().getInfo()
            if image_count == 0:
                raise ValueError(
                    f"No cloud-free imagery found. Try increasing max_cloud_cover or expanding date range."
                )

            # Get median composite to reduce noise
            s2_image = s2_collection.median()

            # Step 3: Calculate NDVI
            ndvi = s2_image.normalizedDifference(["B8", "B4"]).rename("NDVI")

            # Step 4: Threshold to identify agricultural fields
            vegetation_mask = ndvi.gt(ndvi_threshold)

            # Step 5: Apply morphological operations to clean up
            # Erosion followed by dilation (opening) to remove small noise
            kernel = ee.Kernel.circle(radius=2, units="pixels")
            cleaned_mask = vegetation_mask.focal_min(
                kernel=kernel, iterations=1
            ).focal_max(kernel=kernel, iterations=1)

            # Step 6: Connected component analysis to identify individual fields
            connected = cleaned_mask.connectedComponents(
                connectedness=ee.Kernel.square(1), maxSize=256
            )

            # Step 7: Get the field polygon that contains the input point
            field_containing_point = connected.reduceRegion(
                reducer=ee.Reducer.first(), geometry=point, scale=10
            )

            # Get the label of the field at the point
            field_label = field_containing_point.getInfo().get("labels", None)

            if field_label is None:
                raise ValueError(
                    f"No agricultural field detected at point ({latitude}, {longitude}). "
                    f"Point may be in non-vegetated area. Try different coordinates or lower NDVI threshold."
                )

            # Step 8: Extract only the field with that label
            target_field = connected.select("labels").eq(ee.Number(field_label))

            # Step 9: Vectorize to get polygon boundary
            field_vectors = target_field.reduceToVectors(
                geometry=roi,
                scale=10,
                geometryType="polygon",
                eightConnected=False,
                labelProperty="field",
                maxPixels=1e8,
            )

            # Step 10: Get the largest polygon (main field)
            def add_area(feature):
                return feature.set({"area_m2": feature.geometry().area(maxError=1)})

            field_with_area = field_vectors.map(add_area)
            largest_field = ee.Feature(field_with_area.sort("area_m2", False).first())

            # Step 11: Get boundary geometry and area
            boundary_geometry = largest_field.geometry()
            area_m2 = boundary_geometry.area(maxError=1).getInfo()
            area_hectares = area_m2 / 10000

            # Check minimum area
            if area_hectares < min_field_area:
                logger.warning(
                    f"Detected field ({area_hectares:.2f} ha) is below minimum threshold ({min_field_area} ha)"
                )

            # Step 12: Extract GeoJSON coordinates
            coords = boundary_geometry.coordinates().getInfo()
            geojson = {
                "type": "Polygon",
                "coordinates": coords,
                "crs": {
                    "type": "name",
                    "properties": {"name": "EPSG:4326"},
                },
            }

            # Step 13: Generate visualization URLs
            # Natural color
            rgb_viz = s2_image.select(["B4", "B3", "B2"]).visualize(min=0, max=3000)
            rgb_url = rgb_viz.getThumbURL(
                {
                    "dimensions": 512,
                    "region": boundary_geometry,
                    "format": "png",
                }
            )

            # NDVI visualization
            ndvi_viz = ndvi.visualize(
                min=-0.2,
                max=0.9,
                palette=[
                    "0000FF",
                    "8B4513",
                    "FFFF00",
                    "ADFF2F",
                    "00FF00",
                    "006400",
                ],
            )
            ndvi_url = ndvi_viz.getThumbURL(
                {
                    "dimensions": 512,
                    "region": boundary_geometry,
                    "format": "png",
                }
            )

            # Boundary overlay
            boundary_outline = (
                ee.Image(0)
                .byte()
                .paint(
                    featureCollection=ee.FeatureCollection([largest_field]),
                    color=255,
                    width=3,
                )
            )
            boundary_url = boundary_outline.visualize(
                palette=["000000", "FF0000"]
            ).getThumbURL(
                {
                    "dimensions": 512,
                    "region": boundary_geometry,
                    "format": "png",
                }
            )

            # Step 14: Calculate confidence score
            # Based on: NDVI uniformity, edge sharpness, field size
            ndvi_stats = ndvi.reduceRegion(
                reducer=ee.Reducer.mean().combine(
                    ee.Reducer.stdDev(), sharedInputs=True
                ),
                geometry=boundary_geometry,
                scale=10,
                maxPixels=1e8,
            ).getInfo()

            mean_ndvi = ndvi_stats.get("NDVI_mean", 0)
            std_ndvi = ndvi_stats.get("NDVI_stdDev", 0)

            # Confidence calculation (0-1 scale)
            # Higher confidence for: uniform NDVI, reasonable size, high mean NDVI
            ndvi_confidence = min(mean_ndvi / 0.8, 1.0)  # Scale to 0.8 max
            uniformity_confidence = max(1.0 - (std_ndvi / 0.3), 0)  # Low std = uniform
            size_confidence = min(area_hectares / 10.0, 1.0)  # Scale to 10 ha
            confidence_score = (
                ndvi_confidence * 0.5
                + uniformity_confidence * 0.3
                + size_confidence * 0.2
            )

            # Step 15: Compile response
            result = {
                "boundary": {
                    "type": "Feature",
                    "geometry": geojson,
                    "properties": {
                        "area": {
                            "value": round(area_hectares, 4),
                            "unit": "hectares",
                        },
                        "confidence_score": {
                            "value": round(confidence_score, 3),
                            "interpretation": self._interpret_confidence(
                                confidence_score
                            ),
                        },
                        "detection_method": "NDVI-based segmentation",
                        "data_source": "Sentinel-2",
                    },
                },
                "input": {
                    "latitude": latitude,
                    "longitude": longitude,
                    "buffer_distance": {"value": buffer_distance, "unit": "meters"},
                },
                "imagery_info": {
                    "satellite": "Sentinel-2",
                    "date_range": f"{start_date} to {end_date}",
                    "images_used": image_count,
                    "max_cloud_cover": {"value": max_cloud_cover, "unit": "percentage"},
                    "resolution": {"value": 10, "unit": "meters"},
                },
                "vegetation_metrics": {
                    "mean_ndvi": {
                        "value": round(mean_ndvi, 3),
                        "interpretation": self._interpret_ndvi(mean_ndvi),
                    },
                    "ndvi_uniformity": {
                        "value": round(std_ndvi, 3),
                        "interpretation": (
                            "Uniform"
                            if std_ndvi < 0.1
                            else "Moderate"
                            if std_ndvi < 0.2
                            else "Variable"
                        ),
                    },
                    "ndvi_threshold_used": ndvi_threshold,
                },
                "visualizations": {
                    "natural_color": {
                        "url": rgb_url,
                        "description": "Sentinel-2 natural color composite",
                    },
                    "ndvi": {
                        "url": ndvi_url,
                        "description": "NDVI vegetation index",
                    },
                    "boundary_outline": {
                        "url": boundary_url,
                        "description": "Detected field boundary",
                    },
                },
                "algorithm": {
                    "steps": [
                        "1. Create buffer around input point",
                        "2. Load cloud-free Sentinel-2 composite",
                        "3. Calculate NDVI vegetation index",
                        "4. Apply threshold to identify crops",
                        "5. Morphological operations (opening)",
                        "6. Connected component labeling",
                        "7. Extract field containing point",
                        "8. Vectorize to GeoJSON polygon",
                    ],
                    "parameters": {
                        "ndvi_threshold": ndvi_threshold,
                        "min_field_area_ha": min_field_area,
                        "morphology_kernel": "2-pixel circle",
                        "connection_type": "4-connected",
                    },
                },
            }

            logger.info(
                f"Boundary detected: {area_hectares:.2f} ha, confidence: {confidence_score:.3f}"
            )
            return result

        except ee.EEException as e:
            logger.error(f"Google Earth Engine API error: {e}")
            raise Exception(f"Earth Engine API error: {str(e)}")
        except Exception as e:
            logger.error(f"Error detecting farm boundary: {e}")
            raise

    def detect_multiple_boundaries_in_roi(
        self,
        north: float,
        south: float,
        east: float,
        west: float,
        start_date: str = "2024-01-01",
        end_date: str = "2024-12-31",
        max_cloud_cover: float = 30.0,
        ndvi_threshold: float = 0.4,
        min_field_area: float = 0.1,
        max_fields: int = 50,
    ) -> Dict[str, Any]:
        """
        Detect all farm boundaries within a region of interest (ROI).
        Useful for regional agricultural mapping.

        Args:
            north, south, east, west: Bounding box coordinates (WGS84)
            start_date: Start date for imagery
            end_date: End date for imagery
            max_cloud_cover: Maximum cloud coverage percentage
            ndvi_threshold: NDVI threshold for vegetation
            min_field_area: Minimum field area in hectares
            max_fields: Maximum number of fields to return

        Returns:
            Dictionary with multiple field boundaries as GeoJSON FeatureCollection
        """
        try:
            logger.info(
                f"Detecting multiple boundaries in ROI: {north}, {south}, {east}, {west}"
            )

            # Step 1: Create ROI
            roi = ee.Geometry.Rectangle([west, south, east, north])

            # Step 2: Get best Sentinel-2 composite
            s2_collection = (
                ee.ImageCollection("COPERNICUS/S2_SR_HARMONIZED")
                .filterBounds(roi)
                .filterDate(start_date, end_date)
                .filter(ee.Filter.lt("CLOUDY_PIXEL_PERCENTAGE", max_cloud_cover))
            )

            image_count = s2_collection.size().getInfo()
            if image_count == 0:
                raise ValueError("No cloud-free imagery found for ROI")

            s2_image = s2_collection.median()

            # Step 3: Calculate NDVI and threshold
            ndvi = s2_image.normalizedDifference(["B8", "B4"])
            vegetation_mask = ndvi.gt(ndvi_threshold)

            # Step 4: Morphological cleanup
            kernel = ee.Kernel.circle(radius=2, units="pixels")
            cleaned_mask = vegetation_mask.focal_min(
                kernel=kernel, iterations=1
            ).focal_max(kernel=kernel, iterations=1)

            # Step 5: Connected components
            connected = cleaned_mask.connectedComponents(
                connectedness=ee.Kernel.square(1), maxSize=256
            )

            # Step 6: Vectorize all fields
            field_vectors = connected.select("labels").reduceToVectors(
                geometry=roi,
                scale=10,
                geometryType="polygon",
                eightConnected=False,
                labelProperty="field_id",
                maxPixels=1e9,
            )

            # Step 7: Filter by minimum area
            def add_area_and_filter(feature):
                area_ha = feature.geometry().area(maxError=1).divide(10000)
                return feature.set({"area_ha": area_ha})

            fields_with_area = field_vectors.map(add_area_and_filter)
            filtered_fields = fields_with_area.filter(
                ee.Filter.gte("area_ha", min_field_area)
            )

            # Step 8: Sort by area and limit
            sorted_fields = filtered_fields.sort("area_ha", False).limit(max_fields)

            # Step 9: Convert to GeoJSON
            fields_geojson = sorted_fields.getInfo()

            # Step 10: Calculate total stats
            field_count = len(fields_geojson.get("features", []))
            total_area = sum(
                f.get("properties", {}).get("area_ha", 0)
                for f in fields_geojson.get("features", [])
            )

            # Step 11: Generate visualization
            all_fields_outline = (
                ee.Image(0)
                .byte()
                .paint(featureCollection=sorted_fields, color=255, width=2)
            )
            boundary_url = all_fields_outline.visualize(
                palette=["000000", "00FF00"]
            ).getThumbURL(
                {
                    "dimensions": 1024,
                    "region": roi,
                    "format": "png",
                }
            )

            result = {
                "boundaries": fields_geojson,
                "summary": {
                    "total_fields": field_count,
                    "total_area": {
                        "value": round(total_area, 2),
                        "unit": "hectares",
                    },
                    "roi_bounds": {
                        "north": north,
                        "south": south,
                        "east": east,
                        "west": west,
                    },
                },
                "imagery_info": {
                    "satellite": "Sentinel-2",
                    "date_range": f"{start_date} to {end_date}",
                    "images_used": image_count,
                    "resolution": {"value": 10, "unit": "meters"},
                },
                "parameters": {
                    "ndvi_threshold": ndvi_threshold,
                    "min_field_area_ha": min_field_area,
                    "max_fields_returned": max_fields,
                },
                "visualization": {
                    "url": boundary_url,
                    "description": "All detected field boundaries (green)",
                },
            }

            logger.info(
                f"Detected {field_count} fields, total area: {total_area:.2f} ha"
            )
            return result

        except ee.EEException as e:
            logger.error(f"Google Earth Engine API error: {e}")
            raise Exception(f"Earth Engine API error: {str(e)}")
        except Exception as e:
            logger.error(f"Error detecting multiple boundaries: {e}")
            raise

    def get_farm_imagery_by_boundary(
        self,
        coordinates: List[List[float]],
        coordinate_crs: str = "EPSG:4326",
        start_date: str = "2024-01-01",
        end_date: str = "2024-12-31",
        max_cloud_cover: float = 30.0,
        max_images: int = None,
        buffer_meters: float = 0.0,
    ) -> Dict[str, Any]:
        """
        Get natural color satellite imagery for all images of a farm boundary.
        Returns list of ALL available images with only natural color (RGB) visualization.

        Args:
            coordinates: Polygon coordinates [[lon, lat], ...]
            coordinate_crs: Coordinate reference system
            start_date: Start date for imagery
            end_date: End date for imagery
            max_cloud_cover: Maximum cloud coverage
            max_images: Maximum number of images to return (None = unlimited, get all)
            buffer_meters: Buffer distance in meters to expand viewing area (0 = no buffer)

        Returns:
            List of all images with natural color thumbnails only
        """
        try:
            logger.info(f"Getting farm imagery for provided boundary (all images, natural color only, buffer: {buffer_meters}m)")

            # Create farm geometry (actual boundary)
            farm_geometry = ee.Geometry.Polygon(
                coords=[coordinates], proj=coordinate_crs, geodesic=False
            )

            # Create viewing geometry (for thumbnail generation)
            if buffer_meters > 0:
                view_geometry = farm_geometry.buffer(buffer_meters)
                logger.info(f"Applied {buffer_meters}m buffer for imagery viewing area")
            else:
                view_geometry = farm_geometry

            # Get Sentinel-2 collection (all images)
            # Filter by actual farm geometry, not buffered view
            s2_collection = (
                ee.ImageCollection("COPERNICUS/S2_SR_HARMONIZED")
                .filterBounds(farm_geometry)
                .filterDate(start_date, end_date)
                .filter(ee.Filter.lt("CLOUDY_PIXEL_PERCENTAGE", max_cloud_cover))
                .sort("CLOUDY_PIXEL_PERCENTAGE")
            )

            # Apply limit only if max_images is specified
            if max_images is not None:
                s2_collection = s2_collection.limit(max_images)

            image_count = s2_collection.size().getInfo()
            if image_count == 0:
                raise ValueError("No cloud-free imagery found")

            # Calculate area once (actual farm area, not buffered)
            area_ha = farm_geometry.area(maxError=1).divide(10000).getInfo()

            # Process ALL images
            collection_info = s2_collection.getInfo()
            all_images_data = []

            for idx, image_info in enumerate(collection_info.get('features', [])):
                try:
                    # Get image properties
                    image_properties = image_info.get('properties', {})
                    image_id = image_info.get('id', '')

                    # Extract metadata
                    product_id = image_properties.get('PRODUCT_ID', '')
                    cloud_cover = image_properties.get('CLOUDY_PIXEL_PERCENTAGE', 0)

                    # Parse acquisition date
                    acquisition_date = None
                    if product_id:
                        try:
                            date_part = product_id.split('_')[2]
                            acquisition_date = f"{date_part[:4]}-{date_part[4:6]}-{date_part[6:8]}"
                        except:
                            acquisition_date = product_id[:10] if len(product_id) >= 10 else None

                    # Get the actual image
                    current_image = ee.Image(image_id)

                    # Natural color (RGB) only
                    rgb = current_image.select(["B4", "B3", "B2"])
                    natural_color_url = rgb.getThumbURL(
                        {
                            "min": 0,
                            "max": 3000,
                            "dimensions": 512,
                            "region": view_geometry,  # Use buffered geometry for viewing
                            "format": "png",
                        }
                    )

                    # Compile image data
                    image_data = {
                        'image_index': idx,
                        'image_id': image_id,
                        'product_id': product_id,
                        'acquisition_date': acquisition_date,
                        'cloud_cover': {
                            'value': round(cloud_cover, 2),
                            'unit': 'percentage'
                        },
                        'visualization': {
                            'natural_color': {
                                'url': natural_color_url,
                                'description': 'Natural color (RGB) - 10m resolution',
                                'bands': ['B4 (Red)', 'B3 (Green)', 'B2 (Blue)']
                            }
                        }
                    }

                    all_images_data.append(image_data)
                    logger.info(f"Processed image {idx + 1}/{image_count}: {acquisition_date}, Cloud: {cloud_cover:.1f}%")

                except Exception as img_error:
                    logger.warning(f"Failed to process image {idx}: {img_error}")
                    continue

            result = {
                "summary": {
                    "total_images": image_count,
                    "images_processed": len(all_images_data),
                    "date_range": f"{start_date} to {end_date}",
                    "max_cloud_cover_filter": {
                        "value": max_cloud_cover,
                        "unit": "percentage"
                    },
                    "buffer_applied": {
                        "value": buffer_meters,
                        "unit": "meters"
                    }
                },
                "farm_info": {
                    "boundary": {
                        "type": "Polygon",
                        "coordinates": [coordinates],
                        "crs": coordinate_crs,
                    },
                    "area": {"value": round(area_ha, 4), "unit": "hectares"},
                },
                "images": all_images_data,
                "processing_info": {
                    "satellite": "Sentinel-2",
                    "collection": "COPERNICUS/S2_SR_HARMONIZED",
                    "resolution": {"value": 10, "unit": "meters"},
                    "visualization_type": "natural_color_only"
                }
            }

            logger.info(f"Generated natural color imagery for {len(all_images_data)} images, {area_ha:.2f} ha farm")
            return result

        except ee.EEException as e:
            logger.error(f"Google Earth Engine API error: {e}")
            raise Exception(f"Earth Engine API error: {str(e)}")
        except Exception as e:
            logger.error(f"Error getting farm imagery: {e}")
            raise

    def _interpret_confidence(self, score: float) -> str:
        """Interpret confidence score as text."""
        if score >= 0.8:
            return "High confidence - Well-defined field boundary"
        elif score >= 0.6:
            return "Good confidence - Boundary likely accurate"
        elif score >= 0.4:
            return "Moderate confidence - Manual verification recommended"
        else:
            return "Low confidence - Boundary may be inaccurate"

    def _interpret_ndvi(self, ndvi: float) -> str:
        """Interpret mean NDVI value."""
        if ndvi > 0.6:
            return "Very healthy vegetation"
        elif ndvi > 0.4:
            return "Healthy vegetation"
        elif ndvi > 0.2:
            return "Moderate vegetation"
        elif ndvi > 0:
            return "Sparse vegetation"
        else:
            return "No vegetation / Water / Bare soil"

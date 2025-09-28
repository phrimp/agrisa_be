import ee
import logging
from typing import Dict, List, Any, Optional
from app.config.settings import get_settings

logger = logging.getLogger(__name__)


class GoogleEarthEngineService:
    """Service for interacting with Google Earth Engine API."""

    def __init__(self):
        self.settings = get_settings()
        self._initialize_ee()

    def _initialize_ee(self):
        """Initialize Google Earth Engine with service account credentials."""
        try:
            if self.settings.gee_service_account_key:
                # Use service account key file
                credentials = ee.ServiceAccountCredentials(
                    email=None,  # Will be read from key file
                    key_file=self.settings.gee_service_account_key,
                )
                ee.Initialize(credentials, project=self.settings.gee_project_id)
            else:
                # Use default authentication (for development/testing)
                ee.Initialize(project=self.settings.gee_project_id)

            logger.info("Google Earth Engine initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize Google Earth Engine: {e}")
            raise

    def get_satellite_image_for_farm(
        self,
        coordinates: List[List[float]],
        coordinate_crs: str,
        start_date: str,
        end_date: str,
        satellite: str = "LANDSAT_8",
        max_cloud_cover: float = 20.0,
    ) -> Dict[str, Any]:
        """
        Get satellite image for a Vietnamese farm boundary.

        Args:
            coordinates: List of [x, y] coordinates forming a closed polygon
                        Format: [[x1, y1], [x2, y2], ..., [x1, y1]]
            coordinate_crs: Coordinate Reference System of input coordinates
                           Examples: "EPSG:4326" (WGS84), "EPSG:3405" (VN2000)
            start_date: Start date in 'YYYY-MM-DD' format
            end_date: End date in 'YYYY-MM-DD' format
            satellite: Satellite collection name (default: LANDSAT_8)
            max_cloud_cover: Maximum cloud coverage percentage (0-100)

        Returns:
            Dictionary containing complete satellite image information
        """
        try:
            logger.info(
                f"Getting satellite image for farm boundary from {start_date} to {end_date}"
            )
            logger.info(f"Input coordinates CRS: {coordinate_crs}")

            # Step 1: Create Earth Engine geometry with specified CRS
            farm_geometry = ee.Geometry.Polygon(
                coords=[coordinates], proj=coordinate_crs, geodesic=False
            )

            logger.info(
                f"Created farm geometry with {len(coordinates)} coordinates in {coordinate_crs}"
            )

            # Step 2: Define satellite collection
            collection_map = {
                "LANDSAT_8": "LANDSAT/LC08/C02/T1_L2",
                "LANDSAT_9": "LANDSAT/LC09/C02/T1_L2",
                "SENTINEL_2": "COPERNICUS/S2_SR_HARMONIZED",
            }

            if satellite not in collection_map:
                raise ValueError(
                    f"Unsupported satellite: {satellite}. Available: {list(collection_map.keys())}"
                )

            collection_id = collection_map[satellite]

            # Step 3: Filter image collection
            image_collection = (
                ee.ImageCollection(collection_id)
                .filterBounds(farm_geometry)
                .filterDate(start_date, end_date)
                .filter(ee.Filter.lt("CLOUD_COVER", max_cloud_cover))
                .sort("CLOUD_COVER")
            )

            # Step 4: Get the best image (least cloudy)
            image_count = image_collection.size().getInfo()
            if image_count == 0:
                raise ValueError(
                    f"No images found for the specified criteria. "
                    f"Try increasing cloud cover threshold or extending date range."
                )

            best_image = ee.Image(image_collection.first())

            # Step 5: Get comprehensive image information
            image_info = best_image.getInfo()
            image_properties = best_image.toDictionary().getInfo()

            # Debug: Log the structure of image_info to understand the data
            logger.info(f"Image info type: {type(image_info)}")
            logger.info(
                f"Image info keys: {list(image_info.keys()) if isinstance(image_info, dict) else 'Not a dict'}"
            )
            if isinstance(image_info, dict) and "bands" in image_info:
                logger.info(f"Bands type: {type(image_info['bands'])}")
                logger.info(f"Bands content: {image_info['bands']}")

            # Get native projection information
            native_projection = best_image.projection().getInfo()

            # Step 6: Calculate statistics for the farm area
            stats = best_image.reduceRegion(
                reducer=ee.Reducer.mean()
                .combine(ee.Reducer.stdDev(), sharedInputs=True)
                .combine(ee.Reducer.minMax(), sharedInputs=True),
                geometry=farm_geometry,
                scale=self.settings.default_image_scale,
                maxPixels=self.settings.max_image_pixels,
            ).getInfo()

            # Step 7: Generate download URL
            clipped_image = best_image.clip(farm_geometry)

            download_url = clipped_image.getDownloadURL(
                {
                    "scale": self.settings.default_image_scale,
                    "crs": coordinate_crs,
                    "region": farm_geometry,
                    "format": "GEO_TIFF",
                }
            )

            # Step 8: Safely extract band information
            bands_info = (
                image_info.get("bands", []) if isinstance(image_info, dict) else []
            )

            # Handle both dict and list formats for bands
            if isinstance(bands_info, dict):
                band_names = list(bands_info.keys())
                band_info = bands_info
            elif isinstance(bands_info, list):
                # If bands is a list of band objects
                band_names = []
                band_info = {}
                for i, band in enumerate(bands_info):
                    if isinstance(band, dict) and "id" in band:
                        band_name = band["id"]
                        band_names.append(band_name)
                        band_info[band_name] = band
                    else:
                        # Fallback for unknown band structure
                        band_name = f"band_{i}"
                        band_names.append(band_name)
                        band_info[band_name] = band
            else:
                logger.warning(f"Unknown bands format: {type(bands_info)}")
                band_names = []
                band_info = {}

            # Step 9: Compile complete response
            result = {
                "image_info": {
                    "id": image_info.get("id")
                    if isinstance(image_info, dict)
                    else None,
                    "type": image_info.get("type")
                    if isinstance(image_info, dict)
                    else None,
                    "version": image_info.get("version", 0)
                    if isinstance(image_info, dict)
                    else 0,
                    "properties": image_properties,
                    "bands_raw": bands_info,  # Include raw bands for debugging
                },
                "geometry": {
                    "type": "Polygon",
                    "coordinates": [coordinates],
                    "crs": coordinate_crs,
                },
                "image_id": image_info.get("id")
                if isinstance(image_info, dict)
                else "unknown",
                "satellite": satellite,
                "collection": collection_id,
                "acquisition_date": image_properties.get("DATE_ACQUIRED")
                or image_properties.get("SENSING_TIME", "").split("T")[0]
                if "SENSING_TIME" in image_properties
                else None,
                "cloud_cover": image_properties.get("CLOUD_COVER", 0),
                "bands": band_names,
                "band_info": band_info,
                "download_url": download_url,
                "statistics": stats,
                "projection_info": {
                    "input_crs": coordinate_crs,
                    "native_projection": native_projection,
                    "output_crs": coordinate_crs,
                },
                "processing_info": {
                    "scale_meters": self.settings.default_image_scale,
                    "max_pixels": self.settings.max_image_pixels,
                    "date_range": f"{start_date} to {end_date}",
                    "max_cloud_cover": max_cloud_cover,
                    "images_found": image_count,
                },
            }

            logger.info(f"Successfully retrieved satellite image: {result['image_id']}")
            logger.info(
                f"Cloud cover: {result['cloud_cover']}%, Bands: {len(result['bands'])}, Images available: {image_count}"
            )

            return result

        except ee.EEException as e:
            logger.error(f"Google Earth Engine API error: {e}")
            raise Exception(f"Earth Engine API error: {str(e)}")
        except Exception as e:
            logger.error(f"Error getting satellite image: {e}")
            raise

    def get_simple_satellite_data(
        self,
        coordinates: List[List[float]],
        start_date: str = "2024-01-01",
        end_date: str = "2024-12-31",
    ) -> Dict[str, Any]:
        """
        Simple function to get basic satellite data and see what's available.
        Returns raw data structure for inspection.

        Args:
            coordinates: List of [x, y] coordinates forming a closed polygon
            start_date: Start date in 'YYYY-MM-DD' format
            end_date: End date in 'YYYY-MM-DD' format

        Returns:
            Dictionary containing raw satellite data for inspection
        """
        try:
            logger.info("Getting simple satellite data for inspection")

            # Step 1: Create simple geometry
            farm_geometry = ee.Geometry.Polygon(coords=[coordinates], proj="EPSG:4326")

            # Step 2: Get Landsat 8 collection (most common)
            collection = (
                ee.ImageCollection("LANDSAT/LC08/C02/T1_L2")
                .filterBounds(farm_geometry)
                .filterDate(start_date, end_date)
                .filter(ee.Filter.lt("CLOUD_COVER", 50))
                .sort("CLOUD_COVER")
                .limit(5)
            )  # Get max 5 images

            # Step 3: Get collection info
            collection_info = collection.getInfo()
            image_count = collection.size().getInfo()

            # Step 4: Get first image if available
            first_image_data = None
            if image_count > 0:
                first_image = ee.Image(collection.first())
                first_image_data = first_image.getInfo()

                # Get band names
                band_names = first_image.bandNames().getInfo()

                # Get image properties
                properties = first_image.toDictionary().getInfo()

            # Step 5: Return everything for inspection
            result = {
                "basic_info": {
                    "coordinates_used": coordinates,
                    "date_range": f"{start_date} to {end_date}",
                    "images_found": image_count,
                    "collection_id": "LANDSAT/LC08/C02/T1_L2",
                },
                "collection_info": collection_info,
                "first_image_raw": first_image_data,
                "band_names": band_names if image_count > 0 else [],
                "image_properties": properties if image_count > 0 else {},
                "data_structure_info": {
                    "collection_type": type(collection_info).__name__,
                    "collection_keys": list(collection_info.keys())
                    if isinstance(collection_info, dict)
                    else "Not a dict",
                    "first_image_type": type(first_image_data).__name__
                    if first_image_data
                    else "No image",
                    "first_image_keys": list(first_image_data.keys())
                    if isinstance(first_image_data, dict)
                    else "Not a dict",
                },
            }

            logger.info(f"Found {image_count} images, returning raw data structure")
            return result

        except Exception as e:
            logger.error(f"Error getting simple satellite data: {e}")
            raise


    def get_farm_thumbnails(
        self,
        coordinates: List[List[float]],
        coordinate_crs: str = "EPSG:4326",
        start_date: str = "2024-01-01",
        end_date: str = "2024-12-31",
        satellite: str = "LANDSAT_8",
        max_cloud_cover: float = 30.0
    ) -> Dict[str, Any]:
        """
        Generate multiple thumbnail images for Vietnamese farms.
        Supports both Landsat 8 and Sentinel-2 satellites.
        
        Args:
            coordinates: List of [x, y] coordinates forming a closed polygon
            coordinate_crs: Coordinate Reference System
            start_date: Start date in 'YYYY-MM-DD' format
            end_date: End date in 'YYYY-MM-DD' format
            satellite: Satellite collection name ("LANDSAT_8" or "SENTINEL_2")
            max_cloud_cover: Maximum cloud coverage percentage
            
        Returns:
            Dictionary containing multiple thumbnail URLs and metadata
        """
        try:
            logger.info(f"Generating {satellite} farm thumbnail images")
            
            # Step 1: Create farm geometry
            farm_geometry = ee.Geometry.Polygon(
                coords=[coordinates], 
                proj=coordinate_crs,
                geodesic=False
            )
            
            # Step 2: Configure satellite-specific parameters
            if satellite == "LANDSAT_8":
                collection_id = "LANDSAT/LC08/C02/T1_L2"
                cloud_cover_prop = "CLOUD_COVER"
                
                # Landsat band configurations
                band_configs = {
                    'rgb': {
                        'bands': ['SR_B4', 'SR_B3', 'SR_B2'],
                        'min': 0.0, 'max': 0.3,
                        'description': 'Natural color (30m resolution)'
                    },
                    'nir': {
                        'bands': ['SR_B5', 'SR_B4', 'SR_B3'],
                        'min': 0.0, 'max': 0.3, 'gamma': [0.95, 1.1, 1.0],
                        'description': 'False color - vegetation appears red (30m resolution)'
                    },
                    'ndvi': {
                        'bands': ['SR_B5', 'SR_B4'],
                        'description': 'NDVI vegetation health (30m resolution)'
                    },
                    'agriculture': {
                        'bands': ['SR_B6', 'SR_B5', 'SR_B2'],
                        'min': 0.0, 'max': 0.3,
                        'description': 'Agriculture composite (30m resolution)'
                    }
                }
                
                # Metadata field names for Landsat
                metadata_fields = {
                    'image_id': 'LANDSAT_PRODUCT_ID',
                    'date': 'DATE_ACQUIRED',
                    'cloud': 'CLOUD_COVER',
                    'sun_elevation': 'SUN_ELEVATION'
                }
                
            elif satellite == "SENTINEL_2":
                collection_id = "COPERNICUS/S2_SR_HARMONIZED"
                cloud_cover_prop = "CLOUDY_PIXEL_PERCENTAGE"
                
                # Sentinel-2 band configurations
                band_configs = {
                    'rgb': {
                        'bands': ['B4', 'B3', 'B2'],
                        'min': 0, 'max': 3000,
                        'description': 'Natural color (10m resolution)'
                    },
                    'nir': {
                        'bands': ['B8', 'B4', 'B3'],
                        'min': 0, 'max': 3000, 'gamma': [0.95, 1.1, 1.0],
                        'description': 'False color - vegetation appears red (10m resolution)'
                    },
                    'ndvi': {
                        'bands': ['B8', 'B4'],
                        'description': 'NDVI vegetation health (10m resolution)'
                    },
                    'agriculture': {
                        'bands': ['B11', 'B8', 'B2'],
                        'min': 0, 'max': 3000,
                        'description': 'Agriculture composite (20m/10m resolution)'
                    }
                }
                
                # Metadata field names for Sentinel-2
                metadata_fields = {
                    'image_id': 'PRODUCT_ID',
                    'date': 'PRODUCT_ID',  # Date extracted from PRODUCT_ID
                    'cloud': 'CLOUDY_PIXEL_PERCENTAGE',
                    'sun_elevation': 'MEAN_SOLAR_ZENITH_ANGLE'
                }
                
            else:
                raise ValueError(f"Unsupported satellite: {satellite}. Use 'LANDSAT_8' or 'SENTINEL_2'")
            
            # Step 3: Filter and get best image
            image_collection = (ee.ImageCollection(collection_id)
                              .filterBounds(farm_geometry)
                              .filterDate(start_date, end_date)
                              .filter(ee.Filter.lt(cloud_cover_prop, max_cloud_cover))
                              .sort(cloud_cover_prop))
            
            image_count = image_collection.size().getInfo()
            logger.info(f"Found {image_count} {satellite} images matching criteria")
            
            if image_count == 0:
                raise ValueError(f"No {satellite} images found for the specified criteria. "
                               f"Try increasing cloud cover threshold or expanding date range.")
            
            best_image = ee.Image(image_collection.first())
            
            # Step 4: Apply satellite-specific preprocessing
            if satellite == "LANDSAT_8":
                # Apply scaling factors for Landsat Collection 2 Level-2
                optical_bands = best_image.select('SR_B.').multiply(0.0000275).add(-0.2)
                best_image = best_image.addBands(optical_bands, None, True)
            # Sentinel-2 doesn't need scaling - surface reflectance values are already scaled
            
            # Step 5: Create thumbnail visualizations
            thumbnails = {}
            
            # Natural Color (RGB) Thumbnail
            rgb_config = band_configs['rgb']
            rgb_image = best_image.select(rgb_config['bands'])
            thumbnails['natural_color'] = {
                'url': rgb_image.getThumbURL({
                    'bands': rgb_config['bands'],
                    'min': rgb_config['min'],
                    'max': rgb_config['max'],
                    'dimensions': 512,
                    'region': farm_geometry,
                    'format': 'png'
                }),
                'description': rgb_config['description'],
                'bands': rgb_config['bands']
            }
            
            # False Color (NIR) Thumbnail
            nir_config = band_configs['nir']
            nir_image = best_image.select(nir_config['bands'])
            nir_params = {
                'bands': nir_config['bands'],
                'min': nir_config['min'],
                'max': nir_config['max'],
                'dimensions': 512,
                'region': farm_geometry,
                'format': 'png'
            }
            if 'gamma' in nir_config:
                nir_params['gamma'] = nir_config['gamma']
                
            thumbnails['false_color'] = {
                'url': nir_image.getThumbURL(nir_params),
                'description': nir_config['description'],
                'bands': nir_config['bands']
            }
            
            # NDVI Thumbnail (vegetation health)
            ndvi_config = band_configs['ndvi']
            ndvi = best_image.normalizedDifference(ndvi_config['bands'])
            thumbnails['ndvi'] = {
                'url': ndvi.getThumbURL({
                    'min': -1,
                    'max': 1,
                    'palette': [
                        'FFFFFF', 'CE7E45', 'DF923D', 'F1B555', 'FCD163',
                        '99B718', '74A901', '66A000', '529400', '3E8601',
                        '207401', '056201', '004C00', '023B01', '012E01',
                        '011D01', '011301'
                    ],
                    'dimensions': 512,
                    'region': farm_geometry,
                    'format': 'png'
                }),
                'description': ndvi_config['description'],
                'bands': ndvi_config['bands'],
                'interpretation': {
                    'high_values': 'Healthy vegetation (0.6 to 1.0)',
                    'medium_values': 'Moderate vegetation (0.2 to 0.6)', 
                    'low_values': 'Bare soil/water (-1.0 to 0.2)'
                }
            }
            
            # Agriculture Composite
            agri_config = band_configs['agriculture']
            agri_image = best_image.select(agri_config['bands'])
            thumbnails['agriculture'] = {
                'url': agri_image.getThumbURL({
                    'bands': agri_config['bands'],
                    'min': agri_config['min'],
                    'max': agri_config['max'],
                    'dimensions': 512,
                    'region': farm_geometry,
                    'format': 'png'
                }),
                'description': agri_config['description'],
                'bands': agri_config['bands']
            }
            
            # Step 6: Get image metadata
            image_properties = best_image.toDictionary().getInfo()
            
            # Extract metadata using satellite-specific field names
            image_id = image_properties.get(metadata_fields['image_id'], 'unknown')
            
            # Handle date extraction
            if satellite == "SENTINEL_2" and metadata_fields['date'] == 'PRODUCT_ID':
                # Extract date from Sentinel-2 PRODUCT_ID format
                product_id = image_properties.get('PRODUCT_ID', '')
                if len(product_id) > 15:
                    try:
                        date_part = product_id.split('_')[2]  # YYYYMMDD format
                        acquisition_date = f"{date_part[:4]}-{date_part[4:6]}-{date_part[6:8]}"
                    except:
                        acquisition_date = None
                else:
                    acquisition_date = None
            else:
                acquisition_date = image_properties.get(metadata_fields['date'])
            
            cloud_cover = image_properties.get(metadata_fields['cloud'], 0)
            sun_elevation = image_properties.get(metadata_fields['sun_elevation'], 0)
            
            # Convert solar zenith to elevation for Sentinel-2
            if satellite == "SENTINEL_2" and sun_elevation > 0:
                sun_elevation = 90 - sun_elevation  # Convert zenith to elevation
            
            # Step 7: Create farm boundary thumbnail
            boundary_image = ee.Image().byte().paint(
                featureCollection=ee.FeatureCollection([ee.Feature(farm_geometry)]),
                color=1,
                width=3
            )
            
            thumbnails['farm_boundary'] = {
                'url': boundary_image.getThumbURL({
                    'palette': ['FF0000'],  # Red boundary
                    'dimensions': 512,
                    'region': farm_geometry,
                    'format': 'png'
                }),
                'description': 'Farm boundary outline',
                'bands': ['constant']
            }
            
            # Step 8: Calculate farm area with error margin
            try:
                area_hectares = farm_geometry.area(maxError=1).divide(10000).getInfo()
            except Exception as area_error:
                logger.warning(f"Could not calculate area: {area_error}")
                area_hectares = None
            
            # Step 9: Compile response
            result = {
                'farm_info': {
                    'coordinates': coordinates,
                    'crs': coordinate_crs,
                    'area_approx_hectares': area_hectares
                },
                'image_info': {
                    'satellite': satellite,
                    'collection_id': collection_id,
                    'image_id': image_id,
                    'acquisition_date': acquisition_date,
                    'cloud_cover': cloud_cover,
                    'sun_elevation': sun_elevation
                },
                'thumbnails': thumbnails,
                'satellite_comparison': {
                    'landsat_8': '30m resolution, 16-day revisit, good regional coverage',
                    'sentinel_2': '10m resolution, 5-day revisit, excellent detail',
                    'current_satellite': f"{satellite} - {band_configs['rgb']['description'].split('(')[1]}"
                },
                'usage_instructions': {
                    'web_display': 'Use thumbnail URLs directly in <img> tags',
                    'mobile_display': 'Load URLs in ImageView/Image components',
                    'caching': 'URLs are temporary - cache images if needed for offline use',
                    'dimensions': 'All thumbnails are 512px (largest dimension)',
                    'format': 'PNG with transparency support'
                },
                'processing_info': {
                    'date_range': f"{start_date} to {end_date}",
                    'images_found': image_count,
                    'max_cloud_cover': max_cloud_cover,
                    'cloud_filter_property': cloud_cover_prop
                }
            }
            
            logger.info(f"Generated {len(thumbnails)} {satellite} thumbnail images")
            return result
            
        except Exception as e:
            logger.error(f"Error generating {satellite} thumbnails: {e}")
            raise
        

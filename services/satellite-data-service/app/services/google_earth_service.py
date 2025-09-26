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
                    key_file=self.settings.gee_service_account_key
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
        max_cloud_cover: float = 20.0
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
            Dictionary containing complete satellite image information:
            - image_info: Full image metadata and properties
            - geometry: Farm boundary geometry
            - image_id: Google Earth Engine image ID
            - acquisition_date: Image acquisition date
            - cloud_cover: Cloud coverage percentage
            - bands: Available spectral bands information
            - download_url: URL for downloading the clipped image
            - statistics: Image statistics for the farm area
            - projection_info: Original and output projection information
        
        Raises:
            ValueError: If coordinates are invalid or no images found
            Exception: If Google Earth Engine API call fails
        """
        try:
            logger.info(f"Getting satellite image for farm boundary from {start_date} to {end_date}")
            logger.info(f"Input coordinates CRS: {coordinate_crs}")
            
            # Step 1: Create Earth Engine geometry with specified CRS
            # GEE will automatically handle coordinate system conversion
            farm_geometry = ee.Geometry.Polygon(
                coords=[coordinates], 
                proj=coordinate_crs,  # Specify input coordinate system
                geodesic=False
            )
            
            logger.info(f"Created farm geometry with {len(coordinates)} coordinates in {coordinate_crs}")
            
            # Step 2: Define satellite collection
            collection_map = {
                "LANDSAT_8": "LANDSAT/LC08/C02/T1_L2",
                "LANDSAT_9": "LANDSAT/LC09/C02/T1_L2", 
                "SENTINEL_2": "COPERNICUS/S2_SR_HARMONIZED"
            }
            
            if satellite not in collection_map:
                raise ValueError(f"Unsupported satellite: {satellite}. Available: {list(collection_map.keys())}")
            
            collection_id = collection_map[satellite]
            
            # Step 3: Filter image collection
            image_collection = (ee.ImageCollection(collection_id)
                              .filterBounds(farm_geometry)
                              .filterDate(start_date, end_date)
                              .filter(ee.Filter.lt('CLOUD_COVER', max_cloud_cover))
                              .sort('CLOUD_COVER'))
            
            # Step 4: Get the best image (least cloudy)
            image_count = image_collection.size().getInfo()
            if image_count == 0:
                raise ValueError(f"No images found for the specified criteria. "
                               f"Try increasing cloud cover threshold or extending date range.")
            
            best_image = ee.Image(image_collection.first())
            
            # Step 5: Get comprehensive image information
            image_info = best_image.getInfo()
            image_properties = best_image.toDictionary().getInfo()
            
            # Get native projection information
            native_projection = best_image.projection().getInfo()
            
            # Step 6: Calculate statistics for the farm area
            # Let GEE handle the coordinate conversion automatically
            stats = best_image.reduceRegion(
                reducer=ee.Reducer.mean().combine(
                    ee.Reducer.stdDev(), sharedInputs=True
                ).combine(
                    ee.Reducer.minMax(), sharedInputs=True
                ),
                geometry=farm_geometry,
                scale=self.settings.default_image_scale,
                maxPixels=self.settings.max_image_pixels
            ).getInfo()
            
            # Step 7: Generate download URL
            clipped_image = best_image.clip(farm_geometry)
            
            download_url = clipped_image.getDownloadURL({
                'scale': self.settings.default_image_scale,
                'crs': coordinate_crs,  # Output in same CRS as input
                'region': farm_geometry,
                'format': 'GEO_TIFF'
            })
            
            # Step 8: Compile complete response
            result = {
                'image_info': {
                    'id': image_info.get('id'),
                    'type': image_info.get('type'),
                    'version': image_info.get('version', 0),
                    'properties': image_properties,
                    'bands': image_info.get('bands', {})
                },
                'geometry': {
                    'type': 'Polygon',
                    'coordinates': [coordinates],
                    'crs': coordinate_crs
                },
                'image_id': image_info.get('id'),
                'satellite': satellite,
                'collection': collection_id,
                'acquisition_date': image_properties.get('DATE_ACQUIRED') or 
                                  image_properties.get('SENSING_TIME', '').split('T')[0] if 'SENSING_TIME' in image_properties else None,
                'cloud_cover': image_properties.get('CLOUD_COVER', 0),
                'bands': list(image_info.get('bands', {}).keys()) if 'bands' in image_info else [],
                'band_info': image_info.get('bands', {}),
                'download_url': download_url,
                'statistics': stats,
                'projection_info': {
                    'input_crs': coordinate_crs,
                    'native_projection': native_projection,
                    'output_crs': coordinate_crs
                },
                'processing_info': {
                    'scale_meters': self.settings.default_image_scale,
                    'max_pixels': self.settings.max_image_pixels,
                    'date_range': f"{start_date} to {end_date}",
                    'max_cloud_cover': max_cloud_cover,
                    'images_found': image_count
                }
            }
            
            logger.info(f"Successfully retrieved satellite image: {result['image_id']}")
            logger.info(f"Cloud cover: {result['cloud_cover']}%, Bands: {len(result['bands'])}, Images available: {image_count}")
            
            return result
            
        except ee.EEException as e:
            logger.error(f"Google Earth Engine API error: {e}")
            raise Exception(f"Earth Engine API error: {str(e)}")
        except Exception as e:
            logger.error(f"Error getting satellite image: {e}")
            raise

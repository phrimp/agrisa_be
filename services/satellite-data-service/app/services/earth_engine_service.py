import ee
import json
import logging
from typing import Optional, Dict, Any, Tuple
from datetime import datetime, timedelta
from app.config.settings import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()


class EarthEngineService:
    """Service for Google Earth Engine integration."""
    
    def __init__(self):
        self._initialized = False
        self._authenticate_and_initialize()
    
    def _authenticate_and_initialize(self) -> None:
        """Authenticate with Google Earth Engine and initialize the API."""
        try:
            if settings.gee_service_account_key:
                # Use service account authentication
                credentials = ee.ServiceAccountCredentials(
                    email=None,  # Will be read from key file
                    key_file=settings.gee_service_account_key
                )
                ee.Initialize(credentials, project=settings.gee_project_id)
            else:
                # Use default authentication (requires gcloud auth)
                ee.Initialize(project=settings.gee_project_id)
            
            self._initialized = True
            logger.info("Google Earth Engine initialized successfully")
            
        except Exception as e:
            logger.error(f"Failed to initialize Google Earth Engine: {e}")
            self._initialized = False
    
    def is_available(self) -> bool:
        """Check if Earth Engine is properly initialized."""
        return self._initialized
    
    def get_satellite_image(
        self,
        latitude: float,
        longitude: float,
        start_date: Optional[str] = None,
        end_date: Optional[str] = None,
        scale: int = 30,
        dimensions: str = "512x512"
    ) -> Dict[str, Any]:
        """
        Retrieve satellite image for given coordinates.
        
        Args:
            latitude: Latitude coordinate
            longitude: Longitude coordinate  
            start_date: Start date for image filtering (YYYY-MM-DD)
            end_date: End date for image filtering (YYYY-MM-DD)
            scale: Image resolution in meters per pixel
            dimensions: Image dimensions (e.g., "512x512")
            
        Returns:
            Dictionary containing image URL and metadata
        """
        if not self._initialized:
            raise RuntimeError("Google Earth Engine not initialized")
        
        try:
            # Set default date range if not provided (last 30 days)
            if not end_date:
                end_date = datetime.now().strftime('%Y-%m-%d')
            if not start_date:
                start_date = (datetime.now() - timedelta(days=30)).strftime('%Y-%m-%d')
            
            # Create point geometry
            point = ee.Geometry.Point([longitude, latitude])
            
            # Get Sentinel-2 image collection
            collection = (ee.ImageCollection('COPERNICUS/S2_SR_HARMONIZED')
                         .filterBounds(point)
                         .filterDate(start_date, end_date)
                         .filter(ee.Filter.lt('CLOUDY_PIXEL_PERCENTAGE', 20))
                         .sort('CLOUDY_PIXEL_PERCENTAGE'))
            
            # Get the best image (least cloudy)
            image = collection.first()
            
            if image is None:
                # Fallback to Landsat 8/9 if no Sentinel-2 available
                collection = (ee.ImageCollection('LANDSAT/LC08/C02/T1_L2')
                             .merge(ee.ImageCollection('LANDSAT/LC09/C02/T1_L2'))
                             .filterBounds(point)
                             .filterDate(start_date, end_date)
                             .filter(ee.Filter.lt('CLOUD_COVER', 20))
                             .sort('CLOUD_COVER'))
                
                image = collection.first()
                satellite_source = "Landsat 8/9"
                vis_params = {
                    'bands': ['SR_B4', 'SR_B3', 'SR_B2'],
                    'min': 0,
                    'max': 0.3,
                    'gamma': 1.2
                }
            else:
                satellite_source = "Sentinel-2"
                vis_params = {
                    'bands': ['B4', 'B3', 'B2'],
                    'min': 0,
                    'max': 3000,
                    'gamma': 1.2
                }
            
            if image is None:
                raise ValueError("No suitable satellite images found for the specified location and date range")
            
            # Parse dimensions
            width, height = map(int, dimensions.split('x'))
            
            # Create visualization image
            vis_image = image.visualize(**vis_params)
            
            # Get image URL
            image_url = vis_image.getThumbURL({
                'region': point.buffer(scale * max(width, height) / 2).bounds(),
                'dimensions': f'{width}x{height}',
                'format': 'png'
            })
            
            # Get image metadata
            image_info = image.getInfo()
            properties = image_info.get('properties', {})
            
            # Extract metadata
            acquisition_date = self._extract_date_from_properties(properties, satellite_source)
            cloud_cover = self._extract_cloud_cover(properties, satellite_source)
            
            return {
                'image_url': image_url,
                'satellite_source': satellite_source,
                'acquisition_date': acquisition_date,
                'cloud_cover': cloud_cover,
                'scale_meters': scale,
                'bounds': point.buffer(scale * max(width, height) / 2).bounds().getInfo(),
                'properties': properties
            }
            
        except Exception as e:
            logger.error(f"Error retrieving satellite image: {e}")
            raise
    
    def _extract_date_from_properties(self, properties: Dict, satellite_source: str) -> Optional[str]:
        """Extract acquisition date from image properties."""
        try:
            if satellite_source == "Sentinel-2":
                date_str = properties.get('PRODUCT_ID', '')
                if date_str and len(date_str) > 7:
                    # Extract date from PRODUCT_ID format: S2A_MSIL2A_20231201T...
                    date_part = date_str.split('_')[2][:8]  # Extract YYYYMMDD
                    return f"{date_part[:4]}-{date_part[4:6]}-{date_part[6:8]}"
            elif satellite_source == "Landsat 8/9":
                date_acquired = properties.get('DATE_ACQUIRED')
                if date_acquired:
                    return date_acquired
            
            # Fallback: try system:time_start
            time_start = properties.get('system:time_start')
            if time_start:
                timestamp = int(time_start) / 1000  # Convert from milliseconds
                return datetime.fromtimestamp(timestamp).strftime('%Y-%m-%d')
                
        except Exception as e:
            logger.warning(f"Could not extract acquisition date: {e}")
        
        return None
    
    def _extract_cloud_cover(self, properties: Dict, satellite_source: str) -> Optional[float]:
        """Extract cloud cover percentage from image properties."""
        try:
            if satellite_source == "Sentinel-2":
                return properties.get('CLOUDY_PIXEL_PERCENTAGE')
            elif satellite_source == "Landsat 8/9":
                return properties.get('CLOUD_COVER')
        except Exception as e:
            logger.warning(f"Could not extract cloud cover: {e}")
        
        return None
    
    def test_connection(self) -> Dict[str, Any]:
        """Test Google Earth Engine connection."""
        try:
            if not self._initialized:
                return {'status': 'error', 'message': 'Not initialized'}
            
            # Simple test: get info about a dataset
            dataset = ee.ImageCollection('COPERNICUS/S2_SR_HARMONIZED')
            info = dataset.limit(1).getInfo()
            
            return {
                'status': 'ok',
                'message': 'Connected to Google Earth Engine',
                'test_dataset_features': len(info.get('features', []))
            }
            
        except Exception as e:
            return {
                'status': 'error',
                'message': f'Connection test failed: {str(e)}'
            }

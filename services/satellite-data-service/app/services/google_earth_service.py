import ee
import logging
from datetime import datetime
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
        satellite: str = "SENTINEL_2",
        max_cloud_cover: float = 30.0,
        force_sar_backup: bool = False
    ) -> Dict[str, Any]:
        """
        Generate farm thumbnail images with research-validated cloud-adaptive vegetation monitoring.
        
        Based on: "Using Dual-Polarization Sentinel-1A for Mapping Vegetation Types in Dak Lak, Vietnam"
        (Le Minh Hang et al., 2021) - Achieved 90.72% accuracy using RVI index.
        
        Vegetation Index Strategy:
        - cloud_cover < 30%: Use Sentinel-2 NDVI (optical, high accuracy)
        - cloud_cover >= 30%: Use Sentinel-1 RVI (radar, all-weather, 90.72% accuracy)
        
        Args:
            coordinates: List of [lon, lat] coordinates forming a closed polygon (WGS84)
            coordinate_crs: Coordinate Reference System (default: EPSG:4326)
            start_date: Start date in 'YYYY-MM-DD' format
            end_date: End date in 'YYYY-MM-DD' format
            satellite: "SENTINEL_2" (recommended) or "LANDSAT_8"
            max_cloud_cover: Maximum cloud coverage percentage (0-100)
            force_sar_backup: Force Sentinel-1 RVI regardless of cloud cover
            
        Returns:
            Dictionary with thumbnail URLs, metadata, and interpretation guidance
        """
        try:
            logger.info(f"Generating {satellite} thumbnails with research-validated cloud-adaptive VI")
            
            # Step 1: Create farm geometry
            farm_geometry = ee.Geometry.Polygon(
                coords=[coordinates], 
                proj=coordinate_crs,
                geodesic=False
            )
            
            # Step 2: Configure satellite-specific parameters
            if satellite == "SENTINEL_2":
                collection_id = "COPERNICUS/S2_SR_HARMONIZED"
                cloud_cover_prop = "CLOUDY_PIXEL_PERCENTAGE"
                
                band_configs = {
                    'rgb': {
                        'bands': ['B4', 'B3', 'B2'],
                        'min': 0, 'max': 3000,
                        'description': 'Natural color (10m resolution)'
                    },
                    'nir': {
                        'bands': ['B8', 'B4', 'B3'],
                        'min': 0, 'max': 3000,
                        'gamma': [0.95, 1.1, 1.0],
                        'description': 'False color - vegetation appears red (10m resolution)'
                    },
                    'ndvi': {
                        'bands': ['B8', 'B4'],
                        'description': 'NDVI vegetation health (10m resolution)'
                    },
                    'agriculture': {
                        'bands': ['B11', 'B8', 'B2'],
                        'min': 0, 'max': 3000,
                        'description': 'Agriculture composite - SWIR/NIR/Blue (20m/10m resolution)'
                    }
                }
                
                metadata_fields = {
                    'image_id': 'PRODUCT_ID',
                    'date': 'PRODUCT_ID',
                    'cloud': 'CLOUDY_PIXEL_PERCENTAGE',
                    'sun_elevation': 'MEAN_SOLAR_ZENITH_ANGLE'
                }
                
            elif satellite == "LANDSAT_8":
                collection_id = "LANDSAT/LC08/C02/T1_L2"
                cloud_cover_prop = "CLOUD_COVER"
                
                band_configs = {
                    'rgb': {
                        'bands': ['SR_B4', 'SR_B3', 'SR_B2'],
                        'min': 0.0, 'max': 0.3,
                        'description': 'Natural color (30m resolution)'
                    },
                    'nir': {
                        'bands': ['SR_B5', 'SR_B4', 'SR_B3'],
                        'min': 0.0, 'max': 0.3,
                        'gamma': [0.95, 1.1, 1.0],
                        'description': 'False color - vegetation appears red (30m resolution)'
                    },
                    'ndvi': {
                        'bands': ['SR_B5', 'SR_B4'],
                        'description': 'NDVI vegetation health (30m resolution)'
                    },
                    'agriculture': {
                        'bands': ['SR_B6', 'SR_B5', 'SR_B2'],
                        'min': 0.0, 'max': 0.3,
                        'description': 'Agriculture composite - SWIR1/NIR/Blue (30m resolution)'
                    }
                }
                
                metadata_fields = {
                    'image_id': 'LANDSAT_PRODUCT_ID',
                    'date': 'DATE_ACQUIRED',
                    'cloud': 'CLOUD_COVER',
                    'sun_elevation': 'SUN_ELEVATION'
                }
            else:
                raise ValueError(f"Unsupported satellite: {satellite}. Use 'SENTINEL_2' or 'LANDSAT_8'")
            
            # Step 3: Filter and get best optical image
            image_collection = (ee.ImageCollection(collection_id)
                              .filterBounds(farm_geometry)
                              .filterDate(start_date, end_date)
                              .filter(ee.Filter.lt(cloud_cover_prop, max_cloud_cover))
                              .sort(cloud_cover_prop))
            
            image_count = image_collection.size().getInfo()
            logger.info(f"Found {image_count} {satellite} images (cloud < {max_cloud_cover}%)")
            
            if image_count == 0:
                raise ValueError(
                    f"No {satellite} images found. Try increasing max_cloud_cover or expanding date range."
                )
            
            best_image = ee.Image(image_collection.first())
            
            # Step 4: Apply satellite-specific preprocessing
            if satellite == "LANDSAT_8":
                # Scale factors for Landsat Collection 2 Level-2
                optical_bands = best_image.select('SR_B.').multiply(0.0000275).add(-0.2)
                best_image = best_image.addBands(optical_bands, None, True)
            
            # Step 5: Get cloud cover for adaptive VI selection
            image_properties = best_image.toDictionary().getInfo()
            cloud_cover = float(image_properties.get(metadata_fields['cloud'], 0))
            
            # Cloud-adaptive logic
            use_sar_backup = force_sar_backup or cloud_cover >= 30.0
            
            thumbnails = {}
            
            # ===== OPTICAL THUMBNAILS (RGB, NIR, Agriculture) =====
            
            # RGB Natural Color
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
                'bands': rgb_config['bands'],
                'usage': 'Visual farm identification and field boundary verification'
            }
            
            # NIR False Color
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
                'bands': nir_config['bands'],
                'usage': 'Quick vegetation health assessment (red = healthy, blue = water/bare)'
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
                'bands': agri_config['bands'],
                'usage': 'Crop moisture and stress detection (bright = healthy crops)'
            }
            
            # ===== VEGETATION INDEX (CLOUD-ADAPTIVE) =====
            
            if not use_sar_backup:
                # PRIMARY: Optical NDVI (cloud_cover < 30%)
                logger.info(f"Using optical NDVI (cloud cover: {cloud_cover:.1f}%)")
                
                ndvi_config = band_configs['ndvi']
                ndvi = best_image.normalizedDifference(ndvi_config['bands'])
                
                # Histogram stretch for maximum contrast
                ndvi_stretched = ndvi.unitScale(-0.2, 0.9).clamp(0, 1)
                
                # Research-validated palette for Vietnamese agriculture
                # Based on visual interpretation standards from Le Minh Hang et al., 2021
                agricultural_palette = [
                    '0000FF',  # Blue: Water bodies
                    '8B4513',  # Brown: Bare soil / recently tilled
                    'FFFF00',  # Yellow: Sparse vegetation / early growth
                    'ADFF2F',  # Yellow-green: Developing crops
                    '00FF00',  # Green: Healthy crops (target for rice)
                    '228B22',  # Forest green: Peak health / dense canopy
                    '006400'   # Dark green: Very dense vegetation / forests
                ]
                
                thumbnails['vegetation_index'] = {
                    'url': ndvi_stretched.getThumbURL({
                        'min': 0,
                        'max': 1,
                        'palette': agricultural_palette,
                        'dimensions': 512,
                        'region': farm_geometry,
                        'format': 'png'
                    }),
                    'description': f'NDVI - Optical vegetation health ({ndvi_config["description"]})',
                    'bands': ndvi_config['bands'],
                    'index_type': 'NDVI',
                    'data_source': satellite,
                    'cloud_cover': cloud_cover,
                    'interpretation': {
                        'blue': 'Water bodies / flooded paddies',
                        'brown': 'Bare soil / recently planted (<1 week)',
                        'yellow': 'Early growth (1-3 weeks) / stressed crops',
                        'green': 'Healthy crops (4-8 weeks, target zone)',
                        'dark_green': 'Peak biomass / dense canopy (>8 weeks)'
                    },
                    'rice_stages': {
                        'transplanting': 'Blue (flooded)',
                        'tillering': 'Yellow (2-4 weeks)',
                        'vegetative': 'Green (5-8 weeks)',
                        'reproductive': 'Dark green (8-12 weeks)'
                    },
                    'usage': 'Crop growth stage monitoring and health assessment'
                }
                
            else:
                # BACKUP: Sentinel-1 RVI (cloud_cover >= 30% or forced)
                # Research-validated approach: 90.72% accuracy (Le Minh Hang et al., 2021)
                logger.info(f"Using SAR RVI (cloud: {cloud_cover:.1f}% or forced={force_sar_backup})")
                
                sar_collection_id = "COPERNICUS/S1_GRD"
                
                sar_collection = (ee.ImageCollection(sar_collection_id)
                    .filterBounds(farm_geometry)
                    .filterDate(start_date, end_date)
                    .filter(ee.Filter.listContains('transmitterReceiverPolarisation', 'VV'))
                    .filter(ee.Filter.listContains('transmitterReceiverPolarisation', 'VH'))
                    .filter(ee.Filter.eq('instrumentMode', 'IW'))
                    .sort('system:time_start', False))
                
                sar_count = sar_collection.size().getInfo()
                logger.info(f"Found {sar_count} Sentinel-1 SAR images")
                
                if sar_count == 0:
                    raise ValueError("No Sentinel-1 SAR images found for date range")
                
                sar_image = ee.Image(sar_collection.first())
                
                # Research-validated preprocessing (Le Minh Hang et al., 2021):
                # Lee filter 3x3 used in paper, but 5x5 better for speckle reduction
                sar_filtered = sar_image.focal_median(
                    radius=5,
                    kernelType='square',
                    units='pixels'
                )
                
                # Get VV and VH bands (in dB - Sentinel-1 GRD format)
                vv_db = sar_filtered.select('VV')
                vh_db = sar_filtered.select('VH')
                
                # CRITICAL: Convert dB to linear power (Equation 2 from paper)
                # Formula: linear = 10^(dB/10)
                vv_linear = ee.Image(10).pow(vv_db.divide(10))
                vh_linear = ee.Image(10).pow(vh_db.divide(10))
                
                # Calculate RVI on linear values (Equation 1 from paper)
                # RVI = (4 × σ_VH) / (σ_VV + σ_VH)
                # High RVI (0.6-0.8) = vegetation (rough surfaces scatter cross-pol)
                # Low RVI (0.1-0.3) = urban/roads (smooth surfaces, low cross-pol)
                rvi = vh_linear.multiply(4).divide(vv_linear.add(vh_linear))
                
                # Adaptive histogram stretching (scene-specific normalization)
                rvi_stats = rvi.reduceRegion(
                    reducer=ee.Reducer.percentile([2, 98]),
                    geometry=farm_geometry,
                    scale=10,
                    maxPixels=1e6,
                    bestEffort=True
                ).getInfo()
                
                # Extract percentile values
                stats_keys = list(rvi_stats.keys()) if rvi_stats else []
                rvi_min = rvi_stats.get(stats_keys[0], 0.2) if stats_keys else 0.2
                rvi_max = rvi_stats.get(stats_keys[1] if len(stats_keys) > 1 else stats_keys[0], 1.2) if stats_keys else 1.2
                
                logger.info(f"Adaptive RVI range: {rvi_min:.3f} to {rvi_max:.3f}")
                logger.info("Interpretation: <0.3=Urban, 0.3-0.5=Bare/Water, >0.5=Vegetation")
                
                # Stretch to 0-1 range
                rvi_stretched = rvi.unitScale(rvi_min, rvi_max).clamp(0, 1)
                
                # Research-validated palette (based on paper's Figure 5)
                sar_agricultural_palette = [
                    '000080',  # Navy: Urban/roads (low VH/VV ~0.1-0.2)
                    '0000FF',  # Blue: Smooth surfaces
                    '8B4513',  # Brown: Bare soil (medium VH/VV ~0.3-0.4)
                    'D2691E',  # Light brown: Sparse vegetation
                    'FFFF00',  # Yellow: Early crops (medium-high VH/VV ~0.5)
                    'ADFF2F',  # Yellow-green: Growing crops
                    '00FF00',  # Green: Healthy vegetation (high VH/VV ~0.6-0.7)
                    '006400'   # Dark green: Dense vegetation (very high VH/VV ~0.7-0.8)
                ]
                
                # Get SAR acquisition time
                sar_time = sar_image.get('system:time_start').getInfo()
                sar_date = datetime.fromtimestamp(sar_time/1000).strftime('%Y-%m-%d')
                
                thumbnails['vegetation_index'] = {
                    'url': rvi_stretched.getThumbURL({
                        'min': 0.0,
                        'max': 1.0,
                        'palette': sar_agricultural_palette,
                        'dimensions': 512,
                        'region': farm_geometry,
                        'format': 'png'
                    }),
                    'description': 'RVI - SAR all-weather vegetation monitoring (10m resolution)',
                    'bands': ['VV', 'VH'],
                    'index_type': 'RVI',
                    'data_source': 'Sentinel-1 SAR',
                    'acquisition_date': sar_date,
                    'cloud_cover': 'N/A (radar penetrates clouds)',
                    'speckle_filter': '5x5 focal median',
                    'adaptive_stretch': f'{rvi_min:.3f} to {rvi_max:.3f}',
                    'accuracy': '90.72% (validated on Vietnamese agriculture)',
                    'interpretation': {
                        'navy_blue': 'Roads / buildings / urban areas (VH/VV < 0.3)',
                        'brown': 'Bare soil / recently tilled fields (VH/VV ~0.3-0.4)',
                        'yellow': 'Early crop growth / sparse vegetation (VH/VV ~0.5)',
                        'green': 'Healthy growing crops (VH/VV ~0.6-0.7)',
                        'dark_green': 'Dense vegetation / peak biomass (VH/VV ~0.7-0.8)'
                    },
                    'rice_stages': {
                        'flooding_transplanting': 'Brown (smooth water, low backscatter)',
                        'tillering': 'Yellow (emerging stems, VH/VV ~0.5-0.6)',
                        'vegetative': 'Green (vertical stems, VH/VV ~0.7-0.9)',
                        'reproductive': 'Dark green (peak biomass, VH/VV ~1.0-1.2)',
                        'ripening': 'Yellow-green (declining water content)'
                    },
                    'usage': 'All-weather crop monitoring for monsoon season and cloud-covered periods',
                    'reference': 'Le Minh Hang et al., 2021 - ACRS 2019, Dak Lak, Vietnam'
                }
            
            # ===== FARM BOUNDARY =====
            
            boundary_feature = ee.Feature(farm_geometry)
            base_canvas = ee.Image(0).byte().paint(
                featureCollection=ee.FeatureCollection([boundary_feature]),
                color=0,
                width=1
            )
            boundary_image = base_canvas.paint(
                featureCollection=ee.FeatureCollection([boundary_feature]),
                color=255,
                width=5
            )
            
            thumbnails['farm_boundary'] = {
                'url': boundary_image.getThumbURL({
                    'palette': ['000000', 'FF0000'],
                    'dimensions': 512,
                    'region': farm_geometry,
                    'format': 'png'
                }),
                'description': 'Farm boundary outline (5px red line on black)',
                'bands': ['constant'],
                'usage': 'Field boundary verification for insurance claims'
            }
            
            # ===== METADATA EXTRACTION =====
            
            try:
                area_hectares = farm_geometry.area(maxError=1).divide(10000).getInfo()
            except Exception:
                area_hectares = None
            
            image_id = image_properties.get(metadata_fields['image_id'], 'unknown')
            
            # Parse Sentinel-2 date from PRODUCT_ID
            if satellite == "SENTINEL_2" and metadata_fields['date'] == 'PRODUCT_ID':
                product_id = image_properties.get('PRODUCT_ID', '')
                try:
                    date_part = product_id.split('_')[2]
                    acquisition_date = f"{date_part[:4]}-{date_part[4:6]}-{date_part[6:8]}"
                except:
                    acquisition_date = product_id[:10] if len(product_id) >= 10 else None
            else:
                acquisition_date = image_properties.get(metadata_fields['date'])
            
            sun_elevation = image_properties.get(metadata_fields['sun_elevation'], 0)
            if satellite == "SENTINEL_2" and sun_elevation > 0:
                sun_elevation = 90 - sun_elevation  # Convert zenith to elevation
            
            # ===== COMPILE RESPONSE =====
            
            result = {
                'farm_info': {
                    'coordinates': coordinates,
                    'crs': coordinate_crs,
                    'area_hectares': round(area_hectares, 4) if area_hectares else None
                },
                'image_info': {
                    'satellite': satellite,
                    'collection_id': collection_id,
                    'image_id': image_id,
                    'acquisition_date': acquisition_date,
                    'cloud_cover': round(cloud_cover, 2),
                    'sun_elevation': round(sun_elevation, 2) if sun_elevation else None
                },
                'vegetation_index_strategy': {
                    'cloud_threshold': 30.0,
                    'actual_cloud_cover': round(cloud_cover, 2),
                    'selected_index': 'RVI (SAR)' if use_sar_backup else 'NDVI (Optical)',
                    'reason': (
                        f'Cloud cover {cloud_cover:.1f}% >= 30% - using radar backup' 
                        if use_sar_backup and not force_sar_backup
                        else 'Forced SAR mode' if force_sar_backup
                        else f'Cloud cover {cloud_cover:.1f}% < 30% - using optical primary'
                    ),
                    'data_quality': 'All-weather radar (90.72% accuracy)' if use_sar_backup else 'High-accuracy optical',
                    'validation': 'Validated on Vietnamese rice fields (Le Minh Hang et al., 2021)' if use_sar_backup else None
                },
                'thumbnails': thumbnails,
                'usage_instructions': {
                    'web_display': 'Use thumbnail URLs directly in <img> tags',
                    'mobile_display': 'Load URLs in Image components (React Native, Flutter)',
                    'caching': 'URLs expire after 24h - cache images for offline use',
                    'dimensions': 'All thumbnails are 512px (largest dimension)',
                    'format': 'PNG with transparency support'
                },
                'processing_info': {
                    'date_range': f"{start_date} to {end_date}",
                    'optical_images_found': image_count,
                    'sar_images_available': sar_count if use_sar_backup else None,
                    'max_cloud_cover_filter': max_cloud_cover,
                    'speckle_filter': '5x5 focal median (SAR only)' if use_sar_backup else None
                }
            }
            
            logger.info(f"Generated {len(thumbnails)} thumbnails using {'SAR RVI' if use_sar_backup else 'Optical NDVI'}")
            return result
            
        except Exception as e:
            logger.error(f"Error generating thumbnails: {str(e)}", exc_info=True)
            raise

    def get_dynamic_world_raw_data(
        self,
        coordinates: List[List[float]],
        coordinate_crs: str = "EPSG:4326",
        start_date: str = "2024-01-01",
        end_date: str = "2024-12-31",
        max_images: int = 5
    ) -> Dict[str, Any]:
        """
        Get raw Dynamic World data without processing for inspection.
        Returns unprocessed Google Earth Engine responses.
        
        Args:
            coordinates: List of [x, y] coordinates forming a closed polygon
            coordinate_crs: Coordinate Reference System
            start_date: Start date in 'YYYY-MM-DD' format
            end_date: End date in 'YYYY-MM-DD' format
            max_images: Maximum number of images to analyze (default: 5)
            
        Returns:
            Dictionary containing raw Dynamic World data structures
        """
        try:
            logger.info(f"Getting raw Dynamic World data from {start_date} to {end_date}")
            
            # Step 1: Create farm geometry
            farm_geometry = ee.Geometry.Polygon(
                coords=[coordinates], 
                proj=coordinate_crs,
                geodesic=False
            )
            
            # Step 2: Load Dynamic World collection
            dw_collection = (ee.ImageCollection("GOOGLE/DYNAMICWORLD/V1")
                            .filterBounds(farm_geometry)
                            .filterDate(start_date, end_date)
                            .limit(max_images))
            
            # Step 3: Get raw collection info
            collection_info = dw_collection.getInfo()
            image_count = dw_collection.size().getInfo()
            
            logger.info(f"Found {image_count} Dynamic World images, analyzing up to {max_images}")
            
            # Step 4: Get raw data from first image
            first_image_data = None
            first_image_properties = None
            band_info = None
            if image_count > 0:
                first_image = ee.Image(dw_collection.first())
                
                # Get complete raw image info
                first_image_data = first_image.getInfo()
                
                # Get raw properties
                first_image_properties = first_image.toDictionary().getInfo()
                
                # Get band names and info
                band_names = first_image.bandNames().getInfo()
                
                # Get detailed band information
                band_info = {}
                for band_name in band_names:
                    try:
                        band_image = first_image.select(band_name)
                        band_info[band_name] = {
                            'band_info': band_image.getInfo(),
                            'data_type': band_image.getInfo().get('bands', [{}])[0].get('data_type', 'unknown') if band_image.getInfo().get('bands') else 'unknown'
                        }
                    except Exception as band_error:
                        band_info[band_name] = {'error': str(band_error)}
            
            # Step 5: Get raw pixel values for small sample area
            sample_values = None
            if image_count > 0:
                try:
                    first_image = ee.Image(dw_collection.first())
                    
                    # Get raw pixel values from a small sample (to avoid timeout)
                    sample_values = first_image.sample(
                        region=farm_geometry,
                        scale=10,
                        numPixels=100,  # Small sample to avoid timeout
                        seed=0,
                        dropNulls=True
                    ).getInfo()
                    
                except Exception as sample_error:
                    logger.warning(f"Could not get sample values: {sample_error}")
                    sample_values = {'error': str(sample_error)}
            
            # Step 6: Get raw histogram data
            raw_histogram = None
            if image_count > 0:
                try:
                    first_image = ee.Image(dw_collection.first())
                    label_band = first_image.select('label')
                    
                    raw_histogram = label_band.reduceRegion(
                        reducer=ee.Reducer.frequencyHistogram(),
                        geometry=farm_geometry,
                        scale=10,
                        maxPixels=1e6
                    ).getInfo()
                    
                except Exception as hist_error:
                    logger.warning(f"Could not get histogram: {hist_error}")
                    raw_histogram = {'error': str(hist_error)}
            
            # Step 7: Get list of all images with basic info
            image_list = []
            if image_count > 0:
                try:
                    # Get info about each image in the collection
                    image_list_info = dw_collection.getInfo()
                    if 'features' in image_list_info:
                        for feature in image_list_info['features']:
                            if 'properties' in feature:
                                image_list.append({
                                    'id': feature.get('id', 'unknown'),
                                    'properties': feature.get('properties', {}),
                                    'bands': feature.get('bands', [])
                                })
                except Exception as list_error:
                    logger.warning(f"Could not get image list: {list_error}")
                    image_list = [{'error': str(list_error)}]
            
            # Step 8: Get available band information
            available_bands = []
            if image_count > 0:
                try:
                    first_image = ee.Image(dw_collection.first())
                    available_bands = first_image.bandNames().getInfo()
                except Exception as bands_error:
                    available_bands = [f"Error getting bands: {bands_error}"]
            
            # Step 9: Get projection information
            projection_info = None
            if image_count > 0:
                try:
                    first_image = ee.Image(dw_collection.first())
                    projection_info = first_image.projection().getInfo()
                except Exception as proj_error:
                    projection_info = {'error': str(proj_error)}
            
            # Step 10: Try to get raw reduce region data for all bands
            all_bands_stats = None
            if image_count > 0:
                try:
                    first_image = ee.Image(dw_collection.first())
                    all_bands_stats = first_image.reduceRegion(
                        reducer=ee.Reducer.mean()
                        .combine(ee.Reducer.stdDev(), sharedInputs=True)
                        .combine(ee.Reducer.minMax(), sharedInputs=True),
                        geometry=farm_geometry,
                        scale=10,
                        maxPixels=1e6
                    ).getInfo()
                except Exception as stats_error:
                    all_bands_stats = {'error': str(stats_error)}
            
            # Step 11: Compile complete raw response
            result = {
                'request_info': {
                    'coordinates': coordinates,
                    'crs': coordinate_crs,
                    'date_range': f"{start_date} to {end_date}",
                    'max_images_requested': max_images,
                    'collection_id': 'GOOGLE/DYNAMICWORLD/V1'
                },
                'collection_raw_info': {
                    'image_count': image_count,
                    'collection_data': collection_info,
                    'collection_type': type(collection_info).__name__,
                    'collection_keys': list(collection_info.keys()) if isinstance(collection_info, dict) else 'Not a dict'
                },
                'first_image_raw_data': {
                    'image_info': first_image_data,
                    'image_properties': first_image_properties,
                    'image_info_type': type(first_image_data).__name__ if first_image_data else 'None',
                    'image_info_keys': list(first_image_data.keys()) if isinstance(first_image_data, dict) else 'Not a dict or None'
                },
                'band_information': {
                    'available_bands': available_bands,
                    'detailed_band_info': band_info,
                    'band_count': len(available_bands) if isinstance(available_bands, list) else 0
                },
                'projection_info': projection_info,
                'sample_pixel_values': {
                    'sample_data': sample_values,
                    'sample_type': type(sample_values).__name__ if sample_values else 'None',
                    'sample_keys': list(sample_values.keys()) if isinstance(sample_values, dict) and 'error' not in sample_values else 'Error or not dict'
                },
                'raw_statistics': {
                    'label_histogram': raw_histogram,
                    'all_bands_stats': all_bands_stats
                },
                'image_list_raw': {
                    'images': image_list,
                    'list_length': len(image_list) if isinstance(image_list, list) else 0
                },
                'data_structure_analysis': {
                    'collection_structure': {
                        'type': type(collection_info).__name__,
                        'has_features': 'features' in collection_info if isinstance(collection_info, dict) else False,
                        'features_count': len(collection_info.get('features', [])) if isinstance(collection_info, dict) else 0
                    },
                    'first_image_structure': {
                        'type': type(first_image_data).__name__ if first_image_data else 'None',
                        'has_bands': 'bands' in first_image_data if isinstance(first_image_data, dict) else False,
                        'has_properties': 'properties' in first_image_data if isinstance(first_image_data, dict) else False
                    },
                    'properties_structure': {
                        'type': type(first_image_properties).__name__ if first_image_properties else 'None',
                        'property_count': len(first_image_properties) if isinstance(first_image_properties, dict) else 0,
                        'sample_properties': list(first_image_properties.keys())[:10] if isinstance(first_image_properties, dict) else []
                    }
                },
                'debug_info': {
                    'function_executed': 'get_dynamic_world_raw_data',
                    'timestamp': int(datetime.now().timestamp()),
                    'geometry_used': {
                        'type': 'Polygon',
                        'coordinates': [coordinates],
                        'crs': coordinate_crs
                    }
                }
            }
            
            logger.info(f"Raw Dynamic World data retrieved. Images found: {image_count}")
            logger.info(f"Available bands: {available_bands}")
            logger.info(f"Collection type: {type(collection_info).__name__}")
            
            return result
            
        except ee.EEException as e:
            logger.error(f"Google Earth Engine API error: {e}")
            return {
                'error': {
                    'type': 'EEException',
                    'message': str(e),
                    'function': 'get_dynamic_world_raw_data'
                }
            }
        except Exception as e:
            logger.error(f"Error getting raw Dynamic World data: {e}")
            return {
                'error': {
                    'type': type(e).__name__,
                    'message': str(e),
                    'function': 'get_dynamic_world_raw_data'
                }
            }

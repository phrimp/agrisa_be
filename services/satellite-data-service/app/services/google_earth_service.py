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
        satellite: str = "LANDSAT_8",
        max_cloud_cover: float = 30.0,
        force_sar_backup: bool = False  # NEW: Manual override for SAR
    ) -> Dict[str, Any]:
        """
        Generate farm thumbnail images with cloud-adaptive vegetation monitoring.
        
        Vegetation Index Strategy:
        - cloud_cover < 30%: Use Sentinel-2 NDVI (optical, high accuracy)
        - cloud_cover >= 30%: Use Sentinel-1 RVI (radar, all-weather)
        
        Args:
            coordinates: List of [x, y] coordinates forming a closed polygon
            coordinate_crs: Coordinate Reference System
            start_date: Start date in 'YYYY-MM-DD' format
            end_date: End date in 'YYYY-MM-DD' format
            satellite: Satellite collection name ("LANDSAT_8" or "SENTINEL_2")
            max_cloud_cover: Maximum cloud coverage percentage
            force_sar_backup: Force use of Sentinel-1 RVI regardless of clouds
            
        Returns:
            Dictionary containing thumbnail URLs with cloud-adaptive vegetation index
        """
        try:
            logger.info(f"Generating {satellite} farm thumbnail images with cloud-adaptive VI")
            
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
                
                metadata_fields = {
                    'image_id': 'LANDSAT_PRODUCT_ID',
                    'date': 'DATE_ACQUIRED',
                    'cloud': 'CLOUD_COVER',
                    'sun_elevation': 'SUN_ELEVATION'
                }
                
            elif satellite == "SENTINEL_2":
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
                
                metadata_fields = {
                    'image_id': 'PRODUCT_ID',
                    'date': 'PRODUCT_ID',
                    'cloud': 'CLOUDY_PIXEL_PERCENTAGE',
                    'sun_elevation': 'MEAN_SOLAR_ZENITH_ANGLE'
                }
                
            else:
                raise ValueError(f"Unsupported satellite: {satellite}. Use 'LANDSAT_8' or 'SENTINEL_2'")
            
            # Step 3: Filter and get best optical image
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
                optical_bands = best_image.select('SR_B.').multiply(0.0000275).add(-0.2)
                best_image = best_image.addBands(optical_bands, None, True)
            
            # Step 5: Get cloud cover of best image for adaptive VI selection
            image_properties = best_image.toDictionary().getInfo()
            cloud_cover = image_properties.get(metadata_fields['cloud'], 0)
            
            # === CLOUD-ADAPTIVE VEGETATION INDEX LOGIC ===
            use_sar_backup = force_sar_backup or cloud_cover >= 30.0
            
            thumbnails = {}
            
            # Generate RGB, NIR, and Agriculture thumbnails (unchanged)
            # ... [RGB thumbnail code - same as original]
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
            
            # ... [NIR thumbnail code - same as original]
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
            
            # ... [Agriculture thumbnail code - same as original]
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
            
            # === IMPROVED VEGETATION INDEX THUMBNAIL ===
            if not use_sar_backup:
                # PRIMARY: Use Sentinel-2/Landsat NDVI (cloud_cover < 30%)
                logger.info(f"Using optical NDVI (cloud cover: {cloud_cover:.1f}%)")
                
                ndvi_config = band_configs['ndvi']
                ndvi = best_image.normalizedDifference(ndvi_config['bands'])
                
                # Apply histogram stretch for maximum contrast
                # This makes boundaries between fields much more visible
                ndvi_stretched = ndvi.unitScale(-0.2, 0.9).clamp(0, 1)
                
                # OPTION 1: Bold Discrete Zones (Best for boundary detection)
                # Uses distinct color blocks - human eyes excel at detecting color boundaries
                discrete_palette = [
                    '000000',  # Black: Water bodies
                    '8B4513',  # Brown: Bare soil/no vegetation
                    'FFFF00',  # Bright Yellow: Sparse/stressed crops
                    '00FF00',  # Bright Green: Healthy crops
                    '006400'   # Dark Green: Very dense vegetation
                ]
                
                # OPTION 2: High-Contrast Agricultural (Good for crop type identification)
                # More gradual but maintains strong boundaries
                agricultural_palette = [
                    '0000FF',  # Blue: Water
                    'A52A2A',  # Brown: Bare soil
                    'FFD700',  # Gold: Early growth
                    'ADFF2F',  # Yellow-green: Developing crops
                    '32CD32',  # Lime green: Healthy crops
                    '228B22',  # Forest green: Peak health
                    '006400'   # Dark green: Very dense
                ]
                
                # OPTION 3: Rainbow High-Contrast (Best overall visibility)
                # Maximum color differentiation for human perception
                rainbow_palette = [
                    '000080',  # Navy: Water/very low
                    '0000FF',  # Blue: Bare soil
                    '00FFFF',  # Cyan: Poor vegetation
                    '00FF00',  # Green: Moderate vegetation
                    'FFFF00',  # Yellow: Good vegetation
                    'FF8C00',  # Orange: Healthy crops
                    'FF0000'   # Red: Very healthy/dense
                ]
                
                # OPTION 4: Inverted (Healthy = Cool colors, ideal for quick scanning)
                inverted_palette = [
                    'FF0000',  # Red: Water/bare
                    'FF8C00',  # Orange: Poor vegetation
                    'FFFF00',  # Yellow: Moderate
                    '00FF00',  # Green: Good
                    '00FFFF',  # Cyan: Healthy
                    '0000FF',  # Blue: Very healthy
                    '000080'   # Navy: Dense canopy
                ]
                
                # SELECT PALETTE (change this to test different options)
                selected_palette = discrete_palette  # Change to: agricultural_palette, rainbow_palette, inverted_palette
                palette_name = "Discrete Zones"  # Update name when changing
                
                thumbnails['vegetation_index'] = {
                    'url': ndvi_stretched.getThumbURL({
                        'min': 0,      # After stretch, range is 0-1
                        'max': 1,
                        'palette': selected_palette,
                        'dimensions': 512,
                        'region': farm_geometry,
                        'format': 'png'
                    }),
                    'description': f'NDVI - Optical vegetation health ({ndvi_config["description"]})',
                    'bands': ndvi_config['bands'],
                    'index_type': 'NDVI',
                    'data_source': satellite,
                    'cloud_cover': cloud_cover,
                    'palette_type': palette_name,
                    'interpretation': {
                        'black_blue': 'Water bodies / Non-vegetation',
                        'brown': 'Bare soil / Recently planted',
                        'yellow': 'Sparse vegetation / Stressed crops',
                        'green': 'Healthy growing crops',
                        'dark_green': 'Peak health / Dense canopy'
                    },
                    'visual_notes': 'Histogram stretched + discrete colors for maximum boundary visibility',
                    'alternative_palettes': {
                        'discrete_palette': 'Best for boundary detection (5 distinct zones)',
                        'agricultural_palette': 'Good for crop type identification (7 zones)',
                        'rainbow_palette': 'Maximum color differentiation (7 zones)',
                        'inverted_palette': 'Healthy crops = cool colors (7 zones)'
                    }
                }
                
            else:
                # BACKUP: Use Sentinel-1 RVI (cloud_cover >= 30% or forced)
                logger.info(f"Using SAR RVI backup (cloud cover: {cloud_cover:.1f}% or forced)")
                
                # Fetch Sentinel-1 SAR data (all-weather radar)
                sar_collection_id = "COPERNICUS/S1_GRD"  # Ground Range Detected
                
                sar_collection = (ee.ImageCollection(sar_collection_id)
                                .filterBounds(farm_geometry)
                                .filterDate(start_date, end_date)
                                .filter(ee.Filter.listContains('transmitterReceiverPolarisation', 'VV'))
                                .filter(ee.Filter.listContains('transmitterReceiverPolarisation', 'VH'))
                                .filter(ee.Filter.eq('instrumentMode', 'IW'))  # Interferometric Wide swath
                                .filter(ee.Filter.eq('orbitProperties_pass', 'DESCENDING'))
                                .sort('system:time_start', False))  # Get most recent
                
                sar_count = sar_collection.size().getInfo()
                logger.info(f"Found {sar_count} Sentinel-1 SAR images")
                
                if sar_count == 0:
                    raise ValueError("No Sentinel-1 SAR images found. Cannot generate RVI backup.")
                
                sar_image = ee.Image(sar_collection.first())
                
                # Calculate RVI (Radar Vegetation Index)
                # RVI = (4 * VH) / (VV + VH)
                # Range: 0 (bare soil) to ~2+ (dense vegetation)
                vv = sar_image.select('VV')
                vh = sar_image.select('VH')
                rvi = vh.multiply(4).divide(vv.add(vh))
                
                # Apply histogram stretch for maximum contrast
                rvi_stretched = rvi.unitScale(0.0, 1.5).clamp(0, 1)
                
                # OPTION 1: Bold SAR Discrete (Best for boundary detection)
                sar_discrete = [
                    '000080',  # Navy: Water
                    '8B4513',  # Brown: Bare soil
                    'FFFF00',  # Yellow: Sparse vegetation
                    '00FF00',  # Green: Moderate crops
                    'FF0000'   # Red: Dense vegetation
                ]
                
                # OPTION 2: Thermal-style (Good for intensity perception)
                thermal_palette = [
                    '000000',  # Black: Water
                    '0000FF',  # Blue: Very smooth
                    '00FFFF',  # Cyan: Smooth surfaces
                    '00FF00',  # Green: Moderate roughness
                    'FFFF00',  # Yellow: Rough surfaces
                    'FF8C00',  # Orange: Vegetation
                    'FF0000',  # Red: Dense vegetation
                    'FFFFFF'   # White: Very dense
                ]
                
                # OPTION 3: Purple-Orange Diverging (Best contrast for SAR)
                diverging_palette = [
                    '4B0082',  # Indigo: Water
                    '8B00FF',  # Purple: Bare soil
                    '0000FF',  # Blue: Poor vegetation
                    '00FFFF',  # Cyan: Moderate
                    '00FF00',  # Green: Good
                    'FFFF00',  # Yellow: Healthy
                    'FFA500',  # Orange: Very healthy
                    'FF0000'   # Red: Dense canopy
                ]
                
                # SELECT PALETTE
                selected_sar_palette = sar_discrete
                sar_palette_name = "SAR Discrete Zones"
                
                thumbnails['vegetation_index'] = {
                    'url': rvi_stretched.getThumbURL({
                        'min': 0.0,
                        'max': 1.0,
                        'palette': selected_sar_palette,
                        'dimensions': 512,
                        'region': farm_geometry,
                        'format': 'png'
                    }),
                    'description': 'RVI - SAR all-weather vegetation monitoring (10m resolution)',
                    'bands': ['VV', 'VH'],
                    'index_type': 'RVI',
                    'data_source': 'Sentinel-1 SAR',
                    'cloud_cover': 'N/A (radar)',
                    'palette_type': sar_palette_name,
                    'interpretation': {
                        'navy_blue': 'Water bodies (smooth surfaces)',
                        'brown': 'Bare soil / No vegetation',
                        'yellow': 'Sparse vegetation / Young crops',
                        'green': 'Moderate vegetation / Growing crops',
                        'red': 'Dense vegetation / Healthy crops'
                    },
                    'visual_notes': 'SAR-based backup for cloudy conditions - histogram stretched for boundary clarity',
                    'sar_acquisition_date': sar_image.get('system:time_start').getInfo(),
                    'alternative_palettes': {
                        'sar_discrete': 'Best for boundary detection (5 zones)',
                        'thermal_palette': 'Good for intensity perception (8 zones)',
                        'diverging_palette': 'Maximum SAR contrast (8 zones)'
                    }
                }
            
            # === FARM BOUNDARY THUMBNAIL (HIGH CONTRAST) ===
            # Enhanced boundary visualization with thicker lines and better color
            boundary_feature = ee.Feature(farm_geometry)
            
            # Create base image with black background for contrast
            base_canvas = ee.Image(0).byte().paint(
                featureCollection=ee.FeatureCollection([boundary_feature]),
                color=0,
                width=1
            )
            
            # Paint boundary with bright color and thick line
            boundary_image = base_canvas.paint(
                featureCollection=ee.FeatureCollection([boundary_feature]),
                color=255,
                width=5  # Thicker line for better visibility
            )
            
            thumbnails['farm_boundary'] = {
                'url': boundary_image.getThumbURL({
                    'palette': ['000000', 'FF0000'],  # Black background, red boundary
                    'dimensions': 512,
                    'region': farm_geometry,
                    'format': 'png'
                }),
                'description': 'Farm boundary outline (enhanced contrast)',
                'bands': ['constant'],
                'visual_notes': 'Thick red line on black background for maximum visibility'
            }
            
            # Step 6: Calculate farm area
            try:
                area_hectares = farm_geometry.area(maxError=1).divide(10000).getInfo()
            except Exception as area_error:
                logger.warning(f"Could not calculate area: {area_error}")
                area_hectares = None
            
            # Step 7: Extract remaining metadata
            image_id = image_properties.get(metadata_fields['image_id'], 'unknown')
            
            if satellite == "SENTINEL_2" and metadata_fields['date'] == 'PRODUCT_ID':
                product_id = image_properties.get('PRODUCT_ID', '')
                if len(product_id) > 15:
                    try:
                        date_part = product_id.split('_')[2]
                        acquisition_date = f"{date_part[:4]}-{date_part[4:6]}-{date_part[6:8]}"
                    except:
                        acquisition_date = None
                else:
                    acquisition_date = None
            else:
                acquisition_date = image_properties.get(metadata_fields['date'])
            
            sun_elevation = image_properties.get(metadata_fields['sun_elevation'], 0)
            if satellite == "SENTINEL_2" and sun_elevation > 0:
                sun_elevation = 90 - sun_elevation
            
            # Step 8: Compile response with cloud-adaptive info
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
                'vegetation_index_strategy': {
                    'cloud_threshold': 30.0,
                    'actual_cloud_cover': cloud_cover,
                    'selected_index': 'RVI (SAR)' if use_sar_backup else 'NDVI (Optical)',
                    'reason': 'Cloud cover >= 30% - using radar backup' if use_sar_backup else 'Cloud cover < 30% - using optical primary',
                    'data_quality': 'All-weather radar' if use_sar_backup else 'High-accuracy optical'
                },
                'thumbnails': thumbnails,
                'usage_instructions': {
                    'web_display': 'Use thumbnail URLs directly in <img> tags',
                    'mobile_display': 'Load URLs in ImageView/Image components',
                    'caching': 'URLs are temporary - cache images if needed for offline use',
                    'dimensions': 'All thumbnails are 512px (largest dimension)',
                    'format': 'PNG with transparency support',
                    'boundary_visibility': 'Enhanced contrast for clear farm boundary identification'
                },
                'processing_info': {
                    'date_range': f"{start_date} to {end_date}",
                    'images_found': image_count,
                    'max_cloud_cover': max_cloud_cover,
                    'cloud_filter_property': cloud_cover_prop,
                    'sar_images_available': sar_count if use_sar_backup else 'Not queried'
                }
            }
            
            logger.info(f"Generated {len(thumbnails)} thumbnail images with cloud-adaptive VI")
            return result
            
        except Exception as e:
            logger.error(f"Error generating thumbnails: {e}")
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

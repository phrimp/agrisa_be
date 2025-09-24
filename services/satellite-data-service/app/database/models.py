from sqlalchemy import Column, Integer, String, DateTime, Float, Text, Boolean, ForeignKey
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from geoalchemy2 import Geometry
from datetime import datetime, timezone

from .connection import Base
from ..utils.string_utils import (
    generate_satellite_image_id,
    generate_image_analysis_id,
    generate_satellite_collection_id,
    generate_download_request_id,
    get_current_timestamp
)


class SatelliteImage(Base):
    """Model for storing satellite image metadata and references."""
    
    __tablename__ = "satellite_images"
    
    id = Column(String(15), primary_key=True, default=generate_satellite_image_id)
    
    # Image metadata
    name = Column(String(255), nullable=False)
    description = Column(Text)
    satellite_name = Column(String(100), nullable=False)  # e.g., "Landsat-8", "Sentinel-2"
    image_date = Column(Integer, nullable=False)  # Unix timestamp in seconds
    
    # Geospatial data
    geometry = Column(Geometry("POLYGON", srid=4326), nullable=False)  # Image footprint
    center_point = Column(Geometry("POINT", srid=4326), nullable=False)  # Image center
    
    # Technical metadata
    cloud_coverage = Column(Float)  # Cloud coverage percentage (0-100)
    resolution = Column(Float)  # Spatial resolution in meters
    bands_info = Column(JSONB)  # Information about spectral bands
    
    # File storage
    minio_path = Column(String(500), nullable=False)  # Path in MinIO storage
    file_size = Column(Integer)  # File size in bytes
    file_format = Column(String(50))  # e.g., "GeoTIFF", "JP2"
    
    # Processing metadata
    processing_level = Column(String(50))  # e.g., "L1C", "L2A"
    processing_date = Column(Integer)  # Unix timestamp in seconds
    processing_parameters = Column(JSONB)  # Parameters used in processing
    
    # Timestamps (Unix seconds for easier comparison)
    created_at = Column(Integer, default=get_current_timestamp, nullable=False)
    updated_at = Column(Integer, default=get_current_timestamp, onupdate=get_current_timestamp, nullable=False)
    
    # Status
    is_available = Column(Boolean, default=True, nullable=False)
    
    # Relationships
    analysis_results = relationship("ImageAnalysis", back_populates="satellite_image", cascade="all, delete-orphan")


class ImageAnalysis(Base):
    """Model for storing analysis results on satellite images."""
    
    __tablename__ = "image_analysis"
    
    id = Column(String(15), primary_key=True, default=generate_image_analysis_id)
    satellite_image_id = Column(String(15), ForeignKey("satellite_images.id"), nullable=False)
    
    # Analysis metadata
    analysis_type = Column(String(100), nullable=False)  # e.g., "NDVI", "Land_Classification", "Change_Detection"
    analysis_name = Column(String(255))
    description = Column(Text)
    
    # Analysis parameters
    parameters = Column(JSONB)  # Parameters used in the analysis
    algorithm_version = Column(String(50))
    
    # Results
    result_data = Column(JSONB)  # Quantitative results (statistics, metrics)
    result_geometry = Column(Geometry("MULTIPOLYGON", srid=4326))  # Spatial results if applicable
    result_image_path = Column(String(500))  # Path to result visualization in MinIO
    
    # Quality metrics
    accuracy = Column(Float)  # Analysis accuracy if available (0-1)
    confidence_score = Column(Float)  # Confidence in results (0-1)
    
    # Timestamps (Unix seconds for easier comparison)
    created_at = Column(Integer, default=get_current_timestamp, nullable=False)
    updated_at = Column(Integer, default=get_current_timestamp, onupdate=get_current_timestamp, nullable=False)
    
    # Status
    status = Column(String(50), default="completed", nullable=False)  # "pending", "processing", "completed", "failed"
    
    # Relationships
    satellite_image = relationship("SatelliteImage", back_populates="analysis_results")


class SatelliteCollection(Base):
    """Model for organizing satellite images into collections."""
    
    __tablename__ = "satellite_collections"
    
    id = Column(String(15), primary_key=True, default=generate_satellite_collection_id)
    
    # Collection metadata
    name = Column(String(255), nullable=False)
    description = Column(Text)
    collection_type = Column(String(100))  # e.g., "time_series", "mosaic", "campaign"
    
    # Spatial and temporal bounds
    bounding_box = Column(Geometry("POLYGON", srid=4326))
    date_start = Column(Integer)  # Unix timestamp in seconds
    date_end = Column(Integer)  # Unix timestamp in seconds
    
    # Collection properties
    tags = Column(JSONB)  # Tags for categorization
    metadata = Column(JSONB)  # Additional metadata
    
    # Timestamps (Unix seconds for easier comparison)
    created_at = Column(Integer, default=get_current_timestamp, nullable=False)
    updated_at = Column(Integer, default=get_current_timestamp, onupdate=get_current_timestamp, nullable=False)
    
    # Status
    is_public = Column(Boolean, default=False, nullable=False)
    
    # Relationships
    images = relationship("CollectionImage", back_populates="collection", cascade="all, delete-orphan")


class CollectionImage(Base):
    """Association table for many-to-many relationship between collections and images."""
    
    __tablename__ = "collection_images"
    
    collection_id = Column(String(15), ForeignKey("satellite_collections.id"), primary_key=True)
    image_id = Column(String(15), ForeignKey("satellite_images.id"), primary_key=True)
    
    # Association metadata
    order_index = Column(Integer)  # Order within collection
    added_at = Column(Integer, default=get_current_timestamp, nullable=False)  # Unix timestamp in seconds
    
    # Relationships
    collection = relationship("SatelliteCollection", back_populates="images")
    image = relationship("SatelliteImage")


class DownloadRequest(Base):
    """Model for tracking image download requests."""
    
    __tablename__ = "download_requests"
    
    id = Column(String(15), primary_key=True, default=generate_download_request_id)
    
    # Request details
    user_id = Column(String(255))  # User identifier (could be external)
    image_ids = Column(JSONB, nullable=False)  # List of image IDs requested
    
    # Processing details
    format_requested = Column(String(50))  # Requested output format
    compression = Column(String(50))  # Compression type
    coordinate_system = Column(String(100))  # Coordinate system for output
    
    # Status tracking
    status = Column(String(50), default="pending", nullable=False)  # "pending", "processing", "ready", "expired", "failed"
    download_url = Column(String(500))  # Temporary download URL when ready
    expires_at = Column(Integer)  # When the download expires (Unix timestamp in seconds)
    
    # Timestamps (Unix seconds for easier comparison)
    created_at = Column(Integer, default=get_current_timestamp, nullable=False)
    updated_at = Column(Integer, default=get_current_timestamp, onupdate=get_current_timestamp, nullable=False)
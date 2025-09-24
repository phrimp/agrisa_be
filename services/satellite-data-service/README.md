# Satellite Data Service - Infrastructure Layer

This service provides the core infrastructure components (database and storage layers) for satellite data management in the Agrisa platform.

## 🏗️ Infrastructure Components

### Database Layer (PostGIS)
- **PostgreSQL with PostGIS** spatial extension
- **SQLAlchemy async ORM** for database operations
- **Spatial data models** for satellite images and analysis results
- **GeoAlchemy2** for spatial query support

### Storage Layer (MinIO)
- **MinIO object storage** for satellite image files
- **Presigned URL generation** for secure file access
- **Metadata storage** with file upload tracking
- **Async operations** with error handling

### Configuration Layer
- **Pydantic settings** with environment variable support
- **Database connection** configuration
- **MinIO client** configuration
- **Application lifecycle** management

## 📋 Database Models

### SatelliteImage
- Stores satellite image metadata and spatial geometry
- Supports multiple satellite types (Landsat, Sentinel-2, etc.)
- Includes cloud coverage, resolution, and processing information
- Links to MinIO storage paths

### ImageAnalysis
- Stores analysis results linked to satellite images
- Supports various analysis types (NDVI, classification, etc.)
- Includes accuracy metrics and result geometries
- Extensible for custom analysis workflows

### Collections & Downloads
- Organize images into logical collections
- Track download requests and access patterns
- Support batch operations and user management

## 🚀 Usage

### Initialize Infrastructure
```python
from app.database.connection import init_db
from app.storage.minio_client import minio_client

# Initialize database with PostGIS
await init_db()

# Use MinIO client for file operations
await minio_client.upload_file(file_path, file_data)
```

### Database Operations
```python
from app.database.models import SatelliteImage
from app.database.connection import get_db

# Create spatial queries
async with get_db() as db:
    images = await db.execute(
        select(SatelliteImage)
        .where(ST_Intersects(SatelliteImage.geometry, bbox))
    )
```

### Storage Operations
```python
from app.storage.minio_client import minio_client

# Upload satellite image
success = await minio_client.upload_file(
    file_path="images/landsat/2024/image.tif",
    file_data=image_data,
    content_type="image/tiff"
)

# Generate download URL
download_url = minio_client.get_presigned_url("images/landsat/2024/image.tif")
```

## 🔧 Configuration

Configure via environment variables:

```env
# Database
DATABASE_URL=postgresql+asyncpg://user:pass@host:port/dbname

# MinIO Storage
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_SECURE=false
MINIO_BUCKET_NAME=satellite-data

# Application
APP_NAME=Agrisa Satellite Data Service
DEBUG=false
LOG_LEVEL=INFO
```

## 📁 Project Structure

```
app/
├── config/
│   ├── __init__.py
│   └── settings.py          # Pydantic configuration
├── database/
│   ├── __init__.py
│   ├── connection.py        # Async SQLAlchemy setup
│   └── models.py            # PostGIS spatial models
├── storage/
│   ├── __init__.py
│   └── minio_client.py      # MinIO async client
└── main.py                  # Infrastructure initialization
```

## 🎯 Next Steps

This infrastructure layer is ready for:

1. **Service Layer Implementation**
   - Business logic for satellite data processing
   - Analysis workflows and algorithms
   - Data validation and transformation

2. **API Layer Implementation**
   - FastAPI endpoints for CRUD operations
   - Authentication and authorization
   - File upload and download handling

3. **Additional Features**
   - Caching layer (Redis integration)
   - Message queues for async processing
   - Monitoring and metrics collection

The infrastructure provides a solid foundation for building comprehensive satellite data management and analysis capabilities.
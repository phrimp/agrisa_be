import logging
import os
from datetime import datetime

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.config.settings import get_settings
from app.api.v1.endpoints import satellite

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Get application settings
settings = get_settings()

# Create FastAPI application
app = FastAPI(
    title=settings.app_name,
    version=settings.app_version,
    description="""
    Agrisa Satellite Data Processing Service
    
    This service provides satellite imagery retrieval and analysis capabilities
    for agricultural monitoring in Vietnam using Google Earth Engine.
    
    ## Features
    
    * Satellite image retrieval by coordinates
    * Support for Sentinel-2 and Landsat 8/9 imagery
    * Coordinate validation for Vietnam boundaries
    * Health monitoring and service status
    * Automated cloud filtering and image selection
    
    ## Supported Satellites
    
    * **Sentinel-2**: 10-20m resolution, 5-day revisit cycle
    * **Landsat 8/9**: 30m resolution, 16-day revisit cycle
    """,
    docs_url="/docs",
    redoc_url="/redoc",
    openapi_url="/openapi.json"
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include API routers
app.include_router(satellite.router, prefix="/api/v1")


@app.get("/")
async def root():
    """Root endpoint providing basic service information."""
    return {
        "service": settings.app_name,
        "version": settings.app_version,
        "status": "running",
        "timestamp": datetime.utcnow().isoformat(),
        "docs": "/docs",
        "health": "/api/v1/satellite/health"
    }


@app.get("/health")
async def health():
    """Simple health check endpoint."""
    return {
        "status": "healthy",
        "service": settings.app_name,
        "timestamp": datetime.utcnow().isoformat()
    }


@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    """Global exception handler for unhandled errors."""
    logger.error(f"Unhandled exception: {exc}", exc_info=True)
    
    return JSONResponse(
        status_code=500,
        content={
            "success": False,
            "error": "Internal server error",
            "message": str(exc) if settings.debug else "An unexpected error occurred",
            "timestamp": datetime.utcnow().isoformat()
        }
    )


@app.on_event("startup")
async def startup_event():
    """Startup event handler."""
    logger.info(f"Starting {settings.app_name} v{settings.app_version}")
    logger.info(f"Debug mode: {settings.debug}")
    logger.info(f"Google Earth Engine Project ID: {settings.gee_project_id}")
    
    # Log environment variable status
    env_status = {
        "GEE_PROJECT_ID": "✓" if settings.gee_project_id else "✗",
        "GEE_SERVICE_ACCOUNT_KEY": "✓" if settings.gee_service_account_key else "✗",
        "DEBUG": settings.debug,
        "LOG_LEVEL": settings.log_level
    }
    
    logger.info("Environment configuration:")
    for key, value in env_status.items():
        logger.info(f"  {key}: {value}")


@app.on_event("shutdown")
async def shutdown_event():
    """Shutdown event handler."""
    logger.info(f"Shutting down {settings.app_name}")


if __name__ == "__main__":
    import uvicorn
    
    # Run the application
    uvicorn.run(
        "main:app",
        host=settings.host,
        port=settings.port,
        reload=settings.debug,
        log_level=settings.log_level.lower()
    )

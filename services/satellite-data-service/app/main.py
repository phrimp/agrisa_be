import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI

from app.config.settings import get_settings
from app.database.connection import init_db
from app.api.handlers import router

# Configure logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

# Get application settings
settings = get_settings()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Handle application startup and shutdown."""
    # Startup
    logger.info(f"Initializing {settings.app_name} infrastructure")
    logger.info(
        f"Database URL: {settings.database_url.split('@')[1] if '@' in settings.database_url else 'Not configured'}"
    )
    logger.info(f"MinIO Endpoint: {settings.minio_endpoint}")

    try:
        # Initialize database
        await init_db()
        logger.info("Database initialized successfully with PostGIS extension")
        logger.info("Infrastructure layers ready for service implementation")
    except Exception as e:
        logger.error(f"Failed to initialize database: {e}")
        raise

    yield

    # Shutdown
    logger.info(f"Shutting down {settings.app_name} infrastructure")


# Create FastAPI app
app = FastAPI(title=settings.app_name, version=settings.app_version, lifespan=lifespan)

# Include API routes
app.include_router(router)


@app.get("/health")
async def health_check():
    """Basic health check endpoint."""
    return {"status": "healthy", "service": settings.app_name}


# For development/testing only - DON'T use this in production
if __name__ == "__main__":
    import uvicorn

    logger.info("Starting FastAPI server for development...")
    # Use uvicorn.run() directly without asyncio.run()
    uvicorn.run(
        "app.main:app",
        host=settings.host,
        port=settings.port,
        reload=settings.debug,
        log_level=settings.log_level.lower(),
    )

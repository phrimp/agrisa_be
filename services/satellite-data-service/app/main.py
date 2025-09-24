import logging
from contextlib import asynccontextmanager

from app.config.settings import get_settings
from app.database.connection import init_db

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Get application settings
settings = get_settings()


@asynccontextmanager
async def lifespan():
    """Handle application startup and shutdown."""
    # Startup
    logger.info(f"Initializing {settings.app_name} infrastructure")
    logger.info(f"Database URL: {settings.database_url.split('@')[1] if '@' in settings.database_url else 'Not configured'}")
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


async def main():
    """Main application entry point for infrastructure initialization."""
    async with lifespan():
        logger.info("Infrastructure initialized successfully")
        logger.info("Available components:")
        logger.info("  - PostGIS database with spatial models")
        logger.info("  - MinIO client for object storage")
        logger.info("  - Configuration management")
        logger.info("Ready for service and API layer implementation")


if __name__ == "__main__":
    import asyncio
    asyncio.run(main())
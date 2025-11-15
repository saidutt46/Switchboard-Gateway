"""Main FastAPI application for Switchboard Admin API."""

from fastapi import FastAPI, Depends
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import logging

from config import get_settings
from database import init_db, check_db_connection
import redis

# Import routers
from routers import services, routes, consumers, plugins

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

settings = get_settings()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifespan events - startup and shutdown."""
    # Startup
    logger.info("Starting Switchboard Admin API...")
    
    # Check database connection
    if not check_db_connection():
        logger.error("Failed to connect to database!")
        raise Exception("Database connection failed")
    
    logger.info("Database connection established")
    
    # Initialize database tables
    init_db()
    
    # Check Redis connection
    try:
        r = redis.from_url(settings.redis_url)
        r.ping()
        logger.info("Redis connection established")
    except Exception as e:
        logger.warning(f"Redis connection failed: {e}")
    
    logger.info("Admin API ready!")
    
    yield
    
    # Shutdown
    logger.info("Shutting down Admin API...")


# Create FastAPI app
app = FastAPI(
    title=settings.app_name,
    version=settings.app_version,
    description="REST API for managing Switchboard API Gateway configuration",
    lifespan=lifespan,
    docs_url="/docs",
    redoc_url="/redoc",
    openapi_url="/openapi.json",
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(services.router, prefix="/services", tags=["Services"])
app.include_router(routes.router, prefix="/routes", tags=["Routes"])
app.include_router(consumers.router, prefix="/consumers", tags=["Consumers"])
app.include_router(plugins.router, prefix="/plugins", tags=["Plugins"])


@app.get("/")
async def root():
    """Root endpoint - API info."""
    return {
        "name": settings.app_name,
        "version": settings.app_version,
        "environment": settings.environment,
        "docs": "/docs",
        "health": "/health",
    }


@app.get("/health")
async def health():
    """Health check endpoint."""
    db_status = "healthy" if check_db_connection() else "unhealthy"
    
    # Check Redis
    redis_status = "unhealthy"
    try:
        r = redis.from_url(settings.redis_url)
        r.ping()
        redis_status = "healthy"
    except Exception:
        pass
    
    return {
        "status": "healthy" if db_status == "healthy" else "degraded",
        "version": settings.app_version,
        "database": db_status,
        "redis": redis_status,
    }

# TODO: Import and include routers here in next sessions
# from routers import services, routes, consumers, plugins
# app.include_router(services.router, prefix="/services", tags=["Services"])
# app.include_router(routes.router, prefix="/routes", tags=["Routes"])
# app.include_router(consumers.router, prefix="/consumers", tags=["Consumers"])
# app.include_router(plugins.router, prefix="/plugins", tags=["Plugins"])
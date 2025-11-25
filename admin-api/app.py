"""Main FastAPI application for Switchboard Admin API."""

from fastapi import FastAPI, Depends
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import logging

from config import get_settings
from database import init_db, check_db_connection
from pydantic import BaseModel
import redis

# Import routers
from routers import services, routes, consumers, plugins

class HealthResponse(BaseModel):
    """Health check response model."""
    status: str
    version: str
    database: str
    redis: str
    
    class Config:
        json_schema_extra = {
            "example": {
                "status": "healthy",
                "version": "1.0.0",
                "database": "healthy",
                "redis": "healthy"
            }
        }

class RootResponse(BaseModel):
    """Root endpoint response model."""
    name: str
    version: str
    environment: str
    docs: str
    health: str
    
    class Config:
        json_schema_extra = {
            "example": {
                "name": "Switchboard Admin API",
                "version": "1.0.0",
                "environment": "development",
                "docs": "/docs",
                "health": "/health"
            }
        }

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
# Create FastAPI app
app = FastAPI(
    title=settings.app_name,
    version=settings.app_version,
    description="""
    ## Switchboard Admin API
    
    REST API for managing Switchboard API Gateway configuration.
    
    ### Features
    - **Services**: Manage backend microservices
    - **Routes**: Configure request routing rules
    - **Consumers**: Manage API consumers and authentication
    - **Plugins**: Configure gateway plugins (auth, rate limiting, caching, etc.)
    
    ### Authentication
    Currently no authentication required for development. Production deployments
    should add authentication middleware.
    """,
    lifespan=lifespan,
    docs_url="/docs",
    redoc_url="/redoc",
    openapi_url="/openapi.json",
    # Enhanced metadata
    contact={
        "name": "Switchboard Gateway",
        "url": "https://github.com/saidutt46/Switchboard-Gateway",
    },
    license_info={
        "name": "Apache 2.0",
        "url": "https://www.apache.org/licenses/LICENSE-2.0.html",
    },
    openapi_tags=[
        {
            "name": "Services",
            "description": "Manage backend microservices that handle requests",
        },
        {
            "name": "Routes",
            "description": "Configure routing rules (paths, methods, hosts)",
        },
        {
            "name": "Consumers",
            "description": "Manage API consumers and their API keys",
        },
        {
            "name": "Plugins",
            "description": "Configure gateway plugins (auth, rate limiting, CORS, etc.)",
        },
    ],
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


@app.get(
    "/",
    response_model=RootResponse,
    summary="API Information",
    description="Get basic information about the Admin API",
)
async def root():
    """Root endpoint with API metadata and important links."""
    return {
        "name": settings.app_name,
        "version": settings.app_version,
        "environment": settings.environment,
        "docs": "/docs",
        "health": "/health",
    }

@app.get(
    "/health",
    response_model=HealthResponse,
    summary="Health Check",
    description="Check the health status of Admin API and its dependencies",
)
async def health():
    """Health check endpoint with detailed status of all components."""
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
"""Main FastAPI application for Switchboard Admin API."""

from fastapi import FastAPI, Depends
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import logging

from config import get_settings
from database import init_db, check_db_connection, SessionLocal
from pydantic import BaseModel
import redis

# Import routers
from routers import services, routes, consumers, plugins
from routers.auth import router as auth_router, seed_admin_from_env
from auth import get_current_user, require_write_access

class HealthResponse(BaseModel):
    """Health check response model."""
    status: str
    version: str
    database: str
    redis: str

class RootResponse(BaseModel):
    """Root endpoint response model."""
    name: str
    version: str
    environment: str
    docs: str
    health: str

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

    # Seed admin user from environment variables
    db = SessionLocal()
    try:
        seed_admin_from_env(db)
    finally:
        db.close()

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
    description="""
    ## Switchboard Admin API

    REST API for managing Switchboard API Gateway configuration.

    ### Features
    - **Services**: Manage backend microservices
    - **Routes**: Configure request routing rules
    - **Consumers**: Manage API consumers and authentication
    - **Plugins**: Configure gateway plugins (auth, rate limiting, caching, etc.)

    ### Authentication
    All management endpoints require JWT authentication.
    Use `/auth/login` to obtain tokens.
    """,
    lifespan=lifespan,
    docs_url="/docs",
    redoc_url="/redoc",
    openapi_url="/openapi.json",
    contact={
        "name": "Switchboard Gateway",
        "url": "https://github.com/saidutt46/Switchboard-Gateway",
    },
    license_info={
        "name": "Apache 2.0",
        "url": "https://www.apache.org/licenses/LICENSE-2.0.html",
    },
    openapi_tags=[
        {"name": "Auth", "description": "Authentication, user management, and setup"},
        {"name": "Services", "description": "Manage backend microservices that handle requests"},
        {"name": "Routes", "description": "Configure routing rules (paths, methods, hosts)"},
        {"name": "Consumers", "description": "Manage API consumers and their API keys"},
        {"name": "Plugins", "description": "Configure gateway plugins (auth, rate limiting, CORS, etc.)"},
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

# Auth router (public — login, setup, status)
app.include_router(auth_router)

# Protected routers — require JWT auth
app.include_router(services.router, prefix="/services", tags=["Services"], dependencies=[Depends(get_current_user)])
app.include_router(routes.router, prefix="/routes", tags=["Routes"], dependencies=[Depends(get_current_user)])
app.include_router(consumers.router, prefix="/consumers", tags=["Consumers"], dependencies=[Depends(get_current_user)])
app.include_router(plugins.router, prefix="/plugins", tags=["Plugins"], dependencies=[Depends(get_current_user)])


@app.get("/", response_model=RootResponse, summary="API Information")
async def root():
    """Root endpoint with API metadata."""
    return {
        "name": settings.app_name,
        "version": settings.app_version,
        "environment": settings.environment,
        "docs": "/docs",
        "health": "/health",
    }

@app.get("/health", response_model=HealthResponse, summary="Health Check")
async def health():
    """Health check endpoint — always public."""
    db_status = "healthy" if check_db_connection() else "unhealthy"

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

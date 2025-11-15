"""Pydantic schemas for request/response validation."""

from pydantic import BaseModel, Field, validator
from typing import Optional, List
from datetime import datetime
from uuid import UUID


# ============================================================================
# Service Schemas
# ============================================================================

class ServiceBase(BaseModel):
    """Base service schema with common fields."""
    name: str = Field(..., min_length=1, max_length=100)
    protocol: str = Field(default="http", pattern="^(http|https|grpc)$")
    host: str = Field(..., min_length=1, max_length=255)
    port: int = Field(default=80, ge=1, le=65535)
    path: Optional[str] = Field(None, max_length=255)
    connect_timeout_ms: int = Field(default=5000, ge=100)
    read_timeout_ms: int = Field(default=60000, ge=100)
    write_timeout_ms: int = Field(default=60000, ge=100)
    retries: int = Field(default=0, ge=0, le=10)
    load_balancer_type: str = Field(default="round-robin")
    enabled: bool = Field(default=True)


class ServiceCreate(ServiceBase):
    """Schema for creating a service."""
    pass


class ServiceUpdate(BaseModel):
    """Schema for updating a service (all fields optional)."""
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    protocol: Optional[str] = Field(None, pattern="^(http|https|grpc)$")
    host: Optional[str] = Field(None, min_length=1, max_length=255)
    port: Optional[int] = Field(None, ge=1, le=65535)
    path: Optional[str] = Field(None, max_length=255)
    connect_timeout_ms: Optional[int] = Field(None, ge=100)
    read_timeout_ms: Optional[int] = Field(None, ge=100)
    write_timeout_ms: Optional[int] = Field(None, ge=100)
    retries: Optional[int] = Field(None, ge=0, le=10)
    load_balancer_type: Optional[str] = None
    enabled: Optional[bool] = None


class ServiceResponse(ServiceBase):
    """Schema for service response."""
    id: UUID
    created_at: datetime
    updated_at: datetime
    
    class Config:
        from_attributes = True


# ============================================================================
# Route Schemas
# ============================================================================

class RouteBase(BaseModel):
    """Base route schema with common fields."""
    service_id: UUID
    name: Optional[str] = Field(None, max_length=100)
    hosts: Optional[List[str]] = None
    paths: List[str] = Field(..., min_length=1)
    methods: List[str] = Field(default=["GET", "POST", "PUT", "DELETE", "PATCH"])
    strip_path: bool = Field(default=False)
    preserve_host: bool = Field(default=False)
    enabled: bool = Field(default=True)
    
    @validator("methods")
    def validate_methods(cls, v):
        """Validate HTTP methods."""
        valid_methods = ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"]
        for method in v:
            if method.upper() not in valid_methods:
                raise ValueError(f"Invalid HTTP method: {method}")
        return [m.upper() for m in v]
    
    @validator("paths")
    def validate_paths(cls, v):
        """Validate paths start with /."""
        for path in v:
            if not path.startswith("/"):
                raise ValueError(f"Path must start with /: {path}")
        return v


class RouteCreate(RouteBase):
    """Schema for creating a route."""
    pass


class RouteUpdate(BaseModel):
    """Schema for updating a route (all fields optional)."""
    service_id: Optional[UUID] = None
    name: Optional[str] = Field(None, max_length=100)
    hosts: Optional[List[str]] = None
    paths: Optional[List[str]] = None
    methods: Optional[List[str]] = None
    strip_path: Optional[bool] = None
    preserve_host: Optional[bool] = None
    enabled: Optional[bool] = None


class RouteResponse(RouteBase):
    """Schema for route response."""
    id: UUID
    created_at: datetime
    updated_at: datetime
    
    class Config:
        from_attributes = True


# ============================================================================
# Health Check Schema
# ============================================================================

class HealthResponse(BaseModel):
    """Health check response."""
    status: str
    version: str
    database: str
    redis: str
    
# ============================================================================
# Consumer Schemas
# ============================================================================

class ConsumerBase(BaseModel):
    """Base consumer schema with common fields."""
    username: str = Field(..., min_length=1, max_length=100)
    email: Optional[str] = Field(None, max_length=255)
    custom_id: Optional[str] = Field(None, max_length=100)
    custom_metadata: Optional[dict] = Field(default={})


class ConsumerCreate(ConsumerBase):
    """Schema for creating a consumer."""
    pass


class ConsumerUpdate(BaseModel):
    """Schema for updating a consumer (all fields optional)."""
    username: Optional[str] = Field(None, min_length=1, max_length=100)
    email: Optional[str] = Field(None, max_length=255)
    custom_id: Optional[str] = Field(None, max_length=100)
    custom_metadata: Optional[dict] = None


class ConsumerResponse(ConsumerBase):
    """Schema for consumer response."""
    id: UUID
    created_at: datetime
    updated_at: datetime
    
    class Config:
        from_attributes = True


# ============================================================================
# Plugin Schemas
# ============================================================================

class PluginBase(BaseModel):
    """Base plugin schema with common fields."""
    name: str = Field(..., min_length=1, max_length=50)
    scope: str = Field(..., pattern="^(global|service|route|consumer)$")
    service_id: Optional[UUID] = None
    route_id: Optional[UUID] = None
    consumer_id: Optional[UUID] = None
    config: dict = Field(default={})
    enabled: bool = Field(default=True)
    priority: int = Field(default=100, ge=1, le=1000)


class PluginCreate(PluginBase):
    """Schema for creating a plugin."""
    pass


class PluginUpdate(BaseModel):
    """Schema for updating a plugin (all fields optional)."""
    name: Optional[str] = Field(None, min_length=1, max_length=50)
    scope: Optional[str] = Field(None, pattern="^(global|service|route|consumer)$")
    service_id: Optional[UUID] = None
    route_id: Optional[UUID] = None
    consumer_id: Optional[UUID] = None
    config: Optional[dict] = None
    enabled: Optional[bool] = None
    priority: Optional[int] = Field(None, ge=1, le=1000)


class PluginResponse(PluginBase):
    """Schema for plugin response."""
    id: UUID
    created_at: datetime
    updated_at: datetime
    
    class Config:
        from_attributes = True
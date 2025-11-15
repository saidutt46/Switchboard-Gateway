"""SQLAlchemy models for database tables."""

from sqlalchemy import (
    Column, String, Integer, Boolean, DateTime, Text, 
    ForeignKey, ARRAY, JSON, CheckConstraint
)
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
import uuid

from database import Base


class Service(Base):
    """Service model - represents backend services."""
    
    __tablename__ = "services"
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    name = Column(String(100), unique=True, nullable=False)
    
    # Connection
    protocol = Column(String(10), nullable=False, default="http")
    host = Column(String(255), nullable=False)
    port = Column(Integer, nullable=False, default=80)
    path = Column(String(255), nullable=True)
    
    # Timeouts (milliseconds)
    connect_timeout_ms = Column(Integer, default=5000)
    read_timeout_ms = Column(Integer, default=60000)
    write_timeout_ms = Column(Integer, default=60000)
    retries = Column(Integer, default=0)
    
    # Load balancing
    load_balancer_type = Column(String(50), default="round-robin")
    
    # Status
    enabled = Column(Boolean, default=True)
    
    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())
    
    # Relationships
    routes = relationship("Route", back_populates="service", cascade="all, delete-orphan")
    targets = relationship("ServiceTarget", back_populates="service", cascade="all, delete-orphan")
    plugins = relationship("Plugin", back_populates="service", cascade="all, delete-orphan")


class ServiceTarget(Base):
    """Service target model - backend instances for load balancing."""
    
    __tablename__ = "service_targets"
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    service_id = Column(UUID(as_uuid=True), ForeignKey("services.id", ondelete="CASCADE"), nullable=False)
    target = Column(String(255), nullable=False)
    weight = Column(Integer, default=100)
    health_check_path = Column(String(255), default="/health")
    enabled = Column(Boolean, default=True)
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    
    # Relationships
    service = relationship("Service", back_populates="targets")


class Route(Base):
    """Route model - maps requests to services."""
    
    __tablename__ = "routes"
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    service_id = Column(UUID(as_uuid=True), ForeignKey("services.id", ondelete="CASCADE"), nullable=False)
    name = Column(String(100), nullable=True)
    
    # Matching
    hosts = Column(ARRAY(Text), nullable=True)
    paths = Column(ARRAY(Text), nullable=False)
    methods = Column(ARRAY(Text), default=["GET", "POST", "PUT", "DELETE", "PATCH"])
    
    # Path handling
    strip_path = Column(Boolean, default=False)
    preserve_host = Column(Boolean, default=False)
    
    # Status
    enabled = Column(Boolean, default=True)
    
    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())
    
    # Relationships
    service = relationship("Service", back_populates="routes")
    plugins = relationship("Plugin", back_populates="route", cascade="all, delete-orphan")


class Consumer(Base):
    """Consumer model - API clients/applications."""
    
    __tablename__ = "consumers"
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    username = Column(String(100), unique=True, nullable=False)
    email = Column(String(255), nullable=True)
    custom_id = Column(String(100), nullable=True)
    custom_metadata = Column("metadata", JSON, default={})
    
    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())
    
    # Relationships
    api_keys = relationship("APIKey", back_populates="consumer", cascade="all, delete-orphan")
    plugins = relationship("Plugin", back_populates="consumer", cascade="all, delete-orphan")


class APIKey(Base):
    """API Key model - authentication credentials."""
    
    __tablename__ = "api_keys"
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    consumer_id = Column(UUID(as_uuid=True), ForeignKey("consumers.id", ondelete="CASCADE"), nullable=False)
    key_hash = Column(String(64), unique=True, nullable=False)  # SHA256 hash
    name = Column(String(100), nullable=True)
    enabled = Column(Boolean, default=True)
    
    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    last_used_at = Column(DateTime(timezone=True), nullable=True)
    expires_at = Column(DateTime(timezone=True), nullable=True)
    
    # Relationships
    consumer = relationship("Consumer", back_populates="api_keys")


class Plugin(Base):
    """Plugin model - gateway functionality (auth, rate limiting, etc)."""
    
    __tablename__ = "plugins"
    
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    name = Column(String(50), nullable=False)
    scope = Column(String(20), nullable=False)  # global, service, route, consumer
    
    # Foreign keys (nullable based on scope)
    service_id = Column(UUID(as_uuid=True), ForeignKey("services.id", ondelete="CASCADE"), nullable=True)
    route_id = Column(UUID(as_uuid=True), ForeignKey("routes.id", ondelete="CASCADE"), nullable=True)
    consumer_id = Column(UUID(as_uuid=True), ForeignKey("consumers.id", ondelete="CASCADE"), nullable=True)
    
    # Configuration
    config = Column(JSON, nullable=False, default={})
    enabled = Column(Boolean, default=True)
    priority = Column(Integer, default=100)
    
    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())
    
    # Relationships
    service = relationship("Service", back_populates="plugins")
    route = relationship("Route", back_populates="plugins")
    consumer = relationship("Consumer", back_populates="plugins")
    
    # Constraint to enforce scope rules
    __table_args__ = (
        CheckConstraint(
            """
            (scope = 'global' AND service_id IS NULL AND route_id IS NULL AND consumer_id IS NULL) OR
            (scope = 'service' AND service_id IS NOT NULL AND route_id IS NULL AND consumer_id IS NULL) OR
            (scope = 'route' AND route_id IS NOT NULL AND service_id IS NULL AND consumer_id IS NULL) OR
            (scope = 'consumer' AND consumer_id IS NOT NULL AND service_id IS NULL AND route_id IS NULL)
            """,
            name="plugins_scope_check"
        ),
    )
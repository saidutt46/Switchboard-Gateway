"""Plugins CRUD API endpoints."""

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from typing import List, Optional
import logging
from uuid import UUID

from database import get_db
from models import (
    Plugin as PluginModel,
    Service as ServiceModel,
    Route as RouteModel,
    Consumer as ConsumerModel
)
from schemas import PluginCreate, PluginUpdate, PluginResponse

logger = logging.getLogger(__name__)

router = APIRouter()


def validate_plugin_scope(
    scope: str,
    service_id: Optional[UUID],
    route_id: Optional[UUID],
    consumer_id: Optional[UUID],
    db: Session
) -> dict:
    """
    Validate plugin scope and associated entities.
    
    Returns dict with validation results and entity names for logging.
    """
    result = {
        "valid": True,
        "error": None,
        "entity_names": {}
    }
    
    # Validate scope rules
    if scope == "global":
        if service_id or route_id or consumer_id:
            result["valid"] = False
            result["error"] = "Global plugins cannot be associated with service, route, or consumer"
            return result
    
    elif scope == "service":
        if not service_id or route_id or consumer_id:
            result["valid"] = False
            result["error"] = "Service plugins must have service_id only"
            return result
        
        # Verify service exists
        service = db.query(ServiceModel).filter(ServiceModel.id == service_id).first()
        if not service:
            result["valid"] = False
            result["error"] = f"Service with id '{service_id}' not found"
            return result
        result["entity_names"]["service"] = service.name
    
    elif scope == "route":
        if not route_id or service_id or consumer_id:
            result["valid"] = False
            result["error"] = "Route plugins must have route_id only"
            return result
        
        # Verify route exists
        route = db.query(RouteModel).filter(RouteModel.id == route_id).first()
        if not route:
            result["valid"] = False
            result["error"] = f"Route with id '{route_id}' not found"
            return result
        result["entity_names"]["route"] = route.name
        result["entity_names"]["service"] = route.service.name
    
    elif scope == "consumer":
        if not consumer_id or service_id or route_id:
            result["valid"] = False
            result["error"] = "Consumer plugins must have consumer_id only"
            return result
        
        # Verify consumer exists
        consumer = db.query(ConsumerModel).filter(ConsumerModel.id == consumer_id).first()
        if not consumer:
            result["valid"] = False
            result["error"] = f"Consumer with id '{consumer_id}' not found"
            return result
        result["entity_names"]["consumer"] = consumer.username
    
    else:
        result["valid"] = False
        result["error"] = f"Invalid scope: {scope}. Must be one of: global, service, route, consumer"
    
    return result


@router.post("", response_model=PluginResponse, status_code=status.HTTP_201_CREATED)
def create_plugin(
    plugin: PluginCreate,
    db: Session = Depends(get_db)
):
    """
    Create a new plugin.
    
    Plugins add functionality to the gateway (auth, rate limiting, caching, etc).
    
    Scopes:
    - global: Applies to all requests
    - service: Applies to all routes of a service
    - route: Applies to a specific route
    - consumer: Applies to a specific consumer
    """
    logger.info(
        "Creating plugin",
        extra={
            "plugin_name": plugin.name,
            "scope": plugin.scope,
            "priority": plugin.priority
        }
    )
    
    # Validate scope and associated entities
    validation = validate_plugin_scope(
        plugin.scope,
        plugin.service_id,
        plugin.route_id,
        plugin.consumer_id,
        db
    )
    
    if not validation["valid"]:
        logger.warning(
            "Plugin creation failed - invalid scope",
            extra={
                "plugin_name": plugin.name,
                "scope": plugin.scope,
                "error": validation["error"]
            }
        )
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=validation["error"]
        )
    
    # Create plugin
    db_plugin = PluginModel(**plugin.model_dump())
    
    try:
        db.add(db_plugin)
        db.commit()
        db.refresh(db_plugin)
        
        log_extra = {
            "plugin_id": str(db_plugin.id),
            "plugin_name": db_plugin.name,
            "scope": db_plugin.scope,
            "priority": db_plugin.priority,
            "config": db_plugin.config
        }
        log_extra.update(validation["entity_names"])
        
        logger.info(
            "Plugin created successfully",
            extra=log_extra
        )
        
        return db_plugin
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to create plugin",
            extra={
                "plugin_name": plugin.name,
                "scope": plugin.scope,
                "error": str(e)
            },
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to create plugin"
        )


@router.get("", response_model=List[PluginResponse])
def list_plugins(
    skip: int = 0,
    limit: int = 100,
    scope: Optional[str] = None,
    name: Optional[str] = None,
    service_id: Optional[UUID] = None,
    route_id: Optional[UUID] = None,
    consumer_id: Optional[UUID] = None,
    enabled_only: bool = False,
    db: Session = Depends(get_db)
):
    """
    List all plugins.
    
    Query parameters:
    - skip: Number of records to skip (pagination)
    - limit: Maximum number of records to return
    - scope: Filter by scope (global, service, route, consumer)
    - name: Filter by plugin name
    - service_id: Filter by service
    - route_id: Filter by route
    - consumer_id: Filter by consumer
    - enabled_only: If true, only return enabled plugins
    """
    logger.debug(
        "Listing plugins",
        extra={
            "skip": skip,
            "limit": limit,
            "scope": scope,
            "name": name,
            "enabled_only": enabled_only
        }
    )
    
    query = db.query(PluginModel)
    
    # Apply filters
    if scope:
        query = query.filter(PluginModel.scope == scope)
    
    if name:
        query = query.filter(PluginModel.name == name)
    
    if service_id:
        query = query.filter(PluginModel.service_id == service_id)
    
    if route_id:
        query = query.filter(PluginModel.route_id == route_id)
    
    if consumer_id:
        query = query.filter(PluginModel.consumer_id == consumer_id)
    
    if enabled_only:
        query = query.filter(PluginModel.enabled == True)
    
    # Order by priority (lower = runs first)
    query = query.order_by(PluginModel.priority.asc())
    
    plugins = query.offset(skip).limit(limit).all()
    
    logger.info(
        "Plugins retrieved",
        extra={
            "count": len(plugins),
            "scope": scope,
            "plugin_name": name
        }
    )
    
    return plugins


@router.get("/available")
def list_available_plugins():
    """
    List all available plugin types that can be configured.
    
    This is a reference list of plugins supported by the gateway.
    """
    logger.debug("Listing available plugin types")
    
    available_plugins = {
        "authentication": [
            {
                "name": "api-key-auth",
                "description": "API key authentication",
                "config_schema": {
                    "key_names": ["X-API-Key"],
                    "key_in_header": True,
                    "key_in_query": True,
                    "hide_credentials": True
                }
            },
            {
                "name": "jwt-auth",
                "description": "JWT token authentication",
                "config_schema": {
                    "algorithm": "HS256",
                    "secret": "your-secret-key",
                    "claims_to_verify": ["exp", "iss"],
                    "header_name": "Authorization"
                }
            },
            {
                "name": "basic-auth",
                "description": "HTTP Basic Authentication",
                "config_schema": {
                    "hide_credentials": True
                }
            }
        ],
        "traffic_control": [
            {
                "name": "rate-limit",
                "description": "Request rate limiting",
                "config_schema": {
                    "limit": 1000,
                    "window": "1m",
                    "algorithm": "sliding-window",
                    "key_by": "consumer"
                }
            },
            {
                "name": "request-size-limit",
                "description": "Limit request body size",
                "config_schema": {
                    "max_size_mb": 10
                }
            },
            {
                "name": "ip-restriction",
                "description": "IP whitelist/blacklist",
                "config_schema": {
                    "allow": ["10.0.0.0/8"],
                    "deny": ["1.2.3.4"]
                }
            }
        ],
        "performance": [
            {
                "name": "cache",
                "description": "Response caching",
                "config_schema": {
                    "ttl": 300,
                    "vary": ["Accept"],
                    "methods": ["GET", "HEAD"],
                    "status_codes": [200, 301, 404]
                }
            }
        ],
        "resilience": [
            {
                "name": "circuit-breaker",
                "description": "Prevent cascading failures",
                "config_schema": {
                    "max_failures": 5,
                    "timeout": "30s",
                    "half_open_requests": 3
                }
            },
            {
                "name": "timeout",
                "description": "Request timeout enforcement",
                "config_schema": {
                    "connect_timeout_ms": 5000,
                    "read_timeout_ms": 60000
                }
            }
        ],
        "transformation": [
            {
                "name": "cors",
                "description": "CORS headers",
                "config_schema": {
                    "origins": ["*"],
                    "methods": ["GET", "POST", "PUT", "DELETE"],
                    "headers": ["*"],
                    "credentials": True
                }
            },
            {
                "name": "request-transform",
                "description": "Modify request headers/body",
                "config_schema": {
                    "add_headers": {},
                    "remove_headers": [],
                    "replace_headers": {}
                }
            }
        ]
    }
    
    logger.info("Available plugins listed")
    
    return available_plugins

@router.get("/{plugin_id}", response_model=PluginResponse)
def get_plugin(
    plugin_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Get a specific plugin by ID.
    """
    logger.debug(
        "Retrieving plugin",
        extra={"plugin_id": str(plugin_id)}
    )
    
    plugin = db.query(PluginModel).filter(PluginModel.id == plugin_id).first()
    
    if not plugin:
        logger.warning(
            "Plugin not found",
            extra={"plugin_id": str(plugin_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Plugin with id '{plugin_id}' not found"
        )
    
    logger.info(
        "Plugin retrieved",
        extra={
            "plugin_id": str(plugin_id),
            "plugin_name": plugin.name,
            "scope": plugin.scope
        }
    )
    
    return plugin


@router.put("/{plugin_id}", response_model=PluginResponse)
def update_plugin(
    plugin_id: UUID,
    plugin_update: PluginUpdate,
    db: Session = Depends(get_db)
):
    """
    Update a plugin.
    
    Only provided fields will be updated. Omitted fields remain unchanged.
    
    Note: Changing scope requires proper associated IDs.
    """
    logger.info(
        "Updating plugin",
        extra={"plugin_id": str(plugin_id)}
    )
    
    # Get existing plugin
    db_plugin = db.query(PluginModel).filter(PluginModel.id == plugin_id).first()
    
    if not db_plugin:
        logger.warning(
            "Plugin update failed - not found",
            extra={"plugin_id": str(plugin_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Plugin with id '{plugin_id}' not found"
        )
    
    # Get update data
    update_data = plugin_update.model_dump(exclude_unset=True)
    
    # Determine final scope and IDs (use updated values or existing)
    final_scope = update_data.get("scope", db_plugin.scope)
    final_service_id = update_data.get("service_id", db_plugin.service_id)
    final_route_id = update_data.get("route_id", db_plugin.route_id)
    final_consumer_id = update_data.get("consumer_id", db_plugin.consumer_id)
    
    # Validate scope if any scope-related field is being updated
    if any(key in update_data for key in ["scope", "service_id", "route_id", "consumer_id"]):
        validation = validate_plugin_scope(
            final_scope,
            final_service_id,
            final_route_id,
            final_consumer_id,
            db
        )
        
        if not validation["valid"]:
            logger.warning(
                "Plugin update failed - invalid scope",
                extra={
                    "plugin_id": str(plugin_id),
                    "scope": final_scope,
                    "error": validation["error"]
                }
            )
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=validation["error"]
            )
    
    # Update fields
    try:
        for field, value in update_data.items():
            setattr(db_plugin, field, value)
        
        db.commit()
        db.refresh(db_plugin)
        
        logger.info(
            "Plugin updated successfully",
            extra={
                "plugin_id": str(plugin_id),
                "plugin_name": db_plugin.name,
                "scope": db_plugin.scope,
                "updated_fields": list(update_data.keys())
            }
        )
        
        return db_plugin
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to update plugin",
            extra={"plugin_id": str(plugin_id), "error": str(e)},
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to update plugin"
        )


@router.delete("/{plugin_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_plugin(
    plugin_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Delete a plugin.
    """
    logger.info(
        "Deleting plugin",
        extra={"plugin_id": str(plugin_id)}
    )
    
    # Get plugin
    db_plugin = db.query(PluginModel).filter(PluginModel.id == plugin_id).first()
    
    if not db_plugin:
        logger.warning(
            "Plugin deletion failed - not found",
            extra={"plugin_id": str(plugin_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Plugin with id '{plugin_id}' not found"
        )
    
    plugin_name = db_plugin.name
    plugin_scope = db_plugin.scope
    
    try:
        db.delete(db_plugin)
        db.commit()
        
        logger.info(
            "Plugin deleted successfully",
            extra={
                "plugin_id": str(plugin_id),
                "plugin_name": plugin_name,
                "scope": plugin_scope
            }
        )
        
        return None
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to delete plugin",
            extra={
                "plugin_id": str(plugin_id),
                "plugin_name": plugin_name,
                "error": str(e)
            },
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to delete plugin"
        )
"""Routes CRUD API endpoints."""

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from typing import List, Optional
import logging
from uuid import UUID

from database import get_db
from models import Route as RouteModel, Service as ServiceModel
from schemas import RouteCreate, RouteUpdate, RouteResponse
from events import publish_route_change

logger = logging.getLogger(__name__)

router = APIRouter()


@router.post("", response_model=RouteResponse, status_code=status.HTTP_201_CREATED)
def create_route(
    route: RouteCreate,
    db: Session = Depends(get_db)
):
    """
    Create a new route.
    
    A route maps incoming requests to a backend service based on
    path, method, and optionally host.
    """
    logger.info(
        "Creating route",
        extra={
            "route_name": route.name,
            "service_id": str(route.service_id),
            "paths": route.paths,
            "methods": route.methods
        }
    )
    
    # Verify service exists
    service = db.query(ServiceModel).filter(ServiceModel.id == route.service_id).first()
    if not service:
        logger.warning(
            "Route creation failed - service not found",
            extra={"service_id": str(route.service_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Service with id '{route.service_id}' not found"
        )
    
    # Check if route name already exists (if name provided)
    if route.name:
        existing = db.query(RouteModel).filter(RouteModel.name == route.name).first()
        if existing:
            logger.warning(
                "Route creation failed - name already exists",
                extra={"route_name": route.name}
            )
            raise HTTPException(
                status_code=status.HTTP_409_CONFLICT,
                detail=f"Route with name '{route.name}' already exists"
            )
    
    # Create route
    db_route = RouteModel(**route.model_dump())
    
    try:
        db.add(db_route)
        db.commit()
        db.refresh(db_route)
        
        # Publish config change event
        publish_route_change(db_route.id, "created", {
            "name": db_route.name,
            "paths": db_route.paths,
            "service_id": str(db_route.service_id)
        })
        
        logger.info(
            "Route created successfully",
            extra={
                "route_id": str(db_route.id),
                "route_name": db_route.name,
                "service_id": str(db_route.service_id),
                "service_name": service.name,
                "paths": db_route.paths
            }
        )
        
        return db_route
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to create route",
            extra={
                "route_name": route.name,
                "service_id": str(route.service_id),
                "error": str(e)
            },
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to create route"
        )


@router.get("", response_model=List[RouteResponse])
def list_routes(
    skip: int = 0,
    limit: int = 100,
    service_id: Optional[UUID] = None,
    enabled_only: bool = False,
    db: Session = Depends(get_db)
):
    """
    List all routes.
    
    Query parameters:
    - skip: Number of records to skip (pagination)
    - limit: Maximum number of records to return
    - service_id: Filter by service ID
    - enabled_only: If true, only return enabled routes
    """
    logger.debug(
        "Listing routes",
        extra={
            "skip": skip,
            "limit": limit,
            "service_id": str(service_id) if service_id else None,
            "enabled_only": enabled_only
        }
    )
    
    query = db.query(RouteModel)
    
    # Filter by service
    if service_id:
        query = query.filter(RouteModel.service_id == service_id)
    
    # Filter by enabled
    if enabled_only:
        query = query.filter(RouteModel.enabled == True)
    
    routes = query.offset(skip).limit(limit).all()
    
    logger.info(
        "Routes retrieved",
        extra={
            "count": len(routes),
            "service_id": str(service_id) if service_id else None,
            "enabled_only": enabled_only
        }
    )
    
    return routes


@router.get("/{route_id}", response_model=RouteResponse)
def get_route(
    route_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Get a specific route by ID.
    """
    logger.debug(
        "Retrieving route",
        extra={"route_id": str(route_id)}
    )
    
    route = db.query(RouteModel).filter(RouteModel.id == route_id).first()
    
    if not route:
        logger.warning(
            "Route not found",
            extra={"route_id": str(route_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route with id '{route_id}' not found"
        )
    
    logger.info(
        "Route retrieved",
        extra={
            "route_id": str(route_id),
            "route_name": route.name,
            "service_id": str(route.service_id)
        }
    )
    
    return route


@router.put("/{route_id}", response_model=RouteResponse)
def update_route(
    route_id: UUID,
    route_update: RouteUpdate,
    db: Session = Depends(get_db)
):
    """
    Update a route.
    
    Only provided fields will be updated. Omitted fields remain unchanged.
    """
    logger.info(
        "Updating route",
        extra={"route_id": str(route_id)}
    )
    
    # Get existing route
    db_route = db.query(RouteModel).filter(RouteModel.id == route_id).first()
    
    if not db_route:
        logger.warning(
            "Route update failed - not found",
            extra={"route_id": str(route_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route with id '{route_id}' not found"
        )
    
    # Verify new service exists (if changing service)
    if route_update.service_id:
        service = db.query(ServiceModel).filter(
            ServiceModel.id == route_update.service_id
        ).first()
        if not service:
            logger.warning(
                "Route update failed - service not found",
                extra={
                    "route_id": str(route_id),
                    "service_id": str(route_update.service_id)
                }
            )
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Service with id '{route_update.service_id}' not found"
            )
    
    # Check if new name conflicts
    if route_update.name and route_update.name != db_route.name:
        existing = db.query(RouteModel).filter(
            RouteModel.name == route_update.name,
            RouteModel.id != route_id
        ).first()
        if existing:
            logger.warning(
                "Route update failed - name conflict",
                extra={
                    "route_id": str(route_id),
                    "new_name": route_update.name
                }
            )
            raise HTTPException(
                status_code=status.HTTP_409_CONFLICT,
                detail=f"Route with name '{route_update.name}' already exists"
            )
    
    # Update fields
    update_data = route_update.model_dump(exclude_unset=True)
    
    try:
        for field, value in update_data.items():
            setattr(db_route, field, value)
        
        db.commit()
        db.refresh(db_route)
        
        # Publish config change event
        publish_route_change(route_id, "updated", {
            "name": db_route.name,
            "updated_fields": list(update_data.keys())
        })
        
        logger.info(
            "Route updated successfully",
            extra={
                "route_id": str(route_id),
                "route_name": db_route.name,
                "updated_fields": list(update_data.keys())
            }
        )
        
        return db_route
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to update route",
            extra={"route_id": str(route_id), "error": str(e)},
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to update route"
        )


@router.delete("/{route_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_route(
    route_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Delete a route.
    
    Warning: This will also delete all associated plugins
    due to CASCADE constraints.
    """
    logger.info(
        "Deleting route",
        extra={"route_id": str(route_id)}
    )
    
    # Get route
    db_route = db.query(RouteModel).filter(RouteModel.id == route_id).first()
    
    if not db_route:
        logger.warning(
            "Route deletion failed - not found",
            extra={"route_id": str(route_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route with id '{route_id}' not found"
        )
    
    route_name = db_route.name
    service_id = str(db_route.service_id)
    
    try:
        db.delete(db_route)
        db.commit()
        
        # Publish config change event
        publish_route_change(route_id, "deleted", {
            "name": route_name,
            "service_id": service_id
        })
        
        logger.info(
            "Route deleted successfully",
            extra={
                "route_id": str(route_id),
                "route_name": route_name,
                "service_id": service_id
            }
        )
        
        return None
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to delete route",
            extra={
                "route_id": str(route_id),
                "route_name": route_name,
                "error": str(e)
            },
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to delete route"
        )


@router.get("/{route_id}/details")
def get_route_details(
    route_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Get detailed information about a route including service details.
    """
    logger.debug(
        "Retrieving route details",
        extra={"route_id": str(route_id)}
    )
    
    # Get route with service
    route = db.query(RouteModel).filter(RouteModel.id == route_id).first()
    
    if not route:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route with id '{route_id}' not found"
        )
    
    # Build response with service info
    details = {
        "route": {
            "id": str(route.id),
            "name": route.name,
            "paths": route.paths,
            "methods": route.methods,
            "hosts": route.hosts,
            "strip_path": route.strip_path,
            "preserve_host": route.preserve_host,
            "enabled": route.enabled,
            "created_at": route.created_at.isoformat(),
            "updated_at": route.updated_at.isoformat()
        },
        "service": {
            "id": str(route.service.id),
            "name": route.service.name,
            "protocol": route.service.protocol,
            "host": route.service.host,
            "port": route.service.port,
            "enabled": route.service.enabled
        },
        "plugins_count": len(route.plugins)
    }
    
    logger.info(
        "Route details retrieved",
        extra={
            "route_id": str(route_id),
            "service_name": route.service.name
        }
    )
    
    return details
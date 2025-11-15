"""Services CRUD API endpoints."""

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from typing import List
import logging
from uuid import UUID

from database import get_db
from models import Service as ServiceModel
from schemas import ServiceCreate, ServiceUpdate, ServiceResponse

logger = logging.getLogger(__name__)

router = APIRouter()


@router.post("", response_model=ServiceResponse, status_code=status.HTTP_201_CREATED)
def create_service(
    service: ServiceCreate,
    db: Session = Depends(get_db)
):
    """
    Create a new service.
    
    A service represents a backend microservice that the gateway can route to.
    """
    logger.info(
        "Creating service",
        extra={
            "service_name": service.name,
            "host": service.host,
            "port": service.port,
            "protocol": service.protocol
        }
    )
    
    # Check if service with this name already exists
    existing = db.query(ServiceModel).filter(ServiceModel.name == service.name).first()
    if existing:
        logger.warning(
            "Service creation failed - name already exists",
            extra={"service_name": service.name}
        )
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail=f"Service with name '{service.name}' already exists"
        )
    
    # Create service
    db_service = ServiceModel(**service.model_dump())
    
    try:
        db.add(db_service)
        db.commit()
        db.refresh(db_service)
        
        logger.info(
            "Service created successfully",
            extra={
                "service_id": str(db_service.id),
                "service_name": db_service.name,
                "target": f"{db_service.protocol}://{db_service.host}:{db_service.port}"
            }
        )
        
        return db_service
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to create service",
            extra={"service_name": service.name, "error": str(e)},
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to create service"
        )


@router.get("", response_model=List[ServiceResponse])
def list_services(
    skip: int = 0,
    limit: int = 100,
    enabled_only: bool = False,
    db: Session = Depends(get_db)
):
    """
    List all services.
    
    Query parameters:
    - skip: Number of records to skip (pagination)
    - limit: Maximum number of records to return
    - enabled_only: If true, only return enabled services
    """
    logger.debug(
        "Listing services",
        extra={"skip": skip, "limit": limit, "enabled_only": enabled_only}
    )
    
    query = db.query(ServiceModel)
    
    if enabled_only:
        query = query.filter(ServiceModel.enabled == True)
    
    services = query.offset(skip).limit(limit).all()
    
    logger.info(
        "Services retrieved",
        extra={"count": len(services), "enabled_only": enabled_only}
    )
    
    return services


@router.get("/{service_id}", response_model=ServiceResponse)
def get_service(
    service_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Get a specific service by ID.
    """
    logger.debug(
        "Retrieving service",
        extra={"service_id": str(service_id)}
    )
    
    service = db.query(ServiceModel).filter(ServiceModel.id == service_id).first()
    
    if not service:
        logger.warning(
            "Service not found",
            extra={"service_id": str(service_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Service with id '{service_id}' not found"
        )
    
    logger.info(
        "Service retrieved",
        extra={
            "service_id": str(service_id),
            "service_name": service.name
        }
    )
    
    return service


@router.put("/{service_id}", response_model=ServiceResponse)
def update_service(
    service_id: UUID,
    service_update: ServiceUpdate,
    db: Session = Depends(get_db)
):
    """
    Update a service.
    
    Only provided fields will be updated. Omitted fields remain unchanged.
    """
    logger.info(
        "Updating service",
        extra={"service_id": str(service_id)}
    )
    
    # Get existing service
    db_service = db.query(ServiceModel).filter(ServiceModel.id == service_id).first()
    
    if not db_service:
        logger.warning(
            "Service update failed - not found",
            extra={"service_id": str(service_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Service with id '{service_id}' not found"
        )
    
    # Check if new name conflicts with existing service
    if service_update.name and service_update.name != db_service.name:
        existing = db.query(ServiceModel).filter(
            ServiceModel.name == service_update.name,
            ServiceModel.id != service_id
        ).first()
        if existing:
            logger.warning(
                "Service update failed - name conflict",
                extra={
                    "service_id": str(service_id),
                    "new_name": service_update.name
                }
            )
            raise HTTPException(
                status_code=status.HTTP_409_CONFLICT,
                detail=f"Service with name '{service_update.name}' already exists"
            )
    
    # Update fields
    update_data = service_update.model_dump(exclude_unset=True)
    
    try:
        for field, value in update_data.items():
            setattr(db_service, field, value)
        
        db.commit()
        db.refresh(db_service)
        
        logger.info(
            "Service updated successfully",
            extra={
                "service_id": str(service_id),
                "service_name": db_service.name,
                "updated_fields": list(update_data.keys())
            }
        )
        
        return db_service
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to update service",
            extra={"service_id": str(service_id), "error": str(e)},
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to update service"
        )


@router.delete("/{service_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_service(
    service_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Delete a service.
    
    Warning: This will also delete all associated routes, targets, and plugins
    due to CASCADE constraints.
    """
    logger.info(
        "Deleting service",
        extra={"service_id": str(service_id)}
    )
    
    # Get service
    db_service = db.query(ServiceModel).filter(ServiceModel.id == service_id).first()
    
    if not db_service:
        logger.warning(
            "Service deletion failed - not found",
            extra={"service_id": str(service_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Service with id '{service_id}' not found"
        )
    
    service_name = db_service.name
    
    try:
        db.delete(db_service)
        db.commit()
        
        logger.info(
            "Service deleted successfully",
            extra={
                "service_id": str(service_id),
                "service_name": service_name
            }
        )
        
        return None
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to delete service",
            extra={
                "service_id": str(service_id),
                "service_name": service_name,
                "error": str(e)
            },
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to delete service"
        )


@router.get("/{service_id}/stats")
def get_service_stats(
    service_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Get statistics for a service.
    
    Returns counts of associated routes, targets, and plugins.
    """
    logger.debug(
        "Retrieving service stats",
        extra={"service_id": str(service_id)}
    )
    
    # Get service
    service = db.query(ServiceModel).filter(ServiceModel.id == service_id).first()
    
    if not service:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Service with id '{service_id}' not found"
        )
    
    stats = {
        "service_id": str(service_id),
        "service_name": service.name,
        "routes_count": len(service.routes),
        "targets_count": len(service.targets),
        "plugins_count": len(service.plugins),
        "enabled": service.enabled
    }
    
    logger.info(
        "Service stats retrieved",
        extra={
            "service_id": str(service_id),
            "stats": stats
        }
    )
    
    return stats
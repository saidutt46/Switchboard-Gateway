"""Consumers and API Keys CRUD API endpoints."""

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from typing import List
import logging
from uuid import UUID
import secrets
import hashlib
from datetime import datetime

from database import get_db
from models import Consumer as ConsumerModel, APIKey as APIKeyModel
from schemas import ConsumerCreate, ConsumerUpdate, ConsumerResponse

logger = logging.getLogger(__name__)

router = APIRouter()


# ============================================================================
# Consumers CRUD
# ============================================================================

@router.post("", response_model=ConsumerResponse, status_code=status.HTTP_201_CREATED)
def create_consumer(
    consumer: ConsumerCreate,
    db: Session = Depends(get_db)
):
    """
    Create a new consumer.
    
    A consumer represents an application or service that uses the API Gateway.
    """
    logger.info(
        "Creating consumer",
        extra={
            "username": consumer.username,
            "email": consumer.email
        }
    )
    
    # Check if consumer with this username already exists
    existing = db.query(ConsumerModel).filter(
        ConsumerModel.username == consumer.username
    ).first()
    
    if existing:
        logger.warning(
            "Consumer creation failed - username already exists",
            extra={"username": consumer.username}
        )
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail=f"Consumer with username '{consumer.username}' already exists"
        )
    
    # Create consumer
    db_consumer = ConsumerModel(**consumer.model_dump())
    
    try:
        db.add(db_consumer)
        db.commit()
        db.refresh(db_consumer)
        
        logger.info(
            "Consumer created successfully",
            extra={
                "consumer_id": str(db_consumer.id),
                "username": db_consumer.username
            }
        )
        
        return db_consumer
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to create consumer",
            extra={"username": consumer.username, "error": str(e)},
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to create consumer"
        )


@router.get("", response_model=List[ConsumerResponse])
def list_consumers(
    skip: int = 0,
    limit: int = 100,
    db: Session = Depends(get_db)
):
    """
    List all consumers.
    
    Query parameters:
    - skip: Number of records to skip (pagination)
    - limit: Maximum number of records to return
    """
    logger.debug(
        "Listing consumers",
        extra={"skip": skip, "limit": limit}
    )
    
    consumers = db.query(ConsumerModel).offset(skip).limit(limit).all()
    
    logger.info(
        "Consumers retrieved",
        extra={"count": len(consumers)}
    )
    
    return consumers


@router.get("/{consumer_id}", response_model=ConsumerResponse)
def get_consumer(
    consumer_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Get a specific consumer by ID.
    """
    logger.debug(
        "Retrieving consumer",
        extra={"consumer_id": str(consumer_id)}
    )
    
    consumer = db.query(ConsumerModel).filter(ConsumerModel.id == consumer_id).first()
    
    if not consumer:
        logger.warning(
            "Consumer not found",
            extra={"consumer_id": str(consumer_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Consumer with id '{consumer_id}' not found"
        )
    
    logger.info(
        "Consumer retrieved",
        extra={
            "consumer_id": str(consumer_id),
            "username": consumer.username
        }
    )
    
    return consumer


@router.put("/{consumer_id}", response_model=ConsumerResponse)
def update_consumer(
    consumer_id: UUID,
    consumer_update: ConsumerUpdate,
    db: Session = Depends(get_db)
):
    """
    Update a consumer.
    
    Only provided fields will be updated. Omitted fields remain unchanged.
    """
    logger.info(
        "Updating consumer",
        extra={"consumer_id": str(consumer_id)}
    )
    
    # Get existing consumer
    db_consumer = db.query(ConsumerModel).filter(ConsumerModel.id == consumer_id).first()
    
    if not db_consumer:
        logger.warning(
            "Consumer update failed - not found",
            extra={"consumer_id": str(consumer_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Consumer with id '{consumer_id}' not found"
        )
    
    # Check if new username conflicts
    if consumer_update.username and consumer_update.username != db_consumer.username:
        existing = db.query(ConsumerModel).filter(
            ConsumerModel.username == consumer_update.username,
            ConsumerModel.id != consumer_id
        ).first()
        if existing:
            logger.warning(
                "Consumer update failed - username conflict",
                extra={
                    "consumer_id": str(consumer_id),
                    "new_username": consumer_update.username
                }
            )
            raise HTTPException(
                status_code=status.HTTP_409_CONFLICT,
                detail=f"Consumer with username '{consumer_update.username}' already exists"
            )
    
    # Update fields
    update_data = consumer_update.model_dump(exclude_unset=True)
    
    try:
        for field, value in update_data.items():
            setattr(db_consumer, field, value)
        
        db.commit()
        db.refresh(db_consumer)
        
        logger.info(
            "Consumer updated successfully",
            extra={
                "consumer_id": str(consumer_id),
                "username": db_consumer.username,
                "updated_fields": list(update_data.keys())
            }
        )
        
        return db_consumer
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to update consumer",
            extra={"consumer_id": str(consumer_id), "error": str(e)},
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to update consumer"
        )


@router.delete("/{consumer_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_consumer(
    consumer_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Delete a consumer.
    
    Warning: This will also delete all associated API keys and plugins
    due to CASCADE constraints.
    """
    logger.info(
        "Deleting consumer",
        extra={"consumer_id": str(consumer_id)}
    )
    
    # Get consumer
    db_consumer = db.query(ConsumerModel).filter(ConsumerModel.id == consumer_id).first()
    
    if not db_consumer:
        logger.warning(
            "Consumer deletion failed - not found",
            extra={"consumer_id": str(consumer_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Consumer with id '{consumer_id}' not found"
        )
    
    username = db_consumer.username
    api_keys_count = len(db_consumer.api_keys)
    
    try:
        db.delete(db_consumer)
        db.commit()
        
        logger.info(
            "Consumer deleted successfully",
            extra={
                "consumer_id": str(consumer_id),
                "username": username,
                "api_keys_deleted": api_keys_count
            }
        )
        
        return None
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to delete consumer",
            extra={
                "consumer_id": str(consumer_id),
                "username": username,
                "error": str(e)
            },
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to delete consumer"
        )


# ============================================================================
# API Keys Management
# ============================================================================

def generate_api_key(environment: str = "prod") -> tuple[str, str]:
    """
    Generate a secure API key.
    
    Returns:
        tuple: (plaintext_key, hashed_key)
    
    Format: gw_{environment}_{random_32_bytes}
    """
    # Generate 32 random bytes
    random_bytes = secrets.token_urlsafe(32)
    
    # Create key with format: gw_prod_...
    plaintext_key = f"gw_{environment}_{random_bytes}"
    
    # Hash the key (SHA256)
    hashed_key = hashlib.sha256(plaintext_key.encode()).hexdigest()
    
    return plaintext_key, hashed_key


@router.post("/{consumer_id}/keys", status_code=status.HTTP_201_CREATED)
def create_api_key(
    consumer_id: UUID,
    name: str = None,
    db: Session = Depends(get_db)
):
    """
    Generate a new API key for a consumer.
    
    ⚠️ IMPORTANT: The plaintext key is only returned once!
    Save it immediately - you cannot retrieve it later.
    
    The key is stored as a SHA256 hash for security.
    """
    logger.info(
        "Generating API key",
        extra={
            "consumer_id": str(consumer_id),
            "key_name": name
        }
    )
    
    # Verify consumer exists
    consumer = db.query(ConsumerModel).filter(ConsumerModel.id == consumer_id).first()
    
    if not consumer:
        logger.warning(
            "API key creation failed - consumer not found",
            extra={"consumer_id": str(consumer_id)}
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Consumer with id '{consumer_id}' not found"
        )
    
    # Generate API key
    plaintext_key, hashed_key = generate_api_key()
    
    # Create API key record
    db_key = APIKeyModel(
        consumer_id=consumer_id,
        key_hash=hashed_key,
        name=name,
        enabled=True
    )
    
    try:
        db.add(db_key)
        db.commit()
        db.refresh(db_key)
        
        logger.info(
            "API key generated successfully",
            extra={
                "key_id": str(db_key.id),
                "consumer_id": str(consumer_id),
                "consumer_username": consumer.username,
                "key_name": name,
                "key_hash": hashed_key[:16] + "..."  # Log partial hash only
            }
        )
        
        # Return plaintext key (ONLY TIME IT'S SHOWN!)
        return {
            "id": str(db_key.id),
            "key": plaintext_key,  # ⚠️ Save this! Won't be shown again!
            "name": name,
            "consumer_id": str(consumer_id),
            "consumer_username": consumer.username,
            "enabled": True,
            "created_at": db_key.created_at.isoformat(),
            "warning": "Save this key now! It cannot be retrieved later."
        }
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to generate API key",
            extra={
                "consumer_id": str(consumer_id),
                "error": str(e)
            },
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to generate API key"
        )


@router.get("/{consumer_id}/keys")
def list_api_keys(
    consumer_id: UUID,
    db: Session = Depends(get_db)
):
    """
    List all API keys for a consumer.
    
    Note: Plaintext keys are NOT returned (only hashes stored).
    """
    logger.debug(
        "Listing API keys",
        extra={"consumer_id": str(consumer_id)}
    )
    
    # Verify consumer exists
    consumer = db.query(ConsumerModel).filter(ConsumerModel.id == consumer_id).first()
    
    if not consumer:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Consumer with id '{consumer_id}' not found"
        )
    
    keys = db.query(APIKeyModel).filter(APIKeyModel.consumer_id == consumer_id).all()
    
    logger.info(
        "API keys retrieved",
        extra={
            "consumer_id": str(consumer_id),
            "count": len(keys)
        }
    )
    
    # Return key info (without plaintext key!)
    return [
        {
            "id": str(key.id),
            "name": key.name,
            "enabled": key.enabled,
            "created_at": key.created_at.isoformat(),
            "last_used_at": key.last_used_at.isoformat() if key.last_used_at else None,
            "expires_at": key.expires_at.isoformat() if key.expires_at else None,
            "key_preview": key.key_hash[:8] + "..."  # Show partial hash
        }
        for key in keys
    ]


@router.delete("/{consumer_id}/keys/{key_id}", status_code=status.HTTP_204_NO_CONTENT)
def revoke_api_key(
    consumer_id: UUID,
    key_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Revoke (delete) an API key.
    
    This immediately invalidates the key - requests using it will fail.
    """
    logger.info(
        "Revoking API key",
        extra={
            "consumer_id": str(consumer_id),
            "key_id": str(key_id)
        }
    )
    
    # Get API key
    api_key = db.query(APIKeyModel).filter(
        APIKeyModel.id == key_id,
        APIKeyModel.consumer_id == consumer_id
    ).first()
    
    if not api_key:
        logger.warning(
            "API key revocation failed - not found",
            extra={
                "consumer_id": str(consumer_id),
                "key_id": str(key_id)
            }
        )
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"API key with id '{key_id}' not found for this consumer"
        )
    
    key_name = api_key.name
    key_hash_preview = api_key.key_hash[:8]
    
    try:
        db.delete(api_key)
        db.commit()
        
        logger.info(
            "API key revoked successfully",
            extra={
                "key_id": str(key_id),
                "consumer_id": str(consumer_id),
                "key_name": key_name,
                "key_hash_preview": key_hash_preview
            }
        )
        
        return None
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to revoke API key",
            extra={
                "key_id": str(key_id),
                "consumer_id": str(consumer_id),
                "error": str(e)
            },
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to revoke API key"
        )


@router.patch("/{consumer_id}/keys/{key_id}/disable", status_code=status.HTTP_200_OK)
def disable_api_key(
    consumer_id: UUID,
    key_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Disable an API key (without deleting it).
    
    Disabled keys cannot be used for authentication.
    Can be re-enabled later.
    """
    logger.info(
        "Disabling API key",
        extra={
            "consumer_id": str(consumer_id),
            "key_id": str(key_id)
        }
    )
    
    # Get API key
    api_key = db.query(APIKeyModel).filter(
        APIKeyModel.id == key_id,
        APIKeyModel.consumer_id == consumer_id
    ).first()
    
    if not api_key:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"API key with id '{key_id}' not found"
        )
    
    api_key.enabled = False
    
    try:
        db.commit()
        
        logger.info(
            "API key disabled successfully",
            extra={
                "key_id": str(key_id),
                "consumer_id": str(consumer_id),
                "key_name": api_key.name
            }
        )
        
        return {
            "message": "API key disabled",
            "key_id": str(key_id),
            "enabled": False
        }
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to disable API key",
            extra={
                "key_id": str(key_id),
                "error": str(e)
            },
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to disable API key"
        )


@router.patch("/{consumer_id}/keys/{key_id}/enable", status_code=status.HTTP_200_OK)
def enable_api_key(
    consumer_id: UUID,
    key_id: UUID,
    db: Session = Depends(get_db)
):
    """
    Re-enable a previously disabled API key.
    """
    logger.info(
        "Enabling API key",
        extra={
            "consumer_id": str(consumer_id),
            "key_id": str(key_id)
        }
    )
    
    # Get API key
    api_key = db.query(APIKeyModel).filter(
        APIKeyModel.id == key_id,
        APIKeyModel.consumer_id == consumer_id
    ).first()
    
    if not api_key:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"API key with id '{key_id}' not found"
        )
    
    api_key.enabled = True
    
    try:
        db.commit()
        
        logger.info(
            "API key enabled successfully",
            extra={
                "key_id": str(key_id),
                "consumer_id": str(consumer_id),
                "key_name": api_key.name
            }
        )
        
        return {
            "message": "API key enabled",
            "key_id": str(key_id),
            "enabled": True
        }
        
    except Exception as e:
        db.rollback()
        logger.error(
            "Failed to enable API key",
            extra={
                "key_id": str(key_id),
                "error": str(e)
            },
            exc_info=True
        )
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to enable API key"
        )
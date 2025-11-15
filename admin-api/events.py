"""Redis pub/sub events for config changes."""

import redis
import json
import logging
from typing import Optional
from uuid import UUID

from config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()

# Redis client
redis_client = None


def get_redis():
    """Get Redis client (singleton pattern)."""
    global redis_client
    if redis_client is None:
        redis_client = redis.from_url(settings.redis_url, decode_responses=True)
    return redis_client


def publish_config_change(
    event_type: str,
    entity_type: str,
    entity_id: UUID,
    action: str,
    metadata: Optional[dict] = None
):
    """
    Publish a configuration change event to Redis.
    
    Args:
        event_type: Type of event (config_change)
        entity_type: What was changed (service, route, consumer, plugin)
        entity_id: ID of the changed entity
        action: What happened (created, updated, deleted)
        metadata: Additional context
    """
    try:
        r = get_redis()
        
        event = {
            "event_type": event_type,
            "entity_type": entity_type,
            "entity_id": str(entity_id),
            "action": action,
            "metadata": metadata or {}
        }
        
        channel = "gateway:config:changes"
        
        # Publish to Redis
        subscribers = r.publish(channel, json.dumps(event))
        
        logger.info(
            "Config change event published",
            extra={
                "channel": channel,
                "entity_type": entity_type,
                "entity_id": str(entity_id),
                "action": action,
                "subscribers": subscribers
            }
        )
        
        return subscribers
        
    except Exception as e:
        logger.error(
            "Failed to publish config change event",
            extra={
                "entity_type": entity_type,
                "entity_id": str(entity_id),
                "action": action,
                "error": str(e)
            },
            exc_info=True
        )
        # Don't fail the request if Redis is down
        # Just log the error and continue
        return 0


def publish_service_change(service_id: UUID, action: str, metadata: Optional[dict] = None):
    """Publish service change event."""
    return publish_config_change("config_change", "service", service_id, action, metadata)


def publish_route_change(route_id: UUID, action: str, metadata: Optional[dict] = None):
    """Publish route change event."""
    return publish_config_change("config_change", "route", route_id, action, metadata)


def publish_consumer_change(consumer_id: UUID, action: str, metadata: Optional[dict] = None):
    """Publish consumer change event."""
    return publish_config_change("config_change", "consumer", consumer_id, action, metadata)


def publish_plugin_change(plugin_id: UUID, action: str, metadata: Optional[dict] = None):
    """Publish plugin change event."""
    return publish_config_change("config_change", "plugin", plugin_id, action, metadata)
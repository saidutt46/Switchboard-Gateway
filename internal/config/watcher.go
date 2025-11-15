// Package config handles configuration management and hot reload.
package config

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// ConfigChangeEvent represents a configuration change from Admin API.
type ConfigChangeEvent struct {
	EventType  string                 `json:"event_type"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Action     string                 `json:"action"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Watcher listens for configuration changes via Redis pub/sub.
type Watcher struct {
	redis   *redis.Client
	handler ConfigChangeHandler
}

// ConfigChangeHandler handles configuration change events.
type ConfigChangeHandler interface {
	HandleConfigChange(event ConfigChangeEvent) error
}

// NewWatcher creates a new configuration watcher.
func NewWatcher(redisClient *redis.Client, handler ConfigChangeHandler) *Watcher {
	return &Watcher{
		redis:   redisClient,
		handler: handler,
	}
}

// Start begins listening for configuration changes.
func (w *Watcher) Start(ctx context.Context) error {
	log.Println("Starting configuration watcher...")

	// Subscribe to config changes channel
	pubsub := w.redis.Subscribe(ctx, "gateway:config:changes")
	defer pubsub.Close()

	// Wait for subscription to be confirmed
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return err
	}

	log.Println("Subscribed to gateway:config:changes channel")

	// Listen for messages
	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			log.Println("Configuration watcher shutting down...")
			return ctx.Err()

		case msg := <-ch:
			if msg == nil {
				continue
			}

			// Parse event
			var event ConfigChangeEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("Failed to parse config change event: %v", err)
				continue
			}

			log.Printf("Received config change: type=%s entity=%s id=%s action=%s",
				event.EventType, event.EntityType, event.EntityID, event.Action)

			// Handle event
			if err := w.handler.HandleConfigChange(event); err != nil {
				log.Printf("Failed to handle config change: %v", err)
			} else {
				log.Printf("Config change applied successfully: %s %s",
					event.EntityType, event.Action)
			}
		}
	}
}

// HealthCheck verifies the watcher is connected to Redis.
func (w *Watcher) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return w.redis.Ping(ctx).Err()
}

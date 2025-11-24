// Package builtin - Shared helper functions for plugins
package builtin

import (
	"crypto/sha256"
	"fmt"
)

// hashAPIKeyFull generates a full SHA256 hash (64 hex characters).
//
// Used by: API Key Authentication plugin
// Purpose: Database lookup - must match stored hash exactly
// Security: Full hash required for cryptographic security
func hashAPIKeyFull(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", hash) // Full 64 hex chars
}

// hashAPIKeyShort generates a short SHA256 hash (16 hex characters).
//
// Used by: Rate Limiting plugin
// Purpose: Redis key generation for privacy (not security)
// Privacy: Shorter hash keeps Redis keys manageable
func hashAPIKeyShort(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", hash[:8]) // First 8 bytes = 16 hex chars
}

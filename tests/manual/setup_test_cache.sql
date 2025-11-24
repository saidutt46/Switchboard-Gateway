-- Setup test cache plugin
-- Run with: docker exec -i switchboard-postgres psql -U switchboard -d switchboard < tests/manual/setup_test_cache.sql

-- Clean up existing cache plugins
DELETE FROM plugins WHERE name = 'cache';

-- Disable jwt-auth for cache testing
UPDATE plugins SET enabled = false WHERE name = 'jwt-auth';

-- ============================================================================
-- Global Cache Plugin (applies to all routes)
-- ============================================================================
INSERT INTO plugins (name, scope, config, priority, enabled)
VALUES (
  'cache',
  'global',
  '{
    "critical": false,
    "ttl_seconds": 60,
    "max_ttl_seconds": 3600,
    "cache_methods": ["GET", "HEAD"],
    "cache_status_codes": [200, 301, 302],
    "max_body_size_bytes": 1048576,
    "bypass_paths": ["/health", "/ready", "/metrics"],
    "vary_by_consumer": false,
    "vary_by_query": true,
    "vary_by_headers": [],
    "ignore_query_params": ["utm_source", "utm_medium", "utm_campaign", "_"],
    "respect_cache_control": true,
    "respect_no_cache": true,
    "cache_errors": false,
    "error_ttl_seconds": 30,
    "add_debug_headers": true,
    "key_prefix": "cache:"
  }',
  50,
  true
);

-- Verify: Show all plugins
SELECT id, name, scope, priority, enabled
FROM plugins
ORDER BY enabled DESC, priority ASC;